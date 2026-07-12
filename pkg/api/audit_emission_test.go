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
// or nil if no such entry exists. The auditor is drained synchronously via
// Close() before reading.
func findAuditEntry(t *testing.T, server *Server, action string, success bool) *database.AuditLogEntry {
	t.Helper()
	if server.auditor == nil {
		t.Fatal("server.auditor is nil; use auditedTestServer")
	}
	// Drain the async buffer so all emitted entries are persisted.
	server.auditor.Close()
	logs, err := server.auditor.GetLogs(500, 0)
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	for _, e := range logs {
		if e.Action == action && e.Success == success {
			return e
		}
	}
	return nil
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
