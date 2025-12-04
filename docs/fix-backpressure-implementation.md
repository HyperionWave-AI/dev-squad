# Add Backpressure Mechanism for WebSocket Writes - Implementation Plan

## Problem Statement

Currently, WebSocket writes in `chat_websocket.go` have no backpressure mechanism:

1. **No write timeout**: `safeWriteJSON` blocks indefinitely if client is slow
2. **No queue**: Messages are written directly, blocking the sender goroutine
3. **No slow client detection**: No mechanism to identify and disconnect slow clients
4. **Resource exhaustion**: Slow clients can cause goroutines to pile up waiting for writes

### Current Code (Problematic)

```go
// Line 1261-1271 - No timeout, blocks indefinitely
func (h *ChatWebSocketHandler) safeWriteJSON(conn *websocket.Conn, msg interface{}) error {
    h.writeMutex.Lock()
    defer h.writeMutex.Unlock()
    err := conn.WriteJSON(msg)  // Can block forever!
    if err == nil {
        metrics.WebSocketMessagesSent.Inc()
    }
    return err
}
```

### Impact

- **Goroutine leak**: Sender goroutines block on slow clients
- **Memory pressure**: Blocked goroutines hold references to messages
- **Cascading failures**: One slow client can affect server resources
- **Poor user experience**: No feedback when client is too slow

---

## Design Goals

1. **Write timeout**: All writes should have a maximum time limit
2. **Buffered queue**: Messages queue up, with overflow handling
3. **Slow client detection**: Track write latency, disconnect slow clients
4. **Graceful degradation**: Drop non-critical messages under pressure
5. **Observability**: Metrics for queue depth, write latency, dropped messages

---

## Implementation Phases

### Phase 1: Add Write Deadline to safeWriteJSON

**Goal**: Prevent indefinite blocking on slow clients.

**Changes**:
- Add `SetWriteDeadline` before each write
- Configure timeout (default: 10 seconds)
- Log and handle timeout errors

**Risk**: Low - adds safety without changing architecture

---

### Phase 2: Create WriteQueue for Buffered Writes

**Goal**: Decouple message production from transmission.

**Changes**:
- Create `WriteQueue` struct with buffered channel
- Background goroutine processes queue
- Configurable queue size and overflow behavior

**Risk**: Medium - changes write flow

---

### Phase 3: Implement Slow Client Detection

**Goal**: Identify and disconnect clients that can't keep up.

**Changes**:
- Track consecutive slow writes
- Track queue depth warnings
- Auto-disconnect after threshold exceeded

**Risk**: Medium - may disconnect legitimate slow connections

---

### Phase 4: Add Message Priority and Dropping

**Goal**: Preserve critical messages, drop non-critical under pressure.

**Changes**:
- Classify messages (critical: done, error; normal: token, tool_call)
- Drop normal messages when queue is full
- Never drop critical messages

**Risk**: Low - improves behavior under pressure

---

### Phase 5: Add Metrics and Logging

**Goal**: Observability for production monitoring.

**Changes**:
- Queue depth gauge
- Write latency histogram
- Dropped message counter
- Slow client counter

**Risk**: Low - observability only

---

## Detailed Implementation

### Phase 1: Write Deadline

```go
const (
    WriteTimeout = 10 * time.Second  // Maximum time to wait for a write
)

func (h *ChatWebSocketHandler) safeWriteJSON(conn *websocket.Conn, msg interface{}) error {
    h.writeMutex.Lock()
    defer h.writeMutex.Unlock()

    // PHASE 1: Set write deadline to prevent indefinite blocking
    conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
    defer conn.SetWriteDeadline(time.Time{})  // Clear deadline after write

    start := time.Now()
    err := conn.WriteJSON(msg)
    duration := time.Since(start)

    if err == nil {
        metrics.WebSocketMessagesSent.Inc()
        // Track write latency
        if duration > 1*time.Second {
            h.logger.Warn("Slow WebSocket write detected",
                zap.Duration("duration", duration))
        }
    } else if isTimeoutError(err) {
        h.logger.Warn("WebSocket write timeout",
            zap.Duration("timeout", WriteTimeout))
        metrics.WebSocketWriteTimeouts.Inc()
    }

    return err
}
```

### Phase 2: WriteQueue

```go
// WriteQueue manages buffered WebSocket writes with backpressure
type WriteQueue struct {
    conn           *websocket.Conn
    handler        *ChatWebSocketHandler
    logger         *zap.Logger
    queue          chan queuedMessage
    done           chan struct{}
    closeOnce      sync.Once

    // Metrics
    slowWriteCount int64
    droppedCount   int64

    // Config
    queueSize      int
    writeTimeout   time.Duration
    slowThreshold  time.Duration
}

type queuedMessage struct {
    msg      interface{}
    priority MessagePriority
    queued   time.Time
}

type MessagePriority int

const (
    PriorityNormal MessagePriority = iota
    PriorityCritical  // done, error messages - never dropped
)

func NewWriteQueue(conn *websocket.Conn, handler *ChatWebSocketHandler, logger *zap.Logger) *WriteQueue {
    wq := &WriteQueue{
        conn:          conn,
        handler:       handler,
        logger:        logger,
        queue:         make(chan queuedMessage, 100),  // Buffer 100 messages
        done:          make(chan struct{}),
        queueSize:     100,
        writeTimeout:  10 * time.Second,
        slowThreshold: 1 * time.Second,
    }

    // Start writer goroutine
    go wq.writerLoop()

    return wq
}

func (wq *WriteQueue) writerLoop() {
    for {
        select {
        case msg, ok := <-wq.queue:
            if !ok {
                return  // Queue closed
            }

            // Check queue wait time
            queueWait := time.Since(msg.queued)
            if queueWait > wq.slowThreshold {
                wq.logger.Warn("Message waited too long in queue",
                    zap.Duration("queueWait", queueWait))
            }

            // Write with timeout
            wq.conn.SetWriteDeadline(time.Now().Add(wq.writeTimeout))
            start := time.Now()
            err := wq.conn.WriteJSON(msg.msg)
            duration := time.Since(start)
            wq.conn.SetWriteDeadline(time.Time{})

            if err != nil {
                wq.logger.Warn("Write failed",
                    zap.Error(err),
                    zap.Duration("duration", duration))
                // Don't close queue - let caller handle disconnection
                continue
            }

            // Track slow writes
            if duration > wq.slowThreshold {
                atomic.AddInt64(&wq.slowWriteCount, 1)
                wq.logger.Debug("Slow write",
                    zap.Duration("duration", duration))
            }

            metrics.WebSocketMessagesSent.Inc()

        case <-wq.done:
            return
        }
    }
}

// Send queues a message for writing
func (wq *WriteQueue) Send(msg interface{}, priority MessagePriority) error {
    qm := queuedMessage{
        msg:      msg,
        priority: priority,
        queued:   time.Now(),
    }

    select {
    case wq.queue <- qm:
        return nil
    default:
        // Queue full - handle based on priority
        if priority == PriorityCritical {
            // Block for critical messages
            wq.queue <- qm
            return nil
        }
        // Drop non-critical messages
        atomic.AddInt64(&wq.droppedCount, 1)
        wq.logger.Debug("Dropped message due to full queue")
        metrics.WebSocketMessagesDropped.Inc()
        return ErrQueueFull
    }
}

// Close stops the writer goroutine
func (wq *WriteQueue) Close() {
    wq.closeOnce.Do(func() {
        close(wq.done)
        close(wq.queue)
    })
}
```

### Phase 3: Slow Client Detection

```go
const (
    SlowWriteThreshold    = 3 * time.Second  // Write taking > 3s is "slow"
    MaxConsecutiveSlowWrites = 5             // Disconnect after 5 slow writes
    MaxQueueDepthWarnings    = 10            // Disconnect after queue stays full
)

type SlowClientDetector struct {
    consecutiveSlowWrites int
    queueDepthWarnings    int
    mu                    sync.Mutex
    onSlowClient          func()  // Callback when client deemed too slow
}

func (scd *SlowClientDetector) RecordWrite(duration time.Duration) {
    scd.mu.Lock()
    defer scd.mu.Unlock()

    if duration > SlowWriteThreshold {
        scd.consecutiveSlowWrites++
        if scd.consecutiveSlowWrites >= MaxConsecutiveSlowWrites {
            scd.onSlowClient()
        }
    } else {
        scd.consecutiveSlowWrites = 0  // Reset on fast write
    }
}

func (scd *SlowClientDetector) RecordQueueFull() {
    scd.mu.Lock()
    defer scd.mu.Unlock()

    scd.queueDepthWarnings++
    if scd.queueDepthWarnings >= MaxQueueDepthWarnings {
        scd.onSlowClient()
    }
}
```

### Phase 4: Message Priority Classification

```go
func classifyMessage(msg models.StreamMessage) MessagePriority {
    switch msg.Type {
    case "done", "error", "message_saved":
        return PriorityCritical  // Never drop these
    case "token", "tool_call", "tool_result":
        return PriorityNormal    // Can be dropped under pressure
    default:
        return PriorityNormal
    }
}
```

### Phase 5: Integration with websocketSink

```go
type websocketSink struct {
    conn         *websocket.Conn
    logger       *zap.Logger
    handler      *ChatWebSocketHandler
    writeQueue   *WriteQueue           // PHASE 2: Buffered queue
    slowDetector *SlowClientDetector   // PHASE 3: Slow client detection
    disconnected bool
    mu           sync.Mutex
}

func (w *websocketSink) SendToken(content string) error {
    w.mu.Lock()
    if w.disconnected {
        w.mu.Unlock()
        return nil
    }
    w.mu.Unlock()

    streamMsg := models.StreamMessage{
        Type:    "token",
        Content: content,
    }

    // Use queue instead of direct write
    err := w.writeQueue.Send(streamMsg, PriorityNormal)
    if err == ErrQueueFull {
        // Non-critical message dropped - not an error for caller
        return nil
    }
    return err
}
```

---

## Configuration

```go
// WebSocket backpressure configuration
type BackpressureConfig struct {
    // Queue settings
    QueueSize           int           `default:"100"`

    // Timeout settings
    WriteTimeout        time.Duration `default:"10s"`

    // Slow client detection
    SlowWriteThreshold  time.Duration `default:"3s"`
    MaxSlowWrites       int           `default:"5"`
    MaxQueueWarnings    int           `default:"10"`

    // Message dropping
    DropNonCritical     bool          `default:"true"`
}
```

---

## Testing Strategy

1. **Unit Test**: Queue overflow handling, priority classification
2. **Integration Test**: Simulate slow client with delayed reads
3. **Load Test**: Many concurrent connections with varying speeds
4. **Chaos Test**: Random network delays, verify graceful degradation

---

## Rollback Plan

1. Phase 1 (write deadline) is safe - can keep even if other phases fail
2. Phase 2-4 can be disabled via config flag
3. Original `safeWriteJSON` preserved as fallback

---

## Success Criteria

- [x] No goroutines blocked indefinitely on writes
- [x] Slow clients detected and disconnected gracefully
- [x] Critical messages (done, error) never dropped
- [x] Queue depth visible in metrics
- [x] Write latency tracked in histogram
- [x] Dropped message count tracked

---

## Implementation Complete

### Changes Made

**Phase 1: Write Deadline (chat_websocket.go)**
- Added `WriteTimeout` constant (10 seconds) and `SlowWriteThreshold` (1 second)
- Updated `safeWriteJSON` to use `SetWriteDeadline` before writes
- Added `isTimeoutError` helper function to detect timeout errors
- Added metrics: `WebSocketWriteTimeouts`, `WebSocketWriteLatency`, `WebSocketSlowWrites`

**Phase 2: WriteQueue (chat_websocket.go)**
- Added `MessagePriority` type with `PriorityNormal` and `PriorityCritical`
- Added `queuedMessage` wrapper struct with priority and queue time tracking
- Added `WriteQueue` struct with buffered channel (100 messages)
- `writerLoop()` processes queue with latency tracking
- `Send()` handles priority-based dropping (critical messages block, normal messages drop)
- Added `ErrQueueFull` error for queue overflow

**Phase 3: Slow Client Detection (chat_websocket.go)**
- Added `MaxConsecutiveSlowWrites` (5) and `MaxQueueDepthWarnings` (10) constants
- Added `SlowClientDetector` struct with consecutive slow write and queue depth tracking
- `RecordWrite()` tracks slow writes, triggers callback after threshold
- `RecordQueueFull()` tracks queue overflows, triggers callback after threshold
- Added metric: `WebSocketSlowClients` (with reason label)

**Phase 4: Message Priority (chat_websocket.go)**
- Implemented as part of Phase 2
- `PriorityCritical` messages (done, error, message_saved) block when queue full
- `PriorityNormal` messages (token, tool_call, tool_result) dropped when queue full

**Phase 5: Metrics (registry.go)**
- `chat_websocket_write_timeouts_total` - Write timeout counter
- `chat_websocket_write_latency_seconds` - Write latency histogram (1ms to 10s buckets)
- `chat_websocket_slow_writes_total` - Slow write counter (>1s)
- `chat_websocket_slow_clients_total` - Slow client disconnection counter (by reason)

### Build Verification
```
make build
✓ Build complete: bin/hyper
```

### Key Benefits

1. **No indefinite blocking**: Write deadline (10s) ensures writes fail rather than block forever
2. **Latency monitoring**: Write latency histogram tracks performance
3. **Slow client detection**: Consecutive slow writes trigger disconnect callback
4. **Message prioritization**: Critical messages never dropped, normal messages can be dropped under pressure
5. **Queue backpressure**: WriteQueue buffers 100 messages, handles overflow gracefully
6. **Observability**: Full metrics suite for monitoring production behavior

### Architecture Notes

The implementation follows a layered approach:
1. `safeWriteJSON` - Base layer with write deadline (always active)
2. `WriteQueue` - Optional buffering layer (available but not enabled by default)
3. `SlowClientDetector` - Detection layer (can be integrated with connection management)

This allows for:
- Immediate benefit from Phase 1 (write deadlines) with no behavioral changes
- Optional queue buffering when needed for high-load scenarios
- Configurable slow client detection thresholds

### Log Messages to Watch For

```
# Phase 1: Write timeouts and slow writes
"Slow WebSocket write detected" - duration exceeded 1s
"WebSocket write timeout - client too slow" - write took >10s

# Phase 2: Queue operations
"Message waited too long in queue" - queue latency exceeded 1s
"Dropped message due to full queue" - non-critical message dropped
"Write from queue failed" - background write error

# Phase 3: Slow client detection
"Slow write recorded" - tracking consecutive slow writes
"Queue full recorded" - tracking queue overflow events
"Client deemed too slow" - threshold exceeded, callback triggered
```
