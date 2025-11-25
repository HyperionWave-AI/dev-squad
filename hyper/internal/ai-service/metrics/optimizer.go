package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/dev-squad/hyper/internal/storage"
)

// OptimizerMetrics provides optimization metrics analysis
type OptimizerMetrics struct {
	storage storage.MetricsStorage
}

// NewOptimizerMetrics creates a new optimizer metrics instance
func NewOptimizerMetrics(storage storage.MetricsStorage) *OptimizerMetrics {
	return &OptimizerMetrics{
		storage: storage,
	}
}

// EfficiencyMetrics represents efficiency metrics for a period
type EfficiencyMetrics struct {
	TokenEfficiency      float64 `json:"tokenEfficiency"`      // output/input ratio
	CostPerToken         float64 `json:"costPerToken"`         // average cost per token
	CostPerRequest       float64 `json:"costPerRequest"`       // average cost per request
	CacheHitRate         float64 `json:"cacheHitRate"`         // percentage of cache hits
	AverageRequestTime   float64 `json:"averageRequestTime"`   // milliseconds
	TokensPerRequest     float64 `json:"tokensPerRequest"`     // average tokens per request
	ModelDistribution    map[string]int `json:"modelDistribution"` // model usage count
	ProviderDistribution map[string]int `json:"providerDistribution"` // provider usage count
}

// GetEfficiencyMetrics calculates efficiency metrics for a time period
func (om *OptimizerMetrics) GetEfficiencyMetrics(ctx context.Context, userID string, startTime, endTime time.Time) (*EfficiencyMetrics, error) {
	// Get token metrics for the period
	tokenMetrics, err := om.storage.GetTokenMetrics(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get token metrics: %w", err)
	}

	if len(tokenMetrics) == 0 {
		return &EfficiencyMetrics{}, nil
	}

	// Get cost metrics for the period
	costMetrics, err := om.storage.GetCostMetrics(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost metrics: %w", err)
	}

	// Calculate metrics
	metrics := &EfficiencyMetrics{
		ModelDistribution:    make(map[string]int),
		ProviderDistribution: make(map[string]int),
	}

	var totalInputTokens, totalOutputTokens, totalCacheHitTokens int
	var totalCost float64
	var totalRequests int

	for _, tm := range tokenMetrics {
		totalInputTokens += tm.InputTokens
		totalOutputTokens += tm.OutputTokens
		totalCacheHitTokens += tm.CacheHitTokens
		totalRequests++

		// Track model distribution
		metrics.ModelDistribution[tm.Model]++
		metrics.ProviderDistribution[tm.Provider]++
	}

	// Calculate cost metrics
	for _, cm := range costMetrics {
		totalCost += cm.TotalCost
	}

	// Calculate efficiency metrics
	if totalInputTokens > 0 {
		metrics.TokenEfficiency = float64(totalOutputTokens) / float64(totalInputTokens)
	}

	if totalRequests > 0 {
		metrics.CostPerRequest = totalCost / float64(totalRequests)
		metrics.TokensPerRequest = float64(totalInputTokens+totalOutputTokens) / float64(totalRequests)
	}

	totalTokens := totalInputTokens + totalOutputTokens
	if totalTokens > 0 {
		metrics.CostPerToken = totalCost / float64(totalTokens)
		metrics.CacheHitRate = (float64(totalCacheHitTokens) / float64(totalTokens)) * 100
	}

	return metrics, nil
}

// OptimizationOpportunity represents a potential optimization
type OptimizationOpportunity struct {
	Type             string  `json:"type"`             // "prompt_compression", "model_downgrade", "caching", "batching"
	Description      string  `json:"description"`
	PotentialSavings float64 `json:"potentialSavings"` // in cost or tokens
	SavingsUnit      string  `json:"savingsUnit"`      // "cost" or "tokens"
	Difficulty       string  `json:"difficulty"`       // "easy", "medium", "hard"
	EstimatedEffort  string  `json:"estimatedEffort"`  // time estimate
}

// AnalyzeOptimizationOpportunities analyzes usage patterns to find optimization opportunities
func (om *OptimizerMetrics) AnalyzeOptimizationOpportunities(ctx context.Context, userID string, startTime, endTime time.Time) ([]OptimizationOpportunity, error) {
	metrics, err := om.GetEfficiencyMetrics(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	opportunities := []OptimizationOpportunity{}

	// Opportunity 1: Low token efficiency suggests verbose prompts
	if metrics.TokenEfficiency < 0.5 {
		opportunities = append(opportunities, OptimizationOpportunity{
			Type:             "prompt_compression",
			Description:      "Your prompts are generating less output than input. Consider optimizing prompt structure to be more concise.",
			PotentialSavings: metrics.CostPerRequest * 0.2, // estimate 20% savings
			SavingsUnit:      "cost",
			Difficulty:       "medium",
			EstimatedEffort:  "2-4 hours",
		})
	}

	// Opportunity 2: Low cache hit rate suggests missing caching opportunities
	if metrics.CacheHitRate < 10 {
		opportunities = append(opportunities, OptimizationOpportunity{
			Type:             "caching",
			Description:      "You have very few cache hits. Implement caching for repeated queries to reduce costs.",
			PotentialSavings: metrics.CostPerRequest * 0.15, // estimate 15% savings
			SavingsUnit:      "cost",
			Difficulty:       "easy",
			EstimatedEffort:  "1-2 hours",
		})
	}

	// Opportunity 3: High cost per request suggests model optimization
	if metrics.CostPerRequest > 0.10 {
		opportunities = append(opportunities, OptimizationOpportunity{
			Type:             "model_downgrade",
			Description:      "Consider using a more cost-effective model for simpler tasks.",
			PotentialSavings: metrics.CostPerRequest * 0.3, // estimate 30% savings
			SavingsUnit:      "cost",
			Difficulty:       "hard",
			EstimatedEffort:  "4-8 hours",
		})
	}

	// Opportunity 4: High token usage suggests batching opportunities
	if metrics.TokensPerRequest > 5000 {
		opportunities = append(opportunities, OptimizationOpportunity{
			Type:             "batching",
			Description:      "Your requests are large. Consider batching multiple operations to reduce overhead.",
			PotentialSavings: metrics.CostPerRequest * 0.1, // estimate 10% savings
			SavingsUnit:      "cost",
			Difficulty:       "medium",
			EstimatedEffort:  "3-5 hours",
		})
	}

	return opportunities, nil
}

// CostTrendAnalysis represents cost trend information
type CostTrendAnalysis struct {
	CurrentPeriodCost  float64 `json:"currentPeriodCost"`
	PreviousPeriodCost float64 `json:"previousPeriodCost"`
	PercentageChange   float64 `json:"percentageChange"`
	Trend              string  `json:"trend"` // "increasing", "stable", "decreasing"
	ProjectedMonthly   float64 `json:"projectedMonthly"`
}

// AnalyzeCostTrend analyzes cost trends over time
func (om *OptimizerMetrics) AnalyzeCostTrend(ctx context.Context, userID string, currentStart, currentEnd, previousStart, previousEnd time.Time) (*CostTrendAnalysis, error) {
	// Get current period costs
	currentCosts, err := om.storage.GetCostMetrics(ctx, userID, currentStart, currentEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get current period costs: %w", err)
	}

	// Get previous period costs
	previousCosts, err := om.storage.GetCostMetrics(ctx, userID, previousStart, previousEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous period costs: %w", err)
	}

	analysis := &CostTrendAnalysis{}

	// Sum current period costs
	for _, cm := range currentCosts {
		analysis.CurrentPeriodCost += cm.TotalCost
	}

	// Sum previous period costs
	for _, cm := range previousCosts {
		analysis.PreviousPeriodCost += cm.TotalCost
	}

	// Calculate percentage change
	if analysis.PreviousPeriodCost > 0 {
		analysis.PercentageChange = ((analysis.CurrentPeriodCost - analysis.PreviousPeriodCost) / analysis.PreviousPeriodCost) * 100
	}

	// Determine trend
	if analysis.PercentageChange > 5 {
		analysis.Trend = "increasing"
	} else if analysis.PercentageChange < -5 {
		analysis.Trend = "decreasing"
	} else {
		analysis.Trend = "stable"
	}

	// Project monthly cost (assuming 30-day month)
	daysDiff := currentEnd.Sub(currentStart).Hours() / 24
	if daysDiff > 0 {
		analysis.ProjectedMonthly = (analysis.CurrentPeriodCost / daysDiff) * 30
	}

	return analysis, nil
}
