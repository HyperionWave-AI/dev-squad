package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"hyperion-coordinator-mcp/storage"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolsDiscoveryHandler manages MCP tools discovery operations
type ToolsDiscoveryHandler struct {
	toolsStorage storage.ToolsStorageInterface
	httpClient   *http.Client
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

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleDiscoverTools(ctx, args)
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

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleGetToolSchema(ctx, args)
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

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleExecuteTool(ctx, args)
		return result, err
	})

	return nil
}

// handleDiscoverTools handles the discover_tools tool call
func (h *ToolsDiscoveryHandler) handleDiscoverTools(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
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

// handleGetToolSchema handles the get_tool_schema tool call
func (h *ToolsDiscoveryHandler) handleGetToolSchema(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
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

// handleExecuteTool handles the execute_tool tool call
func (h *ToolsDiscoveryHandler) handleExecuteTool(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
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
