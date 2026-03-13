package batch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDefaultBatchConfig(t *testing.T) {
	config := DefaultBatchConfig()
	if config == nil {
		t.Fatal("DefaultBatchConfig() returned nil")
	}
	if config.MaxBatchSize != 125 {
		t.Errorf("Expected MaxBatchSize 125, got %d", config.MaxBatchSize)
	}
	if config.MaxBatchDelay != 10*time.Millisecond {
		t.Errorf("Expected MaxBatchDelay 10ms, got %v", config.MaxBatchDelay)
	}
}

func TestNewBatcher(t *testing.T) {
	handler := func(ctx context.Context, requests []*Request) []*Response {
		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{uint16(i)},
			}
		}
		return responses
	}

	batcher := NewBatcher(nil, handler)
	if batcher == nil {
		t.Fatal("NewBatcher() returned nil")
	}
	if batcher.config == nil {
		t.Error("batcher config is nil")
	}
	if batcher.requestChan == nil {
		t.Error("batcher requestChan is nil")
	}
	if batcher.responseChan == nil {
		t.Error("batcher responseChan is nil")
	}

	batcher.Close()
}

func TestBatcher_Submit(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	handler := func(ctx context.Context, requests []*Request) []*Response {
		mu.Lock()
		callCount++
		mu.Unlock()

		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{req.Address},
			}
		}
		return responses
	}

	batcher := NewBatcher(nil, handler)
	defer batcher.Close()

	// Submit single request
	req := &Request{
		Type:     RequestTypeReadHoldingRegisters,
		SlaveID:  1,
		Address:  100,
		Quantity: 1,
	}

	responseChan := batcher.Submit(req)

	// Wait for response
	select {
	case resp := <-responseChan:
		if resp == nil {
			t.Fatal("Received nil response")
		}
		if resp.Values[0] != 100 {
			t.Errorf("Expected value 100, got %d", resp.Values[0])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}

	if callCount != 1 {
		t.Errorf("Expected handler to be called once, got %d", callCount)
	}
}

func TestBatcher_SubmitMultiple(t *testing.T) {
	batchSizes := make(map[int]int)
	var mu sync.Mutex

	handler := func(ctx context.Context, requests []*Request) []*Response {
		mu.Lock()
		batchSizes[len(requests)]++
		mu.Unlock()

		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{req.Address},
			}
		}
		return responses
	}

	config := &BatchConfig{
		MaxBatchSize:  10,
		MaxBatchDelay: 50 * time.Millisecond,
	}

	batcher := NewBatcher(config, handler)
	defer batcher.Close()

	// Submit 25 requests
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := &Request{
				Type:     RequestTypeReadHoldingRegisters,
				SlaveID:  1,
				Address:  uint16(100 + idx),
				Quantity: 1,
			}
			batcher.Submit(req)
		}(i)
	}

	wg.Wait()

	// Wait a bit for all batches to be processed
	time.Sleep(200 * time.Millisecond)

	// Should have 3 batches: 10 + 10 + 5
	mu.Lock()
	totalRequests := 0
	for size, count := range batchSizes {
		totalRequests += size * count
	}
	mu.Unlock()

	if totalRequests != 25 {
		t.Errorf("Expected 25 requests to be processed, got %d", totalRequests)
	}
}

func TestBatcher_FlushOnMaxBatchSize(t *testing.T) {
	flushCalled := false
	var mu sync.Mutex

	handler := func(ctx context.Context, requests []*Request) []*Response {
		mu.Lock()
		if len(requests) == 10 {
			flushCalled = true
		}
		mu.Unlock()

		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{req.Address},
			}
		}
		return responses
	}

	config := &BatchConfig{
		MaxBatchSize:  10,
		MaxBatchDelay: 1 * time.Second, // Long delay
	}

	batcher := NewBatcher(config, handler)
	defer batcher.Close()

	// Submit exactly MaxBatchSize requests
	for i := 0; i < 10; i++ {
		req := &Request{
			Type:     RequestTypeReadHoldingRegisters,
			SlaveID:  1,
			Address:  uint16(i),
			Quantity: 1,
		}
		batcher.Submit(req)
	}

	// Give some time for flush
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	flushed := flushCalled
	mu.Unlock()

	if !flushed {
		t.Error("Expected batch to flush when MaxBatchSize reached")
	}
}

func TestBatcher_Stats(t *testing.T) {
	handler := func(ctx context.Context, requests []*Request) []*Response {
		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{req.Address},
			}
		}
		return responses
	}

	batcher := NewBatcher(nil, handler)
	defer batcher.Close()

	// Submit some requests
	for i := 0; i < 5; i++ {
		req := &Request{
			Type:     RequestTypeReadHoldingRegisters,
			SlaveID:  1,
			Address:  uint16(100 + i),
			Quantity: 1,
		}
		batcher.Submit(req)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	stats := batcher.Stats()
	if stats.TotalRequests != 5 {
		t.Errorf("Expected TotalRequests 5, got %d", stats.TotalRequests)
	}
	if stats.TotalBatches == 0 {
		t.Error("Expected TotalBatches > 0")
	}
}

func TestOptimizer_Optimize(t *testing.T) {
	config := &BatchConfig{
		MaxBatchSize:       10,
		MaxAddressGap:      10,
		CrossSlaveBatching: false,
		CrossTypeBatching:  false,
	}

	optimizer := NewOptimizer(config)

	requests := []*Request{
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 100, Quantity: 5},
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 105, Quantity: 5},
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 200, Quantity: 5},
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 2, Address: 100, Quantity: 5}, // Different slave
	}

	batches := optimizer.Optimize(requests)

	if len(batches) != 3 {
		t.Fatalf("Expected 3 batches, got %d", len(batches))
	}
}

func TestOptimizer_Compatible(t *testing.T) {
	config := &BatchConfig{
		CrossSlaveBatching: false,
		CrossTypeBatching:  false,
	}

	optimizer := NewOptimizer(config)

	r1 := &Request{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 100, Quantity: 5}
	r2 := &Request{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 105, Quantity: 5}
	r3 := &Request{Type: RequestTypeReadHoldingRegisters, SlaveID: 2, Address: 100, Quantity: 5} // Different slave
	r4 := &Request{Type: RequestTypeWriteSingleRegister, SlaveID: 1, Address: 100, Quantity: 1}  // Write operation

	if !optimizer.compatible(r1, r2) {
		t.Error("Expected r1 and r2 to be compatible")
	}

	if optimizer.compatible(r1, r3) {
		t.Error("Expected r1 and r3 to be incompatible (different slave)")
	}

	if optimizer.compatible(r1, r4) {
		t.Error("Expected r1 and r4 to be incompatible (write operation)")
	}
}

func TestCombineRequests(t *testing.T) {
	requests := []*Request{
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 100, Quantity: 5}, // 100-104
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 110, Quantity: 5}, // 110-114
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 105, Quantity: 5}, // 105-109
	}

	combined, indices, err := CombineRequests(requests)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if combined.SlaveID != 1 {
		t.Errorf("Expected SlaveID 1, got %d", combined.SlaveID)
	}

	// Combined should read from 100 to 114 (100-104, 105-109, 110-114)
	if combined.Address != 100 {
		t.Errorf("Expected Address 100, got %d", combined.Address)
	}

	if combined.Quantity != 15 { // 100 to 114 = 15 registers
		t.Errorf("Expected Quantity 15, got %d", combined.Quantity)
	}

	if len(indices) != 3 {
		t.Fatalf("Expected 3 indices, got %d", len(indices))
	}

	// After sorting: 100, 105, 110
	if indices[0] != 0 {
		t.Errorf("Expected indices[0]=0, got %d", indices[0])
	}

	if indices[1] != 5 {
		t.Errorf("Expected indices[1]=5, got %d", indices[1])
	}

	if indices[2] != 10 {
		t.Errorf("Expected indices[2]=10, got %d", indices[2])
	}
}

func TestCombineRequests_Error(t *testing.T) {
	// Different slave IDs
	requests := []*Request{
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 100, Quantity: 5},
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 2, Address: 110, Quantity: 5},
	}

	_, _, err := CombineRequests(requests)
	if err == nil {
		t.Error("Expected error for different slave IDs")
	}

	// Empty requests
	_, _, err = CombineRequests([]*Request{})
	if err == nil {
		t.Error("Expected error for empty requests")
	}
}

func TestSplitResponse(t *testing.T) {
	originalRequests := []*Request{
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 100, Quantity: 5},
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 110, Quantity: 5},
	}

	indices := []int{0, 10}

	combined := &Response{
		Request: &Request{
			Metadata: originalRequests,
		},
		Values: make([]uint16, 15),
	}

	// Fill values
	for i := range combined.Values {
		combined.Values[i] = uint16(i)
	}

	responses := SplitResponse(combined, indices)

	if len(responses) != 2 {
		t.Fatalf("Expected 2 responses, got %d", len(responses))
	}

	// First response should have values 0-4
	if len(responses[0].Values) != 5 {
		t.Errorf("Expected 5 values in first response, got %d", len(responses[0].Values))
	}

	// Second response should have values 10-14
	if len(responses[1].Values) != 5 {
		t.Errorf("Expected 5 values in second response, got %d", len(responses[1].Values))
	}

	if responses[1].Values[0] != 10 {
		t.Errorf("Expected first value of second response to be 10, got %d", responses[1].Values[0])
	}
}

func TestRequestPool(t *testing.T) {
	pool := NewRequestPool()

	req1 := pool.Get()
	if req1 == nil {
		t.Fatal("Get() returned nil")
	}

	req1.Type = RequestTypeReadHoldingRegisters
	req1.SlaveID = 1
	req1.Address = 100
	req1.Quantity = 5

	// Return to pool
	pool.Put(req1)

	// Get again
	req2 := pool.Get()
	if req2 != req1 {
		t.Error("Expected to get same request from pool")
	}

	// Should be reset
	if req2.Type != 0 {
		t.Error("Expected request to be reset")
	}

	pool.Put(req2)
}

func TestResponsePool(t *testing.T) {
	pool := NewResponsePool()

	resp1 := pool.Get()
	if resp1 == nil {
		t.Fatal("Get() returned nil")
	}

	resp1.Values = make([]uint16, 10)
	for i := range resp1.Values {
		resp1.Values[i] = uint16(i)
	}

	// Return to pool
	pool.Put(resp1)

	// Get again
	resp2 := pool.Get()
	if resp2 != resp1 {
		t.Error("Expected to get same response from pool")
	}

	// Should have capacity but length 0
	if cap(resp2.Values) != 10 {
		t.Errorf("Expected capacity 10, got %d", cap(resp2.Values))
	}

	if len(resp2.Values) != 0 {
		t.Errorf("Expected length 0, got %d", len(resp2.Values))
	}

	pool.Put(resp2)
}

func TestBatcher_Close(t *testing.T) {
	handler := func(ctx context.Context, requests []*Request) []*Response {
		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{req.Address},
			}
		}
		return responses
	}

	batcher := NewBatcher(nil, handler)

	// Submit some requests
	for i := 0; i < 5; i++ {
		req := &Request{
			Type:     RequestTypeReadHoldingRegisters,
			SlaveID:  1,
			Address:  uint16(i),
			Quantity: 1,
		}
		batcher.Submit(req)
	}

	// Close should flush pending requests
	batcher.Close()

	// Wait for responses
	time.Sleep(100 * time.Millisecond)

	stats := batcher.Stats()
	if stats.TotalRequests != 5 {
		t.Errorf("Expected 5 requests processed, got %d", stats.TotalRequests)
	}
}

func TestOptimizer_SplitGroup(t *testing.T) {
	config := &BatchConfig{
		MaxBatchSize:  10,
		MaxAddressGap: 5,
	}

	optimizer := NewOptimizer(config)

	// Create a group of requests that will be split
	requests := []*Request{
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 100, Quantity: 2}, // 100-101
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 110, Quantity: 2}, // 110-111 - gap > 5
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 115, Quantity: 2}, // 115-116 - gap <= 5 from 110
	}

	batches := optimizer.splitGroup(requests)

	if len(batches) != 2 {
		t.Fatalf("Expected 2 batches, got %d", len(batches))
	}

	// First batch: 100-101
	if len(batches[0]) != 1 {
		t.Errorf("Expected 1 request in first batch, got %d", len(batches[0]))
	}

	// Second batch: 110-111 and 115-116
	if len(batches[1]) != 2 {
		t.Errorf("Expected 2 requests in second batch, got %d", len(batches[1]))
	}
}

func TestOptimizer_SplitGroup_SortOrder(t *testing.T) {
	config := &BatchConfig{
		MaxBatchSize:  200, // Large enough to fit all
		MaxAddressGap: 50,  // Allow up to 50 register gap
	}

	optimizer := NewOptimizer(config)

	// Create requests in random order
	requests := []*Request{
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 200, Quantity: 2},
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 100, Quantity: 2},
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 150, Quantity: 2},
	}

	batches := optimizer.splitGroup(requests)

	if len(batches) != 1 {
		t.Fatalf("Expected 1 batch, got %d", len(batches))
	}

	// Should be sorted by address
	if batches[0][0].Address != 100 {
		t.Errorf("Expected first request to have address 100, got %d", batches[0][0].Address)
	}

	if batches[0][1].Address != 150 {
		t.Errorf("Expected second request to have address 150, got %d", batches[0][1].Address)
	}

	if batches[0][2].Address != 200 {
		t.Errorf("Expected third request to have address 200, got %d", batches[0][2].Address)
	}
}

func TestBatcher_SubmitAsync(t *testing.T) {
	handler := func(ctx context.Context, requests []*Request) []*Response {
		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{req.Address},
			}
		}
		return responses
	}

	batcher := NewBatcher(nil, handler)
	defer batcher.Close()

	req := &Request{
		Type:     RequestTypeReadHoldingRegisters,
		SlaveID:  1,
		Address:  100,
		Quantity: 1,
	}

	err := batcher.SubmitAsync(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	stats := batcher.Stats()
	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", stats.TotalRequests)
	}
}

func TestSplitResponse_ErrorHandling(t *testing.T) {
	originalRequests := []*Request{
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 100, Quantity: 5},
	}

	indices := []int{0}

	combined := &Response{
		Request: &Request{
			Metadata: originalRequests,
		},
		Error:  errors.New("test error"),
		Values: nil,
	}

	responses := SplitResponse(combined, indices)

	if len(responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(responses))
	}

	if responses[0].Error == nil {
		t.Error("Expected error to be propagated")
	}
}

func TestOptimizer_NoRequests(t *testing.T) {
	config := DefaultBatchConfig()
	optimizer := NewOptimizer(config)

	batches := optimizer.Optimize([]*Request{})

	if batches != nil {
		t.Error("Expected nil for empty requests")
	}
}

func TestOptimizer_SingleRequest(t *testing.T) {
	config := DefaultBatchConfig()
	optimizer := NewOptimizer(config)

	requests := []*Request{
		{Type: RequestTypeReadHoldingRegisters, SlaveID: 1, Address: 100, Quantity: 5},
	}

	batches := optimizer.Optimize(requests)

	if len(batches) != 1 {
		t.Fatalf("Expected 1 batch, got %d", len(batches))
	}

	if len(batches[0]) != 1 {
		t.Errorf("Expected 1 request in batch, got %d", len(batches[0]))
	}
}

func TestBatcher_ResponseChannel(t *testing.T) {
	handler := func(ctx context.Context, requests []*Request) []*Response {
		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{req.Address},
			}
		}
		return responses
	}

	batcher := NewBatcher(nil, handler)
	defer batcher.Close()

	// Submit request without using Submit (direct to channel)
	req := &Request{
		Type:     RequestTypeReadHoldingRegisters,
		SlaveID:  1,
		Address:  100,
		Quantity: 1,
	}

	batcher.SubmitAsync(req)

	// Wait for response on Responses channel
	select {
	case resp := <-batcher.Responses():
		if resp == nil {
			t.Fatal("Received nil response")
		}
		if resp.Request == nil {
			t.Fatal("Response request is nil")
		}
		if resp.Values[0] != 100 {
			t.Errorf("Expected value 100, got %d", resp.Values[0])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

func TestIsReadOperation(t *testing.T) {
	readOperations := []RequestType{
		RequestTypeReadHoldingRegisters,
		RequestTypeReadInputRegisters,
		RequestTypeReadCoils,
		RequestTypeReadDiscreteInputs,
	}

	for _, op := range readOperations {
		if !isReadOperation(op) {
			t.Errorf("Expected %v to be a read operation", op)
		}
	}

	writeOperations := []RequestType{
		RequestTypeWriteSingleRegister,
		RequestTypeWriteSingleCoil,
		RequestTypeWriteMultipleRegisters,
		RequestTypeWriteMultipleCoils,
	}

	for _, op := range writeOperations {
		if isReadOperation(op) {
			t.Errorf("Expected %v to NOT be a read operation", op)
		}
	}
}

func TestBatchStats_Accumulation(t *testing.T) {
	handler := func(ctx context.Context, requests []*Request) []*Response {
		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{req.Address},
			}
		}
		return responses
	}

	batcher := NewBatcher(nil, handler)
	defer batcher.Close()

	// Submit multiple batches
	for i := 0; i < 30; i++ {
		req := &Request{
			Type:     RequestTypeReadHoldingRegisters,
			SlaveID:  1,
			Address:  uint16(i),
			Quantity: 1,
		}
		batcher.Submit(req)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	stats := batcher.Stats()

	if stats.TotalRequests != 30 {
		t.Errorf("Expected 30 requests, got %d", stats.TotalRequests)
	}

	if stats.TotalBatches == 0 {
		t.Error("Expected at least 1 batch")
	}

	if stats.AverageBatchSize == 0 {
		t.Error("Expected average batch size > 0")
	}

	if stats.MaxBatchSize == 0 {
		t.Error("Expected max batch size > 0")
	}
}

func TestBatcher_ImmediateFlush(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	handler := func(ctx context.Context, requests []*Request) []*Response {
		mu.Lock()
		callCount++
		count := callCount
		mu.Unlock()

		responses := make([]*Response, len(requests))
		for i, req := range requests {
			responses[i] = &Response{
				Request: req,
				Values:  []uint16{uint16(count)},
			}
		}
		return responses
	}

	config := &BatchConfig{
		MaxBatchSize:  100, // Large to prevent auto-flush
		MaxBatchDelay: 100 * time.Millisecond,
	}

	batcher := NewBatcher(config, handler)
	defer batcher.Close()

	// Submit one request
	req := &Request{
		Type:     RequestTypeReadHoldingRegisters,
		SlaveID:  1,
		Address:  100,
		Quantity: 1,
	}
	responseChan := batcher.Submit(req)

	// Wait for flush due to delay
	select {
	case resp := <-responseChan:
		if resp == nil {
			t.Fatal("Received nil response")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout - flush may not have occurred")
	}

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 1 {
		t.Errorf("Expected 1 handler call, got %d", count)
	}
}
