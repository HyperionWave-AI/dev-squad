package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"hyper/internal/models"
	"hyper/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// isOptimisticMessageID checks if a message ID is an optimistic ID (frontend-generated)
// Optimistic IDs have the format "msg-{timestamp}" and haven't been persisted to the database yet
func isOptimisticMessageID(idStr string) bool {
	return strings.HasPrefix(idStr, "msg-")
}

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

	h.logger.Info("📋 ListUserSessions CALLED",
		zap.String("userId", userID),
		zap.String("companyId", companyID))

	sessions, err := h.chatService.GetUserSessions(c.Request.Context(), userID, companyID)
	if err != nil {
		h.logger.Error("❌ ListUserSessions FAILED", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
		return
	}

	h.logger.Info("✅ ListUserSessions SUCCESS",
		zap.String("userId", userID),
		zap.Int("sessionCount", len(sessions)))

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

	h.logger.Info("🔍 GetSession CALLED",
		zap.String("sessionId", sessionIDStr),
		zap.String("companyId", companyID))

	session, err := h.chatService.GetSession(c.Request.Context(), sessionID, companyID)
	if err != nil {
		h.logger.Error("❌ GetSession FAILED",
			zap.String("sessionId", sessionIDStr),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Handle pointer field for logging
	activeSubagentName := ""
	if session.ActiveSubagentName != nil {
		activeSubagentName = *session.ActiveSubagentName
	}
	h.logger.Info("✅ GetSession SUCCESS",
		zap.String("sessionId", sessionIDStr),
		zap.String("title", session.Title),
		zap.Bool("hasActiveSubagent", session.ActiveSubagentID != nil || (session.ActiveSubagentName != nil && *session.ActiveSubagentName != "")),
		zap.String("activeSubagentName", activeSubagentName),
		zap.Bool("errorPreventionMode", session.ErrorPreventionMode),
		zap.Bool("complexityAnalysisMode", session.ComplexityAnalysisMode),
		zap.Int("contextTokenCount", session.ContextTokenCount),
		zap.Float64("contextPercentage", session.ContextPercentage))

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

// getOrCreateDefaultSession finds or creates a default session for the user
func (h *ChatHandler) getOrCreateDefaultSession(c *gin.Context, userID, companyID string) (primitive.ObjectID, error) {
	// Try to find existing default session for this user
	sessions, err := h.chatService.GetUserSessions(c.Request.Context(), userID, companyID)
	if err != nil {
		return primitive.NilObjectID, err
	}

	// Look for a session titled "default-session"
	for _, session := range sessions {
		if session.Title == "default-session" {
			return session.ID, nil
		}
	}

	// No default session found, create one
	session, err := h.chatService.CreateSession(c.Request.Context(), userID, companyID, "default-session")
	if err != nil {
		return primitive.NilObjectID, err
	}

	h.logger.Info("Auto-created default session",
		zap.String("userId", userID),
		zap.String("sessionId", session.ID.Hex()))

	return session.ID, nil
}

// UpdateSession updates a chat session's title
// PUT /api/v1/chat/sessions/:id
func (h *ChatHandler) UpdateSession(c *gin.Context) {
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

	var req models.UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	updatedSession, err := h.chatService.UpdateSession(c.Request.Context(), sessionID, userID, companyID, req.Title)
	if err != nil {
		h.logger.Error("Failed to update session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": updatedSession})
}

// UpdateErrorPreventionMode updates the error prevention mode for a chat session
// PATCH /api/v1/chat/sessions/:id/error-prevention
func (h *ChatHandler) UpdateErrorPreventionMode(c *gin.Context) {
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
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	updatedSession, err := h.chatService.UpdateErrorPreventionMode(c.Request.Context(), sessionID, userID, companyID, req.Enabled)
	if err != nil {
		h.logger.Error("Failed to update error prevention mode", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":             true,
		"errorPreventionMode": updatedSession.ErrorPreventionMode,
		"session":             updatedSession,
	})
}

// UpdateComplexityAnalysisMode updates the complexity analysis mode for a chat session
// PATCH /api/v1/chat/sessions/:id/complexity-analysis
func (h *ChatHandler) UpdateComplexityAnalysisMode(c *gin.Context) {
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
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	updatedSession, err := h.chatService.UpdateComplexityAnalysisMode(c.Request.Context(), sessionID, userID, companyID, req.Enabled)
	if err != nil {
		h.logger.Error("Failed to update complexity analysis mode", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":               true,
		"complexityAnalysisMode": updatedSession.ComplexityAnalysisMode,
		"session":               updatedSession,
	})
}

// GetMessages retrieves messages for a session with pagination
// GET /api/v1/chat/sessions/:id/messages?limit=50&offset=0
func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	sessionIDStr := c.Param("id")

	h.logger.Info("📬 GetMessages HANDLER CALLED",
		zap.String("sessionId", sessionIDStr),
		zap.String("userId", userID),
		zap.String("companyId", companyID),
		zap.String("limitParam", c.Query("limit")),
		zap.String("offsetParam", c.Query("offset")))

	// Handle special "default-session" identifier
	var sessionID primitive.ObjectID
	if sessionIDStr == "default-session" {
		sessionID, err = h.getOrCreateDefaultSession(c, userID, companyID)
		if err != nil {
			h.logger.Error("❌ GetMessages FAILED - default session error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get default session"})
			return
		}
	} else {
		// Parse as MongoDB ObjectID
		sessionID, err = primitive.ObjectIDFromHex(sessionIDStr)
		if err != nil {
			h.logger.Error("❌ GetMessages FAILED - invalid session ID",
				zap.String("sessionId", sessionIDStr),
				zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format. Use a valid ObjectID hex string or 'default-session'"})
			return
		}
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

	h.logger.Info("📬 GetMessages calling service",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	response, err := h.chatService.GetMessages(c.Request.Context(), sessionID, companyID, limit, offset)
	if err != nil {
		h.logger.Error("❌ GetMessages HANDLER FAILED",
			zap.String("sessionId", sessionID.Hex()),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Build role breakdown for logging
	roleCount := make(map[string]int)
	for _, msg := range response.Messages {
		roleCount[msg.Role]++
	}

	h.logger.Info("✅ GetMessages HANDLER SUCCESS",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int64("totalInDB", response.Total),
		zap.Int("returnedCount", len(response.Messages)),
		zap.Bool("hasMore", response.HasMore),
		zap.Any("roleBreakdown", roleCount))

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

// GetContextStatus retrieves the current context usage for a session
// GET /api/v1/chat/sessions/:id/context-status
func (h *ChatHandler) GetContextStatus(c *gin.Context) {
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

	// Get context status
	usage, err := h.chatService.GetContextStatus(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Failed to get context status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get context status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contextStatus": usage,
		"session": gin.H{
			"id":                  session.ID,
			"title":               session.Title,
			"contextTokenCount":   session.ContextTokenCount,
			"contextPercentage":   session.ContextPercentage,
		},
	})
}

// ArchiveMessagesHandler archives messages to free up context
// POST /api/v1/chat/sessions/:id/archive
func (h *ChatHandler) ArchiveMessagesHandler(c *gin.Context) {
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

	var req struct {
		MessageIDs []string `json:"messageIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Filter out optimistic message IDs (those starting with "msg-")
	// These are temporary IDs created on the frontend and haven't been persisted yet
	validMessageIDs := []string{}
	skippedCount := 0
	for _, idStr := range req.MessageIDs {
		// Skip optimistic IDs - they don't exist in the database
		if !isOptimisticMessageID(idStr) {
			validMessageIDs = append(validMessageIDs, idStr)
		} else {
			skippedCount++
		}
	}

	// If no valid message IDs remain, return success (nothing to archive)
	if len(validMessageIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":         true,
			"message":         "No persisted messages to archive (all were temporary)",
			"archivedCount":   0,
			"skippedCount":    skippedCount,
			"contextStatus":   nil,
		})
		return
	}

	// Convert string IDs to ObjectIDs
	messageIDs := make([]primitive.ObjectID, len(validMessageIDs))
	for i, idStr := range validMessageIDs {
		id, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID format: " + idStr})
			return
		}
		messageIDs[i] = id
	}

	// Archive messages
	err = h.chatService.ArchiveMessages(c.Request.Context(), sessionID, messageIDs)
	if err != nil {
		h.logger.Error("Failed to archive messages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive messages"})
		return
	}

	// Get updated context status
	usage, err := h.chatService.GetContextStatus(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Warn("Failed to get updated context status", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"message":         "Messages archived successfully",
		"skippedCount":    skippedCount,
		"archivedCount":   len(messageIDs),
		"contextStatus":   usage,
	})
}

// SummarizeMessagesHandler summarizes old messages to free up context
// POST /api/v1/chat/sessions/:id/summarize
func (h *ChatHandler) SummarizeMessagesHandler(c *gin.Context) {
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

	var req struct {
		KeepRecentMinutes int `json:"keepRecentMinutes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Summarize messages
	summaryMessage, summarizedIDs, err := h.chatService.SummarizeOldMessages(c.Request.Context(), sessionID, req.KeepRecentMinutes)
	if err != nil {
		h.logger.Error("Failed to summarize messages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to summarize messages: " + err.Error()})
		return
	}

	// Get updated context status
	usage, err := h.chatService.GetContextStatus(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Warn("Failed to get updated context status", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"message":           "Messages summarized successfully",
		"summaryMessage":    summaryMessage,
		"summarizedCount":   len(summarizedIDs),
		"contextStatus":     usage,
	})
}

// RegisterChatRoutes registers all chat-related routes
func (h *ChatHandler) RegisterChatRoutes(r *gin.RouterGroup) {
	r.POST("/sessions", h.CreateSession)
	r.GET("/sessions", h.ListUserSessions)
	r.GET("/sessions/:id", h.GetSession)
	r.PUT("/sessions/:id", h.UpdateSession)
	r.DELETE("/sessions/:id", h.DeleteSession)
	r.GET("/sessions/:id/messages", h.GetMessages)
	r.PUT("/sessions/:id/subagent", h.SetSessionSubagent)
	r.GET("/sessions/:id/context-status", h.GetContextStatus)
	r.POST("/sessions/:id/archive", h.ArchiveMessagesHandler)
	r.POST("/sessions/:id/summarize", h.SummarizeMessagesHandler)
	r.PATCH("/sessions/:id/error-prevention", h.UpdateErrorPreventionMode)
	r.PATCH("/sessions/:id/complexity-analysis", h.UpdateComplexityAnalysisMode)
}
