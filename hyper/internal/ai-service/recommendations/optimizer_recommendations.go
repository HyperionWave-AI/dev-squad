package recommendations

import (
	"context"
	"fmt"
	"time"

	"hyper/internal/storage"
)

// RecommendationEngine generates optimization recommendations
type RecommendationEngine struct {
	storage storage.MetricsStorage
}

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine(storage storage.MetricsStorage) *RecommendationEngine {
	return &RecommendationEngine{
		storage: storage,
	}
}

// GenerateRecommendations generates recommendations based on usage patterns
func (re *RecommendationEngine) GenerateRecommendations(ctx context.Context, userID, companyID string, startTime, endTime time.Time) ([]storage.Recommendation, error) {
	recommendations := []storage.Recommendation{}

	// Get optimization metrics
	optMetrics, err := re.storage.GetOptimizationMetrics(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get optimization metrics: %w", err)
	}

	if optMetrics == nil {
		return recommendations, nil
	}

	// Generate recommendations based on metrics
	recommendations = append(recommendations, re.generatePromptOptimizationRecs(optMetrics, userID, companyID)...)
	recommendations = append(recommendations, re.generateModelSelectionRecs(optMetrics, userID, companyID)...)
	recommendations = append(recommendations, re.generateCachingRecs(optMetrics, userID, companyID)...)
	recommendations = append(recommendations, re.generateBatchingRecs(optMetrics, userID, companyID)...)

	// Save recommendations
	for i := range recommendations {
		if err := re.storage.SaveRecommendation(ctx, &recommendations[i]); err != nil {
			return nil, fmt.Errorf("failed to save recommendation: %w", err)
		}
	}

	return recommendations, nil
}

// generatePromptOptimizationRecs generates prompt optimization recommendations
func (re *RecommendationEngine) generatePromptOptimizationRecs(metrics *storage.OptimizationMetrics, userID, companyID string) []storage.Recommendation {
	recommendations := []storage.Recommendation{}

	// If token efficiency is low, suggest prompt optimization
	if metrics.AverageTokensPerRequest > 3000 {
		rec := storage.Recommendation{
			UserID:            userID,
			CompanyID:         companyID,
			Category:          "prompt_optimization",
			Title:             "Optimize Prompt Structure",
			Description:       "Your average request uses a high number of tokens. Consider optimizing your prompts to be more concise and focused.",
			PotentialSavings: metrics.TotalCost * 0.15, // 15% potential savings
			SavingsUnit:      "cost",
			Priority:         "high",
			ActionItems: []string{
				"Review your prompt templates for verbosity",
				"Remove unnecessary context or examples",
				"Use structured formats (JSON, XML) instead of prose",
				"Implement prompt caching for repeated patterns",
			},
			IsImplemented: false,
			CreatedAt:     time.Now(),
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// generateModelSelectionRecs generates model selection recommendations
func (re *RecommendationEngine) generateModelSelectionRecs(metrics *storage.OptimizationMetrics, userID, companyID string) []storage.Recommendation {
	recommendations := []storage.Recommendation{}

	// If average cost per request is high, suggest model downgrade
	if metrics.AverageCostPerRequest > 0.05 {
		rec := storage.Recommendation{
			UserID:            userID,
			CompanyID:         companyID,
			Category:          "model_selection",
			Title:             "Consider Using a More Cost-Effective Model",
			Description:       fmt.Sprintf("Your average cost per request is $%.4f. Consider using a more cost-effective model for simpler tasks.", metrics.AverageCostPerRequest),
			PotentialSavings: metrics.TotalCost * 0.30, // 30% potential savings
			SavingsUnit:      "cost",
			Priority:         "high",
			ActionItems: []string{
				"Analyze which tasks require advanced models",
				"Identify tasks that could use cheaper models",
				"Implement model routing based on task complexity",
				"Test cheaper models on non-critical tasks first",
			},
			IsImplemented: false,
			CreatedAt:     time.Now(),
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// generateCachingRecs generates caching recommendations
func (re *RecommendationEngine) generateCachingRecs(metrics *storage.OptimizationMetrics, userID, companyID string) []storage.Recommendation {
	recommendations := []storage.Recommendation{}

	// If cache hit rate is low, suggest implementing caching
	if metrics.CacheHitRate < 20 {
		rec := storage.Recommendation{
			UserID:            userID,
			CompanyID:         companyID,
			Category:          "caching",
			Title:             "Implement Prompt Caching",
			Description:       fmt.Sprintf("Your cache hit rate is only %.1f%%. Implementing caching for repeated queries could significantly reduce costs.", metrics.CacheHitRate),
			PotentialSavings: metrics.TotalCost * 0.20, // 20% potential savings
			SavingsUnit:      "cost",
			Priority:         "medium",
			ActionItems: []string{
				"Identify frequently repeated queries",
				"Implement caching layer for common requests",
				"Use semantic caching for similar queries",
				"Monitor cache hit rates after implementation",
			},
			IsImplemented: false,
			CreatedAt:     time.Now(),
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// generateBatchingRecs generates batching recommendations
func (re *RecommendationEngine) generateBatchingRecs(metrics *storage.OptimizationMetrics, userID, companyID string) []storage.Recommendation {
	recommendations := []storage.Recommendation{}

	// If there are many requests, suggest batching
	if metrics.TotalRequests > 100 {
		rec := storage.Recommendation{
			UserID:            userID,
			CompanyID:         companyID,
			Category:          "batching",
			Title:             "Batch Similar Requests",
			Description:       fmt.Sprintf("You have %d requests in this period. Batching similar requests could reduce overhead and improve efficiency.", metrics.TotalRequests),
			PotentialSavings: metrics.TotalCost * 0.10, // 10% potential savings
			SavingsUnit:      "cost",
			Priority:         "medium",
			ActionItems: []string{
				"Identify patterns in your requests",
				"Group similar requests for batch processing",
				"Implement batch API endpoints",
				"Monitor performance improvements",
			},
			IsImplemented: false,
			CreatedAt:     time.Now(),
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// GetRecommendationImpact calculates the impact of implementing a recommendation
type RecommendationImpact struct {
	EstimatedSavings float64 `json:"estimatedSavings"`
	SavingsUnit      string  `json:"savingsUnit"`
	ImplementationCost float64 `json:"implementationCost"` // in hours
	ROI              float64 `json:"roi"`                 // return on investment
	TimeToBreakEven  float64 `json:"timeToBreakEven"`     // in days
}

// CalculateImpact calculates the impact of implementing a recommendation
func (re *RecommendationEngine) CalculateImpact(rec *storage.Recommendation, monthlySpend float64) *RecommendationImpact {
	impact := &RecommendationImpact{
		EstimatedSavings: rec.PotentialSavings,
		SavingsUnit:      rec.SavingsUnit,
	}

	// Estimate implementation cost based on difficulty
	switch rec.Category {
	case "prompt_optimization":
		impact.ImplementationCost = 4 // hours
	case "model_selection":
		impact.ImplementationCost = 8 // hours
	case "caching":
		impact.ImplementationCost = 6 // hours
	case "batching":
		impact.ImplementationCost = 8 // hours
	default:
		impact.ImplementationCost = 4 // hours
	}

	// Calculate ROI (assuming $50/hour developer cost)
	developerCost := impact.ImplementationCost * 50
	if rec.SavingsUnit == "cost" {
		// Monthly savings
		monthlySavings := rec.PotentialSavings
		if monthlySavings > 0 {
			impact.ROI = (monthlySavings / developerCost) * 100
			// Time to break even in days
			dailySavings := monthlySavings / 30
			if dailySavings > 0 {
				impact.TimeToBreakEven = developerCost / dailySavings
			}
		}
	}

	return impact
}

// PrioritizeRecommendations sorts recommendations by impact and priority
func (re *RecommendationEngine) PrioritizeRecommendations(recommendations []storage.Recommendation, monthlySpend float64) []storage.Recommendation {
	// Calculate impact for each recommendation
	type recWithImpact struct {
		rec    storage.Recommendation
		impact *RecommendationImpact
	}

	recWithImpacts := make([]recWithImpact, len(recommendations))
	for i, rec := range recommendations {
		recWithImpacts[i] = recWithImpact{
			rec:    rec,
			impact: re.CalculateImpact(&rec, monthlySpend),
		}
	}

	// Sort by ROI (highest first)
	for i := 0; i < len(recWithImpacts); i++ {
		for j := i + 1; j < len(recWithImpacts); j++ {
			if recWithImpacts[j].impact.ROI > recWithImpacts[i].impact.ROI {
				recWithImpacts[i], recWithImpacts[j] = recWithImpacts[j], recWithImpacts[i]
			}
		}
	}

	// Extract sorted recommendations
	sorted := make([]storage.Recommendation, len(recWithImpacts))
	for i, rwi := range recWithImpacts {
		sorted[i] = rwi.rec
	}

	return sorted
}
