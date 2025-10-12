package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolsDiscoveryHandler manages MCP tools discovery operations
type ToolsDiscoveryHandler struct {
	toolsStorage     storage.ToolsStorageInterface
	httpClient       *http.Client
	metadataRegistry *ToolMetadataRegistry
}

// NewToolsDiscoveryHandler creates a new tools discovery handler
func NewToolsDiscoveryHandler(toolsStorage *storage.ToolsStorage) *ToolsDiscoveryHandler {
	return &ToolsDiscoveryHandler{
		toolsStorage: toolsStorage,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetMetadataRegistry sets the metadata registry for tool indexing
func (h *ToolsDiscoveryHandler) SetMetadataRegistry(registry *ToolMetadataRegistry) {
	h.metadataRegistry = registry
}

// addToolWithMetadata adds a tool to the server and registers it for indexing
func (h *ToolsDiscoveryHandler) addToolWithMetadata(server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandler) {
	server.AddTool(tool, handler)
	if h.metadataRegistry != nil {
		h.metadataRegistry.RegisterTool(
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

// RegisterToolsDiscoveryTools registers tools discovery tools with the MCP server
func (h *ToolsDiscoveryHandler) RegisterToolsDiscoveryTools(server *mcp.Server) error {
	// Register discover_tools tool
	if err := h.registerDiscoverTools(server); err != nil {
		return fmt.Errorf("failed to register discover_tools: %w", err)
	}

	// Register get_tool_schema tool
	if err := h.registerGetToolSchema(server); err != nil {
		return fmt.Errorf("failed to register get_tool_schema: %w", err)
	}

	// Register execute_tool tool
	if err := h.registerExecuteTool(server); err != nil {
		return fmt.Errorf("failed to register execute_tool: %w", err)
	}

	// Register mcp_add_server tool
	if err := h.registerMCPAddServer(server); err != nil {
		return fmt.Errorf("failed to register mcp_add_server: %w", err)
	}

	// Register mcp_rediscover_server tool
	if err := h.registerMCPRediscoverServer(server); err != nil {
		return fmt.Errorf("failed to register mcp_rediscover_server: %w", err)
	}

	// Register mcp_remove_server tool
	if err := h.registerMCPRemoveServer(server); err != nil {
		return fmt.Errorf("failed to register mcp_remove_server: %w", err)
	}

	return nil
}

// registerDiscoverTools registers the discover_tools tool
func (h *ToolsDiscoveryHandler) registerDiscoverTools(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "discover_tools",
		Description: "Discover MCP tools using natural language semantic search. Returns matching tool names with descriptions and similarity scores. Use this to find tools by description (e.g., 'video tools', 'database tools', 'file operations').",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"query": {
					Type:        "string",
					Description: "Natural language search query describing the tools you're looking for (e.g., 'tools for video processing', 'database operations', 'file management')",
				},
				"limit": {
					Type:        "number",
					Description: "Maximum number of results to return (default: 5, max: 20)",
				},
			},
			Required: []string{"query"},
		},
	}

	h.addToolWithMetadata(server, tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.HandleDiscoverTools(ctx, args)
		return result, err
	})

	return nil
}

// registerGetToolSchema registers the get_tool_schema tool
func (h *ToolsDiscoveryHandler) registerGetToolSchema(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "get_tool_schema",
		Description: "Get the complete JSON schema for a specific MCP tool. Returns the full tool definition including parameters, types, and descriptions. Use this after discovering tools to understand how to call them.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"toolName": {
					Type:        "string",
					Description: "Exact tool name to get schema for (use discover_tools first to find tool names)",
				},
			},
			Required: []string{"toolName"},
		},
	}

	h.addToolWithMetadata(server, tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.HandleGetToolSchema(ctx, args)
		return result, err
	})

	return nil
}

// registerExecuteTool registers the execute_tool tool
func (h *ToolsDiscoveryHandler) registerExecuteTool(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "execute_tool",
		Description: "Execute an MCP tool by name with specified arguments. This tool calls the actual tool implementation via the MCP HTTP bridge. Use get_tool_schema first to understand required parameters.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"toolName": {
					Type:        "string",
					Description: "Exact tool name to execute (from discover_tools)",
				},
				"args": {
					Type:        "object",
					Description: "Tool-specific arguments as a JSON object (see get_tool_schema for parameter details)",
				},
			},
			Required: []string{"toolName", "args"},
		},
	}

	h.addToolWithMetadata(server, tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.HandleExecuteTool(ctx, args)
		return result, err
	})

	return nil
}

// HandleDiscoverTools handles the discover_tools tool call
func (h *ToolsDiscoveryHandler) HandleDiscoverTools(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract query (required)
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return createErrorResult("query parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract limit (optional, default 5, max 20)
	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 20 {
			limit = 20
		}
		if limit < 1 {
			limit = 1
		}
	}

	// Search for tools
	matches, err := h.toolsStorage.SearchTools(ctx, query, limit)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to search tools: %s", err.Error())), nil, nil
	}

	// Format results
	if len(matches) == 0 {
		// Return empty JSON array for compatibility
		emptyArrayJSON, _ := json.Marshal([]interface{}{})
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(emptyArrayJSON)},
			},
		}, matches, nil
	}

	// Format results as structured JSON for easy parsing
	resultsJSON, err := json.MarshalIndent(matches, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to marshal results: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("Found %d matching tools:\n\n%s", len(matches), string(resultsJSON))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, matches, nil
}

// HandleGetToolSchema handles the get_tool_schema tool call
func (h *ToolsDiscoveryHandler) HandleGetToolSchema(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract toolName (required)
	toolName, ok := args["toolName"].(string)
	if !ok || toolName == "" {
		return createErrorResult("toolName parameter is required and must be a non-empty string"), nil, nil
	}

	// Get tool schema
	metadata, err := h.toolsStorage.GetToolSchema(ctx, toolName)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to get tool schema: %s", err.Error())), nil, nil
	}

	// Format schema as JSON
	schemaJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to marshal schema: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("Tool Schema for '%s':\n\n%s", toolName, string(schemaJSON))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, metadata, nil
}

// HandleExecuteTool handles the execute_tool tool call
func (h *ToolsDiscoveryHandler) HandleExecuteTool(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract toolName (required)
	toolName, ok := args["toolName"].(string)
	if !ok || toolName == "" {
		return createErrorResult("toolName parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract args (required)
	toolArgs, ok := args["args"].(map[string]interface{})
	if !ok {
		return createErrorResult("args parameter is required and must be a JSON object"), nil, nil
	}

	// Call the tool via MCP HTTP bridge
	// The bridge expects: POST to /api/mcp/tools/call with body: {"name": "toolName", "arguments": {...}}
	bridgeURL := "http://localhost:7095/api/mcp/tools/call"

	requestBody := map[string]interface{}{
		"name":      toolName,
		"arguments": toolArgs,
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to marshal request: %s", err.Error())), nil, nil
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", bridgeURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to create request: %s", err.Error())), nil, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", fmt.Sprintf("execute_tool_%d", time.Now().UnixNano()))

	// Execute request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to execute tool: %s", err.Error())), nil, nil
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to read response: %s", err.Error())), nil, nil
	}

	// Check for HTTP error
	if resp.StatusCode != http.StatusOK {
		return createErrorResult(fmt.Sprintf("tool execution failed (status %d): %s", resp.StatusCode, string(respBody))), nil, nil
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return createErrorResult(fmt.Sprintf("failed to parse response: %s", err.Error())), nil, nil
	}

	// Format result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to marshal result: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("Tool '%s' executed successfully:\n\n%s", toolName, string(resultJSON))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, result, nil
}

// registerMCPAddServer registers the mcp_add_server tool
func (h *ToolsDiscoveryHandler) registerMCPAddServer(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "mcp_add_server",
		Description: "Add a new MCP server to the registry, discover its tools, and store them in MongoDB and Qdrant for semantic search. The server must be accessible via HTTP/HTTPS and expose the MCP protocol.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"serverName": {
					Type:        "string",
					Description: "Unique name for this MCP server (e.g., 'openai-mcp', 'github-mcp')",
				},
				"serverUrl": {
					Type:        "string",
					Description: "HTTP/HTTPS URL of the MCP server (e.g., 'http://localhost:3000/mcp')",
				},
				"description": {
					Type:        "string",
					Description: "Human-readable description of what this server provides",
				},
				"headers": {
					Type:        "object",
					Description: "Optional HTTP headers to send with MCP requests (e.g., {\"Authorization\": \"Bearer token\"})",
				},
			},
			Required: []string{"serverName", "serverUrl"},
		},
	}

	h.addToolWithMetadata(server, tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.HandleMCPAddServer(ctx, args)
		return result, err
	})

	return nil
}

// registerMCPRediscoverServer registers the mcp_rediscover_server tool
func (h *ToolsDiscoveryHandler) registerMCPRediscoverServer(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "mcp_rediscover_server",
		Description: "Rediscover and refresh tools from an existing MCP server. This removes old tools and discovers the current set of tools available on the server.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"serverName": {
					Type:        "string",
					Description: "Name of the MCP server to rediscover (must already be registered)",
				},
			},
			Required: []string{"serverName"},
		},
	}

	h.addToolWithMetadata(server, tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.HandleMCPRediscoverServer(ctx, args)
		return result, err
	})

	return nil
}

// registerMCPRemoveServer registers the mcp_remove_server tool
func (h *ToolsDiscoveryHandler) registerMCPRemoveServer(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "mcp_remove_server",
		Description: "Remove an MCP server and all its tools from the registry. This deletes the server metadata and all associated tool data from MongoDB and Qdrant.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"serverName": {
					Type:        "string",
					Description: "Name of the MCP server to remove",
				},
			},
			Required: []string{"serverName"},
		},
	}

	h.addToolWithMetadata(server, tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.HandleMCPRemoveServer(ctx, args)
		return result, err
	})

	return nil
}

// HandleMCPAddServer handles the mcp_add_server tool call
func (h *ToolsDiscoveryHandler) HandleMCPAddServer(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract serverName (required)
	serverName, ok := args["serverName"].(string)
	if !ok || serverName == "" {
		return createErrorResult("serverName parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract serverUrl (required)
	serverURL, ok := args["serverUrl"].(string)
	if !ok || serverURL == "" {
		return createErrorResult("serverUrl parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract description (optional)
	description, _ := args["description"].(string)
	if description == "" {
		description = fmt.Sprintf("MCP server at %s", serverURL)
	}

	// Extract headers (optional)
	headers, _ := args["headers"].(map[string]interface{})

	// Add server to storage
	if err := h.toolsStorage.AddServer(ctx, serverName, serverURL, description); err != nil {
		return createErrorResult(fmt.Sprintf("failed to add server: %s", err.Error())), nil, nil
	}

	// Discover tools from the server
	tools, err := h.discoverServerTools(ctx, serverURL, headers)
	if err != nil {
		return createErrorResult(fmt.Sprintf("server added but tool discovery failed: %s", err.Error())), nil, nil
	}

	// Store each tool
	successCount := 0
	for _, tool := range tools {
		toolName := tool["name"].(string)
		desc, _ := tool["description"].(string)
		schema, _ := tool["inputSchema"].(map[string]interface{})

		if err := h.toolsStorage.StoreToolMetadata(ctx, toolName, desc, schema, serverName); err != nil {
			fmt.Printf("Warning: failed to store tool %s: %v\n", toolName, err)
			continue
		}
		successCount++
	}

	resultText := fmt.Sprintf("Server '%s' added successfully!\n\nDiscovered %d tools, stored %d tools.\nServer URL: %s\nDescription: %s",
		serverName, len(tools), successCount, serverURL, description)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"serverName":  serverName,
		"toolCount":   successCount,
		"totalTools":  len(tools),
	}, nil
}

// HandleMCPRediscoverServer handles the mcp_rediscover_server tool call
func (h *ToolsDiscoveryHandler) HandleMCPRediscoverServer(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract serverName (required)
	serverName, ok := args["serverName"].(string)
	if !ok || serverName == "" {
		return createErrorResult("serverName parameter is required and must be a non-empty string"), nil, nil
	}

	// Get server metadata
	server, err := h.toolsStorage.GetServer(ctx, serverName)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to get server: %s", err.Error())), nil, nil
	}

	// Remove old tools
	if err := h.toolsStorage.RemoveServerTools(ctx, serverName); err != nil {
		return createErrorResult(fmt.Sprintf("failed to remove old tools: %s", err.Error())), nil, nil
	}

	// Discover new tools from the server
	tools, err := h.discoverServerTools(ctx, server.ServerURL, nil)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to discover tools: %s", err.Error())), nil, nil
	}

	// Store each tool
	successCount := 0
	for _, tool := range tools {
		toolName := tool["name"].(string)
		desc, _ := tool["description"].(string)
		schema, _ := tool["inputSchema"].(map[string]interface{})

		if err := h.toolsStorage.StoreToolMetadata(ctx, toolName, desc, schema, serverName); err != nil {
			fmt.Printf("Warning: failed to store tool %s: %v\n", toolName, err)
			continue
		}
		successCount++
	}

	resultText := fmt.Sprintf("Server '%s' rediscovered successfully!\n\nDiscovered %d tools, stored %d tools.\nServer URL: %s",
		serverName, len(tools), successCount, server.ServerURL)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"serverName":  serverName,
		"toolCount":   successCount,
		"totalTools":  len(tools),
	}, nil
}

// HandleMCPRemoveServer handles the mcp_remove_server tool call
func (h *ToolsDiscoveryHandler) HandleMCPRemoveServer(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract serverName (required)
	serverName, ok := args["serverName"].(string)
	if !ok || serverName == "" {
		return createErrorResult("serverName parameter is required and must be a non-empty string"), nil, nil
	}

	// Get server metadata first (to show in result)
	server, err := h.toolsStorage.GetServer(ctx, serverName)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to get server: %s", err.Error())), nil, nil
	}

	// Remove all tools for this server
	if err := h.toolsStorage.RemoveServerTools(ctx, serverName); err != nil {
		return createErrorResult(fmt.Sprintf("failed to remove server tools: %s", err.Error())), nil, nil
	}

	// Remove server from registry
	if err := h.toolsStorage.RemoveServer(ctx, serverName); err != nil {
		return createErrorResult(fmt.Sprintf("failed to remove server: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("Server '%s' removed successfully!\n\nServer URL: %s\nAll tools and metadata deleted from MongoDB and Qdrant.",
		serverName, server.ServerURL)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"serverName": serverName,
		"removed":    true,
	}, nil
}

// discoverServerTools connects to an MCP server and lists its tools
func (h *ToolsDiscoveryHandler) discoverServerTools(ctx context.Context, serverURL string, headers map[string]interface{}) ([]map[string]interface{}, error) {
	// Create MCP tools/list request
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", serverURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Apply custom headers if provided
	for key, val := range headers {
		if strVal, ok := val.(string); ok {
			req.Header.Set(key, strVal)
		}
	}

	// Execute request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP error
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse MCP response
	var mcpResponse struct {
		Result struct {
			Tools []map[string]interface{} `json:"tools"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &mcpResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for MCP error
	if mcpResponse.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", mcpResponse.Error.Code, mcpResponse.Error.Message)
	}

	return mcpResponse.Result.Tools, nil
}
