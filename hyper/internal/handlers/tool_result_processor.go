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
