package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SystemPrompt represents the global system prompt configuration for AI
type SystemPrompt struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    string             `json:"userId" bson:"userId"`
	CompanyID string             `json:"companyId" bson:"companyId"`
	Prompt    string             `json:"prompt" bson:"prompt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Subagent represents a custom AI agent configuration
type Subagent struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       string             `json:"userId" bson:"userId"`
	CompanyID    string             `json:"companyId" bson:"companyId"`
	Name         string             `json:"name" bson:"name" binding:"required"`
	Description  string             `json:"description" bson:"description"`
	SystemPrompt string             `json:"systemPrompt" bson:"systemPrompt" binding:"required"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// UpdateSystemPromptRequest represents the request to update system prompt
type UpdateSystemPromptRequest struct {
	SystemPrompt string `json:"systemPrompt" binding:"required"`
}

// GetSystemPromptResponse represents the response for getting system prompt
type GetSystemPromptResponse struct {
	SystemPrompt string `json:"systemPrompt"`
}

// CreateSubagentRequest represents the request to create a new subagent
type CreateSubagentRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	SystemPrompt string `json:"systemPrompt" binding:"required"`
}

// UpdateSubagentRequest represents the request to update a subagent
type UpdateSubagentRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	SystemPrompt string `json:"systemPrompt" binding:"required"`
}

// ListSubagentsResponse represents the response for listing subagents
type ListSubagentsResponse struct {
	Subagents []Subagent `json:"subagents"`
	Count     int        `json:"count"`
}

// ClaudeAgent represents a Claude agent configuration parsed from .claude/agents/*.md
type ClaudeAgent struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Model        string `json:"model"`
	Color        string `json:"color"`
	SystemPrompt string `json:"systemPrompt"`
}

// ImportClaudeAgentsRequest represents the request to import Claude agents as subagents
type ImportClaudeAgentsRequest struct {
	AgentNames []string `json:"agentNames" binding:"required"`
}

// ImportClaudeAgentsResponse represents the response after importing Claude agents
type ImportClaudeAgentsResponse struct {
	Imported int      `json:"imported"`
	Errors   []string `json:"errors"`
	Success  bool     `json:"success"`
}

// ListClaudeAgentsResponse represents the response for listing Claude agents
type ListClaudeAgentsResponse struct {
	Agents []ClaudeAgent `json:"agents"`
	Count  int           `json:"count"`
}
