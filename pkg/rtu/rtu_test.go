package rtu

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"
)

// MockSerialPort implements SerialPort for testing
type MockSerialPort struct {
	mu          sync.Mutex
	readBuffer  *bytes.Buffer
	writeBuffer *bytes.Buffer
	readDelay   time.Duration
	writeDelay  time.Duration
	readError   error
	writeError  error
	closed      bool
	deadlineSet bool
}

func NewMockSerialPort() *MockSerialPort {
	return &MockSerialPort{
		readBuffer:  &bytes.Buffer{},
		writeBuffer: &bytes.Buffer{},
	}
}

func (m *MockSerialPort) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, io.EOF
	}

	if m.readDelay > 0 {
		m.mu.Unlock()
		time.Sleep(m.readDelay)
		m.mu.Lock()
	}

	if m.readError != nil {
		return 0, m.readError
	}

	// bytes.Buffer.Read returns (0, io.EOF) when empty, which is what we want
	return m.readBuffer.Read(p)
}

func (m *MockSerialPort) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, io.ErrClosedPipe
	}

	if m.writeDelay > 0 {
		time.Sleep(m.writeDelay)
	}

	if m.writeError != nil {
		return 0, m.writeError
	}

	return m.writeBuffer.Write(p)
}

func (m *MockSerialPort) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}

func (m *MockSerialPort) SetDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deadlineSet = true
	return nil
}

// Helper methods for testing
func (m *MockSerialPort) SetReadData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBuffer = bytes.NewBuffer(data)
}

func (m *MockSerialPort) GetWrittenData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeBuffer.Bytes()
}

func (m *MockSerialPort) ClearBuffers() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBuffer = &bytes.Buffer{}
	m.writeBuffer = &bytes.Buffer{}
}

func (m *MockSerialPort) SetReadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readError = err
}

func (m *MockSerialPort) SetWriteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeError = err
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Device != "/dev/ttyUSB0" {
		t.Errorf("Expected device '/dev/ttyUSB0', got '%s'", cfg.Device)
	}

	if cfg.BaudRate != 9600 {
		t.Errorf("Expected baud rate 9600, got %d", cfg.BaudRate)
	}

	if cfg.DataBits != 8 {
		t.Errorf("Expected data bits 8, got %d", cfg.DataBits)
	}

	if cfg.Parity != "None" {
		t.Errorf("Expected parity 'None', got '%s'", cfg.Parity)
	}

	if cfg.StopBits != 1 {
		t.Errorf("Expected stop bits 1, got %d", cfg.StopBits)
	}

	if cfg.SlaveID != 1 {
		t.Errorf("Expected slave ID 1, got %d", cfg.SlaveID)
	}

	if cfg.Timeout != 1000*time.Millisecond {
		t.Errorf("Expected timeout 1000ms, got %v", cfg.Timeout)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Device:   "/dev/ttyUSB0",
				BaudRate: 9600,
				DataBits: 8,
				Parity:   "None",
				StopBits: 1,
				SlaveID:  1,
			},
			wantErr: false,
		},
		{
			name: "empty device",
			cfg: &Config{
				Device:   "",
				BaudRate: 9600,
				DataBits: 8,
				Parity:   "None",
				StopBits: 1,
				SlaveID:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid baud rate",
			cfg: &Config{
				Device:   "/dev/ttyUSB0",
				BaudRate: 12345,
				DataBits: 8,
				Parity:   "None",
				StopBits: 1,
				SlaveID:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid data bits",
			cfg: &Config{
				Device:   "/dev/ttyUSB0",
				BaudRate: 9600,
				DataBits: 9,
				Parity:   "None",
				StopBits: 1,
				SlaveID:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid parity",
			cfg: &Config{
				Device:   "/dev/ttyUSB0",
				BaudRate: 9600,
				DataBits: 8,
				Parity:   "Invalid",
				StopBits: 1,
				SlaveID:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid stop bits",
			cfg: &Config{
				Device:   "/dev/ttyUSB0",
				BaudRate: 9600,
				DataBits: 8,
				Parity:   "None",
				StopBits: 3,
				SlaveID:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid slave ID",
			cfg: &Config{
				Device:   "/dev/ttyUSB0",
				BaudRate: 9600,
				DataBits: 8,
				Parity:   "None",
				StopBits: 1,
				SlaveID:  0,
			},
			wantErr: true,
		},
		{
			name: "valid slave ID 247",
			cfg: &Config{
				Device:   "/dev/ttyUSB0",
				BaudRate: 9600,
				DataBits: 8,
				Parity:   "None",
				StopBits: 1,
				SlaveID:  247,
			},
			wantErr: false,
		},
		{
			name: "valid slave ID 248 (too high)",
			cfg: &Config{
				Device:   "/dev/ttyUSB0",
				BaudRate: 9600,
				DataBits: 8,
				Parity:   "None",
				StopBits: 1,
				SlaveID:  248,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateCRC(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []byte
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: []byte{0xFF, 0xFF},
		},
		{
			name:     "single byte",
			data:     []byte{0x01},
			expected: []byte{0x7E, 0x80},
		},
		{
			name:     "test frame",
			data:     []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0A},
			expected: []byte{0xC5, 0xCD},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCRC(tt.data)
			if len(result) != 2 {
				t.Errorf("CRC length should be 2, got %d", len(result))
			}
			if result[0] != tt.expected[0] || result[1] != tt.expected[1] {
				t.Errorf("CRC = %02X %02X, expected %02X %02X", result[0], result[1], tt.expected[0], tt.expected[1])
			}
		})
	}
}

func TestNewRTUClient(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()

	client, err := NewRTUClient(cfg, mockPort)
	if err != nil {
		t.Fatalf("Failed to create RTU client: %v", err)
	}

	if client == nil {
		t.Error("Client should not be nil")
	}
}

func TestNewRTUClient_Invalid(t *testing.T) {
	mockPort := NewMockSerialPort()

	// Nil config
	_, err := NewRTUClient(nil, mockPort)
	if err == nil {
		t.Error("Should error with nil config")
	}

	// Nil port
	cfg := DefaultConfig()
	_, err = NewRTUClient(cfg, nil)
	if err == nil {
		t.Error("Should error with nil port")
	}
}

func TestReadCoils(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Prepare response: 01 01 02 55 CD 47 39
	// SlaveID=1, Function=0x01, ByteCount=2, Data=0x55CD (coils 1-8: 01010101, 9-10: 11)
	response := []byte{0x01, 0x01, 0x02, 0x55, 0xCD, 0x47, 0x39}
	mockPort.SetReadData(response)

	coils, err := client.ReadCoils(0, 10)
	if err != nil {
		t.Fatalf("Failed to read coils: %v", err)
	}

	if len(coils) != 10 {
		t.Errorf("Expected 10 coils, got %d", len(coils))
	}

	// Check first 8 coils (0x55 = 01010101)
	expected := []bool{true, false, true, false, true, false, true, false}
	for i := 0; i < 8; i++ {
		if coils[i] != expected[i] {
			t.Errorf("Coil %d: expected %v, got %v", i, expected[i], coils[i])
		}
	}

	// Check written request
	written := mockPort.GetWrittenData()
	if len(written) < 8 {
		t.Errorf("Expected at least 8 bytes written, got %d", len(written))
	}
}

func TestReadHoldingRegisters(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Prepare response: 01 03 04 00 0A 00 14 DA 3E
	// SlaveID=1, Function=0x03, ByteCount=4, Data=[10, 20]
	response := []byte{0x01, 0x03, 0x04, 0x00, 0x0A, 0x00, 0x14, 0xDA, 0x3E}
	mockPort.SetReadData(response)

	registers, err := client.ReadHoldingRegisters(0, 2)
	if err != nil {
		t.Fatalf("Failed to read holding registers: %v", err)
	}

	if len(registers) != 2 {
		t.Errorf("Expected 2 registers, got %d", len(registers))
	}

	if registers[0] != 10 {
		t.Errorf("Expected register[0]=10, got %d", registers[0])
	}

	if registers[1] != 20 {
		t.Errorf("Expected register[1]=20, got %d", registers[1])
	}
}

func TestReadInputRegisters(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Prepare response: 01 04 04 00 64 00 C8 BB CD
	response := []byte{0x01, 0x04, 0x04, 0x00, 0x64, 0x00, 0xC8, 0xBB, 0xCD}
	mockPort.SetReadData(response)

	registers, err := client.ReadInputRegisters(0, 2)
	if err != nil {
		t.Fatalf("Failed to read input registers: %v", err)
	}

	if len(registers) != 2 {
		t.Errorf("Expected 2 registers, got %d", len(registers))
	}

	if registers[0] != 100 {
		t.Errorf("Expected register[0]=100, got %d", registers[0])
	}

	if registers[1] != 200 {
		t.Errorf("Expected register[1]=200, got %d", registers[1])
	}
}

func TestWriteSingleCoil(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Prepare echo response
	response := []byte{0x01, 0x05, 0x00, 0x00, 0xFF, 0x00, 0x8C, 0x3A}
	mockPort.SetReadData(response)

	err := client.WriteSingleCoil(0, true)
	if err != nil {
		t.Fatalf("Failed to write single coil: %v", err)
	}

	written := mockPort.GetWrittenData()
	if len(written) < 8 {
		t.Errorf("Expected 8 bytes written, got %d", len(written))
	}
}

func TestWriteSingleRegister(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Prepare echo response
	response := []byte{0x01, 0x06, 0x00, 0x00, 0x00, 0x0A, 0x09, 0xCD}
	mockPort.SetReadData(response)

	err := client.WriteSingleRegister(0, 10)
	if err != nil {
		t.Fatalf("Failed to write single register: %v", err)
	}

	written := mockPort.GetWrittenData()
	if len(written) < 8 {
		t.Errorf("Expected 8 bytes written, got %d", len(written))
	}
}

func TestWriteMultipleRegisters(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Prepare response
	response := []byte{0x01, 0x10, 0x00, 0x00, 0x00, 0x02, 0x41, 0xC8}
	mockPort.SetReadData(response)

	values := []uint16{100, 200}
	err := client.WriteMultipleRegisters(0, values)
	if err != nil {
		t.Fatalf("Failed to write multiple registers: %v", err)
	}

	written := mockPort.GetWrittenData()
	if len(written) < 11 {
		t.Errorf("Expected at least 11 bytes written, got %d", len(written))
	}
}

func TestExceptionError(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Prepare exception response: 01 83 02 CRC
	response := []byte{0x01, 0x83, 0x02, 0xC0, 0x81}
	mockPort.SetReadData(response)

	_, err := client.ReadHoldingRegisters(0, 1)
	if err == nil {
		t.Error("Expected exception error, got nil")
	}

	exceptErr, ok := err.(*ExceptionError)
	if !ok {
		t.Fatalf("Expected *ExceptionError, got %T", err)
	}

	if exceptErr.Function != 0x03 {
		t.Errorf("Expected function 0x03, got 0x%02X", exceptErr.Function)
	}

	if exceptErr.Code != 0x02 {
		t.Errorf("Expected exception code 0x02, got 0x%02X", exceptErr.Code)
	}
}

func TestExceptionCodeToString(t *testing.T) {
	tests := []struct {
		code     byte
		expected string
	}{
		{0x01, "Illegal function"},
		{0x02, "Illegal data address"},
		{0x03, "Illegal data value"},
		{0x04, "Slave device failure"},
		{0x05, "Acknowledge"},
		{0x06, "Slave device busy"},
		{0x07, "Negative acknowledge"},
		{0x08, "Memory parity error"},
		{0x0A, "Gateway path unavailable"},
		{0x0B, "Gateway target device failed to respond"},
		{0xFF, "Unknown exception (0xFF)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := exceptionCodeToString(tt.code)
			if result != tt.expected {
				t.Errorf("exceptionCodeToString(0x%02X) = %s, expected %s", tt.code, result, tt.expected)
			}
		})
	}
}

func TestReadCoils_InvalidQuantity(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Test quantity > 2000
	_, err := client.ReadCoils(0, 2001)
	if err == nil {
		t.Error("Expected error for quantity > 2000")
	}

	// Test quantity < 1
	_, err = client.ReadCoils(0, 0)
	if err == nil {
		t.Error("Expected error for quantity < 1")
	}
}

func TestReadHoldingRegisters_InvalidQuantity(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Test quantity > 125
	_, err := client.ReadHoldingRegisters(0, 126)
	if err == nil {
		t.Error("Expected error for quantity > 125")
	}
}

func TestInterFrameDelay(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	cfg.InterFrameDelay = 10 * time.Millisecond
	client, _ := NewRTUClient(cfg, mockPort)

	response := []byte{0x01, 0x03, 0x02, 0x00, 0x0A, 0x38, 0x43}
	mockPort.SetReadData(response)

	// Send first request
	_, err := client.ReadHoldingRegisters(0, 1)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	// Prepare second response (clear write buffer from first request, set new read data)
	mockPort.writeBuffer = &bytes.Buffer{}
	mockPort.SetReadData(response)

	start := time.Now()
	_, err = client.ReadHoldingRegisters(0, 1)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < cfg.InterFrameDelay {
		t.Errorf("Expected delay >= %v, got %v", cfg.InterFrameDelay, elapsed)
	}
}

func TestBuildFrame(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	req := &Request{
		SlaveID:  1,
		Function: 0x03,
		Data:     []byte{0x00, 0x00, 0x00, 0x0A},
	}

	frame, err := client.buildFrame(req)
	if err != nil {
		t.Fatalf("Failed to build frame: %v", err)
	}

	// Expected: SlaveID + Function + Data + CRC
	expectedLength := 2 + len(req.Data) + 2
	if len(frame) != expectedLength {
		t.Errorf("Expected frame length %d, got %d", expectedLength, len(frame))
	}

	// Check CRC
	expectedCRC := calculateCRC(frame[:len(frame)-2])
	if frame[len(frame)-2] != expectedCRC[0] || frame[len(frame)-1] != expectedCRC[1] {
		t.Error("CRC mismatch")
	}
}

func TestClose(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	err := client.Close()
	if err != nil {
		t.Errorf("Failed to close client: %v", err)
	}

	// Verify port is closed
	if !mockPort.closed {
		t.Error("Serial port should be closed")
	}
}

func TestReadDiscreteInputs(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Prepare response: 01 02 01 01 60 48
	response := []byte{0x01, 0x02, 0x01, 0x01, 0x60, 0x48}
	mockPort.SetReadData(response)

	inputs, err := client.ReadDiscreteInputs(0, 8)
	if err != nil {
		t.Fatalf("Failed to read discrete inputs: %v", err)
	}

	if len(inputs) != 8 {
		t.Errorf("Expected 8 inputs, got %d", len(inputs))
	}

	// 0x01 = 00000001, only first input is true
	if !inputs[0] {
		t.Error("First input should be true")
	}

	for i := 1; i < 8; i++ {
		if inputs[i] {
			t.Errorf("Input %d should be false", i)
		}
	}
}

func TestWriteMultipleCoils(t *testing.T) {
	mockPort := NewMockSerialPort()
	cfg := DefaultConfig()
	client, _ := NewRTUClient(cfg, mockPort)

	// Prepare response
	response := []byte{0x01, 0x0F, 0x00, 0x00, 0x00, 0x0A, 0xD5, 0xCC}
	mockPort.SetReadData(response)

	values := []bool{true, false, true, false, true, false, true, false, true, false}
	err := client.WriteMultipleCoils(0, values)
	if err != nil {
		t.Fatalf("Failed to write multiple coils: %v", err)
	}

	written := mockPort.GetWrittenData()
	if len(written) < 10 {
		t.Errorf("Expected at least 10 bytes written, got %d", len(written))
	}
}
