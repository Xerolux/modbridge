// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package modbus

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestReadFrame(t *testing.T) {
	// Construct a valid frame
	// TransID(2) + ProtoID(2) + Length(2) + UnitID(1) + PDU(n)
	transID := uint16(123)
	protoID := uint16(0)
	unitID := uint8(1)
	pdu := []byte{0x03, 0x00, 0x01, 0x00, 0x01} // Read Holding Registers
	length := uint16(1 + len(pdu))              // UnitID + PDU

	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, transID)
	_ = binary.Write(buf, binary.BigEndian, protoID)
	_ = binary.Write(buf, binary.BigEndian, length)
	_ = buf.WriteByte(unitID)
	_, _ = buf.Write(pdu)

	frame, err := ReadFrame(buf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	expectedLen := 6 + int(length)
	if len(frame) != expectedLen {
		t.Errorf("Expected frame length %d, got %d", expectedLen, len(frame))
	}

	if frame[6] != unitID {
		t.Errorf("Expected UnitID %d, got %d", unitID, frame[6])
	}
}

func TestReadFrameShort(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0x00, 0x01}) // Too short for header
	_, err := ReadFrame(buf)
	if err == nil {
		t.Fatal("Expected error for short buffer, got nil")
	}
}

// TestReadFrameInvalidLength verifies that a zero or oversized MBAP length
// field is rejected without reading further (DoS / malformed-frame guard).
func TestReadFrameInvalidLength(t *testing.T) {
	cases := []struct {
		name string
		hdr  []byte // 6-byte MBAP header
	}{
		{"zero length", []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00}},
		{"oversized length", []byte{0x00, 0x01, 0x00, 0x00, 0xFF, 0xFF}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadFrame(bytes.NewReader(tc.hdr))
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

// TestReadFrameMaxPayload verifies a frame at the protocol maximum length
// (254-byte payload) is accepted and returned intact.
func TestReadFrameMaxPayload(t *testing.T) {
	const payloadLen = 254 // max allowed MBAP length (UnitID + PDU)
	hdr := make([]byte, 6)
	binary.BigEndian.PutUint16(hdr[4:6], payloadLen)

	payload := make([]byte, payloadLen)
	for i := range payload {
		payload[i] = byte(i)
	}
	full := append(hdr, payload...)

	frame, err := ReadFrame(bytes.NewReader(full))
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}
	if len(frame) != 6+payloadLen {
		t.Fatalf("expected frame length %d, got %d", 6+payloadLen, len(frame))
	}
	if !bytes.Equal(frame[6:], payload) {
		t.Fatal("payload mismatch in returned frame")
	}
}

// TestReadFrameTruncatedPayload verifies that a short body read is reported as
// an error (io.ErrUnexpectedEOF / io.EOF) rather than returning partial data.
func TestReadFrameTruncatedPayload(t *testing.T) {
	hdr := make([]byte, 6)
	binary.BigEndian.PutUint16(hdr[4:6], 10) // claim 10 bytes follow
	hdr = append(hdr, 0x01, 0x02)            // only 2 bytes actually follow

	_, err := ReadFrame(bytes.NewReader(hdr))
	if err == nil {
		t.Fatal("expected error for truncated payload, got nil")
	}
}
