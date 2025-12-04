package redis

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PHASE 1: Redis client infrastructure for distributed rate limiting

var (
	client     *redis.Client
	clientOnce sync.Once
	initLogger *zap.Logger
)

// GetClient returns the singleton Redis client
// Returns nil if REDIS_URL is not set or connection fails
// PHASE 1: Singleton pattern with lazy initialization
func GetClient(logger *zap.Logger) *redis.Client {
	clientOnce.Do(func() {
		initLogger = logger
		redisURL := os.Getenv("REDIS_URL")
		if redisURL == "" {
			logger.Info("REDIS_URL not set, Redis features disabled")
			return
		}

		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			logger.Warn("Failed to parse REDIS_URL, Redis features disabled",
				zap.String("url", redisURL),
				zap.Error(err))
			return
		}

		// Configure connection pool
		opt.PoolSize = 10
		opt.MinIdleConns = 2
		opt.DialTimeout = 5 * time.Second
		opt.ReadTimeout = 3 * time.Second
		opt.WriteTimeout = 3 * time.Second

		client = redis.NewClient(opt)

		// Test connection with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			logger.Warn("Redis connection failed, Redis features disabled",
				zap.String("addr", opt.Addr),
				zap.Error(err))
			client.Close()
			client = nil
			return
		}

		logger.Info("Redis client initialized successfully",
			zap.String("addr", opt.Addr),
			zap.Int("poolSize", opt.PoolSize))
	})

	return client
}

// IsAvailable checks if Redis client is available
// PHASE 1: Health check for graceful degradation
func IsAvailable() bool {
	return client != nil
}

// Ping checks Redis connectivity
// PHASE 1: Active health check
func Ping(ctx context.Context) error {
	if client == nil {
		return nil // Not an error if Redis is not configured
	}
	return client.Ping(ctx).Err()
}

// Close closes the Redis client connection
// PHASE 1: Graceful shutdown
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}
