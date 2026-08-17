// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"context"
	"fmt"
	"modbridge/pkg/modbus"
	"sync"
	"sync/atomic"
	"time"
)

// RegisterPoller keeps the registers a client actually asks for warm in the
// cache, refreshing them on its own schedule instead of the client's.
//
// This decouples a slow target from an impatient client. A SolarEdge leader
// relaying follower registers over RS485, or a heating controller that takes
// seconds to answer, cannot be made faster — but it can be read continuously
// in the background so the client is served from the cache and never waits for
// the device at all.
//
// The poller only ever refreshes requests it has seen a client make. It never
// invents reads: a register nobody asked for is never polled, and a request
// that stops arriving is dropped again after idleAfter.
type RegisterPoller struct {
	mu      sync.Mutex
	tracked map[uint64]*trackedRequest

	interval   time.Duration
	idleAfter  time.Duration
	maxEntries int

	refresh func(req []byte) ([]byte, error)
	store   func(key uint64, unitID uint8, resp []byte)
	logf    func(msg string)

	// Merge adjacent ranges into one read when they are no further apart than
	// this. Zero disables batching and every request is refreshed on its own.
	maxAddressGap uint16
	batched       atomic.Int64
	batchedSaved  atomic.Int64

	refreshes  atomic.Int64
	failures   atomic.Int64
	slowRounds atomic.Int64

	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
}

type trackedRequest struct {
	frame    []byte
	unitID   uint8
	lastSeen time.Time
}

// NewRegisterPoller creates a poller. refresh performs one request against the
// target (normally the proxy's forward path, so pacing, retries and the
// connection cap all apply); store writes a successful response to the cache.
// defaultMaxAddressGap is how far apart two ranges may sit and still be worth
// reading as one. Registers between them are read needlessly, which is cheaper
// than a second round trip on the devices this exists for, but only up to a
// point.
const defaultMaxAddressGap = 16

func NewRegisterPoller(interval, idleAfter time.Duration, maxEntries int,
	refresh func(req []byte) ([]byte, error),
	store func(key uint64, unitID uint8, resp []byte),
	logf func(msg string),
) *RegisterPoller {
	if idleAfter <= 0 {
		idleAfter = 5 * time.Minute
	}
	if maxEntries <= 0 {
		maxEntries = 512
	}
	return &RegisterPoller{
		maxAddressGap: defaultMaxAddressGap,
		tracked:       make(map[uint64]*trackedRequest),
		interval:      interval,
		idleAfter:     idleAfter,
		maxEntries:    maxEntries,
		refresh:       refresh,
		store:         store,
		logf:          logf,
	}
}

// Track records that a client asked this request, so the poller keeps it warm.
// The frame is copied: the caller reuses its buffer.
func (rp *RegisterPoller) Track(key uint64, unitID uint8, req []byte) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if entry, ok := rp.tracked[key]; ok {
		entry.lastSeen = time.Now()
		return
	}
	if len(rp.tracked) >= rp.maxEntries {
		return // Bounded: a client sweeping the address space must not grow this without limit
	}

	frame := make([]byte, len(req))
	copy(frame, req)
	rp.tracked[key] = &trackedRequest{frame: frame, unitID: unitID, lastSeen: time.Now()}
}

// Start begins refreshing in the background until ctx is cancelled.
func (rp *RegisterPoller) Start(ctx context.Context) {
	if rp.interval <= 0 {
		return // Passive cache only: entries are filled by client reads and expire on TTL
	}

	ctx, rp.cancel = context.WithCancel(ctx)
	rp.wg.Add(1)
	go rp.loop(ctx)
}

// Stop ends background refreshing and waits for the loop to finish.
func (rp *RegisterPoller) Stop() {
	rp.once.Do(func() {
		if rp.cancel != nil {
			rp.cancel()
		}
	})
	rp.wg.Wait()
}

func (rp *RegisterPoller) loop(ctx context.Context) {
	defer rp.wg.Done()

	ticker := time.NewTicker(rp.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		started := time.Now()
		rp.refreshAll(ctx)
		rp.reportRoundDuration(time.Since(started))
	}
}

// reportRoundDuration flags refresh rounds that take longer than the interval
// they are scheduled at. That means the proxy is polling the target
// continuously and cached values are older than the configured interval
// suggests — usually too many tracked registers, too large a request gap, or
// a target that answers slower than assumed.
func (rp *RegisterPoller) reportRoundDuration(elapsed time.Duration) {
	if elapsed <= rp.interval {
		return
	}
	rp.slowRounds.Add(1)
	if rp.logf != nil {
		tracked, _, _ := rp.Stats()
		rp.logf(fmt.Sprintf(
			"Background refresh round took %v for %d requests, longer than the %v poll interval — cached values are older than the interval implies",
			elapsed.Round(time.Millisecond), tracked, rp.interval,
		))
	}
}

// refreshAll re-reads every tracked request once, sequentially. Sequential is
// deliberate: the targets this exists for are the ones that cannot take
// parallel requests in the first place.
//
// Adjacent ranges are merged into single reads first. On a device that needs a
// pause between requests, the number of round trips is the cost that dominates
// a cycle, and merging is the only thing that reduces it.
func (rp *RegisterPoller) refreshAll(ctx context.Context) {
	due := rp.due()
	if rp.maxAddressGap == 0 {
		for _, entry := range due {
			if !rp.refreshOne(ctx, entry) {
				return
			}
		}
		return
	}

	for _, group := range planBatches(due, rp.maxAddressGap) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if len(group.members) < 2 {
			if len(group.members) == 1 && !rp.refreshOne(ctx, group.members[0]) {
				return
			}
			continue
		}
		rp.refreshGroup(ctx, group)
	}
}

// refreshOne refreshes a single tracked request. It reports whether the loop
// should continue.
func (rp *RegisterPoller) refreshOne(ctx context.Context, entry dueRequest) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}

	resp, err := rp.refresh(entry.frame)
	if err != nil {
		rp.failures.Add(1)
		if rp.logf != nil {
			rp.logf(fmt.Sprintf("Background refresh of unit %d failed: %v", entry.unitID, err))
		}
		return true
	}
	rp.refreshes.Add(1)
	rp.store(entry.key, entry.unitID, resp)
	return true
}

// refreshGroup reads a merged range once and files the pieces under the keys
// the individual requests are cached by. If anything about the answer does not
// add up, the members are refreshed individually rather than cached from a
// response that cannot be trusted.
func (rp *RegisterPoller) refreshGroup(ctx context.Context, group batchGroup) {
	combined := modbus.CreateReadRequest(0, group.unitID, group.function, group.base, group.quantity)

	resp, err := rp.refresh(combined)
	if err == nil {
		var parts map[uint64][]byte
		parts, err = splitBatchResponse(group, resp)
		if err == nil {
			rp.refreshes.Add(1)
			rp.batched.Add(1)
			rp.batchedSaved.Add(int64(len(group.members) - 1))
			for _, member := range group.members {
				if frame, ok := parts[member.key]; ok {
					rp.store(member.key, member.unitID, frame)
				}
			}
			return
		}
	}

	rp.failures.Add(1)
	if rp.logf != nil {
		rp.logf(fmt.Sprintf("Combined read of %d registers from unit %d failed (%v); refreshing them individually",
			group.quantity, group.unitID, err))
	}
	for _, member := range group.members {
		if !rp.refreshOne(ctx, member) {
			return
		}
	}
}

type dueRequest struct {
	key    uint64
	unitID uint8
	frame  []byte
}

// due returns the requests to refresh this round and drops the ones no client
// has asked for in a while.
func (rp *RegisterPoller) due() []dueRequest {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	now := time.Now()
	requests := make([]dueRequest, 0, len(rp.tracked))
	for key, entry := range rp.tracked {
		if now.Sub(entry.lastSeen) > rp.idleAfter {
			delete(rp.tracked, key)
			continue
		}
		requests = append(requests, dueRequest{key: key, unitID: entry.unitID, frame: entry.frame})
	}
	return requests
}

// Stats reports what the poller has done so far.
func (rp *RegisterPoller) Stats() (tracked int, refreshes, failures int64) {
	rp.mu.Lock()
	tracked = len(rp.tracked)
	rp.mu.Unlock()
	return tracked, rp.refreshes.Load(), rp.failures.Load()
}

// BatchStats reports how many combined reads were issued and how many round
// trips that saved.
func (rp *RegisterPoller) BatchStats() (batches, savedRequests int64) {
	return rp.batched.Load(), rp.batchedSaved.Load()
}

// SlowRounds returns how many refresh rounds outran their own interval.
func (rp *RegisterPoller) SlowRounds() int64 {
	return rp.slowRounds.Load()
}
