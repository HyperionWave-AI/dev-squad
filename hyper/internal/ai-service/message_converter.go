package aiservice

import (
	"encoding/json"
	"fmt"
	"hyper/internal/models"
)

// ConvertToLangChainMessages converts MongoDB chat messages to ai-service format
// Maps models.ChatMessage (from MongoDB) to aiservice.Message (for LangChain)
// CRITICAL: Preserves tool call and tool result structured data for AI context continuity
func ConvertToLangChainMessages(dbMessages []models.ChatMessage) []Message {
	// Pre-allocate slice for efficiency
	langchainMsgs := make([]Message, 0, len(dbMessages))

	for _, dbMsg := range dbMessages {
		// Convert each database message to LangChain format
		langchainMsg := Message{
			Role:    dbMsg.Role,    // "user", "assistant", "system", "tool_call", "tool_result"
			Content: dbMsg.Content, // Message content
		}

		// CRITICAL: Preserve tool call structured data
		if dbMsg.ToolCall != nil {
			langchainMsg.ToolCall = &ToolCall{
				ID:   dbMsg.ToolCall.ID,
				Name: dbMsg.ToolCall.Name,
				Args: dbMsg.ToolCall.Args,
			}

			// Enhance content with tool call details for AI context
			// LangChain expects tool information in both structured format and content
			if argsJSON, err := json.Marshal(dbMsg.ToolCall.Args); err == nil {
				langchainMsg.Content = fmt.Sprintf("Tool call: %s (ID: %s)\nArguments: %s",
					dbMsg.ToolCall.Name, dbMsg.ToolCall.ID, string(argsJSON))
			}
		}

		// CRITICAL: Preserve tool result structured data
		if dbMsg.ToolResult != nil {
			langchainMsg.ToolResult = &ToolResult{
				ID:         dbMsg.ToolResult.ID,
				Name:       dbMsg.ToolResult.Name,
				Args:       nil, // Args not stored in ToolResultData, only in ToolCall
				Output:     dbMsg.ToolResult.Output,
				Error:      dbMsg.ToolResult.Error,
				DurationMs: dbMsg.ToolResult.DurationMs,
			}

			// Enhance content with tool result for AI context
			// Include both success and error cases
			if dbMsg.ToolResult.Error != "" {
				langchainMsg.Content = fmt.Sprintf("Tool result for %s (ID: %s) - ERROR: %s\nDuration: %dms",
					dbMsg.ToolResult.Name, dbMsg.ToolResult.ID, dbMsg.ToolResult.Error, dbMsg.ToolResult.DurationMs)
			} else if outputJSON, err := json.Marshal(dbMsg.ToolResult.Output); err == nil {
				langchainMsg.Content = fmt.Sprintf("Tool result for %s (ID: %s):\n%s\nDuration: %dms",
					dbMsg.ToolResult.Name, dbMsg.ToolResult.ID, string(outputJSON), dbMsg.ToolResult.DurationMs)
			}
		}

		langchainMsgs = append(langchainMsgs, langchainMsg)
	}

	return langchainMsgs
}
