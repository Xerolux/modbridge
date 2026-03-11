package api

import (
	"encoding/json"
	"fmt"
	"modbridge/pkg/config"
	"modbridge/pkg/logger"
	"modbridge/pkg/portmanager"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

func (s *Server) handleLogDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=proxy.log")
	w.Header().Set("Content-Type", "application/json")
	logs := s.log.GetRecent(10000)
	json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=config.json")
	w.Header().Set("Content-Type", "application/json")

	cfg := s.cfgMgr.Get()
	cfg.AdminPassHash = ""
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleConfigImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := s.cfgMgr.Update(func(c *config.Config) error {
		pass := c.AdminPassHash
		*c = newCfg
		c.AdminPassHash = pass
		return nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Stop all existing proxies before re-initializing with new config
	s.mgr.StopAll()
	s.mgr.Initialize()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSystemConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cfg := s.cfgMgr.Get()
		// Sanitize sensitive fields before sending to client
		cfg.AdminPassHash = ""
		cfg.EmailPassword = ""
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
		return
	}

	if r.Method == http.MethodPut {
		var req config.Config
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err := s.cfgMgr.Update(func(c *config.Config) error {
			c.LogLevel = req.LogLevel
			c.LogMaxSize = req.LogMaxSize
			c.LogMaxFiles = req.LogMaxFiles
			c.LogMaxAgeDays = req.LogMaxAgeDays
			c.TLSEnabled = req.TLSEnabled
			c.TLSCertFile = req.TLSCertFile
			c.TLSKeyFile = req.TLSKeyFile
			c.SessionTimeout = req.SessionTimeout
			c.CORSAllowedOrigins = req.CORSAllowedOrigins
			c.CORSAllowedMethods = req.CORSAllowedMethods
			c.CORSAllowedHeaders = req.CORSAllowedHeaders
			c.RateLimitEnabled = req.RateLimitEnabled
			c.RateLimitRequests = req.RateLimitRequests
			c.RateLimitBurst = req.RateLimitBurst
			c.IPWhitelistEnabled = req.IPWhitelistEnabled
			c.IPWhitelist = req.IPWhitelist
			c.IPBlacklistEnabled = req.IPBlacklistEnabled
			c.IPBlacklist = req.IPBlacklist
			c.EmailEnabled = req.EmailEnabled
			c.EmailSMTPServer = req.EmailSMTPServer
			c.EmailSMTPPort = req.EmailSMTPPort
			c.EmailFrom = req.EmailFrom
			c.EmailTo = req.EmailTo
			c.EmailUsername = req.EmailUsername
			c.EmailPassword = req.EmailPassword
			c.EmailAlertOnError = req.EmailAlertOnError
			c.EmailAlertOnWarning = req.EmailAlertOnWarning
			c.BackupEnabled = req.BackupEnabled
			c.BackupInterval = req.BackupInterval
			c.BackupRetention = req.BackupRetention
			c.BackupPath = req.BackupPath
			c.BackupDatabase = req.BackupDatabase
			c.BackupConfig = req.BackupConfig
			c.MetricsEnabled = req.MetricsEnabled
			c.MetricsPort = req.MetricsPort
			c.DebugMode = req.DebugMode
			c.MaxConnections = req.MaxConnections
			return nil
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Apply log level change immediately to the running logger
		s.log.SetLogLevel(logger.LogLevel(req.LogLevel))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

var startTime time.Time

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if startTime.IsZero() {
		startTime = time.Now()
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	proxies := s.mgr.GetProxies()
	runningProxies := 0

	for _, p := range proxies {
		if status, ok := p["status"].(string); ok && status == "Running" {
			runningProxies++
		}
	}

	info := map[string]interface{}{
		"uptime_seconds":  time.Since(startTime).Seconds(),
		"uptime_human":    time.Since(startTime).String(),
		"goroutines":      runtime.NumGoroutine(),
		"memory_alloc_mb": memStats.Alloc / 1024 / 1024,
		"memory_sys_mb":   memStats.Sys / 1024 / 1024,
		"memory_gc_mb":    memStats.NextGC / 1024 / 1024,
		"num_cpu":         runtime.NumCPU(),
		"total_proxies":   len(proxies),
		"running_proxies": runningProxies,
		"go_version":      runtime.Version(),
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

// handlePortDiagnostics checks port availability and shows process info
func (s *Server) handlePortDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Ports []int `json:"ports"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pm := portmanager.NewPortManager()
	results := pm.CheckPorts(req.Ports)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

// handlePortRelease forcefully releases a port
func (s *Server) handlePortRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Port int `json:"port"`
		PID  int `json:"pid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pm := portmanager.NewPortManager()

	// Verify the port is actually in use before killing
	portInfo := pm.CheckPort(req.Port)
	if portInfo.IsOpen {
		http.Error(w, "Port is already free", http.StatusConflict)
		return
	}

	// Kill the process
	if err := pm.KillProcess(req.PID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.log.Warn("ADMIN", fmt.Sprintf("Process %d on port %d killed by user", req.PID, req.Port))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Process terminated successfully",
	})
}

// handleCheckProxyPorts checks all configured proxy ports
func (s *Server) handleCheckProxyPorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.Get()
	var ports []int
	portMap := make(map[int]string) // port -> proxy name

	// Extract ports from config
	for _, proxy := range cfg.Proxies {
		// Parse port from listen_addr (format: ":502")
		portStr := proxy.ListenAddr
		if len(portStr) > 0 && portStr[0] == ':' {
			if port, err := strconv.Atoi(portStr[1:]); err == nil {
				ports = append(ports, port)
				portMap[port] = proxy.Name
			}
		}
	}

	// Check web port too
	webPortStr := cfg.WebPort
	if len(webPortStr) > 0 && webPortStr[0] == ':' {
		if port, err := strconv.Atoi(webPortStr[1:]); err == nil {
			ports = append(ports, port)
			portMap[port] = "WebUI"
		}
	}

	pm := portmanager.NewPortManager()
	results := pm.CheckPorts(ports)

	// Enrich results with proxy names
	for port, info := range results {
		if name, ok := portMap[port]; ok {
			info.State = "configured_for_" + name
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ports": results,
		"summary": map[string]int{
			"total":  len(ports),
			"free":   countFreePortsInMap(results),
			"in_use": len(ports) - countFreePortsInMap(results),
		},
	})
}

// countFreePortsInMap counts free ports in the results map
func countFreePortsInMap(results map[int]*portmanager.PortInfo) int {
	count := 0
	for _, info := range results {
		if info.IsOpen {
			count++
		}
	}
	return count
}

// handleProxyConnectivityCheck checks if target devices are reachable
func (s *Server) handleProxyConnectivityCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.Get()
	results := make(map[string]map[string]interface{})

	for _, proxy := range cfg.Proxies {
		testConn, err := net.DialTimeout("tcp", proxy.TargetAddr, 5*time.Second)
		isReachable := err == nil
		var errorMsg string

		if err != nil {
			errorMsg = err.Error()
		}

		if testConn != nil {
			testConn.Close()
		}

		results[proxy.ID] = map[string]interface{}{
			"name":        proxy.Name,
			"target":      proxy.TargetAddr,
			"reachable":   isReachable,
			"error":       errorMsg,
			"status":      "unknown",
			"listen_addr": proxy.ListenAddr,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
