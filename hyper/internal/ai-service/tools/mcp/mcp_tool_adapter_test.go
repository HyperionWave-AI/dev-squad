package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMCPToolAdapter_Name(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name       string
		serverName string
		toolName   string
		expected   string
	}{
		{
			name:       "simple names",
			serverName: "myserver",
			toolName:   "mytool",
			expected:   "mcp_myserver_mytool",
		},
		{
			name:       "server name with hyphen",
			serverName: "my-server",
			toolName:   "my_tool",
			expected:   "mcp_my_server_my_tool",
		},
		{
			name:       "uppercase names",
			serverName: "MyServer",
			toolName:   "MyTool",
			expected:   "mcp_myserver_mytool",
		},
		{
			name:       "names with double underscores",
			serverName: "my__server",
			toolName:   "my__tool",
			expected:   "mcp_my_server_my_tool",
		},
		{
			name:       "names with spaces",
			serverName: "my server",
			toolName:   "my tool",
			expected:   "mcp_my_server_my_tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewMCPToolAdapter(
				tt.toolName,
				"Test description",
				map[string]interface{}{},
				tt.serverName,
				"http://localhost:8080",
				nil,
				logger,
			)

			assert.Equal(t, tt.expected, adapter.Name())
		})
	}
}

func TestMCPToolAdapter_Description(t *testing.T) {
	logger := zap.NewNop()
	adapter := NewMCPToolAdapter(
		"test_tool",
		"This is a test description",
		map[string]interface{}{},
		"testserver",
		"http://localhost:8080",
		nil,
		logger,
	)

	assert.Equal(t, "This is a test description", adapter.Description())
}

func TestMCPToolAdapter_InputSchema(t *testing.T) {
	logger := zap.NewNop()
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query",
			},
		},
		"required": []string{"query"},
	}

	adapter := NewMCPToolAdapter(
		"search_tool",
		"Search for items",
		schema,
		"testserver",
		"http://localhost:8080",
		nil,
		logger,
	)

	assert.Equal(t, schema, adapter.InputSchema())
}

func TestMCPToolAdapter_Execute_Success(t *testing.T) {
	logger := zap.NewNop()

	// Create mock MCP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request
		var req map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "2.0", req["jsonrpc"])
		assert.Equal(t, "tools/call", req["method"])

		params := req["params"].(map[string]interface{})
		assert.Equal(t, "test_tool", params["name"])

		// Send response
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": `{"status": "success", "data": "test result"}`,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	adapter := NewMCPToolAdapter(
		"test_tool",
		"Test tool",
		map[string]interface{}{},
		"testserver",
		server.URL,
		nil,
		logger,
	)

	result, err := adapter.Execute(context.Background(), map[string]interface{}{
		"query": "test",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Result should be parsed JSON
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "success", resultMap["status"])
	assert.Equal(t, "test result", resultMap["data"])
}

func TestMCPToolAdapter_Execute_WithHeaders(t *testing.T) {
	logger := zap.NewNop()

	// Create mock MCP server that checks headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom headers
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

		// Send response
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "authenticated",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	headers := map[string]interface{}{
		"Authorization":   "Bearer test-token",
		"X-Custom-Header": "custom-value",
	}

	adapter := NewMCPToolAdapter(
		"secure_tool",
		"Secure tool",
		map[string]interface{}{},
		"secureserver",
		server.URL,
		headers,
		logger,
	)

	result, err := adapter.Execute(context.Background(), map[string]interface{}{})

	require.NoError(t, err)
	assert.Equal(t, "authenticated", result)
}

func TestMCPToolAdapter_Execute_Error(t *testing.T) {
	logger := zap.NewNop()

	// Create mock MCP server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    -32600,
				"message": "Invalid request",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	adapter := NewMCPToolAdapter(
		"error_tool",
		"Error tool",
		map[string]interface{}{},
		"errorserver",
		server.URL,
		nil,
		logger,
	)

	_, err := adapter.Execute(context.Background(), map[string]interface{}{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid request")
	assert.Contains(t, err.Error(), "-32600")
}

func TestMCPToolAdapter_Execute_ConnectionError(t *testing.T) {
	logger := zap.NewNop()

	adapter := NewMCPToolAdapter(
		"unreachable_tool",
		"Unreachable tool",
		map[string]interface{}{},
		"unreachableserver",
		"http://localhost:99999", // Invalid port
		nil,
		logger,
	)

	_, err := adapter.Execute(context.Background(), map[string]interface{}{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute HTTP request")
}

func TestMCPToolAdapter_GetServerName(t *testing.T) {
	logger := zap.NewNop()
	adapter := NewMCPToolAdapter(
		"test_tool",
		"Test tool",
		map[string]interface{}{},
		"myserver",
		"http://localhost:8080",
		nil,
		logger,
	)

	assert.Equal(t, "myserver", adapter.GetServerName())
}

func TestMCPToolAdapter_GetOriginalToolName(t *testing.T) {
	logger := zap.NewNop()
	adapter := NewMCPToolAdapter(
		"original_tool_name",
		"Test tool",
		map[string]interface{}{},
		"myserver",
		"http://localhost:8080",
		nil,
		logger,
	)

	assert.Equal(t, "original_tool_name", adapter.GetOriginalToolName())
}

func TestSanitizeToolNamePart(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-hyphen", "with_hyphen"},
		{"with_underscore", "with_underscore"},
		{"UPPERCASE", "uppercase"},
		{"with__double", "with_double"},
		{"with   spaces", "with_spaces"},
		{"_leading", "leading"},
		{"trailing_", "trailing"},
		{"_both_", "both"},
		{"mixed-case_NAME", "mixed_case_name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeToolNamePart(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
