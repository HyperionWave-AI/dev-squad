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
// PHASE 2: Uses SafeChannel to prevent double-close panics
type ProgressNotifier struct {
	mu       sync.RWMutex
	channels map[string]*SafeChannel[ProgressEvent]
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
			channels: make(map[string]*SafeChannel[ProgressEvent]),
			logger:   logger,
		}
	})
	return progressNotifierInstance
}

// RegisterSession registers a session to receive progress notifications
// PHASE 2: Uses SafeChannel for safe closing
func (pn *ProgressNotifier) RegisterSession(sessionID primitive.ObjectID) <-chan ProgressEvent {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	sessionKey := sessionID.Hex()
	// PHASE 2: Safe close existing channel if it exists (uses sync.Once - no panic)
	if existingSC, exists := pn.channels[sessionKey]; exists {
		existingSC.Close()
		pn.logger.Debug("Closed existing progress channel for session",
			zap.String("sessionId", sessionKey))
	}

	// Create new safe channel
	safeCh := NewSafeChannel[ProgressEvent](10)
	pn.channels[sessionKey] = safeCh
	pn.logger.Info("Registered session for progress notifications", zap.String("sessionId", sessionKey))
	return safeCh.Chan()
}

// UnregisterSession unregisters a session from receiving progress notifications
// PHASE 2: Uses SafeChannel for safe closing
func (pn *ProgressNotifier) UnregisterSession(sessionID primitive.ObjectID) {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	sessionKey := sessionID.Hex()
	if safeCh, exists := pn.channels[sessionKey]; exists {
		safeCh.Close() // PHASE 2: Safe close - uses sync.Once, no panic on double close
		delete(pn.channels, sessionKey)
		pn.logger.Info("Unregistered session from progress notifications", zap.String("sessionId", sessionKey))
	}
}

// EmitProgress emits a progress event to a specific session
// PHASE 2: Uses SafeChannel.Send for safe, non-blocking send
func (pn *ProgressNotifier) EmitProgress(sessionID primitive.ObjectID, message string) {
	pn.mu.RLock()
	defer pn.mu.RUnlock()

	sessionKey := sessionID.Hex()
	if safeCh, exists := pn.channels[sessionKey]; exists {
		// PHASE 2: Check if closed before sending (prevents send on closed channel)
		if safeCh.IsClosed() {
			pn.logger.Debug("Progress channel already closed, skipping emit",
				zap.String("sessionId", sessionKey))
			return
		}

		if safeCh.Send(ProgressEvent{Message: message}) {
			pn.logger.Debug("Emitted progress event",
				zap.String("sessionId", sessionKey),
				zap.String("message", message))
		} else {
			pn.logger.Warn("Progress channel full or closed, dropping event",
				zap.String("sessionId", sessionKey))
		}
	}
}
