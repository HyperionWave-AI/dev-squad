package handlers

import (
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// PHASE 8: Configurable buffer size for notification channels
// Increased from 1 to 10 to handle burst notifications without dropping
const (
	MessageNotifierBufferSize  = 10  // Buffer for message notifications
	ProgressNotifierBufferSize = 20  // Buffer for progress events (larger for frequent updates)
)

// MessageNotifier manages notifications for new messages in chat sessions
// Used to interrupt running subagents when users send messages
// PHASE 3: Uses SafeChannel to prevent double-close panics
// PHASE 8: Increased buffer size and added statistics
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

	// PHASE 8: Create new safe channel with increased buffer size
	// Increased from 1 to MessageNotifierBufferSize to handle burst notifications
	safeCh := NewSafeChannel[struct{}](MessageNotifierBufferSize)
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
// PHASE 8: Enhanced logging with statistics
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
		// PHASE 8: Track statistics for monitoring
		if safeCh.Send(struct{}{}) {
			mn.logger.Info("Notified session of new message",
				zap.String("sessionId", sessionKey),
				zap.Int64("sentCount", safeCh.SentCount()))
		} else {
			// PHASE 8: Channel full - log warning with dropped count for visibility
			mn.logger.Warn("Notification channel full, message notification dropped",
				zap.String("sessionId", sessionKey),
				zap.Int64("droppedCount", safeCh.DroppedCount()),
				zap.Int("bufferSize", safeCh.BufferSize()),
				zap.Int("currentLen", safeCh.Len()))
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

// PHASE 8: Statistics methods for monitoring

// MessageNotifierStats contains aggregate statistics for all channels
type MessageNotifierStats struct {
	ActiveSessions    int                          // Number of active notification channels
	SessionStats      map[string]SafeChannelStats  // Per-session statistics
	TotalSent         int64                        // Total notifications sent across all sessions
	TotalDropped      int64                        // Total notifications dropped across all sessions
}

// GetStats returns aggregate statistics for all notification channels
// PHASE 8: Provides visibility into notification health
func (mn *MessageNotifier) GetStats() MessageNotifierStats {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	stats := MessageNotifierStats{
		ActiveSessions: len(mn.channels),
		SessionStats:   make(map[string]SafeChannelStats),
	}

	for sessionKey, safeCh := range mn.channels {
		channelStats := safeCh.Stats()
		stats.SessionStats[sessionKey] = channelStats
		stats.TotalSent += channelStats.SentCount
		stats.TotalDropped += channelStats.DroppedCount
	}

	return stats
}

// GetSessionStats returns statistics for a specific session
// PHASE 8: Per-session visibility
func (mn *MessageNotifier) GetSessionStats(sessionID primitive.ObjectID) (SafeChannelStats, bool) {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	sessionKey := sessionID.Hex()
	if safeCh, exists := mn.channels[sessionKey]; exists {
		return safeCh.Stats(), true
	}
	return SafeChannelStats{}, false
}
