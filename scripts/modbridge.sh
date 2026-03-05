#!/bin/bash

# modbridge.sh
# Automates installation, updates, and service management for Modbridge.
# Features:
# - Downloads precompiled binaries from GitHub Releases (no build required)
# - Automatic Go installation as fallback for building from source
# - Automatic service startup and management via systemd

set -e

INSTALL_DIR="/opt/modbridge"
SERVICE_NAME="modbridge.service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"
LOG_FILE="/var/log/modbridge-manager.log"
REPO_URL="https://github.com/Xerolux/modbridge.git"
RELEASES_API="https://api.github.com/repos/Xerolux/modbridge/releases"

# Global variable set by select_modbridge_release()
MODBRIDGE_VERSION=""

# ──────────────────────────────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────────────────────────────

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo "Bitte als root ausführen (z.B. mit sudo)."
        exit 1
    fi
}

# ──────────────────────────────────────────────────────────────────────────────
# Go installation (only needed when building from source)
# ──────────────────────────────────────────────────────────────────────────────

install_go() {
    log "Go wird installiert. Verfügbare Versionen werden abgerufen..."

    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  GOARCH="amd64" ;;
        aarch64) GOARCH="arm64" ;;
        armv7l)  GOARCH="armv6l" ;;
        *) log "Fehler: Nicht unterstützte Architektur: $ARCH"; exit 1 ;;
    esac

    OS=$(uname -s | tr '[:upper:]' '[:lower:]')

    echo ""
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║              Welche Go-Version möchtest du?                  ║"
    echo "╠═══════════════════════════════════════════════════════════════╣"
    echo "║  [1] Release  →  Stabile Version (empfohlen)                 ║"
    echo "║  [2] Beta     →  Vorabversion (neuere Features)              ║"
    echo "║  [3] Alpha    →  Entwicklungsversion (bleeding edge)         ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    read -rp "Wahl [1/2/3] (Standard: 1 - Release): " channel_choice
    channel_choice=${channel_choice:-1}

    case "$channel_choice" in
        1|release|Release) GO_CHANNEL="Release" ;;
        2|beta|Beta)       GO_CHANNEL="Beta" ;;
        3|alpha|Alpha)     GO_CHANNEL="Alpha" ;;
        *) log "Ungültige Eingabe, verwende Release"; GO_CHANNEL="Release" ;;
    esac

    log "Neueste $GO_CHANNEL Go-Version wird ermittelt..."

    GO_VERSIONS=$(curl -sSL "https://go.dev/dl/?mode=json" | grep -o '"version":"go[^"]*"' | cut -d'"' -f4 | sort -V | tac)

    LATEST_GO=""
    for version in $GO_VERSIONS; do
        if [[ "$GO_CHANNEL" == "Release" ]] && [[ ! "$version" =~ (alpha|beta|rc) ]]; then
            LATEST_GO="$version"; break
        elif [[ "$GO_CHANNEL" == "Beta" ]] && [[ "$version" =~ beta ]]; then
            LATEST_GO="$version"; break
        elif [[ "$GO_CHANNEL" == "Alpha" ]] && [[ "$version" =~ alpha ]]; then
            LATEST_GO="$version"; break
        fi
    done

    if [ -z "$LATEST_GO" ]; then
        LATEST_GO=$(echo "$GO_VERSIONS" | head -n 1)
        log "Warnung: Kein $GO_CHANNEL-Release gefunden, verwende: $LATEST_GO"
    fi

    DOWNLOAD_URL="https://go.dev/dl/${LATEST_GO}.${OS}-${GOARCH}.tar.gz"
    TEMP_DIR="/tmp/go_install_$$"
    TAR_FILE="${TEMP_DIR}/${LATEST_GO}.${OS}-${GOARCH}.tar.gz"

    mkdir -p "$TEMP_DIR"
    log "Go $LATEST_GO ($GO_CHANNEL) wird heruntergeladen..."

    if ! curl -fL --progress-bar "$DOWNLOAD_URL" -o "$TAR_FILE"; then
        log "Fehler: Go-Download fehlgeschlagen."
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    log "Alte Go-Installation wird entfernt (falls vorhanden)..."
    rm -rf /usr/local/go

    log "Go $LATEST_GO wird installiert..."
    tar -C /usr/local -xzf "$TAR_FILE"
    rm -rf "$TEMP_DIR"

    if [ ! -f "/etc/profile.d/go.sh" ]; then
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
        chmod +x /etc/profile.d/go.sh
    fi

    log "Go-Installation abgeschlossen."
    /usr/local/go/bin/go version | tee -a "$LOG_FILE"
}

check_go() {
    if ! command -v go &>/dev/null && ! [ -x "/usr/local/go/bin/go" ]; then
        log "Go ist nicht installiert."
        install_go
        export PATH=$PATH:/usr/local/go/bin
    else
        log "Go gefunden: $(go version 2>/dev/null || /usr/local/go/bin/go version)"
    fi
}

check_base_dependencies() {
    log "Abhängigkeiten werden geprüft..."
    local missing=()
    command -v git  &>/dev/null || missing+=("git")
    command -v curl &>/dev/null || missing+=("curl")
    command -v file &>/dev/null || missing+=("file")

    if [ ${#missing[@]} -gt 0 ]; then
        log "Fehler: Folgende Programme fehlen: ${missing[*]}"
        log "Installation z.B. mit: apt install ${missing[*]}"
        exit 1
    fi
}

# ──────────────────────────────────────────────────────────────────────────────
# Release selection
# Sets global MODBRIDGE_VERSION – does NOT use stdout for the result so that
# the function can be called normally (not via $(...) substitution).
# ──────────────────────────────────────────────────────────────────────────────

select_modbridge_release() {
    MODBRIDGE_VERSION=""

    log "Verfügbare Modbridge-Releases werden von GitHub abgerufen..."
    local ALL_RELEASES
    ALL_RELEASES=$(curl -sSL "$RELEASES_API?per_page=20" 2>/dev/null)

    if [ -z "$ALL_RELEASES" ]; then
        log "⚠ GitHub-Releases konnten nicht abgerufen werden."
        return 1
    fi

    # Extract latest tag for each channel
    # JSON lines look like:  "tag_name": "v0.0.3-beta",
    local LATEST_RELEASE LATEST_BETA LATEST_ALPHA
    LATEST_RELEASE=$(echo "$ALL_RELEASES" | grep '"tag_name"' | grep -v 'beta\|alpha\|rc' \
        | head -n 1 | grep -o '"v[^"]*"' | tr -d '"')
    LATEST_BETA=$(echo "$ALL_RELEASES" | grep '"tag_name"' | grep 'beta' \
        | head -n 1 | grep -o '"v[^"]*"' | tr -d '"')
    LATEST_ALPHA=$(echo "$ALL_RELEASES" | grep '"tag_name"' | grep 'alpha' \
        | head -n 1 | grep -o '"v[^"]*"' | tr -d '"')

    # Build display labels
    local RELEASE_LABEL BETA_LABEL ALPHA_LABEL
    RELEASE_LABEL="${LATEST_RELEASE:-nicht verfügbar}"
    BETA_LABEL="${LATEST_BETA:-nicht verfügbar}"
    ALPHA_LABEL="${LATEST_ALPHA:-nicht verfügbar}"

    echo ""
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║         Welche Modbridge-Version möchtest du?                ║"
    echo "╠═══════════════════════════════════════════════════════════════╣"
    printf "║  [1] Release  (Stabil)      →  %-30s ║\n" "$RELEASE_LABEL"
    printf "║  [2] Beta     (Vorabversion) →  %-30s ║\n" "$BETA_LABEL"
    printf "║  [3] Alpha    (Entwicklung)  →  %-30s ║\n" "$ALPHA_LABEL"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    read -rp "Wahl [1/2/3] (Standard: 1 - Release): " release_choice
    release_choice=${release_choice:-1}

    local SELECTED_VERSION=""
    case "$release_choice" in
        1|release|Release) SELECTED_VERSION="$LATEST_RELEASE" ;;
        2|beta|Beta)       SELECTED_VERSION="$LATEST_BETA" ;;
        3|alpha|Alpha)     SELECTED_VERSION="$LATEST_ALPHA" ;;
        *) log "Ungültige Eingabe, verwende Release."; SELECTED_VERSION="$LATEST_RELEASE" ;;
    esac

    # Fallback: if chosen channel has no version, use the absolute latest
    if [ -z "$SELECTED_VERSION" ]; then
        log "⚠ Für diesen Kanal ist keine Version verfügbar. Neueste verfügbare Version wird verwendet."
        SELECTED_VERSION=$(echo "$ALL_RELEASES" | grep '"tag_name"' \
            | head -n 1 | grep -o '"v[^"]*"' | tr -d '"')
    fi

    if [ -z "$SELECTED_VERSION" ]; then
        log "⚠ Keine Releases auf GitHub gefunden."
        return 1
    fi

    log "Gewählte Version: $SELECTED_VERSION"
    MODBRIDGE_VERSION="$SELECTED_VERSION"
    return 0
}

# ──────────────────────────────────────────────────────────────────────────────
# Binary download (primary installation method – no Go required)
# ──────────────────────────────────────────────────────────────────────────────

download_modbridge_binary() {
    log "Precompiled Modbridge-Binary wird heruntergeladen..."

    local ARCH GOARCH
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  GOARCH="amd64" ;;
        aarch64) GOARCH="arm64" ;;
        *) log "⚠ Keine vorkompilierte Binary für Architektur '$ARCH' verfügbar. Quellcode-Build wird gestartet."; return 1 ;;
    esac
    log "Systemarchitektur: linux-${GOARCH}"

    # Let user choose the release (result in $MODBRIDGE_VERSION)
    if ! select_modbridge_release; then
        log "⚠ Release-Auswahl fehlgeschlagen."
        return 1
    fi

    local RELEASE_TAG="$MODBRIDGE_VERSION"

    log "Release-Details für $RELEASE_TAG werden abgerufen..."
    local RELEASE_JSON
    RELEASE_JSON=$(curl -sSL "$RELEASES_API/tags/$RELEASE_TAG" 2>/dev/null)

    if [ -z "$RELEASE_JSON" ]; then
        log "⚠ Release-Details für $RELEASE_TAG konnten nicht abgerufen werden."
        return 1
    fi

    local ASSET_COUNT
    ASSET_COUNT=$(echo "$RELEASE_JSON" | grep -o '"browser_download_url"' | wc -l)
    if [ "$ASSET_COUNT" -eq 0 ]; then
        log "⚠ $RELEASE_TAG enthält keine Release-Assets (keine vorkompilierten Binaries)."
        return 1
    fi

    local BINARY_NAME="modbridge-linux-${GOARCH}"
    local DOWNLOAD_URL
    DOWNLOAD_URL=$(echo "$RELEASE_JSON" \
        | grep -o "\"browser_download_url\": \"[^\"]*${BINARY_NAME}[^\"]*\"" \
        | head -n 1 | cut -d'"' -f4)

    if [ -z "$DOWNLOAD_URL" ]; then
        log "⚠ Kein Binary '${BINARY_NAME}' in Release $RELEASE_TAG gefunden."
        log "  Vorhandene Assets: $(echo "$RELEASE_JSON" | grep -o '"name": "[^"]*"' | grep -v 'tag_name\|target' | cut -d'"' -f4 | tr '\n' ' ')"
        return 1
    fi

    local TEMP_BIN="/tmp/modbridge_temp_$$"
    log "✓ Lade $RELEASE_TAG binary herunter..."
    log "  URL: $DOWNLOAD_URL"

    if ! curl -fL --progress-bar "$DOWNLOAD_URL" -o "$TEMP_BIN"; then
        log "⚠ Download fehlgeschlagen."
        rm -f "$TEMP_BIN"
        return 1
    fi

    chmod +x "$TEMP_BIN"

    # Verify it is a valid ELF executable
    if ! file "$TEMP_BIN" 2>/dev/null | grep -qi "ELF"; then
        log "⚠ Heruntergeladene Datei ist kein gültiges ELF-Binary."
        rm -f "$TEMP_BIN"
        return 1
    fi

    mkdir -p "$INSTALL_DIR"
    mv "$TEMP_BIN" "$INSTALL_DIR/modbridge"
    log "✓ Modbridge $RELEASE_TAG erfolgreich heruntergeladen und installiert."
    return 0
}

# ──────────────────────────────────────────────────────────────────────────────
# Source build (fallback)
# ──────────────────────────────────────────────────────────────────────────────

build_modbridge_from_source() {
    log "Quellcode-Build wird gestartet (dies kann einige Minuten dauern)..."

    check_go   # Install Go only when actually needed

    if [ ! -d "$INSTALL_DIR" ]; then
        log "Repository wird geklont..."
        git clone "$REPO_URL" "$INSTALL_DIR"
    else
        log "Repository existiert bereits. Wird aktualisiert..."
        cd "$INSTALL_DIR"
        git pull
    fi

    cd "$INSTALL_DIR"

    log "Frontend wird gebaut..."
    if [ -f "build.sh" ]; then
        chmod +x build.sh
        ./build.sh
    else
        cd frontend
        export NODE_OPTIONS="--max-old-space-size=2048"
        log "  npm-Abhängigkeiten werden installiert..."
        npm install >/dev/null 2>&1 || npm install
        log "  Frontend wird kompiliert..."
        npm run build
        cd ..
        rm -rf pkg/web/dist/*
        cp -r frontend/dist/* pkg/web/dist/
    fi

    log "Go-Abhängigkeiten werden heruntergeladen..."
    go mod download
    go mod verify

    log "Go-Binary wird kompiliert..."
    CGO_ENABLED=1 go build -ldflags="-s -w" -o modbridge ./main.go

    log "✓ Build erfolgreich abgeschlossen."
}

# ──────────────────────────────────────────────────────────────────────────────
# Install / Update / Service commands
# ──────────────────────────────────────────────────────────────────────────────

install_modbridge() {
    check_root
    check_base_dependencies

    echo ""
    log "🚀 Modbridge-Installation wird gestartet..."
    log "   Installationsverzeichnis: $INSTALL_DIR"

    mkdir -p "$INSTALL_DIR"

    # Prefer binary download; fall back to source build
    if ! download_modbridge_binary; then
        log "⚙ Kein vorkompiliertes Binary verfügbar – wird aus dem Quellcode gebaut..."
        build_modbridge_from_source
    fi

    log "⚙ systemd-Service wird konfiguriert..."
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

    log "✅ Modbridge-Installation abgeschlossen!"
    log "✓ Autostart aktiviert (systemd)"
    log "⏳ Modbridge-Service wird gestartet..."
    systemctl start "$SERVICE_NAME"
    log "✓ Modbridge-Service erfolgreich gestartet."
    echo ""
    log "Status prüfen mit: sudo modbridge status"
}

update_modbridge() {
    check_root
    check_base_dependencies

    if [ ! -d "$INSTALL_DIR" ]; then
        log "❌ Modbridge ist nicht unter $INSTALL_DIR installiert. Bitte zuerst 'modbridge install' ausführen."
        exit 1
    fi

    echo ""
    log "🔄 Modbridge wird aktualisiert..."

    if ! download_modbridge_binary; then
        log "⚙ Kein Binary verfügbar – wird aus dem Quellcode gebaut..."
        build_modbridge_from_source
    fi

    log "⏳ Modbridge-Service wird neu gestartet..."
    systemctl restart "$SERVICE_NAME"
    log "✅ Update abgeschlossen. Service wurde neu gestartet."
    echo ""
}

start_service() {
    check_root
    log "Modbridge-Service wird gestartet..."
    systemctl start "$SERVICE_NAME"
    log "Service gestartet."
}

stop_service() {
    check_root
    log "Modbridge-Service wird gestoppt..."
    systemctl stop "$SERVICE_NAME"
    log "Service gestoppt."
}

status_service() {
    systemctl status "$SERVICE_NAME" || true
}

# ──────────────────────────────────────────────────────────────────────────────
# Command routing
# ──────────────────────────────────────────────────────────────────────────────

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
        echo "║                   Modbridge Manager                          ║"
        echo "╚════════════════════════════════════════════════════════════════╝"
        echo ""
        echo "Verwendung: sudo modbridge {install|update|start|stop|restart|status}"
        echo ""
        echo "Befehle:"
        echo "  install   - Modbridge installieren"
        echo "            (lädt vorkompiliertes Binary herunter oder baut aus Quellcode)"
        echo "            (aktiviert Autostart via systemd)"
        echo ""
        echo "  update    - Modbridge auf die neueste Version aktualisieren"
        echo "            (lädt neues Binary herunter oder baut aus Quellcode)"
        echo ""
        echo "  start     - Modbridge-Service starten"
        echo "  stop      - Modbridge-Service stoppen"
        echo "  restart   - Modbridge-Service neu starten"
        echo "  status    - Status des Modbridge-Service anzeigen"
        echo ""
        echo "Features:"
        echo "  ✓ Lädt vorkompilierte Binaries von GitHub Releases herunter"
        echo "  ✓ Kein Build erforderlich wenn Binary verfügbar ist"
        echo "  ✓ Zeigt verfügbare Versionsnummern im Auswahlmenü an"
        echo "  ✓ Fallback: automatischer Quellcode-Build (inkl. Go-Installation)"
        echo "  ✓ Autostart via systemd"
        echo ""
        echo "Beispiele:"
        echo "  sudo bash modbridge.sh install    # Installieren"
        echo "  sudo bash modbridge.sh update     # Aktualisieren"
        echo "  sudo bash modbridge.sh status     # Status prüfen"
        exit 1
        ;;
esac
