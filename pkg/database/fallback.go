package database

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrDatabaseUnavailable is returned when the database is unavailable
	ErrDatabaseUnavailable = errors.New("database unavailable")
	// ErrCircuitBreakerOpen is returned when the circuit breaker is open
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	// CircuitClosed means the circuit is closed and requests are allowed
	CircuitClosed CircuitBreakerState = iota
	// CircuitOpen means the circuit is open and requests are blocked
	CircuitOpen
	// CircuitHalfOpen means the circuit is half-open testing if connection is restored
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for database connections
type CircuitBreaker struct {
	mu                    sync.RWMutex
	state                 CircuitBreakerState
	failureCount          int
	successCount          int
	failureThreshold      int
	successThreshold      int
	timeout               time.Duration
	lastFailureTime       time.Time
	halfOpenSuccessCount  int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:                CircuitClosed,
		failureThreshold:     failureThreshold,
		successThreshold:     successThreshold,
		timeout:              timeout,
		failureCount:         0,
		successCount:         0,
		halfOpenSuccessCount: 0,
	}
}

// CanProceed checks if the request can proceed based on circuit breaker state
func (cb *CircuitBreaker) CanProceed() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if timeout has passed to try half-open
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenSuccessCount = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return false
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount = 0
	case CircuitHalfOpen:
		cb.halfOpenSuccessCount++
		if cb.halfOpenSuccessCount >= cb.successThreshold {
			cb.state = CircuitClosed
			cb.failureCount = 0
			cb.halfOpenSuccessCount = 0
		}
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = CircuitOpen
			cb.lastFailureTime = time.Now()
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.lastFailureTime = time.Now()
		cb.halfOpenSuccessCount = 0
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// HealthChecker performs health checks on the database
type HealthChecker struct {
	db            *DB
	checkInterval time.Duration
	timeout       time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
	isHealthy     bool
	mu            sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *DB, checkInterval, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		db:            db,
		checkInterval: checkInterval,
		timeout:       timeout,
		stopCh:        make(chan struct{}),
		isHealthy:     true,
	}
}

// Start begins the health check loop
func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.run()
}

// Stop stops the health check loop
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

// run executes the health check loop
func (hc *HealthChecker) run() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	// Initial check
	hc.performCheck()

	for {
		select {
		case <-ticker.C:
			hc.performCheck()
		case <-hc.stopCh:
			return
		}
	}
}

// performCheck performs a single health check
func (hc *HealthChecker) performCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	err := hc.db.conn.PingContext(ctx)
	healthy := err == nil

	hc.mu.Lock()
	hc.isHealthy = healthy
	hc.mu.Unlock()
}

// IsHealthy returns whether the database is healthy
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.isHealthy
}

// DBWithFallback wraps the DB with fallback mechanisms
type DBWithFallback struct {
	*DB
	circuitBreaker *CircuitBreaker
	healthChecker  *HealthChecker
	fallbackCache  *FallbackCache
	mu             sync.RWMutex
}

// FallbackCache provides in-memory fallback for critical data
type FallbackCache struct {
	devices       map[string]*Device
	requestCounts map[string]int64
	mu            sync.RWMutex
	maxSize       int
}

// NewFallbackCache creates a new fallback cache
func NewFallbackCache(maxSize int) *FallbackCache {
	return &FallbackCache{
		devices:       make(map[string]*Device),
		requestCounts: make(map[string]int64),
		maxSize:       maxSize,
	}
}

// SetDevice stores a device in the cache
func (fc *FallbackCache) SetDevice(device *Device) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Evict oldest if at capacity
	if len(fc.devices) >= fc.maxSize {
		// Simple eviction: remove first entry
		for k := range fc.devices {
			delete(fc.devices, k)
			break
		}
	}

	fc.devices[device.IP] = device
}

// GetDevice retrieves a device from the cache
func (fc *FallbackCache) GetDevice(ip string) (*Device, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	device, ok := fc.devices[ip]
	return device, ok
}

// IncrementRequestCount increments the request count in cache
func (fc *FallbackCache) IncrementRequestCount(ip string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.requestCounts[ip]++
}

// NewDBWithFallback creates a new database with fallback mechanisms
func NewDBWithFallback(path string) (*DBWithFallback, error) {
	// Create base database
	db, err := NewDB(path)
	if err != nil {
		return nil, err
	}

	// Create circuit breaker with thresholds
	cb := NewCircuitBreaker(
		5,  // Open circuit after 5 failures
		2,  // Close circuit after 2 successes in half-open
		30*time.Second, // Try to recover after 30 seconds
	)

	// Create health checker
	hc := NewHealthChecker(db, 10*time.Second, 5*time.Second)
	hc.Start()

	// Create fallback cache
	fc := NewFallbackCache(1000) // Cache up to 1000 devices

	dbw := &DBWithFallback{
		DB:             db,
		circuitBreaker: cb,
		healthChecker:  hc,
		fallbackCache:  fc,
	}

	return dbw, nil
}

// Close closes the database and stops the health checker
func (dbw *DBWithFallback) Close() error {
	if dbw.healthChecker != nil {
		dbw.healthChecker.Stop()
	}
	return dbw.DB.Close()
}

// SaveDeviceWithFallback saves a device with fallback to cache
func (dbw *DBWithFallback) SaveDeviceWithFallback(device *Device) error {
	if !dbw.circuitBreaker.CanProceed() {
		// Circuit is open, use cache
		dbw.fallbackCache.SetDevice(device)
		return ErrDatabaseUnavailable
	}

	err := dbw.SaveDevice(device)
	if err != nil {
		dbw.circuitBreaker.RecordFailure()
		// Fallback to cache
		dbw.fallbackCache.SetDevice(device)
		return fmt.Errorf("%w: %v", ErrDatabaseUnavailable, err)
	}

	dbw.circuitBreaker.RecordSuccess()
	return nil
}

// GetDeviceWithFallback retrieves a device with fallback to cache
func (dbw *DBWithFallback) GetDeviceWithFallback(ip string) (*Device, error) {
	// Try cache first
	if device, ok := dbw.fallbackCache.GetDevice(ip); ok {
		return device, nil
	}

	if !dbw.circuitBreaker.CanProceed() {
		return nil, ErrCircuitBreakerOpen
	}

	device, err := dbw.GetDevice(ip)
	if err != nil {
		dbw.circuitBreaker.RecordFailure()
		return nil, err
	}

	dbw.circuitBreaker.RecordSuccess()
	return device, nil
}

// IncrementRequestCountWithFallback increments request count with fallback
func (dbw *DBWithFallback) IncrementRequestCountWithFallback(ip string) error {
	// Update cache immediately
	dbw.fallbackCache.IncrementRequestCount(ip)

	if !dbw.circuitBreaker.CanProceed() {
		// Cache updated, silently fail
		return ErrDatabaseUnavailable
	}

	err := dbw.IncrementRequestCount(ip)
	if err != nil {
		dbw.circuitBreaker.RecordFailure()
		return fmt.Errorf("%w: %v", ErrDatabaseUnavailable, err)
	}

	dbw.circuitBreaker.RecordSuccess()
	return nil
}

// GetAllDevicesWithFallback retrieves all devices with fallback
func (dbw *DBWithFallback) GetAllDevicesWithFallback() ([]*Device, error) {
	if !dbw.circuitBreaker.CanProceed() {
		// Return cached devices
		dbw.fallbackCache.mu.RLock()
		defer dbw.fallbackCache.mu.RUnlock()

		devices := make([]*Device, 0, len(dbw.fallbackCache.devices))
		for _, device := range dbw.fallbackCache.devices {
			devices = append(devices, device)
		}
		return devices, ErrDatabaseUnavailable
	}

	devices, err := dbw.GetAllDevices()
	if err != nil {
		dbw.circuitBreaker.RecordFailure()
		return nil, err
	}

	dbw.circuitBreaker.RecordSuccess()

	// Update cache with fresh data
	for _, device := range devices {
		dbw.fallbackCache.SetDevice(device)
	}

	return devices, nil
}

// IsHealthy returns whether the database is healthy
func (dbw *DBWithFallback) IsHealthy() bool {
	return dbw.healthChecker.IsHealthy()
}

// GetCircuitBreakerState returns the current circuit breaker state
func (dbw *DBWithFallback) GetCircuitBreakerState() CircuitBreakerState {
	return dbw.circuitBreaker.GetState()
}

// GetStats returns database statistics
func (dbw *DBWithFallback) GetStats() map[string]interface{} {
	dbw.mu.RLock()
	defer dbw.mu.RUnlock()

	stats := map[string]interface{}{
		"healthy":          dbw.IsHealthy(),
		"circuit_state":     dbw.GetCircuitBreakerState().String(),
		"cached_devices":   len(dbw.fallbackCache.devices),
	}

	return stats
}

// String returns the string representation of the circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}
