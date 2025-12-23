package api

import (
	"encoding/json"
	"fmt"
	"modbusproxy/pkg/auth"
	"modbusproxy/pkg/config"
	"modbusproxy/pkg/logger"
	"modbusproxy/pkg/manager"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Server is the API server.
type Server struct {
	cfgMgr *config.Manager
	mgr    *manager.Manager
	auth   *auth.Authenticator
	log    *logger.Logger
}

// NewServer creates a new API server.
func NewServer(cfg *config.Manager, mgr *manager.Manager, a *auth.Authenticator, l *logger.Logger) *Server {
	return &Server{
		cfgMgr: cfg,
		mgr:    mgr,
		auth:   a,
		log:    l,
	}
}

// Routes registers routes.
func (s *Server) Routes(mux *http.ServeMux) {
	// Public health endpoints
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/ready", s.handleReady)
	mux.HandleFunc("/api/status", s.corsMiddleware(s.handleStatus))
	mux.HandleFunc("/api/login", s.corsMiddleware(s.handleLogin))
	mux.HandleFunc("/api/setup", s.corsMiddleware(s.handleSetup))

	// Protected
	mux.HandleFunc("/api/proxies", s.corsMiddleware(s.auth.Middleware(s.handleProxies)))
	mux.HandleFunc("/api/proxies/control", s.corsMiddleware(s.auth.Middleware(s.handleProxyControl)))
	mux.HandleFunc("/api/devices", s.corsMiddleware(s.auth.Middleware(s.handleDevices)))
	mux.HandleFunc("/api/logs", s.corsMiddleware(s.auth.Middleware(s.handleLogs)))
	mux.HandleFunc("/api/logs/download", s.corsMiddleware(s.auth.Middleware(s.handleLogDownload)))
	mux.HandleFunc("/api/logs/stream", s.corsMiddleware(s.auth.Middleware(s.handleLogStream)))
	mux.HandleFunc("/api/config/export", s.corsMiddleware(s.auth.Middleware(s.handleConfigExport)))
	mux.HandleFunc("/api/config/import", s.corsMiddleware(s.auth.Middleware(s.handleConfigImport)))
}

// corsMiddleware adds CORS headers to responses.
func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// HealthCheck represents a health check result.
type HealthCheck struct {
	Status      string                    `json:"status"`      // "healthy", "degraded", "unhealthy"
	Timestamp   string                    `json:"timestamp"`
	Uptime      float64                   `json:"uptime_seconds"`
	Version     string                    `json:"version"`
	Checks      map[string]ComponentCheck `json:"checks"`
}

// ComponentCheck represents the health of a component.
type ComponentCheck struct {
	Status  string `json:"status"`  // "pass", "fail", "warn"
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleHealth is an enhanced health check endpoint (liveness probe).
// Returns 200 if the application is alive, 503 otherwise.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := HealthCheck{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0", // TODO: Get from version file
		Checks:    make(map[string]ComponentCheck),
	}

	// Check logger
	if s.log != nil {
		health.Checks["logger"] = ComponentCheck{
			Status:  "pass",
			Message: "Logger operational",
		}
	} else {
		health.Status = "unhealthy"
		health.Checks["logger"] = ComponentCheck{
			Status: "fail",
			Error:  "Logger not initialized",
		}
	}

	// Check config manager
	if s.cfgMgr != nil {
		health.Checks["config"] = ComponentCheck{
			Status:  "pass",
			Message: "Configuration manager operational",
		}
	} else {
		health.Status = "unhealthy"
		health.Checks["config"] = ComponentCheck{
			Status: "fail",
			Error:  "Config manager not initialized",
		}
	}

	// Check proxy manager
	if s.mgr != nil {
		proxies := s.mgr.GetProxies()
		health.Checks["proxy_manager"] = ComponentCheck{
			Status:  "pass",
			Message: fmt.Sprintf("%d proxies managed", len(proxies)),
		}
	} else {
		health.Status = "unhealthy"
		health.Checks["proxy_manager"] = ComponentCheck{
			Status: "fail",
			Error:  "Proxy manager not initialized",
		}
	}

	// Set HTTP status code
	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(health)
}

// handleReady is a readiness check endpoint (readiness probe).
// Returns 200 if the application is ready to serve traffic, 503 otherwise.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	readiness := HealthCheck{
		Status:    "ready",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
		Checks:    make(map[string]ComponentCheck),
	}

	// Check if setup is complete
	cfg := s.cfgMgr.Get()
	if cfg.AdminPassHash == "" {
		readiness.Status = "degraded"
		readiness.Checks["setup"] = ComponentCheck{
			Status:  "warn",
			Message: "Setup not completed",
		}
	} else {
		readiness.Checks["setup"] = ComponentCheck{
			Status:  "pass",
			Message: "Setup complete",
		}
	}

	// Check proxy status
	proxies := s.mgr.GetProxies()
	runningCount := 0
	errorCount := 0
	for _, p := range proxies {
		if status, ok := p["status"].(string); ok {
			if status == "Running" {
				runningCount++
			} else if status == "Error" {
				errorCount++
			}
		}
	}

	if errorCount > 0 {
		readiness.Status = "degraded"
		readiness.Checks["proxies"] = ComponentCheck{
			Status:  "warn",
			Message: fmt.Sprintf("%d running, %d in error state", runningCount, errorCount),
		}
	} else if len(proxies) > 0 {
		readiness.Checks["proxies"] = ComponentCheck{
			Status:  "pass",
			Message: fmt.Sprintf("%d proxies running", runningCount),
		}
	} else {
		readiness.Checks["proxies"] = ComponentCheck{
			Status:  "pass",
			Message: "No proxies configured",
		}
	}

	// Set HTTP status code
	statusCode := http.StatusOK
	if readiness.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(readiness)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfgMgr.Get()
	status := map[string]interface{}{
		"setup_required": cfg.AdminPassHash == "",
		"proxies":        s.mgr.GetProxies(),
	}
	_ = json.NewEncoder(w).Encode(status)
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.Get()
	if cfg.AdminPassHash != "" {
		http.Error(w, "Already setup", http.StatusForbidden)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = s.cfgMgr.Update(func(c *config.Config) error {
		c.AdminPassHash = hash
		return nil
	})

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := s.cfgMgr.Get()
	if !auth.CheckPasswordHash(req.Password, cfg.AdminPassHash) {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	token, err := s.auth.CreateSession()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleProxies(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(s.mgr.GetProxies())
		return
	}

	if r.Method == http.MethodPost {
		var req config.ProxyConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			req.ID = uuid.New().String()
		}

		if err := s.mgr.AddProxy(req, true); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// If enabled, start it
		if req.Enabled {
			_ = s.mgr.StartProxy(req.ID)
		}
		
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if err := s.mgr.RemoveProxy(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (s *Server) handleProxyControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ID     string `json:"id"`
		Action string `json:"action"` // start, stop, restart
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var err error
	switch req.Action {
	case "start":
		err = s.mgr.StartProxy(req.ID)
	case "stop":
		err = s.mgr.StopProxy(req.ID)
	case "restart":
		_ = s.mgr.StopProxy(req.ID)
		time.Sleep(100 * time.Millisecond)
		err = s.mgr.StartProxy(req.ID)
	default:
		err = fmt.Errorf("unknown action")
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs := s.log.GetRecent(100)
	_ = json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.log.Subscribe()
	defer s.log.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case entry := <-ch:
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
