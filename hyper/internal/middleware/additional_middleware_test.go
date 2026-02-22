package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestPanicRecoveryMiddleware_Recovers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PanicRecoveryMiddleware(zap.NewNop()))
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSafeGoAndSafeGoWithCleanup(t *testing.T) {
	done := make(chan struct{})
	SafeGo(zap.NewNop(), func() {
		close(done)
		panic("goroutine panic")
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo goroutine did not complete in time")
	}

	cleanupDone := make(chan struct{})
	SafeGoWithCleanup(zap.NewNop(), func() {
		panic("main panic")
	}, func() {
		close(cleanupDone)
	})

	select {
	case <-cleanupDone:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGoWithCleanup cleanup did not run")
	}

	cleanupPanicDone := make(chan struct{})
	SafeGoWithCleanup(zap.NewNop(), func() {
		// no panic
	}, func() {
		close(cleanupPanicDone)
		panic("cleanup panic")
	})

	select {
	case <-cleanupPanicDone:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGoWithCleanup cleanup panic path did not run")
	}
}

func TestNewDistributedRateLimiter_FallsBackToInMemory(t *testing.T) {
	t.Setenv("REDIS_URL", "")

	limiter := NewDistributedRateLimiter(5, time.Second, zap.NewNop())
	if limiter == nil {
		t.Fatal("expected a rate limiter instance")
	}
	if _, ok := limiter.(*RateLimiter); !ok {
		t.Fatalf("expected in-memory RateLimiter fallback, got %T", limiter)
	}
}

func TestRedisRateLimiter_MiddlewareBranchesAndStop(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := &RedisRateLimiter{
		client:     redisv9.NewClient(&redisv9.Options{Addr: "127.0.0.1:0"}),
		maxTokens:  1,
		refillRate: time.Second,
		keyPrefix:  "ratelimit:",
		logger:     zap.NewNop(),
		scriptSHA:  "missing-script",
	}
	defer rl.client.Close()

	// Missing userId branch
	r1 := gin.New()
	r1.Use(rl.Middleware())
	r1.GET("/test", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r1.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for missing userId, got %d", w.Code)
	}

	// Invalid userId type branch
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("userId", 123); c.Next() })
	r2.Use(rl.Middleware())
	r2.GET("/test", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	r2.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for invalid userId type, got %d", w.Code)
	}

	// Redis error fail-open branch (request should pass)
	r3 := gin.New()
	r3.Use(func(c *gin.Context) { c.Set("userId", "user-1"); c.Next() })
	r3.Use(rl.Middleware())
	r3.GET("/test", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	r3.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected fail-open 200 on redis error, got %d", w.Code)
	}

	rl.Stop()
}

func TestNewRedisRateLimiter_NilClient(t *testing.T) {
	if _, err := NewRedisRateLimiter(nil, 5, time.Second, zap.NewNop()); err == nil {
		t.Fatal("expected error for nil redis client")
	}
}
