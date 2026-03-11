package portmanager

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// PortInfo contains information about a port and its usage
type PortInfo struct {
	Port      int       `json:"port"`
	IsOpen    bool      `json:"is_open"`
	Process   string    `json:"process,omitempty"`
	PID       int       `json:"pid,omitempty"`
	User      string    `json:"user,omitempty"`
	Command   string    `json:"command,omitempty"`
	State     string    `json:"state"` // "free", "listening", "in_use"
	Timestamp time.Time `json:"timestamp"`
}

// PortManager handles port checking and freeing operations
type PortManager struct {
	checkedPorts map[int]*PortInfo
}

// NewPortManager creates a new port manager
func NewPortManager() *PortManager {
	return &PortManager{
		checkedPorts: make(map[int]*PortInfo),
	}
}

// CheckPort checks if a port is available
func (pm *PortManager) CheckPort(port int) *PortInfo {
	info := &PortInfo{
		Port:      port,
		IsOpen:    true,
		State:     "free",
		Timestamp: time.Now(),
	}

	// Try to listen on the port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		info.IsOpen = false
		info.State = "in_use"

		// Get process info
		if procInfo := getProcessUsingPort(port); procInfo != nil {
			info.Process = procInfo.Process
			info.PID = procInfo.PID
			info.User = procInfo.User
			info.Command = procInfo.Command
		}

		pm.checkedPorts[port] = info
		return info
	}

	listener.Close()
	pm.checkedPorts[port] = info
	return info
}

// CheckPorts checks multiple ports
func (pm *PortManager) CheckPorts(ports []int) map[int]*PortInfo {
	results := make(map[int]*PortInfo)
	for _, port := range ports {
		results[port] = pm.CheckPort(port)
	}
	return results
}

// KillProcess terminates a process using a port
func (pm *PortManager) KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		// Try SIGKILL if SIGTERM fails
		return process.Signal(syscall.SIGKILL)
	}

	return nil
}

// ReleasePort forcefully releases a port
func (pm *PortManager) ReleasePort(port int) error {
	portInfo := pm.CheckPort(port)
	if portInfo.IsOpen {
		return fmt.Errorf("port %d is already free", port)
	}

	if portInfo.PID == 0 {
		return fmt.Errorf("could not determine process using port %d", port)
	}

	return pm.KillProcess(portInfo.PID)
}

// ProcessInfo holds process information
type ProcessInfo struct {
	PID     int
	Process string
	User    string
	Command string
}

// getProcessUsingPort gets process information for a port
func getProcessUsingPort(port int) *ProcessInfo {
	// Try lsof first (Linux/macOS)
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN", "-t")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		pid, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err == nil {
			return getProcessInfo(pid)
		}
	}

	// Try netstat as fallback
	cmd = exec.Command("netstat", "-tlnp")
	output, err = cmd.Output()
	if err == nil {
		return parseNetstatOutput(string(output), port)
	}

	// Try ss (newer systems)
	cmd = exec.Command("ss", "-tlnp")
	output, err = cmd.Output()
	if err == nil {
		return parseSsOutput(string(output), port)
	}

	return nil
}

// getProcessInfo gets detailed process information
func getProcessInfo(pid int) *ProcessInfo {
	procPath := fmt.Sprintf("/proc/%d", pid)

	// Get process name from /proc/[pid]/comm
	commFile := fmt.Sprintf("%s/comm", procPath)
	nameBytes, err := os.ReadFile(commFile)
	var name string
	if err == nil {
		name = strings.TrimSpace(string(nameBytes))
	}

	// Get user from stat file
	statFile := fmt.Sprintf("%s/stat", procPath)
	stat, err := os.Stat(statFile)
	var user string
	if err == nil {
		sysstat := stat.Sys().(*syscall.Stat_t)
		user = fmt.Sprintf("%d", sysstat.Uid)
	}

	// Get command line from /proc/[pid]/cmdline
	cmdlineFile := fmt.Sprintf("%s/cmdline", procPath)
	cmdBytes, err := os.ReadFile(cmdlineFile)
	var cmdline string
	if err == nil {
		cmdline = strings.ReplaceAll(string(cmdBytes), "\x00", " ")
		cmdline = strings.TrimSpace(cmdline)
	}

	return &ProcessInfo{
		PID:     pid,
		Process: name,
		User:    user,
		Command: cmdline,
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

// parseSsOutput parses ss output to find port usage
func parseSsOutput(output string, port int) *ProcessInfo {
	portStr := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, portStr) && strings.Contains(line, "LISTEN") {
			// ss output format is different, extract PID from the line
			if strings.Contains(line, "pid=") {
				start := strings.Index(line, "pid=") + 4
				end := strings.Index(line[start:], ",")
				if end == -1 {
					end = strings.Index(line[start:], " ")
				}
				if end > 0 {
					pidStr := line[start : start+end]
					if pid, err := strconv.Atoi(pidStr); err == nil {
						return getProcessInfo(pid)
					}
				}
			}
		}
	}
	return nil
}
