package aiservice

import (
	"log"
	"sort"
	"strings"
)

// message_manager.go
// Message management and context optimization utilities.
// This file contains functions for calculating context size, applying sliding windows,
// and priority-based message selection to optimize token usage in AI conversations.

// calculateContextSize computes total character count across all messages
func calculateContextSize(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content)
	}
	return total
}

// getToolResultPreview extracts the first maxChars of tool result content from messages
// Useful for debugging to see what tool results are being accumulated
func getToolResultPreview(messages []Message, maxChars int) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		// Look for tool result messages (role=system with "Tool '...' result:" pattern)
		if msg.Role == "system" && len(msg.Content) > 0 {
			if len(msg.Content) <= maxChars {
				return msg.Content
			}
			return msg.Content[:maxChars] + "..."
		}
	}
	return "(no tool results found)"
}

// MessagePriority represents the importance of a message for context preservation
type MessagePriority int

const (
	PriorityVeryLow  MessagePriority = 10  // list operations - trim first
	PriorityLow      MessagePriority = 25  // status checks, directory listings
	PriorityMedium   MessagePriority = 50  // read_file, write_file, coordinator ops
	PriorityHigh     MessagePriority = 75  // code_index_search, create_agent_task
	PriorityCritical MessagePriority = 100 // system prompts, user messages - NEVER trim
)

// getMessagePriority classifies messages by importance for context preservation
func getMessagePriority(msg Message) MessagePriority {
	// CRITICAL: Workflow enforcer error messages - contain guidance for next steps
	// These messages guide the LLM on what to do next and must NEVER be trimmed
	if strings.Contains(msg.Content, "BLOCKED") || strings.Contains(msg.Content, "NEXT:") {
		return PriorityCritical
	}

	// CRITICAL: Tool error messages - indicate failures that LLM must respond to
	if strings.Contains(msg.Content, "TOOL ERROR") || strings.Contains(msg.Content, "error:") {
		return PriorityCritical
	}

	// System prompts and user messages are critical - never trim
	if msg.Role == "system" {
		// Check if it's a tool result
		if strings.HasPrefix(msg.Content, "Tool '") {
			// Extract tool name from "Tool 'toolname' result: ..."
			if idx := strings.Index(msg.Content, "' result:"); idx > 6 {
				toolName := msg.Content[6:idx]
				return getToolResultPriority(toolName)
			}
			return PriorityMedium // Default for unparseable tool results
		}
		return PriorityCritical // System prompts are critical
	}
	if msg.Role == "user" {
		return PriorityCritical
	}

	// Assistant messages - check content for tool calls
	if msg.Role == "assistant" {
		// Assistant messages are medium priority by default
		return PriorityMedium
	}

	return PriorityLow
}

// getToolResultPriority assigns priority to tool results based on tool name
func getToolResultPriority(toolName string) MessagePriority {
	// HIGH PRIORITY: Critical for workflow - preserve strongly
	highPriorityTools := map[string]bool{
		"code_index_search":           true, // File discovery results - critical for agent tasks
		"create_agent_task":           true, // Agent task creation - core workflow
		"coordinator_create_human_task": true, // Human task creation - core workflow
		"execute_subagent":            true, // Subagent execution - core workflow
	}

	// MEDIUM PRIORITY: Important but can be re-retrieved if needed
	mediumPriorityTools := map[string]bool{
		"read_file":                   true,
		"write_file":                  true,
		"apply_patch":                 true,
		"coordinator_update_todo_status": true,
		"coordinator_update_task_status": true,
		"coordinator_get_agent_task":  true,
		"bash":                        true,
	}

	// LOW PRIORITY: Informational - can be trimmed
	lowPriorityTools := map[string]bool{
		"list_directory":              true,
		"code_index_status":           true,
		"coordinator_get_popular_collections": true,
		"coordinator_find_similar_tasks": true,
	}

	if highPriorityTools[toolName] {
		return PriorityHigh
	}
	if mediumPriorityTools[toolName] {
		return PriorityMedium
	}
	if lowPriorityTools[toolName] {
		return PriorityLow
	}

	// VERY LOW PRIORITY: List operations - trim first
	if strings.HasPrefix(toolName, "list_") || strings.HasPrefix(toolName, "coordinator_list_") {
		return PriorityVeryLow
	}

	return PriorityMedium // Default for unknown tools
}

// applySlidingWindow keeps only recent messages to prevent context accumulation
// ENHANCED: Uses priority-based trimming to preserve critical tool results
// Strategy: Keep high-priority messages (code_index_search) and trim low-priority first
func applySlidingWindow(messages []Message, maxMessages int) []Message {
	if len(messages) <= maxMessages {
		return messages // No need to trim
	}

	// Identify system prompt (if exists at index 0)
	hasSystemPrompt := len(messages) > 0 && messages[0].Role == "system" && !strings.HasPrefix(messages[0].Content, "Tool '")

	// Find original user message (first "user" role after system prompt)
	var systemMsg, userMsg *Message
	userMsgIdx := -1

	if hasSystemPrompt {
		systemMsg = &messages[0]
		// Find first user message after system
		for i := 1; i < len(messages); i++ {
			if messages[i].Role == "user" {
				userMsg = &messages[i]
				userMsgIdx = i
				break
			}
		}
	} else {
		// No system prompt - first message should be user
		if len(messages) > 0 && messages[0].Role == "user" {
			userMsg = &messages[0]
			userMsgIdx = 0
		}
	}

	// Calculate how many slots we need for critical messages
	reservedSlots := 0
	if systemMsg != nil {
		reservedSlots++
	}
	if userMsg != nil {
		reservedSlots++
	}

	// Get messages after the original user message (the conversation)
	var conversationMsgs []Message
	startIdx := userMsgIdx + 1
	if startIdx < len(messages) {
		conversationMsgs = messages[startIdx:]
	}

	// Calculate target size for conversation messages
	targetConversationSize := maxMessages - reservedSlots
	if targetConversationSize < 0 {
		targetConversationSize = 0
	}

	// SMART TRIMMING: If we need to reduce conversation messages, use priority-based selection
	var selectedConversationMsgs []Message
	if len(conversationMsgs) <= targetConversationSize {
		// No trimming needed for conversation
		selectedConversationMsgs = conversationMsgs
	} else {
		// Need to trim - use priority-based selection
		selectedConversationMsgs = selectMessagesByPriority(conversationMsgs, targetConversationSize)
	}

	// Build final message list
	result := make([]Message, 0, maxMessages)

	// Add system prompt if exists
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}

	// Add original user message if exists
	if userMsg != nil {
		result = append(result, *userMsg)
	}

	// Add selected conversation messages
	result = append(result, selectedConversationMsgs...)

	log.Printf("[Smart Sliding Window] Reduced from %d to %d messages (system: %v, user: %v, conversation: %d, preserved high-priority: %d)",
		len(messages), len(result), systemMsg != nil, userMsg != nil, len(selectedConversationMsgs),
		countHighPriorityMessages(selectedConversationMsgs))

	return result
}

// selectMessagesByPriority selects messages using priority-based algorithm
// Keeps: recent messages + high-priority messages (code_index_search, create_agent_task)
func selectMessagesByPriority(messages []Message, targetSize int) []Message {
	if len(messages) <= targetSize {
		return messages
	}

	// Classify messages by priority
	type indexedMessage struct {
		msg      Message
		index    int
		priority MessagePriority
	}

	indexed := make([]indexedMessage, len(messages))
	for i, msg := range messages {
		indexed[i] = indexedMessage{
			msg:      msg,
			index:    i,
			priority: getMessagePriority(msg),
		}
	}

	// Strategy: Always keep last N messages + highest priority older messages
	// This ensures recent context + critical historical context (code_index_search)

	// Keep last 40% of target as recent messages (maintains conversation flow)
	recentCount := targetSize * 4 / 10
	if recentCount < 2 {
		recentCount = 2 // Minimum recent messages
	}
	if recentCount > len(messages) {
		recentCount = len(messages)
	}

	// Reserve remaining slots for high-priority older messages
	prioritySlots := targetSize - recentCount

	// Always include last N messages (recent context)
	recentStartIdx := len(messages) - recentCount
	selected := make(map[int]bool)
	for i := recentStartIdx; i < len(messages); i++ {
		selected[i] = true
	}

	// Add high-priority older messages (not already selected)
	if prioritySlots > 0 {
		// Sort older messages by priority (descending)
		olderMessages := indexed[:recentStartIdx]
		sort.Slice(olderMessages, func(i, j int) bool {
			// First by priority (higher first), then by recency (later first)
			if olderMessages[i].priority != olderMessages[j].priority {
				return olderMessages[i].priority > olderMessages[j].priority
			}
			return olderMessages[i].index > olderMessages[j].index
		})

		// Add top priority messages up to available slots
		added := 0
		for _, im := range olderMessages {
			if added >= prioritySlots {
				break
			}
			// Only add high or critical priority messages
			if im.priority >= PriorityHigh {
				selected[im.index] = true
				added++
			}
		}
	}

	// Build result maintaining original order
	result := make([]Message, 0, targetSize)
	for i, msg := range messages {
		if selected[i] {
			result = append(result, msg)
		}
	}

	return result
}

// countHighPriorityMessages counts messages with HIGH or CRITICAL priority
func countHighPriorityMessages(messages []Message) int {
	count := 0
	for _, msg := range messages {
		if p := getMessagePriority(msg); p >= PriorityHigh {
			count++
		}
	}
	return count
}
