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
	"runtime"
	"strconv"
	"strings"
	"sync"
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
	"modbridge/pkg/updater"
	"modbridge/pkg/users"
)

// maxJSONBodySize is the maximum allowed size for JSON request bodies.
const maxJSONBodySize = 1 << 20 // 1 MiB

// decodeJSON reads a request body up to maxJSONBodySize and unmarshals it.
// It returns http.ErrBodyReadAfterClose when the body exceeds the limit so
// callers can respond with 413 Request Entity Too Large.
func decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBodySize+1))
	if err != nil {
		return err
	}
	if len(body) > maxJSONBodySize {
		return http.ErrBodyReadAfterClose
	}
	return json.Unmarshal(body, v)
}

// writeJSONDecodeError writes a 413 or 400 response for a decodeJSON error.
func writeJSONDecodeError(w http.ResponseWriter, err error) {
	if err == http.ErrBodyReadAfterClose {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
}

// requirePermission validates the session and checks the required permission.
// It writes the appropriate HTTP error and returns nil if the check fails.
func (s *Server) requirePermission(w http.ResponseWriter, r *http.Request, permission rbac.Permission) *auth.Session {
	if s.auth == nil {
		http.Error(w, "Auth backend unavailable", http.StatusServiceUnavailable)
		return nil
	}

	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	session := s.auth.GetSession(cookie.Value)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	if !rbac.HasPermission(rbac.Role(session.Role), permission) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return nil
	}

	return session
}

// requestMeta extracts the actor identity (IP, User-Agent) from a request for
// audit logging. IP prefers X-Forwarded-For (first hop), falls back to the
// host portion of RemoteAddr.
func requestMeta(r *http.Request) (ip, userAgent string) {
	userAgent = r.UserAgent()
	ip = r.RemoteAddr
	if h := r.Header.Get("X-Forwarded-For"); h != "" {
		if idx := strings.Index(h, ","); idx > 0 {
			ip = strings.TrimSpace(h[:idx])
		} else {
			ip = strings.TrimSpace(h)
		}
	} else if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}
	return
}

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
	updater          *updater.Updater

	restartSignal chan struct{}
	restartOnce   sync.Once
}

// RestartSignal returns a channel that is closed when the server should
// gracefully restart. main.go listens on this channel and triggers a
// controlled shutdown (proxies → HTTP server → os.Exit) instead of the
// handler calling os.Exit directly.
func (s *Server) RestartSignal() <-chan struct{} {
	return s.restartSignal
}

// triggerRestart closes restartSignal exactly once. It is safe to call from
// multiple goroutines (e.g. POST /api/system/restart and the update-module's
// done-state poller) — without the sync.Once guard, a second close would
// panic and crash the process.
func (s *Server) triggerRestart() {
	s.restartOnce.Do(func() { close(s.restartSignal) })
}

// writeJSON encodes v as JSON and logs encoding failures. It avoids
// unchecked error return values throughout the HTTP handlers.
func (s *Server) writeJSON(w http.ResponseWriter, v interface{}) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.log.Warn("API", fmt.Sprintf("failed to encode JSON response: %v", err))
	}
}

// NewServer creates a new API server.
func NewServer(cfg *config.Manager, mgr *manager.Manager, a *auth.Authenticator, l *logger.Logger, db *database.DB, version, buildTime string) *Server {
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
		restartSignal:    make(chan struct{}),
		updater: updater.New("Xerolux/modbridge", updater.BuildInfo{
			Version:   version,
			BuildTime: buildTime,
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		}),
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
	mux.HandleFunc("/api/logout", csrfMW(s.handleLogout))
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

	// Protected routes. State-changing routes use csrfMW so the double-submit
	// cookie is verified. /api/me uses csrfMW as well so the CSRF cookie is
	// refreshed on every auth check performed by the frontend.
	mux.HandleFunc("/api/me", csrfMW(s.handleMe))
	mux.HandleFunc("/api/users", csrfMW(s.handleUsers))
	mux.HandleFunc("/api/users/", csrfMW(s.handleUserByID))
	mux.HandleFunc("/api/proxies", csrfMW(s.handleProxies))
	mux.HandleFunc("/api/proxies/stream", authMW(s.handleProxiesStream))
	mux.HandleFunc("/api/proxies/control", csrfMW(s.handleProxyControl))
	mux.HandleFunc("/api/devices", csrfMW(s.handleDevices))
	mux.HandleFunc("/api/devices/history", authMW(s.handleDeviceHistory))
	mux.HandleFunc("/api/logs", authMW(s.handleLogs))
	mux.HandleFunc("/api/logs/download", authMW(s.handleLogDownload))
	mux.HandleFunc("/api/logs/stream", authMW(s.handleLogStream))
	mux.HandleFunc("/api/audit/logs", authMW(s.handleAuditLogs))
	mux.HandleFunc("/api/audit/logs/export", authMW(s.handleAuditLogsExport))
	mux.HandleFunc("/api/config/export", authMW(s.handleConfigExport))
	mux.HandleFunc("/api/config/import", csrfMW(s.handleConfigImport))
	mux.HandleFunc("/api/config/rollback", csrfMW(s.handleConfigRollback))
	mux.HandleFunc("/api/config/webport", csrfMW(s.handleWebPort))
	mux.HandleFunc("/api/config/password", csrfMW(s.handleChangePassword))
	mux.HandleFunc("/api/config/system", csrfMW(s.handleSystemConfig))
	mux.HandleFunc("/api/system/restart", csrfMW(s.handleSystemRestart))
	mux.HandleFunc("/api/system/info", authMW(s.handleSystemInfo))
	mux.HandleFunc("/api/system/ports/diagnostics", csrfMW(s.handlePortDiagnostics))
	mux.HandleFunc("/api/system/ports/release", csrfMW(s.handlePortRelease))
	mux.HandleFunc("/api/system/ports/check", authMW(s.handleCheckProxyPorts))
	mux.HandleFunc("/api/system/diagnostics/connectivity", authMW(s.handleProxyConnectivityCheck))

	// Update endpoints — admin-only (RBAC checked inside handlers via requirePermission)
	mux.HandleFunc("/api/update/check", authMW(s.handleUpdateCheck))
	mux.HandleFunc("/api/update/perform", csrfMW(s.handleUpdatePerform))
	mux.HandleFunc("/api/update/status", authMW(s.handleUpdateStatus))
}

// Stop gracefully shuts down background goroutines owned by the server.
func (s *Server) Stop() {
	if s.csrf != nil {
		s.csrf.Stop()
	}
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
	if s.loginRateLimiter != nil {
		s.loginRateLimiter.Stop()
	}
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
	v = strings.ReplaceAll(v, "\t", "\\t")
	v = strings.ReplaceAll(v, "\x00", "\\0")
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

func writeSSEHeartbeat(w io.Writer) error {
	_, err := io.WriteString(w, "event: heartbeat\ndata: {}\n\n")
	return err
}

// multiUserEnabled reports whether DB-backed multi-user authentication is
// active. It requires a working user manager and is enabled either via the
// config flag or the MODBRIDGE_MULTI_USER environment override.
func (s *Server) multiUserEnabled() bool {
	if s.userMgr == nil || s.cfgMgr == nil {
		return false
	}
	if s.cfgMgr.Get().MultiUser {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("MODBRIDGE_MULTI_USER")), "true")
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfgMgr.Get()
	proxies := []map[string]interface{}{}
	if s.mgr != nil {
		proxies = s.mgr.GetProxies()
	}
	status := map[string]interface{}{
		"setup_required": cfg.AdminPassHash == "",
		"multi_user":     s.multiUserEnabled(),
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
		// Setup is deprecated: the default admin is auto-created on first run
		// (single-user hash) or via the user store (multi-user). The endpoint
		// is no longer reachable in normal operation; return 410 Gone so any
		// legacy caller gets an explicit signal.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "setup is deprecated; default admin is auto-created on first run",
		})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONDecodeError(w, err)
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

	cfg := s.cfgMgr.Get()

	// Multi-user mode: authenticate against the user database using real
	// identity (username + password), honoring enabled/expiry state.
	if s.multiUserEnabled() {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := decodeJSON(w, r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Username == "" || req.Password == "" {
			http.Error(w, "username and password are required", http.StatusBadRequest)
			return
		}

		user, err := s.userMgr.AuthenticateUser(strings.TrimSpace(req.Username), req.Password)
		if err != nil || user == nil {
			ip, ua := requestMeta(r)
			if s.auditor != nil {
				s.auditor.LogLogin(req.Username, ip, ua, "invalid credentials", false)
			}
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		s.finalizeLogin(w, r, user.ID, user.Username, user.Role, user.MustChangePassword)
		return
	}

	// Legacy single-user mode: a single global admin password from config.
	var req struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONDecodeError(w, err)
		return
	}

	if !auth.CheckPasswordHash(req.Password, cfg.AdminPassHash) {
		ip, ua := requestMeta(r)
		if s.auditor != nil {
			s.auditor.LogLogin("admin", ip, ua, "invalid password", false)
		}
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	s.finalizeLogin(w, r, "admin", "admin", "admin", cfg.ForcePasswordChange)
}

// finalizeLogin creates a session for the authenticated identity and writes the
// session/CSRF cookies plus the JSON success response. Shared by both login
// paths so cookie handling stays consistent.
func (s *Server) finalizeLogin(w http.ResponseWriter, r *http.Request, userID, username, role string, forcePasswordChange bool) {
	cfg := s.cfgMgr.Get()

	sessionTimeoutHours := cfg.SessionTimeout
	if sessionTimeoutHours <= 0 {
		sessionTimeoutHours = 24
	}

	token, err := s.auth.CreateSession(userID, username, role, time.Duration(sessionTimeoutHours)*time.Hour, forcePasswordChange)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit successful login. Logged here (not in handleLogin) because both
	// multi-user and legacy login paths funnel through finalizeLogin.
	ip, ua := requestMeta(r)
	if s.auditor != nil {
		s.auditor.LogLogin(username, ip, ua, "", true)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  time.Now().Add(time.Duration(sessionTimeoutHours) * time.Hour),
		HttpOnly: true,
		Secure:   cfg.TLSEnabled,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":               true,
		"force_password_change": forcePasswordChange,
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
		"user_id":              session.UserID,
		"username":             session.Username,
		"role":                 session.Role,
		"permissions":          permissions,
		"must_change_password": session.MustChangePassword,
	}); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to encode /api/me response: %v", err))
	}
}

// handleLogout ends the caller's server-side session. CSRF-protected because it
// is a state-changing POST. The client also clears its cookies; this endpoint
// guarantees the in-memory session is invalidated immediately and the logout
// is recorded in the audit log.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if c, err := r.Cookie("session_token"); err == nil && c.Value != "" {
		// Capture identity BEFORE invalidating, so the audit log still resolves.
		if session := s.auth.GetSession(c.Value); session != nil {
			if s.auditor != nil {
				s.auditor.LogLogout(session.UserID, session.Username, r.RemoteAddr, r.UserAgent())
			}
		}
		s.auth.InvalidateSession(c.Value)
	}
	// Clear cookies defensively even though the client does it too.
	for _, name := range []string{"session_token", "csrf_token"} {
		http.SetCookie(w, &http.Cookie{
			Name: name, Value: "", Expires: time.Unix(0, 0),
			Path: "/", HttpOnly: true, MaxAge: -1,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
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
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONDecodeError(w, err)
		return
	}

	// Multi-user mode: change the currently logged-in user's DB password.
	if s.multiUserEnabled() {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		session := s.auth.GetSession(cookie.Value)
		if session == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if err := s.userMgr.ChangePassword(session.UserID, req.CurrentPassword, req.NewPassword); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Invalidate all sessions for this user so stolen cookies cannot
		// continue to be used after a password change.
		s.auth.InvalidateUserSessions(session.UserID)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"success": true}); err != nil {
			s.log.Error("API", fmt.Sprintf("Failed to encode change-password response: %v", err))
		}
		return
	}

	// Legacy single-user mode: update the global admin hash.
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

	// In single-user mode there is only the admin account. Invalidate every
	// session so an attacker with a stolen cookie is forced to re-authenticate.
	s.auth.InvalidateAllSessions()

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
	ip, ua := requestMeta(r)

	if r.Method == http.MethodGet {
		if s.userMgr == nil {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        "1",
					"username":  "admin",
					"full_name": "Administrator",
					"email":     "admin@localhost",
					"role":      "admin",
					"enabled":   true,
				},
			}); err != nil {
				s.log.Warn("API", fmt.Sprintf("failed to encode fallback users response: %v", err))
			}
			return
		}

		users, err := s.userMgr.GetAllUsers()
		if err != nil {
			http.Error(w, "Failed to load users", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(users); err != nil {
			s.log.Warn("API", fmt.Sprintf("failed to encode users response: %v", err))
		}
		return
	}

	if r.Method == http.MethodPost {
		if s.userMgr == nil {
			http.Error(w, "User management not available", http.StatusServiceUnavailable)
			return
		}

		var req users.CreateUserRequest
		if err := decodeJSON(w, r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		createdBy := "admin"
		if session != nil && session.Username != "" {
			createdBy = session.Username
		}

		user, err := s.userMgr.CreateUser(&req, createdBy)
		if err != nil {
			if s.auditor != nil {
				s.auditor.LogUserAction("user.created", req.Username, session.UserID, session.Username, err.Error(), ip, ua, false)
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if s.auditor != nil {
			s.auditor.LogUserAction("user.created", user.ID, session.UserID, session.Username, "created user "+user.Username, ip, ua, true)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(user); err != nil {
			s.log.Warn("API", fmt.Sprintf("failed to encode created user response: %v", err))
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requirePermissionForUserRoute(w, r)
	if !ok {
		return
	}
	ip, ua := requestMeta(r)

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
		if err := json.NewEncoder(w).Encode(user); err != nil {
			s.log.Warn("API", fmt.Sprintf("failed to encode user response: %v", err))
		}
		return
	}

	if r.Method == http.MethodPut {
		var req users.UpdateUserRequest
		if err := decodeJSON(w, r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.userMgr.UpdateUser(id, &req); err != nil {
			if s.auditor != nil {
				s.auditor.LogUserAction("user.updated", id, session.UserID, session.Username, err.Error(), ip, ua, false)
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// If the account was disabled, the role changed, or the password
		// changed, force the user to re-authenticate on all devices.
		if (req.Enabled != nil && !*req.Enabled) || req.Role != nil || req.Password != nil {
			s.auth.InvalidateUserSessions(id)
		}

		if s.auditor != nil {
			s.auditor.LogUserAction("user.updated", id, session.UserID, session.Username, "", ip, ua, true)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			s.log.Warn("API", fmt.Sprintf("failed to encode user update response: %v", err))
		}
		return
	}

	if r.Method == http.MethodDelete {
		if session != nil && session.UserID == id {
			http.Error(w, "cannot delete active session user", http.StatusBadRequest)
			return
		}

		if err := s.userMgr.DeleteUser(id); err != nil {
			if s.auditor != nil {
				s.auditor.LogUserAction("user.deleted", id, session.UserID, session.Username, err.Error(), ip, ua, false)
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Ensure the deleted user cannot continue to use any active session.
		s.auth.InvalidateUserSessions(id)

		if s.auditor != nil {
			s.auditor.LogUserAction("user.deleted", id, session.UserID, session.Username, "", ip, ua, true)
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.requirePermission(w, r, rbac.PermProxyView) == nil {
		return
	}
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
			if err := writeSSEHeartbeat(w); err != nil {
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

	permissionByMethod := map[string]rbac.Permission{
		http.MethodGet:    rbac.PermProxyView,
		http.MethodPost:   rbac.PermProxyCreate,
		http.MethodPut:    rbac.PermProxyEdit,
		http.MethodDelete: rbac.PermProxyDelete,
	}

	permission, exists := permissionByMethod[r.Method]
	if !exists {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session := s.requirePermission(w, r, permission)
	if session == nil {
		return
	}
	ip, ua := requestMeta(r)

	if s.mgr == nil {
		http.Error(w, "Proxy manager unavailable", http.StatusServiceUnavailable)
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(s.mgr.GetProxies()); err != nil {
			s.log.Error("API", fmt.Sprintf("Failed to encode proxies response: %v", err))
		}
		return
	}

	if r.Method == http.MethodPost {
		var req config.ProxyConfig
		if err := decodeJSON(w, r, &req); err != nil {
			if err == http.ErrBodyReadAfterClose {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.ID == "" {
			req.ID = uuid.New().String()
		}

		if err := s.validator.ValidateProxyConfig(req); err != nil {
			if s.auditor != nil {
				s.auditor.LogProxyAction("proxy.created", req.ID, session.UserID, session.Username, req.Name, ip, ua, false)
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.mgr.AddProxy(req, true); err != nil {
			if s.auditor != nil {
				s.auditor.LogProxyAction("proxy.created", req.ID, session.UserID, session.Username, req.Name, ip, ua, false)
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if req.Enabled && !req.Paused {
			if err := s.mgr.StartProxy(req.ID); err != nil {
				s.log.Error(req.ID, fmt.Sprintf("Failed to start proxy after creation: %v", err))
				if s.auditor != nil {
					s.auditor.LogProxyAction("proxy.created", req.ID, session.UserID, session.Username, req.Name, ip, ua, false)
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if s.auditor != nil {
			s.auditor.LogProxyAction("proxy.created", req.ID, session.UserID, session.Username, req.Name, ip, ua, true)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodPut {
		var req config.ProxyConfig
		if err := decodeJSON(w, r, &req); err != nil {
			if err == http.ErrBodyReadAfterClose {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.validator.ValidateProxyConfig(req); err != nil {
			if s.auditor != nil {
				s.auditor.LogProxyAction("proxy.updated", req.ID, session.UserID, session.Username, req.Name, ip, ua, false)
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.mgr.UpdateProxy(req); err != nil {
			if s.auditor != nil {
				s.auditor.LogProxyAction("proxy.updated", req.ID, session.UserID, session.Username, req.Name, ip, ua, false)
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if s.auditor != nil {
			s.auditor.LogProxyAction("proxy.updated", req.ID, session.UserID, session.Username, req.Name, ip, ua, true)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if err := s.mgr.RemoveProxy(id); err != nil {
			if s.auditor != nil {
				s.auditor.LogProxyAction("proxy.deleted", id, session.UserID, session.Username, id, ip, ua, false)
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.auditor != nil {
			s.auditor.LogProxyAction("proxy.deleted", id, session.UserID, session.Username, id, ip, ua, true)
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

	session := s.requirePermission(w, r, rbac.PermProxyControl)
	if session == nil {
		return
	}
	ip, ua := requestMeta(r)

	var req struct {
		ID     string `json:"id"`
		Action string `json:"action"` // start, stop, restart, pause, resume, start_all, stop_all
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONDecodeError(w, err)
		return
	}

	var err error
	switch req.Action {
	case "start":
		err = s.mgr.StartProxy(req.ID)
	case "stop":
		err = s.mgr.StopProxy(req.ID)
	case "restart":
		if err = s.mgr.StopProxy(req.ID); err != nil {
			break
		}
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
		if s.auditor != nil {
			auditAction := actionForAudit(req.Action)
			if isBulkAction(req.Action) {
				s.auditor.LogAction("proxy."+req.Action, "proxy", "", session.UserID, session.Username, "", ip, ua, false, err.Error())
			} else {
				s.auditor.LogProxyAction(auditAction, req.ID, session.UserID, session.Username, "", ip, ua, false)
			}
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.auditor != nil {
		if isBulkAction(req.Action) {
			s.auditor.LogAction("proxy."+req.Action, "proxy", "", session.UserID, session.Username, "", ip, ua, true, "")
		} else {
			s.auditor.LogProxyAction(actionForAudit(req.Action), req.ID, session.UserID, session.Username, "", ip, ua, true)
		}
	}
	w.WriteHeader(http.StatusOK)
}

// actionForAudit maps a proxy-control action to its audit action string.
func actionForAudit(action string) string {
	switch action {
	case "start":
		return "proxy.started"
	case "stop":
		return "proxy.stopped"
	case "restart":
		return "proxy.restarted"
	case "pause":
		return "proxy.paused"
	case "resume":
		return "proxy.resumed"
	default:
		return "proxy." + action
	}
}

// isBulkAction reports whether the action targets all proxies (no single ID).
func isBulkAction(action string) bool {
	switch action {
	case "start_all", "stop_all", "restart_all":
		return true
	}
	return false
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if s.requirePermission(w, r, rbac.PermLogsView) == nil {
		return
	}
	logs := s.log.GetRecent(100)
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to encode logs response: %v", err))
	}
}

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	// Permission check MUST come before any flush — a 403 after headers are
	// sent is impossible.
	if s.requirePermission(w, r, rbac.PermLogsView) == nil {
		return
	}
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
			if err := writeSSEHeartbeat(w); err != nil {
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

	session := s.requirePermission(w, r, rbac.PermSystemRestart)
	if session == nil {
		return
	}
	ip, ua := requestMeta(r)

	if s.auditor != nil {
		s.auditor.LogAction("system.restart", "system", "", session.UserID, session.Username, "", ip, ua, true, "")
	}

	s.log.Info("API", "System restart requested via API")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"restarting"}`)); err != nil {
		s.log.Error("API", fmt.Sprintf("Failed to write restart response: %v", err))
		return
	}

	// Signal main() to perform a graceful shutdown. The shutdown sequence in
	// main.go will stop all proxies, call Server.Stop() and run
	// server.Shutdown() before exiting, ensuring in-flight requests and
	// Modbus connections are handled gracefully.
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.triggerRestart()
	}()
}

func (s *Server) handleWebPort(w http.ResponseWriter, r *http.Request) {
	permissionByMethod := map[string]rbac.Permission{
		http.MethodGet: rbac.PermConfigView,
		http.MethodPut: rbac.PermConfigEdit,
	}
	permission, exists := permissionByMethod[r.Method]
	if !exists {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.requirePermission(w, r, permission) == nil {
		return
	}

	if r.Method == http.MethodGet {
		cfg := s.cfgMgr.Get()
		w.Header().Set("Content-Type", "application/json")
		s.writeJSON(w, map[string]string{"web_port": cfg.WebPort})
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			WebPort string `json:"web_port"`
		}
		if err := decodeJSON(w, r, &req); err != nil {
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
		s.writeJSON(w, map[string]string{"status": "ok", "message": "Port updated, restart required"})
		return
	}
}
