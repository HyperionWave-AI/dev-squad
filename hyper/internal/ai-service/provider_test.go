package aiservice

import (
	"fmt"
	"testing"
)

// TestDetectFileType tests file type detection for various file paths
func TestDetectFileType(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	tests := []struct {
		filePath string
		expected string
	}{
		// UI files
		{"./ui/src/components/TaskCard.tsx", "ui"},
		{"./ui/src/pages/Dashboard.jsx", "ui"},
		{"./ui/src/styles/main.css", "ui"},
		{"./ui/src/styles/theme.scss", "ui"},

		// Backend files
		{"./hyper/internal/handlers/auth.go", "backend"},
		{"./hyper/internal/middleware/logger.go", "backend"},
		{"./cmd/server/main.go", "backend"},

		// Test files
		{"./hyper/internal/handlers/auth_test.go", "test"},
		{"./ui/src/components/Button.test.tsx", "test"},
		{"./ui/src/utils/helpers.spec.ts", "test"},

		// Config files
		{".env", "config"},
		{"./config/app.yaml", "config"},
		{"./config/database.json", "config"},
		{"./docker-compose.yml", "config"},

		// Other files
		{"./README.md", "other"},
		{"./Makefile", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := summarizer.DetectFileType(tt.filePath)
			if result != tt.expected {
				t.Errorf("DetectFileType(%s) = %s, want %s", tt.filePath, result, tt.expected)
			}
		})
	}
}

// TestCalculateRelevanceScore tests relevance scoring logic
func TestCalculateRelevanceScore(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	tests := []struct {
		matchType      string
		hasContextHint bool
		expectedMin    float64
		expectedMax    float64
	}{
		// Exact match without context
		{"exact", false, 0.75, 0.85},
		// Exact match with context
		{"exact", true, 0.90, 1.0},
		// Partial match without context
		{"partial", false, 0.45, 0.55},
		// Partial match with context
		{"partial", true, 0.60, 0.70},
		// Contextual match without context
		{"contextual", false, 0.25, 0.35},
		// Contextual match with context
		{"contextual", true, 0.40, 0.50},
	}

	for _, tt := range tests {
		t.Run(tt.matchType, func(t *testing.T) {
			score := summarizer.CalculateRelevanceScore(tt.matchType, tt.hasContextHint)
			if score < tt.expectedMin || score > tt.expectedMax {
				t.Errorf("CalculateRelevanceScore(%s, %v) = %f, want between %f and %f",
					tt.matchType, tt.hasContextHint, score, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// TestExtractMetadata tests metadata extraction from result maps
func TestExtractMetadata(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	result := map[string]interface{}{
		"filePath":  "./ui/src/components/TaskCard.tsx",
		"lineNumber": 42.0,
		"matchType": "exact",
		"context":   "delete button implementation",
	}

	metadata := summarizer.ExtractMetadata(result)

	if metadata.FilePath != "./ui/src/components/TaskCard.tsx" {
		t.Errorf("Expected filePath './ui/src/components/TaskCard.tsx', got '%s'", metadata.FilePath)
	}

	if metadata.LineNumber != 42 {
		t.Errorf("Expected lineNumber 42, got %d", metadata.LineNumber)
	}

	if metadata.FileType != "ui" {
		t.Errorf("Expected fileType 'ui', got '%s'", metadata.FileType)
	}

	if metadata.MatchType != "exact" {
		t.Errorf("Expected matchType 'exact', got '%s'", metadata.MatchType)
	}

	if metadata.RelevanceScore < 0.9 || metadata.RelevanceScore > 1.0 {
		t.Errorf("Expected relevance score between 0.9 and 1.0, got %f", metadata.RelevanceScore)
	}
}

// TestSummarizeResults tests the full summarization pipeline
func TestSummarizeResults(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	results := []interface{}{
		map[string]interface{}{
			"filePath":  "./ui/src/components/TaskCard.tsx",
			"lineNumber": 42.0,
			"matchType": "exact",
			"context":   "delete button",
		},
		map[string]interface{}{
			"filePath":  "./ui/src/pages/Settings.tsx",
			"lineNumber": 15.0,
			"matchType": "partial",
			"context":   "button component",
		},
		map[string]interface{}{
			"filePath":  "./hyper/internal/handlers/auth.go",
			"lineNumber": 89.0,
			"matchType": "exact",
			"context":   "authentication logic",
		},
		map[string]interface{}{
			"filePath":  "./hyper/internal/handlers/handlers_test.go",
			"lineNumber": 12.0,
			"matchType": "contextual",
			"context":   "test case",
		},
	}

	summary := summarizer.SummarizeResults(results)

	// Verify summary contains expected content
	if len(summary) == 0 {
		t.Error("Expected non-empty summary")
	}

	// Check for category headers
	if !contains(summary, "UI Components") {
		t.Error("Expected 'UI Components' in summary")
	}

	if !contains(summary, "Backend Services") {
		t.Error("Expected 'Backend Services' in summary")
	}

	if !contains(summary, "Tests") {
		t.Error("Expected 'Tests' in summary")
	}

	// Check for file paths
	if !contains(summary, "TaskCard.tsx") {
		t.Error("Expected 'TaskCard.tsx' in summary")
	}

	if !contains(summary, "auth.go") {
		t.Error("Expected 'auth.go' in summary")
	}

	// Check for most relevant recommendation
	if !contains(summary, "Most relevant") {
		t.Error("Expected 'Most relevant' in summary")
	}
}

// TestSummarizeResultsEmpty tests summarization with empty results
func TestSummarizeResultsEmpty(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	summary := summarizer.SummarizeResults([]interface{}{})

	if summary != "No results found." {
		t.Errorf("Expected 'No results found.', got '%s'", summary)
	}
}

// TestSummarizeResultsSingleFile tests summarization with single file
func TestSummarizeResultsSingleFile(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	results := []interface{}{
		map[string]interface{}{
			"filePath":  "./ui/src/components/Button.tsx",
			"lineNumber": 10.0,
			"matchType": "exact",
			"context":   "button component",
		},
	}

	summary := summarizer.SummarizeResults(results)

	if !contains(summary, "Found 1 results") {
		t.Error("Expected 'Found 1 results' in summary")
	}

	if !contains(summary, "Button.tsx") {
		t.Error("Expected 'Button.tsx' in summary")
	}
}

// TestSummarizeResultsLargeSet tests summarization with many files
func TestSummarizeResultsLargeSet(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	// Create 15 results
	results := make([]interface{}, 15)
	for i := 0; i < 15; i++ {
		results[i] = map[string]interface{}{
			"filePath":  "./ui/src/components/Component" + string(rune(i)) + ".tsx",
			"lineNumber": float64(i * 10),
			"matchType": "partial",
			"context":   "component",
		}
	}

	summary := summarizer.SummarizeResults(results)

	if !contains(summary, "Found 15 results") {
		t.Error("Expected 'Found 15 results' in summary")
	}

	// Verify summary is much smaller than raw results
	rawSize := len(fmt.Sprintf("%v", results))
	summarySize := len(summary)

	if summarySize >= rawSize {
		t.Errorf("Summary should be smaller than raw results. Raw: %d, Summary: %d", rawSize, summarySize)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ==================== Cache Breakpoint Tests ====================

// TestDefaultCacheBreakpointConfig tests the default configuration values
func TestDefaultCacheBreakpointConfig(t *testing.T) {
	config := DefaultCacheBreakpointConfig()

	if !config.Enabled {
		t.Error("Expected Enabled to be true by default")
	}

	if config.StandardInterval != 10 {
		t.Errorf("Expected StandardInterval 10, got %d", config.StandardInterval)
	}

	if config.MinMessagesForCaching != 5 {
		t.Errorf("Expected MinMessagesForCaching 5, got %d", config.MinMessagesForCaching)
	}
}

// TestAddMessageCacheBreakpoints_DisabledConfig tests that breakpoints are skipped when disabled
func TestAddMessageCacheBreakpoints_DisabledConfig(t *testing.T) {
	provider := &anthropicProvider{
		cacheBreakpointConfig: CacheBreakpointConfig{
			Enabled:               false,
			StandardInterval:      10,
			MinMessagesForCaching: 5,
		},
	}

	messages := createTestMessages(15)
	result := provider.addMessageCacheBreakpoints(messages)

	// Should return messages unchanged
	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}

	// Verify no cache control was added
	for i, msg := range result {
		if hasCacheControl(msg) {
			t.Errorf("Message %d should not have cache control when disabled", i)
		}
	}
}

// TestAddMessageCacheBreakpoints_TooFewMessages tests that breakpoints are skipped for short conversations
func TestAddMessageCacheBreakpoints_TooFewMessages(t *testing.T) {
	provider := &anthropicProvider{
		cacheBreakpointConfig: DefaultCacheBreakpointConfig(),
	}

	messages := createTestMessages(3) // Less than MinMessagesForCaching (5)
	result := provider.addMessageCacheBreakpoints(messages)

	// Should return messages unchanged
	if len(result) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(result))
	}

	// Verify no cache control was added
	for i, msg := range result {
		if hasCacheControl(msg) {
			t.Errorf("Message %d should not have cache control with too few messages", i)
		}
	}
}

// TestAddMessageCacheBreakpoints_StandardInterval tests breakpoints at message 10
func TestAddMessageCacheBreakpoints_StandardInterval(t *testing.T) {
	provider := &anthropicProvider{
		cacheBreakpointConfig: DefaultCacheBreakpointConfig(),
	}

	// Create 12 messages (positions 1-12)
	// With StandardInterval=10, position 10 should get breakpoint
	messages := createTestMessages(12)
	result := provider.addMessageCacheBreakpoints(messages)

	if len(result) != 12 {
		t.Errorf("Expected 12 messages, got %d", len(result))
	}

	// Message at index 9 (position 10) should have cache control
	if !hasCacheControl(result[9]) {
		t.Error("Expected cache control on message at position 10 (index 9)")
	}

	// First breakpoint (later message) should have 5m TTL
	ttl := extractCacheTTL(result[9])
	if ttl != "5m" {
		t.Errorf("Expected TTL '5m' for first breakpoint, got '%s'", ttl)
	}
}

// TestAddMessageCacheBreakpoints_TwoBreakpoints tests that two breakpoints are placed correctly
func TestAddMessageCacheBreakpoints_TwoBreakpoints(t *testing.T) {
	provider := &anthropicProvider{
		cacheBreakpointConfig: DefaultCacheBreakpointConfig(),
	}

	// Create 25 messages (positions 1-25)
	// With StandardInterval=10, positions 20 and 10 should get breakpoints
	messages := createTestMessages(25)
	result := provider.addMessageCacheBreakpoints(messages)

	if len(result) != 25 {
		t.Errorf("Expected 25 messages, got %d", len(result))
	}

	// Count breakpoints
	breakpointCount := 0
	for _, msg := range result {
		if hasCacheControl(msg) {
			breakpointCount++
		}
	}

	if breakpointCount != 2 {
		t.Errorf("Expected 2 breakpoints, got %d", breakpointCount)
	}

	// Message at index 19 (position 20) should have 5m TTL (first found = later message)
	if !hasCacheControl(result[19]) {
		t.Error("Expected cache control on message at position 20")
	}
	ttl20 := extractCacheTTL(result[19])
	if ttl20 != "5m" {
		t.Errorf("Expected TTL '5m' for position 20 (first breakpoint), got '%s'", ttl20)
	}

	// Message at index 9 (position 10) should have 1h TTL (second found = earlier message)
	if !hasCacheControl(result[9]) {
		t.Error("Expected cache control on message at position 10")
	}
	ttl10 := extractCacheTTL(result[9])
	if ttl10 != "1h" {
		t.Errorf("Expected TTL '1h' for position 10 (second breakpoint), got '%s'", ttl10)
	}
}

// TestAddMessageCacheBreakpoints_MaxTwoBreakpoints tests that only 2 breakpoints are added max
func TestAddMessageCacheBreakpoints_MaxTwoBreakpoints(t *testing.T) {
	provider := &anthropicProvider{
		cacheBreakpointConfig: DefaultCacheBreakpointConfig(),
	}

	// Create 45 messages - positions 40, 30, 20, 10 are potential breakpoints
	// Only 2 should be added (40 and 30)
	messages := createTestMessages(45)
	result := provider.addMessageCacheBreakpoints(messages)

	breakpointCount := 0
	for _, msg := range result {
		if hasCacheControl(msg) {
			breakpointCount++
		}
	}

	if breakpointCount != 2 {
		t.Errorf("Expected exactly 2 breakpoints (max limit), got %d", breakpointCount)
	}

	// Should be at positions 40 and 30, NOT 20 and 10
	if !hasCacheControl(result[39]) {
		t.Error("Expected cache control at position 40")
	}
	if !hasCacheControl(result[29]) {
		t.Error("Expected cache control at position 30")
	}
	if hasCacheControl(result[19]) {
		t.Error("Position 20 should NOT have cache control (max 2 breakpoints)")
	}
	if hasCacheControl(result[9]) {
		t.Error("Position 10 should NOT have cache control (max 2 breakpoints)")
	}
}

// TestAddMessageCacheBreakpoints_StringContent tests conversion of string content to array format
func TestAddMessageCacheBreakpoints_StringContent(t *testing.T) {
	provider := &anthropicProvider{
		cacheBreakpointConfig: CacheBreakpointConfig{
			Enabled:               true,
			StandardInterval:      5, // Every 5 messages
			MinMessagesForCaching: 3,
		},
	}

	// Create messages with string content
	messages := []map[string]interface{}{
		{"role": "user", "content": "Message 1"},
		{"role": "assistant", "content": "Message 2"},
		{"role": "user", "content": "Message 3"},
		{"role": "assistant", "content": "Message 4"},
		{"role": "user", "content": "Message 5"}, // Position 5, should get breakpoint
	}

	result := provider.addMessageCacheBreakpoints(messages)

	// Position 5 (index 4) should be converted to array format with cache_control
	msg5 := result[4]
	content, ok := msg5["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected content to be converted to array format")
	}

	if len(content) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(content))
	}

	if content[0]["type"] != "text" {
		t.Errorf("Expected type 'text', got '%v'", content[0]["type"])
	}

	if content[0]["text"] != "Message 5" {
		t.Errorf("Expected text 'Message 5', got '%v'", content[0]["text"])
	}

	cacheControl, ok := content[0]["cache_control"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected cache_control to be present")
	}

	if cacheControl["type"] != "ephemeral" {
		t.Errorf("Expected cache_control type 'ephemeral', got '%v'", cacheControl["type"])
	}
}

// Helper functions for cache breakpoint tests

func createTestMessages(count int) []map[string]interface{} {
	messages := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages[i] = map[string]interface{}{
			"role":    role,
			"content": fmt.Sprintf("Message %d", i+1),
		}
	}
	return messages
}

func hasCacheControl(msg map[string]interface{}) bool {
	// Check string content that was converted to array
	if contentArr, ok := msg["content"].([]map[string]interface{}); ok && len(contentArr) > 0 {
		lastBlock := contentArr[len(contentArr)-1]
		_, hasCacheCtrl := lastBlock["cache_control"]
		return hasCacheCtrl
	}
	return false
}

func extractCacheTTL(msg map[string]interface{}) string {
	if contentArr, ok := msg["content"].([]map[string]interface{}); ok && len(contentArr) > 0 {
		lastBlock := contentArr[len(contentArr)-1]
		if cacheControl, ok := lastBlock["cache_control"].(map[string]interface{}); ok {
			if ttl, ok := cacheControl["ttl"].(string); ok {
				return ttl
			}
		}
	}
	return ""
}
