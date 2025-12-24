package routing

import (
	"context"
	"errors"
	"fmt"
	"modbusproxy/pkg/circuitbreaker"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Strategy defines the routing strategy.
type Strategy string

const (
	// StrategyRoundRobin distributes requests in round-robin fashion.
	StrategyRoundRobin Strategy = "round-robin"
	// StrategyLeastConnections routes to the target with fewest connections.
	StrategyLeastConnections Strategy = "least-connections"
	// StrategyRandom routes randomly.
	StrategyRandom Strategy = "random"
	// StrategyPriority uses priority order with failover.
	StrategyPriority Strategy = "priority"
)

var (
	// ErrNoTargetsAvailable is returned when no targets are available.
	ErrNoTargetsAvailable = errors.New("no targets available")
)

// Target represents a routing target.
type Target struct {
	Address         string
	Priority        int // Lower is higher priority
	Weight          int // For weighted routing
	MaxConnections  int
	HealthCheckInterval time.Duration

	// Internal state
	connections    atomic.Int32
	circuitBreaker *circuitbreaker.CircuitBreaker
	healthy        atomic.Bool
	lastHealthy    time.Time
	mu             sync.RWMutex
}

// NewTarget creates a new routing target.
func NewTarget(address string, priority, weight int) *Target {
	target := &Target{
		Address:             address,
		Priority:            priority,
		Weight:              weight,
		MaxConnections:      100,
		HealthCheckInterval: 30 * time.Second,
	}

	// Initialize circuit breaker
	target.circuitBreaker = circuitbreaker.New(circuitbreaker.Config{
		MaxFailures: 5,
		Timeout:     60 * time.Second,
		MaxRequests: 3,
		OnStateChange: func(from, to circuitbreaker.State) {
			if to == circuitbreaker.StateOpen {
				target.healthy.Store(false)
			} else if to == circuitbreaker.StateClosed {
				target.healthy.Store(true)
			}
		},
	})

	target.healthy.Store(true)
	target.lastHealthy = time.Now()

	return target
}

// IsHealthy returns true if the target is healthy.
func (t *Target) IsHealthy() bool {
	return t.healthy.Load() && t.circuitBreaker.State() != circuitbreaker.StateOpen
}

// IsAvailable returns true if the target can accept more connections.
func (t *Target) IsAvailable() bool {
	if !t.IsHealthy() {
		return false
	}

	if t.MaxConnections > 0 {
		return int(t.connections.Load()) < t.MaxConnections
	}

	return true
}

// IncrementConnections increments the connection count.
func (t *Target) IncrementConnections() {
	t.connections.Add(1)
}

// DecrementConnections decrements the connection count.
func (t *Target) DecrementConnections() {
	t.connections.Add(-1)
}

// GetConnections returns the current connection count.
func (t *Target) GetConnections() int32 {
	return t.connections.Load()
}

// Execute executes a function with circuit breaker protection.
func (t *Target) Execute(fn func() error) error {
	return t.circuitBreaker.Execute(fn)
}

// Router manages routing to multiple targets.
type Router struct {
	mu       sync.RWMutex
	targets  []*Target
	strategy Strategy
	index    atomic.Uint32 // For round-robin
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewRouter creates a new router.
func NewRouter(strategy Strategy) *Router {
	ctx, cancel := context.WithCancel(context.Background())

	router := &Router{
		targets:  make([]*Target, 0),
		strategy: strategy,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Start health check routine
	go router.healthCheckLoop()

	return router
}

// AddTarget adds a target to the router.
func (r *Router) AddTarget(target *Target) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.targets = append(r.targets, target)
}

// RemoveTarget removes a target from the router.
func (r *Router) RemoveTarget(address string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, target := range r.targets {
		if target.Address == address {
			r.targets = append(r.targets[:i], r.targets[i+1:]...)
			break
		}
	}
}

// GetTarget selects a target based on the routing strategy.
func (r *Router) GetTarget() (*Target, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.targets) == 0 {
		return nil, ErrNoTargetsAvailable
	}

	// Filter available targets
	available := make([]*Target, 0, len(r.targets))
	for _, target := range r.targets {
		if target.IsAvailable() {
			available = append(available, target)
		}
	}

	if len(available) == 0 {
		return nil, ErrNoTargetsAvailable
	}

	switch r.strategy {
	case StrategyRoundRobin:
		return r.roundRobin(available), nil

	case StrategyLeastConnections:
		return r.leastConnections(available), nil

	case StrategyPriority:
		return r.priority(available), nil

	default:
		return available[0], nil
	}
}

// roundRobin selects the next target in round-robin fashion.
func (r *Router) roundRobin(targets []*Target) *Target {
	index := r.index.Add(1) % uint32(len(targets))
	return targets[index]
}

// leastConnections selects the target with the fewest connections.
func (r *Router) leastConnections(targets []*Target) *Target {
	var selected *Target
	minConnections := int32(^uint32(0) >> 1) // Max int32

	for _, target := range targets {
		connections := target.GetConnections()
		if connections < minConnections {
			minConnections = connections
			selected = target
		}
	}

	return selected
}

// priority selects the highest priority (lowest number) available target.
func (r *Router) priority(targets []*Target) *Target {
	var selected *Target
	highestPriority := int(^uint(0) >> 1) // Max int

	for _, target := range targets {
		if target.Priority < highestPriority {
			highestPriority = target.Priority
			selected = target
		}
	}

	return selected
}

// healthCheckLoop periodically checks target health.
func (r *Router) healthCheckLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return

		case <-ticker.C:
			r.performHealthChecks()
		}
	}
}

// performHealthChecks checks the health of all targets.
func (r *Router) performHealthChecks() {
	r.mu.RLock()
	targets := make([]*Target, len(r.targets))
	copy(targets, r.targets)
	r.mu.RUnlock()

	for _, target := range targets {
		go r.checkTargetHealth(target)
	}
}

// checkTargetHealth checks if a target is reachable.
func (r *Router) checkTargetHealth(target *Target) {
	conn, err := net.DialTimeout("tcp", target.Address, 5*time.Second)
	if err != nil {
		target.healthy.Store(false)
		return
	}

	_ = conn.Close()

	target.healthy.Store(true)
	target.mu.Lock()
	target.lastHealthy = time.Now()
	target.mu.Unlock()
}

// GetStats returns statistics for all targets.
func (r *Router) GetStats() []TargetStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make([]TargetStats, len(r.targets))
	for i, target := range r.targets {
		stats[i] = TargetStats{
			Address:         target.Address,
			Priority:        target.Priority,
			Connections:     target.GetConnections(),
			Healthy:         target.IsHealthy(),
			CircuitState:    target.circuitBreaker.State().String(),
		}
	}

	return stats
}

// Close closes the router and stops health checks.
func (r *Router) Close() {
	r.cancel()
}

// TargetStats holds statistics for a target.
type TargetStats struct {
	Address      string
	Priority     int
	Connections  int32
	Healthy      bool
	CircuitState string
}

// Dial creates a connection to a selected target.
func (r *Router) Dial(ctx context.Context) (net.Conn, *Target, error) {
	target, err := r.GetTarget()
	if err != nil {
		return nil, nil, err
	}

	var conn net.Conn
	err = target.Execute(func() error {
		var dialErr error
		conn, dialErr = net.DialTimeout("tcp", target.Address, 5*time.Second)
		return dialErr
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to %s: %w", target.Address, err)
	}

	target.IncrementConnections()
	return conn, target, nil
}
