# AGENTS.md — ModBridge AI Assistant Guide

This file describes the codebase structure, development workflows, and conventions for AI assistants working on ModBridge.

---

## Project Overview

**ModBridge** is a Modbus TCP Proxy Manager with a web UI. It proxies Modbus TCP traffic, exposing a REST API and Vue.js frontend for configuration and monitoring. The application is written in Go (backend) with a Vue.js 3 frontend embedded into the binary.

**Current version:** 2.0.7.15
**Go version:** 1.26.4 (see `go.mod`)
**Node version:** 24 (CI/CD, `frontend/`)

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
├── config.json                # Default runtime configuration
├── version.txt                # Current version string
├── .env.example               # Environment variable template
├── pkg/                       # All Go packages (35+, ~26k lines)
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
│   ├── ldap/                  # LDAP/Active Directory integration
│   ├── users/                 # Multi-user management
│   ├── mapping/               # Modbus register transformations
│   ├── caching/               # TTL-based register caching
│   ├── compression/           # Data compression utilities
│   ├── openapi/               # OpenAPI/Swagger spec generation
│   ├── tls/                   # mTLS certificate handling
│   ├── cluster/               # High-availability clustering
│   ├── devices/               # Device tracking
│   ├── batch/                 # Batch Modbus operations
│   ├── pool/                  # Connection pooling
│   ├── portmanager/           # Dynamic port allocation
│   ├── web/                   # Embedded frontend assets (dist/ copied here at build time)
│   └── testing/               # Test utilities: mockmodbus/, integration/, performance/
├── cmd/cli/                   # CLI tooling
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
    ├── workflows/             # CI/CD pipelines (6 workflows)
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
make build-all        # Cross-compile: linux-amd64/arm64/arm, windows-amd64, darwin-amd64/arm64
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

- **Go 1.26.1+** with `CGO_ENABLED=1` (required for `go-sqlite3`)
- **GCC** (for SQLite CGO compilation; cross-compilers for arm: `gcc-aarch64-linux-gnu`, `gcc-arm-linux-gnueabihf`)
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

- **28 test files** spread across packages (co-located with source: `pkg/foo/foo_test.go`)
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

Primary configuration file. Key fields:

```json
{
  "web_port": ":8080",
  "admin_password_hash": "<bcrypt>",
  "tls": { "enabled": false, "cert_file": "", "key_file": "" },
  "proxies": [
    {
      "id": "uuid",
      "name": "Proxy Name",
      "listen_addr": ":5020",
      "target_addr": "192.168.1.100:502",
      "timeout": 30,
      "retries": 3
    }
  ],
  "logging": { "level": "INFO", "rotation": true, "retention_days": 7 },
  "cors": { "allowed_origins": ["*"] },
  "rate_limit": { "enabled": true, "requests_per_minute": 100 },
  "session_timeout_hours": 24,
  "max_connections": 1000
}
```

### Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `WEB_PORT` | `:8080` | HTTP bind address |
| `LOG_LEVEL` | `INFO` | `DEBUG`/`INFO`/`WARN`/`ERROR` |
| `TZ` | `UTC` | Timezone |
| `VERSION` | `dev` | Version string (injected at build) |
| `DEBUG` | — | Enable pprof endpoints |
| `MODBRIDGE_MAX_CONNECTIONS` | `10000` | Global connection limit |
| `MODBRIDGE_CIRCUIT_BREAKER_ENABLED` | `true` | Circuit breaker toggle |
| `MODBRIDGE_CACHE_ENABLED` | `true` | Register cache toggle |
| `MODBRIDGE_CACHE_TTL` | `5s` | Cache TTL |
| `MODBRIDGE_HEALTH_CHECK_INTERVAL` | `30s` | Health check frequency |
| `MODBRIDGE_ALERTING_ENABLED` | `true` | Webhook alerting toggle |

**Configuration priority:** Environment variables > `config.json` > compiled defaults

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

Located in `.github/workflows/`:

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `main.yml` | push/PR to main, tags | Primary: format check, vet, test, build binaries, Docker push, releases |
| `release.yml` | tag `v*` | GitHub release with cross-platform binaries and checksums |
| `docker.yml` | push to main | Docker Hub push (legacy) |
| `headless.yml` | push to main | Build variant without WebUI |
| `pages.yml` | push to main | GitHub Pages docs site |
| `wiki-sync.yml` | push to main | Sync GitHub Wiki |

**Release process:** Push a tag matching `v*` → CI builds all platforms → creates GitHub release with checksums.

---

## Database

- **SQLite3** via `github.com/mattn/go-sqlite3` (requires CGO)
- Database file: `modbridge.db` (auto-created on first run)
- Fallback mode: If DB initialization fails, the app runs without persistence (`pkg/database/fallback.go`)
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
5. **SQLite fallback:** If the database fails to initialize, the app continues in fallback (in-memory/no-persistence) mode — check startup logs.
6. **Frontend output sanitization:** Vite config strips underscore-prefixed filenames from `dist/` to avoid Go embed issues.

---

## Dependencies

### Go (minimal)
| Module | Version | Purpose |
|--------|---------|---------|
| `github.com/google/uuid` | v1.6.0 | UUID generation |
| `golang.org/x/crypto` | v0.48.0 | bcrypt password hashing |
| `github.com/mattn/go-sqlite3` | v1.14.37 | SQLite3 (CGO) |

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
