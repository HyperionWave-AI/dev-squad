# Fix Message Queue Timeout to Use Unique IDs - Implementation Plan

## Problem Statement

The message queue lacked unique identifiers for queued messages, making it difficult to:
- **Track specific message timeouts**: Can't correlate timeout events to specific messages
- **Debug message flow**: No way to trace a message through the system
- **Identify stale messages**: Can't distinguish old messages from new ones
- **Correlate logs**: Multiple logs about the same message can't be linked

### Before Fix

```go
type queuedMessage struct {
    msg      interface{}
    priority MessagePriority
    queued   time.Time
    // No unique ID!
}
```

**Log output before:**
```
"Message waited too long in queue" queueWait=5s
"Write from queue failed" error="timeout"
// Which message? Can't tell!
```

---

## Design Goals

1. **Unique message IDs**: Each queued message gets a unique identifier
2. **Timeout tracking**: Messages that wait too long are dropped with ID logging
3. **Message type tracking**: Include message type (token, tool_call, etc.)
4. **Statistics**: Track dropped, timed out, and total messages
5. **Correlation**: All logs for a message include its ID

---

## Implementation

### Phase 1: Enhanced queuedMessage Struct

```go
// PHASE 6: Added unique ID for timeout tracking and debugging
type queuedMessage struct {
    id       string          // Unique message ID for tracking
    msg      interface{}
    priority MessagePriority
    queued   time.Time
    msgType  string          // Type of message (token, tool_call, etc.)
}
```

### Phase 2: Enhanced WriteQueue Struct

```go
type WriteQueue struct {
    // ... existing fields ...
    timedOutCount  int64  // PHASE 6: Counter for timed out messages
    messageIDSeq   int64  // PHASE 6: Sequence number for unique message IDs
    sessionID      string // PHASE 6: Session ID for message ID prefix
}
```

### Phase 3: Unique Message ID Generation

Message IDs are generated using:
- Session ID (for cross-session uniqueness)
- Nanosecond timestamp (for temporal uniqueness)
- Sequence number (for same-nanosecond uniqueness)

```go
// Format: {sessionID}-{unixNano}-{sequence}
// Example: "507f1f77bcf86cd799439011-1699000000000000000-42"
messageID := fmt.Sprintf("%s-%d-%d", wq.sessionID, time.Now().UnixNano(), seq)
```

### Phase 4: Timeout Detection in Queue

Messages that wait longer than `writeTimeout` (10 seconds) are dropped:

```go
// PHASE 6: Check if message has timed out while waiting in queue
if queueWait > wq.writeTimeout {
    atomic.AddInt64(&wq.timedOutCount, 1)
    wq.logger.Warn("Message timed out in queue - dropping",
        zap.String("messageId", msg.id),
        zap.String("msgType", msg.msgType),
        zap.Duration("queueWait", queueWait),
        zap.Duration("timeout", wq.writeTimeout),
        zap.Int64("timedOutCount", atomic.LoadInt64(&wq.timedOutCount)))
    metrics.WebSocketErrors.WithLabelValues("queue_timeout").Inc()
    continue // Skip this message - it's too old
}
```

### Phase 5: Message Type Extraction

```go
func (wq *WriteQueue) extractMessageType(msg interface{}) string {
    if streamMsg, ok := msg.(models.StreamMessage); ok {
        return streamMsg.Type
    }
    if streamMsgPtr, ok := msg.(*models.StreamMessage); ok && streamMsgPtr != nil {
        return streamMsgPtr.Type
    }
    return "unknown"
}
```

### Phase 6: Statistics API

```go
type WriteQueueStats struct {
    SessionID     string
    DroppedCount  int64  // Messages dropped due to full queue
    TimedOutCount int64  // Messages timed out waiting in queue
    MessageSeq    int64  // Total messages queued
    QueueSize     int    // Maximum queue capacity
    QueueLength   int    // Current queue length
}

func (wq *WriteQueue) GetStats() WriteQueueStats
func (wq *WriteQueue) TimedOutCount() int64
```

---

## Log Output Examples

### Message Queued Successfully
No log (normal operation)

### Message Waited Too Long (but processed)
```
WARN "Message waited too long in queue"
    messageId="507f1f77bcf86cd799439011-1699000000000000000-42"
    msgType="token"
    queueWait="2.5s"
```

### Message Timed Out (dropped)
```
WARN "Message timed out in queue - dropping"
    messageId="507f1f77bcf86cd799439011-1699000000000000000-43"
    msgType="token"
    queueWait="12s"
    timeout="10s"
    timedOutCount=1
```

### Write Failed
```
WARN "Write from queue failed"
    messageId="507f1f77bcf86cd799439011-1699000000000000000-44"
    msgType="tool_result"
    queueWait="500ms"
    writeDuration="10s"
    error="i/o timeout"
```

### Message Dropped (queue full)
```
DEBUG "Dropped message due to full queue"
    messageId="507f1f77bcf86cd799439011-1699000000000000000-45"
    msgType="token"
    droppedCount=5
```

### Slow Write Completed
```
DEBUG "Message write completed (slow)"
    messageId="507f1f77bcf86cd799439011-1699000000000000000-46"
    msgType="done"
    queueWait="100ms"
    writeDuration="1.5s"
    totalLatency="1.6s"
```

---

## Metrics

### New Metric Labels

```promql
# Queue timeout errors (new label value)
chat_websocket_errors_total{error_type="queue_timeout"}

# Existing metrics still work:
chat_websocket_errors_total{error_type="queue_full"}
```

---

## API Changes

### NewWriteQueue

**Before:**
```go
func NewWriteQueue(conn *websocket.Conn, handler *ChatWebSocketHandler, logger *zap.Logger) *WriteQueue
```

**After:**
```go
func NewWriteQueue(conn *websocket.Conn, handler *ChatWebSocketHandler, logger *zap.Logger, sessionID string) *WriteQueue
```

### New Methods

```go
// Get count of messages that timed out in queue
func (wq *WriteQueue) TimedOutCount() int64

// Get comprehensive queue statistics
func (wq *WriteQueue) GetStats() WriteQueueStats
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
| internal/handlers/chat_websocket.go | Enhanced queuedMessage, WriteQueue, Send, writerLoop |

---

## Implementation Complete

### Summary

The message queue now uses unique IDs for timeout tracking:

1. **Unique IDs**: Format `{sessionID}-{unixNano}-{sequence}`
2. **Message type tracking**: Extracted from StreamMessage
3. **Timeout detection**: Messages >10s in queue are dropped
4. **Enhanced logging**: All logs include messageId and msgType
5. **Statistics**: DroppedCount, TimedOutCount, GetStats()

### Key Benefits

- **Debuggability**: Can trace any message through the system
- **Timeout handling**: Stale messages are proactively dropped
- **Correlation**: Multiple logs about same message can be linked
- **Monitoring**: Track queue health with detailed statistics

### Message ID Format

```
{sessionID}-{unixNano}-{sequence}
│          │           │
│          │           └── Monotonic counter (prevents duplicates same nanosecond)
│          └── Unix timestamp in nanoseconds (temporal ordering)
└── MongoDB ObjectID hex (session uniqueness)

Example: "507f1f77bcf86cd799439011-1699000000000000000-42"
```

### Timeout Flow

```
Message Queued (T=0)
    │
    ▼
[Wait in Queue]
    │
    ├── queueWait < 10s ──► Process normally
    │
    └── queueWait > 10s ──► Drop message
                             Log with messageId
                             Increment timedOutCount
                             Record metric
```
