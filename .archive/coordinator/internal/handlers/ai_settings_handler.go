package handlers

import (
	"net/http"

	"hyperion-coordinator/internal/models"
	"hyperion-coordinator/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// AISettingsHandler handles HTTP REST requests for AI settings (system prompt + subagents)
type AISettingsHandler struct {
	aiSettingsService *services.AISettingsService
	logger            *zap.Logger
}

// NewAISettingsHandler creates a new AI settings handler
func NewAISettingsHandler(aiSettingsService *services.AISettingsService, logger *zap.Logger) *AISettingsHandler {
	return &AISettingsHandler{
		aiSettingsService: aiSettingsService,
		logger:            logger,
	}
}

// extractUserContext extracts userId and companyId from context (set by auth middleware)
func (h *AISettingsHandler) extractUserContext(c *gin.Context) (string, string, error) {
	// Extract from context (set by optional auth middleware)
	userIDVal, exists := c.Get("userId")
	if !exists {
		return "", "", &gin.Error{
			Err:  http.ErrAbortHandler,
			Type: gin.ErrorTypePublic,
			Meta: "Missing userId in context",
		}
	}

	companyIDVal, exists := c.Get("companyId")
	if !exists {
		return "", "", &gin.Error{
			Err:  http.ErrAbortHandler,
			Type: gin.ErrorTypePublic,
			Meta: "Missing companyId in context",
		}
	}

	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return "", "", &gin.Error{
			Err:  http.ErrAbortHandler,
			Type: gin.ErrorTypePublic,
			Meta: "Invalid userId in context",
		}
	}

	companyID, ok := companyIDVal.(string)
	if !ok || companyID == "" {
		return "", "", &gin.Error{
			Err:  http.ErrAbortHandler,
			Type: gin.ErrorTypePublic,
			Meta: "Invalid companyId in context",
		}
	}

	return userID, companyID, nil
}

// GetSystemPrompt retrieves the system prompt for the authenticated user
// GET /api/v1/ai/system-prompt
func (h *AISettingsHandler) GetSystemPrompt(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	prompt, err := h.aiSettingsService.GetSystemPrompt(c.Request.Context(), userID, companyID)
	if err != nil {
		h.logger.Error("Failed to get system prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve system prompt"})
		return
	}

	c.JSON(http.StatusOK, models.GetSystemPromptResponse{
		SystemPrompt: prompt,
	})
}

// UpdateSystemPrompt updates the system prompt for the authenticated user
// PUT /api/v1/ai/system-prompt
func (h *AISettingsHandler) UpdateSystemPrompt(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	var req models.UpdateSystemPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	err = h.aiSettingsService.UpdateSystemPrompt(c.Request.Context(), userID, companyID, req.SystemPrompt)
	if err != nil {
		h.logger.Error("Failed to update system prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update system prompt"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "System prompt updated successfully",
	})
}

// ListSubagents lists all subagents for the authenticated user
// GET /api/v1/ai/subagents
func (h *AISettingsHandler) ListSubagents(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	subagents, err := h.aiSettingsService.ListSubagents(c.Request.Context(), userID, companyID)
	if err != nil {
		h.logger.Error("Failed to list subagents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list subagents"})
		return
	}

	c.JSON(http.StatusOK, models.ListSubagentsResponse{
		Subagents: subagents,
		Count:     len(subagents),
	})
}

// GetSubagent retrieves a specific subagent by ID
// GET /api/v1/ai/subagents/:id
func (h *AISettingsHandler) GetSubagent(c *gin.Context) {
	_, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	subagentIDStr := c.Param("id")
	subagentID, err := primitive.ObjectIDFromHex(subagentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subagent ID"})
		return
	}

	subagent, err := h.aiSettingsService.GetSubagent(c.Request.Context(), subagentID, companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subagent": subagent})
}

// CreateSubagent creates a new subagent
// POST /api/v1/ai/subagents
func (h *AISettingsHandler) CreateSubagent(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	var req models.CreateSubagentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	subagent, err := h.aiSettingsService.CreateSubagent(
		c.Request.Context(),
		userID,
		companyID,
		req.Name,
		req.Description,
		req.SystemPrompt,
	)
	if err != nil {
		h.logger.Error("Failed to create subagent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subagent"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"subagent": subagent})
}

// UpdateSubagent updates an existing subagent
// PUT /api/v1/ai/subagents/:id
func (h *AISettingsHandler) UpdateSubagent(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	subagentIDStr := c.Param("id")
	subagentID, err := primitive.ObjectIDFromHex(subagentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subagent ID"})
		return
	}

	var req models.UpdateSubagentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	subagent, err := h.aiSettingsService.UpdateSubagent(
		c.Request.Context(),
		subagentID,
		userID,
		companyID,
		req.Name,
		req.Description,
		req.SystemPrompt,
	)
	if err != nil {
		h.logger.Error("Failed to update subagent", zap.Error(err))
		if err.Error() == "unauthorized: subagent does not belong to user" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else if err.Error() == "subagent not found or access denied" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subagent"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"subagent": subagent})
}

// DeleteSubagent deletes a subagent
// DELETE /api/v1/ai/subagents/:id
func (h *AISettingsHandler) DeleteSubagent(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	subagentIDStr := c.Param("id")
	subagentID, err := primitive.ObjectIDFromHex(subagentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subagent ID"})
		return
	}

	err = h.aiSettingsService.DeleteSubagent(c.Request.Context(), subagentID, userID, companyID)
	if err != nil {
		h.logger.Error("Failed to delete subagent", zap.Error(err))
		if err.Error() == "unauthorized: subagent does not belong to user" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else if err.Error() == "subagent not found or access denied" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subagent deleted successfully",
	})
}

// ListClaudeAgents lists all available Claude agents from .claude/agents directory
// GET /api/v1/ai/claude-agents
func (h *AISettingsHandler) ListClaudeAgents(c *gin.Context) {
	// No auth required - read-only file system operation
	agents, err := h.aiSettingsService.ListClaudeAgents(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to list Claude agents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Claude agents"})
		return
	}

	c.JSON(http.StatusOK, models.ListClaudeAgentsResponse{
		Agents: agents,
		Count:  len(agents),
	})
}

// ImportClaudeAgents imports selected Claude agents as subagents
// POST /api/v1/ai/subagents/import-claude
func (h *AISettingsHandler) ImportClaudeAgents(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	var req models.ImportClaudeAgentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	imported, errors, err := h.aiSettingsService.ImportClaudeAgents(
		c.Request.Context(),
		userID,
		companyID,
		req.AgentNames,
	)
	if err != nil {
		h.logger.Error("Failed to import Claude agents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import Claude agents"})
		return
	}

	c.JSON(http.StatusOK, models.ImportClaudeAgentsResponse{
		Imported: imported,
		Errors:   errors,
		Success:  len(errors) == 0,
	})
}

// ImportAllClaudeAgents imports all available Claude agents as subagents
// POST /api/v1/ai/subagents/import-all-claude
func (h *AISettingsHandler) ImportAllClaudeAgents(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	// Get all available Claude agents
	agents, err := h.aiSettingsService.ListClaudeAgents(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to list Claude agents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Claude agents"})
		return
	}

	// Extract agent names
	agentNames := make([]string, len(agents))
	for i, agent := range agents {
		agentNames[i] = agent.Name
	}

	// Import all agents
	imported, errors, err := h.aiSettingsService.ImportClaudeAgents(
		c.Request.Context(),
		userID,
		companyID,
		agentNames,
	)
	if err != nil {
		h.logger.Error("Failed to import all Claude agents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import all Claude agents"})
		return
	}

	c.JSON(http.StatusOK, models.ImportClaudeAgentsResponse{
		Imported: imported,
		Errors:   errors,
		Success:  len(errors) == 0,
	})
}

// RegisterAISettingsRoutes registers all AI settings-related routes
func (h *AISettingsHandler) RegisterAISettingsRoutes(r *gin.RouterGroup) {
	// System prompt routes
	r.GET("/system-prompt", h.GetSystemPrompt)
	r.PUT("/system-prompt", h.UpdateSystemPrompt)

	// Subagent routes
	r.GET("/subagents", h.ListSubagents)
	r.GET("/subagents/:id", h.GetSubagent)
	r.POST("/subagents", h.CreateSubagent)
	r.PUT("/subagents/:id", h.UpdateSubagent)
	r.DELETE("/subagents/:id", h.DeleteSubagent)

	// Claude agent import routes
	r.GET("/claude-agents", h.ListClaudeAgents)
	r.POST("/subagents/import-claude", h.ImportClaudeAgents)
	r.POST("/subagents/import-all-claude", h.ImportAllClaudeAgents)
}
