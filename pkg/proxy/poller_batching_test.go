// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"context"
	"modbridge/pkg/modbus"
	"sync"
	"testing"
	"time"
)

func trackedFor(t *testing.T, unitID, fc uint8, addr, quantity uint16) dueRequest {
	t.Helper()
	frame := modbus.CreateReadRequest(1, unitID, fc, addr, quantity)
	key, _, ok := modbus.RequestCacheKey(frame)
	if !ok {
		t.Fatalf("read request should be cacheable")
	}
	return dueRequest{key: key, unitID: unitID, frame: frame}
}

// TestPlanBatchesMergesAdjacentRanges is the point of the exercise: turning
// several small reads into one, which is the only thing that reduces the round
// trips a slow device charges for.
func TestPlanBatchesMergesAdjacentRanges(t *testing.T) {
	requests := []dueRequest{
		trackedFor(t, 1, 3, 100, 4),
		trackedFor(t, 1, 3, 104, 4),
		trackedFor(t, 1, 3, 110, 2),
	}

	groups := planBatches(requests, 16)
	if len(groups) != 1 {
		t.Fatalf("expected the three adjacent ranges to merge into one read, got %d groups", len(groups))
	}
	if groups[0].base != 100 || groups[0].quantity != 12 {
		t.Errorf("merged read covers %d..%d, want 100..112", groups[0].base, groups[0].base+groups[0].quantity)
	}
	if len(groups[0].members) != 3 {
		t.Errorf("merged read serves %d requests, want 3", len(groups[0].members))
	}
}

// TestPlanBatchesKeepsIncompatibleRequestsApart guards the things that must
// never be merged: different units and different register spaces.
func TestPlanBatchesKeepsIncompatibleRequestsApart(t *testing.T) {
	tests := []struct {
		name     string
		requests []dueRequest
	}{
		{"different units", []dueRequest{trackedFor(t, 1, 3, 100, 2), trackedFor(t, 2, 3, 102, 2)}},
		{"holding vs input registers", []dueRequest{trackedFor(t, 1, 3, 100, 2), trackedFor(t, 1, 4, 102, 2)}},
		{"far apart", []dueRequest{trackedFor(t, 1, 3, 100, 2), trackedFor(t, 1, 3, 900, 2)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, group := range planBatches(tt.requests, 16) {
				if len(group.members) > 1 {
					t.Errorf("%s were merged into one read", tt.name)
				}
			}
		})
	}
}

// TestSplitBatchResponseRoutesEachRange verifies every original request gets
// exactly its own registers back out of the combined answer.
func TestSplitBatchResponseRoutesEachRange(t *testing.T) {
	first := trackedFor(t, 1, 3, 100, 2)
	second := trackedFor(t, 1, 3, 102, 3)
	group := batchGroup{unitID: 1, function: 3, base: 100, quantity: 5, members: []dueRequest{first, second}}

	// Registers 100..104 carry recognisable values.
	payload := []byte{0, 100, 0, 101, 0, 102, 0, 103, 0, 104}
	combined, err := modbus.CreateReadResponse(0, 1, 3, payload)
	if err != nil {
		t.Fatalf("building the combined response failed: %v", err)
	}

	parts, err := splitBatchResponse(group, combined)
	if err != nil {
		t.Fatalf("splitting failed: %v", err)
	}

	firstData, err := modbus.ParseReadResponse(parts[first.key])
	if err != nil {
		t.Fatalf("first slice unreadable: %v", err)
	}
	if got := []byte{firstData[1], firstData[3]}; got[0] != 100 || got[1] != 101 {
		t.Errorf("first request got registers %v, want 100 and 101", got)
	}

	secondData, err := modbus.ParseReadResponse(parts[second.key])
	if err != nil {
		t.Fatalf("second slice unreadable: %v", err)
	}
	if len(secondData) != 6 {
		t.Fatalf("second request got %d bytes, want 6", len(secondData))
	}
	if secondData[1] != 102 || secondData[5] != 104 {
		t.Errorf("second request got registers %d..%d, want 102..104", secondData[1], secondData[5])
	}
}

// TestSplitBatchResponseRejectsShortAnswer keeps a truncated read from becoming
// silently wrong cache entries.
func TestSplitBatchResponseRejectsShortAnswer(t *testing.T) {
	member := trackedFor(t, 1, 3, 100, 4)
	group := batchGroup{unitID: 1, function: 3, base: 100, quantity: 4, members: []dueRequest{member}}

	short, _ := modbus.CreateReadResponse(0, 1, 3, []byte{0, 1, 0, 2}) // 2 registers, not 4
	if _, err := splitBatchResponse(group, short); err == nil {
		t.Error("a short answer must be refused, not sliced")
	}
}

// TestPollerBatchesRefreshes ties it together: the device sees one read where
// the poller tracks three, and every tracked request still gets its own answer.
func TestPollerBatchesRefreshes(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []ProbeSpec
		stored   = map[uint64]int{}
	)

	poller := NewRegisterPoller(
		50*time.Millisecond,
		time.Minute,
		10,
		func(req []byte) ([]byte, error) {
			_, unitID, fc, addr, quantity, err := modbus.ParseReadRequest(req)
			if err != nil {
				return nil, err
			}
			mu.Lock()
			requests = append(requests, ProbeSpec{UnitID: unitID, Function: fc, Address: addr, Quantity: quantity})
			mu.Unlock()
			return modbus.CreateReadResponse(0, unitID, fc, make([]byte, quantity*2))
		},
		func(key uint64, unitID uint8, resp []byte) {
			mu.Lock()
			stored[key]++
			mu.Unlock()
		},
		nil,
	)

	for _, spec := range []struct{ addr, qty uint16 }{{200, 4}, {204, 4}, {208, 4}} {
		frame := modbus.CreateReadRequest(1, 1, 3, spec.addr, spec.qty)
		key, unitID, _ := modbus.RequestCacheKey(frame)
		poller.Track(key, unitID, frame)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	poller.refreshAll(ctx)

	mu.Lock()
	defer mu.Unlock()

	if len(requests) != 1 {
		t.Errorf("device saw %d reads, want 1 combined read", len(requests))
	}
	if len(requests) == 1 && (requests[0].Address != 200 || requests[0].Quantity != 12) {
		t.Errorf("combined read was %d..%d, want 200..212", requests[0].Address, requests[0].Address+requests[0].Quantity)
	}
	if len(stored) != 3 {
		t.Errorf("%d cache entries were written, want one per tracked request", len(stored))
	}
	if batches, saved := poller.BatchStats(); batches != 1 || saved != 2 {
		t.Errorf("batch stats = %d batches saving %d requests, want 1 and 2", batches, saved)
	}
}
