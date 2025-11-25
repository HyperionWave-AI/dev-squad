package metrics

import (
	"sync"
	"time"
)

// ProviderMetrics tracks metrics for a single AI provider API call
type ProviderMetrics struct {
	// Identification
	Provider  string    // "openai", "anthropic", "groq", etc.
	Model     string    // "gpt-4", "claude-3-opus", etc.
	Timestamp time.Time // When the API call was made

	// Token Usage
	PromptTokens     int // Input tokens consumed
	CompletionTokens int // Output tokens consumed
	TotalTokens      int // Total tokens (prompt + completion)

	// Timing
	DurationMs int64 // API call duration in milliseconds

	// Cost Calculation
	PromptCost     float64 // Cost of prompt tokens
	CompletionCost float64 // Cost of completion tokens
	TotalCost      float64 // Total cost (prompt + completion)

	// Metadata
	RequestID string // Unique request identifier
	UserID    string // User who made the request (optional)
	SessionID string // Chat session ID (optional)
}

// ToolMetrics tracks metrics for a single tool execution
type ToolMetrics struct {
	// Identification
	ToolName  string    // Name of the tool executed
	Timestamp time.Time // When the tool was executed

	// Execution Details
	DurationMs      int64  // Tool execution time in milliseconds
	Status          string // "success", "error", "timeout", "cached"
	Error           string // Error message if failed (empty if successful)
	CacheHit        bool   // Whether result was from cache
	InputTokens     int    // Tokens used for tool input (if applicable)
	OutputTokens    int    // Tokens used for tool output (if applicable)

	// Metadata
	RequestID string // Unique request identifier
	UserID    string // User who made the request (optional)
	SessionID string // Chat session ID (optional)
}

// CostBreakdown provides detailed cost analysis
type CostBreakdown struct {
	// By Provider
	ProviderCosts map[string]float64 // Provider -> Total Cost
	ModelCosts    map[string]float64 // Model -> Total Cost

	// By Time Period
	DailyCosts   map[string]float64 // YYYY-MM-DD -> Total Cost
	HourlyCosts  map[string]float64 // YYYY-MM-DD HH:00 -> Total Cost

	// Aggregates
	TotalCost        float64 // Total cost across all providers
	AveragePerCall   float64 // Average cost per API call
	HighestCostCall  float64 // Most expensive single call
	LowestCostCall   float64 // Least expensive single call

	// Token Statistics
	TotalPromptTokens     int64 // Total input tokens
	TotalCompletionTokens int64 // Total output tokens
	AveragePromptTokens   int   // Average input tokens per call
	AverageCompletionTokens int // Average output tokens per call

	// Efficiency Metrics
	CostPerPromptToken     float64 // Average cost per input token
	CostPerCompletionToken float64 // Average cost per output token
}

// MetricsStore defines the interface for storing and retrieving metrics
type MetricsStore interface {
	// Store provider metrics
	StoreProviderMetrics(metrics *ProviderMetrics) error

	// Store tool metrics
	StoreToolMetrics(metrics *ToolMetrics) error

	// Query provider metrics
	GetProviderMetricsByDateRange(startDate, endDate time.Time) ([]*ProviderMetrics, error)
	GetProviderMetricsByModel(model string, startDate, endDate time.Time) ([]*ProviderMetrics, error)
	GetProviderMetricsByProvider(provider string, startDate, endDate time.Time) ([]*ProviderMetrics, error)

	// Query tool metrics
	GetToolMetricsByDateRange(startDate, endDate time.Time) ([]*ToolMetrics, error)
	GetToolMetricsByName(toolName string, startDate, endDate time.Time) ([]*ToolMetrics, error)

	// Aggregations
	GetCostBreakdown(startDate, endDate time.Time) (*CostBreakdown, error)
	GetDailyCosts(startDate, endDate time.Time) (map[string]float64, error)
	GetWeeklyCosts(startDate, endDate time.Time) (map[string]float64, error)
	GetMonthlyCosts(startDate, endDate time.Time) (map[string]float64, error)

	// Cache statistics
	GetCacheStats(startDate, endDate time.Time) (map[string]interface{}, error)

	// Cleanup
	DeleteMetricsOlderThan(days int) error
}

// MetricsRecorder handles recording provider metrics with cost calculations
type MetricsRecorder struct {
	store MetricsStore
	mu    sync.RWMutex

	// Pricing tiers for different providers and models
	pricingTiers map[string]map[string]PricingTier
}

// PricingTier defines pricing for a specific model
type PricingTier struct {
	PromptTokenPrice     float64 // Price per 1K input tokens
	CompletionTokenPrice float64 // Price per 1K output tokens
}

// ToolMetricsTracker handles recording tool execution metrics
type ToolMetricsTracker struct {
	store MetricsStore
	mu    sync.RWMutex
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	TotalRequests  int64   // Total tool executions
	CacheHits      int64   // Successful cache hits
	CacheMisses    int64   // Cache misses
	HitRate        float64 // Percentage of cache hits (0-100)
	TokensSaved    int64   // Tokens saved by caching
	CostSaved      float64 // Cost saved by caching
	AverageHitTime int64   // Average time for cache hit in ms
}

// RecommendationItem represents a cost optimization recommendation
type RecommendationItem struct {
	Title       string  // Short title of recommendation
	Description string  // Detailed description
	Potential   float64 // Potential savings in dollars
	Priority    string  // "high", "medium", "low"
	Action      string  // Suggested action to take
}

// Recommendations provides cost optimization suggestions
type Recommendations struct {
	Items          []RecommendationItem
	TotalPotential float64 // Total potential savings
	GeneratedAt    time.Time
}
