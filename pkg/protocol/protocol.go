package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ProtocolType represents the Modbus protocol type.
type ProtocolType string

const (
	// ProtocolTCP is Modbus TCP protocol.
	ProtocolTCP ProtocolType = "tcp"
	// ProtocolRTU is Modbus RTU protocol.
	ProtocolRTU ProtocolType = "rtu"
	// ProtocolASCII is Modbus ASCII protocol.
	ProtocolASCII ProtocolType = "ascii"
)

var (
	// ErrInvalidFrame is returned when a frame is invalid.
	ErrInvalidFrame = errors.New("invalid modbus frame")
	// ErrInvalidChecksum is returned when the checksum is invalid.
	ErrInvalidChecksum = errors.New("invalid checksum")
)

// Frame represents a Modbus frame.
type Frame struct {
	TransactionID uint16 // TCP only
	ProtocolID    uint16 // TCP only
	UnitID        byte   // Device address (RTU/TCP)
	FunctionCode  byte
	Data          []byte
	Protocol      ProtocolType
}

// Converter converts between Modbus protocols.
type Converter struct{}

// NewConverter creates a new protocol converter.
func NewConverter() *Converter {
	return &Converter{}
}

// TCPToRTU converts a Modbus TCP frame to RTU format.
func (c *Converter) TCPToRTU(tcpFrame []byte) ([]byte, error) {
	if len(tcpFrame) < 8 {
		return nil, ErrInvalidFrame
	}

	// Extract TCP fields
	// transactionID := binary.BigEndian.Uint16(tcpFrame[0:2])
	// protocolID := binary.BigEndian.Uint16(tcpFrame[2:4])
	// length := binary.BigEndian.Uint16(tcpFrame[4:6])
	unitID := tcpFrame[6]
	functionCode := tcpFrame[7]
	data := tcpFrame[8:]

	// Build RTU frame: [unitID][functionCode][data][crc16]
	rtuFrame := make([]byte, 0, len(data)+4)
	rtuFrame = append(rtuFrame, unitID)
	rtuFrame = append(rtuFrame, functionCode)
	rtuFrame = append(rtuFrame, data...)

	// Calculate and append CRC16
	crc := calculateCRC16(rtuFrame)
	rtuFrame = append(rtuFrame, byte(crc&0xFF))
	rtuFrame = append(rtuFrame, byte(crc>>8))

	return rtuFrame, nil
}

// RTUToTCP converts a Modbus RTU frame to TCP format.
func (c *Converter) RTUToTCP(rtuFrame []byte, transactionID uint16) ([]byte, error) {
	if len(rtuFrame) < 4 {
		return nil, ErrInvalidFrame
	}

	// Verify CRC
	receivedCRC := uint16(rtuFrame[len(rtuFrame)-2]) | (uint16(rtuFrame[len(rtuFrame)-1]) << 8)
	calculatedCRC := calculateCRC16(rtuFrame[:len(rtuFrame)-2])

	if receivedCRC != calculatedCRC {
		return nil, ErrInvalidChecksum
	}

	// Extract RTU fields (without CRC)
	unitID := rtuFrame[0]
	functionCode := rtuFrame[1]
	data := rtuFrame[2 : len(rtuFrame)-2]

	// Build TCP frame: [transID][protID][length][unitID][functionCode][data]
	length := uint16(len(data) + 2) // unitID + functionCode + data
	tcpFrame := make([]byte, 8+len(data))

	binary.BigEndian.PutUint16(tcpFrame[0:2], transactionID)
	binary.BigEndian.PutUint16(tcpFrame[2:4], 0) // Protocol ID always 0
	binary.BigEndian.PutUint16(tcpFrame[4:6], length)
	tcpFrame[6] = unitID
	tcpFrame[7] = functionCode
	copy(tcpFrame[8:], data)

	return tcpFrame, nil
}

// TCPToASCII converts a Modbus TCP frame to ASCII format.
func (c *Converter) TCPToASCII(tcpFrame []byte) ([]byte, error) {
	if len(tcpFrame) < 8 {
		return nil, ErrInvalidFrame
	}

	unitID := tcpFrame[6]
	functionCode := tcpFrame[7]
	data := tcpFrame[8:]

	// Build ASCII frame (without start/end markers)
	asciiData := make([]byte, 0, len(data)+2)
	asciiData = append(asciiData, unitID)
	asciiData = append(asciiData, functionCode)
	asciiData = append(asciiData, data...)

	// Calculate LRC
	lrc := calculateLRC(asciiData)

	// Convert to ASCII hex representation
	result := make([]byte, 0, (len(asciiData)+1)*2+3)
	result = append(result, ':') // Start character

	for _, b := range asciiData {
		result = append(result, fmt.Sprintf("%02X", b)...)
	}

	result = append(result, fmt.Sprintf("%02X", lrc)...)
	result = append(result, '\r', '\n') // End characters

	return result, nil
}

// ASCIIToTCP converts a Modbus ASCII frame to TCP format.
func (c *Converter) ASCIIToTCP(asciiFrame []byte, transactionID uint16) ([]byte, error) {
	if len(asciiFrame) < 5 || asciiFrame[0] != ':' {
		return nil, ErrInvalidFrame
	}

	// Remove start character and CRLF
	hexData := asciiFrame[1:]
	if len(hexData) >= 2 && hexData[len(hexData)-2] == '\r' && hexData[len(hexData)-1] == '\n' {
		hexData = hexData[:len(hexData)-2]
	}

	// Convert ASCII hex to binary
	if len(hexData)%2 != 0 {
		return nil, ErrInvalidFrame
	}

	binaryData := make([]byte, len(hexData)/2)
	for i := 0; i < len(binaryData); i++ {
		_, err := fmt.Sscanf(string(hexData[i*2:i*2+2]), "%02X", &binaryData[i])
		if err != nil {
			return nil, fmt.Errorf("invalid hex data: %w", err)
		}
	}

	if len(binaryData) < 3 {
		return nil, ErrInvalidFrame
	}

	// Verify LRC
	receivedLRC := binaryData[len(binaryData)-1]
	calculatedLRC := calculateLRC(binaryData[:len(binaryData)-1])

	if receivedLRC != calculatedLRC {
		return nil, ErrInvalidChecksum
	}

	// Extract fields (without LRC)
	unitID := binaryData[0]
	functionCode := binaryData[1]
	data := binaryData[2 : len(binaryData)-1]

	// Build TCP frame
	length := uint16(len(data) + 2)
	tcpFrame := make([]byte, 8+len(data))

	binary.BigEndian.PutUint16(tcpFrame[0:2], transactionID)
	binary.BigEndian.PutUint16(tcpFrame[2:4], 0)
	binary.BigEndian.PutUint16(tcpFrame[4:6], length)
	tcpFrame[6] = unitID
	tcpFrame[7] = functionCode
	copy(tcpFrame[8:], data)

	return tcpFrame, nil
}

// calculateCRC16 calculates Modbus RTU CRC-16.
func calculateCRC16(data []byte) uint16 {
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

	return crc
}

// calculateLRC calculates Modbus ASCII LRC (Longitudinal Redundancy Check).
func calculateLRC(data []byte) byte {
	var lrc byte = 0

	for _, b := range data {
		lrc += b
	}

	return (^lrc) + 1 // Two's complement
}

// ParseFrame parses a Modbus frame and determines its protocol type.
func ParseFrame(frame []byte) (*Frame, error) {
	if len(frame) < 2 {
		return nil, ErrInvalidFrame
	}

	// Detect protocol type
	var protocol ProtocolType

	if frame[0] == ':' {
		protocol = ProtocolASCII
	} else if len(frame) >= 8 && binary.BigEndian.Uint16(frame[2:4]) == 0 {
		// Protocol ID is 0 for Modbus TCP
		protocol = ProtocolTCP
	} else {
		protocol = ProtocolRTU
	}

	result := &Frame{
		Protocol: protocol,
	}

	switch protocol {
	case ProtocolTCP:
		if len(frame) < 8 {
			return nil, ErrInvalidFrame
		}

		result.TransactionID = binary.BigEndian.Uint16(frame[0:2])
		result.ProtocolID = binary.BigEndian.Uint16(frame[2:4])
		result.UnitID = frame[6]
		result.FunctionCode = frame[7]
		result.Data = frame[8:]

	case ProtocolRTU:
		if len(frame) < 4 {
			return nil, ErrInvalidFrame
		}

		// Verify CRC
		receivedCRC := uint16(frame[len(frame)-2]) | (uint16(frame[len(frame)-1]) << 8)
		calculatedCRC := calculateCRC16(frame[:len(frame)-2])

		if receivedCRC != calculatedCRC {
			return nil, ErrInvalidChecksum
		}

		result.UnitID = frame[0]
		result.FunctionCode = frame[1]
		result.Data = frame[2 : len(frame)-2]

	case ProtocolASCII:
		// ASCII parsing (simplified, needs full implementation)
		return nil, errors.New("ASCII parsing not fully implemented")
	}

	return result, nil
}
