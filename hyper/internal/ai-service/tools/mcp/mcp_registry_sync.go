package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/mcp/storage"

	"go.uber.org/zap"
)

// MCPRegistrySync handles syncing MCP tools from registered MCP servers to the ToolRegistry
// This allows AI to call MCP tools directly without using discover_tools + execute_tool meta-tools
type MCPRegistrySync struct {
	toolRegistry *aiservice.ToolRegistry
	toolsStorage storage.ToolsStorageInterface
	logger       *zap.Logger
	mu           sync.RWMutex
	// Track registered tools by server for removal
	serverTools map[string][]string // serverName -> []registeredToolNames
}

// NewMCPRegistrySync creates a new MCP registry sync service
func NewMCPRegistrySync(
	registry *aiservice.ToolRegistry,
	toolsStorage storage.ToolsStorageInterface,
	logger *zap.Logger,
) *MCPRegistrySync {
	return &MCPRegistrySync{
		toolRegistry: registry,
		toolsStorage: toolsStorage,
		logger:       logger,
		serverTools:  make(map[string][]string),
	}
}

// SyncAllMCPTools loads all MCP tools from storage and registers them with the ToolRegistry
// Returns the count of tools successfully registered
func (s *MCPRegistrySync) SyncAllMCPTools(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Starting MCP tools sync to ToolRegistry")

	// List all registered MCP servers
	servers, err := s.toolsStorage.ListServers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list MCP servers: %w", err)
	}

	if len(servers) == 0 {
		s.logger.Info("No MCP servers registered, nothing to sync")
		return 0, nil
	}

	s.logger.Info("Found MCP servers to sync",
		zap.Int("serverCount", len(servers)))

	totalRegistered := 0
	var syncErrors []string

	for _, server := range servers {
		count, err := s.syncServerToolsUnlocked(ctx, server)
		if err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("%s: %v", server.ServerName, err))
			s.logger.Warn("Failed to sync tools for server",
				zap.String("serverName", server.ServerName),
				zap.Error(err))
			continue
		}
		totalRegistered += count
	}

	s.logger.Info("MCP tools sync completed",
		zap.Int("totalRegistered", totalRegistered),
		zap.Int("serversProcessed", len(servers)),
		zap.Int("errors", len(syncErrors)))

	if len(syncErrors) > 0 {
		return totalRegistered, fmt.Errorf("some servers failed to sync: %s", strings.Join(syncErrors, "; "))
	}

	return totalRegistered, nil
}

// SyncServerTools syncs tools for a specific server (for use with mcp_add_server and mcp_rediscover_server)
// Returns the count of tools successfully registered
func (s *MCPRegistrySync) SyncServerTools(ctx context.Context, serverName string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Syncing MCP tools for server",
		zap.String("serverName", serverName))

	// Get server metadata
	server, err := s.toolsStorage.GetServer(ctx, serverName)
	if err != nil {
		return 0, fmt.Errorf("failed to get server %s: %w", serverName, err)
	}

	// Remove old tools for this server first (in case of rediscover)
	s.removeServerToolsUnlocked(serverName)

	return s.syncServerToolsUnlocked(ctx, server)
}

// syncServerToolsUnlocked syncs tools for a specific server (caller must hold lock)
func (s *MCPRegistrySync) syncServerToolsUnlocked(ctx context.Context, server *storage.ServerMetadata) (int, error) {
	// Get tools for this server
	tools, err := s.toolsStorage.GetServerTools(ctx, server.ServerName)
	if err != nil {
		return 0, fmt.Errorf("failed to get tools for server %s: %w", server.ServerName, err)
	}

	if len(tools) == 0 {
		s.logger.Debug("No tools found for server",
			zap.String("serverName", server.ServerName))
		return 0, nil
	}

	s.logger.Debug("Registering tools for server",
		zap.String("serverName", server.ServerName),
		zap.Int("toolCount", len(tools)))

	registered := 0
	var registeredToolNames []string

	for _, tool := range tools {
		// Create adapter for this tool
		adapter := NewMCPToolAdapter(
			tool.ToolName,
			tool.Description,
			tool.Schema,
			server.ServerName,
			server.ServerURL,
			server.Headers,
			s.logger,
		)

		// Register with ToolRegistry
		if err := s.toolRegistry.Register(adapter); err != nil {
			// Check if it's a duplicate registration (which is ok during sync)
			if strings.Contains(err.Error(), "already registered") {
				s.logger.Debug("Tool already registered, skipping",
					zap.String("toolName", adapter.Name()),
					zap.String("serverName", server.ServerName))
				registeredToolNames = append(registeredToolNames, adapter.Name())
				registered++
				continue
			}
			s.logger.Warn("Failed to register MCP tool",
				zap.String("toolName", adapter.Name()),
				zap.String("originalName", tool.ToolName),
				zap.String("serverName", server.ServerName),
				zap.Error(err))
			continue
		}

		registeredToolNames = append(registeredToolNames, adapter.Name())
		registered++
		s.logger.Debug("Registered MCP tool",
			zap.String("toolName", adapter.Name()),
			zap.String("originalName", tool.ToolName),
			zap.String("serverName", server.ServerName))
	}

	// Track registered tools for this server
	s.serverTools[server.ServerName] = registeredToolNames

	s.logger.Info("Server tools sync completed",
		zap.String("serverName", server.ServerName),
		zap.Int("registered", registered),
		zap.Int("total", len(tools)))

	return registered, nil
}

// RemoveServerTools removes all tools for a specific server from the ToolRegistry
// This is called when an MCP server is removed via mcp_remove_server
func (s *MCPRegistrySync) RemoveServerTools(serverName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.removeServerToolsUnlocked(serverName)
}

// removeServerToolsUnlocked removes all tools for a server (caller must hold lock)
func (s *MCPRegistrySync) removeServerToolsUnlocked(serverName string) error {
	toolNames, exists := s.serverTools[serverName]
	if !exists {
		s.logger.Debug("No tracked tools found for server",
			zap.String("serverName", serverName))
		return nil
	}

	s.logger.Info("Removing MCP tools for server",
		zap.String("serverName", serverName),
		zap.Int("toolCount", len(toolNames)))

	// Note: ToolRegistry doesn't have a Remove method, so we track tools
	// and they'll be cleaned up on next restart or server sync
	// For now, we just clear our tracking
	delete(s.serverTools, serverName)

	s.logger.Info("Cleared tool tracking for server",
		zap.String("serverName", serverName),
		zap.Int("toolsCleared", len(toolNames)))

	return nil
}

// GetRegisteredToolCount returns the total count of MCP tools registered via sync
func (s *MCPRegistrySync) GetRegisteredToolCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, tools := range s.serverTools {
		count += len(tools)
	}
	return count
}

// GetServerToolNames returns the tool names registered for a specific server
func (s *MCPRegistrySync) GetServerToolNames(serverName string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tools, exists := s.serverTools[serverName]; exists {
		// Return a copy to prevent external modification
		result := make([]string, len(tools))
		copy(result, tools)
		return result
	}
	return nil
}
