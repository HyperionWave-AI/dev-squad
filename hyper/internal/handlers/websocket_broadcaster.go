package handlers

import (
	"hyper/internal/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// managedConnection wraps a WebSocket connection with health monitoring
type managedConnection struct {
	conn        *websocket.Conn
	writeMutex  *sync.Mutex
	done        chan struct{}
	lastPing    time.Time
	mu          sync.Mutex
	closeOnce   sync.Once  // PHASE 1: Prevents double-close panic on done channel
	closed      bool       // PHASE 2: Track if connection was closed
	closedAt    time.Time  // PHASE 2: When it was closed
	closeReason string     // PHASE 2: Why it was closed (for debugging)
}

// Close safely closes the done channel using sync.Once to prevent double-close panic
// PHASE 1: This method can be called multiple times safely
func (mc *managedConnection) Close(reason string) {
	mc.closeOnce.Do(func() {
		mc.mu.Lock()
		mc.closed = true
		mc.closedAt = time.Now()
		mc.closeReason = reason
		mc.mu.Unlock()
		close(mc.done)
	})
}

// IsClosed returns true if the connection has been closed
// PHASE 2: Used to skip operations on already-closed connections
func (mc *managedConnection) IsClosed() bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.closed
}

// WebSocketBroadcaster manages WebSocket connections and allows broadcasting events to specific sessions
type WebSocketBroadcaster struct {
	mu              sync.RWMutex
	connections     map[string]*managedConnection // map[sessionID]managedConn
	logger          *zap.Logger
	healthCheckDone chan struct{}
}

// Global singleton instance
var (
	broadcasterInstance *WebSocketBroadcaster
	broadcasterOnce     sync.Once
)

// GetWebSocketBroadcaster returns the singleton WebSocketBroadcaster instance
func GetWebSocketBroadcaster(logger *zap.Logger) *WebSocketBroadcaster {
	broadcasterOnce.Do(func() {
		broadcasterInstance = &WebSocketBroadcaster{
			connections:     make(map[string]*managedConnection),
			logger:          logger,
			healthCheckDone: make(chan struct{}),
		}
		// Start health monitoring goroutine
		go broadcasterInstance.healthCheckLoop()
	})
	return broadcasterInstance
}

// healthCheckLoop periodically checks connection health and removes dead connections
func (wb *WebSocketBroadcaster) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wb.checkConnectionHealth()
		case <-wb.healthCheckDone:
			return
		}
	}
}

// checkConnectionHealth sends pings to all connections and removes unresponsive ones
// PHASE 3: Improved safety with IsClosed check and safe Close() method
func (wb *WebSocketBroadcaster) checkConnectionHealth() {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	deadConnections := []string{}

	for sessionKey, mc := range wb.connections {
		// PHASE 3: Skip if already marked as closed (e.g., by RegisterConnection replacing it)
		if mc.IsClosed() {
			wb.logger.Debug("Skipping already-closed connection in health check",
				zap.String("sessionId", sessionKey),
				zap.String("closeReason", mc.closeReason))
			deadConnections = append(deadConnections, sessionKey)
			continue
		}

		mc.mu.Lock()
		// Try to send ping
		err := mc.conn.WriteControl(
			websocket.PingMessage,
			[]byte{},
			time.Now().Add(10*time.Second),
		)
		if err != nil {
			// Connection is dead
			wb.logger.Warn("Connection health check failed, marking for removal",
				zap.String("sessionId", sessionKey),
				zap.Error(err))
			deadConnections = append(deadConnections, sessionKey)
		} else {
			mc.lastPing = time.Now()
		}
		mc.mu.Unlock()
	}

	// Remove dead connections (PHASE 1: uses safe Close() method - no panic on double close)
	for _, sessionKey := range deadConnections {
		if mc, exists := wb.connections[sessionKey]; exists {
			mc.Close("health_check_failed")  // PHASE 1: Safe close with reason
			delete(wb.connections, sessionKey)
			wb.logger.Info("Removed dead connection from broadcaster",
				zap.String("sessionId", sessionKey),
				zap.String("reason", "health_check"))
		}
	}
}

// RegisterConnection registers a WebSocket connection for a session
// PHASE 4: Uses safe Close() method to prevent double-close panic on reconnection
func (wb *WebSocketBroadcaster) RegisterConnection(sessionID primitive.ObjectID, conn *websocket.Conn, writeMutex *sync.Mutex) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	sessionKey := sessionID.Hex()

	// PHASE 4: Close existing connection safely if it exists (handles reconnection case)
	if existing, exists := wb.connections[sessionKey]; exists {
		existing.Close("replaced_by_new_connection")  // Safe close - won't panic if already closed
		wb.logger.Debug("Closed existing connection for session (reconnection)",
			zap.String("sessionId", sessionKey))
	}

	// Create managed connection with health monitoring
	mc := &managedConnection{
		conn:       conn,
		writeMutex: writeMutex,
		done:       make(chan struct{}),
		lastPing:   time.Now(),
		// closeOnce, closed, closedAt, closeReason are zero-valued (correct initial state)
	}

	wb.connections[sessionKey] = mc

	wb.logger.Debug("Registered WebSocket connection for session",
		zap.String("sessionId", sessionKey))
}

// UnregisterConnection removes a WebSocket connection for a session
// PHASE 4: Uses safe Close() method to prevent double-close panic
func (wb *WebSocketBroadcaster) UnregisterConnection(sessionID primitive.ObjectID) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	sessionKey := sessionID.Hex()

	if mc, exists := wb.connections[sessionKey]; exists {
		mc.Close("unregistered")  // PHASE 4: Safe close - won't panic if already closed
		delete(wb.connections, sessionKey)

		wb.logger.Debug("Unregistered WebSocket connection for session",
			zap.String("sessionId", sessionKey))
	}
}

// BroadcastToSession sends a message to a specific session's WebSocket connection
// PHASE 4: Added IsClosed check to avoid writing to closed connections
func (wb *WebSocketBroadcaster) BroadcastToSession(sessionID primitive.ObjectID, message models.StreamMessage) error {
	wb.mu.RLock()
	sessionKey := sessionID.Hex()
	mc, exists := wb.connections[sessionKey]
	wb.mu.RUnlock()

	if !exists {
		wb.logger.Debug("No active WebSocket connection for session",
			zap.String("sessionId", sessionKey))
		return nil // Not an error - session may not have active connection
	}

	// PHASE 4: Skip if connection was already closed (prevents writing to closed connection)
	if mc.IsClosed() {
		wb.logger.Debug("Skipping broadcast to closed connection",
			zap.String("sessionId", sessionKey),
			zap.String("closeReason", mc.closeReason))
		return nil
	}

	// Use the session's write mutex to ensure thread-safe writes
	mc.writeMutex.Lock()
	defer mc.writeMutex.Unlock()

	err := mc.conn.WriteJSON(message)
	if err != nil {
		wb.logger.Warn("Failed to broadcast to session",
			zap.String("sessionId", sessionKey),
			zap.Error(err))
		return err
	}

	wb.logger.Debug("Broadcasted message to session",
		zap.String("sessionId", sessionKey),
		zap.String("messageType", message.Type))

	return nil
}
