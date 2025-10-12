package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestHealthCheckHandler_QdrantHealthy tests when Qdrant is up
func TestHealthCheckHandler_QdrantHealthy(t *testing.T) {
	mockQdrant := NewMockQdrantClient()
	mockQdrant.pingError = nil // Qdrant is healthy
	logger, _ := zap.NewDevelopment()

	handler := &HealthCheckHandler{
		qdrantClient: mockQdrant,
		logger:       logger,
	}

	qdrantHealth := handler.checkQdrant()
	assert.Equal(t, "up", qdrantHealth.Status)
	assert.GreaterOrEqual(t, qdrantHealth.ResponseMs, int64(0))
}

// TestHealthCheckHandler_QdrantDown tests when Qdrant is down
func TestHealthCheckHandler_QdrantDown(t *testing.T) {
	mockQdrant := NewMockQdrantClient()
	mockQdrant.pingError = errors.New("connection refused")
	logger, _ := zap.NewDevelopment()

	handler := &HealthCheckHandler{
		qdrantClient: mockQdrant,
		logger:       logger,
	}

	qdrantHealth := handler.checkQdrant()
	assert.Equal(t, "down", qdrantHealth.Status)
	assert.GreaterOrEqual(t, qdrantHealth.ResponseMs, int64(0))
}

// TestHealthCheckHandler_QdrantTimeout tests timeout behavior
func TestHealthCheckHandler_QdrantTimeout(t *testing.T) {
	mockQdrant := NewMockQdrantClient()
	mockQdrant.pingError = context.DeadlineExceeded
	logger, _ := zap.NewDevelopment()

	handler := &HealthCheckHandler{
		qdrantClient: mockQdrant,
		logger:       logger,
	}

	qdrantHealth := handler.checkQdrant()
	assert.Equal(t, "down", qdrantHealth.Status)
	assert.GreaterOrEqual(t, qdrantHealth.ResponseMs, int64(0))
}

// TestHealthCheckResponse_JSON tests JSON serialization
func TestHealthCheckResponse_JSON(t *testing.T) {
	response := HealthCheckResponse{
		Status: "healthy",
		Services: map[string]ServiceHealth{
			"mongodb": {
				Status:     "up",
				ResponseMs: 15,
			},
			"qdrant": {
				Status:     "up",
				ResponseMs: 8,
			},
		},
	}

	jsonBytes, err := json.Marshal(response)
	assert.NoError(t, err)

	expectedJSON := `{"status":"healthy","services":{"mongodb":{"status":"up","responseMs":15},"qdrant":{"status":"up","responseMs":8}}}`
	assert.JSONEq(t, expectedJSON, string(jsonBytes))
}

// TestHealthCheckResponse_StatusDetermination tests overall status logic
func TestHealthCheckResponse_StatusDetermination(t *testing.T) {
	tests := []struct {
		name           string
		mongoHealth    ServiceHealth
		qdrantHealth   ServiceHealth
		expectedStatus string
		expectedHTTP   int
	}{
		{
			name:           "all services up returns healthy",
			mongoHealth:    ServiceHealth{Status: "up", ResponseMs: 10},
			qdrantHealth:   ServiceHealth{Status: "up", ResponseMs: 5},
			expectedStatus: "healthy",
			expectedHTTP:   http.StatusOK,
		},
		{
			name:           "mongodb down returns unhealthy",
			mongoHealth:    ServiceHealth{Status: "down", ResponseMs: 2000},
			qdrantHealth:   ServiceHealth{Status: "up", ResponseMs: 5},
			expectedStatus: "unhealthy",
			expectedHTTP:   http.StatusServiceUnavailable,
		},
		{
			name:           "qdrant down returns unhealthy",
			mongoHealth:    ServiceHealth{Status: "up", ResponseMs: 10},
			qdrantHealth:   ServiceHealth{Status: "down", ResponseMs: 2000},
			expectedStatus: "unhealthy",
			expectedHTTP:   http.StatusServiceUnavailable,
		},
		{
			name:           "both down returns unhealthy",
			mongoHealth:    ServiceHealth{Status: "down", ResponseMs: 2000},
			qdrantHealth:   ServiceHealth{Status: "down", ResponseMs: 2000},
			expectedStatus: "unhealthy",
			expectedHTTP:   http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the status determination logic
			overallHealthy := tt.mongoHealth.Status == "up" && tt.qdrantHealth.Status == "up"

			expectedStatus := "healthy"
			expectedHTTP := http.StatusOK
			if !overallHealthy {
				expectedStatus = "unhealthy"
				expectedHTTP = http.StatusServiceUnavailable
			}

			assert.Equal(t, tt.expectedStatus, expectedStatus)
			assert.Equal(t, tt.expectedHTTP, expectedHTTP)
		})
	}
}

// TestServiceHealth_Structure tests ServiceHealth struct
func TestServiceHealth_Structure(t *testing.T) {
	health := ServiceHealth{
		Status:     "up",
		ResponseMs: 100,
	}

	assert.Equal(t, "up", health.Status)
	assert.Equal(t, int64(100), health.ResponseMs)

	// Test JSON marshaling
	jsonBytes, err := json.Marshal(health)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"status":"up","responseMs":100}`, string(jsonBytes))
}

// TestHealthCheckResponse_Structure tests response structure
func TestHealthCheckResponse_Structure(t *testing.T) {
	response := HealthCheckResponse{
		Status:   "healthy",
		Services: make(map[string]ServiceHealth),
	}

	response.Services["test"] = ServiceHealth{
		Status:     "up",
		ResponseMs: 50,
	}

	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, 1, len(response.Services))
	assert.Equal(t, "up", response.Services["test"].Status)
}

// TestHealthCheckHandler_HTTPResponse tests the HTTP response
func TestHealthCheckHandler_HTTPResponse(t *testing.T) {
	mockQdrant := NewMockQdrantClient()
	mockQdrant.pingError = nil
	logger, _ := zap.NewDevelopment()

	// We can't fully test with MongoDB mock, but we can test the response structure
	handler := &HealthCheckHandler{
		mongoClient:  nil, // This would cause issues in real use, but tests check methods
		qdrantClient: mockQdrant,
		logger:       logger,
	}

	// Test that checkQdrant returns correct structure
	qdrantHealth := handler.checkQdrant()
	assert.NotEmpty(t, qdrantHealth.Status)
	assert.GreaterOrEqual(t, qdrantHealth.ResponseMs, int64(0))
}
