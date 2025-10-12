package handlers

import (
	"context"
	"testing"

	"hyper/internal/mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockToolsStorage is a mock implementation of ToolsStorage for testing
type MockToolsStorage struct {
	mock.Mock
}

func (m *MockToolsStorage) StoreToolMetadata(ctx context.Context, toolName, description string, schema map[string]interface{}, serverName string) error {
	args := m.Called(ctx, toolName, description, schema, serverName)
	return args.Error(0)
}

func (m *MockToolsStorage) SearchTools(ctx context.Context, query string, limit int) ([]*storage.ToolMatch, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ToolMatch), args.Error(1)
}

func (m *MockToolsStorage) GetToolSchema(ctx context.Context, toolName string) (*storage.ToolMetadata, error) {
	args := m.Called(ctx, toolName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ToolMetadata), args.Error(1)
}

// TestHandleDiscoverTools tests the discover_tools handler
func TestHandleDiscoverTools(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		mockResults   []*storage.ToolMatch
		mockError     error
		expectedError bool
		expectedCount int
	}{
		{
			name: "successful search with results",
			args: map[string]interface{}{
				"query": "video tools",
				"limit": float64(5),
			},
			mockResults: []*storage.ToolMatch{
				{
					ToolName:    "mcp__playwright__browser_navigate",
					Description: "Navigate to a URL in the browser",
					ServerName:  "playwright-mcp",
					Score:       0.85,
				},
				{
					ToolName:    "video_process",
					Description: "Process video files with various codecs",
					ServerName:  "video-service",
					Score:       0.92,
				},
			},
			mockError:     nil,
			expectedError: false,
			expectedCount: 2,
		},
		{
			name: "search with no results",
			args: map[string]interface{}{
				"query": "nonexistent",
				"limit": float64(5),
			},
			mockResults:   []*storage.ToolMatch{},
			mockError:     nil,
			expectedError: false,
			expectedCount: 0,
		},
		{
			name: "missing query parameter",
			args: map[string]interface{}{
				"limit": float64(5),
			},
			mockResults:   nil,
			mockError:     nil,
			expectedError: true,
			expectedCount: 0,
		},
		{
			name: "empty query parameter",
			args: map[string]interface{}{
				"query": "",
				"limit": float64(5),
			},
			mockResults:   nil,
			mockError:     nil,
			expectedError: true,
			expectedCount: 0,
		},
		{
			name: "default limit",
			args: map[string]interface{}{
				"query": "database",
			},
			mockResults: []*storage.ToolMatch{
				{
					ToolName:    "db_query",
					Description: "Execute database query",
					ServerName:  "db-service",
					Score:       0.88,
				},
			},
			mockError:     nil,
			expectedError: false,
			expectedCount: 1,
		},
		{
			name: "limit exceeds maximum",
			args: map[string]interface{}{
				"query": "tools",
				"limit": float64(100), // Should be capped at 20
			},
			mockResults: []*storage.ToolMatch{
				{
					ToolName:    "tool1",
					Description: "Tool 1",
					ServerName:  "service1",
					Score:       0.9,
				},
			},
			mockError:     nil,
			expectedError: false,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock storage
			mockStorage := new(MockToolsStorage)

			// Setup expectations only if we expect a call
			if !tt.expectedError || tt.mockResults != nil {
				expectedLimit := 5 // default
				if limit, ok := tt.args["limit"].(float64); ok {
					expectedLimit = int(limit)
					if expectedLimit > 20 {
						expectedLimit = 20
					}
					if expectedLimit < 1 {
						expectedLimit = 1
					}
				}

				if query, ok := tt.args["query"].(string); ok && query != "" {
					mockStorage.On("SearchTools", mock.Anything, query, expectedLimit).
						Return(tt.mockResults, tt.mockError)
				}
			}

			// Create handler
			handler := &ToolsDiscoveryHandler{
				toolsStorage: mockStorage,
			}

			// Execute handler
			result, data, err := handler.handleDiscoverTools(context.Background(), tt.args)

			// Verify results
			if tt.expectedError {
				assert.NotNil(t, result)
				textContent := result.Content[0].(*mcp.TextContent)
				assert.Contains(t, textContent.Text, "query parameter is required")
				assert.Nil(t, data)
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, result)
				assert.Nil(t, err)

				if tt.expectedCount > 0 {
					matches := data.([]*storage.ToolMatch)
					assert.Equal(t, tt.expectedCount, len(matches))
					textContent := result.Content[0].(*mcp.TextContent)
					assert.Contains(t, textContent.Text, "Found")
				} else {
					// Empty results should return empty array
					textContent := result.Content[0].(*mcp.TextContent)
					assert.Contains(t, textContent.Text, "[]")
				}
			}

			// Verify mock expectations
			mockStorage.AssertExpectations(t)
		})
	}
}

// TestHandleGetToolSchema tests the get_tool_schema handler
func TestHandleGetToolSchema(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		mockMetadata  *storage.ToolMetadata
		mockError     error
		expectedError bool
	}{
		{
			name: "successful schema retrieval",
			args: map[string]interface{}{
				"toolName": "video_process",
			},
			mockMetadata: &storage.ToolMetadata{
				ID:          "tool-123",
				ToolName:    "video_process",
				Description: "Process video files",
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file": map[string]interface{}{
							"type":        "string",
							"description": "Video file path",
						},
					},
				},
				ServerName: "video-service",
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name: "tool not found",
			args: map[string]interface{}{
				"toolName": "nonexistent_tool",
			},
			mockMetadata:  nil,
			mockError:     assert.AnError,
			expectedError: true,
		},
		{
			name: "missing toolName parameter",
			args: map[string]interface{}{
				"limit": float64(5),
			},
			mockMetadata:  nil,
			mockError:     nil,
			expectedError: true,
		},
		{
			name: "empty toolName parameter",
			args: map[string]interface{}{
				"toolName": "",
			},
			mockMetadata:  nil,
			mockError:     nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock storage
			mockStorage := new(MockToolsStorage)

			// Setup expectations only if we expect a call
			if toolName, ok := tt.args["toolName"].(string); ok && toolName != "" {
				mockStorage.On("GetToolSchema", mock.Anything, toolName).
					Return(tt.mockMetadata, tt.mockError)
			}

			// Create handler
			handler := &ToolsDiscoveryHandler{
				toolsStorage: mockStorage,
			}

			// Execute handler
			result, data, err := handler.handleGetToolSchema(context.Background(), tt.args)

			// Verify results
			if tt.expectedError {
				assert.NotNil(t, result)
				assert.Nil(t, data)
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, result)
				assert.NotNil(t, data)
				assert.Nil(t, err)

				metadata := data.(*storage.ToolMetadata)
				assert.Equal(t, tt.mockMetadata.ToolName, metadata.ToolName)
				textContent := result.Content[0].(*mcp.TextContent)
				assert.Contains(t, textContent.Text, "Tool Schema")
			}

			// Verify mock expectations
			mockStorage.AssertExpectations(t)
		})
	}
}

// TestHandleExecuteTool tests the execute_tool handler
func TestHandleExecuteTool(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		expectedError  bool
		errorSubstring string
	}{
		{
			name: "missing toolName parameter",
			args: map[string]interface{}{
				"args": map[string]interface{}{
					"param1": "value1",
				},
			},
			expectedError:  true,
			errorSubstring: "toolName parameter is required",
		},
		{
			name: "missing args parameter",
			args: map[string]interface{}{
				"toolName": "test_tool",
			},
			expectedError:  true,
			errorSubstring: "args parameter is required",
		},
		{
			name: "invalid args parameter type",
			args: map[string]interface{}{
				"toolName": "test_tool",
				"args":     "not-an-object",
			},
			expectedError:  true,
			errorSubstring: "args parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock storage
			mockStorage := new(MockToolsStorage)

			// Create handler
			handler := &ToolsDiscoveryHandler{
				toolsStorage: mockStorage,
			}

			// Execute handler
			result, _, err := handler.handleExecuteTool(context.Background(), tt.args)

			// Verify results
			assert.NotNil(t, result)
			assert.Nil(t, err)

			if tt.expectedError {
				// Should have error message in content
				textContent := result.Content[0].(*mcp.TextContent)
				text := textContent.Text
				assert.Contains(t, text, tt.errorSubstring, "Expected error message about missing parameter")
			}
			// Note: We don't test actual HTTP calls to MCP bridge here
			// That would require integration tests or mocking HTTP client
		})
	}
}

// TestCamelCaseParamExtraction tests that parameters are extracted correctly in camelCase format
func TestCamelCaseParamExtraction(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockToolsStorage)

	// Setup expectation with exact camelCase parameter
	mockStorage.On("SearchTools", mock.Anything, "video tools", 5).
		Return([]*storage.ToolMatch{
			{
				ToolName:    "video_tool",
				Description: "A video tool",
				ServerName:  "video-service",
				Score:       0.9,
			},
		}, nil)

	// Create handler
	handler := &ToolsDiscoveryHandler{
		toolsStorage: mockStorage,
	}

	// Test with camelCase parameters
	args := map[string]interface{}{
		"query": "video tools", // camelCase (already correct for single word)
		"limit": float64(5),    // camelCase
	}

	result, data, err := handler.handleDiscoverTools(context.Background(), args)

	// Verify
	assert.NotNil(t, result)
	assert.NotNil(t, data)
	assert.Nil(t, err)

	matches := data.([]*storage.ToolMatch)
	assert.Equal(t, 1, len(matches))
	assert.Equal(t, "video_tool", matches[0].ToolName)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}
