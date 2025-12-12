# Fix Double Channel Close - Use sync.Once for Channel Closing - Implementation Plan

## Problem Statement

Several notifier components in the handlers package have potential double-close panic scenarios:

### Issue 1: ProgressNotifier (progress_notifier.go)

```go
// RegisterSession - closes existing channel
if existingCh, exists := pn.channels[sessionKey]; exists {
    close(existingCh)  // PROBLEM: Could panic if already closed
}

// UnregisterSession - also closes channel
if ch, exists := pn.channels[sessionKey]; exists {
    close(ch)  // PROBLEM: Could panic if RegisterSession already closed it
    delete(pn.channels, sessionKey)
}
```

**Scenario that causes panic:**
1. Session A registered with channel C1
2. `RegisterSession` called again for session A - closes C1, creates C2
3. Goroutine still holding reference to C1 (from `range` loop)
4. `UnregisterSession` called - tries to close C2 (safe)
5. But if there's a reference leak, C1 could be closed again → **PANIC**

### Issue 2: MessageNotifier (message_notifier.go)

Same pattern:
```go
// RegisterSession
if existingCh, exists := mn.channels[sessionKey]; exists {
    close(existingCh)  // PROBLEM: Could panic
}

// UnregisterSession
if ch, exists := mn.channels[sessionKey]; exists {
    close(ch)  // PROBLEM: Could panic
```

### Impact

- **Runtime panic**: `panic: close of closed channel`
- **Server crash**: Unrecovered panic takes down the goroutine/handler
- **Difficult to debug**: Race conditions are intermittent

---

## Design Goals

1. **Safe channel closing**: Use `sync.Once` to ensure channels are closed exactly once
2. **Clear API**: Wrapper type that makes safe closing the default
3. **Backward compatible**: Existing channel operations still work
4. **Minimal overhead**: Only add sync.Once, no additional locks

---

## Implementation Phases

### Phase 1: Create SafeChannel Wrapper Type

**Goal**: Create a generic wrapper that uses `sync.Once` for safe closing.

**Changes**:
- Create `SafeChannel[T]` generic type
- Implement `Close()` method with `sync.Once`
- Implement `IsClosed()` method for status checks
- Implement `Chan()` to get underlying channel for receiving

**Risk**: Low - new type, no changes to existing code

---

### Phase 2: Update ProgressNotifier

**Goal**: Replace raw channels with `SafeChannel` in ProgressNotifier.

**Changes**:
- Change channel type from `chan ProgressEvent` to `*SafeChannel[ProgressEvent]`
- Update `RegisterSession` to use safe close
- Update `UnregisterSession` to use safe close
- Update `EmitProgress` to check if closed before sending

**Risk**: Low - internal implementation change

---

### Phase 3: Update MessageNotifier

**Goal**: Replace raw channels with `SafeChannel` in MessageNotifier.

**Changes**:
- Change channel type from `chan struct{}` to `*SafeChannel[struct{}]`
- Update `RegisterSession` to use safe close
- Update `UnregisterSession` to use safe close
- Update `NotifyNewMessage` to check if closed before sending

**Risk**: Low - internal implementation change

---

### Phase 4: Add Logging for Debug

**Goal**: Track channel lifecycle for debugging.

**Changes**:
- Log when channels are closed
- Log double-close attempts (prevented by sync.Once)

**Risk**: Low - observability only

---

## Detailed Implementation

### Phase 1: SafeChannel Wrapper

```go
// SafeChannel wraps a channel with sync.Once to prevent double-close panics
type SafeChannel[T any] struct {
    ch        chan T
    closeOnce sync.Once
    closed    atomic.Bool
}

// NewSafeChannel creates a new SafeChannel with the given buffer size
func NewSafeChannel[T any](bufferSize int) *SafeChannel[T] {
    return &SafeChannel[T]{
        ch: make(chan T, bufferSize),
    }
}

// Close safely closes the channel using sync.Once
func (sc *SafeChannel[T]) Close() {
    sc.closeOnce.Do(func() {
        sc.closed.Store(true)
        close(sc.ch)
    })
}

// IsClosed returns true if the channel has been closed
func (sc *SafeChannel[T]) IsClosed() bool {
    return sc.closed.Load()
}

// Chan returns the underlying channel for receiving
func (sc *SafeChannel[T]) Chan() <-chan T {
    return sc.ch
}

// Send sends a value to the channel if not closed
// Returns false if channel is closed or full
func (sc *SafeChannel[T]) Send(value T) bool {
    if sc.IsClosed() {
        return false
    }
    select {
    case sc.ch <- value:
        return true
    default:
        return false
    }
}
```

### Phase 2: Updated ProgressNotifier

```go
type ProgressNotifier struct {
    mu       sync.RWMutex
    channels map[string]*SafeChannel[ProgressEvent]
    logger   *zap.Logger
}

func (pn *ProgressNotifier) RegisterSession(sessionID primitive.ObjectID) <-chan ProgressEvent {
    pn.mu.Lock()
    defer pn.mu.Unlock()

    sessionKey := sessionID.Hex()
    // Safe close existing channel if it exists
    if existingSC, exists := pn.channels[sessionKey]; exists {
        existingSC.Close()  // Safe - uses sync.Once
    }

    // Create new safe channel
    safeCh := NewSafeChannel[ProgressEvent](10)
    pn.channels[sessionKey] = safeCh
    return safeCh.Chan()
}
```

### Phase 3: Updated MessageNotifier

```go
type MessageNotifier struct {
    mu       sync.RWMutex
    channels map[string]*SafeChannel[struct{}]
    logger   *zap.Logger
}

func (mn *MessageNotifier) RegisterSession(sessionID primitive.ObjectID) <-chan struct{} {
    mn.mu.Lock()
    defer mn.mu.Unlock()

    sessionKey := sessionID.Hex()
    // Safe close existing channel if any
    if existingSC, exists := mn.channels[sessionKey]; exists {
        existingSC.Close()  // Safe - uses sync.Once
    }

    // Create new safe channel
    safeCh := NewSafeChannel[struct{}](1)
    mn.channels[sessionKey] = safeCh
    return safeCh.Chan()
}
```

---

## Testing Strategy

1. **Unit Test**: Verify Close() can be called multiple times without panic
2. **Unit Test**: Verify IsClosed() returns correct state
3. **Race Test**: Run with `-race` flag to detect data races
4. **Integration Test**: Rapid register/unregister cycles

---

## Rollback Plan

1. Revert to raw channels if issues arise
2. SafeChannel is backward compatible - can be mixed with raw channels
3. Each notifier can be reverted independently

---

## Success Criteria

- [x] No panics from double channel close
- [x] SafeChannel wrapper implemented with sync.Once
- [x] ProgressNotifier uses SafeChannel
- [x] MessageNotifier uses SafeChannel
- [ ] All tests pass with `-race` flag

---

## Implementation Complete

### Changes Made

**Phase 1: SafeChannel Wrapper (safe_channel.go)**

Created new file `internal/handlers/safe_channel.go` with:
```go
type SafeChannel[T any] struct {
    ch        chan T
    closeOnce sync.Once
    closed    atomic.Bool
}

func NewSafeChannel[T any](bufferSize int) *SafeChannel[T]
func (sc *SafeChannel[T]) Close()
func (sc *SafeChannel[T]) IsClosed() bool
func (sc *SafeChannel[T]) Chan() <-chan T
func (sc *SafeChannel[T]) Send(value T) bool
func (sc *SafeChannel[T]) TrySend(value T) bool
```

**Phase 2: ProgressNotifier Updates (progress_notifier.go)**

- Changed `channels` map type from `chan ProgressEvent` to `*SafeChannel[ProgressEvent]`
- `RegisterSession()` - uses `existingSC.Close()` (safe via sync.Once)
- `UnregisterSession()` - uses `safeCh.Close()` (safe via sync.Once)
- `EmitProgress()` - checks `IsClosed()` before send, uses `safeCh.Send()`

**Phase 3: MessageNotifier Updates (message_notifier.go)**

- Changed `channels` map type from `chan struct{}` to `*SafeChannel[struct{}]`
- `RegisterSession()` - uses `existingSC.Close()` (safe via sync.Once)
- `UnregisterSession()` - uses `safeCh.Close()` (safe via sync.Once)
- `NotifyNewMessage()` - checks `IsClosed()` before send, uses `safeCh.Send()`

### Build Verification
```
make build
✓ Build complete: bin/hyper
```

### Key Benefits

1. **No panics**: `sync.Once` ensures channels are closed exactly once
2. **Thread-safe**: Atomic bool for closed state checks
3. **Clean API**: SafeChannel encapsulates safe closing pattern
4. **Non-blocking sends**: `Send()` returns false if channel full/closed
5. **Minimal overhead**: Only adds sync.Once and atomic.Bool per channel
6. **Backward compatible**: `Chan()` returns read-only channel for existing `range` loops

### Log Messages to Watch For

```
# Phase 2: ProgressNotifier
"Closed existing progress channel for session" - safe re-registration
"Progress channel already closed, skipping emit" - IsClosed() check worked
"Progress channel full or closed, dropping event" - Send() returned false

# Phase 3: MessageNotifier
"Closed existing notification channel for session" - safe re-registration
"Notification channel already closed, skipping notify" - IsClosed() check worked
"Notification channel already pending or closed for session" - Send() returned false
```

### Race Condition Prevention

The `sync.Once` pattern prevents this scenario:
1. Goroutine A calls `Close()` - enters `closeOnce.Do()`
2. Goroutine B calls `Close()` - blocks on `closeOnce.Do()`, returns immediately
3. No double-close panic possible

