# Configuration

ModBridge stores its configuration in the `config.json` file by default.

Configuration can be done via the **web interface** (recommended) or by **directly editing `config.json`** in headless mode.

## Headless operation (without WebUI)

ModBridge can be operated entirely without a graphical interface. This is ideal for resource-efficient servers, edge devices (such as Raspberry Pi), or automation via configuration management tools (Ansible, Puppet, etc.).

### Create the configuration file

If no `config.json` exists yet, ModBridge automatically creates a default configuration on first start.

```bash
# Start ModBridge briefly once to generate config.json
./modbridge-linux-amd64-headless &
sleep 2
kill $!
```

### Edit the configuration

Edit the file with the text editor of your choice:

```bash
# e.g. vi config.json
```

### Example configuration

A typical `config.json` for headless operation with two configured proxies:

```json
{
  "web_port": ":8080",
  "admin_pass_hash": "$2a$10$xyz...",
  "force_password_change": false,
  "session_timeout": 24,

  "proxies": [
    {
      "id": "proxy-1-solar",
      "name": "Solar Inverter",
      "listen_addr": ":5020",
      "target_addr": "192.168.1.100:502",
      "enabled": true,
      "paused": false,
      "connection_timeout": 5,
      "read_timeout": 5,
      "max_retries": 3,
      "description": "Rooftop solar system",
      "max_read_size": 0,
      "tags": ["solar", "roof"]
    },
    {
      "id": "proxy-2-hvac",
      "name": "Air Conditioner",
      "listen_addr": ":5021",
      "target_addr": "192.168.1.101:502",
      "enabled": true,
      "paused": false,
      "connection_timeout": 10,
      "read_timeout": 10,
      "max_retries": 1,
      "description": "Climate control building A",
      "max_read_size": 256,
      "tags": ["hvac", "building-a"]
    }
  ],

  "log_level": "INFO",
  "log_max_size": 100,
  "log_max_files": 10,
  "log_max_age_days": 30,

  "metrics_enabled": true,
  "metrics_port": ":9090",

  "debug_mode": false,
  "max_connections": 1000
}
```

### Headless configuration options

Important settings for server operation:

| Field | Description | Recommendation for headless |
|-------|-------------|-----------------------------|
| `log_level` | Log verbosity (`DEBUG`, `INFO`, `WARN`, `ERROR`) | `INFO` for production, `WARN` for minimal logs |
| `metrics_enabled` | Enables the Prometheus metrics endpoint | `true` (very useful for server monitoring) |
| `metrics_port` | Port for the `/metrics` endpoint | `:9090` (default) |
| `max_connections` | Maximum number of concurrent TCP connections | Adjust to server capacity (e.g. `5000`) |

### Proxy configuration (headless)

To add a new proxy in headless mode, extend the `proxies` array in `config.json`.

**Important:**
* Every proxy needs a unique `id` (can be any string, UUID recommended).
* `listen_addr` must be unique and have the format `:PORT` or `IP:PORT`.
* Set `"enabled": true` so the proxy is loaded automatically at startup.

### Restart the service after configuration changes

For changes to `config.json` to take effect, the ModBridge process must be restarted:

**With systemd:**
```bash
sudo systemctl restart modbridge
```

**Docker:**
```bash
docker restart modbridge
```

### Check service status and logs

To make sure your headless configuration was loaded correctly:

```bash
# Check systemd status
sudo systemctl status modbridge

# Check logs for errors
sudo journalctl -u modbridge -f
```

## Proxy fields (reference)

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | UUID (assigned automatically) |
| `name` | string | Display name in the web interface |
| `listen_addr` | string | Local port, e.g. `:5020` |
| `target_addr` | string | Target address, e.g. `192.168.1.100:502` |
| `enabled` | bool | Proxy enabled/disabled |
| `paused` | bool | Proxy paused (connections are rejected) |
| `connection_timeout` | int | Connection timeout in seconds |
| `read_timeout` | int | Read timeout in seconds |
| `max_retries` | int | Maximum retry attempts on error |
| `max_read_size` | int | Max. Modbus read size (0 = unlimited) |
| `connect_delay_ms` | int | Pause after the TCP connect before the first request (ms, 0 = off). For slow devices like Huawei inverters/sDongles |
| `max_target_conns` | int | Max. concurrent connections to the target device (0 = default 10). Use `1` for devices with a single Modbus session, e.g. SolarEdge/SunSpec |
| `min_request_gap_ms` | int | Minimum gap between two requests to the target device (ms, 0 = off) |
| `request_timeout_ms` | int | Hard time budget for a client request including retries (ms, 0 = derived automatically from `read_timeout` and `max_retries`) |
| `calibrated_at` | string | Time of the last measurement (RFC3339). Informative — shows how old the configured values are |
| `device_profile` | string | Last applied device profile. Informative only — remembers which preset the values came from; behavior follows the individual fields |
| `cache_enabled` | bool | Serve repeated reads from a cache (default: off) |
| `cache_ttl_ms` | int | Validity of a cache entry (ms, 0 = 5000) |
| `poll_interval_ms` | int | Refresh polled registers in the background (ms, 0 = off). Requires `cache_enabled` |
| `protocol` | string | `tcp` (default) or `rtu-tcp` for serial adapters that expect raw RTU frames |
| `description` | string | Optional description |
| `tags` | array | Optional tags for categorization |


### Cache and background polling

Some devices cannot be sped up: a SolarEdge leader fetches follower registers across the RS485 chain first, a heating controller simply takes seconds. When the client (e.g. Home Assistant with a 3 s timeout) gives up faster than the device answers, no amount of timeout tuning helps.

For this there are two related options:

- **`cache_enabled`** — repeated reads are served from the cache instead of asking the device again.
- **`poll_interval_ms`** — a background poller refreshes exactly the registers that clients actually query, on its own schedule. The client gets an immediate answer and never waits for the device.

```json
"cache_enabled": true,
"cache_ttl_ms": 20000,
"poll_interval_ms": 5000
```

The poller **bundles** neighboring registers: if a client queries twenty small ranges that lie close together, ModBridge reads them as one block and then distributes the response across the individual entries. This reduces exactly the factor that dominates on slow devices — the number of round trips. Client requests themselves are never merged; a proxy that rewrites what a client asked for is no longer traceable.

**TTL and interval belong together.** `cache_ttl_ms` is an upper bound on the age of a value, not a refresh schedule — the poller keeps entries considerably fresher. If the validity is not **multiple times** the interval, entries expire between two rounds, the client falls through to the device again in that gap, and waits after all. ModBridge logs a warning at startup when the two values are in that ratio.

**What you are accepting:** a cached value is by definition not the live value — it is up to `cache_ttl_ms` old. For dashboards and energy data this is harmless; for control loops it is not. That is why the feature is **off** by default and must be enabled deliberately.

What the cache does **not** do:

- Writes are never cached and always passed through.
- After a write, all cache entries of that unit ID are discarded — a just-changed value would not merely be old, it would be wrong.
- Modbus exceptions never land in the cache.
- The poller only queries registers that a client has requested before, and forgets them again when nobody asks for them for a while.

The proxy status exposes `cache_hits`, `cache_misses`, `cache_entries`, and `polled_requests` for verification.

### Measuring the device (calibration)

Profiles are educated guesses. Calibration replaces them with measurements on the real device: it lowers the gap step by step until the device starts discarding requests, increases the number of parallel sessions until it stops answering, and derives the read timeout from the measured latency.

Found in the proxy dialog under **Measure device**, or via
`POST /api/proxies/calibrate` with `{"id": "<proxy-id>"}`.

**What happens during the measurement:** the proxy accepts no client connections. During that time nothing is queried and nothing is controlled through this proxy. That is why a run is hard-capped at **90 seconds**; when the time expires, it returns what was measured so far instead of continuing.

What the run protects against:

- **Read accesses only.** Never a write.
- It repeats a register that a client is querying anyway — the proxy remembers the last read. Alternatively you specify a register.
- It disconnects connected clients for the duration and holds back new ones: their traffic would falsify the measurement and vice versa. Nobody can pin a controller on demand for 90 seconds, so the proxy does it — a request that is already in flight still gets its answer, and the clients reconnect on their own afterwards.
- It releases the pool connection beforehand and pauses the health check — a device with a single Modbus session cannot be measured otherwise, because the proxy itself holds that session.
- It changes nothing. Applying the values is a separate click, and they are only saved when the proxy is saved.

A single error is not a measurement: each step is judged across a series, and the adopted gap keeps 1.5x distance from the fastest value that still ran cleanly. If the device does not answer reliably even at the most cautious gap, the run deliberately returns conservative values plus a clear note — instead of an error nobody can use.

`calibrated_at` records when the last measurement happened. If `stale_responses` or the error count rises noticeably later, a new measurement is worthwhile — devices behave differently after a firmware update.

### Device profiles

In the proxy dialog of the web interface, a device profile fills the fields at the top with values that match the behavior of the target device. The selection is grouped by category and searchable.

**Important for context:** the profiles describe a *behavior class*, not a manufacturer specification. The values live in nine classes; the roughly 60 device entries only map which class a device falls into — how many Modbus sessions it serves, how much room it needs between requests, how long an answer may take, and whether it speaks Modbus TCP or raw RTU. Measured timing values per model they are explicitly not. A profile is a starting point that keeps a device stable — fine-tuning remains your job.

#### Behavior classes

| Class | Behavior | Core setting |
|-------|----------|--------------|
| `standard` | ModBridge defaults | no limits |
| `multiSession` | fast, many clients | no limits, no gap |
| `fewSessions` | some parallel sessions | 2 connections, 50 ms gap |
| `singleSession` | one Modbus session only | 1 connection, 100 ms gap |
| `singleSessionFast` | one session, client with a short timeout | + budget 2.5 s, read timeout 2 s |
| `singleSessionSlow` | one session, sluggish controller | 250 ms gap, 10 s read timeout |
| `connectDelay` | discards requests right after connect | 3 s pause, 1 connection |
| `serialGateway` | serial line behind TCP | 1 connection, 50 ms gap, reads capped at 125 registers |
| `rtuOverTcp` | raw RTU frames without MBAP header | protocol `rtu-tcp`, 1 connection |

#### Categories and devices

| Category | Devices |
|----------|---------|
| General | Standard, PLC, Modbus-TCP→RTU gateway, serial adapter |
| Inverters / PV | SolarEdge (single), SolarEdge Leader+Follower, SMA, Fronius, Kostal, Huawei, Sungrow, GoodWe, Growatt, SolaX, Deye/Sunsynk, Sofar, Delta, KACO, FIMER/ABB, E3/DC, Victron, generic SunSpec |
| Heat pumps / heating | IDM (Navigator 2.0 / Navigator 10), Stiebel Eltron ISG, Tecalor ISG, NIBE S-series, NIBE MODBUS 40, Lambda, Waterkotte, Ochsner, Nilan, Daikin Altherma, Panasonic Aquarea, LG Therma V, Mitsubishi Ecodan |
| Ventilation / climate | Helios KWL, Zehnder ComfoAir Q, Vallox, Pluggit, Wolf CWL |
| Energy meters | Eastron SDM, Carlo Gavazzi EM24/EM340, Janitza UMG, Schneider iEM3000, Siemens SENTRON PAC, ABB B23/B24, Finder 7M, Iskra WM3, Shelly Pro EM |
| Battery storage | BYD Battery-Box, Pylontech, VARTA |
| Wallboxes / charging | KEBA, Alfen Eve, go-e, Wallbox Pulsar Plus, MENNEKES AMTRON, Webasto, ABL eMH, Heidelberg Energy Control |

Devices that only reach Modbus through a separate adapter (e.g. NIBE MODBUS 40, LG PI485, many meters) are marked accordingly in the hint text — the values apply to the adapter there, not the device.

A new device is an entry in `frontend/src/deviceProfiles.js` with a label and a class; categories, class hints, and notes live in `frontend/src/i18n.js` under `control.profiles.*`.

### Complete config.json (example with all options)

```json
{
  "web_port": ":8080",
  "admin_pass_hash": "",
  "force_password_change": true,
  "session_timeout": 24,

  "proxies": [
    {
      "id": "21e71152-3866-43ac-891d-c5ec85fa1e98",
      "name": "SolarEdge Proxy",
      "listen_addr": ":5020",
      "target_addr": "192.168.1.100:502",
      "enabled": true,
      "paused": false,
      "connection_timeout": 10,
      "read_timeout": 10,
      "max_retries": 3,
      "description": "Connects to the SolarEdge plant",
      "max_read_size": 0,
      "connect_delay_ms": 0,
      "max_target_conns": 1,
      "min_request_gap_ms": 100,
      "request_timeout_ms": 2500,
      "tags": []
    }
  ],

  "log_level": "INFO",
  "log_max_size": 100,
  "log_max_files": 10,
  "log_max_age_days": 30,

  "tls_enabled": false,
  "tls_cert_file": "",
  "tls_key_file": "",

  "cors_allowed_origins": ["*"],
  "cors_allowed_methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
  "cors_allowed_headers": ["Content-Type", "Authorization"],

  "rate_limit_enabled": true,
  "rate_limit_requests": 60,
  "rate_limit_burst": 100,

  "ip_whitelist_enabled": false,
  "ip_whitelist": [],
  "ip_blacklist_enabled": false,
  "ip_blacklist": [],

  "metrics_enabled": true,
  "metrics_port": ":9090",

  "email_enabled": false,
  "email_smtp_server": "",
  "email_smtp_port": 587,
  "email_from": "",
  "email_to": "",
  "email_username": "",
  "email_password": "",
  "email_alert_on_error": true,
  "email_alert_on_warning": false,

  "backup_enabled": true,
  "backup_interval": "daily",
  "backup_retention": 7,
  "backup_path": "./backups",
  "backup_database": true,
  "backup_config": true,

  "debug_mode": false,
  "max_connections": 1000
}
```

## Writes and flash wear

On an SD card or a cheap SSD, the question of what ModBridge actually writes to disk is fair.

**Never touches the disk:** the response cache, the background poller, statistics, latency percentiles, and calibration. All of that lives entirely in RAM and disappears on restart. Enabling the cache generates **no** additional writes.

**Touches the disk:**

| What | When |
|------|------|
| `connection_history` | one row per client connection |
| `devices` | updated per connection |
| `audit_log` | per user action (rare) |
| log files | per log line, with rotation |
| `config.json` | only on configuration changes |

The relevant item is the connection history. A client with one permanent connection generates almost nothing; a client in a reconnect loop generates one row per attempt.

Countermeasures:

- **WAL mode and `synchronous=NORMAL`** are set. This removes the fsync on every commit — the difference on flash storage is substantial. The price: on a power failure the last transactions may be missing. The database is not corrupted by this.
- The connection history is cleaned up automatically after 7 days.
- `log_level` at `WARN` reduces the log volume considerably.

If you want to be absolutely safe, put the database and logs on different media (USB SSD, tmpfs for logs) — that is a question of installation, not configuration.
