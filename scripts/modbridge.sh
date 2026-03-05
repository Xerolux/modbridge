#!/bin/bash

# modbridge.sh
# Automates installation, updates, and service management for Modbridge.

set -e

INSTALL_DIR="/opt/modbridge"
SERVICE_NAME="modbridge.service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"
LOG_FILE="/var/log/modbridge-manager.log"
REPO_URL="https://github.com/Xerolux/modbridge.git"
BIN_TARGET="/usr/local/bin/modbridge"

# Setup logging
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo "Please run this command as root (e.g., using sudo)."
        exit 1
    fi
}

check_dependencies() {
    log "Checking dependencies..."
    if ! command -v git &> /dev/null; then
        log "Error: git is not installed."
        exit 1
    fi
    if ! command -v go &> /dev/null; then
        log "Error: go is not installed. Please install Go (1.24+)."
        exit 1
    fi
    if ! command -v npm &> /dev/null; then
        log "Error: npm is not installed. Please install Node.js."
        exit 1
    fi
}

build_modbridge() {
    log "Building Modbridge..."

    cd "$INSTALL_DIR"

    log "Building frontend..."
    if [ -f "build.sh" ]; then
        chmod +x build.sh
        ./build.sh
    else
        cd frontend
        npm install
        npm run build
        cd ..
        rm -rf pkg/web/dist/*
        cp -r frontend/dist/* pkg/web/dist/
    fi

    log "Downloading Go dependencies..."
    go mod download
    go mod verify

    log "Building Go binary..."
    CGO_ENABLED=1 go build -ldflags="-s -w" -o modbridge ./main.go

    log "Build successful."
}

install_modbridge() {
    check_root
    check_dependencies

    log "Installing Modbridge to $INSTALL_DIR..."

    if [ ! -d "$INSTALL_DIR" ]; then
        log "Cloning repository..."
        git clone "$REPO_URL" "$INSTALL_DIR"
    else
        log "Directory $INSTALL_DIR already exists. Updating instead..."
        cd "$INSTALL_DIR"
        git pull
    fi

    build_modbridge

    log "Installing systemd service..."
    cat > "$SERVICE_FILE" << SYSTEMD_EOF
[Unit]
Description=ModBridge Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/modbridge
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
SYSTEMD_EOF

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"

    # Install CLI command
    if [ ! -f "$BIN_TARGET" ] || [ "$(realpath "$BIN_TARGET")" != "$(realpath "$0")" ]; then
        log "Installing modbridge CLI tool to $BIN_TARGET..."
        cp "$0" "$BIN_TARGET"
        chmod +x "$BIN_TARGET"
    fi

    log "Installation complete. Starting service..."
    systemctl start "$SERVICE_NAME"
    log "Modbridge started successfully."
}

update_modbridge() {
    check_root
    check_dependencies

    if [ ! -d "$INSTALL_DIR" ]; then
        log "Modbridge is not installed at $INSTALL_DIR. Please run install first."
        exit 1
    fi

    log "Updating Modbridge..."
    cd "$INSTALL_DIR"

    log "Pulling latest changes..."
    git pull

    build_modbridge

    log "Restarting service..."
    systemctl restart "$SERVICE_NAME"
    log "Update complete and service restarted."
}

start_service() {
    check_root
    log "Starting Modbridge service..."
    systemctl start "$SERVICE_NAME"
    log "Service started."
}

stop_service() {
    check_root
    log "Stopping Modbridge service..."
    systemctl stop "$SERVICE_NAME"
    log "Service stopped."
}

status_service() {
    systemctl status "$SERVICE_NAME" || true
}

# Command routing
case "$1" in
    install)
        install_modbridge
        ;;
    update)
        update_modbridge
        ;;
    start)
        start_service
        ;;
    stop)
        stop_service
        ;;
    restart)
        stop_service
        start_service
        ;;
    status)
        status_service
        ;;
    *)
        echo "Modbridge Manager"
        echo "Usage: modbridge {install|update|start|stop|restart|status}"
        echo ""
        echo "Commands:"
        echo "  install   - Clones repo, builds from source, and installs systemd service"
        echo "  update    - Pulls latest changes, rebuilds, and restarts service"
        echo "  start     - Starts the Modbridge service"
        echo "  stop      - Stops the Modbridge service"
        echo "  restart   - Restarts the Modbridge service"
        echo "  status    - Shows the status of the Modbridge service"
        exit 1
        ;;
esac
