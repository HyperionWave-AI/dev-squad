package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// MCPRegistrySyncInterface defines the interface for syncing MCP tools to the ToolRegistry
// This allows the ToolsDiscoveryHandler to refresh the registry when servers are added/removed
type MCPRegistrySyncInterface interface {
	SyncServerTools(ctx context.Context, serverName string) (int, error)
	RemoveServerTools(serverName string) error
}

// ToolsDiscoveryHandler manages MCP tools discovery operations
type ToolsDiscoveryHandler struct {
	toolsStorage     storage.ToolsStorageInterface
	metadataRegistry *ToolMetadataRegistry
	mcpServer        *mcp.Server
	registrySync     MCPRegistrySyncInterface
	logger           *zap.Logger
}

// headerRoundTripper is a custom http.RoundTripper that adds headers to every request
type headerRoundTripper struct {
	headers map[string]interface{}
	base    http.RoundTripper
}

// RoundTrip implements http.RoundTripper interface
func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	reqCopy := req.Clone(req.Context())

	// Add custom headers with special handling for certain headers
	for key, value := range h.headers {
		if strValue, ok := value.(string); ok {
			canonicalKey := strings.ToLower(key)

			// Special handling for Accept header: merge with existing values
			// The MCP SDK sets "Accept: application/json, text/event-stream" which is required
			// by the MCP protocol. If custom headers include Accept, we need to merge values
			// rather than overwrite to preserve SDK's required accept types.
			if canonicalKey == "accept" {
				existingAccept := reqCopy.Header.Get("Accept")
				if existingAccept != "" {
					// Parse existing accept values
					existingValues := parseAcceptValues(existingAccept)
					customValues := parseAcceptValues(strValue)

					// Merge: add custom values that aren't already present
					merged := existingValues
					for _, cv := range customValues {
						found := false
						for _, ev := range existingValues {
							if strings.EqualFold(ev, cv) {
								found = true
								break
							}
						}
						if !found {
							merged = append(merged, cv)
						}
					}
					reqCopy.Header.Set("Accept", strings.Join(merged, ", "))
				} else {
					reqCopy.Header.Set(key, strValue)
				}
				continue
			}

			// For all other headers, use Set (replace)
			reqCopy.Header.Set(key, strValue)
		}
	}

	// Use base transport to execute the request
	return h.base.RoundTrip(reqCopy)
}

// parseAcceptValues parses an Accept header value into individual media types
func parseAcceptValues(accept string) []string {
	var values []string
	for _, part := range strings.Split(accept, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

// createCustomHTTPClient creates an http.Client with custom headers support
func createCustomHTTPClient(headers map[string]interface{}) *http.Client {
	if len(headers) == 0 {
		return http.DefaultClient
	}

	return &http.Client{
		Transport: &headerRoundTripper{
			headers: headers,
			base:    http.DefaultTransport,
		},
	}
}

// normalizeToolName extracts the original tool name from various naming conventions
// It handles:
// - "mcp_hyperion_{serverName}_{toolName}" -> "{toolName}"
// - "mcp_{serverName}_{toolName}" -> "{toolName}"
// - "{toolName}" -> "{toolName}" (unchanged)
//
// The function tries to extract the last part after known server name patterns
func normalizeToolName(toolName string) string {
	// If it doesn't start with "mcp_", return as-is
	if !strings.HasPrefix(toolName, "mcp_") {
		return toolName
	}

	// Remove "mcp_hyperion_" prefix if present (legacy naming)
	name := toolName
	if strings.HasPrefix(name, "mcp_hyperion_") {
		name = strings.TrimPrefix(name, "mcp_hyperion_")
	} else {
		// Remove "mcp_" prefix
		name = strings.TrimPrefix(name, "mcp_")
	}

	// Now we have "{serverName}_{toolName}"
	// Common server name patterns to strip (they often end with "_mcp" or "_api")
	// Examples:
	// - "black_forest_labs_mcp_bfl_text_to_image" -> serverName="black_forest_labs_mcp", toolName="bfl_text_to_image"
	// - "google_mcp_google_generate_image" -> serverName="google_mcp", toolName="google_generate_image"
	// - "hedra_mcp_hedra_create_video" -> serverName="hedra_mcp", toolName="hedra_create_video"
	// - "storage_api_list_directory" -> serverName="storage_api", toolName="list_directory"

	// Look for "_mcp_" which separates serverName from toolName
	if idx := strings.Index(name, "_mcp_"); idx != -1 {
		// Return everything after "_mcp_"
		return name[idx+5:] // Skip "_mcp_"
	}

	// Look for "_api_" which separates serverName from toolName (e.g., storage_api_list_directory)
	if idx := strings.Index(name, "_api_"); idx != -1 {
		// Return everything after "_api_"
		return name[idx+5:] // Skip "_api_"
	}

	// If no pattern found, return the name as-is (might be original tool name)
	return name
}

// truncateJSON safely truncates JSON to maxBytes for logging previews
func truncateJSON(data interface{}, maxBytes int) string {
	if data == nil {
		return "{}"
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("{error marshaling: %v}", err)
	}

	if len(jsonBytes) <= maxBytes {
		return string(jsonBytes)
	}

	truncated := string(jsonBytes[:maxBytes])
	return truncated + "...[truncated]"
}

func isExecuteToolDebugEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("MCP_EXECUTE_TOOL_DEBUG")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func debugPrintln(enabled bool, args ...interface{}) {
	if enabled {
		fmt.Println(args...)
	}
}

func debugPrintf(enabled bool, format string, args ...interface{}) {
	if enabled {
		fmt.Printf(format, args...)
	}
}

func isSensitiveHeaderKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	sensitivePatterns := []string{
		"authorization",
		"proxy-authorization",
		"token",
		"api-key",
		"apikey",
		"secret",
		"cookie",
		"set-cookie",
		"session",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

func maskHeaderValue(key, value string) string {
	if isSensitiveHeaderKey(key) {
		return "[REDACTED]"
	}

	return value
}

// legacyJSONRPCRequest represents a JSON-RPC 2.0 request for legacy MCP servers
type legacyJSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// legacyJSONRPCResponse represents a JSON-RPC 2.0 response from legacy MCP servers
type legacyJSONRPCResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// discoverLegacyTools discovers tools from a legacy MCP server using direct JSON-RPC
func (h *ToolsDiscoveryHandler) discoverLegacyTools(ctx context.Context, serverURL string, headers map[string]interface{}) ([]map[string]interface{}, error) {
	h.logger.Info("Using legacy MCP protocol (stateless JSON-RPC)",
		zap.String("serverURL", serverURL),
		zap.String("method", "tools/list"))

	// Create JSON-RPC request
	reqBody := legacyJSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", serverURL, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if strValue, ok := value.(string); ok {
			req.Header.Set(key, strValue)
		}
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
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
			return nil, fmt.Errorf("HTTP %d error from MCP server (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("HTTP %d error from MCP server: %s", resp.StatusCode, bodyPreview)
	}

	// Parse response
	var rpcResp legacyJSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		// Read body for better error message
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyPreview := string(bodyBytes)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "...[truncated]"
		}
		return nil, fmt.Errorf("failed to decode JSON-RPC response: %w (response body: %s)", err, bodyPreview)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: %s (code %d)", rpcResp.Error.Message, rpcResp.Error.Code)
	}

	// Extract tools from result
	toolsList, ok := rpcResp.Result["tools"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: 'tools' field not found or wrong type")
	}

	// Convert to map format
	tools := make([]map[string]interface{}, 0, len(toolsList))
	for _, toolItem := range toolsList {
		if toolMap, ok := toolItem.(map[string]interface{}); ok {
			tools = append(tools, toolMap)
		}
	}

	h.logger.Info("Legacy tool discovery successful",
		zap.String("serverURL", serverURL),
		zap.Int("toolCount", len(tools)))

	return tools, nil
}

// discoverLegacyResources discovers resources from a legacy MCP server using direct JSON-RPC
func (h *ToolsDiscoveryHandler) discoverLegacyResources(ctx context.Context, serverURL string, headers map[string]interface{}) ([]map[string]interface{}, error) {
	h.logger.Info("Using legacy MCP protocol (stateless JSON-RPC)",
		zap.String("serverURL", serverURL),
		zap.String("method", "resources/list"))

	// Create JSON-RPC request
	reqBody := legacyJSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "resources/list",
		Params:  map[string]interface{}{},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", serverURL, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if strValue, ok := value.(string); ok {
			req.Header.Set(key, strValue)
		}
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
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
			return nil, fmt.Errorf("HTTP %d error from MCP server (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("HTTP %d error from MCP server: %s", resp.StatusCode, bodyPreview)
	}

	// Parse response
	var rpcResp legacyJSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		// Read body for better error message
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyPreview := string(bodyBytes)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "...[truncated]"
		}
		return nil, fmt.Errorf("failed to decode JSON-RPC response: %w (response body: %s)", err, bodyPreview)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: %s (code %d)", rpcResp.Error.Message, rpcResp.Error.Code)
	}

	// Extract resources from result
	resourcesList, ok := rpcResp.Result["resources"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: 'resources' field not found or wrong type")
	}

	// Convert to map format
	resources := make([]map[string]interface{}, 0, len(resourcesList))
	for _, resourceItem := range resourcesList {
		if resourceMap, ok := resourceItem.(map[string]interface{}); ok {
			resources = append(resources, resourceMap)
		}
	}

	h.logger.Info("Legacy resource discovery successful",
		zap.String("serverURL", serverURL),
		zap.Int("resourceCount", len(resources)))

	return resources, nil
}

// discoverLegacyPrompts discovers prompts from a legacy MCP server using direct JSON-RPC
func (h *ToolsDiscoveryHandler) discoverLegacyPrompts(ctx context.Context, serverURL string, headers map[string]interface{}) ([]map[string]interface{}, error) {
	h.logger.Info("Using legacy MCP protocol (stateless JSON-RPC)",
		zap.String("serverURL", serverURL),
		zap.String("method", "prompts/list"))

	// Create JSON-RPC request
	reqBody := legacyJSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "prompts/list",
		Params:  map[string]interface{}{},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", serverURL, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if strValue, ok := value.(string); ok {
			req.Header.Set(key, strValue)
		}
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
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
			return nil, fmt.Errorf("HTTP %d error from MCP server (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("HTTP %d error from MCP server: %s", resp.StatusCode, bodyPreview)
	}

	// Parse response
	var rpcResp legacyJSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		// Read body for better error message
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyPreview := string(bodyBytes)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "...[truncated]"
		}
		return nil, fmt.Errorf("failed to decode JSON-RPC response: %w (response body: %s)", err, bodyPreview)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: %s (code %d)", rpcResp.Error.Message, rpcResp.Error.Code)
	}

	// Extract prompts from result
	promptsList, ok := rpcResp.Result["prompts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: 'prompts' field not found or wrong type")
	}

	// Convert to map format
	prompts := make([]map[string]interface{}, 0, len(promptsList))
	for _, promptItem := range promptsList {
		if promptMap, ok := promptItem.(map[string]interface{}); ok {
			prompts = append(prompts, promptMap)
		}
	}

	h.logger.Info("Legacy prompt discovery successful",
		zap.String("serverURL", serverURL),
		zap.Int("promptCount", len(prompts)))

	return prompts, nil
}

// isSessionError checks if an error is related to session management
func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "session not found") ||
		strings.Contains(errMsg, "session") && strings.Contains(errMsg, "failed")
}

// NewToolsDiscoveryHandler creates a new tools discovery handler
func NewToolsDiscoveryHandler(toolsStorage *storage.ToolsStorage, mcpServer *mcp.Server, logger *zap.Logger) *ToolsDiscoveryHandler {
	return &ToolsDiscoveryHandler{
		toolsStorage: toolsStorage,
		mcpServer:    mcpServer,
		logger:       logger,
	}
}

// SetMetadataRegistry sets the metadata registry for tool indexing
func (h *ToolsDiscoveryHandler) SetMetadataRegistry(registry *ToolMetadataRegistry) {
	h.metadataRegistry = registry
}

// SetRegistrySync sets the registry sync service for auto-registering MCP tools
func (h *ToolsDiscoveryHandler) SetRegistrySync(sync MCPRegistrySyncInterface) {
	h.registrySync = sync
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
		result, extractedData, err := h.HandleDiscoverTools(ctx, args)
		if err != nil {
			return result, err
		}
		// Wrap array results in an object for MCP protocol compliance
		// StructuredContent must be an object, not an array
		if extractedData != nil {
			result.StructuredContent = map[string]interface{}{"tools": extractedData}
		}
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
		result, extractedData, err := h.HandleGetToolSchema(ctx, args)
		if err != nil {
			return result, err
		}
		// MCP protocol requires StructuredContent to be an object (map)
		// Wrap non-object types in {"data": ...}
		if extractedData != nil {
			switch extractedData.(type) {
			case map[string]interface{}:
				// Already an object, use as-is
				result.StructuredContent = extractedData
			default:
				// Wrap primitives, arrays, etc. in an object
				result.StructuredContent = map[string]interface{}{"data": extractedData}
			}
		}
		return result, err
	})

	return nil
}

// registerExecuteTool registers the execute_tool tool
func (h *ToolsDiscoveryHandler) registerExecuteTool(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "execute_tool",
		Description: "Execute an MCP tool by name with specified arguments. This tool looks up the tool's server from the registry and makes an HTTP call to that server's MCP endpoint. Works with external MCP servers registered via mcp_add_server. Built-in tools cannot be executed via this tool. Use get_tool_schema first to understand required parameters.",
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
		result, extractedData, err := h.HandleExecuteTool(ctx, args)
		if err != nil {
			return result, err
		}
		// MCP protocol requires StructuredContent to be an object (map)
		// Wrap non-object types in {"data": ...}
		if extractedData != nil {
			switch extractedData.(type) {
			case map[string]interface{}:
				// Already an object, use as-is
				result.StructuredContent = extractedData
			default:
				// Wrap primitives, arrays, etc. in an object
				result.StructuredContent = map[string]interface{}{"data": extractedData}
			}
		}
		h.logger.Debug("execute_tool registration: populated StructuredContent",
			zap.String("preview", truncateJSON(result.StructuredContent, 200)))
		return result, err
	})

	return nil
}

// HandleDiscoverTools handles the discover_tools tool call
func (h *ToolsDiscoveryHandler) HandleDiscoverTools(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	h.logger.Info("discover_tools request started",
		zap.String("argsPreview", truncateJSON(args, 200)))

	// Extract query (required)
	query, ok := args["query"].(string)
	if !ok || query == "" {
		h.logger.Warn("discover_tools failed: missing or invalid query parameter")
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

	h.logger.Info("searching for tools",
		zap.String("query", query),
		zap.Int("limit", limit))

	// Search for tools
	matches, err := h.toolsStorage.SearchTools(ctx, query, limit)
	if err != nil {
		h.logger.Error("discover_tools failed",
			zap.String("query", query),
			zap.Int("limit", limit),
			zap.Error(err))
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

	h.logger.Info("discover_tools completed successfully",
		zap.String("query", query),
		zap.Int("matchCount", len(matches)),
		zap.String("resultsPreview", truncateJSON(matches, 200)))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, matches, nil
}

// HandleGetToolSchema handles the get_tool_schema tool call
func (h *ToolsDiscoveryHandler) HandleGetToolSchema(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	h.logger.Info("get_tool_schema request started",
		zap.String("argsPreview", truncateJSON(args, 200)))

	// Extract toolName (required)
	toolName, ok := args["toolName"].(string)
	if !ok || toolName == "" {
		h.logger.Warn("get_tool_schema failed: missing or invalid toolName parameter")
		return createErrorResult("toolName parameter is required and must be a non-empty string"), nil, nil
	}

	h.logger.Info("fetching tool schema",
		zap.String("toolName", toolName))

	// Get tool schema
	metadata, err := h.toolsStorage.GetToolSchema(ctx, toolName)
	if err != nil {
		h.logger.Error("get_tool_schema failed",
			zap.String("toolName", toolName),
			zap.Error(err))
		return createErrorResult(fmt.Sprintf("failed to get tool schema: %s", err.Error())), nil, nil
	}

	// Format schema as JSON
	schemaJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to marshal schema: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("Tool Schema for '%s':\n\n%s", toolName, string(schemaJSON))

	h.logger.Info("get_tool_schema completed successfully",
		zap.String("toolName", toolName),
		zap.String("schemaPreview", truncateJSON(metadata, 200)))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, metadata, nil
}

// HandleExecuteTool handles the execute_tool tool call
func (h *ToolsDiscoveryHandler) HandleExecuteTool(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	debugEnabled := isExecuteToolDebugEnabled()

	// ============================================================
	// 🔍 EXECUTE_TOOL DEBUG START
	// ============================================================
	debugPrintln(debugEnabled, "\n"+strings.Repeat("=", 80))
	debugPrintln(debugEnabled, "🔍 EXECUTE_TOOL DEBUG - REQUEST RECEIVED")
	debugPrintln(debugEnabled, strings.Repeat("=", 80))

	argsJSON, _ := json.MarshalIndent(args, "", "  ")
	debugPrintf(debugEnabled, "📥 RAW ARGS:\n%s\n", string(argsJSON))

	h.logger.Info("execute_tool request started",
		zap.String("argsPreview", truncateJSON(args, 200)))

	// Extract toolName (required)
	toolName, ok := args["toolName"].(string)
	if !ok || toolName == "" {
		debugPrintln(debugEnabled, "❌ ERROR: Missing or invalid toolName parameter")
		h.logger.Warn("execute_tool failed: missing or invalid toolName parameter")
		return createErrorResult("toolName parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract args (required)
	toolArgs, ok := args["args"].(map[string]interface{})
	if !ok {
		debugPrintf(debugEnabled, "❌ ERROR: Missing or invalid args parameter for tool: %s\n", toolName)
		h.logger.Warn("execute_tool failed: missing or invalid args parameter",
			zap.String("toolName", toolName))
		return createErrorResult("args parameter is required and must be a JSON object"), nil, nil
	}

	toolArgsJSON, _ := json.MarshalIndent(toolArgs, "", "  ")
	debugPrintf(debugEnabled, "🔧 TOOL NAME (original): %s\n", toolName)
	debugPrintf(debugEnabled, "📋 TOOL ARGS:\n%s\n", string(toolArgsJSON))

	// Normalize tool name - handle different naming conventions
	// Tools can be called with:
	// 1. Original name: "bfl_text_to_image"
	// 2. Namespaced (ToolRegistry): "mcp_black_forest_labs_mcp_bfl_text_to_image"
	// 3. Hyperion-prefixed (legacy): "mcp_hyperion_black_forest_labs_mcp_bfl_text_to_image"
	normalizedToolName := normalizeToolName(toolName)
	debugPrintf(debugEnabled, "🔧 TOOL NAME (normalized): %s\n", normalizedToolName)

	h.logger.Info("executing tool",
		zap.String("toolName", toolName),
		zap.String("normalizedToolName", normalizedToolName),
		zap.String("toolArgsPreview", truncateJSON(toolArgs, 200)))

	// Look up the tool metadata to find which server it belongs to
	// Try multiple name variants in order of likelihood
	var toolMetadata *storage.ToolMetadata
	var err error
	triedNames := []string{}

	// Build list of name variants to try
	nameVariants := []string{normalizedToolName}
	if normalizedToolName != toolName {
		nameVariants = append(nameVariants, toolName)
	}

	// Also try with server name prefix stripped differently
	// e.g., "storage_api_list_files" -> try "storage_api_list_files" as well
	if strings.HasPrefix(toolName, "mcp_hyperion_") {
		withoutHyperion := strings.TrimPrefix(toolName, "mcp_hyperion_")
		if withoutHyperion != normalizedToolName && withoutHyperion != toolName {
			nameVariants = append(nameVariants, withoutHyperion)
		}
	} else if strings.HasPrefix(toolName, "mcp_") {
		withoutMcp := strings.TrimPrefix(toolName, "mcp_")
		if withoutMcp != normalizedToolName && withoutMcp != toolName {
			nameVariants = append(nameVariants, withoutMcp)
		}
	}

	// Try each variant
	for _, variant := range nameVariants {
		triedNames = append(triedNames, variant)
		debugPrintf(debugEnabled, "   Trying tool name variant: %s\n", variant)
		toolMetadata, err = h.toolsStorage.GetToolSchema(ctx, variant)
		if err == nil {
			debugPrintf(debugEnabled, "   ✅ Found tool with name: %s\n", variant)
			break
		}
	}

	if err != nil {
		debugPrintf(debugEnabled, "❌ ERROR: Tool not found. Tried names: %v\n", triedNames)
		h.logger.Error("execute_tool failed: tool not found",
			zap.String("toolName", toolName),
			zap.String("normalizedToolName", normalizedToolName),
			zap.Strings("triedNames", triedNames),
			zap.Error(err))
		return createErrorResult(fmt.Sprintf("tool not found: %s (tried: %v)", toolName, triedNames)), nil, nil
	}

	debugPrintf(debugEnabled, "📦 TOOL METADATA:\n")
	debugPrintf(debugEnabled, "   - Server Name: %s\n", toolMetadata.ServerName)
	debugPrintf(debugEnabled, "   - Tool Description: %s\n", truncateJSON(toolMetadata.Description, 100))

	// Check if this is a built-in tool (mcp-builtin server)
	if toolMetadata.ServerName == "mcp-builtin" {
		debugPrintln(debugEnabled, "❌ ERROR: Cannot execute built-in tool via execute_tool")
		return createErrorResult(fmt.Sprintf(
			"Tool '%s' is a built-in tool and cannot be executed via execute_tool. "+
				"Built-in tools are directly available in your MCP client.",
			toolName,
		)), nil, nil
	}

	// Get the server metadata to find the server URL and headers
	serverMetadata, err := h.toolsStorage.GetServer(ctx, toolMetadata.ServerName)
	if err != nil {
		debugPrintf(debugEnabled, "❌ ERROR: Server not found: %s - %v\n", toolMetadata.ServerName, err)
		h.logger.Error("execute_tool failed: server not found",
			zap.String("toolName", toolName),
			zap.String("serverName", toolMetadata.ServerName),
			zap.Error(err))
		return createErrorResult(fmt.Sprintf("failed to get server info: %s", err.Error())), nil, nil
	}

	debugPrintf(debugEnabled, "🌐 SERVER METADATA:\n")
	debugPrintf(debugEnabled, "   - Server URL: %s\n", serverMetadata.ServerURL)
	debugPrintf(debugEnabled, "   - Server Description: %s\n", serverMetadata.Description)

	// Log headers (mask sensitive values)
	debugPrintln(debugEnabled, "🔑 HEADERS CONFIGURED FOR SERVER:")
	if serverMetadata.Headers != nil {
		for key, value := range serverMetadata.Headers {
			if strVal, ok := value.(string); ok {
				debugPrintf(debugEnabled, "   - %s: %s\n", key, maskHeaderValue(key, strVal))
			} else {
				debugPrintf(debugEnabled, "   - %s: (non-string value: %T)\n", key, value)
			}
		}
	} else {
		debugPrintln(debugEnabled, "   (no custom headers)")
	}

	// Check context timeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		debugPrintf(debugEnabled, "⏱️  CONTEXT TIMEOUT: %v remaining (deadline: %v)\n", remaining.Round(time.Second), deadline.Format(time.RFC3339))
		if remaining < 5*time.Minute {
			debugPrintln(debugEnabled, "   ⚠️  WARNING: Context timeout may be too short for long-running tools like BFL image generation!")
		}
	} else {
		debugPrintln(debugEnabled, "⏱️  CONTEXT TIMEOUT: No deadline set")
	}

	debugPrintln(debugEnabled, strings.Repeat("-", 80))
	debugPrintln(debugEnabled, "📤 CALLING REMOTE MCP SERVER...")

	// IMPORTANT: Use the original tool name from metadata (as registered on the MCP server)
	// NOT the namespaced name that the AI uses (e.g., "mcp_hyperion_google_mcp_google_generate_image")
	// The MCP server only knows the original name (e.g., "google_generate_image")
	originalToolName := toolMetadata.ToolName
	debugPrintf(debugEnabled, "🔧 TOOL NAME (for MCP server): %s\n", originalToolName)

	h.logger.Info("calling tool on remote MCP server",
		zap.String("toolName", toolName),
		zap.String("originalToolName", originalToolName),
		zap.String("serverName", toolMetadata.ServerName),
		zap.String("serverURL", serverMetadata.ServerURL))

	// Execute the tool on the remote MCP server with custom headers
	// Use originalToolName (as registered on MCP server), not the namespaced toolName
	startTime := time.Now()
	result, err := h.executeToolOnServer(ctx, serverMetadata.ServerURL, serverMetadata.Headers, originalToolName, toolArgs, debugEnabled)
	elapsed := time.Since(startTime)

	debugPrintf(debugEnabled, "⏱️  EXECUTION TIME: %v\n", elapsed.Round(time.Millisecond))

	if err != nil {
		debugPrintf(debugEnabled, "❌ REMOTE EXECUTION ERROR: %v\n", err)
		debugPrintln(debugEnabled, strings.Repeat("=", 80))
		h.logger.Error("execute_tool failed: remote execution error",
			zap.String("toolName", toolName),
			zap.String("serverName", toolMetadata.ServerName),
			zap.String("serverURL", serverMetadata.ServerURL),
			zap.Error(err))
		return createErrorResult(fmt.Sprintf("failed to execute tool: %s", err.Error())), nil, nil
	}

	// Log raw result
	debugPrintln(debugEnabled, "📥 RAW RESULT FROM MCP SERVER:")
	if result != nil {
		debugPrintf(debugEnabled, "   - IsError: %v\n", result.IsError)
		debugPrintf(debugEnabled, "   - Content blocks: %d\n", len(result.Content))
		for i, content := range result.Content {
			debugPrintf(debugEnabled, "   - Content[%d] type: %T\n", i, content)
			if textContent, ok := content.(*mcp.TextContent); ok {
				preview := textContent.Text
				if len(preview) > 500 {
					preview = preview[:500] + "...[truncated]"
				}
				debugPrintf(debugEnabled, "     Text: %s\n", preview)
			} else if imgContent, ok := content.(*mcp.ImageContent); ok {
				debugPrintf(debugEnabled, "     ImageContent: mimeType=%s, dataLen=%d bytes\n", imgContent.MIMEType, len(imgContent.Data))
			} else if audioContent, ok := content.(*mcp.AudioContent); ok {
				debugPrintf(debugEnabled, "     AudioContent: mimeType=%s, dataLen=%d bytes\n", audioContent.MIMEType, len(audioContent.Data))
			}
		}
		if result.StructuredContent != nil {
			structuredJSON, _ := json.MarshalIndent(result.StructuredContent, "   ", "  ")
			debugPrintf(debugEnabled, "   - StructuredContent: %s\n", string(structuredJSON))
		} else {
			debugPrintln(debugEnabled, "   - StructuredContent: nil")
		}
	} else {
		debugPrintln(debugEnabled, "   ⚠️  RESULT IS NIL!")
	}

	h.logger.Info("execute_tool completed successfully",
		zap.String("toolName", toolName),
		zap.String("serverName", toolMetadata.ServerName),
		zap.String("resultPreview", truncateJSON(result, 200)))

	extractedData := extractResultData(h, result)

	debugPrintln(debugEnabled, "📤 EXTRACTED DATA FOR AI:")
	extractedJSON, _ := json.MarshalIndent(extractedData, "", "  ")
	extractedStr := string(extractedJSON)
	if len(extractedStr) > 1000 {
		extractedStr = extractedStr[:1000] + "...[truncated]"
	}
	debugPrintf(debugEnabled, "%s\n", extractedStr)
	debugPrintln(debugEnabled, strings.Repeat("=", 80))
	debugPrintln(debugEnabled, "🔍 EXECUTE_TOOL DEBUG END")
	debugPrintln(debugEnabled, strings.Repeat("=", 80)+"\n")

	h.logger.Info("execute_tool returning extracted data",
		zap.String("extractedDataPreview", truncateJSON(extractedData, 200)))

	return result, extractedData, nil
}

// stripMarkdownCodeFence removes markdown code fence wrappers from text
// Handles patterns: ```json\n...\n```, ```\n...\n```, or plain text
func stripMarkdownCodeFence(text string) string {
	text = strings.TrimSpace(text)

	// Check if text starts with ``` and ends with ```
	if !strings.HasPrefix(text, "```") || !strings.HasSuffix(text, "```") {
		return text // No markdown fence, return as-is
	}

	// Remove leading ```
	text = strings.TrimPrefix(text, "```")

	// Check for language identifier (json, yaml, etc.) and remove it
	if idx := strings.Index(text, "\n"); idx != -1 {
		text = text[idx+1:] // Skip first line (language identifier)
	}

	// Remove trailing ```
	text = strings.TrimSuffix(text, "```")

	return strings.TrimSpace(text)
}

// extractResultData extracts meaningful data from CallToolResult for AI consumption
// Priority: StructuredContent (if non-empty) > parsed Content[0].Text JSON > raw text
func extractResultData(h *ToolsDiscoveryHandler, result *mcp.CallToolResult) interface{} {
	if result == nil {
		h.logger.Debug("extractResultData: result is nil")
		return nil
	}

	h.logger.Debug("extractResultData: checking StructuredContent",
		zap.Any("StructuredContent", result.StructuredContent),
		zap.Int("ContentLength", len(result.Content)))

	// Check if StructuredContent has meaningful data (not empty object/array)
	if result.StructuredContent != nil {
		switch v := result.StructuredContent.(type) {
		case map[string]interface{}:
			h.logger.Debug("extractResultData: StructuredContent is map", zap.Int("mapLen", len(v)))
			if len(v) > 0 {
				h.logger.Info("extractResultData: returning non-empty StructuredContent map")
				return result.StructuredContent
			}
		case []interface{}:
			h.logger.Debug("extractResultData: StructuredContent is array", zap.Int("arrayLen", len(v)))
			if len(v) > 0 {
				h.logger.Info("extractResultData: returning non-empty StructuredContent array")
				return result.StructuredContent
			}
		default:
			h.logger.Debug("extractResultData: StructuredContent is other type", zap.String("type", fmt.Sprintf("%T", v)))
			// Non-empty primitive or other type
			return result.StructuredContent
		}
	}

	// Try to extract and parse Content blocks as JSON
	// Iterate through ALL Content blocks to find valid JSON data
	// PRIORITY: JSON objects (maps/arrays) > JSON primitives (strings/numbers) > raw text
	if len(result.Content) > 0 {
		h.logger.Debug("extractResultData: checking Content array",
			zap.Int("totalBlocks", len(result.Content)))

		var firstTextContent string
		var firstJSONPrimitive interface{}
		var mediaContents []map[string]interface{} // Collect image/audio content

		// Iterate through ALL Content blocks to prioritize objects over primitives
		for i, content := range result.Content {
			// Handle ImageContent (e.g., from image generation tools)
			if imageContent, ok := content.(*mcp.ImageContent); ok {
				h.logger.Info("extractResultData: found ImageContent block",
					zap.Int("blockIndex", i),
					zap.String("mimeType", imageContent.MIMEType),
					zap.Int("dataLength", len(imageContent.Data)))

				mediaContents = append(mediaContents, map[string]interface{}{
					"type":     "image",
					"mimeType": imageContent.MIMEType,
					"data":     base64.StdEncoding.EncodeToString(imageContent.Data),
				})
				continue
			}

			// Handle AudioContent (e.g., from audio generation tools)
			if audioContent, ok := content.(*mcp.AudioContent); ok {
				h.logger.Info("extractResultData: found AudioContent block",
					zap.Int("blockIndex", i),
					zap.String("mimeType", audioContent.MIMEType),
					zap.Int("dataLength", len(audioContent.Data)))

				mediaContents = append(mediaContents, map[string]interface{}{
					"type":     "audio",
					"mimeType": audioContent.MIMEType,
					"data":     base64.StdEncoding.EncodeToString(audioContent.Data),
				})
				continue
			}

			// Handle TextContent
			if textContent, ok := content.(*mcp.TextContent); ok && textContent.Text != "" {
				text := textContent.Text
				h.logger.Debug("extractResultData: found TextContent block",
					zap.Int("blockIndex", i),
					zap.Int("textLength", len(text)),
					zap.String("textPreview", truncateJSON(text, 100)))

				// Store first non-empty text as fallback
				if firstTextContent == "" {
					firstTextContent = text
				}

				// Strip markdown code fences if present (```json...``` or ```...```)
				cleanedText := stripMarkdownCodeFence(text)

				// Try to parse as JSON
				var jsonData interface{}
				if err := json.Unmarshal([]byte(cleanedText), &jsonData); err == nil {
					// Check if it's a JSON object (map) or array - these have priority
					switch v := jsonData.(type) {
					case map[string]interface{}:
						h.logger.Info("extractResultData: found JSON object in Content block - returning immediately",
							zap.Int("blockIndex", i),
							zap.Int("objectKeys", len(v)),
							zap.String("dataPreview", truncateJSON(jsonData, 150)))
						// JSON object found - return immediately (highest priority)
						return jsonData
					case []interface{}:
						h.logger.Info("extractResultData: found JSON array in Content block - returning immediately",
							zap.Int("blockIndex", i),
							zap.Int("arrayLength", len(v)),
							zap.String("dataPreview", truncateJSON(jsonData, 150)))
						// JSON array found - return immediately (highest priority)
						return jsonData
					default:
						// JSON primitive (string, number, bool, null) - store but keep searching
						h.logger.Debug("extractResultData: found JSON primitive in Content block - storing as fallback",
							zap.Int("blockIndex", i),
							zap.String("primitiveType", fmt.Sprintf("%T", v)),
							zap.String("primitiveValue", truncateJSON(jsonData, 100)))
						if firstJSONPrimitive == nil {
							firstJSONPrimitive = jsonData
						}
					}
				} else {
					h.logger.Debug("extractResultData: Content block is not valid JSON",
						zap.Int("blockIndex", i),
						zap.Error(err))
				}
				continue
			}

			h.logger.Debug("extractResultData: Content block is unhandled type",
				zap.Int("blockIndex", i),
				zap.String("contentType", fmt.Sprintf("%T", content)))
		}

		// If we found media content (images/audio), return it
		// This takes priority over text content for media-generating tools
		if len(mediaContents) > 0 {
			h.logger.Info("extractResultData: returning media content",
				zap.Int("mediaCount", len(mediaContents)))
			if len(mediaContents) == 1 {
				// Single media item - return directly
				return mediaContents[0]
			}
			// Multiple media items - return as array
			return map[string]interface{}{
				"media": mediaContents,
			}
		}

		// No JSON object/array found - return JSON primitive if we found one
		if firstJSONPrimitive != nil {
			h.logger.Info("extractResultData: no JSON object/array found, returning first JSON primitive",
				zap.String("primitiveType", fmt.Sprintf("%T", firstJSONPrimitive)))
			return firstJSONPrimitive
		}

		// No JSON found in any block - return first text content as fallback
		if firstTextContent != "" {
			h.logger.Info("extractResultData: no JSON found, returning first text content as raw string")
			return firstTextContent
		}

		h.logger.Debug("extractResultData: no usable content found in any Content block")
	}

	// Fallback to StructuredContent even if empty (maintain backward compatibility)
	h.logger.Warn("extractResultData: falling back to StructuredContent (may be empty)")
	return result.StructuredContent
}

// executeToolOnServer executes a tool on a remote MCP server using the official SDK
func (h *ToolsDiscoveryHandler) executeToolOnServer(ctx context.Context, serverURL string, headers map[string]interface{}, toolName string, toolArgs map[string]interface{}, debugEnabled bool) (*mcp.CallToolResult, error) {
	debugPrintln(debugEnabled, "   🔌 executeToolOnServer: Creating MCP client...")
	debugPrintf(debugEnabled, "   📍 Server URL: %s\n", serverURL)

	// Create MCP client using official SDK
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "hyperion-mcp-discovery",
		Version: "1.0.0",
	}, nil)

	// Create HTTP client with logging transport that also handles custom headers
	httpClient := createLoggingHTTPClient(headers, debugEnabled)

	// Create transport with custom HTTP client that injects headers
	transport := &mcp.StreamableClientTransport{
		Endpoint:   serverURL,
		HTTPClient: httpClient,
	}

	debugPrintln(debugEnabled, "   🔗 Connecting to MCP server...")
	connectStart := time.Now()

	// Connect to the remote MCP server
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		debugPrintf(debugEnabled, "   ❌ Connection failed after %v: %v\n", time.Since(connectStart), err)
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer session.Close()

	debugPrintf(debugEnabled, "   ✅ Connected in %v\n", time.Since(connectStart))

	// Log the tool call request
	toolArgsJSON, _ := json.Marshal(toolArgs)
	debugPrintf(debugEnabled, "   📤 Calling tool: %s\n", toolName)
	debugPrintf(debugEnabled, "   📋 Tool arguments: %s\n", string(toolArgsJSON))

	callStart := time.Now()

	// Call the tool via the SDK
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: toolArgs,
	})

	debugPrintf(debugEnabled, "   ⏱️  Tool call completed in %v\n", time.Since(callStart))

	// If we got a result, check if it's an error result
	// MCP protocol: tool errors should be returned as results with IsError=true
	// rather than Go errors
	if result != nil {
		debugPrintf(debugEnabled, "   📥 Got result: IsError=%v, ContentBlocks=%d\n", result.IsError, len(result.Content))
		// If the result has IsError=true, return it as-is (this is a tool execution error, not a connection error)
		// Even if err != nil, the result takes precedence
		return result, nil
	}

	// No result returned - this is a connection/protocol error
	if err != nil {
		debugPrintf(debugEnabled, "   ❌ Tool call error (no result): %v\n", err)
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	// Shouldn't reach here, but handle gracefully
	debugPrintln(debugEnabled, "   ⚠️  No result and no error - unexpected state")
	return nil, fmt.Errorf("no result returned from tool call")
}

// loggingRoundTripper wraps an http.RoundTripper to log requests and responses
type loggingRoundTripper struct {
	base    http.RoundTripper
	enabled bool
}

func (t *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.enabled {
		fmt.Println("   📡 HTTP REQUEST:")
		fmt.Printf("      Method: %s\n", req.Method)
		fmt.Printf("      URL: %s\n", req.URL.String())
		fmt.Println("      Headers:")
		for key, values := range req.Header {
			for _, value := range values {
				fmt.Printf("         %s: %s\n", key, maskHeaderValue(key, value))
			}
		}

		// Read and log request body if present
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err == nil {
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset body for actual request
				bodyStr := string(bodyBytes)
				if len(bodyStr) > 500 {
					bodyStr = bodyStr[:500] + "...[truncated]"
				}
				fmt.Printf("      Body: %s\n", bodyStr)
			}
		}
	}

	startTime := time.Now()
	resp, err := t.base.RoundTrip(req)
	elapsed := time.Since(startTime)

	if err != nil {
		if t.enabled {
			fmt.Printf("   ❌ HTTP ERROR after %v: %v\n", elapsed, err)
		}
		return resp, err
	}

	if t.enabled {
		fmt.Println("   📡 HTTP RESPONSE:")
		fmt.Printf("      Status: %s\n", resp.Status)
		fmt.Printf("      Duration: %v\n", elapsed)
		fmt.Println("      Headers:")
		for key, values := range resp.Header {
			for _, value := range values {
				preview := maskHeaderValue(key, value)
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				fmt.Printf("         %s: %s\n", key, preview)
			}
		}
	}

	return resp, err
}

// createLoggingHTTPClient creates an HTTP client with logging and custom header support
func createLoggingHTTPClient(headers map[string]interface{}, debugEnabled bool) *http.Client {
	// Build the transport chain: logging -> headers -> default
	baseTransport := http.DefaultTransport

	// Add header injection layer
	headerTransport := &headerRoundTripper{
		headers: headers,
		base:    baseTransport,
	}

	// Add logging layer on top
	loggingTransport := &loggingRoundTripper{
		base:    headerTransport,
		enabled: debugEnabled,
	}

	return &http.Client{
		Transport: loggingTransport,
	}
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
	if err := h.toolsStorage.AddServer(ctx, serverName, serverURL, description, headers); err != nil {
		return createErrorResult(fmt.Sprintf("failed to add server: %s", err.Error())), nil, nil
	}

	// Discover tools from the server
	tools, err := h.discoverServerTools(ctx, serverURL, headers)
	if err != nil {
		return createErrorResult(fmt.Sprintf("server added but tool discovery failed: %s", err.Error())), nil, nil
	}

	// Store each tool
	toolSuccessCount := 0
	for _, tool := range tools {
		toolName := tool["name"].(string)
		desc, _ := tool["description"].(string)
		schema, _ := tool["inputSchema"].(map[string]interface{})

		if err := h.toolsStorage.StoreToolMetadata(ctx, toolName, desc, schema, serverName); err != nil {
			fmt.Printf("Warning: failed to store tool %s: %v\n", toolName, err)
			continue
		}
		toolSuccessCount++
	}

	// Discover resources from the server
	resources, resourcesErr := h.discoverServerResources(ctx, serverURL, headers)
	resourceSuccessCount := 0
	if resourcesErr == nil {
		for _, resource := range resources {
			uri, _ := resource["uri"].(string)
			name, _ := resource["name"].(string)
			desc, _ := resource["description"].(string)
			mimeType, _ := resource["mimeType"].(string)

			if uri == "" {
				continue // Skip resources without URI
			}

			if err := h.toolsStorage.StoreResourceMetadata(ctx, uri, name, desc, mimeType, serverName); err != nil {
				fmt.Printf("Warning: failed to store resource %s: %v\n", uri, err)
				continue
			}
			resourceSuccessCount++
		}
	}

	// Discover prompts from the server
	prompts, promptsErr := h.discoverServerPrompts(ctx, serverURL, headers)
	promptSuccessCount := 0
	if promptsErr == nil {
		for _, prompt := range prompts {
			name, _ := prompt["name"].(string)
			desc, _ := prompt["description"].(string)
			args, _ := prompt["arguments"].([]interface{})

			// Convert arguments to []map[string]interface{}
			var arguments []map[string]interface{}
			for _, arg := range args {
				if argMap, ok := arg.(map[string]interface{}); ok {
					arguments = append(arguments, argMap)
				}
			}

			if name == "" {
				continue // Skip prompts without name
			}

			if err := h.toolsStorage.StorePromptMetadata(ctx, name, desc, arguments, serverName); err != nil {
				fmt.Printf("Warning: failed to store prompt %s: %v\n", name, err)
				continue
			}
			promptSuccessCount++
		}
	}

	// Sync tools to ToolRegistry for direct AI access
	mcpToolsSynced := 0
	if h.registrySync != nil {
		synced, syncErr := h.registrySync.SyncServerTools(ctx, serverName)
		if syncErr != nil {
			h.logger.Warn("Failed to sync MCP tools to registry",
				zap.String("serverName", serverName),
				zap.Error(syncErr))
		} else {
			mcpToolsSynced = synced
			h.logger.Info("MCP tools synced to registry",
				zap.String("serverName", serverName),
				zap.Int("count", synced))
		}
	}

	resultText := fmt.Sprintf("Server '%s' added successfully!\n\n"+
		"Tools: Discovered %d, stored %d, synced to registry %d\n"+
		"Resources: Discovered %d, stored %d\n"+
		"Prompts: Discovered %d, stored %d\n\n"+
		"Server URL: %s\nDescription: %s",
		serverName,
		len(tools), toolSuccessCount, mcpToolsSynced,
		len(resources), resourceSuccessCount,
		len(prompts), promptSuccessCount,
		serverURL, description)

	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}, map[string]interface{}{
			"serverName":     serverName,
			"toolCount":      toolSuccessCount,
			"toolsSynced":    mcpToolsSynced,
			"resourceCount":  resourceSuccessCount,
			"promptCount":    promptSuccessCount,
			"totalTools":     len(tools),
			"totalResources": len(resources),
			"totalPrompts":   len(prompts),
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

	// Remove old tools, resources, and prompts
	if err := h.toolsStorage.RemoveServerTools(ctx, serverName); err != nil {
		return createErrorResult(fmt.Sprintf("failed to remove old tools: %s", err.Error())), nil, nil
	}
	if err := h.toolsStorage.RemoveServerResources(ctx, serverName); err != nil {
		return createErrorResult(fmt.Sprintf("failed to remove old resources: %s", err.Error())), nil, nil
	}
	if err := h.toolsStorage.RemoveServerPrompts(ctx, serverName); err != nil {
		return createErrorResult(fmt.Sprintf("failed to remove old prompts: %s", err.Error())), nil, nil
	}

	// Discover new tools from the server
	tools, err := h.discoverServerTools(ctx, server.ServerURL, server.Headers)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to discover tools: %s", err.Error())), nil, nil
	}

	// Store each tool
	toolSuccessCount := 0
	for _, tool := range tools {
		toolName := tool["name"].(string)
		desc, _ := tool["description"].(string)
		schema, _ := tool["inputSchema"].(map[string]interface{})

		if err := h.toolsStorage.StoreToolMetadata(ctx, toolName, desc, schema, serverName); err != nil {
			fmt.Printf("Warning: failed to store tool %s: %v\n", toolName, err)
			continue
		}
		toolSuccessCount++
	}

	// Discover resources from the server
	resources, resourcesErr := h.discoverServerResources(ctx, server.ServerURL, server.Headers)
	resourceSuccessCount := 0
	if resourcesErr == nil {
		for _, resource := range resources {
			uri, _ := resource["uri"].(string)
			name, _ := resource["name"].(string)
			desc, _ := resource["description"].(string)
			mimeType, _ := resource["mimeType"].(string)

			if uri == "" {
				continue // Skip resources without URI
			}

			if err := h.toolsStorage.StoreResourceMetadata(ctx, uri, name, desc, mimeType, serverName); err != nil {
				fmt.Printf("Warning: failed to store resource %s: %v\n", uri, err)
				continue
			}
			resourceSuccessCount++
		}
	}

	// Discover prompts from the server
	prompts, promptsErr := h.discoverServerPrompts(ctx, server.ServerURL, server.Headers)
	promptSuccessCount := 0
	if promptsErr == nil {
		for _, prompt := range prompts {
			name, _ := prompt["name"].(string)
			desc, _ := prompt["description"].(string)
			args, _ := prompt["arguments"].([]interface{})

			// Convert arguments to []map[string]interface{}
			var arguments []map[string]interface{}
			for _, arg := range args {
				if argMap, ok := arg.(map[string]interface{}); ok {
					arguments = append(arguments, argMap)
				}
			}

			if name == "" {
				continue // Skip prompts without name
			}

			if err := h.toolsStorage.StorePromptMetadata(ctx, name, desc, arguments, serverName); err != nil {
				fmt.Printf("Warning: failed to store prompt %s: %v\n", name, err)
				continue
			}
			promptSuccessCount++
		}
	}

	// Update server counts in the database
	if err := h.toolsStorage.UpdateServerCounts(ctx, serverName, toolSuccessCount, resourceSuccessCount, promptSuccessCount); err != nil {
		fmt.Printf("Warning: failed to update server counts: %v\n", err)
		// Don't fail the operation - tools/resources/prompts are already stored
	}

	// Sync tools to ToolRegistry for direct AI access
	mcpToolsSynced := 0
	if h.registrySync != nil {
		synced, syncErr := h.registrySync.SyncServerTools(ctx, serverName)
		if syncErr != nil {
			h.logger.Warn("Failed to sync MCP tools to registry",
				zap.String("serverName", serverName),
				zap.Error(syncErr))
		} else {
			mcpToolsSynced = synced
			h.logger.Info("MCP tools synced to registry",
				zap.String("serverName", serverName),
				zap.Int("count", synced))
		}
	}

	resultText := fmt.Sprintf("Server '%s' rediscovered successfully!\n\n"+
		"Tools: Discovered %d, stored %d, synced to registry %d\n"+
		"Resources: Discovered %d, stored %d\n"+
		"Prompts: Discovered %d, stored %d\n\n"+
		"Server URL: %s",
		serverName,
		len(tools), toolSuccessCount, mcpToolsSynced,
		len(resources), resourceSuccessCount,
		len(prompts), promptSuccessCount,
		server.ServerURL)

	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}, map[string]interface{}{
			"serverName":     serverName,
			"toolCount":      toolSuccessCount,
			"toolsSynced":    mcpToolsSynced,
			"resourceCount":  resourceSuccessCount,
			"promptCount":    promptSuccessCount,
			"totalTools":     len(tools),
			"totalResources": len(resources),
			"totalPrompts":   len(prompts),
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

	// Remove tools from ToolRegistry first (before removing from storage)
	if h.registrySync != nil {
		if removeErr := h.registrySync.RemoveServerTools(serverName); removeErr != nil {
			h.logger.Warn("Failed to remove MCP tools from registry",
				zap.String("serverName", serverName),
				zap.Error(removeErr))
		} else {
			h.logger.Info("MCP tools removed from registry",
				zap.String("serverName", serverName))
		}
	}

	// Remove all tools for this server
	if err := h.toolsStorage.RemoveServerTools(ctx, serverName); err != nil {
		return createErrorResult(fmt.Sprintf("failed to remove server tools: %s", err.Error())), nil, nil
	}

	// Remove server from registry
	if err := h.toolsStorage.RemoveServer(ctx, serverName); err != nil {
		return createErrorResult(fmt.Sprintf("failed to remove server: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("Server '%s' removed successfully!\n\nServer URL: %s\nAll tools and metadata deleted from MongoDB, Qdrant, and ToolRegistry.",
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

// discoverServerTools connects to an MCP server and lists its tools using the official SDK
func (h *ToolsDiscoveryHandler) discoverServerTools(ctx context.Context, serverURL string, headers map[string]interface{}) ([]map[string]interface{}, error) {
	// Log discovery attempt with URL and header info (keys only, not values for security)
	headerKeys := make([]string, 0, len(headers))
	for k := range headers {
		headerKeys = append(headerKeys, k)
	}
	h.logger.Debug("Starting tool discovery",
		zap.String("serverURL", serverURL),
		zap.Int("headerCount", len(headers)),
		zap.Strings("headerKeys", headerKeys))

	// Try modern session-based MCP protocol first
	h.logger.Info("Attempting modern MCP protocol (session-based)",
		zap.String("serverURL", serverURL))

	// Create MCP client using official SDK
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "hyperion-mcp-discovery",
		Version: "1.0.0",
	}, nil)

	// Create transport with custom HTTP client that injects headers
	transport := &mcp.StreamableClientTransport{
		Endpoint:   serverURL,
		HTTPClient: createCustomHTTPClient(headers),
	}

	h.logger.Debug("Created MCP client and transport",
		zap.String("clientName", "hyperion-mcp-discovery"),
		zap.String("endpoint", serverURL))

	// Connect to the remote MCP server
	h.logger.Debug("Attempting SDK connection to MCP server")
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		// Check if this is a session-related error
		if isSessionError(err) {
			h.logger.Info("Modern MCP protocol failed, falling back to legacy protocol",
				zap.String("serverURL", serverURL),
				zap.String("reason", "session not supported"))
			// Fall back to legacy stateless JSON-RPC
			return h.discoverLegacyTools(ctx, serverURL, headers)
		}
		h.logger.Debug("SDK connection failed",
			zap.String("serverURL", serverURL),
			zap.Error(err))
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer session.Close()

	h.logger.Debug("SDK connection successful, calling ListTools")

	// Call tools/list via the SDK
	listResult, err := session.ListTools(ctx, nil)
	if err != nil {
		// Check if this is a session-related error during the call
		if isSessionError(err) {
			h.logger.Info("Modern MCP protocol failed during ListTools, falling back to legacy protocol",
				zap.String("serverURL", serverURL),
				zap.String("reason", "session error during API call"))
			// Fall back to legacy stateless JSON-RPC
			return h.discoverLegacyTools(ctx, serverURL, headers)
		}
		h.logger.Debug("ListTools API call failed",
			zap.String("serverURL", serverURL),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	h.logger.Info("Modern MCP protocol successful",
		zap.String("serverURL", serverURL),
		zap.Int("toolCount", len(listResult.Tools)))

	// Convert SDK tools to map format for compatibility
	tools := make([]map[string]interface{}, 0, len(listResult.Tools))
	for _, tool := range listResult.Tools {
		toolMap := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
		tools = append(tools, toolMap)
		h.logger.Debug("Discovered tool",
			zap.String("serverURL", serverURL),
			zap.String("toolName", tool.Name))
	}

	h.logger.Debug("Tool discovery complete",
		zap.String("serverURL", serverURL),
		zap.Int("totalTools", len(tools)))

	return tools, nil
}

// discoverServerResources connects to an MCP server and lists its resources using the official SDK
func (h *ToolsDiscoveryHandler) discoverServerResources(ctx context.Context, serverURL string, headers map[string]interface{}) ([]map[string]interface{}, error) {
	// Log discovery attempt with URL and header info (keys only, not values for security)
	headerKeys := make([]string, 0, len(headers))
	for k := range headers {
		headerKeys = append(headerKeys, k)
	}
	h.logger.Debug("Starting resource discovery",
		zap.String("serverURL", serverURL),
		zap.Int("headerCount", len(headers)),
		zap.Strings("headerKeys", headerKeys))

	// Try modern session-based MCP protocol first
	h.logger.Info("Attempting modern MCP protocol (session-based) for resources",
		zap.String("serverURL", serverURL))

	// Create MCP client using official SDK
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "hyperion-mcp-discovery",
		Version: "1.0.0",
	}, nil)

	// Create transport with custom HTTP client that injects headers
	transport := &mcp.StreamableClientTransport{
		Endpoint:   serverURL,
		HTTPClient: createCustomHTTPClient(headers),
	}

	h.logger.Debug("Created MCP client and transport for resources",
		zap.String("clientName", "hyperion-mcp-discovery"),
		zap.String("endpoint", serverURL))

	// Connect to the remote MCP server
	h.logger.Debug("Attempting SDK connection to MCP server for resources")
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		// Check if this is a session-related error
		if isSessionError(err) {
			h.logger.Info("Modern MCP protocol failed for resources, falling back to legacy protocol",
				zap.String("serverURL", serverURL),
				zap.String("reason", "session not supported"))
			// Fall back to legacy stateless JSON-RPC
			return h.discoverLegacyResources(ctx, serverURL, headers)
		}
		h.logger.Debug("SDK connection failed for resources",
			zap.String("serverURL", serverURL),
			zap.Error(err))
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer session.Close()

	h.logger.Debug("SDK connection successful, calling ListResources")

	// Call resources/list via the SDK
	listResult, err := session.ListResources(ctx, nil)
	if err != nil {
		// Check if this is a session-related error during the call
		if isSessionError(err) {
			h.logger.Info("Modern MCP protocol failed during ListResources, falling back to legacy protocol",
				zap.String("serverURL", serverURL),
				zap.String("reason", "session error during API call"))
			// Fall back to legacy stateless JSON-RPC
			return h.discoverLegacyResources(ctx, serverURL, headers)
		}
		h.logger.Debug("ListResources API call failed",
			zap.String("serverURL", serverURL),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	h.logger.Info("Modern MCP protocol successful for resources",
		zap.String("serverURL", serverURL),
		zap.Int("resourceCount", len(listResult.Resources)))

	// Convert SDK resources to map format for compatibility
	resources := make([]map[string]interface{}, 0, len(listResult.Resources))
	for _, resource := range listResult.Resources {
		resourceMap := map[string]interface{}{
			"uri":         resource.URI,
			"name":        resource.Name,
			"description": resource.Description,
		}
		if resource.MIMEType != "" {
			resourceMap["mimeType"] = resource.MIMEType
		}
		resources = append(resources, resourceMap)
		h.logger.Debug("Discovered resource",
			zap.String("serverURL", serverURL),
			zap.String("resourceURI", resource.URI),
			zap.String("resourceName", resource.Name))
	}

	h.logger.Debug("Resource discovery complete",
		zap.String("serverURL", serverURL),
		zap.Int("totalResources", len(resources)))

	return resources, nil
}

// discoverServerPrompts connects to an MCP server and lists its prompts using the official SDK
func (h *ToolsDiscoveryHandler) discoverServerPrompts(ctx context.Context, serverURL string, headers map[string]interface{}) ([]map[string]interface{}, error) {
	// Log discovery attempt with URL and header info (keys only, not values for security)
	headerKeys := make([]string, 0, len(headers))
	for k := range headers {
		headerKeys = append(headerKeys, k)
	}
	h.logger.Debug("Starting prompt discovery",
		zap.String("serverURL", serverURL),
		zap.Int("headerCount", len(headers)),
		zap.Strings("headerKeys", headerKeys))

	// Try modern session-based MCP protocol first
	h.logger.Info("Attempting modern MCP protocol (session-based) for prompts",
		zap.String("serverURL", serverURL))

	// Create MCP client using official SDK
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "hyperion-mcp-discovery",
		Version: "1.0.0",
	}, nil)

	// Create transport with custom HTTP client that injects headers
	transport := &mcp.StreamableClientTransport{
		Endpoint:   serverURL,
		HTTPClient: createCustomHTTPClient(headers),
	}

	h.logger.Debug("Created MCP client and transport for prompts",
		zap.String("clientName", "hyperion-mcp-discovery"),
		zap.String("endpoint", serverURL))

	// Connect to the remote MCP server
	h.logger.Debug("Attempting SDK connection to MCP server for prompts")
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		// Check if this is a session-related error
		if isSessionError(err) {
			h.logger.Info("Modern MCP protocol failed for prompts, falling back to legacy protocol",
				zap.String("serverURL", serverURL),
				zap.String("reason", "session not supported"))
			// Fall back to legacy stateless JSON-RPC
			return h.discoverLegacyPrompts(ctx, serverURL, headers)
		}
		h.logger.Debug("SDK connection failed for prompts",
			zap.String("serverURL", serverURL),
			zap.Error(err))
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer session.Close()

	h.logger.Debug("SDK connection successful, calling ListPrompts")

	// Call prompts/list via the SDK
	listResult, err := session.ListPrompts(ctx, nil)
	if err != nil {
		// Check if this is a session-related error during the call
		if isSessionError(err) {
			h.logger.Info("Modern MCP protocol failed during ListPrompts, falling back to legacy protocol",
				zap.String("serverURL", serverURL),
				zap.String("reason", "session error during API call"))
			// Fall back to legacy stateless JSON-RPC
			return h.discoverLegacyPrompts(ctx, serverURL, headers)
		}
		h.logger.Debug("ListPrompts API call failed",
			zap.String("serverURL", serverURL),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	h.logger.Info("Modern MCP protocol successful for prompts",
		zap.String("serverURL", serverURL),
		zap.Int("promptCount", len(listResult.Prompts)))

	// Convert SDK prompts to map format for compatibility
	prompts := make([]map[string]interface{}, 0, len(listResult.Prompts))
	for _, prompt := range listResult.Prompts {
		promptMap := map[string]interface{}{
			"name":        prompt.Name,
			"description": prompt.Description,
		}
		if len(prompt.Arguments) > 0 {
			// Convert arguments to []map[string]interface{}
			args := make([]map[string]interface{}, 0, len(prompt.Arguments))
			for _, arg := range prompt.Arguments {
				argMap := map[string]interface{}{
					"name":        arg.Name,
					"description": arg.Description,
					"required":    arg.Required,
				}
				args = append(args, argMap)
			}
			promptMap["arguments"] = args
		}
		prompts = append(prompts, promptMap)
		h.logger.Debug("Discovered prompt",
			zap.String("serverURL", serverURL),
			zap.String("promptName", prompt.Name))
	}

	h.logger.Debug("Prompt discovery complete",
		zap.String("serverURL", serverURL),
		zap.Int("totalPrompts", len(prompts)))

	return prompts, nil
}
