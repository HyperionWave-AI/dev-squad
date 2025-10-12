package handlers

import (
	"context"
	"fmt"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// ToolMetadataForIndexing represents tool metadata for indexing in ToolsStorage
type ToolMetadataForIndexing struct {
	ToolName    string
	Description string
	Schema      map[string]interface{}
}

// ToolMetadataRegistry collects tool metadata during registration for later indexing
type ToolMetadataRegistry struct {
	tools []ToolMetadataForIndexing
}

// NewToolMetadataRegistry creates a new tool metadata registry
func NewToolMetadataRegistry() *ToolMetadataRegistry {
	return &ToolMetadataRegistry{
		tools: make([]ToolMetadataForIndexing, 0),
	}
}

// RegisterTool adds a tool to the registry for later indexing
func (r *ToolMetadataRegistry) RegisterTool(toolName, description string, schema map[string]interface{}) {
	r.tools = append(r.tools, ToolMetadataForIndexing{
		ToolName:    toolName,
		Description: description,
		Schema:      schema,
	})
}

// GetTools returns all registered tools
func (r *ToolMetadataRegistry) GetTools() []ToolMetadataForIndexing {
	return r.tools
}

// RegisterToolWithServer is a helper that registers a tool with the MCP server
// and automatically reports it to the metadata registry for indexing
func (r *ToolMetadataRegistry) RegisterToolWithServer(
	server *mcp.Server,
	tool *mcp.Tool,
	handler mcp.ToolHandler,
) {
	// Register with MCP server
	server.AddTool(tool, handler)

	// Report to metadata registry for indexing
	if r != nil {
		r.RegisterTool(
			tool.Name,
			tool.Description,
			map[string]interface{}{
				"type":        "mcp-tool",
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
			},
		)
	}
}

// IndexRegisteredTools indexes all tools from the registry into ToolsStorage
// This makes the tools discoverable via the discover_tools MCP tool
func IndexRegisteredTools(registry *ToolMetadataRegistry, toolsStorage *storage.ToolsStorage, logger *zap.Logger) (int, error) {
	if registry == nil {
		return 0, fmt.Errorf("registry is nil")
	}
	if toolsStorage == nil {
		return 0, fmt.Errorf("toolsStorage is nil")
	}

	tools := registry.GetTools()
	logger.Info("Starting MCP tool indexing for discovery...",
		zap.Int("toolCount", len(tools)))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexed := 0
	failed := 0

	// Index each tool from the registry
	for _, tool := range tools {
		// Store in ToolsStorage (MongoDB + Qdrant for semantic search)
		if err := toolsStorage.StoreToolMetadata(ctx, tool.ToolName, tool.Description, tool.Schema, "mcp-builtin"); err != nil {
			logger.Warn("Failed to index MCP tool",
				zap.String("toolName", tool.ToolName),
				zap.Error(err))
			failed++
			continue
		}

		indexed++
	}

	logger.Info("MCP tool indexing complete",
		zap.Int("indexed", indexed),
		zap.Int("failed", failed),
		zap.Int("total", len(tools)))

	return indexed, nil
}
