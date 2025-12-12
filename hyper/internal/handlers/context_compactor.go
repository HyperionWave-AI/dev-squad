package handlers

import (
	"hyper/internal/models"
	"go.uber.org/zap"
)

// ContextCompactor handles adaptive context window management using a backward sliding window algorithm.
// It determines which messages should be compacted (summarized) and which should be preserved based on
// token count thresholds and preservation rules.
//
// Algorithm Overview:
// 1. Check if total tokens exceed trigger threshold (90% of 128K = 115,200 tokens)
// 2. If yes, calculate split point using backward sliding window:
//    - Start from the most recent message and work backward
//    - Keep recent messages that fit within target tokens (60% of 128K = 76,800 tokens)
//    - Stop when adding the next message would exceed available tokens
// 3. Adjust split point to preserve tool_call + tool_result pairs together
// 4. Return messages to compact (older) and messages to keep (recent)
type ContextCompactor struct {
	config     *CompactionConfig
	estimator  *TokenEstimator
	tokenCache *MessageTokenCache
	logger     *zap.Logger
}

// NewContextCompactor creates a new context compactor with the given configuration.
// If config is nil, uses DefaultCompactionConfig().
func NewContextCompactor(config *CompactionConfig, logger *zap.Logger) *ContextCompactor {
	if config == nil {
		config = DefaultCompactionConfig()
	}
	return &ContextCompactor{
		config:     config,
		estimator:  NewTokenEstimator(),
		tokenCache: NewMessageTokenCache(),
		logger:     logger,
	}
}

// ShouldCompact checks if the total token count exceeds the trigger threshold.
// Returns true if compaction is needed, false otherwise.
//
// Trigger threshold: 90% of 128,000 = 115,200 tokens
func (c *ContextCompactor) ShouldCompact(messages []models.ChatMessage) bool {
	if len(messages) == 0 {
		return false
	}

	totalTokens := c.estimator.EstimateTotalTokens(messages)
	triggerTokens := c.config.TriggerTokens()

	shouldCompact := totalTokens > triggerTokens

	if shouldCompact {
		c.logger.Debug("Compaction needed",
			zap.Int("totalTokens", totalTokens),
			zap.Int("triggerThreshold", triggerTokens),
			zap.Int("messageCount", len(messages)))
	}

	return shouldCompact
}

// CalculateSplitPoint determines where to split messages using a backward sliding window algorithm.
// It returns (compactCount, keepCount) where:
//   - compactCount: number of older messages to compact/summarize
//   - keepCount: number of recent messages to preserve
//
// Algorithm:
// 1. Calculate available tokens: targetTokens - summaryBufferTokens
// 2. Start from the most recent message and work backward
// 3. Add each message's tokens until we would exceed available tokens
// 4. Ensure minimum recent messages are kept (PreserveRecentCount)
// 5. Adjust for tool pairs if enabled
//
// Example:
//   Messages: [M1, M2, M3, M4, M5, M6] (oldest to newest)
//   Available tokens: 50,000
//   M6: 5,000 tokens (total: 5,000) ✓ keep
//   M5: 8,000 tokens (total: 13,000) ✓ keep
//   M4: 12,000 tokens (total: 25,000) ✓ keep
//   M3: 15,000 tokens (total: 40,000) ✓ keep
//   M2: 20,000 tokens (total: 60,000) ✗ would exceed, stop
//   Result: compact M1, M2 (2 messages) | keep M3, M4, M5, M6 (4 messages)
func (c *ContextCompactor) CalculateSplitPoint(messages []models.ChatMessage) (compactCount int, keepCount int) {
	if len(messages) == 0 {
		return 0, 0
	}

	// Calculate available tokens for kept messages
	// Reserve space for the summary that will replace compacted messages
	availableTokens := c.config.TargetTokens() - c.config.SummaryBufferTokens

	// Backward sliding window: keep recent messages that fit
	recentTokens := 0
	keptCount := 0

	// Start from the most recent message (end of slice) and work backward
	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := c.estimator.EstimateMessageTokens(&messages[i])

		// Cap individual message tokens to prevent single large messages from blocking compaction
		if msgTokens > c.config.PerMessageMaxTokens {
			msgTokens = c.config.PerMessageMaxTokens
		}

		// Check if adding this message would exceed available tokens
		if recentTokens+msgTokens <= availableTokens {
			recentTokens += msgTokens
			keptCount++
		} else {
			// Stop when we can't fit more messages
			break
		}
	}

	// Ensure minimum recent messages are kept (default: 4)
	// This preserves context even if individual messages are large
	if keptCount < c.config.PreserveRecentCount && len(messages) >= c.config.PreserveRecentCount {
		keptCount = c.config.PreserveRecentCount
	}

	compactCount = len(messages) - keptCount

	// Adjust for tool pairs if enabled (default: true)
	// Ensures tool_call and tool_result messages stay together
	if c.config.PreserveToolPairs {
		compactCount = c.adjustForToolPairs(messages, compactCount)
	}

	return compactCount, len(messages) - compactCount
}

// adjustForToolPairs ensures that tool_call and tool_result messages are not split.
// If the split point falls between a tool_call and its tool_result, it adjusts the split
// to keep them together.
//
// This is important because:
// - A tool_call without its result is incomplete context
// - A tool_result without its call is confusing
//
// The function works by checking the message at the split point and its neighbors:
// - If split is at tool_result, move split back to include the preceding tool_call
// - If split is at tool_call, move split forward to include the following tool_result
//
// Parameters:
//   - messages: the full message slice
//   - splitIndex: the current split point (messages[0:splitIndex] will be compacted)
//
// Returns: adjusted splitIndex that preserves tool pairs
func (c *ContextCompactor) adjustForToolPairs(messages []models.ChatMessage, splitIndex int) int {
	if splitIndex <= 0 || splitIndex >= len(messages) {
		return splitIndex
	}

	// Prevent infinite loops with a maximum iteration count
	maxIterations := 10

	for i := 0; i < maxIterations; i++ {
		if splitIndex >= len(messages) {
			break
		}

		// Get the message at the split point (first message to keep)
		splitMsg := messages[splitIndex]

		// Case 1: Split is at a tool_result message
		// This means we're keeping the tool_result but compacting the tool_call
		// Solution: Move split back one position to include the tool_call
		if splitMsg.Role == "tool_result" && splitIndex > 0 {
			prevMsg := messages[splitIndex-1]
			// Check if previous message is the tool_call for this result
			if prevMsg.Role == "tool_call" || prevMsg.ToolCall != nil {
				splitIndex-- // Move split back to keep pair together
				continue
			}
		}

		// Case 2: Split is at a tool_call message
		// This means we're keeping the tool_call but compacting the tool_result
		// Solution: Move split forward one position to include the tool_result
		if (splitMsg.Role == "tool_call" || splitMsg.ToolCall != nil) && splitIndex < len(messages)-1 {
			nextMsg := messages[splitIndex+1]
			// Check if next message is the tool_result for this call
			if nextMsg.Role == "tool_result" {
				splitIndex++ // Move split forward to keep pair together
				continue
			}
		}

		// No adjustment needed, exit loop
		break
	}

	return splitIndex
}

// GetMessagesToCompact returns the slice of older messages that should be summarized.
// These are the messages that will be replaced with a summary message.
//
// Returns nil if no messages need to be compacted.
func (c *ContextCompactor) GetMessagesToCompact(messages []models.ChatMessage) []models.ChatMessage {
	compactCount, _ := c.CalculateSplitPoint(messages)
	if compactCount <= 0 {
		return nil
	}
	return messages[:compactCount]
}

// GetMessagesToKeep returns the slice of recent messages that should be preserved.
// These are the messages that will remain in the context window after compaction.
//
// Returns nil if no messages should be kept (edge case).
func (c *ContextCompactor) GetMessagesToKeep(messages []models.ChatMessage) []models.ChatMessage {
	compactCount, _ := c.CalculateSplitPoint(messages)
	if compactCount >= len(messages) {
		return nil
	}
	return messages[compactCount:]
}
