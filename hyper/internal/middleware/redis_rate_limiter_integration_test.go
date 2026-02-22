package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	redisv9 "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func TestRedisRateLimiter_AllowAndRefill(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redisv9.NewClient(&redisv9.Options{Addr: mr.Addr()})
	defer client.Close()

	rl, err := NewRedisRateLimiter(client, 2, time.Second, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRedisRateLimiter returned error: %v", err)
	}

	ctx := context.Background()

	allowed, remaining, retryAfter, err := rl.allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("allow(1) returned error: %v", err)
	}
	if !allowed || remaining != 1 || retryAfter != 0 {
		t.Fatalf("unexpected first allow result: allowed=%v remaining=%d retryAfter=%d", allowed, remaining, retryAfter)
	}

	allowed, remaining, retryAfter, err = rl.allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("allow(2) returned error: %v", err)
	}
	if !allowed || remaining != 0 || retryAfter != 0 {
		t.Fatalf("unexpected second allow result: allowed=%v remaining=%d retryAfter=%d", allowed, remaining, retryAfter)
	}

	allowed, remaining, retryAfter, err = rl.allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("allow(3) returned error: %v", err)
	}
	if allowed || remaining != 0 || retryAfter < 1 {
		t.Fatalf("expected blocked third request, got allowed=%v remaining=%d retryAfter=%d", allowed, remaining, retryAfter)
	}

	time.Sleep(1100 * time.Millisecond)
	allowed, _, _, err = rl.allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("allow(after refill) returned error: %v", err)
	}
	if !allowed {
		t.Fatal("expected request to be allowed after refill")
	}
}

func TestRedisRateLimiter_MiddlewareAllowAndDeny(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redisv9.NewClient(&redisv9.Options{Addr: mr.Addr()})
	defer client.Close()

	rl, err := NewRedisRateLimiter(client, 2, time.Minute, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRedisRateLimiter returned error: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", "user-2")
		c.Next()
	})
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d expected 200, got %d", i+1, w.Code)
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on third request, got %d", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected Retry-After header for rate-limited response")
	}

	rl.Stop()
}
