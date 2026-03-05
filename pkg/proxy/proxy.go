package proxy

import (
	"context"
	"fmt"
	"io"
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

// getNextRequestID generates a unique request ID
func (p *ProxyInstance) getNextRequestID() int64 {
	return atomic.AddInt64(&p.requestID, 1)
}

// min returns the minimum of two unsigned integers
func min(a, b uint) uint {
	if a < b {
		return a
	}
	return b
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
	MaxConns          int // Maximum concurrent connections (0 = unlimited)

	listener  net.Listener
	connPool  *pool.Pool
	splitMu   sync.Mutex    // Only for split read operations
	connSem   chan struct{} // Semaphore for limiting concurrent connections
	connSemMu sync.Mutex    // Protects connSem initialization

	log           *logger.Logger
	deviceTracker *devices.Tracker
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup

	// Enhanced features
	circuitBreaker *CircuitBreaker
	enhancedStats  *EnhancedStats
	requestID      int64 // Atomic counter for request IDs

	Stats Stats
}

type Stats struct {
	Uptime    time.Duration
	LastStart time.Time
	Requests  atomic.Int64
	Errors    atomic.Int64
	Status    string // "Running", "Stopped", "Error"
}

// NewProxyInstance creates a new proxy.
func NewProxyInstance(id, name, listen, target string, maxReadSize int, l *logger.Logger, tracker *devices.Tracker) *ProxyInstance {
	return &ProxyInstance{
		ID:                id,
		Name:              name,
		ListenAddr:        listen,
		TargetAddr:        target,
		MaxReadSize:       maxReadSize,
		ConnectionTimeout: 5 * time.Second, // Default
		ReadTimeout:       5 * time.Second, // Default
		MaxRetries:        3,               // Default
		MaxConns:          500,             // Default: limit concurrent connections
		log:               l,
		deviceTracker:     tracker,
		Stats:             Stats{Status: "Stopped"},
	}
}

// Start starts the proxy.
func (p *ProxyInstance) Start() error {
	if p.Stats.Status == "Running" {
		return nil
	}

	// Validate addresses before starting
	validator := middleware.NewValidator()
	if err := validator.ValidatePort(p.ListenAddr); err != nil {
		p.Stats.Status = "Error"
		return fmt.Errorf("invalid listen address: %w", err)
	}
	if err := validator.ValidatePort(p.TargetAddr); err != nil {
		p.Stats.Status = "Error"
		return fmt.Errorf("invalid target address: %w", err)
	}

	l, err := net.Listen("tcp", p.ListenAddr)
	if err != nil {
		p.Stats.Status = "Error"
		p.log.Error(p.ID, fmt.Sprintf("Failed to listen: %v", err))
		return err
	}
	p.listener = l

	// Create connection pool for target with optimized settings
	poolCfg := pool.Config{
		InitialSize:    2,                // Optimized: Better initial capacity
		MaxSize:        20,               // Optimized: Increased for higher concurrency
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
		p.Stats.Status = "Error"
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

	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.Stats.Status = "Running"
	p.Stats.LastStart = time.Now()

	p.log.Info(p.ID, fmt.Sprintf("Started proxy listening on %s -> %s (max conns: %d)", p.ListenAddr, p.TargetAddr, p.MaxConns))

	p.wg.Add(1)
	go p.acceptLoop()
	return nil
}

// Stop stops the proxy.
func (p *ProxyInstance) Stop() {
	if p.Stats.Status != "Running" {
		return
	}

	p.log.Info(p.ID, "Stopping proxy")
	p.cancel()
	if p.listener != nil {
		p.listener.Close()
	}
	if p.connPool != nil {
		p.connPool.Close()
	}

	// Wait for all goroutines to finish
	p.wg.Wait()

	p.Stats.Status = "Stopped"
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

		// Check connection limit (if configured)
		p.connSemMu.Lock()
		sem := p.connSem
		p.connSemMu.Unlock()

		if sem != nil {
			select {
			case sem <- struct{}{}: // Acquire semaphore slot
				// Proceed with connection
			case <-time.After(5 * time.Second):
				// Connection limit reached, drop this connection
				conn.Close()
				p.log.Info(p.ID, "Connection limit reached, dropping connection")
				continue
			}
		}

		p.wg.Add(1)
		go p.handleClient(conn, sem)
	}
}

func (p *ProxyInstance) handleClient(clientConn net.Conn, sem chan struct{}) {
	defer p.wg.Done()
	defer clientConn.Close()

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

		if err := clientConn.SetReadDeadline(time.Now().Add(p.ReadTimeout)); err != nil {
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

		// Check if splitting is needed
		if p.MaxReadSize > 0 && modbus.IsReadRequest(reqFrame) {
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
			return
		}
		p.Stats.Requests.Add(1)
		p.circuitBreaker.RecordSuccess()
		bytesWritten = len(respFrame)
		p.enhancedStats.RecordRequestComplete(reqID, bytesRead, bytesWritten, nil)

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

	// Split logic - use lock to ensure atomicity of the split operation
	p.splitMu.Lock()
	defer p.splitMu.Unlock()

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

// forwardRequest sends a request to the target and returns the response.
// Uses connection pool for concurrent request handling.
func (p *ProxyInstance) forwardRequest(req []byte) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= p.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with cap to prevent overflow
			backoffDuration := time.Duration(1<<min(uint(attempt-1), 10)) * 100 * time.Millisecond
			if backoffDuration > 30*time.Second {
				backoffDuration = 30 * time.Second
			}
			time.Sleep(backoffDuration)
		}

		// Get connection from pool
		conn, err := p.connPool.Get(p.ctx)
		if err != nil {
			lastErr = err
			continue
		}

		// Try write
		if err := conn.SetWriteDeadline(time.Now().Add(p.ConnectionTimeout)); err != nil {
			conn.Close()
			lastErr = err
			continue
		}

		if _, err := conn.Write(req); err != nil {
			conn.Close()
			lastErr = err
			continue
		}

		// Try read
		if err := conn.SetReadDeadline(time.Now().Add(p.ReadTimeout)); err != nil {
			conn.Close()
			lastErr = err
			continue
		}

		resp, err := modbus.ReadFrame(conn)
		if err != nil {
			conn.Close()
			lastErr = err
			continue
		}

		// Return connection to pool
		conn.Close()
		return resp, nil
	}

	return nil, fmt.Errorf("failed after %d retries: %w", p.MaxRetries, lastErr)
}
