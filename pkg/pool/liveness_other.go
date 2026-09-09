// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

//go:build !linux && !darwin

package pool

import "net"

// connLiveness has no portable implementation on this platform: there is no way
// to inspect the socket without consuming from it, so the caller is told the
// answer is unknown and must treat the connection as usable.
func connLiveness(conn net.Conn) (alive bool, ok bool) {
	return true, false
}
