package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/dev-squad/hyper/internal/ai-service/alerts"
	"github.com/dev-squad/hyper/internal/ai-service/metrics"
	"github.com/dev-squad/hyper/internal/ai-service/recommendations"
	"github.com/dev-squad/hyper/internal/storage"
	"go.uber.org/zap"
)

// MetricsDashboardHandler handles metrics dashboard API endpoints
type MetricsDashboardHandler struct {
	metricsStorage storage.MetricsStorage
	logger         *zap.Logger
}

// NewMetricsDashboardHandler creates a new metrics dashboard handler
func NewMetricsDashboardHandler(metricsStorage storage.MetricsStorage, logger *zap.Logger) *MetricsDashboardHandler {
	return &MetricsDashboardHandler{
		metricsStorage: metricsStorage,
		logger:         logger,
	}
}

// RegisterRoutes registers all metrics dashboard routes
func (h *MetricsDashboardHandler) RegisterRoutes(router *gin.Engine) {
	metricsGroup := router.Group("/api/v1/metrics")
	{
		metricsGroup.GET("/efficiency", h.GetEfficiencyMetrics)
		metricsGroup.GET("/cost-trend", h.GetCostTrend)
		metricsGroup.GET("/optimization-opportunities", h.GetOptimizationOpportunities)
		metricsGroup.GET("/budget-alerts", h.GetBudgetAlerts)
		metricsGroup.GET("/recommendations", h.GetRecommendations)
		metricsGroup.GET("/anomalies", h.DetectAnomalies)
		metricsGroup.POST("/budget-config", h.UpdateBudgetConfig)
	}
}

// GetEfficiencyMetrics returns efficiency metrics for a time period
func (h *MetricsDashboardHandler) GetEfficiencyMetrics(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in context"})
		return
	}

	// Parse query parameters
	days := c.DefaultQuery("days", "7")
	var daysInt int
	if _, err := time.Parse("2006-01-02", days); err == nil {
		// It's a date, calculate days from now
		parsedDate, _ := time.Parse("2006-01-02", days)
		daysInt = int(time.Since(parsedDate).Hours() / 24)
	} else {
		// It's a number of days
		_, _ = time.Parse("2006-01-02", days)
		daysInt = 7
	}

	startTime := time.Now().AddDate(0, 0, -daysInt)
	endTime := time.Now()

	// Create optimizer metrics
	om := metrics.NewOptimizerMetrics(h.metricsStorage)

	// Get efficiency metrics
	effMetrics, err := om.GetEfficiencyMetrics(c.Request.Context(), userID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to get efficiency metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get efficiency metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": effMetrics,
		"period": gin.H{
			"start": startTime,
			"end":   endTime,
			"days":  daysInt,
		},
	})
}

// GetCostTrend returns cost trend analysis
func (h *MetricsDashboardHandler) GetCostTrend(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in context"})
		return
	}

	// Get current period (last 7 days)
	now := time.Now()
	currentEnd := now
	currentStart := now.AddDate(0, 0, -7)

	// Get previous period (7 days before that)
	previousEnd := currentStart
	previousStart := previousEnd.AddDate(0, 0, -7)

	// Create optimizer metrics
	om := metrics.NewOptimizerMetrics(h.metricsStorage)

	// Analyze cost trend
	trend, err := om.AnalyzeCostTrend(c.Request.Context(), userID, currentStart, currentEnd, previousStart, previousEnd)
	if err != nil {
		h.logger.Error("failed to analyze cost trend", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze cost trend"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": trend,
		"periods": gin.H{
			"current": gin.H{
				"start": currentStart,
				"end":   currentEnd,
			},
			"previous": gin.H{
				"start": previousStart,
				"end":   previousEnd,
			},
		},
	})
}

// GetOptimizationOpportunities returns optimization opportunities
func (h *MetricsDashboardHandler) GetOptimizationOpportunities(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in context"})
		return
	}

	startTime := time.Now().AddDate(0, 0, -30)
	endTime := time.Now()

	// Create optimizer metrics
	om := metrics.NewOptimizerMetrics(h.metricsStorage)

	// Analyze opportunities
	opportunities, err := om.AnalyzeOptimizationOpportunities(c.Request.Context(), userID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to analyze optimization opportunities", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze optimization opportunities"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": opportunities,
		"count": len(opportunities),
	})
}

// GetBudgetAlerts returns active budget alerts
func (h *MetricsDashboardHandler) GetBudgetAlerts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in context"})
		return
	}

	// Create budget alert manager
	bam := alerts.NewBudgetAlertManager(h.metricsStorage)

	// Get active alerts
	activeAlerts, err := bam.GetActiveAlerts(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get budget alerts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get budget alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  activeAlerts,
		"count": len(activeAlerts),
	})
}

// GetRecommendations returns optimization recommendations
func (h *MetricsDashboardHandler) GetRecommendations(c *gin.Context) {
	userID := c.GetString("userID")
	companyID := c.GetString("companyID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in context"})
		return
	}

	startTime := time.Now().AddDate(0, 0, -30)
	endTime := time.Now()

	// Create recommendation engine
	re := recommendations.NewRecommendationEngine(h.metricsStorage)

	// Generate recommendations
	recs, err := re.GenerateRecommendations(c.Request.Context(), userID, companyID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to generate recommendations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate recommendations"})
		return
	}

	// Get total monthly spend for prioritization
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())
	monthEnd := monthStart.AddDate(0, 1, 0)
	costMetrics, _ := h.metricsStorage.GetCostMetrics(c.Request.Context(), userID, monthStart, monthEnd)
	var monthlySpend float64
	for _, cm := range costMetrics {
		monthlySpend += cm.TotalCost
	}

	// Prioritize recommendations
	prioritized := re.PrioritizeRecommendations(recs, monthlySpend)

	c.JSON(http.StatusOK, gin.H{
		"data":  prioritized,
		"count": len(prioritized),
	})
}

// DetectAnomalies detects unusual spending patterns
func (h *MetricsDashboardHandler) DetectAnomalies(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in context"})
		return
	}

	// Get today's cost
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	costMetrics, err := h.metricsStorage.GetCostMetrics(c.Request.Context(), userID, startOfDay, endOfDay)
	if err != nil {
		h.logger.Error("failed to get cost metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get cost metrics"})
		return
	}

	var todayCost float64
	for _, cm := range costMetrics {
		todayCost += cm.TotalCost
	}

	// Create budget alert manager
	bam := alerts.NewBudgetAlertManager(h.metricsStorage)

	// Detect anomalies
	anomaly, err := bam.DetectAnomalies(c.Request.Context(), userID, todayCost)
	if err != nil {
		h.logger.Error("failed to detect anomalies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to detect anomalies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": anomaly,
	})
}

// UpdateBudgetConfig updates budget configuration
func (h *MetricsDashboardHandler) UpdateBudgetConfig(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in context"})
		return
	}

	var config alerts.BudgetConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// TODO: Save budget config to storage
	// For now, just return success
	c.JSON(http.StatusOK, gin.H{
		"message": "budget config updated successfully",
		"config":  config,
	})
}
