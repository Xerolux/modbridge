// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modbridge/pkg/config"
	"modbridge/pkg/database"
)

// proxyConfig is a minimal valid ProxyConfig for audit tests. The proxy is
// created disabled so no real listener is bound.
func proxyConfig(id string) config.ProxyConfig {
	return config.ProxyConfig{
		ID:                id,
		Name:              "test-" + id,
		ListenAddr:        ":15050",
		TargetAddr:        "127.0.0.1:15050",
		ConnectionTimeout: 10,
		ReadTimeout:       30,
		MaxRetries:        3,
		Enabled:           false,
	}
}

// findAuditEntry returns the first audit log entry matching action + success,
// or nil if no such entry exists. It drains the auditor once (idempotent) and
// reads from the DB. NOTE: after the first call, the auditor buffer is closed;
// further emits from handlers will be discarded. For tests that issue multiple
// actions before asserting, collect the logs once via auditEntries and search
// in-memory.
func findAuditEntry(t *testing.T, server *Server, action string, success bool) *database.AuditLogEntry {
	t.Helper()
	for _, e := range auditEntries(t, server) {
		if e.Action == action && e.Success == success {
			return e
		}
	}
	return nil
}

// auditEntries drains the auditor (idempotent) and returns all persisted
// entries. Safe to call multiple times: the first call drains, subsequent
// calls just re-read the DB.
func auditEntries(t *testing.T, server *Server) []*database.AuditLogEntry {
	t.Helper()
	if server.auditor == nil {
		t.Fatal("server.auditor is nil; use auditedTestServer")
	}
	server.auditor.Close()
	logs, err := server.auditor.GetLogs(500, 0)
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	return logs
}

func TestAuditLogin_FailureIsLogged(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()

	body := `{"username":"ghost","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	e := findAuditEntry(t, server, "user.login", false)
	if e == nil {
		t.Fatal("failed login not audited")
	}
	if e.Username != "ghost" {
		t.Errorf("Username = %q, want ghost", e.Username)
	}
}

func TestAuditLogin_SuccessIsLogged(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	// Seed a real user so multi-user login succeeds.
	sessionFor(t, server, "benutzer", "realuser")

	body := `{"username":"realuser","password":"TestPass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	e := findAuditEntry(t, server, "user.login", true)
	if e == nil {
		t.Fatal("successful login not audited")
	}
	if e.Username != "realuser" {
		t.Errorf("Username = %q, want realuser", e.Username)
	}
}

func TestAuditProxy_CreateSuccess(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	pc := proxyConfig("px-1")
	body := proxyConfigJSON(pc)
	req := httptest.NewRequest(http.MethodPost, "/api/proxies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleProxies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	e := findAuditEntry(t, server, "proxy.created", true)
	if e == nil {
		t.Fatal("proxy.created success not audited")
	}
	if e.ResourceID != "px-1" {
		t.Errorf("ResourceID = %q, want px-1", e.ResourceID)
	}
}

func TestAuditProxy_DeleteSuccess(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	// Create first so delete has something to remove.
	if err := server.mgr.AddProxy(proxyConfig("px-del"), true); err != nil {
		t.Fatalf("AddProxy: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies?id=px-del", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleProxies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if findAuditEntry(t, server, "proxy.deleted", true) == nil {
		t.Fatal("proxy.deleted success not audited")
	}
}

// proxyConfigJSON marshals a ProxyConfig to JSON for request bodies.
func proxyConfigJSON(pc config.ProxyConfig) string {
	b, _ := json.Marshal(pc)
	return string(b)
}

func TestAuditProxyControl_StartStopRestartPauseResume(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	if err := server.mgr.AddProxy(proxyConfig("px-ctrl"), true); err != nil {
		t.Fatalf("AddProxy: %v", err)
	}

	actions := []struct {
		act, auditAction string
	}{
		{"start", "proxy.started"},
		{"stop", "proxy.stopped"},
		{"start", "proxy.started"},
		{"restart", "proxy.restarted"},
		{"pause", "proxy.paused"},
		{"resume", "proxy.resumed"},
	}
	// Execute ALL actions before draining the auditor (findAuditEntry closes
	// the auditor on first call, which would discard later emits).
	for _, a := range actions {
		body := `{"id":"px-ctrl","action":"` + a.act + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/proxies/control", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		w := httptest.NewRecorder()
		server.handleProxyControl(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("action %s: status %d, body %s", a.act, w.Code, w.Body.String())
		}
	}
	// Now drain once and check all expected entries are present.
	logs := auditEntries(t, server)
	for _, a := range actions {
		found := false
		for _, e := range logs {
			if e.Action == a.auditAction && e.Success {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("action %s: no %s success entry audited", a.act, a.auditAction)
		}
	}
}

func TestAuditProxyControl_BulkActions(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	// Execute all bulk actions before draining the auditor.
	for _, a := range []string{"start_all", "stop_all", "restart_all"} {
		body := `{"id":"","action":"` + a + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/proxies/control", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		w := httptest.NewRecorder()
		server.handleProxyControl(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d", a, w.Code)
		}
	}
	logs := auditEntries(t, server)
	for _, a := range []string{"start_all", "stop_all", "restart_all"} {
		found := false
		for _, e := range logs {
			if e.Action == "proxy."+a && e.Success {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s not audited", a)
		}
	}
}

func TestAuditConfigImport(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	// Build a validator-passing config via the struct so field names stay
	// correct if the schema changes.
	validCfg := config.Config{
		WebPort:  ":8080",
		Proxies:  []config.ProxyConfig{},
		LogLevel: "INFO", LogMaxSize: 10, LogMaxFiles: 5, LogMaxAgeDays: 30,
		SessionTimeout: 24, MaxConnections: 100,
		CORSAllowedOrigins: []string{"*"},
		CORSAllowedHeaders: []string{"Content-Type"},
		RateLimitEnabled:   true, RateLimitRequests: 100, RateLimitBurst: 100,
	}
	bodyBytes, _ := json.Marshal(validCfg)
	req := httptest.NewRequest(http.MethodPost, "/api/config/import", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleConfigImport(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if findAuditEntry(t, server, "config.imported", true) == nil {
		t.Fatal("config.imported success not audited")
	}
}

func TestAuditSystemRestart(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	// handleSystemRestart writes a response and then triggers restart in a
	// goroutine; we only assert the audit entry is emitted synchronously.
	req := httptest.NewRequest(http.MethodPost, "/api/system/restart", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleSystemRestart(w, req)

	if findAuditEntry(t, server, "system.restart", true) == nil {
		t.Fatal("system.restart not audited")
	}
}

func TestAuditUser_CreateAndDelete(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	// Create
	createBody := `{"username":"victim","full_name":"Victim","email":"v@e.com","password":"Pass123!","role":"benutzer","enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleUsers(w, req)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("create: status %d, body %s", w.Code, w.Body.String())
	}

	target := func() string {
		all, _ := server.userMgr.GetAllUsers()
		for _, u := range all {
			if u.Username == "victim" {
				return u.ID
			}
		}
		t.Fatal("victim user not found after create")
		return ""
	}()

	// Delete
	delReq := httptest.NewRequest(http.MethodDelete, "/api/users/"+target, nil)
	delReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	delW := httptest.NewRecorder()
	server.handleUserByID(delW, delReq)
	if delW.Code != http.StatusNoContent && delW.Code != http.StatusOK {
		t.Fatalf("delete: status %d", delW.Code)
	}

	logs := auditEntries(t, server)
	created := false
	deleted := false
	for _, e := range logs {
		if e.Action == "user.created" && e.Success {
			created = true
		}
		if e.Action == "user.deleted" && e.Success {
			deleted = true
		}
	}
	if !created {
		t.Error("user.created not audited")
	}
	if !deleted {
		t.Error("user.deleted not audited")
	}
}
