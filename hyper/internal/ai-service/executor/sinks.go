package executor

import (
	"sync"
	"time"

	"hyper/internal/models"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// PHASE 1: Write deadline constant for WebSocket writes
const (
	// writeTimeout is the maximum time to wait for a WebSocket write to complete.
	// Prevents goroutines from hanging indefinitely on slow/unresponsive clients.
	writeTimeout = 30 * time.Second
)

// ProgressNotifierInterface defines the interface for progress notifications.
// This avoids import cycles with handlers package.
type ProgressNotifierInterface interface {
	EmitProgress(sessionID primitive.ObjectID, message string)
}

// WebSocketBroadcasterInterface defines the interface for broadcasting to WebSocket connections.
// This avoids import cycles with handlers package.
// Used by SubchatOutputSink to stream directly to subchat WebSocket connections.
type WebSocketBroadcasterInterface interface {
	BroadcastToSession(sessionID primitive.ObjectID, message models.StreamMessage) error
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

// writeWithDeadline writes JSON to WebSocket with a deadline to prevent indefinite blocking.
// PHASE 1: All WebSocket writes should use this method to ensure timeouts.
func (w *WebSocketSink) writeWithDeadline(msg interface{}) error {
	// Set write deadline to prevent hanging on slow clients
	if err := w.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		w.logger.Warn("Failed to set write deadline", zap.Error(err))
		// Continue anyway - deadline failure shouldn't block writes
	}
	defer w.conn.SetWriteDeadline(time.Time{}) // Clear deadline after write

	return w.conn.WriteJSON(msg)
}

// SendToken sends a token to the WebSocket client
// PHASE 1: Uses writeWithDeadline to prevent indefinite blocking
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

	if err := w.writeWithDeadline(streamMsg); err != nil {
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
// PHASE 1: Uses writeWithDeadline to prevent indefinite blocking
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

	if err := w.writeWithDeadline(streamMsg); err != nil {
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
// PHASE 1: Uses writeWithDeadline to prevent indefinite blocking
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

	if err := w.writeWithDeadline(streamMsg); err != nil {
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
// PHASE 1: Uses writeWithDeadline to prevent indefinite blocking
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

	if err := w.writeWithDeadline(doneMsg); err != nil {
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
// PHASE 1: Uses writeWithDeadline to prevent indefinite blocking
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

	if err := w.writeWithDeadline(errMsg); err != nil {
		w.logger.Warn("Failed to send error message", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendMessageSaved sends the message ID to the WebSocket client for frontend reconciliation
// PHASE 1: Uses writeWithDeadline to prevent indefinite blocking
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

	if err := w.writeWithDeadline(savedMsg); err != nil {
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
// NOTE: Tool errors are NOT streamed to UI - they are internal guidance for the AI
func (p *ProgressNotificationSink) SendToolResult(toolID, result, errorMsg string, durationMs int) error {
	// Tool errors are handled internally by the AI, not shown to users
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

// ============================================================================
// SubchatOutputSink - Enhanced Implementation for Subchat Execution
// ============================================================================

// SubchatOutputSink implements StreamOutputSink for subchat execution via execute_subagent.
// This sink:
// 1. Broadcasts tokens to the SUBCHAT WebSocket for real-time display in subchat UI
// 2. Sends progress notifications to the PARENT session for status updates
// 3. Converts tool calls to plain English for user-friendly progress messages
//
// KEY FIX: Uses WebSocketBroadcaster instead of ProgressNotifier for subchat.
// ProgressNotifier requires pre-registration (only happens in streamAIResponse when user sends message).
// WebSocketBroadcaster sends directly to any connected WebSocket - no pre-registration needed.
// Thread-safe implementation with mutex protection.
type SubchatOutputSink struct {
	subchatSessionID primitive.ObjectID
	parentSessionID  primitive.ObjectID
	broadcaster      WebSocketBroadcasterInterface // For subchat real-time streaming
	notifier         ProgressNotifierInterface     // For parent progress updates
	agentName        string
	logger           *zap.Logger
	mu               sync.Mutex
}

// NewSubchatOutputSink creates a new subchat output sink.
// subchatSessionID: The session where subchat messages are displayed (uses broadcaster)
// parentSessionID: The parent session that receives progress notifications (uses notifier)
// broadcaster: The WebSocket broadcaster for direct streaming to subchat UI
// notifier: The progress notifier for parent session updates
// agentName: Name of the agent for progress messages
func NewSubchatOutputSink(
	subchatSessionID primitive.ObjectID,
	parentSessionID primitive.ObjectID,
	broadcaster WebSocketBroadcasterInterface,
	notifier ProgressNotifierInterface,
	agentName string,
	logger *zap.Logger,
) *SubchatOutputSink {
	return &SubchatOutputSink{
		subchatSessionID: subchatSessionID,
		parentSessionID:  parentSessionID,
		broadcaster:      broadcaster,
		notifier:         notifier,
		agentName:        agentName,
		logger:           logger,
	}
}

// SendToken sends a token to the subchat session for real-time display
// Uses WebSocketBroadcaster to send directly to subchat WebSocket (no pre-registration needed)
func (s *SubchatOutputSink) SendToken(content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if content == "" {
		return nil
	}

	// Broadcast to subchat WebSocket for real-time display
	msg := models.StreamMessage{
		Type:    "token",
		Content: content,
	}
	if err := s.broadcaster.BroadcastToSession(s.subchatSessionID, msg); err != nil {
		s.logger.Debug("Failed to broadcast token to subchat (client may not be connected yet)",
			zap.String("subchatSessionId", s.subchatSessionID.Hex()),
			zap.Error(err))
		// Don't return error - subchat may not have WebSocket connected yet
		// Messages are saved to DB and will appear on refresh
	}
	return nil
}

// SendToolCall sends a tool call notification to subchat
// Uses proper tool_call message type for frontend rendering (same as WebSocketSink)
func (s *SubchatOutputSink) SendToolCall(toolName, toolID string, args map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Send proper tool_call event (same format as WebSocketSink) for frontend rendering
	msg := models.StreamMessage{
		Type: "tool_call",
		ToolCall: &models.ToolCallEvent{
			Tool: toolName,
			Args: args,
			ID:   toolID,
		},
	}
	if err := s.broadcaster.BroadcastToSession(s.subchatSessionID, msg); err != nil {
		s.logger.Debug("Failed to broadcast tool call to subchat",
			zap.String("subchatSessionId", s.subchatSessionID.Hex()),
			zap.String("toolName", toolName),
			zap.Error(err))
	}
	return nil
}

// SendToolResult sends a tool result notification
// NOTE: Tool errors are NOT streamed to subchat UI - they are internal guidance for the AI
func (s *SubchatOutputSink) SendToolResult(toolID, result, errorMsg string, durationMs int) error {
	// Tool errors are handled internally by the AI, not shown to users
	// Success is implicit in the streaming flow
	return nil
}

// SendDone sends completion notification to both subchat and parent
func (s *SubchatOutputSink) SendDone() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Notify subchat that processing is done
	doneMsg := models.StreamMessage{
		Type: "done",
	}
	if err := s.broadcaster.BroadcastToSession(s.subchatSessionID, doneMsg); err != nil {
		s.logger.Debug("Failed to broadcast done to subchat",
			zap.String("subchatSessionId", s.subchatSessionID.Hex()),
			zap.Error(err))
	}

	// Notify parent session that subchat completed (via progress notifier)
	s.notifier.EmitProgress(s.parentSessionID, "✅ Subchat completed: "+s.agentName)
	return nil
}

// SendError sends error notification to both subchat and parent
func (s *SubchatOutputSink) SendError(errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Send error to subchat WebSocket
	errMsg := models.StreamMessage{
		Type:    "error",
		Content: errorMsg,
	}
	if err := s.broadcaster.BroadcastToSession(s.subchatSessionID, errMsg); err != nil {
		s.logger.Debug("Failed to broadcast error to subchat",
			zap.String("subchatSessionId", s.subchatSessionID.Hex()),
			zap.Error(err))
	}

	// Notify parent session about the error
	s.notifier.EmitProgress(s.parentSessionID, "⚠️ Subchat error: "+s.agentName+" - "+errorMsg)
	return nil
}

// SendMessageSaved sends the saved message ID to subchat for frontend reconciliation
func (s *SubchatOutputSink) SendMessageSaved(messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := models.StreamMessage{
		Type:    "message_saved",
		Content: messageID, // message ID is sent in Content field
	}
	if err := s.broadcaster.BroadcastToSession(s.subchatSessionID, msg); err != nil {
		s.logger.Debug("Failed to broadcast message_saved to subchat",
			zap.String("subchatSessionId", s.subchatSessionID.Hex()),
			zap.String("messageId", messageID),
			zap.Error(err))
	}
	return nil
}

// IsDisconnected always returns false for subchat (runs to completion in background)
func (s *SubchatOutputSink) IsDisconnected() bool {
	return false
}
