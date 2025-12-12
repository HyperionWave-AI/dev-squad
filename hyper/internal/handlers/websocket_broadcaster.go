package handlers

import (
	"context"
	"hyper/internal/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// PHASE 2: Write deadline constant for broadcaster writes
const (
	// broadcasterWriteTimeout is the maximum time to wait for a broadcast write.
	// Prevents goroutines from hanging indefinitely on slow/unresponsive clients.
	broadcasterWriteTimeout = 30 * time.Second
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

	// PHASE 3 Connection State: Metadata for distributed persistence
	userID      string    // User who owns this connection
	remoteAddr  string    // Client IP address
	userAgent   string    // Client user agent
	connectedAt time.Time // When the connection was established
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

	// PHASE 3 Connection State: Distributed state persistence
	stateStore ConnectionStateStore
}

// Global singleton instance
var (
	broadcasterInstance *WebSocketBroadcaster
	broadcasterOnce     sync.Once
)

// GetWebSocketBroadcaster returns the singleton WebSocketBroadcaster instance
func GetWebSocketBroadcaster(logger *zap.Logger) *WebSocketBroadcaster {
	broadcasterOnce.Do(func() {
		// PHASE 3: Initialize connection state store for distributed persistence
		stateStore := GetConnectionStateStore(logger)

		broadcasterInstance = &WebSocketBroadcaster{
			connections:     make(map[string]*managedConnection),
			logger:          logger,
			healthCheckDone: make(chan struct{}),
			stateStore:      stateStore,
		}
		// Start health monitoring goroutine
		go broadcasterInstance.healthCheckLoop()
		// PHASE 3: Start heartbeat loop for connection state persistence
		go broadcasterInstance.heartbeatLoop()
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

// ConnectionMetadata contains additional info about a connection for state persistence
// PHASE 3: Used to store connection metadata in distributed state store
type ConnectionMetadata struct {
	UserID     string
	RemoteAddr string
	UserAgent  string
}

// RegisterConnection registers a WebSocket connection for a session
// PHASE 4: Uses safe Close() method to prevent double-close panic on reconnection
func (wb *WebSocketBroadcaster) RegisterConnection(sessionID primitive.ObjectID, conn *websocket.Conn, writeMutex *sync.Mutex) {
	// Call the extended version with empty metadata for backward compatibility
	wb.RegisterConnectionWithMetadata(sessionID, conn, writeMutex, ConnectionMetadata{})
}

// RegisterConnectionWithMetadata registers a WebSocket connection with full metadata
// PHASE 3: Extended version that stores state in distributed store
func (wb *WebSocketBroadcaster) RegisterConnectionWithMetadata(sessionID primitive.ObjectID, conn *websocket.Conn, writeMutex *sync.Mutex, metadata ConnectionMetadata) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	sessionKey := sessionID.Hex()
	now := time.Now()

	// PHASE 4: Close existing connection safely if it exists (handles reconnection case)
	if existing, exists := wb.connections[sessionKey]; exists {
		existing.Close("replaced_by_new_connection") // Safe close - won't panic if already closed
		wb.logger.Debug("Closed existing connection for session (reconnection)",
			zap.String("sessionId", sessionKey))
	}

	// Create managed connection with health monitoring and metadata
	mc := &managedConnection{
		conn:        conn,
		writeMutex:  writeMutex,
		done:        make(chan struct{}),
		lastPing:    now,
		userID:      metadata.UserID,
		remoteAddr:  metadata.RemoteAddr,
		userAgent:   metadata.UserAgent,
		connectedAt: now,
		// closeOnce, closed, closedAt, closeReason are zero-valued (correct initial state)
	}

	wb.connections[sessionKey] = mc

	// PHASE 3: Store connection state in distributed store (async to not block)
	if wb.stateStore != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			state := ConnectionState{
				SessionID:    sessionKey,
				UserID:       metadata.UserID,
				InstanceID:   wb.stateStore.GetInstanceID(),
				ConnectedAt:  now,
				LastActivity: now,
				RemoteAddr:   metadata.RemoteAddr,
				UserAgent:    metadata.UserAgent,
			}

			if err := wb.stateStore.Register(ctx, state); err != nil {
				wb.logger.Warn("Failed to register connection state in store",
					zap.String("sessionId", sessionKey),
					zap.Error(err))
			}
		}()
	}

	wb.logger.Debug("Registered WebSocket connection for session",
		zap.String("sessionId", sessionKey),
		zap.String("userId", metadata.UserID))
}

// UnregisterConnection removes a WebSocket connection for a session
// PHASE 4: Uses safe Close() method to prevent double-close panic
func (wb *WebSocketBroadcaster) UnregisterConnection(sessionID primitive.ObjectID) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	sessionKey := sessionID.Hex()

	if mc, exists := wb.connections[sessionKey]; exists {
		mc.Close("unregistered") // PHASE 4: Safe close - won't panic if already closed
		delete(wb.connections, sessionKey)

		// PHASE 3: Remove connection state from distributed store (async)
		if wb.stateStore != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := wb.stateStore.Unregister(ctx, sessionKey); err != nil {
					wb.logger.Warn("Failed to unregister connection state from store",
						zap.String("sessionId", sessionKey),
						zap.Error(err))
				}
			}()
		}

		wb.logger.Debug("Unregistered WebSocket connection for session",
			zap.String("sessionId", sessionKey))
	}
}

// BroadcastToSession sends a message to a specific session's WebSocket connection
// PHASE 4: Added IsClosed check to avoid writing to closed connections
// PHASE 2 Write Deadlines: Added write deadline to prevent indefinite blocking
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

	// PHASE 2: Set write deadline to prevent indefinite blocking on slow clients
	if err := mc.conn.SetWriteDeadline(time.Now().Add(broadcasterWriteTimeout)); err != nil {
		wb.logger.Warn("Failed to set write deadline for broadcast",
			zap.String("sessionId", sessionKey),
			zap.Error(err))
		// Continue anyway - deadline failure shouldn't block writes
	}
	defer mc.conn.SetWriteDeadline(time.Time{}) // Clear deadline after write

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

// heartbeatLoop periodically sends heartbeats for all connections to refresh TTL
// PHASE 3: Keeps connection state alive in Redis
func (wb *WebSocketBroadcaster) heartbeatLoop() {
	// Heartbeat every 30 seconds (connection TTL is 2 minutes)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wb.sendHeartbeats()
		case <-wb.healthCheckDone:
			return
		}
	}
}

// sendHeartbeats sends heartbeats for all active connections
// PHASE 3: Refreshes TTL in distributed state store
func (wb *WebSocketBroadcaster) sendHeartbeats() {
	if wb.stateStore == nil {
		return
	}

	wb.mu.RLock()
	sessionIDs := make([]string, 0, len(wb.connections))
	for sessionID := range wb.connections {
		sessionIDs = append(sessionIDs, sessionID)
	}
	wb.mu.RUnlock()

	if len(sessionIDs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, sessionID := range sessionIDs {
		if err := wb.stateStore.Heartbeat(ctx, sessionID); err != nil {
			wb.logger.Warn("Failed to send heartbeat for connection",
				zap.String("sessionId", sessionID),
				zap.Error(err))
		}
	}

	wb.logger.Debug("Sent heartbeats for connections",
		zap.Int("count", len(sessionIDs)))
}

// GetConnectionCount returns the number of active connections on this instance
// PHASE 3: Local connection count
func (wb *WebSocketBroadcaster) GetConnectionCount() int {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	return len(wb.connections)
}

// GetDistributedConnectionCount returns the total connections across all instances
// PHASE 3: Distributed connection count from state store
func (wb *WebSocketBroadcaster) GetDistributedConnectionCount() (int64, error) {
	if wb.stateStore == nil {
		return int64(wb.GetConnectionCount()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return wb.stateStore.Count(ctx)
}

// ListAllConnections returns all connections across all instances
// PHASE 3: For admin/debugging purposes
func (wb *WebSocketBroadcaster) ListAllConnections() ([]ConnectionState, error) {
	if wb.stateStore == nil {
		// Return local connections only
		wb.mu.RLock()
		defer wb.mu.RUnlock()

		states := make([]ConnectionState, 0, len(wb.connections))
		for sessionID, mc := range wb.connections {
			states = append(states, ConnectionState{
				SessionID:    sessionID,
				UserID:       mc.userID,
				InstanceID:   GetInstanceID(),
				ConnectedAt:  mc.connectedAt,
				LastActivity: mc.lastPing,
				RemoteAddr:   mc.remoteAddr,
				UserAgent:    mc.userAgent,
			})
		}
		return states, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return wb.stateStore.ListAll(ctx)
}
