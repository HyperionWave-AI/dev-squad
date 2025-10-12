package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hyper/internal/models"
	"hyper/internal/mcp/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// HTTPToolsHandler handles HTTP tool management operations
type HTTPToolsHandler struct {
	httpToolsCollection *mongo.Collection
	toolsStorage        storage.ToolsStorageInterface
	logger              *zap.Logger
}

// NewHTTPToolsHandler creates a new HTTP tools handler
func NewHTTPToolsHandler(
	mongoDatabase *mongo.Database,
	toolsStorage storage.ToolsStorageInterface,
	logger *zap.Logger,
) (*HTTPToolsHandler, error) {
	handler := &HTTPToolsHandler{
		httpToolsCollection: mongoDatabase.Collection("http_tools"),
		toolsStorage:        toolsStorage,
		logger:              logger,
	}

	// Create indexes
	ctx := context.Background()

	// Compound index on toolName + companyId for uniqueness per company
	_, err := handler.httpToolsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "toolName", Value: 1},
			{Key: "companyId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create toolName+companyId index: %w", err)
	}

	// Index on companyId for filtering
	_, err = handler.httpToolsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "companyId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create companyId index: %w", err)
	}

	// Index on createdAt for sorting
	_, err = handler.httpToolsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "createdAt", Value: -1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create createdAt index: %w", err)
	}

	return handler, nil
}

// RegisterHTTPToolsRoutes registers HTTP tools routes with the router
func (h *HTTPToolsHandler) RegisterHTTPToolsRoutes(router *gin.RouterGroup) {
	router.POST("", h.CreateHTTPTool)
	router.GET("", h.ListHTTPTools)
	router.DELETE("/:id", h.DeleteHTTPTool)
	router.GET("/:id", h.GetHTTPTool)
}

// CreateHTTPTool handles POST /api/v1/tools/http
func (h *HTTPToolsHandler) CreateHTTPTool(c *gin.Context) {
	// Extract user identity from context (injected by JWT middleware)
	userID, exists := c.Get("userId")
	if !exists {
		h.logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User identity not found. JWT middleware may not be configured correctly.",
		})
		return
	}

	companyID, exists := c.Get("companyId")
	if !exists {
		h.logger.Error("Company ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Company identity not found. JWT middleware may not be configured correctly.",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		h.logger.Error("User ID is not a string", zap.Any("userId", userID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID type in context",
		})
		return
	}

	companyIDStr, ok := companyID.(string)
	if !ok {
		h.logger.Error("Company ID is not a string", zap.Any("companyId", companyID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid company ID type in context",
		})
		return
	}

	// Parse request body
	var req models.CreateHTTPToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate HTTP method
	validMethods := map[models.HTTPMethod]bool{
		models.HTTPMethodGET:    true,
		models.HTTPMethodPOST:   true,
		models.HTTPMethodPUT:    true,
		models.HTTPMethodDELETE: true,
		models.HTTPMethodPATCH:  true,
	}
	if !validMethods[req.Method] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid HTTP method '%s'. Allowed: GET, POST, PUT, DELETE, PATCH", req.Method),
		})
		return
	}

	// Validate auth type
	validAuthTypes := map[models.HTTPAuthType]bool{
		models.AuthTypeNone:   true,
		models.AuthTypeBearer: true,
		models.AuthTypeAPIKey: true,
		models.AuthTypeBasic:  true,
	}
	if !validAuthTypes[req.AuthType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid auth type '%s'. Allowed: none, bearer, apiKey, basic", req.AuthType),
		})
		return
	}

	// Validate endpoint is a valid URL or path
	if req.Endpoint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Endpoint cannot be empty",
		})
		return
	}

	// Generate tool ID
	toolID := uuid.New().String()

	// Create HTTP tool definition
	tool := &models.HTTPToolDefinition{
		ID:             toolID,
		ToolName:       req.ToolName,
		Description:    req.Description,
		Endpoint:       req.Endpoint,
		Method:         req.Method,
		Headers:        req.Headers,
		Parameters:     req.Parameters,
		AuthType:       req.AuthType,
		AuthTokenField: req.AuthTokenField,
		CompanyID:      companyIDStr,
		CreatedBy:      userIDStr,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Store in MongoDB
	ctx := c.Request.Context()
	_, err := h.httpToolsCollection.InsertOne(ctx, tool)
	if err != nil {
		// Check for duplicate key error
		if mongo.IsDuplicateKeyError(err) {
			h.logger.Warn("HTTP tool already exists",
				zap.String("toolName", req.ToolName),
				zap.String("companyId", companyIDStr))
			c.JSON(http.StatusConflict, gin.H{
				"error": fmt.Sprintf("HTTP tool with name '%s' already exists for your company", req.ToolName),
			})
			return
		}

		h.logger.Error("Failed to store HTTP tool in MongoDB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store HTTP tool. Please try again.",
		})
		return
	}

	// Generate semantic description for tool discovery
	semanticDescription := h.generateSemanticDescription(tool)

	// Store tool metadata in ToolsStorage for semantic discovery
	// Build schema representation
	schema := map[string]interface{}{
		"type":       "http",
		"endpoint":   tool.Endpoint,
		"method":     tool.Method,
		"parameters": tool.Parameters,
		"authType":   tool.AuthType,
	}

	// Store in ToolsStorage (MongoDB + Qdrant for semantic search)
	if err := h.toolsStorage.StoreToolMetadata(ctx, tool.ToolName, semanticDescription, schema, "http-custom"); err != nil {
		// Log error but don't fail the request - tool is already stored in MongoDB
		h.logger.Warn("Failed to store tool metadata in ToolsStorage (tool is saved but may not be discoverable via semantic search)",
			zap.String("toolName", tool.ToolName),
			zap.Error(err))
	}

	h.logger.Info("HTTP tool created successfully",
		zap.String("toolId", toolID),
		zap.String("toolName", tool.ToolName),
		zap.String("companyId", companyIDStr),
		zap.String("createdBy", userIDStr))

	c.JSON(http.StatusCreated, gin.H{
		"id":      toolID,
		"message": fmt.Sprintf("HTTP tool '%s' created successfully and is now discoverable via semantic search", tool.ToolName),
		"tool":    tool,
	})
}

// ListHTTPTools handles GET /api/v1/tools/http with pagination
func (h *HTTPToolsHandler) ListHTTPTools(c *gin.Context) {
	// Extract company ID from context
	companyID, exists := c.Get("companyId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Company identity not found",
		})
		return
	}

	companyIDStr, ok := companyID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid company ID type in context",
		})
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Build filter for company-level isolation
	filter := bson.M{"companyId": companyIDStr}

	// Count total documents
	ctx := c.Request.Context()
	total, err := h.httpToolsCollection.CountDocuments(ctx, filter)
	if err != nil {
		h.logger.Error("Failed to count HTTP tools", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to count HTTP tools",
		})
		return
	}

	// Calculate pagination
	skip := (page - 1) * pageSize
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	// Query with pagination
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := h.httpToolsCollection.Find(ctx, filter, opts)
	if err != nil {
		h.logger.Error("Failed to query HTTP tools", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve HTTP tools",
		})
		return
	}
	defer cursor.Close(ctx)

	// Decode results
	var tools []models.HTTPToolDefinition
	if err := cursor.All(ctx, &tools); err != nil {
		h.logger.Error("Failed to decode HTTP tools", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to decode HTTP tools",
		})
		return
	}

	// Ensure tools is not nil (return empty array instead)
	if tools == nil {
		tools = make([]models.HTTPToolDefinition, 0)
	}

	response := models.HTTPToolListResponse{
		Tools:      tools,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteHTTPTool handles DELETE /api/v1/tools/http/:id
func (h *HTTPToolsHandler) DeleteHTTPTool(c *gin.Context) {
	// Extract company ID from context
	companyID, exists := c.Get("companyId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Company identity not found",
		})
		return
	}

	companyIDStr, ok := companyID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid company ID type in context",
		})
		return
	}

	// Get tool ID from URL parameter
	toolID := c.Param("id")
	if toolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tool ID is required",
		})
		return
	}

	// Build filter with company-level isolation
	filter := bson.M{
		"_id":       toolID,
		"companyId": companyIDStr,
	}

	// Delete from MongoDB
	ctx := c.Request.Context()
	result, err := h.httpToolsCollection.DeleteOne(ctx, filter)
	if err != nil {
		h.logger.Error("Failed to delete HTTP tool", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete HTTP tool",
		})
		return
	}

	// Check if tool was found and deleted
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("HTTP tool with ID '%s' not found or you don't have permission to delete it", toolID),
		})
		return
	}

	h.logger.Info("HTTP tool deleted successfully",
		zap.String("toolId", toolID),
		zap.String("companyId", companyIDStr))

	c.JSON(http.StatusOK, gin.H{
		"message": "HTTP tool deleted successfully",
		"id":      toolID,
	})
}

// GetHTTPTool handles GET /api/v1/tools/http/:id
func (h *HTTPToolsHandler) GetHTTPTool(c *gin.Context) {
	// Extract company ID from context
	companyID, exists := c.Get("companyId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Company identity not found",
		})
		return
	}

	companyIDStr, ok := companyID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid company ID type in context",
		})
		return
	}

	// Get tool ID from URL parameter
	toolID := c.Param("id")
	if toolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tool ID is required",
		})
		return
	}

	// Build filter with company-level isolation
	filter := bson.M{
		"_id":       toolID,
		"companyId": companyIDStr,
	}

	// Query from MongoDB
	ctx := c.Request.Context()
	var tool models.HTTPToolDefinition
	err := h.httpToolsCollection.FindOne(ctx, filter).Decode(&tool)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("HTTP tool with ID '%s' not found or you don't have permission to view it", toolID),
			})
			return
		}

		h.logger.Error("Failed to retrieve HTTP tool", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve HTTP tool",
		})
		return
	}

	c.JSON(http.StatusOK, tool)
}

// generateSemanticDescription creates a semantic-friendly description for tool discovery
func (h *HTTPToolsHandler) generateSemanticDescription(tool *models.HTTPToolDefinition) string {
	var parts []string

	// Start with the tool name and description
	parts = append(parts, fmt.Sprintf("%s: %s", tool.ToolName, tool.Description))

	// Add HTTP method and endpoint context
	parts = append(parts, fmt.Sprintf("HTTP %s endpoint at %s", tool.Method, tool.Endpoint))

	// Describe parameters if present
	if len(tool.Parameters) > 0 {
		paramNames := make([]string, 0, len(tool.Parameters))
		for _, param := range tool.Parameters {
			paramNames = append(paramNames, param.Name)
		}
		parts = append(parts, fmt.Sprintf("Accepts parameters: %s", strings.Join(paramNames, ", ")))
	}

	// Describe authentication
	switch tool.AuthType {
	case models.AuthTypeBearer:
		parts = append(parts, "Requires Bearer token authentication")
	case models.AuthTypeAPIKey:
		parts = append(parts, "Requires API key authentication")
	case models.AuthTypeBasic:
		parts = append(parts, "Requires Basic authentication")
	case models.AuthTypeNone:
		parts = append(parts, "No authentication required")
	}

	return strings.Join(parts, ". ")
}
