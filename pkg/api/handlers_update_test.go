// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"modbridge/pkg/auth"
	"modbridge/pkg/config"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
)

// updateTestServer builds a minimal Server wired with an admin session, mirroring
// the proxyTestServer helper in server_test.go. Returns the server and a valid
// admin session token for authenticated requests.
func updateTestServer(t *testing.T) (*Server, string) {
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
	return server, token
}

// TestTriggerRestart_Idempotent verifies the sync.Once guard (Fix 2): calling
// triggerRestart more than once must not panic and must close the channel
// exactly once. Without the guard, the second close crashes the process.
func TestTriggerRestart_Idempotent(t *testing.T) {
	server, _ := updateTestServer(t)

	// First call closes the channel.
	server.triggerRestart()
	select {
	case <-server.RestartSignal():
		// expected: channel is now closed
	default:
		t.Fatal("restart signal not closed after first triggerRestart")
	}

	// Second call must be a no-op, NOT a panic. This is the regression guard.
	server.triggerRestart()

	// Channel must remain closed (receive succeeds immediately).
	select {
	case <-server.RestartSignal():
	default:
		t.Fatal("restart signal should still be closed after second triggerRestart")
	}
}

// TestHandleUpdateCheck_ErrorReturnsBadGateway verifies Fix 5: when the GitHub
// upstream is unreachable, the handler must return 502 Bad Gateway (surfacing
// the outage) rather than HTTP 200 with UpdateAvailable:false (which masks the
// outage as "no update available").
func TestHandleUpdateCheck_ErrorReturnsBadGateway(t *testing.T) {
	server, token := updateTestServer(t)
	// Point the updater at a closed server so CheckForUpdate fails fast.
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	dead.Close() // shut down immediately → connection refused
	server.updater.SetAPIBase(dead.URL)

	req := httptest.NewRequest(http.MethodGet, "/api/update/check", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	server.handleUpdateCheck(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 Bad Gateway on upstream failure, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestHandleUpdateCheck_Success verifies the happy path still returns 200 with
// the expected fields after the Fix 5 error-handling change.
func TestHandleUpdateCheck_Success(t *testing.T) {
	server, token := updateTestServer(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v9.9.9.9","prerelease":false,"assets":[{"name":"modbridge-linux-amd64","browser_download_url":"http://example.com/a","size":100}]}`))
	}))
	defer srv.Close()
	server.updater.SetAPIBase(srv.URL)

	req := httptest.NewRequest(http.MethodGet, "/api/update/check", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()

	server.handleUpdateCheck(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on successful check, got %d (body: %s)", w.Code, w.Body.String())
	}
}
