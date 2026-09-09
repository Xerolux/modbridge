// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package portmanager

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// pidPattern matches the PID inside the process column of `ss -tlnp`, which
// looks like `users:(("modbridge",pid=1234,fd=7))` — not a bare number, so a
// plain Atoi on that field never yields anything but 0.
var pidPattern = regexp.MustCompile(`pid=(\d+)`)

// ProcessInfo contains information about a process
type ProcessInfo struct {
	PID     int
	Process string
	User    string
	Command string
}

// PortInfo contains port information
type PortInfo struct {
	State      string `json:"state"`
	IsOpen     bool   `json:"is_open"`
	Port       int    `json:"port"`
	ProcessPID int    `json:"process_pid,omitempty"`
	Process    string `json:"process,omitempty"`
	User       string `json:"user,omitempty"`
}

// PortManager manages port operations
type PortManager struct{}

// NewPortManager creates a new port manager
func NewPortManager() *PortManager {
	return &PortManager{}
}

// CheckPort checks if a port is in use
func (pm *PortManager) CheckPort(port int) *PortInfo {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("netstat", "-an")
	} else {
		cmd = exec.Command("ss", "-tlnp")
	}

	output, _ := cmd.Output()

	pid, listening := findListener(string(output), port)
	if !listening {
		return &PortInfo{
			State:  "FREE",
			IsOpen: false,
			Port:   port,
		}
	}

	info := getProcessInfo(pid)
	return &PortInfo{
		State:      "LISTEN",
		IsOpen:     true,
		Port:       port,
		ProcessPID: pid,
		Process:    info.Process,
		User:       info.User,
	}
}

// findListener scans `ss -tlnp` or `netstat -an` output for a listener on port
// and returns its PID, if the output carries one.
//
// The port has to terminate the address: a plain substring search for ":502"
// also matches the listeners on :5020 and :50200, and would report every one of
// them as occupying port 502.
func findListener(output string, port int) (pid int, listening bool) {
	portPattern := regexp.MustCompile(fmt.Sprintf(`:%d(\s|$)`, port))

	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "LISTEN") || !portPattern.MatchString(line) {
			continue
		}
		if m := pidPattern.FindStringSubmatch(line); m != nil {
			if p, err := strconv.Atoi(m[1]); err == nil {
				pid = p
			}
		}
		return pid, true
	}
	return 0, false
}

// CheckPorts checks multiple ports
func (pm *PortManager) CheckPorts(ports []int) map[int]*PortInfo {
	results := make(map[int]*PortInfo)
	for _, port := range ports {
		results[port] = pm.CheckPort(port)
	}
	return results
}

// KillProcess kills a process by PID.
//
// PIDs 0 and 1 are refused: 0 means the port scan found no PID at all and would
// signal the caller's own process group, 1 is init. So is our own PID — killing
// it is never what the operator meant by freeing a port.
func (pm *PortManager) KillProcess(pid int) error {
	if pid <= 1 {
		return fmt.Errorf("refusing to kill invalid pid %d", pid)
	}
	if pid == os.Getpid() {
		return fmt.Errorf("refusing to kill own process (pid %d)", pid)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
	default:
		cmd = exec.Command("kill", "-9", strconv.Itoa(pid))
	}

	return cmd.Run()
}

// getProcessInfo gets process information using platform-specific commands
func getProcessInfo(pid int) *ProcessInfo {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	default:
		// Linux/Unix use ps
		cmd = exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "user,comm,args")
	}

	output, err := cmd.Output()
	if err != nil {
		return &ProcessInfo{
			PID:     pid,
			Process: "unknown",
			User:    "unknown",
			Command: "",
		}
	}

	return parseProcessOutput(pid, string(output))
}

// parseProcessOutput parses process command output
func parseProcessOutput(pid int, output string) *ProcessInfo {
	if runtime.GOOS == "windows" {
		return parseWindowsProcessOutput(pid, output)
	}
	return parseUnixProcessOutput(pid, output)
}

// parseWindowsProcessOutput parses Windows tasklist output
func parseWindowsProcessOutput(pid int, output string) *ProcessInfo {
	fields := strings.Split(output, ",")
	if len(fields) >= 2 {
		// Remove quotes from CSV output
		processName := strings.Trim(fields[0], "\"")
		return &ProcessInfo{
			PID:     pid,
			Process: processName,
			User:    "SYSTEM", // Windows doesn't easily show user
			Command: processName,
		}
	}

	return &ProcessInfo{
		PID:     pid,
		Process: "unknown",
		User:    "unknown",
		Command: "",
	}
}

// parseUnixProcessOutput parses Unix ps output
func parseUnixProcessOutput(pid int, output string) *ProcessInfo {
	fields := strings.Fields(output)
	if len(fields) >= 3 {
		user := fields[0]
		comm := fields[1]
		args := strings.Join(fields[2:], " ")

		return &ProcessInfo{
			PID:     pid,
			Process: comm,
			User:    user,
			Command: args,
		}
	}

	return &ProcessInfo{
		PID:     pid,
		Process: "unknown",
		User:    "unknown",
		Command: "",
	}
}
