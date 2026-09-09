// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package pool

import (
	"net"
	"runtime"
	"testing"
	"time"
)

// pipePair returns both ends of a real TCP connection, so the liveness check
// sees an actual socket rather than a net.Pipe that has no file descriptor.
func pipePair(t *testing.T) (client, server net.Conn) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	type accepted struct {
		conn net.Conn
		err  error
	}
	done := make(chan accepted, 1)
	go func() {
		c, err := ln.Accept()
		done <- accepted{c, err}
	}()

	client, err = net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	a := <-done
	if a.err != nil {
		client.Close()
		t.Fatalf("Accept: %v", a.err)
	}
	t.Cleanup(func() {
		client.Close()
		a.conn.Close()
	})
	return client, a.conn
}

func TestConnHealthyOnOpenConnection(t *testing.T) {
	client, _ := pipePair(t)

	if !ConnHealthy(client) {
		t.Error("ConnHealthy = false on an open connection, want true")
	}
}

func TestConnHealthyOnClosedPeer(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("no socket inspection on this platform")
	}

	client, server := pipePair(t)
	server.Close()

	// The FIN has to arrive before the check can see it.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !ConnHealthy(client) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("ConnHealthy stayed true after the peer closed the connection")
}

func TestConnHealthyDoesNotConsumePendingData(t *testing.T) {
	client, server := pipePair(t)

	want := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0x00, 0x00}
	if _, err := server.Write(want); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Let the data arrive, then check twice: peeking must leave the frame in
	// the receive buffer.
	time.Sleep(50 * time.Millisecond)
	if !ConnHealthy(client) {
		t.Fatal("ConnHealthy = false while data was pending")
	}
	if !ConnHealthy(client) {
		t.Fatal("ConnHealthy = false on the second check")
	}

	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	got := make([]byte, len(want))
	if _, err := readFull(client, got); err != nil {
		t.Fatalf("the pending frame was consumed by the liveness check: %v", err)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("frame changed: got %v, want %v", got, want)
		}
	}
}

func TestConnHealthyOnNil(t *testing.T) {
	if ConnHealthy(nil) {
		t.Error("ConnHealthy(nil) = true, want false")
	}
}

func readFull(c net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := c.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
