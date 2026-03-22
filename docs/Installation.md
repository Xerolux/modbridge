# Installation

## Quick Install

Für die schnellste Installation auf einem Linux-System (Ubuntu/Debian) können Sie das offizielle Installationsskript verwenden:

```bash
curl -sSL https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh | sudo bash -s install
```

---

## Methode 1: modbridge.sh (empfohlen)

Das Skript `scripts/modbridge.sh` ist der einfachste Weg, ModBridge zu installieren, zu aktualisieren und als systemd-Service zu verwalten.

### Schritt 1: Repository klonen
```bash
git clone https://github.com/Xerolux/modbridge.git
cd modbridge
```

### Schritt 2: Skript ausführen
```bash
sudo bash scripts/modbridge.sh install
```

Das Skript fragt Sie:
1. **Download Binary** (empfohlen, schnell) oder **Aus Quellcode kompilieren** (erfordert Go)
2. Möchten Sie ModBridge als systemd-Service einrichten? (empfohlen für Autostart)

### Weitere Befehle:
* `sudo bash scripts/modbridge.sh update` (Auf neue Version aktualisieren)
* `sudo bash scripts/modbridge.sh status` (Service-Status prüfen)
* `sudo bash scripts/modbridge.sh logs` (Letzte Logs anzeigen)
* `sudo bash scripts/modbridge.sh uninstall` (Komplett deinstallieren)

---

## Methode 2: Docker / Docker Compose

ModBridge ist vollständig Docker-kompatibel.

### Vorgefertigtes Image

```bash
docker pull ghcr.io/xerolux/modbridge:latest

# Container starten
docker run -d \
  --name modbridge \
  -p 8080:8080 \
  -p 5020-5030:5020-5030 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  ghcr.io/xerolux/modbridge:latest
```

### Docker Compose

Erstellen Sie eine `docker-compose.yml`:

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
    environment:
      - WEB_PORT=:8080
      - LOG_LEVEL=INFO
      - TZ=Europe/Berlin
```

Dann ausführen:
```bash
docker-compose up -d
```

---

## Methode 3: Aus Quellcode kompilieren

Wenn Sie Entwickler sind oder eine spezielle Architektur nutzen:

**Voraussetzungen:**
* Go 1.22 oder höher
* Node.js 20 oder höher (für das Frontend)

```bash
git clone https://github.com/Xerolux/modbridge.git
cd modbridge

# Baut das Frontend und das Go-Binary
make build

# Oder alternativ:
./build.sh
go build -o modbridge .

# Starten
./modbridge
```
