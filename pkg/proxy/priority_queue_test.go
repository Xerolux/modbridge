// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"testing"
	"time"
)

// TestPriorityQueueDeliversToConsumer verifies the queue is pure storage:
// everything enqueued reaches the Dequeue caller. An internal processing loop
// used to race the consumer and silently discard every request.
func TestPriorityQueueDeliversToConsumer(t *testing.T) {
	pq := NewPriorityQueue(PriorityQueueConfig{MaxSize: 10, MaxWaitTime: time.Second})
	defer pq.Stop()

	low := &Request{ID: 1, Priority: PriorityLow}
	high := &Request{ID: 2, Priority: PriorityHigh}

	if err := pq.Enqueue(low); err != nil {
		t.Fatalf("enqueue low: %v", err)
	}
	if err := pq.Enqueue(high); err != nil {
		t.Fatalf("enqueue high: %v", err)
	}

	// Give a would-be internal consumer ample time to steal the entries.
	time.Sleep(50 * time.Millisecond)

	first, ok := pq.Dequeue()
	if !ok || first == nil {
		t.Fatal("high-priority request never reached the consumer")
	}
	if first.ID != high.ID {
		t.Fatalf("first dequeued request = %d, want the high-priority one (%d)", first.ID, high.ID)
	}

	second, ok := pq.Dequeue()
	if !ok || second == nil {
		t.Fatal("low-priority request never reached the consumer")
	}
	if second.ID != low.ID {
		t.Fatalf("second dequeued request = %d, want the low-priority one (%d)", second.ID, low.ID)
	}

	stats := pq.GetStats()
	if got := stats["total_dequeued"]; got != int64(2) {
		t.Errorf("total_dequeued = %v, want 2", got)
	}
}
