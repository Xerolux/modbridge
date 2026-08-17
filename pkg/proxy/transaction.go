// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"context"
	"fmt"
	"modbridge/pkg/modbus"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// maxStaleFrames is the number of non-matching frames the proxy discards on a
// single target connection before it gives up and drops that connection. A
// healthy device never sends any; a device that keeps producing them is out of
// sync and must not be reused.
const maxStaleFrames = 8

// requestPacer enforces a minimum spacing between consecutive requests to a
// target. Many Modbus devices — SolarEdge/SunSpec inverters, small RTU
// gateways — silently drop requests that arrive back-to-back.
type requestPacer struct {
	mu   sync.Mutex
	gap  time.Duration
	next time.Time
}

// reserve claims the next slot and returns how long the caller has to wait for
// it. The slot is reserved under the lock but waited for outside of it, so
// concurrent requests are spread out instead of being fully serialized.
func (rp *requestPacer) reserve() time.Duration {
	if rp == nil || rp.gap <= 0 {
		return 0
	}

	rp.mu.Lock()
	defer rp.mu.Unlock()

	now := time.Now()
	var wait time.Duration
	if rp.next.After(now) {
		wait = rp.next.Sub(now)
	}
	rp.next = now.Add(wait + rp.gap)
	return wait
}

// wait blocks until this request's paced slot is due, or until ctx is done.
func (rp *requestPacer) wait(ctx context.Context) error {
	wait := rp.reserve()
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// nextTargetTxID returns the next transaction ID used towards the target.
// The proxy assigns its own IDs instead of forwarding the client's so that a
// late response from an abandoned transaction can be recognised and discarded
// rather than being handed to the client as the answer to a newer request.
func (p *ProxyInstance) nextTargetTxID() uint16 {
	return uint16(atomic.AddUint32(&p.targetTxID, 1))
}

// requestBudget is the wall-clock cap for a single client request including
// all retries and backoff. Without a cap, a slow target can keep the proxy
// busy far longer than the client is willing to wait; the client then gives up
// and the eventual response arrives for a transaction nobody is waiting for.
func (p *ProxyInstance) requestBudget() time.Duration {
	if p.RequestTimeout > 0 {
		return p.RequestTimeout
	}

	readTimeout, _ := p.currentTimeouts()
	return time.Duration(p.MaxRetries+1)*readTimeout + 2*time.Second
}

// currentTimeouts returns the read and connect timeouts in effect, preferring
// the adaptive values when they are available.
func (p *ProxyInstance) currentTimeouts() (readTimeout, connectTimeout time.Duration) {
	readTimeout = p.ReadTimeout
	connectTimeout = p.ConnectionTimeout
	if p.adaptiveTimeout != nil {
		readTimeout = p.adaptiveTimeout.GetReadTimeout()
		connectTimeout = p.adaptiveTimeout.GetConnectTimeout()
	}
	return readTimeout, connectTimeout
}

// StaleResponses returns how many out-of-band target responses were discarded.
// A non-zero value means the target answered transactions that had already
// been abandoned — usually a device that is slower than the configured
// timeouts allow.
func (p *ProxyInstance) StaleResponses() int64 {
	return atomic.LoadInt64(&p.staleResponses)
}

// CacheStats returns cache counters, or zeroes when no cache is configured.
func (p *ProxyInstance) CacheStats() CacheStatsWithHitRate {
	if p.cache == nil {
		return CacheStatsWithHitRate{}
	}
	return p.cache.GetStatsWithHitRate()
}

// PollerStats returns how many requests the background poller keeps warm and
// how its refreshes have gone.
func (p *ProxyInstance) PollerStats() (tracked int, refreshes, failures int64) {
	if p.poller == nil {
		return 0, 0, 0
	}
	return p.poller.Stats()
}

// readMatchingResponse reads frames from conn until one carries the expected
// transaction ID, discarding late responses to earlier transactions. It
// returns an error once the deadline passes or too many stale frames arrive,
// in which case the caller must drop the connection: whatever is still in
// flight would desynchronise every following request on it.
func (p *ProxyInstance) readMatchingResponse(conn net.Conn, req []byte, wantTxID uint16, deadline time.Time) ([]byte, error) {
	for stale := 0; ; stale++ {
		if err := conn.SetReadDeadline(deadline); err != nil {
			return nil, err
		}

		resp, err := modbus.ReadFrame(conn)
		if err != nil {
			return nil, err
		}

		gotTxID, ok := modbus.FrameTxID(resp)
		if ok && gotTxID == wantTxID {
			if !modbus.ResponseMatchesRequest(req, resp) {
				// Right transaction, wrong unit or function code. Out of spec,
				// but the client asked for this transaction, so pass it on and
				// leave the interpretation to it.
				p.log.Warn(p.ID, fmt.Sprintf("Target response for transaction 0x%04X does not match the request unit/function", wantTxID))
			}
			return resp, nil
		}

		atomic.AddInt64(&p.staleResponses, 1)
		p.log.Warn(p.ID, fmt.Sprintf("Discarding stale target response (transaction 0x%04X, expected 0x%04X)", gotTxID, wantTxID))

		if stale+1 >= maxStaleFrames {
			return nil, fmt.Errorf("target out of sync: %d stale responses while waiting for transaction 0x%04X", stale+1, wantTxID)
		}
		if !time.Now().Before(deadline) {
			return nil, fmt.Errorf("timeout waiting for transaction 0x%04X after %d stale responses", wantTxID, stale+1)
		}
	}
}
