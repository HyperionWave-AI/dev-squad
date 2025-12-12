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
// PHASE 8: Increased buffer size and added statistics
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

	// PHASE 8: Create new safe channel with configured buffer size
	// Uses ProgressNotifierBufferSize from message_notifier.go for consistency
	safeCh := NewSafeChannel[ProgressEvent](ProgressNotifierBufferSize)
	pn.channels[sessionKey] = safeCh
	pn.logger.Info("Registered session for progress notifications",
		zap.String("sessionId", sessionKey),
		zap.Int("bufferSize", ProgressNotifierBufferSize))
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
// PHASE 8: Enhanced logging with statistics
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

		// PHASE 8: Track statistics for monitoring
		if safeCh.Send(ProgressEvent{Message: message}) {
			pn.logger.Debug("Emitted progress event",
				zap.String("sessionId", sessionKey),
				zap.String("message", message),
				zap.Int64("sentCount", safeCh.SentCount()))
		} else {
			// PHASE 8: Log warning with statistics for visibility
			pn.logger.Warn("Progress channel full, dropping event",
				zap.String("sessionId", sessionKey),
				zap.String("message", message),
				zap.Int64("droppedCount", safeCh.DroppedCount()),
				zap.Int("bufferSize", safeCh.BufferSize()),
				zap.Int("currentLen", safeCh.Len()))
		}
	}
}

// PHASE 8: Statistics methods for monitoring

// ProgressNotifierStats contains aggregate statistics for all progress channels
type ProgressNotifierStats struct {
	ActiveSessions int                         // Number of active progress channels
	SessionStats   map[string]SafeChannelStats // Per-session statistics
	TotalSent      int64                       // Total events sent across all sessions
	TotalDropped   int64                       // Total events dropped across all sessions
}

// GetStats returns aggregate statistics for all progress channels
// PHASE 8: Provides visibility into progress notification health
func (pn *ProgressNotifier) GetStats() ProgressNotifierStats {
	pn.mu.RLock()
	defer pn.mu.RUnlock()

	stats := ProgressNotifierStats{
		ActiveSessions: len(pn.channels),
		SessionStats:   make(map[string]SafeChannelStats),
	}

	for sessionKey, safeCh := range pn.channels {
		channelStats := safeCh.Stats()
		stats.SessionStats[sessionKey] = channelStats
		stats.TotalSent += channelStats.SentCount
		stats.TotalDropped += channelStats.DroppedCount
	}

	return stats
}

// GetSessionStats returns statistics for a specific session
// PHASE 8: Per-session visibility
func (pn *ProgressNotifier) GetSessionStats(sessionID primitive.ObjectID) (SafeChannelStats, bool) {
	pn.mu.RLock()
	defer pn.mu.RUnlock()

	sessionKey := sessionID.Hex()
	if safeCh, exists := pn.channels[sessionKey]; exists {
		return safeCh.Stats(), true
	}
	return SafeChannelStats{}, false
}
