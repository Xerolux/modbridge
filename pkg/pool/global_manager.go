package pool

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// GlobalConnectionManager manages connection pools across all proxies
type GlobalConnectionManager struct {
	mu              sync.RWMutex
	pools           map[string]*Pool // key: targetAddress
	globalLimit     int              // Maximum total connections across all pools
	currentTotal    int32            // Current total connections
	config          GlobalManagerConfig
	healthChecker   *HealthChecker
	muHealth        sync.RWMutex
	lastHealthCheck time.Time
}

// GlobalManagerConfig holds configuration for the global manager
type GlobalManagerConfig struct {
	MaxGlobalConnections    int           // Maximum total connections (default: 10000)
	HealthCheckInterval     time.Duration // How often to check connection health (default: 30s)
	IdleTimeout             time.Duration // How long before idle connections are closed (default: 5min)
	MaxIdlePerPool          int           // Maximum idle connections per pool (default: 100)
	CleanupInterval         time.Duration // How often to cleanup idle connections (default: 1min)
}

// DefaultGlobalManagerConfig returns sensible defaults
func DefaultGlobalManagerConfig() GlobalManagerConfig {
	return GlobalManagerConfig{
		MaxGlobalConnections: 10000,
		HealthCheckInterval:  30 * time.Second,
		IdleTimeout:          5 * time.Minute,
		MaxIdlePerPool:       100,
		CleanupInterval:      1 * time.Minute,
	}
}

// NewGlobalConnectionManager creates a new global connection manager
func NewGlobalConnectionManager(config GlobalManagerConfig) (*GlobalConnectionManager, error) {
	if config.MaxGlobalConnections <= 0 {
		config.MaxGlobalConnections = 10000
	}
	if config.HealthCheckInterval <= 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 5 * time.Minute
	}
	if config.MaxIdlePerPool <= 0 {
		config.MaxIdlePerPool = 100
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 1 * time.Minute
	}

	gm := &GlobalConnectionManager{
		pools:        make(map[string]*Pool),
		globalLimit:  config.MaxGlobalConnections,
		config:       config,
		healthChecker: NewHealthChecker(config.HealthCheckInterval),
	}

	// Start background cleanup
	go gm.cleanupLoop()

	// Start health checks
	go gm.healthCheckLoop()

	return gm, nil
}

// GetOrCreatePool gets an existing pool or creates a new one for the target
func (gm *GlobalConnectionManager) GetOrCreatePool(targetAddr string, dialer func(context.Context) (net.Conn, error)) (*Pool, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Check if pool exists
	if pool, exists := gm.pools[targetAddr]; exists {
		return pool, nil
	}

	// Check global limit
	if atomic.LoadInt32(&gm.currentTotal) >= int32(gm.globalLimit) {
		return nil, fmt.Errorf("global connection limit reached: %d", gm.globalLimit)
	}

	// Create new pool with enhanced settings
	poolCfg := Config{
		InitialSize:    2,
		MaxSize:        min(100, gm.globalLimit/10), // Per-pool limit
		MaxIdleTime:    gm.config.IdleTimeout,
		AcquireTimeout: 10 * time.Second,
		Dialer:         dialer,
	}

	pool, err := NewPool(poolCfg)
	if err != nil {
		return nil, err
	}

	// Track pool in global manager
	gm.pools[targetAddr] = pool

	return pool, nil
}

// incrementConnections safely increments global counter
func (gm *GlobalConnectionManager) incrementConnections() error {
	for {
		current := atomic.LoadInt32(&gm.currentTotal)
		if current >= int32(gm.globalLimit) {
			return errors.New("global connection limit reached")
		}
		if atomic.CompareAndSwapInt32(&gm.currentTotal, current, current+1) {
			return nil
		}
	}
}

// decrementConnections safely decrements global counter
func (gm *GlobalConnectionManager) decrementConnections() {
	atomic.AddInt32(&gm.currentTotal, -1)
}

// GetStats returns global connection statistics
func (gm *GlobalConnectionManager) GetStats() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	totalIdle := 0
	totalActive := 0
	totalMax := 0

	poolStats := make(map[string]PoolStats)
	for addr, pool := range gm.pools {
		stats := pool.Stats() // Call the function
		poolStats[addr] = stats
		totalIdle += stats.IdleConns
		totalActive += stats.ActiveConns
		totalMax += stats.MaxSize
	}

	return map[string]interface{}{
		"total_pools":         len(gm.pools),
		"total_connections":   atomic.LoadInt32(&gm.currentTotal),
		"total_idle":          totalIdle,
		"total_active":        totalActive,
		"total_max":           totalMax,
		"global_limit":        gm.globalLimit,
		"utilization_percent": float64(atomic.LoadInt32(&gm.currentTotal)) / float64(gm.globalLimit) * 100,
		"pools":               poolStats,
		"last_health_check":   gm.lastHealthCheck,
	}
}

// cleanupLoop periodically cleans up idle connections
func (gm *GlobalConnectionManager) cleanupLoop() {
	ticker := time.NewTicker(gm.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		gm.mu.RLock()
		pools := make(map[string]*Pool)
		for k, v := range gm.pools {
			pools[k] = v
		}
		gm.mu.RUnlock()

		// Each pool handles its own internal cleanup
		_ = pools
	}
}

// healthCheckLoop performs periodic health checks on all pools
func (gm *GlobalConnectionManager) healthCheckLoop() {
	ticker := time.NewTicker(gm.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		gm.performHealthChecks()
	}
}

// performHealthChecks checks health of all connection pools
func (gm *GlobalConnectionManager) performHealthChecks() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.lastHealthCheck = time.Now()

	for addr, pool := range gm.pools {
		stats := pool.Stats() // Call the function
		// Check if pool has too many idle connections
		if stats.IdleConns > gm.config.MaxIdlePerPool {
			// Trigger cleanup (this is handled by pool's internal cleanup)
			gm.healthChecker.RecordIssue(addr, "too_many_idle", stats.IdleConns)
		}

		// Check pool utilization
		if stats.ActiveConns >= stats.MaxSize {
			gm.healthChecker.RecordIssue(addr, "pool_exhausted", stats.ActiveConns)
		}
	}
}

// Close closes all pools and stops the manager
func (gm *GlobalConnectionManager) Close() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	var lastErr error
	for _, pool := range gm.pools {
		if err := pool.Close(); err != nil {
			lastErr = err
		}
	}

	gm.pools = make(map[string]*Pool)
	return lastErr
}

// HealthChecker tracks health issues
type HealthChecker struct {
	mu             sync.RWMutex
	issues         map[string]*HealthIssue // key: targetAddress
	lastCheck      time.Time
	checkInterval  time.Duration
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Type        string
	Count       int
	FirstSeen   time.Time
	LastSeen    time.Time
	Severity    string // "info", "warning", "critical"
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(interval time.Duration) *HealthChecker {
	return &HealthChecker{
		issues:        make(map[string]*HealthIssue),
		checkInterval: interval,
		lastCheck:     time.Now(),
	}
}

// RecordIssue records a health issue
func (hc *HealthChecker) RecordIssue(target, issueType string, value interface{}) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	now := time.Now()

	if existing, exists := hc.issues[target]; exists {
		existing.Count++
		existing.LastSeen = now
		// Increase severity if issue persists
		if existing.Count > 10 {
			existing.Severity = "critical"
		} else if existing.Count > 5 {
			existing.Severity = "warning"
		}
	} else {
		hc.issues[target] = &HealthIssue{
			Type:      issueType,
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
			Severity:  "info",
		}
	}
}

// GetIssues returns all current health issues
func (hc *HealthChecker) GetIssues() map[string]*HealthIssue {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	issues := make(map[string]*HealthIssue)
	for k, v := range hc.issues {
		issues[k] = v
	}
	return issues
}

// ClearIssue clears a health issue
func (hc *HealthChecker) ClearIssue(target string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.issues, target)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
