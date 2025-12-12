package handlers

import (
	"fmt"
	"strings"
)

// ToolResultProcessed holds the processed tool result with metadata
type ToolResultProcessed struct {
	OutputStr      string // Processed output string (may be truncated or suppressed message)
	ShouldStream   bool   // Whether to stream to WebSocket client
	ShouldSaveFull bool   // Whether to save full content to database
	Tier           string // Size tier: "normal", "truncated", "suppressed", "error"
	OriginalSize   int    // Original size in bytes
	IsTruncated    bool   // Whether output was modified
}

// ProcessedToolResult holds the result after processing with summarization info
type ProcessedToolResult struct {
	OriginalResult interface{} // The original tool result
	ProcessedStr   string      // Processed/summarized output string
	WasSummarized  bool        // Whether the result was summarized
	TokenCount     int         // Estimated token count of the result
}

// ToolResultProcessor handles processing of tool results with token-aware summarization
type ToolResultProcessor struct {
	estimator *TokenEstimator
}

// NewToolResultProcessor creates a new tool result processor
func NewToolResultProcessor() *ToolResultProcessor {
	return &ToolResultProcessor{
		estimator: NewTokenEstimator(),
	}
}

// ProcessToolResult processes a single tool result, potentially summarizing if too large
func (p *ToolResultProcessor) ProcessToolResult(toolName string, result interface{}, remainingContextTokens int) *ProcessedToolResult {
	// Estimate token count for the result
	tokenCount := p.estimator.EstimateTokens(result)

	// Check if we should summarize
	wasSummarized := p.estimator.ShouldUseSummary(tokenCount, remainingContextTokens)

	var processedStr string
	if wasSummarized {
		// Generate a summary instead of the full result
		processedStr = extractToolResultSummary(toolName, result)
	} else {
		// Use the full result
		if str, ok := result.(string); ok {
			processedStr = str
		} else {
			processedStr = fmt.Sprintf("%v", result)
		}
	}

	return &ProcessedToolResult{
		OriginalResult: result,
		ProcessedStr:   processedStr,
		WasSummarized:  wasSummarized,
		TokenCount:     tokenCount,
	}
}

// ProcessMultipleToolResults processes multiple tool results
func (p *ToolResultProcessor) ProcessMultipleToolResults(toolResults map[string]interface{}, remainingContextTokens int) map[string]*ProcessedToolResult {
	results := make(map[string]*ProcessedToolResult)

	// Distribute remaining context tokens among tools
	tokensPerTool := remainingContextTokens
	if len(toolResults) > 0 {
		tokensPerTool = remainingContextTokens / len(toolResults)
	}

	for toolName, result := range toolResults {
		results[toolName] = p.ProcessToolResult(toolName, result, tokensPerTool)
	}

	return results
}

// extractToolResultSummary generates concise metadata for suppressed tool results
func extractToolResultSummary(toolName string, output interface{}) string {
	switch toolName {
	case "read_file", "file_read", "mcp__hyper__read_file":
		if str, ok := output.(string); ok {
			lines := strings.Count(str, "\n")
			words := len(strings.Fields(str))
			return fmt.Sprintf("**File Stats:** %d lines, ~%d words", lines, words)
		}

	case "grep", "search_code", "code_index_search", "mcp__hyper__grep":
		if str, ok := output.(string); ok {
			matches := strings.Count(str, "\n")
			return fmt.Sprintf("**Search Results:** %d matches found", matches)
		}

	case "bash", "execute_command", "mcp__hyper__bash":
		if str, ok := output.(string); ok {
			lines := strings.Count(str, "\n")
			return fmt.Sprintf("**Command Output:** %d lines", lines)
		}

	case "list_files", "glob", "mcp__hyper__glob":
		// Handle array output
		if arr, ok := output.([]interface{}); ok {
			return fmt.Sprintf("**Files Found:** %d items", len(arr))
		}
		// Handle string output (newline-separated)
		if str, ok := output.(string); ok {
			items := strings.Count(str, "\n")
			return fmt.Sprintf("**Files Found:** %d items", items)
		}
	}

	return "**Output:** Too large to display"
}
