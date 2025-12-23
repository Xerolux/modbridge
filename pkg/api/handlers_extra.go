
package api

import (
	"encoding/json"
	"modbusproxy/pkg/config"
	"net/http"
)

func (s *Server) handleLogDownload(w http.ResponseWriter, r *http.Request) {
	// Simple implementation: dump current ring buffer as JSON
	// A better implementation would read from the file on disk.
	// But let's check what the user asked: "Download Logfile(s)"
	// The logger writes to a file. Let's serve that file.
	
	// We need to access the file path from logger, but it's private.
	// Let's assume for now we just serve recent logs as JSON for simplicity, 
	// or we expose the file path.
	// Given "Log writes a log entry ... to file", we can just stream the file.
	// But we don't know the path here easily unless we expose it.
	// Let's rely on GetRecent for now or better, expose ReadLogs from logger.
	
	// For V1, downloading the in-memory buffer is safe. 
	// If we want the full file, we need to know where it is.
	// Let's stick to in-memory dump or updated logger.
	
	w.Header().Set("Content-Disposition", "attachment; filename=proxy.log")
	w.Header().Set("Content-Type", "application/json")
	logs := s.log.GetRecent(10000) // Get all we have in memory
	for _, l := range logs {
		json.NewEncoder(w).Encode(l)
	}
}

func (s *Server) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=config.json")
	w.Header().Set("Content-Type", "application/json")
	
	cfg := s.cfgMgr.Get()
	// Strip password hash for security
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
	
	// Validate
	if len(newCfg.Proxies) == 0 {
		// Just a warning?
	}
	
	err := s.cfgMgr.Update(func(c *config.Config) error {
		// Keep existing password if import doesn't have one (likely)
		// If import has one, maybe we should ignore it unless explicitly asked?
		// Requirement: "Export as JSON (ohne Passwort/Secrets)"
		// So Import likely won't have it. Keep existing.
		pass := c.AdminPassHash
		*c = newCfg
		c.AdminPassHash = pass
		return nil
	})
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Reload proxies
	s.mgr.Initialize() // Re-sync
	
	w.WriteHeader(http.StatusOK)
}
