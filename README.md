# ModBridge - Modbus TCP Proxy Manager

**Version:** 0.1.0

[![GitHub Release](https://img.shields.io/github/release/xerolux/modbridge.svg?style=for-the-badge)](https://github.com/xerolux/modbridge/releases)
[![Downloads](https://img.shields.io/github/downloads/xerolux/modbridge/latest/total.svg?style=for-the-badge)](https://github.com/xerolux/modbridge/releases)
[![GitHub Activity](https://img.shields.io/github/commit-activity/y/xerolux/modbridge.svg?style=for-the-badge)](https://github.com/xerolux/modbridge/commits/main)
[![License](https://img.shields.io/github/license/xerolux/modbridge.svg?style=for-the-badge)](https://github.com/Xerolux/modbridge/blob/main/LICENSE)
[![CI/CD Pipeline](https://github.com/Xerolux/modbridge/actions/workflows/main.yml/badge.svg)](https://github.com/Xerolux/modbridge/actions/workflows/main.yml)
[![Docker Publish](https://github.com/Xerolux/modbridge/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/Xerolux/modbridge/actions/workflows/docker-publish.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/xerolux/modbridge?style=for-the-badge)](https://hub.docker.com/r/xerolux/modbridge)

![ModBridge Logo](./assets/banner.png)

**ModBridge** - Ein moderner, robuster Modbus TCP Proxy Manager mit einer eleganten Web-Oberfläche. Verwalten Sie mehrere Modbus TCP Proxy-Instanzen über eine zentrale, intuitive Webschnittstelle.

## Highlights

- **Einfach zu bedienen** - Intuitive Web-UI
- **Hocheffizient** - Latenz ~3-5ms, ~10.000 req/s
- **Sicher** - Bcrypt-Authentifizierung, Session-Management, CSRF-Schutz, Rate Limiting
- **Echtzeit-Monitoring** - Live-Traffic-Logging via Server-Sent Events (SSE)
- **Docker-Ready** - One-Command Deployment
- **Device Tracking** - SQLite-basierte Verbindungshistorie
- **Metriken** - Integrierter Prometheus-kompatibler Metrics-Endpunkt

---

## Web-UI Vorschau

### Dashboard

![Dashboard](./assets/screenshots/dashboard.png)

### Proxy-Verwaltung

![Proxies](./assets/screenshots/proxies.png)

### Live-Logs

![Logs](./assets/screenshots/logs.png)

### Geräte-Tracking

![Devices](./assets/screenshots/devices.png)

### Konfiguration

![Config](./assets/screenshots/config.png)

---

## Inhaltsverzeichnis

- [Features](#features)
- [Quick Install](#quick-install)
- [Installation](#installation)
  - [Methode 1: modbridge.sh (empfohlen)](#methode-1-modbridgesh-empfohlen)
  - [Methode 2: Docker / Docker Compose](#methode-2-docker--docker-compose)
  - [Methode 3: Binary manuell herunterladen](#methode-3-binary-manuell-herunterladen)
  - [Methode 4: Aus Quellcode kompilieren](#methode-4-aus-quellcode-kompilieren)
- [Erste Schritte](#erste-schritte)
- [Konfiguration](#konfiguration)
- [Skripte](#skripte)
- [Makefile-Befehle](#makefile-befehle)
- [Umgebungsvariablen](#umgebungsvariablen)
- [API-Endpunkte](#api-endpunkte)
- [Troubleshooting](#troubleshooting)
- [Systemanforderungen](#systemanforderungen)
- [Roadmap](#roadmap)

---

## Features

### Kern-Funktionen

- **Multi-Proxy-Unterstützung** - Verwaltung mehrerer Modbus TCP Proxy-Instanzen gleichzeitig
- **Web-Interface** - Moderne, responsive UI (eingebettet im Go-Binary, keine separaten Dateien nötig)
- **Echtzeit-Überwachung** - Live-Traffic-Logging via Server-Sent Events (SSE)
- **Device Tracking** - SQLite-Datenbank zur Verfolgung verbundener Geräte inkl. Verbindungshistorie
- **Single Binary** - Das gesamte Frontend ist im Go-Binary eingebettet
- **Graceful Shutdown** - Sauberes Herunterfahren aller Proxy-Instanzen
- **Health Check** - `GET /api/health` für Container-Umgebungen
- **MaxReadSize** - Automatisches Aufteilen großer Modbus-Read-Requests

### Sicherheit

- Admin-Authentifizierung mit Bcrypt-Hashing
- Zufällig generiertes Admin-Passwort beim ersten Start (Passwortänderung erzwungen)
- Sichere Session-Cookies mit konfigurierbarem Timeout
- CSRF-Schutz
- Rate Limiting (konfigurierbar)
- IP-Whitelist / IP-Blacklist
- Optional: TLS/HTTPS

### Monitoring & Alerting

- Prometheus-kompatibler Metrics-Endpunkt (Standard: `:9090`)
- E-Mail-Benachrichtigungen bei Fehlern oder Warnungen (SMTP)
- Automatisches Backup von Konfiguration und Datenbank

### Performance

| Metrik | Wert |
|--------|------|
| Latenz (avg) | ~3-5 ms |
| Latenz (p99) | ~12 ms |
| Durchsatz | ~10.000 req/s |
| Speicher (idle) | ~2,5 MB |
| Speicher (load) | ~8-15 MB |
| Gleichzeitige Verbindungen | bis 1.000 (konfigurierbar) |
| Thread-Safe | Race-Condition-freie Implementierung |
| Graceful Shutdown | Wartet auf alle aktiven Goroutines |

---

## Quick Install

```bash
# Linux – Installation per Skript (empfohlen)
sudo bash scripts/modbridge.sh install

# Docker Compose
docker-compose up -d

# Vorgefertigtes Docker Image
docker run -d -p 8080:8080 -p 5020-5030:5020-5030 ghcr.io/xerolux/modbridge:latest
```

Nach der Installation: **http://localhost:8080** im Browser öffnen.

---

## Installation

### Methode 1: modbridge.sh (empfohlen)

Das Skript `scripts/modbridge.sh` ist die einfachste Installationsmethode. Es:

- 📍 **Erkennt automatisch Ihre System-Architektur** (amd64, arm64, arm 32-bit)
- 🎨 **Zeigt ein interaktives Menü** zur Auswahl der Variante
- 📥 **Lädt die passende Binary** von GitHub Releases herunter
- ⚙️ **Richtet einen systemd-Service** automatisch ein
- 🔄 **Automatische Selbst-Aktualisierung** vor jedem Befehl

```bash
# Repository klonen
git clone https://github.com/Xerolux/modbridge.git
cd modbridge

# Installieren (root erforderlich)
sudo bash scripts/modbridge.sh install
```

**Installations-Ablauf:**
1. Prüft Abhängigkeiten (`curl`, `jq`, `file`, `lsof`, `whiptail`)
2. Zeigt ein interaktives Menü mit Fragen:
   - **WebUI oder Headless?** - Mit oder ohne grafische Oberfläche
   - **Welche Version?** - Auswahl aus den verfügbaren Releases
3. Lädt das passende vorkompilierte Binary herunter
4. **Bei Headless:** Erstellt automatisch eine Standard-Konfiguration
5. Richtet einen systemd-Service ein und aktiviert Autostart

**Varianten:**

| Variante | Größe | Beschreibung | Verwendung |
|----------|-------|--------------|-------------|
| **Mit WebUI** | ~8.8 MB | Volle grafische Oberfläche | Standard-Installation, bequeme Konfiguration |
| **Headless** | ~6.9 MB | Nur API, keine WebUI | Server, Embedded-Systeme, Config-Datei-Betrieb |

**Nach der Installation:**

**Mit WebUI:**
```bash
# WebUI öffnen
http://<Ihre-IP>:8080

# Das Standard-Passwort wird beim ersten Start automatisch generiert
# und in den Logs angezeigt:
sudo journalctl -u modbridge.service -n 50
```

**Headless (ohne WebUI):**
```bash
# Konfiguration bearbeiten
sudo nano /opt/modbridge/config.json

# Service neu starten
sudo systemctl restart modbridge.service

# Status prüfen
sudo systemctl status modbridge.service
```

**Service-Management:**
```bash
sudo bash scripts/modbridge.sh status   # Service-Status prüfen
sudo bash scripts/modbridge.sh start    # Service starten
sudo bash scripts/modbridge.sh stop     # Service stoppen
sudo bash scripts/modbridge.sh restart  # Service neu starten
sudo bash scripts/modbridge.sh update   # Auf neue Version aktualisieren
```

---

### Methode 2: Docker / Docker Compose

#### Vorgefertigtes Image

```bash
docker run -d \
  --name modbridge \
  -p 8080:8080 \
  -p 5020-5030:5020-5030 \
  -v $(pwd)/config.json:/app/config.json \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  ghcr.io/xerolux/modbridge:latest
```

#### Docker Compose

```bash
# Repository klonen
git clone https://github.com/Xerolux/modbridge.git
cd modbridge

# Starten
docker-compose up -d

# Logs anzeigen
docker-compose logs -f

# Stoppen
docker-compose down
```

Die `docker-compose.yml` enthält bereits sinnvolle Standardwerte (Health Check, Resource Limits, Security Options).

---

### Methode 3: Binary manuell herunterladen

Fertige Binaries für Linux (AMD64, ARM64) und Windows (AMD64) finden Sie unter:

**https://github.com/Xerolux/modbridge/releases**

```bash
# Beispiel: Linux AMD64
wget https://github.com/Xerolux/modbridge/releases/latest/download/modbridge-linux-amd64
chmod +x modbridge-linux-amd64
./modbridge-linux-amd64
```

---

### Methode 4: Aus Quellcode kompilieren

**Voraussetzungen:**
- Go 1.21+
- Node.js 22+

```bash
# Repository klonen
git clone https://github.com/Xerolux/modbridge.git
cd modbridge

# Frontend und Binary in einem Schritt bauen
./build.sh

# Binary starten
./modbridge
```

**Oder manuell Schritt für Schritt:**
```bash
# Frontend bauen
cd frontend && npm install && npm run build && cd ..

# Frontend ins Go-Projekt kopieren
rm -rf pkg/web/dist
cp -r frontend/dist pkg/web/dist

# Go-Binary bauen
go build -ldflags="-s -w" -o modbridge ./main.go

# Starten
./modbridge
```

**Cross-Compile für andere Plattformen:**
```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o modbridge-linux-amd64 ./main.go

# Linux ARM64 (Raspberry Pi 4+)
GOOS=linux GOARCH=arm64 go build -o modbridge-linux-arm64 ./main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o modbridge.exe ./main.go

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o modbridge-darwin-arm64 ./main.go
```

---

## Erste Schritte

### 1. Anwendung starten

```bash
# Binary direkt
./modbridge

# Per systemd (nach Installation mit modbridge.sh)
sudo bash scripts/modbridge.sh start

# Per Docker Compose
docker-compose up -d
```

### 2. Erstes Login

Beim ersten Start wird ein **zufälliges Admin-Passwort** generiert und im Terminal ausgegeben:

```
Default admin password set: XyZ1234abcd56789 (password change required on first login)
```

Notieren Sie dieses Passwort. Beim ersten Login werden Sie aufgefordert, es zu ändern.

### 3. Web-Interface öffnen

```
http://localhost:8080
```

### 4. Proxy anlegen

1. Navigieren Sie zu **Proxies**
2. Klicken Sie auf **"+ Add Proxy"**
3. Füllen Sie die Felder aus:
   - **Name**: z.B. `SolarEdge Proxy`
   - **Listen Address**: z.B. `:5020`
   - **Target Address**: z.B. `192.168.1.100:502`
   - **Connection Timeout**: z.B. `10` (Sekunden)
   - **Read Timeout**: z.B. `10` (Sekunden)
   - **Max Retries**: z.B. `3`
4. Klicken Sie auf **"Save"**

### 5. Modbus-Client verbinden

```python
# Beispiel mit pymodbus
from pymodbus.client import ModbusTcpClient

# Verbindung zum Proxy (nicht direkt zum Gerät!)
client = ModbusTcpClient('localhost', port=5020)
result = client.read_holding_registers(0, 10, slave=1)
print(result.registers)
client.close()
```

---

## Konfiguration

Die Konfiguration wird in `config.json` gespeichert und kann entweder über das Web-Interface oder manuell als JSON-Datei verwaltet werden.

### Headless-Betrieb (ohne WebUI)

Wenn Sie ModBridge ohne WebUI installieren (Headless-Variante), erfolgt die Konfiguration ausschließlich über eine JSON-Datei.

#### Konfigurationsdatei erstellen

Bei der Installation mit dem Script wird automatisch eine Standard-Konfiguration erstellt. Alternativ können Sie eine manuell erstellen:

```bash
/opt/modbridge/modbridge -config > /opt/modbridge/config.json
```

#### Konfiguration bearbeiten

Bearbeiten Sie die Konfigurationsdatei mit Ihrem Lieblingseditor:

```bash
sudo nano /opt/modbridge/config.json
```

#### Konfigurations-Beispiel

```json
{
  "web_port": ":8080",
  "admin_pass_hash": "",
  "force_password_change": true,
  "session_timeout": 24,
  "proxies": [
    {
      "id": "1",
      "name": "PLC1",
      "listen_addr": ":5020",
      "target_addr": "192.168.1.100:502",
      "enabled": true,
      "paused": false,
      "connection_timeout": 10,
      "read_timeout": 10,
      "max_retries": 3,
      "description": "Verbindet sich mit PLC1",
      "max_read_size": 0,
      "tags": ["production", "critical"]
    }
  ],
  "log_level": "INFO",
  "log_max_size": 100,
  "log_max_files": 10,
  "log_max_age_days": 30
}
```

#### Headless-Konfigurationsoptionen

| Option | Typ | Beschreibung |
|--------|-----|--------------|
| `web_port` | string | Port für API (Standard: `:8080`) |
| `admin_pass_hash` | string | Hash des Admin-Passworts (bei Headless optional, leer für Auto-Generierung) |
| `force_password_change` | bool | Erzwingt Passwortänderung beim ersten Login |
| `session_timeout` | int | Session-Timeout in Stunden (Standard: 24) |
| `log_level` | string | Log-Level: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `log_max_size` | int | Max. Log-Dateigröße in MB |
| `proxies` | array | Liste der Modbus-Proxies (siehe unten) |

#### Proxy-Konfiguration (Headless)

Jeder Proxy in der `proxies`-Liste hat folgende Optionen:

| Option | Typ | Beschreibung |
|--------|-----|--------------|
| `id` | string | Eindeutige ID des Proxies (beliebiger String) |
| `name` | string | Anzeigename |
| `listen_addr` | string | Lokaler Port (z.B. `:5020`) |
| `target_addr` | string | Remote-Adresse (z.B. `192.168.1.100:502`) |
| `enabled` | bool | Proxy aktivieren/deaktivieren |
| `paused` | bool | Proxy pausieren (Verbindungen werden abgelehnt) |
| `connection_timeout` | int | Verbindungs-Timeout in Sekunden |
| `read_timeout` | int | Lese-Timeout in Sekunden |
| `max_retries` | int | Maximale Wiederholungsversuche bei Fehler |
| `max_read_size` | int | Max. Modbus-Read-Größe (0 = unbegrenzt) |
| `description` | string | Optionale Beschreibung |
| `tags` | array | Optionale Tags zur Kategorisierung |

#### Service nach Konfigurationsänderung neu starten

```bash
sudo systemctl restart modbridge.service
```

#### Service-Status und Logs prüfen

```bash
# Status prüfen
sudo systemctl status modbridge.service

# Logs anzeigen
sudo journalctl -u modbridge.service -f

# Letzte 100 Logs
sudo journalctl -u modbridge.service -n 100
```

---

### Web-UI-Betrieb (mit WebUI)

Die Konfiguration wird in `config.json` gespeichert und kann auch über das Web-Interface verwaltet werden (Export / Import).

### Vollständige config.json

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
      "description": "Verbindet sich mit SolarEdge Anlage",
      "max_read_size": 0,
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

### Proxy-Felder

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `id` | string | UUID (wird automatisch vergeben) |
| `name` | string | Anzeigename im Web-Interface |
| `listen_addr` | string | Lokaler Port, z.B. `:5020` |
| `target_addr` | string | Zieladresse, z.B. `192.168.1.100:502` |
| `enabled` | bool | Proxy aktiviert/deaktiviert |
| `paused` | bool | Proxy pausiert (Verbindungen werden abgelehnt) |
| `connection_timeout` | int | Verbindungs-Timeout in Sekunden |
| `read_timeout` | int | Lese-Timeout in Sekunden |
| `max_retries` | int | Maximale Wiederholungsversuche bei Fehler |
| `max_read_size` | int | Max. Modbus-Read-Größe (0 = unbegrenzt) |
| `description` | string | Optionale Beschreibung |
| `tags` | array | Optionale Tags zur Kategorisierung |

---

## Skripte

### scripts/modbridge.sh

Das Haupt-Verwaltungsskript für Installation, Updates und Service-Management.

```bash
sudo bash scripts/modbridge.sh <Befehl>
```

| Befehl | Beschreibung |
|--------|--------------|
| `install` | Modbridge installieren (Binary-Download oder Quellcode-Build, systemd-Setup) |
| `update` | Auf neue Version aktualisieren (mit automatischem Rollback bei Fehler) |
| `uninstall` | Vollständig deinstallieren (Service + Dateien) |
| `start` | systemd-Service starten |
| `stop` | systemd-Service stoppen |
| `restart` | systemd-Service neu starten |
| `status` | Service-Status anzeigen |
| `logs [N]` | Letzte N Log-Einträge anzeigen (Standard: 50) |

**Beispiele:**
```bash
sudo bash scripts/modbridge.sh install    # Erstinstallation
sudo bash scripts/modbridge.sh update     # Update auf neue Version
sudo bash scripts/modbridge.sh status     # Status prüfen
sudo bash scripts/modbridge.sh logs 100   # Letzte 100 Log-Zeilen
sudo bash scripts/modbridge.sh uninstall  # Deinstallieren
```

**Update-Ablauf:**
1. Service stoppen
2. Altes Binary sichern (Backup)
3. Neues Binary herunterladen
4. Service starten
5. Bei Fehler: automatischer Rollback auf Backup (die letzten 3 Backups werden behalten)

---

### scripts/go-updater.sh

Hält die lokale Go-Installation aktuell. Nützlich, wenn ModBridge aus dem Quellcode gebaut wird.

```bash
sudo bash scripts/go-updater.sh <Befehl>
```

| Befehl | Beschreibung |
|--------|--------------|
| `update` | Go auf die neueste stabile Version aktualisieren |
| `install` | systemd-Service einrichten (Update bei jedem Systemstart) |
| `uninstall` | systemd-Service entfernen |
| `start` | Service manuell ausführen |
| `stop` | Laufenden Service stoppen |
| `status` | Status und letzte Log-Einträge anzeigen |

```bash
sudo bash scripts/go-updater.sh update    # Go jetzt aktualisieren
sudo bash scripts/go-updater.sh install   # Autostart einrichten
```

---

### build.sh

Schnelles lokales Build-Skript (Frontend + Go-Binary).

```bash
./build.sh
```

Führt folgende Schritte aus:
1. `npm install` + `npm run build` im `frontend/`-Verzeichnis
2. Kopiert das Build-Ergebnis nach `pkg/web/dist/`

---

## Makefile-Befehle

```bash
make help           # Alle verfügbaren Befehle anzeigen
```

| Befehl | Beschreibung |
|--------|--------------|
| `make build` | Frontend bauen und Go-Binary kompilieren |
| `make build-frontend` | Nur Frontend bauen und nach pkg/web/dist/ kopieren |
| `make build-all` | Binaries für alle Plattformen (Linux, Windows, macOS) |
| `make run` | Binary bauen und direkt starten |
| `make test` | Tests mit Race-Detector und Coverage ausführen |
| `make coverage` | Coverage-Report als HTML generieren (`coverage.html`) |
| `make lint` | Linter ausführen (benötigt `golangci-lint`) |
| `make fmt` | Code formatieren (`go fmt` + `gofmt`) |
| `make vet` | `go vet` ausführen |
| `make clean` | Build-Artefakte entfernen |
| `make deps` | Go-Abhängigkeiten herunterladen und aufräumen |
| `make update-deps` | Go-Abhängigkeiten auf neueste Versionen aktualisieren |
| `make install` | Binary systemweit installieren (`go install`) |
| `make dev` | Entwicklungsmodus mit Live-Reload (benötigt `air`) |
| `make docker-build` | Docker Image lokal bauen |
| `make docker-run` | Docker Image bauen und Container starten |
| `make docker-stop` | Docker Container stoppen |
| `make docker-logs` | Docker Container Logs anzeigen |

**Beispiele:**
```bash
make build          # Komplett-Build
make test           # Tests ausführen
make coverage       # Testabdeckung prüfen
make build-all      # Für alle Plattformen bauen
```

---

## Umgebungsvariablen

| Variable | Standard | Beschreibung |
|----------|----------|--------------|
| `WEB_PORT` | `:8080` | Web-UI Port (überschreibt `web_port` aus config.json) |
| `LOG_LEVEL` | `INFO` | Log-Level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `TZ` | `UTC` | Zeitzone für den Container |

Für Docker-Deployments: Kopieren Sie `.env.example` nach `.env` und passen Sie die Werte an.

```bash
cp .env.example .env
# .env anpassen
docker-compose up -d
```

---

## API-Endpunkte

| Endpunkt | Methode | Beschreibung |
|----------|---------|--------------|
| `/api/health` | GET | Health Check (kein Login erforderlich) |
| `/api/login` | POST | Anmelden |
| `/api/logout` | POST | Abmelden |
| `/api/proxies` | GET | Alle Proxies auflisten |
| `/api/proxies` | POST | Neuen Proxy anlegen |
| `/api/proxies/{id}` | PUT | Proxy aktualisieren |
| `/api/proxies/{id}` | DELETE | Proxy löschen |
| `/api/proxies/{id}/start` | POST | Proxy starten |
| `/api/proxies/{id}/stop` | POST | Proxy stoppen |
| `/api/config` | GET | Konfiguration abrufen |
| `/api/config` | PUT | Konfiguration speichern |
| `/api/logs` | GET | Log-Einträge abrufen |
| `/api/logs/stream` | GET | Live-Log-Stream (SSE) |
| `/api/devices` | GET | Verbundene Geräte auflisten |
| `/api/metrics` | GET | Prometheus-Metriken (Port `:9090`) |

---

## Troubleshooting

### Port bereits in Verwendung

```bash
# Prozess finden, der den Port belegt
sudo lsof -i :8080

# Prozess beenden
sudo kill -9 <PID>

# Oder anderen Port verwenden (Umgebungsvariable)
WEB_PORT=:9090 ./modbridge
```

### Keine Verbindung zum Zielgerät

```bash
# Erreichbarkeit prüfen
ping 192.168.1.100

# Port prüfen
nc -zv 192.168.1.100 502
```

### Admin-Passwort vergessen

Das Passwort-Hash steht in `config.json` unter `admin_pass_hash`. Löschen Sie den Wert, um beim nächsten Start ein neues zufälliges Passwort zu generieren:

```bash
# Wert leeren (ModBridge muss gestoppt sein)
# In config.json: "admin_pass_hash": ""
sudo bash scripts/modbridge.sh restart
# Neues Passwort erscheint in den Logs
sudo bash scripts/modbridge.sh logs
```

### Docker Container startet nicht

```bash
docker logs modbridge
docker ps -a
```

### systemd-Service Probleme

```bash
sudo bash scripts/modbridge.sh status
journalctl -u modbridge.service -n 100
```

---

## Automatisierter Build (GitHub Actions)

Das Projekt nutzt GitHub Actions für automatisierte Builds und Releases:

1. Frontend wird mit Node.js 22 gebaut
2. Go-Binaries für Linux (AMD64/ARM64) und Windows (AMD64)
3. Docker-Images mit Multi-Arch-Support
4. Automatische Releases bei Tags (`v*`)

**Release erstellen:**
```bash
git tag v1.0.0
git push origin v1.0.0
```

**Benötigte GitHub Secrets für Docker-Push:**
```
DOCKER_USERNAME   = Docker Hub Username
DOCKER_PASSWORD   = Docker Hub Access Token
```

---

## Systemanforderungen

| Ressource | Minimum | Empfohlen |
|-----------|---------|-----------|
| CPU | 1 Core | 2+ Cores |
| RAM | 128 MB | 512 MB |
| Festplatte | 50 MB | 500 MB (inkl. Logs/Backups) |
| Netzwerk | TCP fähig | – |
| Ports | 8080 (Web-UI) | + 5020-5030 (Proxies), 9090 (Metrics) |

**Betriebssysteme:** Linux (AMD64, ARM64), Windows (AMD64), macOS (AMD64, ARM64)

---

## Roadmap

Details zur zukünftigen Entwicklung finden Sie in [ROADMAP.md](ROADMAP.md).

---

## Beitragen

Beiträge sind willkommen! Bitte lesen Sie [CONTRIBUTING.md](CONTRIBUTING.md) für Details.

---

## Lizenz

MIT License - siehe [LICENSE](LICENSE) für Details.

---

## Autor

- **Xerolux** - [GitHub](https://github.com/Xerolux)

---

**Version**: 0.1.0 | **Status**: Beta | **Letzte Aktualisierung**: März 2026
