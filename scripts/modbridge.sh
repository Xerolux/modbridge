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
set -euo pipefail  # Verbesserte Fehlerbehandlung: Exit on error, undefined vars, pipe failures

# ═══════════════════════════════════════════════════════════════════════════════
# Configuration
# ═══════════════════════════════════════════════════════════════════════════════
INSTALL_DIR="/opt/modbridge"
SERVICE_NAME="modbridge.service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"
LOG_FILE="/var/log/modbridge-install.log"
REPO_URL="https://github.com/Xerolux/modbridge"
RELEASES_API="https://api.github.com/repos/Xerolux/modbridge/releases"
# Colors (optimiert für bessere Lesbarkeit)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ═══════════════════════════════════════════════════════════════════════════════
# Helper Functions (optimiert: Einheitliche Logging-Funktionen)
# ═══════════════════════════════════════════════════════════════════════════════
log() {
    local level="$1"
    shift
    local color=""
    case "$level" in
        INFO) color="${CYAN}" ;;
        WARN) color="${YELLOW}" ;;
        ERROR) color="${RED}" ;;
        *) color="${GREEN}" ;;
    esac
    echo -e "${color}[$(date +'%H:%M:%S') $level]${NC} $*" | tee -a "$LOG_FILE"
}
log_info() { log INFO "$@"; }
log_warn() { log WARN "$@"; }
log_error() { log ERROR "$@"; }

print_header() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC} ${BOLD} ModBridge Installer ${NC} ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC} Version 2.1 - Optimized ${NC} ${CYAN}║${NC}"
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
    log_info "Prüfe Abhängigkeiten..."
    local missing=()
    for dep in curl jq file lsof whiptail; do  # whiptail hinzugefügt als explizite Abhängigkeit
        command -v "$dep" &>/dev/null || missing+=("$dep")
    done
    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Fehlende Programme: ${missing[*]}"
        log_info "Installation: apt install ${missing[*]}"
        exit 1
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Self-Update Function (optimiert: Bessere Fehlerbehandlung, MD5 durch SHA256 ersetzt für Sicherheit)
# ═══════════════════════════════════════════════════════════════════════════════
self_update() {
    if [ "${NO_UPDATE:-0}" = "1" ]; then
        log_info "Script-Update übersprungen (NO_UPDATE=1)"
        return 0
    fi
    local cmd="${1:-}"
    if [[ ! "$cmd" =~ ^(install|update|start|stop|restart)$ ]]; then
        return 0
    fi
    log_info "Prüfe auf Script-Updates..."
    local SCRIPT_PATH="${BASH_SOURCE[0]}"
    local TEMP_SCRIPT="/tmp/modbridge.sh.new"
    local REMOTE_URL="https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh"
    if ! curl -fsSL "$REMOTE_URL" -o "$TEMP_SCRIPT" 2>/dev/null; then
        log_warn "Konnte Script-Update nicht prüfen (Download fehlgeschlagen)"
        rm -f "$TEMP_SCRIPT"
        return 0
    fi
    chmod +x "$TEMP_SCRIPT"
    local CURRENT_HASH=$(sha256sum "$SCRIPT_PATH" 2>/dev/null | awk '{print $1}')
    local NEW_HASH=$(sha256sum "$TEMP_SCRIPT" 2>/dev/null | awk '{print $1}')
    if [ "$CURRENT_HASH" = "$NEW_HASH" ]; then
        log_info "✓ Script ist aktuell"
        rm -f "$TEMP_SCRIPT"
        return 0
    fi
    log ""
    log_info "${YELLOW}NEUE SCRIPT-VERSION VERFÜGBAR${NC}"
    log "Das Script wird automatisch aktualisiert..."
    if mv "$TEMP_SCRIPT" "$SCRIPT_PATH"; then
        log_info "✓ Script erfolgreich aktualisiert"
        log "Starte neu mit dem aktualisierten Script..."
        exec bash "$SCRIPT_PATH" "$@"
    else
        log_error "Konnte Script nicht aktualisieren"
        rm -f "$TEMP_SCRIPT"
        return 1
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Installation Check (optimiert: Bessere Versionsvergleichslogik)
# ═══════════════════════════════════════════════════════════════════════════════
is_modbridge_installed() {
    [ -f "$INSTALL_DIR/modbridge" ]
}

normalize_version() {
    echo "${1#v}" | cut -d'-' -f1
}

compare_versions() {
    local v1=$(normalize_version "$1")
    local v2=$(normalize_version "$2")
    if [ "$v1" = "$v2" ]; then return 0; fi
    local IFS='.'
    local i ver1=($v1) ver2=($v2)
    local max_len=${#ver1[@]}
    [ ${#ver2[@]} -gt $max_len ] && max_len=${#ver2[@]}
    for ((i=0; i<$max_len; i++)); do
        ver1[i]=${ver1[i]:-0}
        ver2[i]=${ver2[i]:-0}
        if ((10#${ver1[i]} < 10#${ver2[i]})); then return 1; fi
        if ((10#${ver1[i]} > 10#${ver2[i]})); then return 2; fi
    done
    return 0
}

get_current_version() {
    is_modbridge_installed && "$INSTALL_DIR/modbridge" -version 2>/dev/null || echo "unbekannt"
}

get_latest_version() {
    curl -s "$RELEASES_API" | jq -r '.[0].tag_name' 2>/dev/null || echo ""
}

check_updates_available() {
    if ! is_modbridge_installed; then return 0; fi  # 0 = up to date
    local CURRENT=$(get_current_version)
    local LATEST=$(get_latest_version)
    [ -z "$LATEST" ] && return 0
    [ "$CURRENT" = "unbekannt" ] && return 2
    compare_versions "$CURRENT" "$LATEST"
}

show_installation_status() {
    if is_modbridge_installed; then
        local CURRENT=$(get_current_version)
        local LATEST=$(get_latest_version)
        local STATUS=$(check_updates_available)
        local SIZE=$(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || echo 0)
        local VARIANT=$([ "$SIZE" -lt 8000000 ] && echo "Headless (ohne WebUI)" || echo "Full (mit WebUI)")
        local SERVICE_STATUS=$([ "$(systemctl is-active "$SERVICE_NAME" 2>/dev/null)" = "active" ] && echo "${GREEN}Läuft${NC}" || echo "${RED}Gestoppt${NC}")
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo -e "${BOLD}${CYAN} ModBridge Installations-Status${NC}"
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo -e " Status: $([ $STATUS -eq 1 ] && echo "${YELLOW}Update verfügbar${NC}" || echo "${GREEN}Aktuell${NC}")"
        echo -e " Installierte Version: ${BOLD}$CURRENT${NC}"
        echo -e " Verfügbar: ${GREEN}$LATEST${NC}"
        echo -e " Variante: $VARIANT"
        echo -e " Service: $SERVICE_STATUS"
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
    else
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo -e "${BOLD}${CYAN} ModBridge ist nicht installiert${NC}"
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Architecture Detection (optimiert: Mehr Architekturen, bessere Namen)
# ═══════════════════════════════════════════════════════════════════════════════
detect_architecture() {
    local ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) echo "amd64|Intel/AMD 64-bit (Standard Server)" ;;
        aarch64) echo "arm64|ARM 64-bit (Raspberry Pi 4/5, ARM Server)" ;;
        armv7l|armv6l) echo "arm|ARM 32-bit (Raspberry Pi Zero/1/2/3)" ;;
        i386|i686) echo "386|Intel 32-bit (veraltet)" ;;
        *) log_error "Unbekannte Architektur: $ARCH"; exit 1 ;;
    esac
}

# ═══════════════════════════════════════════════════════════════════════════════
# Cleanup Functions (optimiert: Timeout reduziert, Logging verbessert)
# ═══════════════════════════════════════════════════════════════════════════════
kill_all_modbridge_processes() {
    log_info "Suche nach laufenden Modbridge-Prozessen..."
    local PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
    [ -z "$PIDS" ] && log_info "✓ Keine Prozesse gefunden" && return 0
    log_warn "Gefundene Prozesse: $PIDS"
    kill $PIDS 2>/dev/null
    local count=0
    while [ $count -lt 5 ]; do  # Reduzierter Timeout
        sleep 0.5
        PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
        [ -z "$PIDS" ] && log_info "✓ Prozesse beendet" && return 0
        count=$((count + 1))
    done
    log_warn "Erzwinge Beendigung (SIGKILL)..."
    kill -9 $PIDS 2>/dev/null
    sleep 1
    PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
    [ -n "$PIDS" ] && log_error "Konnte Prozesse nicht beenden: $PIDS" && return 1
    log_info "✓ Alle Prozesse beendet"
    return 0
}

check_and_wait_for_ports() {
    local PORTS="8080 5020 5021 5022 5023 5024 5025 5026 5027 5028 5029 5030"
    local MAX_WAIT=10  # Reduzierter Timeout
    local BLOCKED_PORTS=()
    for port in $PORTS; do
        lsof -i ":$port" -sTCP:LISTEN >/dev/null 2>&1 && BLOCKED_PORTS+=("$port")
    done
    [ ${#BLOCKED_PORTS[@]} -eq 0 ] && log_info "✓ Alle Ports frei" && return 0
    log_warn "Blockierte Ports: ${BLOCKED_PORTS[*]}"
    log_info "Warte auf Freigabe (max ${MAX_WAIT}s)..."
    local WAIT_COUNT=0
    while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
        sleep 1
        WAIT_COUNT=$((WAIT_COUNT + 1))
        BLOCKED_PORTS=()
        for port in $PORTS; do
            lsof -i ":$port" -sTCP:LISTEN >/dev/null 2>&1 && BLOCKED_PORTS+=("$port")
        done
        [ ${#BLOCKED_PORTS[@]} -eq 0 ] && log_info "✓ Ports frei" && return 0
    done
    log_error "Ports bleiben blockiert: ${BLOCKED_PORTS[*]}"
    return 1
}

cleanup_modbridge() {
    log_info "Cleanup: Beende Prozesse und Ports..."
    systemctl stop "$SERVICE_NAME" 2>/dev/null
    kill_all_modbridge_processes || return 1
    check_and_wait_for_ports || return 1
    log_info "✅ Cleanup erfolgreich"
    return 0
}

# ═══════════════════════════════════════════════════════════════════════════════
# Release Download (optimiert: Bessere Download-Progress, Validierung)
# ═══════════════════════════════════════════════════════════════════════════════
fetch_available_versions() {
    curl -s "$RELEASES_API" | jq -r '.[].tag_name' | head -10
}

download_modbridge_binary() {
    local VERSION=$1 VARIANT=$2 ARCH=$3
    local BINARY_NAME="modbridge-linux-${ARCH}${VARIANT:+"-headless"}"
    local DOWNLOAD_URL="${REPO_URL}/releases/download/${VERSION}/${BINARY_NAME}"
    local TEMP_FILE="/tmp/${BINARY_NAME}"
    log_info "Lade herunter: $BINARY_NAME von $DOWNLOAD_URL"
    curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL" --progress-bar || { log_error "Download fehlgeschlagen"; rm -f "$TEMP_FILE"; return 1; }
    file "$TEMP_FILE" | grep -q "ELF" || { log_error "Ungültige Binary"; rm -f "$TEMP_FILE"; return 1; }
    mv "$TEMP_FILE" "$INSTALL_DIR/modbridge"
    chmod +x "$INSTALL_DIR/modbridge"
    log_info "✓ Binary heruntergeladen"
    return 0
}

# ═══════════════════════════════════════════════════════════════════════════════
# Installation (optimiert: Weniger Redundanz, bessere Menüs)
# ═══════════════════════════════════════════════════════════════════════════════
install_modbridge() {
    check_root
    check_dependencies
    print_header
    IFS='|' read -r ARCH ARCH_NAME <<< "$(detect_architecture)"
    log_info "Erkannte Architektur: ${BOLD}$ARCH_NAME${NC}"
    if is_modbridge_installed && [ "${MODBRIDGE_FORCE:-0}" != "1" ]; then
        local STATUS=$(check_updates_available)
        show_installation_status
        if [ $STATUS -eq 0 ]; then
            whiptail --title "Bereits installiert" --yesno "ModBridge ist aktuell. Neu installieren?" 10 60 --yes-button "Ja" --no-button "Nein" || { log_info "Abgebrochen"; return 0; }
        else
            whiptail --title "Update verfügbar" --yesno "Update durchführen?" 10 60 --yes-button "Ja" --no-button "Nein" || { log_info "Abgebrochen"; return 0; }
            update_modbridge
            return
        fi
    fi
    whiptail --title "Willkommen" --yesno "Fortfahren mit Installation?\nArchitektur: $ARCH_NAME" 10 60 || exit 0
    local VARIANT=$(whiptail --title "Variante wählen" --radiolist "Wähle Variante:" 15 80 2 "full" "Mit WebUI" ON "headless" "Ohne WebUI" OFF 3>&1 1>&2 2>&3)
    cleanup_modbridge || { whiptail --title "Fehler" --msgbox "Cleanup fehlgeschlagen" 10 60; exit 1; }
    mkdir -p "$INSTALL_DIR"
    local VERSIONS=$(fetch_available_versions)
    local VERSION_LIST=()
    local i=1
    while read -r v; do
        VERSION_LIST+=("$v" "Version $i" $([ $i -eq 1 ] && echo ON || echo OFF))
        i=$((i+1))
    done <<< "$VERSIONS"
    local SELECTED_VERSION=$(whiptail --title "Version wählen" --radiolist "Wähle Version:" 20 80 10 "${VERSION_LIST[@]}" 3>&1 1>&2 2>&3)
    [ -z "$SELECTED_VERSION" ] && { log_info "Abgebrochen"; exit 0; }
    download_modbridge_binary "$SELECTED_VERSION" "$VARIANT" "$ARCH" || { whiptail --title "Fehler" --msgbox "Download fehlgeschlagen" 10 60; exit 1; }
    if [ "$VARIANT" = "headless" ]; then
        "$INSTALL_DIR/modbridge" -config > "$INSTALL_DIR/config.json" 2>/dev/null || log_warn "Standard-Config konnte nicht erstellt werden"
    fi
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
    systemctl daemon-reload
    systemctl enable --now "$SERVICE_NAME" || { log_error "Service-Start fehlgeschlagen"; systemctl status "$SERVICE_NAME"; exit 1; }
    local MSG="ModBridge $SELECTED_VERSION installiert!\nVariante: $VARIANT\nArchitektur: $ARCH_NAME\nService: Läuft"
    if [ "$VARIANT" = "headless" ]; then
        MSG+="\nConfig: $INSTALL_DIR/config.json (bearbeiten und restart)"
    else
        MSG+="\nWebUI: http://$(hostname -I | awk '{print $1}'):8080"
    fi
    whiptail --title "Erfolg" --msgbox "$MSG" 15 80
}

# ═══════════════════════════════════════════════════════════════════════════════
# Update (optimiert: Integrierte Backup-Logik, Rollback)
# ═══════════════════════════════════════════════════════════════════════════════
update_modbridge() {
    check_root
    check_dependencies
    [ ! -d "$INSTALL_DIR" ] && { log_error "Nicht installiert"; exit 1; }
    print_header
    show_installation_status
    IFS='|' read -r ARCH ARCH_NAME <<< "$(detect_architecture)"
    local CURRENT_VARIANT=$([ $(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null) -lt 8000000 ] && echo "headless" || echo "full")
    whiptail --title "Update" --yesno "Fortfahren?\nArchitektur: $ARCH_NAME\nAktuell: $CURRENT_VARIANT" 10 60 || exit 0
    local VARIANT=$(whiptail --title "Variante" --radiolist "Wähle:" 15 80 2 "full" "Mit WebUI" ON "headless" "Ohne WebUI" OFF 3>&1 1>&2 2>&3)
    cleanup_modbridge || { whiptail --title "Fehler" --msgbox "Cleanup fehlgeschlagen" 10 60; exit 1; }
    [ -f "$INSTALL_DIR/config.json" ] && backup_config
    cp "$INSTALL_DIR/modbridge" "$INSTALL_DIR/modbridge.backup.$(date +%Y%m%d_%H%M%S)"
    local VERSIONS=$(fetch_available_versions)
    local VERSION_LIST=()
    local i=1
    while read -r v; do
        VERSION_LIST+=("$v" "Version $i" $([ $i -eq 1 ] && echo ON || echo OFF))
        i=$((i+1))
    done <<< "$VERSIONS"
    local SELECTED_VERSION=$(whiptail --title "Version" --radiolist "Wähle:" 20 80 10 "${VERSION_LIST[@]}" 3>&1 1>&2 2>&3)
    [ -z "$SELECTED_VERSION" ] && { log_info "Abgebrochen"; exit 0; }
    download_modbridge_binary "$SELECTED_VERSION" "$VARIANT" "$ARCH" || { whiptail --title "Fehler" --msgbox "Download fehlgeschlagen" 10 60; exit 1; }
    systemctl restart "$SERVICE_NAME" || {
        log_error "Restart fehlgeschlagen - Rollback...";
        LATEST_BACKUP=$(ls -t "$INSTALL_DIR"/modbridge.backup.* | head -1);
        [ -n "$LATEST_BACKUP" ] && cp "$LATEST_BACKUP" "$INSTALL_DIR/modbridge" && chmod +x "$INSTALL_DIR/modbridge" && systemctl restart "$SERVICE_NAME";
        exit 1;
    }
    ls -t "$INSTALL_DIR"/modbridge.backup.* | tail -n +4 | xargs rm -f 2>/dev/null
    whiptail --title "Erfolg" --msgbox "Aktualisiert auf $SELECTED_VERSION!\nVariante: $VARIANT" 10 60
}

# ═══════════════════════════════════════════════════════════════════════════════
# Service Management (optimiert: Einheitliche Aufrufe)
# ═══════════════════════════════════════════════════════════════════════════════
start_service() { check_root; cleanup_modbridge; systemctl start "$SERVICE_NAME"; log_info "✓ Gestartet"; }
stop_service() { check_root; cleanup_modbridge; log_info "✓ Gestoppt"; }
restart_service() { check_root; cleanup_modbridge; systemctl restart "$SERVICE_NAME"; log_info "✓ Neu gestartet"; }
status_service() {
    check_root
    echo "Service Status:"
    systemctl status "$SERVICE_NAME" --no-pager
    echo "Prozesse:"
    pgrep -a modbridge || echo "Keine"
    echo "Ports:"
    for port in 8080 5020 5021 5022 5023; do
        lsof -i ":$port" >/dev/null 2>&1 && echo " $port: ${GREEN}Belegt${NC}" || echo " $port: ${RED}Frei${NC}"
    done
}

# ═══════════════════════════════════════════════════════════════════════════════
# Logs (optimiert: Bessere Handhabung)
# ═══════════════════════════════════════════════════════════════════════════════
logs_service() {
    check_root
    if [ "${1:-}" = "-f" ] || [ "${1:-}" = "--follow" ]; then
        journalctl -u "$SERVICE_NAME" -f
    else
        journalctl -u "$SERVICE_NAME" -n "${1:-50}" --no-pager
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Version & Health (optimiert: Tabellenartige Ausgabe)
# ═══════════════════════════════════════════════════════════════════════════════
version_service() {
    echo "ModBridge Version:"
    echo "Script: 2.1 - Optimized"
    if is_modbridge_installed; then
        local VERSION=$(get_current_version)
        local SIZE_MB=$(awk "BEGIN {printf \"%.2f\", $(stat -c%s "$INSTALL_DIR/modbridge")/1024/1024}")
        local VARIANT=$([ $(stat -c%s "$INSTALL_DIR/modbridge") -lt 8000000 ] && echo "Headless" || echo "Full")
        local SERVICE=$([ "$(systemctl is-active "$SERVICE_NAME")" = "active" ] && echo "${GREEN}Aktiv${NC}" || echo "${RED}Inaktiv${NC}")
        echo "Binary: $VERSION"
        echo "Größe: ${SIZE_MB} MB"
        echo "Variante: $VARIANT"
        echo "Service: $SERVICE"
    else
        echo "${YELLOW}Nicht installiert${NC}"
    fi
}

health_check() {
    local EXIT_CODE=0
    echo "Health Check:"
    [ -f "$INSTALL_DIR/modbridge" ] && echo "[${GREEN}✓${NC}] Binary" || { echo "[${RED}✗${NC}] Binary"; EXIT_CODE=1; }
    [ -f "$INSTALL_DIR/config.json" ] && echo "[${GREEN}✓${NC}] Config" || echo "[${YELLOW}⚠${NC}] Config"
    [ -f "$SERVICE_FILE" ] && echo "[${GREEN}✓${NC}] Service-File" || { echo "[${RED}✗${NC}] Service-File"; EXIT_CODE=1; }
    [ "$(systemctl is-active "$SERVICE_NAME")" = "active" ] && echo "[${GREEN}✓${NC}] Service läuft" || { echo "[${RED}✗${NC}] Service läuft nicht"; EXIT_CODE=1; }
    [ "$(systemctl is-enabled "$SERVICE_NAME")" = "enabled" ] && echo "[${GREEN}✓${NC}] Autostart" || echo "[${YELLOW}⚠${NC}] Autostart nicht aktiviert"
    echo "Ports:"
    for port in 8080 5020 5021 5022 5023; do
        lsof -i ":$port" >/dev/null 2>&1 && echo " $port [${GREEN}Belegt${NC}]" || echo " $port [${RED}Frei${NC}]"
    done
    [ $EXIT_CODE -eq 0 ] && echo "${GREEN}OK${NC}" || echo "${RED}Probleme${NC}"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Config Backup & Uninstall (optimiert: Bessere Pfade)
# ═══════════════════════════════════════════════════════════════════════════════
backup_config() {
    [ ! -f "$INSTALL_DIR/config.json" ] && { log_error "Keine Config"; return 1; }
    local BACKUP_DIR="$INSTALL_DIR/backups"
    mkdir -p "$BACKUP_DIR"
    cp "$INSTALL_DIR/config.json" "$BACKUP_DIR/config-$(date +%Y%m%d_%H%M%S).json"
    log_info "✓ Config gesichert"
}

uninstall_modbridge() {
    check_root
    [ ! -d "$INSTALL_DIR" ] && { log_error "Nicht installiert"; return 1; }
    print_header
    whiptail --title "Deinstallieren" --yesno "Wirklich deinstallieren? Alle Daten werden gelöscht!" 10 60 || exit 0
    whiptail --title "Backup?" --yesno "Config sichern?" 10 60 && backup_config
    systemctl disable --now "$SERVICE_NAME" 2>/dev/null
    cleanup_modbridge
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    rm -rf "$INSTALL_DIR"
    log_info "✓ Deinstalliert"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Help (optimiert: Aktualisierte Texte, klarer)
# ═══════════════════════════════════════════════════════════════════════════════
show_help() {
    echo "${BOLD}ModBridge Script${NC}"
    echo "Verwendung: sudo bash modbridge.sh [COMMAND]"
    echo "Commands:"
    echo "  install   - Installieren"
    echo "  update    - Aktualisieren"
    echo "  start     - Starten"
    echo "  stop      - Stoppen"
    echo "  restart   - Neustarten"
    echo "  status    - Status"
    echo "  logs [N]  - Logs (letzte N, default 50)"
    echo "  logs -f   - Live Logs"
    echo "  version   - Version"
    echo "  health    - Health Check"
    echo "  uninstall - Deinstallieren"
    echo "Optionen: --force (für install), NO_UPDATE=1 (kein Self-Update)"
}

# ═══════════════════════════════════════════════════════════════════════════════
# Main (optimiert: Flag-Handhabung, Self-Update zuerst)
# ═══════════════════════════════════════════════════════════════════════════════
self_update "$@"
MODBRIDGE_FORCE=0
for arg in "$@"; do [ "$arg" = "--force" ] && MODBRIDGE_FORCE=1; done
case "${1:-}" in
    install) install_modbridge ;;
    update) update_modbridge ;;
    start) start_service ;;
    stop) stop_service ;;
    restart) restart_service ;;
    status) show_installation_status; status_service ;;
    logs) logs_service "${2:-}" ;;
    version) version_service ;;
    health) health_check ;;
    uninstall) uninstall_modbridge ;;
    *) show_installation_status; show_help ;;
esac
