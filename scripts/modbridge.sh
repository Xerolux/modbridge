#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════════
# ModBridge Installation Script - Version 2.2 (Fixed version issues, improved handling)
# ═══════════════════════════════════════════════════════════════════════════════
set -euo pipefail

INSTALL_DIR="/opt/modbridge"
SERVICE_NAME="modbridge.service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"
LOG_FILE="/var/log/modbridge-install.log"
REPO_URL="https://github.com/Xerolux/modbridge"
RELEASES_API="https://api.github.com/repos/Xerolux/modbridge/releases"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log() {
    local level="$1"; shift
    local color="${CYAN}"
    [ "$level" = "WARN" ] && color="${YELLOW}"
    [ "$level" = "ERROR" ] && color="${RED}"
    echo -e "${color}[$(date +'%H:%M:%S') $level]${NC} $*" | tee -a "$LOG_FILE"
}
log_info() { log INFO "$@"; }
log_warn() { log WARN "$@"; }
log_error() { log ERROR "$@"; }

print_header() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC} ${BOLD} ModBridge Installer ${NC} ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC} Version 2.2 - Fixed & Optimized ${NC} ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

check_root() { [ "$EUID" -ne 0 ] && { log_error "Run as root"; exit 1; }; }
check_dependencies() {
    log_info "Checking dependencies..."
    local missing=()
    for dep in curl jq file lsof whiptail; do command -v "$dep" &>/dev/null || missing+=("$dep"); done
    [ ${#missing[@]} -gt 0 ] && { log_error "Missing: ${missing[*]}"; log_info "Install: apt install ${missing[*]}"; exit 1; }
}

self_update() {
    [ "${NO_UPDATE:-0}" = "1" ] && { log_info "Update skipped"; return 0; }
    local cmd="${1:-}"
    [[ ! "$cmd" =~ ^(install|update|start|stop|restart)$ ]] && return 0
    log_info "Checking script updates..."
    local SCRIPT_PATH="${BASH_SOURCE[0]}"
    local TEMP_SCRIPT="/tmp/modbridge.sh.new"
    local REMOTE_URL="https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh"
    curl -fsSL "$REMOTE_URL" -o "$TEMP_SCRIPT" 2>/dev/null || { log_warn "Update check failed"; rm -f "$TEMP_SCRIPT"; return 0; }
    chmod +x "$TEMP_SCRIPT"
    local CURRENT_HASH=$(sha256sum "$SCRIPT_PATH" 2>/dev/null | awk '{print $1}')
    local NEW_HASH=$(sha256sum "$TEMP_SCRIPT" 2>/dev/null | awk '{print $1}')
    [ "$CURRENT_HASH" = "$NEW_HASH" ] && { log_info "✓ Script current"; rm -f "$TEMP_SCRIPT"; return 0; }
    log_info "${YELLOW}NEW SCRIPT VERSION AVAILABLE${NC}"
    log_info "Updating script..."
    mv "$TEMP_SCRIPT" "$SCRIPT_PATH" || { log_error "Update failed"; rm -f "$TEMP_SCRIPT"; return 1; }
    log_info "✓ Updated"
    log_info "Restarting with updated script..."
    exec bash "$SCRIPT_PATH" "$@"
}

is_modbridge_installed() { [ -f "$INSTALL_DIR/modbridge" ]; }

normalize_version() { echo "${1#v}" | cut -d'-' -f1 | sed 's/[^0-9.]//g'; }  # Clean non-numeric

compare_versions() {
    local v1=$(normalize_version "$1")
    local v2=$(normalize_version "$2")
    [ -z "$v1" ] || [ -z "$v2" ] && return 1  # If invalid, assume update needed
    [ "$v1" = "$v2" ] && return 0
    local IFS='.'; local ver1=($v1) ver2=($v2)
    local max_len=$(( ${#ver1[@]} > ${#ver2[@]} ? ${#ver1[@]} : ${#ver2[@]} ))
    for ((i=0; i<$max_len; i++)); do
        local a=${ver1[i]:-0}; local b=${ver2[i]:-0}
        [ -z "$a" ] && a=0; [ -z "$b" ] && b=0
        (( a < b )) && return 1
        (( a > b )) && return 2
    done
    return 0
}

get_current_version() {
    if is_modbridge_installed; then
        local ver=$("$INSTALL_DIR/modbridge" -version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unbekannt")
        [ -z "$ver" ] && echo "unbekannt" || echo "$ver"
    else
        echo "unbekannt"
    fi
}

get_latest_version() { curl -s "$RELEASES_API" | jq -r '.[0].tag_name' 2>/dev/null | normalize_version || echo ""; }

check_updates_available() {
    ! is_modbridge_installed && return 0
    local CURRENT=$(get_current_version)
    local LATEST=$(get_latest_version)
    [ -z "$LATEST" ] || [ "$CURRENT" = "unbekannt" ] && return 2  # Unknown or can't fetch
    compare_versions "$CURRENT" "$LATEST"
}

show_installation_status() {
    local CURRENT=$(get_current_version)
    local LATEST=$(get_latest_version)
    local STATUS=$(check_updates_available)
    local SIZE=$(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || echo 0)
    local VARIANT=$([ "$SIZE" -lt 8000000 ] && echo "Headless" || echo "Full")
    local SERVICE_STATUS=$([ "$(systemctl is-active "$SERVICE_NAME" 2>/dev/null)" = "active" ] && echo "${GREEN}Running${NC}" || echo "${RED}Stopped${NC}")
    if is_modbridge_installed; then
        echo -e "${CYAN}══ ModBridge Status ══${NC}"
        echo -e "Status: $([ $STATUS -eq 1 ] && echo "${YELLOW}Update available${NC}" || echo "${GREEN}Current${NC}")"
        echo -e "Installed: $CURRENT"
        echo -e "Available: $LATEST"
        echo -e "Variant: $VARIANT"
        echo -e "Service: $SERVICE_STATUS"
        echo -e "${CYAN}══════════════════════${NC}"
    else
        echo -e "${CYAN}══ ModBridge not installed ══${NC}"
    fi
}

detect_architecture() {
    local ARCH=$(uname -m)
    case "$ARCH" in x86_64) echo "amd64|Intel/AMD 64-bit";; aarch64) echo "arm64|ARM 64-bit";; armv7l|armv6l) echo "arm|ARM 32-bit";; i386|i686) echo "386|Intel 32-bit";; *) log_error "Unknown arch: $ARCH"; exit 1;; esac
}

kill_all_modbridge_processes() {
    log_info "Killing processes..."
    local PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
    [ -z "$PIDS" ] && { log_info "✓ None found"; return 0; }
    kill $PIDS 2>/dev/null
    local count=0
    while [ $count -lt 5 ]; do sleep 0.5; PIDS=$(pgrep -x "modbridge" 2>/dev/null || true); [ -z "$PIDS" ] && { log_info "✓ Killed"; return 0; }; count=$((count+1)); done
    kill -9 $PIDS 2>/dev/null; sleep 1
    PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
    [ -n "$PIDS" ] && { log_error "Failed to kill: $PIDS"; return 1; }
    log_info "✓ All killed"
    return 0
}

check_and_wait_for_ports() {
    local PORTS="8080 5020 5021 5022 5023 5024 5025 5026 5027 5028 5029 5030"
    local MAX_WAIT=10
    local BLOCKED=()
    for port in $PORTS; do lsof -i ":$port" -sTCP:LISTEN >/dev/null 2>&1 && BLOCKED+=("$port"); done
    [ ${#BLOCKED[@]} -eq 0 ] && { log_info "✓ Ports free"; return 0; }
    log_warn "Blocked: ${BLOCKED[*]}"
    local WAIT=0
    while [ $WAIT -lt $MAX_WAIT ]; do
        sleep 1; WAIT=$((WAIT+1)); BLOCKED=()
        for port in $PORTS; do lsof -i ":$port" -sTCP:LISTEN >/dev/null 2>&1 && BLOCKED+=("$port"); done
        [ ${#BLOCKED[@]} -eq 0 ] && { log_info "✓ Ports free"; return 0; }
    done
    log_error "Ports blocked: ${BLOCKED[*]}"
    return 1
}

cleanup_modbridge() {
    log_info "Cleanup..."
    systemctl stop "$SERVICE_NAME" 2>/dev/null
    kill_all_modbridge_processes || return 1
    check_and_wait_for_ports || return 1
    log_info "✅ Done"
    return 0
}

fetch_available_versions() { curl -s "$RELEASES_API" | jq -r '.[].tag_name' | head -10; }

download_modbridge_binary() {
    local VERSION=$1 VARIANT=$2 ARCH=$3
    local SUFFIX=$([ "$VARIANT" = "headless" ] && echo "-headless" || echo "")
    local BINARY_NAME="modbridge-linux-${ARCH}${SUFFIX}"
    local DOWNLOAD_URL="${REPO_URL}/releases/download/${VERSION}/${BINARY_NAME}"
    local TEMP_FILE="/tmp/${BINARY_NAME}"
    log_info "Downloading: $BINARY_NAME from $DOWNLOAD_URL"
    curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL" --progress-bar || { log_error "Download failed"; rm -f "$TEMP_FILE"; return 1; }
    file "$TEMP_FILE" | grep -q "ELF" || { log_error "Invalid binary (not ELF)"; rm -f "$TEMP_FILE"; return 1; }
    mv "$TEMP_FILE" "$INSTALL_DIR/modbridge"
    chmod +x "$INSTALL_DIR/modbridge"
    log_info "✓ Downloaded"
    return 0
}

install_modbridge() {
    check_root; check_dependencies; print_header
    IFS='|' read -r ARCH ARCH_NAME <<< "$(detect_architecture)"
    log_info "Arch: $ARCH_NAME"
    if is_modbridge_installed && [ "${MODBRIDGE_FORCE:-0}" != "1" ]; then
        local STATUS=$(check_updates_available)
        show_installation_status
        if [ $STATUS -eq 0 ]; then whiptail --title "Installed" --yesno "Reinstall?" 10 60 || { log_info "Aborted"; return 0; }; else whiptail --title "Update" --yesno "Update?" 10 60 || { log_info "Aborted"; return 0; }; update_modbridge; return; fi
    fi
    whiptail --title "Welcome" --yesno "Proceed? Arch: $ARCH_NAME" 10 60 || exit 0
    local VARIANT=$(whiptail --title "Variant" --radiolist "Choose:" 15 80 2 "full" "With WebUI" ON "headless" "Without WebUI" OFF 3>&1 1>&2 2>&3)
    cleanup_modbridge || { whiptail --title "Error" --msgbox "Cleanup failed" 10 60; exit 1; }
    mkdir -p "$INSTALL_DIR"
    local VERSIONS=$(fetch_available_versions)
    local VERSION_LIST=(); local i=1
    while read -r v; do VERSION_LIST+=("$v" "Version $i" $([ $i -eq 1 ] && echo ON || echo OFF)); i=$((i+1)); done <<< "$VERSIONS"
    local SELECTED=$(whiptail --title "Version" --radiolist "Choose:" 20 80 10 "${VERSION_LIST[@]}" 3>&1 1>&2 2>&3)
    [ -z "$SELECTED" ] && { log_info "Aborted"; exit 0; }
    download_modbridge_binary "$SELECTED" "$VARIANT" "$ARCH" || { whiptail --title "Error" --msgbox "Download failed" 10 60; exit 1; }
    if [ "$VARIANT" = "headless" ]; then "$INSTALL_DIR/modbridge" -config > "$INSTALL_DIR/config.json" 2>/dev/null || log_warn "Default config failed"; fi
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
    systemctl enable --now "$SERVICE_NAME" || { log_error "Start failed"; systemctl status "$SERVICE_NAME"; exit 1; }
    local MSG="Installed $SELECTED! Variant: $VARIANT Arch: $ARCH_NAME Service: Running"
    [ "$VARIANT" = "headless" ] && MSG+="\nConfig: $INSTALL_DIR/config.json" || MSG+="\nWebUI: http://$(hostname -I | awk '{print $1}'):8080"
    whiptail --title "Success" --msgbox "$MSG" 15 80
}

update_modbridge() {
    check_root; check_dependencies; [ ! -d "$INSTALL_DIR" ] && { log_error "Not installed"; exit 1; }
    print_header; show_installation_status
    IFS='|' read -r ARCH ARCH_NAME <<< "$(detect_architecture)"
    local CURRENT_VARIANT=$([ $(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || echo 0) -lt 8000000 ] && echo "headless" || echo "full")
    whiptail --title "Update" --yesno "Proceed? Arch: $ARCH_NAME Current: $CURRENT_VARIANT" 10 60 || exit 0
    local VARIANT=$(whiptail --title "Variant" --radiolist "Choose:" 15 80 2 "full" "With WebUI" ON "headless" "Without WebUI" OFF 3>&1 1>&2 2>&3)
    cleanup_modbridge || { whiptail --title "Error" --msgbox "Cleanup failed" 10 60; exit 1; }
    [ -f "$INSTALL_DIR/config.json" ] && backup_config
    cp "$INSTALL_DIR/modbridge" "$INSTALL_DIR/modbridge.backup.$(date +%Y%m%d_%H%M%S)"
    local VERSIONS=$(fetch_available_versions)
    local VERSION_LIST=(); local i=1
    while read -r v; do VERSION_LIST+=("$v" "Version $i" $([ $i -eq 1 ] && echo ON || echo OFF)); i=$((i+1)); done <<< "$VERSIONS"
    local SELECTED=$(whiptail --title "Version" --radiolist "Choose:" 20 80 10 "${VERSION_LIST[@]}" 3>&1 1>&2 2>&3)
    [ -z "$SELECTED" ] && { log_info "Aborted"; exit 0; }
    download_modbridge_binary "$SELECTED" "$VARIANT" "$ARCH" || { whiptail --title "Error" --msgbox "Download failed" 10 60; exit 1; }
    systemctl restart "$SERVICE_NAME" || {
        log_error "Restart failed - Rolling back...";
        local LATEST_BACKUP=$(ls -t "$INSTALL_DIR"/modbridge.backup.* | head -1 2>/dev/null);
        [ -n "$LATEST_BACKUP" ] && cp "$LATEST_BACKUP" "$INSTALL_DIR/modbridge" && chmod +x "$INSTALL_DIR/modbridge" && systemctl restart "$SERVICE_NAME";
        exit 1;
    }
    ls -t "$INSTALL_DIR"/modbridge.backup.* | tail -n +4 | xargs rm -f 2>/dev/null
    whiptail --title "Success" --msgbox "Updated to $SELECTED! Variant: $VARIANT" 10 60
}

start_service() { check_root; cleanup_modbridge; systemctl start "$SERVICE_NAME"; log_info "✓ Started"; }
stop_service() { check_root; cleanup_modbridge; log_info "✓ Stopped"; }
restart_service() { check_root; cleanup_modbridge; systemctl restart "$SERVICE_NAME"; log_info "✓ Restarted"; }
status_service() {
    check_root
    systemctl status "$SERVICE_NAME" --no-pager
    pgrep -a modbridge || echo "No processes"
    for port in 8080 5020 5021 5022 5023; do lsof -i ":$port" >/dev/null 2>&1 && echo "$port: ${GREEN}Occupied${NC}" || echo "$port: ${RED}Free${NC}"; done
}

logs_service() {
    check_root
    if [ "${1:-}" = "-f" ] || [ "${1:-}" = "--follow" ]; then journalctl -u "$SERVICE_NAME" -f; else journalctl -u "$SERVICE_NAME" -n "${1:-50}" --no-pager; fi
}

version_service() {
    echo "Script: 2.2 - Fixed"
    if is_modbridge_installed; then
        local VERSION=$(get_current_version)
        local SIZE_MB=$(awk "BEGIN {printf \"%.2f\", $(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || echo 0)/1024/1024}")
        local VARIANT=$([ $(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || echo 0) -lt 8000000 ] && echo "Headless" || echo "Full")
        local SERVICE=$([ "$(systemctl is-active "$SERVICE_NAME")" = "active" ] && echo "${GREEN}Active${NC}" || echo "${RED}Inactive${NC}")
        echo "Binary: $VERSION ($SIZE_MB MB) $VARIANT"
        echo "Service: $SERVICE"
    else
        echo "${YELLOW}Not installed${NC}"
    fi
}

health_check() {
    local RC=0
    [ -f "$INSTALL_DIR/modbridge" ] && echo "[${GREEN}✓${NC}] Binary" || { echo "[${RED}✗${NC}] Binary"; RC=1; }
    [ -f "$INSTALL_DIR/config.json" ] && echo "[${GREEN}✓${NC}] Config" || echo "[${YELLOW}⚠${NC}] Config"
    [ -f "$SERVICE_FILE" ] && echo "[${GREEN}✓${NC}] Service file" || { echo "[${RED}✗${NC}] Service file"; RC=1; }
    [ "$(systemctl is-active "$SERVICE_NAME")" = "active" ] && echo "[${GREEN}✓${NC}] Running" || { echo "[${RED}✗${NC}] Not running"; RC=1; }
    [ "$(systemctl is-enabled "$SERVICE_NAME")" = "enabled" ] && echo "[${GREEN}✓${NC}] Enabled" || echo "[${YELLOW}⚠${NC}] Not enabled"
    for port in 8080 5020 5021 5022 5023; do lsof -i ":$port" >/dev/null 2>&1 && echo "$port [${GREEN}Occupied${NC}]" || echo "$port [${RED}Free${NC}]"; done
    [ $RC -eq 0 ] && echo "${GREEN}OK${NC}" || echo "${RED}Issues${NC}"
}

backup_config() {
    [ ! -f "$INSTALL_DIR/config.json" ] && { log_error "No config"; return 1; }
    mkdir -p "$INSTALL_DIR/backups"
    cp "$INSTALL_DIR/config.json" "$INSTALL_DIR/backups/config-$(date +%Y%m%d_%H%M%S).json"
    log_info "✓ Backed up"
}

uninstall_modbridge() {
    check_root; [ ! -d "$INSTALL_DIR" ] && { log_error "Not installed"; return 1; }
    print_header
    whiptail --title "Uninstall" --yesno "Confirm? All data lost!" 10 60 || exit 0
    whiptail --title "Backup?" --yesno "Backup config?" 10 60 && backup_config
    systemctl disable --now "$SERVICE_NAME" 2>/dev/null
    cleanup_modbridge
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    rm -rf "$INSTALL_DIR"
    log_info "✓ Uninstalled"
}

show_help() {
    echo "${BOLD}ModBridge Script${NC}"
    echo "Usage: sudo bash modbridge.sh [COMMAND]"
    echo "Commands: install update start stop restart status logs [N|-f] version health uninstall"
    echo "Options: --force (install) NO_UPDATE=1 (no self-update)"
}

self_update "$@"
MODBRIDGE_FORCE=0; for arg in "$@"; do [ "$arg" = "--force" ] && MODBRIDGE_FORCE=1; done
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
