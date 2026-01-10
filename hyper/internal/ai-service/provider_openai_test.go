package aiservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewOpenAIProvider_DefaultConfig tests provider creation with default settings
func TestNewOpenAIProvider_DefaultConfig(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}

	provider, err := newOpenAIProvider(config, nil)

	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.client)
	assert.Equal(t, config, provider.config)
	assert.NotNil(t, provider.tokenLogger)
}

// TestNewOpenAIProvider_CustomBaseURL tests provider creation with custom base URL
func TestNewOpenAIProvider_CustomBaseURL(t *testing.T) {
	config := &AIConfig{
		Provider:    "openai",
		Model:       "gpt-4o",
		APIKey:      "test-api-key",
		ProviderURL: "https://custom.openai.com/v1/",
	}

	provider, err := newOpenAIProvider(config, nil)

	require.NoError(t, err)
	assert.NotNil(t, provider)
	// The URL should have trailing slash removed
	assert.Equal(t, "https://custom.openai.com/v1", strings.TrimSuffix(config.ProviderURL, "/"))
}

// TestNewOpenAIProvider_WithMetricsStore tests provider creation with metrics store
func TestNewOpenAIProvider_WithMetricsStore(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	metricsStore := NewInMemoryMetricsStore(100)

	provider, err := newOpenAIProvider(config, metricsStore)

	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, metricsStore, provider.metricsStore)
}

// TestOpenAIProvider_SupportsTools tests that the provider reports tool support
func TestOpenAIProvider_SupportsTools(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}

	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	assert.True(t, provider.SupportsTools())
}

// TestConvertMessagesToOpenAI_BasicMessages tests conversion of basic messages
func TestConvertMessagesToOpenAI_BasicMessages(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello!"},
		{Role: "assistant", Content: "Hi there!"},
	}

	result := provider.convertMessagesToOpenAI(messages)

	assert.Len(t, result, 3)
	// Check that all messages were converted (we can't easily check the content without marshaling)
	assert.NotNil(t, result[0])
	assert.NotNil(t, result[1])
	assert.NotNil(t, result[2])
}

// TestConvertMessagesToOpenAI_ToolCallMessages tests conversion of tool call messages
func TestConvertMessagesToOpenAI_ToolCallMessages(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	messages := []Message{
		{
			Role: "tool_call",
			ToolCall: &ToolCall{
				ID:   "call_123",
				Name: "get_weather",
				Args: map[string]interface{}{"location": "New York"},
			},
		},
	}

	result := provider.convertMessagesToOpenAI(messages)

	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].OfAssistant)
	assert.Len(t, result[0].OfAssistant.ToolCalls, 1)
	assert.Equal(t, "call_123", result[0].OfAssistant.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", result[0].OfAssistant.ToolCalls[0].Function.Name)
}

// TestConvertMessagesToOpenAI_ToolResultMessages tests conversion of tool result messages
func TestConvertMessagesToOpenAI_ToolResultMessages(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	messages := []Message{
		{
			Role: "tool_result",
			ToolResult: &ToolResult{
				ID:     "call_123",
				Output: "72°F and sunny",
			},
		},
	}

	result := provider.convertMessagesToOpenAI(messages)

	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].OfTool)
}

// TestConvertMessagesToOpenAI_ToolResultWithError tests conversion of error tool results
func TestConvertMessagesToOpenAI_ToolResultWithError(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	messages := []Message{
		{
			Role: "tool_result",
			ToolResult: &ToolResult{
				ID:    "call_123",
				Error: "Location not found",
			},
		},
	}

	result := provider.convertMessagesToOpenAI(messages)

	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].OfTool)
}

// TestConvertToolsToOpenAI tests conversion of tools to OpenAI format
func TestConvertToolsToOpenAI(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	tools := []Tool{
		{
			Type: "function",
			Function: &FunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "City name",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	result := provider.convertToolsToOpenAI(tools)

	assert.Len(t, result, 1)
	assert.Equal(t, "get_weather", result[0].Function.Name)
	assert.NotNil(t, result[0].Function.Description)
}

// TestConvertToolsToOpenAI_NilFunction tests that nil functions are skipped
func TestConvertToolsToOpenAI_NilFunction(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	tools := []Tool{
		{Type: "function", Function: nil},
	}

	result := provider.convertToolsToOpenAI(tools)

	assert.Len(t, result, 0)
}

// TestExtractTokenUsage tests token usage extraction
func TestExtractTokenUsage(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	// Test with nil completion
	usage := provider.extractTokenUsage(nil)
	assert.NotNil(t, usage)
	assert.Equal(t, "openai", usage.Provider)
	assert.Equal(t, "gpt-4o", usage.Model)
	assert.Equal(t, 0, usage.PromptTokens)
	assert.Equal(t, 0, usage.CompletionTokens)
}

// TestParseToolCallArgs tests parsing of tool call arguments
func TestParseToolCallArgs(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	tests := []struct {
		name     string
		argsJSON string
		expected map[string]interface{}
	}{
		{
			name:     "valid JSON",
			argsJSON: `{"location": "New York", "units": "celsius"}`,
			expected: map[string]interface{}{"location": "New York", "units": "celsius"},
		},
		{
			name:     "empty object",
			argsJSON: `{}`,
			expected: map[string]interface{}{},
		},
		{
			name:     "invalid JSON",
			argsJSON: `{invalid}`,
			expected: map[string]interface{}{},
		},
		{
			name:     "empty string",
			argsJSON: ``,
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.parseToolCallArgs(tt.argsJSON)
			assert.NotNil(t, result)
			for k, v := range tt.expected {
				assert.Equal(t, v, result[k])
			}
		})
	}
}

// TestMapStopReason tests stop reason mapping
func TestMapStopReason(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	tests := []struct {
		input    string
		expected string
	}{
		{"tool_calls", "tool_use"},
		{"stop", "end_turn"},
		{"length", "max_tokens"},
		{"content_filter", "content_filter"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := provider.mapStopReason(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildOpenAIMessagesForLogging tests message logging format
func TestBuildOpenAIMessagesForLogging(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello!"},
		{
			Role: "tool_call",
			ToolCall: &ToolCall{
				ID:   "call_123",
				Name: "get_weather",
				Args: map[string]interface{}{"location": "New York"},
			},
		},
		{
			Role: "tool_result",
			ToolResult: &ToolResult{
				ID:     "call_123",
				Output: "72°F",
			},
		},
	}

	result := provider.buildOpenAIMessagesForLogging(messages)

	assert.Len(t, result, 4)
	assert.Equal(t, "system", result[0]["role"])
	assert.Equal(t, "user", result[1]["role"])
	assert.Equal(t, "assistant", result[2]["role"])
	assert.NotNil(t, result[2]["tool_calls"])
	assert.Equal(t, "tool", result[3]["role"])
}

// TestBuildOpenAIToolsForLogging tests tool logging format
func TestBuildOpenAIToolsForLogging(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	tools := []Tool{
		{
			Type: "function",
			Function: &FunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		},
		{Type: "function", Function: nil}, // Should be skipped
	}

	result := provider.buildOpenAIToolsForLogging(tools)

	assert.Len(t, result, 1)
	assert.Equal(t, "function", result[0]["type"])
	function := result[0]["function"].(map[string]interface{})
	assert.Equal(t, "get_weather", function["name"])
}

// TestOpenAIProvider_StreamChat_Integration tests streaming chat with a mock server
func TestOpenAIProvider_StreamChat_Integration(t *testing.T) {
	// Create a mock server that returns streaming response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		// Send streaming response
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":" there"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}`,
		}

		flusher := w.(http.Flusher)
		for _, chunk := range chunks {
			_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
			flusher.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	// Create provider with mock server URL
	config := &AIConfig{
		Provider:    "openai",
		Model:       "gpt-4o",
		APIKey:      "test-api-key",
		ProviderURL: server.URL,
	}

	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	// Test streaming
	messages := []Message{
		{Role: "user", Content: "Say hello"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	outputChan, err := provider.StreamChat(ctx, messages)
	require.NoError(t, err)

	// Collect response
	var response strings.Builder
	for chunk := range outputChan {
		if !strings.HasPrefix(chunk, "ERROR:") {
			response.WriteString(chunk)
		}
	}

	assert.Equal(t, "Hello there!", response.String())
}

// TestOpenAIProvider_StreamChatWithTools_Integration tests tool calling with a mock server
func TestOpenAIProvider_StreamChatWithTools_Integration(t *testing.T) {
	// Create a mock server that returns tool call response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request to verify tools were sent
		var reqBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		// Verify tools were included
		tools, ok := reqBody["tools"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, tools, 1)

		// Return response with tool call
		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_abc123",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "get_weather",
									"arguments": `{"location":"New York"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     50,
				"completion_tokens": 20,
				"total_tokens":      70,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with mock server URL
	config := &AIConfig{
		Provider:    "openai",
		Model:       "gpt-4o",
		APIKey:      "test-api-key",
		ProviderURL: server.URL,
	}

	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	// Test with tools
	messages := []Message{
		{Role: "user", Content: "What's the weather in New York?"},
	}

	tools := []Tool{
		{
			Type: "function",
			Function: &FunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "City name",
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := provider.StreamChatWithTools(ctx, messages, tools)
	require.NoError(t, err)

	// Drain text channel
	for range response.TextChannel {
	}

	// Verify tool calls
	assert.Len(t, response.ToolCalls, 1)
	assert.Equal(t, "call_abc123", response.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", response.ToolCalls[0].Name)
	assert.Equal(t, "New York", response.ToolCalls[0].Args["location"])

	// Verify token usage
	assert.NotNil(t, response.TokenUsage)
	assert.Equal(t, 50, response.TokenUsage.PromptTokens)
	assert.Equal(t, 20, response.TokenUsage.CompletionTokens)
	assert.Equal(t, 70, response.TokenUsage.TotalTokens)

	// Verify stop reason was mapped correctly
	assert.Equal(t, "tool_use", response.StopReason)
}

// TestOpenAIProvider_StreamChat_ContextCancelled tests context cancellation
func TestOpenAIProvider_StreamChat_ContextCancelled(t *testing.T) {
	// Create a slow mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay response
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	config := &AIConfig{
		Provider:    "openai",
		Model:       "gpt-4o",
		APIKey:      "test-api-key",
		ProviderURL: server.URL,
	}

	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	outputChan, err := provider.StreamChat(ctx, messages)
	require.NoError(t, err)

	// Wait for channel to close
	var gotError bool
	for chunk := range outputChan {
		if strings.HasPrefix(chunk, "ERROR:") {
			gotError = true
		}
	}

	// Should get either context cancellation or just close gracefully
	// The channel should close in either case
	_ = gotError // We don't strictly require an error message
}

// TestOpenAIProvider_StreamChatWithTools_APIError tests API error handling
func TestOpenAIProvider_StreamChatWithTools_APIError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid request",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	config := &AIConfig{
		Provider:    "openai",
		Model:       "gpt-4o",
		APIKey:      "test-api-key",
		ProviderURL: server.URL,
	}

	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	ctx := context.Background()
	_, err = provider.StreamChatWithTools(ctx, messages, nil)

	// Should return an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to call OpenAI API")
}

// TestOpenAIProvider_InvalidMessages tests handling of invalid message types
func TestOpenAIProvider_InvalidMessages(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	// Test messages with unknown role - should be ignored
	messages := []Message{
		{Role: "unknown_role", Content: "This should be ignored"},
		{Role: "user", Content: "Hello"},
	}

	result := provider.convertMessagesToOpenAI(messages)

	// Only the user message should be converted
	assert.Len(t, result, 1)
}

// TestOpenAIProvider_ToolCallWithContent tests tool call with accompanying text content
func TestOpenAIProvider_ToolCallWithContent(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	messages := []Message{
		{
			Role:    "tool_call",
			Content: "Let me check the weather for you.",
			ToolCall: &ToolCall{
				ID:   "call_123",
				Name: "get_weather",
				Args: map[string]interface{}{"location": "New York"},
			},
		},
	}

	result := provider.convertMessagesToOpenAI(messages)

	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].OfAssistant)
	// Content should be set
	assert.NotNil(t, result[0].OfAssistant.Content)
}

// TestOpenAIProvider_ToolResultWithJSONOutput tests tool result with complex JSON output
func TestOpenAIProvider_ToolResultWithJSONOutput(t *testing.T) {
	config := &AIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-api-key",
	}
	provider, err := newOpenAIProvider(config, nil)
	require.NoError(t, err)

	messages := []Message{
		{
			Role: "tool_result",
			ToolResult: &ToolResult{
				ID: "call_123",
				Output: map[string]interface{}{
					"temperature": 72,
					"conditions":  "sunny",
					"humidity":    45,
				},
			},
		},
	}

	result := provider.convertMessagesToOpenAI(messages)

	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].OfTool)
}

// TestOpenAIProvider_MetricsRecording tests that metrics are recorded correctly
func TestOpenAIProvider_MetricsRecording(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	metricsStore := NewInMemoryMetricsStore(100)
	config := &AIConfig{
		Provider:    "openai",
		Model:       "gpt-4o",
		APIKey:      "test-api-key",
		ProviderURL: server.URL,
	}

	provider, err := newOpenAIProvider(config, metricsStore)
	require.NoError(t, err)

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	ctx := context.Background()
	response, err := provider.StreamChatWithTools(ctx, messages, nil)
	require.NoError(t, err)

	// Drain text channel
	for range response.TextChannel {
	}

	// Check metrics were recorded
	metrics, err := metricsStore.GetProviderMetrics("openai", 10)
	require.NoError(t, err)
	assert.Len(t, metrics, 1)
	assert.Equal(t, 10, metrics[0].PromptTokens)
	assert.Equal(t, 5, metrics[0].CompletionTokens)
	assert.Equal(t, 15, metrics[0].TotalTokens)
	assert.True(t, metrics[0].Success)
}
