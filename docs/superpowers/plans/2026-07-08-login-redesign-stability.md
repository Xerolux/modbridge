# Login-Redesign, Multi-User-Default & UI-Stabilität Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ein Login mit Username + Passwort (Default `admin`/`admin`, Erstlogin erzwingt Passwortänderung), stabile Datenanzeige (SSE + Polling-Fallback), und ein SLZB-Light-inspiriertes Redesign der ModBridge-WebUI.

**Architecture:** Backend bleibt Go mit bestehendem `pkg/auth`, `pkg/users`, `pkg/api`; Multi-User (DB) wird Default mit automatischer Migration bestehender Single-User-Setups. Frontend bleibt Vue 3 + PrimeVue 4 + Tailwind; das Layout wird zu Sidebar + Top-Statusbar umstrukturiert. Live-Daten erhalten ein Polling-Fallback neben SSE.

**Tech Stack:** Go 1.26 (CGO), go-sqlite3, bcrypt; Vue 3 (Composition API), Pinia, Vue Router, PrimeVue 4, Tailwind CSS, Axios, vue-i18n.

**Spec:** `docs/superpowers/specs/2026-07-08-login-redesign-stability-design.md`

## Global Constraints

- **Go-Version:** 1.26.x, `CGO_ENABLED=1` (für go-sqlite3).
- **Build:** `make build` (baut Frontend, dann Go-Binary). Frontend-only: `make build-frontend`.
- **Tests:** `make test` (race detector + coverage). Einzelpaket: `go test -v ./pkg/auth/...`.
- **Lint:** `make lint` (golangci-lint). Frontend hat keinen eigenen Lint-Step.
- **Code-Konventionen:** Receiver kurz (`a *Authenticator`, `m *Manager`), Exported = CamelCase, doc-Kommentare für alle exported Symbole, Fehler früh mit `%w` wrappen, Imports stdlib → external → internal.
- **Frontend-Konventionen:** `<script setup>`, Pinia-Stores in `frontend/src/stores/`, i18n für alle Nutzer-Strings (`frontend/src/locales/`), PrimeVue-Komponenten, `lucide-vue-next` Icons.
- **Default-Login:** Benutzername `admin`, Passwort `admin`, `MustChangePassword=true` beim Erststart. Die starke Passwort-Policy bleibt für alle späteren Änderungen aktiv.
- **Kein Framework-Wechsel:** PrimeVue + Tailwind bleibt; kein Bootstrap.
- **Commit-Style:** Conventional Commits (`feat:`, `fix:`, `refactor:`, `style:`, `test:`, `docs:`).

---

## File Structure

### Backend (Go)
- **Modify:** `pkg/auth/auth.go` — `Session` um `MustChangePassword` erweitern; `HashPasswordUnchecked`, `CreateSession` (mit Flag), `InvalidateSession` ergänzen.
- **Modify:** `pkg/users/users.go` — `EnsureDefaultAdminFromHash` ergänzen; `EnsureDefaultAdmin` nutzt `HashPasswordUnchecked`.
- **Modify:** `pkg/api/server.go` — `finalizeLogin`/`handleMe` erweitern; `/api/logout` neu; `/api/setup` deprecated.
- **Modify:** `pkg/config/config.go` — `MultiUser` Default `true`.
- **Modify:** `main.go` — zentrale Bootstrap-Logik (`bootstrapUsers`) mit Fall A/B/C.
- **Modify:** `pkg/auth/auth_test.go` (neu falls nicht vorhanden) — Tests für `HashPasswordUnchecked`, `InvalidateSession`.
- **Create:** `pkg/users/users_test.go`-Abschnitt — Test für `EnsureDefaultAdminFromHash`.

### Frontend (Vue)
- **Modify:** `frontend/src/stores/auth.js` — `mustChangePassword` State; Auth-Cache-Ersatz; `logout()` ruft `/api/logout`.
- **Modify:** `frontend/src/views/Login.vue` — immer Username+Passwort; `force_password_change` Auswertung.
- **Create:** `frontend/src/views/ChangePassword.vue` — erzwungene Passwortänderung.
- **Modify:** `frontend/src/router/index.js` — `/change-password` Route; Guard für `mustChangePassword`; Login-Prefetch.
- **Modify:** `frontend/src/axios.js` — 401-Gate (Dedup); 403-Behandlung.
- **Modify:** `frontend/src/utils/eventSource.js` — `isConnected` erst nach erster Nachricht.
- **Modify:** `frontend/src/stores/appStore.js` — `error`-Flags; Single Source of Truth für Proxies.
- **Modify:** `frontend/src/views/Dashboard.vue`, `frontend/src/views/Control.vue` — Proxy-Liste vom Store; Polling-Fallback + Watchdog.
- **Modify:** `frontend/src/views/SystemInfo.vue` — `error`-Flag + UI.
- **Create:** `frontend/src/components/AppSidebar.vue` — SLZB-Stil Sidebar mit einklappbaren Gruppen.
- **Create:** `frontend/src/components/AppTopBar.vue` — Top-Statusbar.
- **Modify:** `frontend/src/components/Layout.vue` — Sidebar + TopBar integrieren.
- **Modify:** `frontend/src/assets/` (Theme) — Light-Default, dark-slate Buttons.
- **Modify:** `frontend/src/locales/de.json`, `frontend/src/locales/en.json` — neue Strings.

---

## Task 1: Backend — `Session` erweitern + `HashPasswordUnchecked` + `InvalidateSession`

**Files:**
- Modify: `pkg/auth/auth.go`
- Test: `pkg/auth/auth_test.go`

**Interfaces:**
- Produces: `Session.MustChangePassword bool`; `HashPasswordUnchecked(password string) (string, error)`; `CreateSession(userID, username, role string, ttl time.Duration, mustChangePassword bool) (string, error)`; `InvalidateSession(token string) bool`.

- [ ] **Step 1: Failing test für `HashPasswordUnchecked` und `InvalidateSession`**

Erstelle/ergänze `pkg/auth/auth_test.go` (falls die Datei nicht existiert, neu anlegen mit `package auth`):

```go
package auth

import (
	"testing"
	"time"
)

func TestHashPasswordUncheckedAllowsWeak(t *testing.T) {
	hash, err := HashPasswordUnchecked("admin")
	if err != nil {
		t.Fatalf("HashPasswordUnchecked returned error: %v", err)
	}
	if err := CheckPasswordHash("admin", hash); err != nil {
		t.Fatalf("CheckPasswordHash failed for unchecked hash: %v", err)
	}
}

func TestInvalidateSessionRemovesSession(t *testing.T) {
	a := NewAuthenticator()
	token, err := a.CreateSession("u1", "user", "admin", time.Hour, false)
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if a.GetSession(token) == nil {
		t.Fatal("expected session to exist")
	}
	if ok := a.InvalidateSession(token); !ok {
		t.Fatal("expected InvalidateSession to return true for existing token")
	}
	if a.GetSession(token) != nil {
		t.Fatal("expected session to be removed")
	}
	if ok := a.InvalidateSession("nonexistent"); ok {
		t.Fatal("expected InvalidateSession to return false for missing token")
	}
}

func TestCreateSessionStoresMustChangePassword(t *testing.T) {
	a := NewAuthenticator()
	token, _ := a.CreateSession("u1", "user", "admin", time.Hour, true)
	s := a.GetSession(token)
	if s == nil {
		t.Fatal("expected session")
	}
	if !s.MustChangePassword {
		t.Fatal("expected MustChangePassword=true")
	}
}
```

- [ ] **Step 2: Test ausführen — soll fehlschlagen**

Run: `go test -v ./pkg/auth/ -run "TestHashPasswordUncheckedAllowsWeak|TestInvalidateSessionRemovesSession|TestCreateSessionStoresMustChangePassword"`
Expected: FAIL (Funktionen/Methoden existieren noch nicht, bzw. `CreateSession` hat falsche Signatur).

- [ ] **Step 3: `Session`-Struct erweitern**

In `pkg/auth/auth.go`, ersetze das `Session`-Struct (Zeile 22–28):

```go
type Session struct {
	Token             string
	UserID            string
	Username          string
	Role              string
	ExpiresAt         time.Time
	MustChangePassword bool
}
```

- [ ] **Step 4: `HashPasswordUnchecked` ergänzen**

Füge nach `HashPassword` (Zeile 47) ein:

```go
// HashPasswordUnchecked hashes a password with bcrypt WITHOUT enforcing the
// password-strength policy. Use ONLY for seeding a known default password
// (e.g. "admin/admin" on first run); all user-driven password changes must go
// through HashPassword which enforces ValidatePasswordStrength.
func HashPasswordUnchecked(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}
```

- [ ] **Step 5: `CreateSession` um `mustChangePassword` erweitern**

Ersetze die `CreateSession`-Signatur und den Session-Literal (Zeile 115–136):

```go
// CreateSession creates a new session for the given identity and returns the
// opaque session token. mustChangePassword is stored on the session and
// surfaced via /api/me so the frontend can force a password change.
func (a *Authenticator) CreateSession(userID, username, role string, ttl time.Duration, mustChangePassword bool) (string, error) {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.StdEncoding.EncodeToString(b)

	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions[token] = Session{
		Token:              token,
		UserID:             userID,
		Username:           username,
		Role:               role,
		ExpiresAt:          time.Now().Add(ttl),
		MustChangePassword: mustChangePassword,
	}
	return token, nil
}
```

- [ ] **Step 6: `InvalidateSession` ergänzen**

Füge nach `ValidateSession` (Zeile 163) ein:

```go
// InvalidateSession removes a single session by token. Returns true if the
// token existed. Used by /api/logout to end a specific server-side session.
func (a *Authenticator) InvalidateSession(token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.sessions[token]; ok {
		delete(a.sessions, token)
		return true
	}
	return false
}
```

- [ ] **Step 7: Alle `CreateSession`-Aufrufe anpassen**

Es gibt (laut Spec) Aufrufer in `pkg/api/server.go`. Suche alle Aufrufe und ergänze das neue Argument `forcePasswordChange` (Variable ist in `finalizeLogin` vorhanden):

Run: `grep -rn "CreateSession" pkg/`
Passe jeden Aufruf: `s.auth.CreateSession(userID, username, role, ttl, forcePasswordChange)` — der Wert steht in `finalizeLogin` als Parameter bereit. Bei anderen Aufrufstellen (falls vorhanden) `false` übergeben und im Kommentar begründen.

- [ ] **Step 8: Tests ausführen — sollen bestehen**

Run: `go test -v ./pkg/auth/...`
Expected: PASS für alle Tests.

- [ ] **Step 9: Commit**

```bash
git add pkg/auth/auth.go pkg/auth/auth_test.go pkg/api/server.go
git commit -m "feat(auth): add MustChangePassword to Session, HashPasswordUnchecked, InvalidateSession"
```

---

## Task 2: Backend — `EnsureDefaultAdminFromHash` + `EnsureDefaultAdmin` mit Unchecked-Hash

**Files:**
- Modify: `pkg/users/users.go:385-421`
- Test: `pkg/users/users_test.go` (Abschnitt ergänzen oder Datei anlegen)

**Interfaces:**
- Consumes: `auth.HashPasswordUnchecked` (Task 1).
- Produces: `EnsureDefaultAdminFromHash(username, existingHash, createdBy string) (bool, error)`.

- [ ] **Step 1: Test für `EnsureDefaultAdminFromHash` schreiben**

In `pkg/users/users_test.go` (falls nicht vorhanden: neu anlegen, `package users`; für DB-Setup siehe bestehende Tests in der Datei bzw. nutze einen In-Memory-Stub, falls vorhanden — siehe Hinweis in Step 1b). Falls die Datei existiert, ergänze:

```go
func TestEnsureDefaultAdminFromHashIdempotentAndMigrates(t *testing.T) {
	m, cleanup := newTestManager(t) // siehe Step 1b
	defer cleanup()

	// Use a real bcrypt hash of "oldpass" so CheckPasswordHash works.
	hash, err := auth.HashPasswordUnchecked("oldpass-123-Strong!")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	created, err := m.EnsureDefaultAdminFromHash("admin", hash, "system")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !created {
		t.Fatal("expected created=true on empty store")
	}

	// Second call must be idempotent (no-op).
	created2, err := m.EnsureDefaultAdminFromHash("admin", hash, "system")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if created2 {
		t.Fatal("expected created=false when users already exist")
	}

	users, _ := m.GetAllUsers()
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].MustChangePassword {
		t.Fatal("migrated admin must NOT require password change")
	}
}
```

Step 1b: Prüfe, ob `newTestManager` bereits in `users_test.go` existiert. Falls ja, verwende sie. Falls nein, lege einen Helper an, der `users.NewManager` mit einer Test-DB verbindet — siehe wie in `pkg/testing/` oder bestehenden Tests. Falls in der Codebase kein DB-Test-Setup vorhanden ist, überspringe diesen Test mit `t.Skip("requires DB test harness")` und dokumentiere, dass die Logik stattdessen manuell verifiziert wird (und stelle sicher, dass `go build ./pkg/users/...` und `go vet` sauber sind).

- [ ] **Step 2: Test ausführen — soll fehlschlagen (oder skippen)**

Run: `go test -v ./pkg/users/ -run TestEnsureDefaultAdminFromHash`
Expected: FAIL (Methode existiert nicht) oder SKIP.

- [ ] **Step 3: `EnsureDefaultAdminFromHash` implementieren**

Füge in `pkg/users/users.go` nach `EnsureDefaultAdmin` (Zeile 421) ein:

```go
// EnsureDefaultAdminFromHash seeds an initial admin account from an EXISTING
// bcrypt hash (used when migrating a single-user setup: the previous password
// stays valid). Idempotent: no-op if any user already exists. The migrated
// admin's MustChangePassword is false because the operator already knows the
// password. Returns created=true only when a user was actually inserted.
func (m *Manager) EnsureDefaultAdminFromHash(username, existingHash, createdBy string) (bool, error) {
	existing, err := m.db.GetAllUsers()
	if err != nil {
		return false, err
	}
	if len(existing) > 0 {
		return false, nil
	}

	username = strings.TrimSpace(username)
	if username == "" {
		username = "admin"
	}
	if strings.TrimSpace(existingHash) == "" {
		return false, errors.New("EnsureDefaultAdminFromHash: existing hash must not be empty")
	}

	id, _ := generateID()
	admin := &database.User{
		ID:                 id,
		Username:           username,
		FullName:           "Administrator",
		Email:              "admin@modbridge.local",
		PasswordHash:       existingHash,
		Role:               string(rbac.RoleAdmin),
		Enabled:            true,
		MustChangePassword: false,
		CreatedBy:          createdBy,
		Description:        "Migrated administrator account",
	}
	if err := m.db.CreateUser(admin); err != nil {
		return false, err
	}
	return true, nil
}
```

Prüfe, dass `errors` importiert ist (oben in `users.go`); ggf. ergänzen.

- [ ] **Step 4: `EnsureDefaultAdmin` auf `HashPasswordUnchecked` umstellen**

Ersetze in `EnsureDefaultAdmin` (Zeile 399) die Zeile:
```go
	hash, err := auth.HashPassword(password)
```
durch:
```go
	// Use the UNCHECKED hasher so seeding a default password like "admin" is
	// allowed even though it violates ValidatePasswordStrength. MustChangePassword
	// is set true to force a strong replacement on first login.
	hash, err := auth.HashPasswordUnchecked(password)
```

- [ ] **Step 5: Build + Tests**

Run: `go build ./pkg/users/... && go vet ./pkg/users/... && go test -v ./pkg/users/`
Expected: Build OK, Tests PASS (oder sinnvolle Skips).

- [ ] **Step 6: Commit**

```bash
git add pkg/users/users.go pkg/users/users_test.go
git commit -m "feat(users): add EnsureDefaultAdminFromHash for single-user migration; seed admin/admin"
```

---

## Task 3: Backend — `/api/me` und `/api/logout`, `/api/setup` deprecated

**Files:**
- Modify: `pkg/api/server.go:511-609` (`finalizeLogin`), `server.go:611-650` (`handleMe`), `server.go:472-508` (`handleSetup`), `server.go:234-240` (Routen).

**Interfaces:**
- Consumes: `Session.MustChangePassword`, `InvalidateSession` (Task 1).
- Produces: `handleLogout`; `/api/me` liefert `must_change_password`; `/api/setup` → 410.

- [ ] **Step 1: `finalizeLogin` — `MustChangePassword` in Session speichern**

`finalizeLogin` (Zeile 564) ruft bereits `CreateSession(...)` auf (in Task 1 angepasst). Es ist keine weitere Änderung nötig, außer dass `forcePasswordChange` durchgereicht wird — das ist durch Step 7 von Task 1 bereits erledigt. Verifiziere durch Lesen, dass der Aufruf `s.auth.CreateSession(userID, username, role, ttl, forcePasswordChange)` lautet.

- [ ] **Step 2: `handleMe` erweitern**

Ersetze in `handleMe` (Zeile 642–649) den JSON-Block, sodass `must_change_password` enthalten ist:

```go
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
```

- [ ] **Step 3: `handleLogout` implementieren**

Füge eine neue Handler-Funktion hinzu (z. B. nach `handleMe`, ~Zeile 650):

```go
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
		s.auth.InvalidateSession(c.Value)
		session := s.auth.GetSession(c.Value)
		if session != nil {
			s.audit.LogLogout(session.UserID, session.Username, r.RemoteAddr)
		}
	}
	// Clear cookies defensively even though client does it too.
	for _, name := range []string{"session_token", "csrf_token"} {
		http.SetCookie(w, &http.Cookie{
			Name: name, Value: "", Expires: time.Unix(0, 0),
			Path: "/", HttpOnly: true, MaxAge: -1,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
```

Hinweis: Prüfe, ob `s.audit` ein Feld des Servers ist und ob `LogLogout(userID, username, ip string)` diese Signatur hat. Falls `s.audit` anders heißt oder `GetSession` nach `InvalidateSession` nicht mehr liefert (es ist dann `nil`), passe an: hole die Session VOR `InvalidateSession`. Korrigierte Reihenfolge:

```go
	if c, err := r.Cookie("session_token"); err == nil && c.Value != "" {
		if session := s.auth.GetSession(c.Value); session != nil {
			s.audit.LogLogout(session.UserID, session.Username, r.RemoteAddr)
		}
		s.auth.InvalidateSession(c.Value)
	}
```

Prüfe via `grep -n "audit" pkg/api/server.go | head`, wie das Audit-Logger-Feld heißt; passe den Feldnamen an die Realität an (z. B. `s.auditLog` o. ä.).

- [ ] **Step 4: `/api/setup` deprecated (410)**

Ersetze den Body von `handleSetup` (Zeile 472–508). Ganz am Anfang der Funktion, noch vor der Methodenprüfung, füge ein:

```go
	if cfg := s.cfgMgr.Get(); cfg.AdminPassHash != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone) // 410
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "setup is deprecated; default admin is auto-created on first run",
		})
		return
	}
```

(Damit bleibt der frühe Setup-Pfad für den theoretischen Leer-Hash-Fall erhalten, gibt aber im Normalbetrieb 410 zurück.)

- [ ] **Step 5: Route `/api/logout` registrieren**

In der Routen-Sektion (Zeile 234–240), nach `/api/login`:

```go
	mux.HandleFunc("/api/login", s.cors.Middleware(s.security.Middleware(s.loginRateLimiter.Middleware(s.handleLogin))))
	mux.HandleFunc("/api/logout", csrfMW(s.handleLogout))
```

`csrfMW` ist passend, weil es ein authentifizierter POST ist (Session + CSRF-Check). Prüfe, dass `csrfMW` definiert ist (ja, in `server.go:212-232`).

- [ ] **Step 6: Build + manuelles Smoke-Test**

Run: `go build ./pkg/api/... && go vet ./pkg/api/...`
Expected: OK.

- [ ] **Step 7: Commit**

```bash
git add pkg/api/server.go
git commit -m "feat(api): /api/logout invalidates session; /api/me reports must_change_password; /api/setup deprecated (410)"
```

---

## Task 4: Backend — `MultiUser` Default `true` + zentrale Bootstrap-Logik

**Files:**
- Modify: `pkg/config/config.go` (Default für `MultiUser`)
- Modify: `main.go:65-85, 108-122` (Bootstrap)

**Interfaces:**
- Consumes: `EnsureDefaultAdmin`, `EnsureDefaultAdminFromHash` (Task 2).
- Produces: Start mit admin/admin (Neuinstallation) oder migriertem Admin.

- [ ] **Step 1: `MultiUser` Default auf `true` setzen**

Finde in `pkg/config/config.go` die `Load`-Methode (bzw. wo `json.Unmarshal` läuft). Run: `grep -n "Unmarshal\|func.*Load\|MultiUser" pkg/config/config.go`.

Da `false` der Zero-Value von `bool` ist, kann man „implizit false" nicht von „explizit false" unterscheiden, wenn man nur das Struct betrachtet. Daher: prüfe auf Schlüssel-Präsenz im Roh-JSON. Dekodiere zweistufig — zuerst in eine `map[string]json.RawMessage`, dann das Struct — und setze den Default nur, wenn der Schlüssel fehlt:

```go
func (m *Manager) Load() error {
	raw, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}

	// Detect which keys were explicitly present so we can apply defaults only
	// for truly-absent keys (multi_user defaults to true now).
	var keySet map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keySet); err != nil {
		return err
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}

	// Multi-user (DB-backed auth) is the default mode. An operator can opt out
	// explicitly via {"multi_user": false} in config.json, which is honored
	// because the key is present. Absent key -> default true.
	if _, ok := keySet["multi_user"]; !ok {
		cfg.MultiUser = true
	}

	m.mu.Lock()
	m.config = cfg
	m.mu.Unlock()
	return nil
}
```

(Passe die Feld-/Methodennamen an die echte `Manager`-Struktur an — `m.config`, `m.mu`, `m.path` o.ä. Falls es schon eine `applyDefaults`-Funktion gibt, integriere den `keySet`-Check dorthin.)

- [ ] **Step 2: Test für den Default**

Falls `pkg/config/config_test.go` existiert, ergänze einen Test, der eine leere Config lädt und `MultiUser == true` erwartet. Falls der Default über eine explizite Setzung im Konstruktor läuft, schreibe:

```go
func TestMultiUserDefaultsTrue(t *testing.T) {
	cfg := Default() // oder NewManager+Load mit leeren Bytes
	if !cfg.MultiUser {
		t.Fatal("expected MultiUser to default to true")
	}
}
```

(Passe den Konstruktoraufruf an die echte API an.)

- [ ] **Step 3: Test ausführen — soll fehlschlagen, dann durch Anpassung bestehen**

Run: `go test -v ./pkg/config/ -run TestMultiUser`
Passe die Implementierung an, bis der Test grün ist.

- [ ] **Step 4: `main.go` Bootstrap zentralisieren**

Ersetze in `main.go` die Passagen Zeile 65–85 (Single-User-Bootstrap) und Zeile 108–122 (Multi-User-Bootstrap) durch eine zentrale Funktion. Entferne den alten Single-User-Hash-Block (Zeile 65–85) vollständig und ersetze den Multi-User-Block durch einen Aufruf der neuen Funktion.

Füge eine neue Funktion in `main.go` hinzu (z. B. vor `func main` oder in einer neuen Datei `main_bootstrap.go`, falls `main.go` sonst zu groß — hier: in `main.go` für Einfachheit):

```go
// bootstrapUsers ensures a usable admin account exists in multi-user mode.
//   - Fresh install (no users, no prior AdminPassHash): creates admin/admin with
//     MustChangePassword=true (change forced on first login).
//   - Migration (no users, but a prior single-user AdminPassHash): creates the
//     admin from the EXISTING bcrypt hash so the operator's old password stays
//     valid; MustChangePassword mirrors the prior ForcePasswordChange flag.
//   - Already populated: no-op.
func bootstrapUsers(userMgr *users.Manager, cfg *config.Config, l *logger.Logger) {
	if userMgr == nil {
		return
	}
	existing, err := userMgr.GetAllUsers()
	if err != nil {
		l.Error("SYSTEM", fmt.Sprintf("bootstrap: cannot read users: %v", err))
		return
	}
	if len(existing) > 0 {
		return // already populated
	}

	if strings.TrimSpace(cfg.AdminPassHash) == "" {
		// Fresh install: default admin/admin.
		if _, err := userMgr.EnsureDefaultAdmin("admin", "admin", "system"); err != nil {
			l.Error("SYSTEM", fmt.Sprintf("bootstrap: failed to create default admin: %v", err))
			return
		}
		l.Info("SYSTEM", "Default-Login erstellt — Benutzername: admin / Passwort: admin — BITTE beim ersten Login ändern.")
		log.Println("SYSTEM: Default admin created (username: admin, password: admin). Change it on first login.")
		return
	}

	// Migration: reuse the existing single-user password hash.
	forced := cfg.ForcePasswordChange
	_, err = userMgr.EnsureDefaultAdminFromHash("admin", cfg.AdminPassHash, "system")
	if err != nil {
		l.Error("SYSTEM", fmt.Sprintf("bootstrap: migration failed: %v", err))
		return
	}
	if forced {
		// Migration mit erzwungener Änderung: setze MustChangePassword nachträglich.
		_ = userMgr.SetMustChangePasswordByUsername("admin", true)
	}
	l.Info("SYSTEM", "Bestehendes Admin-Passwort migriert (Benutzername: admin).")
	log.Println("SYSTEM: Existing admin password migrated (username: admin).")
}
```

Hinweis: `SetMustChangePasswordByUsername` ist u. U. nicht vorhanden. Prüfe via `grep -n "MustChangePassword" pkg/users/users.go`. Falls nicht vorhanden, füge sie in Task 2 / hier hinzu:

```go
// SetMustChangePasswordByUsername sets the MustChangePassword flag for the user
// matching username. No-op (returns nil) if the user does not exist.
func (m *Manager) SetMustChangePasswordByUsername(username string, must bool) error {
	u, err := m.db.GetUserByUsername(username)
	if err != nil || u == nil {
		return err
	}
	u.MustChangePassword = must
	return m.db.UpdateUser(u)
}
```

Stelle sicher, dass `GetUserByUsername` und `UpdateUser` in der DB-Schicht existieren (`grep -n "GetUserByUsername\|UpdateUser" pkg/database/*.go`). Falls die Methode anders heißt, passe an.

Imports in `main.go`: stelle sicher, dass `strings`, `fmt`, `log`, `config`, `users`, `logger` vorhanden sind.

- [ ] **Step 5: Bootstrap im Hauptfluss aufrufen**

In `main.go`, an der Stelle des bisherigen Multi-User-Blocks (Zeile 108–122), ersetze durch:

```go
	// 6. User store + bootstrap (multi-user is the default mode).
	if db != nil {
		userMgr := users.NewManager(db)
		bootstrapUsers(userMgr, cfgMgr.Get(), l)

		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-authCtx.Done():
```

(Übernehme den Rest der bisherigen Goroutine für abgelaufene User unverändert — nur der Bootstrap-Aufruf ändert sich.)

Entferne den kompletten alten Single-User-Bootstrap-Block (Zeile 65–85), da `AdminPassHash` im Multi-User-Default nicht mehr proaktiv gesetzt wird. Falls andere Code-Pfade (`handleLogin` Single-User-Fallback) `cfg.AdminPassHash == ""` erwarten, bleibt das Verträglich: leerer Hash → Single-User-Login schlägt fehl, aber Multi-User ist Default und aktiv.

- [ ] **Step 6: Build + vet**

Run: `go build ./... && go vet ./...`
Expected: OK (CGO muss aktiv sein für sqlite; ggf. `CGO_ENABLED=1` setzen).

- [ ] **Step 7: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go main.go pkg/users/users.go
git commit -m "feat(bootstrap): default to multi-user; seed admin/admin or migrate existing password"
```

---

## Task 5: Frontend — Auth-Store (`mustChangePassword`, Auth-Cache-Ersatz, `/api/logout`)

**Files:**
- Modify: `frontend/src/stores/auth.js`

**Interfaces:**
- Consumes: `/api/login` (antwortet `force_password_change`), `/api/me` (antwortet `must_change_password`), `/api/logout`.
- Produces: `auth.mustChangePassword` (ref), angepasstes `checkAuth`, `logout()` ruft `/api/logout`.

- [ ] **Step 1: `mustChangePassword` State hinzufügen**

In `frontend/src/stores/auth.js`, ergänze im State-Block (neben `isAuthenticated`, `user`, etc.):

```js
const mustChangePassword = ref(false)
```

Und expose es im `return` des Stores.

- [ ] **Step 2: Auth-Cache-Ersatz — "verify-once-per-load"**

Ersetze die Cache-Logik (`AUTH_CHECK_CACHE_MS = 5000` und der 5s-Cache in `checkAuth`) durch eine „einmal pro Seitenladung verifizieren"-Strategie:

```js
// Auth is verified exactly once per page load (F5 reload resets this). After
// that, trust the in-memory state and react to 401 via onUnauthorized. This
// avoids the bug where a 5s cache returned "authenticated" while the server
// session was already gone.
let verifiedThisLoad = false

const checkAuth = async () => {
  if (isAuthenticated.value && verifiedThisLoad) {
    return true
  }
  // ... existing in-flight dedup (checkPromise) stays ...
  try {
    const meRes = await axios.get('/api/me')
    user.value = {
      userId: meRes.data.user_id || 'admin',
      username: meRes.data.username || 'admin',
      role: meRes.data.role || 'admin',
      permissions: meRes.data.permissions || [],
    }
    mustChangePassword.value = !!meRes.data.must_change_password
    isAuthenticated.value = true
    verifiedThisLoad = true
    return true
  } catch (e) {
    resetState()
    verifiedThisLoad = false
    return false
  }
}
```

Behalte die bestehende `checkPromise`-Dedup bei (ggf. anpassen, damit parallele Guard-Aufrufe denselben Promise teilen).

- [ ] **Step 3: `login()` wertet `force_password_change` aus**

In `login()` (Zeile 70–89), nach dem `/api/login`-Call:

```js
const login = async (payload) => {
  const res = await axios.post('/api/login', payload)
  mustChangePassword.value = !!res.data.force_password_change
  const meRes = await axios.get('/api/me')
  user.value = {
    userId: meRes.data.user_id || res.data.user_id || 'admin',
    username: meRes.data.username || res.data.username || 'admin',
    role: meRes.data.role || res.data.role || 'admin',
    permissions: meRes.data.permissions || [],
  }
  mustChangePassword.value = mustChangePassword.value || !!meRes.data.must_change_password
  isAuthenticated.value = true
  return { success: true, mustChangePassword: mustChangePassword.value }
}
```

- [ ] **Step 4: `logout()` ruft `/api/logout` auf**

```js
const logout = async () => {
  try {
    await axios.post('/api/logout')
  } catch (e) {
    // ignore — we clear cookies client-side regardless
  }
  resetState()
  // clear cookies client-side as a fallback
  const secureFlag = window.location.protocol === 'https:' ? '; Secure' : ''
  document.cookie = `session_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/; SameSite=Lax${secureFlag}`
  document.cookie = `csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/; SameSite=Lax${secureFlag}`
}
```

- [ ] **Step 5: `resetState` setzt `mustChangePassword` zurück**

Stelle in `resetState()` sicher, dass `mustChangePassword.value = false` und `verifiedThisLoad = false` gesetzt werden.

- [ ] **Step 6: Build prüfen**

Run (aus `frontend/`): `npm run build`
Expected: Build OK (keine Syntaxfehler).

- [ ] **Step 7: Commit**

```bash
git add frontend/src/stores/auth.js
git commit -m "feat(auth-store): evaluate force_password_change; verify-once-per-load; call /api/logout"
```

---

## Task 6: Frontend — Login.vue (immer Username+Passwort) + ChangePassword.vue

**Files:**
- Modify: `frontend/src/views/Login.vue`
- Create: `frontend/src/views/ChangePassword.vue`
- Modify: `frontend/src/router/index.js`

**Interfaces:**
- Consumes: `auth.login(payload)` (gibt `{success, mustChangePassword}`), `auth.mustChangePassword`.
- Produces: `/change-password` Route + View.

- [ ] **Step 1: `Login.vue` — Username immer sichtbar, `force_password_change` Auswertung**

Ersetze in `Login.vue` das `handleLogin` und das Template so, dass:
- Username-Feld immer gerendert wird (das `v-if="multiUser"` am Username-Input entfällt).
- `username` default `''` mit Placeholder `admin`.
- Nach Login:

```js
const handleLogin = async () => {
  error.value = ''
  loading.value = true
  try {
    const payload = {
      username: username.value.trim() || 'admin',
      password: password.value,
    }
    const result = await authStore.login(payload)
    if (result.mustChangePassword) {
      router.push('/change-password')
    } else {
      router.push('/')
    }
  } catch (err) {
    error.value = err?.response?.data?.error || t('login.invalidCredentials')
  } finally {
    loading.value = false
  }
}
```

Im Template: Username-Input ohne `v-if`, mit `v-model="username"`, `placeholder="admin"`, `autocomplete="username"`. Die `onMounted`-Abfrage von `/api/status` (für `multiUser`) kann bleiben, beeinflusst aber nicht mehr die Feld-Sichtbarkeit.

- [ ] **Step 2: `ChangePassword.vue` erstellen**

```vue
<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import axios from 'axios'

const router = useRouter()
const toast = useToast()
const authStore = useAuthStore()
const { t } = useI18n()

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const loading = ref(false)

const submit = async () => {
  if (newPassword.value !== confirmPassword.value) {
    toast.add({ severity: 'warn', summary: t('changePassword.mismatch'), life: 3000 })
    return
  }
  loading.value = true
  try {
    await axios.post('/api/config/password', {
      current_password: currentPassword.value,
      new_password: newPassword.value,
    })
    toast.add({ severity: 'success', summary: t('changePassword.success'), life: 3000 })
    authStore.mustChangePassword = false
    // Sessions are invalidated server-side; force re-login.
    await authStore.logout()
    router.push('/login')
  } catch (err) {
    const msg = err?.response?.data?.error || t('changePassword.error')
    toast.add({ severity: 'error', summary: msg, life: 4000 })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
    <div class="w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-md p-8">
      <h1 class="text-2xl font-semibold mb-2 text-gray-900 dark:text-white">{{ t('changePassword.title') }}</h1>
      <p class="text-sm text-gray-600 dark:text-gray-300 mb-6">{{ t('changePassword.subtitle') }}</p>
      <form @submit.prevent="submit" class="space-y-4">
        <input v-model="currentPassword" type="password" :placeholder="t('changePassword.current')"
               class="w-full px-4 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white" autocomplete="current-password" />
        <input v-model="newPassword" type="password" :placeholder="t('changePassword.new')"
               class="w-full px-4 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white" autocomplete="new-password" />
        <input v-model="confirmPassword" type="password" :placeholder="t('changePassword.confirm')"
               class="w-full px-4 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white" autocomplete="new-password" />
        <button type="submit" :disabled="loading" class="w-full py-2 rounded bg-gray-900 text-white hover:bg-gray-700 disabled:opacity-60">
          {{ loading ? t('common.saving') : t('changePassword.submit') }}
        </button>
      </form>
    </div>
  </div>
</template>
```

- [ ] **Step 3: Route `/change-password` registrieren + Guard**

In `frontend/src/router/index.js`, ergänze in den `routes`:

```js
{
  path: '/change-password',
  name: 'change-password',
  component: () => import('@/views/ChangePassword.vue'),
  meta: { requiresAuth: true, allowDuringMustChange: true },
},
```

Im `beforeEach`-Guard, nach der `checkAuth` und der Login-Weiterleitung, ergänze den `mustChangePassword`-Block:

```js
const valid = await auth.checkAuth()
if (!valid) { return { path: '/login', replace: true } }
if (auth.mustChangePassword && !to.meta.allowDuringMustChange) {
  return { path: '/change-password', replace: true }
}
```

Stelle sicher, dass die `/login`-Route kein `requiresAuth` hat (bereits der Fall) und dass `/change-password` trotz `requiresAuth` erreichbar ist, weil `allowDuringMustChange` den Block umgeht.

- [ ] **Step 4: i18n-Strings ergänzen**

In `frontend/src/locales/de.json` und `en.json` (Struktur anpassen, falls flach/verschachtelt):

de:
```json
"changePassword": {
  "title": "Passwort ändern",
  "subtitle": "Bitte ändern Sie Ihr Passwort beim ersten Login.",
  "current": "Aktuelles Passwort",
  "new": "Neues Passwort",
  "confirm": "Passwort bestätigen",
  "submit": "Passwort ändern",
  "mismatch": "Die Passwörter stimmen nicht überein.",
  "success": "Passwort geändert. Bitte neu einloggen.",
  "error": "Passwort konnte nicht geändert werden."
}
```

en: entsprechende englische Werte.

- [ ] **Step 5: Build prüfen**

Run (aus `frontend/`): `npm run build`
Expected: OK.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/Login.vue frontend/src/views/ChangePassword.vue frontend/src/router/index.js frontend/src/locales/de.json frontend/src/locales/en.json
git commit -m "feat(login): always show username+password; force password change on first login"
```

---

## Task 7: Frontend — Stabilität: 401-Gate, 403-Behandlung, EventSource-Verbesserung

**Files:**
- Modify: `frontend/src/axios.js`
- Modify: `frontend/src/utils/eventSource.js`

**Interfaces:**
- Produces: deduplizierter 401-Redirect; 403-Toast; `isConnected` erst nach erster SSE-Nachricht.

- [ ] **Step 1: 401-Gate (Dedup) und 403-Toast in `axios.js`**

In `frontend/src/axios.js`, ersetze den Response-Interceptor-Block (Zeile 27–56) so, dass ein Modul-Flag den 401-Redirect dedupliziert und 403 sauber behandelt wird:

```js
let isHandlingUnauthorized = false

axios.interceptors.response.use(
  (response) => response,
  async (error) => {
    const config = error.config || {}

    // 401 — dedup: only one redirect + one reset per "wave".
    if (error.response && error.response.status === 401) {
      if (!isHandlingUnauthorized) {
        isHandlingUnauthorized = true
        emitUnauthorized()
        if (window.location.hash !== '#/login') {
          window.location.hash = '#/login'
        }
        setTimeout(() => { isHandlingUnauthorized = false }, 1000)
      }
      return Promise.reject(error)
    }

    // 403 — surface a toast; do NOT redirect.
    if (error.response && error.response.status === 403) {
      // Toast via a custom event; Layout listens and shows PrimeVue toast.
      window.dispatchEvent(new CustomEvent('app:forbidden'))
      return Promise.reject(error)
    }

    // Retry idempotent requests once on 429/5xx/network.
    const isIdempotent = ['get', 'head', 'options', 'put', 'delete'].includes((config.method || '').toLowerCase())
    const retriableStatus = error.response && (error.response.status === 429 || error.response.status >= 503)
    const isNetworkError = !error.response
    if (isIdempotent && !config.__retried && (retriableStatus || isNetworkError)) {
      config.__retried = true
      await new Promise((r) => setTimeout(r, 500))
      return axios(config)
    }
    return Promise.reject(error)
  }
)
```

- [ ] **Step 2: EventSource — `isConnected` erst nach erster Nachricht**

In `frontend/src/utils/eventSource.js`, ändere `onopen` (Zeile 36–40) so, dass es NICHT mehr `isConnected = true` setzt, sondern nur `isConnecting = false`. Setze `isConnected = true` stattdessen in der ersten `onmessage`:

```js
	onopen: () => {
		isConnecting = false
		// Note: isConnected is set to true only after the first message arrives,
		// so consumers don't treat a freshly-opened but data-less connection as live.
	},
	onmessage: (ev) => {
		isConnecting = false
		if (!isConnected.value) {
			isConnected.value = true
		}
		lastMessageAt = Date.now()
		// ... existing message handling ...
	},
```

Füge einen `lastMessageAt`-Ref/State hinzu, falls noch nicht vorhanden (für den Watchdog in Task 8):

```js
const lastMessageAt = ref(Date.now())
// expose lastMessageAt in the composable's return
```

- [ ] **Step 3: Forbidden-Toast im Layout abonnieren**

In `frontend/src/components/Layout.vue` (oder wo der PrimeVue-Toast gemountet ist), ergänze in `<script setup>`:

```js
import { useToast } from 'primevue/usetoast'
const toast = useToast()
onMounted(() => {
  window.addEventListener('app:forbidden', () => {
    toast.add({ severity: 'warn', summary: t('common.forbidden'), life: 3000 })
  })
})
onUnmounted(() => window.removeEventListener('app:forbidden'))
```

(Vorausgesetzt: `onMounted`/`onUnmounted` und `t` sind bereits importiert; ggf. ergänzen.)

- [ ] **Step 4: Build prüfen**

Run (aus `frontend/`): `npm run build`
Expected: OK.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/axios.js frontend/src/utils/eventSource.js frontend/src/components/Layout.vue
git commit -m "fix(stability): dedup 401 redirect; handle 403; mark SSE connected only after first message"
```

---

## Task 8: Frontend — Stabilität: zentrale Proxy-Liste, Polling-Fallback, Watchdog, Error-Flags

**Files:**
- Modify: `frontend/src/stores/appStore.js`
- Modify: `frontend/src/views/Dashboard.vue`
- Modify: `frontend/src/views/Control.vue`
- Modify: `frontend/src/views/SystemInfo.vue`

**Interfaces:**
- Consumes: `appStore.proxies` (Single Source of Truth), `useAutoRefresh`, `useEventSource` mit `lastMessageAt`.
- Produces: Dashboard/Control beziehen Proxies vom Store; Polling-Fallback + Watchdog.

- [ ] **Step 1: `appStore` — `error`-Flags setzen; `fetchProxies` mit `isLoading`**

In `frontend/src/stores/appStore.js`:
- `fetchProxies` (Zeile 78–89): ergänze `isLoading`-Setzung:

```js
const fetchProxies = async () => {
  try {
    isLoading.value = true
    const res = await axios.get('/api/proxies')
    proxies.value = res.data
    error.value = null
  } catch (e) {
    error.value = e
    console.error('fetchProxies', e)
  } finally {
    isLoading.value = false
  }
}
```

- `fetchStatus` (Zeile 167–174) und `fetchWebPort` (Zeile 144–151): ergänze je `error.value = e` statt nur `console.error`.

- [ ] **Step 2: `Dashboard.vue` — Proxies vom Store + Polling-Fallback + Watchdog**

In `frontend/src/views/Dashboard.vue`:
- Ersetze die lokale `proxies`-Ref (Zeile 193) durch eine Ableitung aus dem Store:

```js
import { useAppStore } from '@/stores/appStore'
import { storeToRefs } from 'pinia'
const store = useAppStore()
const { proxies } = storeToRefs(store)
```

(Entferne die lokale `proxies`-Ref und alle Stellen, die sie direkt setzen; SSE-Updates sollen stattdessen `store` mutieren oder via `store.fetchProxies()` neu laden.)

- Polling-Fallback neben SSE: in `onMounted` (Zeile 317–371), nach dem `useEventSource`-Setup, ergänze:

```js
import { useAutoRefresh } from '@/utils/useAutoRefresh'
const { start, stop } = useAutoRefresh(() => store.fetchProxies(), { intervalMs: 10000 })
start()
onUnmounted(() => stop())
```

- Watchdog: überwache `lastMessageAt`/`isConnected` des EventSource; bei >30s ohne Nachricht oder `!isConnected`, sofortiger Poll und Hinweis:

```js
const liveStale = ref(false)
watch([() => es.isConnected.value, () => es.lastMessageAt.value], () => {
  const stale = !es.isConnected.value || (Date.now() - es.lastMessageAt.value) > 30000
  liveStale.value = stale
  if (stale) store.fetchProxies()
}, { immediate: true })
// optional: setInterval that re-checks every 15s while mounted
```

Im Template: ein dezenter Hinweis, wenn `liveStale`:

```html
<div v-if="liveStale" class="text-xs text-amber-600">{{ t('dashboard.liveStale') }}</div>
```

- [ ] **Step 3: `Control.vue` — analog zu Dashboard**

Wende dieselben Änderungen wie in Step 2 auf `Control.vue` an (lokale `proxies`-Ref Zeile 331 → Store; Polling-Fallback; Watchdog).

- [ ] **Step 4: `SystemInfo.vue` — `error`-Flag + UI**

In `frontend/src/views/SystemInfo.vue` (Zeile 273–284), ergänze ein `error`-Ref und zeige einen Fehler-/Retry-Bereich:

```js
const error = ref(null)
const fetchInfo = async () => {
  try {
    error.value = null
    const res = await axios.get('/api/system/info')
    info.value = res.data
  } catch (e) {
    error.value = e
  }
}
```

Im Template (oberhalb der Datenanzeige):

```html
<div v-if="error" class="rounded bg-red-50 dark:bg-red-900 p-4 mb-4 flex items-center justify-between">
  <span class="text-red-700 dark:text-red-200">{{ t('systemInfo.loadError') }}</span>
  <button @click="fetchInfo" class="px-3 py-1 rounded bg-gray-900 text-white">{{ t('common.retry') }}</button>
</div>
```

- [ ] **Step 5: i18n-Strings ergänzen**

`dashboard.liveStale`, `systemInfo.loadError`, `common.retry`, `common.forbidden` in de.json + en.json ergänzen.

- [ ] **Step 6: Build prüfen**

Run (aus `frontend/`): `npm run build`
Expected: OK.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/stores/appStore.js frontend/src/views/Dashboard.vue frontend/src/views/Control.vue frontend/src/views/SystemInfo.vue frontend/src/locales/de.json frontend/src/locales/en.json
git commit -m "fix(stability): single source of truth for proxies; SSE polling fallback + watchdog; surface errors"
```

---

## Task 9: Frontend — Redesign: AppSidebar + AppTopBar (SLZB-Stil)

**Files:**
- Create: `frontend/src/components/AppSidebar.vue`
- Create: `frontend/src/components/AppTopBar.vue`
- Modify: `frontend/src/components/Layout.vue`

**Interfaces:**
- Produces: einklappbare Sidebar-Gruppen + Top-Statusbar im SLZB-Light-Stil.

- [ ] **Step 1: `AppSidebar.vue` — Sidebar mit einklappbaren Gruppen**

```vue
<script setup>
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  LayoutDashboard, SlidersHorizontal, Cpu, ShieldCheck, Settings,
  ScrollText, ChevronDown, Users, FileWarning, Languages,
} from 'lucide-vue-next'

const { t, locale, availableLocales } = useI18n()
const open = ref({ proxies: true, security: false, system: false })

const toggle = (key) => { open.value[key] = !open.value[key] }
const onLocale = (e) => { locale.value = e.target.value }
</script>

<template>
  <aside class="w-64 shrink-0 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 h-screen flex flex-col">
    <nav class="flex-1 overflow-y-auto py-4">
      <!-- Proxies group -->
      <div>
        <button @click="toggle('proxies')" class="w-full flex items-center justify-between px-4 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700">
          <span class="flex items-center gap-2"><SlidersHorizontal :size="18" /> {{ t('nav.proxies') }}</span>
          <ChevronDown :size="16" :class="{ 'rotate-180': open.proxies }" class="transition-transform" />
        </button>
        <div v-show="open.proxies" class="pl-8">
          <RouterLink to="/" class="block px-2 py-1.5 rounded text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2">
            <LayoutDashboard :size="16" /> {{ t('nav.dashboard') }}
          </RouterLink>
          <RouterLink to="/control" class="block px-2 py-1.5 rounded text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2">
            <SlidersHorizontal :size="16" /> {{ t('nav.control') }}
          </RouterLink>
        </div>
      </div>

      <RouterLink to="/devices" class="block px-4 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2">
        <Cpu :size="18" /> {{ t('nav.devices') }}
      </RouterLink>

      <!-- Security group -->
      <div>
        <button @click="toggle('security')" class="w-full flex items-center justify-between px-4 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700">
          <span class="flex items-center gap-2"><ShieldCheck :size="18" /> {{ t('nav.security') }}</span>
          <ChevronDown :size="16" :class="{ 'rotate-180': open.security }" class="transition-transform" />
        </button>
        <div v-show="open.security" class="pl-8">
          <RouterLink to="/users" class="block px-2 py-1.5 rounded text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2">
            <Users :size="16" /> {{ t('nav.users') }}
          </RouterLink>
          <RouterLink to="/audit" class="block px-2 py-1.5 rounded text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2">
            <FileWarning :size="16" /> {{ t('nav.audit') }}
          </RouterLink>
        </div>
      </div>

      <!-- System group -->
      <div>
        <button @click="toggle('system')" class="w-full flex items-center justify-between px-4 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700">
          <span class="flex items-center gap-2"><Settings :size="18" /> {{ t('nav.system') }}</span>
          <ChevronDown :size="16" :class="{ 'rotate-180': open.system }" class="transition-transform" />
        </button>
        <div v-show="open.system" class="pl-8">
          <RouterLink to="/config" class="block px-2 py-1.5 rounded text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2">
            <Settings :size="16" /> {{ t('nav.config') }}
          </RouterLink>
          <RouterLink to="/system" class="block px-2 py-1.5 rounded text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2">
            <Cpu :size="16" /> {{ t('nav.systemInfo') }}
          </RouterLink>
          <RouterLink to="/logs" class="block px-2 py-1.5 rounded text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2">
            <ScrollText :size="16" /> {{ t('nav.logs') }}
          </RouterLink>
        </div>
      </div>
    </nav>

    <div class="border-t border-gray-200 dark:border-gray-700 p-4">
      <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
        <Languages :size="16" />
        <select :value="locale" @change="onLocale" class="bg-transparent border border-gray-300 dark:border-gray-600 rounded px-2 py-1">
          <option v-for="l in availableLocales" :key="l" :value="l">{{ l === 'de' ? 'Deutsch' : 'English' }}</option>
        </select>
      </label>
    </div>
  </aside>
</template>
```

- [ ] **Step 2: `AppTopBar.vue` — Top-Statusbar**

```vue
<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/appStore'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import { Sun, Moon, LogOut } from 'lucide-vue-next'

const router = useRouter()
const store = useAppStore()
const authStore = useAuthStore()
const { t } = useI18n()

const props = defineProps({
  dark: { type: Boolean, default: false },
})
const emit = defineEmits(['toggle-theme'])

const proxyCount = computed(() => store.proxies?.length || 0)
const logout = () => { authStore.logout(); router.push('/login') }
</script>

<template>
  <header class="h-14 flex items-center justify-between px-6 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
    <div class="flex items-center gap-3">
      <span class="font-bold text-gray-900 dark:text-white">ModBridge</span>
      <span class="text-xs text-gray-500 dark:text-gray-400">
        {{ proxyCount }} {{ t('topbar.proxies') }}
      </span>
    </div>
    <div class="flex items-center gap-4">
      <button @click="emit('toggle-theme')" class="text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white" :aria-label="t('topbar.toggleTheme')">
        <Moon v-if="!dark" :size="18" /><Sun v-else :size="18" />
      </button>
      <div class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
        <span>{{ authStore.user?.username }}</span>
        <button @click="logout" class="hover:text-red-500"><LogOut :size="18" /></button>
      </div>
    </div>
  </header>
</template>
```

- [ ] **Step 3: `Layout.vue` — Sidebar + TopBar integrieren**

Ersetze das Layout-Gerüst in `frontend/src/components/Layout.vue` so, dass es Sidebar + TopBar + `<router-view>` enthält (bestehende Logik für Theme/Toast beibehalten, nur Struktur ändern):

```vue
<template>
  <div class="flex h-screen bg-gray-50 dark:bg-gray-900">
    <AppSidebar />
    <div class="flex-1 flex flex-col overflow-hidden">
      <AppTopBar :dark="isDark" @toggle-theme="toggleTheme" />
      <main class="flex-1 overflow-y-auto p-6">
        <router-view />
      </main>
    </div>
  </div>
</template>
```

Importiere `AppSidebar` und `AppTopBar`. Behalte Theme-Logik (`isDark`, `toggleTheme`) aus dem bestehenden Layout bei; passe den Theme-Default in Task 10 an.

- [ ] **Step 4: i18n-Strings ergänzen**

`nav.proxies`, `nav.dashboard`, `nav.control`, `nav.devices`, `nav.security`, `nav.users`, `nav.audit`, `nav.system`, `nav.config`, `nav.systemInfo`, `nav.logs`, `topbar.proxies`, `topbar.toggleTheme` in de.json + en.json.

- [ ] **Step 5: Build prüfen**

Run (aus `frontend/`): `npm run build`
Expected: OK.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/AppSidebar.vue frontend/src/components/AppTopBar.vue frontend/src/components/Layout.vue frontend/src/locales/de.json frontend/src/locales/en.json
git commit -m "style(ui): SLZB-style sidebar + topbar layout"
```

---

## Task 10: Frontend — Theme-System: Light-Default, dark-slate Buttons, Card-Update

**Files:**
- Modify: `frontend/src/assets/` (Theme-Preset / CSS) — genaue Datei prüfen via `ls frontend/src/assets/`
- Modify: Theme-Initialisierung in `frontend/src/stores/appStore.js` oder `frontend/src/App.vue` (Theme-State).

**Interfaces:**
- Produces: Light als Default; dark-slate Primär-Buttons (`#212529`).

- [ ] **Step 1: Theme-Default-Quelle finden**

Run: `grep -rn "dark\|theme" frontend/src/stores/appStore.js frontend/src/App.vue | head -30` (Pfade anpassen, falls Theme woanders liegt).

Identifiziere, wo der Theme-Wert initialisiert wird (z. B. `localStorage.getItem('theme')` mit Fallback `'dark'`). Ändere den Fallback auf `'light'`:

```js
const theme = localStorage.getItem('theme') || 'light'
```

Stelle sicher, dass beim ersten Start (kein `localStorage.theme`) Light angewendet wird.

- [ ] **Step 2: PrimeVue-Theme / Tailwind-Variablen für Light + dark-slate Buttons**

Prüfe, ob ein PrimeVue-Preset genutzt wird (z. B. `@primeuix/themes`). Falls ja, definiere/stelle das Preset auf `aura` mit Light-Default und setze die Primärfarbe auf dark-slate. In `frontend/src/main.js` (oder wo `PrimeVue` registriert wird):

```js
import Aura from '@primeuix/themes/aura'
import { definePreset } from '@primeuix/themes'

const ModbridgePreset = definePreset(Aura, {
  semantic: {
    primary: {
      50:  '#f4f5f7', 100: '#e6e8ec', 200: '#cfd3da', 300: '#aab1bd',
      400: '#7d8696', 500: '#5b6577', 600: '#414a5c', 700: '#2f3744',
      800: '#212529', 900: '#16191c', 950: '#0c0e10',
    },
  },
})

app.use(PrimeVue, {
  theme: {
    preset: ModbridgePreset,
    options: { darkModeSelector: '.app-dark' }, // oder '.dark'
  },
})
```

Passe den `darkModeSelector` an die bestehende Dark-Class-Strategie an (z. B. ob `.dark` auf `<html>` gesetzt wird).

- [ ] **Step 3: Login-Card folgt Theme**

In `Login.vue`/`ChangePassword.vue` ist die Card bereits via Tailwind `dark:`-Varianten responsive auf das Theme (in Task 6 umgesetzt). Verifiziere, dass kein hartkodierter dunkler Hintergrund bleibt.

- [ ] **Step 4: Build prüfen**

Run (aus `frontend/`): `npm run build`
Expected: OK.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/
git commit -m "style(theme): default to light; dark-slate primary buttons (SLZB-style)"
```

---

## Task 11: Integration — Vollständiger Build + Smoke-Test

**Files:** keine Code-Änderungen, nur Verifikation.

- [ ] **Step 1: Go-Build (ganzes Projekt)**

Run: `make build`
Expected: Binary `./modbridge` entsteht (Frontend wird gebaut und eingebettet).

- [ ] **Step 2: Go-Tests**

Run: `make test`
Expected: alle Tests PASS (race detector + coverage). Falls einzelne alte Tests durch die `CreateSession`-Signaturänderung rot sind, passe sie an (neues `mustChangePassword`-Argument).

- [ ] **Step 3: Lint**

Run: `make lint`
Expected: keine neuen Linter-Fehler. Behebe ggf. (unbenutzte Imports, Doc-Kommentare für exported Symbole).

- [ ] **Step 4: Smoke-Test — Erststart (admin/admin)**

- Backup bestehende `config.json` + `modbridge.db` falls vorhanden.
- Starte mit leerer Config + leerer DB: `./modbridge`.
- Erwartet im Log: „Default-Login erstellt — Benutzername: admin / Passwort: admin …".
- Öffne `http://localhost:8080` → Login mit `admin`/`admin` → Weiterleitung zu `/change-password`.
- Ändere Passwort (stark) → Re-Login → Dashboard sichtbar, Daten aktualisieren sich.

- [ ] **Step 5: Smoke-Test — Migration**

- Lege `config.json` mit einem bestehenden `admin_pass_hash` (bcrypt eines bekannten Passworts) an, DB leer.
- Starte → Log: „Bestehendes Admin-Passwort migriert".
- Login mit dem bekannten Passwort → Dashboard.

- [ ] **Step 6: Smoke-Test — Stabilität**

- Dashboard offen → im DevTools „Network" → blockiere `/api/proxies/stream` → nach ~30s erscheint „Live-Verbindung gestört"-Hinweis + Daten aktualisieren sich via Polling.

- [ ] **Step 7: Finaler Commit (falls Anpassungen aus Smoke-Test nötig)**

```bash
git add -A
git commit -m "test: integration smoke tests pass; fix fallout from auth/api changes"
```

- [ ] **Step 8: Merge nach main (falls auf Branch gearbeitet wurde)**

Da der Nutzer auf `main` arbeiten will: die Commits sind bereits auf `main`. Falls doch ein Branch genutzt wurde:

```bash
git checkout main
git merge --no-ff <branch>
```

---

## Hinweise zur Ausführung

- Die Tasks sind so geordnet, dass Backend (1–4) vor den abhängigen Frontend-Tasks (5–8) steht; das Redesign (9–10) ist weitgehend unabhängig und kann parallel/zwischengeschoben werden.
- Bei jeder Signaturänderung (`CreateSession`) prüfen, ob alte Aufrufer/Tests angepasst werden müssen (Task 1 Step 7 + Task 11 Step 2).
- Falls DB-Test-Harness fehlt, sind Tasks 2/4 Step-Tests via `t.Skip` abgesichert; die Logik wird stattdessen im Smoke-Test (Task 11) verifiziert.
