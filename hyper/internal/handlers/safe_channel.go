package handlers

import (
	"sync"
	"sync/atomic"
)

// SafeChannel wraps a channel with sync.Once to prevent double-close panics
// PHASE 1: Generic wrapper for safe channel operations
type SafeChannel[T any] struct {
	ch        chan T
	closeOnce sync.Once
	closed    atomic.Bool
}

// NewSafeChannel creates a new SafeChannel with the given buffer size
// PHASE 1: Factory function for creating safe channels
func NewSafeChannel[T any](bufferSize int) *SafeChannel[T] {
	return &SafeChannel[T]{
		ch: make(chan T, bufferSize),
	}
}

// Close safely closes the channel using sync.Once
// PHASE 1: Can be called multiple times without panic
func (sc *SafeChannel[T]) Close() {
	sc.closeOnce.Do(func() {
		sc.closed.Store(true)
		close(sc.ch)
	})
}

// IsClosed returns true if the channel has been closed
// PHASE 1: Thread-safe status check
func (sc *SafeChannel[T]) IsClosed() bool {
	return sc.closed.Load()
}

// Chan returns the underlying channel for receiving
// PHASE 1: Read-only channel access
func (sc *SafeChannel[T]) Chan() <-chan T {
	return sc.ch
}

// Send sends a value to the channel if not closed
// Returns false if channel is closed or full (non-blocking)
// PHASE 1: Safe non-blocking send
func (sc *SafeChannel[T]) Send(value T) bool {
	if sc.IsClosed() {
		return false
	}
	select {
	case sc.ch <- value:
		return true
	default:
		return false // Channel full
	}
}

// TrySend attempts to send a value, returning whether it succeeded
// This is an alias for Send() for clarity
func (sc *SafeChannel[T]) TrySend(value T) bool {
	return sc.Send(value)
}
