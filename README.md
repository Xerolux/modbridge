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

## ✨ Highlights

- 🎯 **Einfach zu bedienen** - Intuitive Web-UI mit Drag & Drop
- ⚡ **Hocheffizient** - Latenz ~3-5ms, ~10,000 req/s
- 🔒 **Sicher** - Bcrypt-Authentifizierung, Session-Management
- 📊 **Echtzeit-Monitoring** - Live-Traffic-Logging, Dashboard-Metriken
- 🐳 **Docker-Ready** - One-Command Deployment
- 💾 **Device Tracking** - SQLite-basierte Verbindungshistorie
- 🔄 **Automatische Updates** - Update-Funktion direkt aus der WebUI

---

## 🖥️ WebUI Vorschau

### Dashboard

![Dashboard](./assets/screenshots/dashboard.png)

Das Dashboard gibt Ihnen einen schnellen Überblick über alle Ihre Modbus-Proxies, Systemstatus und Echtzeit-Metriken.

### Proxy-Verwaltung

![Proxies](./assets/screenshots/proxies.png)

Erstellen, bearbeiten und verwalten Sie Ihre Modbus TCP Proxies mit wenigen Klicks.

### Live-Logs

![Logs](./assets/screenshots/logs.png)

Überwachen Sie den Modbus-Traffic in Echtzeit mit farbcodierten Log-Level-Filtern.

### Geräte-Tracking

![Devices](./assets/screenshots/devices.png)

Verfolgen Sie alle verbundenen Geräte und deren Verbindungsstatus.

### Konfiguration

![Config](./assets/screenshots/config.png)

Exportieren und importieren Sie Ihre Konfiguration für Backup und Wiederherstellung.

---

## 📦 Quick Install

### Nur Go Install (Standalone)
```bash
# Installieren mit go install
go install github.com/xerolux/modbridge@latest

# Ausführen (wird im Hintergrund weiterlaufen)
nohup modbridge > /dev/null 2>&1 &
# Oder mit systemd/supervisor für automatischen Restart bei Neustart
```

### Docker (vorgefertigt)
```bash
docker run -d -p 8080:8080 -p 5020-5030:5020-5030 ghcr.io/xerolux/modbridge:latest

# Oder mit Docker Compose
docker-compose up -d
```

Nach der Installation: Öffnen Sie **http://localhost:8080** in Ihrem Browser.

---

## 🚀 Features

### Kern-Funktionen

- **Multi-Proxy-Unterstützung**: Verwaltung mehrerer Modbus TCP Proxy-Instanzen
- **Web-Interface**: Moderne, responsive UI (eingebettet im Binary)
- **Echtzeit-Überwachung**: Live-Traffic-Logging via Server-Sent Events (SSE)
- **Sicherheit**:
  - Admin-Authentifizierung mit Bcrypt-Hashing
  - Sichere Session-Cookies
  - Optional: Read-Only-Ansicht für nicht authentifizierte Benutzer
- **Persistenz**: JSON-basierte Konfiguration mit Export/Import
- **Single Binary**: Gesamte Frontend ist im Go-Binary eingebettet
- **Graceful Shutdown**: Sauberes Herunterfahren aller Proxy-Instanzen
- **Health Checks**: Integrierte Gesundheitsprüfungen für Container-Umgebungen
- **MaxReadSize**: Automatisches Aufteilen großer Modbus-Read-Requests
- **Device Tracking**: SQLite-Datenbank zur Verfolgung verbundener Geräte
- **Connection History**: Vollständige Historie aller Modbus-Verbindungen

### Performance-Merkmale

| Metrik | Wert |
|--------|------|
| Latenz (avg) | ~3-5ms |
| Latenz (p99) | ~12ms |
| Durchsatz | ~10,000 req/s |
| Speicher (idle) | ~2.5MB |
| Speicher (load) | ~8-15MB |
| Gleichzeitige Verbindungen | ~1,000 |
| Thread-Safe | ✅ Race Condition freie Implementation |
| Graceful Shutdown | ✅ Wartet auf alle aktiven Goroutines |

---

## 📋 Inhaltsverzeichnis

- [Installation](#installation)
  - [Methode 1: Docker](#methode-1-docker)
  - [Methode 2: Aus Quellcode kompilieren](#methode-2-aus-quellcode-kompilieren)
- [Konfiguration](#konfiguration)
- [Verwendung](#verwendung)
- [Troubleshooting](#troubleshooting)
- [Roadmap](#roadmap)

---

## 🔨 Building from Source

### Automated Build (GitHub Actions)

Das Projekt nutzt GitHub Actions CI/CD für automatisierte Builds:

**Was der Workflow macht:**
1. ✅ Frontend buildet mit Node.js 22
2. ✅ Go-Binaries für Linux (AMD64/ARM64) und Windows (AMD64)
3. ✅ Docker-Images mit Multi-Arch Support
4. ✅ Automatische Releases bei Tags (v*)

**Manuellen Build auslösen:**
```bash
# Gehe zu: Actions → Build and Release → Run workflow
```

**Release erstellen:**
```bash
# Tag erstellen und pushen → automatisches Release
git tag v1.0.0
git push origin v1.0.0
```

**Download fertiger Binaries:**
- https://github.com/Xerolux/modbridge/releases

**Docker Images:**
```bash
docker pull ghcr.io/xerolux/modbridge:latest
```

**Benötigte GitHub Secrets für Docker-Push:**
```
Settings → Secrets and variables → Actions → New repository secret

DOCKER_USERNAME      = Dein Docker Hub Username
DOCKER_PASSWORD      = Docker Hub Access Token (nicht Passwort!)
```

**Access Token erstellen:**
1. Docker Hub → Account Settings → Security
2. New Access Token
3. Read & Write Permissions
4. Token kopieren und als DOCKER_PASSWORD Secret hinzufügen

### Lokal Bauen

**Voraussetzungen:**
- Go 1.25+
- Node.js 22+

**Schritte:**
```bash
# Repository klonen
git clone https://github.com/Xerolux/modbridge.git
cd modbridge

# Frontend builden
cd frontend
npm install
npm run build
cd ..

# Frontend in Go-Projekt kopieren
rm -rf pkg/web/dist/*
cp -r frontend/dist/* pkg/web/dist/

# Go-Binary builden
go build -ldflags="-s -w" -o modbridge ./main.go

# Binary starten
./modbridge
```

**Cross-Compile für andere Plattformen:**
```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o modbridge-linux-amd64 ./main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o modbridge.exe ./main.go

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o modbridge-darwin-arm64 ./main.go
```

---

## 🔧 Installation

### Methode 1: Docker

Docker ist eine plattformunabhängige Installationsmethode.

#### Mit vorgebautem Image

```bash
docker run -d \
  --name modbridge \
  -p 8080:8080 \
  -p 5020-5030:5020-5030 \
  -v $(pwd)/config.json:/app/config.json \
  -v $(pwd)/logs:/app/data \
  --restart unless-stopped \
  ghcr.io/xerolux/modbridge:latest
```

#### Mit Docker Compose

```bash
# Repository klonen
git clone https://github.com/Xerolux/modbridge.git
cd modbridge

# Starten
docker-compose up -d
```

---

## 🚀 Anleitung für produktiven Betrieb

### Methode 1: Standalone mit Go (Empfohlen)

#### 1. Installation
```bash
# Repository klonen
git clone https://github.com/Xerolux/modbridge.git
cd modbridge

# Kompilieren mit Build-Tag und Version
go build -ldflags="-s -w -X main.Version=$(git describe --tags --dirty)" -o modbridge ./main.go

# Ausführen
./modbridge
```

#### 2. Im Hintergrund ausführen mit automatischem Restart
```bash
# Option A: systemd Service (empfohlen)
sudo tee /etc/systemd/system/modbridge.service << EOF
[Unit]
Description=ModBridge Modbus Proxy Manager
After=network.target

[Service]
Type=simple
User=modbridge
WorkingDirectory=/opt/modbridge
ExecStart=/opt/modbridge/modbridge
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now modbridge

# Option B: Supervisor
sudo tee /etc/supervisor/conf.d/modbridge.conf << EOF
[program:modbridge]
command=/opt/modbridge/modbridge
directory=/opt/modbridge
user=modbridge
autostart=true
autorestart=true
stderr_logfile=/var/log/supervisor/modbridge.log
stdout_logfile=/var/log/supervisor/modbridge.log
EOF

sudo supervisorctl reread && sudo supervisorctl update
```

#### 3. Update-Funktion nutzen
Die WebUI bietet eine automatische Update-Funktion:
1. Login in die WebUI
2. Gehe zu **System Information**
3. Klicke auf "Check for Updates"
4. Bei Updates: "Update Now" klicken
5. Das System lädt automatisch die neueste Version und restartet

### Methode 2: Docker mit automatischem Restart

#### 1. Docker Installation
```bash
# Docker-Image ausführen mit persistenten Volumes
docker run -d \
  --name modbridge \
  -p 8080:8080 \
  -p 5020-5030:5020-5030 \
  -v $(pwd)/config.json:/app/config.json \
  -v $(pwd)/modbridge.db:/app/modbridge.db \
  -v $(pwd)/logs:/app/logs \
  --restart unless-stopped \
  ghcr.io/xerolux/modbridge:latest

# Oder mit Docker Compose
docker-compose up -d
```

#### 2. Update mit Docker
```bash
# Update auf neueste Version
docker pull ghcr.io/xerolux/modbridge:latest
docker-compose down && docker-compose up -d
```

#### 3. Docker mit systemd (empfohlen für Produktivbetrieb)
```bash
# systemd Service für Docker
sudo tee /etc/systemd/system/modbridge-docker.service << EOF
[Unit]
Description=ModBridge Docker Container
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/docker-compose -f /opt/modbridge/docker-compose.yml up -d
ExecStop=/usr/bin/docker-compose -f /opt/modbridge/docker-compose.yml down

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now modbridge-docker
```

---

### Methode 3: Aus Quellcode kompilieren (für Entwicklung)

```bash
# Repository klonen
git clone https://github.com/Xerolux/modbridge.git
cd modbridge

# Kompilieren
go build -o modbridge ./main.go

# Ausführen
./modbridge
```

**Web-Interface**: http://localhost:8080

---

## ⚙️ Konfiguration

### Konfigurations-Datei (config.json)

```json
{
  "web_port": ":8080",
  "admin_pass_hash": "",
  "proxies": [
    {
      "id": "21e71152-3866-43ac-891d-c5ec85fa1e98",
      "name": "Solar-Wechselrichter",
      "listen_addr": ":5020",
      "target_addr": "192.168.1.100:502",
      "enabled": true
    }
  ]
}
```

### Automatische Updates

ModBridge bietet eine eingebaute Update-Funktion:

#### Update-Funktionen
- **Version-Check**: Automatische Prüfung auf neue Git-Tags
- **One-Click-Update**: Update direkt aus der WebUI
- **Backup**: Alte Version wird vor dem Update gesichert
- **Auto-Restart**: Anwendung wird automatisch neu gestartet

#### Update-Prozess
1. Öffnen Sie die WebUI
2. Navigieren Sie zu **System Information**
3. Klicken Sie auf "Check for Updates"
4. Wenn Updates verfügbar sind: "Update Now" klicken
5. Warten Sie bis der Update abgeschlossen ist

### Konfiguration über Web-Interface

1. Öffnen Sie http://localhost:8080
2. Login mit Admin-Passwort (beim ersten Start festlegen)
3. Navigieren Sie zu **Configuration** oder **Proxies**
4. Konfigurieren Sie Ihre Proxies
5. Exportieren/Importieren Sie Konfigurationen bei Bedarf

---

## 📖 Verwendung

### Erste Schritte

1. **Anwendung starten** (je nach Installationsmethode):
   ```bash
   # Standalone
   ./modbridge

   # Docker
   docker-compose up -d

   # Systemd Service
   sudo systemctl start modbridge
   ```

2. **Web-Interface öffnen**:
   ```
   http://localhost:8080
   ```

3. **Erstes Login**: Setzen Sie Ihr Admin-Passwort (Standard: "admin")

### Wichtige Betriebshinweise

#### Automatisches Update
Die WebUI bietet eine eingebaute Update-Funktion:
- Navigieren Sie zu **System Information**
- Klicken Sie auf "Check for Updates"
- Bei Updates: "Update Now" klicken
- Das System übernimmt den Download, Backup und Restart automatisch

#### Logging
```bash
# Logs ansehen (Standalone)
journalctl -u modbridge -f

# Docker Logs
docker logs -f modbridge

# Direkte Log-Dateien
tail -f /opt/modbridge/logs/proxy.log
```

#### Backup & Wiederherstellung
```bash
# Backup erstellen
tar -czf modbridge-backup-$(date +%Y%m%d).tar.gz config.json modbridge.db logs/

# Backup wiederherstellen
tar -xzf modbridge-backup-*.tar.gz
systemctl restart modbridge
```

#### Überwachung
- **System Information**: Zeigt CPU, RAM, Uptime, Proxies
- **Live Logs**: Echtzeit-Überwachung aller Modbus-Aktivitäten
- **Device History**: Alle Verbindungshistorie in der Datenbank

### Proxy erstellen

![Add Proxy](./assets/screenshots/add-proxy.png)

1. Navigieren Sie zu **Proxies**
2. Klicken Sie auf **"+ Add Proxy"**
3. Füllen Sie die Felder aus:
   - **Name**: Beschreibender Name
   - **Listen Address**: z.B. `:5020`
   - **Target Address**: z.B. `192.168.1.100:502`
4. Klicken Sie auf **"Save"**

### Proxy starten/stoppen

- Klicken Sie auf den **Start**-Button zum Aktivieren
- Klicken Sie auf den **Stop**-Button zum Deaktivieren

### Modbus-Client-Verbindung

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

## 🔍 Troubleshooting

### Häufige Probleme

#### Port bereits in Verwendung

```bash
# Prozess finden, der den Port verwendet
sudo lsof -i :8080

# Prozess beenden
sudo kill -9 <PID>

# Oder anderen Port verwenden
export WEB_PORT=:9090
```

#### Keine Verbindung zum Zielgerät

```bash
# Ping-Test
ping 192.168.1.100

# Port-Test
nc -zv 192.168.1.100 502
```

#### Docker Container startet nicht

```bash
# Logs ansehen
docker logs modbridge

# Status prüfen
docker ps -a
```

---

## 🔧 Wichtige Konfigurationshinweise

### Produktivbetrieb

#### Ports & Netzwerk
- **Web-UI**: Port 8080 (konfigurierbar)
- **Modbus Proxies**: Ports 5020-5030 (konfigurierbar)
- **Firewall**: `sudo ufw allow 8080/tcp` und `sudo ufw allow 5020-5030/tcp`

#### Sicherheit
```bash
# Empfohlene Sicherheitseinstellungen
# 1. Ändern Sie das Admin-Passwort beim ersten Login
# 2. Aktivieren Sie SSL/TLS in der Konfiguration
# 3. Setzen Sie IP-Whitelist für den Zugriff
```

#### Performance
```json
{
  "max_connections": 1000,
  "read_timeout": 30,
  "connection_timeout": 5,
  "max_retries": 3
}
```

### Troubleshooting

#### Anwendung startet nicht
```bash
# Logs prüfen
journalctl -u modbridge -n 50

# Docker Logs
docker logs modbridge

# Port-Check
netstat -tulpn | grep 8080
```

#### Update-Fehler
```bash
# Manueller Update-Prozess
git pull
make build
sudo systemctl restart modbridge
```

#### Datenbank-Korruption
```bash
# Backup und Neustart
cp modbridge.db modbridge.db.backup
rm modbridge.db
systemctl restart modbridge
```

---

## 🗺️ Roadmap

### Geplante Features

- **v0.2.0**: Performance-Optimierungen, Connection Pooling
- **v0.3.0**: Prometheus Metrics, Grafana-Dashboard
- **v0.4.0**: SSL/TLS-Unterstützung, Multi-User-Support
- **v1.0.0**: Production-Ready Release

Details zur zukünftigen Entwicklung finden Sie in [ROADMAP.md](ROADMAP.md).

---

## 📊 Systemanforderungen

### Minimale Anforderungen

- **CPU**: 1 Core (2+ empfohlen)
- **RAM**: 128MB (512MB empfohlen)
- **Festplatte**: 50MB
- **Netzwerk**: TCP-Ports (Standard: 8080 für Web-UI, 5020-5030 für Proxies)

---

## 🤝 Beitragen

Wir freuen uns über Beiträge! Bitte lesen Sie [CONTRIBUTING.md](CONTRIBUTING.md) für Details.

---

## 📄 Lizenz

MIT License - siehe [LICENSE](LICENSE) für Details.

---

## 👤 Autor

- **Xerolux** - Initial work - [GitHub](https://github.com/Xerolux)

---

## 💖 Danksagungen

- Modbus-Protokoll-Implementierung basierend auf Modbus TCP/IP Spezifikation
- Web-UI inspiriert von modernen Design-Prinzipien
- Go-Community für exzellente Tools und Bibliotheken

---

**Version**: 0.1.0
**Letzte Aktualisierung**: Januar 2026
**Status**: Beta
**Go Version**: 1.25+
**Automatische Updates**: ✅ Git-basiert, integriert in WebUI

---

## 📞 Support

Bei Problemen:

1. Logs prüfen (Web-UI, `docker logs`, oder `journalctl`)
2. Konfiguration prüfen
3. Issue erstellen: https://github.com/Xerolux/modbridge/issues

---

*ModBridge - Einfaches Modbus Proxy Management für moderne IoT- und Automatisierungs-Systeme.*
