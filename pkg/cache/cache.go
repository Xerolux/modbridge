package cache

import (
	"sync"
	"time"
)

// Entry represents a cache entry
type Entry struct {
	Key       string
	Value     any
	ExpiresAt time.Time
	TTL       time.Duration
}

// IsExpired checks if the entry is expired
func (e *Entry) IsExpired() bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.ExpiresAt)
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	MaxSize    int           // Maximum number of entries
	DefaultTTL time.Duration // Default TTL for entries
	CleanupInterval time.Duration // How often to clean expired entries
}

// DefaultConfig returns default cache configuration
func DefaultConfig() *CacheConfig {
	return &CacheConfig{
		MaxSize:         10000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
}

// Cache implements an LRU cache with TTL support
type Cache struct {
	mu       sync.RWMutex
	config   *CacheConfig
	entries  map[string]*Entry
	lruList  []string // LRU tracking (least recently used at end)
	stopChan chan struct{}
	stats    CacheStats
}

// CacheStats holds cache statistics
type CacheStats struct {
	Hits     uint64
	Misses   uint64
	Evictions uint64
	Expirations uint64
	Size     int
}

// NewCache creates a new cache
func NewCache(config *CacheConfig) *Cache {
	if config == nil {
		config = DefaultConfig()
	}

	c := &Cache{
		config:   config,
		entries:  make(map[string]*Entry),
		lruList:  make([]string, 0, config.MaxSize),
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	} else if c.config.DefaultTTL > 0 {
		expiresAt = time.Now().Add(c.config.DefaultTTL)
	}

	entry := &Entry{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
		TTL:       ttl,
	}

	// Check if updating existing entry
	if _, exists := c.entries[key]; !exists {
		// Evict if at capacity
		if len(c.entries) >= c.config.MaxSize {
			c.evictLRU()
		}
	}

	c.entries[key] = entry
	c.updateLRU(key)
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	if entry.IsExpired() {
		delete(c.entries, entry.Key)
		c.stats.Misses++
		c.stats.Expirations++
		return nil, false
	}

	c.stats.Hits++
	c.updateLRU(key)
	return entry.Value, true
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; exists {
		delete(c.entries, key)
		c.removeFromLRU(key)
	}
}

// Clear clears all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*Entry)
	c.lruList = make([]string, 0, c.config.MaxSize)
}

// Size returns the current number of entries
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Size = len(c.entries)
	return stats
}

// Close stops the cleanup goroutine
func (c *Cache) Close() error {
	close(c.stopChan)
	return nil
}

// evictLRU evicts the least recently used entry
func (c *Cache) evictLRU() {
	if len(c.lruList) == 0 {
		return
	}

	// Get LRU key (first in list)
	key := c.lruList[0]
	delete(c.entries, key)
	c.lruList = c.lruList[1:]
	c.stats.Evictions++
}

// updateLRU updates the LRU list for a key
func (c *Cache) updateLRU(key string) {
	// Remove key if it exists
	c.removeFromLRU(key)

	// Add to end (most recently used)
	c.lruList = append(c.lruList, key)
}

// removeFromLRU removes a key from the LRU list
func (c *Cache) removeFromLRU(key string) {
	for i, k := range c.lruList {
		if k == key {
			c.lruList = append(c.lruList[:i], c.lruList[i+1:]...)
			break
		}
	}
}

// cleanupLoop runs periodic cleanup of expired entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			return
		}
	}
}

// cleanup removes expired entries
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.entries {
		if entry.IsExpired() {
			delete(c.entries, key)
			c.removeFromLRU(key)
			c.stats.Expirations++
		}
	}
}

// RegisterCache caches Modbus register values
type RegisterCache struct {
	cache *Cache
}

// NewRegisterCache creates a new register cache
func NewRegisterCache(config *CacheConfig) *RegisterCache {
	return &RegisterCache{
		cache: NewCache(config),
	}
}

// GetRegister gets cached register values
func (r *RegisterCache) GetRegister(deviceID string, address uint16, quantity uint16) ([]uint16, bool) {
	key := makeRegisterKey(deviceID, address, quantity)
	val, ok := r.cache.Get(key)
	if !ok {
		return nil, false
	}

	values, ok := val.([]uint16)
	return values, ok
}

// SetRegister caches register values
func (r *RegisterCache) SetRegister(deviceID string, address uint16, values []uint16, ttl time.Duration) {
	key := makeRegisterKey(deviceID, address, uint16(len(values)))
	r.cache.Set(key, values, ttl)
}

// makeRegisterKey creates a cache key for register values
func makeRegisterKey(deviceID string, address uint16, quantity uint16) string {
	return deviceID + ":" + string(rune(address)) + ":" + string(rune(quantity))
}
