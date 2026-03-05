package proxy

import (
	"sync"
	"time"
)

// ResponseCacheEntry represents a cached response
type ResponseCacheEntry struct {
	Response    []byte
	CachedAt    time.Time
	ExpiresAt   time.Time
	HitCount    int64
	LastAccess  time.Time
	RequestHash uint64
}

// ResponseCache provides intelligent response caching for Modbus
type ResponseCache struct {
	mu             sync.RWMutex
	cache          map[uint64]*ResponseCacheEntry
	maxSize        int
	ttl            time.Duration
	stats          CacheStats
	evictionPolicy EvictionPolicy
	muStats        sync.Mutex
}

// EvictionPolicy defines cache eviction strategy
type EvictionPolicy int

const (
	EvictLRU EvictionPolicy = iota // Least Recently Used
	EvictLFU                       // Least Frequently Used
	EvictTTL                       // Time-based only
)

// CacheStats holds cache statistics
type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Size      int
}

// ResponseCacheConfig holds cache configuration
type ResponseCacheConfig struct {
	MaxSize        int           // Maximum cache entries (default: 10000)
	TTL            time.Duration // Time to live for cached entries (default: 5s)
	EvictionPolicy EvictionPolicy
}

// DefaultResponseCacheConfig returns sensible defaults
func DefaultResponseCacheConfig() ResponseCacheConfig {
	return ResponseCacheConfig{
		MaxSize:        10000,
		TTL:            5 * time.Second,
		EvictionPolicy: EvictLRU,
	}
}

// NewResponseCache creates a new response cache
func NewResponseCache(config ResponseCacheConfig) *ResponseCache {
	if config.MaxSize <= 0 {
		config.MaxSize = 10000
	}
	if config.TTL <= 0 {
		config.TTL = 5 * time.Second
	}

	return &ResponseCache{
		cache:          make(map[uint64]*ResponseCacheEntry),
		maxSize:        config.MaxSize,
		ttl:            config.TTL,
		evictionPolicy: config.EvictionPolicy,
	}
}

// Get retrieves a cached response
func (rc *ResponseCache) Get(hash uint64) ([]byte, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.cache[hash]
	if !exists {
		rc.recordMiss()
		return nil, false
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		rc.mu.RUnlock()
		rc.mu.Lock()
		delete(rc.cache, hash)
		rc.mu.Unlock()
		rc.mu.RLock()
		rc.recordMiss()
		return nil, false
	}

	// Update access info
	entry.LastAccess = time.Now()
	entry.HitCount++

	rc.recordHit()
	return entry.Response, true
}

// Set stores a response in the cache
func (rc *ResponseCache) Set(hash uint64, response []byte) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Check if we need to evict
	if len(rc.cache) >= rc.maxSize {
		rc.evict()
	}

	now := time.Now()

	rc.cache[hash] = &ResponseCacheEntry{
		Response:    make([]byte, len(response)),
		CachedAt:    now,
		ExpiresAt:   now.Add(rc.ttl),
		HitCount:    0,
		LastAccess:  now,
		RequestHash: hash,
	}

	copy(rc.cache[hash].Response, response)
}

// evict removes entries based on eviction policy
func (rc *ResponseCache) evict() {
	switch rc.evictionPolicy {
	case EvictLRU:
		rc.evictLRU()
	case EvictLFU:
		rc.evictLFU()
	case EvictTTL:
		rc.evictExpired()
	}
}

// evictLRU removes least recently used entry
func (rc *ResponseCache) evictLRU() {
	var oldestHash uint64
	var oldestTime time.Time

	for hash, entry := range rc.cache {
		if oldestTime.IsZero() || entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestHash = hash
		}
	}

	if oldestHash != 0 {
		delete(rc.cache, oldestHash)
		rc.recordEviction()
	}
}

// evictLFU removes least frequently used entry
func (rc *ResponseCache) evictLFU() {
	var leastHash uint64
	var leastHits int64

	for hash, entry := range rc.cache {
		if leastHits == 0 || entry.HitCount < leastHits {
			leastHits = entry.HitCount
			leastHash = hash
		}
	}

	if leastHash != 0 {
		delete(rc.cache, leastHash)
		rc.recordEviction()
	}
}

// evictExpired removes all expired entries
func (rc *ResponseCache) evictExpired() {
	now := time.Now()
	for hash, entry := range rc.cache {
		if now.After(entry.ExpiresAt) {
			delete(rc.cache, hash)
			rc.recordEviction()
		}
	}
}

// Clear clears the entire cache
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache = make(map[uint64]*ResponseCacheEntry)
}

// recordHit records a cache hit
func (rc *ResponseCache) recordHit() {
	rc.muStats.Lock()
	rc.stats.Hits++
	rc.muStats.Unlock()
}

// recordMiss records a cache miss
func (rc *ResponseCache) recordMiss() {
	rc.muStats.Lock()
	rc.stats.Misses++
	rc.muStats.Unlock()
}

// recordEviction records a cache eviction
func (rc *ResponseCache) recordEviction() {
	rc.muStats.Lock()
	rc.stats.Evictions++
	rc.muStats.Unlock()
}

// GetStatsWithHitRate returns cache statistics with hit rate
func (rc *ResponseCache) GetStatsWithHitRate() CacheStatsWithHitRate {
	rc.muStats.Lock()
	defer rc.muStats.Unlock()

	rc.mu.RLock()
	rc.stats.Size = len(rc.cache)
	rc.mu.RUnlock()

	hitRate := 0.0
	total := rc.stats.Hits + rc.stats.Misses
	if total > 0 {
		hitRate = float64(rc.stats.Hits) / float64(total) * 100
	}

	return CacheStatsWithHitRate{
		Hits:      rc.stats.Hits,
		Misses:    rc.stats.Misses,
		Evictions: rc.stats.Evictions,
		Size:      rc.stats.Size,
		HitRate:   hitRate,
	}
}

// CacheStats with hit rate
type CacheStatsWithHitRate struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Size      int
	HitRate   float64
}
