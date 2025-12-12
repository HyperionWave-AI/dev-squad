package handlers

import (
	"encoding/json"
	"hyper/internal/models"
	"go.uber.org/zap"
)

const (
	// MongoDBMaxDocSize is the maximum BSON document size in MongoDB (16MB)
	MongoDBMaxDocSize = 16 * 1024 * 1024 // 16MB

	// SizeTriggerThreshold is the percentage of max size at which to trigger compaction (80%)
	SizeTriggerThreshold = 0.80

	// SizeTargetThreshold is the target percentage of max size after compaction (20%)
	SizeTargetThreshold = 0.20

	// BSONStructureOverhead is the estimated overhead for BSON structure per message (~200 bytes)
	BSONStructureOverhead = 200

	// ToolCallOverhead is the estimated overhead for tool call structure (~100 bytes)
	ToolCallOverhead = 100

	// ToolResultOverhead is the estimated overhead for tool result structure (~100 bytes)
	ToolResultOverhead = 100
)

// SizeBasedCompactor monitors BSON document size and triggers compaction
// when approaching MongoDB's 16MB document size limit
type SizeBasedCompactor struct {
	config *CompactionConfig
	logger *zap.Logger
}

// NewSizeBasedCompactor creates a new size-based compactor
func NewSizeBasedCompactor(config *CompactionConfig, logger *zap.Logger) *SizeBasedCompactor {
	if config == nil {
		config = DefaultCompactionConfig()
	}
	return &SizeBasedCompactor{
		config: config,
		logger: logger,
	}
}

// EstimateSessionBSONSize estimates the BSON size of a session's messages
// This includes all message content, tool calls, tool results, and BSON structure overhead
func (c *SizeBasedCompactor) EstimateSessionBSONSize(messages []models.ChatMessage) int {
	totalSize := 0

	for _, msg := range messages {
		// Base message overhead for BSON structure
		totalSize += BSONStructureOverhead

		// Add content size
		totalSize += len(msg.Content)

		// Add tool call size if present
		if msg.ToolCall != nil {
			totalSize += c.estimateToolCallSize(msg.ToolCall)
		}

		// Add tool result size if present
		if msg.ToolResult != nil {
			totalSize += c.estimateToolResultSize(msg.ToolResult)
		}
	}

	return totalSize
}

// estimateToolCallSize estimates the BSON size of a ToolCallData
func (c *SizeBasedCompactor) estimateToolCallSize(toolCall *models.ToolCallData) int {
	size := ToolCallOverhead

	// Add ID size
	size += len(toolCall.ID)

	// Add Name size
	size += len(toolCall.Name)

	// Estimate Args size by marshaling to JSON
	if toolCall.Args != nil {
		if argsJSON, err := json.Marshal(toolCall.Args); err == nil {
			size += len(argsJSON)
		}
	}

	return size
}

// estimateToolResultSize estimates the BSON size of a ToolResultData
func (c *SizeBasedCompactor) estimateToolResultSize(toolResult *models.ToolResultData) int {
	size := ToolResultOverhead

	// Add ID size
	size += len(toolResult.ID)

	// Add Name size
	size += len(toolResult.Name)

	// Add Error size
	size += len(toolResult.Error)

	// Estimate Output size by marshaling to JSON
	if toolResult.Output != nil {
		if outputJSON, err := json.Marshal(toolResult.Output); err == nil {
			size += len(outputJSON)
		}
	}

	return size
}

// ShouldCompactBySize checks if size-based compaction is needed
// Returns true if the estimated BSON size exceeds the trigger threshold
func (c *SizeBasedCompactor) ShouldCompactBySize(messages []models.ChatMessage) bool {
	size := c.EstimateSessionBSONSize(messages)
	maxSize := MongoDBMaxDocSize
	threshold := int(float64(maxSize) * SizeTriggerThreshold)

	if size > threshold {
		c.logger.Debug("Size-based compaction triggered",
			zap.Int("currentSize", size),
			zap.Int("threshold", threshold),
			zap.Int("maxSize", MongoDBMaxDocSize),
			zap.Float64("percentageUsed", float64(size)/float64(MongoDBMaxDocSize)*100))
		return true
	}

	return false
}

// GetSizeCompactionTarget calculates the target size for compaction
// Returns the target size in bytes (20% of max size)
func (c *SizeBasedCompactor) GetSizeCompactionTarget() int {
	maxSize := MongoDBMaxDocSize
	return int(float64(maxSize) * SizeTargetThreshold)
}

// GetSizeTriggerThreshold returns the size threshold that triggers compaction
// Returns the trigger size in bytes (80% of max size)
func (c *SizeBasedCompactor) GetSizeTriggerThreshold() int {
	maxSize := MongoDBMaxDocSize
	return int(float64(maxSize) * SizeTriggerThreshold)
}

// GetSizePercentageUsed calculates the percentage of max BSON size currently used
func (c *SizeBasedCompactor) GetSizePercentageUsed(messages []models.ChatMessage) float64 {
	size := c.EstimateSessionBSONSize(messages)
	return float64(size) / float64(MongoDBMaxDocSize) * 100
}

// GetSizeWarningLevel returns a warning level based on current size usage
// Returns: "ok" (< 50%), "warning" (50-80%), "critical" (80-95%), "emergency" (> 95%)
func (c *SizeBasedCompactor) GetSizeWarningLevel(messages []models.ChatMessage) string {
	percentage := c.GetSizePercentageUsed(messages)

	switch {
	case percentage < 50:
		return "ok"
	case percentage < 80:
		return "warning"
	case percentage < 95:
		return "critical"
	default:
		return "emergency"
	}
}

// CalculateSplitPointBySize determines where to split messages using a backward sliding window
// algorithm based on BSON size (similar to CalculateSplitPoint in ContextCompactor but for size).
// It returns (compactCount, keepCount) where:
//   - compactCount: number of older messages to compact/summarize
//   - keepCount: number of recent messages to preserve
//
// The algorithm starts from the most recent message and works backward, keeping messages
// until the target size threshold (20% of 16MB = 3.2MB) would be exceeded.
func (c *SizeBasedCompactor) CalculateSplitPointBySize(messages []models.ChatMessage) (compactCount int, keepCount int) {
	if len(messages) == 0 {
		return 0, 0
	}

	// Calculate available size for kept messages
	// Target: 20% of max size = 3.2MB
	availableSize := c.GetSizeCompactionTarget()

	// Backward sliding window: keep recent messages that fit within target size
	recentSize := 0
	keptCount := 0

	// Start from the most recent message (end of slice) and work backward
	for i := len(messages) - 1; i >= 0; i-- {
		msgSize := c.estimateMessageBSONSize(&messages[i])

		// Check if adding this message would exceed available size
		if recentSize+msgSize <= availableSize {
			recentSize += msgSize
			keptCount++
		} else {
			// Stop when we can't fit more messages
			break
		}
	}

	// Ensure at least some recent messages are kept (minimum 4)
	minRecentCount := 4
	if keptCount < minRecentCount && len(messages) >= minRecentCount {
		keptCount = minRecentCount
	}

	compactCount = len(messages) - keptCount

	// Adjust for tool pairs to keep tool_call + tool_result together
	compactCount = c.adjustForToolPairs(messages, compactCount)

	c.logger.Debug("Size-based split point calculated",
		zap.Int("totalMessages", len(messages)),
		zap.Int("compactCount", compactCount),
		zap.Int("keepCount", len(messages)-compactCount),
		zap.Int("recentSize", recentSize),
		zap.Int("availableSize", availableSize))

	return compactCount, len(messages) - compactCount
}

// estimateMessageBSONSize estimates the BSON size of a single message
func (c *SizeBasedCompactor) estimateMessageBSONSize(msg *models.ChatMessage) int {
	size := BSONStructureOverhead
	size += len(msg.Content)

	if msg.ToolCall != nil {
		size += c.estimateToolCallSize(msg.ToolCall)
	}

	if msg.ToolResult != nil {
		size += c.estimateToolResultSize(msg.ToolResult)
	}

	return size
}

// adjustForToolPairs ensures that tool_call and tool_result messages are not split.
// Similar to the method in ContextCompactor.
func (c *SizeBasedCompactor) adjustForToolPairs(messages []models.ChatMessage, splitIndex int) int {
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
		// Keep the tool_call with its result
		if splitMsg.Role == "tool_result" && splitIndex > 0 {
			prevMsg := messages[splitIndex-1]
			if prevMsg.Role == "tool_call" || prevMsg.ToolCall != nil {
				splitIndex-- // Move split back to keep pair together
				continue
			}
		}

		// Case 2: Split is at a tool_call message
		// Keep the tool_result with its call
		if (splitMsg.Role == "tool_call" || splitMsg.ToolCall != nil) && splitIndex < len(messages)-1 {
			nextMsg := messages[splitIndex+1]
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

// GetMessagesToCompactBySize returns the slice of older messages that should be summarized
// based on size-based compaction.
func (c *SizeBasedCompactor) GetMessagesToCompactBySize(messages []models.ChatMessage) []models.ChatMessage {
	compactCount, _ := c.CalculateSplitPointBySize(messages)
	if compactCount <= 0 {
		return nil
	}
	return messages[:compactCount]
}

// GetMessagesToKeepBySize returns the slice of recent messages that should be preserved
// based on size-based compaction.
func (c *SizeBasedCompactor) GetMessagesToKeepBySize(messages []models.ChatMessage) []models.ChatMessage {
	compactCount, _ := c.CalculateSplitPointBySize(messages)
	if compactCount >= len(messages) {
		return nil
	}
	return messages[compactCount:]
}
