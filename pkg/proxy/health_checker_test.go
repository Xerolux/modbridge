// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// deadTarget returns an address whose dials are refused: a listener that has
// been closed again.
func deadTarget(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

// liveTarget returns an address whose dials succeed for the test's duration.
func liveTarget(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { l.Close() })
	return l.Addr().String()
}

// waitUntil polls cond until it holds or the deadline passes.
func waitUntil(t *testing.T, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

// TestHealthCheckerDebouncesFailures verifies that a single failed probe does
// not flip the verdict: the checker needs failureThreshold consecutive
// failures before it reports unhealthy and fires onUnhealthy.
func TestHealthCheckerDebouncesFailures(t *testing.T) {
	hc := NewHealthChecker(deadTarget(t), time.Hour, time.Second, nil)

	var unhealthy atomic.Int64
	hc.SetOnUnhealthy(func() { unhealthy.Add(1) })

	hc.check()
	hc.check()
	if !hc.IsHealthy() {
		t.Fatal("checker flipped unhealthy before the failure threshold")
	}

	hc.check()
	if hc.IsHealthy() {
		t.Fatal("checker still healthy after the failure threshold")
	}
	if !waitUntil(t, func() bool { return unhealthy.Load() == 1 }) {
		t.Error("onUnhealthy did not fire on the transition")
	}

	// A later failure must not fire the callback again while already unhealthy.
	hc.check()
	time.Sleep(50 * time.Millisecond)
	if got := unhealthy.Load(); got != 1 {
		t.Errorf("onUnhealthy fired %d times for one outage, want 1", got)
	}
}

// TestHealthCheckerRequiresSustainedRecovery verifies that recovery needs
// successThreshold consecutive successful dials: one successful dial after an
// outage keeps the checker unhealthy (a rebooting device accepts TCP before
// its application answers).
func TestHealthCheckerRequiresSustainedRecovery(t *testing.T) {
	hc := NewHealthChecker(deadTarget(t), time.Hour, time.Second, nil)

	var recovered atomic.Int64
	hc.SetOnRecovery(func() { recovered.Add(1) })

	for i := 0; i < defaultFailureThreshold; i++ {
		hc.check()
	}
	if hc.IsHealthy() {
		t.Fatal("precondition: checker should be unhealthy")
	}

	hc.target = liveTarget(t)
	hc.check()
	if hc.IsHealthy() {
		t.Fatal("checker declared recovery after a single successful dial")
	}
	if got := recovered.Load(); got != 0 {
		t.Errorf("onRecovery fired %d times before the success threshold, want 0", got)
	}

	hc.check()
	if !hc.IsHealthy() {
		t.Fatal("checker still unhealthy after the success threshold")
	}
	if !waitUntil(t, func() bool { return recovered.Load() == 1 }) {
		t.Error("onRecovery did not fire on the transition")
	}
}
