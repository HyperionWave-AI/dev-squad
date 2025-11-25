package aiservice

import (
	"fmt"
	"sync"
	"time"
)

// ProviderMetric represents a single API call to an LLM provider
type ProviderMetric struct {
	ID               string    `json:"id"`
	Provider         string    `json:"provider"`        // "openai", "anthropic", etc.
	Model            string    `json:"model"`           // "gpt-4", "claude-3-opus", etc.
	PromptTokens     int       `json:"promptTokens"`    // Input tokens
	CompletionTokens int       `json:"completionTokens"` // Output tokens
	TotalTokens      int       `json:"totalTokens"`     // Total tokens
	Cost             float64   `json:"cost"`            // USD cost
	DurationMs       int64     `json:"durationMs"`      // Execution time
	Success          bool      `json:"success"`         // Whether the call succeeded
	ErrorMessage     string    `json:"errorMessage,omitempty"` // Error details if failed
	Timestamp        time.Time `json:"timestamp"`       // When the call was made
}

// ToolMetric represents a single tool execution
type ToolMetric struct {
	ID             string    `json:"id"`
	ToolName       string    `json:"toolName"`        // "read_file", "bash", etc.
	DurationMs     int64     `json:"durationMs"`      // Execution time
	Success        bool      `json:"success"`         // Whether execution succeeded
	ErrorMessage   string    `json:"errorMessage,omitempty"` // Error details if failed
	CacheHit       bool      `json:"cacheHit"`        // Whether result came from cache
	InputSize      int       `json:"inputSize"`       // Size of input arguments
	OutputSize     int       `json:"outputSize"`      // Size of output
	Timestamp      time.Time `json:"timestamp"`       // When the tool was executed
}

// CacheMetric represents cache performance statistics
type CacheMetric struct {
	TotalRequests int     `json:"totalRequests"`
	CacheHits     int     `json:"cacheHits"`
	CacheMisses   int     `json:"cacheMisses"`
	HitRate       float64 `json:"hitRate"` // Percentage (0-100)
}

// DailyCostBreakdown represents cost aggregation for a single day
type DailyCostBreakdown struct {
	Date              time.Time              `json:"date"`
	TotalCost         float64                `json:"totalCost"`
	ProviderCosts     map[string]float64     `json:"providerCosts"`     // Cost by provider
	ModelCosts        map[string]float64     `json:"modelCosts"`        // Cost by model
	TokensUsed        int                    `json:"tokensUsed"`
	SuccessfulCalls   int                    `json:"successfulCalls"`
	FailedCalls       int                    `json:"failedCalls"`
}

// MetricsStore defines the interface for storing and querying metrics
type MetricsStore interface {
	// Provider metrics
	RecordProviderMetric(metric *ProviderMetric) error
	GetProviderMetrics(provider string, limit int) ([]*ProviderMetric, error)
	
	// Tool metrics
	RecordToolMetric(metric *ToolMetric) error
	GetToolMetrics(toolName string, limit int) ([]*ToolMetric, error)
	
	// Cache metrics
	RecordCacheHit(toolName string) error
	RecordCacheMiss(toolName string) error
	GetCacheMetrics(toolName string) (*CacheMetric, error)
	
	// Aggregations
	GetDailyCostBreakdown(date time.Time) (*DailyCostBreakdown, error)
	GetCostByProvider(startDate, endDate time.Time) (map[string]float64, error)
	GetCostByModel(startDate, endDate time.Time) (map[string]float64, error)
	GetAverageCostPerCall(provider string) (float64, error)
}

// InMemoryMetricsStore is a thread-safe in-memory implementation of MetricsStore
type InMemoryMetricsStore struct {
	mu                sync.RWMutex
	providerMetrics   []*ProviderMetric
	toolMetrics       []*ToolMetric
	cacheMetrics      map[string]*CacheMetric // Key: toolName
	maxMetricsPerType int                     // Keep last N metrics per type
}

// NewInMemoryMetricsStore creates a new in-memory metrics store
func NewInMemoryMetricsStore(maxMetricsPerType int) *InMemoryMetricsStore {
	if maxMetricsPerType <= 0 {
		maxMetricsPerType = 10000 // Default: keep last 10k metrics
	}
	return &InMemoryMetricsStore{
		providerMetrics:   make([]*ProviderMetric, 0, maxMetricsPerType),
		toolMetrics:       make([]*ToolMetric, 0, maxMetricsPerType),
		cacheMetrics:      make(map[string]*CacheMetric),
		maxMetricsPerType: maxMetricsPerType,
	}
}

// RecordProviderMetric stores a provider metric
func (s *InMemoryMetricsStore) RecordProviderMetric(metric *ProviderMetric) error {
	if metric == nil {
		return fmt.Errorf("metric cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Add metric
	s.providerMetrics = append(s.providerMetrics, metric)

	// Trim if exceeds max
	if len(s.providerMetrics) > s.maxMetricsPerType {
		s.providerMetrics = s.providerMetrics[len(s.providerMetrics)-s.maxMetricsPerType:]
	}

	return nil
}

// GetProviderMetrics retrieves provider metrics
func (s *InMemoryMetricsStore) GetProviderMetrics(provider string, limit int) ([]*ProviderMetric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ProviderMetric
	for _, m := range s.providerMetrics {
		if m.Provider == provider {
			result = append(result, m)
		}
	}

	// Return last N results
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}

	return result, nil
}

// RecordToolMetric stores a tool metric
func (s *InMemoryMetricsStore) RecordToolMetric(metric *ToolMetric) error {
	if metric == nil {
		return fmt.Errorf("metric cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Add metric
	s.toolMetrics = append(s.toolMetrics, metric)

	// Trim if exceeds max
	if len(s.toolMetrics) > s.maxMetricsPerType {
		s.toolMetrics = s.toolMetrics[len(s.toolMetrics)-s.maxMetricsPerType:]
	}

	return nil
}

// GetToolMetrics retrieves tool metrics
func (s *InMemoryMetricsStore) GetToolMetrics(toolName string, limit int) ([]*ToolMetric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ToolMetric
	for _, m := range s.toolMetrics {
		if m.ToolName == toolName {
			result = append(result, m)
		}
	}

	// Return last N results
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}

	return result, nil
}

// RecordCacheHit records a cache hit for a tool
func (s *InMemoryMetricsStore) RecordCacheHit(toolName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cacheMetrics[toolName]; !exists {
		s.cacheMetrics[toolName] = &CacheMetric{}
	}

	metric := s.cacheMetrics[toolName]
	metric.CacheHits++
	metric.TotalRequests++
	metric.HitRate = float64(metric.CacheHits) / float64(metric.TotalRequests) * 100

	return nil
}

// RecordCacheMiss records a cache miss for a tool
func (s *InMemoryMetricsStore) RecordCacheMiss(toolName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cacheMetrics[toolName]; !exists {
		s.cacheMetrics[toolName] = &CacheMetric{}
	}

	metric := s.cacheMetrics[toolName]
	metric.CacheMisses++
	metric.TotalRequests++
	metric.HitRate = float64(metric.CacheHits) / float64(metric.TotalRequests) * 100

	return nil
}

// GetCacheMetrics retrieves cache metrics for a tool
func (s *InMemoryMetricsStore) GetCacheMetrics(toolName string) (*CacheMetric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metric, exists := s.cacheMetrics[toolName]; exists {
		return metric, nil
	}

	return &CacheMetric{}, nil
}

// GetDailyCostBreakdown aggregates costs for a specific day
func (s *InMemoryMetricsStore) GetDailyCostBreakdown(date time.Time) (*DailyCostBreakdown, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Normalize date to start of day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	breakdown := &DailyCostBreakdown{
		Date:          startOfDay,
		ProviderCosts: make(map[string]float64),
		ModelCosts:    make(map[string]float64),
	}

	for _, metric := range s.providerMetrics {
		if metric.Timestamp.After(startOfDay) && metric.Timestamp.Before(endOfDay) {
			breakdown.TotalCost += metric.Cost
			breakdown.ProviderCosts[metric.Provider] += metric.Cost
			breakdown.ModelCosts[metric.Model] += metric.Cost
			breakdown.TokensUsed += metric.TotalTokens

			if metric.Success {
				breakdown.SuccessfulCalls++
			} else {
				breakdown.FailedCalls++
			}
		}
	}

	return breakdown, nil
}

// GetCostByProvider aggregates costs by provider for a date range
func (s *InMemoryMetricsStore) GetCostByProvider(startDate, endDate time.Time) (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	costs := make(map[string]float64)

	for _, metric := range s.providerMetrics {
		if metric.Timestamp.After(startDate) && metric.Timestamp.Before(endDate) {
			costs[metric.Provider] += metric.Cost
		}
	}

	return costs, nil
}

// GetCostByModel aggregates costs by model for a date range
func (s *InMemoryMetricsStore) GetCostByModel(startDate, endDate time.Time) (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	costs := make(map[string]float64)

	for _, metric := range s.providerMetrics {
		if metric.Timestamp.After(startDate) && metric.Timestamp.Before(endDate) {
			costs[metric.Model] += metric.Cost
		}
	}

	return costs, nil
}

// GetAverageCostPerCall calculates average cost per API call for a provider
func (s *InMemoryMetricsStore) GetAverageCostPerCall(provider string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalCost float64
	var count int

	for _, metric := range s.providerMetrics {
		if metric.Provider == provider {
			totalCost += metric.Cost
			count++
		}
	}

	if count == 0 {
		return 0, nil
	}

	return totalCost / float64(count), nil
}

// ===== PRICING CONSTANTS =====

// OpenAI pricing (as of 2024)
const (
	// GPT-4 pricing per 1K tokens
	GPT4InputPrice      = 0.03  // $0.03 per 1K input tokens
	GPT4OutputPrice     = 0.06  // $0.06 per 1K output tokens
	GPT4TurboInputPrice = 0.01  // $0.01 per 1K input tokens
	GPT4TurboOutputPrice = 0.03 // $0.03 per 1K output tokens

	// GPT-3.5 pricing per 1K tokens
	GPT35InputPrice  = 0.0005 // $0.0005 per 1K input tokens
	GPT35OutputPrice = 0.0015 // $0.0015 per 1K output tokens
)

// Anthropic pricing (as of 2024)
const (
	// Claude 3 Opus pricing per 1M tokens
	Claude3OpusInputPrice  = 15.0  // $15 per 1M input tokens
	Claude3OpusOutputPrice = 75.0  // $75 per 1M output tokens

	// Claude 3 Sonnet pricing per 1M tokens
	Claude3SonnetInputPrice  = 3.0  // $3 per 1M input tokens
	Claude3SonnetOutputPrice = 15.0 // $15 per 1M output tokens

	// Claude 3 Haiku pricing per 1M tokens
	Claude3HaikuInputPrice  = 0.25 // $0.25 per 1M input tokens
	Claude3HaikuOutputPrice = 1.25 // $1.25 per 1M output tokens
)

// CalculateOpenAICost calculates the cost of an OpenAI API call
func CalculateOpenAICost(model string, promptTokens, completionTokens int) float64 {
	var inputPrice, outputPrice float64

	switch model {
	case "gpt-4":
		inputPrice = GPT4InputPrice
		outputPrice = GPT4OutputPrice
	case "gpt-4-turbo", "gpt-4-turbo-preview":
		inputPrice = GPT4TurboInputPrice
		outputPrice = GPT4TurboOutputPrice
	case "gpt-3.5-turbo":
		inputPrice = GPT35InputPrice
		outputPrice = GPT35OutputPrice
	default:
		// Default to GPT-3.5 pricing for unknown models
		inputPrice = GPT35InputPrice
		outputPrice = GPT35OutputPrice
	}

	// Cost = (prompt_tokens / 1000) * input_price + (completion_tokens / 1000) * output_price
	inputCost := float64(promptTokens) / 1000.0 * inputPrice
	outputCost := float64(completionTokens) / 1000.0 * outputPrice

	return inputCost + outputCost
}

// CalculateAnthropicCost calculates the cost of an Anthropic API call
func CalculateAnthropicCost(model string, inputTokens, outputTokens int) float64 {
	var inputPrice, outputPrice float64

	switch model {
	case "claude-3-opus", "claude-3-opus-20240229":
		inputPrice = Claude3OpusInputPrice
		outputPrice = Claude3OpusOutputPrice
	case "claude-3-sonnet", "claude-3-sonnet-20240229":
		inputPrice = Claude3SonnetInputPrice
		outputPrice = Claude3SonnetOutputPrice
	case "claude-3-haiku", "claude-3-haiku-20240307":
		inputPrice = Claude3HaikuInputPrice
		outputPrice = Claude3HaikuOutputPrice
	default:
		// Default to Sonnet pricing for unknown Claude models
		inputPrice = Claude3SonnetInputPrice
		outputPrice = Claude3SonnetOutputPrice
	}

	// Cost = (input_tokens / 1_000_000) * input_price + (output_tokens / 1_000_000) * output_price
	inputCost := float64(inputTokens) / 1_000_000.0 * inputPrice
	outputCost := float64(outputTokens) / 1_000_000.0 * outputPrice

	return inputCost + outputCost
}
