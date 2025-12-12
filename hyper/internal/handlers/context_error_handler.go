package handlers

import (
	"fmt"

	"hyper/internal/models"
	"hyper/internal/services"
	"go.uber.org/zap"
)


// ContextErrorHandler handles context-related errors and sends appropriate responses
type ContextErrorHandler struct {
	logger                 *zap.Logger
	errorMessageGenerator  *services.ErrorMessageGenerator
	maxContextTokens       int
}

// NewContextErrorHandler creates a new context error handler
func NewContextErrorHandler(logger *zap.Logger, maxContextTokens int) *ContextErrorHandler {
	return &ContextErrorHandler{
		logger:                logger,
		errorMessageGenerator: services.NewErrorMessageGenerator(maxContextTokens),
		maxContextTokens:      maxContextTokens,
	}
}

// CheckContextLimit checks if adding a message would exceed context limit
func (h *ContextErrorHandler) CheckContextLimit(
	currentTokenCount int,
	messageTokenCount int,
) (bool, *models.ContextError) {
	totalTokens := currentTokenCount + messageTokenCount
	percentage := float64(totalTokens) / float64(h.maxContextTokens) * 100

	// If adding this message would exceed limit, return error
	if totalTokens > h.maxContextTokens {
		h.logger.Warn("Context limit would be exceeded",
			zap.Int("currentTokens", currentTokenCount),
			zap.Int("messageTokens", messageTokenCount),
			zap.Int("totalTokens", totalTokens),
			zap.Int("maxTokens", h.maxContextTokens),
			zap.Float64("percentage", percentage))

		contextErr := h.errorMessageGenerator.GenerateContextExceededError(
			currentTokenCount,
			messageTokenCount,
		)
		return false, contextErr
	}

	// If we're at or above 80%, return a warning
	if percentage >= 80 {
		h.logger.Warn("Context usage is high",
			zap.Int("currentTokens", currentTokenCount),
			zap.Int("messageTokens", messageTokenCount),
			zap.Int("totalTokens", totalTokens),
			zap.Float64("percentage", percentage))

		contextErr := h.errorMessageGenerator.GenerateContextError(totalTokens, percentage)
		return true, contextErr
	}

	return true, nil
}

// GetContextStatus returns the current context status
func (h *ContextErrorHandler) GetContextStatus(tokenCount int) *models.ContextMetadata {
	percentage := float64(tokenCount) / float64(h.maxContextTokens) * 100

	return &models.ContextMetadata{
		TokenCount:     tokenCount,
		MaxTokens:      h.maxContextTokens,
		PercentageUsed: percentage,
		IsNearLimit:    percentage >= 80,
		IsFull:         percentage >= 100,
	}
}

// LogContextError logs a context error with appropriate severity
func (h *ContextErrorHandler) LogContextError(err *models.ContextError, sessionID string) {
	switch err.Code {
	case models.ContextFull:
		h.logger.Error("Context limit reached",
			zap.String("sessionId", sessionID),
			zap.String("message", err.Message),
			zap.String("suggestion", err.Suggestion))
	case models.ContextCritical:
		h.logger.Warn("Context limit critical",
			zap.String("sessionId", sessionID),
			zap.String("message", err.Message),
			zap.String("suggestion", err.Suggestion))
	case models.ContextWarning:
		h.logger.Warn("Context usage warning",
			zap.String("sessionId", sessionID),
			zap.String("message", err.Message),
			zap.String("suggestion", err.Suggestion))
	case models.SummarizationFail:
		h.logger.Error("Summarization failed",
			zap.String("sessionId", sessionID),
			zap.String("message", err.Message))
	}
}

// FormatContextErrorResponse formats a context error for sending to client
func (h *ContextErrorHandler) FormatContextErrorResponse(err *models.ContextError) models.StreamMessage {
	// Create error message with recovery options
	errorContent := fmt.Sprintf(
		"**%s**\n\n%s\n\n%s",
		err.Message,
		err.Suggestion,
		h.formatRecoveryOptions(err.RecoveryOptions),
	)

	return models.StreamMessage{
		Type:  "error",
		Error: errorContent,
	}
}

// formatRecoveryOptions formats recovery options for display
func (h *ContextErrorHandler) formatRecoveryOptions(options []models.RecoveryOption) string {
	if len(options) == 0 {
		return ""
	}

	result := "**Suggested Actions:**\n"
	for i, option := range options {
		result += fmt.Sprintf("%d. **%s** - %s\n", i+1, option.Label, option.Description)
	}
	return result
}
