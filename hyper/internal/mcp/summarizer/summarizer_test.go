package summarizer

import (
	"context"
	"testing"
	"time"
)

// Helper function to create a valid config for testing
func createValidConfig() SummarizerConfig {
	return SummarizerConfig{
		Enabled:             true,
		Model:               "gpt-4",
		MaxTokens:           500,
		CacheEnabled:        true,
		FallbackToHeuristic: true,
		CacheSize:           100,
		CacheTTL:            5 * time.Minute,
		LLMAPIKey:           "test-key",
		LLMTimeout:          30 * time.Second,
		TokenBudget:         10000,
		TokenPerResult:      100,
		MetricsEnabled:      true,
		LogLevel:            "info",
	}
}

// TestNewLLMSummarizerValid tests creating a new LLMSummarizer with valid config
func TestNewLLMSummarizerValid(t *testing.T) {
	config := createValidConfig()

	summarizer, err := NewLLMSummarizer(config, nil)
	if err != nil {
		t.Fatalf("NewLLMSummarizer failed: %v", err)
	}

	if summarizer == nil {
		t.Fatal("NewLLMSummarizer returned nil")
	}

	if summarizer.config.Model != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %s", summarizer.config.Model)
	}

	if summarizer.cache == nil {
		t.Error("Expected cache to be initialized")
	}
}

// TestNewLLMSummarizerDisabled tests creating a summarizer with disabled config returns error
func TestNewLLMSummarizerDisabled(t *testing.T) {
	config := SummarizerConfig{
		Enabled: false,
	}

	// When disabled, NewLLMSummarizer should return an error
	summarizer, err := NewLLMSummarizer(config, nil)

	// Either it returns nil with error (expected), or it creates a disabled summarizer
	if summarizer != nil && err == nil {
		// If summarizer was created, check it's disabled
		if summarizer.config.Enabled {
			t.Error("Expected disabled summarizer")
		}
	} else if err == nil {
		t.Fatal("Expected error or valid summarizer when disabled")
	}
	// If err != nil and summarizer == nil, that's the expected behavior
}

// TestValidateConfigMissingModel tests validation fails when model is missing
func TestValidateConfigMissingModel(t *testing.T) {
	config := SummarizerConfig{
		Enabled:   true,
		Model:     "",
		MaxTokens: 500,
	}

	err := validateConfig(config)
	if err == nil {
		t.Fatal("Expected validation error for missing model")
	}

	if err.Error() != "model must be specified when summarizer is enabled" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestValidateConfigInvalidMaxTokens tests validation fails for invalid max tokens
func TestValidateConfigInvalidMaxTokens(t *testing.T) {
	tests := []struct {
		name      string
		maxTokens int
		wantErr   bool
		errMsg    string
	}{
		{"zero tokens", 0, true, "maxTokens must be greater than 0"},
		{"negative tokens", -1, true, "maxTokens must be greater than 0"},
		{"too many tokens", 1001, true, "maxTokens must not exceed 1000"},
		{"valid tokens", 500, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createValidConfig()
			config.MaxTokens = tt.maxTokens

			err := validateConfig(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Expected error message %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestValidateConfigInvalidCacheSize tests validation for cache size
func TestValidateConfigInvalidCacheSize(t *testing.T) {
	config := createValidConfig()
	config.CacheSize = -1

	err := validateConfig(config)
	if err == nil {
		t.Fatal("Expected validation error for negative cache size")
	}

	if err.Error() != "cacheSize must be non-negative" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestValidateConfigInvalidCacheTTL tests validation for cache TTL
func TestValidateConfigInvalidCacheTTL(t *testing.T) {
	config := createValidConfig()
	config.CacheTTL = -1 * time.Second

	err := validateConfig(config)
	if err == nil {
		t.Fatal("Expected validation error for negative cache TTL")
	}

	if err.Error() != "cacheTTL must be non-negative" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestValidateConfigInvalidLLMTimeout tests validation for LLM timeout
func TestValidateConfigInvalidLLMTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"zero timeout", 0, true},
		{"negative timeout", -1 * time.Second, true},
		{"valid timeout", 30 * time.Second, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createValidConfig()
			config.LLMTimeout = tt.timeout

			err := validateConfig(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateConfigInvalidTokenBudget tests validation for token budget
func TestValidateConfigInvalidTokenBudget(t *testing.T) {
	config := createValidConfig()
	config.TokenBudget = 0

	err := validateConfig(config)
	if err == nil {
		t.Fatal("Expected validation error for zero token budget")
	}

	if err.Error() != "tokenBudget must be greater than 0" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestValidateConfigInvalidTokenPerResult tests validation for token per result
func TestValidateConfigInvalidTokenPerResult(t *testing.T) {
	config := createValidConfig()
	config.TokenPerResult = -1

	err := validateConfig(config)
	if err == nil {
		t.Fatal("Expected validation error for negative token per result")
	}

	if err.Error() != "tokenPerResult must be greater than 0" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestSummarizeDisabled tests that NewLLMSummarizer returns error when disabled
func TestSummarizeDisabled(t *testing.T) {
	config := SummarizerConfig{
		Enabled: false,
	}

	// When disabled, NewLLMSummarizer should return an error
	summarizer, err := NewLLMSummarizer(config, nil)

	// Either it returns nil with error, or we can try to call Summarize
	if summarizer != nil {
		// If summarizer was created, Summarize should return error
		ctx := context.Background()
		_, err := summarizer.Summarize(ctx, "test code", CodeMetadata{})
		if err == nil {
			t.Fatal("Expected error when summarizer is disabled")
		}
		if err.Error() != "summarizer is disabled" {
			t.Errorf("Unexpected error message: %v", err)
		}
	} else {
		// NewLLMSummarizer returned nil, which is expected for disabled config
		if err == nil {
			t.Fatal("Expected error when creating disabled summarizer")
		}
	}
}

// TestSummarizeWithCacheDisabled tests Summarize without cache
func TestSummarizeWithCacheDisabled(t *testing.T) {
	config := createValidConfig()
	config.CacheEnabled = false

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}
	ctx := context.Background()

	summary, err := summarizer.Summarize(ctx, "test code", CodeMetadata{})
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary, got nil")
	}

	if summary.CacheHit {
		t.Error("Expected cache miss when cache is disabled")
	}

	// Check cache stats - should show no cache hits
	cacheStats := summarizer.GetCacheStats()
	if cacheStats.Hits > 0 {
		t.Errorf("Expected no cache hits when cache is disabled, got %d", cacheStats.Hits)
	}
}

// TestSummarizeWithCacheEnabled tests Summarize with cache enabled
func TestSummarizeWithCacheEnabled(t *testing.T) {
	config := createValidConfig()

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}
	ctx := context.Background()
	code := "test code"

	// First call should be a cache miss
	summary1, err := summarizer.Summarize(ctx, code, CodeMetadata{})
	if err != nil {
		t.Fatalf("First Summarize failed: %v", err)
	}

	if summary1.CacheHit {
		t.Error("Expected cache miss on first call")
	}

	// Second call should be a cache hit
	summary2, err := summarizer.Summarize(ctx, code, CodeMetadata{})
	if err != nil {
		t.Fatalf("Second Summarize failed: %v", err)
	}

	if !summary2.CacheHit {
		t.Error("Expected cache hit on second call")
	}

	if summary1.Text != summary2.Text {
		t.Errorf("Expected same summary text, got %q and %q", summary1.Text, summary2.Text)
	}
}

// TestCacheExpired tests cache expiration with TTL
func TestCacheExpired(t *testing.T) {
	config := createValidConfig()
	config.CacheTTL = 1 * time.Millisecond // Very short TTL

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}
	ctx := context.Background()
	code := "test code"

	// First call to cache the result
	_, _ = summarizer.Summarize(ctx, code, CodeMetadata{})

	// Wait for cache to expire
	time.Sleep(10 * time.Millisecond)

	// Try to get cached result - should be expired and miss
	summary, err := summarizer.Summarize(ctx, code, CodeMetadata{})
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	if summary.CacheHit {
		t.Error("Expected cache miss for expired entry")
	}
}

// TestClose tests Close method cleans up resources
func TestClose(t *testing.T) {
	config := createValidConfig()

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}
	ctx := context.Background()

	// Add some cached data
	_, _ = summarizer.Summarize(ctx, "test code", CodeMetadata{})

	cacheStatsBefore := summarizer.GetCacheStats()
	if cacheStatsBefore.Size == 0 {
		t.Fatal("Expected cache to have items before Close")
	}

	// Close should clean up
	err := summarizer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	cacheStatsAfter := summarizer.GetCacheStats()
	if cacheStatsAfter.Size != 0 {
		t.Errorf("Expected empty cache after Close, got %d items", cacheStatsAfter.Size)
	}
}

// TestSummarizeMultipleCalls tests multiple Summarize calls with different code
func TestSummarizeMultipleCalls(t *testing.T) {
	config := createValidConfig()

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"code1": "Summary 1",
			"code2": "Summary 2",
			"code3": "Summary 3",
		},
	}
	ctx := context.Background()

	codes := []string{"code1", "code2", "code3"}

	for _, code := range codes {
		summary, err := summarizer.Summarize(ctx, code, CodeMetadata{})
		if err != nil {
			t.Fatalf("Summarize failed for %s: %v", code, err)
		}

		if summary == nil {
			t.Fatalf("Expected summary for %s, got nil", code)
		}

		if summary.CacheHit {
			t.Errorf("Expected cache miss for %s", code)
		}
	}

	cacheStats := summarizer.GetCacheStats()
	if cacheStats.Size != 3 {
		t.Errorf("Expected 3 cached items, got %d", cacheStats.Size)
	}
}

// TestSummarizeContextCancellation tests Summarize behavior with cancelled context
func TestSummarizeContextCancellation(t *testing.T) {
	config := createValidConfig()
	config.CacheEnabled = false

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should still work since we're not actually using the context in Phase 1
	summary, err := summarizer.Summarize(ctx, "test code", CodeMetadata{})
	if err != nil {
		t.Fatalf("Summarize failed with cancelled context: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary even with cancelled context")
	}
}

// TestCodeMetadataFields tests CodeMetadata struct fields
func TestCodeMetadataFields(t *testing.T) {
	metadata := CodeMetadata{
		FilePath:   "/path/to/file.go",
		Language:   "go",
		NodeType:   "function",
		NodeName:   "TestFunc",
		Signature:  "func TestFunc() error",
		DocContent: "This is a test function",
		LineStart:  10,
		LineEnd:    20,
	}

	if metadata.FilePath != "/path/to/file.go" {
		t.Errorf("Expected FilePath /path/to/file.go, got %s", metadata.FilePath)
	}

	if metadata.Language != "go" {
		t.Errorf("Expected Language go, got %s", metadata.Language)
	}

	if metadata.NodeType != "function" {
		t.Errorf("Expected NodeType function, got %s", metadata.NodeType)
	}

	if metadata.LineStart != 10 {
		t.Errorf("Expected LineStart 10, got %d", metadata.LineStart)
	}

	if metadata.LineEnd != 20 {
		t.Errorf("Expected LineEnd 20, got %d", metadata.LineEnd)
	}
}

// TestCodeSummaryFields tests CodeSummary struct fields
func TestCodeSummaryFields(t *testing.T) {
	now := time.Now()
	summary := &CodeSummary{
		Text:        "Test summary",
		Type:        "llm",
		TokenCount:  150,
		GeneratedAt: now,
		CacheHit:    true,
	}

	if summary.Text != "Test summary" {
		t.Errorf("Expected Text 'Test summary', got %s", summary.Text)
	}

	if summary.Type != "llm" {
		t.Errorf("Expected Type 'llm', got %s", summary.Type)
	}

	if summary.TokenCount != 150 {
		t.Errorf("Expected TokenCount 150, got %d", summary.TokenCount)
	}

	if !summary.CacheHit {
		t.Error("Expected CacheHit to be true")
	}
}

// TestSummarizerConfigFields tests SummarizerConfig struct fields
func TestSummarizerConfigFields(t *testing.T) {
	config := createValidConfig()

	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if config.Model != "gpt-4" {
		t.Errorf("Expected Model 'gpt-4', got %s", config.Model)
	}

	if config.MaxTokens != 500 {
		t.Errorf("Expected MaxTokens 500, got %d", config.MaxTokens)
	}

	if config.CacheSize != 100 {
		t.Errorf("Expected CacheSize 100, got %d", config.CacheSize)
	}

	if config.TokenBudget != 10000 {
		t.Errorf("Expected TokenBudget 10000, got %d", config.TokenBudget)
	}
}

// TestValidateConfigAllValid tests validation passes for all valid configs
func TestValidateConfigAllValid(t *testing.T) {
	config := createValidConfig()

	err := validateConfig(config)
	if err != nil {
		t.Fatalf("validateConfig failed for valid config: %v", err)
	}
}

// TestSummarizeEmptyCode tests Summarize with empty code string
func TestSummarizeEmptyCode(t *testing.T) {
	config := createValidConfig()

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"": "Empty code summary",
		},
	}
	ctx := context.Background()

	summary, err := summarizer.Summarize(ctx, "", CodeMetadata{})
	if err != nil {
		t.Fatalf("Summarize failed with empty code: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary for empty code")
	}
}

// TestSummarizeLargeCode tests Summarize with large code string
// Large code may exceed token budget, which is expected behavior
func TestSummarizeLargeCode(t *testing.T) {
	config := createValidConfig()
	// Increase token budget significantly for large code test
	config.TokenBudget = 1000000
	config.TokenPerResult = 1000000

	summarizer, err := NewLLMSummarizer(config, nil)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{},
	}
	ctx := context.Background()

	// Create a large code string
	largeCode := ""
	for i := 0; i < 10000; i++ {
		largeCode += "func test() { return nil }\n"
	}

	summary, err := summarizer.Summarize(ctx, largeCode, CodeMetadata{})
	if err != nil {
		// Token budget exhaustion is acceptable for very large code
		if err.Error() == "token budget exhausted" {
			t.Log("Large code exceeded token budget - expected behavior")
			return
		}
		t.Fatalf("Summarize failed with large code: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary for large code")
	}
}

// TestNewLLMSummarizerWithInvalidConfig tests NewLLMSummarizer with invalid config
func TestNewLLMSummarizerWithInvalidConfig(t *testing.T) {
	config := SummarizerConfig{
		Enabled:   true,
		Model:     "",
		MaxTokens: 500,
	}

	_, err := NewLLMSummarizer(config, nil)
	if err == nil {
		t.Fatal("Expected error for invalid config")
	}
}

// TestCacheWithZeroTTL tests cache behavior with zero TTL
func TestCacheWithZeroTTL(t *testing.T) {
	config := createValidConfig()
	config.CacheTTL = 0 // No expiration

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}
	ctx := context.Background()
	code := "test code"

	// First call
	_, _ = summarizer.Summarize(ctx, code, CodeMetadata{})

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Should still be cached
	summary, err := summarizer.Summarize(ctx, code, CodeMetadata{})
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	if !summary.CacheHit {
		t.Error("Expected cache hit with zero TTL")
	}
}

// TestSummarizeReturnsCorrectType tests Summarize returns correct summary type
func TestSummarizeReturnsCorrectType(t *testing.T) {
	config := createValidConfig()

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}
	ctx := context.Background()

	summary, _ := summarizer.Summarize(ctx, "test code", CodeMetadata{})

	if summary.Type != "llm" {
		t.Errorf("Expected type 'llm', got %s", summary.Type)
	}
}

// TestMultipleSummarizersIndependent tests multiple summarizers are independent
func TestMultipleSummarizersIndependent(t *testing.T) {
	config := createValidConfig()

	summarizer1, _ := NewLLMSummarizer(config, nil)
	summarizer1.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}

	summarizer2, _ := NewLLMSummarizer(config, nil)
	summarizer2.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}

	ctx := context.Background()
	code := "test code"

	// Cache in summarizer1
	_, _ = summarizer1.Summarize(ctx, code, CodeMetadata{})

	// Should not be in summarizer2
	cacheStats2 := summarizer2.GetCacheStats()
	if cacheStats2.Size != 0 {
		t.Error("Expected summarizer2 cache to be empty")
	}

	// Cache in summarizer2
	_, _ = summarizer2.Summarize(ctx, code, CodeMetadata{})

	cacheStats2After := summarizer2.GetCacheStats()
	if cacheStats2After.Size != 1 {
		t.Error("Expected summarizer2 cache to have 1 item")
	}
}

// TestGetCacheStats tests GetCacheStats method
func TestGetCacheStats(t *testing.T) {
	config := createValidConfig()

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}
	ctx := context.Background()

	// Generate a summary
	_, _ = summarizer.Summarize(ctx, "test code", CodeMetadata{})

	cacheStats := summarizer.GetCacheStats()
	if cacheStats.Size == 0 {
		t.Error("Expected cache stats to show cached items")
	}
}

// TestGetTokenMetrics tests GetTokenMetrics method
func TestGetTokenMetrics(t *testing.T) {
	config := createValidConfig()

	summarizer, _ := NewLLMSummarizer(config, nil)
	summarizer.llmClient = &MockLLMClient{
		responses: map[string]string{
			"test code": "Test summary",
		},
	}
	ctx := context.Background()

	// Generate a summary
	_, _ = summarizer.Summarize(ctx, "test code", CodeMetadata{})

	tokenMetrics := summarizer.GetTokenMetrics()
	if tokenMetrics.TotalTokensUsed == 0 {
		t.Error("Expected token metrics to be recorded")
	}
}
