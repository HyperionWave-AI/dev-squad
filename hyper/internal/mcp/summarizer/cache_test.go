package summarizer

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewLRUCache(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(100, 1*time.Hour, logger)

	if cache == nil {
		t.Fatal("NewLRUCache returned nil")
	}

	if cache.maxSize != 100 {
		t.Errorf("Expected maxSize 100, got %d", cache.maxSize)
	}

	if cache.ttl != 1*time.Hour {
		t.Errorf("Expected ttl 1h, got %v", cache.ttl)
	}
}

func TestLRUCacheSetAndGet(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(10, 1*time.Hour, logger)

	summary := &CodeSummary{
		Text:        "Test summary",
		Type:        "llm",
		TokenCount:  10,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	cache.Set("key1", summary)

	retrieved, found := cache.Get("key1")
	if !found {
		t.Fatal("Expected to find key1 in cache")
	}

	if retrieved.Text != "Test summary" {
		t.Errorf("Expected 'Test summary', got '%s'", retrieved.Text)
	}
}

func TestLRUCacheMiss(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(10, 1*time.Hour, logger)

	_, found := cache.Get("nonexistent")
	if found {
		t.Fatal("Expected cache miss for nonexistent key")
	}
}

func TestLRUCacheExpiration(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(10, 100*time.Millisecond, logger)

	summary := &CodeSummary{
		Text:        "Test summary",
		Type:        "llm",
		TokenCount:  10,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	cache.Set("key1", summary)

	// Should be found immediately
	_, found := cache.Get("key1")
	if !found {
		t.Fatal("Expected to find key1 immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.Get("key1")
	if found {
		t.Fatal("Expected cache miss after expiration")
	}
}

func TestLRUCacheEviction(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(3, 1*time.Hour, logger)

	// Add 3 items
	for i := 1; i <= 3; i++ {
		summary := &CodeSummary{
			Text:        "Summary " + string(rune(i)),
			Type:        "llm",
			TokenCount:  10,
			GeneratedAt: time.Now(),
			CacheHit:    false,
		}
		cache.Set("key"+string(rune(i)), summary)
	}

	// Add 4th item, should evict the least recently used (key1)
	summary := &CodeSummary{
		Text:        "Summary 4",
		Type:        "llm",
		TokenCount:  10,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}
	cache.Set("key4", summary)

	// key1 should be evicted
	_, found := cache.Get("key1")
	if found {
		t.Fatal("Expected key1 to be evicted")
	}

	// key4 should be present
	_, found = cache.Get("key4")
	if !found {
		t.Fatal("Expected key4 to be present")
	}
}

func TestLRUCacheDelete(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(10, 1*time.Hour, logger)

	summary := &CodeSummary{
		Text:        "Test summary",
		Type:        "llm",
		TokenCount:  10,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	cache.Set("key1", summary)
	cache.Delete("key1")

	_, found := cache.Get("key1")
	if found {
		t.Fatal("Expected key1 to be deleted")
	}
}

func TestLRUCacheClear(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(10, 1*time.Hour, logger)

	// Add multiple items
	for i := 1; i <= 5; i++ {
		summary := &CodeSummary{
			Text:        "Summary " + string(rune(i)),
			Type:        "llm",
			TokenCount:  10,
			GeneratedAt: time.Now(),
			CacheHit:    false,
		}
		cache.Set("key"+string(rune(i)), summary)
	}

	cache.Clear()

	// All items should be gone
	for i := 1; i <= 5; i++ {
		_, found := cache.Get("key" + string(rune(i)))
		if found {
			t.Fatalf("Expected key%d to be cleared", i)
		}
	}
}

func TestLRUCacheStats(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(10, 1*time.Hour, logger)

	summary := &CodeSummary{
		Text:        "Test summary",
		Type:        "llm",
		TokenCount:  10,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	cache.Set("key1", summary)
	cache.Get("key1") // Hit
	cache.Get("key2") // Miss

	stats := cache.Stats()

	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}

	if stats.HitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", stats.HitRate)
	}
}

func TestLRUCacheUpdateExisting(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(10, 1*time.Hour, logger)

	summary1 := &CodeSummary{
		Text:        "Summary 1",
		Type:        "llm",
		TokenCount:  10,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	cache.Set("key1", summary1)

	summary2 := &CodeSummary{
		Text:        "Summary 2",
		Type:        "heuristic",
		TokenCount:  20,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	cache.Set("key1", summary2)

	retrieved, found := cache.Get("key1")
	if !found {
		t.Fatal("Expected to find key1")
	}

	if retrieved.Text != "Summary 2" {
		t.Errorf("Expected 'Summary 2', got '%s'", retrieved.Text)
	}

	if retrieved.TokenCount != 20 {
		t.Errorf("Expected token count 20, got %d", retrieved.TokenCount)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	metadata := CodeMetadata{
		FilePath: "/path/to/file.go",
		NodeType: "function",
		NodeName: "TestFunc",
	}

	key1 := GenerateCacheKey("code1", metadata)
	key2 := GenerateCacheKey("code1", metadata)

	if key1 != key2 {
		t.Errorf("Expected same key for same code, got %s and %s", key1, key2)
	}

	key3 := GenerateCacheKey("code2", metadata)
	if key1 == key3 {
		t.Errorf("Expected different keys for different code")
	}
}

func TestLRUCacheThreadSafety(t *testing.T) {
	logger := zap.NewNop()
	cache := NewLRUCache(100, 1*time.Hour, logger)

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				summary := &CodeSummary{
					Text:        "Summary",
					Type:        "llm",
					TokenCount:  10,
					GeneratedAt: time.Now(),
					CacheHit:    false,
				}
				cache.Set("key"+string(rune(id*10+j)), summary)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				cache.Get("key" + string(rune(id*10+j)))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := cache.Stats()
	if stats.Size == 0 {
		t.Fatal("Expected cache to have entries after concurrent operations")
	}
}

func BenchmarkLRUCacheGet(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(1000, 1*time.Hour, logger)

	summary := &CodeSummary{
		Text:        "Test summary",
		Type:        "llm",
		TokenCount:  10,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	cache.Set("key1", summary)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("key1")
	}
}

func BenchmarkLRUCacheSet(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(10000, 1*time.Hour, logger)

	summary := &CodeSummary{
		Text:        "Test summary",
		Type:        "llm",
		TokenCount:  10,
		GeneratedAt: time.Now(),
		CacheHit:    false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key"+string(rune(i)), summary)
	}
}
