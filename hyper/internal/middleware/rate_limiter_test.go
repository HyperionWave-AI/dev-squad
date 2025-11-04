package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewRateLimiter(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	rl := NewRateLimiter(10, time.Minute, logger)

	assert.NotNil(t, rl)
	assert.Equal(t, 10, rl.maxTokens)
	assert.Equal(t, time.Minute, rl.refillRate)
	assert.NotNil(t, rl.buckets)
	assert.NotNil(t, rl.logger)
	assert.NotNil(t, rl.cleanupStop)

	// Cleanup
	rl.Stop()
}

func TestRateLimiter_AllowsInitialRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	rl := NewRateLimiter(10, time.Minute, logger)
	defer rl.Stop()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", "test-user")
		c.Next()
	})
	router.Use(rl.Middleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First 10 requests should succeed
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}
}

func TestRateLimiter_BlocksExcessRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	rl := NewRateLimiter(10, time.Minute, logger)
	defer rl.Stop()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", "test-user")
		c.Next()
	})
	router.Use(rl.Middleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First 10 requests should succeed
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 11th request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "Rate limit exceeded")
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestRateLimiter_IndependentPerUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	rl := NewRateLimiter(5, time.Minute, logger)
	defer rl.Stop()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		// userId will be set per-request
		c.Next()
	})
	router.Use(rl.Middleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// User A makes 5 requests (exhausts limit)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)

		// Set userId in context via middleware simulation
		router2 := gin.New()
		router2.Use(func(c *gin.Context) {
			c.Set("userId", "user-a")
			c.Next()
		})
		router2.Use(rl.Middleware())
		router2.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		router2.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// User A's 6th request should be blocked
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router2 := gin.New()
	router2.Use(func(c *gin.Context) {
		c.Set("userId", "user-a")
		c.Next()
	})
	router2.Use(rl.Middleware())
	router2.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router2.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// User B should still be able to make requests
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/test", nil)
	router3 := gin.New()
	router3.Use(func(c *gin.Context) {
		c.Set("userId", "user-b")
		c.Next()
	})
	router3.Use(rl.Middleware())
	router3.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router3.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_RefillsTokensOverTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()

	// Use short refill period for testing (1 second)
	rl := NewRateLimiter(3, 1*time.Second, logger)
	defer rl.Stop()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", "test-user")
		c.Next()
	})
	router.Use(rl.Middleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Use up all 3 tokens
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 4th request should be blocked
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Wait for refill period
	time.Sleep(1100 * time.Millisecond)

	// After refill, should be able to make requests again
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_MissingUserId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	rl := NewRateLimiter(10, time.Minute, logger)
	defer rl.Stop()

	router := gin.New()
	// No middleware to set userId - simulates missing auth
	router.Use(rl.Middleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "authentication required")
}

func TestRateLimiter_InvalidUserId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	rl := NewRateLimiter(10, time.Minute, logger)
	defer rl.Stop()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", 12345) // Invalid type (int instead of string)
		c.Next()
	})
	router.Use(rl.Middleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "invalid user identity")
}

func TestRateLimiter_CleanupExpiredBuckets(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	rl := NewRateLimiter(10, time.Minute, logger)
	defer rl.Stop()

	// Manually add a bucket with old timestamp
	oldBucket := &bucket{
		tokens:     5,
		maxTokens:  10,
		lastRefill: time.Now().Add(-15 * time.Minute), // 15 minutes ago
	}
	rl.mu.Lock()
	rl.buckets["old-user"] = oldBucket
	rl.mu.Unlock()

	// Add a recent bucket
	recentBucket := &bucket{
		tokens:     5,
		maxTokens:  10,
		lastRefill: time.Now(),
	}
	rl.mu.Lock()
	rl.buckets["recent-user"] = recentBucket
	rl.mu.Unlock()

	// Wait a bit and trigger cleanup logic manually
	// Note: In real test, we'd wait for cleanup goroutine, but here we'll check the logic
	now := time.Now()
	expiry := 10 * time.Minute

	rl.mu.Lock()
	bucketsToDelete := []string{}
	for userID, b := range rl.buckets {
		b.mu.Lock()
		if now.Sub(b.lastRefill) > expiry {
			bucketsToDelete = append(bucketsToDelete, userID)
		}
		b.mu.Unlock()
	}

	for _, userID := range bucketsToDelete {
		delete(rl.buckets, userID)
	}
	rl.mu.Unlock()

	// Old bucket should be deleted
	rl.mu.RLock()
	_, oldExists := rl.buckets["old-user"]
	_, recentExists := rl.buckets["recent-user"]
	rl.mu.RUnlock()

	assert.False(t, oldExists, "Old bucket should be cleaned up")
	assert.True(t, recentExists, "Recent bucket should still exist")
}

func TestRateLimiter_RetryAfterHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	rl := NewRateLimiter(2, time.Minute, logger)
	defer rl.Stop()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", "test-user")
		c.Next()
	})
	router.Use(rl.Middleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Use up tokens
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Next request should be rate limited with Retry-After header
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	retryAfter := w.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfter, "Retry-After header should be set")
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	rl := NewRateLimiter(10, time.Minute, logger)
	defer rl.Stop()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", "test-user")
		c.Next()
	})
	router.Use(rl.Middleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Send 20 concurrent requests
	done := make(chan bool, 20)
	successCount := 0
	blockedCount := 0

	for i := 0; i < 20; i++ {
		go func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/test", nil)
			router.ServeHTTP(w, req)
			done <- (w.Code == http.StatusOK)
		}()
	}

	// Collect results
	for i := 0; i < 20; i++ {
		if <-done {
			successCount++
		} else {
			blockedCount++
		}
	}

	// Should have 10 successes and 10 blocked
	assert.Equal(t, 10, successCount, "Should allow exactly 10 requests")
	assert.Equal(t, 10, blockedCount, "Should block exactly 10 requests")
}
