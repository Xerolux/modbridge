# ModBridge - Modbus TCP Proxy Manager

**Version:** 1.0.17

[![GitHub Release](https://img.shields.io/github/release/xerolux/modbridge.svg?style=for-the-badge)](https://github.com/xerolux/modbridge/releases)
[![Downloads](https://img.shields.io/github/downloads/xerolux/modbridge/latest/total.svg?style=for-the-badge)](https://github.com/xerolux/modbridge/releases)
[![GitHub Activity](https://img.shields.io/github/commit-activity/y/xerolux/modbridge.svg?style=for-the-badge)](https://github.com/xerolux/modbridge/commits/main)
[![License](https://img.shields.io/github/license/xerolux/modbridge.svg?style=for-the-badge)](https://github.com/Xerolux/modbridge/blob/main/LICENSE)
[![CI/CD Pipeline](https://github.com/Xerolux/modbridge/actions/workflows/main.yml/badge.svg)](https://github.com/Xerolux/modbridge/actions/workflows/main.yml)
[![Docker Publish](https://github.com/Xerolux/modbridge/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/Xerolux/modbridge/actions/workflows/docker-publish.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/xerolux/modbridge?style=for-the-badge)](https://hub.docker.com/r/xerolux/modbridge)

[![GitHub Sponsor](https://img.shields.io/github/sponsors/xerolux?logo=github&style=for-the-badge&color=blue)](https://github.com/sponsors/xerolux)
[![Ko-Fi](https://img.shields.io/badge/Ko--fi-xerolux-blue?logo=ko-fi&style=for-the-badge)](https://ko-fi.com/xerolux)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-xerolux-yellow?logo=buy-me-a-coffee&style=for-the-badge)](https://www.buymeacoffee.com/xerolux)
[![PayPal](https://img.shields.io/badge/PayPal-xerolux-blue?logo=paypal&style=for-the-badge)](https://paypal.me/xerolux)
[![Tesla Referral](https://img.shields.io/badge/Tesla-Referral-red?logo=tesla&style=for-the-badge)](https://ts.la/sebastian564489)

![ModBridge Logo](./assets/banner.png)

**ModBridge** ist ein moderner, robuster Modbus TCP Proxy Manager mit einer eleganten Web-Oberfläche. Er ermöglicht das Multiplexing und Management von Modbus-Verbindungen und bietet detailliertes Monitoring, Logging und Sicherheit in einem kompakten, einfach bereitzustellenden Paket.

## 📖 Ausführliche Dokumentation (Wiki)

Alle ausführlichen Informationen zu Konfiguration (Web-UI & Headless) und Nutzung finden Sie in unserem **[GitHub Wiki](https://github.com/Xerolux/modbridge/wiki)**.

### Schnellzugriff:
- ⚙️ **[Konfiguration (WebUI & Headless)](https://github.com/Xerolux/modbridge/wiki/Konfiguration)**
- 🔧 **[Features & API](https://github.com/Xerolux/modbridge/wiki/Features-und-API)**
- 🩺 **[Troubleshooting](https://github.com/Xerolux/modbridge/wiki/Troubleshooting)**

---

## 🚀 Installation mit `modbridge` (empfohlen)

Das Installationsskript übernimmt alles: Binary-Download, systemd-Service mit Autostart und Einrichtung als systemweites CLI-Kommando (`modbridge`).

### Quick Install (einzeilig)

```bash
curl -sSL https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh | sudo bash -s install
```

### Schritt für Schritt

```bash
# 1. Skript herunterladen
curl -sSL -o modbridge.sh https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh
chmod +x modbridge.sh

# 2. Installieren (interaktiv mit Menü)
sudo bash modbridge.sh install

# 3. Danach ist 'modbridge' systemweit verfügbar
sudo modbridge status
```

### Was passiert bei der Installation?

| Schritt | Beschreibung |
|---------|-------------|
| Architektur erkennen | amd64, arm64 oder arm automatisch erkannt |
| Variante wählen | Full (mit WebUI) oder Headless (ohne WebUI) |
| Version wählen | Neueste Release von GitHub, oder ältere wählen |
| Binary download | Passende Binary nach `/opt/modbridge/modbridge` |
| Script installieren | Skript nach `/usr/local/bin/modbridge` kopiert |
| systemd-Service | Service mit Autostart erstellt und gestartet |

Nach der Installation startet ModBridge automatisch bei jedem Systemstart. Alle konfigurierten Proxies werden automatisch mitgestartet.

### Alle Befehle

```bash
modbridge                          # Interaktives TUI-Menü (whiptail)
modbridge install [--auto]         # Installieren (oder Neuinstallation)
modbridge update [--auto]          # Aktualisieren
modbridge start                    # Service starten
modbridge stop                     # Service stoppen
modbridge restart                  # Service neustarten
modbridge status                   # Status anzeigen
modbridge logs [-f]                # Logs (live mit -f)
modbridge health                   # Health-Check
modbridge config                   # Config bearbeiten (nano/vi)
modbridge backup                   # Config + DB sichern
modbridge version                  # Version anzeigen
modbridge uninstall                # Vollständig entfernen
```

### Optionen

| Option | Beschreibung |
|--------|-------------|
| `--auto` | Automatischer Modus: neueste Version, WebUI, keine Dialoge |
| `--headless` | Automatischer Modus, Headless-Variante |
| `--force` | Installation erzwingen (überschreibt bestehende) |
| `NO_UPDATE=1` | Script-Auto-Update überspringen |

### Selbst-Update

Das Skript prüft bei **jedem Aufruf** automatisch auf eine neuere Version. Falls verfügbar, lädt es die neue Version herunter und startet sich selbst neu. Kein manuelles Eingreifen nötig.

```bash
# Prüft automatisch auf Script-Updates, dann installieren
sudo modbridge install

# Update-Prüfung überspringen
NO_UPDATE=1 sudo modbridge install
```

### Update & Neuinstallation — Daten bleiben erhalten

ModBridge schützt Ihre Daten bei Updates und Neuinstallationen:

| Aktion | Config (`config.json`) | Datenbank (`modbridge.db`) | Proxies |
|--------|----------------------|---------------------------|---------|
| `modbridge update` | **Erhalten** + Backup | **Erhalten** | **Erhalten**, Service wird neugestartet |
| `modbridge install` (bereits installiert) | **Erhalten** — bietet Update an | **Erhalten** | **Erhalten** |
| `modbridge install --force` | **Erhalten** + Backup | **Erhalten** | **Erhalten**, Neuinstallation |
| `modbridge uninstall` | Gelöscht (Backup optional) | Gelöscht (Backup optional) | Gelöscht |

**Update-Prozess im Detail:**
1. Service wird gestoppt
2. Config wird automatisch nach `/opt/modbridge/backups/` gesichert
3. Alte Binary wird als `modbridge.backup.ZEITSTEMPEL` behalten
4. Neue Binary wird heruntergeladen
5. Service wird neugestartet
6. Falls der Start fehlschlägt → automatisches Rollback auf die vorherige Binary

**Neuinstallation** (z.B. nach Versionswechsel Full ↔ Headless):
```bash
sudo modbridge install --force
# Config und DB bleiben erhalten, nur Binary wird ausgetauscht
```

### Manuelle Backup-Verwaltung

```bash
# Backup erstellen
sudo modbridge backup
# → /opt/modbridge/backups/config-20260401_120000.json
# → /opt/modbridge/backups/db-20260401_120000.db

# Config bearbeiten
sudo modbridge config

# Nach Config-Änderungen Service neustarten
sudo modbridge restart
```

### Unterstützte Architekturen

| Architektur | System |
|------------|--------|
| `amd64` | Intel/AMD 64-bit (Standard Server, PC) |
| `arm64` | ARM 64-bit (Raspberry Pi 4/5, ARM Server) |
| `arm` | ARM 32-bit (Raspberry Pi Zero/1/2/3, 32-bit OS) |

---

## 🐳 Docker Deployment

Alternative Installation via Docker Compose:

```yaml
version: '3.8'

services:
  modbridge:
    image: ghcr.io/xerolux/modbridge:latest
    container_name: modbridge
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "5020-5030:5020-5030" # Port-Range für Proxies
    volumes:
      - ./config.json:/app/config.json
      - ./data:/app/data
```

```bash
docker-compose up -d
```

---

## 💻 Web-UI

Nach der Installation (Full-Variante) ist die Web-UI erreichbar unter:

```
http://<IP-DES-SERVERS>:8080
```

Das Admin-Passwort wird beim ersten Start automatisch generiert und in den Logs angezeigt:

```bash
modbridge logs | grep -i password
```

---

## 🛠️ Entwicklung & Build

Möchten Sie selbst Hand anlegen oder das Projekt aus den Quellen kompilieren?
Informationen zu `make`-Befehlen, Frontend-Build und mehr finden Sie im Wiki.

Lokaler Build:
```bash
make build
./modbridge
```

---

## 🤝 Beitragen
Beiträge sind willkommen! Bitte lesen Sie [CONTRIBUTING.md](CONTRIBUTING.md) für Details.

## 📄 Lizenz
MIT License - siehe [LICENSE](LICENSE) für Details.

## ✍️ Autor
- **Xerolux** - [GitHub](https://github.com/Xerolux)

---
**Version**: 1.0.17 | **Status**: Beta | **Letzte Aktualisierung**: April 2026
