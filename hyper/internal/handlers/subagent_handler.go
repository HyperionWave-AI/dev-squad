package handlers

import (
	"fmt"
	"net/http"

	"hyper/internal/mcp/storage"
	"hyper/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SubagentHandler handles HTTP REST requests for subagent operations
type SubagentHandler struct {
	subchatStorage *storage.SubchatStorage
	chatService    *services.ChatService
	logger         *zap.Logger
}

// NewSubagentHandler creates a new subagent handler
func NewSubagentHandler(subchatStorage *storage.SubchatStorage, chatService *services.ChatService, logger *zap.Logger) *SubagentHandler {
	return &SubagentHandler{
		subchatStorage: subchatStorage,
		chatService:    chatService,
		logger:         logger,
	}
}

// DTOs for subagent API
type SubagentResponse struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
	Category    string   `json:"category"`
}

type ListSubagentsResponse struct {
	Subagents []SubagentResponse `json:"subagents"`
	Count     int                `json:"count"`
}

// ListSubagents retrieves all available subagents
// GET /api/v1/subagents
func (h *SubagentHandler) ListSubagents(c *gin.Context) {
	subagents, err := h.subchatStorage.ListSubagents()
	if err != nil {
		h.logger.Error("Failed to list subagents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subagents"})
		return
	}

	// Convert to response format (exclude systemPrompt which can be very large)
	responses := make([]SubagentResponse, len(subagents))
	for i, subagent := range subagents {
		responses[i] = SubagentResponse{
			Name:        subagent.Name,
			Description: subagent.Description,
			Tools:       subagent.Tools,
			Category:    subagent.Category,
		}
	}

	c.JSON(http.StatusOK, ListSubagentsResponse{
		Subagents: responses,
		Count:     len(responses),
	})
}

// GetSubagent retrieves a single subagent by name
// GET /api/v1/subagents/:name
func (h *SubagentHandler) GetSubagent(c *gin.Context) {
	name := c.Param("name")

	subagent, err := h.subchatStorage.GetSubagent(name)
	if err != nil {
		h.logger.Error("Failed to get subagent", zap.String("name", name), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Subagent not found"})
		return
	}

	// Return response without systemPrompt
	c.JSON(http.StatusOK, SubagentResponse{
		Name:        subagent.Name,
		Description: subagent.Description,
		Tools:       subagent.Tools,
		Category:    subagent.Category,
	})
}

// CreateAgentSession creates a new chat session with a specific subagent
// POST /api/v1/subagents/:name/sessions
func (h *SubagentHandler) CreateAgentSession(c *gin.Context) {
	// Extract user context from JWT middleware
	userIDVal, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing userId in context"})
		return
	}
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid userId in context"})
		return
	}

	companyIDVal, exists := c.Get("companyId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing companyId in context"})
		return
	}
	companyID, ok := companyIDVal.(string)
	if !ok || companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid companyId in context"})
		return
	}

	// Get agent name from URL parameter
	agentName := c.Param("name")
	if agentName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing agent name"})
		return
	}

	// Validate that the subagent exists
	subagent, err := h.subchatStorage.GetSubagent(agentName)
	if err != nil {
		h.logger.Error("Subagent not found", zap.String("agentName", agentName), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Subagent not found"})
		return
	}

	// Create the session with agent name in title
	title := fmt.Sprintf("Chat with %s", subagent.Name)

	// Create the session using the chat service
	session, err := h.chatService.CreateSession(c.Request.Context(), userID, companyID, title)
	if err != nil {
		h.logger.Error("Failed to create agent session",
			zap.String("agentName", agentName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// System subagents use string IDs (like "go-dev"), not ObjectIDs
	// Set the activeSubagentName field directly
	err = h.chatService.SetSessionSubagentName(c.Request.Context(), session.ID, &agentName, companyID)
	if err != nil {
		h.logger.Error("Failed to set session subagent name",
			zap.String("sessionId", session.ID.Hex()),
			zap.String("agentName", agentName),
			zap.Error(err))
		// Continue anyway - session was created successfully
	} else {
		// Update the session object to include the activeSubagentName for the response
		session.ActiveSubagentName = &agentName
		h.logger.Info("Set active system subagent on session",
			zap.String("sessionId", session.ID.Hex()),
			zap.String("agentName", agentName))
	}

	h.logger.Info("Created agent session successfully",
		zap.String("sessionId", session.ID.Hex()),
		zap.String("agentName", agentName),
		zap.String("userId", userID))

	c.JSON(http.StatusCreated, gin.H{
		"session":   session,
		"agentName": agentName,
	})
}
