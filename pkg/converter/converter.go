package converter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// ModbusTCPHeader represents the MBAP header of Modbus TCP
type ModbusTCPHeader struct {
	TransactionID uint16
	ProtocolID    uint16
	Length        uint16
	UnitID        byte
}

// ModbusTCPFrame represents a complete Modbus TCP frame
type ModbusTCPFrame struct {
	Header   ModbusTCPHeader
	Function byte
	Data     []byte
}

// RTUFrame represents a Modbus RTU frame
type RTUFrame struct {
	SlaveID  byte
	Function byte
	Data     []byte
}

// Config holds converter configuration
type Config struct {
	// Default transaction ID for TCP
	DefaultTransactionID uint16
	// Unit ID to use for TCP->RTU conversion
	DefaultUnitID byte
	// Timeout for operations
	Timeout time.Duration
}

// DefaultConfig returns default converter configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultTransactionID: 0x0001,
		DefaultUnitID:        0x01,
		Timeout:              5 * time.Second,
	}
}

// Converter handles bidirectional conversion between Modbus TCP and RTU
type Converter struct {
	config          *Config
	transactionID   uint16
	transactionIDMu sync.Mutex
}

// NewConverter creates a new Modbus TCP/RTU converter
func NewConverter(cfg *Config) *Converter {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Converter{
		config:        cfg,
		transactionID: cfg.DefaultTransactionID,
	}
}

// TCPToRTU converts a Modbus TCP frame to Modbus RTU format
func (c *Converter) TCPToRTU(tcpFrame []byte) ([]byte, error) {
	if len(tcpFrame) < 8 {
		return nil, errors.New("TCP frame too short: must be at least 8 bytes (MBAP header)")
	}

	// Parse MBAP header
	header := ModbusTCPHeader{
		TransactionID: binary.BigEndian.Uint16(tcpFrame[0:2]),
		ProtocolID:    binary.BigEndian.Uint16(tcpFrame[2:4]),
		Length:        binary.BigEndian.Uint16(tcpFrame[4:6]),
		UnitID:        tcpFrame[6],
	}

	// Validate protocol ID (should be 0 for Modbus)
	if header.ProtocolID != 0 {
		return nil, fmt.Errorf("invalid protocol ID: %d (expected 0)", header.ProtocolID)
	}

	// Validate length
	// Length = UnitID (1) + FunctionCode (1) + Data (Length-2)
	expectedLength := len(tcpFrame) - 6
	if int(header.Length) != expectedLength {
		return nil, fmt.Errorf("length mismatch: header says %d, actual is %d", header.Length, expectedLength)
	}

	if len(tcpFrame) < 7 {
		return nil, errors.New("TCP frame too short: missing function code")
	}

	functionCode := tcpFrame[7]
	var data []byte
	if len(tcpFrame) > 8 {
		data = tcpFrame[8:]
	}

	// Build RTU frame: SlaveID + Function + Data + CRC
	rtuFrame := make([]byte, 1, 2+len(data)+2)
	rtuFrame[0] = header.UnitID // Use UnitID as SlaveID

	// Add function code
	rtuFrame = append(rtuFrame, functionCode)

	// Add data
	rtuFrame = append(rtuFrame, data...)

	// Calculate and append CRC
	crc := calculateCRC(rtuFrame)
	rtuFrame = append(rtuFrame, crc...)

	return rtuFrame, nil
}

// RTUToTCP converts a Modbus RTU frame to Modbus TCP format
func (c *Converter) RTUToTCP(rtuFrame []byte, transactionID uint16) ([]byte, error) {
	if len(rtuFrame) < 4 {
		return nil, errors.New("RTU frame too short: must be at least 4 bytes (SlaveID + Function + 1 byte + CRC)")
	}

	// Extract SlaveID, Function, and Data (excluding CRC)
	slaveID := rtuFrame[0]
	functionCode := rtuFrame[1]

	// Data is everything between Function and CRC (last 2 bytes)
	dataLength := len(rtuFrame) - 4 // Exclude SlaveID, Function, and CRC (2 bytes)
	if dataLength < 0 {
		return nil, errors.New("RTU frame too short")
	}

	var data []byte
	if dataLength > 0 {
		data = rtuFrame[2 : 2+dataLength]
	}

	// Validate CRC
	expectedCRC := calculateCRC(rtuFrame[:len(rtuFrame)-2])
	actualCRC := rtuFrame[len(rtuFrame)-2:]
	if actualCRC[0] != expectedCRC[0] || actualCRC[1] != expectedCRC[1] {
		return nil, fmt.Errorf("CRC mismatch in RTU frame: expected %02X%02X, got %02X%02X",
			expectedCRC[0], expectedCRC[1], actualCRC[0], actualCRC[1])
	}

	// Build TCP frame: MBAP Header + Function + Data
	tcpFrame := make([]byte, 7, 7+len(data)+1)

	// Transaction ID
	binary.BigEndian.PutUint16(tcpFrame[0:2], transactionID)

	// Protocol ID (always 0 for Modbus)
	binary.BigEndian.PutUint16(tcpFrame[2:4], 0)

	// Length = UnitID (1) + Function (1) + Data (variable)
	length := 1 + 1 + len(data)
	binary.BigEndian.PutUint16(tcpFrame[4:6], uint16(length))

	// Unit ID (use SlaveID from RTU)
	tcpFrame[6] = slaveID

	// Function code
	tcpFrame = append(tcpFrame, functionCode)

	// Data
	tcpFrame = append(tcpFrame, data...)

	return tcpFrame, nil
}

// ParseTCPFrame parses a Modbus TCP frame from bytes
func (c *Converter) ParseTCPFrame(data []byte) (*ModbusTCPFrame, error) {
	if len(data) < 8 {
		return nil, errors.New("TCP frame too short: must be at least 8 bytes")
	}

	frame := &ModbusTCPFrame{
		Header: ModbusTCPHeader{
			TransactionID: binary.BigEndian.Uint16(data[0:2]),
			ProtocolID:    binary.BigEndian.Uint16(data[2:4]),
			Length:        binary.BigEndian.Uint16(data[4:6]),
			UnitID:        data[6],
		},
		Function: data[7],
	}

	if len(data) > 8 {
		frame.Data = data[8:]
	}

	// Validate
	if frame.Header.ProtocolID != 0 {
		return nil, fmt.Errorf("invalid protocol ID: %d", frame.Header.ProtocolID)
	}

	expectedLength := 1 + 1 + len(frame.Data)
	if int(frame.Header.Length) != expectedLength {
		return nil, fmt.Errorf("length mismatch: header says %d, actual is %d",
			frame.Header.Length, expectedLength)
	}

	return frame, nil
}

// ParseRTUFrame parses a Modbus RTU frame from bytes
func (c *Converter) ParseRTUFrame(data []byte) (*RTUFrame, error) {
	if len(data) < 4 {
		return nil, errors.New("RTU frame too short: must be at least 4 bytes")
	}

	// Validate CRC
	if len(data) < 4 {
		return nil, errors.New("RTU frame too short for CRC validation")
	}

	expectedCRC := calculateCRC(data[:len(data)-2])
	actualCRC := data[len(data)-2:]
	if actualCRC[0] != expectedCRC[0] || actualCRC[1] != expectedCRC[1] {
		return nil, fmt.Errorf("CRC mismatch: expected %02X%02X, got %02X%02X",
			expectedCRC[0], expectedCRC[1], actualCRC[0], actualCRC[1])
	}

	frame := &RTUFrame{
		SlaveID:  data[0],
		Function: data[1],
	}

	if len(data) > 4 {
		// For function codes 0x01-0x04, skip ByteCount
		// Response format: SlaveID + Function + ByteCount + Data + CRC
		if data[1] == 0x01 || data[1] == 0x02 || data[1] == 0x03 || data[1] == 0x04 {
			if len(data) > 5 {
				frame.Data = data[3 : len(data)-2] // Skip ByteCount
			}
		} else {
			frame.Data = data[2 : len(data)-2]
		}
	}

	return frame, nil
}

// BuildTCPFrame builds a Modbus TCP frame from components
func (c *Converter) BuildTCPFrame(transactionID uint16, unitID byte, function byte, data []byte) []byte {
	length := 1 + 1 + len(data) // UnitID + Function + Data

	frame := make([]byte, 7, 7+len(data))
	binary.BigEndian.PutUint16(frame[0:2], transactionID)
	binary.BigEndian.PutUint16(frame[2:4], 0) // Protocol ID
	binary.BigEndian.PutUint16(frame[4:6], uint16(length))
	frame[6] = unitID
	frame = append(frame, function)
	frame = append(frame, data...)

	return frame
}

// BuildRTUFrame builds a Modbus RTU frame from components
func (c *Converter) BuildRTUFrame(slaveID byte, function byte, data []byte) []byte {
	frame := make([]byte, 2, 2+len(data)+2)
	frame[0] = slaveID
	frame[1] = function
	frame = append(frame, data...)

	crc := calculateCRC(frame)
	frame = append(frame, crc...)

	return frame
}

// GetNextTransactionID returns the next transaction ID (thread-safe)
func (c *Converter) GetNextTransactionID() uint16 {
	c.transactionIDMu.Lock()
	defer c.transactionIDMu.Unlock()

	id := c.transactionID
	c.transactionID++
	if c.transactionID == 0 || c.transactionID > 0xFFFF {
		c.transactionID = 1
	}

	return id
}

// ConvertExceptionToTCP converts an RTU exception response to TCP format
func (c *Converter) ConvertExceptionToTCP(rtuException []byte, transactionID uint16) ([]byte, error) {
	if len(rtuException) < 2 {
		return nil, errors.New("RTU exception too short")
	}

	slaveID := rtuException[0]
	functionCode := rtuException[1]
	exceptionCode := rtuException[2]

	// Build TCP exception response
	tcpFrame := make([]byte, 8) // 7 bytes header + 1 byte exception code
	binary.BigEndian.PutUint16(tcpFrame[0:2], transactionID)
	binary.BigEndian.PutUint16(tcpFrame[2:4], 0) // Protocol ID
	binary.BigEndian.PutUint16(tcpFrame[4:6], 2) // Length = UnitID (1) + Exception code (1)
	tcpFrame[6] = slaveID
	tcpFrame[7] = functionCode | 0x80 // Set exception bit

	if len(rtuException) > 3 {
		tcpFrame = append(tcpFrame, exceptionCode)
	}

	return tcpFrame, nil
}

// ConvertExceptionToRTU converts a TCP exception response to RTU format
func (c *Converter) ConvertExceptionToRTU(tcpException []byte) ([]byte, error) {
	if len(tcpException) < 8 {
		return nil, errors.New("TCP exception too short")
	}

	unitID := tcpException[6]
	functionCode := tcpException[7]

	var exceptionCode byte
	if len(tcpException) > 8 {
		exceptionCode = tcpException[8]
	}

	// Build RTU exception response: SlaveID + Function|0x80 + ExceptionCode + CRC
	rtuFrame := []byte{unitID, functionCode, exceptionCode}
	crc := calculateCRC(rtuFrame)
	rtuFrame = append(rtuFrame, crc...)

	return rtuFrame, nil
}

// StreamConverter handles streaming conversion between TCP and RTU
type StreamConverter struct {
	converter *Converter
}

// NewStreamConverter creates a new stream converter
func NewStreamConverter(cfg *Config) *StreamConverter {
	return &StreamConverter{
		converter: NewConverter(cfg),
	}
}

// ConvertTCPToRTUStream reads TCP frames from a reader and writes RTU frames to a writer
func (sc *StreamConverter) ConvertTCPToRTUStream(tcpReader io.Reader, rtuWriter io.Writer) error {
	for {
		// Read partial MBAP header (6 bytes: TransactionID + ProtocolID + Length)
		header := make([]byte, 6)
		_, err := io.ReadFull(tcpReader, header)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read TCP header: %w", err)
		}

		// Get length
		length := binary.BigEndian.Uint16(header[4:6])
		if length > 253 {
			return fmt.Errorf("invalid length: %d (max 253)", length)
		}

		// Read remaining data (UnitID + Function + Data)
		data := make([]byte, length)
		_, err = io.ReadFull(tcpReader, data)
		if err != nil {
			return fmt.Errorf("failed to read TCP data: %w", err)
		}

		// Combine header and data to form complete TCP frame
		tcpFrame := append(header, data...)

		// Convert to RTU
		rtuFrame, err := sc.converter.TCPToRTU(tcpFrame)
		if err != nil {
			return fmt.Errorf("failed to convert TCP to RTU: %w", err)
		}

		// Write RTU frame
		_, err = rtuWriter.Write(rtuFrame)
		if err != nil {
			return fmt.Errorf("failed to write RTU frame: %w", err)
		}
	}
}

// ConvertRTUToTCPStream reads RTU frames from a reader and writes TCP frames to a writer
func (sc *StreamConverter) ConvertRTUToTCPStream(rtuReader io.Reader, tcpWriter io.Writer, transactionID uint16) error {
	for {
		// Read SlaveID and Function
		header := make([]byte, 2)
		_, err := io.ReadFull(rtuReader, header)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read RTU header: %w", err)
		}

		function := header[1]

		// Determine data length based on function code
		var dataLength int
		switch function {
		case 0x01, 0x02: // Read coils, discrete inputs
			// Read byte count
			byteCount := make([]byte, 1)
			_, err = io.ReadFull(rtuReader, byteCount)
			if err != nil {
				return fmt.Errorf("failed to read byte count: %w", err)
			}
			dataLength = int(byteCount[0])

			// Read data
			data := make([]byte, dataLength)
			_, err = io.ReadFull(rtuReader, data)
			if err != nil {
				return fmt.Errorf("failed to read data: %w", err)
			}

			// Read CRC
			crc := make([]byte, 2)
			_, err = io.ReadFull(rtuReader, crc)
			if err != nil {
				return fmt.Errorf("failed to read CRC: %w", err)
			}

			// Build complete RTU frame and convert
			fullRTU := append(append(header, byteCount...), data...)
			fullRTU = append(fullRTU, crc...)

			tcpFrame, err := sc.converter.RTUToTCP(fullRTU, transactionID)
			if err != nil {
				return fmt.Errorf("failed to convert RTU to TCP: %w", err)
			}

			_, err = tcpWriter.Write(tcpFrame)
			if err != nil {
				return fmt.Errorf("failed to write TCP frame: %w", err)
			}

		case 0x03, 0x04: // Read holding/input registers
			// Similar to coils
			byteCount := make([]byte, 1)
			_, err = io.ReadFull(rtuReader, byteCount)
			if err != nil {
				return fmt.Errorf("failed to read byte count: %w", err)
			}
			dataLength = int(byteCount[0])

			data := make([]byte, dataLength)
			_, err = io.ReadFull(rtuReader, data)
			if err != nil {
				return fmt.Errorf("failed to read data: %w", err)
			}

			crc := make([]byte, 2)
			_, err = io.ReadFull(rtuReader, crc)
			if err != nil {
				return fmt.Errorf("failed to read CRC: %w", err)
			}

			fullRTU := append(append(header, byteCount...), data...)
			fullRTU = append(fullRTU, crc...)

			tcpFrame, err := sc.converter.RTUToTCP(fullRTU, transactionID)
			if err != nil {
				return fmt.Errorf("failed to convert RTU to TCP: %w", err)
			}

			_, err = tcpWriter.Write(tcpFrame)
			if err != nil {
				return fmt.Errorf("failed to write TCP frame: %w", err)
			}

		case 0x05, 0x06: // Write single coil/register
			// Read address (2) + value (2) = 4 bytes
			data := make([]byte, 4)
			_, err = io.ReadFull(rtuReader, data)
			if err != nil {
				return fmt.Errorf("failed to read data: %w", err)
			}

			crc := make([]byte, 2)
			_, err = io.ReadFull(rtuReader, crc)
			if err != nil {
				return fmt.Errorf("failed to read CRC: %w", err)
			}

			fullRTU := append(header, data...)
			fullRTU = append(fullRTU, crc...)

			tcpFrame, err := sc.converter.RTUToTCP(fullRTU, transactionID)
			if err != nil {
				return fmt.Errorf("failed to convert RTU to TCP: %w", err)
			}

			_, err = tcpWriter.Write(tcpFrame)
			if err != nil {
				return fmt.Errorf("failed to write TCP frame: %w", err)
			}

		case 0x0F, 0x10: // Write multiple coils/registers
			// Read address (2) + quantity (2) = 4 bytes
			data := make([]byte, 4)
			_, err = io.ReadFull(rtuReader, data)
			if err != nil {
				return fmt.Errorf("failed to read data: %w", err)
			}

			crc := make([]byte, 2)
			_, err = io.ReadFull(rtuReader, crc)
			if err != nil {
				return fmt.Errorf("failed to read CRC: %w", err)
			}

			fullRTU := append(header, data...)
			fullRTU = append(fullRTU, crc...)

			tcpFrame, err := sc.converter.RTUToTCP(fullRTU, transactionID)
			if err != nil {
				return fmt.Errorf("failed to convert RTU to TCP: %w", err)
			}

			_, err = tcpWriter.Write(tcpFrame)
			if err != nil {
				return fmt.Errorf("failed to write TCP frame: %w", err)
			}

		case 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x8F, 0x90: // Exception responses
			// Read exception code
			exceptionCode := make([]byte, 1)
			_, err = io.ReadFull(rtuReader, exceptionCode)
			if err != nil {
				return fmt.Errorf("failed to read exception code: %w", err)
			}

			crc := make([]byte, 2)
			_, err = io.ReadFull(rtuReader, crc)
			if err != nil {
				return fmt.Errorf("failed to read CRC: %w", err)
			}

			fullRTU := append(header, exceptionCode...)
			fullRTU = append(fullRTU, crc...)

			tcpFrame, err := sc.converter.ConvertExceptionToTCP(fullRTU, transactionID)
			if err != nil {
				return fmt.Errorf("failed to convert exception: %w", err)
			}

			_, err = tcpWriter.Write(tcpFrame)
			if err != nil {
				return fmt.Errorf("failed to write TCP frame: %w", err)
			}

		default:
			return fmt.Errorf("unsupported function code: 0x%02X", function)
		}
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

// ValidateTCPFrame validates a Modbus TCP frame
func ValidateTCPFrame(data []byte) error {
	if len(data) < 8 {
		return errors.New("TCP frame too short: must be at least 8 bytes")
	}

	protocolID := binary.BigEndian.Uint16(data[2:4])
	if protocolID != 0 {
		return fmt.Errorf("invalid protocol ID: %d (expected 0)", protocolID)
	}

	length := binary.BigEndian.Uint16(data[4:6])
	if length < 2 || length > 253 {
		return fmt.Errorf("invalid length: %d (must be 2-253)", length)
	}

	expectedLength := len(data) - 6
	if int(length) != expectedLength {
		return fmt.Errorf("length mismatch: header says %d, actual is %d", length, expectedLength)
	}

	return nil
}

// ValidateRTUFrame validates a Modbus RTU frame (excluding CRC)
func ValidateRTUFrame(data []byte) error {
	if len(data) < 3 {
		return errors.New("RTU frame too short: must be at least 3 bytes (SlaveID + Function + 1 byte)")
	}

	// Check function code
	function := data[1]

	// Validate function code range
	if function < 1 || function > 127 {
		// Check for exception response
		if function < 129 || function > 143 {
			return fmt.Errorf("invalid function code: %d", function)
		}
	}

	return nil
}
