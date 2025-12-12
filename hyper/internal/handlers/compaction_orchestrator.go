package handlers

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"hyper/internal/metrics"
	"hyper/internal/models"
)

// CompactionOrchestrator coordinates the full compaction workflow.
// It orchestrates the interaction between:
// - ContextCompactor: Determines which messages to compact based on tokens
// - SizeBasedCompactor: Determines which messages to compact based on BSON size
// - CompactionSummarizer: Generates AI summaries of compacted messages
// - ChatService: Persists archived messages and summary messages
//
// The orchestrator supports dual compaction triggers:
// - Token-based: Triggers when token count exceeds 90% of context window
// - Size-based: Triggers when BSON size exceeds 80% of MongoDB's 16MB limit
//
// Compaction is triggered if EITHER threshold is exceeded.
type CompactionOrchestrator struct {
	compactor     *ContextCompactor
	sizeCompactor *SizeBasedCompactor
	summarizer    *CompactionSummarizer
	chatService   ChatServiceInterface
	logger        *zap.Logger
	config        *CompactionConfig
}

// NewCompactionOrchestrator creates a new compaction orchestrator.
// If config is nil, uses DefaultCompactionConfig().
func NewCompactionOrchestrator(
	config *CompactionConfig,
	aiService AIServiceInterface,
	chatService ChatServiceInterface,
	logger *zap.Logger,
) *CompactionOrchestrator {
	if config == nil {
		config = DefaultCompactionConfig()
	}

	return &CompactionOrchestrator{
		compactor:     NewContextCompactor(config, logger),
		sizeCompactor: NewSizeBasedCompactor(config, logger),
		summarizer:    NewCompactionSummarizer(aiService, logger),
		chatService:   chatService,
		logger:        logger,
		config:        config,
	}
}

// CompactIfNeeded checks if compaction is needed and performs it if necessary.
// This is the main entry point for the compaction workflow.
//
// The method checks BOTH token-based and size-based triggers:
// - Token-based: Triggered when token count exceeds 90% of context window (115,200 tokens)
// - Size-based: Triggered when BSON size exceeds 80% of MongoDB's 16MB limit (12.8MB)
//
// If either trigger fires, compaction is performed. When both fire, the more aggressive
// split point (more messages compacted) is used.
//
// Returns a CompactionResult with details about what was compacted (if anything).
// Returns an error only if a critical operation fails (e.g., database error).
// Non-critical errors (e.g., AI summarization failure) are logged but don't prevent compaction.
func (o *CompactionOrchestrator) CompactIfNeeded(
	ctx context.Context,
	sessionID primitive.ObjectID,
	messages []models.ChatMessage,
	companyID string,
) (*CompactionResult, error) {
	result := &CompactionResult{
		OriginalTokens: o.compactor.estimator.EstimateTotalTokens(messages),
		OriginalSize:   o.sizeCompactor.EstimateSessionBSONSize(messages),
		Trigger:        TriggerNone,
	}

	// Check both token-based and size-based triggers
	tokenTrigger := o.compactor.ShouldCompact(messages)
	sizeTrigger := o.sizeCompactor.ShouldCompactBySize(messages)

	// Determine trigger type
	switch {
	case tokenTrigger && sizeTrigger:
		result.Trigger = TriggerBoth
	case tokenTrigger:
		result.Trigger = TriggerTokens
	case sizeTrigger:
		result.Trigger = TriggerSize
	default:
		// No compaction needed
		o.logger.Debug("Compaction not needed",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("tokens", result.OriginalTokens),
			zap.Int("tokenThreshold", o.config.TriggerTokens()),
			zap.Int("size", result.OriginalSize),
			zap.Int("sizeThreshold", o.sizeCompactor.GetSizeTriggerThreshold()))
		return result, nil
	}

	// Start timing for metrics
	compactionStart := time.Now()

	o.logger.Info("🗜️ Starting context compaction",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("trigger", string(result.Trigger)),
		zap.Int("originalTokens", result.OriginalTokens),
		zap.Int("originalSize", result.OriginalSize),
		zap.Int("messageCount", len(messages)),
		zap.Float64("tokenPercentage", float64(result.OriginalTokens)/float64(o.config.MaxContextTokens)*100),
		zap.Float64("sizePercentage", o.sizeCompactor.GetSizePercentageUsed(messages)))

	// Calculate which messages to compact based on trigger type
	// When both triggers fire, use the more aggressive (higher compactCount) split point
	var toCompact, toKeep []models.ChatMessage

	switch result.Trigger {
	case TriggerTokens:
		toCompact = o.compactor.GetMessagesToCompact(messages)
		toKeep = o.compactor.GetMessagesToKeep(messages)
	case TriggerSize:
		toCompact = o.sizeCompactor.GetMessagesToCompactBySize(messages)
		toKeep = o.sizeCompactor.GetMessagesToKeepBySize(messages)
	case TriggerBoth:
		// Use the more aggressive split point (whichever compacts more messages)
		tokenCompact := o.compactor.GetMessagesToCompact(messages)
		sizeCompact := o.sizeCompactor.GetMessagesToCompactBySize(messages)

		if len(sizeCompact) > len(tokenCompact) {
			toCompact = sizeCompact
			toKeep = o.sizeCompactor.GetMessagesToKeepBySize(messages)
			o.logger.Debug("Using size-based split (more aggressive)",
				zap.Int("sizeCompactCount", len(sizeCompact)),
				zap.Int("tokenCompactCount", len(tokenCompact)))
		} else {
			toCompact = tokenCompact
			toKeep = o.compactor.GetMessagesToKeep(messages)
			o.logger.Debug("Using token-based split (more aggressive)",
				zap.Int("tokenCompactCount", len(tokenCompact)),
				zap.Int("sizeCompactCount", len(sizeCompact)))
		}
	}

	if len(toCompact) == 0 {
		o.logger.Debug("No messages to compact after split point calculation",
			zap.String("sessionId", sessionID.Hex()))
		return result, nil
	}

	result.MessagesCompacted = len(toCompact)
	result.MessagesKept = len(toKeep)

	o.logger.Info("📊 Split point calculated",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("toCompact", len(toCompact)),
		zap.Int("toKeep", len(toKeep)))

	// Generate summary of compacted messages
	summary, err := o.summarizer.GenerateSummary(ctx, toCompact)
	if err != nil {
		o.logger.Error("Failed to generate summary", zap.Error(err))
		result.Error = err
		return result, err
	}

	if summary == "" {
		o.logger.Warn("Summary generation returned empty string",
			zap.String("sessionId", sessionID.Hex()))
		summary = o.generateFallbackSummary(toCompact)
	}

	o.logger.Info("✅ Summary generated",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("summaryLength", len(summary)))

	// Create summary message
	summaryMsg := &models.ChatMessage{
		SessionID:            sessionID,
		Role:                 "system",
		Content:              summary,
		Timestamp:            time.Now(),
		IsSummary:            true,
		IsArchived:           false,
		OriginalMessageCount: len(toCompact),
		TokenCount:           o.compactor.estimator.EstimateTokens(summary),
	}

	// Archive compacted messages
	err = o.archiveMessages(ctx, sessionID, toCompact)
	if err != nil {
		o.logger.Error("Failed to archive messages", zap.Error(err))
		result.Error = err
		return result, err
	}

	o.logger.Info("📦 Messages archived",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("archivedCount", len(toCompact)))

	// Save summary message
	err = o.saveSummaryMessage(ctx, sessionID, summaryMsg, companyID)
	if err != nil {
		o.logger.Error("Failed to save summary message", zap.Error(err))
		result.Error = err
		return result, err
	}

	o.logger.Info("💾 Summary message saved",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("summaryId", summaryMsg.ID.Hex()))

	// Update result
	result.WasCompacted = true
	result.SummaryGenerated = true
	result.CompactedTokens = o.compactor.estimator.EstimateTotalTokens(toKeep) + summaryMsg.TokenCount

	// Calculate compacted size (kept messages + summary message size)
	summarySize := BSONStructureOverhead + len(summary)
	result.CompactedSize = o.sizeCompactor.EstimateSessionBSONSize(toKeep) + summarySize

	// Calculate reduction percentages
	tokenReduction := 0.0
	if result.OriginalTokens > 0 {
		tokenReduction = float64(result.OriginalTokens-result.CompactedTokens) / float64(result.OriginalTokens) * 100
	}
	sizeReduction := 0.0
	if result.OriginalSize > 0 {
		sizeReduction = float64(result.OriginalSize-result.CompactedSize) / float64(result.OriginalSize) * 100
	}

	// Calculate compaction duration
	compactionDuration := time.Since(compactionStart).Seconds()

	o.logger.Info("✅ Context compaction complete",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("trigger", string(result.Trigger)),
		zap.Int("compactedMessages", result.MessagesCompacted),
		zap.Int("keptMessages", result.MessagesKept),
		zap.Int("originalTokens", result.OriginalTokens),
		zap.Int("newTokens", result.CompactedTokens),
		zap.Float64("tokenReductionPercent", tokenReduction),
		zap.Int("originalSize", result.OriginalSize),
		zap.Int("newSize", result.CompactedSize),
		zap.Float64("sizeReductionPercent", sizeReduction),
		zap.Float64("durationSeconds", compactionDuration))

	// Record compaction metrics
	tokensReduced := result.OriginalTokens - result.CompactedTokens
	sizeReduced := result.OriginalSize - result.CompactedSize
	metrics.RecordCompaction(
		"chat", // sessionType - could be parameterized in future
		string(result.Trigger),
		tokensReduced,
		sizeReduced,
		result.MessagesCompacted,
		compactionDuration,
	)

	// Record summary token count
	metrics.RecordCompactionSummary("chat", summaryMsg.TokenCount)

	// Update context usage metrics for this session
	tokenPercentage := float64(result.CompactedTokens) / float64(o.config.MaxContextTokens) * 100
	sizePercentage := float64(result.CompactedSize) / float64(MongoDBMaxDocSize) * 100
	metrics.RecordContextUsage(
		sessionID.Hex(),
		result.CompactedTokens,
		tokenPercentage,
		result.CompactedSize,
		sizePercentage,
	)

	// Create notification for frontend display
	result.Notification = &models.SystemNotification{
		Category: "compaction",
		Title:    "Context Compacted",
		Message:  fmt.Sprintf("Archived %d messages to save context space", result.MessagesCompacted),
		Severity: "info",
		Metadata: map[string]interface{}{
			"messagesArchived":      result.MessagesCompacted,
			"trigger":               string(result.Trigger),
			"tokenReductionPercent": tokenReduction,
			"sizeReductionPercent":  sizeReduction,
		},
	}

	return result, nil
}

// archiveMessages marks messages as archived in the database.
// Uses the ChatService.ArchiveMessages method to batch update messages.
func (o *CompactionOrchestrator) archiveMessages(
	ctx context.Context,
	sessionID primitive.ObjectID,
	messages []models.ChatMessage,
) error {
	if len(messages) == 0 {
		return nil
	}

	// Extract message IDs for batch archival
	messageIDs := make([]primitive.ObjectID, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.ID
	}

	o.logger.Debug("Archiving messages via ChatService",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("count", len(messageIDs)))

	// Call the actual ChatService method to archive messages
	err := o.chatService.ArchiveMessages(ctx, sessionID, messageIDs)
	if err != nil {
		o.logger.Error("ChatService.ArchiveMessages failed",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("messageCount", len(messageIDs)),
			zap.Error(err))
		return fmt.Errorf("failed to archive messages: %w", err)
	}

	return nil
}

// saveSummaryMessage saves the summary message to the database.
// Uses the ChatService.SaveMessage method to insert the summary.
func (o *CompactionOrchestrator) saveSummaryMessage(
	ctx context.Context,
	sessionID primitive.ObjectID,
	summaryMsg *models.ChatMessage,
	companyID string,
) error {
	o.logger.Debug("Saving summary message via ChatService",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("summaryLength", len(summaryMsg.Content)),
		zap.Int("originalMessageCount", summaryMsg.OriginalMessageCount))

	// Save the summary as a "summary" role message
	// The role "summary" is used to distinguish it from regular system messages
	savedMsg, err := o.chatService.SaveMessage(ctx, sessionID, "summary", summaryMsg.Content, companyID)
	if err != nil {
		o.logger.Error("ChatService.SaveMessage failed for summary",
			zap.String("sessionId", sessionID.Hex()),
			zap.Error(err))
		return fmt.Errorf("failed to save summary message: %w", err)
	}

	// Update the summaryMsg with the database-assigned ID
	if savedMsg != nil {
		summaryMsg.ID = savedMsg.ID
		o.logger.Debug("Summary message saved successfully",
			zap.String("sessionId", sessionID.Hex()),
			zap.String("messageId", savedMsg.ID.Hex()))
	}

	return nil
}

// generateFallbackSummary creates a basic summary when AI summarization fails.
// This ensures compaction can proceed even if the AI service is unavailable.
func (o *CompactionOrchestrator) generateFallbackSummary(messages []models.ChatMessage) string {
	if len(messages) == 0 {
		return ""
	}

	// Count different message types
	userMessages := 0
	assistantMessages := 0
	toolCalls := 0
	toolResults := 0

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			userMessages++
		case "assistant":
			assistantMessages++
		case "tool_call":
			toolCalls++
		case "tool_result":
			toolResults++
		}
	}

	// Build fallback summary
	summary := fmt.Sprintf("**Conversation Summary** (%d messages):\n\n", len(messages))
	summary += "**Message Statistics:**\n"

	if userMessages > 0 {
		summary += fmt.Sprintf("- %d user messages\n", userMessages)
	}
	if assistantMessages > 0 {
		summary += fmt.Sprintf("- %d assistant responses\n", assistantMessages)
	}
	if toolCalls > 0 {
		summary += fmt.Sprintf("- %d tool calls\n", toolCalls)
	}
	if toolResults > 0 {
		summary += fmt.Sprintf("- %d tool results\n", toolResults)
	}

	summary += "\n**Key Events:**\n"

	// Extract first user message as context
	for _, msg := range messages {
		if msg.Role == "user" && len(msg.Content) > 0 {
			content := msg.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			summary += fmt.Sprintf("- Started with: \"%s\"\n", content)
			break
		}
	}

	// Extract last assistant message as final context
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == "assistant" && len(msg.Content) > 0 {
			content := msg.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			summary += fmt.Sprintf("- Concluded with: \"%s\"\n", content)
			break
		}
	}

	summary += "\n---"

	return summary
}

// ShouldCompact is a convenience method that delegates to the compactor.
// Returns true if the total token count exceeds the trigger threshold.
func (o *CompactionOrchestrator) ShouldCompact(messages []models.ChatMessage) bool {
	return o.compactor.ShouldCompact(messages)
}

// GetCompactionStats returns statistics about the current compaction state.
// Useful for monitoring and debugging.
func (o *CompactionOrchestrator) GetCompactionStats(messages []models.ChatMessage) map[string]interface{} {
	// Token-based stats
	totalTokens := o.compactor.estimator.EstimateTotalTokens(messages)
	triggerTokens := o.config.TriggerTokens()
	targetTokens := o.config.TargetTokens()

	compactCount, keepCount := o.compactor.CalculateSplitPoint(messages)

	// Size-based stats
	totalSize := o.sizeCompactor.EstimateSessionBSONSize(messages)
	sizeTriggerThreshold := o.sizeCompactor.GetSizeTriggerThreshold()
	sizeTargetThreshold := o.sizeCompactor.GetSizeCompactionTarget()
	sizeCompactCount, sizeKeepCount := o.sizeCompactor.CalculateSplitPointBySize(messages)

	// Determine trigger status
	tokenShouldCompact := totalTokens > triggerTokens
	sizeShouldCompact := o.sizeCompactor.ShouldCompactBySize(messages)

	return map[string]interface{}{
		// Token-based stats
		"totalTokens":           totalTokens,
		"triggerTokens":         triggerTokens,
		"targetTokens":          targetTokens,
		"tokenPercentageUsed":   float64(totalTokens) / float64(o.config.MaxContextTokens) * 100,
		"tokenShouldCompact":    tokenShouldCompact,
		"tokenCompactCount":     compactCount,
		"tokenKeepCount":        keepCount,

		// Size-based stats
		"totalSize":             totalSize,
		"sizeTriggerThreshold":  sizeTriggerThreshold,
		"sizeTargetThreshold":   sizeTargetThreshold,
		"sizePercentageUsed":    o.sizeCompactor.GetSizePercentageUsed(messages),
		"sizeShouldCompact":     sizeShouldCompact,
		"sizeCompactCount":      sizeCompactCount,
		"sizeKeepCount":         sizeKeepCount,
		"sizeWarningLevel":      o.sizeCompactor.GetSizeWarningLevel(messages),

		// Combined stats
		"shouldCompact":         tokenShouldCompact || sizeShouldCompact,
		"messageCount":          len(messages),
		"summaryBuffer":         o.config.SummaryBufferTokens,
		"perMessageMax":         o.config.PerMessageMaxTokens,
		"maxBSONSize":           MongoDBMaxDocSize,
	}
}
