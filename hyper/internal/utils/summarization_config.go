package utils

// SummarizationConfig holds configuration for message summarization
type SummarizationConfig struct {
	// Enabled indicates if summarization is enabled
	Enabled bool
	// KeepRecentMinutes is the number of minutes of recent messages to keep
	KeepRecentMinutes int
	// SummarizeAfterMinutes is the age threshold for summarization
	SummarizeAfterMinutes int
	// MaxSummaryTokens is the maximum tokens for a summary
	MaxSummaryTokens int
	// TimeWindowMinutes is the duration for time-based grouping
	TimeWindowMinutes int
	// AutoSummarizeThreshold is the context percentage at which to auto-summarize
	AutoSummarizeThreshold float64
}

// DefaultSummarizationConfig returns the default summarization configuration
func DefaultSummarizationConfig() SummarizationConfig {
	return SummarizationConfig{
		Enabled:                true,
		KeepRecentMinutes:      30,      // Keep last 30 minutes of messages
		SummarizeAfterMinutes:  60,      // Summarize messages older than 1 hour
		MaxSummaryTokens:       500,     // Summaries should be concise
		TimeWindowMinutes:      60,      // Group by 1-hour windows
		AutoSummarizeThreshold: 80.0,    // Auto-summarize at 80% context usage
	}
}

// SummarizationPrompt returns the system prompt for message summarization
func SummarizationPrompt() string {
	return `You are an expert at creating concise, accurate summaries of conversations.

Your task is to summarize a group of messages while preserving all important context and decisions.

Guidelines:
1. Be concise but comprehensive - capture all key points
2. Preserve technical details and decisions made
3. Maintain the chronological flow of the conversation
4. Highlight any action items or next steps
5. Keep the summary under 500 tokens
6. Use clear, structured formatting
7. Include any important context about the discussion

Format your summary as:
- Key Topics: [list of main topics discussed]
- Decisions Made: [any decisions or conclusions]
- Action Items: [any tasks or follow-ups]
- Important Context: [any critical information to remember]

Now summarize the following messages:`
}

// SummarizationUserPrompt creates a user prompt for summarizing specific messages
func SummarizationUserPrompt(messages []string) string {
	prompt := "Please summarize these messages:\n\n"
	for i, msg := range messages {
		prompt += msg + "\n"
		if i < len(messages)-1 {
			prompt += "---\n"
		}
	}
	return prompt
}
