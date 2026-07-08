# Design: Login-Redesign, Multi-User-Default & UI-Stabilität

**Datum:** 2026-07-08
**Status:** genehmigt (Übergang zur Implementierung)
**Referenz-Design:** SMLIGHT SLZB-WebUI (`http://192.168.178.192`) — Light-Theme, linke Sidebar, einklappbare Karten

---

## 1. Ziel und Umfang

ModBridge soll (a) ein Login mit **Username + Passwort** (Default `admin`/`admin`) erhalten, (b) stabilere Datenanzeige bekommen („Reload nötig" beseitigen), und (c) optisch dem SLZB-Referenzdesign angenähert werden (Light-Theme, Sidebar, Karten).

**Entwurfsentscheidungen (vom Nutzer getroffen):**

1. **Multi-User (DB) aktivieren** als Default-Modus (statt Single-User/Passwort-only).
2. **Standardpasswort `admin`** beim ersten Start; beim Erstlogin wird zur Passwortänderung gezwungen (starke Policy bleibt für die spätere Änderung).
3. **Stabilität:** SSE + Polling-Fallback (SSE bleibt, zusätzlich regelmäßiges Polling als Sicherheitsnetz).
4. **Bestehende Setups werden automatisch migriert:** vorhandenes Single-User-Passwort bleibt gültig (Hash wird übernommen).
5. **Redesign:** SLZB-Light-Stil nachbauen, PrimeVue+Tailwind beibehalten (kein Bootstrap-Wechsel).
6. **Login-Optik:** zentrierte Card, folgt dem aktiven Theme.

**Nicht im Umfang:** Vollständige Migration auf Bootstrap; neue fachliche Features jenseits Login/Auth/Stabilität/Design.

---

## 2. Architekturüberblick

```
┌──────────────────────────────────────────────────────────────┐
│  Frontend (Vue 3 + PrimeVue + Tailwind)                      │
│  ┌────────────┐  ┌────────────────────────────────────────┐  │
│  │ Sidebar    │  │ Top-Statusbar (Logo, Proxy-Status,     │  │
│  │ (einklapp- │  │ Theme-Toggle, User-Menü)               │  │
│  │  bar)      │  ├────────────────────────────────────────┤  │
│  │            │  │ Hauptbereich: einklappbare Karten       │  │
│  │            │  │ (Dashboard, Control, Config …)          │  │
│  └────────────┘  └────────────────────────────────────────┘  │
│         │  Axios (cookie-basiert, CSRF) + SSE + Polling       │
└─────────┼─────────────────────────────────────────────────────┘
          ▼
┌──────────────────────────────────────────────────────────────┐
│  Backend (Go)                                                │
│  pkg/api   → handleLogin (Multi-User default)                │
│  pkg/auth  → Session, bcrypt, HashPasswordUnchecked (neu)    │
│  pkg/users → EnsureDefaultAdmin (mit Hash-Übernahme, neu)    │
│  main.go   → Bootstrap admin/admin + Migration               │
└──────────────────────────────────────────────────────────────┘
```

Datenfluss bleibt: Vue → `/api/*` → Manager → Proxy → Modbus-Gerät. Live-Updates via SSE (`/api/proxies/stream`) **mit** Polling-Fallback.

---

## 3. Komponenten-Design

### 3.1 Backend — Multi-User als Default + Bootstrap `admin`/`admin`

**`pkg/config/config.go` (Zeile 88):**
- `MultiUser` bleibt als Feld, aber der Default bei der Konfigurations-Initialisierung wird auf `true` gesetzt (Initialisierung in `pkg/config/config.go`, nicht im Struct-Tag). Bestehende `config.json` mit explizitem `multi_user: false` bleibt unangetastet (Nutzer-Willkür bleibt erhalten).

**`pkg/auth/auth.go` (neu):**
- `HashPasswordUnchecked(password string) (string, error)`: bcrypt-Kosten 14, **ohne** Stärke-Validierung. Nur für das gesetzte Default-Passwort `admin` verwendet, da `ValidatePasswordStrength` `"admin"` ablehnt. Normale `HashPassword` bleibt samt Policy für alle Nutzereingaben.

**`pkg/users/users.go` (Erweiterung `EnsureDefaultAdmin`):**
- Neue Methode `EnsureDefaultAdminFromHash(username, existingHash, createdBy string) (created bool, err error)`: idempotent; legt den Admin mit einem **vorgefertigten bcrypt-Hash** an (für Migration statt Neu-Hashen). Setzt `MustChangePassword = false` (das alte Passwort ist dem Nutzer bekannt).
- Bestehendes `EnsureDefaultAdmin(username, password, createdBy)` wird für den Neu-Installationspfad (`admin`/`admin`) so angepasst, dass es **`auth.HashPasswordUnchecked`** statt `HashPassword` verwendet — sonst lehnt die Policy das Passwort `admin` ab. `MustChangePassword = true` bleibt. (Alternativ: `EnsureDefaultAdmin` erhält einen Parameter `bypassPolicy bool`; im Plan zu entscheiden, welche Variante sauber ist.)

**`main.go` Bootstrap (ersetzt Zeilen 65–85 und 108–122):**
- Nur einmal, zentral, im Multi-User-Kontext:
  - **Fall A — Neuinstallation** (`AdminPassHash == ""` und keine User in DB):
    - Admin `admin` mit Klartext-Passwort `admin` anlegen via `EnsureDefaultAdmin("admin", "admin", "system")`.
    - `MustChangePassword = true`.
    - `AdminPassHash` nicht mehr separat in `config.json` pflegen (veraltet im Multi-User-Modus).
    - Log: `"SYSTEM-INFO: Default-Login erstellt — Benutzername: admin / Passwort: admin — BITTE beim ersten Login ändern."`
  - **Fall B — Migration bestehendes Single-User-Setup** (`AdminPassHash != ""` und keine User in DB):
    - Admin `admin` anlegen via `EnsureDefaultAdminFromHash("admin", cfg.AdminPassHash, "system")`.
    - Altes Passwort bleibt gültig.
    - `MustChangePassword = cfg.ForcePasswordChange` (nur true, wenn vorher schon ein Wechsel erzwungen war).
    - Log: `"SYSTEM-INFO: Bestehendes Admin-Passwort wurde migriert (Benutzername: admin)."`
  - **Fall C — User bereits vorhanden:** no-op.
- Die bisherige `generateSecurePassword`-basierte Logik entfällt.
- Single-User-Pfad in `handleLogin` (`pkg/api/server.go:544–558`) bleibt als Fallback für `multi_user=false` erhalten, wird aber nicht mehr standardmäßig durchlaufen.

**`pkg/api/server.go`:**
- `finalizeLogin` (Zeile 564): setzt `Session.MustChangePassword` aus dem übergebenen Flag.
- `handleMe` (Zeile 611): liefert `must_change_password` im JSON mit.
- **Neu: `/api/logout` POST** (`handleLogout`): invalidiert die Session serverseitig via `auth.InvalidateSession(token)` (neue kleine Methode) und schreibt `audit.LogLogout`. Route registriert unter `publicMW` + `csrfMW` (CSRF, weil POST).
- `/api/setup` (`handleSetup`, Zeile 472): als `deprecated` markiert, gibt immer `410 Gone` mit Hinweis zurück (war seit dem Auto-Bootstrap ohnehin tot). Nicht physisch entfernen, um keine API-Breakage zu riskieren; nur klar dokumentieren.

### 3.2 Login- & Auth-Frontend

**`frontend/src/views/Login.vue`:**
- Username-Feld **immer** sichtbar (das `v-if="multiUser"` entfällt), mit `admin` als Platzhalter/Vorausfüllung bei leerer Eingabe.
- `handleLogin` sendet immer `{username, password}`.
- Nach `auth.login(...)`: falls `mustChangePassword === true` → `router.push('/change-password')`, sonst `router.push('/')`.

**`frontend/src/stores/auth.js`:**
- `login()` wertet `force_password_change` aus der `/api/login`-Antwort aus, speichert `mustChangePassword` (neuer State) und gibt es zurück.
- **Auth-Cache-Ersatz:** die 5-Sekunden-Cache-Logik (`AUTH_CHECK_CACHE_MS`, Zeile 7, 36–39) wird ersetzt durch eine **„verify-once-per-page-load"-Strategie**: einmal `/api/me` beim ersten Routenwechsel nach einem Page-Load; danach vertraut der Store auf den Zustand und reagiert nur noch auf echte `401` (über `emitUnauthorized`). Dadurch entfällt das Problem „Cache sagt true, Session ist serverseitig schon weg". Bei `401` → sofortiger Reset + Redirect.
- `logout()` ruft zusätzlich `POST /api/logout` auf (vor dem clientseitigen Cookie-Löschen), damit die Server-Session zuverlässig endet.

**Neue Route `/change-password` + `frontend/src/views/ChangePassword.vue`:**
- Formular: aktuelles Passwort, neues Passwort, Bestätigung.
- `POST /api/config/password` (existiert bereits).
- Nach Erfolg: `mustChangePassword = false`, Re-Login-Hinweis (Sessions werden serverseitig invalidiert → Nutzer muss neu einloggen).
- Router-Guard in `frontend/src/router/index.js`: blockiert alle authentifizierten Routes außer `/change-password`, solange `auth.mustChangePassword === true`.

### 3.3 Stabilität — SSE + Polling-Fallback

**Zentrale Proxy-Liste (Single Source of Truth):**
- `frontend/src/stores/appStore.js` behält `proxies` als alleinige Quelle.
- `Dashboard.vue` (Zeile 193) und `Control.vue` (Zeile 331) leiten davon ab (`storeToRefs` / computed), statt eigener lokaler Refs. Mutations werden sofort konsistent sichtbar.

**Polling-Fallback neben SSE (Dashboard & Control):**
- Zusätzlich zum `useEventSource('/api/proxies/stream')` ein `useAutoRefresh(store.fetchProxies, { intervalMs: 10000 })` als Sicherheitsnetz.
- **Data-Watchdog:** ein Watcher auf `lastMessageAt`/`isConnected` des EventSource. Ist seit > 30 s keine Nachricht eingetroffen ODER `isConnected === false`, wird (a) sofort gepollt und (b) ein dezenter Hinweis eingeblendet („Live-Verbindung gestört — Daten werden regelmäßig aktualisiert").

**`frontend/src/utils/eventSource.js`:**
- `isConnected` wird erst `true`, **nachdem die erste Nachricht empfangen** wurde (nicht schon bei `onopen`). `onopen` setzt nur `isConnecting = false`.
- Heartbeat-Liveness: falls der Server einen SSE-Heartbeat sendet, wird er gezählt; sonst übernimmt der Watchdog in den Views die Liveness-Erkennung (Zeit seit letzter Nachricht).

**`frontend/src/axios.js` 401/403-Handling:**
- **Transient-Redirect-Gate:** ein Modul-Flag `isHandlingUnauthorized` + kurze Sperre (~1 s), sodass ein `401`-Sturm bei parallelen Requests nur **einen** Redirect und **einen** `resetState` auslöst.
- **403-Behandlung:** Toast („Keine Berechtigung für diese Aktion") statt stiller Reject; kein Redirect.

**Fehler nicht mehr verschlucken:**
- `frontend/src/views/SystemInfo.vue` (Zeile 273–284): echtes `error`-Ref + Fehlerbanner/Retry statt `console.error` + Null-Anzeige.
- `frontend/src/stores/appStore.js` `fetchStatus` (Zeile 167) und `fetchWebPort` (Zeile 144): setzen das `error`-Flag statt nur `console.error`.

### 3.4 Redesign — SLZB-Stil (PrimeVue+Tailwind)

**Theme-System:**
- App-Default: **Light/White-Theme** (SLZB-Light: Body `#ffffff`, Text `#212529`).
- Bestehender Dark/Light-Toggle bleibt voll funktionsfähig; Default beim ersten Start = Light.
- PrimeVue-Theme via `usePreset`/Tailwind-CSS-Variablen; Design-Tokens zentral in `frontend/src/assets/` oder `tailwind.config.js`.

**Layout-Neustrukturierung:**
- **Linke Sidebar** (`frontend/src/components/Layout.vue`): navigation mit einklappbaren Gruppen:
  - **Proxies** ▸ (Dashboard, Control)
  - **Geräte** (Devices)
  - **Sicherheit** ▸ (Users, Audit)
  - **System** ▸ (Config, SystemInfo, Logs)
- **Top-Statusbar**: ModBridge-Logo + „ModBridge"-Titel, Proxy-/Verbindungs-Status-Indikatoren, Theme-Toggle, User-Menü (Username, Logout).
- **Einklappbare Karten** auf Dashboard/Status-Seiten (PrimeVue `Panel` mit `toggleable`), im Stil der SLZB „Allgemeiner Status"-Karten.
- **Dark-slate Primär-Aktionen** (`#212529` Button-BG, weißer Text) wie im SLZB-Referenzdesign (auch im Light-Theme).
- **Sprachumschalter** (de/en existiert) prominent in der Sidebar (Combobox-ähnlich, SLZB-Stil).

**Login-Optik:**
- Zentrierte Card auf hellem Grund (Light-Theme) bzw. dunklem Grund (Dark-Theme). Card folgt dem aktiven Theme.
- ModBridge-Logo oben, Username-Feld (Default „admin"), Passwort-Feld, primärer „Anmelden"-Button (dark-slate).
- Mobil-tauglich (volle Breite, Stack-Layout).

### 3.5 Aufräumen
- `/api/setup`: deprecated (410 Gone), siehe 3.1.
- `auth.InvalidateSession(token)` neu (kleine Helfer-Methode) für `/api/logout`.

---

## 4. Datenmodell / Schnittstellen

**`auth.Session` (erweitert):**
```go
type Session struct {
    Token, UserID, Username, Role string
    ExpiresAt                     time.Time
    MustChangePassword            bool   // neu
}
```

**`/api/login` Antwort (erweitert):**
```json
{ "success": true, "force_password_change": <bool> }   // unverändert, wird jetzt frontendseitig ausgewertet
```

**`/api/me` Antwort (erweitert):**
```json
{ "user_id": "...", "username": "...", "role": "...", "permissions": [...], "must_change_password": <bool> }
```

**`/api/logout` (neu):** `POST`, CSRF-geschützt. Antwort `{ "success": true }`. Invalidiert die Server-Session.

**`users.User` (unverändert):** nutzt bereits `MustChangePassword` (`schema_extended.go`).

---

## 5. Fehlerbehandlung

- **Bootstrap schlägt fehl** (z. B. DB nicht beschreibbar): `log.Fatalf` mit klarem Hinweis; App startet nicht. Sicherer als stiller Fallback.
- **`EnsureDefaultAdminFromHash` mit ungültigem Hash:** Fehler zurückgeben, in Log ausgeben, App weiterlaufen lassen (Single-User-Pfad bleibt nutzbar als Notnagel — Multi-User aktiviert sich nicht). Genaueres wird im Plan geklärt; Grundannahme: ungültiger Hash → Nutzer muss `/api/config/password` nutzen oder Single-User bleibt aktiv, bis gültiges Setup existiert.
- **SSE bricht ab:** Watchdog übernimmt, Polling sichert Daten; Hinweis an den Nutzer.
- **`/api/logout` schlägt fehl:** clientseitiges Cookie-Löschen läuft trotzdem; der Nutzer ist praktisch ausgeloggt (Server-Session läuft ohnehin ab).
- **403:** Toast, kein Redirect.
- **Passwortänderung schlägt fehl:** Toast mit Fehler aus Backend-Antwort; Felder bleiben erhalten.

---

## 6. Testen

**Backend (Go, table-driven wo möglich):**
- `auth.HashPasswordUnchecked`: hashbar, stimmt mit `CheckPasswordHash` überein.
- `users.EnsureDefaultAdminFromHash`: idempotent; ungültiger Hash → Fehler; gültiger Hash → Admin mit übernommenem Passwort, `MustChangePassword=false`.
- Bootstrap-Pfade in `main.go` via extrahierter Funktion testbar (Boostrap-Logik in `bootstrapUsers(...)` kapseln): Fall A/B/C.
- `/api/login` mit `admin`/`admin` → 200 + `force_password_change=true`.
- `/api/me` liefert `must_change_password`.
- `/api/logout` invalidiert die Session.
- `handleChangePassword` nach Änderung: neue Session hat `must_change_password=false`.

**Frontend (Komponenten-/Store-Tests wo vorhanden):**
- `auth.login` setzt `mustChangePassword` korrekt aus Antwort.
- Router-Guard leitet auf `/change-password`, wenn Flag true.
- Watchdog triggert Polling nach simuliertem SSE-Ausfall (Mock).
- 401-Gate: mehrere parallele 401 → nur ein Redirect.
- Layout rendert Sidebar + Top-Statusbar; Login-Card zentriert.

**Manuell (über Browser, optional):**
- Erststart mit leerer `config.json` + DB → Login `admin`/`admin` → Passwort-Änderung erzwungen.
- Bestehende `config.json` mit Hash → Migration → Login mit altem Passwort möglich.
- Dashboard: SSE kappen (Netzwerk-Tab) → nach ~30 s Polling-Hinweis + aktualisierte Daten.

---

## 7. Offene Punkte / bewusste Annahmen

1. **`/api/setup` Deprecated-Behandlung (410):** könnte bestehende Aufrufer brechen — wird als sehr unwahrscheinlich eingeschätzt (Endpunkt war seit Auto-Bootstrap nicht erreichbar). Im Plan wird dies als separate, revertierbare Änderung gekennzeichnet.
2. **`MustChangePassword` für migrierten Admin:** auf `cfg.ForcePasswordChange` gesetzt. Falls `false` (regulärer Betrieb), wird keine Änderung erzwungen — erwünscht, da der Nutzer sein Passwort kennt.
3. **PrimeVue-Theme-Wechsel zu Light-Default:** muss geprüft werden, ob das aktuelle Theme-Setup in `frontend/src/stores/appStore.js` (oder wo gepflegt) einen sauberen Default zulässt; ggf. kleine Anpassung der Initialisierung. Wird im Plan als eigener Schritt geführt.
4. **`AdminPassHash` nach Migration:** bleibt zur Abwärtskompatibilität in `config.json` stehen, wird im Multi-User-Modus aber nicht mehr ausgewertet. Kein automatisches Entfernen (vermeidet Überraschungen).
