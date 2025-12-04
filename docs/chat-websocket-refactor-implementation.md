# Chat WebSocket Refactor - Implementation Plan

## Problem Statement

The `chat_websocket.go` file has grown to **3,485 lines**, making it:
- **Hard to navigate**: Multiple concerns mixed together
- **Difficult to test**: Large file = complex test setup
- **Maintenance burden**: Changes risk unintended side effects
- **Code review challenges**: Hard to review changes in a monolithic file

## Current File Structure Analysis

```
chat_websocket.go (3,485 lines)
├── Lines 1-29: Imports
├── Lines 30-62: chatServiceAdapter (executor interface adapter)
├── Lines 63-298: WriteQueue & message priority (backpressure)
├── Lines 300-423: FrameRateLimiter & session validation (security)
├── Lines 425-495: SlowClientDetector (backpressure)
├── Lines 497-736: websocketSink (StreamOutputSink adapter)
├── Lines 738-1505: DefaultSystemPrompt (large constant)
├── Lines 1507-1700: WebSocket upgrader, constants, rate limiting
├── Lines 1702-1732: ChatWebSocketHandler struct & constructor
├── Lines 1734-1865: safeWriteJSON, helpers
├── Lines 1867-2031: HandleChatWebSocket (main entry point)
├── Lines 2033-2300: handleMessages (message loop)
├── Lines 2300-2600: processUserMessage, AI streaming
├── Lines 2600-3000: System prompts, context building
├── Lines 3000-3200: Tool result streaming helpers
├── Lines 3200-3485: Tool result processing, size limits
```

## Proposed Split

| New File | Lines | Responsibility | Dependencies |
|----------|-------|----------------|--------------|
| `write_queue.go` | ~240 | Message queue, backpressure, priority | models, metrics, zap |
| `websocket_sink.go` | ~250 | StreamOutputSink adapter for WebSocket | models, websocket, zap |
| `rate_limiter.go` | ~180 | Frame rate limiting, slow client detection | metrics, zap |
| `tool_result_processor.go` | ~300 | Tool result size limits, deflection | config, models, zap |
| `chat_websocket.go` | ~2,500 | Main handler, orchestration | All above |

## Phase-by-Phase Implementation

---

## Phase 1: Extract `write_queue.go`

### What to Extract
- `MessagePriority` type and constants (lines 66-71)
- `queuedMessage` struct (lines 75-81)
- `WriteQueue` struct (lines 85-100)
- `ErrQueueFull` error (line 103)
- `NewWriteQueue` function (lines 108-125)
- `writerLoop` method (lines 130-191)
- `Send` method (lines 196-235)
- `extractMessageType` method (lines 239-247)
- `Close` method (lines 251-255)
- `DroppedCount` method (lines 258-260)
- `TimedOutCount` method (lines 264-266)
- `GetStats` method (lines 270-279)
- `WriteQueueStats` struct (lines 283-290)
- `MaxConsecutiveSlowWrites`, `MaxQueueDepthWarnings` constants (lines 294-298)

### File Structure
```go
// write_queue.go
package handlers

import (...)

// MessagePriority defines the priority level for queued messages
type MessagePriority int

const (
    PriorityNormal   MessagePriority = iota
    PriorityCritical
)

// Backpressure constants
const (
    MaxConsecutiveSlowWrites = 5
    MaxQueueDepthWarnings    = 10
)

// ErrQueueFull is returned when the write queue is full
var ErrQueueFull = fmt.Errorf("write queue full")

// queuedMessage wraps a message with metadata for queue management
type queuedMessage struct {...}

// WriteQueue manages buffered WebSocket writes with backpressure
type WriteQueue struct {...}

// WriteQueueStats holds statistics for monitoring
type WriteQueueStats struct {...}

// NewWriteQueue creates a buffered write queue
func NewWriteQueue(...) *WriteQueue {...}

// Methods...
```

### Verification
```bash
go build ./internal/handlers/...
```

---

## Phase 2: Extract `websocket_sink.go`

### What to Extract
- `websocketSink` struct (lines 497-505)
- `newWebSocketSink` function (lines 508-515)
- `newWebSocketSinkWithSession` function (lines 519-527)
- `SendToken` method (lines 531-557)
- `SendToolCall` method (lines 560-589)
- `SendToolResult` method (lines 592-622)
- `SendDone` method (lines 625-650)
- `SendError` method (lines 653-673)
- `SendMessageSaved` method (lines 676-701)
- `IsDisconnected` method (lines 704-708)
- `SendSystemNotification` method (lines 711-736)

### File Structure
```go
// websocket_sink.go
package handlers

import (...)

// websocketSink implements executor.StreamOutputSink for WebSocket connections
// This adapter avoids import cycles by implementing the interface locally
type websocketSink struct {
    conn         *websocket.Conn
    logger       *zap.Logger
    handler      *ChatWebSocketHandler
    disconnected bool
    mu           sync.Mutex
    writeQueue   *WriteQueue
    sessionID    string
}

// Constructor functions...
// Send* methods...
```

### Verification
```bash
go build ./internal/handlers/...
```

---

## Phase 3: Extract `rate_limiter.go`

### What to Extract
- Frame rate limiting constants (lines 301-310)
- `FrameRateLimiter` struct (lines 314-321)
- `NewFrameRateLimiter` function (lines 324-331)
- `Allow` method (lines 336-367)
- `ShouldDisconnect` method (lines 371-375)
- `GetViolations` method (lines 378-382)
- `validateSessionOwnership` method (lines 388-423)
- `SlowClientDetector` struct (lines 426-432)
- `NewSlowClientDetector` function (lines 436-441)
- `RecordWrite` method (lines 445-467)
- `RecordQueueFull` method (lines 471-487)
- `Reset` method (lines 490-495)

### File Structure
```go
// rate_limiter.go
package handlers

import (...)

// SECURITY: Frame rate limiting constants
const (
    frameRateLimit         = 60.0
    frameRateBurst         = 100.0
    frameRateRefillInterval = time.Second
    frameRateMaxViolations = 3
)

// FrameRateLimiter implements token bucket rate limiting for WebSocket frames
type FrameRateLimiter struct {...}

// NewFrameRateLimiter creates a new frame rate limiter
func NewFrameRateLimiter(...) *FrameRateLimiter {...}

// Methods...

// SlowClientDetector tracks client write performance
type SlowClientDetector struct {...}

// NewSlowClientDetector creates a slow client detector
func NewSlowClientDetector(...) *SlowClientDetector {...}

// Methods...
```

### Note on validateSessionOwnership
This method belongs on `ChatWebSocketHandler`, so it should stay in `chat_websocket.go`. Only the standalone types (`FrameRateLimiter`, `SlowClientDetector`) should be extracted.

### Verification
```bash
go build ./internal/handlers/...
```

---

## Phase 4: Extract `tool_result_processor.go`

### What to Extract
- `extractToolResultSummary` function (lines 3220-3254)
- `generateSuppressedToolResultMessage` method (lines 3257-3312)
- `ToolResultProcessed` struct (lines 3315-3322)
- `calculateRemainingContext` method (lines 3327-3348)
- `processToolResultWithSizeLimit` method (lines 3351-3484)

### File Structure
```go
// tool_result_processor.go
package handlers

import (...)

// ToolResultProcessed holds the processed tool result with metadata
type ToolResultProcessed struct {
    OutputStr        string
    ShouldStream     bool
    ShouldSaveFull   bool
    Tier             string // "normal", "truncated", "suppressed", "error"
    OriginalSize     int
    IsTruncated      bool
}

// extractToolResultSummary generates concise metadata for suppressed results
func extractToolResultSummary(toolName string, output interface{}) string {...}

// Methods on ChatWebSocketHandler...
```

### Verification
```bash
go build ./internal/handlers/...
```

---

## Phase 5: Final Verification

### Build Check
```bash
make build
```

### Test Check
```bash
go test ./internal/handlers/... -v
```

### Line Count Verification
After all phases:
- `write_queue.go`: ~240 lines
- `websocket_sink.go`: ~250 lines
- `rate_limiter.go`: ~180 lines
- `tool_result_processor.go`: ~300 lines
- `chat_websocket.go`: ~2,500 lines (down from 3,485)

---

## Implementation Rules

### DO
1. Extract complete, self-contained units
2. Maintain all existing functionality
3. Keep imports minimal in extracted files
4. Preserve all comments and documentation
5. Run `go build` after each phase

### DO NOT
1. Change any logic or behavior
2. Rename functions or types
3. Modify method signatures
4. Add new features during refactoring
5. Remove any existing code (only move)

---

## Rollback Plan

Each phase creates a new file. If issues arise:
1. Delete the new file
2. Uncomment the code in `chat_websocket.go`
3. Rebuild and verify

---

## Benefits After Refactor

| Benefit | Before | After |
|---------|--------|-------|
| File size | 3,485 lines | ~2,500 lines main + 4 smaller files |
| Testability | Complex setup | Each file independently testable |
| Navigation | Scroll through 3K lines | Jump to specific file |
| Code review | Large diffs | Focused, smaller diffs |
| Onboarding | Overwhelming | Clear separation of concerns |

---

## File Dependency Graph (After Refactor)

```
chat_websocket.go (main orchestrator)
    ├── write_queue.go
    │   └── Used by: websocket_sink.go, chat_websocket.go
    │
    ├── websocket_sink.go
    │   └── Used by: chat_websocket.go (AI streaming)
    │
    ├── rate_limiter.go
    │   └── Used by: chat_websocket.go (connection handling)
    │
    └── tool_result_processor.go
        └── Used by: chat_websocket.go (AI response handling)
```

---

## Status Tracking

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: write_queue.go | COMPLETED | 249 lines |
| Phase 2: websocket_sink.go | COMPLETED | 253 lines |
| Phase 3: rate_limiter.go | COMPLETED | 164 lines |
| Phase 4: tool_result_processor.go | COMPLETED | 53 lines |
| Phase 5: Final verification | COMPLETED | Build successful |

## Final Results

| File | Lines | Description |
|------|-------|-------------|
| `chat_websocket.go` | 2,803 | Main handler (down from 3,485) |
| `write_queue.go` | 249 | Message queue, backpressure, priority |
| `websocket_sink.go` | 253 | StreamOutputSink adapter |
| `rate_limiter.go` | 164 | Frame rate limiting, slow client detection |
| `tool_result_processor.go` | 53 | Tool result types and helpers |
| **Total** | **3,522** | All chat WebSocket code |

**Reduction in main file: 682 lines (19.6%)**
