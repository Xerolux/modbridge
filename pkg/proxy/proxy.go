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

// Global connection tracking to prevent system-wide socket exhaustion
var (
	// globalActiveConnections tracks total connections across ALL proxies
	globalActiveConnections int64
	// globalMaxConnections is the system-wide limit (0 = unlimited)
	globalMaxConnections int64 = 10000 // Default: 10K connections across all proxies
)

// setGlobalMaxConnections updates the global connection limit
func SetGlobalMaxConnections(max int64) {
	atomic.StoreInt64(&globalMaxConnections, max)
}

// getGlobalMaxConnections returns the current global connection limit
func GetGlobalMaxConnections() int64 {
	return atomic.LoadInt64(&globalMaxConnections)
}

// getNextRequestID generates a unique request ID
func (p *ProxyInstance) getNextRequestID() int64 {
	return atomic.AddInt64(&p.requestID, 1)
}

func tryAcquireConnSlot(sem chan struct{}) bool {
	if sem == nil {
		return true
	}
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
	MaxConns          int    // Maximum concurrent connections (0 = unlimited)
	Protocol          string // "tcp" (default) or "rtu-tcp"

	listener  net.Listener
	connPool  *pool.Pool
	connSem   chan struct{} // Semaphore for limiting concurrent connections
	connSemMu sync.Mutex    // Protects connSem initialization

	log           *logger.Logger
	deviceTracker *devices.Tracker
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup

	// Enhanced features
	circuitBreaker  *CircuitBreaker
	enhancedStats   *EnhancedStats
	requestID       int64
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

	return &ProxyInstance{
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
}

// Start starts the proxy.
func (p *ProxyInstance) Start() error {
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

	// Create connection pool for target with optimized settings
	poolCfg := pool.Config{
		InitialSize:    1,                // Modbus targets usually accept 1-3 connections max
		MaxSize:        10,               // Moderate concurrency limit to avoid dropping connections
		MaxIdleTime:    10 * time.Minute, // Optimized: Longer idle time for reusability
		AcquireTimeout: p.ConnectionTimeout,
		Dialer: func(ctx context.Context) (net.Conn, error) {
			d := net.Dialer{
				Timeout:   p.ConnectionTimeout,
				KeepAlive: 30 * time.Second, // Optimized: TCP keep-alive
			}
			return d.DialContext(ctx, "tcp", p.TargetAddr)
		},
	}

	p.connPool, err = pool.NewPool(poolCfg)
	if err != nil {
		p.listener.Close()
		p.Stats.setStatus("Error")
		p.log.Error(p.ID, fmt.Sprintf("Failed to create connection pool: %v", err))
		return err
	}

	// Initialize connection semaphore if MaxConns > 0
	p.connSemMu.Lock()
	if p.MaxConns > 0 {
		p.connSem = make(chan struct{}, p.MaxConns)
	}
	p.connSemMu.Unlock()

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
			p.recoveryManager.AddTask(p.TargetAddr, 10)
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

	p.recoveryManager = NewRecoveryManager(DefaultRecoveryConfig(), func(target string) error {
		p.log.Info(p.ID, fmt.Sprintf("Recovery check: dialing %s", target))
		return nil
	})

	// Initialize adaptive timeouts
	p.adaptiveTimeout = NewAdaptiveTimeout(p.ReadTimeout, p.ConnectionTimeout)

	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.Stats.setStatus("Running")
	p.Stats.SetLastStart(time.Now())

	p.log.Info(p.ID, fmt.Sprintf("Started proxy listening on %s -> %s (max conns: %d)", p.ListenAddr, p.TargetAddr, p.MaxConns))

	p.wg.Add(1)
	go p.acceptLoop()
	return nil
}

// Stop stops the proxy.
func (p *ProxyInstance) Stop() {
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

	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.ctx.Done():
				return // Normal shutdown
			default:
				p.log.Error(p.ID, fmt.Sprintf("Accept error: %v", err))
				// Backoff slightly to avoid spinning
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		configureTCPConn(conn)

		// Check GLOBAL connection limit first (system-wide across all proxies).
		// Use Add+check to avoid a CAS retry-loop that drops connections on contention.
		maxConns := atomic.LoadInt64(&globalMaxConnections)
		globalLimitApplied := false
		if maxConns > 0 {
			current := atomic.AddInt64(&globalActiveConnections, 1)
			if current > maxConns {
				atomic.AddInt64(&globalActiveConnections, -1) // undo
				conn.Close()
				p.log.Warn(p.ID, fmt.Sprintf("Global connection limit reached (%d), dropping connection", maxConns))
				continue
			}
			globalLimitApplied = true
		}

		// Check connection limit (if configured)
		p.connSemMu.Lock()
		sem := p.connSem
		p.connSemMu.Unlock()

		if !tryAcquireConnSlot(sem) {
			// Connection limit reached, drop this connection immediately.
			// Keeping accept loop non-blocking improves resiliency during load spikes.
			conn.Close()
			if globalLimitApplied {
				atomic.AddInt64(&globalActiveConnections, -1) // Release global counter
			}
			p.log.Info(p.ID, "Connection limit reached, dropping connection")
			continue
		}

		p.wg.Add(1)
		go p.handleClient(conn, sem, globalLimitApplied)
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

func (p *ProxyInstance) handleClient(clientConn net.Conn, sem chan struct{}, globalLimitApplied bool) {
	defer p.wg.Done()
	defer clientConn.Close()

	// Track active connections for this proxy
	p.Stats.ActiveConns.Add(1)
	defer p.Stats.ActiveConns.Add(-1)

	// Release global connection counter only if we incremented it
	if globalLimitApplied {
		defer atomic.AddInt64(&globalActiveConnections, -1)
	}

	// Release semaphore slot when done
	if sem != nil {
		defer func() { <-sem }()
	}

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

		// Debug: Log incoming Modbus request
		p.log.Debug(p.ID, fmt.Sprintf("Received Modbus request: %X (%d bytes)", reqFrame, len(reqFrame)))

		// Check circuit breaker BEFORE forwarding
		if !p.circuitBreaker.AllowRequest() {
			p.log.Error(p.ID, "Circuit breaker is OPEN, rejecting request")
			p.Stats.Errors.Add(1)
			// Send error response to client
			// Modbus exception: Gateway Target Device Failed to Respond
			exceptionResp := modbus.CreateExceptionResponse(reqFrame, 0x0B)
			clientConn.Write(exceptionResp)
			continue
		}

		// Generate unique request ID and track start time
		reqID := p.getNextRequestID()
		p.enhancedStats.RecordRequestStart(reqID)

		var respFrame []byte
		var errFwd error

		forwardStart := time.Now()

		// Route to the appropriate forwarding function based on protocol.
		if p.Protocol == "rtu-tcp" {
			respFrame, errFwd = p.forwardRequestRTU(reqFrame)
		} else if p.MaxReadSize > 0 && modbus.IsReadRequest(reqFrame) {
			respFrame, errFwd = p.handleSplitRead(reqFrame)
		} else {
			respFrame, errFwd = p.forwardRequest(reqFrame)
		}

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
		bytesWritten = len(respFrame)
		p.enhancedStats.RecordRequestComplete(reqID, bytesRead, bytesWritten, nil)
		if p.adaptiveTimeout != nil {
			p.adaptiveTimeout.Record(time.Since(forwardStart))
		}

		// Debug: Log Modbus response
		p.log.Debug(p.ID, fmt.Sprintf("Sending Modbus response: %X (%d bytes)", respFrame, len(respFrame)))

		if _, err := clientConn.Write(respFrame); err != nil {
			p.log.Error(p.ID, fmt.Sprintf("Write response error: %v", err))
			return
		}
	}
}

func (p *ProxyInstance) handleSplitRead(reqFrame []byte) ([]byte, error) {
	txID, unitID, fc, startAddr, quantity, err := modbus.ParseReadRequest(reqFrame)
	if err != nil {
		// Malformed request, just forward it and let target fail or fail here
		return p.forwardRequest(reqFrame)
	}

	// If quantity is within limits, forward normally
	if int(quantity) <= p.MaxReadSize {
		return p.forwardRequest(reqFrame)
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

		// Forward request
		subResp, err := p.forwardRequest(subReq)
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
	respFrame := modbus.CreateReadResponse(txID, unitID, fc, aggregatedData)
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

// forwardRequest sends a request to the target and returns the response.
func (p *ProxyInstance) forwardRequest(req []byte) ([]byte, error) {
	var lastErr error

	readTimeout := p.ReadTimeout
	connectTimeout := p.ConnectionTimeout
	if p.adaptiveTimeout != nil {
		readTimeout = p.adaptiveTimeout.GetReadTimeout()
		connectTimeout = p.adaptiveTimeout.GetConnectTimeout()
	}

	for attempt := 0; attempt <= p.MaxRetries; attempt++ {
		if attempt > 0 {
			backoffDuration := p.retryBackoff(attempt)
			time.Sleep(backoffDuration)
		}

		rawConn, err := p.connPool.Get(p.ctx)
		if err != nil {
			lastErr = err
			continue
		}

		var markBroken func()
		if wc, ok := rawConn.(*pool.WrappedConn); ok {
			markBroken = wc.MarkBroken
		} else {
			markBroken = func() {}
		}

		// Try write
		if err := rawConn.SetWriteDeadline(time.Now().Add(connectTimeout)); err != nil {
			markBroken()
			rawConn.Close()
			lastErr = err
			continue
		}

		if _, err := rawConn.Write(req); err != nil {
			markBroken()
			rawConn.Close()
			lastErr = err
			continue
		}

		// Try read
		if err := rawConn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			markBroken()
			rawConn.Close()
			lastErr = err
			continue
		}

		resp, err := modbus.ReadFrame(rawConn)
		if err != nil {
			markBroken()
			rawConn.Close()
			lastErr = err
			continue
		}

		// Return connection to pool
		rawConn.Close()
		return resp, nil
	}

	return nil, fmt.Errorf("failed after %d retries: %w", p.MaxRetries, lastErr)
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

	var lastErr error
	rtuReadTimeout := p.ReadTimeout
	rtuConnectTimeout := p.ConnectionTimeout
	if p.adaptiveTimeout != nil {
		rtuReadTimeout = p.adaptiveTimeout.GetReadTimeout()
		rtuConnectTimeout = p.adaptiveTimeout.GetConnectTimeout()
	}
	for attempt := 0; attempt <= p.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := p.retryBackoff(attempt)
			time.Sleep(backoff)
		}

		rawConn, err := p.connPool.Get(p.ctx)
		if err != nil {
			lastErr = err
			continue
		}
		var markBroken func()
		if wc, ok := rawConn.(*pool.WrappedConn); ok {
			markBroken = wc.MarkBroken
		} else {
			markBroken = func() {}
		}

		if err := rawConn.SetWriteDeadline(time.Now().Add(rtuConnectTimeout)); err != nil {
			markBroken()
			rawConn.Close()
			lastErr = err
			continue
		}
		if _, err := rawConn.Write(rtuReq); err != nil {
			markBroken()
			rawConn.Close()
			lastErr = err
			continue
		}
		if err := rawConn.SetReadDeadline(time.Now().Add(rtuReadTimeout)); err != nil {
			markBroken()
			rawConn.Close()
			lastErr = err
			continue
		}
		rtuResp, err := modbus.ReadRTUFrame(rawConn, fc)
		if err != nil {
			markBroken()
			rawConn.Close()
			lastErr = err
			continue
		}
		rawConn.Close()

		tcpResp, err := modbus.RTUToTCP(rtuResp, txID)
		if err != nil {
			lastErr = err
			continue
		}
		return tcpResp, nil
	}

	return nil, fmt.Errorf("rtu-tcp: failed after %d retries: %w", p.MaxRetries, lastErr)
}
