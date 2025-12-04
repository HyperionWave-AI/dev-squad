# Fix Notification Channel Buffer Size - Implementation Plan

## Problem Statement

The notification channels had several issues:
- **Small buffer sizes**: MessageNotifier had buffer of 1, causing drops on burst notifications
- **No visibility**: No statistics tracking for sent/dropped notifications
- **Silent failures**: Dropped notifications only logged at DEBUG level
- **No monitoring**: No way to detect notification system health issues

### Before Fix

```go
// MessageNotifier - buffer size of 1
safeCh := NewSafeChannel[struct{}](1)

// ProgressNotifier - buffer size of 10
safeCh := NewSafeChannel[ProgressEvent](10)

// No statistics tracking
// No visibility into dropped notifications
```

**Gap**: Burst notifications could be dropped silently with no visibility.

---

## Design Goals

1. **Adequate buffering**: Increase buffer sizes to handle burst scenarios
2. **Statistics tracking**: Track sent and dropped counts per channel
3. **Visibility**: Log warnings when notifications are dropped
4. **Monitoring**: Provide APIs to query notification health
5. **Consistency**: Use configurable constants for buffer sizes

---

## Implementation

### Phase 1: Enhanced SafeChannel with Statistics

```go
// SafeChannel now tracks statistics
type SafeChannel[T any] struct {
    ch           chan T
    closeOnce    sync.Once
    closed       atomic.Bool
    bufferSize   int           // PHASE 8: Store buffer size for stats
    sentCount    atomic.Int64  // PHASE 8: Number of successful sends
    droppedCount atomic.Int64  // PHASE 8: Number of dropped sends (buffer full)
}

// SafeChannelStats for visibility
type SafeChannelStats struct {
    BufferSize   int   // Configured buffer size
    SentCount    int64 // Number of successful sends
    DroppedCount int64 // Number of dropped sends (buffer full or closed)
    IsClosed     bool  // Whether channel is closed
    CurrentLen   int   // Current number of items in buffer
}
```

### Phase 2: Buffer Size Constants

```go
// Configurable buffer sizes for notification channels
const (
    MessageNotifierBufferSize  = 10  // Buffer for message notifications (was 1)
    ProgressNotifierBufferSize = 20  // Buffer for progress events (was 10)
)
```

### Phase 3: Enhanced Send Method

```go
func (sc *SafeChannel[T]) Send(value T) bool {
    if sc.IsClosed() {
        sc.droppedCount.Add(1)  // PHASE 8: Track drops
        return false
    }
    select {
    case sc.ch <- value:
        sc.sentCount.Add(1)    // PHASE 8: Track sends
        return true
    default:
        sc.droppedCount.Add(1) // PHASE 8: Track drops
        return false
    }
}
```

### Phase 4: Statistics APIs

```go
// SafeChannel methods
func (sc *SafeChannel[T]) Stats() SafeChannelStats
func (sc *SafeChannel[T]) SentCount() int64
func (sc *SafeChannel[T]) DroppedCount() int64
func (sc *SafeChannel[T]) BufferSize() int
func (sc *SafeChannel[T]) Len() int

// MessageNotifier aggregate stats
type MessageNotifierStats struct {
    ActiveSessions int
    SessionStats   map[string]SafeChannelStats
    TotalSent      int64
    TotalDropped   int64
}

func (mn *MessageNotifier) GetStats() MessageNotifierStats
func (mn *MessageNotifier) GetSessionStats(sessionID) (SafeChannelStats, bool)

// ProgressNotifier aggregate stats
type ProgressNotifierStats struct {
    ActiveSessions int
    SessionStats   map[string]SafeChannelStats
    TotalSent      int64
    TotalDropped   int64
}

func (pn *ProgressNotifier) GetStats() ProgressNotifierStats
func (pn *ProgressNotifier) GetSessionStats(sessionID) (SafeChannelStats, bool)
```

---

## Buffer Size Selection

### MessageNotifier: 1 → 10

| Scenario | Old Buffer (1) | New Buffer (10) |
|----------|----------------|-----------------|
| Single message | ✓ | ✓ |
| Rapid typing (5 messages) | 4 dropped | ✓ |
| Burst during AI response | Likely drops | ✓ |

**Rationale**: Users may type multiple messages before the subagent can process them. Buffer of 10 provides adequate headroom.

### ProgressNotifier: 10 → 20

| Scenario | Old Buffer (10) | New Buffer (20) |
|----------|-----------------|-----------------|
| Normal progress | ✓ | ✓ |
| Rapid tool calls (15 updates) | 5 dropped | ✓ |
| Concurrent subagent work | May drop | ✓ |

**Rationale**: AI agents may emit many progress events during tool execution. Buffer of 20 handles burst scenarios.

---

## Log Messages

### Before (DEBUG level, no stats)
```
DEBUG "Notification channel already pending or closed for session"
    sessionId="507f1f77bcf86cd799439011"
```

### After (WARN level with stats)
```
WARN "Notification channel full, message notification dropped"
    sessionId="507f1f77bcf86cd799439011"
    droppedCount=3
    bufferSize=10
    currentLen=10

WARN "Progress channel full, dropping event"
    sessionId="507f1f77bcf86cd799439011"
    message="Executing tool: read_file"
    droppedCount=1
    bufferSize=20
    currentLen=20
```

---

## Statistics Example

```go
// Get MessageNotifier stats
mn := GetMessageNotifier(logger)
stats := mn.GetStats()
// stats = {
//   ActiveSessions: 5,
//   TotalSent: 150,
//   TotalDropped: 2,
//   SessionStats: {
//     "507f1f77bcf86cd799439011": {BufferSize: 10, SentCount: 30, DroppedCount: 0, ...},
//     ...
//   }
// }

// Get per-session stats
sessionStats, ok := mn.GetSessionStats(sessionID)
// sessionStats = {BufferSize: 10, SentCount: 30, DroppedCount: 0, IsClosed: false, CurrentLen: 2}
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
| internal/handlers/safe_channel.go | Added statistics tracking (sentCount, droppedCount, bufferSize) |
| internal/handlers/message_notifier.go | Increased buffer 1→10, added stats, enhanced logging |
| internal/handlers/progress_notifier.go | Increased buffer 10→20, added stats, enhanced logging |

---

## Implementation Complete

### Summary

The notification channels now have:

1. **Larger buffers**: MessageNotifier 1→10, ProgressNotifier 10→20
2. **Statistics tracking**: Every SafeChannel tracks sent/dropped counts
3. **Visibility**: WARN level logging when notifications are dropped
4. **Monitoring APIs**: GetStats() and GetSessionStats() for health checks
5. **Configurable constants**: Buffer sizes defined as named constants

### Key Benefits

- **Fewer drops**: Burst scenarios handled without losing notifications
- **Visibility**: Dropped notifications are clearly logged
- **Debuggability**: Can query statistics to understand notification health
- **Proactive monitoring**: Stats can be exposed via health endpoints

### Buffer Size Summary

| Component | Before | After | Increase |
|-----------|--------|-------|----------|
| MessageNotifier | 1 | 10 | 10x |
| ProgressNotifier | 10 | 20 | 2x |
| SafeChannel stats | none | full | - |

### Statistics Flow

```
Send() called
    │
    ├── Channel closed?
    │       └── Yes → droppedCount++, return false
    │
    ├── Channel full?
    │       └── Yes → droppedCount++, return false, log WARN
    │
    └── Success → sentCount++, return true

GetStats() called
    │
    └── Aggregate all session stats
            ├── TotalSent = sum(sentCount)
            └── TotalDropped = sum(droppedCount)
```

### API Changes

**SafeChannel** (new methods)
```go
func (sc *SafeChannel[T]) Stats() SafeChannelStats
func (sc *SafeChannel[T]) SentCount() int64
func (sc *SafeChannel[T]) DroppedCount() int64
func (sc *SafeChannel[T]) BufferSize() int
func (sc *SafeChannel[T]) Len() int
```

**MessageNotifier** (new methods)
```go
func (mn *MessageNotifier) GetStats() MessageNotifierStats
func (mn *MessageNotifier) GetSessionStats(sessionID) (SafeChannelStats, bool)
```

**ProgressNotifier** (new methods)
```go
func (pn *ProgressNotifier) GetStats() ProgressNotifierStats
func (pn *ProgressNotifier) GetSessionStats(sessionID) (SafeChannelStats, bool)
```

### New Types

```go
type SafeChannelStats struct {
    BufferSize   int
    SentCount    int64
    DroppedCount int64
    IsClosed     bool
    CurrentLen   int
}

type MessageNotifierStats struct {
    ActiveSessions int
    SessionStats   map[string]SafeChannelStats
    TotalSent      int64
    TotalDropped   int64
}

type ProgressNotifierStats struct {
    ActiveSessions int
    SessionStats   map[string]SafeChannelStats
    TotalSent      int64
    TotalDropped   int64
}
```
