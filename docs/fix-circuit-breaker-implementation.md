# Add Circuit Breaker for Stuck/Slow Clients - Implementation Plan

## Problem Statement

Slow or stuck WebSocket clients can cause:
- **Resource exhaustion**: Server holds open connections indefinitely
- **Goroutine leaks**: Blocked writes accumulate goroutines
- **Memory pressure**: Buffered messages pile up
- **Cascading failures**: One slow client affects others

### Current Protections (Before Circuit Breaker)

| Protection | What It Does | Limitation |
|------------|--------------|------------|
| Write deadlines | Timeout after 10s | Still attempts each write |
| Slow client detector | Counts slow writes | Only logs, doesn't block |
| Write queue | Drops non-critical messages | Keeps trying writes |
| Health monitor | Tracks buffer pressure | Only for monitoring |

**Gap**: Server keeps attempting writes even when a client is consistently slow.

---

## Design Goals

1. **Fast failure**: Skip writes to known-slow clients
2. **Self-healing**: Automatically retry after cooldown
3. **Graceful recovery**: Allow test writes in half-open state
4. **Per-connection**: Isolate slow clients, don't affect healthy ones
5. **Observable**: Metrics and logging for visibility

---

## Circuit Breaker Pattern

The circuit breaker has three states:

```
     [Success]        [Cooldown Expired]        [Success x N]
         │                    │                       │
         ▼                    ▼                       ▼
┌────────────────┐    ┌───────────────┐    ┌─────────────────┐
│     CLOSED     │───▶│     OPEN      │───▶│    HALF-OPEN    │
│  (normal ops)  │    │ (blocking)    │    │   (testing)     │
└────────────────┘    └───────────────┘    └─────────────────┘
         ▲                    ▲                       │
         │                    │                       │
         │            [Any Failure]                   │
         │                    └───────────────────────┘
         │                           [Success x N]
         └─────────────────────────────────────────────
```

### State Transitions

| From | To | Trigger |
|------|------|---------|
| CLOSED | OPEN | 5 consecutive failures OR 80% slow calls |
| OPEN | HALF-OPEN | 30 seconds elapsed |
| HALF-OPEN | CLOSED | 3 consecutive successes |
| HALF-OPEN | OPEN | Any failure |

---

## Implementation

### Phase 1: Circuit Breaker Core (internal/handlers/circuit_breaker.go)

**CircuitBreakerConfig**
```go
type CircuitBreakerConfig struct {
    FailureThreshold      int           // 5 failures to open
    SuccessThreshold      int           // 3 successes to close
    OpenTimeout           time.Duration // 30s before half-open
    SlowCallThreshold     time.Duration // 2s = slow call
    SlowCallRateThreshold int           // 80% slow = open
    MinCallsBeforeTrip    int           // Need 3 calls first
}
```

**CircuitBreaker**
```go
type CircuitBreaker struct {
    state            atomic.Int32  // CLOSED, OPEN, HALF_OPEN
    failures         atomic.Int64  // Failure counter
    successes        atomic.Int64  // Success counter
    slowCalls        atomic.Int64  // Slow call counter
    consecutiveFail  atomic.Int64  // Consecutive failures
    openedAt         atomic.Int64  // When circuit opened
}

// Key methods
func (cb *CircuitBreaker) AllowRequest() bool
func (cb *CircuitBreaker) RecordSuccess(duration time.Duration)
func (cb *CircuitBreaker) RecordFailure(err error)
func (cb *CircuitBreaker) RecordSlowCall(duration time.Duration)
func (cb *CircuitBreaker) RecordTimeout()
```

**CircuitBreakerRegistry**
```go
// Global registry - one circuit breaker per session
var circuitBreakerRegistry *CircuitBreakerRegistry

func GetCircuitBreakerRegistry(logger) *CircuitBreakerRegistry
func (r *CircuitBreakerRegistry) Get(sessionID string) *CircuitBreaker
func (r *CircuitBreakerRegistry) Remove(sessionID string)
func (r *CircuitBreakerRegistry) GetOpenCircuits() []string
```

### Phase 2: Integration with safeWriteJSONWithMonitoring

```go
func (h *ChatWebSocketHandler) safeWriteJSONWithMonitoring(conn, msg, sessionID) error {
    // PHASE 5: Check circuit breaker before attempting write
    var circuitBreaker *CircuitBreaker
    if sessionID != "" {
        registry := GetCircuitBreakerRegistry(h.logger)
        circuitBreaker = registry.Get(sessionID)

        if !circuitBreaker.AllowRequest() {
            metrics.WebSocketCircuitBreakerTrips.Inc()
            return ErrCircuitOpen
        }
    }

    // ... existing write logic ...

    // PHASE 5: Record result in circuit breaker
    if circuitBreaker != nil {
        if err != nil {
            if isTimeoutError(err) {
                circuitBreaker.RecordTimeout()
            } else {
                circuitBreaker.RecordFailure(err)
            }
        } else if duration > SlowWriteThreshold {
            circuitBreaker.RecordSlowCall(duration)
        } else {
            circuitBreaker.RecordSuccess(duration)
        }
    }

    return err
}
```

### Phase 3: Connection Lifecycle Integration

```go
// In handleMessages:
// Circuit breaker is auto-created on first write, cleaned up on disconnect
circuitBreakerRegistry := GetCircuitBreakerRegistry(h.logger)
defer circuitBreakerRegistry.Remove(sessionID.Hex())
```

---

## Configuration

### Default Values

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| FailureThreshold | 5 | Trip after 5 consecutive failures |
| SuccessThreshold | 3 | Need 3 good writes to close |
| OpenTimeout | 30s | Test recovery every 30 seconds |
| SlowCallThreshold | 2s | Writes > 2s are "slow" |
| SlowCallRateThreshold | 80% | Trip if 80% of calls are slow |
| MinCallsBeforeTrip | 3 | Need 3 calls before circuit can trip |

### Environment Variables (Future)

```bash
# Override circuit breaker defaults
CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
CIRCUIT_BREAKER_OPEN_TIMEOUT=30s
CIRCUIT_BREAKER_SLOW_THRESHOLD=2s
```

---

## Metrics

### New Prometheus Metrics

```promql
# Total times circuit breaker blocked a request
chat_websocket_circuit_breaker_trips_total

# Combined with existing metrics:
chat_websocket_write_timeouts_total    # Triggers circuit breaker
chat_websocket_slow_writes_total       # Affects slow call rate
chat_websocket_slow_clients_total      # Counts disconnected slow clients
```

### Alerting Examples

```yaml
# Alert when circuit breakers are tripping frequently
- alert: WebSocketCircuitBreakerTrips
  expr: rate(chat_websocket_circuit_breaker_trips_total[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "WebSocket circuit breakers are actively blocking slow clients"
```

---

## Log Messages

```
# Circuit breaker blocking writes
"Circuit breaker open - blocking write" sessionId=xxx state=OPEN

# State transitions
"Circuit breaker state transition" name=xxx from=CLOSED to=OPEN failures=5

# Trip reasons
"Circuit breaker tripping on consecutive failures" consecutiveFailures=5
"Circuit breaker tripping on slow call rate" slowCallRate=85 threshold=80

# Recovery
"Circuit breaker state transition" from=HALF_OPEN to=CLOSED successes=3
```

---

## Testing Strategy

1. **Unit Test**: State transitions work correctly
2. **Integration Test**: Circuit trips after threshold
3. **Recovery Test**: Circuit closes after successful writes
4. **Slow Client Test**: Simulate slow client, verify trip

---

## Build Verification

```
make build
✓ Build complete: bin/hyper
```

---

## Files Changed

| File | Change |
|------|--------|
| internal/handlers/circuit_breaker.go | NEW - Circuit breaker implementation |
| internal/handlers/chat_websocket.go | Integration with safeWriteJSONWithMonitoring |
| internal/metrics/registry.go | WebSocketCircuitBreakerTrips metric |

---

## Implementation Complete

### Summary

The circuit breaker protects the server from slow/stuck WebSocket clients by:

1. **Tracking failures**: Counts consecutive timeouts/errors
2. **Tracking slow calls**: Monitors write latency
3. **Auto-tripping**: Opens circuit after threshold exceeded
4. **Fast failure**: Returns immediately when open (no wasted writes)
5. **Self-healing**: Tests recovery after cooldown period
6. **Per-session isolation**: Each client has own circuit breaker

### Key Benefits

- **Resource protection**: Don't waste time on known-slow clients
- **Automatic recovery**: Self-heals when client recovers
- **Observable**: Metrics and logging for troubleshooting
- **No configuration required**: Sensible defaults work out of box
- **Minimal overhead**: Atomic operations, no locks in hot path

### Interaction with Existing Protections

```
Request Flow:
    │
    ▼
[Circuit Breaker] ─── OPEN ───▶ Return ErrCircuitOpen (fast)
    │
    │ CLOSED/HALF-OPEN
    ▼
[Write Deadline] ─── Timeout ──▶ RecordTimeout() ──▶ May trip circuit
    │
    │ Success
    ▼
[Health Monitor] ─── Record metrics ──▶ For observability
    │
    ▼
[Success/SlowCall recorded] ─── Updates circuit breaker state
```
