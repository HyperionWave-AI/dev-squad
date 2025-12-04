package handlers

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// SECURITY: Frame rate limiting constants to prevent DoS attacks
const (
	// frameRateLimit is the maximum frames per second allowed per connection
	frameRateLimit = 60.0
	// frameRateBurst is the token bucket burst size (allows brief spikes)
	frameRateBurst = 100.0
	// frameRateRefillInterval is how often tokens are refilled
	frameRateRefillInterval = time.Second
	// frameRateMaxViolations is how many violations before disconnection
	frameRateMaxViolations = 3
)

// FrameRateLimiter implements token bucket rate limiting for WebSocket frames
// SECURITY: Prevents clients from flooding the server with high-frequency frames
type FrameRateLimiter struct {
	tokens     float64
	lastRefill time.Time
	violations int
	mu         sync.Mutex
	sessionID  string
	logger     *zap.Logger
}

// NewFrameRateLimiter creates a new frame rate limiter for a connection
func NewFrameRateLimiter(sessionID string, logger *zap.Logger) *FrameRateLimiter {
	return &FrameRateLimiter{
		tokens:     frameRateBurst,
		lastRefill: time.Now(),
		sessionID:  sessionID,
		logger:     logger,
	}
}

// Allow checks if a frame is allowed under the rate limit
// Returns true if allowed, false if rate limited
// SECURITY: Uses token bucket algorithm for smooth rate limiting
func (f *FrameRateLimiter) Allow() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(f.lastRefill)

	// Refill tokens based on elapsed time
	tokensToAdd := elapsed.Seconds() * frameRateLimit
	newTokens := f.tokens + tokensToAdd
	if newTokens > frameRateBurst {
		f.tokens = frameRateBurst
	} else {
		f.tokens = newTokens
	}
	f.lastRefill = now

	// Check if we have tokens available
	if f.tokens >= 1.0 {
		f.tokens -= 1.0
		return true
	}

	// Rate limited - record violation
	f.violations++
	f.logger.Warn("SECURITY: Frame rate limit exceeded",
		zap.String("sessionId", f.sessionID),
		zap.Int("violations", f.violations),
		zap.Float64("tokensRemaining", f.tokens))

	return false
}

// ShouldDisconnect returns true if too many rate limit violations occurred
// SECURITY: Disconnect abusive clients after repeated violations
func (f *FrameRateLimiter) ShouldDisconnect() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.violations >= frameRateMaxViolations
}

// GetViolations returns the current violation count
func (f *FrameRateLimiter) GetViolations() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.violations
}

// PHASE 3 Backpressure: SlowClientDetector tracks client write performance
type SlowClientDetector struct {
	consecutiveSlowWrites int
	queueDepthWarnings    int
	mu                    sync.Mutex
	onSlowClient          func(reason string) // Callback when client deemed too slow
	logger                *zap.Logger
}

// NewSlowClientDetector creates a slow client detector with the given callback
// PHASE 3 Backpressure: Monitors client responsiveness
func NewSlowClientDetector(logger *zap.Logger, onSlowClient func(reason string)) *SlowClientDetector {
	return &SlowClientDetector{
		logger:       logger,
		onSlowClient: onSlowClient,
	}
}

// RecordWrite records a write and checks if client is too slow
// PHASE 3 Backpressure: Tracks consecutive slow writes
func (scd *SlowClientDetector) RecordWrite(duration time.Duration) {
	scd.mu.Lock()
	defer scd.mu.Unlock()

	if duration > SlowWriteThreshold {
		scd.consecutiveSlowWrites++
		scd.logger.Debug("Slow write recorded",
			zap.Duration("duration", duration),
			zap.Int("consecutiveSlowWrites", scd.consecutiveSlowWrites),
			zap.Int("threshold", MaxConsecutiveSlowWrites))

		if scd.consecutiveSlowWrites >= MaxConsecutiveSlowWrites {
			scd.logger.Warn("Client deemed too slow - too many consecutive slow writes",
				zap.Int("consecutiveSlowWrites", scd.consecutiveSlowWrites))
			if scd.onSlowClient != nil {
				scd.onSlowClient("consecutive_slow_writes")
			}
		}
	} else {
		// Reset on fast write
		scd.consecutiveSlowWrites = 0
	}
}

// RecordQueueFull records a queue full event
// PHASE 3 Backpressure: Tracks queue overflow events
func (scd *SlowClientDetector) RecordQueueFull() {
	scd.mu.Lock()
	defer scd.mu.Unlock()

	scd.queueDepthWarnings++
	scd.logger.Debug("Queue full recorded",
		zap.Int("queueDepthWarnings", scd.queueDepthWarnings),
		zap.Int("threshold", MaxQueueDepthWarnings))

	if scd.queueDepthWarnings >= MaxQueueDepthWarnings {
		scd.logger.Warn("Client deemed too slow - too many queue full events",
			zap.Int("queueDepthWarnings", scd.queueDepthWarnings))
		if scd.onSlowClient != nil {
			scd.onSlowClient("queue_depth_exceeded")
		}
	}
}

// Reset clears the slow client detection counters
func (scd *SlowClientDetector) Reset() {
	scd.mu.Lock()
	defer scd.mu.Unlock()
	scd.consecutiveSlowWrites = 0
	scd.queueDepthWarnings = 0
}
