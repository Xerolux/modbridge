// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"context"
	"encoding/binary"
	"modbridge/pkg/modbus"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// countingTarget answers read requests with a per-unit payload and counts how
// many requests it saw, so tests can tell a cache hit from a device read.
func countingTarget(t *testing.T, reads *int64, delay time.Duration) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock target: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
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
					unitID, fc, _ := modbus.FrameUnitAndFunction(frame)
					if modbus.IsWriteFunction(fc) {
						// Echo the request back: good enough for a write ack.
						if _, err := c.Write(frame); err != nil {
							return
						}
						continue
					}
					txID, _, _, _, quantity, perr := modbus.ParseReadRequest(frame)
					if perr != nil {
						return
					}
					atomic.AddInt64(reads, 1)
					if delay > 0 {
						time.Sleep(delay)
					}
					data := make([]byte, quantity*2)
					for i := range data {
						data[i] = unitID
					}
					resp, _ := modbus.CreateReadResponse(txID, unitID, fc, data)
					if _, err := c.Write(resp); err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	return listener
}

// TestCacheServesRepeatedReads verifies that a second identical read is served
// from the cache instead of reaching the device.
func TestCacheServesRepeatedReads(t *testing.T) {
	var reads int64
	target := countingTarget(t, &reads, 0)
	defer target.Close()

	p := startTestProxy(t, target.Addr().String(), func(p *ProxyInstance) {
		p.CacheEnabled = true
		p.CacheTTL = 10 * time.Second
	})
	defer p.Stop()

	conn, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	for i := 0; i < 5; i++ {
		if _, err := conn.Write(modbus.CreateReadRequest(uint16(i+1), 3, 3, 100, 4)); err != nil {
			t.Fatalf("write %d failed: %v", i, err)
		}
		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
			t.Fatalf("set deadline failed: %v", err)
		}
		resp, err := modbus.ReadFrame(conn)
		if err != nil {
			t.Fatalf("read %d failed: %v", i, err)
		}
		// Every answer must carry the transaction ID of the request that asked
		// for it, cached or not.
		if got := binary.BigEndian.Uint16(resp[0:2]); got != uint16(i+1) {
			t.Errorf("request %d answered with transaction ID 0x%04X, want 0x%04X", i, got, i+1)
		}
		if got := len(resp); got != 9+8 {
			t.Errorf("request %d: response length %d, want %d", i, got, 9+8)
		}
	}

	if got := atomic.LoadInt64(&reads); got != 1 {
		t.Errorf("target saw %d reads, want 1 (the rest should come from the cache)", got)
	}
	if stats := p.CacheStats(); stats.Hits != 4 {
		t.Errorf("cache hits = %d, want 4", stats.Hits)
	}
}

// TestCacheExpiresAfterTTL verifies that a stale entry is not served.
func TestCacheExpiresAfterTTL(t *testing.T) {
	var reads int64
	target := countingTarget(t, &reads, 0)
	defer target.Close()

	p := startTestProxy(t, target.Addr().String(), func(p *ProxyInstance) {
		p.CacheEnabled = true
		p.CacheTTL = 150 * time.Millisecond
	})
	defer p.Stop()

	conn, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	read := func(txID uint16) {
		t.Helper()
		if _, err := conn.Write(modbus.CreateReadRequest(txID, 1, 3, 0, 2)); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
			t.Fatalf("set deadline failed: %v", err)
		}
		if _, err := modbus.ReadFrame(conn); err != nil {
			t.Fatalf("read failed: %v", err)
		}
	}

	read(1)
	read(2) // cached
	if got := atomic.LoadInt64(&reads); got != 1 {
		t.Fatalf("target saw %d reads before expiry, want 1", got)
	}

	time.Sleep(250 * time.Millisecond)

	read(3) // entry expired, must reach the device again
	if got := atomic.LoadInt64(&reads); got != 2 {
		t.Errorf("target saw %d reads after expiry, want 2", got)
	}
}

// TestCacheInvalidatedByWrite verifies that a write drops the cached reads of
// that unit — serving a value a write just changed would be wrong, not merely
// stale.
func TestCacheInvalidatedByWrite(t *testing.T) {
	var reads int64
	target := countingTarget(t, &reads, 0)
	defer target.Close()

	p := startTestProxy(t, target.Addr().String(), func(p *ProxyInstance) {
		p.CacheEnabled = true
		p.CacheTTL = 10 * time.Second
	})
	defer p.Stop()

	conn, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	exchange := func(frame []byte) {
		t.Helper()
		if _, err := conn.Write(frame); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
			t.Fatalf("set deadline failed: %v", err)
		}
		if _, err := modbus.ReadFrame(conn); err != nil {
			t.Fatalf("read failed: %v", err)
		}
	}

	readReq := modbus.CreateReadRequest(1, 7, 3, 10, 2)
	exchange(readReq)
	exchange(modbus.CreateReadRequest(2, 7, 3, 10, 2)) // cached
	if got := atomic.LoadInt64(&reads); got != 1 {
		t.Fatalf("target saw %d reads before the write, want 1", got)
	}

	// Write single register on the same unit.
	writeReq := make([]byte, 12)
	binary.BigEndian.PutUint16(writeReq[0:2], 3)
	binary.BigEndian.PutUint16(writeReq[4:6], 6)
	writeReq[6] = 7    // unit
	writeReq[7] = 0x06 // write single register
	binary.BigEndian.PutUint16(writeReq[8:10], 10)
	binary.BigEndian.PutUint16(writeReq[10:12], 0x1234)
	exchange(writeReq)

	exchange(modbus.CreateReadRequest(4, 7, 3, 10, 2))
	if got := atomic.LoadInt64(&reads); got != 2 {
		t.Errorf("target saw %d reads after the write, want 2 (cache must be invalidated)", got)
	}
}

// TestCacheIgnoresOtherUnitsOnWrite verifies invalidation is scoped to the unit
// that was written.
func TestCacheIgnoresOtherUnitsOnWrite(t *testing.T) {
	cache := NewResponseCache(ResponseCacheConfig{MaxSize: 10, TTL: time.Minute})

	cache.SetForUnit(1, 2, []byte("unit two"))
	cache.SetForUnit(2, 3, []byte("unit three"))

	if dropped := cache.InvalidateUnit(2); dropped != 1 {
		t.Errorf("dropped %d entries, want 1", dropped)
	}
	if _, ok := cache.Get(1); ok {
		t.Error("entry of the written unit is still cached")
	}
	if _, ok := cache.Get(2); !ok {
		t.Error("entry of an untouched unit was dropped")
	}
}

// TestCacheGetReturnsCopy guards the transaction-ID rewrite: a served frame is
// modified by the caller and must not alias the stored entry.
func TestCacheGetReturnsCopy(t *testing.T) {
	cache := NewResponseCache(ResponseCacheConfig{MaxSize: 10, TTL: time.Minute})
	cache.Set(1, []byte{0xAA, 0xBB, 0xCC})

	first, ok := cache.Get(1)
	if !ok {
		t.Fatal("expected a hit")
	}
	first[0] = 0xFF

	second, ok := cache.Get(1)
	if !ok {
		t.Fatal("expected a second hit")
	}
	if second[0] != 0xAA {
		t.Errorf("cached entry was modified through a served copy: got 0x%02X, want 0xAA", second[0])
	}
}

// TestPollerKeepsRegistersWarm verifies the background poller refreshes what a
// client asked for, so a slow device is read on the proxy's schedule rather
// than the client's.
func TestPollerKeepsRegistersWarm(t *testing.T) {
	var refreshes int64
	stored := make(chan uint64, 16)

	poller := NewRegisterPoller(
		50*time.Millisecond,
		time.Minute,
		10,
		func(req []byte) ([]byte, error) {
			atomic.AddInt64(&refreshes, 1)
			resp, _ := modbus.CreateReadResponse(0, 1, 3, []byte{0, 1})
			return resp, nil
		},
		func(key uint64, unitID uint8, resp []byte) {
			select {
			case stored <- key:
			default:
			}
		},
		nil,
	)

	req := modbus.CreateReadRequest(1, 1, 3, 0, 1)
	key, unitID, ok := modbus.RequestCacheKey(req)
	if !ok {
		t.Fatal("read request should be cacheable")
	}
	poller.Track(key, unitID, req)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	poller.Start(ctx)
	defer poller.Stop()

	select {
	case got := <-stored:
		if got != key {
			t.Errorf("poller stored key %d, want %d", got, key)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("poller did not refresh the tracked request")
	}

	if got := atomic.LoadInt64(&refreshes); got < 1 {
		t.Errorf("poller performed %d refreshes, want at least 1", got)
	}
}

// TestPollerDropsIdleRequests verifies a request no client asks for any more is
// forgotten instead of polled forever.
func TestPollerDropsIdleRequests(t *testing.T) {
	poller := NewRegisterPoller(
		time.Hour, // never fires on its own during the test
		10*time.Millisecond,
		10,
		func(req []byte) ([]byte, error) { return nil, nil },
		func(key uint64, unitID uint8, resp []byte) {},
		nil,
	)

	req := modbus.CreateReadRequest(1, 1, 3, 0, 1)
	key, unitID, _ := modbus.RequestCacheKey(req)
	poller.Track(key, unitID, req)

	if tracked, _, _ := poller.Stats(); tracked != 1 {
		t.Fatalf("tracked %d requests, want 1", tracked)
	}

	time.Sleep(50 * time.Millisecond)
	poller.due() // sweeps idle entries

	if tracked, _, _ := poller.Stats(); tracked != 0 {
		t.Errorf("tracked %d requests after the idle window, want 0", tracked)
	}
}

// TestPollerReportsSlowRounds verifies the poller notices when a refresh round
// outruns its own interval — the state where the proxy polls the target
// continuously and cached values are older than the interval suggests.
func TestPollerReportsSlowRounds(t *testing.T) {
	var logged atomic.Int64
	poller := NewRegisterPoller(
		50*time.Millisecond,
		time.Minute,
		10,
		func(req []byte) ([]byte, error) {
			time.Sleep(120 * time.Millisecond) // one request already outlasts the interval
			resp, _ := modbus.CreateReadResponse(0, 1, 3, []byte{0, 1})
			return resp, nil
		},
		func(key uint64, unitID uint8, resp []byte) {},
		func(msg string) { logged.Add(1) },
	)

	req := modbus.CreateReadRequest(1, 1, 3, 0, 1)
	key, unitID, _ := modbus.RequestCacheKey(req)
	poller.Track(key, unitID, req)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	poller.Start(ctx)
	defer poller.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && poller.SlowRounds() == 0 {
		time.Sleep(50 * time.Millisecond)
	}

	if poller.SlowRounds() == 0 {
		t.Error("poller did not report a refresh round that outran its interval")
	}
	if logged.Load() == 0 {
		t.Error("slow round was counted but never logged")
	}
}

// TestPollerRespectsEntryLimit keeps a client sweeping the address space from
// growing the warm set without bound.
func TestPollerRespectsEntryLimit(t *testing.T) {
	poller := NewRegisterPoller(time.Hour, time.Hour, 3,
		func(req []byte) ([]byte, error) { return nil, nil },
		func(key uint64, unitID uint8, resp []byte) {},
		nil,
	)

	for addr := 0; addr < 10; addr++ {
		req := modbus.CreateReadRequest(1, 1, 3, uint16(addr), 1)
		key, unitID, _ := modbus.RequestCacheKey(req)
		poller.Track(key, unitID, req)
	}

	if tracked, _, _ := poller.Stats(); tracked != 3 {
		t.Errorf("tracked %d requests, want the configured cap of 3", tracked)
	}
}

// TestCacheNotUsedWhenDisabled is the guard that the feature stays opt-in.
func TestCacheNotUsedWhenDisabled(t *testing.T) {
	var reads int64
	target := countingTarget(t, &reads, 0)
	defer target.Close()

	p := startTestProxy(t, target.Addr().String(), nil)
	defer p.Stop()

	conn, err := net.Dial("tcp", p.ListenAddr)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	for i := 0; i < 3; i++ {
		if _, err := conn.Write(modbus.CreateReadRequest(uint16(i+1), 1, 3, 0, 2)); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
			t.Fatalf("set deadline failed: %v", err)
		}
		if _, err := modbus.ReadFrame(conn); err != nil {
			t.Fatalf("read failed: %v", err)
		}
	}

	if got := atomic.LoadInt64(&reads); got != 3 {
		t.Errorf("target saw %d reads, want 3 — no caching without cache_enabled", got)
	}
}
