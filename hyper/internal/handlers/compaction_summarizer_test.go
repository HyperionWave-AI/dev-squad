package handlers

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"hyper/internal/models"
)

// TestNewCompactionSummarizer tests the constructor
func TestNewCompactionSummarizer(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	if summarizer == nil {
		t.Error("Expected non-nil summarizer")
	}
	if summarizer.logger == nil {
		t.Error("Expected logger to be set")
	}
}

// TestGenerateSummary_EmptyMessages tests with empty message slice
func TestGenerateSummary_EmptyMessages(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	ctx := context.Background()
	summary, err := summarizer.GenerateSummary(ctx, []models.ChatMessage{})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if summary != "" {
		t.Errorf("Expected empty summary for empty messages, got %q", summary)
	}
}

// TestGenerateSummary_FallbackPath tests fallback summary generation
func TestGenerateSummary_FallbackPath(t *testing.T) {
	logger := zap.NewNop()
	// Pass nil aiService to force fallback path
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "What is the capital of France?",
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "The capital of France is Paris.",
			Timestamp: time.Now(),
		},
	}

	ctx := context.Background()
	summary, err := summarizer.GenerateSummary(ctx, messages)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !contains(summary, "Conversation Summary") {
		t.Errorf("Expected summary to contain 'Conversation Summary', got %q", summary)
	}
}

// TestGenerateFallbackSummary_SimpleConversation tests fallback with simple messages
func TestGenerateFallbackSummary_SimpleConversation(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := []models.ChatMessage{
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
			Content:   "I'm doing well, thank you for asking!",
			Timestamp: time.Now(),
		},
	}

	summary := summarizer.generateFallbackSummary(messages)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !contains(summary, "2 messages") {
		t.Errorf("Expected summary to mention message count, got %q", summary)
	}
	if !contains(summary, "user messages") {
		t.Errorf("Expected summary to mention user messages, got %q", summary)
	}
	if !contains(summary, "assistant responses") {
		t.Errorf("Expected summary to mention assistant responses, got %q", summary)
	}
}

// TestGenerateFallbackSummary_WithToolCalls tests fallback with tool calls
func TestGenerateFallbackSummary_WithToolCalls(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Search for authentication code",
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "I'll search for that",
			ToolCall: &models.ToolCallData{
				ID:   "call_123",
				Name: "code_index_search",
				Args: map[string]interface{}{
					"query": "authentication",
				},
			},
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_result",
			Content:   "Search completed",
			ToolResult: &models.ToolResultData{
				ID:         "call_123",
				Name:       "code_index_search",
				Output:     "Found 5 files",
				DurationMs: 150,
			},
			Timestamp: time.Now(),
		},
	}

	summary := summarizer.generateFallbackSummary(messages)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !contains(summary, "tool results") {
		t.Errorf("Expected summary to mention tool results, got %q", summary)
	}
	if !contains(summary, "code_index_search") {
		t.Errorf("Expected summary to mention tool name, got %q", summary)
	}
}

// TestGenerateFallbackSummary_WithErrors tests fallback with tool errors
func TestGenerateFallbackSummary_WithErrors(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Search for something",
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "I'll search",
			ToolCall: &models.ToolCallData{
				ID:   "call_456",
				Name: "code_index_search",
				Args: map[string]interface{}{
					"query": "test",
				},
			},
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_result",
			Content:   "Search failed",
			ToolResult: &models.ToolResultData{
				ID:         "call_456",
				Name:       "code_index_search",
				Output:     "",
				Error:      "Connection timeout",
				DurationMs: 30000,
			},
			Timestamp: time.Now(),
		},
	}

	summary := summarizer.generateFallbackSummary(messages)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !contains(summary, "tool results") {
		t.Errorf("Expected summary to mention tool results, got %q", summary)
	}
}

// TestGenerateFallbackSummary_LongContent tests truncation of long content
func TestGenerateFallbackSummary_LongContent(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	longContent := "This is a very long message that should be truncated in the summary. " +
		"It contains a lot of text that we don't want to include in full. " +
		"The summary should only show the first 200 characters or so. " +
		"This is just filler text to make the message long enough to trigger truncation. " +
		"More text here to ensure we exceed the 200 character limit."

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   longContent,
			Timestamp: time.Now(),
		},
	}

	summary := summarizer.generateFallbackSummary(messages)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	// The summary should truncate long content
	if len(summary) > len(longContent)+200 {
		t.Errorf("Summary seems too long: %d chars", len(summary))
	}
}

// TestBuildConversationText_SimpleMessages tests conversation text building
func TestBuildConversationText_SimpleMessages(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Hello",
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "Hi there!",
			Timestamp: time.Now(),
		},
	}

	text := summarizer.buildConversationText(messages)

	if !contains(text, "[user]: Hello") {
		t.Errorf("Expected user message in text, got %q", text)
	}
	if !contains(text, "[assistant]: Hi there!") {
		t.Errorf("Expected assistant message in text, got %q", text)
	}
}

// TestBuildConversationText_WithToolCall tests conversation text with tool calls
func TestBuildConversationText_WithToolCall(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "I'll search for that",
			ToolCall: &models.ToolCallData{
				ID:   "call_789",
				Name: "code_search",
				Args: map[string]interface{}{
					"query": "authentication",
				},
			},
			Timestamp: time.Now(),
		},
	}

	text := summarizer.buildConversationText(messages)

	if !contains(text, "Tool: code_search") {
		t.Errorf("Expected tool name in text, got %q", text)
	}
	if !contains(text, "Args:") {
		t.Errorf("Expected tool args in text, got %q", text)
	}
}

// TestBuildConversationText_WithToolResult tests conversation text with tool results
func TestBuildConversationText_WithToolResult(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_result",
			Content:   "Result",
			ToolResult: &models.ToolResultData{
				ID:         "call_999",
				Name:       "code_search",
				Output:     "Found 3 files",
				DurationMs: 200,
			},
			Timestamp: time.Now(),
		},
	}

	text := summarizer.buildConversationText(messages)

	if !contains(text, "Result: Found 3 files") {
		t.Errorf("Expected tool result in text, got %q", text)
	}
}

// TestBuildConversationText_WithToolError tests conversation text with tool errors
func TestBuildConversationText_WithToolError(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_result",
			Content:   "Error",
			ToolResult: &models.ToolResultData{
				ID:         "call_error",
				Name:       "code_search",
				Output:     "",
				Error:      "Timeout after 30s",
				DurationMs: 30000,
			},
			Timestamp: time.Now(),
		},
	}

	text := summarizer.buildConversationText(messages)

	if !contains(text, "Error: Timeout after 30s") {
		t.Errorf("Expected error in text, got %q", text)
	}
}

// TestBuildConversationText_LargeOutput tests truncation of large outputs
func TestBuildConversationText_LargeOutput(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	largeOutput := "This is a very large output that should be truncated. " +
		"It contains a lot of data that we don't want to include in full. " +
		"The conversation text builder should truncate this to 500 characters. " +
		"More text here to ensure we exceed the 500 character limit. " +
		"Even more text to make sure we definitely exceed the limit. " +
		"And more and more and more text to fill up space. " +
		"This should definitely be truncated now."

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "tool_result",
			Content:   "Result",
			ToolResult: &models.ToolResultData{
				ID:         "call_large",
				Name:       "code_search",
				Output:     largeOutput,
				DurationMs: 100,
			},
			Timestamp: time.Now(),
		},
	}

	text := summarizer.buildConversationText(messages)

	// The buildConversationText includes the full output in the conversation text
	// (truncation happens in the fallback summary, not in buildConversationText)
	if !contains(text, largeOutput[:50]) {
		t.Errorf("Expected large output to be included in text, got %q", text)
	}
}

// TestGenerateSummary_MultipleMessages tests with many messages
func TestGenerateSummary_MultipleMessages(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	// Create 10 messages
	messages := make([]models.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = models.ChatMessage{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Message " + string(rune(i)),
			Timestamp: time.Now(),
		}
	}

	ctx := context.Background()
	summary, err := summarizer.GenerateSummary(ctx, messages)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !contains(summary, "10 messages") {
		t.Errorf("Expected summary to mention 10 messages, got %q", summary)
	}
}

// TestGenerateFallbackSummary_MixedRoles tests with mixed message roles
func TestGenerateFallbackSummary_MixedRoles(t *testing.T) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "system",
			Content:   "System message",
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "User message",
			Timestamp: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "Assistant message",
			Timestamp: time.Now(),
		},
	}

	summary := summarizer.generateFallbackSummary(messages)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !contains(summary, "system messages") {
		t.Errorf("Expected summary to mention system messages, got %q", summary)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

// Helper function to find substring anywhere in string
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkGenerateFallbackSummary benchmarks fallback summary generation
func BenchmarkGenerateFallbackSummary(b *testing.B) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := make([]models.ChatMessage, 50)
	for i := 0; i < 50; i++ {
		messages[i] = models.ChatMessage{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Message " + string(rune(i)),
			Timestamp: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		summarizer.generateFallbackSummary(messages)
	}
}

// BenchmarkBuildConversationText benchmarks conversation text building
func BenchmarkBuildConversationText(b *testing.B) {
	logger := zap.NewNop()
	summarizer := NewCompactionSummarizer(nil, logger)

	messages := make([]models.ChatMessage, 50)
	for i := 0; i < 50; i++ {
		messages[i] = models.ChatMessage{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Message " + string(rune(i)),
			Timestamp: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		summarizer.buildConversationText(messages)
	}
}
