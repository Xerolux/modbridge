package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalancingPolicy defines the load balancing strategy
type LoadBalancingPolicy int

const (
	RoundRobin LoadBalancingPolicy = iota
	LeastConnections
	WeightedRoundRobin
	IPHash
	Random
)

// TargetEndpoint represents a backend Modbus device
type TargetEndpoint struct {
	Address       string
	Weight        int
	IsHealthy     bool
	CurrentConns  int32
	TotalRequests int64
	FailCount     int64
	LastCheck     time.Time
	mu            sync.RWMutex
}

// LoadBalancer manages multiple target endpoints
type LoadBalancer struct {
	mu            sync.RWMutex
	endpoints     []*TargetEndpoint
	policy        LoadBalancingPolicy
	currentIndex  uint32
	healthChecker *EndpointHealthChecker
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	running       bool
	config        LoadBalancerConfig
}

// LoadBalancerConfig holds configuration
type LoadBalancerConfig struct {
	Policy              LoadBalancingPolicy
	HealthCheckInterval time.Duration
	UnhealthyThreshold  int // Failures before marking unhealthy
	HealthyThreshold    int // Successes before marking healthy
}

// DefaultLoadBalancerConfig returns sensible defaults
func DefaultLoadBalancerConfig() LoadBalancerConfig {
	return LoadBalancerConfig{
		Policy:              RoundRobin,
		HealthCheckInterval: 30 * time.Second,
		UnhealthyThreshold:  3,
		HealthyThreshold:    2,
	}
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(endpoints []string, config LoadBalancerConfig) (*LoadBalancer, error) {
	if config.HealthCheckInterval <= 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.UnhealthyThreshold <= 0 {
		config.UnhealthyThreshold = 3
	}
	if config.HealthyThreshold <= 0 {
		config.HealthyThreshold = 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	lb := &LoadBalancer{
		policy:    config.Policy,
		ctx:       ctx,
		cancel:    cancel,
		running:   true,
		config:    config,
		endpoints: make([]*TargetEndpoint, 0, len(endpoints)),
	}

	// Initialize endpoints
	for _, addr := range endpoints {
		lb.endpoints = append(lb.endpoints, &TargetEndpoint{
			Address:   addr,
			Weight:    1, // Default weight
			IsHealthy: true,
			LastCheck: time.Now(),
		})
	}

	// Create health checker
	lb.healthChecker = NewEndpointHealthChecker(lb, config)

	// Start health check loop
	lb.wg.Add(1)
	go lb.healthCheckLoop()

	return lb, nil
}

// NextEndpoint returns the next target endpoint based on load balancing policy
func (lb *LoadBalancer) NextEndpoint() (*TargetEndpoint, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Filter healthy endpoints
	healthy := make([]*TargetEndpoint, 0)
	for _, ep := range lb.endpoints {
		ep.mu.RLock()
		if ep.IsHealthy {
			healthy = append(healthy, ep)
		}
		ep.mu.RUnlock()
	}

	if len(healthy) == 0 {
		return nil, fmt.Errorf("no healthy endpoints available")
	}

	var selected *TargetEndpoint

	switch lb.policy {
	case RoundRobin:
		selected = lb.roundRobin(healthy)
	case LeastConnections:
		selected = lb.leastConnections(healthy)
	case WeightedRoundRobin:
		selected = lb.weightedRoundRobin(healthy)
	case IPHash:
		selected = lb.ipHash(healthy)
	case Random:
		selected = lb.random(healthy)
	default:
		selected = lb.roundRobin(healthy)
	}

	if selected != nil {
		atomic.AddInt32(&selected.CurrentConns, 1)
		atomic.AddInt64(&selected.TotalRequests, 1)
	}

	return selected, nil
}

// roundRobin implements round-robin load balancing
func (lb *LoadBalancer) roundRobin(healthy []*TargetEndpoint) *TargetEndpoint {
	if len(healthy) == 0 {
		return nil
	}

	index := atomic.AddUint32(&lb.currentIndex, 1) - 1
	return healthy[index%uint32(len(healthy))]
}

// leastConnections selects endpoint with least active connections
func (lb *LoadBalancer) leastConnections(healthy []*TargetEndpoint) *TargetEndpoint {
	var selected *TargetEndpoint
	minConns := int32(1<<31 - 1) // Max int32

	for _, ep := range healthy {
		conns := atomic.LoadInt32(&ep.CurrentConns)
		if conns < minConns {
			minConns = conns
			selected = ep
		}
	}

	return selected
}

// weightedRoundRobin implements weighted round-robin
func (lb *LoadBalancer) weightedRoundRobin(healthy []*TargetEndpoint) *TargetEndpoint {
	if len(healthy) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0
	for _, ep := range healthy {
		totalWeight += ep.Weight
	}

	// Simple weighted selection (could be improved with smooth weighted RR)
	index := int(atomic.AddUint32(&lb.currentIndex, 1))
	weightSum := 0
	for _, ep := range healthy {
		weightSum += ep.Weight
		if index%totalWeight < weightSum {
			return ep
		}
	}

	return healthy[0]
}

// ipHash implements consistent hashing based on IP
func (lb *LoadBalancer) ipHash(healthy []*TargetEndpoint) *TargetEndpoint {
	// This would use client IP for selection
	// For now, fall back to round-robin
	return lb.roundRobin(healthy)
}

// random implements random selection
func (lb *LoadBalancer) random(healthy []*TargetEndpoint) *TargetEndpoint {
	if len(healthy) == 0 {
		return nil
	}

	index := int(atomic.AddUint32(&lb.currentIndex, 1))
	return healthy[index%len(healthy)]
}

// ReleaseEndpoint releases a connection to an endpoint
func (lb *LoadBalancer) ReleaseEndpoint(endpoint *TargetEndpoint) {
	if endpoint != nil {
		atomic.AddInt32(&endpoint.CurrentConns, -1)
	}
}

// RecordFailure records a failure for an endpoint
func (lb *LoadBalancer) RecordFailure(address string) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for _, ep := range lb.endpoints {
		if ep.Address == address {
			ep.mu.Lock()
			ep.FailCount++
			ep.mu.Unlock()

			lb.healthChecker.RecordFailure(address)
			break
		}
	}
}

// RecordSuccess records a successful request for an endpoint
func (lb *LoadBalancer) RecordSuccess(address string) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for _, ep := range lb.endpoints {
		if ep.Address == address {
			ep.mu.Lock()
			ep.FailCount = 0 // Reset on success
			ep.mu.Unlock()

			lb.healthChecker.RecordSuccess(address)
			break
		}
	}
}

// healthCheckLoop runs periodic health checks
func (lb *LoadBalancer) healthCheckLoop() {
	defer lb.wg.Done()

	ticker := time.NewTicker(lb.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lb.ctx.Done():
			return
		case <-ticker.C:
			lb.healthChecker.CheckAll()
		}
	}
}

// GetStats returns load balancer statistics
func (lb *LoadBalancer) GetStats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	stats := make(map[string]interface{})
	endpointStats := make([]map[string]interface{}, 0, len(lb.endpoints))

	totalRequests := int64(0)
	totalActive := int32(0)
	healthyCount := 0

	for _, ep := range lb.endpoints {
		ep.mu.RLock()
		endpointStats = append(endpointStats, map[string]interface{}{
			"address":        ep.Address,
			"is_healthy":     ep.IsHealthy,
			"current_conns":  atomic.LoadInt32(&ep.CurrentConns),
			"total_requests": atomic.LoadInt64(&ep.TotalRequests),
			"fail_count":     atomic.LoadInt64(&ep.FailCount),
			"weight":         ep.Weight,
			"last_check":     ep.LastCheck,
		})
		totalRequests += atomic.LoadInt64(&ep.TotalRequests)
		totalActive += atomic.LoadInt32(&ep.CurrentConns)
		if ep.IsHealthy {
			healthyCount++
		}
		ep.mu.RUnlock()
	}

	stats["policy"] = lb.policy
	stats["total_endpoints"] = len(lb.endpoints)
	stats["healthy_endpoints"] = healthyCount
	stats["total_requests"] = totalRequests
	stats["total_active"] = totalActive
	stats["endpoints"] = endpointStats

	return stats
}

// Stop stops the load balancer
func (lb *LoadBalancer) Stop() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if !lb.running {
		return
	}

	lb.running = false
	lb.cancel()
	lb.wg.Wait()
}

// EndpointHealthChecker manages health checks for endpoints
type EndpointHealthChecker struct {
	lb            *LoadBalancer
	mu            sync.RWMutex
	failCounts    map[string]int
	successCounts map[string]int
	config        LoadBalancerConfig
}

// NewEndpointHealthChecker creates a new health checker
func NewEndpointHealthChecker(lb *LoadBalancer, config LoadBalancerConfig) *EndpointHealthChecker {
	return &EndpointHealthChecker{
		lb:            lb,
		failCounts:    make(map[string]int),
		successCounts: make(map[string]int),
		config:        config,
	}
}

// RecordFailure records a failure for an endpoint
func (hc *EndpointHealthChecker) RecordFailure(address string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.failCounts[address]++
	hc.successCounts[address] = 0

	// Check if endpoint should be marked unhealthy
	if hc.failCounts[address] >= hc.config.UnhealthyThreshold {
		hc.markUnhealthy(address)
	}
}

// RecordSuccess records a success for an endpoint
func (hc *EndpointHealthChecker) RecordSuccess(address string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.successCounts[address]++

	// Check if endpoint should be marked healthy
	if hc.successCounts[address] >= hc.config.HealthyThreshold {
		hc.markHealthy(address)
	}
}

// CheckAll performs health checks on all endpoints
func (hc *EndpointHealthChecker) CheckAll() {
	hc.lb.mu.RLock()
	defer hc.lb.mu.RUnlock()

	for _, ep := range hc.lb.endpoints {
		// Try to connect to endpoint
		conn, err := net.DialTimeout("tcp", ep.Address, 5*time.Second)
		if err != nil {
			hc.RecordFailure(ep.Address)
		} else {
			conn.Close()
			hc.RecordSuccess(ep.Address)
		}
	}
}

// markUnhealthy marks an endpoint as unhealthy
func (hc *EndpointHealthChecker) markUnhealthy(address string) {
	hc.lb.mu.RLock()
	defer hc.lb.mu.RUnlock()

	for _, ep := range hc.lb.endpoints {
		if ep.Address == address {
			ep.mu.Lock()
			ep.IsHealthy = false
			ep.LastCheck = time.Now()
			ep.mu.Unlock()
			break
		}
	}
}

// markHealthy marks an endpoint as healthy
func (hc *EndpointHealthChecker) markHealthy(address string) {
	hc.lb.mu.RLock()
	defer hc.lb.mu.RUnlock()

	for _, ep := range hc.lb.endpoints {
		if ep.Address == address {
			ep.mu.Lock()
			ep.IsHealthy = true
			ep.LastCheck = time.Now()
			ep.FailCount = 0
			ep.mu.Unlock()
			break
		}
	}
}
