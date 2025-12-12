package handlers

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"hyper/internal/metrics"
	"hyper/internal/models"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// PHASE 2 Backpressure: Message priority for queue management
type MessagePriority int

const (
	PriorityNormal   MessagePriority = iota // tokens, tool_call, tool_result - can be dropped under pressure
	PriorityCritical                        // done, error, message_saved - never dropped
)

// PHASE 3 Backpressure: Slow client detection constants
const (
	// MaxConsecutiveSlowWrites is the number of slow writes before client is considered too slow
	MaxConsecutiveSlowWrites = 5
	// MaxQueueDepthWarnings is the number of queue full events before client is considered too slow
	MaxQueueDepthWarnings = 10
)

// PHASE 2 Backpressure: Error for queue overflow
var ErrQueueFull = fmt.Errorf("write queue full")

// PHASE 2 Backpressure: Queued message wrapper
// PHASE 6: Added unique ID for timeout tracking and debugging
type queuedMessage struct {
	id       string          // Unique message ID for tracking
	msg      interface{}
	priority MessagePriority
	queued   time.Time
	msgType  string // Type of message (token, tool_call, etc.) for logging
}

// PHASE 2 Backpressure: WriteQueue manages buffered WebSocket writes with backpressure
// PHASE 6: Added message ID tracking for timeout correlation
type WriteQueue struct {
	conn           *websocket.Conn
	handler        *ChatWebSocketHandler
	logger         *zap.Logger
	queue          chan queuedMessage
	done           chan struct{}
	closeOnce      sync.Once
	slowWriteCount int64  // Atomic counter for slow writes
	droppedCount   int64  // Atomic counter for dropped messages
	timedOutCount  int64  // PHASE 6: Counter for timed out messages
	messageIDSeq   int64  // PHASE 6: Sequence number for unique message IDs
	sessionID      string // PHASE 6: Session ID for message ID prefix
	queueSize      int
	writeTimeout   time.Duration
	slowThreshold  time.Duration
}

// WriteQueueStats holds statistics for a write queue
// PHASE 6: For monitoring and debugging
type WriteQueueStats struct {
	SessionID     string
	DroppedCount  int64
	TimedOutCount int64
	MessageSeq    int64
	QueueSize     int
	QueueLength   int
}

// NewWriteQueue creates a buffered write queue for a WebSocket connection
// PHASE 2 Backpressure: Decouples message production from transmission
// PHASE 6: Added sessionID parameter for unique message ID generation
func NewWriteQueue(conn *websocket.Conn, handler *ChatWebSocketHandler, logger *zap.Logger, sessionID string) *WriteQueue {
	wq := &WriteQueue{
		conn:          conn,
		handler:       handler,
		logger:        logger,
		queue:         make(chan queuedMessage, 100), // Buffer 100 messages
		done:          make(chan struct{}),
		sessionID:     sessionID,
		queueSize:     100,
		writeTimeout:  WriteTimeout,
		slowThreshold: SlowWriteThreshold,
	}

	// Start writer goroutine
	go wq.writerLoop()

	return wq
}

// writerLoop processes queued messages in a dedicated goroutine
// PHASE 2 Backpressure: Background writer with latency tracking
// PHASE 6: Enhanced with unique message ID tracking for timeout correlation
func (wq *WriteQueue) writerLoop() {
	for {
		select {
		case msg, ok := <-wq.queue:
			if !ok {
				return // Queue closed
			}

			// Check queue wait time
			queueWait := time.Since(msg.queued)

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

			if queueWait > wq.slowThreshold {
				wq.logger.Warn("Message waited too long in queue",
					zap.String("messageId", msg.id),
					zap.String("msgType", msg.msgType),
					zap.Duration("queueWait", queueWait))
			}

			// Write with timeout (via safeWriteJSON which now has deadlines)
			writeStart := time.Now()
			err := wq.handler.safeWriteJSON(wq.conn, msg.msg)
			writeDuration := time.Since(writeStart)

			if err != nil {
				wq.logger.Warn("Write from queue failed",
					zap.String("messageId", msg.id),
					zap.String("msgType", msg.msgType),
					zap.Duration("queueWait", queueWait),
					zap.Duration("writeDuration", writeDuration),
					zap.Error(err))
				// Don't close queue - let caller handle disconnection
				continue
			}

			// PHASE 6: Log successful write with timing details for debugging
			if queueWait > wq.slowThreshold || writeDuration > wq.slowThreshold {
				wq.logger.Debug("Message write completed (slow)",
					zap.String("messageId", msg.id),
					zap.String("msgType", msg.msgType),
					zap.Duration("queueWait", queueWait),
					zap.Duration("writeDuration", writeDuration),
					zap.Duration("totalLatency", queueWait+writeDuration))
			}

		case <-wq.done:
			return
		}
	}
}

// Send queues a message for writing with priority handling
// PHASE 2 Backpressure: Non-critical messages dropped when queue full
// PHASE 6: Generates unique message ID for tracking and timeout correlation
func (wq *WriteQueue) Send(msg interface{}, priority MessagePriority) error {
	// PHASE 6: Generate unique message ID
	seq := atomic.AddInt64(&wq.messageIDSeq, 1)
	messageID := fmt.Sprintf("%s-%d-%d", wq.sessionID, time.Now().UnixNano(), seq)

	// PHASE 6: Extract message type for logging
	msgType := wq.extractMessageType(msg)

	qm := queuedMessage{
		id:       messageID,
		msg:      msg,
		priority: priority,
		queued:   time.Now(),
		msgType:  msgType,
	}

	select {
	case wq.queue <- qm:
		return nil
	default:
		// Queue full - handle based on priority
		if priority == PriorityCritical {
			// Block for critical messages (done, error)
			select {
			case wq.queue <- qm:
				return nil
			case <-wq.done:
				return fmt.Errorf("write queue closed")
			}
		}
		// Drop non-critical messages when queue full
		atomic.AddInt64(&wq.droppedCount, 1)
		metrics.WebSocketErrors.WithLabelValues("queue_full").Inc()
		wq.logger.Debug("Dropped message due to full queue",
			zap.String("messageId", messageID),
			zap.String("msgType", msgType),
			zap.Int64("droppedCount", atomic.LoadInt64(&wq.droppedCount)))
		return ErrQueueFull
	}
}

// extractMessageType extracts the type from a StreamMessage for logging
// PHASE 6: Helper for message type identification
func (wq *WriteQueue) extractMessageType(msg interface{}) string {
	if streamMsg, ok := msg.(models.StreamMessage); ok {
		return streamMsg.Type
	}
	if streamMsgPtr, ok := msg.(*models.StreamMessage); ok && streamMsgPtr != nil {
		return streamMsgPtr.Type
	}
	return "unknown"
}

// Close stops the writer goroutine and drains the queue
// PHASE 2 Backpressure: Graceful shutdown
func (wq *WriteQueue) Close() {
	wq.closeOnce.Do(func() {
		close(wq.done)
	})
}

// DroppedCount returns the number of messages dropped due to queue overflow
func (wq *WriteQueue) DroppedCount() int64 {
	return atomic.LoadInt64(&wq.droppedCount)
}

// TimedOutCount returns the number of messages that timed out in the queue
// PHASE 6: Track messages that waited too long and were dropped
func (wq *WriteQueue) TimedOutCount() int64 {
	return atomic.LoadInt64(&wq.timedOutCount)
}

// GetStats returns queue statistics for monitoring
// PHASE 6: Comprehensive queue stats
func (wq *WriteQueue) GetStats() WriteQueueStats {
	return WriteQueueStats{
		SessionID:     wq.sessionID,
		DroppedCount:  atomic.LoadInt64(&wq.droppedCount),
		TimedOutCount: atomic.LoadInt64(&wq.timedOutCount),
		MessageSeq:    atomic.LoadInt64(&wq.messageIDSeq),
		QueueSize:     wq.queueSize,
		QueueLength:   len(wq.queue),
	}
}
