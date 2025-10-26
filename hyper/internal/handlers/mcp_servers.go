package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"hyper/internal/mcp/handlers"
	"hyper/internal/mcp/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MCPServersHandler handles MCP server registry operations
type MCPServersHandler struct {
	toolsStorage           storage.ToolsStorageInterface
	toolsDiscoveryHandler  *handlers.ToolsDiscoveryHandler
	logger                 *zap.Logger
}

// NewMCPServersHandler creates a new MCP servers handler
func NewMCPServersHandler(
	toolsStorage storage.ToolsStorageInterface,
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler,
	logger *zap.Logger,
) *MCPServersHandler {
	return &MCPServersHandler{
		toolsStorage:          toolsStorage,
		toolsDiscoveryHandler: toolsDiscoveryHandler,
		logger:                logger,
	}
}

// RegisterMCPServersRoutes registers MCP server routes with the router
func (h *MCPServersHandler) RegisterMCPServersRoutes(router *gin.RouterGroup) {
	router.POST("", h.AddMCPServer)
	router.GET("", h.ListMCPServers)
	router.DELETE("/:serverName", h.RemoveMCPServer)
	router.POST("/:serverName/rediscover", h.RediscoverMCPServer)
}

// AddMCPServerRequest represents the request to add a new MCP server
type AddMCPServerRequest struct {
	ServerName  string `json:"serverName" binding:"required"`
	ServerURL   string `json:"serverUrl" binding:"required"`
	Description string `json:"description"`
}

// AddMCPServerResponse represents the response after adding an MCP server
type AddMCPServerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// MCPServerDTO represents an MCP server for API responses
type MCPServerDTO struct {
	ServerName  string `json:"serverName"`
	ServerURL   string `json:"serverUrl"`
	Description string `json:"description"`
	ToolCount   int    `json:"toolCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// ListMCPServersResponse represents the response when listing MCP servers
type ListMCPServersResponse struct {
	Servers []MCPServerDTO `json:"servers"`
	Total   int            `json:"total"`
}

// AddMCPServer handles POST /api/v1/mcp/servers
func (h *MCPServersHandler) AddMCPServer(c *gin.Context) {
	var req AddMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate server name (no spaces, alphanumeric + dash/underscore)
	if !isValidServerName(req.ServerName) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid server name. Use only alphanumeric characters, dashes, and underscores.",
		})
		return
	}

	// Add server to registry
	ctx := context.Background()
	err := h.toolsStorage.AddServer(ctx, req.ServerName, req.ServerURL, req.Description)
	if err != nil {
		h.logger.Error("Failed to add MCP server",
			zap.String("serverName", req.ServerName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add MCP server",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("MCP server added successfully",
		zap.String("serverName", req.ServerName),
		zap.String("serverUrl", req.ServerURL))

	c.JSON(http.StatusOK, AddMCPServerResponse{
		Success: true,
		Message: fmt.Sprintf("MCP server '%s' added successfully", req.ServerName),
	})
}

// ListMCPServers handles GET /api/v1/mcp/servers
func (h *MCPServersHandler) ListMCPServers(c *gin.Context) {
	ctx := context.Background()

	servers, err := h.toolsStorage.ListServers(ctx)
	if err != nil {
		h.logger.Error("Failed to list MCP servers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list MCP servers",
			"details": err.Error(),
		})
		return
	}

	// Convert to DTOs
	serverDTOs := make([]MCPServerDTO, 0, len(servers))
	for _, server := range servers {
		serverDTOs = append(serverDTOs, MCPServerDTO{
			ServerName:  server.ServerName,
			ServerURL:   server.ServerURL,
			Description: server.Description,
			ToolCount:   server.ToolCount,
			CreatedAt:   server.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   server.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, ListMCPServersResponse{
		Servers: serverDTOs,
		Total:   len(serverDTOs),
	})
}

// RemoveMCPServer handles DELETE /api/v1/mcp/servers/:serverName
func (h *MCPServersHandler) RemoveMCPServer(c *gin.Context) {
	serverName := c.Param("serverName")
	if serverName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server name is required",
		})
		return
	}

	ctx := context.Background()

	// Remove server and all its tools
	err := h.toolsStorage.RemoveServer(ctx, serverName)
	if err != nil {
		h.logger.Error("Failed to remove MCP server",
			zap.String("serverName", serverName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to remove MCP server",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("MCP server removed successfully",
		zap.String("serverName", serverName))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("MCP server '%s' removed successfully", serverName),
	})
}

// RediscoverMCPServer handles POST /api/v1/mcp/servers/:serverName/rediscover
func (h *MCPServersHandler) RediscoverMCPServer(c *gin.Context) {
	serverName := c.Param("serverName")
	if serverName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server name is required",
		})
		return
	}

	ctx := context.Background()

	// Use the tools discovery handler to rediscover tools
	args := map[string]interface{}{
		"serverName": serverName,
	}

	_, data, err := h.toolsDiscoveryHandler.HandleMCPRediscoverServer(ctx, args)
	if err != nil {
		h.logger.Error("Failed to rediscover tools from MCP server",
			zap.String("serverName", serverName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to rediscover tools from MCP server",
			"details": err.Error(),
		})
		return
	}

	// Extract tool count from the response data
	toolCount := 0
	if dataMap, ok := data.(map[string]interface{}); ok {
		if count, ok := dataMap["toolCount"].(int); ok {
			toolCount = count
		}
	}

	h.logger.Info("MCP server tools rediscovered successfully",
		zap.String("serverName", serverName),
		zap.Int("toolCount", toolCount))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Tools from MCP server '%s' rediscovered successfully (%d tools)", serverName, toolCount),
	})
}

// isValidServerName validates server name format
func isValidServerName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}
