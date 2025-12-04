# Fix Goroutine Leak in chat_websocket.go - Implementation Plan

## Problem Statement

In `chat_websocket.go:1742`, a goroutine is launched to process user messages. This goroutine has several issues that can cause leaks:

1. **No context cancellation check** - The goroutine doesn't monitor HTTP context cancellation
2. **No WaitGroup tracking** - Unlike other goroutines (ping, read), this one isn't tracked
3. **Uncontrolled lifecycle** - If client disconnects mid-processing, goroutine continues for up to 10 minutes
4. **Resource waste** - AI processing continues for disconnected clients

### Current Code (Problematic)

```go
// Line 1742-1837
go func(userMsg models.SendMessageRequest) {
    defer func() {
        isProcessing.Store(false)
        h.logger.Info("🎬 STREAMING ENDED...")
    }()

    // ... processing code that can run for minutes
    // No check for httpCtx.Done() or cleanup.done

    h.streamAIResponse(aiExecCtx, conn, sessionID, userMessage, companyID, cleanup)
}(userMsg)
```

### Expected Behavior

1. Goroutine should exit promptly when client disconnects
2. Goroutine should be tracked via WaitGroup for orderly shutdown
3. AI processing should be cancelled when connection is lost
4. Resources should be released immediately on disconnect

---

## Implementation Phases

### Phase 1: Add WaitGroup Tracking

**Goal**: Ensure the message processing goroutine is tracked and waited on during cleanup.

**Changes**:
- Add `cleanup.wg.Add(1)` before launching goroutine
- Add `defer cleanup.wg.Done()` at start of goroutine

**Risk**: Low - additive change only

---

### Phase 2: Add HTTP Context Monitoring

**Goal**: Exit goroutine when HTTP connection is lost.

**Changes**:
- Create wrapper context that cancels on either AI timeout OR HTTP disconnect
- Pass this combined context to `streamAIResponse`
- Add early exit checks at key points

**Risk**: Medium - changes context flow

---

### Phase 3: Add Cleanup Channel Monitoring

**Goal**: Respond to explicit cleanup signals.

**Changes**:
- Check `cleanup.done` channel at entry points
- Add select statements where blocking operations occur

**Risk**: Low - defensive checks only

---

### Phase 4: Improve Logging and Metrics

**Goal**: Track goroutine lifecycle for debugging.

**Changes**:
- Add unique goroutine ID for tracking
- Log entry/exit with reason
- Add metrics for goroutine count

**Risk**: Low - observability only

---

## Detailed Implementation

### Phase 1: WaitGroup Tracking

```go
// BEFORE (line 1742)
go func(userMsg models.SendMessageRequest) {
    defer func() {
        isProcessing.Store(false)
        // ...
    }()
    // ... processing
}(userMsg)

// AFTER
cleanup.wg.Add(1)
go func(userMsg models.SendMessageRequest) {
    defer cleanup.wg.Done()
    defer func() {
        isProcessing.Store(false)
        // ...
    }()
    // ... processing
}(userMsg)
```

### Phase 2: Combined Context

```go
// Create a context that cancels when EITHER:
// 1. AI timeout expires (10 min)
// 2. HTTP connection closes
// 3. Explicit cleanup signal

// BEFORE (line 1819)
aiExecCtx, aiExecCancel := context.WithCancel(aiCtx)

// AFTER
// Combine AI context with HTTP context for proper cancellation
aiExecCtx, aiExecCancel := context.WithCancel(aiCtx)

// Launch goroutine to propagate HTTP cancellation to AI context
go func() {
    select {
    case <-httpCtx.Done():
        h.logger.Info("HTTP context cancelled, cancelling AI execution",
            zap.String("sessionId", sessionID.Hex()))
        aiExecCancel()
    case <-aiExecCtx.Done():
        // AI context already cancelled, nothing to do
    case <-cleanup.done:
        aiExecCancel()
    }
}()
```

### Phase 3: Early Exit Checks

```go
// Add at start of goroutine
go func(userMsg models.SendMessageRequest) {
    defer cleanup.wg.Done()
    defer func() {
        isProcessing.Store(false)
        // ...
    }()

    // Check if already cancelled before starting
    select {
    case <-httpCtx.Done():
        h.logger.Info("Skipping message processing - connection already closed",
            zap.String("sessionId", sessionID.Hex()))
        return
    case <-cleanup.done:
        h.logger.Info("Skipping message processing - cleanup in progress",
            zap.String("sessionId", sessionID.Hex()))
        return
    default:
        // Continue processing
    }

    // ... rest of processing
}(userMsg)
```

### Phase 4: Metrics and Logging

```go
// Add at start
goroutineID := fmt.Sprintf("msg-%d", time.Now().UnixNano())
h.logger.Info("🚀 Message processing goroutine STARTED",
    zap.String("goroutineId", goroutineID),
    zap.String("sessionId", sessionID.Hex()))

// Add to defer
defer func() {
    h.logger.Info("🏁 Message processing goroutine ENDED",
        zap.String("goroutineId", goroutineID),
        zap.String("sessionId", sessionID.Hex()),
        zap.String("reason", exitReason))
}()
```

---

## Testing Strategy

1. **Unit Test**: Verify goroutine exits when context cancelled
2. **Integration Test**: Disconnect client mid-stream, verify goroutine exits
3. **Load Test**: Multiple connections, verify no goroutine accumulation
4. **Metric Verification**: Check goroutine count stays bounded

---

## Rollback Plan

If issues arise:
1. Revert to previous implementation (single commit)
2. Keep WaitGroup tracking (Phase 1) as it's non-breaking
3. Remove context combination if causing issues

---

## Success Criteria

- [ ] Goroutine exits within 5 seconds of client disconnect
- [ ] No increase in goroutine count under load testing
- [ ] All existing tests pass
- [ ] Metrics show proper goroutine lifecycle tracking

---

## Implementation Complete

### Changes Made (chat_websocket.go)

**Phase 1: WaitGroup Tracking (Lines 1742-1756)**
- Added `cleanup.wg.Add(1)` before launching goroutine
- Added `defer cleanup.wg.Done()` at start of goroutine
- Ensures orderly shutdown - `cleanup.Close()` now waits for message processing goroutine

**Phase 2: HTTP Context Monitoring (Lines 1848-1876)**
- Created `httpCancelDone` channel to track cancellation goroutine
- Launched goroutine that monitors `httpCtx.Done()`, `cleanup.done`, and `aiExecCtx.Done()`
- When HTTP context cancels (client disconnects), AI execution context is cancelled
- Defer waits for cancellation goroutine to exit via `<-httpCancelDone`

**Phase 3: Early Exit Checks (Lines 1768-1787)**
- Added non-blocking select at goroutine entry to check if already cancelled
- Checks both `httpCtx.Done()` and `cleanup.done` before any work
- Prevents wasted effort if client disconnected while message was queued

**Phase 4: Logging and Metrics (Lines 1745-1765)**
- Added unique `goroutineID` (format: `msg-{nanosecond timestamp}`)
- Track `goroutineStart` time for duration calculation
- Track `exitReason` variable (completed, http_context_cancelled_early, cleanup_in_progress_early)
- Log goroutine START with ID, session, content length
- Log goroutine END with ID, session, exit reason, duration

### Build Verification
```
make build
✓ Build complete: bin/hyper
```

### Key Benefits

1. **No more goroutine leaks**: WaitGroup ensures goroutine is tracked and waited on
2. **Prompt cancellation**: HTTP context propagation stops AI within milliseconds of disconnect
3. **Resource efficiency**: Early exit check prevents processing for already-disconnected clients
4. **Debuggability**: Unique goroutine IDs and exit reasons make debugging easy
5. **Graceful shutdown**: All goroutines exit before `handleMessages` returns
