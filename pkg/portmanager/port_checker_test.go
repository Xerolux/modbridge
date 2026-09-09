// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package portmanager

import (
	"os"
	"testing"
)

// ssOutput is a realistic `ss -tlnp` capture. Note the listeners on 5020 and
// 50200: a substring search for ":502" matches both.
const ssOutput = `State  Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
LISTEN 0      4096         0.0.0.0:5020       0.0.0.0:*         users:(("modbridge",pid=1234,fd=7))
LISTEN 0      4096         0.0.0.0:50200      0.0.0.0:*         users:(("other",pid=4321,fd=9))
LISTEN 0      511        127.0.0.1:8080       0.0.0.0:*         users:(("modbridge",pid=1234,fd=3))
LISTEN 0      4096            [::1]:9090          [::]:*         users:(("exporter",pid=777,fd=5))
`

func TestFindListenerMatchesExactPortOnly(t *testing.T) {
	if _, listening := findListener(ssOutput, 502); listening {
		t.Error("port 502 reported as busy, but only 5020 and 50200 are listening")
	}
	if _, listening := findListener(ssOutput, 20); listening {
		t.Error("port 20 reported as busy from a substring match")
	}
	if _, listening := findListener(ssOutput, 808); listening {
		t.Error("port 808 reported as busy, but 8080 is listening")
	}
}

func TestFindListenerFindsRealListeners(t *testing.T) {
	tests := []struct {
		port int
		pid  int
	}{
		{5020, 1234},
		{50200, 4321},
		{8080, 1234},
		{9090, 777},
	}
	for _, tc := range tests {
		pid, listening := findListener(ssOutput, tc.port)
		if !listening {
			t.Errorf("port %d not found, want a listener", tc.port)
			continue
		}
		if pid != tc.pid {
			t.Errorf("port %d: pid = %d, want %d", tc.port, pid, tc.pid)
		}
	}
}

func TestFindListenerWithoutProcessColumn(t *testing.T) {
	// netstat -an and `ss -tln` (no -p) carry no process column; the port must
	// still be reported busy, just without a PID.
	const netstatOutput = `Proto Recv-Q Send-Q Local Address    Foreign Address  State
tcp        0      0 0.0.0.0:5020     0.0.0.0:*        LISTEN
`
	pid, listening := findListener(netstatOutput, 5020)
	if !listening {
		t.Fatal("listener not found in netstat-style output")
	}
	if pid != 0 {
		t.Errorf("pid = %d, want 0 when the output carries no process column", pid)
	}
}

func TestFindListenerIgnoresNonListenLines(t *testing.T) {
	const established = `ESTAB 0 0 10.0.0.1:5020 10.0.0.2:41234 users:(("modbridge",pid=1234,fd=7))
`
	if _, listening := findListener(established, 5020); listening {
		t.Error("an established connection was reported as a listener")
	}
}

func TestFindListenerOnEmptyOutput(t *testing.T) {
	if _, listening := findListener("", 5020); listening {
		t.Error("empty output reported a listener")
	}
}

func TestKillProcessRefusesDangerousPIDs(t *testing.T) {
	pm := NewPortManager()

	// PID 0 signals the caller's own process group, PID 1 is init, and killing
	// ourselves is never what freeing a port means.
	for _, pid := range []int{0, 1, -1, os.Getpid()} {
		if err := pm.KillProcess(pid); err == nil {
			t.Errorf("KillProcess(%d) = nil, want it refused", pid)
		}
	}
}
