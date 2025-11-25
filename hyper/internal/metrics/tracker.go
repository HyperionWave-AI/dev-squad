package metrics

import (
	"sync"
	"time"
)

// ToolExecutionTracker tracks tool execution metrics in real-time
type ToolExecutionTracker struct {
	mu              sync.RWMutex
	executions      []ToolMetrics
	maxHistorySize  int
	metricsRecorder *ToolMetricsTracker
}

// NewToolExecutionTracker creates a new tool execution tracker
func NewToolExecutionTracker(metricsRecorder *ToolMetricsTracker) *ToolExecutionTracker {
	return &ToolExecutionTracker{
		executions:      make([]ToolMetrics, 0, 1000),
		maxHistorySize:  10000, // Keep last 10k executions in memory
		metricsRecorder: metricsRecorder,
	}
}

// TrackExecution records a tool execution and stores it
func (t *ToolExecutionTracker) TrackExecution(toolName string, durationMs int64, status string, cacheHit bool, requestID, userID, sessionID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create metrics record
	metrics := ToolMetrics{
		ToolName:   toolName,
		Timestamp:  time.Now(),
		DurationMs: durationMs,
		Status:     status,
		CacheHit:   cacheHit,
		RequestID:  requestID,
		UserID:     userID,
		SessionID:  sessionID,
	}

	// Add to in-memory history
	t.executions = append(t.executions, metrics)

	// Trim history if it gets too large
	if len(t.executions) > t.maxHistorySize {
		t.executions = t.executions[len(t.executions)-t.maxHistorySize:]
	}

	// Also record to persistent store if available
	if t.metricsRecorder != nil {
		return t.metricsRecorder.RecordToolExecution(toolName, durationMs, status, cacheHit, requestID, userID, sessionID)
	}

	return nil
}

// TrackExecutionWithError records a tool execution with error details
func (t *ToolExecutionTracker) TrackExecutionWithError(toolName string, durationMs int64, status, errorMsg, requestID, userID, sessionID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create metrics record
	metrics := ToolMetrics{
		ToolName:   toolName,
		Timestamp:  time.Now(),
		DurationMs: durationMs,
		Status:     status,
		Error:      errorMsg,
		RequestID:  requestID,
		UserID:     userID,
		SessionID:  sessionID,
	}

	// Add to in-memory history
	t.executions = append(t.executions, metrics)

	// Trim history if it gets too large
	if len(t.executions) > t.maxHistorySize {
		t.executions = t.executions[len(t.executions)-t.maxHistorySize:]
	}

	// Also record to persistent store if available
	if t.metricsRecorder != nil {
		return t.metricsRecorder.RecordToolExecutionWithError(toolName, durationMs, status, errorMsg, requestID, userID, sessionID)
	}

	return nil
}

// GetRecentExecutions returns the most recent tool executions
func (t *ToolExecutionTracker) GetRecentExecutions(limit int) []ToolMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit > len(t.executions) {
		limit = len(t.executions)
	}

	// Return last N executions
	result := make([]ToolMetrics, limit)
	copy(result, t.executions[len(t.executions)-limit:])
	return result
}

// GetExecutionsByTool returns executions for a specific tool
func (t *ToolExecutionTracker) GetExecutionsByTool(toolName string) []ToolMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]ToolMetrics, 0)
	for _, exec := range t.executions {
		if exec.ToolName == toolName {
			result = append(result, exec)
		}
	}
	return result
}

// GetExecutionStats returns statistics for all executions
func (t *ToolExecutionTracker) GetExecutionStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.executions) == 0 {
		return map[string]interface{}{
			"total_executions": 0,
			"cache_hits":       0,
			"cache_misses":     0,
			"successful":       0,
			"failed":           0,
			"avg_duration":     0.0,
		}
	}

	var totalDuration int64
	cacheHits := 0
	cacheMisses := 0
	successful := 0
	failed := 0

	for _, exec := range t.executions {
		totalDuration += exec.DurationMs

		if exec.CacheHit {
			cacheHits++
		} else {
			cacheMisses++
		}

		if exec.Status == "success" {
			successful++
		} else if exec.Status == "error" {
			failed++
		}
	}

	return map[string]interface{}{
		"total_executions": len(t.executions),
		"cache_hits":       cacheHits,
		"cache_misses":     cacheMisses,
		"cache_hit_rate":   float64(cacheHits) / float64(len(t.executions)) * 100,
		"successful":       successful,
		"failed":           failed,
		"success_rate":     float64(successful) / float64(len(t.executions)) * 100,
		"avg_duration":     float64(totalDuration) / float64(len(t.executions)),
		"total_duration":   totalDuration,
	}
}

// ProviderMetricsTracker tracks provider API call metrics in real-time
type ProviderMetricsTracker struct {
	mu              sync.RWMutex
	calls           []ProviderMetrics
	maxHistorySize  int
	metricsRecorder *MetricsRecorder
}

// NewProviderMetricsTracker creates a new provider metrics tracker
func NewProviderMetricsTracker(metricsRecorder *MetricsRecorder) *ProviderMetricsTracker {
	return &ProviderMetricsTracker{
		calls:           make([]ProviderMetrics, 0, 1000),
		maxHistorySize:  10000, // Keep last 10k calls in memory
		metricsRecorder: metricsRecorder,
	}
}

// TrackCall records a provider API call
func (p *ProviderMetricsTracker) TrackCall(provider, model string, promptTokens, completionTokens int, durationMs int64, requestID, userID, sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create metrics record
	metrics := ProviderMetrics{
		Provider:         provider,
		Model:            model,
		Timestamp:        time.Now(),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		DurationMs:       durationMs,
		RequestID:        requestID,
		UserID:           userID,
		SessionID:        sessionID,
	}

	// Calculate costs if recorder available
	if p.metricsRecorder != nil {
		metrics.PromptCost = p.metricsRecorder.calculateTokenCost(provider, model, promptTokens, true)
		metrics.CompletionCost = p.metricsRecorder.calculateTokenCost(provider, model, completionTokens, false)
		metrics.TotalCost = metrics.PromptCost + metrics.CompletionCost
	}

	// Add to in-memory history
	p.calls = append(p.calls, metrics)

	// Trim history if it gets too large
	if len(p.calls) > p.maxHistorySize {
		p.calls = p.calls[len(p.calls)-p.maxHistorySize:]
	}

	// Also record to persistent store if available
	if p.metricsRecorder != nil {
		return p.metricsRecorder.RecordProviderMetrics(provider, model, promptTokens, completionTokens, durationMs, requestID, userID, sessionID)
	}

	return nil
}

// GetRecentCalls returns the most recent provider API calls
func (p *ProviderMetricsTracker) GetRecentCalls(limit int) []ProviderMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit > len(p.calls) {
		limit = len(p.calls)
	}

	// Return last N calls
	result := make([]ProviderMetrics, limit)
	copy(result, p.calls[len(p.calls)-limit:])
	return result
}

// GetCallsByModel returns calls for a specific model
func (p *ProviderMetricsTracker) GetCallsByModel(model string) []ProviderMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]ProviderMetrics, 0)
	for _, call := range p.calls {
		if call.Model == model {
			result = append(result, call)
		}
	}
	return result
}

// GetCallsByProvider returns calls for a specific provider
func (p *ProviderMetricsTracker) GetCallsByProvider(provider string) []ProviderMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]ProviderMetrics, 0)
	for _, call := range p.calls {
		if call.Provider == provider {
			result = append(result, call)
		}
	}
	return result
}

// GetCallStats returns statistics for all calls
func (p *ProviderMetricsTracker) GetCallStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.calls) == 0 {
		return map[string]interface{}{
			"total_calls":           0,
			"total_cost":            0.0,
			"total_tokens":          0,
			"avg_cost_per_call":     0.0,
			"avg_tokens_per_call":   0,
			"avg_duration":          0.0,
			"total_prompt_tokens":   0,
			"total_completion_tokens": 0,
		}
	}

	var totalCost float64
	var totalTokens int
	var totalPromptTokens int
	var totalCompletionTokens int
	var totalDuration int64

	for _, call := range p.calls {
		totalCost += call.TotalCost
		totalTokens += call.TotalTokens
		totalPromptTokens += call.PromptTokens
		totalCompletionTokens += call.CompletionTokens
		totalDuration += call.DurationMs
	}

	return map[string]interface{}{
		"total_calls":             len(p.calls),
		"total_cost":              totalCost,
		"total_tokens":            totalTokens,
		"total_prompt_tokens":     totalPromptTokens,
		"total_completion_tokens": totalCompletionTokens,
		"avg_cost_per_call":       totalCost / float64(len(p.calls)),
		"avg_tokens_per_call":     totalTokens / len(p.calls),
		"avg_duration":            float64(totalDuration) / float64(len(p.calls)),
		"total_duration":          totalDuration,
	}
}

// GetCostBreakdown returns cost breakdown by provider and model
func (p *ProviderMetricsTracker) GetCostBreakdown() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	providerCosts := make(map[string]float64)
	modelCosts := make(map[string]float64)

	for _, call := range p.calls {
		providerCosts[call.Provider] += call.TotalCost
		modelCosts[call.Model] += call.TotalCost
	}

	return map[string]interface{}{
		"by_provider": providerCosts,
		"by_model":    modelCosts,
	}
}
