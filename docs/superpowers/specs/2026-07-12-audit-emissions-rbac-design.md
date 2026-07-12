# Design Spec: Audit-Emissionen + RBAC-Härtung

**Datum:** 2026-07-12
**Status:** Approved (brainstormed)
**Priorität:** 1 von 4 (Audit/Users/Headless/CI/Deps)
**Version nach Implementierung:** 2.0.9.0

---

## 1. Problemstellung

Die Audit- und RBAC-Infrastruktur existiert vollständig (Viewer, Export, Filter,
DB-Schema, 25 Event-Typ-Konstanten, 6 `Auditor.Log*`-Methoden, 4 Rollen, 24
Permissions) — ist aber in der Praxis **nahezu wirkungslos**, weil:

1. **Audit wird fast nie geschrieben.** Nur 1 Emit-Punkt existiert (`LogLogout` in
   `handleLogout`). Login (insbesondere fehlgeschlagene), Proxy-CRUD,
   Config-Änderungen, User-Änderungen, Restart, Update erzeugen **keine**
   Audit-Einträge. Die Audit-Tabelle bleibt praktisch leer; ein Brute-Force-Angriff
   wäre unsichtbar.

2. **RBAC-Lücke.** ~10 Endpunkte sind nur hinter `authMW` (Session-Check)
   registriert, nicht hinter `requirePermission(perm)`. Ein `benutzer` (der laut
   RBAC-Matrix `audit:view` nicht hat) kann `/api/audit/logs` dennoch lesen. Gleiches
   gilt für System-Info, Logs, Config-Export, Port-Checks, Update-Status,
   Proxy-Stream.

Dieses Spec behebt beide Punkte. Es ist das erste von vier dekomponierten
Projekten; Headless/CI-Workflow, Headless-Erweiterung und Dependency-Updates
folgen in separaten Cycles.

## 2. Ziele / Nicht-Ziele

### Ziele
- **Vollständige Audit-Abdeckung** für alle sicherheits- und mutationskritischen
  Handler (~14 Handler, ~25 Emit-Punkte).
- **Fehlgeschlagene Logins werden protokolliert** mit Username + IP + User-Agent
  (Standard für Security-Audit-Logs).
- **RBAC-Lücke schließen**: ~10 Endpunkte erhalten konsequente
  `requirePermission`-Checks gemäß der existierenden Permission-Matrix.
- **Bestehende Architektur nutzen**: keine neuen Auditor-Methoden, keine
  Schema-Migration. Emit folgt dem etablierten `LogLogout`-Muster.
- **TDD**: pro Emit-Punkt und pro RBAC-gehärtetem Endpunkt ein Test.

### Nicht-Ziele (YAGNI)
- **`FileAuditLogger`-Anbindung**: gebaut aber ungenutzt (Dead Code). Separate
  Entscheidung. Nicht Teil dieser Spec.
- **OpenAPI-Doku** für Audit/Users-Endpunkte: unabhängig.
- **Audit-Rotation/Aufbewahrung**: kein Volumenproblem; separates Thema.
- **Frontend-Änderungen**: Audit.vue funktioniert. Keine UI-Arbeit.
- **Lese-Event-Emissionen**: GET /api/proxies, GET /api/users etc. werden **nicht**
  auditiert (zu laut, niedriger Security-Wert). Nur Mutationen + Auth + Restart +
  Update.

## 3. Architektur

### 3.1 Vorgehensweise

Zwei Arbeitsteile in einem Branch:

- **Teil A — RBAC-Härtung**: In ~10 Handlern wird als erste Anweisung
  `s.requirePermission(w, r, perm)` eingefügt (genau wie in `handleUpdateCheck`
  bereits etabliert). Die Middleware-Kette am Routing (`authMW`) bleibt unverändert;
  der Permission-Check ist eine In-Handler-Prüfung.

- **Teil B — Audit-Emissionen**: In ~14 Handlern werden 1–2 explizite
  `s.auditor.Log*`-Aufrufe eingefügt (success und/oder failure). Der
  `*auth.Session`-Rückgabewert von `requirePermission` liefert UserID/Username für
  das Audit. Remote-IP und User-Agent werden aus dem Request extrahiert.

### 3.2 Helper für IP/UserAgent (neu, klein)

Um Boilerplate zu vermeiden, wird ein unexportierter Helper eingeführt:

```go
// requestMeta extracts the actor identity (IP, User-Agent) from a request for
// audit logging. IP prefers X-Forwarded-For (first hop), falls back to
// RemoteAddr (host:port → host).
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

Wird in `pkg/api/server.go` (neben `requirePermission`) platziert. Jeder Emit-Punkt
nutzt `ip, ua := requestMeta(r)` statt dreizehn Handlern各自的 Parsing-Logik.

### 3.3 Fehlgeschlagene Logins — der wichtigste neue Flow

Bei fehlgeschlagenem Login gibt es keine gültige Session und damit keinen
`session.UserID`. Die bestehende `LogLogin`-Signatur ist bereits korrekt:

```go
func (a *Auditor) LogLogin(username, ipAddress, userAgent string, success bool)
```

Sie nimmt den angegebenen Username direkt und setzt `UserID=""` im Eintrag (siehe
`LogAction`-Aufruf in audit.go:660). Die Spalte `Username` wird gefüllt,
`Success=false`. Das ist genau das gewünschte Verhalten für Security-Audit.

**Keine Signaturänderung an `LogLogin` nötig.** Wohlgemerkt: Die heutige
`LogLogin` reicht keinen `reason`/Fehlergrund durch (im Gegensatz zu `LogAction`,
das `errorMsg` hat). Da dies für fehlgeschlagene Logins wertvoll wäre ("invalid
credentials" vs. "user disabled" vs. "user expired"), **erweitern** wir `LogLogin`
minimal um einen `reason`-Parameter:

```go
// Aktuell (audit.go:659):
func (a *Auditor) LogLogin(username, ipAddress, userAgent string, success bool)

// Neu:
func (a *Auditor) LogLogin(username, ipAddress, userAgent, reason string, success bool)
```

Der `reason` fließt in `LogAction` als `errorMsg`. Das ist die einzige
API-Surface-Änderung am `pkg/audit`-Paket. Die bestehende `handleLogout` nutzt
`LogLogout` (nicht `LogLogin`), daher kein Breaking Change an existierendem Code.
Der einzige Aufrufer wäre der Login-Handler — den wir ja gerade neu anbinden.

## 4. Detailliertes Mapping

### 4.1 Teil A — RBAC-Härtung

| Handler | Datei:Zeile | Neue Permission | Wirkung auf `benutzer` |
|---------|-------------|-----------------|------------------------|
| `handleAuditLogs` | handlers_extra.go:444 | `PermAuditView` | verliert Zugriff (korrekt) |
| `handleAuditLogsExport` | handlers_extra.go:521 | `PermAuditExport` | verliert Zugriff (korrekt) |
| `handleLogs` | server.go:1205 | `PermLogsView` | verliert Zugriff (korrekt) |
| `handleLogDownload` | handlers_extra.go:51 | `PermLogsExport` | verliert Zugriff (korrekt) |
| `handleLogStream` (SSE) | server.go:1212 | `PermLogsView` | verliert Zugriff (korrekt) |
| `handleConfigExport` | handlers_extra.go:60 | `PermConfigExport` | verliert Zugriff (korrekt) |
| `handleSystemInfo` | handlers_extra.go:217 | `PermSystemView` | verliert Zugriff (korrekt) |
| `handleCheckProxyPorts` | handlers_extra.go:342 | `PermSystemView` | verliert Zugriff (korrekt) |
| `handleUpdateStatus` | handlers_update.go:126 | `PermSystemView` | verliert Zugriff (korrekt) |
| `handleProxiesStream` (SSE) | server.go:989 | `PermProxyView` | **behält** Zugriff (hat Perm) |

`admin` ist von allen Änderungen unberührt (hat alle 24 Permissions).
`techniker`, `auditor` behalten die jeweils relevanten Lesezugriffe laut Matrix.
Nur `benutzer` (die bewusst restriktivste nicht-`auditor` Rolle) verliert
Zugriffe, die laut Permission-Matrix nie für sie gedacht waren.

**Wichtig**: Bei SSE-Endpunkten (`handleLogStream`, `handleProxiesStream`) muss der
`requirePermission`-Check stattfinden, **bevor** der Response-Writer geflusht wird
— ein 403 nach Flush ist nicht möglich. Die Handler beginnen bereits mit Method +
Setup-Checks; der Permission-Check wird als erstes eingefügt.

### 4.2 Teil B — Audit-Emissionen

Jeder Eintrag: Handler → Action-String → Auditor-Methode → Success/Failure.

| Handler | Action-String | Methode | Punkte |
|---------|---------------|---------|--------|
| `handleLogin` multi-user success | `user.login` | `LogLogin(username, ip, ua, "", true)` | in `finalizeLogin` |
| `handleLogin` multi-user fail | `user.login` | `LogLogin(req.Username, ip, ua, reason, false)` | vor 401 |
| `handleLogin` legacy success | `user.login` | `LogLogin("admin", ip, ua, "", true)` | in `finalizeLogin` |
| `handleLogin` legacy fail | `user.login` | `LogLogin("admin", ip, ua, reason, false)` | vor 401 |
| `handleLogout` | `user.logout` | *(bestehend)* | bereits |
| `handleProxies` POST success | `proxy.created` | `LogProxyAction("proxy.created", id, uid, uname, name, ip, ua, true)` | nach `AddProxy` |
| `handleProxies` POST fail | `proxy.created` | `LogProxyAction(..., false)` | bei Validierungs-/Add-Fehler |
| `handleProxies` PUT success | `proxy.updated` | `LogProxyAction("proxy.updated", id, ...)` | nach `UpdateProxy` |
| `handleProxies` PUT fail | `proxy.updated` | `LogProxyAction(..., false)` | bei Fehler |
| `handleProxies` DELETE success | `proxy.deleted` | `LogProxyAction("proxy.deleted", id, ...)` | nach `RemoveProxy` |
| `handleProxies` DELETE fail | `proxy.deleted` | `LogProxyAction(..., false)` | bei Fehler |
| `handleProxyControl` start | `proxy.started` | `LogProxyAction("proxy.started", id, ...)` | success + fail |
| `handleProxyControl` stop | `proxy.stopped` | `LogProxyAction("proxy.stopped", id, ...)` | success + fail |
| `handleProxyControl` restart | `proxy.restarted` | `LogProxyAction("proxy.restarted", id, ...)` | success + fail |
| `handleProxyControl` pause | `proxy.paused` | `LogProxyAction("proxy.paused", id, ...)` | success + fail |
| `handleProxyControl` resume | `proxy.resumed` | `LogProxyAction("proxy.resumed", id, ...)` | success + fail |
| `handleProxyControl` start_all | `proxy.start_all` | `LogAction("proxy.start_all", "proxy", "", uid, uname, "", ip, ua, true, "")` | success + fail |
| `handleProxyControl` stop_all | `proxy.stop_all` | `LogAction("proxy.stop_all", "proxy", "", ...)` | success + fail |
| `handleProxyControl` restart_all | `proxy.restart_all` | `LogAction("proxy.restart_all", "proxy", "", ...)` | success + fail |
| `handleConfigImport` | `config.imported` | `LogConfigChange("config.imported", uid, ...)` | success + fail |
| `handleUsers` POST | `user.created` | `LogUserAction("user.created", newID, uid, ...)` | success + fail |
| `handleUserByID` PUT | `user.updated` | `LogUserAction("user.updated", targetID, ...)` | success + fail |
| `handleUserByID` DELETE | `user.deleted` | `LogUserAction("user.deleted", targetID, ...)` | success + fail |
| `handleSystemRestart` | `system.restart` | `LogAction("system.restart", "system", "", uid, uname, "", ip, ua, true, "")` | success |
| `handleUpdatePerform` | `system.update` | `LogAction("system.update", "system", "", uid, ...)` | success + fail |

**Summe: ~30 Emit-Punkte über ~14 Handler.** (Proxy-Control hat 8 Action-Varianten, davon 5 single-proxy + 3 bulk.)

### 4.3 Event-Typ-Mapping (Wartung in `mapActionToEventType`)

Die Hilfsfunktion `mapActionToEventType` (audit.go:639-656) mappt nur 6 Actions auf
`EventType`-Konstanten; alle anderen fallen durch auf `EventType(action)`. Da
`EventType` ein String-Alias ist, funktioniert das — aber für Konsistenz erweitern
wir das Mapping um die neuen Action-Strings:

Neue Mappings: `proxy.updated`, `proxy.deleted`, `proxy.restarted`, `proxy.paused`,
`proxy.resumed`, `proxy.start_all`, `proxy.stop_all`, `proxy.restart_all`,
`user.created`, `user.updated`, `user.deleted`, `system.restart`, `system.update`,
`config.imported`.

Diese Ergänzung ist **nur relevant, wenn `FileAuditLogger` später aktiviert wird**
(vorher geht der EventType nur in das File-Event, das ungenutzt ist). Wir machen es
trotzdem für Konsistenz.

## 5. Test-Strategie

### 5.1 Unit-Test: `LogLogin` mit reason

`pkg/audit/audit_test.go` — neuer Test `TestAuditor_LogLogin_WithReason`:
Konstruiert Auditor mit `:memory:`-DB, ruft `LogLogin("attacker", "1.2.3.4", "ua",
"invalid credentials", false)`, liest Logs zurück, prüft: Username="attacker",
Success=false, ErrorMsg="invalid credentials", UserID="" (keine valide Session).

### 5.2 Integration-Tests: Emissionen

`pkg/api/audit_emission_test.go` (neu, package `api`). Pro Emit-Punkt ein Test.
Pattern orientiert sich am bestehenden `proxyTestServer`-Helper (server_test.go:92),
erweitert um einen `auditor` mit `:memory:`-DB.

Beispiel-Test für fehlgeschlagenen Login:

```go
func TestAuditLogin_FailureIsLogged(t *testing.T) {
    server, _ := updateTestServer(t) // existierender Helper
    // (server.auditor ist an eine in-memory DB gebunden)

    req := httptest.NewRequest("POST", "/api/login",
        strings.NewReader(`{"username":"ghost","password":"wrong"}`))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    server.handleLogin(w, req)

    if w.Code != http.StatusUnauthorized { t.Fatalf(...) }

    // Audit muss den fehlgeschlagenen Versuch enthalten
    logs, err := server.auditor.GetLogs(100, 0)
    if err != nil { t.Fatal(err) }
    found := false
    for _, e := range logs {
        if e.Action == "user.login" && !e.Success && e.Username == "ghost" {
            found = true
            break
        }
    }
    if !found { t.Error("fehgeschlagener Login nicht auditiert") }
}
```

Pro Emit-Punkt analog. Wo der Handler die `requirePermission` nutzt, wird vorab ein
Admin-Session-Cookie gesetzt (wie in `TestHandleProxiesGet`).

### 5.3 Integration-Tests: RBAC-Härtung

`pkg/api/rbac_hardening_test.go` (neu, package `api`). Pro gehärtetem Endpunkt ein
Test, der mit einer `benutzer`-Session (die die jeweilige Permission nicht hat)
anfragt und `403` erwartet. Optional ein zweiter Test mit `auditor`-Session, der
die korrekte Permission hat und `200` bekommt.

```go
func TestRBAC_AuditLogs_RequiresAuditView(t *testing.T) {
    server, _ := updateTestServer(t)
    benutzerToken := createUser(server, "benutzer", "benutzer")

    req := httptest.NewRequest("GET", "/api/audit/logs?limit=10", nil)
    req.AddCookie(&http.Cookie{Name: "session_token", Value: benutzerToken})
    w := httptest.NewRecorder()
    server.handleAuditLogs(w, req)

    if w.Code != http.StatusForbidden {
        t.Errorf("benutzer sollte /api/audit/logs nicht lesen dürfen, got %d", w.Code)
    }
}
```

Helper `createUser(server, role, name)` legt via `userMgr.CreateUser` einen User an
und gibt einen Session-Token zurück. Reduziert Boilerplate in den ~10 Tests.

### 5.4 Keine Regression

Die bestehenden Tests (`TestHandleProxiesGet`, `TestHandleProxiesPostInvalid`,
etc.) nutzen `admin`-Sessions und dürfen durch die RBAC-Härtung nicht brechen. Die
full suite (`go test ./...`) muss weiterhin grün sein.

## 6. Datei-Übersicht

### Neu
- `pkg/api/audit_emission_test.go` — ~12 Tests, ~250 Zeilen
- `pkg/api/rbac_hardening_test.go` — ~10 Tests, ~200 Zeilen

### Geändert
- `pkg/api/server.go`:
  - Helper `requestMeta(r)` (neu, ~15 Zeilen)
  - Teil A: RBAC in `handleLogs`, `handleLogStream`, `handleProxiesStream`
  - Teil B: Emit in `handleLogin` (success via `finalizeLogin`, 3 Fail-Punkte),
    `handleProxies` (POST/PUT/DELETE), `handleProxyControl` (start/stop/restart),
    `handleSystemRestart`
- `pkg/api/handlers_extra.go`:
  - Teil A: RBAC in `handleAuditLogs`, `handleAuditLogsExport`, `handleLogDownload`,
    `handleConfigExport`, `handleSystemInfo`, `handleCheckProxyPorts`
  - Teil B: Emit in `handleConfigImport`
- `pkg/api/handlers_update.go`:
  - Teil A: RBAC in `handleUpdateStatus`
  - Teil B: Emit in `handleUpdatePerform`
- `pkg/audit/audit.go`:
  - `LogLogin`-Signatur: `+reason string` (1 Zeile Signatur + 1 Aufruf)
  - `mapActionToEventType`: 9 neue Case-Einträge
- `pkg/audit/audit_test.go`:
  - `TestAuditor_LogLogin_WithReason` (neu)

**Geschätzter Diff**: ~500 Zeilen (inkl. Tests), berührt ~7 Dateien.

## 7. Build-Sequenz

1. `pkg/audit/audit.go` — `LogLogin`-Signatur + Mapping erweitern.
2. `pkg/audit/audit_test.go` — Test für `LogLogin` mit reason.
3. `pkg/api/server.go` — `requestMeta`-Helper + Teil A (RBAC) + Teil B (Emissions für
   Login/Proxies/ProxyControl/Restart).
4. `pkg/api/handlers_extra.go` — Teil A + Emit für Config-Import.
5. `pkg/api/handlers_update.go` — Teil A + Emit für Update-Perform.
6. `pkg/api/audit_emission_test.go` — Emission-Tests.
7. `pkg/api/rbac_hardening_test.go` — RBAC-Tests.
8. Verify: `go vet ./...` · `go build ./...` · `go test ./... -race -count=1` ·
   `gofmt -l pkg/`.
9. `version.txt` → `2.0.9.0`.
10. Commit, FF-merge nach `main`, push.

## 8. Risiken

- **Benutzer-Side-Effects**: `benutzer` verliert Zugriff auf Audit/Logs/etc. Das ist
  korrekt gemäß Permission-Matrix, aber falls ein bestehender Prozess sich darauf
  verlässt, bricht er. **Mitigation**: im Commit-Message klar dokumentieren; ggf.
  Changelog-Eintrag. Da `benutzer` die restriktivste Rolle ist und diese Zugriffe
  nie laut Matrix hätte haben dürfen, ist dies ein Bugfix, kein Breaking Change im
  eigentlichen Sinn.
- **Audit-Buffer-Volumen**: bei fehlgeschlagenen Logins (Brute-Force) könnte die
  Audit-Tabelle schnell wachsen. Heute Puffert der Auditor 1000 Entries (audit.go:550)
  und schreibt async in SQLite. Bei Volumen >1000/s würde er Einträge nach 5s
  verwerfen (audit.go:632). Für ein Industrie-Tool mit einer Handzahl Usern ist das
  ausreichend. Langfristig: Rate-Limiter ist bereits am Login-Endpoint aktiv
  (`loginRateLimiter`).
- **SSE + 403**: muss vor Flush passieren (oben erwähnt). Tests decken das ab.

## 9. Erfolgskriterien

- `go test ./... -race` grün, inkl. ~22 neuer Tests.
- Manuelle Verifikation: nach einem Login (success+failure), Proxy-Create,
  Config-Import erscheinen Einträge in `/api/audit/logs`.
- `benutzer`-Session bekommt `403` auf `/api/audit/logs`.
- `admin`-Session merkt keinen Unterschied außer volleren Audit-Logs.
- Keine Regression in bestehenden Tests.
