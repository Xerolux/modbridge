# Code-Review ModBridge (Stand 2026-09-09, Version 2.0.10.18)

Reine Bestandsaufnahme, keine Änderungen am Code. Schweregrad: **hoch** = Fehlfunktion/Sicherheitslücke im Normalbetrieb, **mittel** = Fehler unter bestimmten Bedingungen oder deutliches Optimierungspotenzial, **niedrig** = Kosmetik, Robustheit, kleinere Performance.

## Werkzeug-Ergebnisse

| Prüfung | Ergebnis |
|---|---|
| `go vet ./...` | sauber |
| `gofmt -l` | sauber |
| `staticcheck ./...` | sauber |
| `golangci-lint` | nicht lauffähig: Binary mit Go 1.25 gebaut, Projekt verlangt 1.26.5 (`make lint` schlägt daher in dieser Umgebung fehl) |
| `go test -race ./...` | siehe Abschnitt „Tests“ am Ende |

## 1. Hoch

| # | Ort | Befund | Vorschlag |
|---|---|---|---|
| H1 | `pkg/tls/tls.go:471` | `MutualTLSConfig` setzt `CAFile` statt `ClientCAFile`. `clientCAs` bleibt nil, Go verifiziert Client-Zertifikate gegen die System-Roots: jedes öffentlich signierte Client-Zertifikat wird akzeptiert. | `ClientCAFile: caFile` setzen. |
| H2 | `pkg/proxy/calibration.go:517,530,560` | Schlägt der Re-Dial nach dem Warm-up fehl, wird `conn` auf nil gesetzt; das `defer conn.Close()` panict dann im HTTP-Handler. | Nil-Guard im Defer oder Rückgabe vor der Zuweisung. |
| H3 | `Makefile:27-31` | `build-all` kompiliert nur `./main.go`; `web.go` (definiert `getWebHandler`) fehlt → Build bricht ab. Außerdem fehlen `CGO_ENABLED=1` und `CC` je Ziel, go-sqlite3 wird als Stub gebaut. | `go build ... .` und CC je Ziel wie in `ci.yml`. |
| H4 | `pkg/logger/logger.go:88-114` | Keine Log-Rotation implementiert. `LogMaxSize/LogMaxFiles/LogMaxAgeDays` werden validiert und per API gesetzt, aber nirgends benutzt. Log-Dateien wachsen unbegrenzt; CLAUDE.md verspricht Rotation. | Rotation implementieren oder Felder und Doku entfernen. |
| H5 | `pkg/database/database.go:22-64` | `sql.Open` ohne DSN-Parameter und ohne `SetMaxOpenConns(1)`. Die PRAGMAs (foreign_keys, busy_timeout, WAL) gelten nur für die erste Pool-Verbindung; weitere Verbindungen haben FK=OFF und busy_timeout=0 → `SQLITE_BUSY` unter Last, CASCADE unzuverlässig. | DSN `file:modbridge.db?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL` oder `SetMaxOpenConns(1)`. |
| H6 | `pkg/config/config.go:252-276` + `docker-compose.yml:15` | `writeConfigFile` nutzt `os.Rename` auf `config.json`; die Datei ist als einzelne Datei bind-gemountet, `rename` schlägt mit EBUSY fehl → jede Konfig-Änderung per UI scheitert im Container. Zudem existiert keine `config.json` im Repo, der Bind-Mount erzeugt dann ein Verzeichnis. | Konfig-Verzeichnis mounten, `config.example.json` beilegen. |
| H7 | `main.go:99,207` + `Dockerfile:52-60` | DB und Logs werden relativ zum cwd `/app` geschrieben, Volumes hängen an `/app/data` und `/app/logs`. Users, Audit, Devices gehen bei `--force-recreate` verloren. | `MODBRIDGE_DATA_DIR` einführen und Pfade daraus ableiten. |

## 2. Mittel

### Sicherheit / API
| # | Ort | Befund | Vorschlag |
|---|---|---|---|
| M1 | `pkg/users/users.go:120-141` | Bei unbekanntem/deaktiviertem User wird vor dem bcrypt-Vergleich abgebrochen → Username-Enumeration per Timing (~1 s Differenz bei Cost 14). | Immer einen Dummy-bcrypt-Vergleich durchführen. |
| M2 | `pkg/api/server.go:181-182` | `authMW`/`csrfMW` ohne Rate-Limiter; Login-Limiter erlaubt 5 req/s pro IP bei bcrypt Cost 14 → CPU-DoS über `/api/login`. | Limiter in alle Ketten, Login auf ~1/s, Burst 5. |
| M3 | `pkg/api/server.go:97` | `requestMeta` vertraut `X-Forwarded-For` bedingungslos; Audit-IP und Device-Zuordnung sind fälschbar. | Nur bei `MODBRIDGE_TRUSTED_PROXIES` auswerten (wie im Rate-Limiter). |
| M4 | `pkg/api/server.go:530` | `/api/status` nutzt `GetSession` ohne Ablaufprüfung; abgelaufene Sessions liefern bis zum stündlichen Cleanup weiter das Proxy-Inventar. | `ValidateSession` verwenden. |
| M5 | `pkg/alerting/alerting.go:214` | Webhook-URL ohne Scheme-/Host-Prüfung, Redirects werden gefolgt, beliebige Header → SSRF auf interne Adressen. (Paket derzeit nicht verdrahtet.) | Nur http(s), private Netze blocken, `CheckRedirect` einschränken. |
| M6 | `pkg/middleware/security.go:29` | HSTS wird bei direktem TLS nie gesetzt (`r.URL.Scheme` leer), `X-Forwarded-Proto` ist client-kontrolliert. | Auf `r.TLS != nil` prüfen. |
| M7 | `pkg/middleware/rate_limiter.go:135` | Bei 10 000 getrackten IPs werden alle neuen Clients pauschal abgelehnt; Angreifer mit vielen Quell-IPs sperrt legitime Neu-Clients aus. | LRU-Eviction statt Ablehnung. |
| M8 | `pkg/audit/audit.go:638` | `LogAction` hält `a.mu` während des bis zu 5 s blockierenden Channel-Sends; bei vollem Puffer serialisieren alle Handler. | Lock vor dem Send freigeben oder nicht-blockierend droppen. |
| M9 | `main.go:335-341` | Shutdown-Reihenfolge: Manager stoppt vor dem HTTP-Server; `Auditor.Close()` wird nie aufgerufen, gepufferte Audit-Einträge gehen verloren. | HTTP-Server → Manager → Auditor.Close. |
| M10 | `main.go:281` | `WriteTimeout: 60s` beendet die SSE-Streams (`/api/proxies/stream`, `/api/logs/stream`) nach 60 s, obwohl die Handler 30 min vorsehen. | In SSE-Handlern `http.NewResponseController(w).SetWriteDeadline(time.Time{})`. |
| M11 | `pkg/metrics/metrics.go:173` | `sorted` wird nie sortiert (kein `sort`-Import): P50/P95/P99 sind Zufallswerte aus dem Ringpuffer. | `sort.Slice` ergänzen. |
| M12 | `pkg/metrics/metrics.go:269` | HELP/TYPE-Zeilen werden pro Proxy wiederholt; Prometheus lehnt doppelte HELP/TYPE ab. | Einmal vor der Schleife ausgeben. |
| M13 | `pkg/api/server.go:207` | `metrics.RegisterProxy/RecordRequest` werden nirgends aufgerufen; `/api/metrics` liefert Totals/Latenzen dauerhaft 0. | Manager/Proxy anbinden oder entfernen. |

### Proxy-Kern
| # | Ort | Befund | Vorschlag |
|---|---|---|---|
| M14 | `pkg/pool/pool.go:84-96` + `pkg/proxy/proxy.go:283-289` | `InitialSize: 1` dialt beim Start synchron; ist das Gerät offline, schlägt `Start()` fehl, der Proxy lauscht nicht (Clients: Connection Refused statt Modbus-Exception 0x0B). | `InitialSize: 0` oder Pre-Populate-Fehler nur loggen. |
| M15 | `pkg/manager/manager.go:611-643` vs. 204-357 | Health-Monitor `checkAndRestartProxies` sieht zwischen `Stop()` und Config-/Map-Update einen „unerwartet gestoppten“ Proxy und startet ihn neu: alte Instanz belegt den Port, die neue schlägt fehl; oder ein bewusst gestoppter Proxy läuft weiter. | Config-Flag/Map vor dem Stop ändern; Monitor gegen In-Flight-Änderungen sperren. |
| M16 | `pkg/proxy/proxy.go:621,636,664,695` | Kein Write-Deadline zum Client; ein nicht lesender Client blockiert Handler-Goroutine, `connSem`-Slot und globalen Limiter-Slot bis `Stop()`. | `SetWriteDeadline` vor jedem Write. |
| M17 | `pkg/proxy/proxy.go:297-303` | HealthChecker öffnet alle 30 s eine eigene TCP-Verbindung, unabhängig von `MaxTargetConns=1`; auf Ein-Session-Geräten stiehlt das die Pool-Session und erzeugt genau die Timeouts, die der Breaker bestraft. | Bei `MaxTargetConns==1` HealthChecker deaktivieren oder Health aus dem Traffic ableiten. |
| M18 | `pkg/proxy/proxy.go:835,940` | `time.Sleep(backoff)` im Retry ignoriert `p.ctx`; `Stop()` hängt bis zu 30 s in `wg.Wait()`. | `select` auf `time.After` und `ctx.Done()`. |
| M19 | `pkg/proxy/transaction.go:83-90` + `adaptive_timeout.go:502` | Request-Budget skaliert mit dem adaptiven Read-Timeout (bis 10× Basis): 5 s Basis × 3 Retries ≈ 200 s pro Client-Request. | Budget an Basis-Timeout koppeln oder Faktor auf 2-3× deckeln. |
| M20 | `pkg/proxy/proxy.go:672-683` | Cache-Invalidierung nach Write nur bei `errFwd == nil`; ein Write, das im Timeout landet, aber das Gerät erreicht hat, lässt veraltete Werte im Cache. | `InvalidateUnit` auch im Fehlerpfad für Write-FCs. |
| M21 | `pkg/proxy/adaptive_timeout.go:488-496` (Perf) | Bei jedem erfolgreichen Request werden unter Lock 100 Samples kopiert und sortiert; blockiert `GetReadTimeout` aller Handler. | Perzentil nur alle N Samples berechnen oder P²-Schätzer. |
| M22 | `pkg/portmanager/port_checker.go:377-381` | `strings.Contains(line, ":502")` matcht auch `:5020`/`:50200` → Port fälschlich belegt; PID-Parsing aus `ss`-Ausgabe scheitert immer (PID 0). Mit `KillProcess` (Z. 422) droht `kill -9 0`. | Regex `[:.]PORT\s`, `pid=(\d+)` extrahieren, `pid <= 1` ablehnen. |
| M23 | `pkg/proxy/load_balancer.go:340-351` (nicht verdrahtet) | `Stop()` hält `lb.mu` während `wg.Wait()`, HealthCheck-Loop braucht `RLock` → Deadlock; `FailCount` teils nicht-atomar. | Lock vor `wg.Wait()` freigeben; atomar konsistent. |
| M24 | `pkg/proxy/priority_queue.go:446,538,590` (nicht verdrahtet) | Send auf `ErrorChan` unter `pq.mu`; ohne Empfänger blockiert das dauerhaft mit gehaltenem Lock. | Nicht-blockierendes `select` oder Paket entfernen. |

### Config / Build / Doku
| # | Ort | Befund | Vorschlag |
|---|---|---|---|
| M25 | `pkg/config/config.go:206-249` | `Load()` ersetzt `m.cfg` komplett; alle Defaults aus `NewManager` (LogMaxSize, RateLimit, SessionTimeout, MaxConnections …) fallen bei fehlendem Key auf 0/"" statt auf den Default. | In die vorbefüllte `m.cfg` hinein unmarshalen. |
| M26 | `pkg/config/validator.go:72` | `Validator.Validate*` wird außerhalb der Tests nirgends aufgerufen; ungültige Werte landen ungeprüft im Betrieb. `PUT /api/config/system` (`handlers_extra.go:190`) schreibt ebenfalls unvalidiert. | Nach `Load()` und in `Manager.Update` validieren. |
| M27 | `pkg/logger/logger.go:51-57` | Existiert am Log-Pfad eine Datei statt eines Verzeichnisses, wird sie kommentarlos gelöscht (Alt-Installation mit `proxy.log`-Datei verliert ihr Log). | Fehler zurückgeben oder umbenennen. |
| M28 | `docker-compose.yml:18-27`, `CLAUDE.md`, `.env.example` | `LOG_LEVEL`, `MODBRIDGE_MAX_CONNECTIONS`, `MODBRIDGE_CIRCUIT_BREAKER_ENABLED`, `MODBRIDGE_CACHE_*`, `MODBRIDGE_HEALTH_CHECK_INTERVAL`, `MODBRIDGE_ALERTING_ENABLED` werden nirgends gelesen. Gelesen werden nur `WEB_PORT`, `DEBUG`, `GO_ENV`, `MODBRIDGE_ENV`, `MODBRIDGE_CSRF_SECRET`, `MODBRIDGE_MULTI_USER`, `MODBRIDGE_TRUSTED_PROXIES`. | Entfernen oder implementieren, Doku angleichen. |
| M29 | `Dockerfile:33`, kein `.dockerignore` | `COPY . .` kopiert `.git`, `node_modules`, lokale `modbridge.db`, `proxy.log/` in den Build-Kontext. | `.dockerignore` anlegen. |
| M30 | `.github/workflows/*.yml` | Alle Actions nur per mutable Major-Tag referenziert (auch Third-Party mit `contents: write`). | SHA-Pinning. |
| M31 | `pkg/openapi/openapi.go:74-80` | Spec enthält nur `/api/v1/devices`, das im Server nicht existiert (37 reale `/api/...`-Routen). | Spec aus Route-Registrierung erzeugen oder Paket entfernen. |
| M32 | `cmd/cli/main.go` | Zweiter Entry-Point mit hartkodierter Version „1.0.0“, anderem Log-Verzeichnis, ohne `WEB_PORT`, ohne History-Cleanup, ohne Reset-Password; driftet von `main.go` weg. | Bootstrap in gemeinsames Paket ziehen oder entfernen. |
| M33 | `CLAUDE.md` | Doku-Drift: Version 1.0.12 (real 2.0.10.18), Go 1.26.1 (real 1.26.5), Abhängigkeitsversionen veraltet, Workflow-Tabelle nennt nicht existierende `main.yml`/`docker.yml`/`headless.yml`, `config.json`-Beispiel nutzt Keys, die es im Struct nicht gibt (`admin_password_hash`, `tls{}`, `logging{}` statt `admin_pass_hash`, `tls_enabled`, `log_level` …), `pkg/batch` als „NOT WIRED UP“ gelistet, wird aber von `poller_batching.go` genutzt. | Doku an Ist-Stand angleichen. |

## 3. Toter Code

Folgende Pakete werden von keinem Nicht-Test-Code importiert (zusammen ca. 9 100 Zeilen):

| Paket | Zeilen | Paket | Zeilen |
|---|---|---|---|
| sanitize | 1576 | degradation | 703 |
| transform | 1336 | cache | 460 |
| rtu | 1303 | cluster | 430 |
| converter | 1255 | alerting | 289 |
| tls | 1093 | timeseries | 258 |
| mapping | 153 | errors | 77 |

Dazu innerhalb genutzter Pakete: `pkg/proxy/load_balancer.go`, `priority_queue.go`, `dead_connection_detector.go` (nicht verdrahtet), `pkg/database/fallback.go` (CircuitBreaker nirgends referenziert, `main.go:102` setzt nur `db = nil`), `pkg/middleware/csrf.go:26` (`secret` gespeichert, nie benutzt; `MODBRIDGE_CSRF_SECRET` ist wirkungslos). `main.go:298` nutzt eigene statische RSA-Suiten ohne PFS statt `pkg/tls` mit sicheren Defaults.

Empfehlung: entfernen oder in einem ADR als „geplant“ dokumentieren. Der tote Code verlängert Build/Test und erzeugt falsche Erwartungen (z. B. Alerting, Log-Rotation, mTLS).

## 4. Niedrig

- `pkg/database/database.go:29-37`: Cleanup-Defer greift nie, weil `err` in allen Folgezeilen geshadowed wird.
- `pkg/database/schema_extended.go:86-103`: Migrationen ohne `PRAGMA user_version` und ohne Transaktion, Fehlererkennung über Fehlertext.
- `pkg/tls/tls.go:123`: `*m.cert` ohne nil-Check → Panic bei leerem Cert-Pfad.
- `pkg/middleware/csrf.go:100`: CSRF-Einträge werden bei Logout nicht entfernt; leeres Token wird trotzdem als Cookie gesetzt.
- `pkg/middleware/cors.go:40,69` + `validator.go:369`: Validator erlaubt `"*"`, Middleware vergleicht literal → Wildcard wirkungslos; Änderungen per API erreichen die laufende Middleware nie.
- `pkg/middleware/rate_limiter.go:159`: `lastRefill = now` auch bei `tokensToAdd == 0` → Bruchteile gehen verloren, Client mit hoher Frequenz bekommt nie Tokens.
- `pkg/api/handlers_extra.go:270`: `startTime` ohne Lock beim ersten Request gesetzt (Data Race), misst Uptime ab erstem Aufruf.
- `pkg/api/handlers_extra.go:477`: Konnektivitätscheck seriell mit 5 s je Proxy → ab ~12 unerreichbaren Zielen über `WriteTimeout`.
- `pkg/api/server.go:1004,1226`: `requirePermissionForUserRoute` dupliziert `requirePermission`; DELETE ohne `id` gibt 500 statt 400.
- `pkg/web/web.go:39`: `Open` auf Verzeichnis gelingt, `ReadFile` scheitert → 500 statt SPA-Fallback.
- `pkg/proxy/stats.go:579-585` (Perf): `RecordRequestStart` iteriert bei jedem Request über die ganze Map unter Write-Lock.
- `pkg/proxy/proxy.go:884-887` (Perf): Request wird bei jedem Retry neu kopiert.
- `pkg/proxy/proxy.go:728` + `modbus/helpers.go:195`: Split-Read mit `quantity > 125` scheitert erst beim Zusammenbauen; Client bekommt 0x0B statt 0x03.
- `pkg/proxy/register_poller.go:233`: Poller-Refreshes laufen an Circuit Breaker und Stats vorbei; bei totem Target bis zu 512 Requests × Retry-Budget je Runde.
- `pkg/proxy/auto_recovery.go:711,801,595`: `DialTimeout` ignoriert `ctx`; `CancelTask` wird von `executeTask` überschrieben; `taskProcessor` pollt im 1-s-Loop.
- `pkg/pool/pool.go:451-469`: `isConnHealthy` erkennt keine halb-offenen Verbindungen (nur Write-Deadline gesetzt/zurückgenommen).
- `pkg/modbus/modbus.go:41`: Protocol-ID wird nicht auf 0 geprüft.
- `pkg/modbus/helpers.go:319`: `ReadRTUFrame` kennt FC 0x16/0x17 u. a. nicht; Request ist dann schon gesendet, Pool-Verbindung wird verworfen.
- `pkg/devices/devices.go:160,283` (Perf): `getMACAddress` enumeriert bei jeder Verbindung alle Interfaces, liefert praktisch immer „unknown“.
- `Makefile:4-5`, `Dockerfile:14`: Fallback-Version „1.0.17“, `BuildTime` fehlt in LDFLAGS, Dockerfile setzt `Version=docker` und `golang:1.26-alpine` ohne Patch-Pinning.
- `.github/workflows/wiki-sync.yml:21`: Token in Clone-URL (landet in `.git/config` des Wiki-Clones).
- `frontend/src/views/Config.vue:343`: CORS-Default-Origins im Frontend dupliziert.

Frontend-Stichprobe ohne Befund: keine Tokens in `localStorage`, kein `v-html`, `innerHTML` in `Dashboard.vue` mit Whitelist, alle Intervalle/EventSources werden in `onUnmounted` aufgeräumt, Router-Guard prüft Auth/Permission/Passwortwechsel. Backend: keine String-Konkatenation in SQL, `rows` überall geschlossen.

## 5. Empfohlene Reihenfolge

1. H1 (mTLS), H2 (Panic), H5 (SQLite-Pool) – kleine Fixes, große Wirkung.
2. H3/M29/H6/H7 – Build und Container-Betrieb.
3. M25/M26 – Config-Defaults und Validierung.
4. M14–M18 – Proxy-Robustheit bei Offline-Geräten und Shutdown.
5. Toten Code und Doku bereinigen (Abschnitt 3, M28, M33).

## Tests
