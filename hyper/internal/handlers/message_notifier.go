package handlers

import (
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// MessageNotifier manages notifications for new messages in chat sessions
// Used to interrupt running subagents when users send messages
type MessageNotifier struct {
	mu       sync.RWMutex
	channels map[string]chan struct{}
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
			channels: make(map[string]chan struct{}),
			logger:   logger,
		}
	})
	return notifierInstance
}

// RegisterSession creates a notification channel for a session
// Returns the channel that will be notified when new messages arrive
func (mn *MessageNotifier) RegisterSession(sessionID primitive.ObjectID) <-chan struct{} {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	sessionKey := sessionID.Hex()

	// Close existing channel if any (cleanup old registration)
	if existingCh, exists := mn.channels[sessionKey]; exists {
		close(existingCh)
		mn.logger.Debug("Closed existing notification channel for session",
			zap.String("sessionId", sessionKey))
	}

	// Create new notification channel (buffered to prevent blocking)
	notifyCh := make(chan struct{}, 1)
	mn.channels[sessionKey] = notifyCh

	mn.logger.Info("Registered session for message notifications",
		zap.String("sessionId", sessionKey))

	return notifyCh
}

// UnregisterSession removes the notification channel for a session
func (mn *MessageNotifier) UnregisterSession(sessionID primitive.ObjectID) {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	sessionKey := sessionID.Hex()
	if ch, exists := mn.channels[sessionKey]; exists {
		close(ch)
		delete(mn.channels, sessionKey)
		mn.logger.Info("Unregistered session from message notifications",
			zap.String("sessionId", sessionKey))
	}
}

// NotifyNewMessage sends a notification for a new message in the session
// This is called by the WebSocket handler after saving a user message
func (mn *MessageNotifier) NotifyNewMessage(sessionID primitive.ObjectID) {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	sessionKey := sessionID.Hex()
	if ch, exists := mn.channels[sessionKey]; exists {
		// Non-blocking send (buffered channel)
		select {
		case ch <- struct{}{}:
			mn.logger.Info("Notified session of new message",
				zap.String("sessionId", sessionKey))
		default:
			// Channel already has pending notification, no need to send another
			mn.logger.Debug("Notification channel already pending for session",
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
