package bitbucket

import (
	"sync"
	"time"
)

const (
	// DefaultCacheTTL is the default time-to-live for cached API responses.
	DefaultCacheTTL = 30 * time.Second
	// LogCacheTTL is the TTL for step log responses (heavier, less frequent).
	LogCacheTTL = 60 * time.Second
	// DefaultMaxEntrySize is the maximum size in bytes for a cached entry.
	// Entries larger than this (e.g., large step logs) are not cached.
	DefaultMaxEntrySize = 256 * 1024 // 256 KB
)

// entry is a single cached item with raw JSON/text bytes and a timestamp.
type entry struct {
	data     []byte
	cachedAt time.Time
	ttl      time.Duration
}

// Cache is a thread-safe in-memory cache for API responses.
// Keys are URL paths (e.g., "/repositories/ws/repo/pipelines?pagelen=25").
// Values are raw JSON or text bytes, which are deserialized on cache hit.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*entry
}

// NewCache creates a new empty cache.
func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]*entry),
	}
}

// Get retrieves a cached entry by key. Returns nil, false if the key is not
// present or has expired.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Since(e.cachedAt) > e.ttl {
		return nil, false
	}
	return e.data, true
}

// Set stores data in the cache with the given key and TTL. If the data exceeds
// maxEntrySize, it is silently dropped.
func (c *Cache) Set(key string, data []byte, ttl time.Duration, maxEntrySize int) {
	if len(data) > maxEntrySize {
		return
	}
	c.mu.Lock()
	c.entries[key] = &entry{
		data:     data,
		cachedAt: time.Now(),
		ttl:      ttl,
	}
	c.mu.Unlock()
}

// InvalidateAll clears every entry in the cache.
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	c.entries = make(map[string]*entry)
	c.mu.Unlock()
}

// InvalidatePrefix clears all entries whose key starts with the given prefix.
// Useful for scoped invalidation, e.g., prefix="/repositories/ws/repo/" clears
// only cache entries for that repository.
func (c *Cache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	for k := range c.entries {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()
}

// Len returns the number of entries currently in the cache (for diagnostics).
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}