package proxy

import (
	"sync"
	"time"
)

// EnhancedStats provides detailed performance metrics
type EnhancedStats struct {
	// Basic counters (atomic for lock-free access)
	requests     int64
	errors       int64
	bytesRead    int64
	bytesWritten int64

	// Timing metrics
	mu                sync.RWMutex
	latencies         []time.Duration // Sliding window of latencies
	maxLatency        time.Duration
	minLatency        time.Duration
	totalLatency      time.Duration
	lastRequestTime   time.Time
	requestStartTimes map[int64]time.Time // Track request start times

	// Connection stats
	activeConnections int
	maxConns          int
	totalConnections  int64

	// Configuration
	latencyWindow int // Number of latencies to track
}

// LatencyPercentiles returns latency statistics
type LatencyPercentiles struct {
	P50  time.Duration `json:"p50"`
	P95  time.Duration `json:"p95"`
	P99  time.Duration `json:"p99"`
	P999 time.Duration `json:"p999"`
	Mean time.Duration `json:"mean"`
	Min  time.Duration `json:"min"`
	Max  time.Duration `json:"max"`
}

// NewEnhancedStats creates a new enhanced stats tracker
func NewEnhancedStats(latencyWindow int) *EnhancedStats {
	if latencyWindow <= 0 {
		latencyWindow = 1000 // Track last 1000 requests
	}

	return &EnhancedStats{
		latencies:         make([]time.Duration, 0, latencyWindow),
		maxLatency:        0,
		minLatency:        0,
		latencyWindow:     latencyWindow,
		requestStartTimes: make(map[int64]time.Time),
		lastRequestTime:   time.Now(),
	}
}

// RecordRequestStart records the start of a request
func (s *EnhancedStats) RecordRequestStart(requestID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requestStartTimes[requestID] = time.Now()
	s.activeConnections++
	s.totalConnections++

	if s.activeConnections > s.maxConns {
		s.maxConns = s.activeConnections
	}
}

// RecordRequestComplete records the completion of a request
func (s *EnhancedStats) RecordRequestComplete(requestID int64, bytesRead, bytesWritten int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime, exists := s.requestStartTimes[requestID]
	if !exists {
		return // Request not tracked
	}
	delete(s.requestStartTimes, requestID)

	latency := time.Since(startTime)
	s.lastRequestTime = time.Now()

	// Update counters
	if err != nil {
		s.errors++
	} else {
		s.requests++
	}

	s.bytesRead += int64(bytesRead)
	s.bytesWritten += int64(bytesWritten)
	s.activeConnections--

	// Update latency metrics
	s.recordLatency(latency)
}

// recordLatency records a latency measurement
func (s *EnhancedStats) recordLatency(latency time.Duration) {
	// Add to sliding window
	if len(s.latencies) >= s.latencyWindow {
		// Remove oldest
		s.latencies = s.latencies[1:]
	}
	s.latencies = append(s.latencies, latency)

	// Update min/max
	if s.minLatency == 0 || latency < s.minLatency {
		s.minLatency = latency
	}
	if latency > s.maxLatency {
		s.maxLatency = latency
	}

	s.totalLatency += latency
}

// GetPercentiles calculates latency percentiles
func (s *EnhancedStats) GetPercentiles() LatencyPercentiles {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.latencies) == 0 {
		return LatencyPercentiles{}
	}

	// Create sorted copy
	sorted := make([]time.Duration, len(s.latencies))
	copy(sorted, s.latencies)

	// Simple sort (for small arrays, more efficient than full quicksort)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return LatencyPercentiles{
		P50:  sorted[len(sorted)*50/100],
		P95:  sorted[len(sorted)*95/100],
		P99:  sorted[len(sorted)*99/100],
		P999: sorted[len(sorted)*999/1000],
		Mean: s.totalLatency / time.Duration(len(s.latencies)),
		Min:  s.minLatency,
		Max:  s.maxLatency,
	}
}

// GetSnapshot returns a snapshot of current stats
func (s *EnhancedStats) GetSnapshot() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	percentiles := s.GetPercentiles()
	requestsPerSecond := 0.0
	if !s.lastRequestTime.IsZero() {
		duration := time.Since(s.lastRequestTime).Seconds()
		if duration > 0 {
			requestsPerSecond = float64(s.requests) / duration
		}
	}

	return map[string]interface{}{
		"requests":           s.requests,
		"errors":             s.errors,
		"error_rate":         float64(s.errors) / float64(s.requests+s.errors) * 100,
		"active_connections": s.activeConnections,
		"max_connections":    s.maxConns,
		"total_connections":  s.totalConnections,
		"bytes_read":         s.bytesRead,
		"bytes_written":      s.bytesWritten,
		"requests_per_sec":   requestsPerSecond,
		"latency":            percentiles,
	}
}

// GetThroughput calculates current throughput
func (s *EnhancedStats) GetThroughput(window time.Duration) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.latencies) == 0 {
		return 0
	}

	// Simple approximation based on recent requests
	// In a production system, we'd track actual timestamps
	count := len(s.latencies)
	return float64(count) / window.Seconds()
}
