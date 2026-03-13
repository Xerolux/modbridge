package mockmodbus

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestMockServer_StartStop(t *testing.T) {
	config := DefaultConfig()
	config.Port = 15021

	server := NewMockServer(config)

	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	if !server.running {
		t.Error("Server should be running")
	}

	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	if server.running {
		t.Error("Server should not be running")
	}
}

func TestMockServer_ReadHoldingRegisters(t *testing.T) {
	config := DefaultConfig()
	config.Port = 15022

	server := NewMockServer(config)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Connect to server
	conn, err := net.Dial("tcp", server.GetAddress())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send Read Holding Registers request (function code 0x03)
	// Transaction ID: 0x0001, Protocol ID: 0x0000, Length: 0x0006, Unit ID: 0x01
	// Function: 0x03, Address: 0x0000, Quantity: 0x0002
	request := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,       // Unit ID
		0x03,       // Function code
		0x00, 0x00, // Address
		0x00, 0x02, // Quantity
	}

	_, err = conn.Write(request)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response
	response := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if n < 9 {
		t.Fatalf("Response too short: %d bytes", n)
	}

	// Check response
	if response[0] != 0x00 || response[1] != 0x01 {
		t.Errorf("Wrong transaction ID")
	}

	if response[7] != 0x03 {
		t.Errorf("Expected function code 0x03, got 0x%02X", response[7])
	}

	// Check byte count (2 registers = 4 bytes)
	if response[8] != 0x04 {
		t.Errorf("Expected byte count 0x04, got 0x%02X", response[8])
	}
}

func TestMockServer_WriteSingleRegister(t *testing.T) {
	config := DefaultConfig()
	config.Port = 15023

	server := NewMockServer(config)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", server.GetAddress())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send Write Single Register request (function code 0x06)
	request := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,       // Unit ID
		0x06,       // Function code
		0x00, 0x01, // Address
		0x00, 0x03, // Value
	}

	_, err = conn.Write(request)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	response := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if n < 12 {
		t.Fatalf("Response too short: %d bytes", n)
	}

	// Echo back address and value
	if !bytes.Equal(response[8:10], []byte{0x00, 0x01}) {
		t.Errorf("Wrong address in response")
	}

	if !bytes.Equal(response[10:12], []byte{0x00, 0x03}) {
		t.Errorf("Wrong value in response")
	}
}

func TestMockServer_ErrorHandling(t *testing.T) {
	config := DefaultConfig()
	config.Port = 15024

	server := NewMockServer(config)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", server.GetAddress())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send invalid function code (0xFF)
	request := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x02, // Length
		0x01, // Unit ID
		0xFF, // Invalid function code
	}

	_, err = conn.Write(request)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	response := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if n < 9 {
		t.Fatalf("Response too short: %d bytes", n)
	}

	// Check for error bit in function code
	if response[7]&0x80 == 0 {
		t.Error("Expected error bit to be set in function code")
	}
}

func TestMockServer_RequestLogging(t *testing.T) {
	config := DefaultConfig()
	config.Port = 15025
	config.LogRequests = true

	server := NewMockServer(config)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", server.GetAddress())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Send a request
	request := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,       // Unit ID
		0x03,       // Function code
		0x00, 0x00, // Address
		0x00, 0x01, // Quantity
	}

	_, err = conn.Write(request)
	if err != nil {
		conn.Close()
		t.Fatalf("Failed to send request: %v", err)
	}

	// Wait for request to be processed
	time.Sleep(50 * time.Millisecond)
	conn.Close()

	// Check request log
	log := server.GetRequestLog()
	if len(log) == 0 {
		t.Error("Expected at least one request in log")
	}

	if log[0].FunctionCode != 0x03 {
		t.Errorf("Expected function code 0x03, got 0x%02X", log[0].FunctionCode)
	}
}

func TestMockServer_ConnectionTracking(t *testing.T) {
	config := DefaultConfig()
	config.Port = 15026

	server := NewMockServer(config)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	if server.GetConnectionCount() != 0 {
		t.Error("Expected no connections initially")
	}

	conn, err := net.Dial("tcp", server.GetAddress())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait a bit for connection to be registered
	time.Sleep(50 * time.Millisecond)

	if server.GetConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", server.GetConnectionCount())
	}

	conn.Close()

	// Wait for connection to be removed
	time.Sleep(50 * time.Millisecond)

	if server.GetConnectionCount() != 0 {
		t.Error("Expected no connections after close")
	}
}

func TestMockServer_ResponseDelay(t *testing.T) {
	config := DefaultConfig()
	config.Port = 15027
	config.Delay = 100 * time.Millisecond

	server := NewMockServer(config)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", server.GetAddress())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	request := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,       // Unit ID
		0x03,       // Function code
		0x00, 0x00, // Address
		0x00, 0x01, // Quantity
	}

	start := time.Now()
	_, err = conn.Write(request)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	response := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("Expected delay of at least 100ms, got %v", elapsed)
	}
}

func TestMockServer_ErrorRate(t *testing.T) {
	config := DefaultConfig()
	config.Port = 15028
	config.ErrorRate = 1.0 // Always return error

	server := NewMockServer(config)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", server.GetAddress())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	request := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,       // Unit ID
		0x03,       // Function code
		0x00, 0x00, // Address
		0x00, 0x01, // Quantity
	}

	_, err = conn.Write(request)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	response := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if n < 9 {
		t.Fatalf("Response too short: %d bytes", n)
	}

	// Check for error response
	if response[7]&0x80 == 0 {
		t.Error("Expected error bit to be set with 100% error rate")
	}
}

func TestMockServer_WaitForConnection(t *testing.T) {
	config := DefaultConfig()
	config.Port = 15029

	server := NewMockServer(config)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Test timeout
	err = server.WaitForConnection(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout waiting for connection")
	}

	// Connect in background
	go func() {
		time.Sleep(50 * time.Millisecond)
		net.Dial("tcp", server.GetAddress())
	}()

	// Should succeed now
	err = server.WaitForConnection(200 * time.Millisecond)
	if err != nil {
		t.Errorf("Expected connection, got error: %v", err)
	}
}
