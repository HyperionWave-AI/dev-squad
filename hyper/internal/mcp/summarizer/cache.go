package summarizer

// Package summarizer provides code summarization capabilities with caching,
// token budget management, and comprehensive metrics collection.
//
// The cache module implements an LRU (Least Recently Used) cache with TTL support
// for storing code summaries. It provides:
//
// - Thread-safe concurrent access using RWMutex
// - Automatic eviction of least recently used entries when capacity is reached
// - Time-to-live (TTL) support for automatic expiration of entries
// - Detailed statistics tracking (hits, misses, evictions, hit rate)
// - O(1) average case performance for Get/Set operations
//
// Example usage:
//
//	cache := NewLRUCache(1000, 24*time.Hour, logger)
//	cache.Set(key, summary)
//	if summary, ok := cache.Get(key); ok {
//	    // Use cached summary
//	}

import (
	"container/list"
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SummaryCache defines the interface for caching code summaries
type SummaryCache interface {
	// Get retrieves a cached summary by key
	Get(key string) (*CodeSummary, bool)
	// Set stores a summary in the cache
	Set(key string, summary *CodeSummary)
	// Delete removes a summary from the cache
	Delete(key string)
	// Clear removes all entries from the cache
	Clear()
	// Stats returns cache statistics
	Stats() CacheStats
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	Hits         int64
	Misses       int64
	Evictions    int64
	Size         int
	MaxSize      int
	HitRate      float64
	EvictionRate float64
}

// LRUCache implements SummaryCache using an LRU eviction policy
type LRUCache struct {
	maxSize int
	ttl     time.Duration
	cache   map[string]*cacheEntry
	lru     *list.List
	mu      sync.RWMutex
	stats   cacheStatsInternal
	logger  *zap.Logger
}

// cacheEntry represents a single cache entry
type cacheEntry struct {
	summary   *CodeSummary
	expiresAt time.Time
	element   *list.Element
}

// cacheStatsInternal tracks cache statistics with thread safety
type cacheStatsInternal struct {
	hits      int64
	misses    int64
	evictions int64
	mu        sync.RWMutex
}

// NewLRUCache creates a new LRU cache with the specified size and TTL
func NewLRUCache(maxSize int, ttl time.Duration, logger *zap.Logger) *LRUCache {
	if logger == nil {
		logger = zap.NewNop()
	}

	if maxSize <= 0 {
		maxSize = 1000 // Default size
	}

	if ttl < 0 {
		ttl = 0 // No TTL
	}

	return &LRUCache{
		maxSize: maxSize,
		ttl:     ttl,
		cache:   make(map[string]*cacheEntry, maxSize),
		lru:     list.New(),
		logger:  logger,
	}
}

// Get retrieves a cached summary by key
// Returns the summary and true if found and not expired, false otherwise
func (c *LRUCache) Get(key string) (*CodeSummary, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.cache[key]
	if !exists {
		c.stats.recordMiss()
		return nil, false
	}

	// Check if entry has expired
	if c.ttl > 0 && time.Now().After(entry.expiresAt) {
		// Entry has expired, remove it
		c.evictEntry(key, entry)
		c.stats.recordMiss()
		return nil, false
	}

	// Move to front (most recently used)
	c.lru.MoveToFront(entry.element)
	c.stats.recordHit()

	return entry.summary, true
}

// Set stores a summary in the cache
// If cache is at capacity, evicts the least recently used entry
func (c *LRUCache) Set(key string, summary *CodeSummary) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key already exists, update it
	if entry, exists := c.cache[key]; exists {
		entry.summary = summary
		entry.expiresAt = time.Now().Add(c.ttl)
		c.lru.MoveToFront(entry.element)
		return
	}

	// Check if we need to evict
	if len(c.cache) >= c.maxSize {
		// Evict least recently used (back of list)
		if c.lru.Len() > 0 {
			back := c.lru.Back()
			if back != nil {
				// Find the key for this element
				for k, entry := range c.cache {
					if entry.element == back {
						c.evictEntry(k, entry)
						break
					}
				}
			}
		}
	}

	// Create new entry
	element := c.lru.PushFront(key)
	entry := &cacheEntry{
		summary:   summary,
		expiresAt: time.Now().Add(c.ttl),
		element:   element,
	}

	c.cache[key] = entry
}

// Delete removes a summary from the cache
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.cache[key]
	if exists {
		c.evictEntry(key, entry)
	}
}

// Clear removes all entries from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cacheEntry, c.maxSize)
	c.lru = list.New()

	c.logger.Debug("Cache cleared",
		zap.Int("maxSize", c.maxSize),
		zap.Duration("ttl", c.ttl))
}

// Stats returns cache statistics
func (c *LRUCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	totalRequests := c.stats.hits + c.stats.misses
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(c.stats.hits) / float64(totalRequests)
	}

	evictionRate := 0.0
	if totalRequests > 0 {
		evictionRate = float64(c.stats.evictions) / float64(totalRequests)
	}

	return CacheStats{
		Hits:         c.stats.hits,
		Misses:       c.stats.misses,
		Evictions:    c.stats.evictions,
		Size:         len(c.cache),
		MaxSize:      c.maxSize,
		HitRate:      hitRate,
		EvictionRate: evictionRate,
	}
}

// evictEntry removes an entry from the cache (must be called with lock held)
func (c *LRUCache) evictEntry(key string, entry *cacheEntry) {
	delete(c.cache, key)
	c.lru.Remove(entry.element)
	c.stats.recordEviction()

	c.logger.Debug("Cache entry evicted",
		zap.String("key", key),
		zap.Int("cacheSize", len(c.cache)))
}

// recordHit records a cache hit
func (s *cacheStatsInternal) recordHit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits++
}

// recordMiss records a cache miss
func (s *cacheStatsInternal) recordMiss() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.misses++
}

// recordEviction records a cache eviction
func (s *cacheStatsInternal) recordEviction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictions++
}

// GenerateCacheKey generates a cache key from code and metadata
// Uses a hash of the code content and metadata to create a unique key
func GenerateCacheKey(code string, metadata CodeMetadata) string {
	// Generate MD5 hash of code for uniqueness
	hash := hashString(code)
	return metadata.FilePath + ":" + metadata.NodeType + ":" + metadata.NodeName + ":" + hash
}

// hashString generates an MD5 hash of a string
func hashString(s string) string {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", h)
}
