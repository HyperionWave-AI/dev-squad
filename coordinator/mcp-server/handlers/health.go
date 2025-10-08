package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// QdrantClient interface for health checks
type QdrantClient interface {
	Ping(ctx context.Context) error
}

// HealthCheckResponse represents the health check response structure
type HealthCheckResponse struct {
	Status   string                    `json:"status"`
	Services map[string]ServiceHealth `json:"services"`
}

// ServiceHealth represents the health status of a single service
type ServiceHealth struct {
	Status     string `json:"status"`
	ResponseMs int64  `json:"responseMs"`
}

// HealthCheckHandler handles health check requests
type HealthCheckHandler struct {
	mongoClient  *mongo.Client
	qdrantClient QdrantClient
	logger       *zap.Logger
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(mongoClient *mongo.Client, qdrantClient QdrantClient, logger *zap.Logger) *HealthCheckHandler {
	return &HealthCheckHandler{
		mongoClient:  mongoClient,
		qdrantClient: qdrantClient,
		logger:       logger,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *HealthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response := HealthCheckResponse{
		Services: make(map[string]ServiceHealth),
	}

	// Check MongoDB
	mongoHealth := h.checkMongoDB()
	response.Services["mongodb"] = mongoHealth

	// Check Qdrant
	qdrantHealth := h.checkQdrant()
	response.Services["qdrant"] = qdrantHealth

	// Determine overall status
	if mongoHealth.Status == "up" && qdrantHealth.Status == "up" {
		response.Status = "healthy"
		w.WriteHeader(http.StatusOK)
	} else {
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health check response", zap.Error(err))
	}
}

// checkMongoDB checks MongoDB connectivity
func (h *HealthCheckHandler) checkMongoDB() ServiceHealth {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := h.mongoClient.Ping(ctx, nil)
	duration := time.Since(start)

	if err != nil {
		h.logger.Warn("MongoDB health check failed", zap.Error(err))
		return ServiceHealth{
			Status:     "down",
			ResponseMs: duration.Milliseconds(),
		}
	}

	return ServiceHealth{
		Status:     "up",
		ResponseMs: duration.Milliseconds(),
	}
}

// checkQdrant checks Qdrant connectivity
func (h *HealthCheckHandler) checkQdrant() ServiceHealth {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := h.qdrantClient.Ping(ctx)
	duration := time.Since(start)

	if err != nil {
		h.logger.Warn("Qdrant health check failed", zap.Error(err))
		return ServiceHealth{
			Status:     "down",
			ResponseMs: duration.Milliseconds(),
		}
	}

	return ServiceHealth{
		Status:     "up",
		ResponseMs: duration.Milliseconds(),
	}
}
