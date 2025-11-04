package handlers

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestExtractResultData_PrioritizesObjectOverPrimitive tests that JSON objects
// are prioritized over JSON primitives when multiple Content blocks are present
func TestExtractResultData_PrioritizesObjectOverPrimitive(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := &ToolsDiscoveryHandler{logger: logger}

	tests := []struct {
		name     string
		result   *mcp.CallToolResult
		expected interface{}
		desc     string
	}{
		{
			name: "object_after_primitive",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: `"Generated secure link for file123"`, // JSON primitive string
					},
					&mcp.TextContent{
						Text: `{"url":"https://example.com/file123","fileId":"file123","metadata":{"size":1024}}`, // JSON object
					},
				},
			},
			expected: map[string]interface{}{
				"url":    "https://example.com/file123",
				"fileId": "file123",
				"metadata": map[string]interface{}{
					"size": float64(1024),
				},
			},
			desc: "Should return JSON object (Content[1]) instead of JSON string primitive (Content[0])",
		},
		{
			name: "only_primitive",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: `"Just a string"`,
					},
				},
			},
			expected: "Just a string",
			desc:     "Should return JSON primitive when no object is available",
		},
		{
			name: "object_first",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: `{"status":"success","data":"value"}`,
					},
					&mcp.TextContent{
						Text: `"Some string"`,
					},
				},
			},
			expected: map[string]interface{}{
				"status": "success",
				"data":   "value",
			},
			desc: "Should return first JSON object immediately without checking remaining blocks",
		},
		{
			name: "array_prioritized",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: `"summary text"`,
					},
					&mcp.TextContent{
						Text: `[{"id":1,"name":"item1"},{"id":2,"name":"item2"}]`,
					},
				},
			},
			expected: []interface{}{
				map[string]interface{}{"id": float64(1), "name": "item1"},
				map[string]interface{}{"id": float64(2), "name": "item2"},
			},
			desc: "Should return JSON array instead of JSON primitive",
		},
		{
			name: "multiple_primitives_then_object",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: `"first string"`,
					},
					&mcp.TextContent{
						Text: `123`,
					},
					&mcp.TextContent{
						Text: `{"result":"final"}`,
					},
				},
			},
			expected: map[string]interface{}{
				"result": "final",
			},
			desc: "Should skip all primitives and return the JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResultData(handler, tt.result)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}

// TestExtractResultData_StructuredContent tests backward compatibility with StructuredContent
func TestExtractResultData_StructuredContent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := &ToolsDiscoveryHandler{logger: logger}

	tests := []struct {
		name     string
		result   *mcp.CallToolResult
		expected interface{}
		desc     string
	}{
		{
			name: "structured_content_map",
			result: &mcp.CallToolResult{
				StructuredContent: map[string]interface{}{
					"key": "value",
				},
			},
			expected: map[string]interface{}{
				"key": "value",
			},
			desc: "Should return non-empty StructuredContent map",
		},
		{
			name: "structured_content_priority",
			result: &mcp.CallToolResult{
				StructuredContent: map[string]interface{}{
					"structured": "data",
				},
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: `{"content":"data"}`,
					},
				},
			},
			expected: map[string]interface{}{
				"structured": "data",
			},
			desc: "StructuredContent should have highest priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResultData(handler, tt.result)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}

// TestExtractResultData_MarkdownWrappedJSON tests extraction of JSON from markdown code fences
func TestExtractResultData_MarkdownWrappedJSON(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := &ToolsDiscoveryHandler{logger: logger}

	tests := []struct {
		name     string
		result   *mcp.CallToolResult
		expected interface{}
		desc     string
	}{
		{
			name: "json_fence_with_object",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "```json\n{\"url\":\"https://example.com/file123\",\"fileId\":\"file123\"}\n```",
					},
				},
			},
			expected: map[string]interface{}{
				"url":    "https://example.com/file123",
				"fileId": "file123",
			},
			desc: "Should extract JSON object from ```json fence",
		},
		{
			name: "generic_fence_with_object",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "```\n{\"status\":\"success\",\"data\":\"value\"}\n```",
					},
				},
			},
			expected: map[string]interface{}{
				"status": "success",
				"data":   "value",
			},
			desc: "Should extract JSON object from generic ``` fence",
		},
		{
			name: "primitive_then_fenced_object",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: `"Generated secure link"`,
					},
					&mcp.TextContent{
						Text: "```json\n{\"url\":\"https://storage.example.com/abc123\",\"expiresAt\":\"2025-12-31T23:59:59Z\"}\n```",
					},
				},
			},
			expected: map[string]interface{}{
				"url":       "https://storage.example.com/abc123",
				"expiresAt": "2025-12-31T23:59:59Z",
			},
			desc: "Should prioritize fenced JSON object over primitive",
		},
		{
			name: "fenced_array",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "```json\n[{\"id\":1,\"name\":\"item1\"},{\"id\":2,\"name\":\"item2\"}]\n```",
					},
				},
			},
			expected: []interface{}{
				map[string]interface{}{"id": float64(1), "name": "item1"},
				map[string]interface{}{"id": float64(2), "name": "item2"},
			},
			desc: "Should extract JSON array from markdown fence",
		},
		{
			name: "yaml_fence_with_json",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "```yaml\n{\"config\":\"value\"}\n```",
					},
				},
			},
			expected: map[string]interface{}{
				"config": "value",
			},
			desc: "Should extract JSON even from ```yaml fence",
		},
		{
			name: "multiline_fenced_object",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "```json\n{\n  \"url\": \"https://example.com/file\",\n  \"metadata\": {\n    \"size\": 2048\n  }\n}\n```",
					},
				},
			},
			expected: map[string]interface{}{
				"url": "https://example.com/file",
				"metadata": map[string]interface{}{
					"size": float64(2048),
				},
			},
			desc: "Should extract multiline formatted JSON from fence",
		},
		{
			name: "backward_compat_plain_json",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: `{"plain":"json","no":"fence"}`,
					},
				},
			},
			expected: map[string]interface{}{
				"plain": "json",
				"no":    "fence",
			},
			desc: "Should maintain backward compatibility with plain JSON (no fence)",
		},
		{
			name: "mixed_fenced_and_plain",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: `"plain string"`,
					},
					&mcp.TextContent{
						Text: "```json\n{\"priority\":\"object\"}\n```",
					},
					&mcp.TextContent{
						Text: `{"also":"object"}`,
					},
				},
			},
			expected: map[string]interface{}{
				"priority": "object",
			},
			desc: "Should find first object whether fenced or plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResultData(handler, tt.result)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}

// TestStripMarkdownCodeFence tests the markdown fence stripping helper
func TestStripMarkdownCodeFence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		desc     string
	}{
		{
			name:     "json_fence",
			input:    "```json\n{\"key\":\"value\"}\n```",
			expected: `{"key":"value"}`,
			desc:     "Should strip ```json fence",
		},
		{
			name:     "generic_fence",
			input:    "```\n{\"data\":123}\n```",
			expected: `{"data":123}`,
			desc:     "Should strip generic ``` fence",
		},
		{
			name:     "yaml_fence",
			input:    "```yaml\nkey: value\n```",
			expected: "key: value",
			desc:     "Should strip ```yaml fence",
		},
		{
			name:     "plain_text",
			input:    `{"plain":"json"}`,
			expected: `{"plain":"json"}`,
			desc:     "Should return plain text unchanged",
		},
		{
			name:     "multiline_json",
			input:    "```json\n{\n  \"nested\": {\n    \"value\": true\n  }\n}\n```",
			expected: "{\n  \"nested\": {\n    \"value\": true\n  }\n}",
			desc:     "Should preserve internal formatting",
		},
		{
			name:     "with_whitespace",
			input:    "  ```json\n  {\"test\":1}  \n  ```  ",
			expected: `{"test":1}`,
			desc:     "Should trim external whitespace",
		},
		{
			name:     "incomplete_fence_start",
			input:    "```json\n{\"no\":\"end\"}",
			expected: "```json\n{\"no\":\"end\"}",
			desc:     "Should return unchanged if no closing fence",
		},
		{
			name:     "incomplete_fence_end",
			input:    "{\"no\":\"start\"}\n```",
			expected: "{\"no\":\"start\"}\n```",
			desc:     "Should return unchanged if no opening fence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownCodeFence(tt.input)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}
