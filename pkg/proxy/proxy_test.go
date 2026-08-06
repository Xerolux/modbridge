// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"modbridge/pkg/logger"
	"modbridge/pkg/modbus"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestSplitLogic_Integration(t *testing.T) {
	// Create a dummy target server
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock target: %v", err)
	}
	defer targetListener.Close()
	targetAddr := targetListener.Addr().String()

	go func() {
		for {
			conn, err := targetListener.Accept()
			if err != nil {
				return
			}
			go handleMockTarget(conn)
		}
	}()

	// Create proxy
	// Use a random port for proxy
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start proxy listener: %v", err)
	}
	proxyAddr := proxyListener.Addr().String()
	proxyListener.Close() // We just wanted a free port, ProxyInstance will listen itself

	l := logger.NewNullLogger(100)
	p := NewProxyInstance("test", "test", proxyAddr, targetAddr, 10, 5, 5, 3, l, nil) // MaxReadSize = 10 registers

	err = p.Start()
	if err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer p.Stop()

	// Connect to proxy
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// 1. Test Request within limit (5 registers)
	req1 := modbus.CreateReadRequest(1, 1, 3, 0, 5)
	_, err = conn.Write(req1)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	resp1, err := modbus.ReadFrame(conn)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Verify response 1
	if len(resp1) < 9 {
		t.Errorf("Response too short")
	}

	// 2. Test Request exceeding limit (25 registers). Max is 10.
	// Should split into 10, 10, 5.
	req2 := modbus.CreateReadRequest(2, 1, 3, 0, 25)
	_, err = conn.Write(req2)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	resp2, err := modbus.ReadFrame(conn)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Verify response 2
	// 25 registers * 2 = 50 bytes data.
	// Frame: MBAP(6) + Unit(1) + FC(1) + Len(1) + Data(50) = 59 bytes.
	if len(resp2) != 9+50 {
		t.Errorf("Split response length mismatch. Got %d, want %d", len(resp2), 59)
	}

	// Verify data content (Mock target returns 0xAA for all bytes)
	data := resp2[9:]
	for i, b := range data {
		if b != 0xAA {
			t.Errorf("Byte %d mismatch: got %x, want AA", i, b)
		}
	}
}

func TestProxyInstance_ConnectDelay(t *testing.T) {
	// Track timing of when the target accepts a connection vs. when it first
	// receives bytes. With ConnectDelay set, there should be a measurable gap
	// on a fresh outbound connection (the pool pre-warms exactly one).
	var (
		acceptNs    int64
		firstByteNs int64
	)

	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock target: %v", err)
	}
	defer targetListener.Close()

	go func() {
		for {
			conn, err := targetListener.Accept()
			if err != nil {
				return
			}
			atomic.StoreInt64(&acceptNs, time.Now().UnixNano())
			go func(c net.Conn) {
				defer c.Close()
				frame, err := modbus.ReadFrame(c)
				if err != nil {
					return
				}
				atomic.StoreInt64(&firstByteNs, time.Now().UnixNano())
				txID, unitID, fc, _, quantity, perr := modbus.ParseReadRequest(frame)
				if perr != nil {
					return
				}
				data := make([]byte, quantity*2)
				for i := range data {
					data[i] = 0xAA
				}
				resp, _ := modbus.CreateReadResponse(txID, unitID, fc, data)
				_, _ = c.Write(resp)
			}(conn)
		}
	}()

	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start proxy listener: %v", err)
	}
	proxyAddr := proxyListener.Addr().String()
	proxyListener.Close()

	const delay = 150 * time.Millisecond
	l := logger.NewNullLogger(100)
	p := NewProxyInstance("delay-test", "delay-test", proxyAddr, targetListener.Addr().String(), 10, 5, 5, 3, l, nil)
	p.ConnectDelay = delay

	if err := p.Start(); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer p.Stop()

	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	req := modbus.CreateReadRequest(1, 1, 3, 0, 5)
	if _, err := conn.Write(req); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if _, err := modbus.ReadFrame(conn); err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	accept := atomic.LoadInt64(&acceptNs)
	first := atomic.LoadInt64(&firstByteNs)
	if accept == 0 || first == 0 {
		t.Fatalf("target did not observe connection (accept=%d firstByte=%d)", accept, first)
	}

	gap := time.Duration(first - accept)
	// The gap should be at least the configured delay. Allow a small tolerance
	// for clock granularity, and a generous upper bound to catch regressions
	// where the delay is applied in the wrong place.
	if gap < delay-30*time.Millisecond {
		t.Errorf("connect delay not applied: gap=%v, want >= %v", gap, delay-30*time.Millisecond)
	}
	if gap > 5*time.Second {
		t.Errorf("connect delay unexpectedly large: gap=%v", gap)
	}
}

func handleMockTarget(conn net.Conn) {
	defer conn.Close()
	for {
		frame, err := modbus.ReadFrame(conn)
		if err != nil {
			return
		}

		// Parse request
		txID, unitID, fc, _, quantity, err := modbus.ParseReadRequest(frame)
		if err != nil {
			return
		}

		// Generate response with 0xAA
		data := make([]byte, quantity*2)
		for i := range data {
			data[i] = 0xAA
		}

		resp, _ := modbus.CreateReadResponse(txID, unitID, fc, data)
		_, _ = conn.Write(resp)
	}
}

func TestTryAcquireConnSlot(t *testing.T) {
	sem := make(chan struct{}, 1)

	if !tryAcquireConnSlot(sem) {
		t.Fatalf("expected first acquire to succeed")
	}
	if tryAcquireConnSlot(sem) {
		t.Fatalf("expected second acquire to fail when semaphore is full")
	}

	<-sem // release

	if !tryAcquireConnSlot(sem) {
		t.Fatalf("expected acquire to succeed after release")
	}
}

func TestConnectionLimiterAcquireRelease(t *testing.T) {
	cl := &ConnectionLimiter{max: 2}

	if !cl.acquire() {
		t.Fatal("expected acquire 1 to succeed")
	}
	if !cl.acquire() {
		t.Fatal("expected acquire 2 to succeed")
	}
	if cl.acquire() {
		t.Fatal("expected acquire 3 to fail (limit 2)")
	}

	cl.release()
	if !cl.acquire() {
		t.Fatal("expected acquire after release to succeed")
	}

	cl.release()
	cl.release()
}

func TestConnectionLimiterUnlimited(t *testing.T) {
	cl := &ConnectionLimiter{max: 0} // unlimited

	for i := 0; i < 100; i++ {
		if !cl.acquire() {
			t.Fatalf("expected acquire %d to succeed with unlimited limiter", i)
		}
	}

	for i := 0; i < 100; i++ {
		cl.release()
	}
}

func TestConnectionLimiterSetMaxWhileRunning(t *testing.T) {
	cl := &ConnectionLimiter{max: 1}
	if !cl.acquire() {
		t.Fatal("expected acquire 1 to succeed")
	}

	// Increase limit dynamically
	atomic.StoreInt64(&cl.max, 3)
	if !cl.acquire() {
		t.Fatal("expected acquire 2 to succeed after max increase")
	}
	if !cl.acquire() {
		t.Fatal("expected acquire 3 to succeed after max increase")
	}
	if cl.acquire() {
		t.Fatal("expected acquire 4 to fail (limit 3)")
	}

	cl.release()
	cl.release()
	cl.release()
}
