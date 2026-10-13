// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"testing"
	"time"
)

// forceOpen drives a closed breaker over its failure threshold.
func forceOpen(cb *CircuitBreaker) {
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
}

// TestCircuitBreakerHalfOpenAttempts verifies the configured probe budget:
// half-open admits up to HalfOpenAttempts concurrent probes and rejects the
// rest until probes are recorded.
func TestCircuitBreakerHalfOpenAttempts(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 5,
		HalfOpenAttempts: 2,
		OpenTimeout:      5 * time.Millisecond,
		SuccessThreshold: 2,
	})

	forceOpen(cb)
	// The adaptive backoff doubles the open window once (openCount = 1), so
	// 30 ms safely clears the effective 10 ms.
	time.Sleep(30 * time.Millisecond)

	if !cb.AllowRequest() {
		t.Fatal("first half-open probe was rejected")
	}
	if !cb.AllowRequest() {
		t.Fatal("second half-open probe was rejected")
	}
	if cb.AllowRequest() {
		t.Fatal("third probe admitted although halfOpenAttempts is 2")
	}

	// Recording the outstanding probes frees the budget again.
	cb.RecordSuccess()
	if !cb.AllowRequest() {
		t.Fatal("probe budget not returned after a recorded probe")
	}
}

// TestCircuitBreakerOnTargetReachable verifies that renewed TCP reachability
// moves an open breaker to half-open (probe with the next real request) and
// forgives the adaptive backoff, but never closes the breaker on its own.
func TestCircuitBreakerOnTargetReachable(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		OpenTimeout:      time.Hour, // long: only OnTargetReachable may end the open state
	})

	forceOpen(cb)
	if cb.GetState() != StateOpen {
		t.Fatal("precondition: breaker should be open")
	}

	cb.OnTargetReachable()

	if cb.GetState() != StateHalfOpen {
		t.Fatalf("breaker should be half-open after target reachability, got %v", cb.GetState())
	}
	if got := cb.GetMetrics()["open_count"]; got != int64(0) {
		t.Fatalf("adaptive backoff not forgiven: open_count = %v", got)
	}

	// Reaching the target again while half-open must not close the breaker.
	cb.OnTargetReachable()
	if cb.GetState() != StateHalfOpen {
		t.Fatalf("reachability must not close the breaker directly, got %v", cb.GetState())
	}

	// A closed breaker stays untouched.
	cb2 := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	cb2.OnTargetReachable()
	if cb2.GetState() != StateClosed {
		t.Fatalf("closed breaker should stay closed, got %v", cb2.GetState())
	}
}

// TestCircuitBreakerClosesAfterSustainedHalfOpenSuccess verifies the full
// recovery: successThreshold successful probes close the breaker.
func TestCircuitBreakerClosesAfterSustainedHalfOpenSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		OpenTimeout:      20 * time.Millisecond,
		SuccessThreshold: 2,
	})

	forceOpen(cb)
	cb.OnTargetReachable()

	cb.AllowRequest()
	cb.RecordSuccess()
	if cb.GetState() != StateHalfOpen {
		t.Fatalf("one success must not close the breaker, got %v", cb.GetState())
	}

	cb.AllowRequest()
	cb.RecordSuccess()
	if cb.GetState() != StateClosed {
		t.Fatalf("breaker should be closed after sustained success, got %v", cb.GetState())
	}
}
