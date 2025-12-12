package handlers

import (
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int32

const (
	// CircuitClosed - normal operation, requests flow through
	CircuitClosed CircuitState = iota
	// CircuitOpen - failures exceeded threshold, requests blocked
	CircuitOpen
	// CircuitHalfOpen - testing if service recovered, limited requests allowed
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold int
	// SuccessThreshold is the number of successes needed to close circuit from half-open
	SuccessThreshold int
	// OpenTimeout is how long to wait before transitioning from open to half-open
	OpenTimeout time.Duration
	// SlowCallThreshold is the duration above which a call is considered slow
	SlowCallThreshold time.Duration
	// SlowCallRateThreshold is the percentage of slow calls (0-100) that triggers open
	SlowCallRateThreshold int
	// MinCallsBeforeTrip is the minimum number of calls before circuit can trip
	MinCallsBeforeTrip int
}

// DefaultCircuitBreakerConfig returns sensible defaults for WebSocket client protection
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:      5,                // Trip after 5 consecutive failures
		SuccessThreshold:      3,                // Need 3 successes to close from half-open
		OpenTimeout:           30 * time.Second, // Wait 30s before testing
		SlowCallThreshold:     2 * time.Second,  // Calls >2s are considered slow
		SlowCallRateThreshold: 80,               // 80% slow calls triggers open
		MinCallsBeforeTrip:    3,                // Need at least 3 calls before tripping
	}
}

// CircuitBreaker implements the circuit breaker pattern for slow/stuck clients
// It protects the server from wasting resources on unresponsive clients
type CircuitBreaker struct {
	config CircuitBreakerConfig
	logger *zap.Logger
	name   string // Identifier for logging (e.g., sessionID)

	// State
	state     atomic.Int32 // CircuitState
	lastError atomic.Value // error

	// Counters (reset on state transitions)
	failures        atomic.Int64
	successes       atomic.Int64
	slowCalls       atomic.Int64
	totalCalls      atomic.Int64
	consecutiveFail atomic.Int64

	// Timing
	openedAt     atomic.Int64 // Unix timestamp when circuit opened
	lastCallTime atomic.Int64 // Unix timestamp of last call

	// State transition callback
	onStateChange func(from, to CircuitState)

	// Mutex for state transitions
	mu sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker with the given config
func NewCircuitBreaker(name string, config CircuitBreakerConfig, logger *zap.Logger) *CircuitBreaker {
	cb := &CircuitBreaker{
		config: config,
		logger: logger,
		name:   name,
	}
	cb.state.Store(int32(CircuitClosed))
	return cb
}

// SetStateChangeCallback sets a callback for state transitions
func (cb *CircuitBreaker) SetStateChangeCallback(callback func(from, to CircuitState)) {
	cb.onStateChange = callback
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	return CircuitState(cb.state.Load())
}

// IsOpen returns true if the circuit is open (blocking requests)
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.State() == CircuitOpen
}

// IsClosed returns true if the circuit is closed (normal operation)
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.State() == CircuitClosed
}

// IsHalfOpen returns true if the circuit is half-open (testing)
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.State() == CircuitHalfOpen
}

// AllowRequest checks if a request should be allowed through
// Returns true if request is allowed, false if circuit is open
func (cb *CircuitBreaker) AllowRequest() bool {
	state := cb.State()

	switch state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if enough time has passed to try again
		openedAt := cb.openedAt.Load()
		if time.Since(time.UnixMilli(openedAt)) >= cb.config.OpenTimeout {
			cb.transitionTo(CircuitHalfOpen)
			return true // Allow test request
		}
		return false

	case CircuitHalfOpen:
		// Allow limited requests for testing
		return true
	}

	return false
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess(duration time.Duration) {
	cb.lastCallTime.Store(time.Now().UnixMilli())
	cb.totalCalls.Add(1)
	cb.successes.Add(1)
	cb.consecutiveFail.Store(0) // Reset consecutive failures

	// Track slow calls
	if duration > cb.config.SlowCallThreshold {
		cb.slowCalls.Add(1)
	}

	state := cb.State()
	if state == CircuitHalfOpen {
		// Check if we have enough successes to close
		if cb.successes.Load() >= int64(cb.config.SuccessThreshold) {
			cb.transitionTo(CircuitClosed)
		}
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure(err error) {
	cb.lastCallTime.Store(time.Now().UnixMilli())
	cb.totalCalls.Add(1)
	cb.failures.Add(1)
	consecutiveFails := cb.consecutiveFail.Add(1)
	cb.lastError.Store(err)

	state := cb.State()

	switch state {
	case CircuitClosed:
		// Check if we should trip the circuit
		if cb.shouldTrip(consecutiveFails) {
			cb.transitionTo(CircuitOpen)
		}

	case CircuitHalfOpen:
		// Any failure in half-open goes back to open
		cb.transitionTo(CircuitOpen)
	}
}

// RecordSlowCall records a slow but successful call
func (cb *CircuitBreaker) RecordSlowCall(duration time.Duration) {
	cb.lastCallTime.Store(time.Now().UnixMilli())
	cb.totalCalls.Add(1)
	cb.successes.Add(1)
	cb.slowCalls.Add(1)
	cb.consecutiveFail.Store(0)

	// Check slow call rate threshold
	if cb.State() == CircuitClosed && cb.shouldTripOnSlowCalls() {
		cb.transitionTo(CircuitOpen)
	}
}

// RecordTimeout records a timeout (special type of failure)
func (cb *CircuitBreaker) RecordTimeout() {
	cb.RecordFailure(&TimeoutError{})
}

// TimeoutError represents a timeout failure
type TimeoutError struct{}

func (e *TimeoutError) Error() string {
	return "operation timed out"
}

// shouldTrip checks if the circuit should trip based on failures
func (cb *CircuitBreaker) shouldTrip(consecutiveFails int64) bool {
	totalCalls := cb.totalCalls.Load()

	// Need minimum calls before tripping
	if totalCalls < int64(cb.config.MinCallsBeforeTrip) {
		return false
	}

	// Trip on consecutive failures
	if consecutiveFails >= int64(cb.config.FailureThreshold) {
		cb.logger.Warn("Circuit breaker tripping on consecutive failures",
			zap.String("name", cb.name),
			zap.Int64("consecutiveFailures", consecutiveFails),
			zap.Int("threshold", cb.config.FailureThreshold))
		return true
	}

	return false
}

// shouldTripOnSlowCalls checks if slow call rate is too high
func (cb *CircuitBreaker) shouldTripOnSlowCalls() bool {
	totalCalls := cb.totalCalls.Load()
	slowCalls := cb.slowCalls.Load()

	// Need minimum calls
	if totalCalls < int64(cb.config.MinCallsBeforeTrip) {
		return false
	}

	slowRate := int(float64(slowCalls) / float64(totalCalls) * 100)
	if slowRate >= cb.config.SlowCallRateThreshold {
		cb.logger.Warn("Circuit breaker tripping on slow call rate",
			zap.String("name", cb.name),
			zap.Int("slowCallRate", slowRate),
			zap.Int("threshold", cb.config.SlowCallRateThreshold),
			zap.Int64("slowCalls", slowCalls),
			zap.Int64("totalCalls", totalCalls))
		return true
	}

	return false
}

// transitionTo transitions the circuit to a new state
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := CircuitState(cb.state.Load())
	if oldState == newState {
		return // Already in this state
	}

	cb.logger.Info("Circuit breaker state transition",
		zap.String("name", cb.name),
		zap.String("from", oldState.String()),
		zap.String("to", newState.String()),
		zap.Int64("failures", cb.failures.Load()),
		zap.Int64("successes", cb.successes.Load()),
		zap.Int64("slowCalls", cb.slowCalls.Load()))

	// Reset counters on state change
	cb.failures.Store(0)
	cb.successes.Store(0)
	cb.slowCalls.Store(0)
	cb.totalCalls.Store(0)
	cb.consecutiveFail.Store(0)

	// Record open time
	if newState == CircuitOpen {
		cb.openedAt.Store(time.Now().UnixMilli())
	}

	cb.state.Store(int32(newState))

	// Notify callback
	if cb.onStateChange != nil {
		go cb.onStateChange(oldState, newState)
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.logger.Info("Circuit breaker manually reset",
		zap.String("name", cb.name),
		zap.String("previousState", cb.State().String()))

	cb.failures.Store(0)
	cb.successes.Store(0)
	cb.slowCalls.Store(0)
	cb.totalCalls.Store(0)
	cb.consecutiveFail.Store(0)
	cb.state.Store(int32(CircuitClosed))
}

// GetStats returns current circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	return CircuitBreakerStats{
		Name:             cb.name,
		State:            cb.State(),
		Failures:         cb.failures.Load(),
		Successes:        cb.successes.Load(),
		SlowCalls:        cb.slowCalls.Load(),
		TotalCalls:       cb.totalCalls.Load(),
		ConsecutiveFails: cb.consecutiveFail.Load(),
		OpenedAt:         time.UnixMilli(cb.openedAt.Load()),
		LastCallTime:     time.UnixMilli(cb.lastCallTime.Load()),
	}
}

// CircuitBreakerStats holds statistics for a circuit breaker
type CircuitBreakerStats struct {
	Name             string
	State            CircuitState
	Failures         int64
	Successes        int64
	SlowCalls        int64
	TotalCalls       int64
	ConsecutiveFails int64
	OpenedAt         time.Time
	LastCallTime     time.Time
}

// CircuitBreakerRegistry manages multiple circuit breakers (one per connection)
type CircuitBreakerRegistry struct {
	breakers sync.Map // map[string]*CircuitBreaker
	config   CircuitBreakerConfig
	logger   *zap.Logger
}

// NewCircuitBreakerRegistry creates a new registry with default config
func NewCircuitBreakerRegistry(logger *zap.Logger) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		config: DefaultCircuitBreakerConfig(),
		logger: logger,
	}
}

// NewCircuitBreakerRegistryWithConfig creates a new registry with custom config
func NewCircuitBreakerRegistryWithConfig(config CircuitBreakerConfig, logger *zap.Logger) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		config: config,
		logger: logger,
	}
}

// Get returns or creates a circuit breaker for the given session
func (r *CircuitBreakerRegistry) Get(sessionID string) *CircuitBreaker {
	if cb, ok := r.breakers.Load(sessionID); ok {
		return cb.(*CircuitBreaker)
	}

	// Create new circuit breaker
	cb := NewCircuitBreaker(sessionID, r.config, r.logger)
	actual, loaded := r.breakers.LoadOrStore(sessionID, cb)
	if loaded {
		return actual.(*CircuitBreaker)
	}

	r.logger.Debug("Created circuit breaker for session",
		zap.String("sessionId", sessionID))
	return cb
}

// Remove removes a circuit breaker for a session
func (r *CircuitBreakerRegistry) Remove(sessionID string) {
	if _, ok := r.breakers.LoadAndDelete(sessionID); ok {
		r.logger.Debug("Removed circuit breaker for session",
			zap.String("sessionId", sessionID))
	}
}

// GetAllStats returns stats for all circuit breakers
func (r *CircuitBreakerRegistry) GetAllStats() map[string]CircuitBreakerStats {
	stats := make(map[string]CircuitBreakerStats)
	r.breakers.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		cb := value.(*CircuitBreaker)
		stats[sessionID] = cb.GetStats()
		return true
	})
	return stats
}

// GetOpenCircuits returns sessions with open circuit breakers
func (r *CircuitBreakerRegistry) GetOpenCircuits() []string {
	var open []string
	r.breakers.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		cb := value.(*CircuitBreaker)
		if cb.IsOpen() {
			open = append(open, sessionID)
		}
		return true
	})
	return open
}

// Global circuit breaker registry
var (
	circuitBreakerRegistry     *CircuitBreakerRegistry
	circuitBreakerRegistryOnce sync.Once
)

// GetCircuitBreakerRegistry returns the global circuit breaker registry
func GetCircuitBreakerRegistry(logger *zap.Logger) *CircuitBreakerRegistry {
	circuitBreakerRegistryOnce.Do(func() {
		circuitBreakerRegistry = NewCircuitBreakerRegistry(logger)
		logger.Info("Circuit breaker registry initialized",
			zap.Int("failureThreshold", circuitBreakerRegistry.config.FailureThreshold),
			zap.Int("successThreshold", circuitBreakerRegistry.config.SuccessThreshold),
			zap.Duration("openTimeout", circuitBreakerRegistry.config.OpenTimeout))
	})
	return circuitBreakerRegistry
}
