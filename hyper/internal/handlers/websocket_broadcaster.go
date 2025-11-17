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
	conn       *websocket.Conn
	writeMutex *sync.Mutex
	done       chan struct{}
	lastPing   time.Time
	mu         sync.Mutex
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
func (wb *WebSocketBroadcaster) checkConnectionHealth() {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	deadConnections := []string{}

	for sessionKey, mc := range wb.connections {
		mc.mu.Lock()
		// Try to send ping
		err := mc.conn.WriteControl(
			websocket.PingMessage,
			[]byte{},
			time.Now().Add(5*time.Second),
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

	// Remove dead connections
	for _, sessionKey := range deadConnections {
		if mc, exists := wb.connections[sessionKey]; exists {
			close(mc.done)
			delete(wb.connections, sessionKey)
			wb.logger.Info("Auto-removed dead connection",
				zap.String("sessionId", sessionKey))
		}
	}
}

// RegisterConnection registers a WebSocket connection for a session
func (wb *WebSocketBroadcaster) RegisterConnection(sessionID primitive.ObjectID, conn *websocket.Conn, writeMutex *sync.Mutex) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	sessionKey := sessionID.Hex()

	// Close existing connection if it exists (handles reconnection case)
	if existing, exists := wb.connections[sessionKey]; exists {
		close(existing.done)
	}

	// Create managed connection with health monitoring
	mc := &managedConnection{
		conn:       conn,
		writeMutex: writeMutex,
		done:       make(chan struct{}),
		lastPing:   time.Now(),
	}

	wb.connections[sessionKey] = mc

	wb.logger.Debug("Registered WebSocket connection for session",
		zap.String("sessionId", sessionKey))
}

// UnregisterConnection removes a WebSocket connection for a session
func (wb *WebSocketBroadcaster) UnregisterConnection(sessionID primitive.ObjectID) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	sessionKey := sessionID.Hex()

	if mc, exists := wb.connections[sessionKey]; exists {
		close(mc.done)
		delete(wb.connections, sessionKey)

		wb.logger.Debug("Unregistered WebSocket connection for session",
			zap.String("sessionId", sessionKey))
	}
}

// BroadcastToSession sends a message to a specific session's WebSocket connection
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
