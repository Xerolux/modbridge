#!/bin/bash

# modbridge.sh
# Automates installation, updates, and service management for Modbridge.
# Features:
# - Automatic Go installation with configurable release channels (Alpha, Beta, Release)
# - No source code building required - uses precompiled binaries
# - Automatic service startup and management

set -e

INSTALL_DIR="/opt/modbridge"
SERVICE_NAME="modbridge.service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"
LOG_FILE="/var/log/modbridge-manager.log"
REPO_URL="https://github.com/Xerolux/modbridge.git"
BIN_TARGET="/usr/local/bin/modbridge"
RELEASES_API="https://api.github.com/repos/Xerolux/modbridge/releases"

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

install_go() {
    log "Go installation required. Fetching latest Go versions..."

    # Determine system architecture
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) GOARCH="amd64" ;;
        aarch64) GOARCH="arm64" ;;
        armv7l) GOARCH="armv6l" ;;
        *) log "Error: Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    OS=$(uname -s | tr '[:upper:]' '[:lower:]')

    # Get latest Go version with channel selection
    echo ""
    echo "Select Go release channel:"
    echo "1) Release (Stable)"
    echo "2) Beta"
    echo "3) Alpha"
    read -p "Choose [1-3] (default: 1): " channel_choice
    channel_choice=${channel_choice:-1}

    case "$channel_choice" in
        1) GO_CHANNEL="Release" ;;
        2) GO_CHANNEL="Beta" ;;
        3) GO_CHANNEL="Alpha" ;;
        *) log "Invalid choice, using Release"; GO_CHANNEL="Release" ;;
    esac

    log "Fetching latest $GO_CHANNEL Go version..."

    # Fetch Go versions from official API
    GO_VERSIONS=$(curl -sSL "https://go.dev/dl/?mode=json" | grep -o '"version":"go[^"]*"' | cut -d'"' -f4 | sort -V | tac)

    LATEST_GO=""
    for version in $GO_VERSIONS; do
        if [[ "$GO_CHANNEL" == "Release" ]] && [[ ! "$version" =~ (alpha|beta|rc) ]]; then
            LATEST_GO="$version"
            break
        elif [[ "$GO_CHANNEL" == "Beta" ]] && [[ "$version" =~ beta ]]; then
            LATEST_GO="$version"
            break
        elif [[ "$GO_CHANNEL" == "Alpha" ]] && [[ "$version" =~ alpha ]]; then
            LATEST_GO="$version"
            break
        fi
    done

    if [ -z "$LATEST_GO" ]; then
        LATEST_GO=$(echo "$GO_VERSIONS" | head -n 1)
        log "Warning: Could not find $GO_CHANNEL version, using latest: $LATEST_GO"
    fi

    DOWNLOAD_URL="https://go.dev/dl/${LATEST_GO}.${OS}-${GOARCH}.tar.gz"
    TEMP_DIR="/tmp/go_install_$$"
    TAR_FILE="${TEMP_DIR}/${LATEST_GO}.${OS}-${GOARCH}.tar.gz"

    mkdir -p "$TEMP_DIR"

    log "Downloading Go $LATEST_GO ($GO_CHANNEL)..."
    log "URL: $DOWNLOAD_URL"

    if ! curl -sSL "$DOWNLOAD_URL" -o "$TAR_FILE"; then
        log "Error: Failed to download Go"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    log "Removing old Go installation if exists..."
    rm -rf /usr/local/go

    log "Installing Go $LATEST_GO..."
    tar -C /usr/local -xzf "$TAR_FILE"

    # Clean up
    rm -rf "$TEMP_DIR"

    # Ensure PATH is set up
    if [ ! -f "/etc/profile.d/go.sh" ]; then
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
        chmod +x /etc/profile.d/go.sh
    fi

    log "Go installation complete."
    /usr/local/go/bin/go version | tee -a "$LOG_FILE"
}

check_go() {
    if ! command -v go &> /dev/null && ! [ -x "/usr/local/go/bin/go" ]; then
        log "Go is not installed."
        install_go
        # Update PATH for current shell
        export PATH=$PATH:/usr/local/go/bin
    else
        log "Go found: $(go version)"
    fi
}

check_dependencies() {
    log "Checking dependencies..."
    if ! command -v git &> /dev/null; then
        log "Error: git is not installed. Please install git."
        exit 1
    fi
    if ! command -v curl &> /dev/null; then
        log "Error: curl is not installed. Please install curl."
        exit 1
    fi
    check_go
}

select_modbridge_release() {
    log "Fetching available Modbridge releases..."

    # Get all releases from GitHub
    ALL_RELEASES=$(curl -sSL "$RELEASES_API?per_page=20" 2>/dev/null)

    if [ -z "$ALL_RELEASES" ]; then
        log "⚠ Failed to fetch releases from GitHub."
        return 1
    fi

    # Parse releases and ask user which channel they prefer
    echo ""
    echo "Available Modbridge versions:"
    echo "1) Release (Stable) - Latest stable version"
    echo "2) Beta - Pre-release versions"
    echo "3) Alpha - Development versions"
    read -p "Choose [1-3] (default: 1): " release_choice
    release_choice=${release_choice:-1}

    case "$release_choice" in
        1) RELEASE_CHANNEL="release" ;;
        2) RELEASE_CHANNEL="beta" ;;
        3) RELEASE_CHANNEL="alpha" ;;
        *) log "Invalid choice, using Release"; RELEASE_CHANNEL="release" ;;
    esac

    # Find the latest release for the selected channel
    if [ "$RELEASE_CHANNEL" = "release" ]; then
        # Get latest non-prerelease version
        SELECTED_RELEASE=$(echo "$ALL_RELEASES" | grep -o '"tag_name":"[^"]*","target_commitish"[^}]*"prerelease":false' | head -n 1 | grep -o '"tag_name":"[^"]*"' | head -n 1 | cut -d'"' -f4)
    elif [ "$RELEASE_CHANNEL" = "beta" ]; then
        # Get latest beta version
        SELECTED_RELEASE=$(echo "$ALL_RELEASES" | grep -o '"tag_name":"[^"]*beta[^"]*"' | head -n 1 | cut -d'"' -f4)
    else
        # Get latest alpha version
        SELECTED_RELEASE=$(echo "$ALL_RELEASES" | grep -o '"tag_name":"[^"]*alpha[^"]*"' | head -n 1 | cut -d'"' -f4)
    fi

    if [ -z "$SELECTED_RELEASE" ]; then
        log "⚠ No $RELEASE_CHANNEL version found. Using latest available."
        SELECTED_RELEASE=$(echo "$ALL_RELEASES" | grep -o '"tag_name":"[^"]*"' | head -n 1 | cut -d'"' -f4)
    fi

    log "Selected version: $SELECTED_RELEASE ($RELEASE_CHANNEL)"
    echo "$SELECTED_RELEASE"
}

download_modbridge_binary() {
    log "Attempting to download Modbridge precompiled binary..."

    # Determine architecture
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) GOARCH="amd64" ;;
        aarch64) GOARCH="arm64" ;;
        *) log "⚠ Unsupported architecture for binary: $ARCH. Will build from source."; return 1 ;;
    esac

    log "System architecture detected: linux-${GOARCH}"

    # Let user select release channel
    RELEASE_TAG=$(select_modbridge_release)
    if [ -z "$RELEASE_TAG" ]; then
        log "⚠ Could not determine release. Will build from source."
        return 1
    fi

    # Get the release JSON for the selected tag
    log "Fetching release details for $RELEASE_TAG..."
    RELEASE_JSON=$(curl -sSL "$RELEASES_API/tags/$RELEASE_TAG" 2>/dev/null)

    if [ -z "$RELEASE_JSON" ]; then
        log "⚠ Failed to fetch release details. Will build from source."
        return 1
    fi

    # Check if release has any assets
    ASSET_COUNT=$(echo "$RELEASE_JSON" | grep -o '"browser_download_url"' | wc -l)
    if [ "$ASSET_COUNT" -eq 0 ]; then
        log "⚠ No release assets found for $RELEASE_TAG. Will build from source."
        return 1
    fi

    # Find the appropriate binary for this system (linux-amd64, linux-arm64)
    BINARY_NAME="modbridge-linux-${GOARCH}"
    DOWNLOAD_URL=$(echo "$RELEASE_JSON" | grep -o "\"browser_download_url\": \"[^\"]*${BINARY_NAME}[^\"]*\"" | head -n 1 | cut -d'"' -f4)

    if [ -z "$DOWNLOAD_URL" ]; then
        log "⚠ No precompiled binary found for ${BINARY_NAME} in $RELEASE_TAG. Will build from source."
        return 1
    fi

    TEMP_BIN="/tmp/modbridge_temp_$$"

    log "✓ Downloading $RELEASE_TAG binary from GitHub..."
    if ! curl -sSL -L "$DOWNLOAD_URL" -o "$TEMP_BIN"; then
        log "⚠ Failed to download binary. Will build from source."
        rm -f "$TEMP_BIN"
        return 1
    fi

    chmod +x "$TEMP_BIN"

    # Test if binary works
    if ! "$TEMP_BIN" -h 2>/dev/null | grep -q "modbridge\|usage\|help" 2>/dev/null && \
       ! "$TEMP_BIN" --help 2>/dev/null | grep -q "modbridge\|usage\|help" 2>/dev/null; then
        log "⚠ Binary verification failed. Will build from source instead."
        rm -f "$TEMP_BIN"
        return 1
    fi

    mkdir -p "$INSTALL_DIR"
    cp "$TEMP_BIN" "$INSTALL_DIR/modbridge"
    rm -f "$TEMP_BIN"

    log "✓ Modbridge $RELEASE_TAG binary downloaded and verified successfully."
    return 0
}

build_modbridge_from_source() {
    log "📦 Building Modbridge from source (this may take a few minutes)..."

    if [ ! -d "$INSTALL_DIR" ]; then
        log "Cloning repository..."
        git clone "$REPO_URL" "$INSTALL_DIR"
    else
        log "Repository already exists. Updating..."
        cd "$INSTALL_DIR"
        git pull
    fi

    cd "$INSTALL_DIR"

    log "📄 Building frontend..."
    if [ -f "build.sh" ]; then
        chmod +x build.sh
        ./build.sh
    else
        cd frontend
        # Increase Node.js memory limit for build
        export NODE_OPTIONS="--max-old-space-size=2048"
        log "  Installing npm dependencies (may take a minute)..."
        npm install >/dev/null 2>&1 || npm install
        log "  Building frontend with increased memory..."
        npm run build
        cd ..
        rm -rf pkg/web/dist/*
        cp -r frontend/dist/* pkg/web/dist/
    fi

    log "⬇ Downloading Go dependencies..."
    go mod download
    go mod verify

    log "🔨 Compiling Go binary..."
    CGO_ENABLED=1 go build -ldflags="-s -w" -o modbridge ./main.go

    log "✓ Build completed successfully."
}

install_modbridge() {
    check_root
    check_dependencies

    echo ""
    log "🚀 Starting Modbridge installation..."
    log "Installation directory: $INSTALL_DIR"

    mkdir -p "$INSTALL_DIR"

    # Try to download precompiled binary first
    if ! download_modbridge_binary; then
        log "⚙ Binary download not available, will build from source..."
        build_modbridge_from_source
    fi

    log "⚙ Configuring systemd service..."
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

    log "✅ Modbridge installation complete!"
    log "✓ Autostart enabled (systemd)"
    log "⏳ Starting Modbridge service..."
    systemctl start "$SERVICE_NAME"
    log "✓ Modbridge service started successfully."
    echo ""
    log "You can check status with: modbridge status"
}

update_modbridge() {
    check_root
    check_dependencies

    if [ ! -d "$INSTALL_DIR" ]; then
        log "❌ Modbridge is not installed at $INSTALL_DIR. Please run 'modbridge install' first."
        exit 1
    fi

    echo ""
    log "🔄 Updating Modbridge..."

    # Try to download new binary first
    if ! download_modbridge_binary; then
        log "⚙ Building from source..."
        build_modbridge_from_source
    fi

    log "⏳ Restarting Modbridge service..."
    systemctl restart "$SERVICE_NAME"
    log "✅ Update complete. Service restarted."
    echo ""
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
        echo "╔════════════════════════════════════════════════════════════════╗"
        echo "║           Modbridge Manager with Auto Go Install              ║"
        echo "╚════════════════════════════════════════════════════════════════╝"
        echo ""
        echo "Usage: modbridge {install|update|start|stop|restart|status}"
        echo ""
        echo "Commands:"
        echo "  install   - Installs Modbridge with automatic Go setup"
        echo "            (downloads binary if available, builds from source otherwise)"
        echo "            (enables autostart via systemd)"
        echo ""
        echo "  update    - Updates Modbridge to latest version"
        echo "            (downloads new binary or rebuilds from source)"
        echo ""
        echo "  start     - Starts the Modbridge service"
        echo "  stop      - Stops the Modbridge service"
        echo "  restart   - Restarts the Modbridge service"
        echo "  status    - Shows the status of the Modbridge service"
        echo ""
        echo "Features:"
        echo "  ✓ Automatic Go installation (configurable: Release/Beta/Alpha)"
        echo "  ✓ Downloads precompiled binaries when available"
        echo "  ✓ Builds from source as fallback"
        echo "  ✓ Autostart enabled via systemd"
        echo "  ✓ Zero manual configuration needed"
        echo ""
        echo "Examples:"
        echo "  sudo modbridge install    # Install with auto Go setup"
        echo "  sudo modbridge update     # Update to latest version"
        echo "  modbridge status          # Check service status"
        exit 1
        ;;
esac
