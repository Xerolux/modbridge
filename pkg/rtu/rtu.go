package rtu

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// SerialPort interface for serial communication
type SerialPort interface {
	io.ReadWriteCloser
	SetDeadline(time.Time) error
}

// Config holds RTU configuration
type Config struct {
	// Serial port device (e.g., /dev/ttyUSB0, COM1)
	Device string `json:"device" yaml:"device"`
	// Baud rate (common: 9600, 19200, 38400, 57600, 115200)
	BaudRate int `json:"baud_rate" yaml:"baud_rate"`
	// Data bits (typically 8)
	DataBits int `json:"data_bits" yaml:"data_bits"`
	// Parity (None, Odd, Even, Mark, Space)
	Parity string `json:"parity" yaml:"parity"`
	// Stop bits (typically 1)
	StopBits int `json:"stop_bits" yaml:"stop_bits"`
	// Timeout for read/write operations
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
	// Inter-frame delay (t3.5) in milliseconds
	InterFrameDelay time.Duration `json:"inter_frame_delay" yaml:"inter_frame_delay"`
	// Slave ID for this RTU device
	SlaveID byte `json:"slave_id" yaml:"slave_id"`
}

// DefaultConfig returns default RTU configuration
func DefaultConfig() *Config {
	return &Config{
		Device:          "/dev/ttyUSB0",
		BaudRate:        9600,
		DataBits:        8,
		Parity:          "None",
		StopBits:        1,
		Timeout:         1000 * time.Millisecond,
		InterFrameDelay: 4 * time.Millisecond, // ~3.5 character time at 9600 baud
		SlaveID:         1,
	}
}

// RTUClient handles Modbus RTU communication
type RTUClient struct {
	config      *Config
	port        SerialPort
	mu          sync.Mutex
	lastRequest time.Time
}

// NewRTUClient creates a new RTU client
func NewRTUClient(cfg *Config, port SerialPort) (*RTUClient, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if port == nil {
		return nil, errors.New("serial port cannot be nil")
	}

	client := &RTUClient{
		config: cfg,
		port:   port,
	}

	return client, nil
}

// Request represents a Modbus RTU request
type Request struct {
	SlaveID  byte
	Function byte
	Data     []byte
}

// Response represents a Modbus RTU response
type Response struct {
	SlaveID  byte
	Function byte
	Data     []byte
}

// ReadCoils reads coils (function code 0x01)
func (c *RTUClient) ReadCoils(startAddr uint16, quantity uint16) ([]bool, error) {
	if quantity < 1 || quantity > 2000 {
		return nil, fmt.Errorf("invalid coil quantity: %d (must be 1-2000)", quantity)
	}

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], startAddr)
	binary.BigEndian.PutUint16(data[2:4], quantity)

	req := &Request{
		SlaveID:  c.config.SlaveID,
		Function: 0x01,
		Data:     data,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return nil, err
	}

	// Parse coil data
	return c.parseCoilData(resp.Data, quantity)
}

// ReadDiscreteInputs reads discrete inputs (function code 0x02)
func (c *RTUClient) ReadDiscreteInputs(startAddr uint16, quantity uint16) ([]bool, error) {
	if quantity < 1 || quantity > 2000 {
		return nil, fmt.Errorf("invalid input quantity: %d (must be 1-2000)", quantity)
	}

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], startAddr)
	binary.BigEndian.PutUint16(data[2:4], quantity)

	req := &Request{
		SlaveID:  c.config.SlaveID,
		Function: 0x02,
		Data:     data,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return nil, err
	}

	return c.parseCoilData(resp.Data, quantity)
}

// ReadHoldingRegisters reads holding registers (function code 0x03)
func (c *RTUClient) ReadHoldingRegisters(startAddr uint16, quantity uint16) ([]uint16, error) {
	if quantity < 1 || quantity > 125 {
		return nil, fmt.Errorf("invalid register quantity: %d (must be 1-125)", quantity)
	}

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], startAddr)
	binary.BigEndian.PutUint16(data[2:4], quantity)

	req := &Request{
		SlaveID:  c.config.SlaveID,
		Function: 0x03,
		Data:     data,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return nil, err
	}

	return c.parseRegisterData(resp.Data, quantity)
}

// ReadInputRegisters reads input registers (function code 0x04)
func (c *RTUClient) ReadInputRegisters(startAddr uint16, quantity uint16) ([]uint16, error) {
	if quantity < 1 || quantity > 125 {
		return nil, fmt.Errorf("invalid register quantity: %d (must be 1-125)", quantity)
	}

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], startAddr)
	binary.BigEndian.PutUint16(data[2:4], quantity)

	req := &Request{
		SlaveID:  c.config.SlaveID,
		Function: 0x04,
		Data:     data,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return nil, err
	}

	return c.parseRegisterData(resp.Data, quantity)
}

// WriteSingleCoil writes a single coil (function code 0x05)
func (c *RTUClient) WriteSingleCoil(addr uint16, value bool) error {
	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], addr)

	if value {
		binary.BigEndian.PutUint16(data[2:4], 0xFF00)
	} else {
		binary.BigEndian.PutUint16(data[2:4], 0x0000)
	}

	req := &Request{
		SlaveID:  c.config.SlaveID,
		Function: 0x05,
		Data:     data,
	}

	_, err := c.SendRequest(req)
	return err
}

// WriteSingleRegister writes a single register (function code 0x06)
func (c *RTUClient) WriteSingleRegister(addr uint16, value uint16) error {
	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], addr)
	binary.BigEndian.PutUint16(data[2:4], value)

	req := &Request{
		SlaveID:  c.config.SlaveID,
		Function: 0x06,
		Data:     data,
	}

	_, err := c.SendRequest(req)
	return err
}

// WriteMultipleCoils writes multiple coils (function code 0x0F)
func (c *RTUClient) WriteMultipleCoils(startAddr uint16, values []bool) error {
	if len(values) < 1 || len(values) > 1968 {
		return fmt.Errorf("invalid coil count: %d (must be 1-1968)", len(values))
	}

	// Calculate byte count
	byteCount := (len(values) + 7) / 8
	data := make([]byte, 5+byteCount)

	binary.BigEndian.PutUint16(data[0:2], startAddr)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(values)))
	data[4] = byte(byteCount)

	// Pack coil values
	for i, value := range values {
		if value {
			byteIndex := i / 8
			bitIndex := i % 8
			data[5+byteIndex] |= 1 << bitIndex
		}
	}

	req := &Request{
		SlaveID:  c.config.SlaveID,
		Function: 0x0F,
		Data:     data,
	}

	_, err := c.SendRequest(req)
	return err
}

// WriteMultipleRegisters writes multiple registers (function code 0x10)
func (c *RTUClient) WriteMultipleRegisters(startAddr uint16, values []uint16) error {
	if len(values) < 1 || len(values) > 123 {
		return fmt.Errorf("invalid register count: %d (must be 1-123)", len(values))
	}

	byteCount := len(values) * 2
	data := make([]byte, 5+byteCount)

	binary.BigEndian.PutUint16(data[0:2], startAddr)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(values)))
	data[4] = byte(byteCount)

	// Pack register values
	for i, value := range values {
		binary.BigEndian.PutUint16(data[5+i*2:5+i*2+2], value)
	}

	req := &Request{
		SlaveID:  c.config.SlaveID,
		Function: 0x10,
		Data:     data,
	}

	_, err := c.SendRequest(req)
	return err
}

// SendRequest sends a Modbus RTU request and waits for response
func (c *RTUClient) SendRequest(req *Request) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Enforce inter-frame delay
	if !c.lastRequest.IsZero() {
		elapsed := time.Since(c.lastRequest)
		if elapsed < c.config.InterFrameDelay {
			time.Sleep(c.config.InterFrameDelay - elapsed)
		}
	}

	// Build RTU frame
	frame, err := c.buildFrame(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build frame: %w", err)
	}

	// Set deadline
	deadline := time.Now().Add(c.config.Timeout)
	if err := c.port.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	// Send request
	if _, err := c.port.Write(frame); err != nil {
		return nil, fmt.Errorf("failed to write: %w", err)
	}

	c.lastRequest = time.Now()

	// Read response
	resp, err := c.readResponse(req.SlaveID, req.Function)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// buildFrame builds an RTU frame with CRC
func (c *RTUClient) buildFrame(req *Request) ([]byte, error) {
	length := 2 + len(req.Data) // SlaveID + Function + Data + CRC
	frame := make([]byte, length, length+2)

	frame[0] = req.SlaveID
	frame[1] = req.Function
	copy(frame[2:], req.Data)

	// Calculate and append CRC
	crc := calculateCRC(frame[:length])
	frame = append(frame, crc[0], crc[1])

	return frame, nil
}

// readResponse reads and validates an RTU response
func (c *RTUClient) readResponse(expectedSlaveID, expectedFunction byte) (*Response, error) {
	// Read SlaveID and Function code
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.port, header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	slaveID := header[0]
	function := header[1]

	// Check for exception response
	if function&0x80 != 0 {
		// Read exception code
		exception := make([]byte, 3)
		if _, err := io.ReadFull(c.port, exception); err != nil {
			return nil, fmt.Errorf("failed to read exception: %w", err)
		}
		return nil, &ExceptionError{
			Function: expectedFunction,
			Code:     exception[0],
		}
	}

	// Validate slave ID and function
	if slaveID != expectedSlaveID {
		return nil, fmt.Errorf("unexpected slave ID: got %d, expected %d", slaveID, expectedSlaveID)
	}

	if function != expectedFunction {
		return nil, fmt.Errorf("unexpected function code: got %d, expected %d", function, expectedFunction)
	}

	// Determine response length based on function code
	var dataLength int
	var byteCount byte // Store for CRC calculation
	switch function {
	case 0x01, 0x02: // Read coils, discrete inputs
		// Read byte count
		byteCountArr := make([]byte, 1)
		if _, err := io.ReadFull(c.port, byteCountArr); err != nil {
			return nil, fmt.Errorf("failed to read byte count: %w", err)
		}
		byteCount = byteCountArr[0]
		dataLength = int(byteCount) // Just the data bytes, not including byte count

	case 0x03, 0x04: // Read holding/input registers
		// Read byte count
		byteCountArr := make([]byte, 1)
		if _, err := io.ReadFull(c.port, byteCountArr); err != nil {
			return nil, fmt.Errorf("failed to read byte count: %w", err)
		}
		byteCount = byteCountArr[0]
		dataLength = int(byteCount) // Just the data bytes, not including byte count

	case 0x05, 0x06: // Write single coil/register
		dataLength = 4 // Address + value

	case 0x0F, 0x10: // Write multiple coils/registers
		dataLength = 4 // Address + quantity

	default:
		return nil, fmt.Errorf("unsupported function code: %d", function)
	}

	// Read data
	data := make([]byte, dataLength)
	if _, err := io.ReadFull(c.port, data); err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	// Read and validate CRC
	crcBytes := make([]byte, 2)
	if _, err := io.ReadFull(c.port, crcBytes); err != nil {
		return nil, fmt.Errorf("failed to read CRC: %w", err)
	}

	// Build response for CRC validation
	// For function codes 0x01-0x04, include byte count in CRC
	var fullResponse []byte
	switch function {
	case 0x01, 0x02, 0x03, 0x04:
		fullResponse = make([]byte, 3+dataLength) // SlaveID + Function + ByteCount + Data
		fullResponse[0] = slaveID
		fullResponse[1] = function
		fullResponse[2] = byteCount
		copy(fullResponse[3:], data)
	default:
		fullResponse = make([]byte, 2+dataLength) // SlaveID + Function + Data
		fullResponse[0] = slaveID
		fullResponse[1] = function
		copy(fullResponse[2:], data)
	}

	expectedCRC := calculateCRC(fullResponse)
	if crcBytes[0] != expectedCRC[0] || crcBytes[1] != expectedCRC[1] {
		return nil, errors.New("CRC mismatch in response")
	}

	return &Response{
		SlaveID:  slaveID,
		Function: function,
		Data:     data,
	}, nil
}

// parseCoilData parses coil data from response
func (c *RTUClient) parseCoilData(data []byte, quantity uint16) ([]bool, error) {
	if len(data) < 1 {
		return nil, errors.New("no coil data")
	}

	coils := make([]bool, quantity)
	for i := uint16(0); i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if byteIndex >= uint16(len(data)) {
			break // Not enough data
		}
		coils[i] = (data[byteIndex] & (1 << bitIndex)) != 0
	}

	return coils, nil
}

// parseRegisterData parses register data from response
func (c *RTUClient) parseRegisterData(data []byte, quantity uint16) ([]uint16, error) {
	if len(data) < 1 {
		return nil, errors.New("no register data")
	}

	expectedBytes := int(quantity * 2)
	if len(data) < expectedBytes {
		return nil, fmt.Errorf("incomplete register data: expected %d bytes, got %d", expectedBytes, len(data))
	}

	registers := make([]uint16, quantity)
	for i := uint16(0); i < quantity; i++ {
		registers[i] = binary.BigEndian.Uint16(data[i*2 : i*2+2])
	}

	return registers, nil
}

// Close closes the RTU client
func (c *RTUClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.port != nil {
		return c.port.Close()
	}

	return nil
}

// ExceptionError represents a Modbus exception response
type ExceptionError struct {
	Function byte
	Code     byte
}

func (e *ExceptionError) Error() string {
	return fmt.Sprintf("modbus exception 0x%02X for function 0x%02X: %s", e.Code, e.Function, exceptionCodeToString(e.Code))
}

// exceptionCodeToString converts exception code to string
func exceptionCodeToString(code byte) string {
	switch code {
	case 0x01:
		return "Illegal function"
	case 0x02:
		return "Illegal data address"
	case 0x03:
		return "Illegal data value"
	case 0x04:
		return "Slave device failure"
	case 0x05:
		return "Acknowledge"
	case 0x06:
		return "Slave device busy"
	case 0x07:
		return "Negative acknowledge"
	case 0x08:
		return "Memory parity error"
	case 0x0A:
		return "Gateway path unavailable"
	case 0x0B:
		return "Gateway target device failed to respond"
	default:
		return fmt.Sprintf("Unknown exception (0x%02X)", code)
	}
}

// calculateCRC calculates CRC-16 (Modbus) for data
func calculateCRC(data []byte) []byte {
	crc := uint16(0xFFFF)

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}

	return []byte{byte(crc & 0xFF), byte(crc >> 8)}
}

// ValidateConfig validates RTU configuration
func ValidateConfig(cfg *Config) error {
	if cfg.Device == "" {
		return errors.New("device path cannot be empty")
	}

	validBaudRates := map[int]bool{
		300: true, 600: true, 1200: true, 2400: true, 4800: true,
		9600: true, 19200: true, 38400: true, 57600: true, 115200: true,
	}
	if !validBaudRates[cfg.BaudRate] {
		return fmt.Errorf("invalid baud rate: %d", cfg.BaudRate)
	}

	if cfg.DataBits < 5 || cfg.DataBits > 8 {
		return fmt.Errorf("invalid data bits: %d (must be 5-8)", cfg.DataBits)
	}

	validParity := map[string]bool{"None": true, "Odd": true, "Even": true, "Mark": true, "Space": true}
	if !validParity[cfg.Parity] {
		return fmt.Errorf("invalid parity: %s", cfg.Parity)
	}

	if cfg.StopBits < 1 || cfg.StopBits > 2 {
		return fmt.Errorf("invalid stop bits: %d (must be 1-2)", cfg.StopBits)
	}

	if cfg.SlaveID == 0 || cfg.SlaveID > 247 {
		return fmt.Errorf("invalid slave ID: %d (must be 1-247)", cfg.SlaveID)
	}

	return nil
}
