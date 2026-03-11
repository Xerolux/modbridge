#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════════
# ModBridge Installation Script
# Features:
# - Interactive graphical menu (whiptail/dialog)
# - Auto-detect system architecture
# - Choose between WebUI and Headless versions
# - Download correct binary from GitHub Releases
# - Automatic service management
# - Automatic config backup before updates
# - Health check and version info
# - Complete uninstall with confirmation
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

# Auto-installation flag (skip all dialogs with defaults)
AUTO_INSTALL=0
DEFAULT_VARIANT="full"  # Default to WebUI (full)

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
# Installation Check
# ═══════════════════════════════════════════════════════════════════════════════

is_modbridge_installed() {
    [ -f "$INSTALL_DIR/modbridge" ]
}

# Normalize version string by removing 'v' prefix and extracting base version
normalize_version() {
    echo "${1#v}" | cut -d'-' -f1  # Remove 'v' prefix and pre-release suffix
}

# Compare two semantic versions. Returns 0 if equal, 1 if first < second, 2 if first > second
compare_versions() {
    local v1=$(normalize_version "$1")
    local v2=$(normalize_version "$2")

    # If strings are equal after normalization
    if [ "$v1" = "$v2" ]; then
        return 0
    fi

    # Try numeric comparison with version parts
    local IFS='.'
    local i ver1=($v1) ver2=($v2)

    for ((i=${#ver1[@]}; i<${#ver2[@]}; i++)); do
        ver1[i]=0
    done

    for ((i=0; i<${#ver1[@]}; i++)); do
        if [[ -z ${ver2[i]} ]]; then
            ver2[i]=0
        fi
        if ((10#${ver1[i]:-0} < 10#${ver2[i]:-0})); then
            return 1
        elif ((10#${ver1[i]:-0} > 10#${ver2[i]:-0})); then
            return 2
        fi
    done

    return 0
}

get_current_version() {
    if ! is_modbridge_installed; then
        echo ""
        return
    fi

    # Try to get version from binary
    "$INSTALL_DIR/modbridge" -version 2>/dev/null || echo "unbekannt"
}

get_latest_version() {
    local VERSIONS=$(fetch_available_versions)
    echo "$VERSIONS" | head -n 1
}

check_updates_available() {
    if ! is_modbridge_installed; then
        echo "0"
        return
    fi

    local CURRENT=$(get_current_version)
    local LATEST=$(get_latest_version)

    if [ -z "$CURRENT" ] || [ "$CURRENT" = "unbekannt" ]; then
        echo "2"  # Unknown current version
        return
    fi

    if [ -z "$LATEST" ]; then
        echo "0"  # Can't determine latest, assume up to date
        return
    fi

    compare_versions "$CURRENT" "$LATEST"
    local result=$?

    if [ $result -eq 0 ]; then
        echo "0"  # Up to date
    elif [ $result -eq 1 ]; then
        echo "1"  # Update available (current < latest)
    else
        echo "0"  # Already on newer version than release (shouldn't happen)
    fi
}

show_installation_status() {
    if is_modbridge_installed; then
        local CURRENT=$(get_current_version)
        local LATEST=$(get_latest_version)
        local STATUS=$(check_updates_available)

        echo ""
        echo "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo "${BOLD}${CYAN}  ModBridge Installations-Status${NC}"
        echo "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo ""

        if [ "$STATUS" = "0" ]; then
            echo -e "  Status: ${GREEN}✓ Aktuell${NC}"
            echo -e "  Installierte Version: ${BOLD}$CURRENT${NC}"
            echo -e "  Verfügbar: ${GREEN}$LATEST${NC}"
        elif [ "$STATUS" = "1" ]; then
            echo -e "  Status: ${YELLOW}⚠ Update verfügbar${NC}"
            echo -e "  Installierte Version: ${BOLD}$CURRENT${NC}"
            echo -e "  Neu verfügbar: ${GREEN}$LATEST${NC}"
        else
            echo -e "  Status: ${YELLOW}⚠ Installiert, Version unklar${NC}"
            echo -e "  Installierte Version: ${YELLOW}$CURRENT${NC}"
            echo -e "  Neu verfügbar: ${GREEN}$LATEST${NC}"
        fi

        # Get variant
        if [ -f "$INSTALL_DIR/modbridge" ]; then
            local SIZE=$(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || stat -f%z "$INSTALL_DIR/modbridge" 2>/dev/null)
            if [ "$SIZE" -lt 8000000 ]; then
                echo -e "  Variante: Headless (ohne WebUI)"
            else
                echo -e "  Variante: Full (mit WebUI)"
            fi
        fi

        # Service status
        if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
            echo -e "  Service: ${GREEN}Läuft${NC}"
        else
            echo -e "  Service: ${RED}Gestoppt${NC}"
        fi

        echo ""
        echo "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo ""
    else
        echo ""
        echo "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo "${BOLD}${CYAN}  ModBridge ist nicht installiert${NC}"
        echo "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo ""
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

        local NEW_BLOCKED_PORTS=()
        for port in "${BLOCKED_PORTS[@]}"; do
            if lsof -i ":$port" -sTCP:LISTEN -t >/dev/null 2>&1; then
                NEW_BLOCKED_PORTS+=("$port")
            fi
        done

        BLOCKED_PORTS=("${NEW_BLOCKED_PORTS[@]}")

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
    local RESPONSE
    local VERSIONS

    # Fetch with timeout and retry logic
    for attempt in 1 2 3; do
        RESPONSE=$(curl -s --connect-timeout 10 --max-time 30 "$RELEASES_API" 2>&1)
        if [ $? -eq 0 ]; then
            break
        fi
        if [ $attempt -lt 3 ]; then
            log_warn "Verbindung fehlgeschlagen, versuche erneut ($attempt/3)..."
            sleep 2
        fi
    done

    if [ -z "$RESPONSE" ]; then
        log_error "Konnte API nicht erreichen"
        return 1
    fi

    # Parse JSON safely
    VERSIONS=$(echo "$RESPONSE" | jq -r '.[].tag_name' 2>/dev/null | head -10)

    if [ -z "$VERSIONS" ]; then
        log_error "Keine Versionen gefunden"
        return 1
    fi

    echo "$VERSIONS"
    return 0
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
    local TEMP_FILE="/tmp/${BINARY_NAME}.$$"

    log "Lade herunter: $BINARY_NAME"
    log "URL: $DOWNLOAD_URL"

    # Download with timeout and retry logic
    local download_success=0
    for attempt in 1 2 3; do
        if curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL" --progress-bar --connect-timeout 10 --max-time 300; then
            download_success=1
            break
        else
            if [ $attempt -lt 3 ]; then
                log_warn "Download fehlgeschlagen, versuche erneut ($attempt/3)..."
                sleep 3
                rm -f "$TEMP_FILE"
            fi
        fi
    done

    if [ $download_success -eq 0 ]; then
        log_error "Download fehlgeschlagen nach 3 Versuchen"
        rm -f "$TEMP_FILE"
        return 1
    fi

    # Check if file exists and has content
    if [ ! -f "$TEMP_FILE" ] || [ ! -s "$TEMP_FILE" ]; then
        log_error "Heruntergeladene Datei ist leer oder existiert nicht"
        rm -f "$TEMP_FILE"
        return 1
    fi

    # Verify it's a valid binary
    if file "$TEMP_FILE" | grep -q "ELF"; then
        mv "$TEMP_FILE" "$INSTALL_DIR/modbridge"
        chmod +x "$INSTALL_DIR/modbridge"
        log "✓ Binary erfolgreich heruntergeladen"
        return 0
    else
        log_error "Heruntergeladene Datei ist keine gültige Binary"
        log_error "Dateityp: $(file "$TEMP_FILE")"
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

    # Check if already installed and show status (unless --force is used)
    if is_modbridge_installed && [ "${MODBRIDGE_FORCE:-0}" != "1" ]; then
        local CURRENT=$(get_current_version)
        local LATEST=$(get_latest_version)
        local STATUS=$(check_updates_available)

        show_installation_status

        if [ "$STATUS" = "1" ] || [ "$STATUS" = "2" ]; then
            # Update available or version unknown
            log "${YELLOW}ModBridge ist bereits installiert (Version: $CURRENT)${NC}"
            log "${GREEN}Neuere Version verfügbar: $LATEST${NC}"
            echo ""

            if command -v whiptail &>/dev/null; then
                if whiptail --title "Update verfügbar" \
                        --yesno "ModBridge ist bereits installiert.\n\n\
Aktuell: $CURRENT\n\
Verfügbar: $LATEST\n\n\
Möchten Sie auf die neueste Version aktualisieren?" \
                        12 80 \
                        --yes-button "Ja, aktualisieren" --no-button "Abbrechen" \
                        3>&1 1>&2 2>&3; then
                    log "Starte Update-Prozess..."
                    update_modbridge
                    return $?
                else
                    log_info "Update abgebrochen"
                    log_info "Verwenden Sie 'modbridge.sh update' für ein manuelles Update"
                    log_info "Verwenden Sie 'modbridge.sh install --force' für eine Neuinstallation"
                    return 0
                fi
            else
                log_info "Verwenden Sie 'modbridge.sh update' zum Aktualisieren"
                log_info "Verwenden Sie 'modbridge.sh install --force' für eine Neuinstallation"
                return 0
            fi
        else
            # Up to date
            log "${GREEN}ModBridge ist bereits installiert und auf dem aktuellen Stand!${NC}"
            log "Version: $CURRENT"
            echo ""

            if command -v whiptail &>/dev/null; then
                if whiptail --title "Bereits installiert" \
                        --yesno "ModBridge ist bereits installiert und aktuell.\n\n\
Version: $CURRENT\n\n\
Möchten Sie die Installation wiederholen (Neuinstallation)?" \
                        12 80 \
                        --yes-button "Ja, neu installieren" --no-button "Abbrechen" \
                        3>&1 1>&2 2>&3; then
                    log "Starte Neuinstallation..."
                    # Continue with normal installation
                else
                    log_info "Installation abgebrochen"
                    log_info "Verwenden Sie 'modbridge.sh update' für ein Update"
                    log_info "Verwenden Sie 'modbridge.sh install --force' für eine erzwungene Neuinstallation"
                    return 0
                fi
            else
                log_info "Verwenden Sie 'modbridge.sh update' für ein Update"
                log_info "Verwenden Sie 'modbridge.sh install --force' für eine erzwungene Neuinstallation"
                return 0
            fi
        fi
    fi

    print_header

    # Detect architecture
    IFS='|' read -r ARCH_INFO ARCH_NAME <<< "$(detect_architecture)"
    log "Erkannte Architektur: ${BOLD}$ARCH_NAME${NC}"

    # Check if whiptail is available (only needed if not in auto-install mode)
    if [ $AUTO_INSTALL -eq 0 ] && ! command -v whiptail &>/dev/null; then
        log_error "whiptail ist nicht installiert"
        log_info "Installation: apt install whiptail"
        exit 1
    fi

    # Choose WebUI variant
    local WEBUI_VARIANT
    if [ $AUTO_INSTALL -eq 1 ]; then
        # Use default variant in auto-install mode
        WEBUI_VARIANT="$DEFAULT_VARIANT"
        log "✓ Verwende Standard-Variante: $([ "$WEBUI_VARIANT" = "full" ] && echo "Mit WebUI" || echo "Headless")"
    else
        # Show welcome message
        whiptail --title "ModBridge Installer" \
                --yesno "Willkommen zum ModBridge Installer!\n\n\
Erkannte Architektur: $ARCH_NAME\n\n\
ModBridge wird jetzt installiert.\n\n\
Fortfahren?" \
                15 80 \
                --yes-button "Ja" --no-button "Nein" \
                3>&1 1>&2 2>&3 || exit 0

        local CHOICE=$(whiptail --title "WebUI oder Headless?" \
                   --radiolist "Wähle die Variante:\n\n\
Mit WebUI = Größere Binary, mit grafischer Oberfläche\n\
Headless = Kleinere Binary (22% weniger), nur Config-Datei" \
                   15 80 2 \
                   "full" "Mit WebUI" "ON" \
                   "headless" "Ohne WebUI (Headless)" "OFF" \
                   3>&1 1>&2 2>&3)

        if [ "$CHOICE" = "headless" ]; then
            WEBUI_VARIANT="headless"
        else
            WEBUI_VARIANT="full"
        fi
    fi

    # Cleanup
    if ! cleanup_modbridge; then
        if [ $AUTO_INSTALL -eq 0 ] && command -v whiptail &>/dev/null; then
            whiptail --title "Fehler" --msgbox "Cleanup fehlgeschlagen. Bitte manuell prüfen." 10 60 || true
        fi
        exit 1
    fi

    # Create directory
    mkdir -p "$INSTALL_DIR"
    log "Installationsverzeichnis: $INSTALL_DIR"

    # Select version
    local VERSIONS
    if ! VERSIONS=$(fetch_available_versions); then
        log_error "Konnte verfügbare Versionen nicht abrufen"
        exit 1
    fi

    local SELECTED_VERSION
    if [ $AUTO_INSTALL -eq 1 ]; then
        # Use latest version (first one) in auto-install mode
        SELECTED_VERSION=$(echo "$VERSIONS" | head -n 1)
        log "✓ Verwende neueste Version: $SELECTED_VERSION"
    else
        local VERSION_COUNT=$(echo "$VERSIONS" | wc -l)
        local LIST_HEIGHT=$((VERSION_COUNT + 2))

        local VERSION_LIST=""
        local i=1
        while IFS= read -r version; do
            if [ $i -eq 1 ]; then
                VERSION_LIST="$VERSION_LIST \"$version\" \"Version $i\" \"ON\""
            else
                VERSION_LIST="$VERSION_LIST \"$version\" \"Version $i\" \"OFF\""
            fi
            i=$((i+1))
        done <<< "$VERSIONS"

        eval "SELECTED_VERSION=\$(whiptail --title \"Version wählen\" \
                                        --radiolist \"Wähle die zu installierende Version:\" \
                                        20 80 $LIST_HEIGHT \
                                        $VERSION_LIST \
                                        3>&1 1>&2 2>&3)"

        if [ -z "$SELECTED_VERSION" ]; then
            log_info "Installation abgebrochen"
            exit 0
        fi
    fi

    log "Gewählte Version: $SELECTED_VERSION"

    # Download binary
    if ! download_modbridge_binary "$SELECTED_VERSION" "$WEBUI_VARIANT" "$ARCH_INFO"; then
        if [ $AUTO_INSTALL -eq 0 ] && command -v whiptail &>/dev/null; then
            whiptail --title "Download fehlgeschlagen" \
                     --msgbox "Der Download ist fehlgeschlagen.\n\n\
Bitte prüfen:\n\
- Internetverbindung\n\
- GitHub Repository verfügbar\n\
- Version existiert" \
                     12 60 || true
        fi
        exit 1
    fi

    # Create default config.json for headless installations
    if [ "$WEBUI_VARIANT" = "headless" ]; then
        log "Erstelle Standard-Konfiguration für Headless-Betrieb..."
        if ! "$INSTALL_DIR/modbridge" -config > "$INSTALL_DIR/config.json" 2>/dev/null; then
            log_warn "Konnte Standard-Konfiguration nicht erstellen"
            log_info "Sie können sie später manuell erstellen mit:"
            log_info "  $ $INSTALL_DIR/modbridge -config > $INSTALL_DIR/config.json"
        else
            log "✓ Standard-Konfiguration erstellt: $INSTALL_DIR/config.json"
        fi
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
    echo ""
    log "${GREEN}═══════════════════════════════════════════════════════════════════${NC}"
    log "${GREEN}✓ Installation erfolgreich!${NC}"
    log "${GREEN}═══════════════════════════════════════════════════════════════════${NC}"
    log ""
    log "Version: ${BOLD}$SELECTED_VERSION${NC}"
    log "Variante: $([ "$WEBUI_VARIANT" = "headless" ] && echo "Headless (ohne WebUI)" || echo "Full (mit WebUI)")"
    log "Architektur: ${BOLD}$ARCH_NAME${NC}"
    log ""

    if [ "$WEBUI_VARIANT" = "headless" ]; then
        log "KONFIGURATION:"
        log "  Config-Datei: $INSTALL_DIR/config.json"
        log "  Erstellen Sie die Konfiguration mit:"
        log "  $ sudo $INSTALL_DIR/modbridge -config > $INSTALL_DIR/config.json"
    else
        local IP=$(hostname -I | awk '{print $1}')
        log "WebUI: http://$IP:8080"
        log "Authentifizierung: Automatisch beim ersten Start generiert"
    fi

    log ""
    log "Service-Befehle:"
    log "  systemctl start $SERVICE_NAME"
    log "  systemctl stop $SERVICE_NAME"
    log "  systemctl restart $SERVICE_NAME"
    log "  systemctl status $SERVICE_NAME"
    log ""

    # Show GUI message if whiptail is available and not in auto-install mode
    if [ $AUTO_INSTALL -eq 0 ] && command -v whiptail &>/dev/null; then
        if [ "$WEBUI_VARIANT" = "headless" ]; then
            whiptail --title "Installation erfolgreich" \
                     --msgbox "ModBridge $SELECTED_VERSION wurde erfolgreich installiert!\n\n\
Version: $SELECTED_VERSION\n\
Variante: Headless (ohne WebUI)\n\
Architektur: $ARCH_NAME\n\n\
${BOLD}KONFIGURATION:${NC}\n\
Die Konfiguration erfolgt über eine JSON-Datei:\n\n\
📁 Config-Datei: $INSTALL_DIR/config.json\n\n\
So erstellen Sie die Konfiguration:\n\
  1. Standard-Konfiguration erstellen:\n\
     $ $INSTALL_DIR/modbridge -config > $INSTALL_DIR/config.json\n\n\
  2. Konfiguration bearbeiten:\n\
     $ sudo nano $INSTALL_DIR/config.json\n\n\
  3. Service neu starten:\n\
     $ sudo systemctl restart $SERVICE_NAME\n\n\
Service: systemctl $SERVICE_NAME {start,stop,restart,status}\n\n\
${BOLD}DOKUMENTATION:${NC}\n\
Ausführliche Informationen finden Sie in der README.md\n\
im GitHub Repository oder mit: modbridge.sh help" \
                     22 80 || true
        else
            local IP=$(hostname -I | awk '{print $1}')
            whiptail --title "Installation erfolgreich" \
                     --msgbox "ModBridge $SELECTED_VERSION wurde erfolgreich installiert!\n\n\
Version: $SELECTED_VERSION\n\
Variante: Full (mit WebUI)\n\
Architektur: $ARCH_NAME\n\n\
Service: systemctl $SERVICE_NAME {start,stop,restart,status}\n\
Config: $INSTALL_DIR/config.json\n\n\
WebUI: http://$IP:8080\n\n\
Das Standard-Passwort wurde beim ersten Start\n\
automatisch generiert und in den Logs angezeigt." \
                     18 80 || true
        fi
    fi
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

    # Show current version info
    if is_modbridge_installed; then
        local CURRENT=$(get_current_version)
        local LATEST=$(get_latest_version)
        log "Aktuelle Version: ${BOLD}$CURRENT${NC}"
        log "Verfügbare Version: ${BOLD}$LATEST${NC}"
        echo ""
    fi

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

    # Choose variant
    local WEBUI_VARIANT
    if [ $AUTO_INSTALL -eq 1 ]; then
        # Use default variant in auto-install mode
        WEBUI_VARIANT="$DEFAULT_VARIANT"
        log "✓ Verwende Standard-Variante: $([ "$WEBUI_VARIANT" = "full" ] && echo "Mit WebUI" || echo "Headless")"
    else
        # Confirm update with dialog
        if command -v whiptail &>/dev/null; then
            whiptail --title "ModBridge Update" \
                    --yesno "ModBridge wird aktualisiert.\n\n\
Architektur: $ARCH_NAME\n\
Aktuell: $CURRENT_VARIANT\n\n\
Fortfahren?" \
                    15 80 \
                    --yes-button "Ja" --no-button "Nein" \
                    3>&1 1>&2 2>&3 || exit 0

            local CHOICE=$(whiptail --title "WebUI oder Headless?" \
                       --radiolist "Wähle die Variante:\n\n\
Mit WebUI = Größere Binary, mit grafischer Oberfläche\n\
Headless = Kleinere Binary (22% weniger), nur Config-Datei" \
                       15 80 2 \
                       "full" "Mit WebUI" "ON" \
                       "headless" "Ohne WebUI (Headless)" "OFF" \
                       3>&1 1>&2 2>&3)

            if [ "$CHOICE" = "headless" ]; then
                WEBUI_VARIANT="headless"
            else
                WEBUI_VARIANT="full"
            fi
        else
            log_error "whiptail nicht verfügbar"
            exit 1
        fi
    fi

    # Cleanup
    if ! cleanup_modbridge; then
        whiptail --title "Fehler" --msgbox "Cleanup fehlgeschlagen. Bitte manuell prüfen." 10 60
        exit 1
    fi

    # Backup config before update
    if [ -f "$INSTALL_DIR/config.json" ]; then
        backup_config
    fi

    # Backup old binary
    if [ -f "$INSTALL_DIR/modbridge" ]; then
        local BACKUP_NAME="modbridge.backup.$(date +%Y%m%d_%H%M%S)"
        cp "$INSTALL_DIR/modbridge" "$INSTALL_DIR/$BACKUP_NAME"
        log "✓ Altes Binary gesichert als: $BACKUP_NAME"
    fi

    # Select version
    local VERSIONS
    if ! VERSIONS=$(fetch_available_versions); then
        log_error "Konnte verfügbare Versionen nicht abrufen"
        exit 1
    fi

    local SELECTED_VERSION
    if [ $AUTO_INSTALL -eq 1 ]; then
        # Use latest version (first one) in auto-install mode
        SELECTED_VERSION=$(echo "$VERSIONS" | head -n 1)
        log "✓ Verwende neueste Version: $SELECTED_VERSION"
    else
        local VERSION_COUNT=$(echo "$VERSIONS" | wc -l)
        local LIST_HEIGHT=$((VERSION_COUNT + 2))

        local VERSION_LIST=""
        local i=1
        while IFS= read -r version; do
            if [ $i -eq 1 ]; then
                VERSION_LIST="$VERSION_LIST \"$version\" \"Version $i\" \"ON\""
            else
                VERSION_LIST="$VERSION_LIST \"$version\" \"Version $i\" \"OFF\""
            fi
            i=$((i+1))
        done <<< "$VERSIONS"

        eval "SELECTED_VERSION=\$(whiptail --title \"Version wählen\" \
                                        --radiolist \"Wähle die zu installierende Version:\" \
                                        20 80 $LIST_HEIGHT \
                                        $VERSION_LIST \
                                        3>&1 1>&2 2>&3)"

        if [ -z "$SELECTED_VERSION" ]; then
            log_info "Update abgebrochen"
            exit 0
        fi
    fi

    log "Gewählte Version: $SELECTED_VERSION"

    # Download binary
    if ! download_modbridge_binary "$SELECTED_VERSION" "$WEBUI_VARIANT" "$ARCH_INFO"; then
        if [ $AUTO_INSTALL -eq 0 ] && command -v whiptail &>/dev/null; then
            whiptail --title "Download fehlgeschlagen" \
                     --msgbox "Der Download ist fehlgeschlagen.\n\n\
Bitte prüfen:\n\
- Internetverbindung\n\
- GitHub Repository verfügbar\n\
- Version existiert" \
                     12 60 || true
        fi
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

    echo ""
    log "${GREEN}═══════════════════════════════════════════════════════════════════${NC}"
    log "${GREEN}✓ Update erfolgreich!${NC}"
    log "${GREEN}═══════════════════════════════════════════════════════════════════${NC}"
    log ""
    log "Version: ${BOLD}$SELECTED_VERSION${NC}"
    log "Variante: $([ "$WEBUI_VARIANT" = "headless" ] && echo "Headless (ohne WebUI)" || echo "Full (mit WebUI)")"
    log "Service: ✓ läuft"
    log ""

    # Show GUI message if whiptail is available and not in auto-install mode
    if [ $AUTO_INSTALL -eq 0 ] && command -v whiptail &>/dev/null; then
        whiptail --title "Update erfolgreich" \
                 --msgbox "ModBridge wurde erfolgreich auf $SELECTED_VERSION aktualisiert!\n\n\
Variante: $([ "$WEBUI_VARIANT" = "headless" ] && echo "Headless (ohne WebUI)" || echo "Full (mit WebUI)")\n\n\
Service läuft!" \
                 12 60 || true
    fi
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
# Logs
# ═══════════════════════════════════════════════════════════════════════════════

logs_service() {
    check_root
    local LINES="${2:-50}"

    if [ "${1:-}" = "--follow" ] || [ "${1:-}" = "-f" ]; then
        log "Zeige Live-Logs (Ctrl+C zum Beenden)..."
        journalctl -u "$SERVICE_NAME" -f
    else
        echo "=== Letzte $LINES Log-Einträge ==="
        journalctl -u "$SERVICE_NAME" -n "$LINES" --no-pager
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Version
# ═══════════════════════════════════════════════════════════════════════════════

version_service() {
    echo "ModBridge Version:"
    echo "=================="
    echo ""

    # Script version
    echo "Script-Version: 2.1 - Enhanced"

    # Check if modbridge binary exists
    if [ -f "$INSTALL_DIR/modbridge" ]; then
        echo "Installiert: $INSTALL_DIR/modbridge"

        # Get version from binary
        local VERSION=$("$INSTALL_DIR/modbridge" -version 2>/dev/null || echo "Unbekannt")
        echo "Binary-Version: $VERSION"

        # Get binary size
        local SIZE=$(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || stat -f%z "$INSTALL_DIR/modbridge" 2>/dev/null)
        local SIZE_MB=$(awk "BEGIN {printf \"%.2f\", $SIZE/1024/1024}")
        echo "Binary-Größe: ${SIZE_MB} MB"

        # Get variant
        if [ "$SIZE" -lt 8000000 ]; then
            echo "Variante: Headless (ohne WebUI)"
        else
            echo "Variante: Full (mit WebUI)"
        fi

        # Check if service is running
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            echo "Service: ${GREEN}Aktiv${NC}"
        else
            echo "Service: ${RED}Inaktiv${NC}"
        fi
    else
        echo "${YELLOW}ModBridge ist nicht installiert${NC}"
    fi
    echo ""
}

# ═══════════════════════════════════════════════════════════════════════════════
# Health Check
# ═══════════════════════════════════════════════════════════════════════════════

health_check() {
    local EXIT_CODE=0

    echo "ModBridge Health Check:"
    echo "====================="
    echo ""

    # Check binary
    if [ -f "$INSTALL_DIR/modbridge" ]; then
        echo -e "[${GREEN}✓${NC}] Binary vorhanden: $INSTALL_DIR/modbridge"
    else
        echo -e "[${RED}✗${NC}] Binary nicht gefunden"
        EXIT_CODE=1
    fi

    # Check config
    if [ -f "$INSTALL_DIR/config.json" ]; then
        echo -e "[${GREEN}✓${NC}] Konfiguration vorhanden: $INSTALL_DIR/config.json"
    else
        echo -e "[${YELLOW}⚠${NC}] Konfiguration nicht gefunden"
    fi

    # Check service file
    if [ -f "$SERVICE_FILE" ]; then
        echo -e "[${GREEN}✓${NC}] Service-Datei vorhanden: $SERVICE_FILE"
    else
        echo -e "[${RED}✗${NC}] Service-Datei nicht gefunden"
        EXIT_CODE=1
    fi

    # Check if service is running
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo -e "[${GREEN}✓${NC}] Service läuft"
    else
        echo -e "[${RED}✗${NC}] Service läuft nicht"
        EXIT_CODE=1
    fi

    # Check if service is enabled
    if systemctl is-enabled --quiet "$SERVICE_NAME"; then
        echo -e "[${GREEN}✓${NC}] Service ist aktiviert (Autostart)"
    else
        echo -e "[${YELLOW}⚠${NC}] Service ist nicht aktiviert"
    fi

    # Check ports
    echo ""
    echo "Port-Status:"
    local PORTS="8080 5020 5021 5022 5023"
    for port in $PORTS; do
        if lsof -i ":$port" -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo -e "  Port $port: [${GREEN}BELEGT${NC}]"
        else
            echo -e "  Port $port: [${RED}FREI${NC}]"
        fi
    done

    echo ""
    if [ $EXIT_CODE -eq 0 ]; then
        echo -e "${GREEN}Overall Health: OK${NC}"
        return 0
    else
        echo -e "${RED}Overall Health: PROBLEMS DETECTED${NC}"
        return 1
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Config Backup
# ═══════════════════════════════════════════════════════════════════════════════

backup_config() {
    if [ ! -f "$INSTALL_DIR/config.json" ]; then
        log_error "Keine Konfiguration gefunden: $INSTALL_DIR/config.json"
        return 1
    fi

    local BACKUP_DIR="$INSTALL_DIR/backups"
    mkdir -p "$BACKUP_DIR"

    local BACKUP_FILE="$BACKUP_DIR/config-backup-$(date +%Y%m%d_%H%M%S).json"
    cp "$INSTALL_DIR/config.json" "$BACKUP_FILE"

    log "✓ Konfiguration gesichert: $BACKUP_FILE"
    return 0
}

# ═══════════════════════════════════════════════════════════════════════════════
# Uninstall
# ═══════════════════════════════════════════════════════════════════════════════

uninstall_modbridge() {
    check_root

    print_header

    # Check if installed
    if [ ! -d "$INSTALL_DIR" ]; then
        log_error "ModBridge ist nicht installiert"
        return 1
    fi

    # Ask for confirmation
    if command -v whiptail &>/dev/null; then
        whiptail --title "ModBridge Deinstallieren" \
                --yesno "Möchten Sie ModBridge wirklich vollständig deinstallieren?\n\n\
Dies wird:\n\
- Den Service stoppen und deaktivieren\n\
- Die Service-Datei löschen\n\
- Das Installationsverzeichnis $INSTALL_DIR löschen\n\
- ALLE Daten inkl. Konfiguration und Datenbank löschen\n\n\
${YELLOW}WARNUNG: Diese Aktion kann nicht rückgängig gemacht werden!${NC}" \
                18 80 \
                --yes-button "Ja, deinstallieren" --no-button "Abbrechen" \
                3>&1 1>&2 2>&3 || exit 0

        # Ask about config backup
        if whiptail --title "Konfiguration sichern?" \
                    --yesno "Möchten Sie die Konfiguration sichern vor dem Löschen?\n\n\
Die Konfiguration wird nach $INSTALL_DIR/backups/ kopiert." \
                    10 80 \
                    --yes-button "Ja, sichern" --no-button "Nein, löschen" \
                    3>&1 1>&2 2>&3; then
            backup_config
        fi
    else
        log_error "whiptail nicht verfügbar. Breche ab."
        log_info "Verwenden Sie --force um ohne Bestätigung zu deinstallieren"
        return 1
    fi

    log "Stoppe und deaktiviere Service..."
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    systemctl disable "$SERVICE_NAME" 2>/dev/null || true

    # Cleanup processes and ports
    if ! cleanup_modbridge; then
        log_warn "Cleanup nicht vollständig erfolgreich"
    fi

    # Remove service file
    log "Entferne Service-Datei..."
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload

    # Remove installation directory
    log "Entferne Installationsverzeichnis..."
    rm -rf "$INSTALL_DIR"

    log "${GREEN}✓ ModBridge erfolgreich deinstalliert${NC}"
    echo ""
    log "Backup-Verzeichnis (falls vorhanden) wurde gelöscht"
    log "Service-Datei wurde entfernt"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Help
# ═══════════════════════════════════════════════════════════════════════════════

show_help() {
    printf "${BOLD}ModBridge Installation Script${NC}\n"
    printf "${BOLD}============================${NC}\n\n"

    printf "${CYAN}Verwendung:${NC}\n"
    printf "  sudo bash modbridge.sh ${GREEN}install${NC}      - Installation mit interaktivem Menü\n"
    printf "  sudo bash modbridge.sh ${GREEN}update${NC}       - Update mit interaktivem Menü\n"
    printf "  sudo bash modbridge.sh ${GREEN}start${NC}        - Service starten\n"
    printf "  sudo bash modbridge.sh ${GREEN}stop${NC}         - Service stoppen\n"
    printf "  sudo bash modbridge.sh ${GREEN}restart${NC}      - Service neustarten\n"
    printf "  sudo bash modbridge.sh ${GREEN}status${NC}       - Service-Status anzeigen\n"
    printf "  sudo bash modbridge.sh ${GREEN}logs${NC} [N]     - Logs anzeigen (letzte N Einträge)\n"
    printf "  sudo bash modbridge.sh ${GREEN}logs${NC} -f      - Live-Logs anzeigen\n"
    printf "  sudo bash modbridge.sh ${GREEN}version${NC}      - Version anzeigen\n"
    printf "  sudo bash modbridge.sh ${GREEN}health${NC}       - Health-Check ausführen\n"
    printf "  sudo bash modbridge.sh ${GREEN}uninstall${NC}    - Vollständig deinstallieren\n\n"

    printf "${CYAN}Optionen:${NC}\n"
    printf "  ${YELLOW}--force${NC}                    - Installation erzwingen (überschreibt vorhandene)\n"
    printf "  ${YELLOW}--auto${NC}                     - Automatische Installation mit Defaults (WebUI + Latest)\n"
    printf "  ${YELLOW}--headless${NC}                 - Automatische Installation ohne WebUI\n"
    printf "  ${YELLOW}NO_UPDATE=1${NC}                - Script-Auto-Update überspringen\n\n"

    printf "${CYAN}Features:${NC}\n"
    printf "  ✓ Automatische Script-Aktualisierung vor jeder Ausführung\n"
    printf "  ✓ Interaktive grafische Menüs (whiptail)\n"
    printf "  ✓ Automatische Architektur-Erkennung (amd64, arm64, arm)\n"
    printf "  ✓ Wahl zwischen WebUI und Headless-Versionen\n"
    printf "  ✓ Download der passenden Binary von GitHub Releases\n"
    printf "  ✓ Automatische Service-Verwaltung via systemd\n"
    printf "  ✓ Robuster Cleanup: Beendet alle Prozesse, gibt Ports frei\n"
    printf "  ✓ Backup-Management mit automatischem Rollback\n\n"

    printf "${CYAN}Unterstützte Architekturen:${NC}\n"
    printf "  • ${YELLOW}amd64${NC}  - Intel/AMD 64-bit (Standard Server)\n"
    printf "  • ${YELLOW}arm64${NC}  - ARM 64-bit (Raspberry Pi 4/5, ARM Server)\n"
    printf "  • ${YELLOW}arm${NC}    - ARM 32-bit (Raspberry Pi Zero/1/2/3, 32-bit OS)\n\n"

    printf "${CYAN}Varianten:${NC}\n"
    printf "  • ${GREEN}Full${NC}     - Mit WebUI (~8.8 MB)\n"
    printf "  • ${GREEN}Headless${NC} - Ohne WebUI (~6.9 MB, 22% kleiner)\n\n"

    printf "${CYAN}Auto-Update:${NC}\n"
    printf "  Das Script prüft vor jedem Befehl (install, update, start, stop, restart),\n"
    printf "  ob eine neuere Version im Git Repository verfügbar ist.\n"
    printf "  Falls ja, wird es automatisch aktualisiert und neu gestartet.\n\n"

    printf "${CYAN}Beispiele:${NC}\n"
    printf "  # Interaktive Installation (mit Dialogen)\n"
    printf "  $ sudo bash modbridge.sh install\n\n"
    printf "  # Automatische Installation (neueste Version, WebUI, ohne Dialoge)\n"
    printf "  $ sudo bash modbridge.sh install --auto\n\n"
    printf "  # Automatische Installation ohne WebUI\n"
    printf "  $ sudo bash modbridge.sh install --headless\n\n"
    printf "  # Update auf neueste Version\n"
    printf "  $ sudo bash modbridge.sh update --auto\n\n"

    printf "${CYAN}Weitere Befehle:${NC}\n"
    printf "  sudo bash modbridge.sh ${GREEN}logs${NC} [100]       - Logs anzeigen (letzte 100 Einträge)\n"
    printf "  sudo bash modbridge.sh ${GREEN}logs${NC} -f         - Live-Logs anzeigen (follow)\n"
    printf "  sudo bash modbridge.sh ${GREEN}version${NC}         - Version anzeigen\n"
    printf "  sudo bash modbridge.sh ${GREEN}health${NC}          - Health-Check ausführen\n"
    printf "  sudo bash modbridge.sh ${GREEN}uninstall${NC}       - Vollständig deinstallieren\n\n"

    printf "${CYAN}Backup & Restore:${NC}\n"
    printf "  Vor jedem Update wird automatisch ein Backup der Konfiguration erstellt.\n"
    printf "  Backups werden in $INSTALL_DIR/backups/ gespeichert.\n\n"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════════════════════

# Self-update before doing anything else
self_update "$@"

# Check for flags
FORCE_INSTALL=0
SKIP_STATUS=0

for arg in "$@"; do
    if [ "$arg" = "--force" ]; then
        FORCE_INSTALL=1
    fi
    if [ "$arg" = "--skip-status" ]; then
        SKIP_STATUS=1
    fi
    if [ "$arg" = "--auto" ]; then
        AUTO_INSTALL=1
        DEFAULT_VARIANT="full"
    fi
    if [ "$arg" = "--headless" ]; then
        AUTO_INSTALL=1
        DEFAULT_VARIANT="headless"
    fi
done

case "${1:-}" in
    install)
        if [ $FORCE_INSTALL -eq 1 ]; then
            log "${YELLOW}Force-Installation: Überspringe Installationsprüfung${NC}"
            export MODBRIDGE_FORCE=1
        fi
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
        if [ $SKIP_STATUS -eq 0 ]; then
            show_installation_status
        fi
        status_service
        ;;
    logs)
        logs_service "$2" "$3"
        ;;
    version|--version|-v)
        version_service
        ;;
    health|--health)
        health_check
        ;;
    uninstall)
        uninstall_modbridge
        ;;
    *)
        show_installation_status
        show_help
        ;;
esac
