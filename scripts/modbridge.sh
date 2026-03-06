#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════════
# ModBridge Installation Script
# Features:
# - Interactive graphical menu (whiptail/dialog)
# - Auto-detect system architecture
# - Choose between WebUI and Headless versions
# - Download correct binary from GitHub Releases
# - Automatic service management
# ═══════════════════════════════════════════════════════════════════════════════

set -e

# ═══════════════════════════════════════════════════════════════════════════════
# Configuration
# ═══════════════════════════════════════════════════════════════════════════════
INSTALL_DIR="/opt/modbridge"
SERVICE_NAME="modbridge.service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"
LOG_FILE="/var/log/modbridge-install.log"
REPO_URL="https://github.com/Xerolux/modbridge"
RELEASES_API="https://api.github.com/repos/Xerolux/modbridge/releases"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ═══════════════════════════════════════════════════════════════════════════════
# Helper Functions
# ═══════════════════════════════════════════════════════════════════════════════

log() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Self-Update Function
# ═══════════════════════════════════════════════════════════════════════════════

self_update() {
    # Skip update if NO_UPDATE env var is set
    if [ "${NO_UPDATE:-}" = "1" ]; then
        return 0
    fi

    # Only update when running with install, update, start, stop, restart commands
    local cmd="${1:-}"
    if [[ ! "$cmd" =~ ^(install|update|start|stop|restart)$ ]]; then
        return 0
    fi

    log_info "Prüfe auf Script-Updates..."

    # Get the script's own location
    local SCRIPT_PATH="${BASH_SOURCE[0]}"
    local SCRIPT_DIR=$(dirname "$SCRIPT_PATH")
    local TEMP_SCRIPT="$SCRIPT_DIR/modbridge.sh.new"

    # Download latest version from GitHub
    local REMOTE_URL="https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh"

    if ! curl -fsSL "$REMOTE_URL" -o "$TEMP_SCRIPT" 2>/dev/null; then
        log_warn "Konnte Script-Update nicht prüfen (Download fehlgeschlagen)"
        rm -f "$TEMP_SCRIPT"
        return 0
    fi

    # Make it executable
    chmod +x "$TEMP_SCRIPT"

    # Compare versions using checksum
    local CURRENT_MD5=$(md5sum "$SCRIPT_PATH" 2>/dev/null | awk '{print $1}')
    local NEW_MD5=$(md5sum "$TEMP_SCRIPT" 2>/dev/null | awk '{print $1}')

    if [ "$CURRENT_MD5" = "$NEW_MD5" ]; then
        log_info "✓ Script ist aktuell"
        rm -f "$TEMP_SCRIPT"
        return 0
    fi

    # Versions are different - update available
    log ""
    log "${YELLOW}══════════════════════════════════════════════════════════════════${NC}"
    log "${BOLD}${YELLOW}🔄 NEUE SCRIPT-VERSION VERFÜGBAR${NC}"
    log "${YELLOW}══════════════════════════════════════════════════════════════════${NC}"
    log ""
    log "Das Script wird automatisch aktualisiert..."
    log ""

    # Replace old script with new one
    if mv "$TEMP_SCRIPT" "$SCRIPT_PATH"; then
        log "${GREEN}✓ Script erfolgreich aktualisiert${NC}"
        log ""
        log "Starte neu mit dem aktualisierten Script..."
        log ""

        # Re-execute this script with the same arguments
        exec bash "$SCRIPT_PATH" "$@"
    else
        log_error "Konnte Script nicht aktualisieren"
        rm -f "$TEMP_SCRIPT"
        return 1
    fi
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$LOG_FILE"
}

log_info() {
    echo -e "${CYAN}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

print_header() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC} ${BOLD}                  ModBridge Installer                      ${NC} ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}                    Version 2.0 - Enhanced                       ${NC} ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "Bitte als root ausführen (z.B. mit sudo)"
        exit 1
    fi
}

check_dependencies() {
    log "Prüfe Abhängigkeiten..."
    local missing=()
    command -v curl  &>/dev/null || missing+=("curl")
    command -v jq    &>/dev/null || missing+=("jq")
    command -v file  &>/dev/null || missing+=("file")
    command -v lsof  &>/dev/null || missing+=("lsof")

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Fehlende Programme: ${missing[*]}"
        log_info "Installation: apt install ${missing[*]}"
        exit 1
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Architecture Detection
# ═══════════════════════════════════════════════════════════════════════════════

detect_architecture() {
    local ARCH=$(uname -m)
    local DETECTED_ARCH=""
    local ARCH_NAME=""

    case "$ARCH" in
        x86_64)
            DETECTED_ARCH="amd64"
            ARCH_NAME="Intel/AMD 64-bit (Standard Server)"
            ;;
        aarch64)
            DETECTED_ARCH="arm64"
            ARCH_NAME="ARM 64-bit (Raspberry Pi 4/5, ARM Server)"
            ;;
        armv7l|armv6l)
            DETECTED_ARCH="arm"
            ARCH_NAME="ARM 32-bit (Raspberry Pi Zero/1/2/3, 32-bit OS)"
            ;;
        i386|i686)
            DETECTED_ARCH="386"
            ARCH_NAME="Intel 32-bit (veraltet)"
            ;;
        *)
            log_error "Unbekannte Architektur: $ARCH"
            exit 1
            ;;
    esac

    echo "$DETECTED_ARCH|$ARCH_NAME"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Cleanup Functions
# ═══════════════════════════════════════════════════════════════════════════════

kill_all_modbridge_processes() {
    log "🔍 Suche nach laufenden Modbridge-Prozessen..."

    local PIDS
    PIDS=$(pgrep -x "modbridge" 2>/dev/null | grep -v "^$$\$" || true)

    # Also try to find by install directory if pgrep doesn't work
    if [ -z "$PIDS" ] && [ -x "$INSTALL_DIR/modbridge" ]; then
        PIDS=$(pidof modbridge 2>/dev/null || true)
    fi

    if [ -n "$PIDS" ]; then
        log "⚠  Gefundene Prozesse: $PIDS"
        log "🔪 Beende alle Modbridge-Prozesse..."

        kill $PIDS 2>/dev/null || true

        local count=0
        while [ $count -lt 10 ]; do
            sleep 0.5
            PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
            if [ -z "$PIDS" ]; then
                log "✓ Alle Prozesse wurden sauber beendet."
                break
            fi
            count=$((count + 1))
        done

        PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
        if [ -n "$PIDS" ]; then
            log_warn "Einige Prozesse laufen noch, erzwinges Beendigung (SIGKILL)..."
            kill -9 $PIDS 2>/dev/null || true
            sleep 1
        fi

        PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
        if [ -n "$PIDS" ]; then
            log_error "Konnte Prozesse nicht beenden: $PIDS"
            return 1
        else
            log "✓ Alle Modbridge-Prozesse beendet."
        fi
    else
        log "✓ Keine laufenden Modbridge-Prozesse gefunden."
    fi

    return 0
}

check_and_wait_for_ports() {
    local PORTS="8080 5020 5021 5022 5023 5024 5025 5026 5027 5028 5029 5030"
    local MAX_WAIT=15
    local WAIT_COUNT=0

    log "🔍 Prüfe ob Ports belegt sind: $PORTS"

    local BLOCKED_PORTS=()
    for port in $PORTS; do
        if lsof -i ":$port" -sTCP:LISTEN -t >/dev/null 2>&1; then
            BLOCKED_PORTS+=("$port")
        fi
    done

    if [ ${#BLOCKED_PORTS[@]} -eq 0 ]; then
        log "✓ Alle Ports sind frei."
        return 0
    fi

    log "⚠  Blockierte Ports gefunden: ${BLOCKED_PORTS[*]}"
    log "⏳ Warte auf Freigabe der Ports (max ${MAX_WAIT}s)..."

    while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
        sleep 1
        WAIT_COUNT=$((WAIT_COUNT + 1))

        BLOCKED_PORTS=()
        for port in "${BLOCKED_PORTS[@]}"; do
            if lsof -i ":$port" -sTCP:LISTEN -t >/dev/null 2>&1; then
                BLOCKED_PORTS+=("$port")
            fi
        done

        if [ ${#BLOCKED_PORTS[@]} -eq 0 ]; then
            log "✓ Alle Ports sind jetzt frei."
            return 0
        fi

        if [ $((WAIT_COUNT % 5)) -eq 0 ]; then
            log "   Noch blockiert: ${BLOCKED_PORTS[*]} (${WAIT_COUNT}s)"
        fi
    done

    log_error "Ports werden nicht freigegeben: ${BLOCKED_PORTS[*]}"
    return 1
}

cleanup_modbridge() {
    log "🧹 Cleanup: Beende alle Prozesse und gib Ports frei..."

    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        log "⏹  Stoppe systemd-Service..."
        systemctl stop "$SERVICE_NAME" 2>/dev/null || true
        sleep 1
    fi

    if ! kill_all_modbridge_processes; then
        log_error "Cleanup fehlgeschlagen. Abbruch."
        return 1
    fi

    if ! check_and_wait_for_ports; then
        log_error "Ports werden nicht freigegeben. Abbruch."
        return 1
    fi

    log "✅ Cleanup erfolgreich: Alle Prozesse beendet, alle Ports frei."
    return 0
}

# ═══════════════════════════════════════════════════════════════════════════════
# Release Download
# ═══════════════════════════════════════════════════════════════════════════════

fetch_available_versions() {
    log "Frage verfügbare Versionen ab..."
    local VERSIONS=$(curl -s "$RELEASES_API" | jq -r '.[].tag_name' | head -10)
    if [ -z "$VERSIONS" ]; then
        log_error "Keine Versionen gefunden"
        exit 1
    fi
    echo "$VERSIONS"
}

download_modbridge_binary() {
    local VERSION=$1
    local VARIANT=$2  # "full" or "headless"
    local ARCH=$3

    local BINARY_NAME="modbridge-linux-${ARCH}"
    if [ "$VARIANT" = "headless" ]; then
        BINARY_NAME="${BINARY_NAME}-headless"
    fi

    local DOWNLOAD_URL="${REPO_URL}/releases/download/${VERSION}/${BINARY_NAME}"
    local TEMP_FILE="/tmp/${BINARY_NAME}"

    log "Lade herunter: $BINARY_NAME"
    log "URL: $DOWNLOAD_URL"

    if curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL" --progress-bar; then
        # Verify it's a valid binary
        if file "$TEMP_FILE" | grep -q "ELF"; then
            mv "$TEMP_FILE" "$INSTALL_DIR/modbridge"
            chmod +x "$INSTALL_DIR/modbridge"
            log "✓ Binary erfolgreich heruntergeladen"
            return 0
        else
            log_error "Heruntergeladene Datei ist keine gültige Binary"
            rm -f "$TEMP_FILE"
            return 1
        fi
    else
        log_error "Download fehlgeschlagen"
        rm -f "$TEMP_FILE"
        return 1
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Installation
# ═══════════════════════════════════════════════════════════════════════════════

install_modbridge() {
    check_root
    check_dependencies

    print_header

    # Detect architecture
    IFS='|' read -r ARCH_INFO ARCH_NAME <<< "$(detect_architecture)"
    log "Erkannte Architektur: ${BOLD}$ARCH_NAME${NC}"

    # Check if whiptail is available
    if ! command -v whiptail &>/dev/null; then
        log_error "whiptail ist nicht installiert"
        log_info "Installation: apt install whiptail"
        exit 1
    fi

    # Show welcome message
    whiptail --title "ModBridge Installer" \
            --yesno "Willkommen zum ModBridge Installer!\n\n\
Erkannte Architektur: $ARCH_NAME\n\n\
ModBridge wird jetzt installiert.\n\n\
Fortfahren?" \
            --yes-button "Ja" --no-button "Nein" \
            15 80 || exit 0

    # Choose WebUI variant
    local WEBUI_VARIANT
    if whiptail --title "WebUI oder Headless?" \
               --radiolist "Wähle die Variante:\n\n\
Mit WebUI = Größere Binary, mit grafischer Oberfläche\n\
Headless = Kleinere Binary (22% weniger), nur Config-Datei" \
               15 80 \
               "Mit WebUI" "ON" \
               "Ohne WebUI (Headless)" "OFF" \
               2>&1 >/dev/tty; then
        WEBUI_VARIANT="headless"
    else
        WEBUI_VARIANT="full"
    fi

    # Cleanup
    if ! cleanup_modbridge; then
        whiptail --title "Fehler" --msgbox "Cleanup fehlgeschlagen. Bitte manuell prüfen." 10 60
        exit 1
    fi

    # Create directory
    mkdir -p "$INSTALL_DIR"
    log "Installationsverzeichnis: $INSTALL_DIR"

    # Select version
    local VERSIONS=$(fetch_available_versions)
    local VERSION_LIST=""
    local i=1
    while IFS= read -r version; do
        if [ $i -eq 1 ]; then
            VERSION_LIST="$version $i ON"
        else
            VERSION_LIST="$VERSION_LIST $version $i OFF"
        fi
        i=$((i+1))
    done <<< "$VERSIONS"

    local SELECTED_VERSION
    SELECTED_VERSION=$(whiptail --title "Version wählen" \
                                    --radiolist "Wähle die zu installierende Version:" \
                                    20 80 \
                                    $VERSION_LIST \
                                    3>&1 >/dev/tty)

    if [ -z "$SELECTED_VERSION" ]; then
        log_info "Installation abgebrochen"
        exit 0
    fi

    log "Gewählte Version: $SELECTED_VERSION"

    # Download binary
    if ! download_modbridge_binary "$SELECTED_VERSION" "$WEBUI_VARIANT" "$ARCH_INFO"; then
        whiptail --title "Download fehlgeschlagen" \
                 --msgbox "Der Download ist fehlgeschlagen.\n\n\
Bitte prüfen:\n\
- Internetverbindung\n\
- GitHub Repository verfügbar\n\
- Version existiert" \
                 12 60
        exit 1
    fi

    # Create systemd service
    log "Erstelle systemd-Service..."
    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=ModBridge Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/modbridge -config $INSTALL_DIR/config.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    # Enable and start service
    log "Aktiviere und starte Service..."
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "✓ Service läuft erfolgreich"
    else
        log_error "Service konnte nicht gestartet werden"
        systemctl status "$SERVICE_NAME" --no-pager
        exit 1
    fi

    # Success message
    whiptail --title "Installation erfolgreich" \
             --msgbox "ModBridge $SELECTED_VERSION wurde erfolgreich installiert!\n\n\
Version: $SELECTED_VERSION\n\
Variante: $([ "$WEBUI_VARIANT" = "headless" ] && echo "Headless (ohne WebUI)" || echo "Full (mit WebUI)")\n\
Architektur: $ARCH_NAME\n\n\
Service: systemctl $SERVICE_NAME {start,stop,restart,status}\n\
Config: $INSTALL_DIR/config.json\n\n\
WebUI: http://$(hostname -I | awk '{print $1}'):8080" \
             15 80
}

# ═══════════════════════════════════════════════════════════════════════════════
# Update
# ═══════════════════════════════════════════════════════════════════════════════

update_modbridge() {
    check_root
    check_dependencies

    if [ ! -d "$INSTALL_DIR" ]; then
        log_error "Modbridge ist nicht unter $INSTALL_DIR installiert"
        log_info "Bitte zuerst 'modbridge.sh install' ausführen"
        exit 1
    fi

    print_header

    # Detect current architecture
    IFS='|' read -r ARCH_INFO ARCH_NAME <<< "$(detect_architecture)"
    log "Erkannte Architektur: ${BOLD}$ARCH_NAME${NC}"

    # Check current installation
    local CURRENT_VARIANT=""
    if file "$INSTALL_DIR/modbridge" 2>/dev/null | grep -q "not stripped"; then
        CURRENT_VARIANT="full (mit WebUI)"
    else
        # Try to detect from size
        local SIZE=$(stat -f%z "$INSTALL_DIR/modbridge" 2>/dev/null || stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null)
        if [ "$SIZE" -lt 8000000 ]; then
            CURRENT_VARIANT="headless (ohne WebUI)"
        else
            CURRENT_VARIANT="full (mit WebUI)"
        fi
    fi

    log "Aktuelle Installation: $CURRENT_VARIANT"

    # Confirm update
    whiptail --title "ModBridge Update" \
            --yesno "ModBridge wird aktualisiert.\n\n\
Architektur: $ARCH_NAME\n\
Aktuell: $CURRENT_VARIANT\n\n\
Fortfahren?" \
            --yes-button "Ja" --no-button "Nein" \
            15 80 || exit 0

    # Choose variant
    local WEBUI_VARIANT
    if whiptail --title "WebUI oder Headless?" \
               --defaultitem \
               --radiolist "Wähle die Variante:\n\n\
Mit WebUI = Größere Binary, mit grafischer Oberfläche\n\
Headless = Kleinere Binary (22% weniger), nur Config-Datei" \
               15 80 \
               "Mit WebUI" "ON" \
               "Ohne WebUI (Headless)" "OFF" \
               2>&1 >/dev/tty; then
        WEBUI_VARIANT="headless"
    else
        WEBUI_VARIANT="full"
    fi

    # Cleanup
    if ! cleanup_modbridge; then
        whiptail --title "Fehler" --msgbox "Cleanup fehlgeschlagen. Bitte manuell prüfen." 10 60
        exit 1
    fi

    # Backup
    if [ -f "$INSTALL_DIR/modbridge" ]; then
        local BACKUP_NAME="modbridge.backup.$(date +%Y%m%d_%H%M%S)"
        cp "$INSTALL_DIR/modbridge" "$INSTALL_DIR/$BACKUP_NAME"
        log "✓ Altes Binary gesichert als: $BACKUP_NAME"
    fi

    # Select version
    local VERSIONS=$(fetch_available_versions)
    local VERSION_LIST=""
    local i=1
    while IFS= read -r version; do
        if [ $i -eq 1 ]; then
            VERSION_LIST="$version $i ON"
        else
            VERSION_LIST="$VERSION_LIST $version $i OFF"
        fi
        i=$((i+1))
    done <<< "$VERSIONS"

    local SELECTED_VERSION
    SELECTED_VERSION=$(whiptail --title "Version wählen" \
                                    --radiolist "Wähle die zu installierende Version:" \
                                    20 80 \
                                    $VERSION_LIST \
                                    3>&1 >/dev/tty)

    if [ -z "$SELECTED_VERSION" ]; then
        log_info "Update abgebrochen"
        exit 0
    fi

    log "Gewählte Version: $SELECTED_VERSION"

    # Download binary
    if ! download_modbridge_binary "$SELECTED_VERSION" "$WEBUI_VARIANT" "$ARCH_INFO"; then
        whiptail --title "Download fehlgeschlagen" \
                 --msgbox "Der Download ist fehlgeschlagen.\n\n\
Bitte prüfen:\n\
- Internetverbindung\n\
- GitHub Repository verfügbar\n\
- Version existiert" \
                 12 60
        exit 1
    fi

    # Restart service
    log "Starte Service neu..."
    systemctl restart "$SERVICE_NAME"

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "✓ Update erfolgreich. Service läuft."
    else
        log_error "Service konnte nicht gestartet werden"
        systemctl status "$SERVICE_NAME" --no-pager

        # Rollback
        local LATEST_BACKUP
        LATEST_BACKUP=$(ls -t "$INSTALL_DIR"/modbridge.backup.* 2>/dev/null | head -n 1)
        if [ -n "$LATEST_BACKUP" ]; then
            log "Rollback wird versucht..."
            cp "$LATEST_BACKUP" "$INSTALL_DIR/modbridge"
            chmod +x "$INSTALL_DIR/modbridge"
            systemctl restart "$SERVICE_NAME" 2>/dev/null || true
            if systemctl is-active --quiet "$SERVICE_NAME"; then
                log "✓ Rollback erfolgreich"
            else
                log_error "Rollback fehlgeschlagen"
            fi
        fi
        exit 1
    fi

    # Cleanup old backups (keep last 3)
    local BACKUP_COUNT
    BACKUP_COUNT=$(ls "$INSTALL_DIR"/modbridge.backup.* 2>/dev/null | wc -l)
    if [ "$BACKUP_COUNT" -gt 3 ]; then
        ls -t "$INSTALL_DIR"/modbridge.backup.* | tail -n +4 | xargs rm -f
        log "✓ Alte Backups aufgeräumt (3 behalten)."
    fi

    whiptail --title "Update erfolgreich" \
             --msgbox "ModBridge wurde erfolgreich auf $SELECTED_VERSION aktualisiert!\n\n\
Variante: $([ "$WEBUI_VARIANT" = "headless" ] && echo "Headless (ohne WebUI)" || echo "Full (mit WebUI)")\n\n\
Service läuft!" \
             12 60
}

# ═══════════════════════════════════════════════════════════════════════════════
# Service Management
# ═══════════════════════════════════════════════════════════════════════════════

start_service() {
    check_root
    log "Starte Modbridge-Service..."
    if ! cleanup_modbridge; then
        log_error "Cleanup fehlgeschlagen"
        exit 1
    fi
    systemctl start "$SERVICE_NAME"
    log "✓ Service gestartet"
}

stop_service() {
    check_root
    log "Stoppe Modbridge-Service..."
    if ! cleanup_modbridge; then
        log_warn "Cleanup nicht vollständig erfolgreich, aber Service-Stop wurde versucht."
    fi
    log "✓ Service gestoppt und Ports freigegeben."
}

restart_service() {
    check_root
    log "Starte Modbridge-Service neu..."
    if ! cleanup_modbridge; then
        log_warn "Cleanup nicht vollständig erfolgreich. Versuche trotzdem Neustart..."
    fi
    systemctl restart "$SERVICE_NAME"
    log "✓ Service neu gestartet."
}

status_service() {
    check_root
    echo "ModBridge Service Status:"
    echo "========================="
    systemctl status "$SERVICE_NAME" --no-pager
    echo ""
    echo "Prozesse:"
    pgrep -a modbridge || echo "Keine Prozesse gefunden"
    echo ""
    echo "Ports:"
    for port in 8080 5020 5021 5022 5023; do
        if lsof -i ":$port" -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo "  Port $port: ${GREEN}BELEGT${NC}"
        else
            echo "  Port $port: ${RED}FREI${NC}"
        fi
    done
}

# ═══════════════════════════════════════════════════════════════════════════════
# Help
# ═══════════════════════════════════════════════════════════════════════════════

show_help() {
    cat <<EOF
${BOLD}ModBridge Installation Script${NC}
${BOLD}============================${NC}

${CYAN}Verwendung:${NC}
  sudo bash modbridge.sh ${GREEN}install${NC}      - Installation mit interaktivem Menü
  sudo bash modbridge.sh ${GREEN}update${NC}       - Update mit interaktivem Menü
  sudo bash modbridge.sh ${GREEN}start${NC}        - Service starten
  sudo bash modbridge.sh ${GREEN}stop${NC}         - Service stoppen
  sudo bash modbridge.sh ${GREEN}restart${NC}      - Service neustarten
  sudo bash modbridge.sh ${GREEN}status${NC}       - Service-Status anzeigen

${CYAN}Optionen:${NC}
  ${YELLOW}NO_UPDATE=1${NC} sudo bash modbridge.sh install
  # Überspringt die automatische Script-Aktualisierung

${CYAN}Features:${NC}
  ✓ Automatische Script-Aktualisierung vor jeder Ausführung
  ✓ Interaktive grafische Menüs (whiptail)
  ✓ Automatische Architektur-Erkennung (amd64, arm64, arm)
  ✓ Wahl zwischen WebUI und Headless-Versionen
  ✓ Download der passenden Binary von GitHub Releases
  ✓ Automatische Service-Verwaltung via systemd
  ✓ Robuster Cleanup: Beendet alle Prozesse, gibt Ports frei
  ✓ Backup-Management mit automatischem Rollback

${CYAN}Unterstützte Architekturen:${NC}
  • ${YELLOW}amd64${NC}  - Intel/AMD 64-bit (Standard Server)
  • ${YELLOW}arm64${NC}  - ARM 64-bit (Raspberry Pi 4/5, ARM Server)
  • ${YELLOW}arm${NC}    - ARM 32-bit (Raspberry Pi Zero/1/2/3, 32-bit OS)

${CYAN}Varianten:${NC}
  • ${GREEN}Full${NC}     - Mit WebUI (~8.8 MB)
  • ${GREEN}Headless${NC} - Ohne WebUI (~6.9 MB, 22% kleiner)

${CYAN}Auto-Update:${NC}
  Das Script prüft vor jedem Befehl (install, update, start, stop, restart),
  ob eine neuere Version im Git Repository verfügbar ist.
  Falls ja, wird es automatisch aktualisiert und neu gestartet.

${CYAN}Beispiel:${NC}
  $ sudo bash modbridge.sh install
  # → Prüft auf Script-Updates
  # → Falls nötig: Aktualisiert sich selbst und startet neu
  # → Öffnet interaktives Menü
  # → Zeigt erkannte Architektur
  # → Auswahl: WebUI oder Headless
  # → Auswahl der Version
  # → Download und Installation

EOF
}

# ═══════════════════════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════════════════════

# Self-update before doing anything else
self_update "$@"

case "${1:-}" in
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
        restart_service
        ;;
    status)
        status_service
        ;;
    *)
        show_help
        ;;
esac
