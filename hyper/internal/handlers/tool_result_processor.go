package handlers

import (
	"fmt"
	"log"

	"hyper/internal/models"
)

// ToolResultProcessor handles tool result processing with summarization
type ToolResultProcessor struct {
	estimator *TokenEstimator
}

// NewToolResultProcessor creates a new processor
func NewToolResultProcessor() *ToolResultProcessor {
	return &ToolResultProcessor{
		estimator: NewTokenEstimator(),
	}
}

// ProcessedToolResult represents a processed tool result
type ProcessedToolResult struct {
	OriginalResult interface{}                // Full result
	DisplayResult  interface{}                // What to send to LLM
	Summary        string                     // Summary if used
	WasSummarized  bool                       // Whether summarization was applied
	TokenCount     int                        // Estimated tokens
	Reason         string                     // Why summarization was used
	Notification   *models.SystemNotification // Notification to send to frontend (nil if no summarization)
}

// ProcessToolResult processes a tool result with optional summarization
func (trp *ToolResultProcessor) ProcessToolResult(
	toolName string,
	toolResult interface{},
	remainingContextTokens int,
) ProcessedToolResult {

	// Estimate tokens for this result
	resultTokens := trp.estimator.EstimateTokens(toolResult)

	processed := ProcessedToolResult{
		OriginalResult: toolResult,
		DisplayResult:  toolResult,
		TokenCount:     resultTokens,
		WasSummarized:  false,
	}

	// Check if we should use summary
	if trp.estimator.ShouldUseSummary(resultTokens, remainingContextTokens) {
		summary := extractToolResultSummary(toolName, toolResult)
		processed.DisplayResult = summary
		processed.Summary = summary
		processed.WasSummarized = true
		processed.Reason = fmt.Sprintf(
			"Result too large (%d tokens, %d%% of remaining context)",
			resultTokens,
			(resultTokens * 100) / remainingContextTokens,
		)

		summaryTokens := trp.estimator.EstimateTokens(summary)
		reductionPercent := float64(resultTokens-summaryTokens) / float64(resultTokens) * 100

		log.Printf(
			"[ToolResult] Summarized %s: %s (original: %d tokens, summary: %d tokens)",
			toolName,
			processed.Reason,
			resultTokens,
			summaryTokens,
		)

		// Create notification for frontend display
		processed.Notification = &models.SystemNotification{
			Category: "summarization",
			Title:    "Search Results Summarized",
			Message:  fmt.Sprintf("Condensed %s results to save tokens", toolName),
			Severity: "info",
			Metadata: map[string]interface{}{
				"toolName":       toolName,
				"originalTokens": resultTokens,
				"finalTokens":    summaryTokens,
				"reduction":      fmt.Sprintf("%.0f%%", reductionPercent),
			},
		}
	}

	return processed
}

// ProcessMultipleToolResults processes multiple tool results and tracks total tokens
func (trp *ToolResultProcessor) ProcessMultipleToolResults(
	toolResults map[string]interface{},
	remainingContextTokens int,
) map[string]ProcessedToolResult {

	processed := make(map[string]ProcessedToolResult)
	currentRemaining := remainingContextTokens

	for toolName, result := range toolResults {
		pr := trp.ProcessToolResult(toolName, result, currentRemaining)
		processed[toolName] = pr

		// Update remaining context
		displayTokens := trp.estimator.EstimateTokens(pr.DisplayResult)
		currentRemaining -= displayTokens

		if currentRemaining < 0 {
			currentRemaining = 0
		}
	}

	return processed
}
