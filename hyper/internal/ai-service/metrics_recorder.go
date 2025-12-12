package aiservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"hyper/internal/mcp/storage"
)

// MetricsRecorder handles recording metrics to MongoDB
type MetricsRecorder struct {
	metricsStorage storage.MetricsStorage
	logger         *zap.Logger
}

// NewMetricsRecorder creates a new metrics recorder
func NewMetricsRecorder(metricsStorage storage.MetricsStorage, logger *zap.Logger) *MetricsRecorder {
	return &MetricsRecorder{
		metricsStorage: metricsStorage,
		logger:         logger,
	}
}

// RecordProviderMetrics records metrics from a provider response
func (r *MetricsRecorder) RecordProviderMetrics(ctx context.Context, provider string, model string, tokenUsage *TokenUsage, duration int64, requestID string) error {
	if r.metricsStorage == nil || tokenUsage == nil {
		return nil // Silently skip if storage not configured
	}

	// Calculate cost using pricing table
	inputCost, outputCost, totalCost := storage.CalculateCost(
		provider,
		model,
		tokenUsage.PromptTokens,
		tokenUsage.CompletionTokens,
		0, // cacheCreationTokens
		0, // cacheReadTokens
	)

	// Create metrics record
	metrics := &storage.TokenMetrics{
		ID:           uuid.New().String(),
		RequestID:    requestID,
		Provider:     provider,
		Model:        model,
		InputTokens:  tokenUsage.PromptTokens,
		OutputTokens: tokenUsage.CompletionTokens,
		TotalTokens:  tokenUsage.TotalTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    totalCost,
		Duration:     duration,
		Status:       "success",
	}

	// Record to MongoDB
	err := r.metricsStorage.RecordMetrics(ctx, metrics)
	if err != nil {
		r.logger.Warn("failed to record provider metrics",
			zap.String("provider", provider),
			zap.String("model", model),
			zap.Error(err))
		return err
	}

	r.logger.Debug("recorded provider metrics",
		zap.String("provider", provider),
		zap.String("model", model),
		zap.Int("inputTokens", tokenUsage.PromptTokens),
		zap.Int("outputTokens", tokenUsage.CompletionTokens),
		zap.Float64("totalCost", totalCost))

	return nil
}

// RecordToolMetrics records metrics for a tool execution
func (r *MetricsRecorder) RecordToolMetrics(ctx context.Context, toolName string, cacheHit bool, duration int64, requestID string) error {
	if r.metricsStorage == nil {
		return nil // Silently skip if storage not configured
	}

	// Create metrics record for tool execution
	cacheHits := 0
	cacheMisses := 1
	if cacheHit {
		cacheHits = 1
		cacheMisses = 0
	}

	metrics := &storage.TokenMetrics{
		ID:        uuid.New().String(),
		RequestID: requestID,
		ToolName:  toolName,
		CacheHits: cacheHits,
		CacheMisses: cacheMisses,
		Duration:  duration,
		Status:    "success",
	}

	// Record to MongoDB
	err := r.metricsStorage.RecordMetrics(ctx, metrics)
	if err != nil {
		r.logger.Warn("failed to record tool metrics",
			zap.String("toolName", toolName),
			zap.Bool("cacheHit", cacheHit),
			zap.Error(err))
		return err
	}

	r.logger.Debug("recorded tool metrics",
		zap.String("toolName", toolName),
		zap.Bool("cacheHit", cacheHit),
		zap.Int64("duration", duration))

	return nil
}

// RecordErrorMetrics records metrics for a failed operation
func (r *MetricsRecorder) RecordErrorMetrics(ctx context.Context, provider string, model string, errorMsg string, duration int64, requestID string) error {
	if r.metricsStorage == nil {
		return nil // Silently skip if storage not configured
	}

	metrics := &storage.TokenMetrics{
		ID:           uuid.New().String(),
		RequestID:    requestID,
		Provider:     provider,
		Model:        model,
		Duration:     duration,
		Status:       "error",
		ErrorMessage: errorMsg,
	}

	err := r.metricsStorage.RecordMetrics(ctx, metrics)
	if err != nil {
		r.logger.Warn("failed to record error metrics",
			zap.String("provider", provider),
			zap.String("model", model),
			zap.Error(err))
		return err
	}

	r.logger.Debug("recorded error metrics",
		zap.String("provider", provider),
		zap.String("model", model),
		zap.String("error", errorMsg))

	return nil
}

// GetCostAnalysis returns cost analysis for a time period
func (r *MetricsRecorder) GetCostAnalysis(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]interface{}, error) {
	if r.metricsStorage == nil {
		return nil, fmt.Errorf("metrics storage not configured")
	}

	// Get total cost
	totalCost, err := r.metricsStorage.GetTotalCost(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Get cost breakdown by model
	breakdown, err := r.metricsStorage.GetCostByModel(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Get cache hit rates
	cacheStats, err := r.metricsStorage.GetCacheHitRateStats(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalCost":     totalCost,
		"breakdown":     breakdown,
		"cacheHitRates": cacheStats,
		"periodStart":   startTime,
		"periodEnd":     endTime,
	}, nil
}

// GetMetricsStats returns statistics for a provider/model
func (r *MetricsRecorder) GetMetricsStats(ctx context.Context, provider string, model string, startTime time.Time, endTime time.Time) (*storage.MetricsStats, error) {
	if r.metricsStorage == nil {
		return nil, fmt.Errorf("metrics storage not configured")
	}

	return r.metricsStorage.GetMetricsStats(ctx, provider, model, startTime, endTime)
}
