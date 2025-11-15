package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// InternalToolRegistry manages internal (non-public) tool handlers
// These tools are not exposed via MCP listTools but can be:
// 1. Discovered via discover_tools (semantic search)
// 2. Executed via execute_tool (direct handler call)
type InternalToolRegistry struct {
	mu       sync.RWMutex
	handlers map[string]mcp.ToolHandler
	tools    map[string]*mcp.Tool
}

// NewInternalToolRegistry creates a new internal tool registry
func NewInternalToolRegistry() *InternalToolRegistry {
	return &InternalToolRegistry{
		handlers: make(map[string]mcp.ToolHandler),
		tools:    make(map[string]*mcp.Tool),
	}
}

// RegisterInternalTool registers a tool that should NOT appear in MCP listTools
// but can still be discovered and executed via the tool hub
func (r *InternalToolRegistry) RegisterInternalTool(tool *mcp.Tool, handler mcp.ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools[tool.Name] = tool
	r.handlers[tool.Name] = handler
}

// GetHandler retrieves the handler for an internal tool
func (r *InternalToolRegistry) GetHandler(toolName string) (mcp.ToolHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[toolName]
	return handler, exists
}

// GetTool retrieves the tool definition for an internal tool
func (r *InternalToolRegistry) GetTool(toolName string) (*mcp.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[toolName]
	return tool, exists
}

// HasTool checks if a tool is registered as internal
func (r *InternalToolRegistry) HasTool(toolName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.handlers[toolName]
	return exists
}

// ExecuteInternalTool executes an internal tool directly via its handler
func (r *InternalToolRegistry) ExecuteInternalTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	handler, exists := r.GetHandler(toolName)
	if !exists {
		return nil, nil, fmt.Errorf("internal tool '%s' not found", toolName)
	}

	// Marshal args to JSON for CallToolParamsRaw.Arguments
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal args: %w", err)
	}

	// Create a CallToolRequest (which is ServerRequest[*CallToolParamsRaw])
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      toolName,
			Arguments: argsJSON,
		},
	}

	// Execute the handler directly
	result, err := handler(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("internal tool execution failed: %w", err)
	}

	// MCP handlers return just result and error, no separate data
	// Extract data from result if needed
	return result, result, nil
}

// GetAllTools returns all registered internal tools for indexing
func (r *InternalToolRegistry) GetAllTools() []*mcp.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*mcp.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToolCount returns the number of registered internal tools
func (r *InternalToolRegistry) ToolCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.handlers)
}
