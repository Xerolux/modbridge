# ModBridge SSH Deployment Guide

Automatisiertes Deployment von ModBridge auf deinen Server via SSH und Kompilierung von Quelle.

## Voraussetzungen

### Auf deinem lokalen PC:
- Git installiert
- SSH-Zugang zum Server (Public Key)
- Bash Shell (Linux/Mac) oder WSL2 (Windows)

### Auf dem Server (192.168.178.196):
- Go 1.26+ installiert
- Node.js 22+ installiert (für Frontend-Build)
- Git installiert
- SSH-Server aktiv
- Ausreichend Speicherplatz (~500MB für Build)

## Installation vorbereiten

### 1. Go auf dem Server installieren (falls nicht vorhanden)

```bash
# SSH zum Server
ssh basti@192.168.178.196

# Go 1.26 installieren
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verifizieren
go version
```

### 2. Node.js auf dem Server installieren (falls nicht vorhanden)

```bash
# Option A: Mit nvm (empfohlen)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
source ~/.bashrc
nvm install 22
nvm use 22

# Option B: Mit apt (Ubuntu/Debian)
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt-get install -y nodejs

# Verifizieren
node --version
npm --version
```

## Deployment ausführen

### Methode 1: Einfach (Standardbenutzer & SSH-Key)

```bash
cd /c/Users/Basti/Documents/GitHub/modbridge
chmod +x deploy.sh
./deploy.sh
```

Das Skript wird automatisch versuchen, dich mit `basti` als Benutzer zu verbinden.

### Methode 2: Benutzerdefiniert (andere Benutzer/Port/Pfad)

```bash
# Mit benutzerdefinierten Variablen
REMOTE_USER=admin \
REMOTE_HOST=192.168.178.196 \
REMOTE_PORT=2222 \
REMOTE_PATH=/home/admin/modbridge \
./deploy.sh
```

### Methode 3: Mit SSH-Key

```bash
# Falls dein SSH-Key an einem speziellen Ort ist
SSH_KEY=~/.ssh/server_key \
./deploy.sh
```

## Was das Skript tut

1. **SSH-Verbindung testen** - Prüft Erreichbarkeit
2. **Tools überprüfen** - Go, Node.js, Git auf dem Server
3. **Repository klonen/aktualisieren** - Holt neuesten Code
4. **Frontend bauen** - npm ci + npm run build
5. **Go-Binary kompilieren** - Mit CGO für SQLite3
6. **Binary deployen** - Backup alt, aktiviere neu
7. **Service neu starten** - Falls Systemd-Service vorhanden
8. **Health-Check** - Prüfe ob Server läuft

## Troubleshooting

### SSH-Verbindung fehlgeschlagen

```bash
# Test SSH-Verbindung manuell
ssh -p 22 basti@192.168.178.196 "echo test"

# Mit Verbose
ssh -v -p 22 basti@192.168.178.196 "echo test"

# Key-Probleme? SSH-Key hinzufügen
ssh-add ~/.ssh/id_rsa
```

### Go nicht installiert

```bash
ssh basti@192.168.178.196
go version  # Prüfe ob installiert
# Falls nicht: Siehe "Installation vorbereiten" oben
```

### Node.js nicht installiert

```bash
# Frontend-Build wird fehlschlagen
# Installation erforderlich (siehe oben)
```

### Binary-Build fehlgeschlagen

```bash
# SSH zum Server
ssh basti@192.168.178.196

# Manueller Build
cd /home/modbridge
go mod download
CGO_ENABLED=1 go build -o modbridge ./main.go

# Errors überprüfen
go build -v ./main.go
```

### Service startet nicht

```bash
# Logs überprüfen
ssh basti@192.168.178.196
sudo journalctl -u modbridge -n 50 -f

# Service-Status
systemctl status modbridge

# Manuell starten
cd /home/modbridge
./modbridge
```

## Service Setup (optional)

Falls du einen Systemd-Service möchtest:

```bash
# SSH zum Server
ssh basti@192.168.178.196

# Service-Datei erstellen
sudo tee /etc/systemd/system/modbridge.service > /dev/null << EOF
[Unit]
Description=ModBridge Modbus TCP Proxy
After=network.target

[Service]
Type=simple
User=basti
WorkingDirectory=/home/modbridge
ExecStart=/home/modbridge/modbridge
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Service laden und starten
sudo systemctl daemon-reload
sudo systemctl enable modbridge
sudo systemctl start modbridge

# Status überprüfen
systemctl status modbridge
```

## Health Check nach Deployment

```bash
# Webseite erreichbar?
curl http://192.168.178.196:8080/api/health

# Logs überprüfen
ssh basti@192.168.178.196 "tail -f /home/modbridge/proxy.log"

# Service-Status
ssh basti@192.168.178.196 "systemctl status modbridge"
```

## Rollback (falls was schiefgeht)

```bash
# SSH zum Server
ssh basti@192.168.178.196

# Sehe verfügbare Backups
ls -lh /home/modbridge/modbridge.backup.*

# Stelle altes Binary wieder her
cp /home/modbridge/modbridge.backup.YYYYMMDD_HHMMSS /home/modbridge/modbridge
chmod +x /home/modbridge/modbridge

# Starte Service neu
sudo systemctl restart modbridge
```

## Häufig verwendete Befehle

```bash
# Deployen
./deploy.sh

# Status überprüfen
ssh basti@192.168.178.196 "systemctl status modbridge"

# Logs live verfolgen
ssh basti@192.168.178.196 "sudo journalctl -u modbridge -f"

# Logs als Datei
ssh basti@192.168.178.196 "tail -1000 /home/modbridge/proxy.log > /tmp/logs.txt" && \
  scp basti@192.168.178.196:/tmp/logs.txt ./proxy.log

# Config exportieren
curl -H "Authorization: Bearer TOKEN" http://192.168.178.196:8080/api/config/export

# Stoppe Service
ssh basti@192.168.178.196 "sudo systemctl stop modbridge"

# Starte Service
ssh basti@192.168.178.196 "sudo systemctl start modbridge"
```

## Performance-Hinweise

- **Compilation**: 30-60 Sekunden (abhängig von CPU)
- **Frontend-Build**: 5-10 Sekunden
- **Downtime**: ~5 Sekunden während Service-Restart

## Umgebungsvariablen

Das Skript verwendet folgende Variablen:

| Variable | Standard | Beschreibung |
|----------|----------|--------------|
| `REMOTE_USER` | `basti` | SSH-Benutzer |
| `REMOTE_HOST` | `192.168.178.196` | Server-Adresse |
| `REMOTE_PORT` | `22` | SSH-Port |
| `REMOTE_PATH` | `/home/modbridge` | Deployment-Verzeichnis |
| `SSH_KEY` | (automatisch) | Pfad zu SSH-Schlüssel |
| `GO_VERSION` | `1.26` | Erforderliche Go-Version |
| `NODE_VERSION` | `22` | Erforderliche Node-Version |

Beispiel mit allen Variablen:
```bash
REMOTE_USER=admin \
REMOTE_HOST=192.168.0.100 \
REMOTE_PORT=2222 \
REMOTE_PATH=/home/admin/apps/modbridge \
SSH_KEY=~/.ssh/custom_key \
./deploy.sh
```
