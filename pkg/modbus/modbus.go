// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

// MBAPHeaderLength is the length of the Modbus Application Protocol header prefix (Transaction ID, Protocol ID, Length).
const MBAPHeaderLength = 6

var (
	headerPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, MBAPHeaderLength)
		},
	}
	payloadPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, 256)
			return &b
		},
	}
	framePool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, 260)
			return &b
		},
	}
)

// ReadFrame reads a full Modbus TCP frame from the reader.
func ReadFrame(r io.Reader) ([]byte, error) {
	// Get header buffer from pool
	header := headerPool.Get().([]byte)
	defer headerPool.Put(header)

	// Read the first 6 bytes to get the length
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	// Parse the length field (bytes 4 and 5)
	length := binary.BigEndian.Uint16(header[4:6])

	// The length field represents the number of bytes following.
	// We need to read 'length' bytes.
	// Sanity check for length (Modbus PDU max is usually 253 + 1 Unit ID = 254, but let's allow a bit more just in case, standard says 260 bytes max total frame)
	if length == 0 || length > 300 {
		return nil, fmt.Errorf("invalid modbus length: %d", length)
	}

	var payload []byte
	pp := payloadPool.Get().(*[]byte)
	if cap(*pp) >= int(length) {
		payload = (*pp)[:length]
	} else {
		payload = make([]byte, length)
	}

	if _, err := io.ReadFull(r, payload); err != nil {
		if cap(payload) <= 256 {
			*pp = payload[:0]
			payloadPool.Put(pp)
		}
		return nil, err
	}

	fp := framePool.Get().(*[]byte)
	totalLen := MBAPHeaderLength + int(length)
	if cap(*fp) < totalLen {
		*fp = make([]byte, 0, totalLen)
	}
	frame := (*fp)[:0]
	frame = append(frame, header...)
	frame = append(frame, payload...)

	if cap(payload) <= 256 {
		*pp = payload[:0]
		payloadPool.Put(pp)
	}

	return frame, nil
}

// ReleaseFrame returns a frame buffer to the pool.
func ReleaseFrame(frame []byte) {
	if cap(frame) >= MBAPHeaderLength {
		fp := framePool.Get().(*[]byte)
		*fp = frame[:0]
		framePool.Put(fp)
	}
}

func init() {
	_ = payloadPool.Get
	_ = framePool.Get
}
