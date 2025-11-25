// Package summarizer provides code summarization with caching, token budget management, and metrics.
//
// The summarizer module is the main entry point for code summarization. It orchestrates
// caching, token budget management, and metrics collection while delegating actual
// summarization to an LLM client with graceful fallback to heuristic summarization.
//
// Key Features:
//   - LLM-based code summarization with configurable models
//   - Automatic caching with TTL and LRU eviction
//   - Per-user token budget enforcement
//   - Comprehensive metrics collection
//   - Graceful fallback to heuristic summarization
//   - Thread-safe operations
//
// Configuration:
//
//	config := SummarizerConfig{
//		Enabled:             true,
//		Model:               "gpt-4",
//		MaxTokens:           500,
//		CacheEnabled:        true,
//		CacheSize:           1000,
//		CacheTTL:            24 * time.Hour,
//		TokenBudget:         5000,
//		TokenPerResult:      100,
//	}
//	summarizer, err := NewLLMSummarizer(config, logger)
//
package summarizer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CodeSummary represents a generated summary of code
type CodeSummary struct {
	Text        string    `json:"text"`
	Type        string    `json:"type"` // "llm", "heuristic", "cached"
	TokenCount  int       `json:"tokenCount"`
	GeneratedAt time.Time `json:"generatedAt"`
	CacheHit    bool      `json:"cacheHit"`
}

// SummarizerConfig holds configuration for the code summarizer
type SummarizerConfig struct {
	Enabled             bool
	Model               string
	MaxTokens           int
	CacheEnabled        bool
	FallbackToHeuristic bool
	CacheSize           int
	CacheTTL            time.Duration
	LLMAPIKey           string
	LLMTimeout          time.Duration
	TokenBudget         int
	TokenPerResult      int
	MetricsEnabled      bool
	LogLevel            string
}

// CodeMetadata contains context about the code being summarized
type CodeMetadata struct {
	FilePath   string
	Language   string
	NodeType   string
	NodeName   string
	Signature  string
	DocContent string
	LineStart  int
	LineEnd    int
}

// CodeSummarizer is the interface for code summarization
type CodeSummarizer interface {
	// Summarize generates a summary for the given code
	Summarize(ctx context.Context, code string, metadata CodeMetadata) (*CodeSummary, error)
	// Close cleans up resources
	Close() error
}

// LLMSummarizer implements CodeSummarizer using an LLM
type LLMSummarizer struct {
	config         SummarizerConfig
	cache          SummaryCache
	tokenManager   *TokenManager
	metrics        *MetricsCollector
	llmClient      LLMClient
	logger         *zap.Logger
	mu             sync.RWMutex
}

// NewLLMSummarizer creates a new LLMSummarizer instance
func NewLLMSummarizer(config SummarizerConfig, logger *zap.Logger) (*LLMSummarizer, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid summarizer config: %w", err)
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	// Create LLM client
	llmClient, err := NewLLMClientFromConfig(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Create cache
	cache := NewLRUCache(config.CacheSize, config.CacheTTL, logger)

	// Create token manager
	tokenManager := NewTokenManager(config.TokenBudget, config.TokenPerResult, &SimpleTokenEstimator{}, logger)

	// Create metrics collector
	metricsCollector := NewMetricsCollector(logger)

	return &LLMSummarizer{
		config:         config,
		cache:          cache,
		tokenManager:   tokenManager,
		metrics:        metricsCollector,
		llmClient:      llmClient,
		logger:         logger,
	}, nil
}

// validateConfig validates the summarizer configuration
func validateConfig(config SummarizerConfig) error {
	if !config.Enabled {
		return nil // Disabled is valid
	}

	if config.Model == "" {
		return fmt.Errorf("model must be specified when summarizer is enabled")
	}

	if config.MaxTokens <= 0 {
		return fmt.Errorf("maxTokens must be greater than 0")
	}

	if config.MaxTokens > 1000 {
		return fmt.Errorf("maxTokens must not exceed 1000")
	}

	if config.CacheSize < 0 {
		return fmt.Errorf("cacheSize must be non-negative")
	}

	if config.CacheTTL < 0 {
		return fmt.Errorf("cacheTTL must be non-negative")
	}

	if config.LLMTimeout <= 0 {
		return fmt.Errorf("llmTimeout must be greater than 0")
	}

	if config.TokenBudget <= 0 {
		return fmt.Errorf("tokenBudget must be greater than 0")
	}

	if config.TokenPerResult <= 0 {
		return fmt.Errorf("tokenPerResult must be greater than 0")
	}

	return nil
}

// Summarize generates a summary for the given code
func (s *LLMSummarizer) Summarize(ctx context.Context, code string, metadata CodeMetadata) (*CodeSummary, error) {
	if !s.config.Enabled {
		return nil, fmt.Errorf("summarizer is disabled")
	}

	startTime := time.Now()
	cacheKey := GenerateCacheKey(code, metadata)

	// Check cache first if enabled
	if s.config.CacheEnabled {
		if cached, ok := s.cache.Get(cacheKey); ok {
			s.logger.Debug("Cache hit for code summary",
				zap.String("file", metadata.FilePath),
				zap.String("node_type", metadata.NodeType),
			)
			cached.CacheHit = true
			latencyMs := time.Since(startTime).Milliseconds()
			s.metrics.RecordSummarization("cached", latencyMs, cached.TokenCount, true)
			return cached, nil
		}
	}

	// Check token budget
	if !s.tokenManager.CanSummarize(code) {
		s.logger.Warn("Token budget exhausted, cannot summarize",
			zap.String("file", metadata.FilePath),
		)
		s.metrics.RecordError("token_budget_exhausted")
		return nil, fmt.Errorf("token budget exhausted")
	}

	// Call LLM client to generate summary
	summaryText, err := s.llmClient.Summarize(ctx, code, metadata)
	if err != nil {
		s.logger.Error("Failed to generate summary with LLM",
			zap.String("file", metadata.FilePath),
			zap.Error(err),
		)
		s.metrics.RecordError("llm_error")
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	summary := &CodeSummary{
		Text:        summaryText,
		Type:        "llm",
		TokenCount:  estimateTokenCount(summaryText),
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	// Record token usage
	s.tokenManager.RecordUsage(summary.TokenCount)

	// Cache the result if enabled
	if s.config.CacheEnabled {
		s.cache.Set(cacheKey, summary)
	}

	latencyMs := time.Since(startTime).Milliseconds()
	s.metrics.RecordSummarization("llm", latencyMs, summary.TokenCount, false)

	return summary, nil
}

// Close cleans up resources
func (s *LLMSummarizer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.llmClient != nil {
		s.llmClient.Close()
	}

	if s.cache != nil {
		s.cache.Clear()
	}

	// Log final metrics
	if s.metrics != nil {
		s.metrics.LogMetrics()
	}

	return nil
}

// estimateTokenCount provides a rough estimate of token count
// Typically 1 token ≈ 4 characters in English
func estimateTokenCount(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4
}

// GetMetrics returns the current metrics
func (s *LLMSummarizer) GetMetrics() SummarizationMetrics {
	return s.metrics.GetMetrics()
}

// GetCacheStats returns cache statistics
func (s *LLMSummarizer) GetCacheStats() CacheStats {
	return s.cache.Stats()
}

// GetTokenMetrics returns token usage metrics
func (s *LLMSummarizer) GetTokenMetrics() TokenMetrics {
	return s.tokenManager.GetMetrics()
}

// ResetMetrics resets all metrics
func (s *LLMSummarizer) ResetMetrics() {
	s.metrics.ResetMetrics()
	s.tokenManager.Reset()
}
