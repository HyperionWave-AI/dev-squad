package redis

import (
	"context"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func resetRedisStateForTest() {
	if client != nil {
		_ = client.Close()
	}
	client = nil
	clientOnce = sync.Once{}
	initLogger = nil
}

func TestGetClient_NoRedisURL(t *testing.T) {
	resetRedisStateForTest()
	t.Setenv("REDIS_URL", "")

	got := GetClient(zap.NewNop())
	if got != nil {
		t.Fatalf("expected nil client when REDIS_URL is not set, got %#v", got)
	}
	if IsAvailable() {
		t.Fatal("expected Redis to be unavailable without REDIS_URL")
	}
}

func TestGetClient_InvalidRedisURL(t *testing.T) {
	resetRedisStateForTest()
	t.Setenv("REDIS_URL", "://bad-url")

	got := GetClient(zap.NewNop())
	if got != nil {
		t.Fatalf("expected nil client for invalid REDIS_URL, got %#v", got)
	}
}

func TestGetClient_UnreachableRedis(t *testing.T) {
	resetRedisStateForTest()
	t.Setenv("REDIS_URL", "redis://127.0.0.1:0/0")

	got := GetClient(zap.NewNop())
	if got != nil {
		t.Fatalf("expected nil client for unreachable Redis, got %#v", got)
	}
}

func TestPingAndCloseWithoutClient(t *testing.T) {
	resetRedisStateForTest()

	if err := Ping(context.Background()); err != nil {
		t.Fatalf("expected nil ping error when client is nil, got %v", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("expected nil close error when client is nil, got %v", err)
	}
}

func TestPingWithClientErrorAndCloseBranch(t *testing.T) {
	resetRedisStateForTest()
	client = goredis.NewClient(&goredis.Options{
		Addr: "127.0.0.1:0",
	})
	t.Cleanup(func() {
		_ = Close()
		resetRedisStateForTest()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := Ping(ctx); err == nil {
		t.Fatal("expected ping error for invalid redis address")
	}

	if err := Close(); err != nil {
		t.Fatalf("expected close to succeed, got %v", err)
	}
}
