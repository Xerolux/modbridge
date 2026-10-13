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
	UnitID      uint8
}

// ResponseCache provides response caching for Modbus reads.
//
// Only read requests are ever cached, and a write to a unit drops every entry
// belonging to that unit: a cached register that a write just changed would
// hand the client a value that is not merely old but wrong.
type ResponseCache struct {
	mu             sync.Mutex
	cache          map[uint64]*ResponseCacheEntry
	maxSize        int
	ttl            time.Duration
	stats          CacheStats
	evictionPolicy EvictionPolicy

	// unitGenerations counts invalidations per unit. A reader captures the
	// generation before its request goes out and stores the answer only when
	// the generation still matches, so a read answered by the device just
	// before a concurrent write can never repopulate the cache with the
	// pre-write value after the write's invalidation.
	unitGenerations map[uint8]uint64
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
		cache:           make(map[uint64]*ResponseCacheEntry),
		maxSize:         config.MaxSize,
		ttl:             config.TTL,
		evictionPolicy:  config.EvictionPolicy,
		unitGenerations: make(map[uint8]uint64),
	}
}

// TTL returns the configured lifetime of a cached entry.
func (rc *ResponseCache) TTL() time.Duration {
	return rc.ttl
}

// Get returns a copy of the cached response for hash, if it has not expired.
//
// The copy matters: the caller rewrites the transaction ID of the frame it
// serves, and doing that in place would corrupt the entry for every client
// after it.
func (rc *ResponseCache) Get(hash uint64) ([]byte, bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	entry, exists := rc.cache[hash]
	if !exists {
		rc.stats.Misses++
		return nil, false
	}

	now := time.Now()
	if now.After(entry.ExpiresAt) {
		delete(rc.cache, hash)
		rc.stats.Misses++
		return nil, false
	}

	entry.LastAccess = now
	entry.HitCount++
	rc.stats.Hits++

	response := make([]byte, len(entry.Response))
	copy(response, entry.Response)
	return response, true
}

// Set stores a response under hash, replacing any existing entry.
func (rc *ResponseCache) Set(hash uint64, response []byte) {
	rc.SetForUnit(hash, 0, response)
}

// Generation returns the invalidation generation of a unit. Callers capture it
// before forwarding a read and hand it to SetForUnitIfFresh when storing.
func (rc *ResponseCache) Generation(unitID uint8) uint64 {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.unitGenerations[unitID]
}

// SetForUnit stores a response and remembers which unit it belongs to, so that
// a later write to that unit can invalidate it.
func (rc *ResponseCache) SetForUnit(hash uint64, unitID uint8, response []byte) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.storeLocked(hash, unitID, response)
}

// SetForUnitIfFresh stores a response only when the unit's invalidation
// generation still equals gen — no write invalidated the unit between the
// moment the read was issued and now. It reports whether the entry was stored.
func (rc *ResponseCache) SetForUnitIfFresh(gen uint64, hash uint64, unitID uint8, response []byte) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.unitGenerations[unitID] != gen {
		return false
	}
	rc.storeLocked(hash, unitID, response)
	return true
}

// storeLocked inserts or replaces a cache entry. The caller must hold rc.mu.
func (rc *ResponseCache) storeLocked(hash uint64, unitID uint8, response []byte) {
	_, exists := rc.cache[hash]
	if !exists && len(rc.cache) >= rc.maxSize {
		rc.evict()
	}

	now := time.Now()
	stored := make([]byte, len(response))
	copy(stored, response)

	rc.cache[hash] = &ResponseCacheEntry{
		Response:    stored,
		CachedAt:    now,
		ExpiresAt:   now.Add(rc.ttl),
		HitCount:    0,
		LastAccess:  now,
		RequestHash: hash,
		UnitID:      unitID,
	}
}

// InvalidateUnit drops every entry belonging to a unit. Called after a write:
// the write may have changed any register of that unit, and there is no way to
// tell from the frame alone which cached reads it affects. Bumping the unit's
// generation also stops in-flight reads of that unit from storing answers the
// device gave before the write was applied.
func (rc *ResponseCache) InvalidateUnit(unitID uint8) int {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.unitGenerations[unitID]++

	dropped := 0
	for hash, entry := range rc.cache {
		if entry.UnitID == unitID {
			delete(rc.cache, hash)
			dropped++
		}
	}
	return dropped
}

// evict removes one entry according to the configured policy.
// The caller must hold rc.mu.
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
	found := false

	for hash, entry := range rc.cache {
		if !found || entry.LastAccess.Before(oldestTime) {
			found = true
			oldestTime = entry.LastAccess
			oldestHash = hash
		}
	}

	if found {
		delete(rc.cache, oldestHash)
		rc.stats.Evictions++
	}
}

func (rc *ResponseCache) evictLFU() {
	var leastHash uint64
	var leastHits int64
	found := false

	for hash, entry := range rc.cache {
		if !found || entry.HitCount < leastHits {
			found = true
			leastHits = entry.HitCount
			leastHash = hash
		}
	}

	if found {
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
	rc.mu.Lock()
	defer rc.mu.Unlock()

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
