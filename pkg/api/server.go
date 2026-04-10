package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"modbridge/pkg/audit"
	"modbridge/pkg/auth"
	"modbridge/pkg/config"
	"modbridge/pkg/database"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
	"modbridge/pkg/metrics"
	"modbridge/pkg/middleware"
	"modbridge/pkg/rbac"
	"modbridge/pkg/users"
)

// checkPortAvailable checks if a port is available for binding
func checkPortAvailable(port string) error {
	// Try to create a listener on the port
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("port already in use or invalid: %w", err)
	}
	// Close the listener immediately
	ln.Close()
	return nil
}

// Server is the API server.
type Server struct {
	cfgMgr           *config.Manager
	mgr              *manager.Manager
	auth             *auth.Authenticator
	log              *logger.Logger
	cors             *middleware.CORSMiddleware
	security         *middleware.SecurityMiddleware
	rateLimiter      *middleware.RateLimiter
	loginRateLimiter *middleware.RateLimiter
	csrf             *middleware.CSRFMiddleware
	validator        *middleware.Validator
	metrics          *metrics.Metrics
	userMgr          *users.Manager
	auditor          *audit.Auditor
}

// NewServer creates a new API server.
func NewServer(cfg *config.Manager, mgr *manager.Manager, a *auth.Authenticator, l *logger.Logger, db *database.DB) *Server {
	csrfSecret, err := buildCSRFSecret()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize CSRF secret: %v", err))
	}

	corsAllowedOrigins := []string{}
	if cfg != nil {
		corsAllowedOrigins = cfg.Get().CORSAllowedOrigins
	}

	// Initialize middlewares
	corsMW := middleware.NewCORSMiddleware(corsAllowedOrigins)
	secMW := middleware.NewSecurityMiddleware()
	rateLimiter := middleware.NewRateLimiter(60, 100)
	loginRateLimiter := middleware.NewRateLimiter(5, 10)
	csrfMW := middleware.NewCSRFMiddleware(csrfSecret)
	validator := middleware.NewValidator()

	var userMgr *users.Manager
	if db != nil {
		userMgr = users.NewManager(db)
	}

	var auditorInstance *audit.Auditor
	if db != nil {
		auditorInstance = audit.NewAuditor(db)
	}

	return &Server{
		cfgMgr:           cfg,
		mgr:              mgr,
		auth:             a,
		log:              l,
		cors:             corsMW,
		security:         secMW,
		rateLimiter:      rateLimiter,
		loginRateLimiter: loginRateLimiter,
		csrf:             csrfMW,
		validator:        validator,
		metrics:          metrics.NewMetrics(),
		userMgr:          userMgr,
		auditor:          auditorInstance,
	}
}

func buildCSRFSecret() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("MODBRIDGE_CSRF_SECRET")); configured != "" {
		return configured, nil
	}

	goEnv := strings.ToLower(strings.TrimSpace(os.Getenv("GO_ENV")))
	modbridgeEnv := strings.ToLower(strings.TrimSpace(os.Getenv("MODBRIDGE_ENV")))
	if goEnv == "production" || modbridgeEnv == "production" {
		return "", fmt.Errorf("MODBRIDGE_CSRF_SECRET is required in production")
	}

	csrfSecretBytes := make([]byte, 32)
	if _, err := rand.Read(csrfSecretBytes); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}

	return hex.EncodeToString(csrfSecretBytes), nil
}

// Routes registers routes.
func (s *Server) Routes(mux *http.ServeMux) {
	// Helper function to compose middlewares
	compose := func(middlewares ...func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
		return func(handler http.HandlerFunc) http.HandlerFunc {
			for i := len(middlewares) - 1; i >= 0; i-- {
				handler = middlewares[i](handler)
			}
			return handler
		}
	}

	// Common middlewares
	authMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}
	if s.auth != nil {
		authMiddleware = s.auth.Middleware
	}

	csrfMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "CSRF protection unavailable", http.StatusServiceUnavailable)
		}
	}
	if s.csrf != nil {
		csrfMiddleware = s.csrf.Middleware
	}

	publicMW := compose(s.cors.Middleware, s.security.Middleware, s.rateLimiter.Middleware)
	authMW := compose(s.cors.Middleware, s.security.Middleware, authMiddleware)
	csrfMW := compose(s.cors.Middleware, s.security.Middleware, authMiddleware, csrfMiddleware)

	// Public routes
	mux.HandleFunc("/api/health", publicMW(s.handleHealth))
	mux.HandleFunc("/api/ready", publicMW(s.handleReady))
	mux.HandleFunc("/api/status", publicMW(s.handleStatus))
	mux.HandleFunc("/api/metrics", s.cors.Middleware(s.handleMetrics))
	mux.HandleFunc("/api/login", s.cors.Middleware(s.security.Middleware(s.loginRateLimiter.Middleware(s.handleLogin))))
	mux.HandleFunc("/api/setup", s.cors.Middleware(s.security.Middleware(s.handleSetup)))

	// Pprof endpoints (debug mode only)
	if os.Getenv("DEBUG") == "true" {
		debugMW := compose(s.cors.Middleware, s.security.Middleware, authMiddleware)
		mux.Handle("/debug/pprof/", debugMW(pprof.Index))
		mux.Handle("/debug/pprof/cmdline", debugMW(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", debugMW(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", debugMW(pprof.Symbol))
		mux.Handle("/debug/pprof/trace", debugMW(pprof.Trace))
	}

	// Protected routes
	mux.HandleFunc("/api/me", authMW(s.handleMe))
	mux.HandleFunc("/api/users", authMW(s.handleUsers))
	mux.HandleFunc("/api/users/", authMW(s.handleUserByID))
	mux.HandleFunc("/api/proxies", authMW(s.handleProxies))
	mux.HandleFunc("/api/proxies/stream", authMW(s.handleProxiesStream))
	mux.HandleFunc("/api/proxies/control", csrfMW(s.handleProxyControl))
	mux.HandleFunc("/api/devices", authMW(s.handleDevices))
	mux.HandleFunc("/api/devices/history", authMW(s.handleDeviceHistory))
	mux.HandleFunc("/api/logs", authMW(s.handleLogs))
	mux.HandleFunc("/api/logs/download", authMW(s.handleLogDownload))
	mux.HandleFunc("/api/logs/stream", authMW(s.handleLogStream))
	mux.HandleFunc("/api/audit/logs", authMW(s.handleAuditLogs))
	mux.HandleFunc("/api/audit/logs/export", authMW(s.handleAuditLogsExport))
	mux.HandleFunc("/api/config/export", authMW(s.handleConfigExport))
	mux.HandleFunc("/api/config/import", csrfMW(s.handleConfigImport))
	mux.HandleFunc("/api/config/webport", csrfMW(s.handleWebPort))
	mux.HandleFunc("/api/config/password", csrfMW(s.handleChangePassword))
	mux.HandleFunc("/api/config/system", csrfMW(s.handleSystemConfig))
	mux.HandleFunc("/api/system/restart", csrfMW(s.handleSystemRestart))
	mux.HandleFunc("/api/system/info", authMW(s.handleSystemInfo))
	mux.HandleFunc("/api/system/ports/diagnostics", csrfMW(s.handlePortDiagnostics))
	mux.HandleFunc("/api/system/ports/release", csrfMW(s.handlePortRelease))
	mux.HandleFunc("/api/system/ports/check", authMW(s.handleCheckProxyPorts))
	mux.HandleFunc("/api/system/diagnostics/connectivity", authMW(s.handleProxyConnectivityCheck))
}

// handleHealth is a health check endpoint.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	}); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to encode health response: %v", err))
	}
}

// handleReady is a readiness check endpoint.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	checks := map[string]string{
		"server": "ok",
	}
	ready := true

	if s.cfgMgr == nil {
		checks["config"] = "missing"
		ready = false
	} else {
		cfg := s.cfgMgr.Get()
		if strings.TrimSpace(cfg.WebPort) == "" {
			checks["config"] = "invalid"
			ready = false
		} else {
			checks["config"] = "ok"
		}
	}

	if s.mgr == nil {
		checks["manager"] = "missing"
		ready = false
	} else {
		checks["manager"] = "ok"
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"ready":     ready,
		"checks":    checks,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to encode ready response: %v", err))
	}
}

// handleMetrics returns Prometheus metrics.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	var output strings.Builder
	output.WriteString(s.metrics.GetPrometheusMetrics())

	if s.mgr != nil {
		proxies := s.mgr.GetProxies()
		if len(proxies) > 0 {
			output.WriteString("# HELP modbridge_proxy_requests_current Current request counter snapshot per proxy\n")
			output.WriteString("# TYPE modbridge_proxy_requests_current gauge\n")
			output.WriteString("# HELP modbridge_proxy_errors_current Current error counter snapshot per proxy\n")
			output.WriteString("# TYPE modbridge_proxy_errors_current gauge\n")
			output.WriteString("# HELP modbridge_proxy_active_connections_current Current active connections per proxy\n")
			output.WriteString("# TYPE modbridge_proxy_active_connections_current gauge\n")
			output.WriteString("# HELP modbridge_proxy_latency_p95_ms Current p95 latency in milliseconds per proxy\n")
			output.WriteString("# TYPE modbridge_proxy_latency_p95_ms gauge\n")
			output.WriteString("# HELP modbridge_proxy_status_running 1 when proxy is running, else 0\n")
			output.WriteString("# TYPE modbridge_proxy_status_running gauge\n")

			for _, proxyData := range proxies {
				id, _ := proxyData["id"].(string)
				if id == "" {
					continue
				}
				label := fmt.Sprintf(`proxy_id="%s"`, escapePrometheusLabelValue(id))
				output.WriteString(fmt.Sprintf("modbridge_proxy_requests_current{%s} %d\n", label, getInt64MetricValue(proxyData["requests"])))
				output.WriteString(fmt.Sprintf("modbridge_proxy_errors_current{%s} %d\n", label, getInt64MetricValue(proxyData["errors"])))
				output.WriteString(fmt.Sprintf("modbridge_proxy_active_connections_current{%s} %d\n", label, getInt64MetricValue(proxyData["active_connections"])))
				output.WriteString(fmt.Sprintf("modbridge_proxy_latency_p95_ms{%s} %f\n", label, getFloat64MetricValue(proxyData["latency_p95_ms"])))

				status, _ := proxyData["status"].(string)
				running := 0
				if status == "Running" {
					running = 1
				}
				output.WriteString(fmt.Sprintf("modbridge_proxy_status_running{%s} %d\n", label, running))
			}
			output.WriteString("\n")
		}
	}

	_, _ = w.Write([]byte(output.String()))
}

func escapePrometheusLabelValue(v string) string {
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "\"", "\\\"")
	v = strings.ReplaceAll(v, "\n", "\\n")
	return v
}

func getInt64MetricValue(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		parsed, err := n.Int64()
		if err == nil {
			return parsed
		}
	}
	return 0
}

func getFloat64MetricValue(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case json.Number:
		parsed, err := n.Float64()
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseFloat(n, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfgMgr.Get()
	proxies := []map[string]interface{}{}
	if s.mgr != nil {
		proxies = s.mgr.GetProxies()
	}
	status := map[string]interface{}{
		"setup_required": cfg.AdminPassHash == "",
		"proxies":        proxies,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to encode status response: %v", err))
	}
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

	err = s.cfgMgr.Update(func(c *config.Config) error {
		c.AdminPassHash = hash
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	token, err := s.auth.CreateSession("admin", "admin", "admin")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   cfg.TLSEnabled, // Only send over HTTPS if TLS is enabled
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Generate and set CSRF token
	csrfToken := s.csrf.GenerateToken(token)
	if csrfToken == "" {
		http.Error(w, "Failed to generate CSRF token", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   cfg.TLSEnabled,
		SameSite: http.SameSiteStrictMode,
	})

	// Return login status including force_password_change flag
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":               true,
		"force_password_change": cfg.ForcePasswordChange,
	}); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to encode login response: %v", err))
	}
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.auth == nil {
		http.Error(w, "Auth backend unavailable", http.StatusServiceUnavailable)
		return
	}

	c, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	session := s.auth.GetSession(c.Value)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	permissions := []string{}
	if perms := rbac.GetRolePermissions(rbac.Role(session.Role)); len(perms) > 0 {
		permissions = make([]string, len(perms))
		for i, p := range perms {
			permissions[i] = string(p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":     session.UserID,
		"username":    session.Username,
		"role":        session.Role,
		"permissions": permissions,
	}); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to encode /api/me response: %v", err))
	}
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := s.cfgMgr.Get()
	if !auth.CheckPasswordHash(req.CurrentPassword, cfg.AdminPassHash) {
		http.Error(w, "Invalid current password", http.StatusUnauthorized)
		return
	}

	// HashPassword now includes validation for password strength
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		// Return bad request for validation errors, internal error for actual failures
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.cfgMgr.Update(func(c *config.Config) error {
		c.AdminPassHash = hash
		c.ForcePasswordChange = false
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	}); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to encode change-password response: %v", err))
	}
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requirePermissionForUserRoute(w, r)
	if !ok {
		return
	}

	if r.Method == http.MethodGet {
		if s.userMgr == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        "1",
					"username":  "admin",
					"full_name": "Administrator",
					"email":     "admin@localhost",
					"role":      "admin",
					"enabled":   true,
				},
			})
			return
		}

		users, err := s.userMgr.GetAllUsers()
		if err != nil {
			http.Error(w, "Failed to load users", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
		return
	}

	if r.Method == http.MethodPost {
		if s.userMgr == nil {
			http.Error(w, "User management not available", http.StatusServiceUnavailable)
			return
		}

		var req users.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		createdBy := "admin"
		if session != nil && session.Username != "" {
			createdBy = session.Username
		}

		user, err := s.userMgr.CreateUser(&req, createdBy)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requirePermissionForUserRoute(w, r)
	if !ok {
		return
	}

	if s.userMgr == nil {
		http.Error(w, "User management not available", http.StatusServiceUnavailable)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if id == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		user, err := s.userMgr.GetUser(id)
		if err != nil {
			http.Error(w, "Failed to get user", http.StatusInternalServerError)
			return
		}
		if user == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
		return
	}

	if r.Method == http.MethodPut {
		var req users.UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.userMgr.UpdateUser(id, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	if r.Method == http.MethodDelete {
		if session != nil && session.UserID == id {
			http.Error(w, "cannot delete active session user", http.StatusBadRequest)
			return
		}

		if err := s.userMgr.DeleteUser(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) requirePermissionForUserRoute(w http.ResponseWriter, r *http.Request) (*auth.Session, bool) {
	permissionByMethod := map[string]rbac.Permission{
		http.MethodGet:    rbac.PermUserView,
		http.MethodPost:   rbac.PermUserCreate,
		http.MethodPut:    rbac.PermUserEdit,
		http.MethodDelete: rbac.PermUserDelete,
	}

	permission, exists := permissionByMethod[r.Method]
	if !exists {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return nil, false
	}

	if s.auth == nil {
		http.Error(w, "Auth backend unavailable", http.StatusServiceUnavailable)
		return nil, false
	}

	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}

	session := s.auth.GetSession(cookie.Value)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}

	if !rbac.HasPermission(rbac.Role(session.Role), permission) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return nil, false
	}

	return session, true
}

// handleProxiesStream streams proxy updates via SSE
func (s *Server) handleProxiesStream(w http.ResponseWriter, r *http.Request) {
	if s.mgr == nil {
		http.Error(w, "Proxy manager unavailable", http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := s.mgr.GetProxyEventsSubscription()
	defer s.mgr.UnsubscribeProxyEvents(ch)

	timeout := time.NewTimer(30 * time.Minute)
	defer timeout.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
			timeout.Reset(30 * time.Minute)
		case <-timeout.C:
			s.log.Info("SSE stream timeout, closing connection", "")
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				s.log.Error("API", fmt.Sprintf("Failed to marshal SSE event: %v", err))
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return // Client disconnected
			}
			flusher.Flush()
			timeout.Reset(30 * time.Minute)
		}
	}
}

func (s *Server) handleProxies(w http.ResponseWriter, r *http.Request) {
	s.log.Info("API", fmt.Sprintf("handleProxies called: %s %s", r.Method, r.URL.Path))

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(s.mgr.GetProxies()); err != nil {
			s.log.Error("API", fmt.Sprintf("Failed to encode proxies response: %v", err))
		}
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

		if err := s.validator.ValidateProxyConfig(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.mgr.AddProxy(req, true); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if req.Enabled && !req.Paused {
			_ = s.mgr.StartProxy(req.ID)
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodPut {
		// Read raw JSON to handle flexible tags field
		var rawMap map[string]interface{}
		const maxProxyPayloadBytes = 1 << 20 // 1 MiB
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxProxyPayloadBytes+1))
		if err != nil {
			s.log.Error("API", fmt.Sprintf("Failed to read body: %v", err))
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		if len(bodyBytes) > maxProxyPayloadBytes {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		s.log.Info("API", fmt.Sprintf("PUT /api/proxies payload_size=%d", len(bodyBytes)))

		if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
			s.log.Error("API", fmt.Sprintf("Failed to unmarshal to map: %v", err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Build ProxyConfig manually from rawMap to handle tags flexibly
		req := config.ProxyConfig{}

		// Parse required fields
		if v, ok := rawMap["id"].(string); ok {
			req.ID = v
		}
		if v, ok := rawMap["name"].(string); ok {
			req.Name = v
		}
		if v, ok := rawMap["listen_addr"].(string); ok {
			req.ListenAddr = v
		}
		if v, ok := rawMap["target_addr"].(string); ok {
			req.TargetAddr = v
		}
		if v, ok := rawMap["enabled"].(bool); ok {
			req.Enabled = v
		}
		if v, ok := rawMap["paused"].(bool); ok {
			req.Paused = v
		}
		if v, ok := rawMap["description"].(string); ok {
			req.Description = v
		}

		// Parse numeric fields
		if v, ok := rawMap["connection_timeout"].(float64); ok {
			req.ConnectionTimeout = int(v)
		}
		if v, ok := rawMap["read_timeout"].(float64); ok {
			req.ReadTimeout = int(v)
		}
		if v, ok := rawMap["max_retries"].(float64); ok {
			req.MaxRetries = int(v)
		}
		if v, ok := rawMap["max_read_size"].(float64); ok {
			req.MaxReadSize = int(v)
		}

		// Handle tags field flexibly (string or array)
		if tagsVal, ok := rawMap["tags"]; ok {
			switch v := tagsVal.(type) {
			case string:
				if v == "" {
					req.Tags = config.FlexibleTags{}
				} else {
					tags := strings.Split(v, ",")
					result := make([]string, 0, len(tags))
					for _, tag := range tags {
						trimmed := strings.TrimSpace(tag)
						if trimmed != "" {
							result = append(result, trimmed)
						}
					}
					req.Tags = config.FlexibleTags(result)
				}
			case []interface{}:
				result := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						result = append(result, str)
					}
				}
				req.Tags = config.FlexibleTags(result)
			default:
				req.Tags = config.FlexibleTags{}
			}
		} else {
			req.Tags = config.FlexibleTags{}
		}

		if err := s.validator.ValidateProxyConfig(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.mgr.UpdateProxy(req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleProxyControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID     string `json:"id"`
		Action string `json:"action"` // start, stop, restart, pause, resume, start_all, stop_all
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
	case "pause":
		err = s.mgr.PauseProxy(req.ID)
	case "resume":
		err = s.mgr.ResumeProxy(req.ID)
	case "start_all":
		s.mgr.StartAll()
	case "stop_all":
		s.mgr.StopAll()
	case "restart_all":
		s.mgr.StopAll()
		time.Sleep(100 * time.Millisecond)
		s.mgr.StartAll()
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
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to encode logs response: %v", err))
	}
}

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	if s.log == nil {
		http.Error(w, "Logger unavailable", http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := s.log.Subscribe()
	defer s.log.Unsubscribe(ch)

	timeout := time.NewTimer(30 * time.Minute)
	defer timeout.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
			timeout.Reset(30 * time.Minute)
		case <-timeout.C:
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(entry)
			if err != nil {
				s.log.Error("API", fmt.Sprintf("Failed to marshal log entry: %v", err))
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return // Client disconnected
			}
			flusher.Flush()
			timeout.Reset(30 * time.Minute)
		}
	}
}

func (s *Server) handleSystemRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.log.Info("System restart requested via API", "")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"restarting"}`)); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to write restart response: %v", err))
		return
	}

	// Stop all proxies gracefully and exit for restart
	go func() {
		time.Sleep(500 * time.Millisecond)
		if s.mgr != nil {
			s.mgr.StopAll()
		}
		time.Sleep(500 * time.Millisecond)
		s.log.Info("System restarting now", "")
		// Exit with code 0 so the process manager (systemd, docker, etc.) can restart it
		os.Exit(0)
	}()
}

func (s *Server) handleWebPort(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cfg := s.cfgMgr.Get()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"web_port": cfg.WebPort})
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			WebPort string `json:"web_port"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate port format
		if req.WebPort == "" {
			http.Error(w, "web_port cannot be empty", http.StatusBadRequest)
			return
		}

		if err := s.validator.ValidatePort(req.WebPort); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if port is available by trying to bind to it
		if err := checkPortAvailable(req.WebPort); err != nil {
			http.Error(w, fmt.Sprintf("port %s is not available: %v", req.WebPort, err), http.StatusBadRequest)
			return
		}

		// Update config
		if err := s.cfgMgr.Update(func(c *config.Config) error {
			c.WebPort = req.WebPort
			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Port updated, restart required"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
