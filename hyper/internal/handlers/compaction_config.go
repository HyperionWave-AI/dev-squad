package handlers

import "hyper/internal/models"

// CompactionConfig defines thresholds for context compaction
type CompactionConfig struct {
	// Context window settings
	MaxContextTokens int     // 128,000 (Claude's limit)
	TriggerThreshold float64 // 0.90 (90%)
	TargetThreshold  float64 // 0.60 (60%)

	// Buffer settings
	SummaryBufferTokens int // 2,000 (reserve for AI summary)
	PerMessageMaxTokens int // 8,000 (truncate large messages)

	// Behavior settings
	PreserveToolPairs       bool // Keep tool_call + tool_result together
	PreserveRecentCount     int  // Minimum recent messages to keep (e.g., 4)
	AggressiveMode          bool // Force compaction of all but last message
	ValidateAfterCompaction bool // Verify token count post-compaction
}

// DefaultCompactionConfig returns production-ready defaults
func DefaultCompactionConfig() *CompactionConfig {
	return &CompactionConfig{
		MaxContextTokens:        128000,
		TriggerThreshold:        0.90, // 90% of context window
		TargetThreshold:         0.60,
		SummaryBufferTokens:     2000,
		PerMessageMaxTokens:     8000,
		PreserveToolPairs:       true,
		PreserveRecentCount:     4,
		AggressiveMode:          false,
		ValidateAfterCompaction: true,
	}
}

// TriggerTokens returns the token count at which compaction should be triggered
func (c *CompactionConfig) TriggerTokens() int {
	return int(float64(c.MaxContextTokens) * c.TriggerThreshold)
}

// TargetTokens returns the target token count after compaction
func (c *CompactionConfig) TargetTokens() int {
	return int(float64(c.MaxContextTokens) * c.TargetThreshold)
}

// CompactionTrigger represents what triggered the compaction
type CompactionTrigger string

const (
	// TriggerNone indicates no compaction was triggered
	TriggerNone CompactionTrigger = "none"
	// TriggerTokens indicates compaction was triggered by token count
	TriggerTokens CompactionTrigger = "tokens"
	// TriggerSize indicates compaction was triggered by BSON size
	TriggerSize CompactionTrigger = "size"
	// TriggerBoth indicates compaction was triggered by both token count and BSON size
	TriggerBoth CompactionTrigger = "both"
)

// CompactionResult holds the result of a compaction operation
type CompactionResult struct {
	WasCompacted      bool              // Whether compaction was performed
	Trigger           CompactionTrigger // What triggered the compaction (tokens, size, or both)
	OriginalTokens    int               // Token count before compaction
	CompactedTokens   int               // Token count after compaction
	OriginalSize      int               // BSON size before compaction (bytes)
	CompactedSize     int               // BSON size after compaction (bytes)
	MessagesCompacted int               // Number of messages that were compacted
	MessagesKept      int               // Number of messages that were preserved
	SummaryGenerated  bool              // Whether a summary was generated
	Error             error             // Any error that occurred during compaction
	Notification      *models.SystemNotification // Notification to send to frontend (nil if no compaction)
}
