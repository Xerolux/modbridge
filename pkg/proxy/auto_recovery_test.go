// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestRecoveryManagerConcurrentTaskAccess hammers the RecoveryManager with
// many tasks while the task processor scans, marks and completes them. Run
// with -race: this used to trip a data race between executeTask writing
// task.Status/task.Attempts/task.Error and getNextTask/GetStats reading the
// same fields.
func TestRecoveryManagerConcurrentTaskAccess(t *testing.T) {
	cfg := RecoveryConfig{
		MaxConcurrent: 2,
		RetryInterval: 5 * time.Millisecond,
		MaxAttempts:   2,
		TaskTimeout:   500 * time.Millisecond,
	}
	// Targets are unreachable, so tasks go through the full pending →
	// running → failed lifecycle with retries.
	rm := NewRecoveryManager(cfg, nil)
	defer rm.Stop()

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Add tasks in a loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			if _, err := rm.AddTask(fmt.Sprintf("127.0.0.1:%d", 50000+i%200), i%5); err != nil {
				t.Errorf("AddTask failed: %v", err)
			}
		}
	}()

	// Concurrently read stats (reads every task's Status under RLock).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			rm.GetStats()
		}
	}()

	// Concurrently cancel tasks (writes Status under Lock).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = rm.CancelTask("recovery_127.0.0.1:50000_1")
		}
	}()

	time.Sleep(2 * time.Second)
	close(stop)
	wg.Wait()
}
