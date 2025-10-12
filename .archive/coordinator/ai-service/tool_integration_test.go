package aiservice

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

// MockTool implements ToolExecutor for testing
type MockTool struct {
	name        string
	description string
	schema      map[string]interface{}
	execFunc    func(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) InputSchema() map[string]interface{} {
	return m.schema
}

func (m *MockTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, input)
	}
	return map[string]interface{}{"result": "mock result"}, nil
}

// MockToolProvider implements ToolCapableProvider for testing
type MockToolProvider struct {
	supportsTools bool
	toolCalls     []ToolCall
	responseText  string
}

func (m *MockToolProvider) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		ch <- m.responseText
	}()
	return ch, nil
}

func (m *MockToolProvider) SupportsTools() bool {
	return m.supportsTools
}

func (m *MockToolProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []llms.Tool) (*ToolResponse, error) {
	textChan := make(chan string, 1)
	go func() {
		defer close(textChan)
		textChan <- m.responseText
	}()

	return &ToolResponse{
		TextChannel: textChan,
		ToolCalls:   m.toolCalls,
	}, nil
}

// TestToolRegistry tests the ToolRegistry functionality
func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	t.Run("Register valid tool", func(t *testing.T) {
		tool := &MockTool{
			name:        "test_tool",
			description: "A test tool",
			schema:      map[string]interface{}{"type": "object"},
		}

		err := registry.Register(tool)
		assert.NoError(t, err)

		// Verify tool is registered
		retrievedTool, err := registry.Get("test_tool")
		assert.NoError(t, err)
		assert.Equal(t, "test_tool", retrievedTool.Name())
	})

	t.Run("Reject invalid tool names", func(t *testing.T) {
		invalidNames := []string{
			"TestTool",      // uppercase
			"test-tool",     // hyphen
			"test__tool",    // double underscore
			"123_tool",      // starts with number
			"test tool",     // space
		}

		for _, name := range invalidNames {
			tool := &MockTool{name: name}
			err := registry.Register(tool)
			assert.Error(t, err, "Should reject tool name: %s", name)
		}
	})

	t.Run("Reject duplicate tool names", func(t *testing.T) {
		tool1 := &MockTool{name: "duplicate_tool"}
		tool2 := &MockTool{name: "duplicate_tool"}

		err := registry.Register(tool1)
		assert.NoError(t, err)

		err = registry.Register(tool2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("List tools", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.Register(&MockTool{name: "tool_a"})
		registry.Register(&MockTool{name: "tool_b"})

		tools := registry.List()
		assert.Len(t, tools, 2)
		assert.Contains(t, tools, "tool_a")
		assert.Contains(t, tools, "tool_b")
	})

	t.Run("Execute tool", func(t *testing.T) {
		registry := NewToolRegistry()

		tool := &MockTool{
			name: "echo_tool",
			execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return input["message"], nil
			},
		}
		registry.Register(tool)

		result, err := registry.Execute(context.Background(), "echo_tool", map[string]interface{}{
			"message": "hello",
		})

		assert.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("Execute tool with timeout", func(t *testing.T) {
		registry := NewToolRegistry()

		tool := &MockTool{
			name: "slow_tool",
			execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				select {
				case <-time.After(60 * time.Second):
					return "completed", nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		}
		registry.Register(tool)

		// Should timeout (registry has 30s default timeout)
		result, err := registry.Execute(context.Background(), "slow_tool", map[string]interface{}{})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Get LangChain tools format", func(t *testing.T) {
		registry := NewToolRegistry()

		tool := &MockTool{
			name:        "format_tool",
			description: "Test formatting",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{"type": "string"},
				},
			},
		}
		registry.Register(tool)

		langChainTools := registry.GetToolsForLangChain()
		require.Len(t, langChainTools, 1)

		lcTool := langChainTools[0]
		assert.Equal(t, "function", lcTool.Type)
		assert.Equal(t, "format_tool", lcTool.Function.Name)
		assert.Equal(t, "Test formatting", lcTool.Function.Description)
		assert.NotNil(t, lcTool.Function.Parameters)
	})
}

// TestChatServiceWithTools tests tool calling integration
func TestChatServiceWithTools(t *testing.T) {
	t.Run("Text-only mode when no tools registered", func(t *testing.T) {
		config := &AIConfig{
			Provider:    "openai",
			Model:       "gpt-4",
			APIKey:      "test-key",
			Temperature: 0.7,
		}

		service := &ChatService{
			provider:     &MockToolProvider{supportsTools: true, responseText: "Hello!"},
			config:       config,
			toolRegistry: NewToolRegistry(),
		}

		ctx := context.Background()
		eventChan, err := service.StreamChatWithTools(ctx, []Message{
			{Role: "user", Content: "Hi"},
		}, 5)

		assert.NoError(t, err)

		var events []StreamEvent
		for event := range eventChan {
			events = append(events, event)
		}

		// Should only have token events (no tools registered)
		assert.NotEmpty(t, events)
		for _, event := range events {
			assert.Equal(t, StreamEventToken, event.Type)
		}
	})

	t.Run("Provider doesn't support tools", func(t *testing.T) {
		config := &AIConfig{
			Provider:    "openai",
			Model:       "gpt-4",
			APIKey:      "test-key",
			Temperature: 0.7,
		}

		registry := NewToolRegistry()
		registry.Register(&MockTool{name: "test_tool"})

		service := &ChatService{
			provider:     &MockToolProvider{supportsTools: false, responseText: "Hello!"},
			config:       config,
			toolRegistry: registry,
		}

		ctx := context.Background()
		eventChan, err := service.StreamChatWithTools(ctx, []Message{
			{Role: "user", Content: "Hi"},
		}, 5)

		assert.NoError(t, err)

		var events []StreamEvent
		for event := range eventChan {
			events = append(events, event)
		}

		// Should fallback to text-only
		assert.NotEmpty(t, events)
	})

	t.Run("Tool call and result flow", func(t *testing.T) {
		config := &AIConfig{
			Provider:    "openai",
			Model:       "gpt-4",
			APIKey:      "test-key",
			Temperature: 0.7,
		}

		registry := NewToolRegistry()
		registry.Register(&MockTool{
			name: "calculator",
			execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{"result": 42}, nil
			},
		})

		provider := &MockToolProvider{
			supportsTools: true,
			responseText:  "The answer is 42",
			toolCalls: []ToolCall{
				{
					ID:   "call_123",
					Name: "calculator",
					Args: map[string]interface{}{"expr": "6*7"},
				},
			},
		}

		service := &ChatService{
			provider:     provider,
			config:       config,
			toolRegistry: registry,
		}

		ctx := context.Background()
		eventChan, err := service.StreamChatWithTools(ctx, []Message{
			{Role: "user", Content: "What is 6*7?"},
		}, 5)

		assert.NoError(t, err)

		var events []StreamEvent
		for event := range eventChan {
			events = append(events, event)
		}

		// Should have: token events, tool call event, tool result event
		assert.NotEmpty(t, events)

		// Check for tool call event
		var hasToolCall, hasToolResult bool
		for _, event := range events {
			if event.Type == StreamEventToolCall {
				hasToolCall = true
				assert.NotNil(t, event.ToolCall)
				assert.Equal(t, "calculator", event.ToolCall.Name)
			}
			if event.Type == StreamEventToolResult {
				hasToolResult = true
				assert.NotNil(t, event.ToolResult)
				assert.Equal(t, "calculator", event.ToolResult.Name)
				assert.NoError(t, nil) // No error in result
			}
		}

		assert.True(t, hasToolCall, "Should have tool call event")
		assert.True(t, hasToolResult, "Should have tool result event")
	})

	t.Run("Max tool calls limit", func(t *testing.T) {
		config := &AIConfig{
			Provider:    "openai",
			Model:       "gpt-4",
			APIKey:      "test-key",
			Temperature: 0.7,
		}

		registry := NewToolRegistry()
		registry.Register(&MockTool{
			name: "loop_tool",
			execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return "keep looping", nil
			},
		})

		// Provider that always requests the same tool (infinite loop)
		provider := &MockToolProvider{
			supportsTools: true,
			responseText:  "looping",
			toolCalls: []ToolCall{
				{ID: "call_1", Name: "loop_tool", Args: map[string]interface{}{}},
			},
		}

		service := &ChatService{
			provider:     provider,
			config:       config,
			toolRegistry: registry,
		}

		ctx := context.Background()
		eventChan, err := service.StreamChatWithTools(ctx, []Message{
			{Role: "user", Content: "Loop forever"},
		}, 3) // Max 3 tool calls

		assert.NoError(t, err)

		toolCallCount := 0
		for event := range eventChan {
			if event.Type == StreamEventToolCall {
				toolCallCount++
			}
			if event.Type == StreamEventError {
				assert.Contains(t, event.Error, "maximum tool calls")
			}
		}

		// Should be limited to max tool calls
		assert.LessOrEqual(t, toolCallCount, 3)
	})
}

// TestValidToolNames tests tool name validation
func TestValidToolNames(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"bash", true},
		{"read_file", true},
		{"write_file", true},
		{"list_directory", true},
		{"apply_patch", true},
		{"tool_123", true},
		{"a_b_c", true},
		{"TestTool", false},      // uppercase
		{"test-tool", false},     // hyphen
		{"test__tool", false},    // double underscore
		{"123tool", false},       // starts with number
		{"test tool", false},     // space
		{"test.tool", false},     // dot
		{"_tool", false},         // starts with underscore
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidToolName(tt.name)
			assert.Equal(t, tt.valid, result, "Tool name '%s' validation mismatch", tt.name)
		})
	}
}
