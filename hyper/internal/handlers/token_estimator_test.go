package handlers

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"hyper/internal/models"
)

// TestMessageTokenCache_GetOrEstimate tests the cache functionality
func TestMessageTokenCache_GetOrEstimate(t *testing.T) {
	cache := NewMessageTokenCache()
	estimator := NewTokenEstimator()

	// Create a test message
	msgID := primitive.NewObjectID()
	msg := &models.ChatMessage{
		ID:      msgID,
		Role:    "user",
		Content: "Hello, this is a test message",
	}

	// First call should estimate and cache
	tokens1 := cache.GetOrEstimate(msg, estimator)
	if tokens1 <= 0 {
		t.Errorf("Expected positive token count, got %d", tokens1)
	}

	// Second call should return cached value
	tokens2 := cache.GetOrEstimate(msg, estimator)
	if tokens1 != tokens2 {
		t.Errorf("Expected cached value to match: %d != %d", tokens1, tokens2)
	}
}

// TestMessageTokenCache_Clear tests cache clearing
func TestMessageTokenCache_Clear(t *testing.T) {
	cache := NewMessageTokenCache()
	estimator := NewTokenEstimator()

	msgID := primitive.NewObjectID()
	msg := &models.ChatMessage{
		ID:      msgID,
		Role:    "user",
		Content: "Test message",
	}

	// Cache a message
	cache.GetOrEstimate(msg, estimator)

	// Clear cache
	cache.Clear()

	// Verify cache is empty by checking internal state
	cache.mu.RLock()
	if len(cache.cache) != 0 {
		t.Errorf("Expected empty cache after Clear(), got %d entries", len(cache.cache))
	}
	cache.mu.RUnlock()
}

// TestEstimateMessageTokens_SimpleMessage tests token estimation for simple messages
func TestEstimateMessageTokens_SimpleMessage(t *testing.T) {
	estimator := NewTokenEstimator()

	msg := &models.ChatMessage{
		ID:      primitive.NewObjectID(),
		Role:    "user",
		Content: "Hello world",
	}

	tokens := estimator.EstimateMessageTokens(msg)

	// Should have at least content tokens + 10 for overhead
	expectedMin := (len("Hello world") + 3) / 4 + 10
	if tokens < expectedMin {
		t.Errorf("Expected at least %d tokens, got %d", expectedMin, tokens)
	}
}

// TestEstimateMessageTokens_WithToolCall tests token estimation with tool calls
func TestEstimateMessageTokens_WithToolCall(t *testing.T) {
	estimator := NewTokenEstimator()

	msg := &models.ChatMessage{
		ID:      primitive.NewObjectID(),
		Role:    "assistant",
		Content: "I'll help you with that",
		ToolCall: &models.ToolCallData{
			ID:   "call_123",
			Name: "code_index_search",
			Args: map[string]interface{}{
				"query": "find authentication logic",
				"limit": 10,
			},
		},
	}

	tokens := estimator.EstimateMessageTokens(msg)

	// Should include content + tool call tokens
	if tokens <= 0 {
		t.Errorf("Expected positive token count, got %d", tokens)
	}

	// Estimate without tool call for comparison
	msgNoTool := &models.ChatMessage{
		ID:      primitive.NewObjectID(),
		Role:    "assistant",
		Content: "I'll help you with that",
	}
	tokensNoTool := estimator.EstimateMessageTokens(msgNoTool)

	// With tool call should have more tokens
	if tokens <= tokensNoTool {
		t.Errorf("Expected more tokens with tool call: %d vs %d", tokens, tokensNoTool)
	}
}

// TestEstimateMessageTokens_WithToolResult tests token estimation with tool results
func TestEstimateMessageTokens_WithToolResult(t *testing.T) {
	estimator := NewTokenEstimator()

	msg := &models.ChatMessage{
		ID:      primitive.NewObjectID(),
		Role:    "tool_result",
		Content: "Tool execution result",
		ToolResult: &models.ToolResultData{
			ID:         "result_123",
			Name:       "code_index_search",
			Output:     "Found 5 matching files",
			DurationMs: 150,
		},
	}

	tokens := estimator.EstimateMessageTokens(msg)

	if tokens <= 0 {
		t.Errorf("Expected positive token count, got %d", tokens)
	}
}

// TestEstimateMessageTokens_WithError tests token estimation with tool result errors
func TestEstimateMessageTokens_WithError(t *testing.T) {
	estimator := NewTokenEstimator()

	msg := &models.ChatMessage{
		ID:      primitive.NewObjectID(),
		Role:    "tool_result",
		Content: "Tool failed",
		ToolResult: &models.ToolResultData{
			ID:         "result_123",
			Name:       "code_index_search",
			Output:     "Error occurred",
			Error:      "Connection timeout after 30 seconds",
			DurationMs: 30000,
		},
	}

	tokens := estimator.EstimateMessageTokens(msg)

	if tokens <= 0 {
		t.Errorf("Expected positive token count, got %d", tokens)
	}
}

// TestEstimateMessageTokens_CachedValue tests that cached TokenCount is used
func TestEstimateMessageTokens_CachedValue(t *testing.T) {
	estimator := NewTokenEstimator()

	msg := &models.ChatMessage{
		ID:         primitive.NewObjectID(),
		Role:       "user",
		Content:    "Test message",
		TokenCount: 42, // Pre-cached value
	}

	tokens := estimator.EstimateMessageTokens(msg)

	if tokens != 42 {
		t.Errorf("Expected cached value 42, got %d", tokens)
	}
}

// TestEstimateTotalTokens tests total token estimation for message slices
func TestEstimateTotalTokens(t *testing.T) {
	estimator := NewTokenEstimator()

	messages := []models.ChatMessage{
		{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "First message",
		},
		{
			ID:      primitive.NewObjectID(),
			Role:    "assistant",
			Content: "Second message",
		},
		{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "Third message",
		},
	}

	totalTokens := estimator.EstimateTotalTokens(messages)

	// Should be sum of individual message tokens
	expectedMin := 0
	for i := range messages {
		expectedMin += estimator.EstimateMessageTokens(&messages[i])
	}

	if totalTokens != expectedMin {
		t.Errorf("Expected %d tokens, got %d", expectedMin, totalTokens)
	}
}

// TestEstimateTotalTokens_EmptySlice tests with empty message slice
func TestEstimateTotalTokens_EmptySlice(t *testing.T) {
	estimator := NewTokenEstimator()

	messages := []models.ChatMessage{}
	totalTokens := estimator.EstimateTotalTokens(messages)

	if totalTokens != 0 {
		t.Errorf("Expected 0 tokens for empty slice, got %d", totalTokens)
	}
}

// TestEstimateTotalTokens_LargeSlice tests with many messages
func TestEstimateTotalTokens_LargeSlice(t *testing.T) {
	estimator := NewTokenEstimator()

	// Create 100 messages
	messages := make([]models.ChatMessage, 100)
	for i := 0; i < 100; i++ {
		messages[i] = models.ChatMessage{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "Message " + string(rune(i)),
		}
	}

	totalTokens := estimator.EstimateTotalTokens(messages)

	if totalTokens <= 0 {
		t.Errorf("Expected positive token count for 100 messages, got %d", totalTokens)
	}

	// Verify it's reasonable (should be at least 100 * 10 for overhead)
	if totalTokens < 1000 {
		t.Errorf("Expected at least 1000 tokens for 100 messages, got %d", totalTokens)
	}
}

// TestMessageTokenCache_Concurrency tests thread safety
func TestMessageTokenCache_Concurrency(t *testing.T) {
	cache := NewMessageTokenCache()
	estimator := NewTokenEstimator()

	// Create multiple messages
	messages := make([]*models.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = &models.ChatMessage{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "Message " + string(rune(i)),
		}
	}

	// Simulate concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				cache.GetOrEstimate(messages[idx], estimator)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all messages are cached
	cache.mu.RLock()
	if len(cache.cache) != 10 {
		t.Errorf("Expected 10 cached messages, got %d", len(cache.cache))
	}
	cache.mu.RUnlock()
}

// TestEstimateMessageTokens_ComplexToolCall tests with complex tool arguments
func TestEstimateMessageTokens_ComplexToolCall(t *testing.T) {
	estimator := NewTokenEstimator()

	msg := &models.ChatMessage{
		ID:      primitive.NewObjectID(),
		Role:    "assistant",
		Content: "Analyzing code",
		ToolCall: &models.ToolCallData{
			ID:   "call_complex",
			Name: "analyze_code",
			Args: map[string]interface{}{
				"files": []string{
					"src/main.go",
					"src/handlers.go",
					"src/models.go",
				},
				"options": map[string]interface{}{
					"depth":     3,
					"recursive": true,
					"filters": []string{
						"*.go",
						"*.test.go",
					},
				},
			},
		},
	}

	tokens := estimator.EstimateMessageTokens(msg)

	if tokens <= 0 {
		t.Errorf("Expected positive token count, got %d", tokens)
	}
}

// BenchmarkEstimateMessageTokens benchmarks message token estimation
func BenchmarkEstimateMessageTokens(b *testing.B) {
	estimator := NewTokenEstimator()

	msg := &models.ChatMessage{
		ID:      primitive.NewObjectID(),
		Role:    "user",
		Content: "This is a test message for benchmarking token estimation",
		ToolCall: &models.ToolCallData{
			ID:   "call_123",
			Name: "test_tool",
			Args: map[string]interface{}{
				"param1": "value1",
				"param2": 42,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateMessageTokens(msg)
	}
}

// BenchmarkMessageTokenCache benchmarks cache performance
func BenchmarkMessageTokenCache(b *testing.B) {
	cache := NewMessageTokenCache()
	estimator := NewTokenEstimator()

	msg := &models.ChatMessage{
		ID:      primitive.NewObjectID(),
		Role:    "user",
		Content: "Test message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetOrEstimate(msg, estimator)
	}
}

// BenchmarkEstimateTotalTokens benchmarks total token estimation
func BenchmarkEstimateTotalTokens(b *testing.B) {
	estimator := NewTokenEstimator()

	messages := make([]models.ChatMessage, 50)
	for i := 0; i < 50; i++ {
		messages[i] = models.ChatMessage{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "Message " + string(rune(i)),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateTotalTokens(messages)
	}
}
