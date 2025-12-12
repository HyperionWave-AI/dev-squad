package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"hyper/internal/models"
)

// CompactionSummarizer generates intelligent summaries of compacted messages using AI
type CompactionSummarizer struct {
	aiService AIServiceInterface
	logger    *zap.Logger
}

// NewCompactionSummarizer creates a new compaction summarizer
func NewCompactionSummarizer(aiService AIServiceInterface, logger *zap.Logger) *CompactionSummarizer {
	return &CompactionSummarizer{
		aiService: aiService,
		logger:    logger,
	}
}

// SummaryPrompt is the system prompt for generating compaction summaries
// This prompt guides the AI to create concise, complete summaries that preserve
// all critical information while reducing token count
const SummaryPrompt = `You are a conversation summarizer. Your task is to create a concise summary of the following conversation segment that will replace these messages in the context window.

REQUIREMENTS:
1. Preserve ALL key decisions, code changes, and technical details
2. Maintain tool call outcomes and their results
3. Keep file paths, function names, and important identifiers
4. Note any errors encountered and how they were resolved
5. Be concise but complete - this summary replaces the original messages
6. Use bullet points for clarity and scannability
7. Include code snippets only if critical to understanding the context

FORMAT:
- Start with "**Conversation Summary** (X messages):"
- Use bullet points for key events
- Include code snippets only if critical
- Maximum 500 words
- End with "---" separator

CONVERSATION TO SUMMARIZE:`

// GenerateSummary creates an AI-powered summary of messages
// It attempts to use the AI service to generate a high-quality summary,
// falling back to a basic summary if the AI service is unavailable
func (s *CompactionSummarizer) GenerateSummary(
	ctx context.Context,
	messages []models.ChatMessage,
) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	s.logger.Debug("Generating summary for compacted messages",
		zap.Int("messageCount", len(messages)))

	// Build conversation text for summarization
	conversationText := s.buildConversationText(messages)

	// Attempt AI-powered summary
	summary, err := s.generateAISummary(ctx, conversationText)
	if err != nil {
		s.logger.Warn("AI summary generation failed, using fallback",
			zap.Error(err),
			zap.Int("messageCount", len(messages)))
		// Fall back to simple summary
		return s.generateFallbackSummary(messages), nil
	}

	if summary == "" {
		s.logger.Debug("AI returned empty summary, using fallback")
		return s.generateFallbackSummary(messages), nil
	}

	s.logger.Info("✅ AI summary generated successfully",
		zap.Int("messageCount", len(messages)),
		zap.Int("summaryLength", len(summary)))

	return summary, nil
}

// buildConversationText converts messages to a readable conversation format
// for the AI summarizer to process
func (s *CompactionSummarizer) buildConversationText(messages []models.ChatMessage) string {
	var builder strings.Builder

	for _, msg := range messages {
		// Add role and content
		builder.WriteString(fmt.Sprintf("\n[%s]: %s", msg.Role, msg.Content))

		// Add tool call information if present
		if msg.ToolCall != nil {
			builder.WriteString(fmt.Sprintf("\n  Tool: %s", msg.ToolCall.Name))
			if len(msg.ToolCall.Args) > 0 {
				argsJSON, _ := json.Marshal(msg.ToolCall.Args)
				builder.WriteString(fmt.Sprintf("\n  Args: %s", string(argsJSON)))
			}
		}

		// Add tool result information if present
		if msg.ToolResult != nil {
			output := fmt.Sprintf("%v", msg.ToolResult.Output)
			// Truncate very large outputs for the summary prompt
			if len(output) > 500 {
				output = output[:500] + "..."
			}
			builder.WriteString(fmt.Sprintf("\n  Result: %s", output))
			if msg.ToolResult.Error != "" {
				builder.WriteString(fmt.Sprintf("\n  Error: %s", msg.ToolResult.Error))
			}
		}
	}

	return builder.String()
}

// generateAISummary calls the AI service to generate a summary
// Returns empty string if AI service is unavailable or returns empty response
func (s *CompactionSummarizer) generateAISummary(ctx context.Context, conversationText string) (string, error) {
	if s.aiService == nil {
		return "", fmt.Errorf("AI service not available")
	}

	// Create a simple message for the AI to summarize
	// We use a user message with the summary prompt and conversation text
	messages := []interface{}{
		map[string]interface{}{
			"role":    "system",
			"content": SummaryPrompt,
		},
		map[string]interface{}{
			"role":    "user",
			"content": conversationText,
		},
	}

	// Convert to aiservice.Message format
	// This is a simplified approach - in production, you might want to use
	// the full aiservice.Message struct with proper typing
	aiMessages := make([]interface{}, len(messages))
	for i, msg := range messages {
		aiMessages[i] = msg
	}

	s.logger.Debug("Calling AI service for summary generation",
		zap.Int("conversationLength", len(conversationText)))

	// For now, we'll use a fallback approach since we don't have direct
	// access to a simple "GenerateCompletion" method on AIServiceInterface
	// The AIServiceInterface is designed for streaming chat with tools,
	// not for simple text generation.
	//
	// In a production system, you would either:
	// 1. Add a GenerateCompletion method to AIServiceInterface
	// 2. Use the streaming interface and collect all tokens
	// 3. Call the underlying AI provider directly

	s.logger.Debug("AI service integration not yet implemented for summarization",
		zap.String("reason", "AIServiceInterface is designed for streaming chat with tools"))

	return "", fmt.Errorf("AI summarization not yet implemented")
}

// generateFallbackSummary creates a basic summary without AI
// This provides a reasonable fallback when AI service is unavailable
func (s *CompactionSummarizer) generateFallbackSummary(messages []models.ChatMessage) string {
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("**Conversation Summary** (%d messages):\n\n", len(messages)))

	// Count different message types
	userMessages := 0
	assistantMessages := 0
	toolCalls := 0
	toolResults := 0
	systemMessages := 0

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			userMessages++
		case "assistant":
			assistantMessages++
		case "tool_call":
			toolCalls++
		case "tool_result":
			toolResults++
		case "system":
			systemMessages++
		}
	}

	// Add statistics
	summary.WriteString("**Message Statistics:**\n")
	if userMessages > 0 {
		summary.WriteString(fmt.Sprintf("- %d user messages\n", userMessages))
	}
	if assistantMessages > 0 {
		summary.WriteString(fmt.Sprintf("- %d assistant responses\n", assistantMessages))
	}
	if toolCalls > 0 {
		summary.WriteString(fmt.Sprintf("- %d tool calls\n", toolCalls))
	}
	if toolResults > 0 {
		summary.WriteString(fmt.Sprintf("- %d tool results\n", toolResults))
	}
	if systemMessages > 0 {
		summary.WriteString(fmt.Sprintf("- %d system messages\n", systemMessages))
	}

	summary.WriteString("\n**Key Events:**\n")

	// Extract first user message as context
	for _, msg := range messages {
		if msg.Role == "user" && len(msg.Content) > 0 {
			content := msg.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			summary.WriteString(fmt.Sprintf("- Started with: \"%s\"\n", content))
			break
		}
	}

	// Extract tool calls and their outcomes
	toolCallCount := 0
	for i, msg := range messages {
		if msg.ToolCall != nil && toolCallCount < 3 { // Limit to first 3 tool calls
			summary.WriteString(fmt.Sprintf("- Tool call: %s\n", msg.ToolCall.Name))
			toolCallCount++

			// Look for corresponding result
			if i+1 < len(messages) && messages[i+1].ToolResult != nil {
				result := messages[i+1].ToolResult
				if result.Error != "" {
					summary.WriteString(fmt.Sprintf("  - Error: %s\n", result.Error))
				} else {
					summary.WriteString("  - Completed successfully\n")
				}
			}
		}
	}

	// Extract last assistant message as final context
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == "assistant" && len(msg.Content) > 0 {
			content := msg.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			summary.WriteString(fmt.Sprintf("- Concluded with: \"%s\"\n", content))
			break
		}
	}

	summary.WriteString("\n---")

	return summary.String()
}
