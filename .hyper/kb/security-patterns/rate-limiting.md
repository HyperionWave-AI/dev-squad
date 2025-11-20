# Hyperion Rate Limiting - Token Bucket Algorithm

**Collection:** security-patterns
**Tags:** rate-limiting, token-bucket, DDoS-protection
**File Reference:** middleware/rate_limiter.go
**Version:** 1.0

---

HYPERION RATE LIMITING - TOKEN BUCKET ALGORITHM

Rate Limiter Implementation (middleware/rate_limiter.go:12-47):

TOKEN BUCKET PATTERN:
Algorithm: Per-user token bucket rate limiting
- Burst capacity: maxTokens (e.g., 10 requests)
- Refill rate: time interval (e.g., per-minute)
- Example: NewRateLimiter(10, time.Minute) = 10 requests/minute

BUCKET STRUCTURE:
```go
bucket {
  tokens: int              // Current tokens available
  maxTokens: int           // Burst capacity
  lastRefill: time.Time    // Last token refresh timestamp
  mu: sync.Mutex           // Protects token modifications
}
```

RATE LIMITER COMPONENTS:
- buckets: map[string]*bucket (per client/user tracking)
- mu: sync.RWMutex (protects map)
- maxTokens: burst capacity
- refillRate: time.Duration (e.g., time.Minute)
- cleanupStop: channel for graceful shutdown
- Background cleanup: Prevents memory leaks from expired buckets

ENFORCEMENT:
Middleware() returns gin.HandlerFunc:
- Per-request token deduction
- Returns 429 Too Many Requests if no tokens available
- Automatic refilling on interval

PERFORMANCE:
- Token bucket: O(1) per request
- Background cleanup: Goroutine removes inactive clients
- Memory efficient: Only tracks active clients

CONFIGURATION:
- maxTokens: Burst size (10, 100, etc.)
- refillRate: Refill window (time.Minute, time.Hour, etc.)
- Per-user tracking: Different buckets per userId/clientId

USE CASES:
- API endpoint protection
- DDoS mitigation
- Fair resource allocation
- Prevent abuse of expensive operations
