package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"hyper/internal/mcp/review"
	"hyper/internal/mcp/storage"
	"hyper/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// KnowledgeHandler handles HTTP REST requests for knowledge base operations
type KnowledgeHandler struct {
	knowledgeStorage   storage.KnowledgeStorage
	reviewOrchestrator *review.ReviewOrchestrator
	chatService        interface {
		CreateSession(ctx context.Context, userID, companyID, title string) (*models.ChatSession, error)
		SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*models.ChatMessage, error)
	}
	logger *zap.Logger
}

// NewKnowledgeHandler creates a new knowledge handler
func NewKnowledgeHandler(
	knowledgeStorage storage.KnowledgeStorage,
	reviewOrchestrator *review.ReviewOrchestrator,
	chatService interface {
		CreateSession(ctx context.Context, userID, companyID, title string) (*models.ChatSession, error)
		SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*models.ChatMessage, error)
	},
	logger *zap.Logger,
) *KnowledgeHandler {
	return &KnowledgeHandler{
		knowledgeStorage:   knowledgeStorage,
		reviewOrchestrator: reviewOrchestrator,
		chatService:        chatService,
		logger:             logger,
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
		Collection string  `json:"collection" binding:"required"`
		Query      string  `json:"query" binding:"required"`
		Limit      int     `json:"limit"`
		TaskId     *string `json:"taskId,omitempty"`
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

	// Query knowledge storage with optional taskId filter
	results, err := h.knowledgeStorage.Query(req.Collection, req.Query, limit, req.TaskId)
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
			"id":          col.ID,
			"name":        col.Name,
			"category":    col.Category,
			"count":       col.Count,
			"description": col.Description,
			"tags":        col.Tags,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"collections": responseCollections,
	})
}

// UpdateKnowledgeEntry updates an existing knowledge entry
// PUT /api/v1/knowledge/entries/:id
func (h *KnowledgeHandler) UpdateKnowledgeEntry(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entry ID is required"})
		return
	}

	var req struct {
		Text     string                 `json:"text" binding:"required"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Update entry in storage
	updatedEntry, err := h.knowledgeStorage.UpdateEntry(id, req.Text, req.Metadata)
	if err != nil {
		if err.Error() == "knowledge entry not found: "+id {
			c.JSON(http.StatusNotFound, gin.H{"error": "Knowledge entry not found"})
			return
		}
		h.logger.Error("Failed to update knowledge entry",
			zap.String("id", id),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update knowledge entry"})
		return
	}

	// Return updated entry
	c.JSON(http.StatusOK, gin.H{
		"entry": gin.H{
			"id":         updatedEntry.ID,
			"collection": updatedEntry.Collection,
			"text":       updatedEntry.Text,
			"metadata":   updatedEntry.Metadata,
			"createdAt":  updatedEntry.CreatedAt,
		},
	})
}

// DeleteKnowledgeEntry deletes a knowledge entry
// DELETE /api/v1/knowledge/entries/:id
func (h *KnowledgeHandler) DeleteKnowledgeEntry(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entry ID is required"})
		return
	}

	// Delete entry from storage
	err := h.knowledgeStorage.DeleteEntry(id)
	if err != nil {
		if err.Error() == "knowledge entry not found: "+id {
			c.JSON(http.StatusNotFound, gin.H{"error": "Knowledge entry not found"})
			return
		}
		h.logger.Error("Failed to delete knowledge entry",
			zap.String("id", id),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete knowledge entry"})
		return
	}

	// Return 204 No Content on successful deletion
	c.Status(http.StatusNoContent)
}

// UpdateCollectionMetadataHandler updates collection metadata (description, tags, category)
// PUT /api/v1/knowledge/collections/:name/metadata
func (h *KnowledgeHandler) UpdateCollectionMetadataHandler(c *gin.Context) {
	collectionName := c.Param("name")
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collection name is required"})
		return
	}

	var req struct {
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		Category    string   `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Update metadata in storage
	metadata, err := h.knowledgeStorage.UpdateCollectionMetadata(collectionName, req.Description, req.Tags, req.Category)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
			return
		}
		h.logger.Error("Failed to update collection metadata",
			zap.String("collection", collectionName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collection metadata"})
		return
	}

	// Return updated metadata
	c.JSON(http.StatusOK, gin.H{
		"metadata": metadata,
	})
}

// RenameCollectionHandler renames a collection
// POST /api/v1/knowledge/collections/:name/rename
func (h *KnowledgeHandler) RenameCollectionHandler(c *gin.Context) {
	oldName := c.Param("name")
	if oldName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collection name is required"})
		return
	}

	var req struct {
		NewName string `json:"newName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.NewName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New collection name cannot be empty"})
		return
	}

	// Rename collection
	count, err := h.knowledgeStorage.RenameCollection(oldName, req.NewName)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist: "+oldName) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
			return
		}
		if strings.Contains(err.Error(), "already exists: "+req.NewName) {
			c.JSON(http.StatusConflict, gin.H{"error": "New collection name already exists"})
			return
		}
		h.logger.Error("Failed to rename collection",
			zap.String("oldName", oldName),
			zap.String("newName", req.NewName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rename collection"})
		return
	}

	// Return success with count
	c.JSON(http.StatusOK, gin.H{
		"message":        "Collection renamed successfully",
		"oldName":        oldName,
		"newName":        req.NewName,
		"entriesUpdated": count,
	})
}

// VoteOnKnowledgeEntry records a user's vote on a knowledge entry
// POST /api/v1/knowledge/entries/:id/vote
func (h *KnowledgeHandler) VoteOnKnowledgeEntry(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entry ID is required"})
		return
	}

	var req struct {
		Vote   string `json:"vote" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Extract user ID from JWT (for now, use placeholder)
	// TODO: Extract from JWT context
	userID := "system"

	// Record the vote
	voteRecord, err := h.knowledgeStorage.VoteOnEntry(id, userID, req.Vote, req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Knowledge entry not found"})
			return
		}
		if strings.Contains(err.Error(), "must be") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Failed to vote on knowledge entry",
			zap.String("id", id),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record vote"})
		return
	}

	// Get vote summary
	summary, err := h.knowledgeStorage.GetEntryVotes(id, userID)
	if err != nil {
		h.logger.Error("Failed to get vote summary",
			zap.String("id", id),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get vote summary"})
		return
	}

	// Return vote record and summary
	c.JSON(http.StatusOK, gin.H{
		"voteRecorded": gin.H{
			"entryId":   voteRecord.EntryID,
			"userId":    voteRecord.UserID,
			"vote":      voteRecord.Vote,
			"reason":    voteRecord.Reason,
			"createdAt": voteRecord.CreatedAt,
			"updatedAt": voteRecord.UpdatedAt,
		},
		"summary": gin.H{
			"upvotes":   summary.Upvotes,
			"downvotes": summary.Downvotes,
			"netScore":  summary.NetScore,
			"userVote":  summary.UserVote,
		},
	})
}

// GetKnowledgeEntryVotes retrieves voting summary for a knowledge entry
// GET /api/v1/knowledge/entries/:id/votes
func (h *KnowledgeHandler) GetKnowledgeEntryVotes(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entry ID is required"})
		return
	}

	// Extract user ID from JWT (for now, use placeholder)
	// TODO: Extract from JWT context
	userID := "system"

	// Get vote summary
	summary, err := h.knowledgeStorage.GetEntryVotes(id, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Knowledge entry not found"})
			return
		}
		h.logger.Error("Failed to get vote summary",
			zap.String("id", id),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get vote summary"})
		return
	}

	// Return summary
	c.JSON(http.StatusOK, gin.H{
		"entryId":   id,
		"upvotes":   summary.Upvotes,
		"downvotes": summary.Downvotes,
		"netScore":  summary.NetScore,
		"userVote":  summary.UserVote,
	})
}

// BatchSyncVotes syncs all vote data to Qdrant for vote-weighted search
// POST /api/v1/knowledge/sync-votes?collection=xxx (collection is optional)
func (h *KnowledgeHandler) BatchSyncVotes(c *gin.Context) {
	// Optional collection filter
	collectionName := c.Query("collection")

	// Perform batch sync
	count, err := h.knowledgeStorage.BatchSyncVotesToQdrant(collectionName)
	if err != nil {
		// Check if this is a partial success (some entries synced, some failed)
		if count > 0 {
			h.logger.Warn("Batch vote sync completed with errors",
				zap.String("collection", collectionName),
				zap.Int("syncedCount", count),
				zap.Error(err))
			c.JSON(http.StatusOK, gin.H{
				"message":     "Batch sync completed with errors",
				"syncedCount": count,
				"errors":      err.Error(),
			})
			return
		}

		// Complete failure
		h.logger.Error("Failed to batch sync votes",
			zap.String("collection", collectionName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync vote data to Qdrant"})
		return
	}

	// Complete success
	c.JSON(http.StatusOK, gin.H{
		"message":     "Vote data synced successfully",
		"syncedCount": count,
	})
}

// CreateCollectionHandler creates a new collection
// POST /api/v1/knowledge/collections
func (h *KnowledgeHandler) CreateCollectionHandler(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Category    string   `json:"category" binding:"required"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collection name cannot be empty"})
		return
	}

	if req.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category cannot be empty"})
		return
	}

	// Initialize tags if nil
	if req.Tags == nil {
		req.Tags = []string{}
	}

	// Create collection in storage
	collection, err := h.knowledgeStorage.CreateCollection(req.Name, req.Category, req.Description, req.Tags)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "Collection already exists"})
			return
		}
		h.logger.Error("Failed to create collection",
			zap.String("name", req.Name),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create collection"})
		return
	}

	// Return created collection
	c.JSON(http.StatusCreated, gin.H{
		"collection": gin.H{
			"id":          collection.ID.Hex(),
			"name":        collection.Name,
			"qdrantName":  collection.QdrantName,
			"category":    collection.Category,
			"description": collection.Description,
			"tags":        collection.Tags,
			"entryCount":  collection.EntryCount,
			"createdAt":   collection.CreatedAt,
			"updatedAt":   collection.UpdatedAt,
		},
	})
}

// MigrateCollectionsHandler migrates old string-based collections to Collection objects
// POST /api/v1/knowledge/migrate
func (h *KnowledgeHandler) MigrateCollectionsHandler(c *gin.Context) {
	// Type assertion to get MongoKnowledgeStorage
	mongoStorage, ok := h.knowledgeStorage.(*storage.MongoKnowledgeStorage)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Migration only supported with MongoDB storage"})
		return
	}

	// Run migration
	h.logger.Info("Starting collection migration...")
	stats, err := mongoStorage.MigrateToCollectionObjects()
	if err != nil {
		h.logger.Error("Failed to migrate collections",
			zap.Error(err),
			zap.Any("stats", stats))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Migration failed",
			"details": err.Error(),
			"stats":   stats,
		})
		return
	}

	h.logger.Info("Collection migration completed successfully",
		zap.Any("stats", stats))

	c.JSON(http.StatusOK, gin.H{
		"message": "Migration completed successfully",
		"stats":   stats,
	})
}

// ReviewEntryHandler reviews a single knowledge entry
// POST /api/v1/knowledge/entries/:id/review
func (h *KnowledgeHandler) ReviewEntryHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entry ID is required"})
		return
	}

	// Check if review orchestrator is available
	if h.reviewOrchestrator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Review system not initialized"})
		return
	}

	// Parse request body (optional parameters)
	var req struct {
		Mode   string `json:"mode"`   // "automatic" or "interactive" (default: "automatic")
		DryRun bool   `json:"dryRun"` // default: false
	}

	// Ignore binding errors - parameters are optional
	_ = c.ShouldBindJSON(&req)

	// Determine review mode (default to automatic)
	mode := review.ReviewModeAutomatic
	if req.Mode == "interactive" {
		mode = review.ReviewModeInteractive
	}

	// Review the entry
	ctx := context.Background()
	result, err := h.reviewOrchestrator.ReviewEntry(ctx, id, mode, req.DryRun)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Knowledge entry not found"})
			return
		}
		h.logger.Error("Failed to review knowledge entry",
			zap.String("id", id),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to review knowledge entry"})
		return
	}

	// Transform broken references for UI
	brokenRefs := []gin.H{}
	for _, ref := range result.BrokenReferences {
		brokenRefs = append(brokenRefs, gin.H{
			"type":  string(ref.Type),
			"value": ref.Value,
			"error": ref.ErrorMessage,
		})
	}

	// Transform actions for UI
	actions := []gin.H{}
	for _, action := range result.ActionsTaken {
		actions = append(actions, gin.H{
			"type":        action,
			"description": action,
			"applied":     true,
		})
	}
	for _, action := range result.SuggestedActions {
		actions = append(actions, gin.H{
			"type":        action,
			"description": action,
			"applied":     false,
		})
	}

	// Return review result in UI format
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"entryId": result.EntryID,
		"scores": gin.H{
			"alignment":  result.AlignmentScore,
			"freshness":  result.FreshnessScore,
			"verbosity":  result.VerbosityScore,
			"uniqueness": result.UniquenessScore,
			"health":     result.HealthScore,
		},
		"verification": gin.H{
			"totalReferences":  result.TotalReferences,
			"validReferences":  result.ValidReferences,
			"brokenReferences": brokenRefs,
		},
		"actions": actions,
	})
}

// ReviewCollectionHandler reviews all entries in a collection
// POST /api/v1/knowledge/collections/:name/review
func (h *KnowledgeHandler) ReviewCollectionHandler(c *gin.Context) {
	collectionName := c.Param("name")
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collection name is required"})
		return
	}

	// Check if review orchestrator is available
	if h.reviewOrchestrator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Review system not initialized"})
		return
	}

	// Parse optional query parameters
	var req struct {
		MinHealthScore float64 `json:"minHealthScore"`
		Limit          int     `json:"limit"`
	}

	// Try to bind JSON body, but don't fail if empty
	_ = c.ShouldBindJSON(&req)

	// Set defaults
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}

	// Review the collection
	ctx := context.Background()
	results, err := h.reviewOrchestrator.ReviewCollection(ctx, collectionName, req.MinHealthScore, limit)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
			return
		}
		h.logger.Error("Failed to review collection",
			zap.String("collection", collectionName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to review collection"})
		return
	}

	// Calculate summary statistics
	totalEntries := len(results)
	totalHealth := 0.0
	lowHealthCount := 0

	entrySummaries := make([]gin.H, 0, totalEntries)
	for _, result := range results {
		totalHealth += result.HealthScore
		if result.HealthScore < 40 {
			lowHealthCount++
		}

		entrySummaries = append(entrySummaries, gin.H{
			"entryId":        result.EntryID,
			"healthScore":    result.HealthScore,
			"alignmentScore": result.AlignmentScore,
			"wordCount":      result.ActualWordCount,
			"actionsTaken":   len(result.ActionsTaken),
		})
	}

	avgHealth := 0.0
	if totalEntries > 0 {
		avgHealth = totalHealth / float64(totalEntries)
	}

	// Return collection review summary
	timestamp := ""
	if totalEntries > 0 {
		timestamp = results[0].ReviewedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	c.JSON(http.StatusOK, gin.H{
		"collection":     collectionName,
		"totalEntries":   totalEntries,
		"avgHealthScore": avgHealth,
		"lowHealthCount": lowHealthCount,
		"entries":        entrySummaries,
		"timestamp":      timestamp,
	})
}

// CompactEntryHandler compacts a knowledge entry using LLM
// POST /api/v1/knowledge/entries/:id/compact
func (h *KnowledgeHandler) CompactEntryHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entry ID is required"})
		return
	}

	// Check if review orchestrator is available
	if h.reviewOrchestrator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Review system not initialized"})
		return
	}

	// Parse request body
	var req struct {
		DryRun          bool `json:"dryRun"`
		TargetWordCount int  `json:"targetWordCount"`
	}

	// Default to dry-run for safety
	req.DryRun = true

	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body provided, use defaults
		h.logger.Debug("No request body provided, using defaults",
			zap.String("id", id))
	}

	// Compact the entry
	ctx := context.Background()
	result, err := h.reviewOrchestrator.CompactEntry(ctx, id, req.TargetWordCount, req.DryRun)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Knowledge entry not found"})
			return
		}
		h.logger.Error("Failed to compact knowledge entry",
			zap.String("id", id),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compact knowledge entry"})
		return
	}

	// Calculate compression ratio
	compressionRatio := 0.0
	if result.OriginalWords > 0 {
		compressionRatio = float64(result.CompactedWords) / float64(result.OriginalWords)
	}

	// Count preserved elements by type
	// For now, we'll provide counts based on PreservedAll flag
	// In the future, this could be enhanced to count specific elements
	filePathsCount := 0
	functionNamesCount := 0

	if result.PreservedAll {
		// If all elements preserved, show positive counts
		// Extract rough counts from the missing elements check
		for _, missing := range result.MissingElements {
			if strings.Contains(missing, "file path") {
				filePathsCount++
			} else if strings.Contains(missing, "function") {
				functionNamesCount++
			}
		}
		// If nothing missing, default to showing some were preserved
		if len(result.MissingElements) == 0 {
			filePathsCount = 1 // At least one file path likely preserved
			functionNamesCount = 1 // At least one function likely preserved
		}
	}

	// Return compaction result in UI format
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"original": gin.H{
			"text":      result.OriginalText,
			"wordCount": result.OriginalWords,
		},
		"compacted": gin.H{
			"text":      result.CompactedText,
			"wordCount": result.CompactedWords,
		},
		"compressionRatio": compressionRatio,
		"preserved": gin.H{
			"filePaths":     filePathsCount,
			"functionNames": functionNamesCount,
		},
	})
}

// DeleteCollectionHandler deletes an entire collection including all entries and Qdrant data
// DELETE /api/v1/knowledge/collections/:id
func (h *KnowledgeHandler) DeleteCollectionHandler(c *gin.Context) {
	collectionID := c.Param("id")
	if collectionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collection ID is required"})
		return
	}

	// Delete collection
	collectionName, entriesDeleted, err := h.knowledgeStorage.DeleteCollection(collectionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
			return
		}
		h.logger.Error("Failed to delete collection",
			zap.String("id", collectionID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete collection"})
		return
	}

	h.logger.Info("Collection deleted",
		zap.String("id", collectionID),
		zap.String("name", collectionName),
		zap.Int64("entriesDeleted", entriesDeleted))

	c.JSON(http.StatusOK, gin.H{
		"message":        "Collection deleted successfully",
		"collectionId":   collectionID,
		"collectionName": collectionName,
		"entriesDeleted": entriesDeleted,
	})
}

// RegisterRoutes registers all knowledge-related routes
// VerifyKnowledgeArticle creates a chat session to verify a knowledge article against source code
// POST /api/v1/knowledge/entries/:id/verify
func (h *KnowledgeHandler) VerifyKnowledgeArticle(c *gin.Context) {
	knowledgeID := c.Param("id")

	// Extract user context from JWT (assuming middleware provides this)
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	companyID, exists := c.Get("companyId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Company ID not found"})
		return
	}

	// Fetch knowledge article by ID using storage interface
	ctx := c.Request.Context()
	entry, err := h.knowledgeStorage.GetEntryByID(knowledgeID)
	if err != nil {
		h.logger.Error("Failed to fetch knowledge entry",
			zap.String("entryId", knowledgeID),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Knowledge article not found"})
		return
	}

	// Create verification system prompt
	verificationPrompt := `# Knowledge Article Verification Task

You are tasked with verifying the accuracy of a knowledge article against the current source code implementation.

## Article to Verify:
**Collection**: ` + entry.Collection + `
**Created**: ` + entry.CreatedAt.Format("2006-01-02 15:04:05") + `

**Content**:
` + entry.Text + `

## Your Task:
1. **Read the article carefully** and understand what it claims about the codebase
2. **Use code_index_search** to find relevant source code files mentioned in the article
3. **Compare** the article's claims against the actual implementation
4. **Verify** each key claim:
   - Are the file paths correct?
   - Are the function names accurate?
   - Are the described patterns actually implemented?
   - Are there any outdated references?
5. **Report your findings**:
   - ✅ What is accurate and up-to-date
   - ❌ What is outdated or incorrect
   - 📝 Specific updates needed (with file paths and line numbers)

## Guidelines:
- Be thorough - check multiple files if needed
- Cite specific files and line numbers in your findings
- If something is accurate, say so explicitly
- If something needs updating, provide the exact new information
- Use code_index_search with queries like: "file path from article", "function name mentioned", "pattern described"

**Start by reading the article and identifying what needs to be verified.**
`

	// Create chat session
	session, err := h.chatService.CreateSession(ctx, userID.(string), companyID.(string), "Verify: "+entry.Collection)
	if err != nil {
		h.logger.Error("Failed to create chat session",
			zap.String("entryId", knowledgeID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create verification chat session"})
		return
	}

	// Save system prompt as first message
	_, err = h.chatService.SaveMessage(ctx, session.ID, "system", verificationPrompt, companyID.(string))
	if err != nil {
		h.logger.Error("Failed to save system prompt",
			zap.String("entryId", knowledgeID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize verification session"})
		return
	}

	h.logger.Info("Knowledge article verification session created",
		zap.String("entryId", knowledgeID),
		zap.String("sessionId", session.ID.Hex()))

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"sessionId": session.ID.Hex(),
		"message":   "Verification chat session created. Navigate to the chat to see the verification results.",
	})
}

// RebuildCollectionCountsHandler recalculates entry counts for all collections
// POST /api/v1/knowledge/collections/rebuild-counts
func (h *KnowledgeHandler) RebuildCollectionCountsHandler(c *gin.Context) {
	h.logger.Info("Rebuilding collection counts")

	// Type assertion to get MongoKnowledgeStorage (needed for RebuildCollectionCounts method)
	mongoStorage, ok := h.knowledgeStorage.(*storage.MongoKnowledgeStorage)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rebuild counts only supported with MongoDB storage"})
		return
	}

	// Call the storage method to rebuild counts
	stats, err := mongoStorage.RebuildCollectionCounts()
	if err != nil {
		h.logger.Error("Failed to rebuild collection counts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rebuild collection counts: " + err.Error()})
		return
	}

	// Extract stats
	collectionsUpdated := stats["collectionsUpdated"].(int)
	totalEntries := stats["totalEntries"].(int)
	details := stats["details"].([]map[string]interface{})
	errors := []string{}
	if errList, ok := stats["errors"].([]string); ok && len(errList) > 0 {
		errors = errList
	}

	h.logger.Info("Rebuild collection counts completed",
		zap.Int("collectionsUpdated", collectionsUpdated),
		zap.Int("totalEntries", totalEntries))

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"collectionsUpdated": collectionsUpdated,
		"totalEntries":       totalEntries,
		"details":            details,
		"errors":             errors,
	})
}

func (h *KnowledgeHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/popular-collections", h.GetPopularCollections)
	r.GET("/collections", h.GetAllCollections)
	r.POST("/collections", h.CreateCollectionHandler)
	r.POST("/collections/rebuild-counts", h.RebuildCollectionCountsHandler)
	r.DELETE("/collections/:id", h.DeleteCollectionHandler)
	r.GET("/browse", h.BrowseKnowledge)
	r.POST("/query", h.QueryKnowledge)
	r.PUT("/entries/:id", h.UpdateKnowledgeEntry)
	r.DELETE("/entries/:id", h.DeleteKnowledgeEntry)
	r.POST("/entries/:id/vote", h.VoteOnKnowledgeEntry)
	r.GET("/entries/:id/votes", h.GetKnowledgeEntryVotes)
	r.POST("/entries/:id/review", h.ReviewEntryHandler)
	r.POST("/entries/:id/compact", h.CompactEntryHandler)
	r.POST("/entries/:id/verify", h.VerifyKnowledgeArticle)
	r.PUT("/collections/:name/metadata", h.UpdateCollectionMetadataHandler)
	r.POST("/collections/:name/rename", h.RenameCollectionHandler)
	r.POST("/collections/:name/review", h.ReviewCollectionHandler)
	r.POST("/sync-votes", h.BatchSyncVotes)
	r.POST("/migrate", h.MigrateCollectionsHandler)
}
