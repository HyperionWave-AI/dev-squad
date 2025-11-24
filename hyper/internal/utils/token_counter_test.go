package utils

import (
	"testing"
	"time"

	"hyper/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestTokenCounter tests the token counting functionality
func TestTokenCounter(t *testing.T) {
	tc := NewTokenCounter()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "short text",
			text:     "hello",
			expected: 1, // 5 chars * 0.25 = 1.25, rounded to 1
		},
		{
			name:     "medium text",
			text:     "This is a test message with multiple words",
			expected: 10, // 42 chars * 0.25 = 10.5
		},
		{
			name:     "long text",
			text:     "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
			expected: 30, // 123 chars * 0.25 = 30.75
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tc.CountTokens(tt.text)
			if result != tt.expected {
				t.Errorf("CountTokens(%q) = %d, want %d", tt.text, result, tt.expected)
			}
		})
	}
}

// TestCountMessageTokens tests message token counting
func TestCountMessageTokens(t *testing.T) {
	tc := NewTokenCounter()

	msg := &models.ChatMessage{
		ID:        primitive.NewObjectID(),
		SessionID: primitive.NewObjectID(),
		Role:      "user",
		Content:   "Hello, how are you?",
		Timestamp: time.Now(),
	}

	tokens := tc.CountMessageTokens(msg)
	if tokens <= 0 {
		t.Errorf("CountMessageTokens() = %d, want > 0", tokens)
	}
}

// TestCountSessionTokens tests session token counting
func TestCountSessionTokens(t *testing.T) {
	tc := NewTokenCounter()

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
			Content:   "Hi there! How can I help you?",
			Timestamp: time.Now(),
		},
	}

	tokens := tc.CountSessionTokens(messages)
	if tokens <= 0 {
		t.Errorf("CountSessionTokens() = %d, want > 0", tokens)
	}
}

// TestFormatTokenCount tests token count formatting
func TestFormatTokenCount(t *testing.T) {
	tc := NewTokenCounter()

	tests := []struct {
		tokens   int
		expected string
	}{
		{100, "100"},
		{1000, "1.0K"},
		{50000, "50.0K"},
		{100000, "100.0K"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tc.FormatTokenCount(tt.tokens)
			if result != tt.expected {
				t.Errorf("FormatTokenCount(%d) = %q, want %q", tt.tokens, result, tt.expected)
			}
		})
	}
}

// TestCalculatePercentage tests percentage calculation
func TestCalculatePercentage(t *testing.T) {
	tc := NewTokenCounter()

	tests := []struct {
		used     int
		max      int
		expected float64
	}{
		{50, 100, 50.0},
		{80, 100, 80.0},
		{100, 100, 100.0},
		{0, 100, 0.0},
		{0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tc.CalculatePercentage(tt.used, tt.max)
			if result != tt.expected {
				t.Errorf("CalculatePercentage(%d, %d) = %f, want %f", tt.used, tt.max, result, tt.expected)
			}
		})
	}
}
