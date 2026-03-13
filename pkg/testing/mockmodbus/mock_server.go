package mockmodbus

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// MockServer represents a mock Modbus TCP server for testing
type MockServer struct {
	addr        string
	listener    net.Listener
	wg          sync.WaitGroup
	mu          sync.Mutex
	running     bool
	connections map[net.Conn]bool
	delay       time.Duration
	errorRate   float64
	requestLog  []RequestLog
}

// RequestLog represents a logged Modbus request
type RequestLog struct {
	Timestamp    time.Time
	RemoteAddr   string
	FunctionCode byte
	Data         []byte
}

// Config holds the mock server configuration
type Config struct {
	Host        string
	Port        int
	Delay       time.Duration
	ErrorRate   float64 // 0.0 to 1.0
	LogRequests bool
}

// DefaultConfig returns default mock server configuration
func DefaultConfig() Config {
	return Config{
		Host:        "127.0.0.1",
		Port:        15020,
		Delay:       0,
		ErrorRate:   0.0,
		LogRequests: true,
	}
}

// NewMockServer creates a new mock Modbus server
func NewMockServer(config Config) *MockServer {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	return &MockServer{
		addr:        addr,
		connections: make(map[net.Conn]bool),
		delay:       config.Delay,
		errorRate:   config.ErrorRate,
		requestLog:  make([]RequestLog, 0),
	}
}

// Start starts the mock Modbus server
func (m *MockServer) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	m.mu.Unlock()

	listener, err := net.Listen("tcp", m.addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	m.listener = listener
	m.running = true

	m.wg.Add(1)
	go m.acceptConnections()

	return nil
}

// Stop stops the mock Modbus server
func (m *MockServer) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = false
	m.mu.Unlock()

	if m.listener != nil {
		m.listener.Close()
	}

	m.mu.Lock()
	for conn := range m.connections {
		conn.Close()
	}
	m.mu.Unlock()

	m.wg.Wait()
	return nil
}

// GetAddress returns the server address
func (m *MockServer) GetAddress() string {
	return m.addr
}

// acceptConnections accepts incoming connections
func (m *MockServer) acceptConnections() {
	defer m.wg.Done()

	for {
		conn, err := m.listener.Accept()
		if err != nil {
			m.mu.Lock()
			if !m.running {
				m.mu.Unlock()
				return
			}
			m.mu.Unlock()
			continue
		}

		m.mu.Lock()
		m.connections[conn] = true
		m.mu.Unlock()

		m.wg.Add(1)
		go m.handleConnection(conn)
	}
}

// handleConnection handles a single Modbus connection
func (m *MockServer) handleConnection(conn net.Conn) {
	defer m.wg.Done()
	defer func() {
		m.mu.Lock()
		delete(m.connections, conn)
		m.mu.Unlock()
		conn.Close()
	}()

	buf := make([]byte, 256)

	for {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				return
			}
			return
		}

		if n < 8 { // Minimum Modbus TCP frame size
			continue
		}

		// Parse Modbus TCP frame
		transactionID := uint16(buf[0])<<8 | uint16(buf[1])
		protocolID := uint16(buf[2])<<8 | uint16(buf[3])
		_ = uint16(buf[4])<<8 | uint16(buf[5]) // length
		unitID := buf[6]
		functionCode := buf[7]

		// Log request
		m.logRequest(conn.RemoteAddr().String(), functionCode, buf[8:n])

		// Apply delay if configured
		if m.delay > 0 {
			time.Sleep(m.delay)
		}

		// Check if we should return an error
		if m.shouldReturnError() {
			m.sendError(conn, transactionID, protocolID, unitID, functionCode, 0x04) // Gateway target failed
			continue
		}

		// Generate response based on function code
		response := m.generateResponse(transactionID, protocolID, unitID, functionCode, buf[8:n])

		_, err = conn.Write(response)
		if err != nil {
			return
		}
	}
}

// logRequest logs a Modbus request
func (m *MockServer) logRequest(remoteAddr string, functionCode byte, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestLog = append(m.requestLog, RequestLog{
		Timestamp:    time.Now(),
		RemoteAddr:   remoteAddr,
		FunctionCode: functionCode,
		Data:         data,
	})

	// Keep only last 1000 requests
	if len(m.requestLog) > 1000 {
		m.requestLog = m.requestLog[len(m.requestLog)-1000:]
	}
}

// shouldReturnError determines if an error should be returned based on error rate
func (m *MockServer) shouldReturnError() bool {
	if m.errorRate <= 0 {
		return false
	}
	if m.errorRate >= 1.0 {
		return true
	}

	// Simple random check
	return float64(time.Now().UnixNano()%100) < (m.errorRate * 100)
}

// generateResponse generates a Modbus response for the given function code
func (m *MockServer) generateResponse(transactionID, protocolID uint16, unitID, functionCode byte, data []byte) []byte {
	var response []byte

	// MBAP header
	response = append(response,
		byte(transactionID>>8),
		byte(transactionID),
		byte(protocolID>>8),
		byte(protocolID),
		0x00, // Length (placeholder)
		0x00, // Length (placeholder)
		unitID,
		functionCode,
	)

	switch functionCode {
	case 0x01: // Read Coils
		// Always return 8 coils (1 byte)
		response = append(response, 0x01) // Byte count
		response = append(response, 0xFF) // Coil values
		response[4] = 0x00
		response[5] = 0x04 // Length = 3 (1 byte count + 1 coil data + 1 unit ID + 1 func code)

	case 0x02: // Read Discrete Inputs
		response = append(response, 0x01) // Byte count
		response = append(response, 0xAA) // Input values
		response[4] = 0x00
		response[5] = 0x04

	case 0x03: // Read Holding Registers
		response = append(response, 0x04)       // Byte count (2 registers = 4 bytes)
		response = append(response, 0x00, 0x01) // Register values
		response = append(response, 0x00, 0x02)
		response[4] = 0x00
		response[5] = 0x06 // Length

	case 0x04: // Read Input Registers
		response = append(response, 0x02)       // Byte count
		response = append(response, 0x00, 0x10) // Register values
		response = append(response, 0x00, 0x20)
		response[4] = 0x00
		response[5] = 0x06

	case 0x05: // Write Single Coil
		response = append(response, data[0], data[1]) // Address
		response = append(response, data[2])          // Value
		response[4] = 0x00
		response[5] = 0x06

	case 0x06: // Write Single Register
		response = append(response, data[0], data[1]) // Address
		response = append(response, data[2], data[3]) // Value
		response[4] = 0x00
		response[5] = 0x06

	case 0x0F: // Write Multiple Coils
		response = append(response, data[0], data[1]) // Address
		response = append(response, data[2], data[3]) // Quantity
		response[4] = 0x00
		response[5] = 0x06

	case 0x10: // Write Multiple Registers
		response = append(response, data[0], data[1]) // Address
		response = append(response, data[2], data[3]) // Quantity
		response[4] = 0x00
		response[5] = 0x06

	default:
		// Return error for unsupported function codes
		m.sendErrorToBuffer(&response, transactionID, protocolID, unitID, functionCode, 0x01)
	}

	return response
}

// sendError sends a Modbus error response
func (m *MockServer) sendError(conn net.Conn, transactionID, protocolID uint16, unitID, functionCode, exceptionCode byte) {
	response := []byte{
		byte(transactionID >> 8),
		byte(transactionID),
		byte(protocolID >> 8),
		byte(protocolID),
		0x00,
		0x03, // Length
		unitID,
		functionCode | 0x80, // Error bit set
		exceptionCode,
	}

	conn.Write(response)
}

// sendErrorToBuffer appends an error response to the buffer
func (m *MockServer) sendErrorToBuffer(response *[]byte, transactionID, protocolID uint16, unitID, functionCode, exceptionCode byte) {
	*response = append(*response,
		byte(transactionID>>8),
		byte(transactionID),
		byte(protocolID>>8),
		byte(protocolID),
		0x00,
		0x03, // Length
		unitID,
		functionCode|0x80, // Error bit set
		exceptionCode,
	)
}

// GetRequestLog returns the request log
func (m *MockServer) GetRequestLog() []RequestLog {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy
	log := make([]RequestLog, len(m.requestLog))
	copy(log, m.requestLog)

	return log
}

// ClearRequestLog clears the request log
func (m *MockServer) ClearRequestLog() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestLog = make([]RequestLog, 0)
}

// GetConnectionCount returns the current number of connections
func (m *MockServer) GetConnectionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.connections)
}

// SetDelay sets the response delay
func (m *MockServer) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.delay = delay
}

// SetErrorRate sets the error rate (0.0 to 1.0)
func (m *MockServer) SetErrorRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorRate = rate
}

// WaitForConnection waits for at least one connection
func (m *MockServer) WaitForConnection(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for connection")
		case <-ticker.C:
			if m.GetConnectionCount() > 0 {
				return nil
			}
		}
	}
}
