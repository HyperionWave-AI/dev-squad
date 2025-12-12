// Package summarizer provides code summarization with caching, token budget management, and metrics.
//
// The token_manager module manages token budget allocation and enforcement for code summarization.
// It tracks token usage across summarization requests and prevents exceeding configured budgets.
//
// Key Features:
//   - Per-user token budget enforcement
//   - Token estimation for code snippets
//   - Metrics tracking (total usage, blocked requests, block rate)
//   - Thread-safe operations with RWMutex
//   - Graceful degradation when budget is exhausted
//
// Usage:
//
//	estimator := &SimpleTokenEstimator{}
//	tm := NewTokenManager(5000, 100, estimator, logger)
//
//	if tm.CanSummarize(code) {
//		// Perform summarization
//		tm.RecordUsage(estimatedTokens)
//	} else {
//		// Skip summarization, budget exhausted
//	}
//
//	metrics := tm.GetMetrics()
package summarizer

import (
	"sync"

	"go.uber.org/zap"
)

// TokenManager manages token budget allocation and enforcement
type TokenManager struct {
	budget        int
	used          int
	perResult     int
	estimator     TokenEstimator
	mu            sync.RWMutex
	logger        *zap.Logger
	metrics       tokenMetrics
}

// TokenEstimator provides token count estimation for code
type TokenEstimator interface {
	// Estimate returns the estimated token count for the given code
	Estimate(code string) int
}

// SimpleTokenEstimator provides a basic token estimation
type SimpleTokenEstimator struct{}

// Estimate returns a simple token estimate (1 token ≈ 4 characters)
func (e *SimpleTokenEstimator) Estimate(code string) int {
	if len(code) == 0 {
		return 0
	}
	return (len(code) + 3) / 4
}

// tokenMetrics tracks token usage metrics
type tokenMetrics struct {
	totalTokensUsed    int64
	totalRequests      int64
	requestsBlocked    int64
	mu                 sync.RWMutex
}

// NewTokenManager creates a new token manager with the specified budget
func NewTokenManager(budget int, perResult int, estimator TokenEstimator, logger *zap.Logger) *TokenManager {
	if logger == nil {
		logger = zap.NewNop()
	}

	if estimator == nil {
		estimator = &SimpleTokenEstimator{}
	}

	if budget <= 0 {
		budget = 5000 // Default budget
	}

	if perResult <= 0 {
		perResult = 100 // Default per-result allocation
	}

	return &TokenManager{
		budget:    budget,
		used:      0,
		perResult: perResult,
		estimator: estimator,
		logger:    logger,
	}
}

// CanSummarize checks if there is sufficient token budget to summarize the code
// Returns true if the estimated tokens for the code are within the remaining budget
func (tm *TokenManager) CanSummarize(code string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	estimated := tm.estimator.Estimate(code)
	remaining := tm.budget - tm.used

	canSummarize := estimated <= remaining

	if !canSummarize {
		tm.metrics.recordBlocked()
		tm.logger.Warn("Token budget exhausted",
			zap.Int("estimated", estimated),
			zap.Int("remaining", remaining),
			zap.Int("budget", tm.budget),
			zap.Int("used", tm.used))
	}

	return canSummarize
}

// RecordUsage records the actual token usage for a summary
func (tm *TokenManager) RecordUsage(tokens int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.used += tokens
	tm.metrics.recordUsage(int64(tokens))

	tm.logger.Debug("Token usage recorded",
		zap.Int("tokens", tokens),
		zap.Int("totalUsed", tm.used),
		zap.Int("remaining", tm.budget-tm.used),
		zap.Int("budget", tm.budget))
}

// RemainingBudget returns the remaining token budget
func (tm *TokenManager) RemainingBudget() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.budget - tm.used
}

// UsedBudget returns the total tokens used so far
func (tm *TokenManager) UsedBudget() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.used
}

// TotalBudget returns the total token budget
func (tm *TokenManager) TotalBudget() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.budget
}

// Reset resets the token usage counter and metrics
func (tm *TokenManager) Reset() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.used = 0

	// Reset metrics
	tm.metrics.mu.Lock()
	tm.metrics.totalTokensUsed = 0
	tm.metrics.totalRequests = 0
	tm.metrics.requestsBlocked = 0
	tm.metrics.mu.Unlock()

	tm.logger.Debug("Token budget reset",
		zap.Int("budget", tm.budget))
}

// GetMetrics returns token usage metrics
func (tm *TokenManager) GetMetrics() TokenMetrics {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tm.metrics.mu.RLock()
	defer tm.metrics.mu.RUnlock()

	return TokenMetrics{
		TotalTokensUsed:  tm.metrics.totalTokensUsed,
		TotalRequests:    tm.metrics.totalRequests,
		RequestsBlocked:  tm.metrics.requestsBlocked,
		CurrentUsage:     int64(tm.used),
		RemainingBudget:  int64(tm.budget - tm.used),
		TotalBudget:      int64(tm.budget),
		BlockRate:        calculateBlockRate(tm.metrics.totalRequests, tm.metrics.requestsBlocked),
	}
}

// TokenMetrics contains token usage statistics
type TokenMetrics struct {
	TotalTokensUsed  int64
	TotalRequests    int64
	RequestsBlocked  int64
	CurrentUsage     int64
	RemainingBudget  int64
	TotalBudget      int64
	BlockRate        float64
}

// recordUsage records token usage in metrics
func (m *tokenMetrics) recordUsage(tokens int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalTokensUsed += tokens
	m.totalRequests++
}

// recordBlocked records a blocked request in metrics
func (m *tokenMetrics) recordBlocked() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestsBlocked++
	m.totalRequests++
}

// calculateBlockRate calculates the block rate as a percentage
func calculateBlockRate(total int64, blocked int64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(blocked) / float64(total) * 100.0
}

// EstimateTokensForCode estimates the token count for the given code
func EstimateTokensForCode(code string) int {
	estimator := &SimpleTokenEstimator{}
	return estimator.Estimate(code)
}

// EstimateTokensForSummary estimates the token count for a summary
// Typically summaries are 20-30% of the original code size
func EstimateTokensForSummary(code string) int {
	codeTokens := EstimateTokensForCode(code)
	// Assume summary is about 25% of original code size
	return (codeTokens + 3) / 4
}
