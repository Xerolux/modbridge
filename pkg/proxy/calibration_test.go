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
	listener net.Listener
	minGap   time.Duration
	latency  time.Duration

	// Session slots, one per connection the device is willing to serve.
	slots chan struct{}

	// releaseDelay is how long the device keeps a session occupied after the
	// peer has gone. Real single-session devices do not free the slot at the
	// instant the socket closes; they notice, and that takes a moment.
	releaseDelay time.Duration

	// ignoreFirstRequest makes every connection swallow its first request, the
	// behaviour of a device that needs a moment after the handshake.
	ignoreFirstRequest bool

	// live counts connections the device still holds open.
	live atomic.Int32

	mu       sync.Mutex
	lastSeen time.Time
}

// sessionHandover is how long a new connection waits for a slot before the
// device treats it as one session too many.
//
// A slot is returned when the previous connection's handler notices the close,
// which happens some scheduling delay after the peer actually closed. Deciding
// at the instant of accept makes the first connection of a phase look like a
// second session whenever that delay lands in between — a real failure on a
// loaded machine, and one that says nothing about the code under test.
//
// The window has to sit between the two intervals it separates: longer than
// reaping a closed connection, shorter than a step in which two connections are
// genuinely open at once. Reaping is a runnable goroutine away; a step holds its
// connections for at least twice this long by construction (see the request
// counts below), so there is room on both sides.
const sessionHandover = 150 * time.Millisecond

func newPickyTarget(t *testing.T, minGap time.Duration, maxSessions int, latency time.Duration) *pickyTarget {
	return newSlowReleasingTarget(t, minGap, maxSessions, latency, 0)
}

// newSlowReleasingTarget adds the one behaviour that separates a device holding
// a session from a device refusing one: it keeps the slot for a while after the
// peer disconnects.
func newSlowReleasingTarget(t *testing.T, minGap time.Duration, maxSessions int, latency, releaseDelay time.Duration) *pickyTarget {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock target: %v", err)
	}

	target := &pickyTarget{
		listener:     listener,
		minGap:       minGap,
		latency:      latency,
		releaseDelay: releaseDelay,
		slots:        make(chan struct{}, maxSessions),
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
	pt.live.Add(1)
	defer pt.live.Add(-1)
	defer conn.Close()

	seen := 0

	var over bool
	select {
	case pt.slots <- struct{}{}:
		defer func() {
			if pt.releaseDelay > 0 {
				time.Sleep(pt.releaseDelay)
			}
			<-pt.slots
		}()
	case <-time.After(sessionHandover):
		over = true
	}

	for {
		frame, err := modbus.ReadFrame(conn)
		if err != nil {
			return
		}
		if over {
			continue // accepted the socket, answers nothing — the usual failure mode
		}

		seen++
		if pt.ignoreFirstRequest && seen == 1 {
			continue // the handshake is done, the device is not ready yet
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

	// Eight requests split across two connections means each of them holds its
	// session for three gaps — 300 ms against the recommended 100 ms spacing —
	// so the second connection is still waiting when its grace runs out and is
	// correctly seen as one session too many.
	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:            ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep:  8,
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

// waitForClients blocks until the proxy reports the given number of live client
// connections, so a test does not race the accept loop.
func waitForClients(t *testing.T, p *ProxyInstance, want int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if p.Stats.ActiveConns.Load() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("proxy reports %d client(s), want %d", p.Stats.ActiveConns.Load(), want)
}

// TestCalibrationReleasesConnectedClients is the promise that replaced the old
// refusal: a run still never competes with live traffic, but the person who
// pressed the button is not the one who has to go unplug a controller.
func TestCalibrationReleasesConnectedClients(t *testing.T) {
	target := newPickyTarget(t, 0, 4, 0)
	p := calibrationProxy(t, target.addr())

	client, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect a client: %v", err)
	}
	defer client.Close()
	waitForClients(t, p, 1)

	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:            ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep:  2,
		GapStepsMs:       []int{50},
		ConnectionLevels: []int{1},
	})
	if err != nil {
		t.Fatalf("calibration refused to run instead of releasing the client: %v", err)
	}
	if p.Stats.ActiveConns.Load() != 0 {
		t.Errorf("%d client(s) still connected after the run", p.Stats.ActiveConns.Load())
	}

	// The client learns it was let go, rather than sitting on a socket the
	// proxy has stopped answering.
	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	if _, err := modbus.ReadFrame(client); err == nil {
		t.Error("the client connection was still open after the run")
	}

	// And the report says so, in numbers rather than a finished sentence.
	var released *Note
	for i := range result.Notes {
		if result.Notes[i].Code == "clientsReleased" {
			released = &result.Notes[i]
		}
	}
	if released == nil {
		t.Fatalf("no note about the released client, got %+v", result.Notes)
	}
	if released.Args["clients"] != 1 {
		t.Errorf("note reports %d released client(s), want 1", released.Args["clients"])
	}
}

// TestCalibrationLetsAClientFinishItsRequest is the difference between handing
// a connection back and cutting it. A client waiting on an answer gets it; only
// then does the run take the device. Without that, a write in flight would be
// left unanswered and its sender would not know whether it landed.
func TestCalibrationLetsAClientFinishItsRequest(t *testing.T) {
	target := newPickyTarget(t, 0, 4, 300*time.Millisecond)
	p := calibrationProxy(t, target.addr())
	p.ReadTimeout = 2 * time.Second // the target is deliberately slow here

	client, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect a client: %v", err)
	}
	defer client.Close()
	waitForClients(t, p, 1)

	if _, err := client.Write(modbus.CreateReadRequest(7, 1, 3, 0, 2)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	// Long enough that the request is with the target, short enough that its
	// answer is not back yet.
	time.Sleep(100 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		_, err := p.Calibrate(t.Context(), CalibrationConfig{
			Probe:            ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
			RequestsPerStep:  2,
			GapStepsMs:       []int{50},
			ConnectionLevels: []int{1},
		})
		done <- err
	}()

	if err := client.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	frame, err := modbus.ReadFrame(client)
	if err != nil {
		t.Fatalf("the request in flight was dropped instead of answered: %v", err)
	}
	if len(frame) < 8 || frame[7] != 3 {
		t.Errorf("client got %X, want an answer to its read", frame)
	}

	if err := <-done; err != nil {
		t.Fatalf("calibration failed: %v", err)
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

// TestCalibrationHoldsTheBackgroundPoller covers the other thing that talks to
// the device without a client asking. A poller left running fires its own reads
// between the probes, and on a device that wants a pause between requests the
// probe following a poll is dropped — so the run blames the spacing for traffic
// it created itself. Found by measuring a proxy that had the cache enabled: it
// reported 500 ms against a device that measured 100 ms once the poller was
// held.
func TestCalibrationHoldsTheBackgroundPoller(t *testing.T) {
	target := newPickyTarget(t, 60*time.Millisecond, 4, 5*time.Millisecond)
	p := calibrationProxy(t, target.addr())

	// A poller that would otherwise hammer the target throughout the run.
	var polls atomic.Int64
	p.poller = NewRegisterPoller(20*time.Millisecond, time.Minute, 16,
		func(req []byte) ([]byte, error) {
			polls.Add(1)
			return nil, nil
		},
		func(key uint64, unitID uint8, resp []byte) {},
		func(msg string) {},
	)
	p.poller.Track(1, 1, modbus.CreateReadRequest(1, 1, 3, 0, 2))
	p.poller.Start(t.Context())
	defer p.poller.Stop()

	// It has to be demonstrably alive, or the test proves nothing.
	deadline := time.Now().Add(2 * time.Second)
	for polls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if polls.Load() == 0 {
		t.Fatal("the poller never ran, so holding it cannot be observed")
	}

	before := polls.Load()
	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:           ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep: 4,
		GapStepsMs:      []int{200, 100},
		MaxDuration:     30 * time.Second,
	})
	during := polls.Load() - before
	if err != nil {
		t.Fatalf("calibration failed: %v", err)
	}
	// A round already in flight may finish, but a 20 ms poller across a run of
	// seconds would otherwise poll dozens of times.
	if during > 1 {
		t.Errorf("the poller ran %d times during the measurement, want it held", during)
	}
	if len(result.GapSteps) == 0 || result.GapSteps[0].Errors != 0 {
		t.Errorf("undisturbed 200 ms spacing should be clean, got %+v", result.GapSteps)
	}

	// And it has to come back afterwards, or a measurement would silently cost
	// the proxy its cache refreshes.
	resumed := polls.Load()
	deadline = time.Now().Add(2 * time.Second)
	for polls.Load() <= resumed && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if polls.Load() <= resumed {
		t.Error("the poller did not resume after the measurement")
	}
}

// TestCalibrationDoesNotBlameParallelismForAHandover covers the moment between
// the two phases. A device that serves one session at a time frees it shortly
// after the socket closes, not at the instant of closing; dialling into that
// window made the first connection look like a second session, and the run then
// reported that a single-session device cannot serve a single session.
func TestCalibrationDoesNotBlameParallelismForAHandover(t *testing.T) {
	// Holds its session for 200 ms after the peer leaves — long enough that
	// dialling straight into the handover window is judged a second session.
	target := newSlowReleasingTarget(t, 0, 1, 5*time.Millisecond, 200*time.Millisecond)
	p := calibrationProxy(t, target.addr())

	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:            ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep:  8,
		GapStepsMs:       []int{50},
		ConnectionLevels: []int{1},
		MaxDuration:      30 * time.Second,
	})
	if err != nil {
		t.Fatalf("calibration failed: %v", err)
	}

	if len(result.ConnectionSteps) == 0 {
		t.Fatal("no connection step was measured")
	}
	if first := result.ConnectionSteps[0]; first.Connections != 1 || first.Errors != 0 {
		t.Errorf("a single connection to a single-session device should be clean, got %+v", first)
	}
	if result.Recommended.MaxTargetConns != 1 {
		t.Errorf("recommended %d connections against a single-session device", result.Recommended.MaxTargetConns)
	}
	for _, n := range result.Notes {
		if n.Code == "singleConnectionErrors" {
			t.Errorf("the device served its one session; note says otherwise: %q", n.Text)
		}
	}
}

// TestCalibrationNotesCarryCodesAndNumbers guards what makes the notes
// translatable: a code the interface can look up and the numbers separately,
// rather than a finished English sentence it can only print.
func TestCalibrationNotesCarryCodesAndNumbers(t *testing.T) {
	target := newPickyTarget(t, 60*time.Millisecond, 4, 5*time.Millisecond)
	p := calibrationProxy(t, target.addr())

	result, err := p.Calibrate(t.Context(), CalibrationConfig{
		Probe:            ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		RequestsPerStep:  6,
		GapStepsMs:       []int{200, 100, 25},
		ConnectionLevels: []int{1},
		MaxDuration:      30 * time.Second,
	})
	if err != nil {
		t.Fatalf("calibration failed: %v", err)
	}
	if len(result.Notes) == 0 {
		t.Fatal("a run that found a limit should say so in its notes")
	}

	byCode := map[string]Note{}
	for _, n := range result.Notes {
		if n.Code == "" {
			t.Errorf("note without a code cannot be translated: %+v", n)
		}
		if n.Text == "" {
			t.Errorf("note %q has no English fallback", n.Code)
		}
		byCode[n.Code] = n
	}

	// The device gives up at 25 ms, so that note must carry the spacing it
	// failed at — as a number, not only inside the sentence.
	ceiling, ok := byCode["spacingCeiling"]
	if !ok {
		t.Fatalf("expected a spacingCeiling note, got %v", byCode)
	}
	if ceiling.Args["gapMs"] != 25 {
		t.Errorf("spacingCeiling reports gapMs %d, want 25", ceiling.Args["gapMs"])
	}
	if ceiling.Args["errors"] <= 0 {
		t.Errorf("spacingCeiling reports %d errors, want the failures it saw", ceiling.Args["errors"])
	}
}

// TestMeasureGapClosesTheConnectionItReplaces covers a leak on the path that
// matters most. The warm-up request exists because the first request after a
// fresh connection carries the transition; when it fails, the step dials again.
// The close was written as `defer conn.Close()`, which binds the connection the
// statement saw — so the replacement was never closed. On a device that serves
// one session that leaked socket holds the session for the rest of the run, and
// every later step then measures a device busy with us.
func TestMeasureGapClosesTheConnectionItReplaces(t *testing.T) {
	// Swallows the first request on every connection, so the warm-up always
	// fails and the redial always happens.
	target := newPickyTarget(t, 0, 8, 2*time.Millisecond)
	target.ignoreFirstRequest = true
	p := calibrationProxy(t, target.addr())

	// The proxy's own health check may hold a connection; what matters is that
	// the step leaves no more behind than it found.
	time.Sleep(300 * time.Millisecond)
	before := target.live.Load()

	step, err := p.measureGap(t.Context(), ProbeSpec{UnitID: 1, Function: 3, Address: 0, Quantity: 2},
		10, 3, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("measureGap failed: %v", err)
	}
	if step.Requests != 3 {
		t.Fatalf("measured %d requests, want 3", step.Requests)
	}

	// Give the device a moment to notice the closes, then insist on silence.
	deadline := time.Now().Add(2 * time.Second)
	for target.live.Load() > before && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if held := target.live.Load(); held > before {
		t.Errorf("the step left %d connection(s) open beyond the %d it started with; the redial was never closed",
			held-before, before)
	}
}
