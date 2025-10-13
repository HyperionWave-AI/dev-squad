package handlers

import (
	"net/http"

	"hyper/internal/mcp/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SubagentHandler handles HTTP REST requests for subagent operations
type SubagentHandler struct {
	subchatStorage *storage.SubchatStorage
	logger         *zap.Logger
}

// NewSubagentHandler creates a new subagent handler
func NewSubagentHandler(subchatStorage *storage.SubchatStorage, logger *zap.Logger) *SubagentHandler {
	return &SubagentHandler{
		subchatStorage: subchatStorage,
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
