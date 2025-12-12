package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PHASE 2: Redis-backed distributed rate limiter implementation

// Lua script for atomic token bucket rate limiting
// Returns: [allowed (0/1), tokens_remaining, retry_after_seconds]
// PHASE 2: Atomic operations prevent race conditions across distributed instances
const rateLimitScript = `
local key = KEYS[1]
local max_tokens = tonumber(ARGV[1])
local refill_rate_ms = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])

-- Get current state
local state = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(state[1]) or max_tokens
local last_refill = tonumber(state[2]) or now_ms

-- Calculate token refill
local elapsed = now_ms - last_refill
local periods = math.floor(elapsed / refill_rate_ms)

if periods > 0 then
    tokens = math.min(tokens + (periods * max_tokens), max_tokens)
    last_refill = last_refill + (periods * refill_rate_ms)
end

-- Check if request allowed
if tokens > 0 then
    tokens = tokens - 1
    redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
    redis.call('PEXPIRE', key, refill_rate_ms * 2)
    return {1, tokens, 0}
else
    -- Calculate retry after
    local time_until_refill = refill_rate_ms - (now_ms - last_refill)
    local retry_after = math.ceil(time_until_refill / 1000)
    if retry_after < 1 then retry_after = 1 end
    return {0, 0, retry_after}
end
`

// RedisRateLimiter implements distributed rate limiting using Redis
// PHASE 2: Uses Lua script for atomic token bucket operations
type RedisRateLimiter struct {
	client     *redis.Client
	maxTokens  int
	refillRate time.Duration
	keyPrefix  string
	logger     *zap.Logger
	scriptSHA  string
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
// PHASE 2: Loads Lua script on initialization for performance
func NewRedisRateLimiter(client *redis.Client, maxTokens int, refillRate time.Duration, logger *zap.Logger) (*RedisRateLimiter, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	rl := &RedisRateLimiter{
		client:     client,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		keyPrefix:  "ratelimit:",
		logger:     logger,
	}

	// Load Lua script into Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sha, err := client.ScriptLoad(ctx, rateLimitScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load rate limit script: %w", err)
	}
	rl.scriptSHA = sha

	logger.Info("Redis rate limiter initialized",
		zap.Int("maxTokens", maxTokens),
		zap.Duration("refillRate", refillRate),
		zap.String("scriptSHA", sha[:8]+"..."))

	return rl, nil
}

// Middleware returns a Gin middleware function that enforces rate limiting
// PHASE 2: Same interface as in-memory RateLimiter for compatibility
func (r *RedisRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		r.logger.Debug("Redis rate limiter middleware invoked",
			zap.String("path", c.FullPath()),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()))

		// Extract userId from context (set by auth middleware)
		userIDInterface, exists := c.Get("userId")
		if !exists {
			r.logger.Error("Redis rate limiter: userId not found in context - auth middleware must run first")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error - authentication required",
			})
			c.Abort()
			return
		}

		userID, ok := userIDInterface.(string)
		if !ok || userID == "" {
			r.logger.Error("Redis rate limiter: invalid userId format in context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error - invalid user identity",
			})
			c.Abort()
			return
		}

		// Check rate limit using Redis
		allowed, tokensRemaining, retryAfter, err := r.allow(c.Request.Context(), userID)
		if err != nil {
			r.logger.Error("Redis rate limit check failed, allowing request (fail-open)",
				zap.String("userId", userID),
				zap.Error(err))
			// On Redis error, allow request (fail open for availability)
			c.Next()
			return
		}

		r.logger.Debug("Redis rate limiter decision",
			zap.String("userId", userID),
			zap.Bool("allowed", allowed),
			zap.Int("tokensRemaining", tokensRemaining),
			zap.Int("retryAfter", retryAfter))

		if !allowed {
			r.logger.Warn("Rate limit exceeded (Redis)",
				zap.String("userId", userID),
				zap.String("endpoint", c.FullPath()),
				zap.String("method", c.Request.Method),
				zap.Int("retryAfter", retryAfter))

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":      "Rate limit exceeded. Maximum requests per minute reached.",
				"retryAfter": retryAfter,
			})
			c.Abort()
			return
		}

		r.logger.Debug("Request allowed by Redis rate limiter",
			zap.String("userId", userID),
			zap.Int("tokensRemaining", tokensRemaining))

		c.Next()
	}
}

// allow checks if a request is allowed for the given user using Redis
// Returns (allowed, tokensRemaining, retryAfterSeconds, error)
// PHASE 2: Uses EVALSHA for efficient Lua script execution
func (r *RedisRateLimiter) allow(ctx context.Context, userID string) (bool, int, int, error) {
	key := r.keyPrefix + userID
	nowMs := time.Now().UnixMilli()
	refillRateMs := r.refillRate.Milliseconds()

	// Execute Lua script atomically
	result, err := r.client.EvalSha(ctx, r.scriptSHA, []string{key},
		r.maxTokens, refillRateMs, nowMs).Result()
	if err != nil {
		// Script might not be loaded (Redis restart), try reloading
		if err.Error() == "NOSCRIPT No matching script. Please use EVAL." {
			sha, loadErr := r.client.ScriptLoad(ctx, rateLimitScript).Result()
			if loadErr != nil {
				return false, 0, 0, fmt.Errorf("failed to reload script: %w", loadErr)
			}
			r.scriptSHA = sha
			r.logger.Info("Reloaded rate limit script after NOSCRIPT error",
				zap.String("newSHA", sha[:8]+"..."))

			// Retry with new SHA
			result, err = r.client.EvalSha(ctx, r.scriptSHA, []string{key},
				r.maxTokens, refillRateMs, nowMs).Result()
			if err != nil {
				return false, 0, 0, err
			}
		} else {
			return false, 0, 0, err
		}
	}

	// Parse results: [allowed, tokens_remaining, retry_after]
	results, ok := result.([]interface{})
	if !ok || len(results) < 3 {
		return false, 0, 0, fmt.Errorf("unexpected script result format")
	}

	allowed := results[0].(int64) == 1
	tokensRemaining := int(results[1].(int64))
	retryAfter := int(results[2].(int64))

	return allowed, tokensRemaining, retryAfter, nil
}

// Stop is a no-op for Redis rate limiter
// PHASE 2: Cleanup handled by Redis TTL, no background goroutines
func (r *RedisRateLimiter) Stop() {
	// No cleanup needed - Redis handles TTL automatically
}
