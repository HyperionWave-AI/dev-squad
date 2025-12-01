package handlers

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ConnectionHealthMonitor tracks WebSocket connection health with ping/pong timeouts
// and detects write buffer issues
type ConnectionHealthMonitor struct {
	conn                *websocket.Conn
	logger              *zap.Logger
	writeMutex          *sync.Mutex
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
	isHealthy           atomic.Bool
	lastPongTime        atomic.Int64
	pingsSent           atomic.Int64
	pongsReceived       atomic.Int64
	writeBufferWarnings atomic.Int64
	disconnectReason    string
	disconnectMutex     sync.Mutex
}

// HealthStatus represents the current health status of a connection
type HealthStatus struct {
	IsHealthy              bool
	LastPongTime           time.Time
	PingsSent              int64
	PongsReceived          int64
	WriteBufferWarnings    int64
	PongResponseRate       float64 // percentage of pongs received
	DisconnectReason       string
	TimeSinceLastPong      time.Duration
	EstimatedBufferUsage   int // percentage
}

// NewConnectionHealthMonitor creates a new health monitor for a WebSocket connection
func NewConnectionHealthMonitor(conn *websocket.Conn, logger *zap.Logger, writeMutex *sync.Mutex) *ConnectionHealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &ConnectionHealthMonitor{
		conn:       conn,
		logger:     logger,
		writeMutex: writeMutex,
		ctx:        ctx,
		cancel:     cancel,
	}

	monitor.isHealthy.Store(true)
	monitor.lastPongTime.Store(time.Now().UnixMilli())

	return monitor
}

// Start begins monitoring connection health
// This should be called in a separate goroutine
func (m *ConnectionHealthMonitor) Start() {
	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop stops the health monitor
func (m *ConnectionHealthMonitor) Stop() {
	m.cancel()
	m.wg.Wait()
}

// GetStatus returns the current health status
func (m *ConnectionHealthMonitor) GetStatus() HealthStatus {
	lastPongMs := m.lastPongTime.Load()
	lastPongTime := time.UnixMilli(lastPongMs)
	timeSinceLastPong := time.Since(lastPongTime)

	pingsSent := m.pingsSent.Load()
	pongsReceived := m.pongsReceived.Load()

	pongResponseRate := 0.0
	if pingsSent > 0 {
		pongResponseRate = (float64(pongsReceived) / float64(pingsSent)) * 100
	}

	m.disconnectMutex.Lock()
	disconnectReason := m.disconnectReason
	m.disconnectMutex.Unlock()

	return HealthStatus{
		IsHealthy:            m.isHealthy.Load(),
		LastPongTime:         lastPongTime,
		PingsSent:            pingsSent,
		PongsReceived:        pongsReceived,
		WriteBufferWarnings:  m.writeBufferWarnings.Load(),
		PongResponseRate:     pongResponseRate,
		DisconnectReason:     disconnectReason,
		TimeSinceLastPong:    timeSinceLastPong,
		EstimatedBufferUsage: m.estimateBufferUsage(),
	}
}

// RecordPongReceived records that a pong was received
func (m *ConnectionHealthMonitor) RecordPongReceived() {
	m.pongsReceived.Add(1)
	m.lastPongTime.Store(time.Now().UnixMilli())

	// Restore health if it was degraded
	if !m.isHealthy.Load() {
		m.isHealthy.Store(true)
		m.logger.Info("Connection health restored after pong received")
	}
}

// RecordWriteBufferWarning records a write buffer warning
func (m *ConnectionHealthMonitor) RecordWriteBufferWarning() {
	m.writeBufferWarnings.Add(1)
}

// SetDisconnectReason sets the reason for disconnection
func (m *ConnectionHealthMonitor) SetDisconnectReason(reason string) {
	m.disconnectMutex.Lock()
	defer m.disconnectMutex.Unlock()
	m.disconnectReason = reason
}

// monitorLoop runs the health check loop
func (m *ConnectionHealthMonitor) monitorLoop() {
	defer m.wg.Done()

	// Ping ticker - send ping every 30 seconds
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Health check ticker - check health every 5 seconds
	healthCheckTicker := time.NewTicker(5 * time.Second)
	defer healthCheckTicker.Stop()

	// Pong timeout - 10 seconds to receive pong after sending ping
	const pongTimeout = 10 * time.Second

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Debug("Connection health monitor stopped")
			return

		case <-pingTicker.C:
			// Send ping
			m.writeMutex.Lock()
			err := m.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
			m.writeMutex.Unlock()

			if err != nil {
				m.isHealthy.Store(false)
				m.logger.Warn("Failed to send ping",
					zap.Error(err),
					zap.String("reason", "write_error"))
				m.SetDisconnectReason("ping_write_failed")
				return
			}

			m.pingsSent.Add(1)
			m.logger.Debug("Ping sent",
				zap.Int64("pingsSent", m.pingsSent.Load()))

		case <-healthCheckTicker.C:
			// Check if we've received a pong recently
			lastPongMs := m.lastPongTime.Load()
			lastPongTime := time.UnixMilli(lastPongMs)
			timeSinceLastPong := time.Since(lastPongTime)

			if timeSinceLastPong > pongTimeout {
				m.isHealthy.Store(false)
				m.logger.Warn("Connection unhealthy - pong timeout",
					zap.Duration("timeSinceLastPong", timeSinceLastPong),
					zap.Duration("timeout", pongTimeout),
					zap.Int64("pingsSent", m.pingsSent.Load()),
					zap.Int64("pongsReceived", m.pongsReceived.Load()))
				m.SetDisconnectReason("pong_timeout")
				return
			}

			// Check pong response rate
			pingsSent := m.pingsSent.Load()
			pongsReceived := m.pongsReceived.Load()

			if pingsSent > 5 {
				responseRate := float64(pongsReceived) / float64(pingsSent)
				if responseRate < 0.5 {
					m.isHealthy.Store(false)
					m.logger.Warn("Connection unhealthy - low pong response rate",
						zap.Float64("responseRate", responseRate),
						zap.Int64("pingsSent", pingsSent),
						zap.Int64("pongsReceived", pongsReceived))
					m.SetDisconnectReason("low_pong_response_rate")
					return
				}
			}

			// Check write buffer usage
			bufferUsage := m.estimateBufferUsage()
			if bufferUsage > 80 {
				m.logger.Warn("Connection unhealthy - high write buffer usage",
					zap.Int("bufferUsage", bufferUsage))
				m.SetDisconnectReason("write_buffer_full")
				return
			}

			m.logger.Debug("Connection health check passed",
				zap.Duration("timeSinceLastPong", timeSinceLastPong),
				zap.Int("bufferUsage", bufferUsage),
				zap.Float64("pongResponseRate", float64(pongsReceived)/float64(pingsSent)*100))
		}
	}
}

// estimateBufferUsage estimates the write buffer usage as a percentage
// This is a heuristic based on the bufferedAmount property
func (m *ConnectionHealthMonitor) estimateBufferUsage() int {
	// Get bufferedAmount from the underlying connection
	// Note: This is a Go-specific property that may not be available on all platforms
	// For now, we return 0 as a placeholder - in production, you'd need to
	// track this through the write operations
	return 0
}

// ConnectionHealthMonitorPool manages multiple health monitors for different connections
type ConnectionHealthMonitorPool struct {
	monitors sync.Map // map[string]*ConnectionHealthMonitor
	logger   *zap.Logger
}

// NewConnectionHealthMonitorPool creates a new pool of health monitors
func NewConnectionHealthMonitorPool(logger *zap.Logger) *ConnectionHealthMonitorPool {
	return &ConnectionHealthMonitorPool{
		logger: logger,
	}
}

// Register registers a new health monitor
func (p *ConnectionHealthMonitorPool) Register(sessionID string, monitor *ConnectionHealthMonitor) {
	p.monitors.Store(sessionID, monitor)
	monitor.Start()
	p.logger.Debug("Health monitor registered", zap.String("sessionID", sessionID))
}

// Unregister unregisters a health monitor
func (p *ConnectionHealthMonitorPool) Unregister(sessionID string) {
	if val, ok := p.monitors.LoadAndDelete(sessionID); ok {
		monitor := val.(*ConnectionHealthMonitor)
		monitor.Stop()
		p.logger.Debug("Health monitor unregistered", zap.String("sessionID", sessionID))
	}
}

// GetStatus gets the status of a specific health monitor
func (p *ConnectionHealthMonitorPool) GetStatus(sessionID string) (HealthStatus, bool) {
	if val, ok := p.monitors.Load(sessionID); ok {
		monitor := val.(*ConnectionHealthMonitor)
		return monitor.GetStatus(), true
	}
	return HealthStatus{}, false
}

// GetAllStatuses gets the status of all health monitors
func (p *ConnectionHealthMonitorPool) GetAllStatuses() map[string]HealthStatus {
	statuses := make(map[string]HealthStatus)
	p.monitors.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		monitor := value.(*ConnectionHealthMonitor)
		statuses[sessionID] = monitor.GetStatus()
		return true
	})
	return statuses
}

// Global health monitor pool
var healthMonitorPool *ConnectionHealthMonitorPool

// GetHealthMonitorPool returns the global health monitor pool
func GetHealthMonitorPool(logger *zap.Logger) *ConnectionHealthMonitorPool {
	if healthMonitorPool == nil {
		healthMonitorPool = NewConnectionHealthMonitorPool(logger)
	}
	return healthMonitorPool
}
