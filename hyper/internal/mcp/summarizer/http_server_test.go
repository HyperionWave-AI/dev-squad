package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ============================================================================
// Mock Summarizer for Testing
// ============================================================================

type MockSummarizer struct {
	shouldFail bool
	delay      time.Duration
}

func (m *MockSummarizer) Summarize(ctx context.Context, code string, metadata CodeMetadata) (*CodeSummary, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.shouldFail {
		return nil, fmt.Errorf("mock summarization error")
	}

	return &CodeSummary{
		Text:        "This is a mock summary of the code.",
		Type:        "llm",
		TokenCount:  10,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}, nil
}

func (m *MockSummarizer) Close() error {
	return nil
}

// ============================================================================
// Test Helpers
// ============================================================================

func setupTestRouter(summarizer CodeSummarizer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	handlers := NewSummarizerHandlers(summarizer, logger)

	engine := gin.New()
	engine.Use(handlers.ErrorHandlingMiddleware())
	engine.Use(handlers.RequestIDMiddleware())
	engine.Use(handlers.LoggingMiddleware())

	registerRoutes(engine, handlers)

	return engine
}

// ============================================================================
// Tests for POST /api/summarize
// ============================================================================

func TestHandleSummarize_Success(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	req := SummarizeRequest{
		Code: "func hello() { println(\"Hello\") }",
		Metadata: CodeMetadata{
			FilePath: "main.go",
			Language: "go",
		},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp SummarizeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Summary)
	assert.Equal(t, "llm", resp.Summary.Type)
	assert.NotEmpty(t, resp.RequestID)
}

func TestHandleSummarize_MissingCode(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	req := SummarizeRequest{
		Metadata: CodeMetadata{
			FilePath: "main.go",
		},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.Error)
}

func TestHandleSummarize_CodeTooLarge(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	// Create code larger than 100KB
	largeCode := make([]byte, 101000)
	for i := range largeCode {
		largeCode[i] = 'a'
	}

	req := SummarizeRequest{
		Code: string(largeCode),
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSummarize_SummarizationError(t *testing.T) {
	summarizer := &MockSummarizer{shouldFail: true}
	router := setupTestRouter(summarizer)

	req := SummarizeRequest{
		Code: "func hello() {}",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp SummarizeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.NotEmpty(t, resp.Error)
}

// ============================================================================
// Tests for POST /api/summarize/batch
// ============================================================================

func TestHandleBatchSummarize_Success(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	req := BatchSummarizeRequest{
		Items: []BatchItem{
			{
				ID:   "item1",
				Code: "func hello() {}",
			},
			{
				ID:   "item2",
				Code: "func world() {}",
			},
		},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize/batch", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp BatchSummarizeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Equal(t, 2, len(resp.Results))
	assert.Equal(t, 2, resp.Statistics.SuccessfulItems)
	assert.Equal(t, 0, resp.Statistics.FailedItems)
}

func TestHandleBatchSummarize_PartialFailure(t *testing.T) {
	summarizer := &MockSummarizer{shouldFail: true}
	router := setupTestRouter(summarizer)

	req := BatchSummarizeRequest{
		Items: []BatchItem{
			{
				ID:   "item1",
				Code: "func hello() {}",
			},
		},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize/batch", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp BatchSummarizeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.Equal(t, 1, resp.Statistics.FailedItems)
}

func TestHandleBatchSummarize_EmptyItems(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	req := BatchSummarizeRequest{
		Items: []BatchItem{},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize/batch", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleBatchSummarize_DuplicateIDs(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	req := BatchSummarizeRequest{
		Items: []BatchItem{
			{
				ID:   "item1",
				Code: "func hello() {}",
			},
			{
				ID:   "item1", // Duplicate ID
				Code: "func world() {}",
			},
		},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize/batch", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleBatchSummarize_TooManyItems(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	items := make([]BatchItem, 101)
	for i := 0; i < 101; i++ {
		items[i] = BatchItem{
			ID:   fmt.Sprintf("item%d", i),
			Code: "func hello() {}",
		}
	}

	req := BatchSummarizeRequest{
		Items: items,
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize/batch", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Tests for GET /api/metrics
// ============================================================================

func TestHandleMetrics_Success(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	httpReq := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp MetricsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
}

// ============================================================================
// Tests for GET /api/health
// ============================================================================

func TestHandleHealth_Success(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	httpReq := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "healthy", resp.Status)
	assert.GreaterOrEqual(t, resp.Uptime, int64(0))
}

// ============================================================================
// Tests for GET /api/ready
// ============================================================================

func TestHandleReadiness_Ready(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	httpReq := httptest.NewRequest("GET", "/api/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp["ready"].(bool))
}

// ============================================================================
// Tests for GET /api/live
// ============================================================================

func TestHandleLiveness_Alive(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	httpReq := httptest.NewRequest("GET", "/api/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp["alive"].(bool))
}

// ============================================================================
// Tests for Request ID Tracking
// ============================================================================

func TestRequestIDTracking(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	req := SummarizeRequest{
		Code: "func hello() {}",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/summarize", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Request-ID", "test-request-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, "test-request-123", w.Header().Get("X-Request-ID"))

	var resp SummarizeResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "test-request-123", resp.RequestID)
}

// ============================================================================
// Tests for 404 Handling
// ============================================================================

func TestNotFound(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	httpReq := httptest.NewRequest("GET", "/api/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "NOT_FOUND", resp.Code)
}

// ============================================================================
// Tests for Root Endpoint
// ============================================================================

func TestRootEndpoint(t *testing.T) {
	summarizer := &MockSummarizer{}
	router := setupTestRouter(summarizer)

	httpReq := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "code-summarizer", resp["service"])
	assert.NotNil(t, resp["endpoints"])
}
