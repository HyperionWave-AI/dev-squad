package aiservice

import (
	"hyper/internal/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestConvertToLangChainMessages_BasicMessages tests conversion of basic user/assistant/system messages
func TestConvertToLangChainMessages_BasicMessages(t *testing.T) {
	dbMessages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Hello, how are you?",
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "I'm doing well, thank you!",
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "system",
			Content:   "You are a helpful assistant.",
			Timestamp: time.Now(),
		},
	}

	result := ConvertToLangChainMessages(dbMessages)

	require.Len(t, result, 3, "Should convert all 3 messages")

	// Check user message
	assert.Equal(t, "user", result[0].Role)
	assert.Equal(t, "Hello, how are you?", result[0].Content)
	assert.Nil(t, result[0].ToolCall)
	assert.Nil(t, result[0].ToolResult)

	// Check assistant message
	assert.Equal(t, "assistant", result[1].Role)
	assert.Equal(t, "I'm doing well, thank you!", result[1].Content)
	assert.Nil(t, result[1].ToolCall)
	assert.Nil(t, result[1].ToolResult)

	// Check system message
	assert.Equal(t, "system", result[2].Role)
	assert.Equal(t, "You are a helpful assistant.", result[2].Content)
	assert.Nil(t, result[2].ToolCall)
	assert.Nil(t, result[2].ToolResult)
}

// TestConvertToLangChainMessages_ToolCall tests conversion of tool_call messages with structured data
func TestConvertToLangChainMessages_ToolCall(t *testing.T) {
	toolCallID := "call_123"
	toolName := "search_documents"
	toolArgs := map[string]interface{}{
		"query": "golang best practices",
		"limit": float64(10),
	}

	dbMessages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_call",
			Content:   "", // May be empty initially
			Timestamp: time.Now(),
			ToolCall: &models.ToolCallData{
				ID:   toolCallID,
				Name: toolName,
				Args: toolArgs,
			},
		},
	}

	result := ConvertToLangChainMessages(dbMessages)

	require.Len(t, result, 1)
	msg := result[0]

	// Verify role
	assert.Equal(t, "tool_call", msg.Role)

	// Verify ToolCall structured data is preserved
	require.NotNil(t, msg.ToolCall, "ToolCall should be preserved")
	assert.Equal(t, toolCallID, msg.ToolCall.ID)
	assert.Equal(t, toolName, msg.ToolCall.Name)
	assert.Equal(t, toolArgs, msg.ToolCall.Args)

	// Verify content is enhanced with tool call details
	assert.Contains(t, msg.Content, "Tool call: search_documents")
	assert.Contains(t, msg.Content, "call_123")
	assert.Contains(t, msg.Content, "golang best practices")

	// ToolResult should be nil
	assert.Nil(t, msg.ToolResult)
}

// TestConvertToLangChainMessages_ToolResult_Success tests conversion of successful tool_result messages
func TestConvertToLangChainMessages_ToolResult_Success(t *testing.T) {
	toolResultID := "call_456"
	toolName := "get_weather"
	toolOutput := map[string]interface{}{
		"temperature": 72.5,
		"condition":   "sunny",
		"humidity":    45,
	}

	dbMessages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_result",
			Content:   "", // May be empty initially
			Timestamp: time.Now(),
			ToolResult: &models.ToolResultData{
				ID:         toolResultID,
				Name:       toolName,
				Output:     toolOutput,
				Error:      "",
				DurationMs: 250,
			},
		},
	}

	result := ConvertToLangChainMessages(dbMessages)

	require.Len(t, result, 1)
	msg := result[0]

	// Verify role
	assert.Equal(t, "tool_result", msg.Role)

	// Verify ToolResult structured data is preserved
	require.NotNil(t, msg.ToolResult, "ToolResult should be preserved")
	assert.Equal(t, toolResultID, msg.ToolResult.ID)
	assert.Equal(t, toolName, msg.ToolResult.Name)
	assert.Equal(t, toolOutput, msg.ToolResult.Output)
	assert.Empty(t, msg.ToolResult.Error)
	assert.Equal(t, int64(250), msg.ToolResult.DurationMs)

	// Verify content is enhanced with tool result details
	assert.Contains(t, msg.Content, "Tool result for get_weather")
	assert.Contains(t, msg.Content, "call_456")
	assert.Contains(t, msg.Content, "temperature")
	assert.Contains(t, msg.Content, "Duration: 250ms")

	// ToolCall should be nil
	assert.Nil(t, msg.ToolCall)
}

// TestConvertToLangChainMessages_ToolResult_Error tests conversion of failed tool_result messages
func TestConvertToLangChainMessages_ToolResult_Error(t *testing.T) {
	toolResultID := "call_789"
	toolName := "delete_file"
	errorMsg := "permission denied: file is read-only"

	dbMessages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_result",
			Content:   "",
			Timestamp: time.Now(),
			ToolResult: &models.ToolResultData{
				ID:         toolResultID,
				Name:       toolName,
				Output:     nil,
				Error:      errorMsg,
				DurationMs: 50,
			},
		},
	}

	result := ConvertToLangChainMessages(dbMessages)

	require.Len(t, result, 1)
	msg := result[0]

	// Verify role
	assert.Equal(t, "tool_result", msg.Role)

	// Verify ToolResult error is preserved
	require.NotNil(t, msg.ToolResult)
	assert.Equal(t, toolResultID, msg.ToolResult.ID)
	assert.Equal(t, toolName, msg.ToolResult.Name)
	assert.Equal(t, errorMsg, msg.ToolResult.Error)
	assert.Equal(t, int64(50), msg.ToolResult.DurationMs)

	// Verify content includes ERROR message
	assert.Contains(t, msg.Content, "Tool result for delete_file")
	assert.Contains(t, msg.Content, "ERROR")
	assert.Contains(t, msg.Content, errorMsg)
	assert.Contains(t, msg.Content, "Duration: 50ms")
}

// TestConvertToLangChainMessages_MixedConversation tests a realistic conversation with mixed message types
func TestConvertToLangChainMessages_MixedConversation(t *testing.T) {
	dbMessages := []models.ChatMessage{
		// User asks a question
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "What's the weather in San Francisco?",
			Timestamp: time.Now(),
		},
		// AI makes a tool call
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_call",
			Content:   "",
			Timestamp: time.Now(),
			ToolCall: &models.ToolCallData{
				ID:   "call_001",
				Name: "get_weather",
				Args: map[string]interface{}{
					"city": "San Francisco",
				},
			},
		},
		// Tool returns result
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_result",
			Content:   "",
			Timestamp: time.Now(),
			ToolResult: &models.ToolResultData{
				ID:   "call_001",
				Name: "get_weather",
				Output: map[string]interface{}{
					"temperature": 68.0,
					"condition":   "partly cloudy",
				},
				DurationMs: 150,
			},
		},
		// AI responds with answer
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "The weather in San Francisco is partly cloudy with a temperature of 68°F.",
			Timestamp: time.Now(),
		},
		// User follows up referencing previous tool result
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "What about tomorrow?",
			Timestamp: time.Now(),
		},
	}

	result := ConvertToLangChainMessages(dbMessages)

	require.Len(t, result, 5, "Should preserve all messages in conversation")

	// User message
	assert.Equal(t, "user", result[0].Role)
	assert.Nil(t, result[0].ToolCall)
	assert.Nil(t, result[0].ToolResult)

	// Tool call message
	assert.Equal(t, "tool_call", result[1].Role)
	require.NotNil(t, result[1].ToolCall)
	assert.Equal(t, "call_001", result[1].ToolCall.ID)
	assert.Equal(t, "get_weather", result[1].ToolCall.Name)
	assert.Contains(t, result[1].Content, "Tool call: get_weather")

	// Tool result message
	assert.Equal(t, "tool_result", result[2].Role)
	require.NotNil(t, result[2].ToolResult)
	assert.Equal(t, "call_001", result[2].ToolResult.ID)
	assert.Contains(t, result[2].Content, "Tool result for get_weather")
	assert.Contains(t, result[2].Content, "68")

	// Assistant message
	assert.Equal(t, "assistant", result[3].Role)
	assert.Contains(t, result[3].Content, "partly cloudy")
	assert.Nil(t, result[3].ToolCall)
	assert.Nil(t, result[3].ToolResult)

	// Follow-up user message
	assert.Equal(t, "user", result[4].Role)
	assert.Equal(t, "What about tomorrow?", result[4].Content)
}

// TestConvertToLangChainMessages_EmptyInput tests handling of empty message slice
func TestConvertToLangChainMessages_EmptyInput(t *testing.T) {
	result := ConvertToLangChainMessages([]models.ChatMessage{})
	assert.NotNil(t, result, "Should return non-nil slice")
	assert.Len(t, result, 0, "Should be empty")
}

// TestConvertToLangChainMessages_LargeToolResult tests handling of large tool results
func TestConvertToLangChainMessages_LargeToolResult(t *testing.T) {
	// Create a large output (e.g., 1000 documents)
	largeOutput := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		largeOutput[i] = map[string]interface{}{
			"id":    i,
			"title": "Document " + string(rune(i)),
			"content": "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		}
	}

	dbMessages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_result",
			Content:   "",
			Timestamp: time.Now(),
			ToolResult: &models.ToolResultData{
				ID:         "call_large",
				Name:       "search_all",
				Output:     largeOutput,
				DurationMs: 5000,
			},
		},
	}

	result := ConvertToLangChainMessages(dbMessages)

	require.Len(t, result, 1)
	msg := result[0]

	// Should still preserve the full output
	require.NotNil(t, msg.ToolResult)
	assert.Equal(t, largeOutput, msg.ToolResult.Output)

	// Content should be generated (though large)
	assert.Contains(t, msg.Content, "Tool result for search_all")
	assert.Contains(t, msg.Content, "call_large")
}

// TestConvertToLangChainMessages_ComplexToolArgs tests handling of complex nested arguments
func TestConvertToLangChainMessages_ComplexToolArgs(t *testing.T) {
	complexArgs := map[string]interface{}{
		"query": "golang",
		"filters": map[string]interface{}{
			"category": []string{"tutorial", "documentation"},
			"date_range": map[string]interface{}{
				"start": "2024-01-01",
				"end":   "2024-12-31",
			},
		},
		"pagination": map[string]interface{}{
			"limit":  50,
			"offset": 0,
		},
	}

	dbMessages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_call",
			Content:   "",
			Timestamp: time.Now(),
			ToolCall: &models.ToolCallData{
				ID:   "call_complex",
				Name: "advanced_search",
				Args: complexArgs,
			},
		},
	}

	result := ConvertToLangChainMessages(dbMessages)

	require.Len(t, result, 1)
	msg := result[0]

	// Verify complex args are preserved
	require.NotNil(t, msg.ToolCall)
	assert.Equal(t, complexArgs, msg.ToolCall.Args)

	// Content should contain JSON representation
	assert.Contains(t, msg.Content, "Tool call: advanced_search")
	assert.Contains(t, msg.Content, "golang")
	assert.Contains(t, msg.Content, "filters")
}

// TestConvertToLangChainMessages_PreservesMessageOrder tests that message order is maintained
func TestConvertToLangChainMessages_PreservesMessageOrder(t *testing.T) {
	dbMessages := []models.ChatMessage{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Message 2"},
		{Role: "user", Content: "Message 3"},
		{Role: "tool_call", Content: "", ToolCall: &models.ToolCallData{ID: "1", Name: "tool1", Args: map[string]interface{}{}}},
		{Role: "tool_result", Content: "", ToolResult: &models.ToolResultData{ID: "1", Name: "tool1", Output: "result", DurationMs: 100}},
		{Role: "assistant", Content: "Message 6"},
	}

	result := ConvertToLangChainMessages(dbMessages)

	require.Len(t, result, 6)

	// Verify order is preserved
	assert.Equal(t, "user", result[0].Role)
	assert.Equal(t, "Message 1", result[0].Content)
	assert.Equal(t, "assistant", result[1].Role)
	assert.Equal(t, "Message 2", result[1].Content)
	assert.Equal(t, "user", result[2].Role)
	assert.Equal(t, "Message 3", result[2].Content)
	assert.Equal(t, "tool_call", result[3].Role)
	assert.Equal(t, "tool_result", result[4].Role)
	assert.Equal(t, "assistant", result[5].Role)
	assert.Equal(t, "Message 6", result[5].Content)
}
