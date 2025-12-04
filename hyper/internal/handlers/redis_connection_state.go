package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PHASE 2: Redis-backed connection state store for distributed persistence

const (
	// Redis key prefixes
	connectionKeyPrefix = "conn:"          // conn:{sessionId} -> ConnectionState JSON
	instanceSetPrefix   = "conn:instance:" // conn:instance:{instanceId} -> SET of sessionIds
	allConnectionsKey   = "conn:all"       // SET of all active sessionIds

	// Connection TTL - connections expire if no heartbeat received
	connectionTTL = 2 * time.Minute
)

// RedisConnectionStateStore implements ConnectionStateStore using Redis
// PHASE 2: Provides distributed connection state visibility across instances
type RedisConnectionStateStore struct {
	client     *redis.Client
	instanceID string
	logger     *zap.Logger
}

// NewRedisConnectionStateStore creates a new Redis-backed connection state store
// PHASE 2: Used when REDIS_URL is configured
func NewRedisConnectionStateStore(client *redis.Client, instanceID string, logger *zap.Logger) *RedisConnectionStateStore {
	if client == nil {
		return nil
	}
	return &RedisConnectionStateStore{
		client:     client,
		instanceID: instanceID,
		logger:     logger,
	}
}

// Register stores a new connection state in Redis
// PHASE 2: Uses pipeline for atomic multi-key operations
func (s *RedisConnectionStateStore) Register(ctx context.Context, state ConnectionState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal connection state: %w", err)
	}

	pipe := s.client.Pipeline()

	// Store connection state with TTL
	connKey := connectionKeyPrefix + state.SessionID
	pipe.Set(ctx, connKey, data, connectionTTL)

	// Add to instance set (tracks which connections this instance owns)
	instanceKey := instanceSetPrefix + state.InstanceID
	pipe.SAdd(ctx, instanceKey, state.SessionID)
	pipe.Expire(ctx, instanceKey, connectionTTL)

	// Add to global set (tracks all active connections)
	pipe.SAdd(ctx, allConnectionsKey, state.SessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to register connection in Redis: %w", err)
	}

	s.logger.Debug("Connection state registered in Redis",
		zap.String("sessionId", state.SessionID),
		zap.String("instanceId", state.InstanceID),
		zap.String("userId", state.UserID))

	return nil
}

// Unregister removes a connection state from Redis
// PHASE 2: Cleans up all related keys atomically
func (s *RedisConnectionStateStore) Unregister(ctx context.Context, sessionID string) error {
	pipe := s.client.Pipeline()

	// Remove connection state
	connKey := connectionKeyPrefix + sessionID
	pipe.Del(ctx, connKey)

	// Remove from instance set
	instanceKey := instanceSetPrefix + s.instanceID
	pipe.SRem(ctx, instanceKey, sessionID)

	// Remove from global set
	pipe.SRem(ctx, allConnectionsKey, sessionID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unregister connection from Redis: %w", err)
	}

	s.logger.Debug("Connection state unregistered from Redis",
		zap.String("sessionId", sessionID))

	return nil
}

// Heartbeat updates the last activity time and refreshes TTL
// PHASE 2: Called periodically to keep connection state alive
func (s *RedisConnectionStateStore) Heartbeat(ctx context.Context, sessionID string) error {
	connKey := connectionKeyPrefix + sessionID

	// Get current state
	data, err := s.client.Get(ctx, connKey).Bytes()
	if err == redis.Nil {
		// Connection doesn't exist (already unregistered or expired)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get connection state: %w", err)
	}

	var state ConnectionState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal connection state: %w", err)
	}

	// Update last activity
	state.LastActivity = time.Now()
	data, err = json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal updated state: %w", err)
	}

	// Update with refreshed TTL
	pipe := s.client.Pipeline()
	pipe.Set(ctx, connKey, data, connectionTTL)

	// Also refresh instance set TTL
	instanceKey := instanceSetPrefix + state.InstanceID
	pipe.Expire(ctx, instanceKey, connectionTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return nil
}

// Get retrieves a connection state by session ID
// PHASE 2: Returns nil if connection not found or expired
func (s *RedisConnectionStateStore) Get(ctx context.Context, sessionID string) (*ConnectionState, error) {
	connKey := connectionKeyPrefix + sessionID

	data, err := s.client.Get(ctx, connKey).Bytes()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get connection state: %w", err)
	}

	var state ConnectionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connection state: %w", err)
	}

	return &state, nil
}

// ListByInstance returns all connections for a specific instance
// PHASE 2: Useful for instance-specific operations
func (s *RedisConnectionStateStore) ListByInstance(ctx context.Context, instanceID string) ([]ConnectionState, error) {
	instanceKey := instanceSetPrefix + instanceID
	sessionIDs, err := s.client.SMembers(ctx, instanceKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance connections: %w", err)
	}

	return s.getStatesByIDs(ctx, sessionIDs)
}

// ListAll returns all active connections across all instances
// PHASE 2: Provides cluster-wide connection visibility
func (s *RedisConnectionStateStore) ListAll(ctx context.Context) ([]ConnectionState, error) {
	sessionIDs, err := s.client.SMembers(ctx, allConnectionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all connections: %w", err)
	}

	return s.getStatesByIDs(ctx, sessionIDs)
}

// getStatesByIDs retrieves multiple connection states efficiently
// PHASE 2: Uses MGET for batch retrieval
func (s *RedisConnectionStateStore) getStatesByIDs(ctx context.Context, sessionIDs []string) ([]ConnectionState, error) {
	if len(sessionIDs) == 0 {
		return []ConnectionState{}, nil
	}

	// Build keys for MGET
	keys := make([]string, len(sessionIDs))
	for i, id := range sessionIDs {
		keys[i] = connectionKeyPrefix + id
	}

	// Batch get all connection states
	results, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to batch get connection states: %w", err)
	}

	states := make([]ConnectionState, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue // Connection expired or deleted
		}
		str, ok := result.(string)
		if !ok {
			continue
		}
		var state ConnectionState
		if err := json.Unmarshal([]byte(str), &state); err != nil {
			s.logger.Warn("Failed to unmarshal connection state",
				zap.Error(err))
			continue
		}
		states = append(states, state)
	}

	return states, nil
}

// Count returns the total number of active connections
// PHASE 2: O(1) operation using Redis SET cardinality
func (s *RedisConnectionStateStore) Count(ctx context.Context) (int64, error) {
	count, err := s.client.SCard(ctx, allConnectionsKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count connections: %w", err)
	}
	return count, nil
}

// GetInstanceID returns the instance ID for this store
func (s *RedisConnectionStateStore) GetInstanceID() string {
	return s.instanceID
}

// CleanupStaleConnections removes expired sessions from the global set
// PHASE 2: Called periodically to clean up orphaned set entries
// Note: The actual connection keys expire via TTL, but set membership needs cleanup
func (s *RedisConnectionStateStore) CleanupStaleConnections(ctx context.Context) (int64, error) {
	sessionIDs, err := s.client.SMembers(ctx, allConnectionsKey).Result()
	if err != nil {
		return 0, err
	}

	var removed int64
	for _, sessionID := range sessionIDs {
		connKey := connectionKeyPrefix + sessionID
		exists, err := s.client.Exists(ctx, connKey).Result()
		if err != nil {
			continue
		}
		if exists == 0 {
			// Connection key expired, remove from sets
			s.client.SRem(ctx, allConnectionsKey, sessionID)
			removed++
		}
	}

	if removed > 0 {
		s.logger.Info("Cleaned up stale connection entries",
			zap.Int64("removed", removed))
	}

	return removed, nil
}
