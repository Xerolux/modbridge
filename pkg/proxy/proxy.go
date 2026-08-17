// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"modbridge/pkg/devices"
	"modbridge/pkg/logger"
	"modbridge/pkg/middleware"
	"modbridge/pkg/modbus"
	"modbridge/pkg/pool"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionLimiter tracks system-wide connections and is injectable for tests.
type ConnectionLimiter struct {
	active int64
	max    int64
}

// globalLimiter is the default package-level limiter.
var globalLimiter = &ConnectionLimiter{max: 10000}

// SetGlobalMaxConnections updates the global connection limit.
func SetGlobalMaxConnections(max int64) {
	atomic.StoreInt64(&globalLimiter.max, max)
}

// GetGlobalMaxConnections returns the current global connection limit.
func GetGlobalMaxConnections() int64 {
	return atomic.LoadInt64(&globalLimiter.max)
}

// acquire attempts to increment the active count; returns false when at capacity.
func (cl *ConnectionLimiter) acquire() bool {
	max := atomic.LoadInt64(&cl.max)
	if max <= 0 {
		atomic.AddInt64(&cl.active, 1)
		return true
	}
	for {
		current := atomic.LoadInt64(&cl.active)
		if current >= max {
			return false
		}
		if atomic.CompareAndSwapInt64(&cl.active, current, current+1) {
			return true
		}
	}
}

// release decrements the active count.
func (cl *ConnectionLimiter) release() {
	atomic.AddInt64(&cl.active, -1)
}

// getNextRequestID generates a unique request ID
func (p *ProxyInstance) getNextRequestID() int64 {
	return atomic.AddInt64(&p.requestID, 1)
}

func tryAcquireConnSlot(sem chan struct{}) bool {
	select {
	case sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// ProxyInstance represents a running proxy.
type ProxyInstance struct {
	ID         string
	Name       string
	ListenAddr string
	TargetAddr string

	// Config
	MaxReadSize       int
	ConnectionTimeout time.Duration
	ReadTimeout       time.Duration
	MaxRetries        int
	MaxConns          int           // Maximum concurrent connections (0 = unlimited)
	Protocol          string        // "tcp" (default) or "rtu-tcp"
	ConnectDelay      time.Duration // Optional pause after TCP connect before first request (for slow devices like Huawei inverters/sDongles)
	MaxTargetConns    int           // Maximum simultaneous connections to the target (0 = default). Set to 1 for devices that accept a single Modbus session (SolarEdge/SunSpec inverters).
	MinRequestGap     time.Duration // Minimum spacing between two requests to the target (0 = none)
	RequestTimeout    time.Duration // Hard cap for one client request including retries (0 = derived from read timeout and retries)
	CacheEnabled      bool          // Serve repeated reads from a cache instead of asking the target every time
	CacheTTL          time.Duration // How long a cached read stays valid (0 = 5s default)
	PollInterval      time.Duration // Refresh cached reads in the background at this interval (0 = passive cache only)

	listener net.Listener
	connPool *pool.Pool
	connSem  chan struct{} // Semaphore for limiting concurrent connections
	startMu  sync.Mutex    // Protects Start/Stop lifecycle
	pacer    *requestPacer // Enforces MinRequestGap towards the target
	cache    *ResponseCache
	poller   *RegisterPoller

	log           *logger.Logger
	deviceTracker *devices.Tracker
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup

	// Enhanced features
	circuitBreaker  *CircuitBreaker
	enhancedStats   *EnhancedStats
	requestID       int64
	targetTxID      uint32
	staleResponses  int64
	healthChecker   *HealthChecker
	adaptiveTimeout *AdaptiveTimeout
	recoveryManager *RecoveryManager

	Stats Stats
}

// LatencyPercentiles returns current latency percentile snapshot.
func (p *ProxyInstance) LatencyPercentiles() LatencyPercentiles {
	if p.enhancedStats == nil {
		return LatencyPercentiles{}
	}
	return p.enhancedStats.GetPercentiles()
}

type Stats struct {
	Uptime        time.Duration
	lastStartNano atomic.Int64 // stores UnixNano; use SetLastStart/GetLastStart
	Requests      atomic.Int64
	Errors        atomic.Int64
	ActiveConns   atomic.Int64
	status        atomic.Value // stores string
}

// SetLastStart stores the start time atomically.
func (s *Stats) SetLastStart(t time.Time) {
	s.lastStartNano.Store(t.UnixNano())
}

// GetLastStart returns the last start time atomically.
func (s *Stats) GetLastStart() time.Time {
	n := s.lastStartNano.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}

func (s *Stats) GetStatus() string {
	v := s.status.Load()
	if v == nil {
		return "Stopped"
	}
	return v.(string)
}

func (s *Stats) setStatus(status string) {
	s.status.Store(status)
}

// NewProxyInstance creates a new proxy.
func NewProxyInstance(id, name, listen, target string, maxReadSize, connectionTimeout, readTimeout, maxRetries int, l *logger.Logger, tracker *devices.Tracker) *ProxyInstance {
	if connectionTimeout <= 0 {
		connectionTimeout = 5
	}
	if readTimeout <= 0 {
		readTimeout = 5
	}
	if maxRetries < 0 {
		maxRetries = 3
	}

	p := &ProxyInstance{
		ID:                id,
		Name:              name,
		ListenAddr:        listen,
		TargetAddr:        target,
		MaxReadSize:       maxReadSize,
		ConnectionTimeout: time.Duration(connectionTimeout) * time.Second,
		ReadTimeout:       time.Duration(readTimeout) * time.Second,
		MaxRetries:        maxRetries,
		MaxConns:          500,
		Protocol:          "tcp",
		log:               l,
		deviceTracker:     tracker,
	}
	// Initialize the connection semaphore once so that a restart does not
	// leave old connections holding a reference to a stale channel.
	p.connSem = make(chan struct{}, p.MaxConns)
	return p
}

// Start starts the proxy.
func (p *ProxyInstance) Start() error {
	p.startMu.Lock()
	defer p.startMu.Unlock()

	if p.Stats.GetStatus() == "Running" {
		return nil
	}

	validator := middleware.NewValidator()
	if err := validator.ValidatePort(p.ListenAddr); err != nil {
		p.Stats.setStatus("Error")
		return fmt.Errorf("invalid listen address: %w", err)
	}
	if err := validator.ValidatePort(p.TargetAddr); err != nil {
		p.Stats.setStatus("Error")
		return fmt.Errorf("invalid target address: %w", err)
	}

	l, err := net.Listen("tcp", p.ListenAddr)
	if err != nil {
		p.Stats.setStatus("Error")
		p.log.Error(p.ID, fmt.Sprintf("Port %s already in use or invalid: %v", p.ListenAddr, err))
		return fmt.Errorf("port %s already in use: %w", p.ListenAddr, err)
	}
	p.listener = l

	// Create connection pool for target with optimized settings.
	// MaxTargetConns caps how many sockets the target ever sees at once. Many
	// inverters (SolarEdge/SunSpec among them) accept exactly one Modbus
	// session and answer nothing on the others, so this is configurable.
	maxTargetConns := p.MaxTargetConns
	if maxTargetConns <= 0 {
		maxTargetConns = 10 // Moderate concurrency limit to avoid dropping connections
	}
	p.pacer = &requestPacer{gap: p.MinRequestGap}

	poolCfg := pool.Config{
		InitialSize:    1, // Modbus targets usually accept 1-3 connections max
		MaxSize:        maxTargetConns,
		MaxIdleTime:    10 * time.Minute, // Optimized: Longer idle time for reusability
		AcquireTimeout: p.ConnectionTimeout,
		Dialer: func(ctx context.Context) (net.Conn, error) {
			d := net.Dialer{
				Timeout:   p.ConnectionTimeout,
				KeepAlive: 30 * time.Second, // Optimized: TCP keep-alive
			}
			conn, err := d.DialContext(ctx, "tcp", p.TargetAddr)
			if err != nil {
				return nil, err
			}
			// Some Modbus devices (e.g. Huawei inverters/sDongles) drop or
			// ignore requests that arrive immediately after the TCP handshake.
			// Apply the configured connect delay once per fresh outbound
			// connection before handing it back to the pool consumer. This is
			// NOT applied when a pooled connection is reused.
			if p.ConnectDelay > 0 {
				select {
				case <-time.After(p.ConnectDelay):
				case <-ctx.Done():
					conn.Close()
					return nil, ctx.Err()
				}
			}
			return conn, nil
		},
	}

	p.connPool, err = pool.NewPool(poolCfg)
	if err != nil {
		p.listener.Close()
		p.Stats.setStatus("Error")
		p.log.Error(p.ID, fmt.Sprintf("Failed to create connection pool: %v", err))
		return err
	}

	// Initialize enhanced features
	p.circuitBreaker = NewCircuitBreaker(DefaultCircuitBreakerConfig())
	p.enhancedStats = NewEnhancedStats(1000) // Track last 1000 requests
	p.requestID = 0

	// Initialize health checker
	p.healthChecker = NewHealthChecker(
		p.TargetAddr,
		30*time.Second,
		p.ConnectionTimeout,
		func(id, msg string) { p.log.Info(id, msg) },
	)
	p.healthChecker.Start()

	p.healthChecker.SetOnUnhealthy(func() {
		p.log.Info(p.ID, "Health checker detected target failure, triggering recovery")
		if p.recoveryManager != nil {
			if _, err := p.recoveryManager.AddTask(p.TargetAddr, 10); err != nil {
				p.log.Error(p.ID, fmt.Sprintf("failed to schedule recovery task: %v", err))
			}
		}
	})

	p.healthChecker.SetOnRecovery(func() {
		p.log.Info(p.ID, "Health checker detected target recovery, resetting circuit breaker and pre-warming pool")
		if p.circuitBreaker != nil {
			p.circuitBreaker.Reset()
		}
		if p.connPool != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := p.connPool.PreWarm(ctx, 2); err != nil {
				p.log.Error(p.ID, fmt.Sprintf("Pool pre-warm failed: %v", err))
			}
		}
	})

	// RecoveryManager already performs a real TCP dial in attemptRecovery.
	// No additional onRecovery callback is needed here.
	p.recoveryManager = NewRecoveryManager(DefaultRecoveryConfig(), nil)

	// Initialize adaptive timeouts
	p.adaptiveTimeout = NewAdaptiveTimeout(p.ReadTimeout, p.ConnectionTimeout)

	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Read cache and background poller. Both are opt-in: a cached register is
	// by definition not the live value, which is right for dashboards and
	// wrong for control loops, so the operator decides.
	if p.CacheEnabled {
		cacheCfg := DefaultResponseCacheConfig()
		if p.CacheTTL > 0 {
			cacheCfg.TTL = p.CacheTTL
		}
		p.cache = NewResponseCache(cacheCfg)
		p.poller = NewRegisterPoller(
			p.PollInterval,
			10*cacheCfg.TTL,
			512,
			p.forwardClientRequest,
			func(key uint64, unitID uint8, resp []byte) { p.cache.SetForUnit(key, unitID, resp) },
			func(msg string) { p.log.Debug(p.ID, msg) },
		)
		p.poller.Start(p.ctx)
		p.log.Info(p.ID, fmt.Sprintf("Response cache enabled (ttl %v, background poll %v)", cacheCfg.TTL, p.PollInterval))
		if cacheTTLTooTight(cacheCfg.TTL, p.PollInterval) {
			p.log.Warn(p.ID, fmt.Sprintf(
				"Cache lifetime %v is not longer than the %v poll interval: entries expire between refresh rounds and clients will wait for the target anyway. Set it to several times the interval.",
				cacheCfg.TTL, p.PollInterval))
		}
	}

	p.Stats.setStatus("Running")
	p.Stats.SetLastStart(time.Now())

	p.log.Info(p.ID, fmt.Sprintf("Started proxy listening on %s -> %s (max conns: %d)", p.ListenAddr, p.TargetAddr, p.MaxConns))

	p.wg.Add(1)
	go p.acceptLoop()
	return nil
}

// Stop stops the proxy.
func (p *ProxyInstance) Stop() {
	p.startMu.Lock()
	defer p.startMu.Unlock()

	if p.Stats.GetStatus() != "Running" {
		return
	}

	p.log.Info(p.ID, "Stopping proxy")

	// Ensure the listener is closed first to immediately free the port
	if p.listener != nil {
		p.listener.Close()
	}

	// Then cancel the context to signal goroutines to exit
	p.cancel()

	if p.connPool != nil {
		p.connPool.Close()
	}

	if p.poller != nil {
		p.poller.Stop()
	}

	if p.healthChecker != nil {
		p.healthChecker.Stop()
	}

	if p.recoveryManager != nil {
		p.recoveryManager.Stop()
	}

	// Wait for all goroutines to finish
	p.wg.Wait()

	p.Stats.setStatus("Stopped")
}

func (p *ProxyInstance) acceptLoop() {
	defer p.wg.Done()

	const (
		initialBackoff = 100 * time.Millisecond
		maxBackoff     = 5 * time.Second
	)
	backoff := initialBackoff
	consecutiveErrors := 0

acceptLoop:
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.ctx.Done():
				return // Normal shutdown
			default:
				p.log.Error(p.ID, fmt.Sprintf("Accept error: %v", err))
				consecutiveErrors++
				// Exponential backoff to avoid spinning on persistent errors.
				if consecutiveErrors > 10 {
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}
				time.Sleep(backoff)
				continue
			}
		}

		// Reset backoff on successful accept.
		backoff = initialBackoff
		consecutiveErrors = 0

		configureTCPConn(conn)

		// Check GLOBAL connection limit first (system-wide across all proxies).
		if !globalLimiter.acquire() {
			conn.Close()
			p.log.Warn(p.ID, "Global connection limit reached, dropping connection")
			continue acceptLoop
		}

		// Check per-proxy connection limit.
		if !tryAcquireConnSlot(p.connSem) {
			conn.Close()
			globalLimiter.release()
			p.log.Info(p.ID, "Connection limit reached, dropping connection")
			continue
		}

		p.wg.Add(1)
		go p.handleClient(conn, p.connSem)
	}
}

func configureTCPConn(conn net.Conn) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}

	_ = tcpConn.SetKeepAlive(true)
	_ = tcpConn.SetKeepAlivePeriod(30 * time.Second)
	_ = tcpConn.SetNoDelay(true)
}

func (p *ProxyInstance) handleClient(clientConn net.Conn, sem chan struct{}) {
	defer p.wg.Done()
	defer clientConn.Close()

	// Release global connection counter
	defer globalLimiter.release()

	// Track active connections for this proxy
	p.Stats.ActiveConns.Add(1)
	defer p.Stats.ActiveConns.Add(-1)

	// Release semaphore slot when done
	defer func() { <-sem }()

	// Track the device connection
	if p.deviceTracker != nil {
		p.deviceTracker.TrackConnection(clientConn, p.ID)
	}

	for {
		// Check context
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		// Use a generous idle timeout for the client connection (e.g., 5 minutes)
		// rather than the strict ReadTimeout (which is for target device reads)
		idleTimeout := 5 * time.Minute
		if err := clientConn.SetReadDeadline(time.Now().Add(idleTimeout)); err != nil {
			p.log.Error(p.ID, fmt.Sprintf("SetReadDeadline failed: %v", err))
			return
		}
		reqFrame, err := modbus.ReadFrame(clientConn)
		if err != nil {
			if err != io.EOF {
				p.log.Info(p.ID, fmt.Sprintf("Client read error: %v", err))
			}
			return
		}

		// Debug: Log incoming Modbus request. Guard the formatting call so we
		// skip the (expensive) %X sprintf on every request when DEBUG is off,
		// which is the production default.
		if p.log.IsDebugEnabled() {
			p.log.Debug(p.ID, fmt.Sprintf("Received Modbus request: %X (%d bytes)", reqFrame, len(reqFrame)))
		}

		// Serve reads from the cache when one is enabled. This runs before the
		// circuit breaker on purpose: when the target is unreachable, recent
		// data is more useful to the client than an exception, and the TTL
		// bounds how long that can go on.
		cacheKey, cacheUnit, cacheable := modbus.RequestCacheKey(reqFrame)
		if p.cache != nil && cacheable {
			if p.poller != nil {
				p.poller.Track(cacheKey, cacheUnit, reqFrame)
			}
			if cached, hit := p.cache.Get(cacheKey); hit {
				clientTxID, _ := modbus.FrameTxID(reqFrame)
				modbus.SetFrameTxID(cached, clientTxID)
				p.Stats.Requests.Add(1)
				if _, err := clientConn.Write(cached); err != nil {
					p.log.Error(p.ID, fmt.Sprintf("Write cached response error: %v", err))
					return
				}
				continue
			}
		}

		// Check circuit breaker BEFORE forwarding
		if !p.circuitBreaker.AllowRequest() {
			p.log.Error(p.ID, "Circuit breaker is OPEN, rejecting request")
			p.Stats.Errors.Add(1)
			// Send error response to client
			// Modbus exception: Gateway Target Device Failed to Respond
			exceptionResp := modbus.CreateExceptionResponse(reqFrame, 0x0B)
			if _, writeErr := clientConn.Write(exceptionResp); writeErr != nil {
				p.log.Error(p.ID, fmt.Sprintf("Write exception response error: %v", writeErr))
				return
			}
			continue
		}

		// Generate unique request ID and track start time
		reqID := p.getNextRequestID()
		p.enhancedStats.RecordRequestStart(reqID)

		var respFrame []byte
		var errFwd error

		forwardStart := time.Now()

		// Route to the appropriate forwarding function based on protocol.
		respFrame, errFwd = p.forwardClientRequest(reqFrame)

		// Record completion
		bytesRead := len(reqFrame)
		bytesWritten := 0
		if errFwd != nil {
			p.log.Error(p.ID, fmt.Sprintf("Forward error: %v", errFwd))
			p.Stats.Errors.Add(1)
			p.circuitBreaker.RecordFailure()
			p.enhancedStats.RecordRequestComplete(reqID, bytesRead, 0, errFwd)
			exceptionResp := modbus.CreateExceptionResponse(reqFrame, 0x0B)
			if _, writeErr := clientConn.Write(exceptionResp); writeErr != nil {
				p.log.Error(p.ID, fmt.Sprintf("Write exception response error: %v", writeErr))
				return
			}
			continue
		}
		p.Stats.Requests.Add(1)
		p.circuitBreaker.RecordSuccess()
		if p.cache != nil {
			if cacheable {
				// Never cache an exception: it describes a moment, not a value.
				if !modbus.IsExceptionResponse(respFrame) {
					p.cache.SetForUnit(cacheKey, cacheUnit, respFrame)
				}
			} else if unitID, fc, ok := modbus.FrameUnitAndFunction(reqFrame); ok && modbus.IsWriteFunction(fc) {
				// A write may have changed any register of that unit, and the
				// frame does not say which cached reads it touches.
				p.cache.InvalidateUnit(unitID)
			}
		}
		bytesWritten = len(respFrame)
		p.enhancedStats.RecordRequestComplete(reqID, bytesRead, bytesWritten, nil)
		if p.adaptiveTimeout != nil {
			p.adaptiveTimeout.Record(time.Since(forwardStart))
		}

		// Debug: Log Modbus response (guarded — see request-side comment).
		if p.log.IsDebugEnabled() {
			p.log.Debug(p.ID, fmt.Sprintf("Sending Modbus response: %X (%d bytes)", respFrame, len(respFrame)))
		}

		if _, err := clientConn.Write(respFrame); err != nil {
			p.log.Error(p.ID, fmt.Sprintf("Write response error: %v", err))
			return
		}
	}
}

// forwardClientRequest routes a client request to the right forwarding path.
// The background poller uses it too, so a refreshed register goes over exactly
// the same wire path — pacing, retries and split reads included — as a live
// client read.
func (p *ProxyInstance) forwardClientRequest(reqFrame []byte) ([]byte, error) {
	switch {
	case p.Protocol == "rtu-tcp":
		return p.forwardRequestRTU(reqFrame)
	case p.MaxReadSize > 0 && modbus.IsReadRequest(reqFrame):
		return p.handleSplitRead(reqFrame)
	default:
		return p.forwardRequest(reqFrame)
	}
}

func (p *ProxyInstance) handleSplitRead(reqFrame []byte) ([]byte, error) {
	// One budget for the whole client request, not one per chunk.
	deadline := time.Now().Add(p.requestBudget())

	txID, unitID, fc, startAddr, quantity, err := modbus.ParseReadRequest(reqFrame)
	if err != nil {
		// Malformed request, just forward it and let target fail or fail here
		return p.forwardRequest(reqFrame)
	}

	// If quantity is within limits, forward normally
	if int(quantity) <= p.MaxReadSize {
		return p.forwardRequestBefore(reqFrame, deadline)
	}

	expectedBytes := int(quantity) * 2 // 2 bytes per register
	aggregatedData := make([]byte, 0, expectedBytes)
	remaining := quantity
	currentAddr := startAddr

	for remaining > 0 {
		chunkSize := uint16(p.MaxReadSize)
		if remaining < chunkSize {
			chunkSize = remaining
		}

		// Create sub-request with TxID=0 (target doesn't care)
		subReq := modbus.CreateReadRequest(0, unitID, fc, currentAddr, chunkSize)

		// Forward request under the shared deadline
		subResp, err := p.forwardRequestBefore(subReq, deadline)
		if err != nil {
			return nil, err
		}

		// Parse response
		data, err := modbus.ParseReadResponse(subResp)
		if err != nil {
			// Check if it's an exception
			if len(subResp) > 7 && (subResp[7]&0x80) != 0 {
				// It is an exception. Return it immediately (with corrected TxID).
				subResp[0] = byte(txID >> 8)
				subResp[1] = byte(txID)
				return subResp, nil
			}
			return nil, fmt.Errorf("split read failed: %v", err)
		}

		if len(data) != int(chunkSize)*2 {
			return nil, fmt.Errorf("split read received unexpected data length: got %d, want %d", len(data), chunkSize*2)
		}

		aggregatedData = append(aggregatedData, data...)
		remaining -= chunkSize
		currentAddr += chunkSize
	}

	// Construct final response
	respFrame, err := modbus.CreateReadResponse(txID, unitID, fc, aggregatedData)
	if err != nil {
		return nil, fmt.Errorf("create read response: %w", err)
	}
	return respFrame, nil
}

// retryBackoff calculates exponential backoff with jitter for retry attempts.
func (p *ProxyInstance) retryBackoff(attempt int) time.Duration {
	base := time.Duration(1<<min(uint(attempt-1), 10)) * 100 * time.Millisecond
	if base > 30*time.Second {
		base = 30 * time.Second
	}
	jitter := time.Duration(rand.Int63n(int64(base) / 2))
	return base + jitter
}

// brokenMarker returns a function that flags a pooled connection as unusable
// so it is closed instead of handed to the next request.
func brokenMarker(conn net.Conn) func() {
	if wc, ok := conn.(*pool.WrappedConn); ok {
		return wc.MarkBroken
	}
	return func() {}
}

// proxyContext returns the proxy's lifecycle context, falling back to a
// background context when the proxy has not been started.
func (p *ProxyInstance) proxyContext() context.Context {
	if p.ctx != nil {
		return p.ctx
	}
	return context.Background()
}

// forwardRequest sends a request to the target and returns the response.
//
// The whole call — every retry and backoff included — is bounded by
// requestBudget. Without that bound a slow target can keep the proxy busy long
// after the client gave up waiting; the client then reuses its connection for
// the next request and receives the late answer to the previous one, which
// desynchronises every transaction that follows.
func (p *ProxyInstance) forwardRequest(req []byte) ([]byte, error) {
	return p.forwardRequestBefore(req, time.Now().Add(p.requestBudget()))
}

// forwardRequestBefore forwards a request under an externally supplied
// deadline. Split reads share one deadline across all their chunks so that the
// budget covers the client's request as a whole.
func (p *ProxyInstance) forwardRequestBefore(req []byte, deadline time.Time) ([]byte, error) {
	clientTxID, _ := modbus.FrameTxID(req)
	budget := time.Until(deadline)

	var lastErr error
	for attempt := 0; attempt <= p.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := p.retryBackoff(attempt)
			if time.Now().Add(backoff).After(deadline) {
				break
			}
			time.Sleep(backoff)
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}

		resp, err := p.forwardAttempt(req, clientTxID, remaining)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		return nil, fmt.Errorf("request budget of %v exhausted before an attempt could run", budget)
	}
	return nil, fmt.Errorf("failed within budget %v (max %d retries): %w", budget, p.MaxRetries, lastErr)
}

// forwardAttempt performs a single request/response exchange with the target.
//
// The client's transaction ID is replaced by a proxy-owned one for the hop to
// the target and restored on the response. Responses that do not carry the
// expected ID are discarded rather than forwarded, so a late answer from an
// earlier transaction can never be mistaken for the current one.
func (p *ProxyInstance) forwardAttempt(req []byte, clientTxID uint16, remaining time.Duration) ([]byte, error) {
	readTimeout, connectTimeout := p.currentTimeouts()
	if readTimeout > remaining {
		readTimeout = remaining
	}
	if connectTimeout > remaining {
		connectTimeout = remaining
	}

	ctx, cancel := context.WithTimeout(p.proxyContext(), remaining)
	defer cancel()

	if err := p.pacer.wait(ctx); err != nil {
		return nil, err
	}

	rawConn, err := p.connPool.Get(ctx)
	if err != nil {
		return nil, err
	}
	markBroken := brokenMarker(rawConn)

	out := make([]byte, len(req))
	copy(out, req)
	txID := p.nextTargetTxID()
	modbus.SetFrameTxID(out, txID)

	if err := rawConn.SetWriteDeadline(time.Now().Add(connectTimeout)); err != nil {
		markBroken()
		rawConn.Close()
		return nil, err
	}
	if _, err := rawConn.Write(out); err != nil {
		markBroken()
		rawConn.Close()
		return nil, err
	}

	resp, err := p.readMatchingResponse(rawConn, out, txID, time.Now().Add(readTimeout))
	if err != nil {
		// A response may still be in flight; it would land in front of the next
		// request on this connection, so the connection must not be reused.
		markBroken()
		rawConn.Close()
		return nil, err
	}

	rawConn.Close() // Returns the connection to the pool
	modbus.SetFrameTxID(resp, clientTxID)
	return resp, nil
}

// forwardRequestRTU converts a Modbus TCP frame to RTU, sends it to the target,
// reads the RTU response, and converts it back to a TCP frame.
// Used when Protocol == "rtu-tcp".
func (p *ProxyInstance) forwardRequestRTU(tcpReq []byte) ([]byte, error) {
	if len(tcpReq) < 8 {
		return nil, fmt.Errorf("rtu-tcp: tcp request too short (%d bytes)", len(tcpReq))
	}
	txID := uint16(tcpReq[0])<<8 | uint16(tcpReq[1])
	fc := tcpReq[7]

	rtuReq, err := modbus.TCPToRTU(tcpReq)
	if err != nil {
		return nil, fmt.Errorf("rtu-tcp: tcp→rtu conversion: %w", err)
	}

	unitID := tcpReq[6]
	budget := p.requestBudget()
	deadline := time.Now().Add(budget)

	var lastErr error
	for attempt := 0; attempt <= p.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := p.retryBackoff(attempt)
			if time.Now().Add(backoff).After(deadline) {
				break
			}
			time.Sleep(backoff)
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}

		tcpResp, err := p.forwardAttemptRTU(rtuReq, unitID, fc, txID, remaining)
		if err == nil {
			return tcpResp, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		return nil, fmt.Errorf("rtu-tcp: request budget of %v exhausted before an attempt could run", budget)
	}
	return nil, fmt.Errorf("rtu-tcp: failed within budget %v (max %d retries): %w", budget, p.MaxRetries, lastErr)
}

// forwardAttemptRTU performs a single RTU-over-TCP exchange. RTU frames carry
// no transaction ID, so the response is matched on slave address and function
// code instead; a mismatch means the stream is out of sync and the connection
// is dropped rather than reused.
func (p *ProxyInstance) forwardAttemptRTU(rtuReq []byte, unitID, fc byte, txID uint16, remaining time.Duration) ([]byte, error) {
	readTimeout, connectTimeout := p.currentTimeouts()
	if readTimeout > remaining {
		readTimeout = remaining
	}
	if connectTimeout > remaining {
		connectTimeout = remaining
	}

	ctx, cancel := context.WithTimeout(p.proxyContext(), remaining)
	defer cancel()

	if err := p.pacer.wait(ctx); err != nil {
		return nil, err
	}

	rawConn, err := p.connPool.Get(ctx)
	if err != nil {
		return nil, err
	}
	markBroken := brokenMarker(rawConn)

	if err := rawConn.SetWriteDeadline(time.Now().Add(connectTimeout)); err != nil {
		markBroken()
		rawConn.Close()
		return nil, err
	}
	if _, err := rawConn.Write(rtuReq); err != nil {
		markBroken()
		rawConn.Close()
		return nil, err
	}
	if err := rawConn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		markBroken()
		rawConn.Close()
		return nil, err
	}

	rtuResp, err := modbus.ReadRTUFrame(rawConn, fc)
	if err != nil {
		markBroken()
		rawConn.Close()
		return nil, err
	}

	if rtuResp[0] != unitID || (rtuResp[1] != fc && rtuResp[1] != fc|0x80) {
		atomic.AddInt64(&p.staleResponses, 1)
		p.log.Warn(p.ID, fmt.Sprintf("Discarding stale RTU response (slave %d fc 0x%02X, expected slave %d fc 0x%02X)", rtuResp[0], rtuResp[1], unitID, fc))
		markBroken()
		rawConn.Close()
		return nil, fmt.Errorf("rtu response mismatch: got slave %d fc 0x%02X, want slave %d fc 0x%02X", rtuResp[0], rtuResp[1], unitID, fc)
	}

	rawConn.Close()

	return modbus.RTUToTCP(rtuResp, txID)
}
