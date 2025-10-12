package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Note: MockToolsStorage is defined in tools_discovery_test.go and reused here

// Test ToolMetadataRegistry creation
func TestNewToolMetadataRegistry(t *testing.T) {
	registry := NewToolMetadataRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.tools)
	assert.Equal(t, 0, len(registry.tools))
}

// Test RegisterTool
func TestRegisterTool(t *testing.T) {
	registry := NewToolMetadataRegistry()

	schema := map[string]interface{}{
		"type":        "mcp-tool",
		"name":        "test_tool",
		"description": "Test tool description",
	}

	registry.RegisterTool("test_tool", "Test tool description", schema)

	tools := registry.GetTools()
	assert.Equal(t, 1, len(tools))
	assert.Equal(t, "test_tool", tools[0].ToolName)
	assert.Equal(t, "Test tool description", tools[0].Description)
	assert.Equal(t, schema, tools[0].Schema)
}

// Test RegisterTool - multiple tools
func TestRegisterTool_Multiple(t *testing.T) {
	registry := NewToolMetadataRegistry()

	for i := 0; i < 5; i++ {
		registry.RegisterTool(
			"tool_"+string(rune('A'+i)),
			"Description "+string(rune('A'+i)),
			map[string]interface{}{"index": i},
		)
	}

	tools := registry.GetTools()
	assert.Equal(t, 5, len(tools))

	// Verify order preservation
	for i := 0; i < 5; i++ {
		assert.Equal(t, "tool_"+string(rune('A'+i)), tools[i].ToolName)
	}
}

// Test GetTools - empty registry
func TestGetTools_Empty(t *testing.T) {
	registry := NewToolMetadataRegistry()
	tools := registry.GetTools()
	assert.NotNil(t, tools)
	assert.Equal(t, 0, len(tools))
}

// Test RegisterToolWithServer
func TestRegisterToolWithServer(t *testing.T) {
	registry := NewToolMetadataRegistry()
	server := mcp.NewServer()

	tool := &mcp.Tool{
		Name:        "test_tool",
		Description: "Test tool for testing",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"param1": {Type: "string"},
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "test result"}},
		}, nil
	}

	// Register tool
	registry.RegisterToolWithServer(server, tool, handler)

	// Verify tool was added to registry
	tools := registry.GetTools()
	assert.Equal(t, 1, len(tools))
	assert.Equal(t, "test_tool", tools[0].ToolName)
	assert.Equal(t, "Test tool for testing", tools[0].Description)

	// Verify schema structure
	assert.Contains(t, tools[0].Schema, "type")
	assert.Contains(t, tools[0].Schema, "name")
	assert.Contains(t, tools[0].Schema, "description")
	assert.Contains(t, tools[0].Schema, "inputSchema")
}

// Test RegisterToolWithServer - nil registry (should not panic)
func TestRegisterToolWithServer_NilRegistry(t *testing.T) {
	var registry *ToolMetadataRegistry
	server := mcp.NewServer()

	tool := &mcp.Tool{
		Name:        "test_tool",
		Description: "Test tool",
		InputSchema: &jsonschema.Schema{Type: "object"},
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "test"}},
		}, nil
	}

	// Should not panic even with nil registry
	assert.NotPanics(t, func() {
		registry.RegisterToolWithServer(server, tool, handler)
	})
}

// Test IndexRegisteredTools - success
func TestIndexRegisteredTools_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := NewToolMetadataRegistry()
	mockStorage := new(MockToolsStorage)

	// Register some tools
	registry.RegisterTool("tool_1", "Description 1", map[string]interface{}{"type": "test"})
	registry.RegisterTool("tool_2", "Description 2", map[string]interface{}{"type": "test"})
	registry.RegisterTool("tool_3", "Description 3", map[string]interface{}{"type": "test"})

	// Setup mock expectations
	mockStorage.On("StoreToolMetadata", mock.Anything, "tool_1", "Description 1", mock.Anything, "mcp-builtin").Return(nil)
	mockStorage.On("StoreToolMetadata", mock.Anything, "tool_2", "Description 2", mock.Anything, "mcp-builtin").Return(nil)
	mockStorage.On("StoreToolMetadata", mock.Anything, "tool_3", "Description 3", mock.Anything, "mcp-builtin").Return(nil)

	// Index tools
	indexed, err := IndexRegisteredTools(registry, mockStorage, logger)

	assert.NoError(t, err)
	assert.Equal(t, 3, indexed)
	mockStorage.AssertExpectations(t)
}

// Test IndexRegisteredTools - partial failure
func TestIndexRegisteredTools_PartialFailure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := NewToolMetadataRegistry()
	mockStorage := new(MockToolsStorage)

	// Register tools
	registry.RegisterTool("tool_1", "Description 1", map[string]interface{}{})
	registry.RegisterTool("tool_2", "Description 2", map[string]interface{}{})
	registry.RegisterTool("tool_3", "Description 3", map[string]interface{}{})

	// Setup mock - tool_2 fails
	mockStorage.On("StoreToolMetadata", mock.Anything, "tool_1", mock.Anything, mock.Anything, "mcp-builtin").Return(nil)
	mockStorage.On("StoreToolMetadata", mock.Anything, "tool_2", mock.Anything, mock.Anything, "mcp-builtin").Return(assert.AnError)
	mockStorage.On("StoreToolMetadata", mock.Anything, "tool_3", mock.Anything, mock.Anything, "mcp-builtin").Return(nil)

	// Index tools
	indexed, err := IndexRegisteredTools(registry, mockStorage, logger)

	assert.NoError(t, err) // Function returns nil error even with partial failures
	assert.Equal(t, 2, indexed) // Only 2 succeeded
	mockStorage.AssertExpectations(t)
}

// Test IndexRegisteredTools - empty registry
func TestIndexRegisteredTools_EmptyRegistry(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := NewToolMetadataRegistry()
	mockStorage := new(MockToolsStorage)

	// Index empty registry
	indexed, err := IndexRegisteredTools(registry, mockStorage, logger)

	assert.NoError(t, err)
	assert.Equal(t, 0, indexed)
}

// Test IndexRegisteredTools - nil registry
func TestIndexRegisteredTools_NilRegistry(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockStorage := new(MockToolsStorage)

	// Index with nil registry
	indexed, err := IndexRegisteredTools(nil, mockStorage, logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry is nil")
	assert.Equal(t, 0, indexed)
}

// Test IndexRegisteredTools - nil storage
func TestIndexRegisteredTools_NilStorage(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := NewToolMetadataRegistry()

	// Index with nil storage
	indexed, err := IndexRegisteredTools(registry, nil, logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "toolsStorage is nil")
	assert.Equal(t, 0, indexed)
}

// Test IndexRegisteredTools - timeout handling
func TestIndexRegisteredTools_Timeout(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := NewToolMetadataRegistry()
	mockStorage := new(MockToolsStorage)

	registry.RegisterTool("tool_1", "Description 1", map[string]interface{}{})

	// Setup mock to simulate slow operation (but within 30s timeout)
	mockStorage.On("StoreToolMetadata", mock.Anything, "tool_1", mock.Anything, mock.Anything, "mcp-builtin").
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			// Check that context has timeout
			_, hasDeadline := ctx.Deadline()
			assert.True(t, hasDeadline, "Context should have timeout")
		}).
		Return(nil)

	// Index tools
	indexed, err := IndexRegisteredTools(registry, mockStorage, logger)

	assert.NoError(t, err)
	assert.Equal(t, 1, indexed)
	mockStorage.AssertExpectations(t)
}

// Test IndexRegisteredTools - context deadline validation
func TestIndexRegisteredTools_ContextDeadline(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := NewToolMetadataRegistry()
	mockStorage := new(MockToolsStorage)

	registry.RegisterTool("tool_1", "Description", map[string]interface{}{})

	// Verify context has 30-second timeout
	mockStorage.On("StoreToolMetadata", mock.Anything, "tool_1", mock.Anything, mock.Anything, "mcp-builtin").
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			deadline, ok := ctx.Deadline()
			assert.True(t, ok, "Context should have deadline")

			// Check deadline is approximately 30 seconds from now
			timeUntilDeadline := time.Until(deadline)
			assert.True(t, timeUntilDeadline <= 30*time.Second)
			assert.True(t, timeUntilDeadline > 29*time.Second) // Allow 1s variance
		}).
		Return(nil)

	IndexRegisteredTools(registry, mockStorage, logger)
	mockStorage.AssertExpectations(t)
}

// Test concurrent tool registration (safety check)
func TestRegisterTool_Concurrent(t *testing.T) {
	registry := NewToolMetadataRegistry()
	done := make(chan bool)

	// Register tools concurrently
	for i := 0; i < 10; i++ {
		go func(index int) {
			registry.RegisterTool(
				"tool_"+string(rune('A'+index)),
				"Description",
				map[string]interface{}{},
			)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	tools := registry.GetTools()
	assert.Equal(t, 10, len(tools), "All tools should be registered")
}
