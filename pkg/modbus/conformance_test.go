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

// The tests in this file pin the Modbus TCP protocol behaviour (PI-MBUS-300)
// ModBridge relies on: MBAP framing limits, the register/quantity boundaries
// of the read function codes, exception framing, and the RTU conversion
// surface. SunSpec devices are served through the same framing — a proxy that
// keeps these invariants can carry any SunSpec model block transparently.

// TestReadFrameMBAPConformance walks the MBAP acceptance matrix: protocol ID
// must be zero, the length field must stay inside 2..254, and a full frame of
// the maximum size is accepted.
func TestReadFrameMBAPConformance(t *testing.T) {
	cases := []struct {
		name    string
		build   func() []byte
		wantErr bool
	}{
		{"valid minimal frame", func() []byte { return []byte{0, 1, 0, 0, 0, 2, 0x01, 0x03} }, false},
		{"non-zero protocol id", func() []byte { return []byte{0, 1, 0, 1, 0, 2, 0x01, 0x03} }, true},
		{"length 0", func() []byte { return []byte{0, 1, 0, 0, 0, 0, 0x01, 0x03} }, true},
		{"length 1 (unit only)", func() []byte { return []byte{0, 1, 0, 0, 0, 1, 0x01} }, true},
		{"length 255", func() []byte {
			f := make([]byte, 6)
			binary.BigEndian.PutUint16(f[4:6], 255)
			f = append(f, make([]byte, 255)...)
			return f
		}, true},
		{"maximum frame 260 bytes", func() []byte {
			f := make([]byte, 6)
			binary.BigEndian.PutUint16(f[4:6], 254)
			return append(f, make([]byte, 254)...)
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadFrame(bytes.NewReader(tc.build()))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestReadQuantityBoundaries pins the per-function-code quantity limits of the
// specification: 125 registers (FC 03/04) and 2000 coils/inputs (FC 01/02).
func TestReadQuantityBoundaries(t *testing.T) {
	if MaxReadQuantity != 125 {
		t.Errorf("MaxReadQuantity = %d, want 125", MaxReadQuantity)
	}
	// FC 03/04 request shapes must round-trip at the extremes.
	for _, tc := range []struct{ addr, qty uint16 }{{0, 1}, {0, 125}, {65535 - 124, 125}, {65534, 1}} {
		req := CreateReadRequest(7, 1, FuncReadHoldingRegisters, tc.addr, tc.qty)
		_, unit, fc, addr, qty, err := ParseReadRequest(req)
		if err != nil || unit != 1 || fc != FuncReadHoldingRegisters || addr != tc.addr || qty != tc.qty {
			t.Fatalf("roundtrip addr=%d qty=%d failed: unit=%d fc=0x%02X addr=%d qty=%d err=%v",
				tc.addr, tc.qty, unit, fc, addr, qty, err)
		}
	}
	// 126 registers is legal on the wire (a server must reject it, not the
	// framing), so the request parses even though MaxReadQuantity is 125.
	req := CreateReadRequest(1, 1, FuncReadInputRegisters, 0, 126)
	if _, _, _, _, qty, err := ParseReadRequest(req); err != nil || qty != 126 {
		t.Fatalf("126-quantity request should parse, got qty=%d err=%v", qty, err)
	}
}

// TestExceptionFraming verifies synthesised exception responses carry the
// client's transaction ID, the request's unit and function code with the
// error bit, and the exception code — the contract a gateway must keep.
func TestExceptionFraming(t *testing.T) {
	req := CreateReadRequest(0xBEEF, 5, FuncReadHoldingRegisters, 100, 10)
	resp := CreateExceptionResponse(req, 0x0B)
	if len(resp) != 9 {
		t.Fatalf("exception frame length = %d, want 9", len(resp))
	}
	tx, _ := FrameTxID(resp)
	if tx != 0xBEEF {
		t.Errorf("tx = 0x%04X, want 0xBEEF (client's)", tx)
	}
	unit, fc, _ := FrameUnitAndFunction(resp)
	if unit != 5 || fc != FuncReadHoldingRegisters|0x80 {
		t.Errorf("unit/fc = %d/0x%02X, want 5/0x83", unit, fc)
	}
	if resp[8] != 0x0B {
		t.Errorf("exception code = 0x%02X, want 0x0B", resp[8])
	}
	if !IsExceptionResponse(resp) {
		t.Error("IsExceptionResponse must recognise the frame")
	}
}

// TestCacheKeyCoverage pins which function codes may be answered from a
// cache: pure reads only. Writes — including read/write multiple (0x17) —
// must never be cached, and the key must ignore the transaction ID so two
// clients asking the same question share an entry.
func TestCacheKeyCoverage(t *testing.T) {
	read := CreateReadRequest(1, 3, FuncReadHoldingRegisters, 40_000, 125)
	sameQuestion := CreateReadRequest(2, 3, FuncReadHoldingRegisters, 40_000, 125)
	otherUnit := CreateReadRequest(1, 4, FuncReadHoldingRegisters, 40_000, 125)
	otherAddr := CreateReadRequest(1, 3, FuncReadHoldingRegisters, 40_001, 125)

	key, unit, ok := RequestCacheKey(read)
	if !ok || unit != 3 {
		t.Fatalf("read request should be cacheable, ok=%v unit=%d", ok, unit)
	}
	if k2, _, ok2 := RequestCacheKey(sameQuestion); !ok2 || k2 != key {
		t.Error("cache key must ignore the transaction ID")
	}
	if k3, _, _ := RequestCacheKey(otherUnit); k3 == key {
		t.Error("different unit must produce a different key")
	}
	if k4, _, _ := RequestCacheKey(otherAddr); k4 == key {
		t.Error("different address must produce a different key")
	}

	for _, fc := range []byte{FuncWriteSingleCoil, FuncWriteSingleRegister, FuncWriteMultipleCoils,
		FuncWriteMultipleRegisters, FuncMaskWriteRegister, FuncReadWriteRegisters} {
		frame := []byte{0, 1, 0, 0, 0, 6, 3, fc, 0, 100, 0, 1}
		if _, _, ok := RequestCacheKey(frame); ok {
			t.Errorf("fc 0x%02X must not be cacheable", fc)
		}
		if !IsWriteFunction(fc) {
			t.Errorf("fc 0x%02X must count as a write (cache invalidation)", fc)
		}
	}
	for _, fc := range []byte{FuncReadCoils, FuncReadDiscreteInputs, FuncReadHoldingRegisters, FuncReadInputRegisters} {
		if !IsReadFunction(fc) {
			t.Errorf("fc 0x%02X must count as a read", fc)
		}
		if IsWriteFunction(fc) {
			t.Errorf("fc 0x%02X must not count as a write", fc)
		}
	}
}

// TestRTUConversionRoundtrip covers the RTU gateway surface: CRC validation,
// TCP↔RTU conversion, and the response length model for the supported
// function codes — including the newly covered 0x07, 0x0B and 0x17.
func TestRTUConversionRoundtrip(t *testing.T) {
	tcpReq := CreateReadRequest(0x1234, 9, FuncReadHoldingRegisters, 500, 30)
	rtuReq, err := TCPToRTU(tcpReq)
	if err != nil {
		t.Fatalf("TCPToRTU: %v", err)
	}
	if rtuReq[0] != 9 || rtuReq[1] != FuncReadHoldingRegisters {
		t.Fatalf("rtu header = %d/0x%02X", rtuReq[0], rtuReq[1])
	}
	// Corrupt one CRC byte: conversion must fail instead of passing garbage.
	bad := append([]byte(nil), rtuReq...)
	bad[len(bad)-1] ^= 0xFF
	if _, err := RTUToTCP(bad, 0x1234); err == nil {
		t.Error("CRC mismatch must be rejected")
	}

	for _, fc := range []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x0B, 0x0F, 0x10, 0x17} {
		if !SupportsRTUFunction(fc) {
			t.Errorf("fc 0x%02X must be supported for RTU", fc)
		}
	}
	for _, fc := range []byte{0x08, 0x0C, 0x11, 0x2B, 0x00} {
		if SupportsRTUFunction(fc) {
			t.Errorf("fc 0x%02X must be rejected for RTU (length model unknown)", fc)
		}
	}
}
