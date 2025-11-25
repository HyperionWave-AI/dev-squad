package aiservice

import (
	"context"
	"fmt"
	"strings"
)

// TaskSummarizer provides AI-powered task summarization
type TaskSummarizer struct {
	provider ChatProvider
	config   *AIConfig
}

// NewTaskSummarizer creates a new task summarizer
func NewTaskSummarizer(config *AIConfig) (*TaskSummarizer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	provider, err := NewChatProvider(config, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat provider: %w", err)
	}

	return &TaskSummarizer{
		provider: provider,
		config:   config,
	}, nil
}

// SummarizeTask generates a concise summary of a task (max 100 tokens)
// Returns the summary string or error if generation fails
func (s *TaskSummarizer) SummarizeTask(ctx context.Context, content string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("content cannot be empty")
	}

	// Truncate content if too long (keep first 2000 chars to avoid context limits)
	if len(content) > 2000 {
		content = content[:2000] + "..."
	}

	// Create summarization messages
	messages := []Message{
		{
			Role:    "system",
			Content: "You are a task summarization assistant. Summarize tasks in 1-2 concise sentences. Be direct and focus on the main goal.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Summarize this task in 1-2 sentences (max 100 tokens):\n\n%s", content),
		},
	}

	// Call AI provider - note: we use StreamChat and collect all tokens
	// The provider will respect the maxTokens setting from config
	outputChan, err := s.provider.StreamChat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("AI summarization failed: %w", err)
	}

	// Collect all tokens from the stream
	var sb strings.Builder
	for token := range outputChan {
		if strings.HasPrefix(token, "ERROR: ") {
			return "", fmt.Errorf("AI error: %s", strings.TrimPrefix(token, "ERROR: "))
		}
		sb.WriteString(token)
	}

	summary := strings.TrimSpace(sb.String())
	if summary == "" {
		return "", fmt.Errorf("AI returned empty summary")
	}

	return summary, nil
}

// SummarizeTaskWithFallback attempts to summarize, but returns truncated content on failure
// This ensures task creation is never blocked by AI failures
func (s *TaskSummarizer) SummarizeTaskWithFallback(ctx context.Context, content string, maxFallbackLength int) string {
	summary, err := s.SummarizeTask(ctx, content)
	if err != nil {
		// Fallback: use first N characters of content
		if maxFallbackLength <= 0 {
			maxFallbackLength = 200
		}
		if len(content) <= maxFallbackLength {
			return content
		}
		return content[:maxFallbackLength] + "..."
	}
	return summary
}
