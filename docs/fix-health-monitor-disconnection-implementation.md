# Add Actual Disconnection Mechanism to Health Monitor - Implementation Plan

## Problem Statement

The health monitor was detecting unhealthy connections but not actually disconnecting them:
- **Detection without action**: Health issues logged but connection stayed open
- **Resource accumulation**: Unhealthy connections continued consuming resources
- **No cleanup coordination**: Health monitor couldn't signal handler to clean up
- **Manual intervention required**: Operators had to manually disconnect stuck clients

### Before Fix

```go
// Health monitor could only mark connection as unhealthy
m.isHealthy.Store(false)
m.SetDisconnectReason(reason)
// But connection stayed open!
```

**Gap**: Health monitor detected problems but couldn't act on them.

---

## PHASE 7 FIX: Timing Bug Resolution

### Bug Discovered

After implementing the disconnection mechanism, a critical timing bug was discovered:
- Health check ran every 5 seconds checking for pong timeout (10s)
- But first ping was sent at 30 seconds
- Result: All connections were disconnected at exactly 10 seconds because `lastPongTime` was initialized to connection start time, and no pings had been sent yet

### Root Cause

```
T=0s:  Connection opens, lastPongTime = now()
T=5s:  Health check: timeSinceLastPong = 5s (OK)
T=10s: Health check: timeSinceLastPong = 10s > 10s (TIMEOUT!)
       BUT pingsSent = 0, pongsReceived = 0
       Connection disconnected prematurely!
T=30s: First ping would have been sent (never reached)
```

### Fix Applied

1. **Send first ping immediately** on connection start
2. **Only check pong timeout after pings sent** (`pingsSent > 0`)
3. **Require multiple missed pongs** before disconnect (`missedPongs > 1`)
4. **Reduced ping interval** from 30s to 15s for faster detection
5. **Extracted methods** for cleaner code (`sendPing()`, `performHealthCheck()`)

---

## Design Goals

1. **Actual disconnection**: Health monitor closes unhealthy connections
2. **Handler coordination**: Notify WebSocket handler to clean up resources
3. **Graceful shutdown**: Send proper WebSocket close frame before closing
4. **One-time operation**: Ensure disconnect is called exactly once
5. **Observable**: Log and emit metrics for disconnections

---

## Implementation

### Phase 1: DisconnectCallback Type

```go
// DisconnectCallback is called when the health monitor decides to disconnect a client
// PHASE 7: Actual disconnection mechanism
type DisconnectCallback func(reason string)
```

### Phase 2: Enhanced ConnectionHealthMonitor Struct

```go
type ConnectionHealthMonitor struct {
    // ... existing fields ...

    // PHASE 7: Disconnection mechanism
    onDisconnect      DisconnectCallback // Callback to trigger actual disconnection
    disconnectOnce    sync.Once          // Ensure disconnect is only called once
    sessionID         string             // Session ID for logging
}
```

### Phase 3: Constructor with Callback

```go
// NewConnectionHealthMonitor creates a new health monitor for a WebSocket connection
// PHASE 7: Deprecated - use NewConnectionHealthMonitorWithCallback instead
func NewConnectionHealthMonitor(conn *websocket.Conn, logger *zap.Logger, writeMutex *sync.Mutex) *ConnectionHealthMonitor {
    return NewConnectionHealthMonitorWithCallback(conn, logger, writeMutex, "", nil)
}

// NewConnectionHealthMonitorWithCallback creates a new health monitor with disconnect callback
// PHASE 7: Added sessionID and onDisconnect callback for actual disconnection
func NewConnectionHealthMonitorWithCallback(conn *websocket.Conn, logger *zap.Logger, writeMutex *sync.Mutex, sessionID string, onDisconnect DisconnectCallback) *ConnectionHealthMonitor
```

### Phase 4: Trigger Disconnect Method

```go
// triggerDisconnect triggers the actual disconnection of the WebSocket connection
// PHASE 7: This method actually closes the connection and calls the callback
func (m *ConnectionHealthMonitor) triggerDisconnect(reason string) {
    m.disconnectOnce.Do(func() {
        m.isHealthy.Store(false)
        m.SetDisconnectReason(reason)

        m.logger.Warn("Health monitor triggering disconnection",
            zap.String("sessionId", m.sessionID),
            zap.String("reason", reason),
            zap.Int64("pingsSent", m.pingsSent.Load()),
            zap.Int64("pongsReceived", m.pongsReceived.Load()),
            zap.Int("bufferUsage", m.estimateBufferUsage()))

        // Record metric for health-triggered disconnection
        metrics.WebSocketSlowClients.WithLabelValues(reason).Inc()

        // Close the WebSocket connection with appropriate close code
        m.writeMutex.Lock()
        closeMsg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "Connection unhealthy: "+reason)
        _ = m.conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(5*time.Second))
        _ = m.conn.Close()
        m.writeMutex.Unlock()

        // Call the disconnect callback to notify the handler
        if m.onDisconnect != nil {
            m.onDisconnect(reason)
        }
    })
}
```

### Phase 5: Public ForceDisconnect Method

```go
// ForceDisconnect allows external callers to trigger disconnection
// PHASE 7: For use by circuit breaker or other components
func (m *ConnectionHealthMonitor) ForceDisconnect(reason string) {
    m.triggerDisconnect(reason)
}
```

### Phase 6: Monitor Loop Integration

The `monitorLoop()` now calls `triggerDisconnect()` instead of just returning:

```go
// Disconnect reasons handled:
// - "ping_write_failed": Failed to send ping
// - "pong_timeout": No pong received within timeout
// - "low_pong_response_rate": Less than 50% pongs received
// - "write_buffer_full": Buffer usage > 80%

case <-pingTicker.C:
    // ... send ping ...
    if err != nil {
        m.triggerDisconnect("ping_write_failed")
        return
    }

case <-healthCheckTicker.C:
    if timeSinceLastPong > pongTimeout {
        m.triggerDisconnect("pong_timeout")
        return
    }
    if responseRate < 0.5 {
        m.triggerDisconnect("low_pong_response_rate")
        return
    }
    if bufferUsage > 80 {
        m.triggerDisconnect("write_buffer_full")
        return
    }
```

### Phase 7: WebSocket Handler Integration

```go
// PHASE 7: Register health monitor with disconnect callback for this connection
// The callback triggers cleanup when health monitor detects unhealthy connection
healthDisconnectOnce := sync.Once{}
healthDisconnectCallback := func(reason string) {
    healthDisconnectOnce.Do(func() {
        h.logger.Warn("Health monitor triggered disconnection",
            zap.String("sessionId", sessionID.Hex()),
            zap.String("reason", reason))
        // Signal cleanup to all goroutines
        cleanup.Close()
    })
}
healthMonitor := NewConnectionHealthMonitorWithCallback(conn, h.logger, &h.writeMutex, sessionID.Hex(), healthDisconnectCallback)
```

---

## Disconnect Flow

```
Health Monitor Detects Issue
    │
    ▼
triggerDisconnect(reason)
    │
    ├── disconnectOnce.Do() (ensures single execution)
    │       │
    │       ├── Mark unhealthy: isHealthy.Store(false)
    │       ├── Set reason: SetDisconnectReason(reason)
    │       ├── Log: "Health monitor triggering disconnection"
    │       ├── Metric: WebSocketSlowClients.Inc()
    │       │
    │       ├── Close WebSocket:
    │       │       ├── WriteControl(CloseMessage, "Connection unhealthy: reason")
    │       │       └── conn.Close()
    │       │
    │       └── Callback: onDisconnect(reason)
    │               │
    │               └── Handler cleanup.Close()
    │                       │
    │                       └── Signal all goroutines to exit
    │
    └── (subsequent calls are no-op)
```

---

## Disconnect Reasons

| Reason | Trigger | Threshold |
|--------|---------|-----------|
| `ping_write_failed` | Failed to send ping | Any error |
| `pong_timeout` | No pong received | 10 seconds |
| `low_pong_response_rate` | Pongs/Pings ratio | < 50% (after 5 pings) |
| `write_buffer_full` | Estimated buffer usage | > 80% |

---

## Metrics

### Updated Prometheus Metrics

```promql
# Slow client disconnections by reason (PHASE 7: now actually triggers disconnection)
chat_websocket_slow_clients_total{reason="pong_timeout"}
chat_websocket_slow_clients_total{reason="ping_write_failed"}
chat_websocket_slow_clients_total{reason="low_pong_response_rate"}
chat_websocket_slow_clients_total{reason="write_buffer_full"}
```

---

## Log Messages

```
# Health monitor detecting and disconnecting
WARN "Health monitor triggering disconnection"
    sessionId="507f1f77bcf86cd799439011"
    reason="pong_timeout"
    pingsSent=10
    pongsReceived=3
    bufferUsage=25

# Handler receiving disconnect callback
WARN "Health monitor triggered disconnection"
    sessionId="507f1f77bcf86cd799439011"
    reason="pong_timeout"

# Connection cleanup
INFO "Connection health monitor registered with disconnect callback"
    sessionId="507f1f77bcf86cd799439011"
```

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
| internal/handlers/connection_health_monitor.go | Added DisconnectCallback, triggerDisconnect, ForceDisconnect |
| internal/handlers/chat_websocket.go | Integrated callback in handleMessages |

---

## Implementation Complete

### Summary

The health monitor now actually disconnects unhealthy connections:

1. **DisconnectCallback**: Handler provides callback for cleanup coordination
2. **triggerDisconnect**: Closes WebSocket and calls callback (once only)
3. **ForceDisconnect**: Public API for external callers (e.g., circuit breaker)
4. **Handler integration**: Callback signals cleanup.Close() to stop all goroutines
5. **Metrics**: Tracks disconnections by reason

### Key Benefits

- **Resource protection**: Unhealthy connections are actually closed
- **Coordinated cleanup**: Handler can clean up resources properly
- **Observable**: Metrics and logging for all disconnections
- **Safe**: sync.Once ensures disconnect happens exactly once
- **Graceful**: Sends proper WebSocket close frame before closing

### Interaction with Other Components

```
Health Monitor ─── triggerDisconnect() ──────────────────────────┐
     │                                                            │
     │  onDisconnect callback                                     │
     ▼                                                            ▼
Handler cleanup.Close() ◄───────────────────────── WebSocket closed
     │
     ├── Signal all goroutines
     ├── Cancel stream context
     └── Clean up resources

Circuit Breaker ─── ForceDisconnect() ───▶ Health Monitor
```

### API Changes

**NewConnectionHealthMonitor** (deprecated)
```go
// Still works, but without disconnect capability
func NewConnectionHealthMonitor(conn *websocket.Conn, logger *zap.Logger, writeMutex *sync.Mutex) *ConnectionHealthMonitor
```

**NewConnectionHealthMonitorWithCallback** (new, preferred)
```go
// Full functionality with disconnect callback
func NewConnectionHealthMonitorWithCallback(conn *websocket.Conn, logger *zap.Logger, writeMutex *sync.Mutex, sessionID string, onDisconnect DisconnectCallback) *ConnectionHealthMonitor
```

**ForceDisconnect** (new)
```go
// For external callers to trigger disconnection
func (m *ConnectionHealthMonitor) ForceDisconnect(reason string)
```
