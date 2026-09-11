# CLAUDE.md — ModBridge AI Assistant Guide

This file describes the codebase structure, development workflows, and conventions for AI assistants working on ModBridge.

---

## Project Overview

**ModBridge** is a Modbus TCP Proxy Manager with a web UI. It proxies Modbus TCP traffic, exposing a REST API and Vue.js frontend for configuration and monitoring. The application is written in Go (backend) with a Vue.js 3 frontend embedded into the binary.

**Current version:** see `version.txt`
**Go version:** see the `go` directive in `go.mod` (currently 1.26.5)
**Node version:** 24 (CI/CD, `frontend/`)

Version numbers are deliberately not repeated here — read them from
`version.txt`, `go.mod` and `frontend/package.json`.

---

## Repository Layout

```
modbridge/
├── main.go                    # Entry point: DB → config → logger → manager → API server
├── web.go                     # Static file serving for embedded frontend
├── go.mod / go.sum            # Go module definition (CGO_ENABLED=1 required)
├── Makefile                   # All build/test/lint/docker targets
├── Dockerfile                 # Multi-stage Docker build
├── docker-compose.yml         # Container orchestration
├── config.example.json        # Example runtime configuration (copy to config.json)
├── version.txt                # Current version string
├── .env.example               # Environment variable template
├── pkg/                       # All Go packages (33, ~25k lines of non-test Go)
│   ├── api/                   # HTTP handlers, routes, middleware composition
│   ├── manager/               # Proxy lifecycle management
│   ├── proxy/                 # Proxy instance: stats, circuit breaker, load balancer, alerting, auto-recovery
│   ├── config/                # Config loading, validation, JSON unmarshaling
│   ├── auth/                  # Authentication, sessions, password hashing (bcrypt)
│   ├── modbus/                # Modbus TCP frame read/write, helpers
│   ├── logger/                # Structured logging
│   ├── database/              # SQLite3 (CGO), schema, fallback mode
│   ├── middleware/            # CORS, security headers, rate limiter, CSRF, cache
│   ├── metrics/               # Prometheus metrics export
│   ├── rbac/                  # Role-based access control (Admin/Operator/Viewer/Auditor)
│   ├── audit/                 # Async audit logging with CSV/JSON export
│   ├── alerting/              # Webhook notifications (Slack, Teams, Discord)
│   ├── users/                 # Multi-user management
│   ├── openapi/               # OpenAPI/Swagger spec generation
│   ├── tls/                   # mTLS certificate handling
│   ├── devices/               # Device tracking
│   ├── pool/                  # Connection pooling
│   ├── batch/                 # Batched register reads (used by pkg/proxy poller)
│   ├── cache/                 # Response cache — NOT WIRED UP
│   ├── cluster/               # HA coordination — NOT WIRED UP
│   ├── converter/             # Value conversion — NOT WIRED UP
│   ├── degradation/           # Graceful degradation — NOT WIRED UP
│   ├── mapping/               # Register transformations — NOT WIRED UP
│   ├── rtu/                   # Modbus RTU support — NOT WIRED UP
│   ├── sanitize/              # Input sanitizing — NOT WIRED UP
│   ├── timeseries/            # Metric history — NOT WIRED UP
│   ├── transform/             # Register transforms — NOT WIRED UP
│   ├── portmanager/           # Dynamic port allocation
│   ├── web/                   # Embedded frontend assets (dist/ copied here at build time)
│   └── testing/               # Test utilities: mockmodbus/, integration/, performance/
├── cmd/                       # Additional entry points
│   ├── modbridge-headless/    # Build variant without the WebUI
│   ├── cli/                   # CLI tooling
│   └── openapi/               # Writes docs/openapi.json
├── frontend/                  # Vue.js 3 + Vite source
│   ├── src/
│   │   ├── views/             # Page-level Vue components
│   │   ├── components/        # Reusable UI components
│   │   ├── stores/            # Pinia state stores
│   │   ├── router/            # Vue Router configuration
│   │   └── locales/           # i18n translations (de/en)
│   ├── vite.config.js         # Vite build config; dev proxy → :8080
│   └── package.json           # Node dependencies
├── docs/                      # Extended docs (German + English, ADRs)
│   └── adr/                   # Architecture Decision Records
└── .github/
    ├── workflows/             # CI/CD pipelines (4 workflows)
    ├── ISSUE_TEMPLATE/        # Bug/feature templates
    └── dependabot.yml         # Automated dependency updates
```

---

## Architecture

```
Vue.js Frontend (SPA)
      │ HTTP/REST
      ▼
pkg/api/         ← HTTP server, routing, middleware chain
      │
      ▼
pkg/manager/     ← Proxy lifecycle (create/start/stop/delete)
      │
      ▼
pkg/proxy/       ← Per-proxy goroutine: accepts TCP, reads Modbus frames,
                   forwards to target, tracks stats, applies circuit breaker
      │
      ▼
Modbus TCP Target Device
```

**Startup order** (`main.go`):
Database → Config → Logger → Manager → Authenticator → API Server → Graceful Shutdown

**Embedded frontend:** `make build-frontend` runs `npm run build` inside `frontend/`, then copies `dist/` to `pkg/web/dist/`. The Go binary embeds this directory and serves it statically.

---

## Build System

All development tasks go through `make`. Run `make help` to see all targets.

### Essential Commands

```bash
make build            # Build frontend then compile Go binary (./modbridge)
make build-frontend   # Vue.js only: npm install + vite build → pkg/web/dist/
make build-all        # Cross-compile the released targets: linux amd64/arm64/arm (needs cross-toolchains)
make run              # Build and run locally
make dev              # Live reload with `air` (requires: go install github.com/air-verse/air)
make test             # Run all tests with race detector + coverage
make coverage         # Generate coverage.html
make lint             # Run golangci-lint
make fmt              # gofmt all Go files
make vet              # go vet all packages
make clean            # Remove build artifacts
make docker-build     # Build Docker image
make docker-compose-up # Start via docker-compose
make deps             # Download Go dependencies
make update-deps      # Update Go dependencies
```

### Build Requirements

- **Go 1.26.5+** with `CGO_ENABLED=1` (required for `go-sqlite3`)
- **GCC** (for SQLite CGO compilation; cross-compilers for arm: `gcc-aarch64-linux-gnu`, `gcc-arm-linux-gnueabihf`)
- Cross-compiling always needs `CGO_ENABLED=1` plus a matching `CC`; a CGO-less build compiles but fails at runtime with "Binary was compiled with CGO_ENABLED=0"
- **Node 24+** (frontend build)
- **npm** (frontend dependency management)

### Version Injection

Version and build time are injected at compile time:

```bash
go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" -trimpath
```

---

## Testing

### Running Tests

```bash
make test                      # Recommended: race detector + coverage
go test -v ./...               # Verbose output for all packages
go test -v ./pkg/auth/...      # Single package
go test -run TestFunctionName  # Single test
```

### Test Organization

- Test files live next to the source they cover (`pkg/foo/foo_test.go`)
- **Mock Modbus server:** `pkg/testing/mockmodbus/` — use for proxy/modbus tests
- **Integration tests:** `pkg/testing/integration/`
- **Performance tests:** `pkg/testing/performance/`

### Test Patterns

- Use **table-driven tests** for validation logic (see `pkg/config/`)
- Use `httptest.NewRequest` / `httptest.NewRecorder` for HTTP handler tests
- Name tests: `Test<Function><Scenario>` (e.g., `TestHashPasswordEmptyInput`)
- Use `t.Parallel()` where safe

---

## Configuration

### config.json

Primary configuration file. `config.example.json` in the repo root is a
complete, valid example — copy it to `config.json` rather than writing one from
scratch. The keys are flat and come straight from the `Config` struct in
`pkg/config/config.go`:

```json
{
  "web_port": ":8080",
  "admin_pass_hash": "<bcrypt>",
  "multi_user": true,
  "log_level": "INFO",
  "log_max_size": 100,
  "log_max_files": 10,
  "log_max_age_days": 30,
  "tls_enabled": false,
  "tls_cert_file": "",
  "tls_key_file": "",
  "session_timeout": 24,
  "cors_allowed_origins": ["http://localhost:8080"],
  "rate_limit_enabled": true,
  "rate_limit_requests": 60,
  "rate_limit_burst": 100,
  "max_connections": 1000,
  "proxies": [
    {
      "id": "uuid",
      "name": "Proxy Name",
      "listen_addr": ":5020",
      "target_addr": "192.168.1.100:502",
      "enabled": true,
      "connection_timeout": 10,
      "read_timeout": 5,
      "max_retries": 3,
      "max_target_conns": 1
    }
  ]
}
```

A key that is absent keeps its compiled-in default (`config.DefaultConfig`);
only keys actually present in the file override it. Saving through the API
rejects a change that would make the configuration invalid.

`log_max_size` (MB), `log_max_files` and `log_max_age_days` drive log rotation
in `pkg/logger`. Rotation is off when `log_max_size` is 0.

### Environment Variables

Only these are read by the code. Anything else found in an older
`docker-compose.yml` or README is a no-op.

| Variable | Default | Purpose |
|----------|---------|---------|
| `WEB_PORT` | `:8080` | HTTP bind address |
| `LOG_LEVEL` | `INFO` | `DEBUG`/`INFO`/`WARN`/`ERROR`; overrides `log_level` |
| `MODBRIDGE_DATA_DIR` | `.` | Directory for `modbridge.db` |
| `MODBRIDGE_LOG_DIR` | `proxy.log` | Directory for log files |
| `MODBRIDGE_CONFIG` | `config.json` | Path of the config file |
| `MODBRIDGE_CSRF_SECRET` | — | CSRF token secret; required in production |
| `MODBRIDGE_MULTI_USER` | — | Force multi-user auth on/off |
| `MODBRIDGE_TRUSTED_PROXIES` | — | CIDRs whose `X-Forwarded-For` is trusted |
| `MODBRIDGE_ENV` / `GO_ENV` | — | `production` enables production checks |
| `DEBUG` | — | Enable pprof endpoints |
| `TZ` | `UTC` | Timezone (read by the Go runtime, not by ModBridge) |

Everything else is configured in `config.json`, per proxy where it belongs
(connection limits, circuit breaker, health checks, response cache).

**Configuration priority:** Environment variables > `config.json` > compiled defaults

**Response caching** is configured per proxy in `config.json` (`cache_enabled`,
`cache_ttl_ms`, `poll_interval_ms`), not through environment variables. It is
off by default: a cached register is not the live value. See
`docs/Konfiguration.md` for the guarantees the cache keeps (writes are never
cached and invalidate the unit; exceptions are never cached).

---

## Go Code Conventions

### Naming

- Package names: lowercase, single word (`modbus`, `portmanager`, not `port_manager`)
- Exported symbols: `CamelCase`
- Unexported symbols: `camelCase`
- Receiver parameter: short type abbreviation (`m *Manager`, `p *Proxy`)
- Constants: `UPPER_SNAKE_CASE` or `AllCaps`

### Patterns

**Error handling:** Return errors early with `%v` or `%w` wrapping:
```go
if err != nil {
    return fmt.Errorf("failed to start proxy: %w", err)
}
```

**Concurrency:**
- `sync.RWMutex` for protecting shared maps (prefer `RLock` for reads)
- `sync.WaitGroup` for goroutine lifecycle
- `context.Context` for cancellation and timeouts
- Semaphore channel pattern for connection limits: `connSem := make(chan struct{}, maxConns)`
- Atomic counters (`sync/atomic`) for stats

**Resource cleanup:** Always use `defer` for `Close()`, `Unlock()`, etc.

**Middleware:** Functional composition pattern in `pkg/api/server.go`:
```go
handler = middleware.CORS(handler)
handler = middleware.RateLimit(handler)
handler = middleware.CSRF(handler)
```

**Configuration validation:** Flexible JSON unmarshaling in `pkg/config/` supports both string and numeric types for resilience.

### Imports

Group imports in order: stdlib → external → internal (enforced by `gofmt`):
```go
import (
    "context"
    "fmt"

    "github.com/google/uuid"

    "modbridge/pkg/config"
)
```

### Comments

- All exported types and functions must have doc comments
- Inline comments only for non-obvious logic
- Do not add comments to changed code unless the logic needs explanation

---

## Frontend Conventions

**Stack:** Vue.js 3 (Composition API), Pinia, Vue Router, PrimeVue 4, Tailwind CSS, Axios, vue-i18n

- Components use `<script setup>` syntax
- State management via Pinia stores in `frontend/src/stores/`
- All user-facing strings go through `vue-i18n` (locales in `frontend/src/locales/`)
- API calls use Axios; dev proxy redirects `/api/*` to `:8080`
- UI components from PrimeVue; icons from `lucide-vue-next`
- Dashboard layouts use `gridstack` for drag-and-drop

**Dev server:**
```bash
cd frontend
npm install
npm run dev   # Starts Vite dev server with API proxy to :8080
```

---

## CI/CD Pipelines

Four workflows in `.github/workflows/`:

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci.yml` | push/PR to main | Version stamp, quality (fmt/vet/lint), tests, benchmark guardrails, frontend build, E2E, binaries, headless binaries, auto-release |
| `release.yml` | tag `v*` | GitHub release with binaries, headless variants and checksums |
| `pages.yml` | push to main | GitHub Pages docs site |
| `wiki-sync.yml` | push to main | Sync GitHub Wiki |

Released platforms are `linux/amd64` and `linux/arm64` (plus `linux/arm` for
the headless variant). There are no Windows or macOS builds: CGO is required
for `go-sqlite3`, so every target needs a matching cross-toolchain.

**Release process:** Push a tag matching `v*` → CI builds the released platforms → creates a GitHub release with checksums.

---

## Database

- **SQLite3** via `github.com/mattn/go-sqlite3` (requires CGO)
- Pragmas (`journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout`, `foreign_keys`) are passed through the DSN, because they are per-connection; the pool is capped at one connection since SQLite serialises writers
- Database file: `modbridge.db` (auto-created on first run)
- If DB initialization fails, `main.go` leaves `db` nil and the app runs without persistence — users, audit and device history are then unavailable. `pkg/database/fallback.go` holds an unused circuit breaker and is not part of this path
- Schema defined in `pkg/database/schema_extended.go`

---

## Docker

```bash
# Build image
make docker-build

# Run with docker-compose
make docker-compose-up

# Publish to registry
make docker-push
```

Multi-arch build targets: `linux/amd64`, `linux/arm64` (via QEMU in CI).

Default ports:
- `:8080` — Web UI & API
- `:5020-5030` — Modbus proxy ports
- `:9090` — Prometheus metrics

---

## Key Gotchas

1. **CGO required:** `go-sqlite3` needs CGO. Always build with `CGO_ENABLED=1`. Cross-compilation needs the appropriate cross-compiler installed.
2. **Frontend must be built before Go binary:** `make build` handles this, but `go build` alone won't include updated frontend assets.
3. **Admin password:** On first run with no config, a random admin password is generated and printed to stdout. Check logs.
4. **Version injection:** The `Version` and `BuildTime` variables in `main.go` are only populated via `ldflags`. `go run main.go` shows `dev`/empty.
5. **No database:** If the database fails to initialize, the app continues without persistence (`db` is nil; users, audit and device history are unavailable) — check startup logs. `pkg/database/fallback.go` contains an unused circuit breaker, it is not wired into this path.
6. **Frontend output sanitization:** Vite config strips underscore-prefixed filenames from `dist/` to avoid Go embed issues.
7. **Data locations:** By default the database and logs are written relative to the working directory. In a container set `MODBRIDGE_DATA_DIR`, `MODBRIDGE_LOG_DIR` and `MODBRIDGE_CONFIG` to the mounted volumes, or the state is lost when the container is recreated (`docker-compose.yml` does this).
8. **Second entry point:** `cmd/modbridge-headless/` builds the WebUI-less variant. `cmd/cli/` is a separate, less complete bootstrap — prefer `main.go` when changing startup behaviour.

---

## Dependencies

### Go (minimal)

Versions live in `go.mod`; this table only says what each module is for.

| Module | Purpose |
|--------|---------|
| `github.com/google/uuid` | UUID generation |
| `golang.org/x/crypto` | bcrypt password hashing |
| `github.com/mattn/go-sqlite3` | SQLite3 (CGO) |

### Frontend (key)
| Package | Purpose |
|---------|---------|
| `vue` ^3.5 | UI framework |
| `pinia` ^3.0 | State management |
| `vue-router` ^5.0 | Client-side routing |
| `primevue` ^4.5 | UI component library |
| `axios` ^1.13 | HTTP client |
| `vue-i18n` ^11.3 | Internationalization |
| `tailwindcss` ^3.4 | Utility CSS |
| `vite` ^8.0 | Build tool |
| `gridstack` ^12.4 | Dashboard drag-and-drop |

---

## Architecture Decision Records

See `docs/adr/` for rationale on key design decisions:
- `001-multi-user-rbac.md` — Why RBAC with 4 roles
- `002-audit-logging.md` — Async audit log design
- `003-modbus-enhancements.md` — Modbus protocol extensions
