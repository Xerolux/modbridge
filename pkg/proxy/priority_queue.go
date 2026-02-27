package proxy

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RequestPriority defines priority levels for requests
type RequestPriority int

const (
	PriorityLow      RequestPriority = 0
	PriorityNormal   RequestPriority = 1
	PriorityHigh     RequestPriority = 2
	PriorityCritical RequestPriority = 3
)

// Request represents a queued request
type Request struct {
	ID            int64
	Priority      RequestPriority
	Data          []byte
	ResponseChan  chan []byte
	ErrorChan     chan error
	SubmittedAt   time.Time
	Deadline      time.Time
	RetryCount    int
	MaxRetries    int
}

// PriorityQueue manages prioritized request queues
type PriorityQueue struct {
	mu            sync.Mutex
	queues        [4][]*Request // One queue per priority level
	cond          *sync.Cond
	maxSize       int
	maxWaitTime   time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	running       bool

	// Metrics
	totalEnqueued  int64
	totalDequeued  int64
	totalRejected  int64
	totalTimeout   int64
}

// PriorityQueueConfig holds configuration
type PriorityQueueConfig struct {
	MaxSize     int           // Maximum queue size (default: 10000)
	MaxWaitTime time.Duration // Maximum time in queue (default: 30s)
}

// DefaultPriorityQueueConfig returns sensible defaults
func DefaultPriorityQueueConfig() PriorityQueueConfig {
	return PriorityQueueConfig{
		MaxSize:     10000,
		MaxWaitTime: 30 * time.Second,
	}
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(config PriorityQueueConfig) *PriorityQueue {
	if config.MaxSize <= 0 {
		config.MaxSize = 10000
	}
	if config.MaxWaitTime <= 0 {
		config.MaxWaitTime = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	pq := &PriorityQueue{
		maxSize:     config.MaxSize,
		maxWaitTime: config.MaxWaitTime,
		ctx:         ctx,
		cancel:      cancel,
		running:     true,
	}

	pq.cond = sync.NewCond(&pq.mu)

	// Start worker goroutine
	pq.wg.Add(1)
	go pq.processLoop()

	// Start timeout checker
	pq.wg.Add(1)
	go pq.timeoutChecker()

	return pq
}

// Enqueue adds a request to the queue
func (pq *PriorityQueue) Enqueue(req *Request) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if !pq.running {
		return ErrQueueClosed
	}

	// Check queue size
	totalSize := 0
	for _, q := range pq.queues {
		totalSize += len(q)
	}

	if totalSize >= pq.maxSize {
		// Check if we can drop low priority requests
		if len(pq.queues[PriorityLow]) > 0 {
			// Drop oldest low priority request
			dropped := pq.queues[PriorityLow][0]
			pq.queues[PriorityLow] = pq.queues[PriorityLow][1:]
			dropped.ErrorChan <- ErrRequestDropped
			close(dropped.ErrorChan)
		} else {
			pq.totalRejected++
			return ErrQueueFull
		}
	}

	req.SubmittedAt = time.Now()
	if req.Deadline.IsZero() {
		req.Deadline = time.Now().Add(pq.maxWaitTime)
	}

	pq.queues[req.Priority] = append(pq.queues[req.Priority], req)
	pq.totalEnqueued++

	// Signal waiting goroutine
	pq.cond.Signal()

	return nil
}

// Dequeue removes and returns the highest priority request
func (pq *PriorityQueue) Dequeue() (*Request, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for !pq.hasRequest() {
		if !pq.running {
			return nil, false
		}
		pq.cond.Wait()
	}

	return pq.getNextRequest(), true
}

// processLoop processes requests from the queue
func (pq *PriorityQueue) processLoop() {
	defer pq.wg.Done()

	for {
		select {
		case <-pq.ctx.Done():
			return
		default:
		}

		_, ok := pq.Dequeue()
		if !ok {
			return
		}

		pq.totalDequeued++

		// Request processing is handled by the caller
		// The response will be sent through ResponseChan
	}
}

// timeoutChecker checks for timed out requests
func (pq *PriorityQueue) timeoutChecker() {
	defer pq.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pq.ctx.Done():
			return
		case <-ticker.C:
			pq.checkTimeouts()
		}
	}
}

// checkTimeouts removes timed out requests
func (pq *PriorityQueue) checkTimeouts() {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	now := time.Now()

	for priority := range pq.queues {
		newQueue := make([]*Request, 0, len(pq.queues[priority]))

		for _, req := range pq.queues[priority] {
			if now.After(req.Deadline) {
				pq.totalTimeout++
				req.ErrorChan <- ErrRequestTimeout
				close(req.ErrorChan)
			} else {
				newQueue = append(newQueue, req)
			}
		}

		pq.queues[priority] = newQueue
	}
}

// hasRequest checks if there are any requests in any queue
func (pq *PriorityQueue) hasRequest() bool {
	for _, q := range pq.queues {
		if len(q) > 0 {
			return true
		}
	}
	return false
}

// getNextRequest returns the highest priority request
func (pq *PriorityQueue) getNextRequest() *Request {
	// Check from highest to lowest priority
	for i := len(pq.queues) - 1; i >= 0; i-- {
		if len(pq.queues[i]) > 0 {
			req := pq.queues[i][0]
			pq.queues[i] = pq.queues[i][1:]
			return req
		}
	}
	return nil
}

// Stop stops the priority queue
func (pq *PriorityQueue) Stop() {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if !pq.running {
		return
	}

	pq.running = false
	pq.cancel()
	pq.cond.Broadcast()

	// Close all pending request channels
	for _, queue := range pq.queues {
		for _, req := range queue {
			req.ErrorChan <- ErrQueueClosed
			close(req.ErrorChan)
		}
	}

	pq.wg.Wait()
}

// GetStats returns queue statistics
func (pq *PriorityQueue) GetStats() map[string]interface{} {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	totalSize := 0
	sizes := make([]int, 4)
	for i, q := range pq.queues {
		sizes[i] = len(q)
		totalSize += len(q)
	}

	return map[string]interface{}{
		"total_queued":    totalSize,
		"max_size":        pq.maxSize,
		"utilization":     float64(totalSize) / float64(pq.maxSize) * 100,
		"critical_queue":  sizes[PriorityCritical],
		"high_queue":      sizes[PriorityHigh],
		"normal_queue":    sizes[PriorityNormal],
		"low_queue":       sizes[PriorityLow],
		"total_enqueued":  pq.totalEnqueued,
		"total_dequeued":  pq.totalDequeued,
		"total_rejected":  pq.totalRejected,
		"total_timeout":   pq.totalTimeout,
	}
}

// Errors
var (
	ErrQueueClosed    = fmt.Errorf("queue is closed")
	ErrQueueFull      = fmt.Errorf("queue is full")
	ErrRequestTimeout = fmt.Errorf("request timed out")
	ErrRequestDropped = fmt.Errorf("request dropped (higher priority)")
)
