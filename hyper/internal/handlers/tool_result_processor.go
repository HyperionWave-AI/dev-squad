package handlers

import (
	"fmt"
	"log"
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
	OriginalResult interface{} // Full result
	DisplayResult  interface{} // What to send to LLM
	Summary        string      // Summary if used
	WasSummarized  bool        // Whether summarization was applied
	TokenCount     int         // Estimated tokens
	Reason         string      // Why summarization was used
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

		log.Printf(
			"[ToolResult] Summarized %s: %s (original: %d tokens, summary: %d tokens)",
			toolName,
			processed.Reason,
			resultTokens,
			trp.estimator.EstimateTokens(summary),
		)
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
