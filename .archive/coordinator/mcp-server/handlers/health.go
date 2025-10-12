package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	Status     string   `json:"status"`
	ResponseMs int64    `json:"responseMs"`
	Service    string   `json:"service,omitempty"`
	URL        string   `json:"url,omitempty"`
	Models     []string `json:"models,omitempty"`
	Message    string   `json:"message,omitempty"`
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

	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://qdrant:6333"
	}

	if err != nil {
		h.logger.Warn("Qdrant health check failed", zap.Error(err))
		return ServiceHealth{
			Status:     "down",
			ResponseMs: duration.Milliseconds(),
			Service:    "qdrant",
			URL:        qdrantURL,
			Message:    fmt.Sprintf("Connection failed: %v", err),
		}
	}

	return ServiceHealth{
		Status:     "up",
		ResponseMs: duration.Milliseconds(),
		Service:    "qdrant",
		URL:        qdrantURL,
	}
}

// checkOllama checks Ollama embedding service connectivity
func (h *HealthCheckHandler) checkOllama() ServiceHealth {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	start := time.Now()

	// Check Ollama by calling /api/tags endpoint to list models
	req, err := http.NewRequestWithContext(ctx, "GET", ollamaURL+"/api/tags", nil)
	if err != nil {
		return ServiceHealth{
			Status:     "down",
			ResponseMs: time.Since(start).Milliseconds(),
			Service:    "ollama",
			URL:        ollamaURL,
			Message:    fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.logger.Warn("Ollama health check failed", zap.Error(err))
		return ServiceHealth{
			Status:     "down",
			ResponseMs: time.Since(start).Milliseconds(),
			Service:    "ollama",
			URL:        ollamaURL,
			Message:    fmt.Sprintf("Connection failed: %v", err),
		}
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ServiceHealth{
			Status:     "down",
			ResponseMs: duration.Milliseconds(),
			Service:    "ollama",
			URL:        ollamaURL,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}

	// Parse response to get model list
	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return ServiceHealth{
			Status:     "up",
			ResponseMs: duration.Milliseconds(),
			Service:    "ollama",
			URL:        ollamaURL,
			Message:    "Connected but failed to parse models",
		}
	}

	// Extract model names
	models := make([]string, len(tagsResp.Models))
	for i, model := range tagsResp.Models {
		models[i] = model.Name
	}

	return ServiceHealth{
		Status:     "up",
		ResponseMs: duration.Milliseconds(),
		Service:    "ollama",
		URL:        ollamaURL,
		Models:     models,
	}
}

// ServeQdrantHealth handles GET /health/qdrant
func (h *HealthCheckHandler) ServeQdrantHealth(w http.ResponseWriter, r *http.Request) {
	health := h.checkQdrant()
	w.Header().Set("Content-Type", "application/json")

	if health.Status == "up" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.Error("Failed to encode Qdrant health response", zap.Error(err))
	}
}

// ServeOllamaHealth handles GET /health/ollama
func (h *HealthCheckHandler) ServeOllamaHealth(w http.ResponseWriter, r *http.Request) {
	health := h.checkOllama()
	w.Header().Set("Content-Type", "application/json")

	if health.Status == "up" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.Error("Failed to encode Ollama health response", zap.Error(err))
	}
}
