package handlers

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PHASE 1: Connection state model and storage interface for distributed persistence

// ConnectionState represents the persisted state of a WebSocket connection
// PHASE 1: This state is stored in Redis for cross-instance visibility
type ConnectionState struct {
	SessionID    string    `json:"sessionId"`    // MongoDB ObjectID hex string
	UserID       string    `json:"userId"`       // User who owns this connection
	InstanceID   string    `json:"instanceId"`   // Server instance owning this connection
	ConnectedAt  time.Time `json:"connectedAt"`  // When the connection was established
	LastActivity time.Time `json:"lastActivity"` // Last heartbeat/activity time
	RemoteAddr   string    `json:"remoteAddr"`   // Client IP address
	UserAgent    string    `json:"userAgent"`    // Client user agent
}

// ConnectionStateStore defines the interface for connection state persistence
// PHASE 1: Both Redis and in-memory implementations satisfy this interface
type ConnectionStateStore interface {
	// Register stores a new connection state
	Register(ctx context.Context, state ConnectionState) error

	// Unregister removes a connection state
	Unregister(ctx context.Context, sessionID string) error

	// Heartbeat updates the last activity time (refreshes TTL in Redis)
	Heartbeat(ctx context.Context, sessionID string) error

	// Get retrieves a connection state by session ID
	Get(ctx context.Context, sessionID string) (*ConnectionState, error)

	// ListByInstance returns all connections for a specific instance
	ListByInstance(ctx context.Context, instanceID string) ([]ConnectionState, error)

	// ListAll returns all active connections across all instances
	ListAll(ctx context.Context) ([]ConnectionState, error)

	// Count returns the total number of active connections
	Count(ctx context.Context) (int64, error)

	// GetInstanceID returns the instance ID for this store
	GetInstanceID() string
}

// Instance ID management
// PHASE 1: Each server instance has a unique ID for distributed tracking

var (
	instanceID     string
	instanceIDOnce sync.Once
)

// GetInstanceID returns the unique instance ID for this server
// PHASE 1: Uses INSTANCE_ID env var or generates a UUID
func GetInstanceID() string {
	instanceIDOnce.Do(func() {
		instanceID = os.Getenv("INSTANCE_ID")
		if instanceID == "" {
			// Generate a unique instance ID
			instanceID = uuid.New().String()[:8] // Short UUID for readability
		}
	})
	return instanceID
}

// InMemoryConnectionStateStore provides a local-only implementation
// PHASE 1: Used as fallback when Redis is not available
type InMemoryConnectionStateStore struct {
	mu         sync.RWMutex
	states     map[string]ConnectionState // sessionID -> ConnectionState
	instanceID string
	logger     *zap.Logger
}

// NewInMemoryConnectionStateStore creates a new in-memory connection state store
// PHASE 1: Fallback implementation for single-instance deployments
func NewInMemoryConnectionStateStore(instanceID string, logger *zap.Logger) *InMemoryConnectionStateStore {
	return &InMemoryConnectionStateStore{
		states:     make(map[string]ConnectionState),
		instanceID: instanceID,
		logger:     logger,
	}
}

func (s *InMemoryConnectionStateStore) Register(ctx context.Context, state ConnectionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state.SessionID] = state
	s.logger.Debug("Connection state registered (in-memory)",
		zap.String("sessionId", state.SessionID))
	return nil
}

func (s *InMemoryConnectionStateStore) Unregister(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, sessionID)
	s.logger.Debug("Connection state unregistered (in-memory)",
		zap.String("sessionId", sessionID))
	return nil
}

func (s *InMemoryConnectionStateStore) Heartbeat(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state, exists := s.states[sessionID]; exists {
		state.LastActivity = time.Now()
		s.states[sessionID] = state
	}
	return nil
}

func (s *InMemoryConnectionStateStore) Get(ctx context.Context, sessionID string) (*ConnectionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if state, exists := s.states[sessionID]; exists {
		return &state, nil
	}
	return nil, nil
}

func (s *InMemoryConnectionStateStore) ListByInstance(ctx context.Context, instanceID string) ([]ConnectionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var states []ConnectionState
	for _, state := range s.states {
		if state.InstanceID == instanceID {
			states = append(states, state)
		}
	}
	return states, nil
}

func (s *InMemoryConnectionStateStore) ListAll(ctx context.Context) ([]ConnectionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states := make([]ConnectionState, 0, len(s.states))
	for _, state := range s.states {
		states = append(states, state)
	}
	return states, nil
}

func (s *InMemoryConnectionStateStore) Count(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return int64(len(s.states)), nil
}

func (s *InMemoryConnectionStateStore) GetInstanceID() string {
	return s.instanceID
}
