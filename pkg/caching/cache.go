package caching

import (
	"sync"
	"time"
)

// CacheEntry represents a cached register value
type CacheEntry struct {
	Value     interface{}
	RawValue  uint16
	Quality   string
	Timestamp time.Time
	TTL       time.Duration
}

// Manager manages register caching
type Manager struct {
	mu    sync.RWMutex
	cache map[string]map[int]*CacheEntry // proxy_id -> register_address -> entry
	ttl   time.Duration
}

// NewManager creates a new cache manager
func NewManager(defaultTTL time.Duration) *Manager {
	m := &Manager{
		cache: make(map[string]map[int]*CacheEntry),
		ttl:   defaultTTL,
	}
	go m.cleanupExpired()
	return m
}

// Set sets a cached value
func (m *Manager) Set(proxyID string, registerAddress int, value interface{}, rawValue uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cache[proxyID] == nil {
		m.cache[proxyID] = make(map[int]*CacheEntry)
	}

	m.cache[proxyID][registerAddress] = &CacheEntry{
		Value:     value,
		RawValue:  rawValue,
		Quality:   "good",
		Timestamp: time.Now(),
		TTL:       m.ttl,
	}
}

// Get gets a cached value
func (m *Manager) Get(proxyID string, registerAddress int) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.cache[proxyID] == nil {
		return nil, false
	}

	entry, ok := m.cache[proxyID][registerAddress]
	if !ok {
		return nil, false
	}

	// Check if expired
	if time.Since(entry.Timestamp) > entry.TTL {
		return nil, false
	}

	return entry.Value, true
}

// GetWithQuality gets a cached value with quality information
func (m *Manager) GetWithQuality(proxyID string, registerAddress int) (interface{}, string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.cache[proxyID] == nil {
		return nil, "", false
	}

	entry, ok := m.cache[proxyID][registerAddress]
	if !ok {
		return nil, "", false
	}

	// Check if expired
	if time.Since(entry.Timestamp) > entry.TTL {
		return nil, "stale", true
	}

	return entry.Value, entry.Quality, true
}

// Invalidate invalidates a cached value
func (m *Manager) Invalidate(proxyID string, registerAddress int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cache[proxyID] != nil {
		delete(m.cache[proxyID], registerAddress)
	}
}

// InvalidateProxy invalidates all cached values for a proxy
func (m *Manager) InvalidateProxy(proxyID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.cache, proxyID)
}

// cleanupExpired periodically removes expired cache entries
func (m *Manager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for proxyID, registers := range m.cache {
			for addr, entry := range registers {
				if now.Sub(entry.Timestamp) > entry.TTL {
					delete(registers, addr)
				}
			}
			if len(m.cache[proxyID]) == 0 {
				delete(m.cache, proxyID)
			}
		}
		m.mu.Unlock()
	}
}

// GetStats returns cache statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalEntries := 0
	for _, registers := range m.cache {
		totalEntries += len(registers)
	}

	return map[string]interface{}{
		"total_proxies": len(m.cache),
		"total_entries": totalEntries,
		"default_ttl":   m.ttl.String(),
	}
}
