package handlers

import (
	"sync"

	"hyper/internal/models"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// websocketSink is a local adapter that implements executor.StreamOutputSink for WebSocket connections.
// This avoids import cycles by implementing the interface locally in handlers package.
type websocketSink struct {
	conn         *websocket.Conn
	logger       *zap.Logger
	handler      *ChatWebSocketHandler
	disconnected bool
	mu           sync.Mutex
	writeQueue   *WriteQueue // PHASE 2: Buffered write queue
	sessionID    string      // PHASE 3 Buffer Monitoring: Session ID for health monitor lookup
}

// newWebSocketSink creates a WebSocket sink adapter.
func newWebSocketSink(conn *websocket.Conn, handler *ChatWebSocketHandler, logger *zap.Logger) *websocketSink {
	return &websocketSink{
		conn:         conn,
		handler:      handler,
		logger:       logger,
		disconnected: false,
	}
}

// newWebSocketSinkWithSession creates a WebSocket sink adapter with session ID for buffer monitoring
// PHASE 3 Buffer Monitoring: Enables buffer usage tracking via health monitor
func newWebSocketSinkWithSession(conn *websocket.Conn, handler *ChatWebSocketHandler, logger *zap.Logger, sessionID string) *websocketSink {
	return &websocketSink{
		conn:         conn,
		handler:      handler,
		logger:       logger,
		disconnected: false,
		sessionID:    sessionID,
	}
}

// SendToken implements executor.StreamOutputSink
// PHASE 3 Buffer Monitoring: Uses monitored write when session ID is available
func (w *websocketSink) SendToken(content string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	streamMsg := models.StreamMessage{
		Type:    "token",
		Content: content,
	}

	// PHASE 3: Use monitored write for buffer tracking
	if err := w.handler.safeWriteJSONWithMonitoring(w.conn, streamMsg, w.sessionID); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected during token streaming")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send token to WebSocket", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendToolCall implements executor.StreamOutputSink
func (w *websocketSink) SendToolCall(toolName, toolID string, args map[string]interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	streamMsg := models.StreamMessage{
		Type: "tool_call",
		ToolCall: &models.ToolCallEvent{
			Tool: toolName,
			Args: args,
			ID:   toolID,
		},
	}

	if err := w.handler.safeWriteJSON(w.conn, streamMsg); err != nil {
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

// SendToolResult implements executor.StreamOutputSink
func (w *websocketSink) SendToolResult(toolID, result, errorMsg string, durationMs int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
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

	if err := w.handler.safeWriteJSON(w.conn, streamMsg); err != nil {
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

// SendDone implements executor.StreamOutputSink
func (w *websocketSink) SendDone() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	doneMsg := models.StreamMessage{
		Type:    "done",
		Content: "",
	}

	if err := w.handler.safeWriteJSON(w.conn, doneMsg); err != nil {
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

// SendError implements executor.StreamOutputSink
func (w *websocketSink) SendError(errorMsg string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	errMsg := models.StreamMessage{
		Type:    "error",
		Content: errorMsg,
	}

	if err := w.handler.safeWriteJSON(w.conn, errMsg); err != nil {
		w.logger.Warn("Failed to send error message", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendMessageSaved implements executor.StreamOutputSink
func (w *websocketSink) SendMessageSaved(messageID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	savedMsg := models.StreamMessage{
		Type:    "message_saved",
		Content: messageID,
	}

	if err := w.handler.safeWriteJSON(w.conn, savedMsg); err != nil {
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

// IsDisconnected implements executor.StreamOutputSink
func (w *websocketSink) IsDisconnected() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.disconnected
}

// SendSystemNotification sends a system notification through the WebSocket
func (w *websocketSink) SendSystemNotification(notification models.SystemNotification) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	msg := models.StreamMessage{
		Type:         "system_notification",
		Notification: &notification,
	}

	if err := w.handler.safeWriteJSON(w.conn, msg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected during system notification")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send system notification to WebSocket", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}
