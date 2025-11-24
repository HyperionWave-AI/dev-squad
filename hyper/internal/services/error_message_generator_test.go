package services

import (
	"testing"

	"hyper/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestErrorMessageGenerator_GenerateContextError(t *testing.T) {
	maxTokens := 100000
	gen := NewErrorMessageGenerator(maxTokens)

	tests := []struct {
		name              string
		tokenCount        int
		expectedCode      models.ContextErrorCode
		expectedHasOptions bool
	}{
		{
			name:              "Context full (100%)",
			tokenCount:        100000,
			expectedCode:      models.ContextFull,
			expectedHasOptions: true,
		},
		{
			name:              "Context critical (95%)",
			tokenCount:        95000,
			expectedCode:      models.ContextCritical,
			expectedHasOptions: true,
		},
		{
			name:              "Context warning (85%)",
			tokenCount:        85000,
			expectedCode:      models.ContextWarning,
			expectedHasOptions: true,
		},
		{
			name:              "Context normal (50%)",
			tokenCount:        50000,
			expectedCode:      models.ContextWarning,
			expectedHasOptions: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentage := float64(tt.tokenCount) / float64(maxTokens) * 100
			err := gen.GenerateContextError(tt.tokenCount, percentage)

			assert.NotNil(t, err)
			assert.Equal(t, tt.expectedCode, err.Code)
			assert.NotEmpty(t, err.Message)
			assert.NotEmpty(t, err.Suggestion)
			assert.NotNil(t, err.ContextMetadata)
			assert.Equal(t, tt.tokenCount, err.ContextMetadata.TokenCount)
			assert.Equal(t, maxTokens, err.ContextMetadata.MaxTokens)

			if tt.expectedHasOptions {
				assert.Greater(t, len(err.RecoveryOptions), 0)
			}
		})
	}
}

func TestErrorMessageGenerator_GenerateContextExceededError(t *testing.T) {
	maxTokens := 100000
	gen := NewErrorMessageGenerator(maxTokens)

	currentTokens := 95000
	attemptedTokens := 10000

	err := gen.GenerateContextExceededError(currentTokens, attemptedTokens)

	assert.NotNil(t, err)
	assert.Equal(t, models.ContextFull, err.Code)
	assert.NotEmpty(t, err.Message)
	assert.NotEmpty(t, err.Suggestion)
	assert.Greater(t, len(err.RecoveryOptions), 0)
	assert.NotNil(t, err.ContextMetadata)
	assert.True(t, err.ContextMetadata.IsFull)
}

func TestErrorMessageGenerator_GenerateSummarizationError(t *testing.T) {
	gen := NewErrorMessageGenerator(100000)

	err := gen.GenerateSummarizationError()

	assert.NotNil(t, err)
	assert.Equal(t, models.SummarizationFail, err.Code)
	assert.NotEmpty(t, err.Message)
	assert.NotEmpty(t, err.Suggestion)
	assert.Greater(t, len(err.RecoveryOptions), 0)
}

func TestErrorMessageGenerator_ContextMetadata(t *testing.T) {
	maxTokens := 100000
	gen := NewErrorMessageGenerator(maxTokens)

	tests := []struct {
		name           string
		tokenCount     int
		expectedNear   bool
		expectedFull   bool
	}{
		{
			name:           "Below 80%",
			tokenCount:     70000,
			expectedNear:   false,
			expectedFull:   false,
		},
		{
			name:           "At 80%",
			tokenCount:     80000,
			expectedNear:   true,
			expectedFull:   false,
		},
		{
			name:           "At 100%",
			tokenCount:     100000,
			expectedNear:   true,
			expectedFull:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentage := float64(tt.tokenCount) / float64(maxTokens) * 100
			err := gen.GenerateContextError(tt.tokenCount, percentage)

			assert.Equal(t, tt.expectedNear, err.ContextMetadata.IsNearLimit)
			assert.Equal(t, tt.expectedFull, err.ContextMetadata.IsFull)
		})
	}
}

func TestErrorMessageGenerator_RecoveryOptions(t *testing.T) {
	maxTokens := 100000
	gen := NewErrorMessageGenerator(maxTokens)

	// Test full context has all recovery options
	err := gen.GenerateContextError(100000, 100)
	assert.Equal(t, 3, len(err.RecoveryOptions))
	assert.Equal(t, "archive", err.RecoveryOptions[0].Action)
	assert.Equal(t, "new_chat", err.RecoveryOptions[1].Action)
	assert.Equal(t, "clear", err.RecoveryOptions[2].Action)

	// Test critical context has summarize option
	err = gen.GenerateContextError(95000, 95)
	assert.Equal(t, 3, len(err.RecoveryOptions))
	hasArchive := false
	hasSummarize := false
	hasNewChat := false
	for _, opt := range err.RecoveryOptions {
		if opt.Action == "archive" {
			hasArchive = true
		}
		if opt.Action == "summarize" {
			hasSummarize = true
		}
		if opt.Action == "new_chat" {
			hasNewChat = true
		}
	}
	assert.True(t, hasArchive)
	assert.True(t, hasSummarize)
	assert.True(t, hasNewChat)
}

func TestErrorMessageGenerator_GetRecoveryActionDescription(t *testing.T) {
	gen := NewErrorMessageGenerator(100000)

	tests := []struct {
		action   string
		expected string
	}{
		{"archive", "Archive old messages to free up context space"},
		{"new_chat", "Start a new conversation with empty context"},
		{"summarize", "Automatically summarize old messages to compress context"},
		{"clear", "Clear all messages and reset context"},
		{"unknown", "Take action to resolve context issue"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			desc := gen.GetRecoveryActionDescription(tt.action)
			assert.Equal(t, tt.expected, desc)
		})
	}
}
