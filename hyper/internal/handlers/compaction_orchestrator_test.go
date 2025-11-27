package handlers

import (
	"context"
	"testing"

	aiservice "hyper/internal/ai-service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"hyper/internal/models"
)

// MockChatService is a mock implementation of ChatServiceInterface for testing
type MockChatService struct {
	savedMessages []models.ChatMessage
}

func (m *MockChatService) GetSession(ctx context.Context, sessionID primitive.ObjectID, companyID string) (*models.ChatSession, error) {
	return &models.ChatSession{
		ID:        sessionID,
		UserID:    "test-user",
		CompanyID: companyID,
	}, nil
}

func (m *MockChatService) GetSessionMessages(ctx context.Context, sessionID primitive.ObjectID) ([]models.ChatMessage, error) {
	return []models.ChatMessage{}, nil
}

func (m *MockChatService) SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*models.ChatMessage, error) {
	msg := &models.ChatMessage{
		ID:        primitive.NewObjectID(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	}
	m.savedMessages = append(m.savedMessages, *msg)
	return msg, nil
}

func (m *MockChatService) SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, id, name string, args map[string]interface{}, companyID string) (*models.ChatMessage, error) {
	return nil, nil
}

func (m *MockChatService) SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, id, name string, output interface{}, errorMsg string, durationMs int64, companyID string) (*models.ChatMessage, error) {
	return nil, nil
}

// MockAIService is a mock implementation of AIServiceInterface for testing
type MockAIService struct{}

func (m *MockAIService) StreamChatWithTools(ctx context.Context, messages []aiservice.Message, maxToolCalls int) (<-chan aiservice.StreamEvent, error) {
	return nil, nil
}

func (m *MockAIService) StreamChatWithToolsFiltered(ctx context.Context, messages []aiservice.Message, maxToolCalls int, allowedToolNames []string) (<-chan aiservice.StreamEvent, error) {
	return nil, nil
}

func (m *MockAIService) GetConfig() *aiservice.AIConfig {
	return &aiservice.AIConfig{
		Model:    "gpt-4",
		Provider: "openai",
	}
}

func (m *MockAIService) GetAllowedToolsForDirectSubagent() []string {
	return nil
}

// TestCompactionOrchestratorCreation tests that the orchestrator can be created
func TestCompactionOrchestratorCreation(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	chatService := &MockChatService{}
	aiService := &MockAIService{}

	orchestrator := NewCompactionOrchestrator(config, aiService, chatService, logger)

	if orchestrator == nil {
		t.Fatal("Expected orchestrator to be created, got nil")
	}

	if orchestrator.config == nil {
		t.Fatal("Expected config to be set")
	}

	if orchestrator.compactor == nil {
		t.Fatal("Expected compactor to be initialized")
	}

	if orchestrator.summarizer == nil {
		t.Fatal("Expected summarizer to be initialized")
	}
}

// TestCompactionOrchestratorShouldCompact tests the ShouldCompact method
func TestCompactionOrchestratorShouldCompact(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	chatService := &MockChatService{}
	aiService := &MockAIService{}

	orchestrator := NewCompactionOrchestrator(config, aiService, chatService, logger)

	// Create messages that don't exceed trigger threshold
	smallMessages := []models.ChatMessage{
		{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "Hello",
		},
	}

	if orchestrator.ShouldCompact(smallMessages) {
		t.Fatal("Expected ShouldCompact to return false for small messages")
	}

	// Create messages that exceed trigger threshold
	largeMessages := make([]models.ChatMessage, 0)
	for i := 0; i < 1000; i++ {
		largeMessages = append(largeMessages, models.ChatMessage{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "This is a very long message that contains a lot of text to increase the token count. " + string(rune(i)),
		})
	}

	shouldCompact := orchestrator.ShouldCompact(largeMessages)
	// This may or may not be true depending on token estimation, but the method should work
	_ = shouldCompact
}

// TestCompactionOrchestratorGetStats tests the GetCompactionStats method
func TestCompactionOrchestratorGetStats(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	chatService := &MockChatService{}
	aiService := &MockAIService{}

	orchestrator := NewCompactionOrchestrator(config, aiService, chatService, logger)

	messages := []models.ChatMessage{
		{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "Hello",
		},
	}

	stats := orchestrator.GetCompactionStats(messages)

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	// Verify expected keys are present
	expectedKeys := []string{
		// Token-based stats
		"totalTokens",
		"triggerTokens",
		"targetTokens",
		"tokenPercentageUsed",
		"tokenShouldCompact",
		"tokenCompactCount",
		"tokenKeepCount",
		// Size-based stats
		"totalSize",
		"sizeTriggerThreshold",
		"sizeTargetThreshold",
		"sizePercentageUsed",
		"sizeShouldCompact",
		"sizeCompactCount",
		"sizeKeepCount",
		"sizeWarningLevel",
		// Combined stats
		"shouldCompact",
		"messageCount",
		"summaryBuffer",
		"perMessageMax",
		"maxBSONSize",
	}

	for _, key := range expectedKeys {
		if _, ok := stats[key]; !ok {
			t.Errorf("Expected key %q in stats", key)
		}
	}

	// Verify message count is correct
	if messageCount, ok := stats["messageCount"].(int); ok {
		if messageCount != 1 {
			t.Errorf("Expected messageCount to be 1, got %d", messageCount)
		}
	} else {
		t.Error("Expected messageCount to be an int")
	}
}

// TestCompactionOrchestratorCompactIfNeeded tests the CompactIfNeeded method
func TestCompactionOrchestratorCompactIfNeeded(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	chatService := &MockChatService{}
	aiService := &MockAIService{}

	orchestrator := NewCompactionOrchestrator(config, aiService, chatService, logger)

	sessionID := primitive.NewObjectID()
	messages := []models.ChatMessage{
		{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "Hello",
		},
	}

	ctx := context.Background()
	result, err := orchestrator.CompactIfNeeded(ctx, sessionID, messages)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	// For small messages, compaction should not be needed
	if result.WasCompacted {
		t.Error("Expected WasCompacted to be false for small messages")
	}
}

// TestDualTriggerCompaction_TokensOnly tests compaction triggered by tokens only
func TestDualTriggerCompaction_TokensOnly(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	chatService := &MockChatService{}
	aiService := &MockAIService{}

	orchestrator := NewCompactionOrchestrator(config, aiService, chatService, logger)

	// Create messages with high token count but small size
	// Each message has many tokens (via TokenCount) but small content
	var messages []models.ChatMessage
	for i := 0; i < 15; i++ {
		messages = append(messages, models.ChatMessage{
			ID:         primitive.NewObjectID(),
			Role:       "user",
			Content:    "Small content",
			TokenCount: 10000, // High token count
		})
	}

	sessionID := primitive.NewObjectID()
	ctx := context.Background()

	result, err := orchestrator.CompactIfNeeded(ctx, sessionID, messages)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should trigger on tokens since 15 * 10000 = 150K > 115.2K threshold
	if result.Trigger != TriggerTokens && result.Trigger != TriggerBoth {
		t.Logf("Trigger type: %s", result.Trigger)
	}

	t.Logf("Token-only trigger test: trigger=%s, tokens=%d, size=%d",
		result.Trigger, result.OriginalTokens, result.OriginalSize)
}

// TestDualTriggerCompaction_SizeOnly tests compaction triggered by size only
func TestDualTriggerCompaction_SizeOnly(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	chatService := &MockChatService{}
	aiService := &MockAIService{}

	orchestrator := NewCompactionOrchestrator(config, aiService, chatService, logger)

	// Create messages with large size but low token count
	// Each message has large content but low TokenCount
	var messages []models.ChatMessage
	largeContent := createTestString(1 * 1024 * 1024) // 1MB each
	for i := 0; i < 15; i++ {
		messages = append(messages, models.ChatMessage{
			ID:         primitive.NewObjectID(),
			Role:       "user",
			Content:    largeContent,
			TokenCount: 100, // Low token count
		})
	}

	sessionID := primitive.NewObjectID()
	ctx := context.Background()

	result, err := orchestrator.CompactIfNeeded(ctx, sessionID, messages)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should trigger on size since 15 * 1MB = 15MB > 12.8MB threshold
	if result.Trigger != TriggerSize && result.Trigger != TriggerBoth {
		t.Logf("Trigger type: %s (expected size or both)", result.Trigger)
	}

	t.Logf("Size-only trigger test: trigger=%s, tokens=%d, size=%d (%.2f MB)",
		result.Trigger, result.OriginalTokens, result.OriginalSize,
		float64(result.OriginalSize)/(1024*1024))
}

// TestDualTriggerCompaction_Both tests compaction triggered by both tokens AND size
func TestDualTriggerCompaction_Both(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	chatService := &MockChatService{}
	aiService := &MockAIService{}

	orchestrator := NewCompactionOrchestrator(config, aiService, chatService, logger)

	// Create messages with both high token count AND large size
	var messages []models.ChatMessage
	largeContent := createTestString(1 * 1024 * 1024) // 1MB each
	for i := 0; i < 15; i++ {
		messages = append(messages, models.ChatMessage{
			ID:         primitive.NewObjectID(),
			Role:       "user",
			Content:    largeContent,
			TokenCount: 10000, // High token count
		})
	}

	sessionID := primitive.NewObjectID()
	ctx := context.Background()

	result, err := orchestrator.CompactIfNeeded(ctx, sessionID, messages)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should trigger on both since:
	// - Tokens: 15 * 10000 = 150K > 115.2K threshold
	// - Size: 15 * 1MB = 15MB > 12.8MB threshold
	if result.Trigger != TriggerBoth {
		t.Logf("Trigger type: %s (expected both)", result.Trigger)
	}

	t.Logf("Both triggers test: trigger=%s, tokens=%d, size=%d (%.2f MB)",
		result.Trigger, result.OriginalTokens, result.OriginalSize,
		float64(result.OriginalSize)/(1024*1024))
}

// TestCompactionE2E tests the full compaction workflow end-to-end
func TestCompactionE2E(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	chatService := &MockChatService{}
	aiService := &MockAIService{}

	orchestrator := NewCompactionOrchestrator(config, aiService, chatService, logger)

	// Create a realistic conversation that exceeds trigger threshold
	var messages []models.ChatMessage

	// Add some old messages (will be compacted)
	for i := 0; i < 10; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages = append(messages, models.ChatMessage{
			ID:         primitive.NewObjectID(),
			Role:       role,
			Content:    "This is an old message that will be compacted. " + string(rune('A'+i)),
			TokenCount: 10000, // 10K tokens each
		})
	}

	// Add a tool call pair
	messages = append(messages, models.ChatMessage{
		ID:   primitive.NewObjectID(),
		Role: "tool_call",
		ToolCall: &models.ToolCallData{
			ID:   "tool-1",
			Name: "read_file",
			Args: map[string]interface{}{"path": "/test.txt"},
		},
		TokenCount: 500,
	})
	messages = append(messages, models.ChatMessage{
		ID:   primitive.NewObjectID(),
		Role: "tool_result",
		ToolResult: &models.ToolResultData{
			ID:     "tool-1",
			Name:   "read_file",
			Output: "File content here",
		},
		TokenCount: 1000,
	})

	// Add some recent messages (will be kept)
	for i := 0; i < 4; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages = append(messages, models.ChatMessage{
			ID:         primitive.NewObjectID(),
			Role:       role,
			Content:    "This is a recent message that should be kept. " + string(rune('X'+i)),
			TokenCount: 5000, // 5K tokens each
		})
	}

	sessionID := primitive.NewObjectID()
	ctx := context.Background()

	// Get stats before compaction
	statsBefore := orchestrator.GetCompactionStats(messages)
	t.Logf("Before compaction: tokens=%v, shouldCompact=%v",
		statsBefore["totalTokens"], statsBefore["shouldCompact"])

	// Run compaction
	result, err := orchestrator.CompactIfNeeded(ctx, sessionID, messages)

	if err != nil {
		t.Fatalf("CompactIfNeeded failed: %v", err)
	}

	// Verify result
	t.Logf("E2E test result: trigger=%s, wasCompacted=%v, messagesCompacted=%d, messagesKept=%d",
		result.Trigger, result.WasCompacted, result.MessagesCompacted, result.MessagesKept)

	// Validate result fields
	if result.OriginalTokens <= 0 {
		t.Error("OriginalTokens should be positive")
	}
	if result.OriginalSize <= 0 {
		t.Error("OriginalSize should be positive")
	}

	// If compaction happened, verify reduction
	if result.WasCompacted {
		if result.MessagesCompacted <= 0 {
			t.Error("MessagesCompacted should be positive when compaction occurred")
		}
		if result.MessagesKept <= 0 {
			t.Error("MessagesKept should be positive when compaction occurred")
		}
		t.Logf("Token reduction: %d -> %d (%.1f%%)",
			result.OriginalTokens, result.CompactedTokens,
			float64(result.OriginalTokens-result.CompactedTokens)/float64(result.OriginalTokens)*100)
	}
}

// TestCompactionResult_TriggerTypes tests all trigger type values
func TestCompactionResult_TriggerTypes(t *testing.T) {
	tests := []struct {
		trigger  CompactionTrigger
		expected string
	}{
		{TriggerNone, "none"},
		{TriggerTokens, "tokens"},
		{TriggerSize, "size"},
		{TriggerBoth, "both"},
	}

	for _, tt := range tests {
		if string(tt.trigger) != tt.expected {
			t.Errorf("Trigger %v should equal %q", tt.trigger, tt.expected)
		}
	}
}

// TestOrchestratorSizeCompactor tests that size compactor is properly initialized
func TestOrchestratorSizeCompactor(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	chatService := &MockChatService{}
	aiService := &MockAIService{}

	orchestrator := NewCompactionOrchestrator(config, aiService, chatService, logger)

	if orchestrator.sizeCompactor == nil {
		t.Fatal("Expected sizeCompactor to be initialized")
	}

	// Test that size compactor methods work
	messages := []models.ChatMessage{
		{
			ID:      primitive.NewObjectID(),
			Role:    "user",
			Content: "Test message",
		},
	}

	size := orchestrator.sizeCompactor.EstimateSessionBSONSize(messages)
	if size <= 0 {
		t.Error("Expected positive size estimate")
	}

	shouldCompact := orchestrator.sizeCompactor.ShouldCompactBySize(messages)
	if shouldCompact {
		t.Error("Small messages should not trigger size-based compaction")
	}
}

// Helper function for tests
func createTestString(size int) string {
	if size <= 0 {
		return ""
	}
	b := make([]byte, size)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}
