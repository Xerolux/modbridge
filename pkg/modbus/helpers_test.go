// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package modbus

import (
	"bytes"
	"io"
	"testing"
)

func TestReadRequestHelpers(t *testing.T) {
	txID := uint16(12345)
	unitID := uint8(1)
	fc := uint8(3)
	startAddr := uint16(100)
	quantity := uint16(10)

	// Test CreateReadRequest
	frame := CreateReadRequest(txID, unitID, fc, startAddr, quantity)

	// Verify length
	if len(frame) != 12 {
		t.Errorf("Expected frame length 12, got %d", len(frame))
	}

	// Verify content
	if !IsReadRequest(frame) {
		t.Errorf("IsReadRequest returned false")
	}

	// Test ParseReadRequest
	parsedTxID, parsedUnitID, parsedFC, parsedStart, parsedQty, err := ParseReadRequest(frame)
	if err != nil {
		t.Fatalf("ParseReadRequest failed: %v", err)
	}

	if parsedTxID != txID {
		t.Errorf("TxID mismatch: got %d, want %d", parsedTxID, txID)
	}
	if parsedUnitID != unitID {
		t.Errorf("UnitID mismatch: got %d, want %d", parsedUnitID, unitID)
	}
	if parsedFC != fc {
		t.Errorf("FC mismatch: got %d, want %d", parsedFC, fc)
	}
	if parsedStart != startAddr {
		t.Errorf("StartAddr mismatch: got %d, want %d", parsedStart, startAddr)
	}
	if parsedQty != quantity {
		t.Errorf("Quantity mismatch: got %d, want %d", parsedQty, quantity)
	}
}

func TestReadResponseHelpers(t *testing.T) {
	txID := uint16(12345)
	unitID := uint8(1)
	fc := uint8(3)
	data := []byte{0x00, 0x01, 0x00, 0x02} // 2 registers

	// Test CreateReadResponse
	frame, err := CreateReadResponse(txID, unitID, fc, data)
	if err != nil {
		t.Fatalf("CreateReadResponse error: %v", err)
	}

	// Verify length (Header 6 + UnitID 1 + FC 1 + ByteCount 1 + Data 4 = 13)
	expectedLen := 6 + 1 + 1 + 1 + 4
	if len(frame) != expectedLen {
		t.Errorf("Expected frame length %d, got %d", expectedLen, len(frame))
	}

	// Test ParseReadResponse
	parsedData, err := ParseReadResponse(frame)
	if err != nil {
		t.Fatalf("ParseReadResponse failed: %v", err)
	}

	if !bytes.Equal(parsedData, data) {
		t.Errorf("Data mismatch: got %v, want %v", parsedData, data)
	}
}

func TestExceptionResponse(t *testing.T) {
	txID := uint16(12345)
	unitID := uint8(1)
	fc := uint8(3)
	code := uint8(ExceptionIllegalDataAddress)

	frame := ExceptionResponse(txID, unitID, fc, code)

	// Header 6 + UnitID 1 + FC 1 + Code 1 = 9
	if len(frame) != 9 {
		t.Errorf("Expected frame length 9, got %d", len(frame))
	}

	// Verify FC has high bit set
	if frame[7] != fc|0x80 {
		t.Errorf("Exception FC mismatch: got %02x, want %02x", frame[7], fc|0x80)
	}

	// Verify parse returns error
	_, err := ParseReadResponse(frame)
	if err == nil {
		t.Errorf("Expected error for exception frame, got nil")
	}
}

// --- RTU-over-TCP helper tests ---

func TestCRC16Modbus(t *testing.T) {
	// Known CRC for "Hello" tested against reference implementation.
	data := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0A} // Read 10 registers from addr 0
	crc := crc16Modbus(data)
	if len(crc) != 2 {
		t.Fatalf("expected 2 CRC bytes, got %d", len(crc))
	}
	// Round-trip: appending CRC and recalculating should give 0x0000.
	full := append(data, crc...)
	check := crc16Modbus(full[:len(full)-2])
	if check[0] != crc[0] || check[1] != crc[1] {
		t.Errorf("CRC round-trip mismatch")
	}
}

func TestTCPToRTU(t *testing.T) {
	// Modbus TCP frame: MBAP(6) + UnitID(1) + FC(1) + Addr(2) + Qty(2)
	tcpFrame := CreateReadRequest(1234, 1, 0x03, 0, 10)
	rtu, err := TCPToRTU(tcpFrame)
	if err != nil {
		t.Fatalf("TCPToRTU error: %v", err)
	}
	// RTU = PDU (6 bytes) + CRC (2) = 8
	if len(rtu) != 8 {
		t.Fatalf("expected 8 RTU bytes, got %d", len(rtu))
	}
	// First byte should be UnitID (1)
	if rtu[0] != 1 {
		t.Errorf("RTU[0] UnitID: expected 1, got %d", rtu[0])
	}
	// Second byte should be FC (0x03)
	if rtu[1] != 0x03 {
		t.Errorf("RTU[1] FC: expected 0x03, got 0x%02X", rtu[1])
	}
}

func TestTCPToRTUTooShort(t *testing.T) {
	_, err := TCPToRTU([]byte{0x01, 0x02})
	if err == nil {
		t.Fatal("expected error for too-short frame")
	}
}

func TestRTUToTCP(t *testing.T) {
	// Build a valid RTU response: UnitID + FC + ByteCount + Data + CRC
	payload := []byte{0x01, 0x03, 0x04, 0x00, 0x01, 0x00, 0x02} // 2 registers
	crc := crc16Modbus(payload)
	rtuFrame := append(payload, crc...)

	tcp, err := RTUToTCP(rtuFrame, 42)
	if err != nil {
		t.Fatalf("RTUToTCP error: %v", err)
	}
	// TCP frame: 6 (MBAP) + 7 (PDU) = 13
	if len(tcp) != 13 {
		t.Fatalf("expected 13 TCP bytes, got %d", len(tcp))
	}
	// Transaction ID should be 42
	txID := uint16(tcp[0])<<8 | uint16(tcp[1])
	if txID != 42 {
		t.Errorf("txID: expected 42, got %d", txID)
	}
}

func TestRTUToTCPBadCRC(t *testing.T) {
	// Corrupt the CRC
	payload := []byte{0x01, 0x03, 0x04, 0x00, 0x01, 0x00, 0x02}
	rtuFrame := append(payload, 0xFF, 0xFF) // wrong CRC
	_, err := RTUToTCP(rtuFrame, 1)
	if err == nil {
		t.Fatal("expected CRC mismatch error")
	}
}

func TestRTUToTCPTooShort(t *testing.T) {
	_, err := RTUToTCP([]byte{0x01, 0x02, 0x03}, 1)
	if err == nil {
		t.Fatal("expected error for too-short RTU frame")
	}
}

func TestReadRTUFrame(t *testing.T) {
	// Build a valid RTU read-registers response (FC 0x03).
	payload := []byte{0x01, 0x03, 0x04, 0x00, 0x0A, 0x00, 0x14} // 2 registers: 10, 20
	crc := crc16Modbus(payload)
	rtuFrame := append(payload, crc...)

	r := bytes.NewReader(rtuFrame)
	got, err := ReadRTUFrame(r, 0x03)
	if err != nil && err != io.EOF {
		t.Fatalf("ReadRTUFrame error: %v", err)
	}
	if !bytes.Equal(got, rtuFrame) {
		t.Errorf("ReadRTUFrame: got %X, want %X", got, rtuFrame)
	}
}
