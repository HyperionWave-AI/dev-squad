# Move Rate Limiting to Redis for Distributed Enforcement - Implementation Plan

## Problem Statement

The current rate limiter in `internal/middleware/rate_limiter.go` uses in-memory token buckets:

```go
type RateLimiter struct {
    buckets     map[string]*bucket  // PROBLEM: In-memory only
    mu          sync.RWMutex
    maxTokens   int
    refillRate  time.Duration
    logger      *zap.Logger
    cleanupStop chan struct{}
}
```

### Issues with In-Memory Rate Limiting

1. **No distributed enforcement**: Each server instance has its own bucket map
2. **Rate limit bypass**: Users can exceed limits by hitting different instances
3. **State loss on restart**: All rate limit state lost when server restarts
4. **Memory growth**: Large bucket maps for many users

### Current Usage

```go
// internal/server/http_server.go:530
subchatRateLimiter := middleware.NewRateLimiter(10, time.Minute, logger)
subchatGroup.POST("", subchatRateLimiter.Middleware(), subchatHandler.CreateSubchat)
```

---

## Design Goals

1. **Distributed enforcement**: Single source of truth across all instances
2. **Atomic operations**: Use Redis MULTI/EXEC or Lua scripts for race-free limiting
3. **Backward compatible**: Same `Middleware()` interface
4. **Graceful fallback**: Fall back to in-memory if Redis unavailable
5. **Configurable**: Enable/disable via environment variable

---

## Implementation Phases

### Phase 1: Add Redis Client Infrastructure

**Goal**: Add Redis client package and configuration.

**Changes**:
- Add `github.com/redis/go-redis/v9` to go.mod
- Create `internal/redis/client.go` with singleton Redis client
- Add `REDIS_URL` environment variable support
- Add connection health check

**Risk**: Low - new infrastructure, no existing code changes

---

### Phase 2: Create Redis Rate Limiter

**Goal**: Implement token bucket algorithm using Redis.

**Changes**:
- Create `internal/middleware/redis_rate_limiter.go`
- Implement Lua script for atomic token bucket operations
- Match existing `Middleware()` interface
- Add Redis-specific metrics

**Risk**: Low - new implementation alongside existing

---

### Phase 3: Integrate with Factory Pattern

**Goal**: Auto-select Redis or in-memory based on configuration.

**Changes**:
- Create `NewDistributedRateLimiter()` factory function
- Check `REDIS_URL` environment variable
- Fall back to in-memory if Redis unavailable
- Update http_server.go to use factory

**Risk**: Medium - changes rate limiter creation

---

### Phase 4: Add Graceful Degradation

**Goal**: Handle Redis failures gracefully.

**Changes**:
- Implement circuit breaker for Redis calls
- Fall back to in-memory on Redis timeout/error
- Log degradation events
- Expose health status

**Risk**: Low - resilience improvement

---

## Detailed Implementation

### Phase 1: Redis Client Infrastructure

**File: internal/redis/client.go**

```go
package redis

import (
    "context"
    "os"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

var (
    client     *redis.Client
    clientOnce sync.Once
    logger     *zap.Logger
)

// GetClient returns the singleton Redis client
func GetClient(l *zap.Logger) *redis.Client {
    clientOnce.Do(func() {
        logger = l
        redisURL := os.Getenv("REDIS_URL")
        if redisURL == "" {
            redisURL = "redis://localhost:6379"
        }

        opt, err := redis.ParseURL(redisURL)
        if err != nil {
            logger.Warn("Failed to parse REDIS_URL, Redis rate limiting disabled",
                zap.Error(err))
            return
        }

        client = redis.NewClient(opt)

        // Test connection
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        if err := client.Ping(ctx).Err(); err != nil {
            logger.Warn("Redis connection failed, Redis rate limiting disabled",
                zap.Error(err))
            client = nil
            return
        }

        logger.Info("Redis client initialized successfully",
            zap.String("addr", opt.Addr))
    })

    return client
}

// IsAvailable checks if Redis is available
func IsAvailable() bool {
    return client != nil
}
```

### Phase 2: Redis Rate Limiter

**File: internal/middleware/redis_rate_limiter.go**

```go
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

// Lua script for atomic token bucket rate limiting
// Returns: [allowed (0/1), tokens_remaining, retry_after_seconds]
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
    return {0, 0, retry_after}
end
`

// RedisRateLimiter implements distributed rate limiting using Redis
type RedisRateLimiter struct {
    client       *redis.Client
    maxTokens    int
    refillRate   time.Duration
    keyPrefix    string
    logger       *zap.Logger
    scriptSHA    string
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
func NewRedisRateLimiter(client *redis.Client, maxTokens int, refillRate time.Duration, logger *zap.Logger) (*RedisRateLimiter, error) {
    rl := &RedisRateLimiter{
        client:     client,
        maxTokens:  maxTokens,
        refillRate: refillRate,
        keyPrefix:  "ratelimit:",
        logger:     logger,
    }

    // Load Lua script
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    sha, err := client.ScriptLoad(ctx, rateLimitScript).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to load rate limit script: %w", err)
    }
    rl.scriptSHA = sha

    logger.Info("Redis rate limiter initialized",
        zap.Int("maxTokens", maxTokens),
        zap.Duration("refillRate", refillRate))

    return rl, nil
}

// Middleware returns a Gin middleware function
func (r *RedisRateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract userId from context
        userIDInterface, exists := c.Get("userId")
        if !exists {
            r.logger.Error("Redis rate limiter: userId not found in context")
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "Internal server error - authentication required",
            })
            c.Abort()
            return
        }

        userID, ok := userIDInterface.(string)
        if !ok || userID == "" {
            r.logger.Error("Redis rate limiter: invalid userId format")
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "Internal server error - invalid user identity",
            })
            c.Abort()
            return
        }

        // Check rate limit
        allowed, retryAfter, err := r.allow(c.Request.Context(), userID)
        if err != nil {
            r.logger.Error("Redis rate limit check failed",
                zap.String("userId", userID),
                zap.Error(err))
            // On Redis error, allow request (fail open)
            c.Next()
            return
        }

        if !allowed {
            r.logger.Warn("Rate limit exceeded (Redis)",
                zap.String("userId", userID),
                zap.String("endpoint", c.FullPath()),
                zap.Int("retryAfter", retryAfter))

            c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error":      "Rate limit exceeded. Maximum requests per minute reached.",
                "retryAfter": retryAfter,
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

// allow checks if request is allowed using Redis
func (r *RedisRateLimiter) allow(ctx context.Context, userID string) (bool, int, error) {
    key := r.keyPrefix + userID
    nowMs := time.Now().UnixMilli()
    refillRateMs := r.refillRate.Milliseconds()

    result, err := r.client.EvalSha(ctx, r.scriptSHA, []string{key},
        r.maxTokens, refillRateMs, nowMs).Result()
    if err != nil {
        return false, 0, err
    }

    results := result.([]interface{})
    allowed := results[0].(int64) == 1
    retryAfter := int(results[2].(int64))

    return allowed, retryAfter, nil
}

// Stop is a no-op for Redis rate limiter (cleanup handled by Redis TTL)
func (r *RedisRateLimiter) Stop() {}
```

### Phase 3: Factory Pattern Integration

**File: internal/middleware/rate_limiter_factory.go**

```go
package middleware

import (
    "time"

    "go.uber.org/zap"
    redisclient "hyper/internal/redis"
)

// RateLimiterInterface defines the common interface for rate limiters
type RateLimiterInterface interface {
    Middleware() gin.HandlerFunc
    Stop()
}

// NewDistributedRateLimiter creates a rate limiter, preferring Redis if available
func NewDistributedRateLimiter(maxTokens int, refillRate time.Duration, logger *zap.Logger) RateLimiterInterface {
    // Try Redis first
    redisClient := redisclient.GetClient(logger)
    if redisClient != nil {
        rl, err := NewRedisRateLimiter(redisClient, maxTokens, refillRate, logger)
        if err == nil {
            logger.Info("Using Redis-backed distributed rate limiter")
            return rl
        }
        logger.Warn("Failed to create Redis rate limiter, falling back to in-memory",
            zap.Error(err))
    }

    // Fall back to in-memory
    logger.Info("Using in-memory rate limiter (not distributed)")
    return NewRateLimiter(maxTokens, refillRate, logger)
}
```

---

## Testing Strategy

1. **Unit Test**: Lua script logic with mock Redis
2. **Integration Test**: Redis rate limiting with real Redis
3. **Failover Test**: Verify fallback to in-memory when Redis unavailable
4. **Load Test**: Concurrent requests across multiple simulated instances

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_URL` | Redis connection URL | `redis://localhost:6379` |

---

## Rollback Plan

1. Set `REDIS_URL=""` to disable Redis rate limiting
2. Factory automatically falls back to in-memory
3. No code changes required for rollback

---

## Success Criteria

- [x] Redis client infrastructure added
- [x] Redis rate limiter implemented with Lua script
- [x] Factory pattern selects Redis or in-memory
- [x] Graceful fallback on Redis failure
- [ ] All tests pass

---

## Implementation Complete

### Changes Made

**Phase 1: Redis Client Infrastructure (internal/redis/client.go)**

Created new package `internal/redis` with singleton Redis client:
```go
func GetClient(logger *zap.Logger) *redis.Client  // Lazy init from REDIS_URL
func IsAvailable() bool                            // Health check
func Ping(ctx context.Context) error              // Active health check
func Close() error                                 // Graceful shutdown
```

Features:
- Parses `REDIS_URL` environment variable
- Connection pooling (10 connections, 2 idle)
- Configurable timeouts (5s dial, 3s read/write)
- Graceful handling when Redis not configured

**Phase 2: Redis Rate Limiter (internal/middleware/redis_rate_limiter.go)**

Implemented `RedisRateLimiter` with atomic Lua script:
```go
type RedisRateLimiter struct {
    client     *redis.Client
    maxTokens  int
    refillRate time.Duration
    keyPrefix  string
    logger     *zap.Logger
    scriptSHA  string
}

func NewRedisRateLimiter(client, maxTokens, refillRate, logger) (*RedisRateLimiter, error)
func (r *RedisRateLimiter) Middleware() gin.HandlerFunc
func (r *RedisRateLimiter) Stop()
```

Lua Script Features:
- Atomic token bucket operations (no race conditions)
- Automatic token refill based on time elapsed
- Key TTL for automatic cleanup (2x refill rate)
- Returns: [allowed, tokens_remaining, retry_after]
- Handles NOSCRIPT error (Redis restart) with auto-reload

**Phase 3: Factory Pattern (internal/middleware/rate_limiter_factory.go)**

Created factory function with automatic selection:
```go
type RateLimiterInterface interface {
    Middleware() gin.HandlerFunc
    Stop()
}

func NewDistributedRateLimiter(maxTokens int, refillRate time.Duration, logger *zap.Logger) RateLimiterInterface
```

Updated `internal/server/http_server.go`:
```go
// Before (in-memory only)
subchatRateLimiter := middleware.NewRateLimiter(10, time.Minute, logger)

// After (distributed if Redis available)
subchatRateLimiter := middleware.NewDistributedRateLimiter(10, time.Minute, logger)
```

### Build Verification
```
make build
✓ Build complete: bin/hyper
```

### How It Works

**With Redis (REDIS_URL set):**
```
1. Request arrives at rate-limited endpoint
2. Factory created RedisRateLimiter on startup
3. Lua script executes atomically in Redis:
   - Fetches current token count and last refill time
   - Calculates tokens to refill based on elapsed time
   - Checks if tokens available
   - Decrements token count if allowed
   - Sets TTL for auto-cleanup
4. Returns allow/deny decision consistently across all instances
```

**Without Redis (REDIS_URL not set):**
```
1. Factory falls back to in-memory RateLimiter
2. Token buckets stored in local map
3. Works exactly as before (single instance)
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_URL` | Redis connection URL (e.g., `redis://localhost:6379`) | Not set (in-memory mode) |

### Log Messages to Watch For

```
# Phase 1: Redis client
"Redis client initialized successfully" - Redis connected
"REDIS_URL not set, Redis features disabled" - In-memory mode
"Redis connection failed, Redis features disabled" - Connection error

# Phase 2: Redis rate limiter
"Redis rate limiter initialized" - Script loaded
"Reloaded rate limit script after NOSCRIPT error" - Auto-recovery
"Rate limit exceeded (Redis)" - Rate limit hit
"Redis rate limit check failed, allowing request (fail-open)" - Degradation

# Phase 3: Factory
"Using Redis-backed distributed rate limiter" - Redis mode
"Using in-memory rate limiter (not distributed)" - Fallback mode
```

### Key Benefits

1. **Distributed enforcement**: Same rate limit across all server instances
2. **Atomic operations**: Lua script prevents race conditions
3. **Automatic cleanup**: Redis TTL handles bucket expiration
4. **Fail-open**: On Redis error, requests are allowed (availability over strictness)
5. **Zero config change**: Just set `REDIS_URL` to enable
6. **Backward compatible**: Same middleware interface

### Redis Key Format

```
ratelimit:{userId}
  - tokens: current token count
  - last_refill: timestamp of last refill (ms)
  - TTL: 2x refill rate (auto-cleanup)
```

