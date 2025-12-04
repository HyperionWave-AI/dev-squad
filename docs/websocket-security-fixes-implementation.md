# WebSocket Security Fixes - Implementation Plan

## Security Concerns Overview

| Issue | Severity | Description | Status |
|-------|----------|-------------|--------|
| Error message information leak | Low | "Invalid sessionId", "Missing sessionId" help attackers enumerate valid sessions. | ✅ Phase 1 COMPLETED |
| No WebSocket frame rate limiting | Medium | Only message-level limiting. High-frequency ping/pong could be exploited for DoS. | ✅ Phase 2 COMPLETED |
| Session validation depends on prior auth | Low | No redundant validation that user owns session in WebSocket handler itself. | ✅ Phase 3 COMPLETED |

---

## Phase 1: WebSocket Frame Rate Limiting (Medium Severity)

### Problem

Currently, only message-level rate limiting exists. An attacker could:
- Send high-frequency WebSocket frames (not full messages) to exhaust server resources
- Abuse ping/pong mechanism to keep connections alive while consuming CPU
- Flood with malformed frames that get rejected but still consume processing time

### Solution

Add frame-level rate limiting using a token bucket algorithm per connection:
- Track frames received per second
- Disconnect clients that exceed threshold
- Log suspicious activity for monitoring

### Implementation

**File:** `internal/handlers/chat_websocket.go`

```go
// Frame rate limiting constants
const (
    maxFramesPerSecond    = 60   // Max frames per second per connection
    frameRateBucketSize   = 100  // Token bucket size (burst allowance)
    frameRateViolationMax = 3    // Disconnection after N violations
)

// FrameRateLimiter tracks frame rate per connection
type FrameRateLimiter struct {
    tokens           float64
    lastRefill       time.Time
    violations       int
    mu               sync.Mutex
}

func (f *FrameRateLimiter) Allow() bool {
    // Token bucket algorithm
    // Returns false if rate exceeded
}
```

### Affected Code Locations

| File | Line | Change |
|------|------|--------|
| `chat_websocket.go` | ~1998 | Add frame counting before `ReadMessage()` |
| `chat_websocket.go` | ~1889 | Add rate limiting to pong handler |

---

## Phase 2: Error Message Information Leak Fix (Low Severity)

### Problem

Specific error messages reveal information to attackers:
- `"Missing sessionId parameter"` - Confirms parameter name
- `"Invalid sessionId"` - Confirms format validation exists
- `"Invalid session ID format. Use a valid ObjectID hex string"` - Reveals MongoDB ObjectID format

This enables:
- Session ID enumeration attacks
- Understanding of backend technology (MongoDB)
- Mapping of validation logic

### Solution

Replace specific error messages with generic ones:
- All session-related errors return the same generic message
- Log detailed errors server-side for debugging
- Use consistent HTTP status codes

### Before/After

| Before | After |
|--------|-------|
| `"Missing sessionId parameter"` | `"Invalid request"` |
| `"Invalid sessionId"` | `"Invalid request"` |
| `"Invalid session ID format..."` | `"Invalid request"` |
| `"Session not found or access denied"` | `"Invalid request"` |

### Implementation

**File:** `internal/handlers/chat_websocket.go`

```go
// Generic error response for security (prevents information leakage)
const (
    errInvalidRequest = "Invalid request"
    errUnauthorized   = "Unauthorized"
    errServerError    = "Internal server error"
)
```

### Affected Code Locations

| File | Line | Current Error | New Error |
|------|------|---------------|-----------|
| `chat_websocket.go` | 1764 | `"Missing sessionId parameter"` | `errInvalidRequest` |
| `chat_websocket.go` | 1770 | `"Invalid sessionId"` | `errInvalidRequest` |
| `chat_websocket.go` | 1777 | `"Session not found or access denied"` | `errInvalidRequest` |
| `chat_handler.go` | multiple | Various specific errors | Generic errors |

---

## Phase 3: Redundant Session Ownership Validation (Low Severity) ✅ COMPLETED

### Problem

Current flow:
1. Auth middleware validates JWT and extracts userId
2. WebSocket handler trusts this and checks session.UserID == userId

If auth middleware is bypassed or misconfigured, session access could leak.

### Solution

Add redundant validation in the WebSocket handler:
1. Fresh session fetch via database query (not from cache)
2. Verify session ownership (userID matches)
3. Verify company matches

### Implementation

**File:** `internal/handlers/chat_websocket.go`

```go
// SECURITY PHASE 3: Redundant session ownership validation
// validateSessionOwnership performs a belt-and-suspenders validation of session ownership
// This is called AFTER the initial validation in HandleChatWebSocket for defense in depth
// Returns error if validation fails (session not found, ownership mismatch, or company mismatch)
func (h *ChatWebSocketHandler) validateSessionOwnership(ctx context.Context, sessionID primitive.ObjectID, userID, companyID string) error {
    // Fresh database query (not from cache) to verify session still exists and belongs to user
    session, err := h.chatService.GetSession(ctx, sessionID, companyID)
    if err != nil {
        // SECURITY: Log detailed error server-side, return generic error
        h.logger.Warn("SECURITY: Session ownership validation failed - session fetch error",
            zap.String("sessionId", sessionID.Hex()),
            zap.String("userId", userID),
            zap.Error(err))
        return fmt.Errorf("session validation failed")
    }

    // Verify user still owns this session
    if session.UserID != userID {
        // SECURITY: Potential unauthorized access - log as warning
        h.logger.Warn("SECURITY: Session ownership validation failed - user mismatch",
            zap.String("sessionId", sessionID.Hex()),
            zap.String("requestUserId", userID),
            zap.String("sessionOwnerId", session.UserID))
        metrics.WebSocketOwnershipViolations.Inc()
        return fmt.Errorf("session ownership mismatch")
    }

    // Verify company still matches
    if session.CompanyID != companyID {
        // SECURITY: Cross-company access attempt - log as warning
        h.logger.Warn("SECURITY: Session ownership validation failed - company mismatch",
            zap.String("sessionId", sessionID.Hex()),
            zap.String("requestCompanyId", companyID),
            zap.String("sessionCompanyId", session.CompanyID))
        metrics.WebSocketOwnershipViolations.Inc()
        return fmt.Errorf("session company mismatch")
    }

    return nil
}
```

### Integration in handleMessages

```go
// At the start of handleMessages:
// SECURITY PHASE 3: Redundant session ownership validation (belt and suspenders)
// This is called AFTER the initial validation in HandleChatWebSocket for defense in depth
if err := h.validateSessionOwnership(httpCtx, sessionID, userID, companyID); err != nil {
    h.logger.Warn("SECURITY: Redundant session validation failed - closing connection",
        zap.String("sessionId", sessionID.Hex()),
        zap.String("userId", userID),
        zap.Error(err))
    // Send error to client and close (using generic error for security)
    errMsg := models.StreamMessage{
        Type:  "error",
        Error: errInvalidRequest,
    }
    h.safeWriteJSON(conn, errMsg)
    return
}
```

### Metrics Added

**File:** `internal/metrics/registry.go`

```go
// SECURITY PHASE 3: Session ownership validation failures
WebSocketOwnershipViolations = prometheus.NewCounter(prometheus.CounterOpts{
    Name: "chat_websocket_ownership_violations_total",
    Help: "Total number of session ownership validation failures (potential unauthorized access attempts)",
})
```

### Affected Code Locations

| File | Line | Change |
|------|------|--------|
| `chat_websocket.go` | ~379-417 | Added validateSessionOwnership method |
| `chat_websocket.go` | ~2054-2069 | Called redundant validation at start of handleMessages |
| `registry.go` | ~77-80 | Added WebSocketOwnershipViolations metric |
| `registry.go` | ~291 | Registered the metric |

---

## Implementation Order

1. **Phase 2 first** (easiest, low risk) - Error message fixes
2. **Phase 1 second** (medium complexity) - Frame rate limiting
3. **Phase 3 last** (requires careful testing) - Redundant validation

---

## Testing Strategy

### Phase 1 Tests
```bash
# Test frame flooding
wscat -c ws://localhost:5555/api/v1/chat/stream?sessionId=xxx
# Send rapid frames and verify disconnection
```

### Phase 2 Tests
```bash
# Verify generic errors
curl "http://localhost:5555/api/v1/chat/stream?sessionId="
curl "http://localhost:5555/api/v1/chat/stream?sessionId=invalid"
curl "http://localhost:5555/api/v1/chat/stream?sessionId=000000000000000000000000"
# All should return same generic error
```

### Phase 3 Tests
```bash
# Test with manipulated session ID
# Verify redundant validation catches mismatches
```

---

## Metrics to Add

```go
// Prometheus metrics for security monitoring
WebSocketFrameRateViolations = prometheus.NewCounterVec(...)
WebSocketInvalidRequests = prometheus.NewCounterVec(...)
WebSocketOwnershipViolations = prometheus.NewCounter(...)
```

---

## Rollback Plan

Each phase is independent. If issues arise:
1. Revert specific phase changes
2. Feature flags can disable new validation (Phase 3)
3. Rate limits can be increased if legitimate users affected (Phase 1)
