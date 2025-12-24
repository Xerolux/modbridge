package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed allows requests to pass through.
	StateClosed State = iota
	// StateOpen blocks all requests.
	StateOpen
	// StateHalfOpen allows a limited number of requests to test if the service recovered.
	StateHalfOpen
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned when too many requests are sent in half-open state.
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// Config holds circuit breaker configuration.
type Config struct {
	// MaxFailures is the maximum number of failures before opening the circuit.
	MaxFailures int

	// Timeout is the duration to wait before transitioning from open to half-open.
	Timeout time.Duration

	// MaxRequests is the maximum number of requests allowed in half-open state.
	MaxRequests int

	// OnStateChange is called when the state changes (optional).
	OnStateChange func(from, to State)
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config Config

	mu              sync.RWMutex
	state           State
	failures        int
	successes       int
	requests        int
	lastFailureTime time.Time
	lastStateChange time.Time
}

// New creates a new circuit breaker.
func New(config Config) *CircuitBreaker {
	// Set defaults
	if config.MaxFailures == 0 {
		config.MaxFailures = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxRequests == 0 {
		config.MaxRequests = 1
	}

	return &CircuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Execute executes the given function if the circuit breaker allows it.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if we can proceed
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if the request can proceed.
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.setState(StateHalfOpen)
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Limit the number of requests in half-open state
		if cb.requests >= cb.config.MaxRequests {
			return ErrTooManyRequests
		}
		cb.requests++
		return nil

	default:
		return ErrCircuitOpen
	}
}

// afterRequest records the result of a request.
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure handles a failed request.
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.MaxFailures {
			cb.setState(StateOpen)
		}

	case StateHalfOpen:
		// Any failure in half-open state opens the circuit
		cb.setState(StateOpen)
	}
}

// onSuccess handles a successful request.
func (cb *CircuitBreaker) onSuccess() {
	cb.successes++

	switch cb.state {
	case StateHalfOpen:
		// Enough successful requests close the circuit
		if cb.successes >= cb.config.MaxRequests {
			cb.setState(StateClosed)
		}

	case StateClosed:
		// Reset failure count on success
		cb.failures = 0
	}
}

// setState transitions the circuit breaker to a new state.
func (cb *CircuitBreaker) setState(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Reset counters
	cb.failures = 0
	cb.successes = 0
	cb.requests = 0

	// Call state change callback
	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(oldState, newState)
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns statistics about the circuit breaker.
func (cb *CircuitBreaker) Stats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		State:           cb.state,
		Failures:        cb.failures,
		Successes:       cb.successes,
		Requests:        cb.requests,
		LastFailure:     cb.lastFailureTime,
		LastStateChange: cb.lastStateChange,
	}
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
}

// Stats holds circuit breaker statistics.
type Stats struct {
	State           State
	Failures        int
	Successes       int
	Requests        int
	LastFailure     time.Time
	LastStateChange time.Time
}
