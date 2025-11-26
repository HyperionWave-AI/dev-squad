package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// MCPToolAdapter wraps an MCP tool for direct execution via ToolRegistry
// It implements the aiservice.ToolExecutor interface to allow MCP tools
// from registered servers to be called directly by the AI without going
// through discover_tools + execute_tool meta-tools
type MCPToolAdapter struct {
	toolName    string                 // Original tool name from MCP server
	description string                 // Tool description
	schema      map[string]interface{} // JSON schema for tool parameters
	serverName  string                 // Name of the MCP server this tool belongs to
	serverURL   string                 // URL of the MCP server
	headers     map[string]interface{} // Optional headers for authentication
	logger      *zap.Logger
}

// NewMCPToolAdapter creates a new adapter for an MCP tool
func NewMCPToolAdapter(
	toolName string,
	description string,
	schema map[string]interface{},
	serverName string,
	serverURL string,
	headers map[string]interface{},
	logger *zap.Logger,
) *MCPToolAdapter {
	return &MCPToolAdapter{
		toolName:    toolName,
		description: description,
		schema:      schema,
		serverName:  serverName,
		serverURL:   serverURL,
		headers:     headers,
		logger:      logger,
	}
}

// Name returns the namespaced tool name: mcp_{serverName}_{toolName}
// IMPORTANT: Use single underscores only - double underscores are rejected by ToolRegistry validation
func (a *MCPToolAdapter) Name() string {
	// Sanitize server name and tool name to ensure valid tool names
	// Replace any double underscores, hyphens, or spaces with single underscores
	sanitizedServer := sanitizeToolNamePart(a.serverName)
	sanitizedTool := sanitizeToolNamePart(a.toolName)
	return fmt.Sprintf("mcp_%s_%s", sanitizedServer, sanitizedTool)
}

// sanitizeToolNamePart sanitizes a string for use in tool names
// Converts to lowercase, replaces special characters with underscores,
// and removes consecutive underscores
func sanitizeToolNamePart(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace hyphens and spaces with underscores
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")

	// Remove consecutive underscores by iterating until no doubles remain
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}

	// Trim leading/trailing underscores
	s = strings.Trim(s, "_")

	return s
}

// Description returns the tool description
func (a *MCPToolAdapter) Description() string {
	return a.description
}

// InputSchema returns the JSON schema for tool parameters
func (a *MCPToolAdapter) InputSchema() map[string]interface{} {
	return a.schema
}

// Execute calls the tool on the remote MCP server
func (a *MCPToolAdapter) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	a.logger.Info("Executing MCP tool via adapter",
		zap.String("toolName", a.toolName),
		zap.String("serverName", a.serverName),
		zap.String("serverURL", a.serverURL))

	// Create JSON-RPC request for the MCP tool call
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      a.toolName,
			"arguments": input,
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", a.serverURL, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range a.headers {
		if strValue, ok := value.(string); ok {
			req.Header.Set(key, strValue)
		}
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request to MCP server %s: %w", a.serverName, err)
	}
	defer resp.Body.Close()

	// Check HTTP status code before attempting JSON decoding
	if resp.StatusCode != http.StatusOK {
		// Read response body for error details
		bodyBytes, readErr := io.ReadAll(resp.Body)
		bodyPreview := string(bodyBytes)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500] + "...[truncated]"
		}
		if readErr != nil {
			return nil, fmt.Errorf("HTTP %d error from MCP server %s (failed to read response body: %w)", resp.StatusCode, a.serverName, readErr)
		}
		return nil, fmt.Errorf("HTTP %d error from MCP server %s: %s", resp.StatusCode, a.serverName, bodyPreview)
	}

	// Parse JSON-RPC response
	var rpcResp struct {
		JSONRPC string                 `json:"jsonrpc"`
		ID      int                    `json:"id"`
		Result  map[string]interface{} `json:"result,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON-RPC response from %s: %w", a.serverName, err)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP tool %s error: %s (code %d)", a.toolName, rpcResp.Error.Message, rpcResp.Error.Code)
	}

	// Extract content from result
	// MCP tools return results in the format: { "content": [...], "isError": bool }
	if content, ok := rpcResp.Result["content"]; ok {
		// Try to extract text content from content array
		if contentArr, ok := content.([]interface{}); ok && len(contentArr) > 0 {
			// Return the first text content
			for _, item := range contentArr {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						// Try to parse as JSON for structured data
						var jsonResult interface{}
						if err := json.Unmarshal([]byte(text), &jsonResult); err == nil {
							a.logger.Debug("MCP tool returned JSON result",
								zap.String("toolName", a.toolName))
							return jsonResult, nil
						}
						// Return as plain text
						return text, nil
					}
				}
			}
		}
	}

	// Return the full result if content extraction fails
	a.logger.Debug("MCP tool completed",
		zap.String("toolName", a.toolName),
		zap.String("serverName", a.serverName))

	return rpcResp.Result, nil
}

// GetServerName returns the server name this tool belongs to
func (a *MCPToolAdapter) GetServerName() string {
	return a.serverName
}

// GetOriginalToolName returns the original tool name (without namespace prefix)
func (a *MCPToolAdapter) GetOriginalToolName() string {
	return a.toolName
}
