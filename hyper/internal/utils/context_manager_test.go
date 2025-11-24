package utils

import (
	"context"
	"testing"
	"time"

	"hyper/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// TestContextManager tests the context manager functionality
func TestContextManager(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultContextLimitConfig()
	cm := NewContextManager(config, logger)

	sessionID := primitive.NewObjectID().Hex()

	// Create test messages
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
			Content:   "I'm doing well, thank you for asking! How can I help you today?",
			Timestamp: time.Now(),
		},
	}

	// Test UpdateContextUsage
	usage := cm.UpdateContextUsage(context.Background(), sessionID, messages)
	if usage == nil {
		t.Fatal("UpdateContextUsage returned nil")
	}
	if usage.TotalTokens <= 0 {
		t.Errorf("UpdateContextUsage TotalTokens = %d, want > 0", usage.TotalTokens)
	}
	if usage.PercentageUsed <= 0 {
		t.Errorf("UpdateContextUsage PercentageUsed = %f, want > 0", usage.PercentageUsed)
	}

	// Test GetContextUsage
	retrieved := cm.GetContextUsage(sessionID)
	if retrieved == nil {
		t.Fatal("GetContextUsage returned nil")
	}
	if retrieved.TotalTokens != usage.TotalTokens {
		t.Errorf("GetContextUsage TotalTokens = %d, want %d", retrieved.TotalTokens, usage.TotalTokens)
	}

	// Test CheckContextHealth
	health := cm.CheckContextHealth(sessionID)
	if health != ContextHealthHealthy {
		t.Errorf("CheckContextHealth = %s, want %s", health, ContextHealthHealthy)
	}

	// Test ClearSessionUsage
	cm.ClearSessionUsage(sessionID)
	cleared := cm.GetContextUsage(sessionID)
	if cleared.TotalTokens != 0 {
		t.Errorf("After ClearSessionUsage, TotalTokens = %d, want 0", cleared.TotalTokens)
	}
}

// TestContextLimitThresholds tests warning and critical thresholds
func TestContextLimitThresholds(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultContextLimitConfig()
	cm := NewContextManager(config, logger)

	sessionID := primitive.NewObjectID().Hex()

	// Create messages that will exceed warning threshold
	// Each message is roughly 50 tokens, so we need ~1600 messages to reach 80K tokens
	// For testing, we'll create a smaller set and verify the logic

	// Test with low usage (should be healthy)
	lowMessages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Hi",
			Timestamp: time.Now(),
		},
	}

	usage := cm.UpdateContextUsage(context.Background(), sessionID, lowMessages)
	if usage.IsWarning || usage.IsCritical {
		t.Errorf("Low usage should not trigger warning or critical")
	}

	// Test CanAddMessage
	canAdd, _ := cm.CanAddMessage(sessionID, "test message")
	if !canAdd {
		t.Errorf("CanAddMessage should return true for low usage")
	}
}

// TestContextError tests context error creation
func TestContextError(t *testing.T) {
	usage := &ContextUsage{
		SessionID:      "test-session",
		TotalTokens:    95000,
		MaxTokens:      100000,
		PercentageUsed: 95.0,
		IsCritical:     true,
		IsWarning:      false,
	}

	err := NewContextError("CONTEXT_LIMIT_EXCEEDED", "Context limit reached", usage)
	if err == nil {
		t.Fatal("NewContextError returned nil")
	}
	if err.Code != "CONTEXT_LIMIT_EXCEEDED" {
		t.Errorf("Error code = %s, want CONTEXT_LIMIT_EXCEEDED", err.Code)
	}
	if err.CurrentTokens != 95000 {
		t.Errorf("Error CurrentTokens = %d, want 95000", err.CurrentTokens)
	}
	if !err.CanArchiveMessages {
		t.Errorf("Error CanArchiveMessages should be true for critical usage")
	}
	if len(err.RecoveryOptions) == 0 {
		t.Errorf("Error RecoveryOptions should not be empty for critical usage")
	}

	// Test error string
	errStr := err.Error()
	if errStr == "" {
		t.Errorf("Error.Error() returned empty string")
	}
}

// TestGetAllSessionUsage tests retrieving all session usage
func TestGetAllSessionUsage(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultContextLimitConfig()
	cm := NewContextManager(config, logger)

	// Create multiple sessions
	session1 := primitive.NewObjectID().Hex()
	session2 := primitive.NewObjectID().Hex()

	messages := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "Test message",
			Timestamp: time.Now(),
		},
	}

	cm.UpdateContextUsage(context.Background(), session1, messages)
	cm.UpdateContextUsage(context.Background(), session2, messages)

	allUsage := cm.GetAllSessionUsage()
	if len(allUsage) != 2 {
		t.Errorf("GetAllSessionUsage returned %d sessions, want 2", len(allUsage))
	}
}
