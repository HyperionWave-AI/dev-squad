package handlers

import (
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// MessageNotifier manages notifications for new messages in chat sessions
// Used to interrupt running subagents when users send messages
// PHASE 3: Uses SafeChannel to prevent double-close panics
type MessageNotifier struct {
	mu       sync.RWMutex
	channels map[string]*SafeChannel[struct{}]
	logger   *zap.Logger
}

// Global singleton instance
var (
	notifierInstance *MessageNotifier
	notifierOnce     sync.Once
)

// GetMessageNotifier returns the singleton MessageNotifier instance
func GetMessageNotifier(logger *zap.Logger) *MessageNotifier {
	notifierOnce.Do(func() {
		notifierInstance = &MessageNotifier{
			channels: make(map[string]*SafeChannel[struct{}]),
			logger:   logger,
		}
	})
	return notifierInstance
}

// RegisterSession creates a notification channel for a session
// Returns the channel that will be notified when new messages arrive
// PHASE 3: Uses SafeChannel for safe closing
func (mn *MessageNotifier) RegisterSession(sessionID primitive.ObjectID) <-chan struct{} {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	sessionKey := sessionID.Hex()

	// PHASE 3: Safe close existing channel if any (uses sync.Once - no panic)
	if existingSC, exists := mn.channels[sessionKey]; exists {
		existingSC.Close()
		mn.logger.Debug("Closed existing notification channel for session",
			zap.String("sessionId", sessionKey))
	}

	// Create new safe channel (buffered to prevent blocking)
	safeCh := NewSafeChannel[struct{}](1)
	mn.channels[sessionKey] = safeCh

	mn.logger.Info("Registered session for message notifications",
		zap.String("sessionId", sessionKey))

	return safeCh.Chan()
}

// UnregisterSession removes the notification channel for a session
// PHASE 3: Uses SafeChannel for safe closing
func (mn *MessageNotifier) UnregisterSession(sessionID primitive.ObjectID) {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	sessionKey := sessionID.Hex()
	if safeCh, exists := mn.channels[sessionKey]; exists {
		safeCh.Close() // PHASE 3: Safe close - uses sync.Once, no panic on double close
		delete(mn.channels, sessionKey)
		mn.logger.Info("Unregistered session from message notifications",
			zap.String("sessionId", sessionKey))
	}
}

// NotifyNewMessage sends a notification for a new message in the session
// This is called by the WebSocket handler after saving a user message
// PHASE 3: Uses SafeChannel.Send for safe, non-blocking send
func (mn *MessageNotifier) NotifyNewMessage(sessionID primitive.ObjectID) {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	sessionKey := sessionID.Hex()
	if safeCh, exists := mn.channels[sessionKey]; exists {
		// PHASE 3: Check if closed before sending (prevents send on closed channel)
		if safeCh.IsClosed() {
			mn.logger.Debug("Notification channel already closed, skipping notify",
				zap.String("sessionId", sessionKey))
			return
		}

		// PHASE 3: Safe non-blocking send
		if safeCh.Send(struct{}{}) {
			mn.logger.Info("Notified session of new message",
				zap.String("sessionId", sessionKey))
		} else {
			// Channel already has pending notification or is closed
			mn.logger.Debug("Notification channel already pending or closed for session",
				zap.String("sessionId", sessionKey))
		}
	} else {
		// No active listener for this session (normal for non-subchat sessions)
		mn.logger.Debug("No notification channel registered for session",
			zap.String("sessionId", sessionKey))
	}
}

// IsSessionRegistered checks if a session is registered for notifications
func (mn *MessageNotifier) IsSessionRegistered(sessionID primitive.ObjectID) bool {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	sessionKey := sessionID.Hex()
	_, exists := mn.channels[sessionKey]
	return exists
}

// GetActiveSessionCount returns the number of sessions currently registered
func (mn *MessageNotifier) GetActiveSessionCount() int {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	return len(mn.channels)
}
