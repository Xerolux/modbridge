// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package pool

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

var (
	// ErrPoolClosed is returned when the pool is closed.
	ErrPoolClosed = errors.New("connection pool is closed")
	// ErrPoolExhausted is returned when the pool is exhausted.
	ErrPoolExhausted = errors.New("connection pool exhausted")
)

// Config holds pool configuration.
type Config struct {
	// Initial size of the pool
	InitialSize int
	// Maximum size of the pool
	MaxSize int
	// Maximum idle time before connection is closed
	MaxIdleTime time.Duration
	// Timeout for acquiring a connection
	AcquireTimeout time.Duration
	// Dialer function
	Dialer func(context.Context) (net.Conn, error)
}

// Pool is a connection pool.
type Pool struct {
	mu      sync.Mutex
	conns   chan *poolConn
	factory func(context.Context) (net.Conn, error)

	maxIdleTime    time.Duration
	acquireTimeout time.Duration

	closed  bool
	size    int
	maxSize int
	ctx     context.Context
	cancel  context.CancelFunc
}

type poolConn struct {
	conn     net.Conn
	lastUsed time.Time
	inUse    bool
}

// NewPool creates a new connection pool.
func NewPool(cfg Config) (*Pool, error) {
	if cfg.InitialSize < 0 {
		cfg.InitialSize = 0
	}
	if cfg.MaxSize < 1 {
		cfg.MaxSize = 10
	}
	if cfg.MaxIdleTime == 0 {
		cfg.MaxIdleTime = 5 * time.Minute
	}
	if cfg.AcquireTimeout == 0 {
		cfg.AcquireTimeout = 30 * time.Second
	}

	p := &Pool{
		conns:          make(chan *poolConn, cfg.MaxSize),
		factory:        cfg.Dialer,
		maxIdleTime:    cfg.MaxIdleTime,
		acquireTimeout: cfg.AcquireTimeout,
		maxSize:        cfg.MaxSize,
	}
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Pre-populate pool
	for i := 0; i < cfg.InitialSize; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		conn, err := p.factory(ctx)
		cancel()

		if err != nil {
			close(p.conns)
			for pc := range p.conns {
				pc.conn.Close()
			}
			p.cancel()
			return nil, err
		}

		p.conns <- &poolConn{
			conn:     conn,
			lastUsed: time.Now(),
		}
		p.size++
	}

	go p.cleanup()

	return p, nil
}

// Get acquires a connection from the pool.
func (p *Pool) Get(ctx context.Context) (net.Conn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}
	p.mu.Unlock()

	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.acquireTimeout)
		defer cancel()
		deadline, _ = ctx.Deadline()
	}

	timeout := time.Until(deadline)
	if timeout < 0 {
		timeout = 0
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case pc, ok := <-p.conns:
			if !ok {
				return nil, ErrPoolClosed
			}

			if time.Since(pc.lastUsed) > p.maxIdleTime {
				pc.conn.Close()
				p.mu.Lock()
				p.size--
				p.mu.Unlock()
				continue
			}

			if !isConnHealthy(pc.conn) {
				pc.conn.Close()
				p.mu.Lock()
				p.size--
				p.mu.Unlock()
				continue
			}

			pc.inUse = true
			return &WrappedConn{
				Conn: pc.conn,
				pool: p,
				pc:   pc,
			}, nil
		default:
		}

		// Try to create a new connection before waiting.
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, ErrPoolClosed
		}
		if p.size < p.maxSize {
			p.size++
			p.mu.Unlock()

			conn, err := p.factory(ctx)
			if err != nil {
				p.mu.Lock()
				p.size--
				p.mu.Unlock()
				return nil, err
			}

			pc := &poolConn{
				conn:     conn,
				lastUsed: time.Now(),
				inUse:    true,
			}

			return &WrappedConn{
				Conn: conn,
				pool: p,
				pc:   pc,
			}, nil
		}
		p.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, ErrPoolExhausted
		case pc, ok := <-p.conns:
			if !ok {
				return nil, ErrPoolClosed
			}
			if time.Since(pc.lastUsed) > p.maxIdleTime {
				pc.conn.Close()
				p.mu.Lock()
				p.size--
				p.mu.Unlock()
				continue
			}
			if !isConnHealthy(pc.conn) {
				pc.conn.Close()
				p.mu.Lock()
				p.size--
				p.mu.Unlock()
				continue
			}

			pc.inUse = true
			return &WrappedConn{
				Conn: pc.conn,
				pool: p,
				pc:   pc,
			}, nil
		}
	}
}

// put returns a connection to the pool.
func (p *Pool) put(pc *poolConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		pc.conn.Close()
		p.size--
		return
	}

	pc.lastUsed = time.Now()
	pc.inUse = false

	select {
	case p.conns <- pc:
		// Successfully returned to pool
	default:
		// Pool is full, close the connection
		pc.conn.Close()
		p.size--
	}
}

// Close closes the pool and all connections.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.cancel()
	close(p.conns)

	for pc := range p.conns {
		pc.conn.Close()
	}

	return nil
}

// Stats returns pool statistics.
func (p *Pool) Stats() PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	return PoolStats{
		TotalConns:  p.size,
		IdleConns:   len(p.conns),
		ActiveConns: p.size - len(p.conns),
		MaxSize:     p.maxSize,
	}
}

// PreWarm creates new connections up to the specified count.
// This is useful after a target recovers to avoid latency spikes.
func (p *Pool) PreWarm(ctx context.Context, count int) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrPoolClosed
	}
	available := p.maxSize - p.size
	p.mu.Unlock()

	if available < count {
		count = available
	}
	if count <= 0 {
		return nil
	}

	var firstErr error
	for i := 0; i < count; i++ {
		conn, err := p.factory(ctx)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			conn.Close()
			return ErrPoolClosed
		}
		pc := &poolConn{
			conn:     conn,
			lastUsed: time.Now(),
		}
		p.size++
		p.mu.Unlock()

		p.conns <- pc
	}

	return firstErr
}

// PoolStats represents pool statistics.
type PoolStats struct {
	TotalConns  int
	IdleConns   int
	ActiveConns int
	MaxSize     int
}

// cleanup periodically cleans up idle connections.
func (p *Pool) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
		}

		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return
		}
		p.mu.Unlock()

		count := len(p.conns)
		for i := 0; i < count; i++ {
			select {
			case pc := <-p.conns:
				if time.Since(pc.lastUsed) > p.maxIdleTime {
					pc.conn.Close()
					p.mu.Lock()
					p.size--
					p.mu.Unlock()
				} else {
					p.conns <- pc
				}
			default:
				goto done
			}
		}
	done:
	}
}

// WrappedConn wraps a connection to return it to the pool on close.
type WrappedConn struct {
	net.Conn
	pool   *Pool
	pc     *poolConn
	once   sync.Once
	broken bool
}

// MarkBroken marks the connection as broken so it will be discarded instead of returned to the pool.
func (w *WrappedConn) MarkBroken() {
	w.broken = true
}

func (w *WrappedConn) Close() error {
	w.once.Do(func() {
		if w.broken {
			w.pool.mu.Lock()
			w.pc.conn.Close()
			w.pool.size--
			w.pool.mu.Unlock()
			return
		}
		w.pool.put(w.pc)
	})
	return nil
}

func isConnHealthy(conn net.Conn) bool {
	if conn == nil {
		return false
	}
	// Check connection health non-destructively.
	// Reading data from the connection to test liveness destroys pending
	// Modbus frames. Instead, rely on TCP keepalive probes and let the
	// pool detect broken connections on the next read/write attempt.
	// Set a short deadline to validate the write side of the connection.
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return true
	}
	if err := tcpConn.SetWriteDeadline(time.Now().Add(1 * time.Millisecond)); err != nil {
		return false
	}
	tcpConn.SetWriteDeadline(time.Time{})
	return true
}
