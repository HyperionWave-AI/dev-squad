package utils

import (
	"context"
	"testing"
	"time"

	"hyper/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// TestMessageSummarizer tests the message summarizer functionality
func TestMessageSummarizer(t *testing.T) {
	logger := zap.NewNop()
	ms := NewMessageSummarizer(logger)

	// Create test messages
	now := time.Now()
	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Can you help me implement a feature?",
			Timestamp: now.Add(-2 * time.Hour),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "Sure! What feature would you like to implement?",
			Timestamp: now.Add(-2 * time.Hour),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "I need to add user authentication",
			Timestamp: now.Add(-1 * time.Hour),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "I can help with that. Let me provide some guidance.",
			Timestamp: now.Add(-1 * time.Hour),
		},
	}

	// Test IdentifyMessagesForSummarization
	toSummarize := ms.IdentifyMessagesForSummarization(messages, 1)
	if len(toSummarize) != 3 {
		t.Errorf("IdentifyMessagesForSummarization returned %d messages, want 3", len(toSummarize))
	}

	// Test GroupMessagesByTimeWindow
	groups := ms.GroupMessagesByTimeWindow(messages, 1*time.Hour)
	if len(groups) == 0 {
		t.Errorf("GroupMessagesByTimeWindow returned 0 groups, want > 0")
	}

	// Test SummarizeMessages with TimeWindow strategy
	result, err := ms.SummarizeMessages(context.Background(), messages, StrategyTimeWindow)
	if err != nil {
		t.Errorf("SummarizeMessages failed: %v", err)
	}
	if result == nil {
		t.Fatal("SummarizeMessages returned nil result")
	}
	if result.Strategy != StrategyTimeWindow {
		t.Errorf("SummarizeMessages Strategy = %s, want %s", result.Strategy, StrategyTimeWindow)
	}
}

// TestGroupMessagesByTimeWindow tests time window grouping
func TestGroupMessagesByTimeWindow(t *testing.T) {
	logger := zap.NewNop()
	ms := NewMessageSummarizer(logger)

	now := time.Now()
	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Message 1",
			Timestamp: now.Add(-2 * time.Hour),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "Message 2",
			Timestamp: now.Add(-1 * time.Hour),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Message 3",
			Timestamp: now,
		},
	}

	groups := ms.GroupMessagesByTimeWindow(messages, 1*time.Hour)
	if len(groups) == 0 {
		t.Errorf("GroupMessagesByTimeWindow returned 0 groups, want > 0")
	}
}

// TestGenerateSummary tests summary generation
func TestGenerateSummary(t *testing.T) {
	logger := zap.NewNop()
	ms := NewMessageSummarizer(logger)

	now := time.Now()
	group := MessageGroup{
		Messages: []models.ChatMessage{
			{
				ID:        primitive.NewObjectID(),
				SessionID: primitive.NewObjectID(),
				Role:      "user",
				Content:   "How do I implement authentication?",
				Timestamp: now.Add(-1 * time.Hour),
			},
			{
				ID:        primitive.NewObjectID(),
				SessionID: primitive.NewObjectID(),
				Role:      "assistant",
				Content:   "You can use JWT tokens for authentication.",
				Timestamp: now,
			},
		},
		TimeWindowStart: now.Add(-1 * time.Hour),
		TimeWindowEnd:   now,
	}

	summary := ms.GenerateSummary(context.Background(), group)
	if summary == "" {
		t.Errorf("GenerateSummary returned empty string")
	}
}

// TestCalculateSummarizationImpact tests impact calculation
func TestCalculateSummarizationImpact(t *testing.T) {
	logger := zap.NewNop()
	ms := NewMessageSummarizer(logger)

	now := time.Now()
	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Message 1",
			Timestamp: now.Add(-2 * time.Hour),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "assistant",
			Content:   "Message 2",
			Timestamp: now.Add(-1 * time.Hour),
		},
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Message 3",
			Timestamp: now,
		},
	}

	_, messageCount := ms.CalculateSummarizationImpact(messages, 1)
	if messageCount != 2 {
		t.Errorf("CalculateSummarizationImpact messageCount = %d, want 2", messageCount)
	}
}

// TestSummarizationConfig tests summarization configuration
func TestSummarizationConfig(t *testing.T) {
	config := DefaultSummarizationConfig()

	if !config.Enabled {
		t.Errorf("DefaultSummarizationConfig Enabled = %v, want true", config.Enabled)
	}
	if config.KeepRecentMinutes <= 0 {
		t.Errorf("DefaultSummarizationConfig KeepRecentMinutes = %d, want > 0", config.KeepRecentMinutes)
	}
	if config.MaxSummaryTokens <= 0 {
		t.Errorf("DefaultSummarizationConfig MaxSummaryTokens = %d, want > 0", config.MaxSummaryTokens)
	}
	if config.AutoSummarizeThreshold <= 0 {
		t.Errorf("DefaultSummarizationConfig AutoSummarizeThreshold = %f, want > 0", config.AutoSummarizeThreshold)
	}
}

// TestSummarizationPrompt tests prompt generation
func TestSummarizationPrompt(t *testing.T) {
	prompt := SummarizationPrompt()
	if prompt == "" {
		t.Errorf("SummarizationPrompt returned empty string")
	}
	if len(prompt) < 50 {
		t.Errorf("SummarizationPrompt too short: %q", prompt)
	}
}

// TestSummarizationUserPrompt tests user prompt generation
func TestSummarizationUserPrompt(t *testing.T) {
	messages := []string{
		"User: How do I implement authentication?",
		"Assistant: You can use JWT tokens.",
	}

	prompt := SummarizationUserPrompt(messages)
	if prompt == "" {
		t.Errorf("SummarizationUserPrompt returned empty string")
	}
	if len(prompt) < 50 {
		t.Errorf("SummarizationUserPrompt too short: %q", prompt)
	}
}
