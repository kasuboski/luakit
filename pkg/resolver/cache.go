package resolver

import (
	"sync"
)

// CacheEntry holds cached image configuration
type CacheEntry struct {
	config any
	err    error
}

// Cache provides per-reference caching with thread safety
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
}

// NewCache creates a new cache
func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
	}
}

// Get retrieves a cached entry for the given reference
func (c *Cache) Get(ref string) (any, error) {
	c.mu.RLock()
	entry, exists := c.entries[ref]
	c.mu.RUnlock()

	if !exists {
		return nil, nil
	}

	return entry.config, entry.err
}

// Set stores a cached entry for the given reference
func (c *Cache) Set(ref string, config any, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[ref] = &CacheEntry{
		config: config,
		err:    err,
	}
}

// Clear removes all cached entries
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
}
