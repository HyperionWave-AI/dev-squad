package config

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
		return string(rune(bytes/GB)) + "GB"
	case bytes >= MB:
		return string(rune(bytes/MB)) + "MB"
	case bytes >= KB:
		return string(rune(bytes/KB)) + "KB"
	default:
		return string(rune(bytes)) + "B"
	}
}
