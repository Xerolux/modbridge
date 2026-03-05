package proxy

import (
	"sync"
	"time"
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	StateClosed   CircuitBreakerState = iota // Normal operation
	StateOpen                                // Failing, reject requests
	StateHalfOpen                            // Testing if recovery is possible
)

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	lastStateChange time.Time

	// Configuration
	threshold        int           // Failures before opening
	halfOpenAttempts int           // Success attempts in half-open
	timeout          time.Duration // Time to stay in open state
	successThreshold int           // Successes to close again

	// Metrics
	totalRequests    int64
	totalFailures    int64
	totalSuccesses   int64
	rejectedRequests int64
	lastResetTime    time.Time
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           // Consecutive failures to open circuit (default: 5)
	HalfOpenAttempts int           // Attempts in half-open state (default: 3)
	OpenTimeout      time.Duration // Time in open state (default: 60s)
	SuccessThreshold int           // Successes to close again (default: 2)
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		HalfOpenAttempts: 3,
		OpenTimeout:      60 * time.Second,
		SuccessThreshold: 2,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.HalfOpenAttempts <= 0 {
		config.HalfOpenAttempts = 3
	}
	if config.OpenTimeout <= 0 {
		config.OpenTimeout = 60 * time.Second
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}

	return &CircuitBreaker{
		state:            StateClosed,
		threshold:        config.FailureThreshold,
		halfOpenAttempts: config.HalfOpenAttempts,
		timeout:          config.OpenTimeout,
		successThreshold: config.SuccessThreshold,
		lastResetTime:    time.Now(),
	}
}

// AllowRequest checks if a request should be allowed
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	// Check if we should transition from open to half-open
	if cb.state == StateOpen && time.Since(cb.lastStateChange) > cb.timeout {
		cb.transitionToHalfOpen()
		return true
	}

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		cb.rejectedRequests++
		return false
	case StateHalfOpen:
		// In half-open, we allow a limited number of requests
		return cb.successCount < cb.halfOpenAttempts
	}

	return false
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalSuccesses++

	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.transitionToClosed()
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalFailures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.threshold {
			cb.transitionToOpen()
		}
	case StateHalfOpen:
		cb.transitionToOpen()
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns current metrics
func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stateName := "closed"
	switch cb.state {
	case StateOpen:
		stateName = "open"
	case StateHalfOpen:
		stateName = "half-open"
	}

	return map[string]interface{}{
		"state":             stateName,
		"failure_count":     cb.failureCount,
		"success_count":     cb.successCount,
		"total_requests":    cb.totalRequests,
		"total_failures":    cb.totalFailures,
		"total_successes":   cb.totalSuccesses,
		"rejected_requests": cb.rejectedRequests,
		"last_failure_time": cb.lastFailureTime,
		"last_state_change": cb.lastStateChange,
		"failure_rate":      safeFailureRate(cb.totalFailures, cb.totalRequests),
	}
}

// Reset resets the circuit breaker to initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()
	cb.lastResetTime = time.Now()
	// Don't reset metrics for observability
}

// safeFailureRate calculates failure rate avoiding division by zero
func safeFailureRate(failures, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(failures) / float64(total) * 100
}

// Private transition methods

func (cb *CircuitBreaker) transitionToOpen() {
	cb.state = StateOpen
	cb.lastStateChange = time.Now()
}

func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.state = StateHalfOpen
	cb.successCount = 0
	cb.lastStateChange = time.Now()
}

func (cb *CircuitBreaker) transitionToClosed() {
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()
}
