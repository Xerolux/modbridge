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

// MBAPHeaderLength is the length of the Modbus Application Protocol header prefix (Transaction ID, Protocol ID, Length).
const MBAPHeaderLength = 6

// MaxFrameLength is the largest Modbus TCP ADU we accept (6-byte MBAP header +
// max 254-byte PDU = 260). The MBAP length field covers UnitID + PDU, so a
// valid length is at most 254. Anything larger indicates a malformed or
// malicious frame and also guards against allocating oversized buffers.
const MaxFrameLength = 260

// maxPDULength is the largest payload (UnitID + PDU) accepted after the header.
const maxPDULength = MaxFrameLength - MBAPHeaderLength

// ReadFrame reads a single Modbus TCP frame (MBAP header + PDU) from r and
// returns a freshly allocated buffer containing the complete frame.
//
// The returned slice is owned by the caller; there is no pool to return it to.
// A previous implementation used three sync.Pools whose bookkeeping was broken
// (buffers were leaked on large frames and ReleaseFrame was never called), so
// the pooling added complexity and overhead without recycling anything. This
// single-allocation version is simpler, leak-free and avoids a per-request
// DoS vector since the length field is strictly validated.
func ReadFrame(r io.Reader) ([]byte, error) {
	var header [MBAPHeaderLength]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint16(header[4:6])
	if length < 2 || int(length) > maxPDULength {
		return nil, fmt.Errorf("invalid modbus length: %d", length)
	}

	// One allocation for the whole frame: header + payload.
	frame := make([]byte, MBAPHeaderLength+int(length))
	copy(frame, header[:])
	if _, err := io.ReadFull(r, frame[MBAPHeaderLength:]); err != nil {
		return nil, err
	}
	return frame, nil
}
