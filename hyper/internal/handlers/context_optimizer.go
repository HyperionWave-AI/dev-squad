package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ContextWindowConfig holds configuration for context window optimization
type ContextWindowConfig struct {
	ModelContextWindow int // Total tokens available (e.g., 8000 for GPT-4, 100000 for Claude)
	SafetyMargin       int // Reserved buffer tokens (default: 1000)
	MinRemainingTokens int // Minimum tokens before fallback to summary (default: 500)
}

// TokenCounter provides token estimation utilities
type TokenCounter struct {
	config ContextWindowConfig
}

// NewTokenCounter creates a new token counter with default config
func NewTokenCounter(modelContextWindow int) *TokenCounter {
	return &TokenCounter{
		config: ContextWindowConfig{
			ModelContextWindow: modelContextWindow,
			SafetyMargin:       1000,
			MinRemainingTokens: 500,
		},
	}
}

// EstimateTokenCount estimates token count for any output type
// Uses heuristic: ~4 characters = 1 token (standard for most LLMs)
func (tc *TokenCounter) EstimateTokenCount(output interface{}) int {
	switch v := output.(type) {
	case string:
		// Heuristic: 4 characters ≈ 1 token
		return (len(v) + 3) / 4
	case []interface{}:
		// For arrays, estimate each element
		totalTokens := 0
		for _, item := range v {
			totalTokens += tc.EstimateTokenCount(item)
		}
		return totalTokens
	case map[string]interface{}:
		// For maps, estimate key-value pairs
		totalTokens := 0
		for k, val := range v {
			totalTokens += (len(k) + 3) / 4
			totalTokens += tc.EstimateTokenCount(val)
		}
		return totalTokens
	case []string:
		totalTokens := 0
		for _, s := range v {
			totalTokens += (len(s) + 3) / 4
		}
		return totalTokens
	case []map[string]interface{}:
		totalTokens := 0
		for _, m := range v {
			totalTokens += tc.EstimateTokenCount(m)
		}
		return totalTokens
	default:
		// Fallback: convert to JSON and estimate
		if jsonBytes, err := json.Marshal(v); err == nil {
			return (len(jsonBytes) + 3) / 4
		}
		return 0
	}
}

// ContextUsage tracks current context window usage
type ContextUsage struct {
	SystemPromptTokens int
	MessagesTokens     int
	ToolResultsTokens  int
}

// CalculateRemainingContext calculates available tokens for next tool result
func (tc *TokenCounter) CalculateRemainingContext(usage ContextUsage) int {
	usedTokens := usage.SystemPromptTokens + usage.MessagesTokens + usage.ToolResultsTokens
	availableTokens := tc.config.ModelContextWindow - usedTokens - tc.config.SafetyMargin
	
	if availableTokens < 0 {
		return 0
	}
	return availableTokens
}

// ShouldFallbackToSummary determines if result should be summarized instead of included
func (tc *TokenCounter) ShouldFallbackToSummary(resultTokens int, remainingTokens int) bool {
	// If result takes more than 50% of remaining context, use summary
	if resultTokens > remainingTokens/2 {
		return true
	}
	// If less than minimum remaining tokens after adding result, use summary
	if remainingTokens-resultTokens < tc.config.MinRemainingTokens {
		return true
	}
	return false
}

// TruncatedResult represents a tool result that may have been truncated
type TruncatedResult struct {
	Original      interface{}
	Truncated     interface{}
	TokensUsed    int
	WasTruncated  bool
	TruncationReason string
}

// TruncateStringResult truncates a string result intelligently
func (tc *TokenCounter) TruncateStringResult(content string, maxTokens int) TruncatedResult {
	estimatedTokens := tc.EstimateTokenCount(content)
	
	if estimatedTokens <= maxTokens {
		return TruncatedResult{
			Original:     content,
			Truncated:    content,
			TokensUsed:   estimatedTokens,
			WasTruncated: false,
		}
	}
	
	// Calculate max characters based on token limit
	maxChars := maxTokens * 4
	if maxChars < 100 {
		maxChars = 100 // Minimum to avoid useless truncation
	}
	
	truncated := content
	if len(content) > maxChars {
		truncated = content[:maxChars] + "\n... (truncated)"
	}
	
	return TruncatedResult{
		Original:         content,
		Truncated:        truncated,
		TokensUsed:       tc.EstimateTokenCount(truncated),
		WasTruncated:     true,
		TruncationReason: fmt.Sprintf("Exceeded token limit: %d > %d", estimatedTokens, maxTokens),
	}
}

// CodeSearchResult represents a single code search result
type CodeSearchResult struct {
	FilePath      string  `json:"file_path"`
	LineNumber    int     `json:"line_number"`
	Content       string  `json:"content"`
	RelevanceScore float64 `json:"relevance_score"`
}

// TruncateCodeSearchResults truncates code search results by relevance
func (tc *TokenCounter) TruncateCodeSearchResults(results []CodeSearchResult, maxTokens int) TruncatedResult {
	if len(results) == 0 {
		return TruncatedResult{
			Original:     results,
			Truncated:    results,
			TokensUsed:   0,
			WasTruncated: false,
		}
	}
	
	// Sort by relevance score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})
	
	// Keep adding results until we exceed token limit
	var truncated []CodeSearchResult
	currentTokens := 0
	
	for _, result := range results {
		resultTokens := tc.EstimateTokenCount(result)
		if currentTokens+resultTokens > maxTokens && len(truncated) > 0 {
			// Stop adding results if we'd exceed limit (but keep at least 1)
			break
		}
		truncated = append(truncated, result)
		currentTokens += resultTokens
	}
	
	wasTruncated := len(truncated) < len(results)
	reason := ""
	if wasTruncated {
		reason = fmt.Sprintf("Kept top %d of %d results by relevance", len(truncated), len(results))
	}
	
	return TruncatedResult{
		Original:         results,
		Truncated:        truncated,
		TokensUsed:       currentTokens,
		WasTruncated:     wasTruncated,
		TruncationReason: reason,
	}
}

// TruncateListResults truncates list results intelligently
func (tc *TokenCounter) TruncateListResults(items []string, maxTokens int) TruncatedResult {
	if len(items) == 0 {
		return TruncatedResult{
			Original:     items,
			Truncated:    items,
			TokensUsed:   0,
			WasTruncated: false,
		}
	}
	
	var truncated []string
	currentTokens := 0
	
	for _, item := range items {
		itemTokens := tc.EstimateTokenCount(item)
		if currentTokens+itemTokens > maxTokens && len(truncated) > 0 {
			break
		}
		truncated = append(truncated, item)
		currentTokens += itemTokens
	}
	
	wasTruncated := len(truncated) < len(items)
	reason := ""
	if wasTruncated {
		reason = fmt.Sprintf("Kept %d of %d items", len(truncated), len(items))
	}
	
	return TruncatedResult{
		Original:         items,
		Truncated:        truncated,
		TokensUsed:       currentTokens,
		WasTruncated:     wasTruncated,
		TruncationReason: reason,
	}
}

// ProcessToolResultWithOptimization processes a tool result with context optimization
func (tc *TokenCounter) ProcessToolResultWithOptimization(
	toolName string,
	result interface{},
	usage ContextUsage,
) (interface{}, bool, string) {
	remainingTokens := tc.CalculateRemainingContext(usage)
	resultTokens := tc.EstimateTokenCount(result)
	
	// Check if we should use summary instead
	if tc.ShouldFallbackToSummary(resultTokens, remainingTokens) {
		return result, true, fmt.Sprintf("Result too large (%d tokens), consider using summary", resultTokens)
	}
	
	// If result fits, return as-is
	if resultTokens <= remainingTokens {
		return result, false, ""
	}
	
	// Result needs truncation - apply tool-specific truncation
	switch toolName {
	case "code_index_search":
		if results, ok := result.([]CodeSearchResult); ok {
			truncated := tc.TruncateCodeSearchResults(results, remainingTokens)
			return truncated.Truncated, truncated.WasTruncated, truncated.TruncationReason
		}
	case "list_directory", "list_files":
		if items, ok := result.([]string); ok {
			truncated := tc.TruncateListResults(items, remainingTokens)
			return truncated.Truncated, truncated.WasTruncated, truncated.TruncationReason
		}
	case "read_file", "grep", "bash":
		if content, ok := result.(string); ok {
			truncated := tc.TruncateStringResult(content, remainingTokens)
			return truncated.Truncated, truncated.WasTruncated, truncated.TruncationReason
		}
	}
	
	// Generic truncation for unknown tools
	if content, ok := result.(string); ok {
		truncated := tc.TruncateStringResult(content, remainingTokens)
		return truncated.Truncated, truncated.WasTruncated, truncated.TruncationReason
	}
	
	return result, false, ""
}

// FormatContextReport generates a human-readable context usage report
func (tc *TokenCounter) FormatContextReport(usage ContextUsage) string {
	totalUsed := usage.SystemPromptTokens + usage.MessagesTokens + usage.ToolResultsTokens
	remaining := tc.CalculateRemainingContext(usage)
	percentUsed := float64(totalUsed) / float64(tc.config.ModelContextWindow) * 100
	
	report := fmt.Sprintf(`
Context Window Report:
  Total Capacity: %d tokens
  System Prompt: %d tokens
  Messages: %d tokens
  Tool Results: %d tokens
  Total Used: %d tokens (%.1f%%)
  Remaining: %d tokens
  Safety Margin: %d tokens
`, tc.config.ModelContextWindow, usage.SystemPromptTokens, usage.MessagesTokens,
		usage.ToolResultsTokens, totalUsed, percentUsed, remaining, tc.config.SafetyMargin)
	
	return strings.TrimSpace(report)
}

// ContextOptimizationMetrics tracks optimization decisions
type ContextOptimizationMetrics struct {
	TotalToolResults      int
	ResultsTruncated      int
	ResultsUsingSummary   int
	TokensSaved           int
	AverageCompressionRatio float64
}

// NewContextOptimizationMetrics creates a new metrics tracker
func NewContextOptimizationMetrics() *ContextOptimizationMetrics {
	return &ContextOptimizationMetrics{}
}

// RecordTruncation records a truncation event
func (m *ContextOptimizationMetrics) RecordTruncation(originalTokens, truncatedTokens int) {
	m.TotalToolResults++
	m.ResultsTruncated++
	m.TokensSaved += originalTokens - truncatedTokens
	
	if m.TotalToolResults > 0 {
		m.AverageCompressionRatio = float64(m.TokensSaved) / float64(m.TotalToolResults)
	}
}

// RecordSummary records a summary fallback event
func (m *ContextOptimizationMetrics) RecordSummary(originalTokens int) {
	m.TotalToolResults++
	m.ResultsUsingSummary++
	m.TokensSaved += originalTokens
	
	if m.TotalToolResults > 0 {
		m.AverageCompressionRatio = float64(m.TokensSaved) / float64(m.TotalToolResults)
	}
}

// RecordFullResult records a result that wasn't truncated
func (m *ContextOptimizationMetrics) RecordFullResult() {
	m.TotalToolResults++
}

// String returns a formatted metrics report
func (m *ContextOptimizationMetrics) String() string {
	return fmt.Sprintf(
		"Optimization Metrics: %d results processed, %d truncated, %d using summaries, %d tokens saved (avg compression: %.2f tokens/result)",
		m.TotalToolResults, m.ResultsTruncated, m.ResultsUsingSummary, m.TokensSaved, m.AverageCompressionRatio,
	)
}
