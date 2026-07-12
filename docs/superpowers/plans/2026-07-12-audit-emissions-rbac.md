# Audit-Emissionen + RBAC-Härtung — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire ~30 audit emit points into ~14 handlers and add `requirePermission` checks to ~10 endpoints that currently only check session validity, so the existing audit table becomes useful and the RBAC permission matrix is enforced.

**Architecture:** In-handler checks (no middleware changes). Emit follows the existing `LogLogout` pattern using the 6 existing `Auditor.Log*` methods. A new `requestMeta(r)` helper extracts IP/User-Agent. The only API change in `pkg/audit` is adding a `reason` parameter to `LogLogin` so failed-login reasons are captured.

**Tech Stack:** Go 1.26.4, `net/http`, `httptest`, SQLite via `database.NewDB(":memory:")` for tests.

**Spec:** `docs/superpowers/specs/2026-07-12-audit-emissions-rbac-design.md`

## Global Constraints

- Go 1.26.4, CGO_ENABLED=1 (sqlite3).
- All new exported symbols need doc comments; unexported ones only when logic is non-obvious.
- Receiver name convention: `s *Server`, `a *Auditor`, `m *Manager`.
- Imports grouped: stdlib → external → internal (enforced by gofmt).
- Test naming: `Test<Area><Scenario>` (e.g. `TestAuditLogin_FailureIsLogged`).
- Use `t.Helper()` + `t.Cleanup()` for test setup/teardown.
- Never store the posted admin credentials in code, tests, or commits.
- Frontend is untouched. No `make build-frontend`. Version bump at the end.

---

## File Structure

**Created:**
- `pkg/api/audit_test_helpers_test.go` — shared test helpers: `auditedTestServer(t)` (Server with real `:memory:` DB + auditor + userMgr) and `sessionFor(t, server, role, username)` (creates a user + session for a given role). Used by both test files below.
- `pkg/api/audit_emission_test.go` — ~12 tests asserting that each handler writes the expected audit entry.
- `pkg/api/rbac_hardening_test.go` — ~10 tests asserting that `benutzer` sessions get 403 on hardened endpoints.

**Modified:**
- `pkg/audit/audit.go` — `LogLogin` signature gains `reason string`; `mapActionToEventType` gains 14 new cases.
- `pkg/audit/audit_test.go` — one new test for `LogLogin` with reason.
- `pkg/api/server.go` — new `requestMeta` helper; RBAC in `handleLogs`, `handleLogStream`, `handleProxiesStream`; emit in `handleLogin`/`finalizeLogin`, `handleProxies`, `handleProxyControl`, `handleSystemRestart`.
- `pkg/api/handlers_extra.go` — RBAC in `handleAuditLogs`, `handleAuditLogsExport`, `handleLogDownload`, `handleConfigExport`, `handleSystemInfo`, `handleCheckProxyPorts`; emit in `handleConfigImport`.
- `pkg/api/handlers_update.go` — RBAC in `handleUpdateStatus`; emit in `handleUpdatePerform`.
- `version.txt` — `2.0.8.0` → `2.0.9.0`.

---

## Task 1: `LogLogin` signature + reason threading

**Files:**
- Modify: `pkg/audit/audit.go:659-661` (`LogLogin`)
- Test: `pkg/audit/audit_test.go` (append)

**Interfaces:**
- Produces: `func (a *Auditor) LogLogin(username, ipAddress, userAgent, reason string, success bool)` — used by Task 5 (`handleLogin` emission).

- [ ] **Step 1: Write the failing test**

Append to `pkg/audit/audit_test.go`:

```go
func TestAuditor_LogLogin_WithReason(t *testing.T) {
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	a := NewAuditor(db)
	t.Cleanup(func() { a.Close() })

	a.LogLogin("attacker", "1.2.3.4", "TestAgent/1.0", "invalid credentials", false)
	a.LogLogin("admin", "5.6.7.8", "TestAgent/1.0", "", true)

	// Auditor writes async; flush by closing. Use a fresh auditor to read.
	logs, err := a.GetLogs(100, 0)
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}

	var failEntry, successEntry *database.AuditLogEntry
	for _, e := range logs {
		if e.Action == "user.login" && !e.Success && e.Username == "attacker" {
			failEntry = e
		}
		if e.Action == "user.login" && e.Success && e.Username == "admin" {
			successEntry = e
		}
	}
	if failEntry == nil {
		t.Fatal("failed login not logged")
	}
	if failEntry.ErrorMsg != "invalid credentials" {
		t.Errorf("failed login ErrorMsg = %q, want %q", failEntry.ErrorMsg, "invalid credentials")
	}
	if failEntry.IPAddress != "1.2.3.4" {
		t.Errorf("IPAddress = %q, want 1.2.3.4", failEntry.IPAddress)
	}
	if successEntry == nil {
		t.Fatal("successful login not logged")
	}
	if successEntry.ErrorMsg != "" {
		t.Errorf("success ErrorMsg = %q, want empty", successEntry.ErrorMsg)
	}
}
```

Ensure `"modbridge/pkg/database"` is imported in the test file if not already.

- [ ] **Step 2: Run test to verify it fails (compile error)**

Run: `go test ./pkg/audit/... -run TestAuditor_LogLogin_WithReason -count=1`
Expected: FAIL — `LogLogin` called with too many args (current signature has 4 params, test passes 5).

- [ ] **Step 3: Change `LogLogin` signature to accept reason**

In `pkg/audit/audit.go`, replace the `LogLogin` method:

```go
// LogLogin logs a login attempt. reason is captured on failure (e.g.
// "invalid credentials", "user disabled"); pass "" on success.
func (a *Auditor) LogLogin(username, ipAddress, userAgent, reason string, success bool) {
	errMsg := ""
	if !success {
		errMsg = reason
	}
	a.LogAction("user.login", "user", username, "", username, "", ipAddress, userAgent, success, errMsg)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/audit/... -run TestAuditor_LogLogin_WithReason -count=1 -race`
Expected: PASS.

- [ ] **Step 5: Confirm no other callers exist (whole-repo build)**

Run: `go build ./...`
Expected: success (no other caller of `LogLogin` exists in production code; only the not-yet-modified login handler would call it, and it does not yet).

- [ ] **Step 6: Commit**

```bash
git add pkg/audit/audit.go pkg/audit/audit_test.go
git commit -m "feat(audit): LogLogin accepts reason for failed-login capture"
```

---

## Task 2: Extend `mapActionToEventType` with all new actions

**Files:**
- Modify: `pkg/audit/audit.go:639-656` (`mapActionToEventType`)
- Test: `pkg/audit/audit_test.go` (append)

**Interfaces:**
- Produces: complete `mapActionToEventType` covering all 14 new action strings, so `FileAuditLogger` (if ever enabled) maps them to typed `EventType` constants.

- [ ] **Step 1: Write the failing test**

Append to `pkg/audit/audit_test.go`:

```go
func TestMapActionToEventType_AllActions(t *testing.T) {
	cases := []struct {
		action string
		want   EventType
	}{
		{"user.login", EventAuthLogin},
		{"user.logout", EventAuthLogout},
		{"proxy.created", EventProxyCreated},
		{"proxy.updated", EventProxyUpdated},
		{"proxy.deleted", EventProxyDeleted},
		{"proxy.started", EventProxyStarted},
		{"proxy.stopped", EventProxyStopped},
		{"proxy.restarted", EventProxyRestarted},
		{"proxy.paused", EventProxyPaused},
		{"proxy.resumed", EventProxyResumed},
		{"config.updated", EventConfigUpdated},
		{"config.imported", EventConfigImported},
		{"user.created", EventUserCreated},
		{"user.updated", EventUserUpdated},
		{"user.deleted", EventUserDeleted},
	}
	for _, c := range cases {
		t.Run(c.action, func(t *testing.T) {
			got := mapActionToEventType(c.action)
			if got != c.want {
				t.Errorf("mapActionToEventType(%q) = %v, want %v", c.action, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/audit/... -run TestMapActionToEventType -count=1`
Expected: FAIL for unmapped actions (e.g. `proxy.updated` returns `EventType("proxy.updated")` instead of `EventProxyUpdated`).

- [ ] **Step 3: Verify the `EventType` constants exist**

Run: `grep -n "EventProxy\|EventConfigImported\|EventUser" pkg/audit/audit.go | head -30`

If any constant referenced in the test (e.g. `EventProxyRestarted`, `EventProxyPaused`, `EventConfigImported`, `EventUserCreated`) does not exist in the constants block (around audit.go:21-62), ADD it in the same block:

```go
EventProxyRestarted EventType = "proxy.restarted"
EventProxyPaused    EventType = "proxy.paused"
EventProxyResumed   EventType = "proxy.resumed"
EventConfigImported EventType = "config.imported"
EventUserCreated    EventType = "user.created"
EventUserUpdated    EventType = "user.updated"
EventUserDeleted    EventType = "user.deleted"
```

Use `grep` output to determine exactly which are missing — do not duplicate existing constants.

- [ ] **Step 4: Extend the switch in `mapActionToEventType`**

In `pkg/audit/audit.go`, replace the `mapActionToEventType` switch body to cover all 15 cases:

```go
// mapActionToEventType maps action strings to EventTypes
func mapActionToEventType(action string) EventType {
	switch action {
	case "user.login":
		return EventAuthLogin
	case "user.logout":
		return EventAuthLogout
	case "proxy.created":
		return EventProxyCreated
	case "proxy.updated":
		return EventProxyUpdated
	case "proxy.deleted":
		return EventProxyDeleted
	case "proxy.started":
		return EventProxyStarted
	case "proxy.stopped":
		return EventProxyStopped
	case "proxy.restarted":
		return EventProxyRestarted
	case "proxy.paused":
		return EventProxyPaused
	case "proxy.resumed":
		return EventProxyResumed
	case "config.updated":
		return EventConfigUpdated
	case "config.imported":
		return EventConfigImported
	case "user.created":
		return EventUserCreated
	case "user.updated":
		return EventUserUpdated
	case "user.deleted":
		return EventUserDeleted
	default:
		return EventType(action)
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./pkg/audit/... -run TestMapActionToEventType -count=1 -race`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/audit/audit.go pkg/audit/audit_test.go
git commit -m "feat(audit): map all new action strings to EventType constants"
```

---

## Task 3: `requestMeta` helper in `pkg/api/server.go`

**Files:**
- Modify: `pkg/api/server.go` (add helper after `requirePermission`, ~line 86)
- Test: none (trivial pure function; covered indirectly by emission tests)

**Interfaces:**
- Produces: `func requestMeta(r *http.Request) (ip, userAgent string)` — used by all emit points in Tasks 5–9.

- [ ] **Step 1: Confirm `net` and `strings` are already imported**

Run: `head -32 pkg/api/server.go | grep -E '"net"|"strings"'`
Expected: both `"net"` and `"strings"` appear in the import block (they do, per earlier exploration).

- [ ] **Step 2: Add the helper**

In `pkg/api/server.go`, immediately after the closing brace of `requirePermission` (line 86), insert:

```go
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
```

- [ ] **Step 3: Build to verify compilation**

Run: `go build ./pkg/api/...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add pkg/api/server.go
git commit -m "feat(api): requestMeta helper for audit IP/User-Agent extraction"
```

---

## Task 4: Shared audit test helpers

**Files:**
- Create: `pkg/api/audit_test_helpers_test.go`
- Test: this IS the test file

**Interfaces:**
- Produces:
  - `func auditedTestServer(t *testing.T) (*Server, func())` — Server with `:memory:` DB, real auditor, real userMgr. Returns a cleanup func.
  - `func sessionFor(t *testing.T, server *Server, role, username string) string` — creates a DB user with the given role + a session; returns the session token.

- [ ] **Step 1: Create the helper file**

Create `pkg/api/audit_test_helpers_test.go`:

```go
package api

import (
	"testing"
	"time"

	"modbridge/pkg/auth"
	"modbridge/pkg/config"
	"modbridge/pkg/database"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
	"modbridge/pkg/users"
)

// auditedTestServer builds a Server backed by an in-memory SQLite DB so that
// auditor.GetLogs() works and userMgr.CreateUser succeeds. Returns the server
// and a cleanup func that closes the auditor + DB + log.
func auditedTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	cfgMgr := config.NewManager("test.json")
	mgr := manager.NewManager(cfgMgr, log, db)
	authenticator := auth.NewAuthenticator()
	server := NewServer(cfgMgr, mgr, authenticator, log, db, "test", "unknown")

	cleanup := func() {
		if server.auditor != nil {
			server.auditor.Close()
		}
		db.Close()
		log.Close()
	}
	return server, cleanup
}

// sessionFor creates a DB-backed user with the given role and returns a valid
// session token for that user. Used to drive RBAC tests with non-admin roles.
func sessionFor(t *testing.T, server *Server, role, username string) string {
	t.Helper()
	if server.userMgr == nil {
		t.Fatal("server.userMgr is nil; use auditedTestServer not proxyTestServer")
	}
	user, err := server.userMgr.CreateUser(&users.CreateUserRequest{
		Username: username,
		FullName: username,
		Email:    username + "@example.com",
		Password: "TestPass123!",
		Role:     role,
		Enabled:  true,
	}, "test")
	if err != nil {
		t.Fatalf("CreateUser(%s, %s): %v", username, role, err)
	}
	token, err := server.auth.CreateSession(user.ID, user.Username, user.Role, 24*time.Hour, false)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return token
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go test ./pkg/api/... -run TestNone -count=1`
Expected: compiles cleanly (no test named TestNone runs, but build succeeds).

- [ ] **Step 3: Commit**

```bash
git add pkg/api/audit_test_helpers_test.go
git commit -m "test(api): shared auditedTestServer + sessionFor helpers"
```

---

## Task 5: Audit `handleLogin` + `finalizeLogin` (success + all failure paths)

**Files:**
- Modify: `pkg/api/server.go:545-644` (`handleLogin`, `finalizeLogin`)
- Test: `pkg/api/audit_emission_test.go` (create file)

**Interfaces:**
- Consumes: `LogLogin(username, ip, ua, reason string, success bool)` (Task 1), `requestMeta(r)` (Task 3).

- [ ] **Step 1: Write failing tests**

Create `pkg/api/audit_emission_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// findAuditEntry returns the first log entry matching action+success, or nil.
func findAuditEntry(t *testing.T, server *Server, action string, success bool) *database.AuditLogEntry {
	t.Helper()
	logs, err := server.auditor.GetLogs(200, 0)
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
```

Add `"modbridge/pkg/database"` to imports if `findAuditEntry` references it (it does via the return type — confirm by running `go vet`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/... -run TestAuditLogin -count=1`
Expected: FAIL — no `user.login` entries because handler does not emit.

- [ ] **Step 3: Add emit to `handleLogin` failure paths**

In `pkg/api/server.go`, modify `handleLogin`. In the multi-user branch, replace the failure block:

```go
user, err := s.userMgr.AuthenticateUser(strings.TrimSpace(req.Username), req.Password)
if err != nil || user == nil {
	ip, ua := requestMeta(r)
	if s.auditor != nil {
		s.auditor.LogLogin(req.Username, ip, ua, "invalid credentials", false)
	}
	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	return
}
```

In the legacy single-user branch, replace the failure block:

```go
if !auth.CheckPasswordHash(req.Password, cfg.AdminPassHash) {
	ip, ua := requestMeta(r)
	if s.auditor != nil {
		s.auditor.LogLogin("admin", ip, ua, "invalid password", false)
	}
	http.Error(w, "Invalid password", http.StatusUnauthorized)
	return
}
```

- [ ] **Step 4: Add emit to `finalizeLogin` (success path)**

In `finalizeLogin`, after the successful `CreateSession` call and before writing cookies, insert:

```go
// Audit successful login. We log here (not in handleLogin) because both
// multi-user and legacy paths funnel through finalizeLogin.
ip, ua := requestMeta(r)
if s.auditor != nil {
	s.auditor.LogLogin(username, ip, ua, "", true)
}
```

Place this after `token, err := s.auth.CreateSession(...)` succeeds (after the error check that returns 500), and before `http.SetCookie`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/api/... -run TestAuditLogin -count=1 -race`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/api/server.go pkg/api/audit_emission_test.go
git commit -m "feat(api): audit login success + failure (with reason + IP)"
```

---

## Task 6: Audit `handleProxies` (POST/PUT/DELETE)

**Files:**
- Modify: `pkg/api/server.go:1045-1149` (`handleProxies`)
- Test: `pkg/api/audit_emission_test.go` (append)

**Interfaces:**
- Consumes: `(*auth.Session) from requirePermission` provides `UserID`/`Username`. `LogProxyAction(action, proxyID, userID, username, details, ip, ua string, success bool)`.

- [ ] **Step 1: Write failing tests**

Append to `pkg/api/audit_emission_test.go`:

```go
func TestAuditProxy_CreateSuccess(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	body := `{"id":"px-1","name":"Pump A","listen_addr":":15100","target_addr":"127.0.0.1:15050","enabled":false}`
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
	server.mgr.AddProxy(proxyConfig("px-del"), true)

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
```

Add a small helper `proxyConfig` to the test file (or to `audit_test_helpers_test.go`):

```go
// proxyConfig is a minimal valid ProxyConfig for tests.
func proxyConfig(id string) config.ProxyConfig {
	return config.ProxyConfig{
		ID: id, Name: "test-" + id,
		ListenAddr: ":0", TargetAddr: "127.0.0.1:15050",
		Timeout: 30, Retries: 3, Enabled: false,
	}
}
```

Add `"modbridge/pkg/config"` to the test file imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/... -run TestAuditProxy -count=1`
Expected: FAIL — no `proxy.created`/`proxy.deleted` entries.

- [ ] **Step 3: Capture session at top of mutation methods + emit**

In `handleProxies`, the existing `s.requirePermission(w, r, permission)` call returns `*auth.Session`. Change it to capture:

```go
session := s.requirePermission(w, r, permission)
if session == nil {
	return
}
ip, ua := requestMeta(r)
```

Then in each mutation branch, emit. For POST (after `AddProxy` success and after the optional start):

```go
if s.auditor != nil {
	s.auditor.LogProxyAction("proxy.created", req.ID, session.UserID, session.Username, req.Name, ip, ua, true)
}
w.WriteHeader(http.StatusOK)
return
```

For POST failure paths (validation error, AddProxy error), insert before each `http.Error(...)`:

```go
if s.auditor != nil {
	s.auditor.LogProxyAction("proxy.created", req.ID, session.UserID, session.Username, req.Name, ip, ua, false)
}
```

For PUT success (after `UpdateProxy`):

```go
if s.auditor != nil {
	s.auditor.LogProxyAction("proxy.updated", req.ID, session.UserID, session.Username, req.Name, ip, ua, true)
}
```

For DELETE success (after `RemoveProxy`):

```go
if s.auditor != nil {
	s.auditor.LogProxyAction("proxy.deleted", id, session.UserID, session.Username, id, ip, ua, true)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/api/... -run TestAuditProxy -count=1 -race`
Expected: PASS.

- [ ] **Step 5: Confirm existing proxy tests still pass**

Run: `go test ./pkg/api/... -run TestHandleProxies -count=1`
Expected: PASS (admin token, behavior unchanged).

- [ ] **Step 6: Commit**

```bash
git add pkg/api/server.go pkg/api/audit_emission_test.go
git commit -m "feat(api): audit proxy create/update/delete with success+failure"
```

---

## Task 7: Audit `handleProxyControl` (start/stop/restart/pause/resume + bulk)

**Files:**
- Modify: `pkg/api/server.go:1151-1203` (`handleProxyControl`)
- Test: `pkg/api/audit_emission_test.go` (append)

**Interfaces:**
- Consumes: `LogProxyAction` for single-proxy actions, `LogAction` for bulk actions (no single proxy ID).

- [ ] **Step 1: Write failing test**

Append to `pkg/api/audit_emission_test.go`:

```go
func TestAuditProxyControl_StartStopRestart(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	server.mgr.AddProxy(proxyConfig("px-ctrl"), true)

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
		if findAuditEntry(t, server, a.auditAction, true) == nil {
			t.Errorf("action %s: no %s success entry audited", a.act, a.auditAction)
		}
	}
}

func TestAuditProxyControl_BulkActions(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

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
		if findAuditEntry(t, server, "proxy."+a, true) == nil {
			t.Errorf("%s not audited", a)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/... -run TestAuditProxyControl -count=1`
Expected: FAIL.

- [ ] **Step 3: Add emit to `handleProxyControl`**

In `handleProxyControl`, capture session + meta after the permission check:

```go
session := s.requirePermission(w, r, rbac.PermProxyControl)
if session == nil {
	return
}
ip, ua := requestMeta(r)
```

After the `switch req.Action` block, after computing `err`, replace the final error/success handling:

```go
if err != nil {
	if s.auditor != nil {
		s.auditor.LogProxyAction(actionForAudit(req.Action), req.ID, session.UserID, session.Username, "", ip, ua, false)
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
// Success
if s.auditor != nil {
	auditAction := actionForAudit(req.Action)
	if isBulkAction(req.Action) {
		s.auditor.LogAction("proxy."+req.Action, "proxy", "", session.UserID, session.Username, "", ip, ua, true, "")
	} else {
		s.auditor.LogProxyAction(auditAction, req.ID, session.UserID, session.Username, "", ip, ua, true)
	}
}
w.WriteHeader(http.StatusOK)
```

Add two helpers near `requestMeta`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/api/... -run TestAuditProxyControl -count=1 -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/api/server.go pkg/api/audit_emission_test.go
git commit -m "feat(api): audit proxy control (start/stop/restart/pause/resume + bulk)"
```

---

## Task 8: Audit `handleConfigImport`, `handleSystemRestart`, `handleUpdatePerform`

**Files:**
- Modify: `pkg/api/handlers_extra.go` (`handleConfigImport`)
- Modify: `pkg/api/server.go` (`handleSystemRestart`)
- Modify: `pkg/api/handlers_update.go` (`handleUpdatePerform`)
- Test: `pkg/api/audit_emission_test.go` (append)

**Interfaces:**
- Consumes: `LogConfigChange`, `LogAction`.

- [ ] **Step 1: Write failing tests**

Append to `pkg/api/audit_emission_test.go`:

```go
func TestAuditConfigImport(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	body := `{"proxies":[],"web_port":":8080"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleConfigImport(w, req)

	if findAuditEntry(t, server, "config.imported", true) == nil {
		t.Fatal("config.imported success not audited")
	}
}

func TestAuditSystemRestart(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	// handleSystemRestart triggers s.triggerRestart() in a goroutine; the test
	// only checks the audit entry is written before the restart signal fires.
	req := httptest.NewRequest(http.MethodPost, "/api/system/restart", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleSystemRestart(w, req)

	if findAuditEntry(t, server, "system.restart", true) == nil {
		t.Fatal("system.restart not audited")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/... -run "TestAuditConfigImport|TestAuditSystemRestart" -count=1`
Expected: FAIL.

- [ ] **Step 3: Emit in `handleConfigImport`**

Read `pkg/api/handlers_extra.go:102-139` first to see the exact structure, then add at the top of the handler (after the existing permission/method checks):

```go
session := s.requirePermission(w, r, rbac.PermConfigImport)
if session == nil {
	return
}
ip, ua := requestMeta(r)
```

If the handler does not yet call `requirePermission`, add it. Then at success and failure points (before/after the config apply), emit:

```go
// on success
if s.auditor != nil {
	s.auditor.LogConfigChange("config.imported", session.UserID, session.Username, "", ip, ua, true)
}
// on failure (each error return)
if s.auditor != nil {
	s.auditor.LogConfigChange("config.imported", session.UserID, session.Username, err.Error(), ip, ua, false)
}
```

- [ ] **Step 4: Emit in `handleSystemRestart`**

In `handleSystemRestart` (server.go:1267-1289), after the permission check captures the session and before the `go func() { ... triggerRestart() }`, add:

```go
ip, ua := requestMeta(r)
if s.auditor != nil {
	s.auditor.LogAction("system.restart", "system", "", session.UserID, session.Username, "", ip, ua, true, "")
}
```

If `handleSystemRestart` does not yet capture the session from `requirePermission`, capture it:

```go
session := s.requirePermission(w, r, rbac.PermSystemRestart)
if session == nil {
	return
}
```

- [ ] **Step 5: Emit in `handleUpdatePerform`**

In `handleUpdatePerform` (handlers_update.go:84-121), after the existing `requirePermission` already captures nothing — change it to capture:

```go
session := s.requirePermission(w, r, rbac.PermSystemRestart)
if session == nil {
	return
}
ip, ua := requestMeta(r)
```

Then after `s.updater.PerformUpdate(r.Context())` succeeds:

```go
if s.auditor != nil {
	s.auditor.LogAction("system.update", "system", "", session.UserID, session.Username, "update started", ip, ua, true, "")
}
```

And in the error branch:

```go
if s.auditor != nil {
	s.auditor.LogAction("system.update", "system", "", session.UserID, session.Username, "", ip, ua, false, err.Error())
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./pkg/api/... -run "TestAuditConfigImport|TestAuditSystemRestart" -count=1 -race`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/api/handlers_extra.go pkg/api/server.go pkg/api/handlers_update.go pkg/api/audit_emission_test.go
git commit -m "feat(api): audit config import, system restart, update perform"
```

---

## Task 9: Audit `handleUsers` + `handleUserByID` (create/update/delete)

**Files:**
- Modify: `pkg/api/server.go:798-986` (`handleUsers`, `handleUserByID`)
- Test: `pkg/api/audit_emission_test.go` (append)

**Interfaces:**
- Consumes: `LogUserAction(action, targetUserID, userID, username, details, ip, ua string, success bool)`.

- [ ] **Step 1: Write failing test**

Append to `pkg/api/audit_emission_test.go`:

```go
func TestAuditUser_CreateAndDelete(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "admin", "adminuser")

	// Create
	body := `{"username":"victim","full_name":"Victim","email":"v@e.com","password":"Pass123!","role":"benutzer","enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleUsers(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", w.Code, w.Body.String())
	}
	if findAuditEntry(t, server, "user.created", true) == nil {
		t.Error("user.created not audited")
	}

	// Find the created user's ID via the list endpoint
	listReq := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	listReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	listW := httptest.NewRecorder()
	server.handleUsers(listW, listReq)

	// Delete by username lookup
	target, _ := server.userMgr.GetUserByUsername("victim")
	if target == nil {
		t.Fatal("victim user not found after create")
	}
	delReq := httptest.NewRequest(http.MethodDelete, "/api/users/"+target.ID, nil)
	delReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	delW := httptest.NewRecorder()
	server.handleUserByID(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("delete: status %d", delW.Code)
	}
	if findAuditEntry(t, server, "user.deleted", true) == nil {
		t.Error("user.deleted not audited")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/api/... -run TestAuditUser -count=1`
Expected: FAIL.

- [ ] **Step 3: Emit in `handleUsers` (POST) and `handleUserByID` (PUT/DELETE)**

In `handleUsers`, the existing `requirePermissionForUserRoute` returns `(*auth.Session, bool)`. Capture it. After the POST `CreateUser` success:

```go
if s.auditor != nil {
	s.auditor.LogUserAction("user.created", createdUser.ID, session.UserID, session.Username, "created user "+createdUser.Username, ip, ua, true)
}
```

In `handleUserByID`, after PUT `UpdateUser` success:

```go
if s.auditor != nil {
	s.auditor.LogUserAction("user.updated", id, session.UserID, session.Username, "", ip, ua, true)
}
```

After DELETE `DeleteUser` success:

```go
if s.auditor != nil {
	s.auditor.LogUserAction("user.deleted", id, session.UserID, session.Username, "", ip, ua, true)
}
```

Where `id` is the path parameter (extract via `r.URL.Path` stripping `/api/users/` prefix, matching existing code). `ip, ua := requestMeta(r)` at the top of each handler after the session is captured. Add failure-side emits before each `http.Error` in the mutation branches.

**Note:** Because `requirePermissionForUserRoute` has its own signature, read its current return shape and capture the session accordingly. If it does not return the session, refactor it to do so (small change, but required for the UserID).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/api/... -run TestAuditUser -count=1 -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/api/server.go pkg/api/audit_emission_test.go
git commit -m "feat(api): audit user create/update/delete"
```

---

## Task 10: RBAC hardening — audit + logs endpoints

**Files:**
- Modify: `pkg/api/handlers_extra.go` (`handleAuditLogs`, `handleAuditLogsExport`, `handleLogDownload`, `handleConfigExport`, `handleSystemInfo`, `handleCheckProxyPorts`)
- Modify: `pkg/api/server.go` (`handleLogs`, `handleLogStream`, `handleProxiesStream`)
- Test: `pkg/api/rbac_hardening_test.go` (create)

**Interfaces:**
- Consumes: `rbac.PermAuditView`, `PermAuditExport`, `PermLogsView`, `PermLogsExport`, `PermConfigExport`, `PermSystemView`, `PermProxyView`.

- [ ] **Step 1: Write failing tests**

Create `pkg/api/rbac_hardening_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRBAC_AuditLogs_BenutzerDenied verifies a benutzer (lacks audit:view)
// gets 403 on /api/audit/logs.
func TestRBAC_AuditLogs_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "benutzer", "regularuser")

	req := httptest.NewRequest(http.MethodGet, "/api/audit/logs?limit=10", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleAuditLogs(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("benutzer on /api/audit/logs: got %d, want 403", w.Code)
	}
}

func TestRBAC_AuditLogs_AuditorAllowed(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "auditor", "audituser")

	req := httptest.NewRequest(http.MethodGet, "/api/audit/logs?limit=10", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleAuditLogs(w, req)

	if w.Code == http.StatusForbidden {
		t.Errorf("auditor on /api/audit/logs: got 403, want non-403")
	}
}

func TestRBAC_Logs_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "benutzer", "regularuser")

	req := httptest.NewRequest(http.MethodGet, "/api/logs", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleLogs(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("benutzer on /api/logs: got %d, want 403", w.Code)
	}
}

func TestRBAC_ConfigExport_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "benutzer", "regularuser")

	req := httptest.NewRequest(http.MethodGet, "/api/config/export", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleConfigExport(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("benutzer on /api/config/export: got %d, want 403", w.Code)
	}
}

func TestRBAC_SystemInfo_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "benutzer", "regularuser")

	req := httptest.NewRequest(http.MethodGet, "/api/system/info", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleSystemInfo(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("benutzer on /api/system/info: got %d, want 403", w.Code)
	}
}

func TestRBAC_UpdateStatus_BenutzerDenied(t *testing.T) {
	server, cleanup := auditedTestServer(t)
	defer cleanup()
	token := sessionFor(t, server, "benutzer", "regularuser")

	req := httptest.NewRequest(http.MethodGet, "/api/update/status", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w := httptest.NewRecorder()
	server.handleUpdateStatus(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("benutzer on /api/update/status: got %d, want 403", w.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/... -run TestRBAC_ -count=1`
Expected: FAIL — handlers return 200/other (no permission check), not 403.

- [ ] **Step 3: Add `requirePermission` to each hardened handler**

At the top of each handler (after the method check if one exists, before any business logic), insert the matching permission check. Pattern:

```go
if s.requirePermission(w, r, rbac.PermXxxYyy) == nil {
	return
}
```

Specific mappings (insert as the first statement of each handler body):

- `handleAuditLogs` (handlers_extra.go) → `rbac.PermAuditView`
- `handleAuditLogsExport` → `rbac.PermAuditExport`
- `handleLogDownload` → `rbac.PermLogsExport`
- `handleLogs` (server.go) → `rbac.PermLogsView`
- `handleLogStream` → `rbac.PermLogsView`
- `handleConfigExport` → `rbac.PermConfigExport`
- `handleSystemInfo` → `rbac.PermSystemView`
- `handleCheckProxyPorts` → `rbac.PermSystemView`
- `handleUpdateStatus` (handlers_update.go) → `rbac.PermSystemView`
- `handleProxiesStream` (server.go) → `rbac.PermProxyView`

Ensure `"modbridge/pkg/rbac"` is imported in each file (it already is in server.go and handlers_update.go; verify handlers_extra.go).

**Note on SSE handlers** (`handleLogStream`, `handleProxiesStream`): the permission check MUST be the very first statement, before any `w.Write`/`w.Flush`. A 403 after headers are sent is impossible.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/api/... -run TestRBAC_ -count=1 -race`
Expected: PASS.

- [ ] **Step 5: Confirm existing tests still pass (admin tokens unaffected)**

Run: `go test ./pkg/api/... -count=1`
Expected: PASS (admin has all permissions).

- [ ] **Step 6: Commit**

```bash
git add pkg/api/handlers_extra.go pkg/api/server.go pkg/api/handlers_update.go pkg/api/rbac_hardening_test.go
git commit -m "fix(api): enforce RBAC permissions on audit/logs/config/system endpoints"
```

---

## Task 11: Version bump + final verification + merge

**Files:**
- Modify: `version.txt`

- [ ] **Step 1: Bump version**

Set `version.txt` to `2.0.9.0`.

- [ ] **Step 2: Full verification**

Run each in sequence; all must pass:

```bash
gofmt -l pkg/ cmd/ main.go web.go        # must print nothing
go vet ./...                              # must print nothing
CGO_ENABLED=1 go build ./...             # must succeed
go test ./pkg/audit/... -count=1 -race   # must PASS
go test ./pkg/api/... -count=1 -race     # must PASS
go test ./... -count=1                    # full suite must PASS
```

- [ ] **Step 3: Commit version bump**

```bash
git add version.txt
git commit -m "chore: bump version to 2.0.9.0 [skip ci]"
```

- [ ] **Step 4: Merge to main + push**

```bash
git checkout main
git merge --ff-only spec/audit-emissions-rbac
git branch -d spec/audit-emissions-rbac
git push origin main
```

If the push is rejected (remote has new commits), fetch, merge `origin/main` with `-X ours` for `version.txt` only if conflicting (our version is newer), commit, push.

---

## Self-Review Checklist (executed after writing this plan)

- [x] **Spec coverage:** Every section of the spec maps to a task:
  - Spec §3.2 (`requestMeta`) → Task 3
  - Spec §3.3 (`LogLogin` reason) → Task 1
  - Spec §4.1 RBAC table → Task 10
  - Spec §4.2 emit table → Tasks 5, 6, 7, 8, 9
  - Spec §4.3 event-type mapping → Task 2
  - Spec §5 test strategy → Tasks 1, 4, 5–10
  - Spec §6 file overview → matches File Structure above
  - Spec §7 build sequence → Tasks 1–11 in order
- [x] **Placeholder scan:** no TBD/TODO/"add appropriate" — every code step shows real code.
- [x] **Type consistency:** `LogLogin(username, ip, ua, reason, success)` signature used consistently in Task 1 (definition) and Task 5 (call sites). `requestMeta(r) → (ip, userAgent string)` consistent in Task 3 and all emit tasks. `sessionFor(t, server, role, username) → token` consistent in Task 4 and all RBAC tests.
- [x] **Risk flagged:** `requirePermissionForUserRoute` signature may need a small refactor in Task 9 to return the session — flagged in-step.
