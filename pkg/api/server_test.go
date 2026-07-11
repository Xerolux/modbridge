package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"modbridge/pkg/auth"
	"modbridge/pkg/config"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
	"modbridge/pkg/middleware"
)

func TestHandleHealth(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	server := NewServer(config.NewManager("test.json"), manager.NewManager(config.NewManager("test.json"), log, nil), nil, log, nil, "test", "unknown")

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", response["status"])
	}
}

func TestHandleStatus(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	cfgMgr := config.NewManager("test.json")
	mgr := manager.NewManager(cfgMgr, log, nil)
	server := NewServer(cfgMgr, mgr, nil, log, nil, "test", "unknown")

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()

	server.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["setup_required"] != true {
		t.Error("Expected setup_required true")
	}

	proxies, ok := response["proxies"].([]interface{})
	if !ok {
		t.Fatalf("proxies is not a list, got %T: %v", response["proxies"], response["proxies"])
	}

	if len(proxies) != 0 {
		t.Errorf("Expected 0 proxies, got %d", len(proxies))
	}
}

// proxyTestServer returns a server and an admin session cookie for use in
// handler tests that now require RBAC checks.
func proxyTestServer(t *testing.T) (*Server, *manager.Manager, string) {
	t.Helper()
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	t.Cleanup(func() { log.Close() })
	cfgMgr := config.NewManager("test.json")
	mgr := manager.NewManager(cfgMgr, log, nil)
	authenticator := auth.NewAuthenticator()
	server := NewServer(cfgMgr, mgr, authenticator, log, nil, "test", "unknown")
	token, err := authenticator.CreateSession("1", "admin", "admin", 24*time.Hour, false)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	return server, mgr, token
}

func TestHandleProxiesGet(t *testing.T) {
	server, _, token := proxyTestServer(t)

	req := httptest.NewRequest("GET", "/api/proxies", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	server.handleProxies(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var proxies []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &proxies); err != nil {
		t.Fatalf("Failed to unmarshal proxies: %v", err)
	}

	if len(proxies) != 0 {
		t.Errorf("Expected 0 proxies, got %d", len(proxies))
	}
}

func TestHandleProxiesPostInvalid(t *testing.T) {
	server, _, token := proxyTestServer(t)

	req := httptest.NewRequest("POST", "/api/proxies", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	server.handleProxies(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleProxiesPostValid(t *testing.T) {
	server, mgr, token := proxyTestServer(t)

	cfg := config.ProxyConfig{
		ID:                "test-id",
		Name:              "Test Proxy",
		ListenAddr:        ":5020",
		TargetAddr:        "192.168.1.100:502",
		ConnectionTimeout: 5,
		ReadTimeout:       5,
		MaxRetries:        3,
		MaxReadSize:       100,
	}

	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest("POST", "/api/proxies", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	server.handleProxies(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	proxies := mgr.GetProxies()
	if len(proxies) != 1 {
		t.Errorf("Expected 1 proxy, got %d", len(proxies))
	}
}

func TestHandleProxiesMethodNotAllowed(t *testing.T) {
	server, _, token := proxyTestServer(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/proxies", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	server.handleProxies(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleProxiesPutRejectsOversizedBody(t *testing.T) {
	server, _, token := proxyTestServer(t)

	oversized := bytes.Repeat([]byte("a"), (1<<20)+1)
	req := httptest.NewRequest(http.MethodPut, "/api/proxies", bytes.NewReader(oversized))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	server.handleProxies(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("Expected status %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
}

func TestMiddlewareChain(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	secMW := middleware.NewSecurityMiddleware()
	corsMW := middleware.NewCORSMiddleware([]string{})

	called := false
	handler := secMW.Middleware(corsMW.Middleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if !called {
		t.Error("Handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options header missing")
	}

	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options header missing")
	}
}

func TestHandleReady(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	cfgMgr := config.NewManager("test.json")
	mgr := manager.NewManager(cfgMgr, log, nil)
	server := NewServer(cfgMgr, mgr, auth.NewAuthenticator(), log, nil, "test", "unknown")

	req := httptest.NewRequest("GET", "/api/ready", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Ready  bool              `json:"ready"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Ready {
		t.Fatalf("Expected ready=true")
	}
	if response.Checks["manager"] != "ok" {
		t.Fatalf("Expected manager check ok, got %q", response.Checks["manager"])
	}
	if response.Checks["config"] != "ok" {
		t.Fatalf("Expected config check ok, got %q", response.Checks["config"])
	}
}

func TestHandleMe(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	cfgMgr := config.NewManager("test.json")
	mgr := manager.NewManager(cfgMgr, log, nil)
	authenticator := auth.NewAuthenticator()
	server := NewServer(cfgMgr, mgr, authenticator, log, nil, "test", "unknown")

	token, err := authenticator.CreateSession("1", "admin", "admin", 24*time.Hour, false)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	server.handleMe(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if response["username"] != "admin" {
		t.Fatalf("Expected username=admin, got %v", response["username"])
	}
}

func TestBuildCSRFSecretProductionRequiresEnv(t *testing.T) {
	originalGoEnv := os.Getenv("GO_ENV")
	originalModbridgeEnv := os.Getenv("MODBRIDGE_ENV")
	originalSecret := os.Getenv("MODBRIDGE_CSRF_SECRET")
	defer func() {
		_ = os.Setenv("GO_ENV", originalGoEnv)
		_ = os.Setenv("MODBRIDGE_ENV", originalModbridgeEnv)
		_ = os.Setenv("MODBRIDGE_CSRF_SECRET", originalSecret)
	}()

	_ = os.Setenv("GO_ENV", "production")
	_ = os.Setenv("MODBRIDGE_ENV", "")
	_ = os.Setenv("MODBRIDGE_CSRF_SECRET", "")

	secret, err := buildCSRFSecret()
	if err == nil {
		t.Fatalf("Expected error, got secret=%q", secret)
	}
}

func TestNewServerLoadsCORSOriginsFromConfig(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	cfgMgr := config.NewManager("test.json")
	if err := cfgMgr.Update(func(c *config.Config) error {
		c.CORSAllowedOrigins = []string{"https://example.com"}
		return nil
	}); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	server := NewServer(cfgMgr, manager.NewManager(cfgMgr, log, nil), auth.NewAuthenticator(), log, nil, "test", "unknown")
	if !server.cors.IsOriginAllowed("https://example.com") {
		t.Fatalf("Expected configured origin to be allowed")
	}
}

func TestRoutesWithNilAuthDoesNotPanic(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	cfgMgr := config.NewManager("test.json")
	srv := NewServer(cfgMgr, manager.NewManager(cfgMgr, log, nil), nil, log, nil, "test", "unknown")

	mux := http.NewServeMux()
	srv.Routes(mux)
}

func TestHandleMetricsIncludesProxySnapshotMetrics(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	cfgMgr := config.NewManager("test.json")
	mgr := manager.NewManager(cfgMgr, log, nil)
	srv := NewServer(cfgMgr, mgr, auth.NewAuthenticator(), log, nil, "test", "unknown")

	err = mgr.AddProxy(config.ProxyConfig{
		ID:                "proxy-a",
		Name:              "Proxy A",
		ListenAddr:        ":15020",
		TargetAddr:        "127.0.0.1:502",
		Enabled:           false,
		ConnectionTimeout: 5,
		ReadTimeout:       5,
		MaxRetries:        1,
	}, false)
	if err != nil {
		t.Fatalf("Failed to add proxy: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()
	srv.handleMetrics(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "modbridge_proxy_requests_current") {
		t.Fatalf("Expected proxy requests snapshot metric in output")
	}
	if !strings.Contains(body, `proxy_id="proxy-a"`) {
		t.Fatalf("Expected proxy_id label for proxy-a in output")
	}
}

func TestHandleProxiesRBACDeniesNonAdmin(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	cfgMgr := config.NewManager("test.json")
	mgr := manager.NewManager(cfgMgr, log, nil)
	authenticator := auth.NewAuthenticator()
	// Create session for a "benutzer" role — only proxy:view is granted.
	token, err := authenticator.CreateSession("2", "viewer", "benutzer", 24*time.Hour, false)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	srv := NewServer(cfgMgr, mgr, authenticator, log, nil, "test", "unknown")

	t.Run("POST /api/proxies blocked for viewer", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{}`))
		req := httptest.NewRequest("POST", "/api/proxies", body)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		w := httptest.NewRecorder()
		srv.handleProxies(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("DELETE /api/proxies blocked for viewer", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/proxies?id=foo", nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		w := httptest.NewRecorder()
		srv.handleProxies(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("GET /api/proxies allowed for viewer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/proxies", nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		w := httptest.NewRecorder()
		srv.handleProxies(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}
	})
}

func TestHandleProxiesBlockedWithoutAuth(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	cfgMgr := config.NewManager("test.json")
	mgr := manager.NewManager(cfgMgr, log, nil)
	srv := NewServer(cfgMgr, mgr, auth.NewAuthenticator(), log, nil, "test", "unknown")

	// No cookie — requirePermission returns 401
	req := httptest.NewRequest("GET", "/api/proxies", nil)
	w := httptest.NewRecorder()
	srv.handleProxies(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestHandleProxiesPostOversizedBody(t *testing.T) {
	server, _, token := proxyTestServer(t)

	oversized := bytes.Repeat([]byte("a"), maxJSONBodySize+1)
	req := httptest.NewRequest("POST", "/api/proxies", bytes.NewReader(oversized))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	server.handleProxies(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("Expected status %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
}

func TestChangePasswordInvalidatesSessions(t *testing.T) {
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	cfgMgr := config.NewManager("test.json")
	// Create a bcrypt hash for the current password: "CurrentP@ss1"
	hash, err := auth.HashPassword("CurrentP@ss1")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_ = cfgMgr.Update(func(c *config.Config) error {
		c.AdminPassHash = hash
		// Disable multi-user so we hit the legacy single-user path
		c.MultiUser = false
		return nil
	})

	mgr := manager.NewManager(cfgMgr, log, nil)
	authenticator := auth.NewAuthenticator()
	srv := NewServer(cfgMgr, mgr, authenticator, log, nil, "test", "unknown")

	// Create a session before password change
	token, err := authenticator.CreateSession("1", "admin", "admin", 24*time.Hour, false)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Session is valid now
	if !authenticator.ValidateSession(token) {
		t.Fatal("Expected session to be valid before password change")
	}

	// Change password
	body := `{"current_password":"CurrentP@ss1","new_password":"NewStr0ng!2024"}`
	req := httptest.NewRequest("POST", "/api/config/password", bytes.NewReader([]byte(body)))
	cookie := &http.Cookie{Name: "session_token", Value: token}
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.handleChangePassword(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Password change failed: %d %s", w.Code, w.Body.String())
	}

	// Old session must be invalidated
	if authenticator.ValidateSession(token) {
		t.Fatal("Expected old session to be invalid after password change")
	}
}
