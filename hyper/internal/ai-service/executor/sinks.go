package executor

import (
	"sync"

	"hyper/internal/models"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// ProgressNotifierInterface defines the interface for progress notifications.
// This avoids import cycles with handlers package.
type ProgressNotifierInterface interface {
	EmitProgress(sessionID primitive.ObjectID, message string)
}

// StreamOutputSink defines the interface for streaming AI outputs to different destinations.
// This abstraction allows the executor to work with both WebSocket (parent chat) and
// ProgressNotifier (subchat) without knowing the specific implementation.
//
// Universal Stream Architecture:
// - WebSocketSink: For parent chat, streams directly to WebSocket client
// - ProgressNotificationSink: For subchat, streams via progress notification system
// - Both implement the same interface, making the executor context-agnostic
type StreamOutputSink interface {
	// SendToken sends a token (text chunk) to the output destination
	SendToken(content string) error

	// SendToolCall sends a tool call event to the output destination
	SendToolCall(toolName, toolID string, args map[string]interface{}) error

	// SendToolResult sends a tool result event to the output destination
	SendToolResult(toolID, result, errorMsg string, durationMs int) error

	// SendDone signals that streaming is complete
	SendDone() error

	// SendError sends an error message to the output destination
	SendError(errorMsg string) error

	// SendMessageSaved sends the database ID of a saved message for frontend reconciliation.
	// Format: "role:messageId" (e.g., "assistant:507f1f77bcf86cd799439011")
	SendMessageSaved(messageID string) error

	// IsDisconnected returns true if the client has disconnected
	// Used to detect early termination and avoid unnecessary work
	IsDisconnected() bool
}

// ============================================================================
// WebSocketSink - Implementation for Parent Chat
// ============================================================================

// WebSocketSink implements StreamOutputSink for WebSocket connections (parent chat).
// Thread-safe implementation with mutex protection for concurrent access.
type WebSocketSink struct {
	conn         *websocket.Conn
	logger       *zap.Logger
	disconnected bool
	mu           sync.Mutex
}

// NewWebSocketSink creates a new WebSocket output sink for parent chat streaming.
func NewWebSocketSink(conn *websocket.Conn, logger *zap.Logger) *WebSocketSink {
	return &WebSocketSink{
		conn:         conn,
		logger:       logger,
		disconnected: false,
	}
}

// SendToken sends a token to the WebSocket client
func (w *WebSocketSink) SendToken(content string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil // Silent return if already disconnected
	}

	streamMsg := models.StreamMessage{
		Type:    "token",
		Content: content,
	}

	if err := w.conn.WriteJSON(streamMsg); err != nil {
		// Check if this is a normal disconnection
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected during token streaming")
			w.disconnected = true
			return nil // Return nil to continue processing in background
		}
		w.logger.Warn("Failed to send token to WebSocket", zap.Error(err))
		w.disconnected = true
		return nil // Return nil to continue processing
	}

	return nil
}

// SendToolCall sends a tool call event to the WebSocket client
func (w *WebSocketSink) SendToolCall(toolName, toolID string, args map[string]interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil // Silent return if already disconnected
	}

	streamMsg := models.StreamMessage{
		Type: "tool_call",
		ToolCall: &models.ToolCallEvent{
			Tool: toolName,
			Args: args,
			ID:   toolID,
		},
	}

	if err := w.conn.WriteJSON(streamMsg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected during tool call streaming")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send tool call to WebSocket", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendToolResult sends a tool result event to the WebSocket client
func (w *WebSocketSink) SendToolResult(toolID, result, errorMsg string, durationMs int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil // Silent return if already disconnected
	}

	streamMsg := models.StreamMessage{
		Type: "tool_result",
		ToolResult: &models.ToolResultEvent{
			ID:         toolID,
			Result:     result,
			Error:      errorMsg,
			DurationMs: durationMs,
		},
	}

	if err := w.conn.WriteJSON(streamMsg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected during tool result streaming")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send tool result to WebSocket", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendDone sends completion signal to the WebSocket client
func (w *WebSocketSink) SendDone() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil // Silent return if already disconnected
	}

	doneMsg := models.StreamMessage{
		Type:    "done",
		Content: "",
	}

	if err := w.conn.WriteJSON(doneMsg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected before completion message")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send done message", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendError sends an error message to the WebSocket client
func (w *WebSocketSink) SendError(errorMsg string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil // Silent return if already disconnected
	}

	errMsg := models.StreamMessage{
		Type:    "error",
		Content: errorMsg,
	}

	if err := w.conn.WriteJSON(errMsg); err != nil {
		w.logger.Warn("Failed to send error message", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendMessageSaved sends the message ID to the WebSocket client for frontend reconciliation
func (w *WebSocketSink) SendMessageSaved(messageID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil // Silent return if already disconnected
	}

	savedMsg := models.StreamMessage{
		Type:    "message_saved",
		Content: messageID,
	}

	if err := w.conn.WriteJSON(savedMsg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected during message_saved")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send message_saved", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// IsDisconnected returns true if the WebSocket client has disconnected
func (w *WebSocketSink) IsDisconnected() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.disconnected
}

// ============================================================================
// ProgressNotificationSink - Implementation for Subchat
// ============================================================================

// ProgressNotificationSink implements StreamOutputSink for subchat execution.
// Sends output via the progress notification system to parent chat.
// Thread-safe implementation with mutex protection.
type ProgressNotificationSink struct {
	parentSessionID primitive.ObjectID
	notifier        ProgressNotifierInterface
	logger          *zap.Logger
	mu              sync.Mutex
}

// NewProgressNotificationSink creates a new progress notification sink for subchat streaming.
func NewProgressNotificationSink(parentSessionID primitive.ObjectID, notifier ProgressNotifierInterface, logger *zap.Logger) *ProgressNotificationSink {
	return &ProgressNotificationSink{
		parentSessionID: parentSessionID,
		notifier:        notifier,
		logger:          logger,
	}
}

// SendToken sends a token as a progress notification
// Tokens are NOT sent individually to avoid flooding the parent chat
func (p *ProgressNotificationSink) SendToken(content string) error {
	// For subchat, we don't stream individual tokens to parent
	// Tokens are accumulated and saved to database for subchat history
	return nil
}

// SendToolCall sends a tool call as a progress notification
func (p *ProgressNotificationSink) SendToolCall(toolName, toolID string, args map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.notifier.EmitProgress(p.parentSessionID, "🔧 Executing tool: "+toolName)
	return nil
}

// SendToolResult sends a tool result as a progress notification
func (p *ProgressNotificationSink) SendToolResult(toolID, result, errorMsg string, durationMs int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if errorMsg != "" {
		p.notifier.EmitProgress(p.parentSessionID, "⚠️ Tool error: "+errorMsg)
	}
	// Success is implicit, no need to spam parent chat
	return nil
}

// SendDone sends completion notification
func (p *ProgressNotificationSink) SendDone() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.notifier.EmitProgress(p.parentSessionID, "✅ Subchat execution complete")
	return nil
}

// SendError sends error notification
func (p *ProgressNotificationSink) SendError(errorMsg string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.notifier.EmitProgress(p.parentSessionID, "❌ Subchat error: "+errorMsg)
	return nil
}

// SendMessageSaved is a no-op for subchat (progress notifications don't need reconciliation)
func (p *ProgressNotificationSink) SendMessageSaved(messageID string) error {
	// Subchat messages are saved to their own session, no frontend reconciliation needed
	return nil
}

// IsDisconnected always returns false for subchat (no client connection)
// Subchat runs to completion in background regardless of parent chat state
func (p *ProgressNotificationSink) IsDisconnected() bool {
	return false
}
