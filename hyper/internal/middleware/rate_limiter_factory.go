package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	redisclient "hyper/internal/redis"
)

// PHASE 3: Factory pattern for automatic Redis/in-memory selection

// RateLimiterInterface defines the common interface for rate limiters
// PHASE 3: Both RedisRateLimiter and RateLimiter implement this interface
type RateLimiterInterface interface {
	Middleware() gin.HandlerFunc
	Stop()
}

// NewDistributedRateLimiter creates a rate limiter, preferring Redis if available
// PHASE 3: Automatically selects Redis or in-memory based on configuration
// - If REDIS_URL is set and Redis is reachable: uses Redis (distributed)
// - Otherwise: falls back to in-memory (single instance)
func NewDistributedRateLimiter(maxTokens int, refillRate time.Duration, logger *zap.Logger) RateLimiterInterface {
	// Try Redis first
	redisClient := redisclient.GetClient(logger)
	if redisClient != nil {
		rl, err := NewRedisRateLimiter(redisClient, maxTokens, refillRate, logger)
		if err == nil {
			logger.Info("Using Redis-backed distributed rate limiter",
				zap.Int("maxTokens", maxTokens),
				zap.Duration("refillRate", refillRate))
			return rl
		}
		logger.Warn("Failed to create Redis rate limiter, falling back to in-memory",
			zap.Error(err))
	}

	// Fall back to in-memory rate limiter
	logger.Info("Using in-memory rate limiter (not distributed)",
		zap.Int("maxTokens", maxTokens),
		zap.Duration("refillRate", refillRate))
	return NewRateLimiter(maxTokens, refillRate, logger)
}
