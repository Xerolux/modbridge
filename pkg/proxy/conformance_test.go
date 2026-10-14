// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package proxy

import (
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"modbridge/pkg/logger"
	"modbridge/pkg/modbus"
)

// scriptedDevice is a minimal Modbus TCP server answering from a 65536-word
// register map. It supports the function codes SunSpec and ordinary clients
// use (01-06, 0x0F, 0x10, 0x16, 0x17), can be told to answer one exception,
// and records every request it saw. Registers default to the value of their
// address so responses are verifiable end to end.
type scriptedDevice struct {
	listener net.Listener

	mu        sync.Mutex
	registers [65536]uint16
	coils     [65536]bool
	exception byte // 0 = answer normally, otherwise answer this exception code
	requests  []byte
}

func startScriptedDevice(t *testing.T) *scriptedDevice {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	d := &scriptedDevice{listener: l}
	for i := range d.registers {
		d.registers[i] = uint16(i)
	}
	t.Cleanup(func() { l.Close() })
	go d.serve()
	return d
}

func (d *scriptedDevice) addr() string { return d.listener.Addr().String() }

func (d *scriptedDevice) serve() {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			for {
				req, err := modbus.ReadFrame(c)
				if err != nil {
					return
				}
				d.mu.Lock()
				d.requests = append(d.requests, req[7])
				resp := d.answer(req)
				d.mu.Unlock()
				if _, err := c.Write(resp); err != nil {
					return
				}
			}
		}(conn)
	}
}

func (d *scriptedDevice) answer(req []byte) []byte {
	tx := binary.BigEndian.Uint16(req[0:2])
	unit, fc := req[6], req[7]

	if d.exception != 0 {
		return modbus.ExceptionResponse(tx, unit, fc, d.exception)
	}

	switch fc {
	case modbus.FuncReadHoldingRegisters, modbus.FuncReadInputRegisters:
		addr := binary.BigEndian.Uint16(req[8:10])
		qty := binary.BigEndian.Uint16(req[10:12])
		data := make([]byte, qty*2)
		for i := 0; i < int(qty); i++ {
			binary.BigEndian.PutUint16(data[i*2:], d.registers[int(addr)+i])
		}
		resp, _ := modbus.CreateReadResponse(tx, unit, fc, data)
		return resp
	case modbus.FuncReadCoils, modbus.FuncReadDiscreteInputs:
		addr := binary.BigEndian.Uint16(req[8:10])
		qty := binary.BigEndian.Uint16(req[10:12])
		bytes := (qty + 7) / 8
		data := make([]byte, bytes)
		for i := 0; i < int(qty); i++ {
			if d.coils[int(addr)+i] {
				data[i/8] |= 1 << (i % 8)
			}
		}
		resp, _ := modbus.CreateReadResponse(tx, unit, fc, data)
		return resp
	case modbus.FuncWriteSingleRegister:
		addr := binary.BigEndian.Uint16(req[8:10])
		val := binary.BigEndian.Uint16(req[10:12])
		d.registers[addr] = val
		out := make([]byte, 12)
		binary.BigEndian.PutUint16(out[0:2], tx)
		binary.BigEndian.PutUint16(out[4:6], 6)
		out[6], out[7] = unit, fc
		binary.BigEndian.PutUint16(out[8:10], addr)
		binary.BigEndian.PutUint16(out[10:12], val)
		return out
	case modbus.FuncWriteMultipleRegisters:
		addr := binary.BigEndian.Uint16(req[8:10])
		qty := binary.BigEndian.Uint16(req[10:12])
		for i := 0; i < int(qty); i++ {
			d.registers[int(addr)+i] = binary.BigEndian.Uint16(req[13+i*2 : 15+i*2])
		}
		out := make([]byte, 12)
		binary.BigEndian.PutUint16(out[0:2], tx)
		binary.BigEndian.PutUint16(out[4:6], 6)
		out[6], out[7] = unit, fc
		binary.BigEndian.PutUint16(out[8:10], addr)
		binary.BigEndian.PutUint16(out[10:12], qty)
		return out
	case modbus.FuncMaskWriteRegister, modbus.FuncWriteSingleCoil, modbus.FuncWriteMultipleCoils:
		// Echo the request as its own response (spec shape for 0x16; good
		// enough for the passthrough assertions made here).
		out := append([]byte(nil), req...)
		return out
	case modbus.FuncReadWriteRegisters:
		readAddr := binary.BigEndian.Uint16(req[8:10])
		readQty := binary.BigEndian.Uint16(req[10:12])
		writeAddr := binary.BigEndian.Uint16(req[12:14])
		writeQty := binary.BigEndian.Uint16(req[14:16])
		byteCount := int(req[16])
		for i := 0; i < int(writeQty); i++ {
			d.registers[int(writeAddr)+i] = binary.BigEndian.Uint16(req[17+i*2 : 19+i*2])
		}
		_ = byteCount
		data := make([]byte, readQty*2)
		for i := 0; i < int(readQty); i++ {
			binary.BigEndian.PutUint16(data[i*2:], d.registers[int(readAddr)+i])
		}
		resp, _ := modbus.CreateReadResponse(tx, unit, fc, data)
		return resp
	}
	return modbus.ExceptionResponse(tx, unit, fc, 0x01)
}

func (d *scriptedDevice) setRegister(addr uint16, val uint16) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.registers[addr] = val
}

func (d *scriptedDevice) seenFunctionCodes() []byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]byte(nil), d.requests...)
}

// conformanceProxy starts a ProxyInstance in front of dev with the given
// overrides applied.
func conformanceProxy(t *testing.T, dev *scriptedDevice, tune func(p *ProxyInstance)) (*ProxyInstance, string) {
	t.Helper()
	log, err := logger.NewLogger(t.TempDir(), 100)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	t.Cleanup(func() { log.Close() })

	pl, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("proxy listen: %v", err)
	}
	proxyAddr := pl.Addr().String()
	pl.Close()

	p := NewProxyInstance("conf", "conformance", proxyAddr, dev.addr(), 0, 2, 2, 0, log, nil)
	if tune != nil {
		tune(p)
	}
	if err := p.Start(); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(p.Stop)
	return p, proxyAddr
}

// exchange sends one request through the proxy and returns the response.
func exchange(t *testing.T, addr string, frame []byte) []byte {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	if _, err := conn.Write(frame); err != nil {
		t.Fatalf("write: %v", err)
	}
	resp, err := modbus.ReadFrame(conn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp
}

func readRegs(t *testing.T, addr string, tx uint16, unit byte, start uint16, qty uint16) []byte {
	req := modbus.CreateReadRequest(tx, unit, modbus.FuncReadHoldingRegisters, start, qty)
	resp := exchange(t, addr, req)
	gotTx, _ := modbus.FrameTxID(resp)
	if gotTx != tx {
		t.Fatalf("tx echo: got 0x%04X want 0x%04X", gotTx, tx)
	}
	return resp
}

// TestProxySunSpecWalkAndSplitReads simulates a SunSpec client: it reads the
// identity block, a full 125-register model block and an oversized block that
// must be reassembled from split reads — all through the proxy, with the
// client's transaction IDs echoed and the data intact.
func TestProxySunSpecWalkAndSplitReads(t *testing.T) {
	dev := startScriptedDevice(t)
	dev.setRegister(40000, 0x5375) // "Su"
	dev.setRegister(40001, 0x6E53) // "nS"
	dev.setRegister(40002, 1)      // model 1
	dev.setRegister(40121, 701)    // vendor model
	dev.setRegister(40499, 0x1234)

	p, addr := conformanceProxy(t, dev, func(p *ProxyInstance) { p.MaxReadSize = 60 })
	_ = p

	resp := readRegs(t, addr, 0x0001, 1, 40000, 2)
	data, err := modbus.ParseReadResponse(resp)
	if err != nil {
		t.Fatalf("identity read: %v", err)
	}
	if data[0] != 0x53 || data[1] != 0x75 || data[2] != 0x6E || data[3] != 0x53 {
		t.Fatalf("SunS magic not carried: % X", data)
	}

	// Full 125-register block in one go.
	resp = readRegs(t, addr, 0x0002, 1, 40000, 125)
	if data, err = modbus.ParseReadResponse(resp); err != nil || len(data) != 250 {
		t.Fatalf("125-register block: len=%d err=%v", len(data), err)
	}

	// 120 registers with MaxReadSize 50: reassembled transparently from
	// 50+50+20 chunk reads (a single response may carry at most 125).
	dev.setRegister(40119, 0xABCD)
	resp = readRegs(t, addr, 0x0003, 1, 40000, 120)
	data, err = modbus.ParseReadResponse(resp)
	if err != nil || len(data) != 240 {
		t.Fatalf("120-register split read: len=%d err=%v", len(data), err)
	}
	if got := binary.BigEndian.Uint16(data[119*2:]); got != 0xABCD {
		t.Fatalf("last register of split read = 0x%04X, want 0xABCD", got)
	}

	// 126 registers exceeds the protocol limit for one response: the proxy
	// must answer exception 0x03 (illegal data value), not a gateway error.
	resp = readRegs(t, addr, 0x0004, 1, 40000, 126)
	if !modbus.IsExceptionResponse(resp) || resp[8] != modbus.ExceptionIllegalDataValue {
		t.Fatalf("126-register read must yield exception 0x03, got % X", resp)
	}
}

// TestProxyPassthroughFunctionCodes verifies that every function code outside
// the split machinery is forwarded transparently — the property that makes
// the proxy compatible with SunSpec vendor models and mixed-client plants.
func TestProxyPassthroughFunctionCodes(t *testing.T) {
	dev := startScriptedDevice(t)
	_, addr := conformanceProxy(t, dev, nil)

	frame := func(fc byte, extra []byte) []byte {
		out := make([]byte, 8+len(extra))
		binary.BigEndian.PutUint16(out[0:2], 0x4242)
		binary.BigEndian.PutUint16(out[4:6], uint16(2+len(extra)))
		out[6], out[7] = 1, fc
		copy(out[8:], extra)
		return out
	}

	// FC05 write single coil
	if resp := exchange(t, addr, frame(0x05, []byte{0, 100, 0xFF, 0x00})); modbus.IsExceptionResponse(resp) {
		t.Fatalf("FC05 passthrough failed: % X", resp)
	}
	// FC06 write single register
	if resp := exchange(t, addr, frame(0x06, []byte{1, 0, 0x30, 0x39})); modbus.IsExceptionResponse(resp) {
		t.Fatalf("FC06 passthrough failed: % X", resp)
	}
	// FC16 mask write
	if resp := exchange(t, addr, frame(0x16, []byte{0, 10, 0xFF, 0xFF, 0, 0})); modbus.IsExceptionResponse(resp) {
		t.Fatalf("FC16 passthrough failed: % X", resp)
	}
	// FC17 read/write multiple registers: write 42 to 5000, read it back
	rw := []byte{0x13, 0x88, 0, 1, 0x13, 0x88, 0, 1, 2, 0x00, 0x2A}
	resp := exchange(t, addr, frame(0x17, rw))
	if modbus.IsExceptionResponse(resp) {
		t.Fatalf("FC17 passthrough failed: % X", resp)
	}
	if data, err := modbus.ParseReadResponse(resp); err != nil || binary.BigEndian.Uint16(data) != 42 {
		t.Fatalf("FC17 readback = % X err=%v, want 42", data, err)
	}
	// Coil read of 2000 coils (legal maximum for FC01) must pass through.
	coilReq := modbus.CreateReadRequest(9, 1, modbus.FuncReadCoils, 0, 2000)
	if resp := exchange(t, addr, coilReq); modbus.IsExceptionResponse(resp) {
		t.Fatalf("2000-coil read must pass through, got % X", resp)
	}

	seen := dev.seenFunctionCodes()
	for _, want := range []byte{0x05, 0x06, 0x16, 0x17, modbus.FuncReadCoils} {
		found := false
		for _, got := range seen {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("device never saw function code 0x%02X", want)
		}
	}
}

// TestProxyDeviceExceptionPassthrough pins that a device-originated exception
// (e.g. SunSpec end-of-map reads answered with 0x02) is forwarded unchanged
// apart from the transaction ID, and never translated into a gateway error.
func TestProxyDeviceExceptionPassthrough(t *testing.T) {
	dev := startScriptedDevice(t)
	dev.mu.Lock()
	dev.exception = modbus.ExceptionIllegalDataAddress
	dev.mu.Unlock()
	_, addr := conformanceProxy(t, dev, nil)

	req := modbus.CreateReadRequest(0x0777, 3, modbus.FuncReadHoldingRegisters, 40000, 10)
	resp := exchange(t, addr, req)
	if !modbus.IsExceptionResponse(resp) {
		t.Fatalf("expected exception frame, got % X", resp)
	}
	if resp[8] != modbus.ExceptionIllegalDataAddress {
		t.Fatalf("exception code = 0x%02X, want 0x02 (device's own)", resp[8])
	}
	if tx, _ := modbus.FrameTxID(resp); tx != 0x0777 {
		t.Fatalf("tx echo broken on device exception: 0x%04X", tx)
	}
}

// TestProxyGatewayExceptionFraming verifies the gateway's own failure answer:
// an unreachable target yields exception 0x0B with the client's transaction
// ID, unit and function code.
func TestProxyGatewayExceptionFraming(t *testing.T) {
	dead := startScriptedDevice(t)
	deadAddr := dead.addr()
	dead.listener.Close() // target now refuses connections

	log, err := logger.NewLogger(t.TempDir(), 100)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	t.Cleanup(func() { log.Close() })
	pl, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	proxyAddr := pl.Addr().String()
	pl.Close()

	p := NewProxyInstance("dead", "dead", proxyAddr, deadAddr, 0, 1, 1, 0, log, nil)
	p.RequestTimeout = 1500 * time.Millisecond
	if err := p.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(p.Stop)

	req := modbus.CreateReadRequest(0x0BAD, 7, modbus.FuncReadInputRegisters, 100, 5)
	resp := exchange(t, proxyAddr, req)
	if !modbus.IsExceptionResponse(resp) || resp[8] != 0x0B {
		t.Fatalf("gateway failure must answer 0x0B, got % X", resp)
	}
	if tx, _ := modbus.FrameTxID(resp); tx != 0x0BAD {
		t.Fatalf("tx echo broken on gateway exception: 0x%04X", tx)
	}
	if _, fc := resp[6], resp[7]; fc != modbus.FuncReadInputRegisters|0x80 {
		t.Fatalf("fc = 0x%02X, want 0x84", fc)
	}
}

// TestProxyPipelinedRequestsKeepsOrder verifies back-to-back requests on one
// connection — Modbus TCP allows multiple outstanding transactions — are
// answered in order with the right transaction IDs.
func TestProxyPipelinedRequestsKeepsOrder(t *testing.T) {
	dev := startScriptedDevice(t)
	_, addr := conformanceProxy(t, dev, nil)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))

	for _, tx := range []uint16{0x0101, 0x0202, 0x0303} {
		if _, err := conn.Write(modbus.CreateReadRequest(tx, 1, modbus.FuncReadHoldingRegisters, 40000, 5)); err != nil {
			t.Fatalf("write %04X: %v", tx, err)
		}
	}
	for _, want := range []uint16{0x0101, 0x0202, 0x0303} {
		resp, err := modbus.ReadFrame(conn)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if got, _ := modbus.FrameTxID(resp); got != want {
			t.Fatalf("out-of-order or wrong tx: got 0x%04X want 0x%04X", got, want)
		}
	}
}

// TestProxyWriteInvalidatesCacheAcrossWriteCodes proves the cache invariant
// SunSpec-heavy clients depend on: after ANY write function code touching a
// unit — including read/write multiple 0x17 — the next read returns the
// post-write value, never a cached pre-write one.
func TestProxyWriteInvalidatesCacheAcrossWriteCodes(t *testing.T) {
	dev := startScriptedDevice(t)
	_, addr := conformanceProxy(t, dev, func(p *ProxyInstance) {
		p.CacheEnabled = true
		p.CacheTTL = time.Minute // long enough that only invalidation can help
	})

	read := func() uint16 {
		resp := readRegs(t, addr, 0x00A1, 1, 5000, 1)
		data, err := modbus.ParseReadResponse(resp)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		return binary.BigEndian.Uint16(data)
	}
	write := func(fc byte) {
		frame := make([]byte, 8)
		binary.BigEndian.PutUint16(frame[4:6], 6)
		frame[6], frame[7] = 1, fc
		switch fc {
		case modbus.FuncWriteSingleRegister:
			frame = append(frame, byte(5000>>8), byte(5000&0xFF), 0x00, 0x63)
		case modbus.FuncWriteMultipleRegisters:
			// unit+fc+addr(2)+qty(2)+byteCount(1)+data(2) = 9 bytes after the header
			frame = append(frame, byte(5000>>8), byte(5000&0xFF), 0, 1, 2, 0x00, 0x64)
			binary.BigEndian.PutUint16(frame[4:6], 9)
		case modbus.FuncReadWriteRegisters:
			// unit+fc+readAddr(2)+readQty(2)+writeAddr(2)+writeQty(2)+byteCount(1)+data(2) = 13
			frame = append(frame, byte(5000>>8), byte(5000&0xFF), 0, 1, byte(5000>>8), byte(5000&0xFF), 0, 1, 2, 0x00, 0x65)
			binary.BigEndian.PutUint16(frame[4:6], 13)
		}
		if resp := exchange(t, addr, frame); modbus.IsExceptionResponse(resp) {
			t.Fatalf("write fc=0x%02X failed: % X", fc, resp)
		}
	}

	if got := read(); got != 5000 {
		t.Fatalf("initial register value = %d, want 5000 (address default)", got)
	}

	write(modbus.FuncWriteSingleRegister) // 99
	if got := read(); got != 99 {
		t.Fatalf("after FC06 write, read = %d, want 99 (cache not invalidated)", got)
	}
	write(modbus.FuncWriteMultipleRegisters) // 100
	if got := read(); got != 100 {
		t.Fatalf("after FC10 write, read = %d, want 100 (cache not invalidated)", got)
	}
	write(modbus.FuncReadWriteRegisters) // 101
	if got := read(); got != 101 {
		t.Fatalf("after FC17 write, read = %d, want 101 (cache not invalidated)", got)
	}
}
