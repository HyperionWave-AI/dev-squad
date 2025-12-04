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

	// PHASE 1 Buffer Monitoring: Metrics for actual buffer usage detection
	pendingWrites         atomic.Int64 // Number of writes currently in progress
	totalWriteLatencyMs   atomic.Int64 // Sum of write latencies for average calculation
	writeCount            atomic.Int64 // Number of writes recorded
	consecutiveSlowWrites atomic.Int64 // Count of consecutive slow writes (>1s)
	lastWriteLatencyMs    atomic.Int64 // Most recent write latency in milliseconds
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
	EstimatedBufferUsage   int // percentage (0-100)

	// PHASE 1 Buffer Monitoring: Detailed buffer metrics
	PendingWrites         int64 // Number of writes currently in progress
	AverageWriteLatencyMs int64 // Average write latency in milliseconds
	ConsecutiveSlowWrites int64 // Count of consecutive slow writes
	LastWriteLatencyMs    int64 // Most recent write latency
	TotalWriteCount       int64 // Total number of writes recorded
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

	// PHASE 1: Calculate average write latency
	writeCount := m.writeCount.Load()
	avgWriteLatencyMs := int64(0)
	if writeCount > 0 {
		avgWriteLatencyMs = m.totalWriteLatencyMs.Load() / writeCount
	}

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

		// PHASE 1 Buffer Monitoring: Include detailed buffer metrics
		PendingWrites:         m.pendingWrites.Load(),
		AverageWriteLatencyMs: avgWriteLatencyMs,
		ConsecutiveSlowWrites: m.consecutiveSlowWrites.Load(),
		LastWriteLatencyMs:    m.lastWriteLatencyMs.Load(),
		TotalWriteCount:       writeCount,
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

// PHASE 2 Buffer Monitoring: Write operation tracking methods

// RecordWriteStart records the start of a write operation
// Call this before starting a WebSocket write
func (m *ConnectionHealthMonitor) RecordWriteStart() {
	m.pendingWrites.Add(1)
}

// RecordWriteEnd records the completion of a write operation
// Call this after a WebSocket write completes (success or failure)
func (m *ConnectionHealthMonitor) RecordWriteEnd(duration time.Duration) {
	m.pendingWrites.Add(-1)

	latencyMs := duration.Milliseconds()
	m.lastWriteLatencyMs.Store(latencyMs)
	m.totalWriteLatencyMs.Add(latencyMs)
	m.writeCount.Add(1)

	// Track consecutive slow writes (> 1 second)
	// This is a key indicator of buffer pressure
	if duration > time.Second {
		m.consecutiveSlowWrites.Add(1)
		m.logger.Debug("Slow write recorded",
			zap.Duration("duration", duration),
			zap.Int64("consecutiveSlowWrites", m.consecutiveSlowWrites.Load()))
	} else {
		// Reset on fast write - client has recovered
		if m.consecutiveSlowWrites.Load() > 0 {
			m.consecutiveSlowWrites.Store(0)
			m.logger.Debug("Slow write counter reset after fast write")
		}
	}
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

// estimateBufferUsage estimates the write buffer usage as a percentage (0-100)
// PHASE 2 Buffer Monitoring: Actual implementation based on write metrics
//
// This uses a heuristic based on multiple factors that indicate buffer pressure:
// 1. Pending writes - more pending writes = more buffer pressure
// 2. Average write latency - higher latency = slower client = more pressure
// 3. Consecutive slow writes - sustained slow writes = definite pressure
//
// The gorilla/websocket library doesn't expose bufferedAmount directly,
// so we infer buffer pressure from these observable metrics.
func (m *ConnectionHealthMonitor) estimateBufferUsage() int {
	pendingWrites := m.pendingWrites.Load()
	consecutiveSlowWrites := m.consecutiveSlowWrites.Load()

	// Calculate average write latency
	avgLatencyMs := int64(0)
	writeCount := m.writeCount.Load()
	if writeCount > 0 {
		avgLatencyMs = m.totalWriteLatencyMs.Load() / writeCount
	}

	// Calculate buffer usage as weighted sum of factors:
	usage := 0

	// Factor 1: Pending writes contribution (max 40%)
	// Each pending write adds 10%, capped at 40%
	// More pending writes = messages backing up in the write pipeline
	pendingContrib := int(pendingWrites * 10)
	if pendingContrib > 40 {
		pendingContrib = 40
	}
	usage += pendingContrib

	// Factor 2: Average latency contribution (max 30%)
	// Every 200ms of average latency adds 5%
	// High latency = client is slow to acknowledge writes
	latencyContrib := int(avgLatencyMs / 200 * 5)
	if latencyContrib > 30 {
		latencyContrib = 30
	}
	usage += latencyContrib

	// Factor 3: Consecutive slow writes contribution (max 50%)
	// Each consecutive slow write adds 10%
	// This is the strongest indicator of a slow client
	slowContrib := int(consecutiveSlowWrites * 10)
	if slowContrib > 50 {
		slowContrib = 50
	}
	usage += slowContrib

	// Cap at 100%
	if usage > 100 {
		usage = 100
	}

	return usage
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
