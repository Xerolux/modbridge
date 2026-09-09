// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

//go:build linux || darwin

package pool

import (
	"net"
	"syscall"
)

// connLiveness reports whether conn still has a peer on the other end.
//
// It peeks at the receive queue (MSG_PEEK) without blocking (MSG_DONTWAIT), so
// nothing is consumed: a Modbus frame already waiting in the buffer stays where
// it is. A zero-length result means the peer has closed its side, an error
// other than "would block" means the socket is broken.
//
// The second return value is false when the connection cannot be inspected this
// way; the caller then has to assume it is usable.
func connLiveness(conn net.Conn) (alive bool, ok bool) {
	sc, isSyscallConn := conn.(syscall.Conn)
	if !isSyscallConn {
		return true, false
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return false, true
	}

	var buf [1]byte
	live := true
	ctrlErr := raw.Control(func(fd uintptr) {
		n, _, rerr := syscall.Recvfrom(int(fd), buf[:], syscall.MSG_PEEK|syscall.MSG_DONTWAIT)
		switch {
		case rerr == syscall.EAGAIN || rerr == syscall.EWOULDBLOCK:
			// Nothing pending and the socket is open.
		case rerr != nil:
			live = false
		case n == 0:
			live = false
		}
	})
	if ctrlErr != nil {
		return false, true
	}
	return live, true
}
