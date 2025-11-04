package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"hyper/internal/storage"
)

// UserSettingsHandler handles user settings API endpoints
type UserSettingsHandler struct {
	storage *storage.UserSettingsStorage
	logger  *zap.Logger
}

// NewUserSettingsHandler creates a new UserSettingsHandler
func NewUserSettingsHandler(storage *storage.UserSettingsStorage, logger *zap.Logger) *UserSettingsHandler {
	return &UserSettingsHandler{
		storage: storage,
		logger:  logger,
	}
}

// GetUserSettings handles GET /api/v1/user/settings
// @Summary Get user settings
// @Description Retrieves the current user's settings including conversation mode preference
// @Tags user-settings
// @Accept json
// @Produce json
// @Success 200 {object} storage.UserSettings
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/user/settings [get]
func (h *UserSettingsHandler) GetUserSettings(c *gin.Context) {
	userID := c.GetString("userId")
	companyID := c.GetString("companyId")

	h.logger.Info("Fetching user settings",
		zap.String("userId", userID),
		zap.String("companyId", companyID))

	settings, err := h.storage.GetUserSettings(c.Request.Context(), userID, companyID)
	if err != nil {
		h.logger.Error("Failed to get user settings",
			zap.Error(err),
			zap.String("userId", userID),
			zap.String("companyId", companyID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve user settings",
		})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateUserSettingsRequest represents the request body for updating user settings
type UpdateUserSettingsRequest struct {
	ConversationMode string `json:"conversationMode" binding:"required,oneof=debug default"`
}

// UpdateUserSettings handles PATCH /api/v1/user/settings
// @Summary Update user settings
// @Description Updates the current user's settings (currently supports conversation mode)
// @Tags user-settings
// @Accept json
// @Produce json
// @Param request body UpdateUserSettingsRequest true "Settings to update"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/user/settings [patch]
func (h *UserSettingsHandler) UpdateUserSettings(c *gin.Context) {
	userID := c.GetString("userId")
	companyID := c.GetString("companyId")

	var req UpdateUserSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body",
			zap.Error(err),
			zap.String("userId", userID))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: conversationMode must be 'debug' or 'default'",
		})
		return
	}

	h.logger.Info("Updating user settings",
		zap.String("userId", userID),
		zap.String("companyId", companyID),
		zap.String("conversationMode", req.ConversationMode))

	// Convert string to ConversationMode type
	mode := storage.ConversationMode(req.ConversationMode)

	err := h.storage.UpdateConversationMode(c.Request.Context(), userID, companyID, mode)
	if err != nil {
		h.logger.Error("Failed to update conversation mode",
			zap.Error(err),
			zap.String("userId", userID),
			zap.String("companyId", companyID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user settings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User settings updated successfully",
		"conversationMode": req.ConversationMode,
	})
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Error string `json:"error"`
}
