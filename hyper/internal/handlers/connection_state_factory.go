package handlers

import (
	"go.uber.org/zap"

	redisclient "hyper/internal/redis"
)

// PHASE 2: Factory function for connection state store selection

// Global connection state store singleton
var (
	connectionStateStore ConnectionStateStore
)

// NewConnectionStateStore creates the appropriate connection state store
// PHASE 2: Selects Redis or in-memory based on configuration
// - If REDIS_URL is set and Redis is reachable: uses Redis (distributed)
// - Otherwise: falls back to in-memory (single instance only)
func NewConnectionStateStore(logger *zap.Logger) ConnectionStateStore {
	instanceID := GetInstanceID()

	// Try Redis first
	redisClient := redisclient.GetClient(logger)
	if redisClient != nil {
		store := NewRedisConnectionStateStore(redisClient, instanceID, logger)
		if store != nil {
			logger.Info("Using Redis-backed connection state store (distributed)",
				zap.String("instanceId", instanceID))
			return store
		}
	}

	// Fall back to in-memory
	logger.Info("Using in-memory connection state store (not distributed)",
		zap.String("instanceId", instanceID))
	return NewInMemoryConnectionStateStore(instanceID, logger)
}

// GetConnectionStateStore returns the singleton connection state store
// PHASE 2: Lazy initialization on first call
func GetConnectionStateStore(logger *zap.Logger) ConnectionStateStore {
	if connectionStateStore == nil {
		connectionStateStore = NewConnectionStateStore(logger)
	}
	return connectionStateStore
}
