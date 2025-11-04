package handlers

import (
	"fmt"
	"net/http"
	"time"

	"hyper/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PanicTestHandler provides test endpoints to verify panic recovery
type PanicTestHandler struct {
	logger *zap.Logger
}

// NewPanicTestHandler creates a new panic test handler
func NewPanicTestHandler(logger *zap.Logger) *PanicTestHandler {
	return &PanicTestHandler{
		logger: logger,
	}
}

// RegisterTestRoutes registers panic test endpoints (for development/testing only)
func (h *PanicTestHandler) RegisterTestRoutes(r *gin.Engine) {
	testGroup := r.Group("/test/panic")
	{
		testGroup.GET("/http-handler", h.TestHTTPHandlerPanic)
		testGroup.GET("/goroutine", h.TestGoroutinePanic)
		testGroup.GET("/goroutine-with-cleanup", h.TestGoroutineWithCleanup)
		testGroup.GET("/nil-pointer", h.TestNilPointerPanic)
		testGroup.GET("/array-bounds", h.TestArrayBoundsPanic)
		testGroup.GET("/status", h.TestStatus)
	}

	h.logger.Info("🧪 Panic test endpoints registered",
		zap.String("basePath", "/test/panic"),
		zap.Strings("endpoints", []string{
			"/test/panic/http-handler",
			"/test/panic/goroutine",
			"/test/panic/goroutine-with-cleanup",
			"/test/panic/nil-pointer",
			"/test/panic/array-bounds",
			"/test/panic/status",
		}))
}

// TestHTTPHandlerPanic tests panic recovery in HTTP handler
func (h *PanicTestHandler) TestHTTPHandlerPanic(c *gin.Context) {
	h.logger.Info("🧪 TEST START: HTTP handler panic test",
		zap.String("path", c.Request.URL.Path),
		zap.String("clientIP", c.ClientIP()))

	// This will panic and should be caught by PanicRecoveryMiddleware
	panic("TEST PANIC: Deliberate HTTP handler panic for testing")
}

// TestGoroutinePanic tests panic recovery in goroutine
func (h *PanicTestHandler) TestGoroutinePanic(c *gin.Context) {
	h.logger.Info("🧪 TEST START: Goroutine panic test",
		zap.String("path", c.Request.URL.Path))

	// Launch goroutine with SafeGo wrapper
	middleware.SafeGo(h.logger, func() {
		h.logger.Info("🧪 Inside test goroutine, about to panic...")
		time.Sleep(100 * time.Millisecond) // Simulate some work
		panic("TEST PANIC: Deliberate goroutine panic for testing")
	})

	// Respond immediately (goroutine will panic in background)
	c.JSON(http.StatusOK, gin.H{
		"message": "Goroutine launched - check logs in ~100ms for panic recovery",
		"test":    "goroutine-panic",
	})
}

// TestGoroutineWithCleanup tests panic recovery with cleanup guarantees
func (h *PanicTestHandler) TestGoroutineWithCleanup(c *gin.Context) {
	h.logger.Info("🧪 TEST START: Goroutine with cleanup test",
		zap.String("path", c.Request.URL.Path))

	// Launch goroutine with SafeGoWithCleanup
	middleware.SafeGoWithCleanup(h.logger,
		// Main function (will panic)
		func() {
			h.logger.Info("🧪 Inside test goroutine with cleanup, about to panic...")
			time.Sleep(100 * time.Millisecond)
			panic("TEST PANIC: Deliberate goroutine panic with cleanup test")
		},
		// Cleanup function (MUST run even after panic)
		func() {
			h.logger.Info("✅ CLEANUP CALLED: This proves cleanup runs even after panic")
		},
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Goroutine with cleanup launched - check logs for cleanup execution",
		"test":    "goroutine-cleanup",
	})
}

// TestNilPointerPanic tests panic recovery for nil pointer dereference
func (h *PanicTestHandler) TestNilPointerPanic(c *gin.Context) {
	h.logger.Info("🧪 TEST START: Nil pointer panic test",
		zap.String("path", c.Request.URL.Path))

	var nilPointer *string
	// This will cause nil pointer dereference panic
	_ = *nilPointer

	// Should never reach here
	c.JSON(http.StatusOK, gin.H{"message": "This should never be returned"})
}

// TestArrayBoundsPanic tests panic recovery for array out of bounds
func (h *PanicTestHandler) TestArrayBoundsPanic(c *gin.Context) {
	h.logger.Info("🧪 TEST START: Array bounds panic test",
		zap.String("path", c.Request.URL.Path))

	arr := []int{1, 2, 3}
	// This will cause index out of bounds panic
	_ = arr[100]

	// Should never reach here
	c.JSON(http.StatusOK, gin.H{"message": "This should never be returned"})
}

// TestStatus returns server status (to verify server is still running after panics)
func (h *PanicTestHandler) TestStatus(c *gin.Context) {
	h.logger.Info("🧪 TEST: Status check endpoint called",
		zap.String("path", c.Request.URL.Path))

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"message":   "Server is running normally",
		"timestamp": time.Now().Format(time.RFC3339),
		"test":      "status-check",
	})
}

// TestPanicRecoverySummary provides a summary of all tests
func (h *PanicTestHandler) GetTestInstructions() string {
	return `
🧪 PANIC RECOVERY TEST SUITE
===============================

Test these endpoints to verify panic recovery:

1. HTTP Handler Panic:
   curl http://localhost:5555/test/panic/http-handler
   Expected: 500 error, server stays running, panic logged

2. Goroutine Panic:
   curl http://localhost:5555/test/panic/goroutine
   Expected: 200 response, panic logged after ~100ms, server stays running

3. Goroutine with Cleanup:
   curl http://localhost:5555/test/panic/goroutine-with-cleanup
   Expected: 200 response, cleanup log appears, panic logged, server stays running

4. Nil Pointer Panic:
   curl http://localhost:5555/test/panic/nil-pointer
   Expected: 500 error, server stays running, panic logged

5. Array Bounds Panic:
   curl http://localhost:5555/test/panic/array-bounds
   Expected: 500 error, server stays running, panic logged

6. Status Check (after panics):
   curl http://localhost:5555/test/panic/status
   Expected: 200 response with "healthy" status

SUCCESS CRITERIA:
✅ All panic endpoints return 500 (except goroutine tests which return 200)
✅ Server stays running and responds to /test/panic/status
✅ All panics are logged with stack traces
✅ Cleanup function runs in goroutine-with-cleanup test
✅ No process crash or restart

Look for these log messages:
- "🚨 PANIC RECOVERED in HTTP handler" (for HTTP panics)
- "🚨 PANIC RECOVERED in goroutine" (for goroutine panics)
- "✅ CLEANUP CALLED" (for cleanup test)
`
}

// LogTestInstructions logs the test instructions
func (h *PanicTestHandler) LogTestInstructions() {
	fmt.Println(h.GetTestInstructions())
}
