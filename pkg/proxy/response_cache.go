// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

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

func (rc *ResponseCache) Get(hash uint64) ([]byte, bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	entry, exists := rc.cache[hash]
	if !exists {
		rc.stats.Misses++
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(rc.cache, hash)
		rc.stats.Misses++
		return nil, false
	}

	entry.LastAccess = time.Now()
	entry.HitCount++

	rc.stats.Hits++
	return entry.Response, true
}

func (rc *ResponseCache) Set(hash uint64, response []byte) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

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
		rc.stats.Evictions++
	}
}

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
		rc.stats.Evictions++
	}
}

func (rc *ResponseCache) evictExpired() {
	now := time.Now()
	for hash, entry := range rc.cache {
		if now.After(entry.ExpiresAt) {
			delete(rc.cache, hash)
			rc.stats.Evictions++
		}
	}
}

func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache = make(map[uint64]*ResponseCacheEntry)
}

func (rc *ResponseCache) GetStatsWithHitRate() CacheStatsWithHitRate {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	size := len(rc.cache)

	hitRate := 0.0
	total := rc.stats.Hits + rc.stats.Misses
	if total > 0 {
		hitRate = float64(rc.stats.Hits) / float64(total) * 100
	}

	return CacheStatsWithHitRate{
		Hits:      rc.stats.Hits,
		Misses:    rc.stats.Misses,
		Evictions: rc.stats.Evictions,
		Size:      size,
		HitRate:   hitRate,
	}
}

type CacheStatsWithHitRate struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Size      int
	HitRate   float64
}
