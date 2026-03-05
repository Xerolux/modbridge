#!/bin/bash

# go-updater.sh
# Automates updating Go to the latest version.
# Features:
# - Find and remove old manual Go installation.
# - Download and install the newest Go version from official source.
# - Start, stop, and status management for systemd service.

set -e

INSTALL_DIR="/usr/local/go"
TEMP_DIR="/tmp/go_updater"
SERVICE_NAME="go-updater.service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"
LOG_FILE="/var/log/go-updater.log"

# Setup logging
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

get_latest_version() {
    # Get latest version from official Go download page API or tags
    # Since we need to parse tags, we can use the official Go repo on GitHub
    # or the golang.org/dl page. We'll use the golang releases JSON API.
    LATEST=$(curl -sSL "https://go.dev/dl/?mode=json" | grep -o '"version": "[^"]*' | head -n 1 | cut -d'"' -f4)
    if [ -z "$LATEST" ]; then
        log "Error: Could not determine the latest Go version."
        exit 1
    fi
    echo "$LATEST"
}

get_current_version() {
    if [ -x "/usr/local/go/bin/go" ]; then
        /usr/local/go/bin/go version | awk '{print $3}'
    elif command -v go >/dev/null 2>&1; then
        go version | awk '{print $3}'
    else
        echo "none"
    fi
}

update_go() {
    log "Checking for Go updates..."

    LATEST_VERSION=$(get_latest_version)
    CURRENT_VERSION=$(get_current_version)

    log "Current version: $CURRENT_VERSION"
    log "Latest version: $LATEST_VERSION"

    if [ "$CURRENT_VERSION" == "$LATEST_VERSION" ]; then
        log "Go is already up to date ($CURRENT_VERSION)."
        return 0
    fi

    log "Updating Go from $CURRENT_VERSION to $LATEST_VERSION..."

    # Determine architecture
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) GOARCH="amd64" ;;
        aarch64) GOARCH="arm64" ;;
        armv7l) GOARCH="armv6l" ;; # Go uses armv6l for 32-bit arm usually
        *) log "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    OS=$(uname -s | tr '[:upper:]' '[:lower:]')

    DOWNLOAD_URL="https://go.dev/dl/${LATEST_VERSION}.${OS}-${GOARCH}.tar.gz"
    TAR_FILE="${TEMP_DIR}/${LATEST_VERSION}.${OS}-${GOARCH}.tar.gz"

    mkdir -p "$TEMP_DIR"

    log "Downloading $DOWNLOAD_URL..."
    curl -sSL "$DOWNLOAD_URL" -o "$TAR_FILE"

    log "Removing old installation at $INSTALL_DIR..."
    rm -rf "$INSTALL_DIR"

    log "Extracting new version..."
    tar -C /usr/local -xzf "$TAR_FILE"

    # Clean up
    rm -rf "$TEMP_DIR"

    log "Update complete. New version:"
    /usr/local/go/bin/go version | tee -a "$LOG_FILE"

    # Ensure /usr/local/go/bin is in PATH for all users if not already
    if [ ! -f "/etc/profile.d/go.sh" ]; then
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
        chmod +x /etc/profile.d/go.sh
        log "Added Go to system PATH in /etc/profile.d/go.sh"
    fi
}

install_service() {
    log "Installing systemd service..."

    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=Go Auto Updater
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=$(realpath "$0") update
StandardOutput=append:$LOG_FILE
StandardError=append:$LOG_FILE

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    log "Service installed and enabled to run at startup."
}

uninstall_service() {
    log "Uninstalling systemd service..."
    if [ -f "$SERVICE_FILE" ]; then
        systemctl disable "$SERVICE_NAME" || true
        rm -f "$SERVICE_FILE"
        systemctl daemon-reload
        log "Service uninstalled."
    else
        log "Service not found."
    fi
}

# Command routing
case "$1" in
    update)
        update_go
        ;;
    install)
        if [ "$EUID" -ne 0 ]; then
            echo "Please run as root to install the service."
            exit 1
        fi
        install_service
        update_go
        ;;
    uninstall)
        if [ "$EUID" -ne 0 ]; then
            echo "Please run as root to uninstall the service."
            exit 1
        fi
        uninstall_service
        ;;
    start)
        if [ "$EUID" -ne 0 ]; then
            echo "Please run as root to start the service."
            exit 1
        fi
        systemctl start "$SERVICE_NAME"
        log "Service started."
        ;;
    stop)
        if [ "$EUID" -ne 0 ]; then
            echo "Please run as root to stop the service."
            exit 1
        fi
        systemctl stop "$SERVICE_NAME"
        log "Service stopped."
        ;;
    status)
        systemctl status "$SERVICE_NAME" || true
        echo ""
        echo "Last log entries:"
        tail -n 10 "$LOG_FILE" 2>/dev/null || echo "No logs yet."
        ;;
    *)
        echo "Usage: $0 {update|install|uninstall|start|stop|status}"
        echo ""
        echo "Commands:"
        echo "  update    - Check for and install Go updates immediately"
        echo "  install   - Install systemd service to update on boot and run update"
        echo "  uninstall - Remove systemd service"
        echo "  start     - Run the update service manually"
        echo "  stop      - Stop the update service (if currently running)"
        echo "  status    - Show service status and recent logs"
        exit 1
        ;;
esac
