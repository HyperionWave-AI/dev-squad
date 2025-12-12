package aiservice

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"hyper/internal/mcp/storage"
)

// DashboardAggregator provides aggregated metrics for dashboard display
type DashboardAggregator struct {
	metricsStorage storage.MetricsStorage
	logger         *zap.Logger
}

// NewDashboardAggregator creates a new dashboard aggregator
func NewDashboardAggregator(metricsStorage storage.MetricsStorage, logger *zap.Logger) *DashboardAggregator {
	return &DashboardAggregator{
		metricsStorage: metricsStorage,
		logger:         logger,
	}
}

// DailySummary represents daily metrics summary
type DailySummary struct {
	Date                time.Time `json:"date"`
	TotalRequests       int       `json:"totalRequests"`
	TotalTokens         int       `json:"totalTokens"`
	TotalCost           float64   `json:"totalCost"`
	AverageCostPerRequest float64 `json:"averageCostPerRequest"`
	SuccessRate         float64   `json:"successRate"`
	AverageCacheHitRate float64   `json:"averageCacheHitRate"`
}

// WeeklySummary represents weekly metrics summary
type WeeklySummary struct {
	WeekStart           time.Time `json:"weekStart"`
	WeekEnd             time.Time `json:"weekEnd"`
	TotalRequests       int       `json:"totalRequests"`
	TotalTokens         int       `json:"totalTokens"`
	TotalCost           float64   `json:"totalCost"`
	AverageCostPerRequest float64 `json:"averageCostPerRequest"`
	SuccessRate         float64   `json:"successRate"`
	AverageCacheHitRate float64   `json:"averageCacheHitRate"`
	DailySummaries      []DailySummary `json:"dailySummaries"`
}

// MonthlySummary represents monthly metrics summary
type MonthlySummary struct {
	Month               time.Time `json:"month"`
	TotalRequests       int       `json:"totalRequests"`
	TotalTokens         int       `json:"totalTokens"`
	TotalCost           float64   `json:"totalCost"`
	AverageCostPerRequest float64 `json:"averageCostPerRequest"`
	SuccessRate         float64   `json:"successRate"`
	AverageCacheHitRate float64   `json:"averageCacheHitRate"`
	WeeklySummaries     []WeeklySummary `json:"weeklySummaries"`
}

// GetDailySummary returns metrics summary for a specific day
func (d *DashboardAggregator) GetDailySummary(ctx context.Context, date time.Time) (*DailySummary, error) {
	if d.metricsStorage == nil {
		return nil, fmt.Errorf("metrics storage not configured")
	}

	// Set time range for the day
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endTime := startTime.AddDate(0, 0, 1)

	// Get all provider/model combinations for the day
	stats, err := d.metricsStorage.GetMetricsByProvider(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Aggregate across all providers/models
	totalRequests := 0
	totalTokens := 0
	totalCost := 0.0
	totalDuration := int64(0)
	totalCacheHitRate := 0.0
	successCount := 0

	for _, stat := range stats {
		totalRequests += stat.TotalRequests
		totalTokens += stat.TotalTokens
		totalCost += stat.TotalCost
		totalDuration += stat.AverageDuration * int64(stat.TotalRequests)
		totalCacheHitRate += stat.AverageCacheHitRate * float64(stat.TotalRequests)
		successCount += int(float64(stat.TotalRequests) * stat.SuccessRate)
	}

	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
		totalCacheHitRate = totalCacheHitRate / float64(totalRequests)
	}

	averageCostPerRequest := 0.0
	if totalRequests > 0 {
		averageCostPerRequest = totalCost / float64(totalRequests)
	}

	return &DailySummary{
		Date:                  startTime,
		TotalRequests:         totalRequests,
		TotalTokens:           totalTokens,
		TotalCost:             totalCost,
		AverageCostPerRequest: averageCostPerRequest,
		SuccessRate:           successRate,
		AverageCacheHitRate:   totalCacheHitRate,
	}, nil
}

// GetWeeklySummary returns metrics summary for a week
func (d *DashboardAggregator) GetWeeklySummary(ctx context.Context, startDate time.Time) (*WeeklySummary, error) {
	if d.metricsStorage == nil {
		return nil, fmt.Errorf("metrics storage not configured")
	}

	// Calculate week boundaries
	weekStart := startDate
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekEnd := weekStart.AddDate(0, 0, 7)

	// Get daily summaries
	dailySummaries := make([]DailySummary, 0, 7)
	totalRequests := 0
	totalTokens := 0
	totalCost := 0.0
	totalCacheHitRate := 0.0
	successCount := 0

	for i := 0; i < 7; i++ {
		dayDate := weekStart.AddDate(0, 0, i)
		daily, err := d.GetDailySummary(ctx, dayDate)
		if err != nil {
			d.logger.Warn("failed to get daily summary",
				zap.Time("date", dayDate),
				zap.Error(err))
			continue
		}

		dailySummaries = append(dailySummaries, *daily)
		totalRequests += daily.TotalRequests
		totalTokens += daily.TotalTokens
		totalCost += daily.TotalCost
		totalCacheHitRate += daily.AverageCacheHitRate * float64(daily.TotalRequests)
		successCount += int(float64(daily.TotalRequests) * daily.SuccessRate)
	}

	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
		totalCacheHitRate = totalCacheHitRate / float64(totalRequests)
	}

	averageCostPerRequest := 0.0
	if totalRequests > 0 {
		averageCostPerRequest = totalCost / float64(totalRequests)
	}

	return &WeeklySummary{
		WeekStart:             weekStart,
		WeekEnd:               weekEnd,
		TotalRequests:         totalRequests,
		TotalTokens:           totalTokens,
		TotalCost:             totalCost,
		AverageCostPerRequest: averageCostPerRequest,
		SuccessRate:           successRate,
		AverageCacheHitRate:   totalCacheHitRate,
		DailySummaries:        dailySummaries,
	}, nil
}

// GetMonthlySummary returns metrics summary for a month
func (d *DashboardAggregator) GetMonthlySummary(ctx context.Context, year int, month time.Month) (*MonthlySummary, error) {
	if d.metricsStorage == nil {
		return nil, fmt.Errorf("metrics storage not configured")
	}

	// Calculate month boundaries
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	// Get weekly summaries
	weeklySummaries := make([]WeeklySummary, 0)
	totalRequests := 0
	totalTokens := 0
	totalCost := 0.0
	totalCacheHitRate := 0.0
	successCount := 0

	// Iterate through weeks in the month
	currentWeekStart := monthStart
	for currentWeekStart.Before(monthEnd) {
		weekly, err := d.GetWeeklySummary(ctx, currentWeekStart)
		if err != nil {
			d.logger.Warn("failed to get weekly summary",
				zap.Time("weekStart", currentWeekStart),
				zap.Error(err))
		} else {
			// Only include weeks that overlap with the month
			if weekly.WeekStart.Before(monthEnd) {
				weeklySummaries = append(weeklySummaries, *weekly)
				totalRequests += weekly.TotalRequests
				totalTokens += weekly.TotalTokens
				totalCost += weekly.TotalCost
				totalCacheHitRate += weekly.AverageCacheHitRate * float64(weekly.TotalRequests)
				successCount += int(float64(weekly.TotalRequests) * weekly.SuccessRate)
			}
		}

		currentWeekStart = currentWeekStart.AddDate(0, 0, 7)
	}

	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
		totalCacheHitRate = totalCacheHitRate / float64(totalRequests)
	}

	averageCostPerRequest := 0.0
	if totalRequests > 0 {
		averageCostPerRequest = totalCost / float64(totalRequests)
	}

	return &MonthlySummary{
		Month:                 monthStart,
		TotalRequests:         totalRequests,
		TotalTokens:           totalTokens,
		TotalCost:             totalCost,
		AverageCostPerRequest: averageCostPerRequest,
		SuccessRate:           successRate,
		AverageCacheHitRate:   totalCacheHitRate,
		WeeklySummaries:       weeklySummaries,
	}, nil
}

// GetCostTrend returns cost trend over time
func (d *DashboardAggregator) GetCostTrend(ctx context.Context, days int) ([]map[string]interface{}, error) {
	if d.metricsStorage == nil {
		return nil, fmt.Errorf("metrics storage not configured")
	}

	trend := make([]map[string]interface{}, 0, days)

	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		daily, err := d.GetDailySummary(ctx, date)
		if err != nil {
			d.logger.Warn("failed to get daily summary for trend",
				zap.Time("date", date),
				zap.Error(err))
			continue
		}

		trend = append(trend, map[string]interface{}{
			"date":                  daily.Date,
			"totalCost":             daily.TotalCost,
			"totalRequests":         daily.TotalRequests,
			"averageCostPerRequest": daily.AverageCostPerRequest,
		})
	}

	return trend, nil
}

// GetProviderComparison returns cost comparison between providers
func (d *DashboardAggregator) GetProviderComparison(ctx context.Context, startTime time.Time, endTime time.Time) ([]map[string]interface{}, error) {
	if d.metricsStorage == nil {
		return nil, fmt.Errorf("metrics storage not configured")
	}

	stats, err := d.metricsStorage.GetMetricsByProvider(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	comparison := make([]map[string]interface{}, 0, len(stats))
	for key, stat := range stats {
		comparison = append(comparison, map[string]interface{}{
			"provider":              stat.Provider,
			"model":                 stat.Model,
			"totalRequests":         stat.TotalRequests,
			"totalTokens":           stat.TotalTokens,
			"totalCost":             stat.TotalCost,
			"averageCostPerRequest": stat.AverageCostPerRequest,
			"successRate":           stat.SuccessRate,
			"averageCacheHitRate":   stat.AverageCacheHitRate,
			"key":                   key,
		})
	}

	return comparison, nil
}

// GetEfficiencyMetrics returns efficiency metrics and recommendations
func (d *DashboardAggregator) GetEfficiencyMetrics(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]interface{}, error) {
	if d.metricsStorage == nil {
		return nil, fmt.Errorf("metrics storage not configured")
	}

	// Get cost breakdown
	breakdown, err := d.metricsStorage.GetCostByModel(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Get cache hit rates
	cacheStats, err := d.metricsStorage.GetCacheHitRateStats(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Generate recommendations
	recommendations := make([]string, 0)

	// Check for low cache hit rates
	for model, hitRate := range cacheStats {
		if hitRate < 0.3 {
			recommendations = append(recommendations, fmt.Sprintf("Low cache hit rate for %s (%.1f%%). Consider optimizing cache key generation.", model, hitRate*100))
		}
	}

	// Check for expensive models
	totalCost := 0.0
	for _, cb := range breakdown {
		totalCost += cb.TotalCost
	}

	for _, cb := range breakdown {
		percentage := (cb.TotalCost / totalCost) * 100
		if percentage > 50 {
			recommendations = append(recommendations, fmt.Sprintf("Model %s accounts for %.1f%% of costs. Consider using cheaper alternatives.", cb.Model, percentage))
		}
	}

	return map[string]interface{}{
		"costBreakdown":     breakdown,
		"cacheHitRates":     cacheStats,
		"recommendations":   recommendations,
		"periodStart":       startTime,
		"periodEnd":         endTime,
	}, nil
}
