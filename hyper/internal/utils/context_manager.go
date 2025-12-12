package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hyper/internal/models"
	"go.uber.org/zap"
)

// ContextLimitConfig holds configuration for context limit management
type ContextLimitConfig struct {
	// MaxTokens is the hard limit for context (default: 100,000)
	MaxTokens int
	// WarningThreshold is the percentage at which to warn (default: 80%)
	WarningThreshold float64
	// CriticalThreshold is the percentage at which to block new messages (default: 90%)
	CriticalThreshold float64
	// AutoSummarizeThreshold is the percentage at which to trigger auto-summarization (default: 80%)
	AutoSummarizeThreshold float64
}

// DefaultContextLimitConfig returns the default configuration
func DefaultContextLimitConfig() ContextLimitConfig {
	return ContextLimitConfig{
		MaxTokens:              100000,
		WarningThreshold:       80.0,
		CriticalThreshold:      90.0,
		AutoSummarizeThreshold: 80.0,
	}
}

// ContextUsage tracks token usage for a session
type ContextUsage struct {
	SessionID              string    `json:"sessionId"`
	TotalTokens            int       `json:"totalTokens"`
	MaxTokens              int       `json:"maxTokens"`
	PercentageUsed         float64   `json:"percentageUsed"`
	IsWarning              bool      `json:"isWarning"`
	IsCritical             bool      `json:"isCritical"`
	NeedsSummarization     bool      `json:"needsSummarization"`
	MessageCount           int       `json:"messageCount"`
	LastUpdated            time.Time `json:"lastUpdated"`
	SummarizedMessageCount int       `json:"summarizedMessageCount"`
}

// ContextManager manages context limits and token usage for chat sessions
type ContextManager struct {
	config       ContextLimitConfig
	tokenCounter *TokenCounter
	logger       *zap.Logger
	mu           sync.RWMutex
	// sessionUsage tracks usage per session
	sessionUsage map[string]*ContextUsage
}

// NewContextManager creates a new context manager
func NewContextManager(config ContextLimitConfig, logger *zap.Logger) *ContextManager {
	return &ContextManager{
		config:       config,
		tokenCounter: NewTokenCounter(),
		logger:       logger,
		sessionUsage: make(map[string]*ContextUsage),
	}
}

// UpdateContextUsage updates the context usage for a session
func (cm *ContextManager) UpdateContextUsage(ctx context.Context, sessionID string, messages []models.ChatMessage) *ContextUsage {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	totalTokens := cm.tokenCounter.CountSessionTokens(messages)
	percentage := cm.tokenCounter.CalculatePercentage(totalTokens, cm.config.MaxTokens)

	usage := &ContextUsage{
		SessionID:      sessionID,
		TotalTokens:    totalTokens,
		MaxTokens:      cm.config.MaxTokens,
		PercentageUsed: percentage,
		IsWarning:      percentage >= cm.config.WarningThreshold && percentage < cm.config.CriticalThreshold,
		IsCritical:     percentage >= cm.config.CriticalThreshold,
		NeedsSummarization: percentage >= cm.config.AutoSummarizeThreshold,
		MessageCount:   len(messages),
		LastUpdated:    time.Now(),
	}

	cm.sessionUsage[sessionID] = usage

	// Log context usage
	cm.logContextUsage(usage)

	return usage
}

// GetContextUsage returns the current context usage for a session
func (cm *ContextManager) GetContextUsage(sessionID string) *ContextUsage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if usage, exists := cm.sessionUsage[sessionID]; exists {
		return usage
	}

	// Return empty usage if not found
	return &ContextUsage{
		SessionID:   sessionID,
		MaxTokens:   cm.config.MaxTokens,
		LastUpdated: time.Now(),
	}
}

// CanAddMessage checks if a new message can be added without exceeding limits
func (cm *ContextManager) CanAddMessage(sessionID string, messageContent string) (bool, *ContextUsage) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	usage, exists := cm.sessionUsage[sessionID]
	if !exists {
		// If no usage tracked yet, allow the message
		return true, nil
	}

	// Estimate tokens for new message
	estimatedTokens := cm.tokenCounter.EstimateTokensForContent(messageContent)
	projectedTotal := usage.TotalTokens + estimatedTokens

	// Check if adding this message would exceed the limit
	canAdd := projectedTotal <= cm.config.MaxTokens

	return canAdd, usage
}

// CheckContextHealth returns the health status of context usage
func (cm *ContextManager) CheckContextHealth(sessionID string) ContextHealth {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	usage, exists := cm.sessionUsage[sessionID]
	if !exists {
		return ContextHealthHealthy
	}

	if usage.IsCritical {
		return ContextHealthCritical
	}
	if usage.IsWarning {
		return ContextHealthWarning
	}
	return ContextHealthHealthy
}

// logContextUsage logs context usage information
func (cm *ContextManager) logContextUsage(usage *ContextUsage) {
	fields := []zap.Field{
		zap.String("sessionId", usage.SessionID),
		zap.Int("totalTokens", usage.TotalTokens),
		zap.Int("maxTokens", usage.MaxTokens),
		zap.Float64("percentageUsed", usage.PercentageUsed),
		zap.Int("messageCount", usage.MessageCount),
	}

	if usage.IsCritical {
		cm.logger.Warn("🚨 CRITICAL: Context usage at critical level",
			append(fields, zap.String("status", "CRITICAL"))...)
	} else if usage.IsWarning {
		cm.logger.Warn("⚠️ WARNING: Context usage approaching limit",
			append(fields, zap.String("status", "WARNING"))...)
	} else if usage.NeedsSummarization {
		cm.logger.Info("📊 Context usage at summarization threshold",
			append(fields, zap.String("status", "SUMMARIZATION_NEEDED"))...)
	} else {
		cm.logger.Debug("Context usage updated",
			append(fields, zap.String("status", "HEALTHY"))...)
	}
}

// ClearSessionUsage clears the usage tracking for a session
func (cm *ContextManager) ClearSessionUsage(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.sessionUsage, sessionID)
	cm.logger.Debug("Session usage cleared", zap.String("sessionId", sessionID))
}

// GetAllSessionUsage returns usage for all tracked sessions
func (cm *ContextManager) GetAllSessionUsage() []ContextUsage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	usages := make([]ContextUsage, 0, len(cm.sessionUsage))
	for _, usage := range cm.sessionUsage {
		usages = append(usages, *usage)
	}
	return usages
}

// ContextHealth represents the health status of context usage
type ContextHealth string

const (
	ContextHealthHealthy  ContextHealth = "healthy"
	ContextHealthWarning  ContextHealth = "warning"
	ContextHealthCritical ContextHealth = "critical"
)

// String returns the string representation of context health
func (ch ContextHealth) String() string {
	return string(ch)
}

// ContextError represents an error related to context limits
type ContextError struct {
	Code              string        `json:"code"`
	Message           string        `json:"message"`
	CurrentTokens     int           `json:"currentTokens"`
	MaxTokens         int           `json:"maxTokens"`
	PercentageUsed    float64       `json:"percentageUsed"`
	RecoveryOptions   []string      `json:"recoveryOptions"`
	SuggestedAction   string        `json:"suggestedAction"`
	CanArchiveMessages bool         `json:"canArchiveMessages"`
	CanSummarize      bool         `json:"canSummarize"`
}

// NewContextError creates a new context error
func NewContextError(code, message string, usage *ContextUsage) *ContextError {
	recoveryOptions := []string{}
	suggestedAction := ""
	canArchive := false
	canSummarize := false

	if usage.IsCritical {
		recoveryOptions = append(recoveryOptions, "archive_old_messages")
		recoveryOptions = append(recoveryOptions, "summarize_conversation")
		recoveryOptions = append(recoveryOptions, "start_new_session")
		suggestedAction = "Archive old messages or start a new conversation"
		canArchive = true
		canSummarize = true
	} else if usage.IsWarning {
		recoveryOptions = append(recoveryOptions, "summarize_conversation")
		recoveryOptions = append(recoveryOptions, "archive_old_messages")
		suggestedAction = "Consider summarizing the conversation to free up context"
		canSummarize = true
		canArchive = true
	}

	return &ContextError{
		Code:              code,
		Message:           message,
		CurrentTokens:     usage.TotalTokens,
		MaxTokens:         usage.MaxTokens,
		PercentageUsed:    usage.PercentageUsed,
		RecoveryOptions:   recoveryOptions,
		SuggestedAction:   suggestedAction,
		CanArchiveMessages: canArchive,
		CanSummarize:      canSummarize,
	}
}

// Error implements the error interface
func (ce *ContextError) Error() string {
	return fmt.Sprintf("%s: %s (%.1f%% of %d tokens used)", ce.Code, ce.Message, ce.PercentageUsed, ce.MaxTokens)
}
