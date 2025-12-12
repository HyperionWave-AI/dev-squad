package handlers

import (
	"context"
	"fmt"

	"hyper/internal/models"
	"hyper/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// ContextLimitHandler manages context limit enforcement for chat sessions
type ContextLimitHandler struct {
	chatService       ChatServiceInterface
	contextManager    *utils.ContextManager
	messageSummarizer *utils.MessageSummarizer
	logger            *zap.Logger
}

// NewContextLimitHandler creates a new context limit handler
func NewContextLimitHandler(
	chatService ChatServiceInterface,
	contextManager *utils.ContextManager,
	messageSummarizer *utils.MessageSummarizer,
	logger *zap.Logger,
) *ContextLimitHandler {
	return &ContextLimitHandler{
		chatService:       chatService,
		contextManager:    contextManager,
		messageSummarizer: messageSummarizer,
		logger:            logger,
	}
}

// CheckContextBeforeMessage checks if a message can be added to the session
// Returns (canAdd, usage, error)
func (h *ContextLimitHandler) CheckContextBeforeMessage(
	ctx context.Context,
	sessionID primitive.ObjectID,
	messageContent string,
	companyID string,
) (bool, *utils.ContextUsage, error) {
	// Get all messages in the session
	messages, err := h.chatService.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get session messages: %w", err)
	}

	// Update context usage
	usage := h.contextManager.UpdateContextUsage(ctx, sessionID.Hex(), messages)

	// Check if we can add this message
	canAdd, _ := h.contextManager.CanAddMessage(sessionID.Hex(), messageContent)

	if !canAdd {
		h.logger.Warn("🚨 Context limit exceeded - cannot add message",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("currentTokens", usage.TotalTokens),
			zap.Int("maxTokens", usage.MaxTokens),
			zap.Float64("percentageUsed", usage.PercentageUsed))
		return false, usage, nil
	}

	// Log context status
	if usage.IsCritical {
		h.logger.Warn("⚠️ CRITICAL: Context usage at critical level",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("currentTokens", usage.TotalTokens),
			zap.Float64("percentageUsed", usage.PercentageUsed))
	} else if usage.IsWarning {
		h.logger.Warn("⚠️ WARNING: Context usage approaching limit",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("currentTokens", usage.TotalTokens),
			zap.Float64("percentageUsed", usage.PercentageUsed))
	}

	return true, usage, nil
}

// GetContextStatus returns the current context status for a session
func (h *ContextLimitHandler) GetContextStatus(
	ctx context.Context,
	sessionID primitive.ObjectID,
) (*utils.ContextUsage, error) {
	messages, err := h.chatService.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
	}

	usage := h.contextManager.UpdateContextUsage(ctx, sessionID.Hex(), messages)
	return usage, nil
}

// ShouldTriggerSummarization checks if summarization should be triggered
func (h *ContextLimitHandler) ShouldTriggerSummarization(usage *utils.ContextUsage) bool {
	return usage.NeedsSummarization
}

// GetSummarizationRecommendation provides a summarization recommendation
func (h *ContextLimitHandler) GetSummarizationRecommendation(
	ctx context.Context,
	messages []models.ChatMessage,
) (*utils.SummarizationResult, error) {
	// Use oldest-first strategy to summarize oldest messages
	result, err := h.messageSummarizer.SummarizeMessages(
		ctx,
		messages,
		utils.StrategyOldestFirst,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summarization recommendation: %w", err)
	}

	h.logger.Info("Generated summarization recommendation",
		zap.Int("messageCount", len(messages)),
		zap.Int("groupCount", len(result.MessageGroups)),
		zap.Int("tokensSaved", result.TotalTokensSaved))

	return result, nil
}

// CreateContextError creates a user-friendly context error
func (h *ContextLimitHandler) CreateContextError(usage *utils.ContextUsage) *utils.ContextError {
	return utils.NewContextError("CONTEXT_LIMIT_EXCEEDED", "Context limit reached", usage)
}

// LogContextMetrics logs comprehensive context metrics
func (h *ContextLimitHandler) LogContextMetrics(sessionID primitive.ObjectID, usage *utils.ContextUsage) {
	h.logger.Info("📊 Context metrics",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("totalTokens", usage.TotalTokens),
		zap.Int("maxTokens", usage.MaxTokens),
		zap.Float64("percentageUsed", usage.PercentageUsed),
		zap.Int("messageCount", usage.MessageCount),
		zap.Bool("isWarning", usage.IsWarning),
		zap.Bool("isCritical", usage.IsCritical),
		zap.Bool("needsSummarization", usage.NeedsSummarization))
}
