// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// Modbus Function Codes
	FuncReadHoldingRegisters = 0x03
	FuncReadInputRegisters   = 0x04

	// Modbus Exceptions
	ExceptionIllegalFunction    = 0x01
	ExceptionIllegalDataAddress = 0x02
	ExceptionIllegalDataValue   = 0x03
	ExceptionSlaveDeviceFailure = 0x04
)

// IsReadRequest checks if the frame is a Read Holding/Input Registers request.
func IsReadRequest(frame []byte) bool {
	if len(frame) < 12 {
		return false
	}
	fc := frame[7]
	return fc == FuncReadHoldingRegisters || fc == FuncReadInputRegisters
}

// ParseReadRequest extracts parameters from a read request frame.
func ParseReadRequest(frame []byte) (txID uint16, unitID uint8, fc uint8, startAddr uint16, quantity uint16, err error) {
	if len(frame) < 12 {
		return 0, 0, 0, 0, 0, fmt.Errorf("frame too short")
	}
	txID = binary.BigEndian.Uint16(frame[0:2])
	unitID = frame[6]
	fc = frame[7]
	startAddr = binary.BigEndian.Uint16(frame[8:10])
	quantity = binary.BigEndian.Uint16(frame[10:12])
	return
}

// CreateReadRequest constructs a Modbus TCP read request frame.
func CreateReadRequest(txID uint16, unitID uint8, fc uint8, startAddr uint16, quantity uint16) []byte {
	frame := make([]byte, 12)
	binary.BigEndian.PutUint16(frame[0:2], txID)
	// Protocol ID is 0
	frame[2] = 0
	frame[3] = 0
	// Length is 6 (UnitID + FC + StartAddr + Quantity)
	binary.BigEndian.PutUint16(frame[4:6], 6)
	frame[6] = unitID
	frame[7] = fc
	binary.BigEndian.PutUint16(frame[8:10], startAddr)
	binary.BigEndian.PutUint16(frame[10:12], quantity)
	return frame
}

// ParseReadResponse extracts data from a read response frame.
func ParseReadResponse(frame []byte) ([]byte, error) {
	if len(frame) < 9 {
		return nil, fmt.Errorf("frame too short")
	}
	// length := binary.BigEndian.Uint16(frame[4:6])
	fc := frame[7]
	if fc >= 0x80 {
		return nil, fmt.Errorf("modbus exception: %d", frame[8])
	}
	byteCount := frame[8]
	if len(frame) < 9+int(byteCount) {
		return nil, fmt.Errorf("frame data incomplete")
	}
	return frame[9 : 9+byteCount], nil
}

// CreateReadResponse constructs a Modbus TCP read response frame.
func CreateReadResponse(txID uint16, unitID uint8, fc uint8, data []byte) []byte {
	byteCount := len(data)
	// Validate byteCount to prevent overflow
	if byteCount > 255 {
		byteCount = 255
	}
	length := 3 + byteCount // UnitID(1) + FC(1) + ByteCount(1) + Data(N)
	frame := make([]byte, 6+length)

	binary.BigEndian.PutUint16(frame[0:2], txID)
	frame[2] = 0
	frame[3] = 0
	binary.BigEndian.PutUint16(frame[4:6], uint16(length))
	frame[6] = unitID
	frame[7] = fc
	frame[8] = uint8(byteCount) // Safe now due to validation above
	copy(frame[9:], data[:byteCount])
	return frame
}

// ExceptionResponse constructs a Modbus TCP exception response.
func ExceptionResponse(txID uint16, unitID uint8, fc uint8, exceptionCode uint8) []byte {
	length := 3 // UnitID + FC + ExceptionCode
	frame := make([]byte, 6+length)

	binary.BigEndian.PutUint16(frame[0:2], txID)
	frame[2] = 0
	frame[3] = 0
	binary.BigEndian.PutUint16(frame[4:6], uint16(length))
	frame[6] = unitID
	frame[7] = fc | 0x80
	frame[8] = exceptionCode
	return frame
}

// CreateExceptionResponse creates an exception response from a request frame
func CreateExceptionResponse(reqFrame []byte, exceptionCode uint8) []byte {
	if len(reqFrame) < 8 {
		return nil
	}

	txID := binary.BigEndian.Uint16(reqFrame[0:2])
	unitID := reqFrame[6]
	fc := reqFrame[7]

	return ExceptionResponse(txID, unitID, fc, exceptionCode)
}

// --- RTU-over-TCP helpers ---
// These functions convert between Modbus TCP frames (with MBAP header) and
// Modbus RTU frames (with CRC-16 instead of MBAP header).  They are used by
// the proxy when the target speaks RTU over a raw TCP socket.

// TCPToRTU converts a Modbus TCP frame to a Modbus RTU frame.
// It strips the 6-byte MBAP header and appends a CRC-16.
func TCPToRTU(tcpFrame []byte) ([]byte, error) {
	// Minimum TCP frame: 6 (MBAP) + 1 (unit ID) + 1 (function) = 8 bytes
	if len(tcpFrame) < 8 {
		return nil, fmt.Errorf("tcp frame too short for RTU conversion: %d bytes", len(tcpFrame))
	}
	// PDU starts at byte 6 (unit ID + function code + data)
	pdu := tcpFrame[6:]
	rtu := make([]byte, len(pdu)+2)
	copy(rtu, pdu)
	crc := crc16Modbus(pdu)
	rtu[len(pdu)] = crc[0]
	rtu[len(pdu)+1] = crc[1]
	return rtu, nil
}

// RTUToTCP converts a Modbus RTU response frame to a Modbus TCP frame,
// re-using the transaction ID from the original request TCP frame.
func RTUToTCP(rtuFrame []byte, txID uint16) ([]byte, error) {
	// Minimum RTU response: 1 (slave) + 1 (function) + 1 (data) + 2 (CRC) = 5
	if len(rtuFrame) < 5 {
		return nil, fmt.Errorf("rtu frame too short for TCP conversion: %d bytes", len(rtuFrame))
	}
	// Validate CRC
	payload := rtuFrame[:len(rtuFrame)-2]
	expectedCRC := crc16Modbus(payload)
	gotCRC := rtuFrame[len(rtuFrame)-2:]
	if gotCRC[0] != expectedCRC[0] || gotCRC[1] != expectedCRC[1] {
		return nil, fmt.Errorf("RTU CRC mismatch: got %02X%02X, expected %02X%02X",
			gotCRC[0], gotCRC[1], expectedCRC[0], expectedCRC[1])
	}
	// Build TCP frame: MBAP (6) + PDU (without CRC)
	pduLen := uint16(len(payload))
	tcp := make([]byte, 6+int(pduLen))
	binary.BigEndian.PutUint16(tcp[0:2], txID)
	tcp[2] = 0 // Protocol ID high
	tcp[3] = 0 // Protocol ID low
	binary.BigEndian.PutUint16(tcp[4:6], pduLen)
	copy(tcp[6:], payload)
	return tcp, nil
}

// ReadRTUFrame reads one Modbus RTU frame from r, given the expected function
// code and whether an exception response is expected.  It reads enough bytes to
// determine the full frame length, then reads the CRC.
func ReadRTUFrame(r io.Reader, fc byte) ([]byte, error) {
	// Read slave ID + function code
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("rtu header read: %w", err)
	}
	isException := header[1]&0x80 != 0

	var dataLen int
	if isException {
		dataLen = 1 // exception code
	} else {
		switch fc {
		case 0x01, 0x02, 0x03, 0x04:
			// Next byte is byte count
			bc := make([]byte, 1)
			if _, err := io.ReadFull(r, bc); err != nil {
				return nil, fmt.Errorf("rtu byte count read: %w", err)
			}
			// header + byteCount byte + data bytes + CRC
			full := make([]byte, 2+1+int(bc[0])+2)
			copy(full, header)
			full[2] = bc[0]
			if _, err := io.ReadFull(r, full[3:]); err != nil {
				return nil, fmt.Errorf("rtu data read: %w", err)
			}
			return full, nil
		case 0x05, 0x06, 0x0F, 0x10:
			dataLen = 4 // address (2) + value/quantity (2)
		default:
			return nil, fmt.Errorf("unsupported RTU function code 0x%02X", fc)
		}
	}

	rest := make([]byte, dataLen+2) // data + CRC
	if _, err := io.ReadFull(r, rest); err != nil {
		return nil, fmt.Errorf("rtu data read: %w", err)
	}
	frame := append(header, rest...)
	return frame, nil
}

// crc16Modbus calculates the CRC-16/Modbus checksum.
func crc16Modbus(data []byte) []byte {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return []byte{byte(crc), byte(crc >> 8)}
}
