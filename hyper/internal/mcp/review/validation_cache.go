package review

import (
	"time"
)

// NewValidationCache creates a new validation cache with the specified TTL
func NewValidationCache(ttl time.Duration) *ValidationCache {
	return &ValidationCache{
		cache: make(map[string]*ValidationCacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves a cached validation result
// Returns (result, found) where found is false if key doesn't exist or is expired
func (vc *ValidationCache) Get(key string) (bool, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	entry, exists := vc.cache[key]
	if !exists {
		return false, false
	}

	// Check if expired
	if time.Since(entry.Timestamp) > vc.ttl {
		return false, false
	}

	return entry.Result, true
}

// Set stores a validation result in the cache
func (vc *ValidationCache) Set(key string, result bool) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.cache[key] = &ValidationCacheEntry{
		Result:    result,
		Timestamp: time.Now(),
	}
}

// Clear removes all entries from the cache
func (vc *ValidationCache) Clear() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.cache = make(map[string]*ValidationCacheEntry)
}

// CleanExpired removes expired entries from the cache
func (vc *ValidationCache) CleanExpired() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	now := time.Now()
	for key, entry := range vc.cache {
		if now.Sub(entry.Timestamp) > vc.ttl {
			delete(vc.cache, key)
		}
	}
}

// Size returns the current number of non-expired cached entries
func (vc *ValidationCache) Size() int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// Count only non-expired entries
	count := 0
	now := time.Now()
	for _, entry := range vc.cache {
		if now.Sub(entry.Timestamp) <= vc.ttl {
			count++
		}
	}
	return count
}

