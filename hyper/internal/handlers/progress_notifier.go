package handlers

import (
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// ProgressEvent represents a progress update message
type ProgressEvent struct {
	Message string
}

// ProgressNotifier manages progress notifications for subchat execution
type ProgressNotifier struct {
	mu       sync.RWMutex
	channels map[string]chan ProgressEvent
	logger   *zap.Logger
}

var (
	progressNotifierInstance *ProgressNotifier
	progressNotifierOnce     sync.Once
)

// GetProgressNotifier returns the singleton ProgressNotifier instance
func GetProgressNotifier(logger *zap.Logger) *ProgressNotifier {
	progressNotifierOnce.Do(func() {
		progressNotifierInstance = &ProgressNotifier{
			channels: make(map[string]chan ProgressEvent),
			logger:   logger,
		}
	})
	return progressNotifierInstance
}

// RegisterSession registers a session to receive progress notifications
func (pn *ProgressNotifier) RegisterSession(sessionID primitive.ObjectID) <-chan ProgressEvent {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	sessionKey := sessionID.Hex()
	// Close existing channel if it exists
	if existingCh, exists := pn.channels[sessionKey]; exists {
		close(existingCh)
	}

	// Create new buffered channel
	progressCh := make(chan ProgressEvent, 10)
	pn.channels[sessionKey] = progressCh
	pn.logger.Info("Registered session for progress notifications", zap.String("sessionId", sessionKey))
	return progressCh
}

// UnregisterSession unregisters a session from receiving progress notifications
func (pn *ProgressNotifier) UnregisterSession(sessionID primitive.ObjectID) {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	sessionKey := sessionID.Hex()
	if ch, exists := pn.channels[sessionKey]; exists {
		close(ch)
		delete(pn.channels, sessionKey)
		pn.logger.Info("Unregistered session from progress notifications", zap.String("sessionId", sessionKey))
	}
}

// EmitProgress emits a progress event to a specific session
func (pn *ProgressNotifier) EmitProgress(sessionID primitive.ObjectID, message string) {
	pn.mu.RLock()
	defer pn.mu.RUnlock()

	sessionKey := sessionID.Hex()
	if ch, exists := pn.channels[sessionKey]; exists {
		select {
		case ch <- ProgressEvent{Message: message}:
			pn.logger.Debug("Emitted progress event", zap.String("sessionId", sessionKey), zap.String("message", message))
		default:
			pn.logger.Warn("Progress channel full, dropping event", zap.String("sessionId", sessionKey))
		}
	}
}
