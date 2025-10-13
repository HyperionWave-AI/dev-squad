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

// QueryKnowledge searches the knowledge base
// POST /api/v1/knowledge/query
func (h *KnowledgeHandler) QueryKnowledge(c *gin.Context) {
	var req struct {
		Collection string `json:"collection" binding:"required"`
		Query      string `json:"query" binding:"required"`
		Limit      int    `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Set default limit
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	// Query knowledge storage
	results, err := h.knowledgeStorage.Query(req.Collection, req.Query, limit)
	if err != nil {
		h.logger.Error("Failed to query knowledge",
			zap.String("collection", req.Collection),
			zap.String("query", req.Query),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query knowledge base"})
		return
	}

	// Transform QueryResult to response format
	entries := make([]gin.H, 0, len(results))
	for _, result := range results {
		entries = append(entries, gin.H{
			"id":         result.Entry.ID,
			"collection": req.Collection,
			"text":       result.Entry.Text,
			"metadata":   result.Entry.Metadata,
			"score":      result.Score,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
	})
}

// BrowseKnowledge lists knowledge entries without search
// GET /api/v1/knowledge/browse?collection=xxx&limit=10
func (h *KnowledgeHandler) BrowseKnowledge(c *gin.Context) {
	collection := c.Query("collection")

	// Parse limit
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 100 {
				limit = 100 // Max limit
			}
		}
	}

	var allEntries []*storage.KnowledgeEntry

	// If no collection specified, get from popular collections
	if collection == "" {
		// Get popular collections
		popular, err := h.knowledgeStorage.GetPopularCollections(5)
		if err != nil {
			h.logger.Error("Failed to get popular collections", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to browse knowledge base"})
			return
		}

		// Collect entries from popular collections
		perCollection := limit / len(popular)
		if perCollection < 1 {
			perCollection = 1
		}

		for _, col := range popular {
			entries, err := h.knowledgeStorage.ListKnowledge(col.Collection, perCollection)
			if err != nil {
				h.logger.Warn("Failed to list knowledge from collection",
					zap.String("collection", col.Collection),
					zap.Error(err))
				continue
			}
			allEntries = append(allEntries, entries...)
		}

		// Limit total results
		if len(allEntries) > limit {
			allEntries = allEntries[:limit]
		}
	} else {
		// List knowledge entries from specific collection
		entries, err := h.knowledgeStorage.ListKnowledge(collection, limit)
		if err != nil {
			h.logger.Error("Failed to list knowledge",
				zap.String("collection", collection),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to browse knowledge base"})
			return
		}
		allEntries = entries
	}

	// Transform to response format
	responseEntries := make([]gin.H, 0, len(allEntries))
	for _, entry := range allEntries {
		responseEntries = append(responseEntries, gin.H{
			"id":         entry.ID,
			"collection": entry.Collection,
			"text":       entry.Text,
			"metadata":   entry.Metadata,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": responseEntries,
	})
}

// GetAllCollections retrieves all collections with metadata
// GET /api/v1/knowledge/collections
func (h *KnowledgeHandler) GetAllCollections(c *gin.Context) {
	collections, err := h.knowledgeStorage.GetCollectionStatsWithMetadata()
	if err != nil {
		h.logger.Error("Failed to get all collections", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve collections"})
		return
	}

	// Transform to response format
	responseCollections := make([]gin.H, 0, len(collections))
	for _, col := range collections {
		responseCollections = append(responseCollections, gin.H{
			"name":     col.Name,
			"category": col.Category,
			"count":    col.Count,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"collections": responseCollections,
	})
}

// RegisterRoutes registers all knowledge-related routes
func (h *KnowledgeHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/popular-collections", h.GetPopularCollections)
	r.GET("/collections", h.GetAllCollections)
	r.GET("/browse", h.BrowseKnowledge)
	r.POST("/query", h.QueryKnowledge)
}
