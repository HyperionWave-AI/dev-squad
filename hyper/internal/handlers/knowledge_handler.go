package handlers

import (
	"net/http"
	"strconv"

	"hyper/internal/mcp/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// KnowledgeHandler handles HTTP REST requests for knowledge base operations
type KnowledgeHandler struct {
	knowledgeStorage storage.KnowledgeStorage
	logger           *zap.Logger
}

// NewKnowledgeHandler creates a new knowledge handler
func NewKnowledgeHandler(knowledgeStorage storage.KnowledgeStorage, logger *zap.Logger) *KnowledgeHandler {
	return &KnowledgeHandler{
		knowledgeStorage: knowledgeStorage,
		logger:           logger,
	}
}

// GetPopularCollections retrieves popular collections with entry counts
// GET /api/v1/knowledge/popular-collections?limit=20
func (h *KnowledgeHandler) GetPopularCollections(c *gin.Context) {
	// Parse limit parameter (default 20, max 100)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 100 {
				limit = 100 // Max limit
			}
		}
	}

	// Get popular collections from storage
	collections, err := h.knowledgeStorage.GetPopularCollections(limit)
	if err != nil {
		h.logger.Error("Failed to get popular collections", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve popular collections"})
		return
	}

	// Return collections (empty array if no results)
	c.JSON(http.StatusOK, gin.H{
		"collections": collections,
		"count":       len(collections),
	})
}

// RegisterRoutes registers all knowledge-related routes
func (h *KnowledgeHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/popular-collections", h.GetPopularCollections)
}
