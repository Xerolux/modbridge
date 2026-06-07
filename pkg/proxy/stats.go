// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"sort"
	"sync"
	"time"
)

// EnhancedStats provides detailed performance metrics
type EnhancedStats struct {
	requests     int64
	errors       int64
	bytesRead    int64
	bytesWritten int64

	mu                sync.RWMutex
	latencies         []time.Duration
	latencyBuf        []time.Duration
	latencyWriteIdx   int
	latencyCount      int
	maxLatency        time.Duration
	minLatency        time.Duration
	totalLatency      time.Duration
	lastRequestTime   time.Time
	requestStartTimes map[int64]time.Time
	requestsResetTime time.Time

	activeConnections int
	maxConns          int
	totalConnections  int64

	latencyWindow  int
	requestsWindow time.Duration
	totalRequests  int64
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

	now := time.Now()
	return &EnhancedStats{
		latencyBuf:        make([]time.Duration, latencyWindow),
		latencyWriteIdx:   0,
		latencyCount:      0,
		maxLatency:        0,
		minLatency:        0,
		latencyWindow:     latencyWindow,
		requestStartTimes: make(map[int64]time.Time),
		lastRequestTime:   now,
		requestsResetTime: now,
		requestsWindow:    60 * time.Minute,
	}
}

// staleRequestAge is the maximum duration a request start entry is kept
// without a corresponding completion record. Entries older than this are
// evicted during the next RecordRequestStart call to prevent map growth
// caused by goroutines that exit without calling RecordRequestComplete.
const staleRequestAge = 5 * time.Minute

// RecordRequestStart records the start of a request.
func (s *EnhancedStats) RecordRequestStart(requestID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Evict stale entries (requests that never completed) to prevent map growth.
	now := time.Now()
	for id, t := range s.requestStartTimes {
		if now.Sub(t) > staleRequestAge {
			delete(s.requestStartTimes, id)
			s.activeConnections-- // were counted as active; undo
		}
	}

	s.requestStartTimes[requestID] = now
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

	// Check if we need to reset the request counter window
	s.checkRequestWindow()

	// Update counters
	if err != nil {
		s.errors++
	} else {
		s.requests++
		s.totalRequests++ // Track total ever
	}

	s.bytesRead += int64(bytesRead)
	s.bytesWritten += int64(bytesWritten)
	s.activeConnections--

	// Update latency metrics
	s.recordLatency(latency)
}

// recordLatency records a latency measurement
func (s *EnhancedStats) recordLatency(latency time.Duration) {
	if s.latencyCount < s.latencyWindow {
		s.latencyBuf[s.latencyWriteIdx] = latency
		s.latencyCount++
	} else {
		s.latencyBuf[s.latencyWriteIdx] = latency
	}
	s.latencyWriteIdx = (s.latencyWriteIdx + 1) % s.latencyWindow

	if s.minLatency == 0 || latency < s.minLatency {
		s.minLatency = latency
	}
	if latency > s.maxLatency {
		s.maxLatency = latency
	}

	s.totalLatency += latency
}

// checkRequestWindow checks if the request window has expired and resets if needed
// Caller must hold lock
func (s *EnhancedStats) checkRequestWindow() {
	if time.Since(s.requestsResetTime) > s.requestsWindow {
		// Reset request counter for new window
		s.requests = 0
		s.requestsResetTime = time.Now()
	}
}

// getPercentilesLocked calculates latency percentiles (caller must hold at least RLock)
func (s *EnhancedStats) getPercentilesLocked() LatencyPercentiles {
	n := s.latencyCount
	if n == 0 {
		return LatencyPercentiles{}
	}

	sorted := make([]time.Duration, n)
	copy(sorted, s.latencyBuf[:n])
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	p999Idx := n * 999 / 1000
	if p999Idx >= n {
		p999Idx = n - 1
	}
	p99Idx := n * 99 / 100
	if p99Idx >= n {
		p99Idx = n - 1
	}
	p95Idx := n * 95 / 100
	if p95Idx >= n {
		p95Idx = n - 1
	}

	return LatencyPercentiles{
		P50:  sorted[n*50/100],
		P95:  sorted[p95Idx],
		P99:  sorted[p99Idx],
		P999: sorted[p999Idx],
		Mean: s.totalLatency / time.Duration(n),
		Min:  s.minLatency,
		Max:  s.maxLatency,
	}
}

// GetPercentiles calculates latency percentiles
func (s *EnhancedStats) GetPercentiles() LatencyPercentiles {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getPercentilesLocked()
}

// GetSnapshot returns a snapshot of current stats
func (s *EnhancedStats) GetSnapshot() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we need to reset the request counter window
	s.checkRequestWindow()

	percentiles := s.getPercentilesLocked()

	total := s.requests + s.errors
	errorRate := 0.0
	if total > 0 {
		errorRate = float64(s.errors) / float64(total) * 100
	}

	requestsPerSecond := 0.0
	if !s.lastRequestTime.IsZero() && s.requests > 0 {
		duration := time.Since(s.lastRequestTime).Seconds()
		if duration > 0 {
			requestsPerSecond = float64(s.requests) / duration
		}
	}

	return map[string]interface{}{
		"requests":              s.requests,
		"total_requests":        s.totalRequests,
		"errors":                s.errors,
		"error_rate":            errorRate,
		"active_connections":    s.activeConnections,
		"max_connections":       s.maxConns,
		"total_connections":     s.totalConnections,
		"bytes_read":            s.bytesRead,
		"bytes_written":         s.bytesWritten,
		"requests_per_sec":      requestsPerSecond,
		"latency":               percentiles,
		"requests_window_reset": s.requestsResetTime,
	}
}

// GetThroughput calculates current throughput
func (s *EnhancedStats) GetThroughput(window time.Duration) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latencyCount == 0 || window <= 0 {
		return 0
	}

	return float64(s.latencyCount) / window.Seconds()
}
