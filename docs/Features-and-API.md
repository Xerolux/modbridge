# Features & API

## Core features

* **Multiplexing:** Multiple Modbus clients can simultaneously access a single physical device that only allows one connection.
* **Headless mode:** Can be compiled and operated entirely without a web interface (lower resource consumption).
* **Connection pooling & keep-alive:** Intelligent reuse of connections to the target device, reducing latency and overhead.
* **Latency optimization:** Efficient batching and pipelining of requests.
* **Intelligent polling:** Requests for the same registers can be merged.
* **Modbus TCP validation:** Blocks invalid or malformed frames before they reach the target device.

## Security

* **Authentication:** The web UI is protected by password-based authentication (bcrypt).
* **Session management:** Secure, time-based sessions with automatic timeout.
* **Rate limiting:** Protection against brute-force and DoS attacks through IP-based rate limiting.
* **CSRF protection:** All state-changing API endpoints are protected against cross-site request forgery.
* **Secure headers:** Implementation of common security headers (HSTS, X-Content-Type-Options, etc.).
* **Password policies:** Enforces complex passwords during setup.

## API endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Health check (no login required) |
| `/api/status` | GET | Server status |
| `/api/login` | POST | Log in |
| `/api/logout` | POST | Log out |
| `/api/proxies` | GET | List all proxies |
| `/api/proxies` | POST | Create a new proxy |
| `/api/proxies` | PUT | Update a proxy (ID in body) |
| `/api/proxies?id={id}` | DELETE | Delete a proxy |
| `/api/proxies/control` | POST | Control a proxy (`{id, action: start\|stop\|restart\|pause\|resume}`) |
| `/api/proxies/control` | POST | Control all proxies (`{action: start_all\|stop_all}`) |
| `/api/proxies/stream` | GET | Live proxy updates (SSE) |
| `/api/config/system` | GET | Get system configuration |
| `/api/config/system` | PUT | Save system configuration |
| `/api/config/password` | POST | Change password |
| `/api/logs` | GET | Get log entries |
| `/api/logs/stream` | GET | Live log stream (SSE) |
| `/api/devices` | GET | List connected devices |
| `/api/system/info` | GET | System information & metrics |
| `/api/system/diagnostics/connectivity` | GET | Check connectivity of all proxy targets |
| `/api/metrics` | GET | Prometheus metrics (port `:9090`) |

## Scripts

### scripts/modbridge.sh

The main management script for installation, updates, and service management.

```bash
sudo bash scripts/modbridge.sh <command>
```

| Command | Description |
|---------|-------------|
| `install` | Install ModBridge (binary download or source build, systemd setup) |
| `update` | Update to a new version (with automatic rollback on failure) |
| `uninstall` | Uninstall completely (service + files) |
| `start` | Start the systemd service |
| `stop` | Stop the systemd service |
| `restart` | Restart the systemd service |
| `status` | Show service status |
| `logs [N]` | Show the last N log entries (default: 50) |

**Examples:**
```bash
sudo bash scripts/modbridge.sh install    # initial installation
sudo bash scripts/modbridge.sh update     # update to a new version
sudo bash scripts/modbridge.sh status     # check status
sudo bash scripts/modbridge.sh logs 100   # last 100 log lines
sudo bash scripts/modbridge.sh uninstall  # uninstall
```

**Update procedure:**
1. Stop the service
2. Back up the old binary
3. Download the new binary
4. Start the service
5. On failure: automatic rollback to the backup (the last 3 backups are kept)

---

### scripts/go-updater.sh

Keeps the local Go installation up to date. Useful when ModBridge is built from source.

```bash
sudo bash scripts/go-updater.sh <command>
```

| Command | Description |
|---------|-------------|
| `update` | Update Go to the latest stable version |
| `install` | Set up a systemd service (updates on every system start) |
| `uninstall` | Remove the systemd service |
| `start` | Run the service manually |
| `stop` | Stop the running service |
| `status` | Show status and last log entries |

```bash
sudo bash scripts/go-updater.sh update    # update Go now
sudo bash scripts/go-updater.sh install   # set up autostart
```

---

### build.sh

Quick local build script (frontend + Go binary).

```bash
./build.sh
```

It performs the following steps:
1. `npm install` + `npm run build` in the `frontend/` directory
2. Copies the build result to `pkg/web/dist/`

---

## Calibration

`POST /api/proxies/calibrate`

```json
{ "id": "<proxy-id>" }
```

Optionally you can specify the probe register (`unit_id`, `function`, `address`, `quantity`); without a specification, the last read the proxy saw is repeated.

Connected clients are disconnected at the start: anyone waiting for an answer still gets it, afterwards the proxy returns the connection. The clients reconnect on their own after the run. How many there were is noted in the report.

The response contains every measured step (gap, error rate, p50/p95), the results for parallel connections, the recommended values, and hints in plain text. It changes nothing in the configuration.

**During the measurement the proxy accepts no client connections.** A run is limited to 90 seconds.

## Prometheus metrics

In addition to requests, errors, connections, and latency:

| Metric | Meaning |
|--------|---------|
| `modbridge_proxy_stale_responses_total` | Discarded responses to requests already given up on |
| `modbridge_proxy_cache_hits_total` | Reads served from the cache |
| `modbridge_proxy_cache_misses_total` | Reads that had to go to the device |
| `modbridge_proxy_cache_entries` | Registers currently held in the cache |
| `modbridge_proxy_polled_requests` | Requests kept warm by the background poller |

---
