package handlers

import (
	"fmt"
	"net/http"

	"hyper/internal/mcp/storage"
	"hyper/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// SubchatHandler handles HTTP REST requests for subchat operations
type SubchatHandler struct {
	subchatStorage *storage.SubchatStorage
	taskStorage    storage.TaskStorage
	chatService    *services.ChatService
	logger         *zap.Logger
}

// NewSubchatHandler creates a new subchat handler
func NewSubchatHandler(subchatStorage *storage.SubchatStorage, taskStorage storage.TaskStorage, chatService *services.ChatService, logger *zap.Logger) *SubchatHandler {
	return &SubchatHandler{
		subchatStorage: subchatStorage,
		taskStorage:    taskStorage,
		chatService:    chatService,
		logger:         logger,
	}
}

// extractUserContext extracts userId and companyId from context (set by auth middleware)
func (h *SubchatHandler) extractUserContext(c *gin.Context) (string, string, error) {
	// Extract from context (set by optional auth middleware)
	userIDVal, exists := c.Get("userId")
	if !exists {
		return "", "", fmt.Errorf("missing userId in context")
	}

	companyIDVal, exists := c.Get("companyId")
	if !exists {
		return "", "", fmt.Errorf("missing companyId in context")
	}

	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return "", "", fmt.Errorf("invalid userId in context")
	}

	companyID, ok := companyIDVal.(string)
	if !ok || companyID == "" {
		return "", "", fmt.Errorf("invalid companyId in context")
	}

	return userID, companyID, nil
}

// DTOs for subchat API
type CreateSubchatRequest struct {
	ParentChatID  string  `json:"parentChatId" binding:"required"`
	SubagentName  string  `json:"subagentName" binding:"required"`
	TaskID        *string `json:"taskId,omitempty"`
	TodoID        *string `json:"todoId,omitempty"`
}

type SubchatResponse struct {
	ID             string  `json:"id"`
	ParentChatID   string  `json:"parentChatId"`
	SessionID      *string `json:"sessionId,omitempty"` // Chat session ID for WebSocket streaming
	SubagentName   string  `json:"subagentName"`
	AssignedTaskID *string `json:"assignedTaskId,omitempty"`
	AssignedTodoID *string `json:"assignedTodoId,omitempty"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

type ListSubchatsResponse struct {
	Subchats []SubchatResponse `json:"subchats"`
	Count    int               `json:"count"`
}

// CreateSubchat creates a new subchat
// POST /api/v1/subchats
func (h *SubchatHandler) CreateSubchat(c *gin.Context) {
	var req CreateSubchatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Extract user context for ChatSession creation
	userID, companyID, err := h.extractUserContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	// Step 1: Create the subchat
	subchat, err := h.subchatStorage.CreateSubchat(req.ParentChatID, req.SubagentName, req.TaskID, req.TodoID)
	if err != nil {
		h.logger.Error("Failed to create subchat", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subchat: " + err.Error()})
		return
	}

	// Step 1.5: Create ChatSession with parentChatId for subchat tree visualization
	// Parse parent chat ID as ObjectID
	parentChatOID, err := primitive.ObjectIDFromHex(req.ParentChatID)
	if err != nil {
		h.logger.Error("Invalid parent chat ID", zap.String("parentChatId", req.ParentChatID), zap.Error(err))
		// Continue without creating ChatSession - subchat is already created
	} else {
		// Create ChatSession with parent chat ID
		sessionTitle := fmt.Sprintf("Subchat: %s", req.SubagentName)
		chatSession, err := h.chatService.CreateSessionWithParent(c.Request.Context(), userID, companyID, sessionTitle, &parentChatOID)
		if err != nil {
			h.logger.Error("Failed to create chat session for subchat",
				zap.String("subchatId", subchat.ID),
				zap.String("parentChatId", req.ParentChatID),
				zap.Error(err))
			// Continue without linking - subchat exists but won't have session
		} else {
			// Link the ChatSession to the Subchat
			sessionIDStr := chatSession.ID.Hex()
			err = h.subchatStorage.UpdateSubchatSessionID(subchat.ID, sessionIDStr)
			if err != nil {
				h.logger.Error("Failed to link chat session to subchat",
					zap.String("subchatId", subchat.ID),
					zap.String("sessionId", sessionIDStr),
					zap.Error(err))
			} else {
				// Update subchat object with session ID
				subchat.SessionID = &sessionIDStr

				h.logger.Info("Successfully created and linked chat session to subchat",
					zap.String("subchatId", subchat.ID),
					zap.String("sessionId", sessionIDStr),
					zap.String("parentChatId", req.ParentChatID),
					zap.String("subagentName", req.SubagentName))
			}
		}
	}

	// Step 2: Automatically create an agent task for this subchat
	// Create a human task first (required parent for agent task)
	humanTaskPrompt := fmt.Sprintf("Subchat work assigned to %s for parent chat %s", req.SubagentName, req.ParentChatID)
	humanTask, err := h.taskStorage.CreateHumanTask(humanTaskPrompt)
	if err != nil {
		h.logger.Error("Failed to create human task for subchat",
			zap.String("subchatId", subchat.ID),
			zap.Error(err))
		// Continue without agent task - subchat is already created
	} else {
		// Create agent task
		agentRole := fmt.Sprintf("Execute work in subchat %s", subchat.ID)
		todos := []storage.TodoItemInput{
			{
				Description: fmt.Sprintf("Complete assigned work in subchat (agent: %s)", req.SubagentName),
			},
		}

		agentTask, err := h.taskStorage.CreateAgentTask(
			humanTask.ID,
			req.SubagentName,
			agentRole,
			todos,
			fmt.Sprintf("This agent task is automatically assigned to subchat %s", subchat.ID),
			[]string{},  // filesModified
			[]string{},  // qdrantCollections
			"",          // priorWorkSummary
		)

		if err != nil {
			h.logger.Error("Failed to create agent task for subchat",
				zap.String("subchatId", subchat.ID),
				zap.String("humanTaskId", humanTask.ID),
				zap.Error(err))
		} else {
			// Step 3: Link the agent task to the subchat
			agentTaskID := agentTask.ID
			err = h.subchatStorage.UpdateSubchatAgentTask(subchat.ID, &agentTaskID)
			if err != nil {
				h.logger.Error("Failed to link agent task to subchat",
					zap.String("subchatId", subchat.ID),
					zap.String("agentTaskId", agentTask.ID),
					zap.Error(err))
			} else {
				// Update subchat object with agent task ID
				subchat.AssignedTaskID = &agentTaskID

				h.logger.Info("Successfully created and linked agent task to subchat",
					zap.String("subchatId", subchat.ID),
					zap.String("agentTaskId", agentTask.ID),
					zap.String("subagentName", req.SubagentName))
			}
		}
	}

	c.JSON(http.StatusCreated, h.toSubchatResponse(subchat))
}

// GetSubchat retrieves a single subchat by ID
// GET /api/v1/subchats/:id
func (h *SubchatHandler) GetSubchat(c *gin.Context) {
	id := c.Param("id")

	subchat, err := h.subchatStorage.GetSubchat(id)
	if err != nil {
		h.logger.Error("Failed to get subchat", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Subchat not found"})
		return
	}

	c.JSON(http.StatusOK, h.toSubchatResponse(subchat))
}

// GetSubchatsByParent retrieves all subchats for a parent chat
// GET /api/v1/chats/:chatId/subchats
func (h *SubchatHandler) GetSubchatsByParent(c *gin.Context) {
	parentChatID := c.Param("chatId")

	subchats, err := h.subchatStorage.GetSubchatsByParent(parentChatID)
	if err != nil {
		h.logger.Error("Failed to get subchats by parent", zap.String("parentChatId", parentChatID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subchats"})
		return
	}

	// Convert to response format
	responses := make([]SubchatResponse, len(subchats))
	for i, subchat := range subchats {
		responses[i] = h.toSubchatResponse(subchat)
	}

	c.JSON(http.StatusOK, ListSubchatsResponse{
		Subchats: responses,
		Count:    len(responses),
	})
}

// UpdateSubchatStatus updates the status of a subchat
// PUT /api/v1/subchats/:id/status
func (h *SubchatHandler) UpdateSubchatStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate status
	status := storage.SubchatStatus(req.Status)
	if status != storage.SubchatStatusActive &&
		status != storage.SubchatStatusCompleted &&
		status != storage.SubchatStatusFailed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be: active, completed, or failed"})
		return
	}

	err := h.subchatStorage.UpdateSubchatStatus(id, status)
	if err != nil {
		h.logger.Error("Failed to update subchat status", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subchat status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subchat status updated successfully",
	})
}

// Helper function to convert storage model to response DTO
func (h *SubchatHandler) toSubchatResponse(subchat *storage.Subchat) SubchatResponse {
	return SubchatResponse{
		ID:             subchat.ID,
		ParentChatID:   subchat.ParentChatID,
		SessionID:      subchat.SessionID, // Include session ID for WebSocket streaming
		SubagentName:   subchat.SubagentName,
		AssignedTaskID: subchat.AssignedTaskID,
		AssignedTodoID: subchat.AssignedTodoID,
		Status:         string(subchat.Status),
		CreatedAt:      subchat.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt:      subchat.UpdatedAt.Format("2006-01-02T15:04:05.000Z"),
	}
}
