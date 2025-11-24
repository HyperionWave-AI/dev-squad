package services

import (
	"fmt"
	"time"

	"hyper/internal/models"
)

// ErrorMessageGenerator generates user-friendly error messages for context-related issues
// ErrorMessageGenerator generates user-friendly error messages for context-related issues
type ErrorMessageGenerator struct {
	maxTokens int
}

// NewErrorMessageGenerator creates a new error message generator
func NewErrorMessageGenerator(maxTokens int) *ErrorMessageGenerator {
	return &ErrorMessageGenerator{
		maxTokens: maxTokens,
	}
}

// GenerateContextError creates a context error with user-friendly message and recovery options
func (g *ErrorMessageGenerator) GenerateContextError(
	tokenCount int,
	percentage float64,
) *models.ContextError {
	contextMeta := &models.ContextMetadata{
		TokenCount:     tokenCount,
		MaxTokens:      g.maxTokens,
		PercentageUsed: percentage,
		IsNearLimit:    percentage >= 80,
		IsFull:         percentage >= 100,
		LastUpdated:    time.Now(),
	}

	// Determine error code and message based on percentage
	var code models.ContextErrorCode
	var message string
	var suggestion string
	var recoveryOptions []models.RecoveryOption

	if percentage >= 100 {
		code = models.ContextFull
		message = fmt.Sprintf(
			"Context limit reached (%dK / %dK tokens)",
			tokenCount/1000,
			g.maxTokens/1000,
		)
		suggestion = "Your conversation has reached its maximum context capacity. " +
			"You must take action before continuing."
		recoveryOptions = []models.RecoveryOption{
			{
				Label:       "Archive Old Messages",
				Action:      "archive",
				Description: "Move older messages to archive to free up context space",
			},
			{
				Label:       "Start New Conversation",
				Action:      "new_chat",
				Description: "Begin a fresh conversation with empty context",
			},
			{
				Label:       "Clear All Messages",
				Action:      "clear",
				Description: "Delete all messages and reset context (cannot be undone)",
			},
		}
	} else if percentage >= 90 {
		code = models.ContextCritical
		message = fmt.Sprintf(
			"Context limit nearly reached (%dK / %dK tokens, %.1f%%)",
			tokenCount/1000,
			g.maxTokens/1000,
			percentage,
		)
		suggestion = "Your conversation is using 90% of available context. " +
			"You should archive old messages or start a new conversation soon."
		recoveryOptions = []models.RecoveryOption{
			{
				Label:       "Archive Old Messages",
				Action:      "archive",
				Description: "Move older messages to archive to free up context space",
			},
			{
				Label:       "Summarize Conversation",
				Action:      "summarize",
				Description: "Automatically summarize old messages to compress context",
			},
			{
				Label:       "Start New Conversation",
				Action:      "new_chat",
				Description: "Begin a fresh conversation with empty context",
			},
		}
	} else if percentage >= 80 {
		code = models.ContextWarning
		message = fmt.Sprintf(
			"Context usage is high (%dK / %dK tokens, %.1f%%)",
			tokenCount/1000,
			g.maxTokens/1000,
			percentage,
		)
		suggestion = "Your conversation is using a significant portion of available context. " +
			"Consider archiving old messages to free up space and improve performance."
		recoveryOptions = []models.RecoveryOption{
			{
				Label:       "Archive Old Messages",
				Action:      "archive",
				Description: "Move older messages to archive to free up context space",
			},
			{
				Label:       "Summarize Conversation",
				Action:      "summarize",
				Description: "Automatically summarize old messages to compress context",
			},
		}
	} else {
		// This shouldn't be called for normal usage, but handle it anyway
		code = models.ContextWarning
		message = fmt.Sprintf(
			"Context usage: %dK / %dK tokens (%.1f%%)",
			tokenCount/1000,
			g.maxTokens/1000,
			percentage,
		)
		suggestion = "Your conversation is using context normally."
		recoveryOptions = []models.RecoveryOption{}
	}

	return &models.ContextError{
		Code:            code,
		Message:         message,
		Suggestion:      suggestion,
		RecoveryOptions: recoveryOptions,
		ContextMetadata: contextMeta,
		Timestamp:       time.Now(),
	}
}

// GenerateSummarizationError creates an error for failed summarization
func (g *ErrorMessageGenerator) GenerateSummarizationError() *models.ContextError {
	return &models.ContextError{
		Code: models.SummarizationFail,
		Message: "Failed to summarize old messages",
		Suggestion: "Automatic summarization encountered an error. " +
			"Please try archiving messages manually or start a new conversation.",
		RecoveryOptions: []models.RecoveryOption{
			{
				Label:       "Archive Old Messages",
				Action:      "archive",
				Description: "Manually move older messages to archive",
			},
			{
				Label:       "Start New Conversation",
				Action:      "new_chat",
				Description: "Begin a fresh conversation with empty context",
			},
		},
		Timestamp: time.Now(),
	}
}

// GenerateContextExceededError creates an error for when context is exceeded
func (g *ErrorMessageGenerator) GenerateContextExceededError(
	tokenCount int,
	attemptedTokens int,
) *models.ContextError {
	return &models.ContextError{
		Code: models.ContextFull,
		Message: fmt.Sprintf(
			"Unable to process message - context limit exceeded (%dK / %dK tokens, +%dK attempted)",
			tokenCount/1000,
			g.maxTokens/1000,
			attemptedTokens/1000,
		),
		Suggestion: "Your message cannot be processed because it would exceed the context limit. " +
			"Please start a new conversation or archive old messages.",
		RecoveryOptions: []models.RecoveryOption{
			{
				Label:       "Start New Conversation",
				Action:      "new_chat",
				Description: "Begin a fresh conversation with empty context",
			},
			{
				Label:       "Archive Old Messages",
				Action:      "archive",
				Description: "Move older messages to archive to free up context space",
			},
		},
		ContextMetadata: &models.ContextMetadata{
			TokenCount:     tokenCount,
			MaxTokens:      g.maxTokens,
			PercentageUsed: float64(tokenCount) / float64(g.maxTokens) * 100,
			IsNearLimit:    true,
			IsFull:         true,
			LastUpdated:    time.Now(),
		},
		Timestamp: time.Now(),
	}
}

// GetRecoveryActionDescription returns a human-readable description of a recovery action
func (g *ErrorMessageGenerator) GetRecoveryActionDescription(action string) string {
	switch action {
	case "archive":
		return "Archive old messages to free up context space"
	case "new_chat":
		return "Start a new conversation with empty context"
	case "summarize":
		return "Automatically summarize old messages to compress context"
	case "clear":
		return "Clear all messages and reset context"
	default:
		return "Take action to resolve context issue"
	}
}
