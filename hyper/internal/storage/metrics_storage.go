package storage

import (
	"context"
	"time"
)

// MetricsStorage defines the interface for storing and retrieving AI metrics
type MetricsStorage interface {
	// SaveTokenMetrics saves token usage metrics for a request
	SaveTokenMetrics(ctx context.Context, metrics *TokenMetrics) error

	// SaveCostMetrics saves cost metrics for a request
	SaveCostMetrics(ctx context.Context, metrics *CostMetrics) error

	// GetTokenMetrics retrieves token metrics for a time period
	GetTokenMetrics(ctx context.Context, userID string, startTime, endTime time.Time) ([]*TokenMetrics, error)

	// GetCostMetrics retrieves cost metrics for a time period
	GetCostMetrics(ctx context.Context, userID string, startTime, endTime time.Time) ([]*CostMetrics, error)

	// GetOptimizationMetrics retrieves optimization metrics
	GetOptimizationMetrics(ctx context.Context, userID string, startTime, endTime time.Time) (*OptimizationMetrics, error)

	// SaveBudgetAlert saves a budget alert
	SaveBudgetAlert(ctx context.Context, alert *BudgetAlert) error

	// GetBudgetAlerts retrieves budget alerts for a user
	GetBudgetAlerts(ctx context.Context, userID string, limit int) ([]*BudgetAlert, error)

	// SaveRecommendation saves an optimization recommendation
	SaveRecommendation(ctx context.Context, rec *Recommendation) error

	// GetRecommendations retrieves recommendations for a user
	GetRecommendations(ctx context.Context, userID string, limit int) ([]*Recommendation, error)
}

// TokenMetrics represents token usage for a single request
type TokenMetrics struct {
	ID              string    `bson:"_id,omitempty" json:"id"`
	UserID          string    `bson:"userId" json:"userId"`
	CompanyID       string    `bson:"companyId" json:"companyId"`
	RequestID       string    `bson:"requestId" json:"requestId"`
	Model           string    `bson:"model" json:"model"`
	Provider        string    `bson:"provider" json:"provider"`
	InputTokens     int       `bson:"inputTokens" json:"inputTokens"`
	OutputTokens    int       `bson:"outputTokens" json:"outputTokens"`
	TotalTokens     int       `bson:"totalTokens" json:"totalTokens"`
	CacheHitTokens  int       `bson:"cacheHitTokens" json:"cacheHitTokens"`
	CacheMissTokens int       `bson:"cacheMissTokens" json:"cacheMissTokens"`
	Timestamp       time.Time `bson:"timestamp" json:"timestamp"`
}

// CostMetrics represents cost information for a single request
type CostMetrics struct {
	ID                 string    `bson:"_id,omitempty" json:"id"`
	UserID             string    `bson:"userId" json:"userId"`
	CompanyID          string    `bson:"companyId" json:"companyId"`
	RequestID          string    `bson:"requestId" json:"requestId"`
	Model              string    `bson:"model" json:"model"`
	InputCost          float64   `bson:"inputCost" json:"inputCost"`
	OutputCost         float64   `bson:"outputCost" json:"outputCost"`
	TotalCost          float64   `bson:"totalCost" json:"totalCost"`
	CacheHitSavings    float64   `bson:"cacheHitSavings" json:"cacheHitSavings"`
	Currency           string    `bson:"currency" json:"currency"`
	Timestamp          time.Time `bson:"timestamp" json:"timestamp"`
}

// OptimizationMetrics represents aggregated optimization metrics
type OptimizationMetrics struct {
	UserID                  string    `json:"userId"`
	CompanyID               string    `json:"companyId"`
	Period                  string    `json:"period"` // "daily", "weekly", "monthly"
	TotalRequests           int       `json:"totalRequests"`
	TotalTokens             int       `json:"totalTokens"`
	AverageTokensPerRequest float64   `json:"averageTokensPerRequest"`
	TotalCost               float64   `json:"totalCost"`
	AverageCostPerRequest   float64   `json:"averageCostPerRequest"`
	TokenEfficiency         float64   `json:"tokenEfficiency"` // output/input ratio
	CacheHitRate            float64   `json:"cacheHitRate"`    // percentage
	MostUsedModel           string    `json:"mostUsedModel"`
	MostExpensiveModel      string    `json:"mostExpensiveModel"`
	CostTrend               string    `json:"costTrend"` // "increasing", "stable", "decreasing"
	Timestamp               time.Time `json:"timestamp"`
}

// BudgetAlert represents a budget threshold alert
type BudgetAlert struct {
	ID              string    `bson:"_id,omitempty" json:"id"`
	UserID          string    `bson:"userId" json:"userId"`
	CompanyID       string    `bson:"companyId" json:"companyId"`
	AlertType       string    `bson:"alertType" json:"alertType"` // "daily", "monthly", "per_request"
	Threshold       float64   `bson:"threshold" json:"threshold"`
	CurrentValue    float64   `bson:"currentValue" json:"currentValue"`
	PercentageUsed  float64   `bson:"percentageUsed" json:"percentageUsed"`
	Severity        string    `bson:"severity" json:"severity"` // "warning", "critical"
	Message         string    `bson:"message" json:"message"`
	IsResolved      bool      `bson:"isResolved" json:"isResolved"`
	CreatedAt       time.Time `bson:"createdAt" json:"createdAt"`
	ResolvedAt      *time.Time `bson:"resolvedAt,omitempty" json:"resolvedAt,omitempty"`
}

// Recommendation represents an optimization recommendation
type Recommendation struct {
	ID              string    `bson:"_id,omitempty" json:"id"`
	UserID          string    `bson:"userId" json:"userId"`
	CompanyID       string    `bson:"companyId" json:"companyId"`
	Category        string    `bson:"category" json:"category"` // "prompt_optimization", "model_selection", "caching", "batching"
	Title           string    `bson:"title" json:"title"`
	Description     string    `bson:"description" json:"description"`
	PotentialSavings float64  `bson:"potentialSavings" json:"potentialSavings"`
	SavingsUnit     string    `bson:"savingsUnit" json:"savingsUnit"` // "tokens", "cost"
	Priority        string    `bson:"priority" json:"priority"`       // "low", "medium", "high"
	ActionItems     []string  `bson:"actionItems" json:"actionItems"`
	IsImplemented   bool      `bson:"isImplemented" json:"isImplemented"`
	CreatedAt       time.Time `bson:"createdAt" json:"createdAt"`
	ImplementedAt   *time.Time `bson:"implementedAt,omitempty" json:"implementedAt,omitempty"`
}

// MetricsCollector provides methods to collect and aggregate metrics
type MetricsCollector struct {
	storage MetricsStorage
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(storage MetricsStorage) *MetricsCollector {
	return &MetricsCollector{
		storage: storage,
	}
}

// RecordTokenUsage records token usage for a request
func (mc *MetricsCollector) RecordTokenUsage(ctx context.Context, metrics *TokenMetrics) error {
	return mc.storage.SaveTokenMetrics(ctx, metrics)
}

// RecordCost records cost information for a request
func (mc *MetricsCollector) RecordCost(ctx context.Context, metrics *CostMetrics) error {
	return mc.storage.SaveCostMetrics(ctx, metrics)
}

// GetMetricsSummary retrieves a summary of metrics for a time period
func (mc *MetricsCollector) GetMetricsSummary(ctx context.Context, userID string, startTime, endTime time.Time) (*OptimizationMetrics, error) {
	return mc.storage.GetOptimizationMetrics(ctx, userID, startTime, endTime)
}
