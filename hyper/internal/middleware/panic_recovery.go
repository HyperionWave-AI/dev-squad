package middleware

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PanicRecoveryMiddleware recovers from panics in HTTP handlers
// Logs the panic with stack trace and returns a 500 error to the client
func PanicRecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Capture stack trace
				stack := debug.Stack()

				// Log panic with full details
				logger.Error("🚨 PANIC RECOVERED in HTTP handler",
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("clientIP", c.ClientIP()),
					zap.Any("error", err),
					zap.String("stackTrace", string(stack)),
					zap.Time("timestamp", time.Now()))

				// Return error response to client
				requestID := c.GetString("requestId")
				if requestID == "" {
					requestID = fmt.Sprintf("panic-%d", time.Now().Unix())
				}

				c.AbortWithStatusJSON(500, gin.H{
					"error":     "Internal server error - an unexpected error occurred",
					"requestId": requestID,
					"timestamp": time.Now().Format(time.RFC3339),
				})
			}
		}()
		c.Next()
	}
}

// SafeGo wraps a goroutine with panic recovery
// Use this wrapper for all background goroutines to prevent crashes
//
// Example:
//   SafeGo(logger, func() {
//       // Your goroutine code here
//   })
func SafeGo(logger *zap.Logger, fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				// Capture stack trace
				stack := debug.Stack()

				// Log panic with full details
				logger.Error("🚨 PANIC RECOVERED in goroutine",
					zap.Any("error", err),
					zap.String("stackTrace", string(stack)),
					zap.Time("timestamp", time.Now()))
			}
		}()
		fn()
	}()
}

// SafeGoWithCleanup wraps a goroutine with panic recovery and cleanup guarantees
// The cleanup function will ALWAYS run, even if the goroutine panics
//
// Example:
//   SafeGoWithCleanup(logger, func() {
//       // Your goroutine code here
//   }, func() {
//       // Cleanup code (close connections, channels, etc.)
//   })
func SafeGoWithCleanup(logger *zap.Logger, fn func(), cleanup func()) {
	go func() {
		defer func() {
			// Always run cleanup first
			if cleanup != nil {
				defer func() {
					if cleanupErr := recover(); cleanupErr != nil {
						logger.Error("🚨 PANIC in cleanup function",
							zap.Any("error", cleanupErr),
							zap.String("stackTrace", string(debug.Stack())))
					}
				}()
				cleanup()
			}

			// Then check for panic in main function
			if err := recover(); err != nil {
				stack := debug.Stack()
				logger.Error("🚨 PANIC RECOVERED in goroutine (with cleanup)",
					zap.Any("error", err),
					zap.String("stackTrace", string(stack)),
					zap.Time("timestamp", time.Now()))
			}
		}()
		fn()
	}()
}
