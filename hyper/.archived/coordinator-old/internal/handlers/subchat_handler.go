package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"hyperion-coordinator/internal/services"
)

// SubchatHandler handles subchat-related HTTP requests
type SubchatHandler struct {
	subchatService *services.SubchatService
}

// NewSubchatHandler creates a new subchat handler
func NewSubchatHandler(subchatService *services.SubchatService) *SubchatHandler {
	return &SubchatHandler{
		subchatService: subchatService,
	}
}

// CreateSubchatRequest represents the request to create a new subchat
type CreateSubchatRequest struct {
	ParentChatID   string `json:"parentChatId" binding:"required"`
	SubagentName   string `json:"subagentName" binding:"required"`
	Title          string `json:"title" binding:"required"`
	AssignedTaskID string `json:"assignedTaskId,omitempty"`
	AssignedTodoID string `json:"assignedTodoId,omitempty"`
}

// CreateSubchat handles POST /api/v1/subchats
func (h *SubchatHandler) CreateSubchat(c *gin.Context) {
	// Extract user identity from JWT context (dev mode injects mock values)
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user identity not found in context"})
		return
	}

	companyID, exists := c.Get("companyId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company identity not found in context"})
		return
	}

	var req CreateSubchatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create subchat
	subchat, err := h.subchatService.CreateSubchat(
		c.Request.Context(),
		userID.(string),
		companyID.(string),
		req.Title,
		req.ParentChatID,
		req.SubagentName,
		req.AssignedTaskID,
		req.AssignedTodoID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, subchat)
}

// GetSubchat handles GET /api/v1/subchats/:id
func (h *SubchatHandler) GetSubchat(c *gin.Context) {
	subchatID := c.Param("id")
	if subchatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subchat ID is required"})
		return
	}

	companyID, exists := c.Get("companyId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company identity not found in context"})
		return
	}

	subchat, err := h.subchatService.GetSubchatByID(c.Request.Context(), subchatID, companyID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subchat)
}

// GetSubchatsByParent handles GET /api/v1/chats/:parentId/subchats
func (h *SubchatHandler) GetSubchatsByParent(c *gin.Context) {
	parentID := c.Param("parentId")
	if parentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parent chat ID is required"})
		return
	}

	companyID, exists := c.Get("companyId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company identity not found in context"})
		return
	}

	subchats, err := h.subchatService.GetSubchatsByParent(c.Request.Context(), parentID, companyID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subchats": subchats,
		"count":    len(subchats),
	})
}

// RegisterSubchatRoutes registers all subchat routes
func RegisterSubchatRoutes(router *gin.RouterGroup, handler *SubchatHandler) {
	router.POST("/subchats", handler.CreateSubchat)
	router.GET("/subchats/:id", handler.GetSubchat)
	router.GET("/chats/:parentId/subchats", handler.GetSubchatsByParent)
}
