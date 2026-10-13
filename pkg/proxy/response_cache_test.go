// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"testing"
	"time"
)

// TestResponseCacheLFUEviction is a regression test for a bug where evictLFU
// used `leastHits == 0` as the "uninitialized" sentinel. When the true
// least-frequently-used entry had 0 hits, the guard stayed true for every
// subsequent entry, so the loop evicted an arbitrary (often high-hit) entry
// instead of the genuine LFU entry.
func TestResponseCacheLFUEviction(t *testing.T) {
	rc := NewResponseCache(ResponseCacheConfig{
		MaxSize:        3,
		TTL:            60_000_000_000, // long TTL — no expiry interference
		EvictionPolicy: EvictLFU,
	})

	// Three distinct hashes; simulate hit counts via Get().
	// hashA: 0 hits, hashB: 5 hits, hashC: 2 hits.
	rc.Set(1, []byte("a"))
	rc.Set(2, []byte("b"))
	rc.Set(3, []byte("c"))

	for i := 0; i < 5; i++ {
		rc.Get(2) // B → 5 hits
	}
	for i := 0; i < 2; i++ {
		rc.Get(3) // C → 2 hits
	}
	// A stays at 0 hits.

	// Inserting a 4th entry forces an eviction. The LFU entry is hashA (0 hits).
	rc.Set(4, []byte("d"))

	// hashA (1) must have been evicted; B and C (the higher-hit entries) survive.
	if _, ok := rc.Get(1); ok {
		t.Fatal("expected LFU entry hash=1 (0 hits) to be evicted, but it is still present")
	}
	if _, ok := rc.Get(2); !ok {
		t.Fatal("expected high-hit entry hash=2 to be retained, but it was evicted")
	}
	if _, ok := rc.Get(3); !ok {
		t.Fatal("expected entry hash=3 to be retained, but it was evicted")
	}
}

// TestResponseCacheLRUEvictionZeroHash verifies the LRU path evicts the oldest
// entry even when its hash is 0 (previously guarded by `oldestHash != 0`).
func TestResponseCacheLRUEvictionZeroHash(t *testing.T) {
	rc := NewResponseCache(ResponseCacheConfig{
		MaxSize:        1,
		TTL:            60_000_000_000,
		EvictionPolicy: EvictLRU,
	})

	rc.Set(0, []byte("first"))   // hash 0, oldest
	rc.Set(99, []byte("second")) // forces eviction of hash 0

	if _, ok := rc.Get(0); ok {
		t.Fatal("expected oldest entry hash=0 to be evicted, but it is still present")
	}
	if _, ok := rc.Get(99); !ok {
		t.Fatal("expected newly inserted entry hash=99 to be present")
	}
}

// TestResponseCacheSetIfFreshBlockedByInvalidation verifies the write-race
// guard: a read whose answer comes back after a write invalidated the unit
// must not repopulate the cache with the pre-write value.
func TestResponseCacheSetIfFreshBlockedByInvalidation(t *testing.T) {
	rc := NewResponseCache(ResponseCacheConfig{TTL: 60 * time.Second})

	gen := rc.Generation(7)
	if !rc.SetForUnitIfFresh(gen, 100, 7, []byte("old")) {
		t.Fatal("store with matching generation was rejected")
	}
	if resp, ok := rc.Get(100); !ok || string(resp) != "old" {
		t.Fatal("precondition: entry should be cached")
	}

	// A write to the unit invalidates the entry and bumps the generation.
	rc.InvalidateUnit(7)
	if _, ok := rc.Get(100); ok {
		t.Fatal("invalidation did not drop the entry")
	}

	if rc.SetForUnitIfFresh(gen, 100, 7, []byte("stale")) {
		t.Error("store with a pre-invalidation generation was accepted")
	}
	if _, ok := rc.Get(100); ok {
		t.Error("stale pre-write value re-entered the cache")
	}

	// A read started after the write stores normally.
	if !rc.SetForUnitIfFresh(rc.Generation(7), 100, 7, []byte("fresh")) {
		t.Error("store with the current generation was rejected")
	}
	if resp, ok := rc.Get(100); !ok || string(resp) != "fresh" {
		t.Fatal("fresh value not served from the cache")
	}
}
