// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"context"
	"fmt"
	"math"
	"modbridge/pkg/modbus"
	"net"
	"sort"
	"sync"
	"time"
)

// Calibration measures what a target device actually tolerates instead of
// assuming it. Device profiles carry reasoned guesses; this replaces them with
// numbers taken from the device in front of you.
//
// It is an experiment on live hardware, so it is deliberately constrained:
//
//   - Reads only. A calibration run never writes a register.
//   - It probes a request the device is already being asked for, so it touches
//     no address the client does not touch anyway.
//   - It runs on its own connections, never through the proxy's pool, so a
//     running proxy is not reconfigured underneath its clients.
//   - It only ever reports. Applying the result stays a deliberate act.
//   - It refuses to run while clients are connected, because their traffic
//     both distorts the measurement and gets distorted by it.
//
// A single error is not a measurement either: every step sends a series of
// requests and is judged on the error rate, and the accepted value keeps a
// safety margin over the fastest one that worked. Devices behave differently
// when warm, when the network is busy, or when a second client shows up.

// ProbeSpec is the read request a calibration run repeats.
type ProbeSpec struct {
	UnitID   uint8  `json:"unit_id"`
	Function uint8  `json:"function"`
	Address  uint16 `json:"address"`
	Quantity uint16 `json:"quantity"`
}

// Valid reports whether the spec describes a usable read.
func (ps ProbeSpec) Valid() bool {
	return modbus.IsReadFunction(ps.Function) && ps.Quantity > 0 && ps.Quantity <= 125
}

func (ps ProbeSpec) frame(txID uint16) []byte {
	return modbus.CreateReadRequest(txID, ps.UnitID, ps.Function, ps.Address, ps.Quantity)
}

// GapStep is one measured request spacing.
type GapStep struct {
	GapMs    int     `json:"gap_ms"`
	Requests int     `json:"requests"`
	Errors   int     `json:"errors"`
	P50Ms    float64 `json:"p50_ms"`
	P95Ms    float64 `json:"p95_ms"`
}

// ConnectionStep is one measured level of parallel connections.
type ConnectionStep struct {
	Connections int     `json:"connections"`
	Requests    int     `json:"requests"`
	Errors      int     `json:"errors"`
	P95Ms       float64 `json:"p95_ms"`
}

// RecommendedSettings is what the measurements suggest.
type RecommendedSettings struct {
	MinRequestGapMs int `json:"min_request_gap_ms"`
	MaxTargetConns  int `json:"max_target_conns"`
	ReadTimeoutS    int `json:"read_timeout"`
}

// CalibrationResult is the full record of a run: every step that was measured,
// not just the conclusion, so the recommendation can be checked rather than
// trusted.
type CalibrationResult struct {
	TargetAddr      string              `json:"target_addr"`
	Probe           ProbeSpec           `json:"probe"`
	GapSteps        []GapStep           `json:"gap_steps"`
	ConnectionSteps []ConnectionStep    `json:"connection_steps"`
	Recommended     RecommendedSettings `json:"recommended"`
	Notes           []Note              `json:"notes"`
	DurationMs      int64               `json:"duration_ms"`
}

// Note explains one finding of a run. It carries a code and the numbers that
// belong to it rather than a finished sentence, because the interface speaks
// the operator's language and the server does not know which one that is. Text
// is the same statement in English, for anything reading the API directly and
// as a fallback when a code is newer than the interface showing it.
type Note struct {
	Code string         `json:"code"`
	Args map[string]int `json:"args,omitempty"`
	Text string         `json:"text"`
}

// sessionReleaseGrace is the pause between the spacing phase and the connection
// phase. A device that serves one session at a time releases it a moment after
// the socket closes, not at the instant of closing, and measuring inside that
// window blames parallelism for a handover.
const sessionReleaseGrace = 300 * time.Millisecond

// note builds a Note; the English sentence is written at the call site so the
// code and its wording stay next to each other.
func note(code, text string, args map[string]int) Note {
	return Note{Code: code, Args: args, Text: text}
}

// CalibrationConfig tunes a run. The zero value is a sensible run.
type CalibrationConfig struct {
	Probe            ProbeSpec     // Read to repeat; falls back to the last read the proxy saw
	RequestsPerStep  int           // Requests per step (default 20). One error is noise; a rate is a measurement.
	GapStepsMs       []int         // Spacings to try, descending (default 200…10)
	ConnectionLevels []int         // Parallel connections to try, ascending (default 1, 2, 4)
	SafetyFactor     float64       // Margin over the fastest clean spacing (default 1.5)
	Force            bool          // Run even while clients are connected
	MaxDuration      time.Duration // Hard ceiling for the whole run (default 90s)
}

func (cfg *CalibrationConfig) applyDefaults() {
	if cfg.RequestsPerStep <= 0 {
		cfg.RequestsPerStep = 20
	}
	if len(cfg.GapStepsMs) == 0 {
		cfg.GapStepsMs = []int{200, 150, 100, 50, 25, 10}
	}
	if len(cfg.ConnectionLevels) == 0 {
		cfg.ConnectionLevels = []int{1, 2, 4}
	}
	if cfg.SafetyFactor <= 0 {
		cfg.SafetyFactor = 1.5
	}
	if cfg.MaxDuration <= 0 {
		// Clients are held off for the duration, so nothing is being polled or
		// controlled while this runs. A minute and a half is a defensible pause;
		// ten minutes is not, whatever the measurement would be worth.
		cfg.MaxDuration = 90 * time.Second
	}
}

// LastObservedRead returns a probe derived from the most recent read a client
// asked this proxy for, so calibration touches nothing new.
func (p *ProxyInstance) LastObservedRead() (ProbeSpec, bool) {
	v := p.lastRead.Load()
	if v == nil {
		return ProbeSpec{}, false
	}
	spec, ok := v.(ProbeSpec)
	return spec, ok && spec.Valid()
}

// recordObservedRead remembers a client read so it can be replayed as a probe.
func (p *ProxyInstance) recordObservedRead(frame []byte) {
	_, unitID, fc, addr, quantity, err := modbus.ParseReadRequest(frame)
	if err != nil {
		return
	}
	spec := ProbeSpec{UnitID: unitID, Function: fc, Address: addr, Quantity: quantity}
	if spec.Valid() {
		p.lastRead.Store(spec)
	}
}

// Calibrate measures the target and returns what it tolerates. It does not
// change the proxy's configuration.
func (p *ProxyInstance) Calibrate(ctx context.Context, cfg CalibrationConfig) (*CalibrationResult, error) {
	cfg.applyDefaults()

	if p.Stats.GetStatus() != "Running" {
		return nil, fmt.Errorf("proxy is not running")
	}
	if active := p.Stats.ActiveConns.Load(); active > 0 && !cfg.Force {
		return nil, fmt.Errorf("%d client(s) connected: their traffic would distort the measurement and be distorted by it", active)
	}

	probe := cfg.Probe
	if !probe.Valid() {
		observed, ok := p.LastObservedRead()
		if !ok {
			return nil, fmt.Errorf("no read request observed yet and none supplied: let a client poll once, or name a register to probe")
		}
		probe = observed
	}

	// Hold clients off for the duration. The check above rules out the ones
	// already connected; this rules out the one that connects mid-run.
	p.calibrating.Store(true)
	defer p.calibrating.Store(false)

	// A device that serves a single Modbus session cannot be measured while the
	// proxy is holding that session, and the health check would take it too.
	// No clients are connected at this point, so releasing both is safe and the
	// pool simply dials again afterwards.
	if p.healthChecker != nil {
		p.healthChecker.SetPaused(true)
		defer p.healthChecker.SetPaused(false)
	}
	// The background poller is the other thing that talks to the device without
	// a client asking. Left running, it fires its own reads between the probes:
	// on a device that wants a pause between requests, the probe that follows a
	// poll is dropped and the run blames the spacing. Measured against a device
	// with a 60 ms floor, a proxy with a 5 s poller reported 500 ms where the
	// same device measured 100 ms once the poller was held.
	if p.poller != nil {
		p.poller.SetPaused(true)
		defer p.poller.SetPaused(false)
	}
	if p.connPool != nil {
		if drained := p.connPool.DrainIdle(); drained > 0 {
			p.log.Info(p.ID, fmt.Sprintf("Calibration released %d pooled connection(s) so the target is idle", drained))
			// Same handover as between the phases: the socket is closed, the
			// device has not noticed yet. Probing into that window measures a
			// device that is still busy with the connection we just gave back.
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sessionReleaseGrace):
			}
		}
	}

	started := time.Now()
	deadline := started.Add(cfg.MaxDuration)
	result := &CalibrationResult{
		TargetAddr: p.TargetAddr,
		Probe:      probe,
		Notes:      []Note{},
	}

	// Spacing: walk down until the device starts failing. The last clean step
	// is the fastest it tolerated, and it is kept with a margin.
	fastestClean := -1
	// The first step runs on the proxy's own read timeout because nothing is
	// known yet; afterwards the measured latency sets a far tighter bound.
	configuredRead, _ := p.currentTimeouts()
	probeTimeout := configuredRead

	for _, gapMs := range cfg.GapStepsMs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if time.Now().After(deadline) {
			result.Notes = append(result.Notes, note("stoppedOnTime", fmt.Sprintf(
				"stopped after %v to give the device back to its clients; the remaining spacings were not measured",
				cfg.MaxDuration), map[string]int{"seconds": int(cfg.MaxDuration.Seconds())}))
			break
		}

		step, err := p.measureGap(ctx, probe, gapMs, cfg.RequestsPerStep, probeTimeout)
		if err != nil {
			return nil, fmt.Errorf("measuring %d ms spacing: %w", gapMs, err)
		}
		result.GapSteps = append(result.GapSteps, step)

		if step.Errors > 0 {
			result.Notes = append(result.Notes, note("spacingCeiling", fmt.Sprintf(
				"%d ms spacing produced %d error(s) in %d requests — this is where the device stops keeping up",
				gapMs, step.Errors, step.Requests),
				map[string]int{"gapMs": gapMs, "errors": step.Errors, "requests": step.Requests}))
			break
		}
		fastestClean = gapMs
		probeTimeout = probeTimeoutFrom(step.P95Ms, configuredRead)
	}

	if len(result.GapSteps) > 0 && result.GapSteps[0].Errors > 0 {
		// The device failed even at the most careful spacing. Returning an error
		// would leave the operator with nothing; the useful answer is the safe
		// end of the range plus a clear statement of what was seen.
		result.Recommended = RecommendedSettings{
			MinRequestGapMs: cfg.GapStepsMs[0],
			MaxTargetConns:  1,
			ReadTimeoutS:    readTimeoutFrom(result.GapSteps),
		}
		result.Notes = append(result.Notes, note("unreliableDevice", fmt.Sprintf(
			"the device failed %d of %d requests even at the most careful spacing (%d ms) — these are conservative fallback settings, not a measurement. Check the device and the link before trusting them.",
			result.GapSteps[0].Errors, result.GapSteps[0].Requests, cfg.GapStepsMs[0]),
			map[string]int{"errors": result.GapSteps[0].Errors, "requests": result.GapSteps[0].Requests, "gapMs": cfg.GapStepsMs[0]}))
		result.DurationMs = time.Since(started).Milliseconds()
		return result, nil
	}

	if fastestClean < 0 {
		result.Recommended.MinRequestGapMs = cfg.GapStepsMs[0]
		result.Notes = append(result.Notes, note("noCleanSpacing",
			"no spacing ran cleanly; keeping the most careful value", nil))
	} else {
		result.Recommended.MinRequestGapMs = withSafetyMargin(fastestClean, cfg.SafetyFactor)
		if result.Recommended.MinRequestGapMs != fastestClean {
			result.Notes = append(result.Notes, note("spacingMargin", fmt.Sprintf(
				"fastest clean spacing was %d ms; recommending %d ms so the device keeps headroom when warm or busy",
				fastestClean, result.Recommended.MinRequestGapMs),
				map[string]int{"fastestMs": fastestClean, "recommendedMs": result.Recommended.MinRequestGapMs}))
		}
	}

	// Parallel sessions: many devices serve exactly one and ignore the rest,
	// which shows up as timeouts rather than as a refusal.
	//
	// The spacing phase has just closed its connection, and a device that keeps
	// one session does not free it the instant the socket closes — it notices.
	// Dialling straight into that window makes the very first connection look
	// like a second session, and the run then concludes that a device cannot
	// serve the one session it is plainly serving. Waiting a moment costs
	// nothing next to a run of tens of seconds.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(sessionReleaseGrace):
	}

	safeGap := result.Recommended.MinRequestGapMs
	result.Recommended.MaxTargetConns = 1
	for _, level := range cfg.ConnectionLevels {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if time.Now().After(deadline) {
			result.Notes = append(result.Notes, note("stoppedBeforeConnections",
				"stopped before measuring more parallel connections; keeping the safe single session", nil))
			break
		}

		step, err := p.measureConnections(ctx, probe, level, safeGap, cfg.RequestsPerStep, probeTimeout)
		if err != nil {
			return nil, fmt.Errorf("measuring %d connection(s): %w", level, err)
		}
		result.ConnectionSteps = append(result.ConnectionSteps, step)

		if step.Errors > 0 {
			// At one connection there is no "that many sessions" to blame: the
			// device failed on the only session there was, which says something
			// about the device or the link, not about parallelism.
			if level <= 1 {
				result.Notes = append(result.Notes, note("singleConnectionErrors", fmt.Sprintf(
					"the device failed %d of %d requests on a single connection — check the device or the link; keeping one session",
					step.Errors, step.Requests),
					map[string]int{"errors": step.Errors, "requests": step.Requests}))
			} else {
				result.Notes = append(result.Notes, note("connectionsRefused", fmt.Sprintf(
					"%d parallel connection(s) produced %d error(s) — the device does not serve that many sessions",
					level, step.Errors),
					map[string]int{"connections": level, "errors": step.Errors}))
			}
			break
		}
		result.Recommended.MaxTargetConns = level
	}

	// Read timeout from what the device actually needed, with room for a slow
	// moment rather than a number picked in advance.
	result.Recommended.ReadTimeoutS = readTimeoutFrom(result.GapSteps)
	result.DurationMs = time.Since(started).Milliseconds()
	return result, nil
}

// probeTimeoutFrom turns a measured p95 into a probe timeout: generous enough
// that a healthy answer is never called an error, tight enough that a dropped
// request is recognised quickly. Never longer than the proxy's own timeout.
func probeTimeoutFrom(p95Ms float64, configured time.Duration) time.Duration {
	timeout := time.Duration(p95Ms*5) * time.Millisecond
	if timeout < 500*time.Millisecond {
		timeout = 500 * time.Millisecond
	}
	if configured > 0 && timeout > configured {
		timeout = configured
	}
	return timeout
}

// withSafetyMargin scales a measured spacing up and rounds it to a value a
// human would have chosen.
func withSafetyMargin(gapMs int, factor float64) int {
	scaled := int(math.Ceil(float64(gapMs) * factor))
	for _, nice := range []int{10, 25, 50, 100, 150, 200, 250, 500, 1000} {
		if scaled <= nice {
			return nice
		}
	}
	return scaled
}

// readTimeoutFrom derives a read timeout from the slowest p95 that was seen,
// with generous headroom, floored at 2 seconds.
func readTimeoutFrom(steps []GapStep) int {
	worstP95 := 0.0
	for _, step := range steps {
		if step.P95Ms > worstP95 {
			worstP95 = step.P95Ms
		}
	}
	seconds := int(math.Ceil(worstP95 * 4 / 1000))
	if seconds < 2 {
		seconds = 2
	}
	if seconds > 60 {
		seconds = 60
	}
	return seconds
}

// maxStepErrors ends a step once the device has failed often enough to call
// it: waiting out the remaining timeouts adds minutes and no information.
const maxStepErrors = 3

// measureGap sends a series of probes at one spacing over a single connection.
func (p *ProxyInstance) measureGap(ctx context.Context, probe ProbeSpec, gapMs, requests int, probeTimeout time.Duration) (GapStep, error) {
	step := GapStep{GapMs: gapMs, Requests: requests}

	conn, err := p.dialTarget(ctx)
	if err != nil {
		return step, err
	}
	// Closed through the variable, not through the connection this statement
	// happens to see: the warm-up below may replace it, and `defer conn.Close()`
	// would then close the first socket and leak the second. On a device that
	// serves one session that leak holds the session for the rest of the run,
	// so every later step measures a device that is busy with us.
	defer func() { conn.Close() }()

	latencies := make([]float64, 0, requests)
	gap := time.Duration(gapMs) * time.Millisecond

	// One warm-up request, not counted. It carries the transition from
	// whatever came before — the previous step, a fresh connection — and would
	// otherwise be blamed on this spacing.
	if gap > 0 {
		time.Sleep(gap)
	}
	if _, err := p.probeOnce(conn, probe, 0xFFFF, probeTimeout); err != nil {
		conn.Close()
		if conn, err = p.dialTarget(ctx); err != nil {
			return step, err
		}
	}

	for i := 0; i < requests; i++ {
		select {
		case <-ctx.Done():
			return step, ctx.Err()
		default:
		}

		// Every counted request waits, including the first: the warm-up came
		// immediately before it and its spacing has to be honoured too.
		if gap > 0 {
			time.Sleep(gap)
		}

		elapsed, err := p.probeOnce(conn, probe, uint16(i+1), probeTimeout)
		if err != nil {
			step.Errors++
			if step.Errors >= maxStepErrors {
				// Enough to call the step failed. Report how far it got, so the
				// error rate stays honest.
				step.Requests = i + 1
				break
			}
			// A broken connection cannot serve the rest of the run; reconnect
			// so the remaining requests still measure the device.
			conn.Close()
			conn, err = p.dialTarget(ctx)
			if err != nil {
				return step, err
			}
			continue
		}
		latencies = append(latencies, float64(elapsed.Microseconds())/1000)
	}

	step.P50Ms, step.P95Ms = percentiles(latencies)
	return step, nil
}

// measureConnections runs probes across several connections at once.
func (p *ProxyInstance) measureConnections(ctx context.Context, probe ProbeSpec, connections, gapMs, requests int, probeTimeout time.Duration) (ConnectionStep, error) {
	step := ConnectionStep{Connections: connections, Requests: requests}
	if connections <= 1 {
		gapStep, err := p.measureGap(ctx, probe, gapMs, requests, probeTimeout)
		if err != nil {
			return step, err
		}
		step.Errors = gapStep.Errors
		step.P95Ms = gapStep.P95Ms
		return step, nil
	}

	perConn := requests / connections
	if perConn < 1 {
		perConn = 1
	}
	step.Requests = perConn * connections

	var (
		mu        sync.Mutex
		latencies []float64
		errors    int
		wg        sync.WaitGroup
		dialErr   error
	)

	for c := 0; c < connections; c++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			conn, err := p.dialTarget(ctx)
			if err != nil {
				mu.Lock()
				if dialErr == nil {
					dialErr = err
				}
				mu.Unlock()
				return
			}
			defer conn.Close()

			for i := 0; i < perConn; i++ {
				if gapMs > 0 && i > 0 {
					time.Sleep(time.Duration(gapMs) * time.Millisecond)
				}
				elapsed, err := p.probeOnce(conn, probe, uint16(worker*1000+i+1), probeTimeout)
				mu.Lock()
				if err != nil {
					errors++
				} else {
					latencies = append(latencies, float64(elapsed.Microseconds())/1000)
				}
				mu.Unlock()
			}
		}(c)
	}
	wg.Wait()

	if dialErr != nil && len(latencies) == 0 {
		return step, dialErr
	}
	step.Errors = errors
	_, step.P95Ms = percentiles(latencies)
	return step, nil
}

// dialTarget opens a probe connection, honouring the configured connect delay
// for devices that ignore requests arriving right after the handshake.
func (p *ProxyInstance) dialTarget(ctx context.Context) (net.Conn, error) {
	dialer := net.Dialer{Timeout: p.ConnectionTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", p.TargetAddr)
	if err != nil {
		return nil, err
	}
	if p.ConnectDelay > 0 {
		select {
		case <-time.After(p.ConnectDelay):
		case <-ctx.Done():
			conn.Close()
			return nil, ctx.Err()
		}
	}
	return conn, nil
}

// probeOnce sends one read and waits for its answer, returning how long it took.
// The timeout is the calibration's own, not the proxy's: a dropped request has
// to be recognised in a fraction of a second, or measuring the spacing where a
// device starts failing would take longer than anyone would wait.
func (p *ProxyInstance) probeOnce(conn net.Conn, probe ProbeSpec, txID uint16, readTimeout time.Duration) (time.Duration, error) {
	req := probe.frame(txID)

	started := time.Now()
	if err := conn.SetWriteDeadline(time.Now().Add(p.ConnectionTimeout)); err != nil {
		return 0, err
	}
	if _, err := conn.Write(req); err != nil {
		return 0, err
	}
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return 0, err
	}

	resp, err := modbus.ReadFrame(conn)
	if err != nil {
		return 0, err
	}
	elapsed := time.Since(started)

	if got, ok := modbus.FrameTxID(resp); !ok || got != txID {
		return 0, fmt.Errorf("answer for transaction 0x%04X arrived while waiting for 0x%04X", got, txID)
	}
	if modbus.IsExceptionResponse(resp) {
		return 0, fmt.Errorf("device answered with a Modbus exception")
	}
	return elapsed, nil
}

// percentiles returns the 50th and 95th percentile of the samples in ms.
func percentiles(samples []float64) (p50, p95 float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	sorted := append([]float64(nil), samples...)
	sort.Float64s(sorted)

	at := func(q float64) float64 {
		idx := int(float64(len(sorted)-1) * q)
		return math.Round(sorted[idx]*100) / 100
	}
	return at(0.50), at(0.95)
}

// calibrationInFlight guards against two runs hammering the same target.
var calibrationInFlight sync.Map

// TryLockCalibration reserves a proxy for a calibration run.
func TryLockCalibration(proxyID string) bool {
	_, busy := calibrationInFlight.LoadOrStore(proxyID, struct{}{})
	return !busy
}

// UnlockCalibration releases the reservation.
func UnlockCalibration(proxyID string) {
	calibrationInFlight.Delete(proxyID)
}
