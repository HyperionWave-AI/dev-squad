package handlers

import (
	"net/http"
	"time"

	aiservice "hyper/internal/ai-service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MetricsHandler handles Phase 1 metrics API endpoints
type MetricsHandler struct {
	metricsStore aiservice.MetricsStore
	logger       *zap.Logger
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(metricsStore aiservice.MetricsStore, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		metricsStore: metricsStore,
		logger:       logger,
	}
}

// ProviderMetricsResponse represents provider metrics for API response
type ProviderMetricsResponse struct {
	Provider         string                 `json:"provider"`
	Model            string                 `json:"model"`
	TotalCalls       int                    `json:"totalCalls"`
	SuccessfulCalls  int                    `json:"successfulCalls"`
	FailedCalls      int                    `json:"failedCalls"`
	TotalTokens      int                    `json:"totalTokens"`
	TotalCost        float64                `json:"totalCost"`
	AverageCostPerCall float64              `json:"averageCostPerCall"`
	RecentMetrics    []*aiservice.ProviderMetric `json:"recentMetrics"`
}

// ToolMetricsResponse represents tool metrics for API response
type ToolMetricsResponse struct {
	ToolName       string                `json:"toolName"`
	TotalExecutions int                  `json:"totalExecutions"`
	SuccessfulExecutions int              `json:"successfulExecutions"`
	FailedExecutions int                  `json:"failedExecutions"`
	AverageDurationMs float64             `json:"averageDurationMs"`
	CacheHitRate   float64               `json:"cacheHitRate"`
	RecentMetrics  []*aiservice.ToolMetric `json:"recentMetrics"`
}

// CostBreakdownResponse represents cost breakdown for API response
type CostBreakdownResponse struct {
	Date             time.Time              `json:"date"`
	TotalCost        float64                `json:"totalCost"`
	ProviderCosts    map[string]float64     `json:"providerCosts"`
	ModelCosts       map[string]float64     `json:"modelCosts"`
	TokensUsed       int                    `json:"tokensUsed"`
	SuccessfulCalls  int                    `json:"successfulCalls"`
	FailedCalls      int                    `json:"failedCalls"`
}

// Phase1MetricsResponse represents all Phase 1 metrics
type Phase1MetricsResponse struct {
	ProviderMetrics map[string]*ProviderMetricsResponse `json:"providerMetrics"`
	ToolMetrics     map[string]*ToolMetricsResponse     `json:"toolMetrics"`
	CostByProvider  map[string]float64                  `json:"costByProvider"`
	CostByModel     map[string]float64                  `json:"costByModel"`
	DailyCostBreakdown *CostBreakdownResponse           `json:"dailyCostBreakdown"`
	CacheMetrics    map[string]*aiservice.CacheMetric   `json:"cacheMetrics"`
}

// GetPhase1Metrics returns all Phase 1 metrics
// GET /api/v1/metrics/phase1
func (h *MetricsHandler) GetPhase1Metrics(c *gin.Context) {
	// Get date range from query params (default to today)
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.logger.Warn("Invalid date format", zap.String("date", dateStr), zap.Error(err))
		date = time.Now()
	}

	// Get date range for cost aggregations (last 30 days)
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	response := &Phase1MetricsResponse{
		ProviderMetrics: make(map[string]*ProviderMetricsResponse),
		ToolMetrics:     make(map[string]*ToolMetricsResponse),
		CacheMetrics:    make(map[string]*aiservice.CacheMetric),
	}

	// Get provider metrics (OpenAI and Anthropic)
	for _, provider := range []string{"openai", "anthropic"} {
		metrics, err := h.metricsStore.GetProviderMetrics(provider, 100)
		if err != nil {
			h.logger.Warn("Failed to get provider metrics", zap.String("provider", provider), zap.Error(err))
			continue
		}

		if len(metrics) == 0 {
			continue
		}

		// Calculate aggregates
		var totalCost float64
		var totalTokens int
		var successCount, failCount int

		for _, m := range metrics {
			totalCost += m.Cost
			totalTokens += m.TotalTokens
			if m.Success {
				successCount++
			} else {
				failCount++
			}
		}

		avgCost := totalCost / float64(len(metrics))

		response.ProviderMetrics[provider] = &ProviderMetricsResponse{
			Provider:           provider,
			TotalCalls:         len(metrics),
			SuccessfulCalls:    successCount,
			FailedCalls:        failCount,
			TotalTokens:        totalTokens,
			TotalCost:          totalCost,
			AverageCostPerCall: avgCost,
			RecentMetrics:      metrics,
		}
	}

	// Get cost by provider
	costByProvider, err := h.metricsStore.GetCostByProvider(startDate, endDate)
	if err != nil {
		h.logger.Warn("Failed to get cost by provider", zap.Error(err))
	} else {
		response.CostByProvider = costByProvider
	}

	// Get cost by model
	costByModel, err := h.metricsStore.GetCostByModel(startDate, endDate)
	if err != nil {
		h.logger.Warn("Failed to get cost by model", zap.Error(err))
	} else {
		response.CostByModel = costByModel
	}

	// Get daily cost breakdown
	dailyBreakdown, err := h.metricsStore.GetDailyCostBreakdown(date)
	if err != nil {
		h.logger.Warn("Failed to get daily cost breakdown", zap.Error(err))
	} else {
		response.DailyCostBreakdown = &CostBreakdownResponse{
			Date:            dailyBreakdown.Date,
			TotalCost:       dailyBreakdown.TotalCost,
			ProviderCosts:   dailyBreakdown.ProviderCosts,
			ModelCosts:      dailyBreakdown.ModelCosts,
			TokensUsed:      dailyBreakdown.TokensUsed,
			SuccessfulCalls: dailyBreakdown.SuccessfulCalls,
			FailedCalls:     dailyBreakdown.FailedCalls,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetProviderMetrics returns metrics for a specific provider
// GET /api/v1/metrics/providers/:provider
func (h *MetricsHandler) GetProviderMetrics(c *gin.Context) {
	provider := c.Param("provider")
	limit := 100

	metrics, err := h.metricsStore.GetProviderMetrics(provider, limit)
	if err != nil {
		h.logger.Error("Failed to get provider metrics", zap.String("provider", provider), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metrics"})
		return
	}

	if len(metrics) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"provider": provider,
			"metrics":  []interface{}{},
		})
		return
	}

	// Calculate aggregates
	var totalCost float64
	var totalTokens int
	var successCount, failCount int

	for _, m := range metrics {
		totalCost += m.Cost
		totalTokens += m.TotalTokens
		if m.Success {
			successCount++
		} else {
			failCount++
		}
	}

	avgCost, err := h.metricsStore.GetAverageCostPerCall(provider)
	if err != nil {
		h.logger.Warn("Failed to get average cost", zap.String("provider", provider), zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"provider":           provider,
		"totalCalls":         len(metrics),
		"successfulCalls":    successCount,
		"failedCalls":        failCount,
		"totalTokens":        totalTokens,
		"totalCost":          totalCost,
		"averageCostPerCall": avgCost,
		"metrics":            metrics,
	})
}

// GetToolMetrics returns metrics for a specific tool
// GET /api/v1/metrics/tools/:toolName
func (h *MetricsHandler) GetToolMetrics(c *gin.Context) {
	toolName := c.Param("toolName")
	limit := 100

	metrics, err := h.metricsStore.GetToolMetrics(toolName, limit)
	if err != nil {
		h.logger.Error("Failed to get tool metrics", zap.String("toolName", toolName), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metrics"})
		return
	}

	if len(metrics) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"toolName": toolName,
			"metrics":  []interface{}{},
		})
		return
	}

	// Calculate aggregates
	var totalDuration int64
	var successCount, failCount, cacheHits int

	for _, m := range metrics {
		totalDuration += m.DurationMs
		if m.Success {
			successCount++
		} else {
			failCount++
		}
		if m.CacheHit {
			cacheHits++
		}
	}

	avgDuration := float64(totalDuration) / float64(len(metrics))
	cacheHitRate := float64(cacheHits) / float64(len(metrics)) * 100

	c.JSON(http.StatusOK, gin.H{
		"toolName":              toolName,
		"totalExecutions":       len(metrics),
		"successfulExecutions":  successCount,
		"failedExecutions":      failCount,
		"averageDurationMs":     avgDuration,
		"cacheHitRate":          cacheHitRate,
		"metrics":               metrics,
	})
}

// GetCacheMetrics returns cache metrics for a specific tool
// GET /api/v1/metrics/cache/:toolName
func (h *MetricsHandler) GetCacheMetrics(c *gin.Context) {
	toolName := c.Param("toolName")

	cacheMetric, err := h.metricsStore.GetCacheMetrics(toolName)
	if err != nil {
		h.logger.Error("Failed to get cache metrics", zap.String("toolName", toolName), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve cache metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"toolName":       toolName,
		"cacheMetrics":   cacheMetric,
	})
}

// GetCostBreakdown returns cost breakdown for a specific date
// GET /api/v1/metrics/costs/daily?date=2024-01-15
func (h *MetricsHandler) GetCostBreakdown(c *gin.Context) {
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.logger.Warn("Invalid date format", zap.String("date", dateStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format (use YYYY-MM-DD)"})
		return
	}

	breakdown, err := h.metricsStore.GetDailyCostBreakdown(date)
	if err != nil {
		h.logger.Error("Failed to get cost breakdown", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve cost breakdown"})
		return
	}

	c.JSON(http.StatusOK, &CostBreakdownResponse{
		Date:            breakdown.Date,
		TotalCost:       breakdown.TotalCost,
		ProviderCosts:   breakdown.ProviderCosts,
		ModelCosts:      breakdown.ModelCosts,
		TokensUsed:      breakdown.TokensUsed,
		SuccessfulCalls: breakdown.SuccessfulCalls,
		FailedCalls:     breakdown.FailedCalls,
	})
}

// RegisterMetricsRoutes registers all metrics routes
func (h *MetricsHandler) RegisterMetricsRoutes(r *gin.RouterGroup) {
	r.GET("/phase1", h.GetPhase1Metrics)
	r.GET("/providers/:provider", h.GetProviderMetrics)
	r.GET("/tools/:toolName", h.GetToolMetrics)
	r.GET("/cache/:toolName", h.GetCacheMetrics)
	r.GET("/costs/daily", h.GetCostBreakdown)
}
