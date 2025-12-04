package mcp

import (
	"context"

	mcphandlers "hyper/internal/mcp/handlers"
)

// DiscoverToolsExecutor implements the discover_tools tool executor
type DiscoverToolsExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *DiscoverToolsExecutor) Name() string {
	return "discover_tools"
}

func (e *DiscoverToolsExecutor) Description() string {
	return "Discover MCP tools using natural language semantic search. Returns matching tool names with descriptions and similarity scores. Use this to find tools by description (e.g., 'video tools', 'database tools', 'file operations')."
}

func (e *DiscoverToolsExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Natural language search query describing the tools you're looking for (e.g., 'tools for video processing', 'database operations', 'file management')",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of results to return (default: 5, max: 20)",
			},
		},
		"required": []string{"query"},
	}
}

func (e *DiscoverToolsExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleDiscoverTools(ctx, args)
	return data, err
}

// GetToolSchemaExecutor implements the get_tool_schema tool executor
type GetToolSchemaExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *GetToolSchemaExecutor) Name() string {
	return "get_tool_schema"
}

func (e *GetToolSchemaExecutor) Description() string {
	return "Get the complete JSON schema for a specific MCP tool. Returns the full tool definition including parameters, types, and descriptions. Use this after discovering tools to understand how to call them."
}

func (e *GetToolSchemaExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"toolName": map[string]interface{}{
				"type":        "string",
				"description": "Exact tool name to get schema for (use discover_tools first to find tool names)",
			},
		},
		"required": []string{"toolName"},
	}
}

func (e *GetToolSchemaExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleGetToolSchema(ctx, args)
	return data, err
}

// ExecuteToolExecutor implements the execute_tool tool executor
type ExecuteToolExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *ExecuteToolExecutor) Name() string {
	return "execute_tool"
}

func (e *ExecuteToolExecutor) Description() string {
	return "Execute an MCP tool by name with specified arguments. This tool looks up the tool's server from the registry and makes an HTTP call to that server's MCP endpoint. Works with external MCP servers registered via mcp_add_server. Built-in tools cannot be executed via this tool. Use get_tool_schema first to understand required parameters."
}

func (e *ExecuteToolExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"toolName": map[string]interface{}{
				"type":        "string",
				"description": "Exact tool name to execute (from discover_tools)",
			},
			"args": map[string]interface{}{
				"type":        "object",
				"description": "Tool-specific arguments as a JSON object (see get_tool_schema for parameter details)",
			},
		},
		"required": []string{"toolName", "args"},
	}
}

func (e *ExecuteToolExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleExecuteTool(ctx, args)
	return data, err
}

// McpAddServerExecutor implements the mcp_add_server tool executor
type McpAddServerExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *McpAddServerExecutor) Name() string {
	return "mcp_add_server"
}

func (e *McpAddServerExecutor) Description() string {
	return "Add a new MCP server to the registry, discover its tools, and store them in MongoDB and Qdrant for semantic search. The server must be accessible via HTTP/HTTPS and expose the MCP protocol."
}

func (e *McpAddServerExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"serverName": map[string]interface{}{
				"type":        "string",
				"description": "Unique name for this MCP server (e.g., 'openai-mcp', 'github-mcp')",
			},
			"serverUrl": map[string]interface{}{
				"type":        "string",
				"description": "HTTP/HTTPS URL of the MCP server (e.g., 'http://localhost:3000/mcp')",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Human-readable description of what this server provides",
			},
		},
		"required": []string{"serverName", "serverUrl"},
	}
}

func (e *McpAddServerExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleMCPAddServer(ctx, args)
	return data, err
}

// McpRediscoverServerExecutor implements the mcp_rediscover_server tool executor
type McpRediscoverServerExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *McpRediscoverServerExecutor) Name() string {
	return "mcp_rediscover_server"
}

func (e *McpRediscoverServerExecutor) Description() string {
	return "Rediscover and refresh tools from an existing MCP server. This removes old tools and discovers the current set of tools available on the server."
}

func (e *McpRediscoverServerExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"serverName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the MCP server to rediscover (must already be registered)",
			},
		},
		"required": []string{"serverName"},
	}
}

func (e *McpRediscoverServerExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleMCPRediscoverServer(ctx, args)
	return data, err
}

// McpRemoveServerExecutor implements the mcp_remove_server tool executor
type McpRemoveServerExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *McpRemoveServerExecutor) Name() string {
	return "mcp_remove_server"
}

func (e *McpRemoveServerExecutor) Description() string {
	return "Remove an MCP server and all its tools from the registry. This deletes the server metadata and all associated tool data from MongoDB and Qdrant."
}

func (e *McpRemoveServerExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"serverName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the MCP server to remove",
			},
		},
		"required": []string{"serverName"},
	}
}

func (e *McpRemoveServerExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleMCPRemoveServer(ctx, args)
	return data, err
}
