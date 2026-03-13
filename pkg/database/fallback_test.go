package database

import (
	"fmt"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 5*time.Second)

	// Initially closed
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected initial state to be Closed, got %v", cb.GetState())
	}

	// Can proceed when closed
	if !cb.CanProceed() {
		t.Error("Expected CanProceed() to return true when closed")
	}

	// Record success, should stay closed
	cb.RecordSuccess()
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state to remain Closed after success, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_OpenAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 5*time.Second)

	// Record failures up to threshold
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Should open after threshold
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected state to be Open after threshold failures, got %v", cb.GetState())
	}

	// Cannot proceed when open
	if cb.CanProceed() {
		t.Error("Expected CanProceed() to return false when open")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Call CanProceed to trigger state transition
	cb.CanProceed()

	// Should be half-open now
	if cb.GetState() != CircuitHalfOpen {
		t.Errorf("Expected state to be HalfOpen after timeout, got %v", cb.GetState())
	}

	// Can proceed when half-open
	if !cb.CanProceed() {
		t.Error("Expected CanProceed() to return true when half-open")
	}
}

func TestCircuitBreaker_CloseAfterSuccesses(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Call CanProceed to trigger state transition to half-open
	cb.CanProceed()

	// Record successes to close
	cb.RecordSuccess()
	cb.RecordSuccess()

	// Should be closed now
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state to be Closed after successes, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_ReopenOnHalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Record failure in half-open
	cb.RecordFailure()

	// Should reopen
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected state to reopen to Open after failure in HalfOpen, got %v", cb.GetState())
	}
}

func TestFallbackCache_SetAndGetDevice(t *testing.T) {
	fc := NewFallbackCache(10)

	device := &Device{
		IP:           "192.168.1.1",
		Name:         "Test Device",
		RequestCount: 100,
	}

	fc.SetDevice(device)

	// Retrieve device
	retrieved, ok := fc.GetDevice(device.IP)
	if !ok {
		t.Fatal("Expected device to be found in cache")
	}

	if retrieved.IP != device.IP {
		t.Errorf("Expected IP %s, got %s", device.IP, retrieved.IP)
	}

	if retrieved.Name != device.Name {
		t.Errorf("Expected Name %s, got %s", device.Name, retrieved.Name)
	}
}

func TestFallbackCache_Eviction(t *testing.T) {
	fc := NewFallbackCache(2) // Small cache size

	// Add more devices than capacity
	for i := 0; i < 3; i++ {
		device := &Device{
			IP:   fmt.Sprintf("192.168.1.%d", i+1),
			Name: fmt.Sprintf("Device %d", i+1),
		}
		fc.SetDevice(device)
	}

	// First device should be evicted
	_, ok := fc.GetDevice("192.168.1.1")
	if ok {
		t.Error("Expected first device to be evicted")
	}

	// Third device should still be there
	_, ok = fc.GetDevice("192.168.1.3")
	if !ok {
		t.Error("Expected third device to be in cache")
	}
}

func TestFallbackCache_IncrementRequestCount(t *testing.T) {
	fc := NewFallbackCache(10)

	ip := "192.168.1.1"

	// Increment multiple times
	for i := 0; i < 5; i++ {
		fc.IncrementRequestCount(ip)
	}

	// Check count
	fc.mu.RLock()
	count := fc.requestCounts[ip]
	fc.mu.RUnlock()

	if count != 5 {
		t.Errorf("Expected request count to be 5, got %d", count)
	}
}

func TestCircuitBreakerState_String(t *testing.T) {
	tests := []struct {
		state    CircuitBreakerState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitBreakerState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("State.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCircuitBreaker_ResetOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 5*time.Second)

	// Add some failures
	cb.RecordFailure()
	cb.RecordFailure()

	// Success should reset failure count
	cb.RecordSuccess()

	// Add two more failures (should not open because previous failures were reset)
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state to remain Closed after success reset failures, got %v", cb.GetState())
	}
}

func TestFallbackCache_ConcurrentAccess(t *testing.T) {
	fc := NewFallbackCache(100)

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			device := &Device{
				IP:   fmt.Sprintf("192.168.1.%d", id),
				Name: fmt.Sprintf("Device %d", id),
			}
			fc.SetDevice(device)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			fc.GetDevice(fmt.Sprintf("192.168.1.%d", id))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify all devices were written
	for i := 0; i < 10; i++ {
		_, ok := fc.GetDevice(fmt.Sprintf("192.168.1.%d", i))
		if !ok {
			t.Errorf("Expected device %d to be in cache", i)
		}
	}
}
