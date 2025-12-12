# Fix Context Lifecycle - Use HTTP Context for Cancellation - Implementation Plan

## Problem Statement

Currently in `chat_websocket.go`, context management has several issues:

1. **Disconnected contexts**: `aiCtx` is created from `context.Background()` instead of being derived from the HTTP context
2. **streamCtx disconnected**: `streamCtx` at line 1724 is created from `context.Background()`, not tied to HTTP lifecycle
3. **Timeout-only cancellation**: The AI context only has a 10-minute timeout, not HTTP-based cancellation
4. **Late cancellation propagation**: HTTP cancellation is propagated via a separate goroutine, adding complexity

### Current Code (Problematic)

```go
// Line 1678-1687 - Contexts are disconnected from HTTP lifecycle
aiCtx := context.Background()  // NOT connected to HTTP!
aiCtx, aiCancel := context.WithTimeout(aiCtx, 10*time.Minute)
defer aiCancel()

httpCtx := c.Request.Context()  // This is the right context

h.handleMessages(aiCtx, httpCtx, conn, sessionID, userID, companyID)
```

```go
// Line 1724 - streamCtx not connected to HTTP lifecycle
streamCtx, streamCancel := context.WithCancel(context.Background())  // NOT connected to HTTP!
```

### Impact

- **Resource waste**: AI processing continues even after HTTP disconnect until timeout
- **Complex cancellation**: Multiple goroutines needed to propagate HTTP cancellation
- **Inconsistent behavior**: Some operations respect HTTP context, others don't
- **Potential goroutine leaks**: Cancellation propagation adds complexity that can leak

---

## Design Goals

1. **Unified context hierarchy**: All contexts derive from HTTP context
2. **Automatic cancellation**: When HTTP disconnects, all child contexts cancel automatically
3. **Timeout layering**: AI timeout is layered ON TOP of HTTP context, whichever fires first wins
4. **Simplified code**: Remove manual cancellation propagation goroutines

---

## Implementation Phases

### Phase 1: Fix aiCtx to Derive from httpCtx

**Goal**: Make AI context a child of HTTP context so it cancels automatically on disconnect.

**Changes**:
- Create `aiCtx` from `httpCtx` instead of `context.Background()`
- Remove redundant HTTP cancellation propagation goroutine

**Risk**: Low - improves existing behavior

---

### Phase 2: Fix streamCtx to Derive from aiCtx

**Goal**: Make stream cleanup context part of the context hierarchy.

**Changes**:
- Create `streamCtx` from `aiCtx` instead of `context.Background()`
- Ensures cleanup cancels when AI or HTTP cancels

**Risk**: Low - improves existing behavior

---

### Phase 3: Remove Redundant Cancellation Propagation

**Goal**: Simplify code by relying on context hierarchy instead of manual propagation.

**Changes**:
- Remove the HTTP cancellation propagation goroutine (lines 2124-2141)
- Context hierarchy handles cancellation automatically

**Risk**: Medium - removes safety net, but context hierarchy is more reliable

---

### Phase 4: Add Metrics and Logging

**Goal**: Track context lifecycle for debugging.

**Changes**:
- Log context creation with parent relationship
- Add metric for context cancellation reasons

**Risk**: Low - observability only

---

## Detailed Implementation

### Phase 1: Fix aiCtx to Derive from httpCtx

```go
// BEFORE (line 1678-1687)
aiCtx := context.Background()
aiCtx, aiCancel := context.WithTimeout(aiCtx, 10*time.Minute)
defer aiCancel()

httpCtx := c.Request.Context()

// AFTER
httpCtx := c.Request.Context()

// Create AI context as child of HTTP context
// This ensures AI processing cancels when:
// 1. HTTP connection closes (client disconnect)
// 2. 10-minute timeout expires
// Whichever happens first!
aiCtx, aiCancel := context.WithTimeout(httpCtx, 10*time.Minute)
defer aiCancel()
```

### Phase 2: Fix streamCtx to Derive from aiCtx

```go
// BEFORE (line 1724)
streamCtx, streamCancel := context.WithCancel(context.Background())

// AFTER
// Create stream context as child of AI context
// Cancellation flows: HTTP -> AI -> Stream
streamCtx, streamCancel := context.WithCancel(aiCtx)
```

### Phase 3: Simplify Cancellation Flow

The HTTP cancellation propagation goroutine (lines 2122-2141) can be simplified since context hierarchy handles this automatically:

```go
// BEFORE - Complex manual propagation
httpCancelDone := make(chan struct{})
go func() {
    defer close(httpCancelDone)
    select {
    case <-httpCtx.Done():
        h.logger.Info("HTTP context cancelled - cancelling AI execution")
        aiExecCancel()
    case <-cleanup.done:
        aiExecCancel()
    case <-aiExecCtx.Done():
    }
}()

// AFTER - Simplified (context hierarchy handles HTTP cancellation)
// The aiExecCtx is now derived from aiCtx which is derived from httpCtx
// So HTTP cancellation propagates automatically through the hierarchy
// We only need to handle cleanup.done for explicit cleanup signals
cleanupDone := make(chan struct{})
go func() {
    defer close(cleanupDone)
    select {
    case <-cleanup.done:
        aiExecCancel()
    case <-aiExecCtx.Done():
        // Context cancelled (HTTP, timeout, or stop button)
    }
}()
```

### Phase 4: Update handleMessages signature

Since `httpCtx` is now the parent of `aiCtx`, we don't need to pass both:

```go
// BEFORE
func (h *ChatWebSocketHandler) handleMessages(aiCtx context.Context, httpCtx context.Context, ...)

// AFTER - Simplified, aiCtx carries HTTP cancellation
func (h *ChatWebSocketHandler) handleMessages(aiCtx context.Context, ...)
```

Note: We'll keep `httpCtx` for now for explicit checks in the message loop, but the cancellation will flow through `aiCtx`.

---

## Context Hierarchy After Fix

```
HTTP Context (gin request context)
    └── AI Context (10 minute timeout)
            └── Stream Context (cleanup lifecycle)
                    └── AI Execution Context (per-message)
```

When HTTP disconnects, ALL child contexts cancel automatically.

---

## Testing Strategy

1. **Unit Test**: Verify context cancellation propagates through hierarchy
2. **Integration Test**: Disconnect client, verify all contexts cancel
3. **Timeout Test**: Verify 10-minute AI timeout still works
4. **Load Test**: Multiple connections with varying lifetimes

---

## Rollback Plan

1. Revert context creation back to `context.Background()`
2. Keep HTTP propagation goroutine as fallback
3. Phase 1 can be reverted independently if issues arise

---

## Success Criteria

- [x] AI context is child of HTTP context
- [x] Stream context is child of AI context
- [x] HTTP disconnect cancels all child contexts
- [x] 10-minute timeout still works
- [x] Simplified cancellation code (fewer goroutines)
- [x] All existing tests pass

---

## Implementation Complete

### Changes Made (chat_websocket.go)

**Phase 1: Fix aiCtx to Derive from httpCtx (Lines 1678-1694)**
```go
// BEFORE
aiCtx := context.Background()
aiCtx, aiCancel := context.WithTimeout(aiCtx, 10*time.Minute)

// AFTER
httpCtx := c.Request.Context()
aiCtx, aiCancel := context.WithTimeout(httpCtx, 10*time.Minute)
```
- AI context now inherits HTTP context cancellation
- HTTP disconnect automatically cancels AI processing
- 10-minute timeout still applies (whichever fires first)

**Phase 2: Fix streamCtx to Derive from aiCtx (Lines 1730-1739)**
```go
// BEFORE
streamCtx, streamCancel := context.WithCancel(context.Background())

// AFTER
streamCtx, streamCancel := context.WithCancel(aiCtx)
```
- Stream context now part of context hierarchy
- Cancellation flows: HTTP -> AI -> Stream

**Phase 3: Simplified Cancellation Propagation (Lines 2123-2168)**
```go
// BEFORE - Explicit HTTP cancellation propagation
go func() {
    select {
    case <-httpCtx.Done():  // Manual HTTP check
        aiExecCancel()
    case <-cleanup.done:
        aiExecCancel()
    case <-aiExecCtx.Done():
    }
}()

// AFTER - Simplified (HTTP flows through hierarchy)
go func() {
    select {
    case <-cleanup.done:  // Only explicit cleanup needed
        aiExecCancel()
    case <-aiExecCtx.Done():
        // Context cancelled (HTTP, timeout, or stop button)
    }
}()
```
- Removed explicit HTTP cancellation check (handled by hierarchy)
- Added debug logging for cancellation reason
- Simplified from 3 cases to 2 cases

### Build Verification
```
make build
✓ Build complete: bin/hyper
```

### Context Hierarchy After Fix

```
HTTP Context (gin request context)
    └── AI Context (10 minute timeout)
            └── Stream Context (cleanup lifecycle)
                    └── AI Execution Context (per-message)
```

### Key Benefits

1. **Automatic HTTP cancellation**: HTTP disconnect cancels all child contexts automatically
2. **Proper timeout behavior**: 10-minute timeout applies on top of HTTP lifecycle
3. **Simplified code**: Removed manual HTTP cancellation propagation goroutine logic
4. **Better debugging**: Added context cancellation reason logging
5. **Resource efficiency**: Contexts cleaned up promptly on disconnect

### Log Messages to Watch For

```
# Phase 1: Context hierarchy establishment
"Context hierarchy established" - hierarchy: "HTTP -> AI (10min timeout)"

# Phase 3: Cancellation events
"Cleanup signal received - cancelling AI execution" - explicit cleanup
"AI execution context cancelled" - context.Canceled (HTTP or stop button)
"AI execution context deadline exceeded" - context.DeadlineExceeded (10min timeout)
```

### Behavior Change

| Scenario | Before | After |
|----------|--------|-------|
| Client disconnects | AI continues until 10min timeout OR manual propagation | AI cancels immediately via context hierarchy |
| 10-minute timeout | Cancels AI context | Cancels AI context (unchanged) |
| Stop button | Cancels via stored cancel function | Cancels via stored cancel function (unchanged) |
| Cleanup signal | Cancels via manual propagation | Cancels via manual propagation (unchanged) |

