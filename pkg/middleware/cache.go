package middleware

import (
	"sync"
	"time"
)

// CacheEntry represents a cached item
type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
}

// Expired checks if the cache entry has expired
func (e *CacheEntry) Expired() bool {
	return time.Now().After(e.Expiration)
}

// Cache is a simple in-memory cache with TTL
type Cache struct {
	items map[string]CacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewCache creates a new cache
func NewCache(ttl time.Duration) *Cache {
	cache := &Cache{
		items: make(map[string]CacheEntry),
		ttl:   ttl,
	}
	go cache.cleanup()
	return cache
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(c.ttl),
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found || item.Expired() {
		return nil, false
	}
	return item.Value, true
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]CacheEntry)
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// cleanup removes expired entries
func (c *Cache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.Expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
