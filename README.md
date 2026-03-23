# ModBridge - Modbus TCP Proxy Manager

**Version:** 1.0.12

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

Alle ausführlichen Informationen zu Installation, Konfiguration (Web-UI & Headless) und Nutzung finden Sie in unserem **[GitHub Wiki](https://github.com/Xerolux/modbridge/wiki)**.

### Schnellzugriff:
- 🚀 **[Installation & Deployment](https://github.com/Xerolux/modbridge/wiki/Installation)**
- ⚙️ **[Konfiguration (WebUI & Headless)](https://github.com/Xerolux/modbridge/wiki/Konfiguration)**
- 🔧 **[Features & API](https://github.com/Xerolux/modbridge/wiki/Features-und-API)**
- 🩺 **[Troubleshooting](https://github.com/Xerolux/modbridge/wiki/Troubleshooting)**

---

## 🔥 Quick Install (Linux)

Für eine schnelle Installation auf einem Linux-System (Ubuntu/Debian) können Sie das offizielle Installationsskript verwenden:

```bash
curl -sSL https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh | sudo bash -s install
```

---

## 🐳 Docker Deployment

Das einfachste Setup via Docker Compose:

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

## 💻 Web-UI Vorschau

Eine Vorschau der eleganten und reaktionsschnellen Benutzeroberfläche finden Sie in unserem Repository unter `assets/screenshots/`.

*(Tipp: Screenshots und aktuelle Bilder der WebUI finden Sie im [Wiki](https://github.com/Xerolux/modbridge/wiki) oder direkt in den Releases.)*

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
**Version**: 1.0.12 | **Status**: Beta | **Letzte Aktualisierung**: März 2026
