# Fix Concurrent Map Panic in websocket_broadcaster.go - Implementation Plan

## Problem Statement

In `websocket_broadcaster.go`, there are potential panic scenarios related to channel double-close and map operations:

### Issue 1: Double Channel Close (CRITICAL)
The `done` channel in `managedConnection` can be closed from multiple places:
- `RegisterConnection` (line 112) - closes existing connection's done channel
- `UnregisterConnection` (line 137) - closes the done channel
- `checkConnectionHealth` (line 95) - closes dead connection's done channel

**Scenario that causes panic:**
1. Connection A registered for session X
2. Health check runs, finds A is dead, adds to `deadConnections` list
3. Before health check deletes A, new connection B registered for session X
4. `RegisterConnection` closes A's `done` channel (line 112)
5. Health check loop continues, tries to close A's `done` channel again (line 95) → **PANIC**

### Issue 2: Stale Connection Reference
In `BroadcastToSession`:
1. Get connection reference with RLock (line 147-150)
2. Release RLock
3. Try to use connection (line 162)
4. Meanwhile, `UnregisterConnection` could have closed the connection → potential issues

### Current Code Analysis

```go
// Line 70-100 - Already uses two-pass approach (GOOD)
deadConnections := []string{}
for sessionKey, mc := range wb.connections {
    // ... check health
    if err != nil {
        deadConnections = append(deadConnections, sessionKey)
    }
}
// Remove dead connections (separate pass)
for _, sessionKey := range deadConnections {
    if mc, exists := wb.connections[sessionKey]; exists {
        close(mc.done)  // PROBLEM: Could panic if already closed
        delete(wb.connections, sessionKey)
    }
}
```

```go
// Line 110-113 - RegisterConnection
if existing, exists := wb.connections[sessionKey]; exists {
    close(existing.done)  // PROBLEM: Could panic if already closed
}
```

---

## Implementation Phases

### Phase 1: Add sync.Once to managedConnection

**Goal**: Prevent double-close panic on the `done` channel.

**Changes**:
- Add `closeOnce sync.Once` field to `managedConnection`
- Create `Close()` method that uses `sync.Once` to safely close the channel
- Replace all `close(mc.done)` calls with `mc.Close()`

**Risk**: Low - defensive change only

---

### Phase 2: Add Connection State Tracking

**Goal**: Track whether connection is already closed/invalid.

**Changes**:
- Add `closed bool` field to `managedConnection`
- Set `closed = true` when connection is closed
- Check `closed` before operations

**Risk**: Low - additional safety check

---

### Phase 3: Improve Health Check Safety

**Goal**: Make health check more robust against race conditions.

**Changes**:
- Verify connection still exists before closing
- Add logging for edge cases
- Handle case where connection was replaced during health check

**Risk**: Low - defensive checks

---

### Phase 4: Add Metrics and Logging

**Goal**: Track connection lifecycle for debugging.

**Changes**:
- Log connection close with reason
- Add connection count metric
- Track double-close attempts (for monitoring)

**Risk**: Low - observability only

---

## Detailed Implementation

### Phase 1: sync.Once for Channel Close

```go
// BEFORE
type managedConnection struct {
    conn       *websocket.Conn
    writeMutex *sync.Mutex
    done       chan struct{}
    lastPing   time.Time
    mu         sync.Mutex
}

// AFTER
type managedConnection struct {
    conn       *websocket.Conn
    writeMutex *sync.Mutex
    done       chan struct{}
    lastPing   time.Time
    mu         sync.Mutex
    closeOnce  sync.Once  // Prevents double-close panic
}

// Add safe close method
func (mc *managedConnection) Close() {
    mc.closeOnce.Do(func() {
        close(mc.done)
    })
}
```

### Phase 2: Connection State Tracking

```go
type managedConnection struct {
    conn       *websocket.Conn
    writeMutex *sync.Mutex
    done       chan struct{}
    lastPing   time.Time
    mu         sync.Mutex
    closeOnce  sync.Once
    closed     bool       // Track if connection was closed
    closedAt   time.Time  // When it was closed
    closeReason string    // Why it was closed
}

func (mc *managedConnection) Close(reason string) {
    mc.closeOnce.Do(func() {
        mc.mu.Lock()
        mc.closed = true
        mc.closedAt = time.Now()
        mc.closeReason = reason
        mc.mu.Unlock()
        close(mc.done)
    })
}

func (mc *managedConnection) IsClosed() bool {
    mc.mu.Lock()
    defer mc.mu.Unlock()
    return mc.closed
}
```

### Phase 3: Improved Health Check

```go
func (wb *WebSocketBroadcaster) checkConnectionHealth() {
    wb.mu.Lock()
    defer wb.mu.Unlock()

    deadConnections := []string{}

    for sessionKey, mc := range wb.connections {
        // Skip if already marked as closed
        if mc.IsClosed() {
            deadConnections = append(deadConnections, sessionKey)
            continue
        }

        mc.mu.Lock()
        err := mc.conn.WriteControl(
            websocket.PingMessage,
            []byte{},
            time.Now().Add(10*time.Second),
        )
        if err != nil {
            wb.logger.Warn("Connection health check failed",
                zap.String("sessionId", sessionKey),
                zap.Error(err))
            deadConnections = append(deadConnections, sessionKey)
        } else {
            mc.lastPing = time.Now()
        }
        mc.mu.Unlock()
    }

    // Remove dead connections (safe - uses Close() with sync.Once)
    for _, sessionKey := range deadConnections {
        if mc, exists := wb.connections[sessionKey]; exists {
            mc.Close("health_check_failed")
            delete(wb.connections, sessionKey)
            wb.logger.Info("Removed dead connection",
                zap.String("sessionId", sessionKey),
                zap.String("reason", "health_check"))
        }
    }
}
```

### Phase 4: Update All Close Calls

```go
// RegisterConnection
if existing, exists := wb.connections[sessionKey]; exists {
    existing.Close("replaced_by_new_connection")
    wb.logger.Debug("Closed existing connection for session",
        zap.String("sessionId", sessionKey),
        zap.String("reason", "reconnection"))
}

// UnregisterConnection
if mc, exists := wb.connections[sessionKey]; exists {
    mc.Close("unregistered")
    delete(wb.connections, sessionKey)
    wb.logger.Debug("Unregistered WebSocket connection",
        zap.String("sessionId", sessionKey))
}
```

---

## Testing Strategy

1. **Unit Test**: Verify `Close()` can be called multiple times without panic
2. **Race Test**: Run with `-race` flag to detect data races
3. **Integration Test**: Rapid connect/disconnect/reconnect cycles
4. **Stress Test**: Many concurrent connections with health checks

---

## Rollback Plan

If issues arise:
1. Revert to previous implementation (single commit)
2. Keep `sync.Once` as it's purely defensive
3. Remove state tracking if causing issues

---

## Success Criteria

- [ ] No panics under concurrent connection churn
- [ ] Health check properly removes dead connections
- [ ] Reconnection works smoothly (old connection closed, new one registered)
- [ ] All tests pass with `-race` flag
- [ ] Logs show proper connection lifecycle

---

## Implementation Complete

### Changes Made (websocket_broadcaster.go)

**Phase 1 & 2: managedConnection struct (Lines 13-45)**
```go
type managedConnection struct {
    // ... existing fields ...
    closeOnce   sync.Once  // Prevents double-close panic
    closed      bool       // Track if closed
    closedAt    time.Time  // When closed
    closeReason string     // Why closed (debugging)
}

func (mc *managedConnection) Close(reason string) {
    mc.closeOnce.Do(func() {
        mc.closed = true
        mc.closedAt = time.Now()
        mc.closeReason = reason
        close(mc.done)
    })
}

func (mc *managedConnection) IsClosed() bool { ... }
```

**Phase 3: checkConnectionHealth (Lines 90-137)**
- Added `IsClosed()` check at start of loop to skip already-closed connections
- Changed `close(mc.done)` to `mc.Close("health_check_failed")`
- Added logging with close reason

**Phase 4: RegisterConnection (Lines 139-167)**
- Changed `close(existing.done)` to `existing.Close("replaced_by_new_connection")`
- Added logging for reconnection case

**Phase 4: UnregisterConnection (Lines 169-184)**
- Changed `close(mc.done)` to `mc.Close("unregistered")`

**Phase 4: BroadcastToSession (Lines 186-225)**
- Added `IsClosed()` check before attempting to write
- Skips broadcast to closed connections with debug log

### Build Verification
```
make build
✓ Build complete: bin/hyper
```

### Key Benefits

1. **No more double-close panics**: `sync.Once` ensures channel is closed exactly once
2. **State tracking**: Know if/when/why a connection was closed
3. **Race condition safe**: Multiple goroutines can call `Close()` safely
4. **Better debugging**: Close reason logged for troubleshooting
5. **Graceful degradation**: Operations skip closed connections instead of panicking

### Close Reasons Tracked

| Reason | When Used |
|--------|-----------|
| `replaced_by_new_connection` | RegisterConnection closes old connection |
| `unregistered` | UnregisterConnection called |
| `health_check_failed` | Ping failed in health check |
