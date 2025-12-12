package handlers

import (
	"sync"
	"sync/atomic"
)

// SafeChannel wraps a channel with sync.Once to prevent double-close panics
// PHASE 1: Generic wrapper for safe channel operations
// PHASE 8: Added statistics tracking for dropped events
type SafeChannel[T any] struct {
	ch           chan T
	closeOnce    sync.Once
	closed       atomic.Bool
	bufferSize   int           // PHASE 8: Store buffer size for stats
	sentCount    atomic.Int64  // PHASE 8: Number of successful sends
	droppedCount atomic.Int64  // PHASE 8: Number of dropped sends (buffer full)
}

// NewSafeChannel creates a new SafeChannel with the given buffer size
// PHASE 1: Factory function for creating safe channels
// PHASE 8: Now stores buffer size for statistics
func NewSafeChannel[T any](bufferSize int) *SafeChannel[T] {
	return &SafeChannel[T]{
		ch:         make(chan T, bufferSize),
		bufferSize: bufferSize,
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
// PHASE 8: Now tracks sent and dropped counts
func (sc *SafeChannel[T]) Send(value T) bool {
	if sc.IsClosed() {
		sc.droppedCount.Add(1)
		return false
	}
	select {
	case sc.ch <- value:
		sc.sentCount.Add(1)
		return true
	default:
		sc.droppedCount.Add(1)
		return false // Channel full
	}
}

// TrySend attempts to send a value, returning whether it succeeded
// This is an alias for Send() for clarity
func (sc *SafeChannel[T]) TrySend(value T) bool {
	return sc.Send(value)
}

// PHASE 8: Statistics methods

// SafeChannelStats contains statistics about channel operations
type SafeChannelStats struct {
	BufferSize   int   // Configured buffer size
	SentCount    int64 // Number of successful sends
	DroppedCount int64 // Number of dropped sends (buffer full or closed)
	IsClosed     bool  // Whether channel is closed
	CurrentLen   int   // Current number of items in buffer
}

// Stats returns current statistics for this channel
// PHASE 8: Provides visibility into channel health
func (sc *SafeChannel[T]) Stats() SafeChannelStats {
	return SafeChannelStats{
		BufferSize:   sc.bufferSize,
		SentCount:    sc.sentCount.Load(),
		DroppedCount: sc.droppedCount.Load(),
		IsClosed:     sc.closed.Load(),
		CurrentLen:   len(sc.ch),
	}
}

// SentCount returns the number of successful sends
func (sc *SafeChannel[T]) SentCount() int64 {
	return sc.sentCount.Load()
}

// DroppedCount returns the number of dropped sends
func (sc *SafeChannel[T]) DroppedCount() int64 {
	return sc.droppedCount.Load()
}

// BufferSize returns the configured buffer size
func (sc *SafeChannel[T]) BufferSize() int {
	return sc.bufferSize
}

// Len returns the current number of items in the channel buffer
func (sc *SafeChannel[T]) Len() int {
	return len(sc.ch)
}
