# ModBridge - Modbus TCP Proxy Manager

**Version:** v2.0.10

[![GitHub Release](https://img.shields.io/github/release/xerolux/modbridge.svg?style=for-the-badge)](https://github.com/xerolux/modbridge/releases)
[![Downloads](https://img.shields.io/github/downloads/xerolux/modbridge/latest/total.svg?style=for-the-badge)](https://github.com/xerolux/modbridge/releases)
[![GitHub Activity](https://img.shields.io/github/commit-activity/y/xerolux/modbridge.svg?style=for-the-badge)](https://github.com/xerolux/modbridge/commits/main)
[![License](https://img.shields.io/github/license/xerolux/modbridge.svg?style=for-the-badge)](https://github.com/Xerolux/modbridge/blob/main/LICENSE)
[![CI](https://github.com/Xerolux/modbridge/actions/workflows/ci.yml/badge.svg)](https://github.com/Xerolux/modbridge/actions/workflows/ci.yml)

[![GitHub Sponsor](https://img.shields.io/github/sponsors/xerolux?logo=github&style=for-the-badge&color=blue)](https://github.com/sponsors/xerolux)
[![Ko-Fi](https://img.shields.io/badge/Ko--fi-xerolux-blue?logo=ko-fi&style=for-the-badge)](https://ko-fi.com/xerolux)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-xerolux-yellow?logo=buy-me-a-coffee&style=for-the-badge)](https://www.buymeacoffee.com/xerolux)
[![PayPal](https://img.shields.io/badge/PayPal-xerolux-blue?logo=paypal&style=for-the-badge)](https://paypal.me/xerolux)
[![Tesla Referral](https://img.shields.io/badge/Tesla-Referral-red?logo=tesla&style=for-the-badge)](https://ts.la/sebastian564489)

![ModBridge — Modbus TCP Proxy Manager](./assets/banner.svg)

**ModBridge** is a modern, robust Modbus TCP proxy manager with an elegant web interface. It multiplexes and manages Modbus connections and provides detailed monitoring, logging, and security in a compact, easy-to-deploy package.

## 📖 Full Documentation (Wiki)

All detailed information about configuration (Web UI & headless) and usage can be found in the **[GitHub Wiki](https://github.com/Xerolux/modbridge/wiki)**.

### Quick access:
- ⚙️ **[Configuration (WebUI & Headless)](https://github.com/Xerolux/modbridge/wiki/Configuration)**
- 🔧 **[Features & API](https://github.com/Xerolux/modbridge/wiki/Features-and-API)**
- 🩺 **[Troubleshooting](https://github.com/Xerolux/modbridge/wiki/Troubleshooting)**

---

## ⚙️ What ModBridge does between client and device

A proxy is not a cable. ModBridge sits between your client (Home Assistant, SCADA, your own script) and the Modbus device and solves the problems that arise there:

![How ModBridge sits between clients and Modbus devices](./docs/assets/diagrams/uebersicht.svg)

**Always active, no configuration needed:**

- **Transaction matching** — towards the device, ModBridge assigns its own transaction IDs and discards responses that do not belong to the current request. Without this, a late response becomes the answer to the *next* request, and from then on every query fails.
- **Time budget per request** — when it expires, the client gets a clean Modbus exception instead of a late response nobody is waiting for anymore.

**Configurable per proxy, off by default:**

| Option | Purpose |
|--------|---------|
| `max_target_conns` | Devices that serve only one Modbus session (SolarEdge/SunSpec and many inverters) |
| `min_request_gap_ms` | Devices that discard requests sent without a pause. Note: this cost applies **per request** |
| `request_timeout_ms` | Hard upper bound including retries |
| `cache_enabled` + `poll_interval_ms` | Keep registers warm in the background so the client never waits for a sluggish device |

**In the web interface:**

- **Device profiles** — about 60 entries in seven categories (inverters, heat pumps, ventilation, meters, storage, wallboxes, general). They fill the form with values that fit the device class and change nothing else.
- **Measure the device** — probes gap, connections, and response time on the real device and suggests values instead of guessing. Read access only, 90 seconds at most, and nothing is applied until you click. For the duration, the proxy disconnects connected clients and accepts no new ones; they reconnect on their own afterwards.

Details in the [Wiki](https://github.com/Xerolux/modbridge/wiki):
[Configuration](https://github.com/Xerolux/modbridge/wiki/Configuration) and
[Troubleshooting](https://github.com/Xerolux/modbridge/wiki/Troubleshooting).

### Cache and background polling

A sluggish device makes every client wait. The cache keeps exactly the registers that are actually queried warm, and the background poller refreshes them on its own schedule — the client gets its answer immediately, and the device is asked less often.

![Cache and background polling working together](./docs/assets/diagrams/cache-und-poller.svg)

Both are off by default, because the trade-off should be a deliberate choice:
**a cached value is not the live value.** Writes never go through the cache and invalidate the affected entries.

### Measuring the device

Instead of estimating values, ModBridge measures them on the device in front of you: the gap between requests is shortened step by step until the device starts discarding requests, and the last clean step gets a safety margin.

![How the calibration measures](./docs/assets/diagrams/kalibrierung.svg)

## 🚀 Installation with `modbridge` (recommended)

The install script handles everything: binary download, systemd service with autostart, and setup as a system-wide CLI command (`modbridge`).

### Quick Install (one-liner)

```bash
curl -sSL https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh | sudo bash -s install
```

### Step by step

```bash
# 1. Download the script
curl -sSL -o modbridge.sh https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh
chmod +x modbridge.sh

# 2. Install (interactive, with menus)
sudo bash modbridge.sh install

# 3. Afterwards 'modbridge' is available system-wide
sudo modbridge status
```

### What happens during installation?

| Step | Description |
|------|-------------|
| Detect architecture | amd64, arm64, or arm detected automatically |
| Choose variant | Full (with WebUI) or Headless (without WebUI) |
| Choose version | Latest release from GitHub, or pick an older one |
| Download binary | Matching binary to `/opt/modbridge/modbridge` |
| Install script | Script copied to `/usr/local/bin/modbridge` |
| systemd service | Service created with autostart and started |

After installation, ModBridge starts automatically on every system boot. All configured proxies are started automatically as well.

### All commands

```bash
modbridge                          # Interactive TUI menu (whiptail)
modbridge install [--auto]         # Install (or reinstall)
modbridge update [--auto]          # Update
modbridge start                    # Start the service
modbridge stop                     # Stop the service
modbridge restart                  # Restart the service
modbridge status                   # Show status
modbridge logs [-f]                # Logs (live with -f)
modbridge health                   # Health check
modbridge config                   # Edit config (nano/vi)
modbridge backup                   # Back up config + DB
modbridge version                  # Show version
modbridge uninstall                # Remove completely
```

### Options

| Option | Description |
|--------|-------------|
| `--auto` | Automatic mode: latest version, WebUI, no dialogs |
| `--headless` | Automatic mode, headless variant |
| `--force` | Force installation (overwrites existing one) |
| `NO_UPDATE=1` | Skip the script self-update |

### Self-update

The script automatically checks for a newer version on **every invocation**. If one is available, it downloads the new version and restarts itself. No manual intervention required.

```bash
# Checks for script updates automatically, then installs
sudo modbridge install

# Skip the update check
NO_UPDATE=1 sudo modbridge install
```

### Update & reinstall — your data is preserved

ModBridge protects your data during updates and reinstalls:

| Action | Config (`config.json`) | Database (`modbridge.db`) | Proxies |
|--------|------------------------|---------------------------|---------|
| `modbridge update` | **Preserved** + backup | **Preserved** | **Preserved**, service is restarted |
| `modbridge install` (already installed) | **Preserved** — offers an update | **Preserved** | **Preserved** |
| `modbridge install --force` | **Preserved** + backup | **Preserved** | **Preserved**, reinstall |
| `modbridge uninstall` | Deleted (backup optional) | Deleted (backup optional) | Deleted |

**Update process in detail:**
1. Service is stopped
2. Config is automatically backed up to `/opt/modbridge/backups/`
3. Old binary is kept as `modbridge.backup.TIMESTAMP`
4. New binary is downloaded
5. Service is restarted
6. If startup fails → automatic rollback to the previous binary

**Reinstall** (e.g. after switching variants Full ↔ Headless):
```bash
sudo modbridge install --force
# Config and DB are preserved, only the binary is replaced
```

### Manual backup management

```bash
# Create a backup
sudo modbridge backup
# → /opt/modbridge/backups/config-20260401_120000.json
# → /opt/modbridge/backups/db-20260401_120000.db

# Edit the config
sudo modbridge config

# Restart the service after config changes
sudo modbridge restart
```

### Supported architectures

| Architecture | System |
|--------------|--------|
| `amd64` | Intel/AMD 64-bit (standard servers, PCs) |
| `arm64` | ARM 64-bit (Raspberry Pi 4/5, ARM servers) |
| `arm` | ARM 32-bit (Raspberry Pi Zero/1/2/3, 32-bit OS) |

---

## 🐳 Docker Deployment

Alternative installation via Docker Compose:

```yaml
version: '3.8'

services:
  modbridge:
    image: ghcr.io/xerolux/modbridge:latest
    container_name: modbridge
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "5020-5030:5020-5030" # Port range for proxies
    volumes:
      - ./config.json:/app/config.json
      - ./data:/app/data
```

```bash
docker-compose up -d
```

---

## 💻 Web UI

After installation (Full variant), the web UI is available at:

```
http://<SERVER-IP>:8080
```

The admin password is generated automatically on first start and printed to the logs:

```bash
modbridge logs | grep -i password
```

### What it looks like

| | |
|---|---|
| ![Dashboard](./docs/assets/screenshots/dashboard.png) | ![Proxy control](./docs/assets/screenshots/proxies.png) |
| **Dashboard** — the state of all proxies at a glance | **Control** — create, start, and group proxies |
| ![Edit proxy](./docs/assets/screenshots/proxy-dialog.png) | ![Last measurement](./docs/assets/screenshots/calibration-report.png) |
| **Proxy dialog** — profile, gap, cache, protocol | **Measurement report** — every step traceable, applied on click |

More views — devices, logs, dark theme, and mobile format — in the
[Wiki](https://github.com/Xerolux/modbridge/wiki/Screenshots).

---

## 🛠️ Development & Build

Want to get your hands dirty or compile the project from source?
See the Wiki for `make` commands, frontend builds, and more.

Local build:
```bash
make build
./modbridge
```

---

## 🤝 Contributing
Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## 📄 License
MIT License — see [LICENSE](LICENSE) for details.

## ✍️ Author
- **Xerolux** - [GitHub](https://github.com/Xerolux)

---
**Version**: 2.0.10 | **Status**: Beta | **Last updated**: August 2026
