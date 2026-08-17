// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"encoding/binary"
	"modbridge/pkg/logger"
	"modbridge/pkg/modbus"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// startTestProxy starts a proxy in front of the given target address and
// returns it. The caller is responsible for stopping it.
func startTestProxy(t *testing.T, targetAddr string, configure func(*ProxyInstance)) *ProxyInstance {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve proxy port: %v", err)
	}
	proxyAddr := listener.Addr().String()
	listener.Close()

	p := NewProxyInstance("tx-test", "tx-test", proxyAddr, targetAddr, 0, 5, 5, 3, logger.NewNullLogger(100), nil)
	if configure != nil {
		configure(p)
	}
	if err := p.Start(); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	return p
}

// TestForwardRequestDiscardsStaleResponse verifies that a late response to an
// earlier transaction is dropped instead of being handed to the client as the
// answer to the current request.
func TestForwardRequestDiscardsStaleResponse(t *testing.T) {
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock target: %v", err)
	}
	defer targetListener.Close()

	go func() {
		for {
			conn, err := targetListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				for {
					frame, err := modbus.ReadFrame(c)
					if err != nil {
						return
					}
					txID, unitID, fc, _, quantity, err := modbus.ParseReadRequest(frame)
					if err != nil {
						return
					}
					data := make([]byte, quantity*2)
					for i := range data {
						data[i] = 0xAA
					}
					// A late answer to a transaction the proxy has already
					// given up on, followed by the real one.
					stale, _ := modbus.CreateReadResponse(txID-1, unitID, fc, data)
					if _, err := c.Write(stale); err != nil {
						return
					}
					resp, _ := modbus.CreateReadResponse(txID, unitID, fc, data)
					if _, err := c.Write(resp); err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	p := startTestProxy(t, targetListener.Addr().String(), nil)
	defer p.Stop()

	conn, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	const clientTxID = 0x1234
	if _, err := conn.Write(modbus.CreateReadRequest(clientTxID, 1, 3, 0, 4)); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	resp, err := modbus.ReadFrame(conn)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	gotTxID := binary.BigEndian.Uint16(resp[0:2])
	if gotTxID != clientTxID {
		t.Errorf("response transaction ID = 0x%04X, want 0x%04X", gotTxID, clientTxID)
	}
	if got := len(resp); got != 9+8 {
		t.Errorf("response length = %d, want %d", got, 9+8)
	}
	if stale := p.StaleResponses(); stale != 1 {
		t.Errorf("stale responses = %d, want 1", stale)
	}
}

// TestForwardRequestRewritesTargetTransactionID verifies that the proxy uses
// its own transaction IDs towards the target while the client still sees its
// own ID on the response.
func TestForwardRequestRewritesTargetTransactionID(t *testing.T) {
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock target: %v", err)
	}
	defer targetListener.Close()

	var seenTxID uint32 = 0xFFFFFFFF
	go func() {
		for {
			conn, err := targetListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				for {
					frame, err := modbus.ReadFrame(c)
					if err != nil {
						return
					}
					txID, unitID, fc, _, quantity, err := modbus.ParseReadRequest(frame)
					if err != nil {
						return
					}
					atomic.CompareAndSwapUint32(&seenTxID, 0xFFFFFFFF, uint32(txID))
					data := make([]byte, quantity*2)
					resp, _ := modbus.CreateReadResponse(txID, unitID, fc, data)
					if _, err := c.Write(resp); err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	p := startTestProxy(t, targetListener.Addr().String(), nil)
	defer p.Stop()

	conn, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	const clientTxID = 0x4711
	if _, err := conn.Write(modbus.CreateReadRequest(clientTxID, 1, 3, 0, 2)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	resp, err := modbus.ReadFrame(conn)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if got := binary.BigEndian.Uint16(resp[0:2]); got != clientTxID {
		t.Errorf("client received transaction ID 0x%04X, want 0x%04X", got, clientTxID)
	}
	if got := atomic.LoadUint32(&seenTxID); got == clientTxID {
		t.Errorf("target saw the client transaction ID 0x%04X; proxy should assign its own", got)
	}
}

// TestForwardRequestHonoursRequestBudget verifies that a silent target does
// not keep the proxy busy past the configured per-request budget: the client
// gets a Modbus exception in time instead of a late response it no longer
// expects.
func TestForwardRequestHonoursRequestBudget(t *testing.T) {
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock target: %v", err)
	}
	defer targetListener.Close()

	// Accept connections but never answer.
	go func() {
		var held []net.Conn
		defer func() {
			for _, c := range held {
				c.Close()
			}
		}()
		for {
			conn, err := targetListener.Accept()
			if err != nil {
				return
			}
			held = append(held, conn)
		}
	}()

	p := startTestProxy(t, targetListener.Addr().String(), func(p *ProxyInstance) {
		p.RequestTimeout = 400 * time.Millisecond
	})
	defer p.Stop()

	conn, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	const clientTxID = 0x0B0B
	start := time.Now()
	if _, err := conn.Write(modbus.CreateReadRequest(clientTxID, 2, 3, 0, 4)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	resp, err := modbus.ReadFrame(conn)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 3*time.Second {
		t.Errorf("proxy answered after %v, want well inside the 400ms budget", elapsed)
	}
	if got := binary.BigEndian.Uint16(resp[0:2]); got != clientTxID {
		t.Errorf("exception transaction ID = 0x%04X, want 0x%04X", got, clientTxID)
	}
	if len(resp) != 9 {
		t.Fatalf("exception frame length = %d, want 9", len(resp))
	}
	if resp[7] != 0x83 {
		t.Errorf("exception function code = 0x%02X, want 0x83", resp[7])
	}
	if resp[8] != 0x0B {
		t.Errorf("exception code = 0x%02X, want 0x0B (gateway target failed to respond)", resp[8])
	}
}

// TestMaxTargetConnsLimitsTargetSessions verifies that the target never sees
// more simultaneous connections than configured — the behaviour single-session
// devices such as SolarEdge inverters require.
func TestMaxTargetConnsLimitsTargetSessions(t *testing.T) {
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock target: %v", err)
	}
	defer targetListener.Close()

	var open, maxOpen int64
	go func() {
		for {
			conn, err := targetListener.Accept()
			if err != nil {
				return
			}
			current := atomic.AddInt64(&open, 1)
			for {
				peak := atomic.LoadInt64(&maxOpen)
				if current <= peak || atomic.CompareAndSwapInt64(&maxOpen, peak, current) {
					break
				}
			}
			go func(c net.Conn) {
				defer func() {
					atomic.AddInt64(&open, -1)
					c.Close()
				}()
				for {
					frame, err := modbus.ReadFrame(c)
					if err != nil {
						return
					}
					txID, unitID, fc, _, quantity, err := modbus.ParseReadRequest(frame)
					if err != nil {
						return
					}
					time.Sleep(20 * time.Millisecond)
					data := make([]byte, quantity*2)
					resp, _ := modbus.CreateReadResponse(txID, unitID, fc, data)
					if _, err := c.Write(resp); err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	p := startTestProxy(t, targetListener.Addr().String(), func(p *ProxyInstance) {
		p.MaxTargetConns = 1
	})
	defer p.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(client int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", p.ListenAddr)
			if err != nil {
				t.Errorf("client %d: dial failed: %v", client, err)
				return
			}
			defer conn.Close()
			for req := 0; req < 3; req++ {
				if _, err := conn.Write(modbus.CreateReadRequest(uint16(client*10+req), 1, 3, 0, 2)); err != nil {
					t.Errorf("client %d: write failed: %v", client, err)
					return
				}
				if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
					t.Errorf("client %d: set deadline failed: %v", client, err)
					return
				}
				if _, err := modbus.ReadFrame(conn); err != nil {
					t.Errorf("client %d: read failed: %v", client, err)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	if peak := atomic.LoadInt64(&maxOpen); peak > 1 {
		t.Errorf("target saw %d simultaneous connections, want at most 1", peak)
	}
}

func TestRequestPacerSpacesRequests(t *testing.T) {
	pacer := &requestPacer{gap: 50 * time.Millisecond}

	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := pacer.wait(t.Context()); err != nil {
			t.Fatalf("wait %d failed: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	// The first slot is free, the two after it are spaced by the gap.
	if elapsed < 100*time.Millisecond {
		t.Errorf("three paced requests took %v, want at least 100ms", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Errorf("three paced requests took %v, far more than the configured gap", elapsed)
	}
}

func TestRequestPacerDisabled(t *testing.T) {
	pacer := &requestPacer{}

	start := time.Now()
	for i := 0; i < 5; i++ {
		if err := pacer.wait(t.Context()); err != nil {
			t.Fatalf("wait %d failed: %v", i, err)
		}
	}

	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("unpaced requests took %v, want no delay", elapsed)
	}
}

func TestRequestBudget(t *testing.T) {
	p := NewProxyInstance("budget", "budget", ":0", "127.0.0.1:502", 0, 5, 3, 2, logger.NewNullLogger(10), nil)

	// Derived: (retries + 1) attempts of read timeout, plus slack for backoff.
	if got, want := p.requestBudget(), 3*3*time.Second+2*time.Second; got != want {
		t.Errorf("derived budget = %v, want %v", got, want)
	}

	p.RequestTimeout = 2500 * time.Millisecond
	if got := p.requestBudget(); got != 2500*time.Millisecond {
		t.Errorf("explicit budget = %v, want 2.5s", got)
	}
}
