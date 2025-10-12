package handlers

import (
	"net/http"
	"strconv"

	"hyper/internal/models"
	"hyper/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// ChatHandler handles HTTP REST requests for chat sessions
type ChatHandler struct {
	chatService *services.ChatService
	logger      *zap.Logger
}

// NewChatHandler creates a new chat handler
func NewChatHandler(chatService *services.ChatService, logger *zap.Logger) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		logger:      logger,
	}
}

// extractUserContext extracts userId and companyId from context (set by auth middleware)
func (h *ChatHandler) extractUserContext(c *gin.Context) (string, string, error) {
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

// CreateSession creates a new chat session
// POST /api/v1/chat/sessions
func (h *ChatHandler) CreateSession(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	var req models.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	session, err := h.chatService.CreateSession(c.Request.Context(), userID, companyID, req.Title)
	if err != nil {
		h.logger.Error("Failed to create session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"session": session})
}

// ListUserSessions lists all chat sessions for the authenticated user
// GET /api/v1/chat/sessions
func (h *ChatHandler) ListUserSessions(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	sessions, err := h.chatService.GetUserSessions(c.Request.Context(), userID, companyID)
	if err != nil {
		h.logger.Error("Failed to list sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// GetSession retrieves a specific chat session
// GET /api/v1/chat/sessions/:id
func (h *ChatHandler) GetSession(c *gin.Context) {
	_, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := primitive.ObjectIDFromHex(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	session, err := h.chatService.GetSession(c.Request.Context(), sessionID, companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// DeleteSession deletes a chat session and all its messages
// DELETE /api/v1/chat/sessions/:id
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := primitive.ObjectIDFromHex(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	err = h.chatService.DeleteSession(c.Request.Context(), sessionID, userID, companyID)
	if err != nil {
		h.logger.Error("Failed to delete session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session deleted successfully",
	})
}

// GetMessages retrieves messages for a session with pagination
// GET /api/v1/chat/sessions/:id/messages?limit=50&offset=0
func (h *ChatHandler) GetMessages(c *gin.Context) {
	_, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := primitive.ObjectIDFromHex(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	// Parse pagination parameters
	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 100 {
				limit = 100 // Max limit
			}
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	response, err := h.chatService.GetMessages(c.Request.Context(), sessionID, companyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// SetSessionSubagent sets or clears the active subagent for a session
// PUT /api/v1/chat/sessions/:id/subagent
func (h *ChatHandler) SetSessionSubagent(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := primitive.ObjectIDFromHex(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	var req struct {
		SubagentID *string `json:"subagentId"` // null to clear, ObjectID hex string to set
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Verify session belongs to user
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID, companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found or access denied"})
		return
	}
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Convert subagent ID
	var subagentObjID *primitive.ObjectID
	if req.SubagentID != nil && *req.SubagentID != "" {
		id, err := primitive.ObjectIDFromHex(*req.SubagentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subagent ID"})
			return
		}
		subagentObjID = &id
	}

	// Update session
	err = h.chatService.SetSessionSubagent(c.Request.Context(), sessionID, subagentObjID, companyID)
	if err != nil {
		h.logger.Error("Failed to set session subagent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session subagent updated successfully",
	})
}

// RegisterChatRoutes registers all chat-related routes
func (h *ChatHandler) RegisterChatRoutes(r *gin.RouterGroup) {
	r.POST("/sessions", h.CreateSession)
	r.GET("/sessions", h.ListUserSessions)
	r.GET("/sessions/:id", h.GetSession)
	r.DELETE("/sessions/:id", h.DeleteSession)
	r.GET("/sessions/:id/messages", h.GetMessages)
	r.PUT("/sessions/:id/subagent", h.SetSessionSubagent)
}
