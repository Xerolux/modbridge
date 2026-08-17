// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"fmt"
	"modbridge/pkg/batch"
	"modbridge/pkg/modbus"
	"sort"
)

// Batching the background refreshes is where the round-trip cost actually
// lives. A client polling a hundred small register ranges costs a hundred
// requests, and on a device that needs a pause between them, the spacing alone
// dominates the cycle. Reading one contiguous block instead and slicing it up
// afterwards turns those hundred requests into a handful.
//
// This is deliberately confined to the poller. Client traffic is forwarded
// untouched: a proxy that rewrites what a client asked for is a proxy nobody
// can debug. The poller, by contrast, owns its requests — it knows exactly
// which ranges it wants and under which key each answer belongs — so merging
// there is safe and reversible.

// maxBatchRegisters is the Modbus ceiling for one read of holding or input
// registers. A merged range may never exceed it.
const maxBatchRegisters = 125

// batchGroup is a set of tracked requests answered by one combined read.
type batchGroup struct {
	unitID   uint8
	function uint8
	base     uint16
	quantity uint16
	members  []dueRequest
}

// planBatches groups the poller's due requests into combined reads. Requests
// that cannot be merged come back as groups of one, so the caller has a single
// path to execute.
func planBatches(requests []dueRequest, maxAddressGap uint16) []batchGroup {
	if len(requests) == 0 {
		return nil
	}

	// Describe every request in the terms pkg/batch understands, keeping the
	// original so the answer can be routed back to its cache key.
	byIndex := make(map[*batch.Request]dueRequest, len(requests))
	batchRequests := make([]*batch.Request, 0, len(requests))
	for _, req := range requests {
		unitID, fc, ok := modbus.FrameUnitAndFunction(req.frame)
		if !ok {
			continue
		}
		_, _, _, addr, quantity, err := modbus.ParseReadRequest(req.frame)
		if err != nil {
			continue
		}
		reqType, ok := batchRequestType(fc)
		if !ok {
			continue
		}

		br := &batch.Request{
			Type:     reqType,
			SlaveID:  unitID,
			Address:  addr,
			Quantity: quantity,
		}
		byIndex[br] = req
		batchRequests = append(batchRequests, br)
	}

	optimizer := batch.NewOptimizer(&batch.BatchConfig{
		// MaxBatchSize is compared against the register span of a merged read,
		// not the number of requests in it, so the Modbus read limit is the
		// value that belongs here.
		MaxBatchSize:       maxBatchRegisters,
		MaxAddressGap:      maxAddressGap,
		CrossSlaveBatching: false, // A merged read addresses exactly one unit
		CrossTypeBatching:  false, // Holding and input registers are separate spaces
	})

	var groups []batchGroup
	for _, planned := range optimizer.Optimize(batchRequests) {
		if len(planned) == 0 {
			continue
		}
		sort.Slice(planned, func(i, j int) bool { return planned[i].Address < planned[j].Address })

		base := planned[0].Address
		end := base
		members := make([]dueRequest, 0, len(planned))
		for _, br := range planned {
			if reqEnd := br.Address + br.Quantity; reqEnd > end {
				end = reqEnd
			}
			members = append(members, byIndex[br])
		}

		span := int(end) - int(base)
		if span <= 0 || span > maxBatchRegisters || len(members) == 0 {
			// Too wide to read in one go: fall back to individual reads rather
			// than splitting into an arbitrary second batching scheme.
			for _, member := range members {
				groups = append(groups, batchGroup{members: []dueRequest{member}})
			}
			continue
		}

		unitID, fc, _ := modbus.FrameUnitAndFunction(members[0].frame)
		groups = append(groups, batchGroup{
			unitID:   unitID,
			function: fc,
			base:     base,
			quantity: uint16(span),
			members:  members,
		})
	}

	return groups
}

// batchRequestType maps a Modbus function code to the batch package's type.
func batchRequestType(fc uint8) (batch.RequestType, bool) {
	switch fc {
	case modbus.FuncReadHoldingRegisters:
		return batch.RequestTypeReadHoldingRegisters, true
	case modbus.FuncReadInputRegisters:
		return batch.RequestTypeReadInputRegisters, true
	}
	// Coils and discrete inputs are bit-addressed; slicing their responses back
	// apart is a different exercise and not worth guessing at.
	return 0, false
}

// splitBatchResponse carves a combined read's payload into one response per
// original request. Returns an error rather than a short slice if the device
// answered with less data than the range covers — a truncated answer must not
// become silently wrong cache entries.
func splitBatchResponse(group batchGroup, resp []byte) (map[uint64][]byte, error) {
	payload, err := modbus.ParseReadResponse(resp)
	if err != nil {
		return nil, err
	}
	if len(payload) != int(group.quantity)*2 {
		return nil, fmt.Errorf("combined read of %d registers answered with %d bytes", group.quantity, len(payload))
	}

	out := make(map[uint64][]byte, len(group.members))
	for _, member := range group.members {
		_, _, _, addr, quantity, err := modbus.ParseReadRequest(member.frame)
		if err != nil {
			return nil, err
		}
		start := (int(addr) - int(group.base)) * 2
		end := start + int(quantity)*2
		if start < 0 || end > len(payload) {
			return nil, fmt.Errorf("register %d..%d falls outside the combined read", addr, addr+quantity)
		}

		// Transaction ID 0: the cache rewrites it to whichever client is served.
		frame, err := modbus.CreateReadResponse(0, group.unitID, group.function, payload[start:end])
		if err != nil {
			return nil, err
		}
		out[member.key] = frame
	}
	return out, nil
}
