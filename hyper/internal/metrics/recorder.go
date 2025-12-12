package metrics

import (
	"fmt"
	"time"
)

// NewMetricsRecorder creates a new metrics recorder with pricing tiers
func NewMetricsRecorder(store MetricsStore) *MetricsRecorder {
	recorder := &MetricsRecorder{
		store:        store,
		pricingTiers: make(map[string]map[string]PricingTier),
	}

	// Initialize pricing tiers for all supported providers
	recorder.initializePricingTiers()

	return recorder
}

// initializePricingTiers sets up pricing for all supported models
func (r *MetricsRecorder) initializePricingTiers() {
	// OpenAI Pricing (as of 2024)
	r.pricingTiers["openai"] = map[string]PricingTier{
		"gpt-4": {
			PromptTokenPrice:     0.03,    // $0.03 per 1K input tokens
			CompletionTokenPrice: 0.06,   // $0.06 per 1K output tokens
		},
		"gpt-4-turbo": {
			PromptTokenPrice:     0.01,   // $0.01 per 1K input tokens
			CompletionTokenPrice: 0.03,  // $0.03 per 1K output tokens
		},
		"gpt-3.5-turbo": {
			PromptTokenPrice:     0.0005, // $0.0005 per 1K input tokens
			CompletionTokenPrice: 0.0015, // $0.0015 per 1K output tokens
		},
		"gpt-4o": {
			PromptTokenPrice:     0.005,  // $0.005 per 1K input tokens
			CompletionTokenPrice: 0.015,  // $0.015 per 1K output tokens
		},
	}

	// Anthropic Pricing (as of 2024)
	r.pricingTiers["anthropic"] = map[string]PricingTier{
		"claude-3-opus": {
			PromptTokenPrice:     0.015,   // $0.015 per 1K input tokens
			CompletionTokenPrice: 0.075,  // $0.075 per 1K output tokens
		},
		"claude-3-sonnet": {
			PromptTokenPrice:     0.003,   // $0.003 per 1K input tokens
			CompletionTokenPrice: 0.015,  // $0.015 per 1K output tokens
		},
		"claude-3-haiku": {
			PromptTokenPrice:     0.00025, // $0.00025 per 1K input tokens
			CompletionTokenPrice: 0.00125, // $0.00125 per 1K output tokens
		},
		"claude-3.5-sonnet": {
			PromptTokenPrice:     0.003,   // $0.003 per 1K input tokens
			CompletionTokenPrice: 0.015,  // $0.015 per 1K output tokens
		},
	}

	// Groq Pricing (typically free or very cheap)
	r.pricingTiers["groq"] = map[string]PricingTier{
		"mixtral-8x7b-32768": {
			PromptTokenPrice:     0.0,    // Free
			CompletionTokenPrice: 0.0,   // Free
		},
		"llama2-70b-4096": {
			PromptTokenPrice:     0.0,    // Free
			CompletionTokenPrice: 0.0,   // Free
		},
	}
}

// RecordProviderMetrics records metrics for an AI provider API call
func (r *MetricsRecorder) RecordProviderMetrics(provider, model string, promptTokens, completionTokens int, durationMs int64, requestID, userID, sessionID string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Calculate costs
	promptCost := r.calculateTokenCost(provider, model, promptTokens, true)
	completionCost := r.calculateTokenCost(provider, model, completionTokens, false)
	totalCost := promptCost + completionCost

	metrics := &ProviderMetrics{
		Provider:         provider,
		Model:            model,
		Timestamp:        time.Now(),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		DurationMs:       durationMs,
		PromptCost:       promptCost,
		CompletionCost:   completionCost,
		TotalCost:        totalCost,
		RequestID:        requestID,
		UserID:           userID,
		SessionID:        sessionID,
	}

	return r.store.StoreProviderMetrics(metrics)
}

// calculateTokenCost calculates the cost for a given number of tokens
func (r *MetricsRecorder) calculateTokenCost(provider, model string, tokens int, isPrompt bool) float64 {
	if tokens == 0 {
		return 0.0
	}

	// Get pricing tier for provider
	providerTiers, exists := r.pricingTiers[provider]
	if !exists {
		// Unknown provider - return 0 cost
		return 0.0
	}

	// Get pricing tier for model
	tier, exists := providerTiers[model]
	if !exists {
		// Unknown model - return 0 cost
		return 0.0
	}

	// Calculate cost based on token type
	var pricePerThousand float64
	if isPrompt {
		pricePerThousand = tier.PromptTokenPrice
	} else {
		pricePerThousand = tier.CompletionTokenPrice
	}

	// Cost = (tokens / 1000) * price_per_1000
	return (float64(tokens) / 1000.0) * pricePerThousand
}

// GetProviderCostBreakdown returns cost breakdown for a date range
func (r *MetricsRecorder) GetProviderCostBreakdown(startDate, endDate time.Time) (*CostBreakdown, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.store.GetCostBreakdown(startDate, endDate)
}

// GetDailyCosts returns daily cost breakdown
func (r *MetricsRecorder) GetDailyCosts(startDate, endDate time.Time) (map[string]float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.store.GetDailyCosts(startDate, endDate)
}

// GetWeeklyCosts returns weekly cost breakdown
func (r *MetricsRecorder) GetWeeklyCosts(startDate, endDate time.Time) (map[string]float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.store.GetWeeklyCosts(startDate, endDate)
}

// GetMonthlyCosts returns monthly cost breakdown
func (r *MetricsRecorder) GetMonthlyCosts(startDate, endDate time.Time) (map[string]float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.store.GetMonthlyCosts(startDate, endDate)
}

// GetModelEfficiency returns efficiency metrics for each model
func (r *MetricsRecorder) GetModelEfficiency(startDate, endDate time.Time) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics, err := r.store.GetProviderMetricsByDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Group by model and calculate efficiency
	modelStats := make(map[string]map[string]interface{})

	for _, m := range metrics {
		if _, exists := modelStats[m.Model]; !exists {
			modelStats[m.Model] = map[string]interface{}{
				"calls":              0,
				"total_cost":         0.0,
				"total_tokens":       0,
				"avg_cost_per_call":  0.0,
				"avg_tokens_per_call": 0,
				"provider":           m.Provider,
			}
		}

		stats := modelStats[m.Model]
		stats["calls"] = stats["calls"].(int) + 1
		stats["total_cost"] = stats["total_cost"].(float64) + m.TotalCost
		stats["total_tokens"] = stats["total_tokens"].(int) + m.TotalTokens
	}

	// Calculate averages
	for model, stats := range modelStats {
		calls := stats["calls"].(int)
		if calls > 0 {
			stats["avg_cost_per_call"] = stats["total_cost"].(float64) / float64(calls)
			stats["avg_tokens_per_call"] = stats["total_tokens"].(int) / calls
		}
		modelStats[model] = stats
	}

	return map[string]interface{}{
		"models": modelStats,
	}, nil
}

// GetProviderComparison returns comparison metrics for all providers
func (r *MetricsRecorder) GetProviderComparison(startDate, endDate time.Time) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics, err := r.store.GetProviderMetricsByDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Group by provider and calculate stats
	providerStats := make(map[string]map[string]interface{})

	for _, m := range metrics {
		if _, exists := providerStats[m.Provider]; !exists {
			providerStats[m.Provider] = map[string]interface{}{
				"calls":              0,
				"total_cost":         0.0,
				"total_tokens":       0,
				"avg_cost_per_call":  0.0,
				"avg_tokens_per_call": 0,
				"models":             make(map[string]int),
			}
		}

		stats := providerStats[m.Provider]
		stats["calls"] = stats["calls"].(int) + 1
		stats["total_cost"] = stats["total_cost"].(float64) + m.TotalCost
		stats["total_tokens"] = stats["total_tokens"].(int) + m.TotalTokens

		// Track models used
		models := stats["models"].(map[string]int)
		models[m.Model]++
	}

	// Calculate averages
	for provider, stats := range providerStats {
		calls := stats["calls"].(int)
		if calls > 0 {
			stats["avg_cost_per_call"] = stats["total_cost"].(float64) / float64(calls)
			stats["avg_tokens_per_call"] = stats["total_tokens"].(int) / calls
		}
		providerStats[provider] = stats
	}

	return map[string]interface{}{
		"providers": providerStats,
	}, nil
}

// GenerateRecommendations generates cost optimization recommendations
func (r *MetricsRecorder) GenerateRecommendations(startDate, endDate time.Time) (*Recommendations, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	breakdown, err := r.store.GetCostBreakdown(startDate, endDate)
	if err != nil {
		return nil, err
	}

	recommendations := &Recommendations{
		Items:       make([]RecommendationItem, 0),
		GeneratedAt: time.Now(),
	}

	// Recommendation 1: High-cost models
	if breakdown.TotalCost > 0 {
		for model, cost := range breakdown.ModelCosts {
			percentOfTotal := (cost / breakdown.TotalCost) * 100
			if percentOfTotal > 30 {
				potential := cost * 0.2 // Assume 20% savings possible
				recommendations.Items = append(recommendations.Items, RecommendationItem{
					Title:       fmt.Sprintf("Optimize %s usage", model),
					Description: fmt.Sprintf("%s accounts for %.1f%% of total costs. Consider using a cheaper model for non-critical tasks.", model, percentOfTotal),
					Potential:   potential,
					Priority:    "high",
					Action:      fmt.Sprintf("Evaluate using a cheaper alternative to %s for some requests", model),
				})
				recommendations.TotalPotential += potential
			}
		}
	}

	// Recommendation 2: Cache optimization
	cacheStats, err := r.store.GetCacheStats(startDate, endDate)
	if err == nil {
		if hitRate, ok := cacheStats["hit_rate"].(float64); ok && hitRate < 20 {
			potential := breakdown.TotalCost * 0.15 // Assume 15% savings with better caching
			recommendations.Items = append(recommendations.Items, RecommendationItem{
				Title:       "Improve cache hit rate",
				Description: fmt.Sprintf("Current cache hit rate is only %.1f%%. Improving caching strategies could save significant costs.", hitRate),
				Potential:   potential,
				Priority:    "high",
				Action:      "Review and optimize caching strategies for frequently used queries",
			})
			recommendations.TotalPotential += potential
		}
	}

	// Recommendation 3: Token optimization
	if breakdown.AveragePromptTokens > 5000 {
		potential := breakdown.TotalCost * 0.1 // Assume 10% savings with better prompting
		recommendations.Items = append(recommendations.Items, RecommendationItem{
			Title:       "Reduce prompt size",
			Description: fmt.Sprintf("Average prompt size is %.0f tokens. Optimizing prompts could reduce token usage.", float64(breakdown.AveragePromptTokens)),
			Potential:   potential,
			Priority:    "medium",
			Action:      "Review and optimize system prompts and context to reduce token consumption",
		})
		recommendations.TotalPotential += potential
	}

	return recommendations, nil
}

// CleanupOldMetrics removes metrics older than specified days
func (r *MetricsRecorder) CleanupOldMetrics(days int) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.store.DeleteMetricsOlderThan(days)
}

// NewToolMetricsTracker creates a new tool metrics tracker
func NewToolMetricsTracker(store MetricsStore) *ToolMetricsTracker {
	return &ToolMetricsTracker{
		store: store,
	}
}

// RecordToolExecution records metrics for a tool execution
func (t *ToolMetricsTracker) RecordToolExecution(toolName string, durationMs int64, status string, cacheHit bool, requestID, userID, sessionID string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	metrics := &ToolMetrics{
		ToolName:   toolName,
		Timestamp:  time.Now(),
		DurationMs: durationMs,
		Status:     status,
		CacheHit:   cacheHit,
		RequestID:  requestID,
		UserID:     userID,
		SessionID:  sessionID,
	}

	return t.store.StoreToolMetrics(metrics)
}

// RecordToolExecutionWithError records tool execution with error details
func (t *ToolMetricsTracker) RecordToolExecutionWithError(toolName string, durationMs int64, status, errorMsg, requestID, userID, sessionID string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	metrics := &ToolMetrics{
		ToolName:   toolName,
		Timestamp:  time.Now(),
		DurationMs: durationMs,
		Status:     status,
		Error:      errorMsg,
		RequestID:  requestID,
		UserID:     userID,
		SessionID:  sessionID,
	}

	return t.store.StoreToolMetrics(metrics)
}

// GetCacheStats returns cache performance statistics
func (t *ToolMetricsTracker) GetCacheStats(startDate, endDate time.Time) (*CacheStats, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	metrics, err := t.store.GetToolMetricsByDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{
		TotalRequests: int64(len(metrics)),
		CacheHits:     0,
		CacheMisses:   0,
		TokensSaved:   0,
		CostSaved:     0.0,
		AverageHitTime: 0,
	}

	var totalHitTime int64

	for _, m := range metrics {
		if m.CacheHit {
			stats.CacheHits++
			totalHitTime += m.DurationMs
		} else {
			stats.CacheMisses++
		}
	}

	if stats.TotalRequests > 0 {
		stats.HitRate = (float64(stats.CacheHits) / float64(stats.TotalRequests)) * 100
	}

	if stats.CacheHits > 0 {
		stats.AverageHitTime = totalHitTime / stats.CacheHits
	}

	return stats, nil
}

// GetToolStats returns execution statistics for a specific tool
func (t *ToolMetricsTracker) GetToolStats(toolName string, startDate, endDate time.Time) (map[string]interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	metrics, err := t.store.GetToolMetricsByName(toolName, startDate, endDate)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"tool_name":      toolName,
		"total_calls":    len(metrics),
		"successful":     0,
		"failed":         0,
		"cached":         0,
		"avg_duration":   0.0,
		"min_duration":   int64(0),
		"max_duration":   int64(0),
		"error_rate":     0.0,
		"cache_hit_rate": 0.0,
	}

	if len(metrics) == 0 {
		return stats, nil
	}

	var totalDuration int64
	minDuration := int64(^uint64(0) >> 1) // Max int64
	maxDuration := int64(0)
	successful := 0
	failed := 0
	cached := 0

	for _, m := range metrics {
		totalDuration += m.DurationMs

		if m.DurationMs < minDuration {
			minDuration = m.DurationMs
		}
		if m.DurationMs > maxDuration {
			maxDuration = m.DurationMs
		}

		if m.Status == "success" {
			successful++
		} else if m.Status == "error" {
			failed++
		}

		if m.CacheHit {
			cached++
		}
	}

	stats["successful"] = successful
	stats["failed"] = failed
	stats["cached"] = cached
	stats["avg_duration"] = float64(totalDuration) / float64(len(metrics))
	stats["min_duration"] = minDuration
	stats["max_duration"] = maxDuration
	stats["error_rate"] = (float64(failed) / float64(len(metrics))) * 100
	stats["cache_hit_rate"] = (float64(cached) / float64(len(metrics))) * 100

	return stats, nil
}
