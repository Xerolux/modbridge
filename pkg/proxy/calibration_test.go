// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"modbridge/pkg/modbus"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// pickyTarget answers reads, but drops a request when it arrives sooner than
// minGap after the previous one, and refuses to serve more than maxSessions
// connections at a time — the two behaviours calibration is meant to find.
type pickyTarget struct {
	listener    net.Listener
	minGap      time.Duration
	maxSessions int32
	latency     time.Duration

	mu       sync.Mutex
	lastSeen time.Time
	sessions atomic.Int32
}

func newPickyTarget(t *testing.T, minGap time.Duration, maxSessions int, latency time.Duration) *pickyTarget {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock target: %v", err)
	}

	target := &pickyTarget{
		listener:    listener,
		minGap:      minGap,
		maxSessions: int32(maxSessions),
		latency:     latency,
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go target.serve(conn)
		}
	}()

	t.Cleanup(func() { listener.Close() })
	return target
}

func (pt *pickyTarget) serve(conn net.Conn) {
	defer conn.Close()

	over := pt.sessions.Add(1) > pt.maxSessions
	defer pt.sessions.Add(-1)

	for {
		frame, err := modbus.ReadFrame(conn)
		if err != nil {
			return
		}
		if over {
			continue // accepted the socket, answers nothing — the usual failure mode
		}

		pt.mu.Lock()
		tooSoon := !pt.lastSeen.IsZero() && time.Since(pt.lastSeen) < pt.minGap
		pt.lastSeen = time.Now()
		pt.mu.Unlock()
		if tooSoon {
			continue // request dropped on the floor, exactly like the real thing
		}

		txID, unitID, fc, _, quantity, perr := modbus.ParseReadRequest(frame)
		if perr != nil {
			return
		}
		if pt.latency > 0 {
			time.Sleep(pt.latency)
		}
		resp, _ := modbus.CreateReadResponse(txID, unitID, fc, make([]byte, quantity*2))
		if _, err := conn.Write(resp); err != nil {
			return
		}
	}
}

func (pt *pickyTarget) addr() string { return pt.listener.Addr().String() }

func calibrationProxy(t *testing.T, targetAddr string) *ProxyInstance {
	t.Helper()
	p := startTestProxy(t, targetAddr, func(p *ProxyInstance) {
		p.ReadTimeout = time.Second // keep a dropped request cheap to detect
	})
	t.Cleanup(p.Stop)
	return p
}

// TestCalibrationFindsTheGapTheDeviceNeeds is the core claim: the run must land
// on a spacing the device tolerates, not on the fastest one attempted.
func TestCalibrationFindsTheGapTheDeviceNeeds(t *testing.T) {
	target := newPickyTarget(t, 60*time.Millisecond, 4, 5*time.Millisecond)
	p := calibrationProxy(t, target.addr())

	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:            ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep:  6,
		GapStepsMs:       []int{200, 100, 50, 25},
		ConnectionLevels: []int{1},
	})
	if err != nil {
		t.Fatalf("calibration failed: %v", err)
	}

	// 200 and 100 ms clear the device's 60 ms floor, 50 ms does not.
	if len(result.GapSteps) < 3 {
		t.Fatalf("expected the run to reach the failing step, got %d steps", len(result.GapSteps))
	}
	for _, step := range result.GapSteps[:2] {
		if step.Errors != 0 {
			t.Errorf("%d ms spacing reported %d errors, but the device tolerates it", step.GapMs, step.Errors)
		}
	}
	failing := result.GapSteps[2]
	if failing.GapMs != 50 || failing.Errors == 0 {
		t.Errorf("expected 50 ms spacing to fail, got %+v", failing)
	}

	// The recommendation must clear the device's floor, with margin.
	if result.Recommended.MinRequestGapMs < 60 {
		t.Errorf("recommended %d ms, below the 60 ms the device needs", result.Recommended.MinRequestGapMs)
	}
	if result.Recommended.MinRequestGapMs > 200 {
		t.Errorf("recommended %d ms, more careful than necessary", result.Recommended.MinRequestGapMs)
	}
	if len(result.Notes) == 0 {
		t.Error("a run that found a limit should say so in its notes")
	}
}

// TestCalibrationFindsSessionLimit covers the second measurement: devices that
// accept extra sockets and then answer nothing on them.
func TestCalibrationFindsSessionLimit(t *testing.T) {
	target := newPickyTarget(t, 0, 1, 5*time.Millisecond)
	p := calibrationProxy(t, target.addr())

	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:            ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep:  4,
		GapStepsMs:       []int{50},
		ConnectionLevels: []int{1, 2},
	})
	if err != nil {
		t.Fatalf("calibration failed: %v", err)
	}

	if result.Recommended.MaxTargetConns != 1 {
		t.Errorf("recommended %d connections against a single-session device", result.Recommended.MaxTargetConns)
	}
	if len(result.ConnectionSteps) < 2 || result.ConnectionSteps[1].Errors == 0 {
		t.Errorf("the second session should have failed, got %+v", result.ConnectionSteps)
	}
}

// TestCalibrationRefusesWhileClientsConnected guards the promise that a run
// never quietly competes with live traffic.
func TestCalibrationRefusesWhileClientsConnected(t *testing.T) {
	target := newPickyTarget(t, 0, 4, 0)
	p := calibrationProxy(t, target.addr())

	client, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect a client: %v", err)
	}
	defer client.Close()

	deadline := time.Now().Add(2 * time.Second)
	for p.Stats.ActiveConns.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	_, err = p.Calibrate(t.Context(), CalibrationConfig{
		Probe:           ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep: 2,
		GapStepsMs:      []int{50},
	})
	if err == nil {
		t.Fatal("calibration ran while a client was connected")
	}
}

// TestCalibrationUsesLastObservedRead verifies a run probes a register the
// client already asks for rather than inventing an address.
func TestCalibrationUsesLastObservedRead(t *testing.T) {
	target := newPickyTarget(t, 0, 4, 0)
	p := calibrationProxy(t, target.addr())

	client, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if _, err := client.Write(modbus.CreateReadRequest(1, 9, 4, 4711, 3)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := client.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	if _, err := modbus.ReadFrame(client); err != nil {
		t.Fatalf("read failed: %v", err)
	}
	client.Close()

	probe, ok := p.LastObservedRead()
	if !ok {
		t.Fatal("the proxy did not remember the client's read")
	}
	if probe.UnitID != 9 || probe.Function != 4 || probe.Address != 4711 || probe.Quantity != 3 {
		t.Errorf("remembered %+v, want unit 9 fc 4 addr 4711 qty 3", probe)
	}
}

func TestWithSafetyMargin(t *testing.T) {
	tests := []struct {
		gap  int
		want int
	}{
		{10, 25},   // 15 rounds up to the next sane value
		{50, 100},  // 75 -> 100
		{100, 150}, // exactly 150
		{200, 500}, // 300 -> 500
	}
	for _, tt := range tests {
		if got := withSafetyMargin(tt.gap, 1.5); got != tt.want {
			t.Errorf("withSafetyMargin(%d) = %d, want %d", tt.gap, got, tt.want)
		}
	}
}

func TestReadTimeoutFrom(t *testing.T) {
	if got := readTimeoutFrom([]GapStep{{P95Ms: 40}}); got != 2 {
		t.Errorf("fast device: got %ds, want the 2s floor", got)
	}
	if got := readTimeoutFrom([]GapStep{{P95Ms: 1200}, {P95Ms: 900}}); got != 5 {
		t.Errorf("slow device: got %ds, want 5s (1.2s p95 with headroom)", got)
	}
	if got := readTimeoutFrom(nil); got != 2 {
		t.Errorf("no samples: got %ds, want the 2s floor", got)
	}
}

func TestCalibrationLockIsExclusive(t *testing.T) {
	if !TryLockCalibration("proxy-a") {
		t.Fatal("first lock should succeed")
	}
	if TryLockCalibration("proxy-a") {
		t.Error("a second run on the same proxy should be refused")
	}
	if !TryLockCalibration("proxy-b") {
		t.Error("a different proxy should not be blocked")
	}
	UnlockCalibration("proxy-a")
	UnlockCalibration("proxy-b")
	if !TryLockCalibration("proxy-a") {
		t.Error("lock should be reusable after release")
	}
	UnlockCalibration("proxy-a")
}

// TestProbeTimeoutFrom guards the bound that keeps a run short: a dropped
// request must be recognised in a fraction of a second rather than waiting out
// the proxy's own read timeout, which would turn one failing step into minutes.
func TestProbeTimeoutFrom(t *testing.T) {
	tests := []struct {
		name       string
		p95Ms      float64
		configured time.Duration
		want       time.Duration
	}{
		{"fast device keeps the floor", 20, 30 * time.Second, 500 * time.Millisecond},
		{"slow device gets headroom", 400, 30 * time.Second, 2 * time.Second},
		{"never exceeds the proxy timeout", 5000, 3 * time.Second, 3 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := probeTimeoutFrom(tt.p95Ms, tt.configured); got != tt.want {
				t.Errorf("probeTimeoutFrom(%v, %v) = %v, want %v", tt.p95Ms, tt.configured, got, tt.want)
			}
		})
	}
}

// TestCalibrationFinishesQuicklyAgainstAPickyDevice is the regression for a run
// that took minutes: a failing step waited out a full read timeout for every
// request instead of calling it after a few.
func TestCalibrationFinishesQuicklyAgainstAPickyDevice(t *testing.T) {
	target := newPickyTarget(t, 80*time.Millisecond, 4, 5*time.Millisecond)
	p := startTestProxy(t, target.addr(), func(p *ProxyInstance) {
		p.ReadTimeout = 30 * time.Second // the default a new proxy carries
	})
	defer p.Stop()

	started := time.Now()
	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:            ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep:  6,
		GapStepsMs:       []int{200, 100, 50},
		ConnectionLevels: []int{1},
	})
	if err != nil {
		t.Fatalf("calibration failed: %v", err)
	}

	if elapsed := time.Since(started); elapsed > 20*time.Second {
		t.Errorf("run took %v against a 30s read timeout; a failing step must not wait it out per request", elapsed)
	}
	if result.Recommended.MinRequestGapMs < 80 {
		t.Errorf("recommended %d ms, below the 80 ms the device needs", result.Recommended.MinRequestGapMs)
	}
}

// TestCalibrationStopsAtItsDeadline is the promise that matters most in the
// field: clients are held off for the whole run, so nothing is being polled or
// controlled while it lasts. A run that overruns its budget is worse than a
// coarser measurement.
func TestCalibrationStopsAtItsDeadline(t *testing.T) {
	target := newPickyTarget(t, 0, 4, 120*time.Millisecond)
	p := calibrationProxy(t, target.addr())

	started := time.Now()
	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:            ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep:  20,
		GapStepsMs:       []int{200, 150, 100, 50, 25, 10},
		ConnectionLevels: []int{1, 2, 4},
		MaxDuration:      3 * time.Second,
	})
	if err != nil {
		t.Fatalf("calibration failed: %v", err)
	}

	elapsed := time.Since(started)
	if elapsed > 12*time.Second {
		t.Errorf("run took %v against a 3s ceiling", elapsed)
	}
	if len(result.GapSteps) == 0 {
		t.Error("a run cut short must still report what it measured")
	}
	if result.Recommended.MinRequestGapMs == 0 {
		t.Error("a run cut short must still recommend something usable")
	}
}

// TestCalibrationFallsBackWhenTheDeviceIsUnreliable verifies that a device
// failing even at the most careful spacing yields conservative settings and a
// plain statement, rather than an error that leaves the operator with nothing.
func TestCalibrationFallsBackWhenTheDeviceIsUnreliable(t *testing.T) {
	// A floor far above any spacing the run will try.
	target := newPickyTarget(t, 5*time.Second, 4, 0)
	p := calibrationProxy(t, target.addr())

	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:           ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep: 4,
		GapStepsMs:      []int{200, 100},
		MaxDuration:     20 * time.Second,
	})
	if err != nil {
		t.Fatalf("expected conservative settings, got an error: %v", err)
	}
	if result.Recommended.MinRequestGapMs != 200 {
		t.Errorf("fallback spacing = %d ms, want the most careful value 200", result.Recommended.MinRequestGapMs)
	}
	if result.Recommended.MaxTargetConns != 1 {
		t.Errorf("fallback connections = %d, want 1", result.Recommended.MaxTargetConns)
	}
	if len(result.Notes) == 0 {
		t.Error("a fallback must say that it is one")
	}
}
