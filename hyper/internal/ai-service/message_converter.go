package aiservice

import (
	"hyper/internal/models"
)

// ConvertToLangChainMessages converts MongoDB chat messages to ai-service format
// Maps models.ChatMessage (from MongoDB) to aiservice.Message (for LangChain)
func ConvertToLangChainMessages(dbMessages []models.ChatMessage) []Message {
	// Pre-allocate slice for efficiency
	langchainMsgs := make([]Message, 0, len(dbMessages))

	for _, dbMsg := range dbMessages {
		// Convert each database message to LangChain format
		langchainMsg := Message{
			Role:    dbMsg.Role,    // "user", "assistant", "system"
			Content: dbMsg.Content, // Message content
		}
		langchainMsgs = append(langchainMsgs, langchainMsg)
	}

	return langchainMsgs
}
