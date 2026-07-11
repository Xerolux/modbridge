# Update-Modul für ModBridge — Design-Spec

**Datum:** 2026-07-11
**Status:** Entwurf (zur Review)
**Ziel:** Professioneller, vollautomatischer Update-Modul in der WebUI, der neue Versionen aus GitHub Releases erkennt, lädt, verifiziert und den laufenden Go-Prozess sicher ersetzt — über alle unterstützten Architekturen hinweg.

---

## 1. Ziel & Motivation

Heute ist ein Update von ModBridge ein manueller Mehrschrittprozess: Release-Asset von GitHub herunterladen, per SCP auf den Server übertragen, Service stoppen, Binary tauschen, Service starten. Das ist fehleranfällig (siehe der aus einem `+dirty`-Arbeitsbaum gebaute Stand, der diese ganze Session ausgelöst hat) und nicht admin-freundlich.

Der Update-Modul bringt diesen Workflow in die WebUI: Ein Klick auf der System-Seite zeigt die neueste Version, das Changelog, und installiert nach Bestätigung das Update inkl. SHA256-Verifikation, Backup und atomarem Tausch — ohne SSH-Zugang.

**Nicht-Ziele (YAGNI):**
- Delta-Updates / Binary-Patching (volle Binary ist ~13 MB, Download-Strafe akzeptabel)
- Rollback über WebUI nach fehlgeschlagenem Restart (dafür braucht man SSH — dokumentierter Notfall-Pfad)
- Automatische Updates nach Zeitplan (Cron) — erstmal manuell ausgelöst
- Eigene Update-Server-Infrastruktur

---

## 2. Architektur

```
┌──────────────────────────────────────────────────────────┐
│  WebUI /#/system — neuer Bereich "Update"                │
│  ┌────────────────────────────────────────────────────┐  │
│  │ Installiert:  v2.0.7.17  (BuildTime, GOARCH)       │  │
│  │ Neueste:      v2.0.7.18  ●Update verfügbar          │  │
│  │ Changelog (Release-Body, Markdown gerendert)        │  │
│  │ [Update installieren]  [Erneut prüfen]  [GitHub]    │  │
│  │ Fortschrittsanzeige während des Updates             │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
            │ HTTP (neue API-Endpoints)
            ▼
┌──────────────────────────────────────────────────────────┐
│  pkg/api/server.go — neue Routen (admin + CSRF)         │
│    GET  /api/update/check   → CheckForUpdate()           │
│    POST /api/update/perform → PerformUpdate()            │
│    GET  /api/update/status  → Status()                   │
└──────────────────────────────────────────────────────────┘
            │
            ▼
┌──────────────────────────────────────────────────────────┐
│  pkg/updater/ (NEU)                                      │
│    updater.go — CheckForUpdate, PerformUpdate, Status    │
│    github.go  — GitHub-API-Client (releases/latest)      │
│    verify.go  — SHA256-Verifikation gegen checksums.txt  │
│    swap.go    — atomarer Binary-Tausch + Backup          │
│    updater_test.go                                        │
└──────────────────────────────────────────────────────────┘
            │ os.Exit über bestehendes restartSignal-Pattern
            ▼
        systemd (Restart=always) → neue Binary
```

---

## 3. Komponenten im Detail

### 3.1 Backend: `pkg/updater/` (neues Package)

**`updater.go` — Kern-Orchestrierung**

```go
type Updater struct {
    repo        string        // "Xerolux/modbridge"
    httpClient  *http.Client  // Timeout 30s
    log         *logger.Logger
    current     BuildInfo

    mu          sync.RWMutex
    status      UpdateStatus  // geschützter Zustand für /status-Polling
}

type BuildInfo struct {
    Version   string // main.Version, z.B. "2.0.7.17"
    BuildTime string
    GoVersion string
    OS        string // runtime.GOOS
    Arch      string // runtime.GOARCH
}

type ReleaseInfo struct {
    TagName       string    // "v2.0.7.18"
    Version       string    // "2.0.7.18" (ohne v-Präfix)
    PublishedAt   time.Time
    ReleaseNotes  string    // Release-Body (Markdown/Changelog)
    HTMLURL       string    // Link zum GitHub-Release
    Prerelease    bool
    Assets        []Asset
}

type Asset struct {
    Name string // z.B. "modbridge-linux-amd64"
    URL  string // Browser-Download-URL
    Size int64
}

type UpdateStatus struct {
    State       State  `json:"state"`        // idle|checking|downloading|verifying|swapping|restarting|done|error
    Progress    int    `json:"progress"`     // 0-100
    Message     string `json:"message"`      // menschenlesbar, lokalisiert via i18n-Keys
    Error       string `json:"error,omitempty"`
    StartedAt   time.Time `json:"started_at,omitempty"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

Öffentliche API:
- `New(repo string, current BuildInfo, log *logger.Logger) *Updater`
- `(*Updater).CheckForUpdate(ctx) (*ReleaseInfo, error)` — GitHub-API-Cache 60s, um Rate-Limits zu respektieren
- `(*Updater).PerformUpdate(ctx, release *ReleaseInfo) error` — startet Goroutine, nutzt interne Status-Maschine
- `(*Updater).GetStatus() UpdateStatus` — für Polling

**`github.go` — GitHub-API-Client**

- `fetchLatestRelease(ctx)` — `GET https://api.github.com/repos/Xerolux/modbridge/releases/latest`
- Parst JSON: `tag_name`, `name`, `body` (Changelog), `published_at`, `html_url`, `prerelease`, `browser_download_url` je Asset
- **Asset-Auswahl nach Runtime:** mappt `runtime.GOOS`+`runtime.GOARCH` → erwarteten Asset-Namen:
  - `linux/amd64` → `modbridge-linux-amd64`
  - `linux/arm64` → `modbridge-linux-arm64`
  - `linux/arm` → `modbridge-linux-arm` (falls vorhanden)
  - `darwin/*` → `modbridge-darwin-*` (falls vorhanden)
  - `windows/*` → Hinweis "Windows-Update nicht unterstützt, bitte manuell"
- Falls Asset für aktuelle Plattform fehlt → klarer Fehler, kein Tausch
- HTTP-Header: `Accept: application/vnd.github+json`, `User-Agent: ModBridge-Updater/<version>`
- Timeout 30s, Retry bei 5xx/429 (bestehendes axios-Retry-Pattern als Vorbild)

**`verify.go` — SHA256-Verifikation**

- `verifyChecksum(assetBytes []byte, assetName string, checksumsTxt []byte) error`
- Parst `checksums.txt` (Format: `<sha256>  modbridge-linux-amd64`, getrennt durch 2 Leerzeichen — Standard `sha256sum`-Output)
- Vergleicht `sha256(assetBytes)` mit dem erwarteten Wert
- Bei Mismatch → `ErrChecksumMismatch` (klar unterscheidbar von anderen Fehlern)
- Optional später erweiterbar für GPG (kein eigener Key nötig, SHA256 als Minimum)

**`swap.go` — Atomarer Binary-Tausch**

```
PerformUpdate-Ablauf:
  1. Phase "downloading" (0-60%):
     - HTTP GET assetURL → schreibe nach <tempdir>/modbridge.new
     - Progress an Status-Maschine melden (Bytes/Total)
  2. Phase "verifying" (60-80%):
     - HTTP GET checksums.txt aus selben Release-Assets
     - verifyChecksum(newBytes, assetName, checksums)
     - Bei Fehler → Status=error, ggf. temp-Datei löschen, return
  3. Phase "swapping" (80-95%):
     - executablePath, _ := os.Executable()  // echtes Binary, nicht cwd
     - backupPath := executablePath + ".bak." + timestamp
     - os.Rename(executablePath, backupPath)   // Backup
     - os.Rename(tempPath, executablePath)     // atomic auf gleichem Filesystem
     - os.Chmod(executablePath, 0755)
     - Bei Rename-Fehler → Rollback: Rename(backupPath, executablePath)
  4. Phase "restarting" (95-100%):
     - status = "restarting"
     - close(s.restartSignal)  // bestehendes Pattern aus server.go:120
  5. main.go fährt kontrolliert herunter (proxies → HTTP → exit)
     - Kein Sonder-Flag nötig: der Binary-Tausch ist schon passiert,
       os.Exit(0) führt zum systemd-Restart, der automatisch die neue
       Binary startet. Das bestehende restartSignal-Pattern reicht.
  6. systemd Restart=always → neue Binary startet
```

**Wichtige Designentscheidungen im Swap:**
- `os.Executable()` statt hardcodiert `/opt/modbridge/modbridge` — funktioniert auf jedem Installationspfad
- `os.Rename` ist atomic nur **innerhalb desselben Filesystems** → temp-Datei via `os.CreateTemp` in **gleichem Verzeichnis** wie die Binary anlegen (nicht `/tmp`!)
- Backup-Datei-Präfix `.bak.<timestamp>` — mehrere Backups möglich, alteste können später aufgeräumt werden (erstmal kein Auto-Cleanup, YAGNI)

**`updater_test.go` — Tests** (TDD, vor Implementierung)

- `TestCheckForUpdate_ParsesLatestRelease` — Mock-HTTP-Server mit Beispiel-JSON
- `TestCheckForUpdate_CachesResult` — zweiter Aufruf innerhalb 60s sollte keinen neuen HTTP-Request machen
- `TestSelectAsset_ForLinuxAmd64` → "modbridge-linux-amd64"
- `TestSelectAsset_ForLinuxArm64` → "modbridge-linux-arm64"
- `TestSelectAsset_UnsupportedPlatform` → Fehler
- `TestVerifyChecksum_Valid` → OK
- `TestVerifyChecksum_Mismatch` → `ErrChecksumMismatch`
- `TestVerifyChecksum_MissingEntry` → Fehler
- `TestCompareVersions_NewerAvailable` → "2.0.7.18" > "2.0.7.17" = true
- `TestCompareVersions_Equal` → false
- `TestPerformUpdate_RollbackOnRenameFailure` — simulierter Rename-Fehler → Backup bleibt erhalten

### 3.2 API-Routen: `pkg/api/server.go`

Drei neue Routen, eingefügt in bestehende `Routes()`-Methode (Zeile ~200):

```go
// Update-Endpoints — admin-only via requirePermission
mux.HandleFunc("/api/update/check", authMW(s.rbacAdmin(s.handleUpdateCheck)))
mux.HandleFunc("/api/update/perform", csrfMW(s.rbacAdmin(s.handleUpdatePerform)))
mux.HandleFunc("/api/update/status", authMW(s.rbacAdmin(s.handleUpdateStatus)))
```

**`rbacAdmin`-Wrapper** — nutzt bestehendes `requirePermission(w, r, rbac.Permission...)`. Update ist Admin-only (keine Delegation an Operator/Viewer).

**Handler (in `pkg/api/handlers_update.go` neu):**

- `handleUpdateCheck`: ruft `updater.CheckForUpdate(ctx)`, antwortet:
  ```json
  {
    "current_version": "2.0.7.17",
    "current_build_time": "2026-07-11T18:38:00Z",
    "go_version": "go1.26.5",
    "os": "linux",
    "arch": "amd64",
    "latest_version": "2.0.7.18",
    "update_available": true,
    "release_notes": "## Changes\n- fix: ...",
    "release_url": "https://github.com/Xerolux/modbridge/releases/v2.0.7.18",
    "published_at": "2026-07-15T10:00:00Z",
    "prerelease": false,
    "asset_name": "modbridge-linux-amd64",
    "asset_size": 12861136
  }
  ```

- `handleUpdatePerform`: nur POST, **CSRF-required**. Ruft `updater.PerformUpdate(ctx, release)`. Liefert sofort `{job_started: true}` zurück (Update läuft asynchron in Goroutine). Vor Start prüfen: ist schon ein Update in progress? → 409 Conflict.

- `handleUpdateStatus`: GET, für Polling. Liefert `UpdateStatus`-JSON. Frontend pollt alle 1-2s während Update läuft.

### 3.3 Frontend: `frontend/src/views/SystemInfo.vue` erweitern

**Keine neue Route** — Update-Bereich wird als neue Sektion auf der bestehenden System-Seite eingebettet (natürlicher Ort, kein Menü-Müll).

**Neue UI-Elemente:**

```html
<!-- Update-Karte, unter bestehenden System-Infos -->
<section class="update-panel glass-card">
  <header>
    <h3>{{ t('update.title') }}</h3>
    <Badge :severity="updateAvailable ? 'warn' : 'success'">
      {{ updateAvailable ? t('update.available') : t('update.upToDate') }}
    </Badge>
  </header>

  <!-- Versions-Vergleich -->
  <div class="version-grid">
    <div>
      <label>{{ t('update.installed') }}</label>
      <strong>{{ currentVersion }}</strong>
      <small>{{ buildInfo.os }}/{{ buildInfo.arch }} · {{ buildInfo.go_version }}</small>
    </div>
    <div>
      <label>{{ t('update.latest') }}</label>
      <strong>{{ latestVersion }}</strong>
      <small>{{ formatDate(publishedAt) }}</small>
    </div>
  </div>

  <!-- Changelog -->
  <div v-if="releaseNotes" class="changelog" v-html="renderedChangelog"></div>

  <!-- Aktionen -->
  <div class="actions">
    <Button :label="t('update.checkAgain')" icon="pi pi-refresh" @click="checkUpdate" :loading="checking" severity="secondary" />
    <Button v-if="updateAvailable" :label="t('update.install')" icon="pi pi-download" @click="confirmInstall" :disabled="updating" />
    <a :href="releaseUrl" target="_blank">{{ t('update.viewOnGithub') }}</a>
  </div>

  <!-- Fortschritt während Update -->
  <div v-if="updating" class="update-progress">
    <ProgressBar :value="status.progress" />
    <p>{{ t(`update.state.${status.state}`) }}</p>
    <small>{{ status.message }}</small>
  </div>
</section>
```

**State-Management (lokaler `ref`, kein Pinia nötig — Update ist System-seitig):**
- `currentVersion`, `latestVersion`, `updateAvailable`, `releaseNotes`, `status`
- `checkUpdate()` — GET `/api/update/check`
- `confirmInstall()` — Dialog "Version X installieren? Dienst startet neu (~5s Unterbrechung)" → bei Bestätigung POST `/api/update/perform`
- Nach Start: `pollStatus()` alle 1.5s GET `/api/update/status`, bis `state` === `done` oder `error`
- Bei `done`: Toast "Update erfolgreich, Seite lädt neu…", nach 3s `window.location.reload(true)`
- Bei `error`: Toast mit Fehlernachricht, Button reaktivieren

**Auto-Check:** Beim Mounten der System-Seite automatisch `checkUpdate()` (nicht blockierend, im Hintergrund). Vermeidet unnötige GitHub-Requests durch 60s-Cache im Backend.

**Markdown-Rendering des Changelogs:** Nutzt `marked` (schon via Vite verfügbar?) oder einfache Pre-Formatierung. **Empfehlung:** erstmal als `<pre>` mit escapeten HTML rendern (Security: kein ungefiltertes `v-html`), `marked` optional später.

### 3.4 i18n-Erweiterung: `frontend/src/i18n.js`

Neuer `update`-Block in DE und EN:

```js
update: {
  title: 'Update',
  installed: 'Installiert',
  latest: 'Neueste Version',
  available: 'Update verfügbar',
  upToDate: 'Aktuell',
  checkAgain: 'Erneut prüfen',
  install: 'Update installieren',
  viewOnGithub: 'Auf GitHub ansehen',
  confirmTitle: 'Update installieren?',
  confirmMessage: 'Der Dienst wird für ca. 5 Sekunden neu gestartet. Bestehende Proxy-Verbindungen werden unterbrochen.',
  installSuccess: 'Update erfolgreich. Die Seite wird neu geladen.',
  installFailed: 'Update fehlgeschlagen: {error}',
  state: {
    idle: 'Bereit',
    checking: 'Prüfe…',
    downloading: 'Lade herunter…',
    verifying: 'Verifiziere Prüfsumme…',
    swapping: 'Tausche Binary…',
    restarting: 'Starte neu…',
    done: 'Fertig',
    error: 'Fehler'
  }
}
```

---

## 4. Sicherheitsmodell

| Schicht | Maßnahme |
|---------|----------|
| Authorisierung | Alle 3 Endpoints admin-only via `requirePermission(rbac.PermSystemRestart)` (bestehende Permission aus `pkg/rbac/rbac.go:49`, wird schon für `/api/system/restart` genutzt — konsistent). `check`/`status` könnten auf `PermSystemView` heruntergestuft werden, falls Viewer den Status sehen sollen; Default: admin-only für alle drei. |
| CSRF | `POST /api/update/perform` geht durch `csrfMW` (bestehende Middleware) |
| Integrität | SHA256-Verifikation **zwingend** — kein Tausch ohne gültige Checksumme aus `checksums.txt` |
| Backup | Vor jedem Tausch: `modbridge → modbridge.bak.<timestamp>` |
| Rollback | Schlägt `os.Rename` fehl → Backup wird zurückbenannt. Schlägt die neue Binary beim Start fehl → systemd gibt nach 3 Versuchen auf → Admin restauriert `.bak` via SSH (dokumentiert) |
| GitHub-Rate-Limits | Backend cached `/releases/latest` 60s. Maximal 1 Request/min, selbst wenn mehrere Clients pollen |
| Netzwerk | HTTPS zu `api.github.com` und `github.com/.../releases/download/...` |
| Markdown-XSS | Changelog wird **nicht** ungefiltert als `v-html` gerendert — erst escaped `<pre>`, optionales `marked` mit `sanitize` später |

---

## 5. Fehlerbehandlung

| Fehlerfall | Reaktion |
|------------|----------|
| GitHub-API nicht erreichbar / 5xx | `check` liefert klarere Fehlermeldung, UI zeigt "Update-Check fehlgeschlagen", rest der UI funktioniert |
| GitHub-Rate-Limit (403) | Backend erkennt `X-RateLimit-Remaining: 0`, liefert собственных Fehler "Limit erreicht, später erneut" |
| Kein Asset für Plattform | `check` liefert `update_available: true`, aber zusätzlich `asset_unavailable: true` mit Hinweis-Message "Version X verfügbar, aber kein Binary für windows/amd64 — bitte manuell von GitHub laden". UI zeigt gelbes Badge mit Hinweis statt Install-Button. |
| SHA256-Mismatch | `perform` bricht ab, Status=`error`, Message "Prüfsumme stimmt nicht überein — Download abgelehnt", temp-Datei gelöscht |
| Kein Schreibrecht | Rename schlägt fehl, Rollback-Logik, Status=`error`, Message "Keine Schreibrechte auf Binary" |
| Update läuft schon | `perform` liefert 409 Conflict, UI zeigt Toast "Update läuft bereits" |
| Binary nach Restart defekt | systemd gibt nach 3 Starts auf → dokumentierter SSH-Rollback (Notfall-Pfad, nicht via WebUI automatisierbar — systemd kann nicht wissen, ob die Binary "kaputt" oder nur "langsam hochfahrend" ist) |

---

## 6. Test-Strategie

**Unit-Tests (`pkg/updater/updater_test.go`):**
- Version-Compare-Logik (semver-ish, „2.0.7.18" > „2.0.7.17")
- Asset-Auswahl pro Plattform
- SHA256-Verifikation (valid, mismatch, missing entry)
- GitHub-JSON-Parsing (Mock-HTTP-Server via `httptest`)
- Cache-Verhalten (zweiter Aufruf innerhalb 60s macht keinen neuen Request)

**Integration:**
- Echter Download + Swap in einer Testumgebung (z.B. auf der DEV-VM)
- Test gegen manipuliertes Asset (Byte flippen → Checksummen-Fehler)

**Manuelle Verifikation nach Implementierung:**
- Auf 192.168.178.196 einloggen, System-Seite öffnen → Update-Bereich sichtbar
- „Erneut prüfen" → aktuelle Version korrekt angezeigt
- (Wenn zwischenzeitlich ein neues Release entsteht): echtes Update durchspielen

---

## 7. Build-Sequenz

Die Implementierung erfolgt in dieser Reihenfolge (jeder Schritt ist unabhängig testbar):

1. **Backend `pkg/updater/`** — Package + Tests zuerst (TDD)
2. **API-Handler + Routen** in `pkg/api/`
3. **Frontend i18n-Keys** in `i18n.js`
4. **Frontend UI** in `SystemInfo.vue`
5. **Integration** — alles verkabeln, RBAC prüfen
6. **Build + Deploy + manuelle Verifikation** auf 192.168.178.196

---

## 8. Offene Punkte / spätere Erweiterungen

- **GPG-Signaturen:** Aktuell SHA256 als Minimum. GPG wäre stärker, braucht aber verteilte Public-Keys — später.
- **Auto-Cleanup alter Backups:** `.bak`-Dateien sammeln sich. Später: letzte 3 behalten, Rest löschen.
- **Geplante Auto-Updates:** Cron-artig „jede Nacht prüfen, automatisch installieren". Erstmal bewusst manuell.
- **Pre-Release-Option:** Toggle in den Settings, um auch Beta/Alpha-Releases zu sehen. Default: nur stable.
- **Multi-Arch-Build lokal:** Aktuell bauen wir im Docker `golang:1.26-bookworm` nur amd64. Für ARM-Tests müsste der Build-Prozess erweitert werden — nicht Teil dieser Spec.
