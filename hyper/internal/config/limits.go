package config

import "fmt"

// Message and content size limits for chat system
// These limits protect against DoS attacks, memory exhaustion, and database bloat
const (
	// MaxMessageBytes is the maximum size of a raw WebSocket message (JSON payload)
	// This is the first line of defense against oversized messages
	MaxMessageBytes = 1 * 1024 * 1024 // 1MB

	// MaxContentBytes is the maximum size of actual message content
	// Applied after JSON unmarshaling to validate the content field
	MaxContentBytes = 1 * 1024 * 1024 // 1MB

	// MaxToolResultBytes is the maximum size for tool execution results
	// Tool outputs can be larger than user messages (e.g., file contents, search results)
	MaxToolResultBytes = 10 * 1024 * 1024 // 10MB

	// MaxStreamBufferBytes is the maximum accumulated size for AI streaming responses
	// Prevents unbounded memory growth during long AI responses
	MaxStreamBufferBytes = 5 * 1024 * 1024 // 5MB

	// Tool result display tiers (for size-aware rendering)
	// Tier 1: Display fully without modification
	MaxToolResultNormalBytes = 50 * 1024 // 50KB

	// Tier 2: Display truncated with preview + expand option
	MaxToolResultTruncatedBytes = 500 * 1024 // 500KB

	// Tier 3: Suppress result, show helpful message only
	MaxToolResultSuppressedBytes = 10 * 1024 * 1024 // 10MB (same as MaxToolResultBytes)

	// Preview size when truncating tool results
	ToolResultPreviewBytes = 10 * 1024 // 10KB

	// Maximum size for summary metadata when suppressing results
	ToolResultSummaryMaxBytes = 1 * 1024 // 1KB
)

// FormatSizeLimit formats a byte limit into a human-readable string
func FormatSizeLimit(bytes int) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%dGB", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%dMB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%dKB", bytes/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// FormatSize formats a byte size into a human-readable string with decimal precision
func FormatSize(bytes int) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
