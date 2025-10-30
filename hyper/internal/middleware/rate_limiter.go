package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimiter implements a token bucket rate limiter for API endpoints
type RateLimiter struct {
	buckets     map[string]*bucket
	mu          sync.RWMutex
	maxTokens   int
	refillRate  time.Duration
	logger      *zap.Logger
	cleanupStop chan struct{}
}

// bucket represents a token bucket for rate limiting
type bucket struct {
	tokens     int
	maxTokens  int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// maxTokens: maximum number of requests allowed (burst capacity)
// refillRate: time period for refilling tokens (e.g., time.Minute for per-minute limiting)
// Example: NewRateLimiter(10, time.Minute, logger) = 10 requests per minute
func NewRateLimiter(maxTokens int, refillRate time.Duration, logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		buckets:     make(map[string]*bucket),
		maxTokens:   maxTokens,
		refillRate:  refillRate,
		logger:      logger,
		cleanupStop: make(chan struct{}),
	}

	// Start background cleanup goroutine to prevent memory leaks
	go rl.cleanupExpiredBuckets()

	return rl
}

// Middleware returns a Gin middleware function that enforces rate limiting
func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		r.logger.Info("🚦 Rate limiter middleware INVOKED",
			zap.String("path", c.FullPath()),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()))

		// Extract userId from context (set by auth middleware)
		userIDInterface, exists := c.Get("userId")
		if !exists {
			r.logger.Error("Rate limiter: userId not found in context - auth middleware must run first")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error - authentication required",
			})
			c.Abort()
			return
		}

		userID, ok := userIDInterface.(string)
		if !ok || userID == "" {
			r.logger.Error("Rate limiter: invalid userId format in context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error - invalid user identity",
			})
			c.Abort()
			return
		}

		r.logger.Info("✅ UserID extracted from context",
			zap.String("userId", userID))

		// Check if request is allowed
		allowed, retryAfter := r.allow(userID)

		r.logger.Info("🔍 Rate limiter decision",
			zap.String("userId", userID),
			zap.Bool("allowed", allowed),
			zap.Int("retryAfter", retryAfter))

		if !allowed {
			r.logger.Warn("⛔ Rate limit EXCEEDED",
				zap.String("userId", userID),
				zap.String("endpoint", c.FullPath()),
				zap.String("method", c.Request.Method),
				zap.Int("retryAfter", retryAfter))

			c.Header("Retry-After", string(rune(retryAfter)))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":      "Rate limit exceeded. Maximum requests per minute reached.",
				"retryAfter": retryAfter,
			})
			c.Abort()
			return
		}

		r.logger.Info("✅ Request ALLOWED - continuing to handler",
			zap.String("userId", userID))

		// Request allowed, continue
		c.Next()
	}
}

// allow checks if a request is allowed for the given user
// Returns (allowed, retryAfterSeconds)
func (r *RateLimiter) allow(userID string) (bool, int) {
	r.mu.Lock()
	b, exists := r.buckets[userID]
	bucketCount := len(r.buckets)
	if !exists {
		r.logger.Info("🆕 Creating NEW bucket for user",
			zap.String("userId", userID),
			zap.Int("initialTokens", r.maxTokens),
			zap.Int("totalBuckets", bucketCount+1))

		// Create new bucket for this user
		b = &bucket{
			tokens:     r.maxTokens,
			maxTokens:  r.maxTokens,
			lastRefill: time.Now(),
		}
		r.buckets[userID] = b
	} else {
		r.logger.Info("♻️  Using EXISTING bucket",
			zap.String("userId", userID),
			zap.Int("currentTokens", b.tokens),
			zap.Int("totalBuckets", bucketCount))
	}
	r.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	tokensBefore := b.tokens

	if elapsed >= r.refillRate {
		// Full refill period has elapsed
		periodsElapsed := int(elapsed / r.refillRate)
		tokensToAdd := periodsElapsed * b.maxTokens

		b.tokens = min(b.tokens+tokensToAdd, b.maxTokens)
		b.lastRefill = now

		r.logger.Info("🔄 Token refill occurred",
			zap.String("userId", userID),
			zap.Int("tokensBefore", tokensBefore),
			zap.Int("tokensAfter", b.tokens),
			zap.Int("periodsElapsed", periodsElapsed))
	}

	// Check if tokens available
	if b.tokens > 0 {
		b.tokens--
		r.logger.Info("✅ Token consumed - request ALLOWED",
			zap.String("userId", userID),
			zap.Int("tokensRemaining", b.tokens),
			zap.Int("maxTokens", b.maxTokens),
			zap.Duration("elapsedSinceRefill", elapsed))
		return true, 0
	}

	// Calculate retry-after time
	timeUntilRefill := r.refillRate - elapsed
	retryAfter := int(timeUntilRefill.Seconds()) + 1

	r.logger.Warn("⛔ NO tokens available - request DENIED",
		zap.String("userId", userID),
		zap.Int("tokensRemaining", b.tokens),
		zap.Int("retryAfter", retryAfter),
		zap.Duration("timeUntilRefill", timeUntilRefill))

	return false, retryAfter
}

// cleanupExpiredBuckets removes buckets that haven't been used in 10 minutes
// This prevents memory leaks from one-time users
func (r *RateLimiter) cleanupExpiredBuckets() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			now := time.Now()
			expiry := 10 * time.Minute

			for userID, b := range r.buckets {
				b.mu.Lock()
				if now.Sub(b.lastRefill) > expiry {
					delete(r.buckets, userID)
					r.logger.Debug("Cleaned up expired rate limit bucket",
						zap.String("userId", userID))
				}
				b.mu.Unlock()
			}
			r.mu.Unlock()

		case <-r.cleanupStop:
			return
		}
	}
}

// Stop gracefully shuts down the rate limiter
func (r *RateLimiter) Stop() {
	close(r.cleanupStop)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
