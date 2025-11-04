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
	router.POST("/rediscover-all", h.RediscoverAllServers)
	router.GET("/:serverName", h.GetServerDetails)
	router.PUT("/:serverName", h.UpdateServer)
	router.DELETE("/:serverName", h.RemoveMCPServer)
	router.POST("/:serverName/rediscover", h.RediscoverMCPServer)
}

// AddMCPServerRequest represents the request to add a new MCP server
type AddMCPServerRequest struct {
	ServerName  string                 `json:"serverName" binding:"required"`
	ServerURL   string                 `json:"serverUrl" binding:"required"`
	Description string                 `json:"description"`
	Headers     map[string]interface{} `json:"headers,omitempty"`
}

// UpdateMCPServerRequest represents the request to update an existing MCP server
type UpdateMCPServerRequest struct {
	ServerURL   string                 `json:"serverUrl" binding:"required"`
	Description string                 `json:"description"`
	Headers     map[string]interface{} `json:"headers,omitempty"`
}

// AddMCPServerResponse represents the response after adding an MCP server
type AddMCPServerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// MCPServerDTO represents an MCP server for API responses
type MCPServerDTO struct {
	ServerName    string                 `json:"serverName"`
	ServerURL     string                 `json:"serverUrl"`
	Description   string                 `json:"description"`
	Headers       map[string]interface{} `json:"headers,omitempty"`
	ToolCount     int                    `json:"toolCount"`
	ResourceCount int                    `json:"resourceCount"`
	PromptCount   int                    `json:"promptCount"`
	CreatedAt     string                 `json:"createdAt"`
	UpdatedAt     string                 `json:"updatedAt"`
}

// ToolDTO represents a tool for API responses
type ToolDTO struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
}

// ResourceDTO represents a resource for API responses
type ResourceDTO struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// PromptDTO represents a prompt for API responses
type PromptDTO struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Arguments   []map[string]interface{} `json:"arguments,omitempty"`
	CreatedAt   string                   `json:"createdAt"`
	UpdatedAt   string                   `json:"updatedAt"`
}

// ServerDetailsDTO represents detailed server info including tools, resources, and prompts
type ServerDetailsDTO struct {
	ServerName    string                 `json:"serverName"`
	ServerURL     string                 `json:"serverUrl"`
	Description   string                 `json:"description"`
	Headers       map[string]interface{} `json:"headers,omitempty"`
	ToolCount     int                    `json:"toolCount"`
	ResourceCount int                    `json:"resourceCount"`
	PromptCount   int                    `json:"promptCount"`
	CreatedAt     string                 `json:"createdAt"`
	UpdatedAt     string                 `json:"updatedAt"`
	Tools         []ToolDTO              `json:"tools"`
	Resources     []ResourceDTO          `json:"resources"`
	Prompts       []PromptDTO            `json:"prompts"`
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

	ctx := context.Background()

	// Check if server already exists
	existingServer, err := h.toolsStorage.GetServer(ctx, req.ServerName)
	isReplacement := false

	if err == nil && existingServer != nil {
		// Server exists - remove it first to ensure clean replacement
		isReplacement = true
		h.logger.Info("Server already exists, replacing with new configuration",
			zap.String("serverName", req.ServerName),
			zap.String("oldUrl", existingServer.ServerURL),
			zap.String("newUrl", req.ServerURL))

		// Remove old server and all associated data (tools, resources, prompts)
		if err := h.toolsStorage.RemoveServer(ctx, req.ServerName); err != nil {
			h.logger.Error("Failed to remove existing server during replacement",
				zap.String("serverName", req.ServerName),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to replace existing server",
				"details": err.Error(),
			})
			return
		}

		// Remove old tools
		if err := h.toolsStorage.RemoveServerTools(ctx, req.ServerName); err != nil {
			h.logger.Warn("Failed to remove old tools during replacement",
				zap.String("serverName", req.ServerName),
				zap.Error(err))
		}

		// Remove old resources
		if err := h.toolsStorage.RemoveServerResources(ctx, req.ServerName); err != nil {
			h.logger.Warn("Failed to remove old resources during replacement",
				zap.String("serverName", req.ServerName),
				zap.Error(err))
		}

		// Remove old prompts
		if err := h.toolsStorage.RemoveServerPrompts(ctx, req.ServerName); err != nil {
			h.logger.Warn("Failed to remove old prompts during replacement",
				zap.String("serverName", req.ServerName),
				zap.Error(err))
		}
	}

	// Add server to registry (either new or replacement)
	err = h.toolsStorage.AddServer(ctx, req.ServerName, req.ServerURL, req.Description, req.Headers)
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

	// Log appropriate message based on whether this was an add or replacement
	action := "added"
	if isReplacement {
		action = "updated"
	}

	h.logger.Info(fmt.Sprintf("MCP server %s successfully", action),
		zap.String("serverName", req.ServerName),
		zap.String("serverUrl", req.ServerURL))

	c.JSON(http.StatusOK, AddMCPServerResponse{
		Success: true,
		Message: fmt.Sprintf("MCP server '%s' %s successfully", req.ServerName, action),
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
			ServerName:    server.ServerName,
			ServerURL:     server.ServerURL,
			Description:   server.Description,
			Headers:       server.Headers,
			ToolCount:     server.ToolCount,
			ResourceCount: server.ResourceCount,
			PromptCount:   server.PromptCount,
			CreatedAt:     server.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     server.UpdatedAt.Format(time.RFC3339),
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

// RediscoverAllServers handles POST /api/v1/mcp/servers/rediscover-all
func (h *MCPServersHandler) RediscoverAllServers(c *gin.Context) {
	ctx := context.Background()

	// Get all registered servers
	servers, err := h.toolsStorage.ListServers(ctx)
	if err != nil {
		h.logger.Error("Failed to list MCP servers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list MCP servers",
			"details": err.Error(),
		})
		return
	}

	totalServers := len(servers)
	successCount := 0
	failureCount := 0
	totalTools := 0
	totalResources := 0
	totalPrompts := 0
	var errors []map[string]interface{}

	// Rediscover tools for each server
	for _, server := range servers {
		args := map[string]interface{}{
			"serverName": server.ServerName,
		}

		_, data, err := h.toolsDiscoveryHandler.HandleMCPRediscoverServer(ctx, args)
		if err != nil {
			h.logger.Warn("Failed to rediscover tools from MCP server",
				zap.String("serverName", server.ServerName),
				zap.Error(err))
			failureCount++
			errors = append(errors, map[string]interface{}{
				"serverName": server.ServerName,
				"error":      err.Error(),
			})
			continue
		}

		successCount++

		// Extract counts from the response data
		if dataMap, ok := data.(map[string]interface{}); ok {
			if count, ok := dataMap["toolCount"].(int); ok {
				totalTools += count
			}
			if count, ok := dataMap["resourceCount"].(int); ok {
				totalResources += count
			}
			if count, ok := dataMap["promptCount"].(int); ok {
				totalPrompts += count
			}
		}
	}

	h.logger.Info("Rediscovered all MCP servers",
		zap.Int("totalServers", totalServers),
		zap.Int("successCount", successCount),
		zap.Int("failureCount", failureCount),
		zap.Int("totalTools", totalTools),
		zap.Int("totalResources", totalResources),
		zap.Int("totalPrompts", totalPrompts))

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"totalServers":   totalServers,
		"successCount":   successCount,
		"failureCount":   failureCount,
		"totalTools":     totalTools,
		"totalResources": totalResources,
		"totalPrompts":   totalPrompts,
		"errors":         errors,
		"message":        fmt.Sprintf("Rediscovered %d/%d servers: %d tools, %d resources, %d prompts found", successCount, totalServers, totalTools, totalResources, totalPrompts),
	})
}

// UpdateServer handles PUT /api/v1/mcp/servers/:serverName
func (h *MCPServersHandler) UpdateServer(c *gin.Context) {
	serverName := c.Param("serverName")
	if serverName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server name is required",
		})
		return
	}

	var req UpdateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	ctx := context.Background()

	// Update server
	err := h.toolsStorage.UpdateServer(ctx, serverName, req.ServerURL, req.Description, req.Headers)
	if err != nil {
		h.logger.Error("Failed to update MCP server",
			zap.String("serverName", serverName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update MCP server",
			"details": err.Error(),
		})
		return
	}

	// Get updated server details
	server, err := h.toolsStorage.GetServer(ctx, serverName)
	if err != nil {
		h.logger.Error("Failed to retrieve updated server",
			zap.String("serverName", serverName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Server updated but failed to retrieve details",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("MCP server updated successfully",
		zap.String("serverName", serverName),
		zap.String("serverUrl", req.ServerURL))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("MCP server '%s' updated successfully", serverName),
		"server": MCPServerDTO{
			ServerName:    server.ServerName,
			ServerURL:     server.ServerURL,
			Description:   server.Description,
			Headers:       server.Headers,
			ToolCount:     server.ToolCount,
			ResourceCount: server.ResourceCount,
			PromptCount:   server.PromptCount,
			CreatedAt:     server.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     server.UpdatedAt.Format(time.RFC3339),
		},
	})
}

// GetServerDetails handles GET /api/v1/mcp/servers/:serverName
func (h *MCPServersHandler) GetServerDetails(c *gin.Context) {
	serverName := c.Param("serverName")
	if serverName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server name is required",
		})
		return
	}

	ctx := context.Background()

	// Get server metadata
	server, err := h.toolsStorage.GetServer(ctx, serverName)
	if err != nil {
		h.logger.Error("Failed to get server",
			zap.String("serverName", serverName),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Server not found",
			"details": err.Error(),
		})
		return
	}

	// Get server tools
	tools, err := h.toolsStorage.GetServerTools(ctx, serverName)
	if err != nil {
		h.logger.Error("Failed to get server tools",
			zap.String("serverName", serverName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve server tools",
			"details": err.Error(),
		})
		return
	}

	// Get server resources
	resources, err := h.toolsStorage.GetServerResources(ctx, serverName)
	if err != nil {
		h.logger.Error("Failed to get server resources",
			zap.String("serverName", serverName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve server resources",
			"details": err.Error(),
		})
		return
	}

	// Get server prompts
	prompts, err := h.toolsStorage.GetServerPrompts(ctx, serverName)
	if err != nil {
		h.logger.Error("Failed to get server prompts",
			zap.String("serverName", serverName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve server prompts",
			"details": err.Error(),
		})
		return
	}

	// Convert tools to DTOs
	toolDTOs := make([]ToolDTO, 0, len(tools))
	for _, tool := range tools {
		toolDTOs = append(toolDTOs, ToolDTO{
			Name:        tool.ToolName,
			Description: tool.Description,
			InputSchema: tool.Schema,
			CreatedAt:   tool.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   tool.UpdatedAt.Format(time.RFC3339),
		})
	}

	// Convert resources to DTOs
	resourceDTOs := make([]ResourceDTO, 0, len(resources))
	for _, resource := range resources {
		resourceDTOs = append(resourceDTOs, ResourceDTO{
			URI:         resource.URI,
			Name:        resource.Name,
			Description: resource.Description,
			MimeType:    resource.MimeType,
			CreatedAt:   resource.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   resource.UpdatedAt.Format(time.RFC3339),
		})
	}

	// Convert prompts to DTOs
	promptDTOs := make([]PromptDTO, 0, len(prompts))
	for _, prompt := range prompts {
		promptDTOs = append(promptDTOs, PromptDTO{
			Name:        prompt.Name,
			Description: prompt.Description,
			Arguments:   prompt.Arguments,
			CreatedAt:   prompt.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   prompt.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"server": ServerDetailsDTO{
			ServerName:    server.ServerName,
			ServerURL:     server.ServerURL,
			Description:   server.Description,
			Headers:       server.Headers,
			ToolCount:     len(tools),
			ResourceCount: len(resources),
			PromptCount:   len(prompts),
			CreatedAt:     server.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     server.UpdatedAt.Format(time.RFC3339),
			Tools:         toolDTOs,
			Resources:     resourceDTOs,
			Prompts:       promptDTOs,
		},
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
