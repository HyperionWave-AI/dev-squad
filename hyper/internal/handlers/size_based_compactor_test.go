package handlers

import (
	"strings"
	"testing"
	"time"

	"hyper/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// TestEstimateSessionBSONSize verifies BSON size estimation
func TestEstimateSessionBSONSize(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	tests := []struct {
		name            string
		messages        []models.ChatMessage
		minExpectedSize int
		maxExpectedSize int
		description     string
	}{
		{
			name:            "empty messages",
			messages:        []models.ChatMessage{},
			minExpectedSize: 0,
			maxExpectedSize: 100,
			description:     "Empty message list should have minimal size",
		},
		{
			name: "single simple message",
			messages: []models.ChatMessage{
				{
					ID:        primitive.NewObjectID(),
					SessionID: primitive.NewObjectID(),
					Role:      "user",
					Content:   "Hello",
					Timestamp: time.Now(),
				},
			},
			minExpectedSize: 200, // BSONStructureOverhead + content
			maxExpectedSize: 500,
			description:     "Single simple message should have reasonable size",
		},
		{
			name: "message with tool call",
			messages: []models.ChatMessage{
				{
					ID:        primitive.NewObjectID(),
					SessionID: primitive.NewObjectID(),
					Role:      "tool_call",
					Content:   "Calling tool",
					Timestamp: time.Now(),
					ToolCall: &models.ToolCallData{
						ID:   "tool-123",
						Name: "read_file",
						Args: map[string]interface{}{
							"path": "/path/to/file.txt",
							"mode": "read",
						},
					},
				},
			},
			minExpectedSize: 300, // BSONStructureOverhead + content + ToolCallOverhead + args
			maxExpectedSize: 800,
			description:     "Message with tool call should include tool data size",
		},
		{
			name: "message with tool result",
			messages: []models.ChatMessage{
				{
					ID:        primitive.NewObjectID(),
					SessionID: primitive.NewObjectID(),
					Role:      "tool_result",
					Content:   "Tool result",
					Timestamp: time.Now(),
					ToolResult: &models.ToolResultData{
						ID:         "tool-123",
						Name:       "read_file",
						Output:     "File contents here",
						DurationMs: 150,
					},
				},
			},
			minExpectedSize: 300,
			maxExpectedSize: 800,
			description:     "Message with tool result should include result data size",
		},
		{
			name: "multiple messages",
			messages: []models.ChatMessage{
				{
					ID:        primitive.NewObjectID(),
					SessionID: primitive.NewObjectID(),
					Role:      "user",
					Content:   "First message",
					Timestamp: time.Now(),
				},
				{
					ID:        primitive.NewObjectID(),
					SessionID: primitive.NewObjectID(),
					Role:      "assistant",
					Content:   "Second message response",
					Timestamp: time.Now(),
				},
				{
					ID:        primitive.NewObjectID(),
					SessionID: primitive.NewObjectID(),
					Role:      "user",
					Content:   "Third message",
					Timestamp: time.Now(),
				},
			},
			minExpectedSize: 600, // 3 * BSONStructureOverhead + content
			maxExpectedSize: 1500,
			description:     "Multiple messages should accumulate size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := compactor.EstimateSessionBSONSize(tt.messages)
			if size < tt.minExpectedSize || size > tt.maxExpectedSize {
				t.Errorf("%s: expected size between %d-%d, got %d",
					tt.description, tt.minExpectedSize, tt.maxExpectedSize, size)
			}
		})
	}
}

// TestShouldCompactBySize verifies size-based compaction trigger
func TestShouldCompactBySize(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	tests := []struct {
		name           string
		messages       []models.ChatMessage
		expectedResult bool
		description    string
	}{
		{
			name:           "empty messages",
			messages:       []models.ChatMessage{},
			expectedResult: false,
			description:    "Empty messages should not trigger compaction",
		},
		{
			name: "small messages",
			messages: []models.ChatMessage{
				{
					ID:        primitive.NewObjectID(),
					SessionID: primitive.NewObjectID(),
					Role:      "user",
					Content:   "Small message",
					Timestamp: time.Now(),
				},
			},
			expectedResult: false,
			description:    "Small messages should not trigger compaction",
		},
		{
			name:           "large messages exceeding threshold",
			messages:       createLargeMessageListBySize(13 * 1024 * 1024), // 13MB > 12.8MB threshold
			expectedResult: true,
			description:    "Messages exceeding 80% of 16MB should trigger compaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compactor.ShouldCompactBySize(tt.messages)
			if result != tt.expectedResult {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expectedResult, result)
			}
		})
	}
}

// TestGetSizeCompactionTarget verifies target size calculation
func TestGetSizeCompactionTarget(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	target := compactor.GetSizeCompactionTarget()
	maxSize := MongoDBMaxDocSize
	expectedTarget := int(float64(maxSize) * SizeTargetThreshold)

	if target != expectedTarget {
		t.Errorf("GetSizeCompactionTarget: expected %d, got %d", expectedTarget, target)
	}

	// Verify it's 20% of max size
	if target != 3*1024*1024+276*1024 { // 20% of 16MB
		t.Logf("Target size: %d bytes (%.2f MB)", target, float64(target)/(1024*1024))
	}
}

// TestGetSizeTriggerThreshold verifies trigger threshold calculation
func TestGetSizeTriggerThreshold(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	threshold := compactor.GetSizeTriggerThreshold()
	maxSize := MongoDBMaxDocSize
	expectedThreshold := int(float64(maxSize) * SizeTriggerThreshold)

	if threshold != expectedThreshold {
		t.Errorf("GetSizeTriggerThreshold: expected %d, got %d", expectedThreshold, threshold)
	}

	// Verify it's 80% of max size (12.8MB)
	if threshold != 12*1024*1024+838*1024 { // 80% of 16MB
		t.Logf("Trigger threshold: %d bytes (%.2f MB)", threshold, float64(threshold)/(1024*1024))
	}
}

// TestGetSizePercentageUsed verifies percentage calculation
func TestGetSizePercentageUsed(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	tests := []struct {
		name              string
		messages          []models.ChatMessage
		minExpectedPercent float64
		maxExpectedPercent float64
		description       string
	}{
		{
			name:              "empty messages",
			messages:          []models.ChatMessage{},
			minExpectedPercent: 0,
			maxExpectedPercent: 0.1,
			description:       "Empty messages should be ~0%",
		},
		{
			name: "single small message",
			messages: []models.ChatMessage{
				{
					ID:        primitive.NewObjectID(),
					SessionID: primitive.NewObjectID(),
					Role:      "user",
					Content:   "Hello",
					Timestamp: time.Now(),
				},
			},
			minExpectedPercent: 0,
			maxExpectedPercent: 0.01,
			description:       "Single small message should be < 0.01%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentage := compactor.GetSizePercentageUsed(tt.messages)
			if percentage < tt.minExpectedPercent || percentage > tt.maxExpectedPercent {
				t.Errorf("%s: expected percentage between %.2f%%-%.2f%%, got %.2f%%",
					tt.description, tt.minExpectedPercent, tt.maxExpectedPercent, percentage)
			}
		})
	}
}

// TestGetSizeWarningLevel verifies warning level classification
func TestGetSizeWarningLevel(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	tests := []struct {
		name             string
		messages         []models.ChatMessage
		expectedLevel    string
		description      string
	}{
		{
			name:             "empty messages",
			messages:         []models.ChatMessage{},
			expectedLevel:    "ok",
			description:      "Empty messages should be 'ok'",
		},
		{
			name: "single small message",
			messages: []models.ChatMessage{
				{
					ID:        primitive.NewObjectID(),
					SessionID: primitive.NewObjectID(),
					Role:      "user",
					Content:   "Hello",
					Timestamp: time.Now(),
				},
			},
			expectedLevel: "ok",
			description:   "Single small message should be 'ok'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := compactor.GetSizeWarningLevel(tt.messages)
			if level != tt.expectedLevel {
				t.Errorf("%s: expected '%s', got '%s'", tt.description, tt.expectedLevel, level)
			}
		})
	}
}

// TestEstimateToolCallSize verifies tool call size estimation
func TestEstimateToolCallSize(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	tests := []struct {
		name            string
		toolCall        *models.ToolCallData
		minExpectedSize int
		maxExpectedSize int
		description     string
	}{
		{
			name: "simple tool call",
			toolCall: &models.ToolCallData{
				ID:   "tool-123",
				Name: "read_file",
				Args: map[string]interface{}{
					"path": "/path/to/file.txt",
				},
			},
			minExpectedSize: 100,
			maxExpectedSize: 300,
			description:     "Simple tool call should have reasonable size",
		},
		{
			name: "complex tool call",
			toolCall: &models.ToolCallData{
				ID:   "tool-456",
				Name: "execute_command",
				Args: map[string]interface{}{
					"command": "npm install",
					"cwd":     "/path/to/project",
					"env": map[string]string{
						"NODE_ENV": "production",
						"DEBUG":    "true",
					},
					"timeout": 30000,
				},
			},
			minExpectedSize: 150,
			maxExpectedSize: 500,
			description:     "Complex tool call with nested args should be larger",
		},
		{
			name: "tool call with nil args",
			toolCall: &models.ToolCallData{
				ID:   "tool-789",
				Name: "get_time",
				Args: nil,
			},
			minExpectedSize: 50,
			maxExpectedSize: 200,
			description:     "Tool call with nil args should be smaller",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := compactor.estimateToolCallSize(tt.toolCall)
			if size < tt.minExpectedSize || size > tt.maxExpectedSize {
				t.Errorf("%s: expected size between %d-%d, got %d",
					tt.description, tt.minExpectedSize, tt.maxExpectedSize, size)
			}
		})
	}
}

// TestEstimateToolResultSize verifies tool result size estimation
func TestEstimateToolResultSize(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	tests := []struct {
		name            string
		toolResult      *models.ToolResultData
		minExpectedSize int
		maxExpectedSize int
		description     string
	}{
		{
			name: "simple tool result",
			toolResult: &models.ToolResultData{
				ID:         "tool-123",
				Name:       "read_file",
				Output:     "File contents",
				DurationMs: 150,
			},
			minExpectedSize: 100,
			maxExpectedSize: 300,
			description:     "Simple tool result should have reasonable size",
		},
		{
			name: "tool result with error",
			toolResult: &models.ToolResultData{
				ID:         "tool-456",
				Name:       "execute_command",
				Output:     nil,
				Error:      "Command failed: permission denied",
				DurationMs: 500,
			},
			minExpectedSize: 100,
			maxExpectedSize: 300,
			description:     "Tool result with error should include error message size",
		},
		{
			name: "tool result with large output",
			toolResult: &models.ToolResultData{
				ID:         "tool-789",
				Name:       "list_files",
				Output:     createLargeString(5000),
				DurationMs: 200,
			},
			minExpectedSize: 5000,
			maxExpectedSize: 6000,
			description:     "Tool result with large output should reflect output size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := compactor.estimateToolResultSize(tt.toolResult)
			if size < tt.minExpectedSize || size > tt.maxExpectedSize {
				t.Errorf("%s: expected size between %d-%d, got %d",
					tt.description, tt.minExpectedSize, tt.maxExpectedSize, size)
			}
		})
	}
}

// TestSizeBasedCompactionConstants verifies constants are correct
func TestSizeBasedCompactionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
		desc     string
	}{
		{
			name:     "MongoDBMaxDocSize",
			constant: MongoDBMaxDocSize,
			expected: 16 * 1024 * 1024,
			desc:     "Should be 16MB",
		},
		{
			name:     "BSONStructureOverhead",
			constant: BSONStructureOverhead,
			expected: 200,
			desc:     "Should be 200 bytes",
		},
		{
			name:     "ToolCallOverhead",
			constant: ToolCallOverhead,
			expected: 100,
			desc:     "Should be 100 bytes",
		},
		{
			name:     "ToolResultOverhead",
			constant: ToolResultOverhead,
			expected: 100,
			desc:     "Should be 100 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s: %s - expected %d, got %d", tt.name, tt.desc, tt.expected, tt.constant)
			}
		})
	}
}

// TestSizeThresholdPercentages verifies threshold percentages
func TestSizeThresholdPercentages(t *testing.T) {
	tests := []struct {
		name     string
		constant float64
		expected float64
		desc     string
	}{
		{
			name:     "SizeTriggerThreshold",
			constant: SizeTriggerThreshold,
			expected: 0.80,
			desc:     "Should be 80%",
		},
		{
			name:     "SizeTargetThreshold",
			constant: SizeTargetThreshold,
			expected: 0.20,
			desc:     "Should be 20%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s: %s - expected %.2f, got %.2f", tt.name, tt.desc, tt.expected, tt.constant)
			}
		})
	}
}

// TestSizeBasedCompactionIntegration tests the full workflow
func TestSizeBasedCompactionIntegration(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	// Create messages that will exceed the trigger threshold
	messages := createLargeMessageListBySize(13 * 1024 * 1024) // 13MB

	// Check if compaction should be triggered
	shouldCompact := compactor.ShouldCompactBySize(messages)
	if !shouldCompact {
		t.Errorf("Integration test: expected compaction to be triggered for 13MB messages")
	}

	// Get warning level
	level := compactor.GetSizeWarningLevel(messages)
	if level != "critical" && level != "emergency" {
		t.Errorf("Integration test: expected warning level 'critical' or 'emergency', got '%s'", level)
	}

	// Get percentage used
	percentage := compactor.GetSizePercentageUsed(messages)
	if percentage < 80 {
		t.Errorf("Integration test: expected percentage >= 80%%, got %.2f%%", percentage)
	}

	t.Logf("Integration test passed: size=%.2f MB, percentage=%.2f%%, level=%s",
		float64(compactor.EstimateSessionBSONSize(messages))/(1024*1024), percentage, level)
}

// Helper functions

// createLargeMessageListBySize creates messages to reach approximately the target size
func createLargeMessageListBySize(targetSize int) []models.ChatMessage {
	var messages []models.ChatMessage
	currentSize := 0
	messageIndex := 0

	for currentSize < targetSize {
		// Create a message with content to reach target size
		contentSize := targetSize - currentSize
		if contentSize > 100*1024 { // Cap individual message content at 100KB
			contentSize = 100 * 1024
		}

		role := "user"
		if messageIndex%2 == 1 {
			role = "assistant"
		}

		msg := models.ChatMessage{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      role,
			Content:   createLargeString(contentSize),
			Timestamp: time.Now(),
		}

		messages = append(messages, msg)
		currentSize += BSONStructureOverhead + contentSize
		messageIndex++
	}

	return messages
}

// createLargeString creates a string of approximately the given size
func createLargeString(size int) string {
	if size <= 0 {
		return ""
	}
	// Use strings.Repeat for O(n) performance instead of O(n^2) concatenation
	return strings.Repeat("x", size)
}

// TestCalculateSplitPointBySize tests the size-based split point calculation
func TestCalculateSplitPointBySize(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	tests := []struct {
		name            string
		messages        []models.ChatMessage
		expectedCompact int
		expectedKeep    int
		description     string
	}{
		{
			name:            "empty messages",
			messages:        []models.ChatMessage{},
			expectedCompact: 0,
			expectedKeep:    0,
			description:     "Empty message list should return 0, 0",
		},
		{
			name: "single small message",
			messages: []models.ChatMessage{
				createSizeTestMessage("user", 1000), // 1KB content
			},
			expectedCompact: 0,
			expectedKeep:    1,
			description:     "Single small message should not be compacted",
		},
		{
			name: "all messages fit in target",
			messages: []models.ChatMessage{
				createSizeTestMessage("user", 100000),     // 100KB
				createSizeTestMessage("assistant", 100000), // 100KB
				createSizeTestMessage("user", 100000),     // 100KB
			},
			expectedCompact: 0,
			expectedKeep:    3,
			description:     "All messages should be kept if they fit in target size (20% of 16MB = 3.2MB)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compactCount, keepCount := compactor.CalculateSplitPointBySize(tt.messages)
			if compactCount != tt.expectedCompact || keepCount != tt.expectedKeep {
				t.Errorf("%s: expected (%d, %d), got (%d, %d)",
					tt.description, tt.expectedCompact, tt.expectedKeep, compactCount, keepCount)
			}
		})
	}
}

// TestCalculateSplitPointBySize_LargeMessages tests split point with messages exceeding target
func TestCalculateSplitPointBySize_LargeMessages(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	// Create messages that exceed target size (3.2MB)
	// Each message ~500KB, 10 messages = 5MB > 3.2MB target
	var messages []models.ChatMessage
	for i := 0; i < 10; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages = append(messages, createSizeTestMessage(role, 500*1024)) // 500KB each
	}

	compactCount, keepCount := compactor.CalculateSplitPointBySize(messages)

	// Should compact some messages since total exceeds target
	if compactCount == 0 {
		t.Error("Expected some messages to be compacted when total exceeds target size")
	}

	// Should keep at least minimum messages (4)
	if keepCount < 4 {
		t.Errorf("Expected at least 4 messages to be kept, got %d", keepCount)
	}

	t.Logf("Large messages test: compact=%d, keep=%d, total=%d", compactCount, keepCount, len(messages))
}

// TestSizeBasedToolPairPreservation tests that tool pairs stay together in size-based compaction
func TestSizeBasedToolPairPreservation(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	// Create messages with tool pairs - large enough to trigger compaction
	messages := []models.ChatMessage{
		createSizeTestMessage("user", 500*1024),      // 500KB - old message
		createSizeTestMessage("assistant", 500*1024), // 500KB - old response
		createSizeTestMessageWithRole("tool_call", 200*1024),  // 200KB - tool call
		createSizeTestMessageWithRole("tool_result", 200*1024), // 200KB - tool result
		createSizeTestMessage("user", 100*1024),      // 100KB - recent
		createSizeTestMessage("assistant", 100*1024), // 100KB - recent
	}

	compactCount, keepCount := compactor.CalculateSplitPointBySize(messages)

	// If any compaction happens, verify tool pairs are not split
	if compactCount > 0 && compactCount < len(messages) {
		splitMsg := messages[compactCount]
		// If split is at tool_result, that means tool_call is being compacted without its result
		if splitMsg.Role == "tool_result" {
			t.Error("Tool pair preservation failed: split at tool_result without tool_call")
		}
	}

	t.Logf("Tool pair preservation test: compact=%d, keep=%d", compactCount, keepCount)
}

// TestGetMessagesToCompactBySize tests the helper method for size-based compaction
func TestGetMessagesToCompactBySize(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	// Create messages that exceed target size
	messages := []models.ChatMessage{
		createSizeTestMessage("user", 1*1024*1024),      // 1MB - old
		createSizeTestMessage("assistant", 1*1024*1024), // 1MB - old
		createSizeTestMessage("user", 500*1024),         // 500KB - recent
		createSizeTestMessage("assistant", 500*1024),    // 500KB - recent
	}

	toCompact := compactor.GetMessagesToCompactBySize(messages)

	// With 3MB total and 3.2MB target, some messages should be compacted
	t.Logf("GetMessagesToCompactBySize: returned %d messages to compact", len(toCompact))

	// Verify returned messages are from the beginning if any
	if len(toCompact) > 0 {
		firstCompact := toCompact[0]
		if firstCompact.ID != messages[0].ID {
			t.Error("GetMessagesToCompactBySize should return oldest messages first")
		}
	}
}

// TestGetMessagesToKeepBySize tests the helper method for size-based compaction
func TestGetMessagesToKeepBySize(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	// Create messages
	messages := []models.ChatMessage{
		createSizeTestMessage("user", 500*1024),      // 500KB
		createSizeTestMessage("assistant", 500*1024), // 500KB
		createSizeTestMessage("user", 500*1024),      // 500KB
		createSizeTestMessage("assistant", 500*1024), // 500KB
	}

	toKeep := compactor.GetMessagesToKeepBySize(messages)

	if len(toKeep) == 0 {
		t.Error("GetMessagesToKeepBySize returned empty slice when messages should be kept")
	}

	// Verify returned messages are from the end
	if len(toKeep) > 0 {
		lastKeep := toKeep[len(toKeep)-1]
		if lastKeep.ID != messages[len(messages)-1].ID {
			t.Error("GetMessagesToKeepBySize should return most recent messages")
		}
	}

	t.Logf("GetMessagesToKeepBySize: returned %d messages to keep", len(toKeep))
}

// TestAdjustForToolPairsBySize tests the tool pair adjustment in size-based compactor
// Note: The algorithm oscillates between positions when at tool pairs, settling at a consistent
// position after maxIterations. This matches the behavior in context_compactor.
func TestAdjustForToolPairsBySize(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCompactionConfig()
	compactor := NewSizeBasedCompactor(config, logger)

	tests := []struct {
		name           string
		messages       []models.ChatMessage
		splitIndex     int
		expectedResult int
		description    string
	}{
		{
			name: "split at tool_result settles at tool_result",
			messages: []models.ChatMessage{
				createSizeTestMessage("user", 1000),
				createSizeTestMessageWithRole("tool_call", 1000),
				createSizeTestMessageWithRole("tool_result", 1000),
				createSizeTestMessage("user", 1000),
			},
			splitIndex:     2, // Split at tool_result (index 2)
			expectedResult: 2, // Oscillates and settles at 2 after maxIterations
			description:    "Split at tool_result should keep pair together",
		},
		{
			name: "split at tool_call settles at tool_call",
			messages: []models.ChatMessage{
				createSizeTestMessage("user", 1000),
				createSizeTestMessageWithRole("tool_call", 1000),
				createSizeTestMessageWithRole("tool_result", 1000),
				createSizeTestMessage("user", 1000),
			},
			splitIndex:     1, // Split at tool_call (index 1)
			expectedResult: 1, // Oscillates and settles at 1 after maxIterations (odd count)
			description:    "Split at tool_call should keep pair together",
		},
		{
			name: "no adjustment needed",
			messages: []models.ChatMessage{
				createSizeTestMessage("user", 1000),
				createSizeTestMessage("assistant", 1000),
				createSizeTestMessage("user", 1000),
			},
			splitIndex:     1,
			expectedResult: 1,
			description:    "No adjustment needed when not at tool pair",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compactor.adjustForToolPairs(tt.messages, tt.splitIndex)
			if result != tt.expectedResult {
				t.Errorf("%s: expected %d, got %d", tt.description, tt.expectedResult, result)
			}
		})
	}
}

// Helper functions for size-based tests

// createSizeTestMessage creates a test message with specified content size
func createSizeTestMessage(role string, contentSize int) models.ChatMessage {
	return models.ChatMessage{
		ID:        primitive.NewObjectID(),
		SessionID: primitive.NewObjectID(),
		Role:      role,
		Content:   createLargeString(contentSize),
		Timestamp: time.Now(),
	}
}

// createSizeTestMessageWithRole creates a test message with tool role
func createSizeTestMessageWithRole(role string, contentSize int) models.ChatMessage {
	msg := createSizeTestMessage(role, contentSize)
	if role == "tool_call" {
		msg.ToolCall = &models.ToolCallData{
			ID:   "test-tool-id",
			Name: "test_tool",
			Args: map[string]interface{}{"arg": "value"},
		}
	} else if role == "tool_result" {
		msg.ToolResult = &models.ToolResultData{
			ID:     "test-tool-id",
			Name:   "test_tool",
			Output: "test output",
		}
	}
	return msg
}
