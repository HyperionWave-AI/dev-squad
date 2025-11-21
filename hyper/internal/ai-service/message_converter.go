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
			// CRITICAL LOGGING: Check if ID is empty when retrieved from database
			if dbMsg.ToolResult.ID == "" {
				fmt.Printf("🚨 BUG DETECTED: ToolResult.ID is EMPTY when retrieved from MongoDB!\n")
				fmt.Printf("   Tool Name: %s\n", dbMsg.ToolResult.Name)
				fmt.Printf("   Message ID: %s\n", dbMsg.ID.Hex())
				fmt.Printf("   Message Role: %s\n", dbMsg.Role)
				fmt.Printf("   This will cause 'tool_call_id is missing' error in next AI request!\n")
			}

			langchainMsg.ToolResult = &ToolResult{
				ID:         dbMsg.ToolResult.ID,
				Name:       dbMsg.ToolResult.Name,
				Args:       nil, // Args not stored in ToolResultData, only in ToolCall
				Output:     dbMsg.ToolResult.Output,
				Error:      dbMsg.ToolResult.Error,
				DurationMs: dbMsg.ToolResult.DurationMs,
			}

			// Enhance content with tool result for AI context
			// Pass through output as-is without forcing JSON conversion
			if dbMsg.ToolResult.Error != "" {
				langchainMsg.Content = fmt.Sprintf("Tool result for %s (ID: %s) - ERROR: %s\nDuration: %dms",
					dbMsg.ToolResult.Name, dbMsg.ToolResult.ID, dbMsg.ToolResult.Error, dbMsg.ToolResult.DurationMs)
			} else {
				// Convert output to string representation
				var outputStr string
				switch v := dbMsg.ToolResult.Output.(type) {
				case string:
					outputStr = v
				default:
					// Only marshal to JSON if not already a string
					if outputJSON, err := json.Marshal(v); err == nil {
						outputStr = string(outputJSON)
					} else {
						outputStr = fmt.Sprintf("%v", v)
					}
				}

				langchainMsg.Content = fmt.Sprintf("Tool result for %s (ID: %s):\n%s\nDuration: %dms",
					dbMsg.ToolResult.Name, dbMsg.ToolResult.ID, outputStr, dbMsg.ToolResult.DurationMs)

				// DETAILED LOGGING: Show actual tool result data being sent to AI
				fmt.Printf("[DETAILED TOOL RESULT → AI] Tool: %s | ID: %s\n", dbMsg.ToolResult.Name, dbMsg.ToolResult.ID)
				fmt.Printf("[DETAILED TOOL RESULT → AI] Output Type: %T\n", dbMsg.ToolResult.Output)
				fmt.Printf("[DETAILED TOOL RESULT → AI] Output Length: %d bytes\n", len(outputStr))
				if len(outputStr) > 500 {
					fmt.Printf("[DETAILED TOOL RESULT → AI] Output Preview (first 500 chars): %s...\n", outputStr[:500])
				} else {
					fmt.Printf("[DETAILED TOOL RESULT → AI] Full Output: %s\n", outputStr)
				}
			}
		}

		langchainMsgs = append(langchainMsgs, langchainMsg)
	}

	return langchainMsgs
}
