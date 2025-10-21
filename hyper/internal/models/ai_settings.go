package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SystemPrompt represents the global system prompt configuration for AI (legacy, kept for backward compatibility)
type SystemPrompt struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    string             `json:"userId" bson:"userId"`
	CompanyID string             `json:"companyId" bson:"companyId"`
	Prompt    string             `json:"prompt" bson:"prompt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// SystemPromptVersion represents a versioned system prompt with history tracking
type SystemPromptVersion struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      string             `json:"userId" bson:"userId"`
	CompanyID   string             `json:"companyId" bson:"companyId"`
	Version     int                `json:"version" bson:"version"`                             // Auto-increment version number
	Prompt      string             `json:"prompt" bson:"prompt"`                               // The prompt content
	Description string             `json:"description,omitempty" bson:"description,omitempty"` // Optional version description
	IsActive    bool               `json:"isActive" bson:"isActive"`                           // Only one version can be active per user
	IsDefault   bool               `json:"isDefault" bson:"isDefault"`                         // Marks the system default prompt (read-only)
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy   string             `json:"createdBy" bson:"createdBy"` // UserID who created this version
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

// CreateSystemPromptVersionRequest represents the request to create a new system prompt version
type CreateSystemPromptVersionRequest struct {
	Prompt      string `json:"prompt" binding:"required"`
	Description string `json:"description"`
	Activate    bool   `json:"activate"` // If true, immediately set as active version
}

// ListSystemPromptVersionsResponse represents the response for listing system prompt versions
type ListSystemPromptVersionsResponse struct {
	Versions []SystemPromptVersion `json:"versions"`
	Count    int                   `json:"count"`
}

// GetDefaultSystemPromptResponse represents the response for getting the default system prompt
type GetDefaultSystemPromptResponse struct {
	Prompt string `json:"prompt"`
}
