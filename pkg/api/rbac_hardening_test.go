// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// These tests verify that endpoints previously guarded only by authMW (session
// check) now enforce the correct RBAC permission.
//
// IMPORTANT: assertions use role/permission pairs that genuinely mismatch per
// the rbac.RolePermissions matrix:
//   - benutzer LACKS: audit:view, audit:export, logs:export, config:export,
//     config:import, system:restart
//   - benutzer HAS: logs:view, system:view, config:view (so those endpoints
//     return 200 for benutzer and are NOT tested for denial here).
//
// This means the hardening closes the audit/exports/restart/import surface
// (which truly was open to any authenticated user before) without changing
// behavior for the read-only views benutzer legitimately has.

// denyWith asserts that a session with the given role gets 403 on the handler.
func denyWith(t *testing.T, server *Server, role, username string, fn func(w http.ResponseWriter, r *http.Request), method, target string) {
	t.Helper()
	token := sessionFor(t, server, role, username)
	req := httptest.NewRequest(method, target, nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	fn(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("%s on %s: got %d, want 403", role, target, w.Code)
	}
}

func TestRBAC_AuditLogs_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "u1", server.handleAuditLogs, http.MethodGet, "/api/audit/logs?limit=10")
}

func TestRBAC_AuditLogsExport_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "u2", server.handleAuditLogsExport, http.MethodGet, "/api/audit/logs/export")
}

func TestRBAC_LogDownload_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "u3", server.handleLogDownload, http.MethodGet, "/api/logs/download")
}

func TestRBAC_ConfigExport_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "u4", server.handleConfigExport, http.MethodGet, "/api/config/export")
}

func TestRBAC_ConfigImport_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "u5", server.handleConfigImport, http.MethodPost, "/api/config/import")
}

func TestRBAC_ConfigSystemEdit_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "config-editor", server.handleSystemConfig, http.MethodPut, "/api/config/system")
}

func TestRBAC_ConfigRollback_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "config-rollback", server.handleConfigRollback, http.MethodPost, "/api/config/rollback")
}

func TestRBAC_WebPortEdit_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "webport-editor", server.handleWebPort, http.MethodPut, "/api/config/webport")
}

func TestRBAC_SystemRestart_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "u6", server.handleSystemRestart, http.MethodPost, "/api/system/restart")
}

func TestRBAC_PortDiagnostics_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "port-diagnostics", server.handlePortDiagnostics, http.MethodPost, "/api/system/ports/diagnostics")
}

func TestRBAC_PortRelease_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "port-release", server.handlePortRelease, http.MethodPost, "/api/system/ports/release")
}

func TestRBAC_UpdateStatus_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	// /api/update/* uses PermSystemRestart internally (not PermSystemView).
	// benutzer lacks system:restart, so all three update endpoints must deny.
	denyWith(t, server, "benutzer", "u7", server.handleUpdateStatus, http.MethodGet, "/api/update/status")
}

func TestRBAC_UpdateCheck_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "u8", server.handleUpdateCheck, http.MethodGet, "/api/update/check")
}

func TestRBAC_UpdatePerform_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	denyWith(t, server, "benutzer", "u9", server.handleUpdatePerform, http.MethodPost, "/api/update/perform")
}

// Positive control: auditor role DOES have audit:view, so it must NOT be denied.
// This catches the inverse bug (over-restrictive permission check).
func TestRBAC_AuditLogs_AuditorAllowed(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "auditor", "audituser")

	req := httptest.NewRequest(http.MethodGet, "/api/audit/logs?limit=10", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleAuditLogs(w, req)

	if w.Code == http.StatusForbidden {
		t.Errorf("auditor on /api/audit/logs: got 403, want non-403 (auditor has audit:view)")
	}
}
