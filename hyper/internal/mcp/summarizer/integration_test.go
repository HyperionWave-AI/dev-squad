package summarizer

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// MockLLMClient is a mock implementation of LLMClient for testing
type MockLLMClient struct {
	callCount int
	responses map[string]string
}

func (m *MockLLMClient) Summarize(ctx context.Context, code string, metadata CodeMetadata) (string, error) {
	m.callCount++
	if response, ok := m.responses[code]; ok {
		return response, nil
	}
	return "Mock summary for: " + metadata.NodeName, nil
}

func (m *MockLLMClient) Close() error {
	return nil
}

// TestIntegrationCacheAndTokenManager tests cache and token manager integration
func TestIntegrationCacheAndTokenManager(t *testing.T) {
	logger := zap.NewNop()
	config := SummarizerConfig{
		Enabled:             true,
		Model:               "gpt-4",
		MaxTokens:           500,
		CacheEnabled:        true,
		CacheSize:           100,
		CacheTTL:            5 * time.Minute,
		LLMAPIKey:           "test-key",
		LLMTimeout:          10 * time.Second,
		TokenBudget:         1000,
		TokenPerResult:      100,
		MetricsEnabled:      true,
	}

	summarizer, err := NewLLMSummarizer(config, logger)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}
	defer summarizer.Close()

	// Replace with mock client
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"func test() {}": "Test function",
		},
	}

	code := "func test() {}"
	metadata := CodeMetadata{
		FilePath:  "test.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "test",
		LineStart: 1,
		LineEnd:   1,
	}

	// First call should hit LLM
	summary1, err := summarizer.Summarize(context.Background(), code, metadata)
	if err != nil {
		t.Fatalf("First summarize failed: %v", err)
	}

	if summary1.Type != "llm" {
		t.Errorf("Expected type 'llm', got '%s'", summary1.Type)
	}

	if summary1.CacheHit {
		t.Error("First call should not be a cache hit")
	}

	// Check token usage was recorded
	tokenMetrics := summarizer.GetTokenMetrics()
	if tokenMetrics.TotalTokensUsed == 0 {
		t.Error("Token usage should be recorded")
	}

	// Second call should hit cache
	summary2, err := summarizer.Summarize(context.Background(), code, metadata)
	if err != nil {
		t.Fatalf("Second summarize failed: %v", err)
	}

	if summary2.Type != "cached" {
		t.Errorf("Expected type 'cached', got '%s'", summary2.Type)
	}

	if !summary2.CacheHit {
		t.Error("Second call should be a cache hit")
	}

	// Check cache stats
	cacheStats := summarizer.GetCacheStats()
	if cacheStats.Hits == 0 {
		t.Error("Cache hits should be recorded")
	}

	if cacheStats.HitRate == 0 {
		t.Error("Cache hit rate should be calculated")
	}
}

// TestIntegrationMetricsCollection tests metrics collection during summarization
func TestIntegrationMetricsCollection(t *testing.T) {
	logger := zap.NewNop()
	config := SummarizerConfig{
		Enabled:             true,
		Model:               "gpt-4",
		MaxTokens:           500,
		CacheEnabled:        true,
		CacheSize:           100,
		CacheTTL:            5 * time.Minute,
		LLMAPIKey:           "test-key",
		LLMTimeout:          10 * time.Second,
		TokenBudget:         1000,
		TokenPerResult:      100,
		MetricsEnabled:      true,
	}

	summarizer, err := NewLLMSummarizer(config, logger)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}
	defer summarizer.Close()

	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"code1": "Summary 1",
			"code2": "Summary 2",
		},
	}

	// Generate multiple summaries
	for i := 0; i < 3; i++ {
		code := "code1"
		if i == 2 {
			code = "code2"
		}

		metadata := CodeMetadata{
			FilePath:  "test.go",
			Language:  "go",
			NodeType:  "function",
			NodeName:  "test",
			LineStart: 1,
			LineEnd:   1,
		}

		_, err := summarizer.Summarize(context.Background(), code, metadata)
		if err != nil {
			t.Fatalf("Summarize failed: %v", err)
		}
	}

	// Check metrics
	metrics := summarizer.GetMetrics()

	if metrics.TotalCount != 3 {
		t.Errorf("Expected 3 total summaries, got %d", metrics.TotalCount)
	}

	if metrics.AICount != 2 {
		t.Errorf("Expected 2 AI summaries, got %d", metrics.AICount)
	}

	if metrics.CachedCount != 1 {
		t.Errorf("Expected 1 cached summary, got %d", metrics.CachedCount)
	}

	breakdown := summarizer.metrics.GetSummaryBreakdown()
	if breakdown["total"] != int64(3) {
		t.Errorf("Expected total 3 in breakdown, got %v", breakdown["total"])
	}
}

// TestIntegrationTokenBudgetEnforcement tests token budget enforcement
func TestIntegrationTokenBudgetEnforcement(t *testing.T) {
	logger := zap.NewNop()
	config := SummarizerConfig{
		Enabled:             true,
		Model:               "gpt-4",
		MaxTokens:           500,
		CacheEnabled:        false,
		CacheSize:           100,
		CacheTTL:            5 * time.Minute,
		LLMAPIKey:           "test-key",
		LLMTimeout:          10 * time.Second,
		TokenBudget:         50,  // Very small budget
		TokenPerResult:      100,
		MetricsEnabled:      true,
	}

	summarizer, err := NewLLMSummarizer(config, logger)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}
	defer summarizer.Close()

	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"large code block": "This is a very long summary that will consume many tokens",
		},
	}

	code := "large code block"
	metadata := CodeMetadata{
		FilePath:  "test.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "test",
		LineStart: 1,
		LineEnd:   1,
	}

	// First call should fail due to token budget
	_, err = summarizer.Summarize(context.Background(), code, metadata)
	if err == nil {
		t.Error("Expected error due to token budget exhaustion")
	}

	// Check that error was recorded in metrics
	metrics := summarizer.GetMetrics()
	if metrics.ErrorCount == 0 {
		t.Error("Error should be recorded in metrics")
	}
}

// TestIntegrationCacheExpiration tests cache TTL expiration
func TestIntegrationCacheExpiration(t *testing.T) {
	logger := zap.NewNop()
	config := SummarizerConfig{
		Enabled:             true,
		Model:               "gpt-4",
		MaxTokens:           500,
		CacheEnabled:        true,
		CacheSize:           100,
		CacheTTL:            100 * time.Millisecond, // Very short TTL for testing
		LLMAPIKey:           "test-key",
		LLMTimeout:          10 * time.Second,
		TokenBudget:         1000,
		TokenPerResult:      100,
		MetricsEnabled:      true,
	}

	summarizer, err := NewLLMSummarizer(config, logger)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}
	defer summarizer.Close()

	callCount := 0
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}

	code := "test code"
	metadata := CodeMetadata{
		FilePath:  "test.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "test",
		LineStart: 1,
		LineEnd:   1,
	}

	// First call - should hit LLM
	_, err = summarizer.Summarize(context.Background(), code, metadata)
	if err != nil {
		t.Fatalf("First summarize failed: %v", err)
	}
	callCount++

	// Second call - should hit cache
	_, err = summarizer.Summarize(context.Background(), code, metadata)
	if err != nil {
		t.Fatalf("Second summarize failed: %v", err)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call - should hit LLM again (cache expired)
	_, err = summarizer.Summarize(context.Background(), code, metadata)
	if err != nil {
		t.Fatalf("Third summarize failed: %v", err)
	}
	callCount++

	// Check cache stats - should show misses after expiration
	cacheStats := summarizer.GetCacheStats()
	if cacheStats.Misses == 0 {
		t.Error("Cache misses should be recorded after expiration")
	}
}

// TestIntegrationMetricsReset tests metrics reset functionality
func TestIntegrationMetricsReset(t *testing.T) {
	logger := zap.NewNop()
	config := SummarizerConfig{
		Enabled:             true,
		Model:               "gpt-4",
		MaxTokens:           500,
		CacheEnabled:        true,
		CacheSize:           100,
		CacheTTL:            5 * time.Minute,
		LLMAPIKey:           "test-key",
		LLMTimeout:          10 * time.Second,
		TokenBudget:         1000,
		TokenPerResult:      100,
		MetricsEnabled:      true,
	}

	summarizer, err := NewLLMSummarizer(config, logger)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}
	defer summarizer.Close()

	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"code": "Summary",
		},
	}

	// Generate a summary
	metadata := CodeMetadata{
		FilePath:  "test.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "test",
		LineStart: 1,
		LineEnd:   1,
	}

	_, err = summarizer.Summarize(context.Background(), "code", metadata)
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	// Check metrics before reset
	metricsBefore := summarizer.GetMetrics()
	if metricsBefore.TotalCount == 0 {
		t.Error("Metrics should be recorded")
	}

	// Reset metrics
	summarizer.ResetMetrics()

	// Check metrics after reset
	metricsAfter := summarizer.GetMetrics()
	if metricsAfter.TotalCount != 0 {
		t.Error("Metrics should be reset to zero")
	}

	tokenMetrics := summarizer.GetTokenMetrics()
	if tokenMetrics.TotalTokensUsed != 0 {
		t.Error("Token metrics should be reset to zero")
	}
}

// TestIntegrationPerformanceStats tests performance statistics
func TestIntegrationPerformanceStats(t *testing.T) {
	logger := zap.NewNop()
	config := SummarizerConfig{
		Enabled:             true,
		Model:               "gpt-4",
		MaxTokens:           500,
		CacheEnabled:        true,
		CacheSize:           100,
		CacheTTL:            5 * time.Minute,
		LLMAPIKey:           "test-key",
		LLMTimeout:          10 * time.Second,
		TokenBudget:         1000,
		TokenPerResult:      100,
		MetricsEnabled:      true,
	}

	summarizer, err := NewLLMSummarizer(config, logger)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}
	defer summarizer.Close()

	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"code": "Summary",
		},
	}

	// Generate summaries
	metadata := CodeMetadata{
		FilePath:  "test.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "test",
		LineStart: 1,
		LineEnd:   1,
	}

	for i := 0; i < 5; i++ {
		_, err := summarizer.Summarize(context.Background(), "code", metadata)
		if err != nil {
			t.Fatalf("Summarize failed: %v", err)
		}
	}

	// Get performance stats
	perfStats := summarizer.metrics.GetPerformanceStats()

	if perfStats["averageLatencyMs"] == nil {
		t.Error("Average latency should be recorded")
	}

	if perfStats["tokensUsed"] == nil {
		t.Error("Tokens used should be recorded")
	}

	if perfStats["cacheHitRate"] == nil {
		t.Error("Cache hit rate should be recorded")
	}
}
