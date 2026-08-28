# Installation

## Quick Install

For the fastest installation on a Linux system (Ubuntu/Debian), use the official install script:

```bash
curl -sSL https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh | sudo bash -s install
```

---

## Method 1: modbridge.sh (recommended)

The script `scripts/modbridge.sh` is the easiest way to install ModBridge, update it, and manage it as a systemd service.

### Step 1: Clone the repository
```bash
git clone https://github.com/Xerolux/modbridge.git
cd modbridge
```

### Step 2: Run the script
```bash
sudo bash scripts/modbridge.sh install
```

The script will ask you:
1. **Download binary** (recommended, fast) or **compile from source** (requires Go)
2. Whether to set up ModBridge as a systemd service (recommended for autostart)

### Additional commands:
* `sudo bash scripts/modbridge.sh update` (update to a new version)
* `sudo bash scripts/modbridge.sh status` (check service status)
* `sudo bash scripts/modbridge.sh logs` (show recent logs)
* `sudo bash scripts/modbridge.sh uninstall` (uninstall completely)

---

## Method 2: Docker / Docker Compose

ModBridge is fully Docker-compatible.

### Prebuilt image

```bash
docker pull ghcr.io/xerolux/modbridge:latest

# Start the container
docker run -d \
  --name modbridge \
  -p 8080:8080 \
  -p 5020-5030:5020-5030 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  ghcr.io/xerolux/modbridge:latest
```

### Docker Compose

Create a `docker-compose.yml`:

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
    environment:
      - WEB_PORT=:8080
      - LOG_LEVEL=INFO
      - TZ=Europe/Berlin
```

Then run:
```bash
docker-compose up -d
```

---

## Method 3: Compile from source

If you are a developer or use a special architecture:

**Prerequisites:**
* Go 1.26 or later
* Node.js 24 or later (for the frontend)

```bash
git clone https://github.com/Xerolux/modbridge.git
cd modbridge

# Builds the frontend and the Go binary
make build

# Alternatively:
./build.sh
go build -o modbridge .

# Start
./modbridge
```
