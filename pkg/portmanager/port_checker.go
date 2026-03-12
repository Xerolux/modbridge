package portmanager

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

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
	cmd := exec.Command("netstat", "-an")
	if runtime.GOOS == "windows" {
		cmd = exec.Command("netstat", "-an")
	} else {
		cmd = exec.Command("ss", "-tlnp")
	}

	output, _ := cmd.Output()
	portStr := fmt.Sprintf(":%d", port)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, portStr) && strings.Contains(line, "LISTEN") {
			// Try to extract PID
			fields := strings.Fields(line)
			pid := 0
			if len(fields) >= 7 {
				if pidStr := strings.TrimPrefix(fields[6], ""); pidStr != "" {
					if p, err := strconv.Atoi(pidStr); err == nil {
						pid = p
					}
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
	}

	return &PortInfo{
		State:  "FREE",
		IsOpen: false,
		Port:   port,
	}
}

// CheckPorts checks multiple ports
func (pm *PortManager) CheckPorts(ports []int) map[int]*PortInfo {
	results := make(map[int]*PortInfo)
	for _, port := range ports {
		results[port] = pm.CheckPort(port)
	}
	return results
}

// KillProcess kills a process by PID
func (pm *PortManager) KillProcess(pid int) error {
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

// parseNetstatOutput parses netstat output to find port usage
func parseNetstatOutput(output string, port int) *ProcessInfo {
	portStr := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, portStr) && strings.Contains(line, "LISTEN") {
			fields := strings.Fields(line)
			if len(fields) >= 7 {
				pidStr := strings.TrimPrefix(fields[6], "")
				if pid, err := strconv.Atoi(pidStr); err == nil {
					return getProcessInfo(pid)
				}
			}
		}
	}
	return nil
}
