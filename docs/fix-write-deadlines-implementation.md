# Add Write Deadlines to All WebSocket Writes - Implementation Plan

## Problem Statement

Several WebSocket write locations don't have write deadlines, which can cause goroutines to hang indefinitely when clients are slow or unresponsive:

### Locations WITHOUT Write Deadlines

**1. internal/ai-service/executor/sinks.go - WebSocketSink**
```go
// SendToken - NO deadline
if err := w.conn.WriteJSON(streamMsg); err != nil { ... }

// SendToolCall - NO deadline
if err := w.conn.WriteJSON(streamMsg); err != nil { ... }

// SendToolResult - NO deadline
if err := w.conn.WriteJSON(streamMsg); err != nil { ... }

// SendDone - NO deadline
if err := w.conn.WriteJSON(doneMsg); err != nil { ... }

// SendError - NO deadline
if err := w.conn.WriteJSON(errMsg); err != nil { ... }

// SendMessageSaved - NO deadline
if err := w.conn.WriteJSON(savedMsg); err != nil { ... }
```

**2. internal/handlers/websocket_broadcaster.go**
```go
// BroadcastToSession - NO deadline
err := mc.conn.WriteJSON(message)
```

### Locations WITH Write Deadlines (Good)

**internal/handlers/chat_websocket.go - safeWriteJSON**
```go
// ALREADY HAS DEADLINE ✓
if err := conn.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil { ... }
err := conn.WriteJSON(msg)
defer conn.SetWriteDeadline(time.Time{}) // Clear deadline after write
```

### Impact of Missing Deadlines

- **Goroutine leak**: Write hangs indefinitely on slow client
- **Resource exhaustion**: Server runs out of goroutines
- **Memory pressure**: Blocked goroutines hold memory
- **No timeout detection**: Can't detect slow clients

---

## Design Goals

1. **Consistent deadline handling**: All WebSocket writes have deadlines
2. **Configurable timeout**: Use shared constant for consistency
3. **Graceful handling**: Timeout errors are handled properly
4. **Minimal code duplication**: Helper functions where appropriate

---

## Implementation Phases

### Phase 1: Add Write Deadline Helper to Executor Sinks

**Goal**: Add write deadline to all WebSocketSink methods.

**Changes**:
- Add `writeTimeout` constant to sinks.go
- Create `writeWithDeadline` helper method
- Update all SendXxx methods to use deadline

**Risk**: Low - internal implementation change

---

### Phase 2: Add Write Deadline to WebSocket Broadcaster

**Goal**: Add write deadline to BroadcastToSession.

**Changes**:
- Add deadline before WriteJSON in BroadcastToSession
- Clear deadline after write

**Risk**: Low - single location change

---

### Phase 3: Verify All Write Locations

**Goal**: Ensure no WebSocket writes are missing deadlines.

**Changes**:
- Audit all WriteJSON/WriteMessage/WriteControl calls
- Add deadlines to any missed locations

**Risk**: Low - verification only

---

## Detailed Implementation

### Phase 1: Executor Sinks

```go
// internal/ai-service/executor/sinks.go

import (
    "time"
    // ...
)

const (
    // writeTimeout is the maximum time to wait for a WebSocket write
    writeTimeout = 30 * time.Second
)

// writeWithDeadline writes JSON with a deadline to prevent indefinite blocking
func (w *WebSocketSink) writeWithDeadline(msg interface{}) error {
    // Set write deadline
    if err := w.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
        w.logger.Warn("Failed to set write deadline", zap.Error(err))
    }
    defer w.conn.SetWriteDeadline(time.Time{}) // Clear deadline

    return w.conn.WriteJSON(msg)
}

// Updated SendToken using writeWithDeadline
func (w *WebSocketSink) SendToken(content string) error {
    // ... existing checks ...

    streamMsg := models.StreamMessage{
        Type:    "token",
        Content: content,
    }

    if err := w.writeWithDeadline(streamMsg); err != nil {
        // ... existing error handling ...
    }

    return nil
}
```

### Phase 2: WebSocket Broadcaster

```go
// internal/handlers/websocket_broadcaster.go

const (
    // writeTimeout for broadcaster writes
    broadcasterWriteTimeout = 30 * time.Second
)

func (wb *WebSocketBroadcaster) BroadcastToSession(...) error {
    // ... existing checks ...

    mc.writeMutex.Lock()
    defer mc.writeMutex.Unlock()

    // Set write deadline
    if err := mc.conn.SetWriteDeadline(time.Now().Add(broadcasterWriteTimeout)); err != nil {
        wb.logger.Warn("Failed to set write deadline", zap.Error(err))
    }
    defer mc.conn.SetWriteDeadline(time.Time{}) // Clear deadline

    err := mc.conn.WriteJSON(message)
    // ... rest of method ...
}
```

---

## Testing Strategy

1. **Unit Test**: Verify deadlines are set before writes
2. **Timeout Test**: Simulate slow client, verify timeout triggers
3. **Recovery Test**: Verify server recovers after timeout

---

## Success Criteria

- [x] WebSocketSink has deadlines on all WriteJSON calls
- [x] WebSocketBroadcaster has deadline on BroadcastToSession
- [x] All WebSocket write locations audited
- [x] Build passes

---

## Implementation Complete

### Changes Made

**Phase 1: Executor Sinks (internal/ai-service/executor/sinks.go)**

Added write deadline constant and helper method:
```go
const writeTimeout = 30 * time.Second

func (w *WebSocketSink) writeWithDeadline(msg interface{}) error {
    if err := w.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
        w.logger.Warn("Failed to set write deadline", zap.Error(err))
    }
    defer w.conn.SetWriteDeadline(time.Time{}) // Clear deadline
    return w.conn.WriteJSON(msg)
}
```

Updated all WebSocketSink methods to use `writeWithDeadline()`:
- `SendToken()` - token streaming
- `SendToolCall()` - tool call events
- `SendToolResult()` - tool result events
- `SendDone()` - completion signal
- `SendError()` - error messages
- `SendMessageSaved()` - message ID for reconciliation

**Phase 2: WebSocket Broadcaster (internal/handlers/websocket_broadcaster.go)**

Added write deadline constant:
```go
const broadcasterWriteTimeout = 30 * time.Second
```

Updated `BroadcastToSession()`:
```go
// Set write deadline before write
if err := mc.conn.SetWriteDeadline(time.Now().Add(broadcasterWriteTimeout)); err != nil {
    wb.logger.Warn("Failed to set write deadline for broadcast", ...)
}
defer mc.conn.SetWriteDeadline(time.Time{}) // Clear deadline

err := mc.conn.WriteJSON(message)
```

### Build Verification
```
make build
✓ Build complete: bin/hyper
```

### All WebSocket Write Locations (Audited)

| Location | File | Has Deadline |
|----------|------|--------------|
| safeWriteJSON | chat_websocket.go:1519 | ✓ (already had) |
| safeWriteControl | chat_websocket.go:1572 | ✓ (uses deadline param) |
| SendToken | executor/sinks.go | ✓ (added) |
| SendToolCall | executor/sinks.go | ✓ (added) |
| SendToolResult | executor/sinks.go | ✓ (added) |
| SendDone | executor/sinks.go | ✓ (added) |
| SendError | executor/sinks.go | ✓ (added) |
| SendMessageSaved | executor/sinks.go | ✓ (added) |
| BroadcastToSession | websocket_broadcaster.go | ✓ (added) |
| WriteControl (health check) | websocket_broadcaster.go:126 | ✓ (uses deadline param) |
| WriteControl (health monitor) | connection_health_monitor.go:216 | ✓ (uses deadline param) |

### Key Benefits

1. **No hanging goroutines**: All writes timeout after 30 seconds
2. **Slow client detection**: Timeout errors indicate unresponsive clients
3. **Resource protection**: Prevents goroutine/memory exhaustion
4. **Consistent behavior**: Same timeout across all write operations

### Timeout Behavior

When a write times out:
1. `WriteJSON` returns a `net.Error` with `Timeout() == true`
2. Error is logged with appropriate context
3. Connection is marked as disconnected (in WebSocketSink)
4. Subsequent writes are skipped (fast path for disconnected clients)

### Log Messages to Watch For

```
# Phase 1: Executor sinks
"Failed to set write deadline" - Deadline couldn't be set (rare)
"Failed to send token to WebSocket" - Write failed (timeout or disconnect)
"WebSocket client disconnected during token streaming" - Clean disconnect

# Phase 2: Broadcaster
"Failed to set write deadline for broadcast" - Deadline couldn't be set
"Failed to broadcast to session" - Write failed (timeout or disconnect)
```

