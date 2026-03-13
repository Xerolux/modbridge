package batch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RequestType defines the type of Modbus request
type RequestType int

const (
	RequestTypeReadHoldingRegisters RequestType = iota
	RequestTypeReadInputRegisters
	RequestTypeReadCoils
	RequestTypeReadDiscreteInputs
	RequestTypeWriteSingleRegister
	RequestTypeWriteSingleCoil
	RequestTypeWriteMultipleRegisters
	RequestTypeWriteMultipleCoils
)

// Request represents a single Modbus request
type Request struct {
	Type     RequestType
	SlaveID  byte
	Address  uint16
	Quantity uint16
	Values   []uint16 // For write operations
	Coils    []bool   // For write coil operations
	Metadata any      // Optional metadata for tracking
}

// Response represents a single Modbus response
type Response struct {
	Request *Request // Reference to original request
	Values  []uint16
	Coils   []bool
	Error   error
}

// BatchConfig holds configuration for batching
type BatchConfig struct {
	// Maximum number of requests in a single batch
	MaxBatchSize int `json:"max_batch_size" yaml:"max_batch_size"`
	// Maximum time to wait before flushing a partial batch
	MaxBatchDelay time.Duration `json:"max_batch_delay" yaml:"max_batch_delay"`
	// Maximum gap between consecutive addresses to batch them (0 = no limit)
	MaxAddressGap uint16 `json:"max_address_gap" yaml:"max_address_gap"`
	// Whether to batch across different slave IDs
	CrossSlaveBatching bool `json:"cross_slave_batching" yaml:"cross_slave_batching"`
	// Whether to batch different request types
	CrossTypeBatching bool `json:"cross_type_batching" yaml:"cross_type_batching"`
}

// DefaultBatchConfig returns default batching configuration
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		MaxBatchSize:       125, // Max registers per Modbus read
		MaxBatchDelay:      10 * time.Millisecond,
		MaxAddressGap:      0, // No gap limit by default
		CrossSlaveBatching: false,
		CrossTypeBatching:  false,
	}
}

// Batcher handles request batching
type Batcher struct {
	config *BatchConfig

	mu              sync.Mutex
	pendingRequests []*Request
	requestChan     chan *Request
	responseChan    chan *Response
	ctx             context.Context
	cancel          context.CancelFunc

	// Handler for executing batches
	executeHandler func(ctx context.Context, requests []*Request) []*Response

	// Statistics
	stats BatchStats
}

// BatchStats holds batching statistics
type BatchStats struct {
	TotalRequests     uint64
	TotalBatches      uint64
	TotalCombined     uint64 // Number of requests combined into batches
	AverageBatchSize  float64
	MaxBatchSize      int
	TotalFlushTime    time.Duration
	AverageFlushTime  time.Duration
	PendingRequests   int
	TotalRequestsLost uint64 // Requests that couldn't be batched
}

// NewBatcher creates a new batcher
func NewBatcher(config *BatchConfig, handler func(ctx context.Context, requests []*Request) []*Response) *Batcher {
	if config == nil {
		config = DefaultBatchConfig()
	}
	if handler == nil {
		handler = defaultExecuteHandler
	}

	ctx, cancel := context.WithCancel(context.Background())

	b := &Batcher{
		config:          config,
		requestChan:     make(chan *Request, 1000),
		responseChan:    make(chan *Response, 1000),
		ctx:             ctx,
		cancel:          cancel,
		executeHandler:  handler,
		pendingRequests: make([]*Request, 0, config.MaxBatchSize),
	}

	go b.run()

	return b
}

// defaultExecuteHandler is the default batch execution handler
func defaultExecuteHandler(ctx context.Context, requests []*Request) []*Response {
	responses := make([]*Response, len(requests))
	for i, req := range requests {
		responses[i] = &Response{
			Request: req,
			Error:   errors.New("no handler configured"),
		}
	}
	return responses
}

// Submit submits a request for batching
func (b *Batcher) Submit(req *Request) <-chan *Response {
	responseChan := make(chan *Response, 1)

	// Add metadata to track response
	if req.Metadata == nil {
		req.Metadata = responseChan
	} else {
		// Store response channel in metadata map if possible
		if m, ok := req.Metadata.(map[string]any); ok {
			m["responseChan"] = responseChan
		}
	}

	select {
	case b.requestChan <- req:
		b.mu.Lock()
		b.stats.TotalRequests++
		b.stats.PendingRequests++
		b.mu.Unlock()
	case <-b.ctx.Done():
		close(responseChan)
		return responseChan
	}

	return responseChan
}

// SubmitAsync submits a request and returns immediately
func (b *Batcher) SubmitAsync(req *Request) error {
	select {
	case b.requestChan <- req:
		b.mu.Lock()
		b.stats.TotalRequests++
		b.stats.PendingRequests++
		b.mu.Unlock()
		return nil
	case <-b.ctx.Done():
		return errors.New("batcher is shut down")
	}
}

// Responses returns the response channel
func (b *Batcher) Responses() <-chan *Response {
	return b.responseChan
}

// run is the main batching loop
func (b *Batcher) run() {
	defer close(b.responseChan)

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	timerActive := false

	for {
		select {
		case req := <-b.requestChan:
			b.handleRequest(req)

			b.mu.Lock()
			pendingCount := len(b.pendingRequests)
			b.mu.Unlock()

			if pendingCount > 0 {
				if timerActive {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
				}
				timer.Reset(b.config.MaxBatchDelay)
				timerActive = true
			} else if timerActive {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timerActive = false
			}

		case <-timer.C:
			timerActive = false
			b.flush()

			// Check if requests arrived during flush
			b.mu.Lock()
			pendingCount := len(b.pendingRequests)
			b.mu.Unlock()

			if pendingCount > 0 {
				timer.Reset(b.config.MaxBatchDelay)
				timerActive = true
			}

		case <-b.ctx.Done():
			if timerActive && !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			// Flush remaining requests
			b.flush()
			return
		}
	}
}

// handleRequest handles an incoming request
func (b *Batcher) handleRequest(req *Request) {
	b.mu.Lock()
	b.pendingRequests = append(b.pendingRequests, req)

	shouldFlush := len(b.pendingRequests) >= b.config.MaxBatchSize

	b.mu.Unlock()

	if shouldFlush {
		b.flush()
		return
	}
}

// flush flushes pending requests as a batch
func (b *Batcher) flush() {
	b.mu.Lock()

	if len(b.pendingRequests) == 0 {
		b.mu.Unlock()
		return
	}

	// Copy pending requests
	requests := make([]*Request, len(b.pendingRequests))
	copy(requests, b.pendingRequests)
	b.pendingRequests = b.pendingRequests[:0]

	// Update statistics
	b.stats.TotalBatches++
	b.stats.PendingRequests -= len(requests)
	if len(requests) > b.stats.MaxBatchSize {
		b.stats.MaxBatchSize = len(requests)
	}
	b.stats.TotalCombined += uint64(len(requests))
	b.stats.AverageBatchSize = float64(b.stats.TotalCombined) / float64(b.stats.TotalBatches)

	b.mu.Unlock()

	// Execute batch
	startTime := time.Now()
	responses := b.executeHandler(b.ctx, requests)
	flushTime := time.Since(startTime)

	b.mu.Lock()
	b.stats.TotalFlushTime += flushTime
	b.stats.AverageFlushTime = time.Duration(uint64(b.stats.TotalFlushTime) / b.stats.TotalBatches)
	b.mu.Unlock()

	// Send responses
	for _, resp := range responses {
		select {
		case b.responseChan <- resp:
			// Try to send to response channel from metadata
			if resp.Request != nil && resp.Request.Metadata != nil {
				if ch, ok := resp.Request.Metadata.(chan *Response); ok {
					select {
					case ch <- resp:
					default:
						// Channel closed or full
					}
					close(ch)
				} else if m, ok := resp.Request.Metadata.(map[string]any); ok {
					if ch, ok := m["responseChan"].(chan *Response); ok {
						select {
						case ch <- resp:
						default:
						}
						close(ch)
					}
				}
			}
		case <-b.ctx.Done():
			return
		}
	}
}

// Close closes the batcher
func (b *Batcher) Close() error {
	b.cancel()
	return nil
}

// Stats returns current statistics
func (b *Batcher) Stats() BatchStats {
	b.mu.Lock()
	defer b.mu.Unlock()

	stats := b.stats
	stats.PendingRequests = len(b.pendingRequests)
	return stats
}

// Optimizer optimizes multiple requests into minimal batches
type Optimizer struct {
	config *BatchConfig
}

// NewOptimizer creates a new request optimizer
func NewOptimizer(config *BatchConfig) *Optimizer {
	if config == nil {
		config = DefaultBatchConfig()
	}
	return &Optimizer{config: config}
}

// Optimize optimizes a list of requests into minimal batches
func (o *Optimizer) Optimize(requests []*Request) [][]*Request {
	if len(requests) == 0 {
		return nil
	}

	// Group by compatibility
	groups := o.groupCompatible(requests)

	// Split groups by size and address gap
	var batches [][]*Request
	for _, group := range groups {
		batches = append(batches, o.splitGroup(group)...)
	}

	return batches
}

// groupCompatible groups requests that can be batched together
func (o *Optimizer) groupCompatible(requests []*Request) [][]*Request {
	var groups [][]*Request

	for _, req := range requests {
		placed := false

		// Try to place in existing group
		for i, group := range groups {
			if o.compatible(group[0], req) {
				groups[i] = append(group, req)
				placed = true
				break
			}
		}

		// Create new group if not placed
		if !placed {
			groups = append(groups, []*Request{req})
		}
	}

	return groups
}

// compatible checks if two requests can be batched together
func (o *Optimizer) compatible(r1, r2 *Request) bool {
	// Check slave ID
	if !o.config.CrossSlaveBatching && r1.SlaveID != r2.SlaveID {
		return false
	}

	// Check request type
	if !o.config.CrossTypeBatching && r1.Type != r2.Type {
		return false
	}

	// Only read operations can be batched
	if !isReadOperation(r1.Type) || !isReadOperation(r2.Type) {
		return false
	}

	// Must be same function code for Modbus
	if r1.Type != r2.Type {
		return false
	}

	return true
}

// isReadOperation checks if a request type is a read operation
func isReadOperation(t RequestType) bool {
	return t == RequestTypeReadHoldingRegisters ||
		t == RequestTypeReadInputRegisters ||
		t == RequestTypeReadCoils ||
		t == RequestTypeReadDiscreteInputs
}

// splitGroup splits a group of compatible requests into batches
func (o *Optimizer) splitGroup(requests []*Request) [][]*Request {
	if len(requests) == 0 {
		return nil
	}

	// Sort requests by address (simple bubble sort for small lists)
	for i := 0; i < len(requests)-1; i++ {
		for j := 0; j < len(requests)-i-1; j++ {
			if requests[j].Address > requests[j+1].Address {
				requests[j], requests[j+1] = requests[j+1], requests[j]
			}
		}
	}

	var batches [][]*Request
	currentBatch := []*Request{requests[0]}

	for i := 1; i < len(requests); i++ {
		req := requests[i]
		lastReq := currentBatch[len(currentBatch)-1]

		// Calculate combined range
		minAddr := currentBatch[0].Address
		maxAddr := req.Address + req.Quantity - 1
		totalRegisters := maxAddr - minAddr + 1

		// Check if we can add to current batch
		canAdd := true

		// Check batch size limit
		if totalRegisters > uint16(o.config.MaxBatchSize) {
			canAdd = false
		}

		// Check address gap
		if o.config.MaxAddressGap > 0 && (req.Address-(lastReq.Address+lastReq.Quantity) > o.config.MaxAddressGap) {
			canAdd = false
		}

		if canAdd {
			currentBatch = append(currentBatch, req)
		} else {
			// Start new batch
			batches = append(batches, currentBatch)
			currentBatch = []*Request{req}
		}
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// CombineRequests combines multiple read requests into a single request
// Returns a combined request and indices mapping for response splitting
func CombineRequests(requests []*Request) (*Request, []int, error) {
	if len(requests) == 0 {
		return nil, nil, errors.New("no requests to combine")
	}

	// All requests must be compatible
	if len(requests) > 1 {
		for i := 1; i < len(requests); i++ {
			if requests[i].SlaveID != requests[0].SlaveID {
				return nil, nil, errors.New("cannot combine requests with different slave IDs")
			}
			if requests[i].Type != requests[0].Type {
				return nil, nil, errors.New("cannot combine requests with different types")
			}
		}
	}

	// Sort by address
	sorted := make([]*Request, len(requests))
	copy(sorted, requests)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Address > sorted[j+1].Address {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// Find range
	minAddr := sorted[0].Address
	maxAddr := sorted[0].Address + sorted[0].Quantity - 1

	for _, req := range sorted[1:] {
		endAddr := req.Address + req.Quantity - 1
		if endAddr > maxAddr {
			maxAddr = endAddr
		}
	}

	totalQuantity := maxAddr - minAddr + 1

	// Create combined request
	combined := &Request{
		Type:     sorted[0].Type,
		SlaveID:  sorted[0].SlaveID,
		Address:  minAddr,
		Quantity: totalQuantity,
		Metadata: sorted, // Store original requests for response splitting
	}

	// Create index mapping
	indices := make([]int, len(sorted))
	for i, req := range sorted {
		indices[i] = int(req.Address - minAddr)
	}

	return combined, indices, nil
}

// SplitResponse splits a combined response into individual responses
func SplitResponse(combined *Response, indices []int) []*Response {
	if combined.Request == nil {
		return nil
	}

	originalRequests, ok := combined.Request.Metadata.([]*Request)
	if !ok {
		return nil
	}

	responses := make([]*Response, len(originalRequests))

	for i, origReq := range originalRequests {
		resp := &Response{
			Request: origReq,
			Error:   combined.Error,
		}

		if combined.Error == nil {
			// Extract relevant portion of values
			startIdx := indices[i]
			endIdx := startIdx + int(origReq.Quantity)

			if combined.Values != nil && startIdx+len(combined.Values) >= 0 && endIdx <= len(combined.Values) {
				resp.Values = combined.Values[startIdx:endIdx]
			} else {
				resp.Error = fmt.Errorf("response index out of range")
			}
		}

		responses[i] = resp
	}

	return responses
}

// RequestPool provides a pool of request objects for reuse
type RequestPool struct {
	pool sync.Pool
}

// NewRequestPool creates a new request pool
func NewRequestPool() *RequestPool {
	return &RequestPool{
		pool: sync.Pool{
			New: func() any {
				return &Request{}
			},
		},
	}
}

// Get gets a request from the pool
func (p *RequestPool) Get() *Request {
	return p.pool.Get().(*Request)
}

// Put returns a request to the pool
func (p *RequestPool) Put(req *Request) {
	// Reset request
	req.Type = 0
	req.SlaveID = 0
	req.Address = 0
	req.Quantity = 0
	req.Values = nil
	req.Coils = nil
	req.Metadata = nil

	p.pool.Put(req)
}

// ResponsePool provides a pool of response objects for reuse
type ResponsePool struct {
	pool sync.Pool
}

// NewResponsePool creates a new response pool
func NewResponsePool() *ResponsePool {
	return &ResponsePool{
		pool: sync.Pool{
			New: func() any {
				return &Response{}
			},
		},
	}
}

// Get gets a response from the pool
func (p *ResponsePool) Get() *Response {
	return p.pool.Get().(*Response)
}

// Put returns a response to the pool
func (p *ResponsePool) Put(resp *Response) {
	// Reset response (but keep slices to reuse capacity)
	resp.Request = nil
	resp.Error = nil
	resp.Values = resp.Values[:0]
	resp.Coils = resp.Coils[:0]

	p.pool.Put(resp)
}
