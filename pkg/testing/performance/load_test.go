package performance

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"modbridge/pkg/config"
	"modbridge/pkg/manager"
	"modbridge/pkg/testing/mockmodbus"
)

// LoadTestResult contains the results of a load test
type LoadTestResult struct {
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedReqs       int64
	TotalDuration    time.Duration
	RequestsPerSec   float64
	AvgLatency       time.Duration
	MinLatency       time.Duration
	MaxLatency       time.Duration
	P50Latency       time.Duration
	P95Latency       time.Duration
	P99Latency       time.Duration
	ErrorRate        float64
	ConcurrentUsers  int
}

// BenchmarkOptions contains options for running benchmarks
type BenchmarkOptions struct {
	Duration          time.Duration
	ConcurrentUsers   int
	RequestsPerUser   int
	WarmupDuration    time.Duration
	MockServerPort    int
	ProxyPort         int
	TargetAddress     string
	ResponseDelay     time.Duration
}

// DefaultBenchmarkOptions returns default benchmark options
func DefaultBenchmarkOptions() BenchmarkOptions {
	return BenchmarkOptions{
		Duration:         30 * time.Second,
		ConcurrentUsers:  10,
		RequestsPerUser:  100,
		WarmupDuration:   5 * time.Second,
		MockServerPort:   15050,
		ProxyPort:        15051,
		ResponseDelay:    0,
	}
}

// RunLoadTest runs a load test against the proxy
func RunLoadTest(t *testing.T, opts BenchmarkOptions) *LoadTestResult {
	// Start mock Modbus server
	mockConfig := mockmodbus.DefaultConfig()
	mockConfig.Port = opts.MockServerPort
	mockConfig.Delay = opts.ResponseDelay
	mockServer := mockmodbus.NewMockServer(mockConfig)

	err := mockServer.Start()
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Create and start proxy
	mgr := manager.New()

	proxyCfg := config.ProxyConfig{
		ID:         fmt.Sprintf("load-test-proxy-%d", time.Now().UnixNano()),
		Name:       "Load Test Proxy",
		ListenAddr: fmt.Sprintf(":%d", opts.ProxyPort),
		TargetAddr: mockServer.GetAddress(),
		Enabled:    true,
	}

	err = mgr.AddProxy(&proxyCfg)
	if err != nil {
		t.Fatalf("Failed to add proxy: %v", err)
	}

	err = mgr.StartProxy(proxyCfg.ID)
	if err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer mgr.StopProxy(proxyCfg.ID)

	// Wait for proxy to start
	time.Sleep(200 * time.Millisecond)

	// Warmup period
	if opts.WarmupDuration > 0 {
		t.Logf("Warmup period: %v", opts.WarmupDuration)
		runWarmup(opts.ProxyPort, opts.ConcurrentUsers, opts.WarmupDuration)
	}

	// Run actual load test
	t.Logf("Starting load test: %d concurrent users, %d requests per user",
		opts.ConcurrentUsers, opts.RequestsPerUser)

	result := &LoadTestResult{
		ConcurrentUsers: opts.ConcurrentUsers,
		MinLatency:      time.Hour,
	}

	var wg sync.WaitGroup
	var requestCount int64
	var successCount int64
	var failureCount int64

	latencies := make([]time.Duration, 0, opts.ConcurrentUsers*opts.RequestsPerUser)
	var latenciesMu sync.Mutex

	startTime := time.Now()

	// Start load test goroutines
	for i := 0; i < opts.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", opts.ProxyPort))
			if err != nil {
				atomic.AddInt64(&failureCount, 1)
				return
			}
			defer conn.Close()

			// Create Modbus request template
			request := createModbusRequest(userID)

			for j := 0; j < opts.RequestsPerUser; j++ {
				atomic.AddInt64(&requestCount, 1)

				reqStart := time.Now()

				// Send request
				_, err := conn.Write(request)
				if err != nil {
					atomic.AddInt64(&failureCount, 1)
					continue
				}

				// Read response
				response := make([]byte, 256)
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, err = conn.Read(response)
				latency := time.Since(reqStart)

				if err != nil {
					atomic.AddInt64(&failureCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)

					latenciesMu.Lock()
					latencies = append(latencies, latency)
					latenciesMu.Unlock()

					if latency < result.MinLatency {
						result.MinLatency = latency
					}
					if latency > result.MaxLatency {
						result.MaxLatency = latency
					}
				}

				// Small delay between requests
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	result.TotalDuration = time.Since(startTime)

	// Calculate statistics
	result.TotalRequests = requestCount
	result.SuccessfulReqs = successCount
	result.FailedReqs = failureCount

	if result.TotalDuration > 0 {
		result.RequestsPerSec = float64(successCount) / result.TotalDuration.Seconds()
	}

	if successCount > 0 && len(latencies) > 0 {
		// Calculate percentiles
		result.AvgLatency = calculateAverage(latencies)
		result.P50Latency = calculatePercentile(latencies, 0.50)
		result.P95Latency = calculatePercentile(latencies, 0.95)
		result.P99Latency = calculatePercentile(latencies, 0.99)
	}

	if requestCount > 0 {
		result.ErrorRate = float64(failureCount) / float64(requestCount) * 100
	}

	return result
}

// BenchmarkProxyConnection benchmarks proxy connection establishment
func BenchmarkProxyConnection(b *testing.B) {
	mockConfig := mockmodbus.DefaultConfig()
	mockConfig.Port = 15052
	mockServer := mockmodbus.NewMockServer(mockConfig)

	err := mockServer.Start()
	if err != nil {
		b.Fatalf("Failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	time.Sleep(100 * time.Millisecond)

	mgr := manager.New()

	proxyCfg := config.ProxyConfig{
		ID:         "bench-proxy",
		Name:       "Benchmark Proxy",
		ListenAddr: ":15053",
		TargetAddr: mockServer.GetAddress(),
		Enabled:    true,
	}

	err = mgr.AddProxy(&proxyCfg)
	if err != nil {
		b.Fatalf("Failed to add proxy: %v", err)
	}

	err = mgr.StartProxy(proxyCfg.ID)
	if err != nil {
		b.Fatalf("Failed to start proxy: %v", err)
	}
	defer mgr.StopProxy(proxyCfg.ID)

	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", "127.0.0.1:15053")
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}
		conn.Close()
	}
}

// BenchmarkProxyRequest benchmarks a single Modbus request through proxy
func BenchmarkProxyRequest(b *testing.B) {
	mockConfig := mockmodbus.DefaultConfig()
	mockConfig.Port = 15054
	mockServer := mockmodbus.NewMockServer(mockConfig)

	err := mockServer.Start()
	if err != nil {
		b.Fatalf("Failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	time.Sleep(100 * time.Millisecond)

	mgr := manager.New()

	proxyCfg := config.ProxyConfig{
		ID:         "bench-proxy-req",
		Name:       "Benchmark Request Proxy",
		ListenAddr: ":15055",
		TargetAddr: mockServer.GetAddress(),
		Enabled:    true,
	}

	err = mgr.AddProxy(&proxyCfg)
	if err != nil {
		b.Fatalf("Failed to add proxy: %v", err)
	}

	err = mgr.StartProxy(proxyCfg.ID)
	if err != nil {
		b.Fatalf("Failed to start proxy: %v", err)
	}
	defer mgr.StopProxy(proxyCfg.ID)

	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", "127.0.0.1:15055")
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	request := createModbusRequest(1)

	// Warmup
	for i := 0; i < 100; i++ {
		conn.Write(request)
		response := make([]byte, 256)
		conn.Read(response)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := conn.Write(request)
		if err != nil {
			b.Fatalf("Failed to send request: %v", err)
		}

		response := make([]byte, 256)
		_, err = conn.Read(response)
		if err != nil {
			b.Fatalf("Failed to read response: %v", err)
		}
	}
}

// BenchmarkProxyConcurrent benchmarks concurrent requests
func BenchmarkProxyConcurrent(b *testing.B) {
	mockConfig := mockmodbus.DefaultConfig()
	mockConfig.Port = 15056
	mockServer := mockmodbus.NewMockServer(mockConfig)

	err := mockServer.Start()
	if err != nil {
		b.Fatalf("Failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	time.Sleep(100 * time.Millisecond)

	mgr := manager.New()

	proxyCfg := config.ProxyConfig{
		ID:         "bench-proxy-conc",
		Name:       "Benchmark Concurrent Proxy",
		ListenAddr: ":15057",
		TargetAddr: mockServer.GetAddress(),
		Enabled:    true,
	}

	err = mgr.AddProxy(&proxyCfg)
	if err != nil {
		b.Fatalf("Failed to add proxy: %v", err)
	}

	err = mgr.StartProxy(proxyCfg.ID)
	if err != nil {
		b.Fatalf("Failed to start proxy: %v", err)
	}
	defer mgr.StopProxy(proxyCfg.ID)

	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		conn, err := net.Dial("tcp", "127.0.0.1:15057")
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		request := createModbusRequest(1)

		for pb.Next() {
			_, err := conn.Write(request)
			if err != nil {
				continue
			}

			response := make([]byte, 256)
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			conn.Read(response)
		}
	})
}

// runWarmup runs a warmup phase
func runWarmup(port int, users int, duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < users; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				return
			}
			defer conn.Close()

			request := createModbusRequest(i)

			for {
				select {
				case <-ctx.Done():
					return
				default:
					conn.Write(request)
					response := make([]byte, 256)
					conn.SetReadDeadline(time.Now().Add(1 * time.Second))
					conn.Read(response)
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	wg.Wait()
}

// createModbusRequest creates a Modbus read holding registers request
func createModbusRequest(transactionID int) []byte {
	return []byte{
		byte(transactionID >> 8),
		byte(transactionID),
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,       // Unit ID
		0x03,       // Function code (Read Holding Registers)
		0x00, 0x00, // Address
		0x00, 0x01, // Quantity
	}
}

// calculateAverage calculates the average of durations
func calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}

	return sum / time.Duration(len(durations))
}

// calculatePercentile calculates the percentile of durations
func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Sort durations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// Simple insertion sort
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}

// PrintLoadTestResult prints a formatted load test result
func PrintLoadTestResult(result *LoadTestResult) {
	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration:           %v\n", result.TotalDuration)
	fmt.Printf("Concurrent Users:   %d\n", result.ConcurrentUsers)
	fmt.Printf("Total Requests:     %d\n", result.TotalRequests)
	fmt.Printf("Successful:         %d\n", result.SuccessfulReqs)
	fmt.Printf("Failed:             %d\n", result.FailedReqs)
	fmt.Printf("Error Rate:         %.2f%%\n", result.ErrorRate)
	fmt.Printf("Requests/sec:       %.2f\n", result.RequestsPerSec)
	fmt.Println("\nLatency:")
	fmt.Printf("  Average:          %v\n", result.AvgLatency)
	fmt.Printf("  Min:              %v\n", result.MinLatency)
	fmt.Printf("  Max:              %v\n", result.MaxLatency)
	fmt.Printf("  P50:              %v\n", result.P50Latency)
	fmt.Printf("  P95:              %v\n", result.P95Latency)
	fmt.Printf("  P99:              %v\n", result.P99Latency)
	fmt.Println("========================\n")
}

// TestProxyLoad tests proxy under load
func TestProxyLoad(t *testing.T) {
	opts := DefaultBenchmarkOptions()
	opts.Duration = 10 * time.Second
	opts.ConcurrentUsers = 10
	opts.RequestsPerUser = 50

	result := RunLoadTest(t, opts)
	PrintLoadTestResult(result)

	// Assertions
	if result.ErrorRate > 5.0 {
		t.Errorf("Error rate too high: %.2f%%", result.ErrorRate)
	}

	if result.RequestsPerSec < 10 {
		t.Errorf("Requests per second too low: %.2f", result.RequestsPerSec)
	}

	if result.P99Latency > 1*time.Second {
		t.Errorf("P99 latency too high: %v", result.P99Latency)
	}
}

// TestProxyScalability tests proxy scalability
func TestProxyScalability(t *testing.T) {
	userCounts := []int{1, 5, 10, 20}
	requestsPerUser := 20

	t.Log("Testing proxy scalability...")

	for _, users := range userCounts {
		t.Logf("Testing with %d concurrent users", users)

		opts := DefaultBenchmarkOptions()
		opts.ConcurrentUsers = users
		opts.RequestsPerUser = requestsPerUser
		opts.Duration = 5 * time.Second
		opts.MockServerPort = 15060 + users
		opts.ProxyPort = 15160 + users

		result := RunLoadTest(t, opts)

		t.Logf("  Results: %.2f req/s, P95: %v, Error rate: %.2f%%",
			result.RequestsPerSec, result.P95Latency, result.ErrorRate)

		// Check for degradation
		if result.ErrorRate > 10.0 {
			t.Errorf("High error rate with %d users: %.2f%%", users, result.ErrorRate)
		}
	}
}

// TestProxyLatencyDistribution tests latency distribution
func TestProxyLatencyDistribution(t *testing.T) {
	opts := DefaultBenchmarkOptions()
	opts.ConcurrentUsers = 5
	opts.RequestsPerUser = 100
	opts.Duration = 10 * time.Second

	result := RunLoadTest(t, opts)

	t.Log("Latency Distribution:")
	t.Logf("  P50: %v (median)", result.P50Latency)
	t.Logf("  P95: %v (95th percentile)", result.P95Latency)
	t.Logf("  P99: %v (99th percentile)", result.P99Latency)
	t.Logf("  Min: %v", result.MinLatency)
	t.Logf("  Max: %v", result.MaxLatency)

	// Verify reasonable distribution
	if result.P99Latency > 10*result.P50Latency {
		t.Error("P99 latency is more than 10x P50, indicating tail latency issues")
	}
}

// TestProxySustainedLoad tests proxy under sustained load
func TestProxySustainedLoad(t *testing.T) {
	opts := DefaultBenchmarkOptions()
	opts.Duration = 30 * time.Second
	opts.ConcurrentUsers = 10
	opts.RequestsPerUser = 1000 // Will send continuously for 30s

	result := RunLoadTest(t, opts)

	t.Logf("Sustained load test (%v): %.2f req/s, P95: %v, Error rate: %.2f%%",
		opts.Duration, result.RequestsPerSec, result.P95Latency, result.ErrorRate)

	// Check for performance degradation
	if result.ErrorRate > 1.0 {
		t.Errorf("Error rate increased over time: %.2f%%", result.ErrorRate)
	}

	if result.P99Latency > 500*time.Millisecond {
		t.Errorf("P99 latency degraded over time: %v", result.P99Latency)
	}
}
