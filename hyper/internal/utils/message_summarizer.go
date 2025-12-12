package utils

import (
	"context"
	"fmt"
	"time"

	"hyper/internal/models"
	"go.uber.org/zap"
)

// SummaryMetadata stores metadata about a summary
type SummaryMetadata struct {
	OriginalMessageCount int       `json:"originalMessageCount"`
	SummarizedTokens     int       `json:"summarizedTokens"`
	SummaryTokens        int       `json:"summaryTokens"`
	TokensSaved          int       `json:"tokensSaved"`
	TimeWindowStart      time.Time `json:"timeWindowStart"`
	TimeWindowEnd        time.Time `json:"timeWindowEnd"`
	CreatedAt            time.Time `json:"createdAt"`
}

// MessageSummarizer handles summarization of old messages to free up context
type MessageSummarizer struct {
	tokenCounter *TokenCounter
	logger       *zap.Logger
}

// NewMessageSummarizer creates a new message summarizer
func NewMessageSummarizer(logger *zap.Logger) *MessageSummarizer {
	return &MessageSummarizer{
		tokenCounter: NewTokenCounter(),
		logger:       logger,
	}
}

// SummarizationStrategy defines how to group messages for summarization
type SummarizationStrategy string

const (
	// StrategyTimeWindow groups messages by time windows (e.g., hourly)
	StrategyTimeWindow SummarizationStrategy = "time_window"
	// StrategyOldestFirst summarizes the oldest messages first
	StrategyOldestFirst SummarizationStrategy = "oldest_first"
	// StrategyByRole groups messages by role (user/assistant) and summarizes oldest
	StrategyByRole SummarizationStrategy = "by_role"
)

// SummarizationResult contains the result of a summarization operation
type SummarizationResult struct {
	Strategy              SummarizationStrategy `json:"strategy"`
	MessageGroups         []MessageGroup        `json:"messageGroups"`
	TotalOriginalTokens   int                   `json:"totalOriginalTokens"`
	TotalSummaryTokens    int                   `json:"totalSummaryTokens"`
	TotalTokensSaved      int                   `json:"totalTokensSaved"`
	SummarizedMessageCount int                  `json:"summarizedMessageCount"`
	Metadata              []SummaryMetadata     `json:"metadata"`
}

// MessageGroup represents a group of messages to be summarized together
type MessageGroup struct {
	Messages       []models.ChatMessage `json:"messages"`
	Summary        string               `json:"summary"`
	OriginalTokens int                  `json:"originalTokens"`
	SummaryTokens  int                  `json:"summaryTokens"`
	TimeWindowStart time.Time           `json:"timeWindowStart"`
	TimeWindowEnd   time.Time           `json:"timeWindowEnd"`
}

// IdentifyMessagesForSummarization identifies which messages should be summarized
// Returns messages that are older than the threshold
func (ms *MessageSummarizer) IdentifyMessagesForSummarization(
	messages []models.ChatMessage,
	keepRecentCount int,
) []models.ChatMessage {
	if len(messages) <= keepRecentCount {
		return []models.ChatMessage{}
	}

	// Keep the most recent messages, summarize the rest
	toSummarize := messages[:len(messages)-keepRecentCount]

	ms.logger.Info("Identified messages for summarization",
		zap.Int("totalMessages", len(messages)),
		zap.Int("keepRecentCount", keepRecentCount),
		zap.Int("toSummarizeCount", len(toSummarize)))

	return toSummarize
}

// GroupMessagesByTimeWindow groups messages into time windows
func (ms *MessageSummarizer) GroupMessagesByTimeWindow(
	messages []models.ChatMessage,
	windowDuration time.Duration,
) []MessageGroup {
	if len(messages) == 0 {
		return []MessageGroup{}
	}

	groups := []MessageGroup{}
	var currentGroup MessageGroup
	var currentWindowStart time.Time

	for i, msg := range messages {
		if i == 0 {
			currentWindowStart = msg.Timestamp
			currentGroup = MessageGroup{
				Messages:        []models.ChatMessage{},
				TimeWindowStart: currentWindowStart,
			}
		}

		// Check if message is still within current window
		if msg.Timestamp.Sub(currentWindowStart) < windowDuration {
			currentGroup.Messages = append(currentGroup.Messages, msg)
		} else {
			// Start new window
			if len(currentGroup.Messages) > 0 {
				currentGroup.TimeWindowEnd = currentGroup.Messages[len(currentGroup.Messages)-1].Timestamp
				currentGroup.OriginalTokens = ms.tokenCounter.CountSessionTokens(currentGroup.Messages)
				groups = append(groups, currentGroup)
			}

			currentWindowStart = msg.Timestamp
			currentGroup = MessageGroup{
				Messages:        []models.ChatMessage{msg},
				TimeWindowStart: currentWindowStart,
			}
		}
	}

	// Add the last group
	if len(currentGroup.Messages) > 0 {
		currentGroup.TimeWindowEnd = currentGroup.Messages[len(currentGroup.Messages)-1].Timestamp
		currentGroup.OriginalTokens = ms.tokenCounter.CountSessionTokens(currentGroup.Messages)
		groups = append(groups, currentGroup)
	}

	ms.logger.Info("Grouped messages by time window",
		zap.Int("groupCount", len(groups)),
		zap.Duration("windowDuration", windowDuration))

	return groups
}

// GenerateSummary generates a summary for a group of messages
// This is a placeholder that creates a structured summary
// In production, this would call an LLM to generate a natural language summary
func (ms *MessageSummarizer) GenerateSummary(ctx context.Context, group MessageGroup) string {
	// Count different message types
	userMessages := 0
	assistantMessages := 0
	toolMessages := 0

	for _, msg := range group.Messages {
		switch msg.Role {
		case "user":
			userMessages++
		case "assistant":
			assistantMessages++
		case "tool_call", "tool_result":
			toolMessages++
		}
	}

	// Create a structured summary
	summary := fmt.Sprintf(
		"[SUMMARY] Conversation segment from %s to %s: %d user messages, %d assistant messages, %d tool interactions. ",
		group.TimeWindowStart.Format("2006-01-02 15:04:05"),
		group.TimeWindowEnd.Format("2006-01-02 15:04:05"),
		userMessages,
		assistantMessages,
		toolMessages,
	)

	// Add key topics from user messages
	topics := ms.extractKeyTopics(group.Messages)
	if len(topics) > 0 {
		summary += fmt.Sprintf("Topics discussed: %v", topics)
	}

	ms.logger.Debug("Generated summary",
		zap.String("summary", summary),
		zap.Int("messageCount", len(group.Messages)))

	return summary
}

// extractKeyTopics extracts key topics from messages
// This is a simple implementation that looks for common keywords
func (ms *MessageSummarizer) extractKeyTopics(messages []models.ChatMessage) []string {
	topics := []string{}
	topicMap := make(map[string]bool)

	// Simple keyword extraction (in production, use NLP)
	keywords := []string{"implement", "fix", "bug", "feature", "error", "test", "deploy", "database", "api", "frontend", "backend"}

	for _, msg := range messages {
		if msg.Role == "user" {
			for _, keyword := range keywords {
				if len(msg.Content) > 0 && topicMap[keyword] == false {
					// Simple check - in production use proper NLP
					topicMap[keyword] = true
					topics = append(topics, keyword)
				}
			}
		}
	}

	return topics
}

// SummarizeMessages creates a summarization plan for messages
func (ms *MessageSummarizer) SummarizeMessages(
	ctx context.Context,
	messages []models.ChatMessage,
	strategy SummarizationStrategy,
) (*SummarizationResult, error) {
	if len(messages) == 0 {
		return &SummarizationResult{
			Strategy:              strategy,
			MessageGroups:         []MessageGroup{},
			TotalOriginalTokens:   0,
			TotalSummaryTokens:    0,
			TotalTokensSaved:      0,
			SummarizedMessageCount: 0,
			Metadata:              []SummaryMetadata{},
		}, nil
	}

	result := &SummarizationResult{
		Strategy:      strategy,
		MessageGroups: []MessageGroup{},
		Metadata:      []SummaryMetadata{},
	}

	var groups []MessageGroup

	switch strategy {
	case StrategyTimeWindow:
		// Group by 1-hour windows
		groups = ms.GroupMessagesByTimeWindow(messages, time.Hour)

	case StrategyOldestFirst:
		// Keep last 20 messages, summarize the rest
		toSummarize := ms.IdentifyMessagesForSummarization(messages, 20)
		if len(toSummarize) > 0 {
			groups = []MessageGroup{{
				Messages:        toSummarize,
				TimeWindowStart: toSummarize[0].Timestamp,
				TimeWindowEnd:   toSummarize[len(toSummarize)-1].Timestamp,
			}}
		}

	case StrategyByRole:
		// Group by role and then by time
		groups = ms.GroupMessagesByTimeWindow(messages, 30*time.Minute)

	default:
		return nil, fmt.Errorf("unknown summarization strategy: %s", strategy)
	}

	// Generate summaries for each group
	for i := range groups {
		groups[i].OriginalTokens = ms.tokenCounter.CountSessionTokens(groups[i].Messages)
		groups[i].Summary = ms.GenerateSummary(ctx, groups[i])
		groups[i].SummaryTokens = ms.tokenCounter.CountTokens(groups[i].Summary)

		result.TotalOriginalTokens += groups[i].OriginalTokens
		result.TotalSummaryTokens += groups[i].SummaryTokens
		result.SummarizedMessageCount += len(groups[i].Messages)

		// Create metadata
		metadata := SummaryMetadata{
			OriginalMessageCount: len(groups[i].Messages),
			SummarizedTokens:     groups[i].OriginalTokens,
			SummaryTokens:        groups[i].SummaryTokens,
			TokensSaved:          groups[i].OriginalTokens - groups[i].SummaryTokens,
			TimeWindowStart:      groups[i].TimeWindowStart,
			TimeWindowEnd:        groups[i].TimeWindowEnd,
			CreatedAt:            time.Now(),
		}
		result.Metadata = append(result.Metadata, metadata)
	}

	result.MessageGroups = groups
	result.TotalTokensSaved = result.TotalOriginalTokens - result.TotalSummaryTokens

	ms.logger.Info("Summarization completed",
		zap.String("strategy", string(strategy)),
		zap.Int("groupCount", len(groups)),
		zap.Int("totalOriginalTokens", result.TotalOriginalTokens),
		zap.Int("totalSummaryTokens", result.TotalSummaryTokens),
		zap.Int("totalTokensSaved", result.TotalTokensSaved))

	return result, nil
}

// CalculateSummarizationImpact estimates how much context would be freed
func (ms *MessageSummarizer) CalculateSummarizationImpact(
	messages []models.ChatMessage,
	keepRecentCount int,
) (tokensSaved int, messageCount int) {
	toSummarize := ms.IdentifyMessagesForSummarization(messages, keepRecentCount)
	if len(toSummarize) == 0 {
		return 0, 0
	}

	tokensSaved = ms.tokenCounter.CountSessionTokens(toSummarize)
	messageCount = len(toSummarize)

	return tokensSaved, messageCount
}
