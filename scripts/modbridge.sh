#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════════
# ModBridge Manager Script v3.0
#
# Features:
#   - Self-update: checks GitHub for newer version before every run
#   - Self-install: copies itself to /usr/local/bin/modbridge
#   - Whiptail TUI menu when called without arguments
#   - Full CLI mode with all commands
#   - systemd service with proxy autostart
#   - Interactive install / update / uninstall
#   - Config backup & restore, health check, log viewer
#
# Usage:
#   bash modbridge.sh              → interactive TUI menu
#   bash modbridge.sh install      → install ModBridge
#   bash modbridge.sh update       → update ModBridge
#   bash modbridge.sh start        → start service
#   bash modbridge.sh stop         → stop service
#   bash modbridge.sh restart      → restart service
#   bash modbridge.sh status       → show status
#   bash modbridge.sh logs [-f]    → show logs
#   bash modbridge.sh health       → health check
#   bash modbridge.sh config       → edit config
#   bash modbridge.sh backup       → backup config + db
#   bash modbridge.sh version      → show version
#   bash modbridge.sh uninstall    → full uninstall
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail

# ── Configuration ─────────────────────────────────────────────────────────────
INSTALL_DIR="/opt/modbridge"
SERVICE_NAME="modbridge.service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"
SCRIPT_TARGET="/usr/local/bin/modbridge"
LOG_FILE="/var/log/modbridge-install.log"
REPO_URL="https://github.com/Xerolux/modbridge"
RELEASES_API="https://api.github.com/repos/Xerolux/modbridge/releases"
SCRIPT_RAW_URL="https://raw.githubusercontent.com/Xerolux/modbridge/main/scripts/modbridge.sh"
SCRIPT_VERSION="3.0"

AUTO_INSTALL=0
DEFAULT_VARIANT="full"

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ── Logging helpers ───────────────────────────────────────────────────────────

log()        { echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE" 2>/dev/null || echo -e "$1"; }
log_error()  { echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE" 2>/dev/null || echo -e "${RED}[ERROR]${NC} $1" >&2; }
log_warn()   { echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$LOG_FILE" 2>/dev/null || echo -e "$1"; }
log_info()   { echo -e "${CYAN}[INFO]${NC} $1" | tee -a "$LOG_FILE" 2>/dev/null || echo -e "$1"; }

print_header() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC} ${BOLD}              ModBridge Manager v${SCRIPT_VERSION}${NC}                    ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}           Industrial Modbus TCP Proxy Manager                 ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# ── Prerequisite checks ──────────────────────────────────────────────────────

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "Bitte als root ausfuehren (sudo bash modbridge.sh ...)"
        exit 1
    fi
}

check_dependencies() {
    local missing=()
    command -v curl &>/dev/null || missing+=("curl")
    command -v jq   &>/dev/null || missing+=("jq")
    command -v file &>/dev/null || missing+=("file")

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Fehlende Programme: ${missing[*]}"
        log_info "Installation: apt install ${missing[*]}"
        exit 1
    fi
}

ensure_whiptail() {
    if ! command -v whiptail &>/dev/null; then
        log_info "Installiere whiptail..."
        apt-get update -qq && apt-get install -y -qq whiptail 2>/dev/null || true
        if ! command -v whiptail &>/dev/null; then
            log_error "whiptail konnte nicht installiert werden"
            exit 1
        fi
    fi
}

# ── Self-Update ───────────────────────────────────────────────────────────────

self_update() {
    [ "${NO_UPDATE:-}" = "1" ] && return 0

    local SCRIPT_PATH="${BASH_SOURCE[0]}"
    local TEMP_SCRIPT="${SCRIPT_PATH}.new"

    log_info "Pruefe auf Script-Updates..."

    if ! curl -fsSL --connect-timeout 5 --max-time 15 "$SCRIPT_RAW_URL" -o "$TEMP_SCRIPT" 2>/dev/null; then
        log_warn "Script-Update-Pruefung uebersprungen (Download fehlgeschlagen)"
        rm -f "$TEMP_SCRIPT"
        return 0
    fi

    local CUR_MD5 NEW_MD5
    CUR_MD5=$(md5sum "$SCRIPT_PATH" 2>/dev/null | awk '{print $1}')
    NEW_MD5=$(md5sum "$TEMP_SCRIPT" 2>/dev/null | awk '{print $1}')

    if [ "$CUR_MD5" = "$NEW_MD5" ]; then
        log_info "Script ist aktuell (v${SCRIPT_VERSION})"
        rm -f "$TEMP_SCRIPT"
        return 0
    fi

    chmod +x "$TEMP_SCRIPT"
    log ""
    log "${YELLOW}Neue Script-Version verfuegbar - aktualisiere...${NC}"

    if mv "$TEMP_SCRIPT" "$SCRIPT_PATH"; then
        log "${GREEN}Script aktualisiert - starte neu...${NC}"
        exec bash "$SCRIPT_PATH" "$@"
    else
        log_warn "Konnte Script nicht aktualisieren"
        rm -f "$TEMP_SCRIPT"
    fi
}

# ── Self-Install (script → /usr/local/bin/modbridge) ─────────────────────────

self_install_script() {
    local SCRIPT_PATH="${BASH_SOURCE[0]}"

    if [ "$SCRIPT_PATH" = "$SCRIPT_TARGET" ]; then
        return 0
    fi

    if [ ! -f "$SCRIPT_TARGET" ] || [ "$(md5sum "$SCRIPT_PATH" | awk '{print $1}')" != "$(md5sum "$SCRIPT_TARGET" 2>/dev/null | awk '{print $1}')" ]; then
        cp "$SCRIPT_PATH" "$SCRIPT_TARGET"
        chmod +x "$SCRIPT_TARGET"
        log_info "Script installiert nach $SCRIPT_TARGET"
    fi
}

# ── Installation state ────────────────────────────────────────────────────────

is_modbridge_installed() {
    [ -f "$INSTALL_DIR/modbridge" ]
}

normalize_version() {
    echo "${1#v}" | cut -d'-' -f1
}

compare_versions() {
    local v1=$(normalize_version "$1")
    local v2=$(normalize_version "$2")
    [ "$v1" = "$v2" ] && return 0

    local IFS='.' i
    local ver1=($v1) ver2=($v2)

    for ((i=${#ver1[@]}; i<${#ver2[@]}; i++)); do ver1[i]=0; done
    for ((i=0; i<${#ver1[@]}; i++)); do
        [[ -z ${ver2[i]} ]] && ver2[i]=0
        if ((10#${ver1[i]:-0} < 10#${ver2[i]:-0})); then return 1
        elif ((10#${ver1[i]:-0} > 10#${ver2[i]:-0})); then return 2; fi
    done
    return 0
}

get_current_version() {
    is_modbridge_installed || { echo ""; return; }
    "$INSTALL_DIR/modbridge" -version 2>/dev/null || echo "unbekannt"
}

get_latest_version() {
    local V
    V=$(fetch_available_versions 2>/dev/null) || return 1
    echo "$V" | head -n 1
}

check_updates_available() {
    is_modbridge_installed || { echo "0"; return; }
    local CUR LATEST
    CUR=$(get_current_version)
    LATEST=$(get_latest_version 2>/dev/null || echo "")

    [ -z "$CUR" ] || [ "$CUR" = "unbekannt" ] && { echo "2"; return; }
    [ -z "$LATEST" ] && { echo "0"; return; }

    compare_versions "$CUR" "$LATEST"
    local rc=$?
    [ $rc -eq 1 ] && echo "1" || echo "0"
}

show_installation_status() {
    if is_modbridge_installed; then
        local CUR LATEST STATUS
        CUR=$(get_current_version)
        LATEST=$(get_latest_version 2>/dev/null || echo "unbekannt")
        STATUS=$(check_updates_available)

        echo ""
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo -e "${BOLD}${CYAN}  ModBridge Installations-Status${NC}"
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo ""

        if [ "$STATUS" = "0" ]; then
            echo -e "  Status:       ${GREEN}Aktuell${NC}"
            echo -e "  Installiert:  ${BOLD}$CUR${NC}"
            echo -e "  Verfuegbar:   ${GREEN}$LATEST${NC}"
        elif [ "$STATUS" = "1" ]; then
            echo -e "  Status:       ${YELLOW}Update verfuegbar${NC}"
            echo -e "  Installiert:  ${BOLD}$CUR${NC}"
            echo -e "  Verfuegbar:   ${GREEN}$LATEST${NC}"
        else
            echo -e "  Status:       ${YELLOW}Installiert, Version unklar${NC}"
            echo -e "  Installiert:  ${YELLOW}$CUR${NC}"
            echo -e "  Verfuegbar:   ${GREEN}$LATEST${NC}"
        fi

        local SIZE
        SIZE=$(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || stat -f%z "$INSTALL_DIR/modbridge" 2>/dev/null || echo 0)
        if [ "$SIZE" -lt 8000000 ]; then
            echo -e "  Variante:     Headless (ohne WebUI)"
        else
            echo -e "  Variante:     Full (mit WebUI)"
        fi

        if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
            echo -e "  Service:      ${GREEN}Laeuft${NC}"
        else
            echo -e "  Service:      ${RED}Gestoppt${NC}"
        fi

        if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
            echo -e "  Autostart:    ${GREEN}Aktiviert${NC}"
        else
            echo -e "  Autostart:    ${RED}Deaktiviert${NC}"
        fi

        echo ""
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo ""
    else
        echo ""
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo -e "${BOLD}${CYAN}  ModBridge ist nicht installiert${NC}"
        echo -e "${CYAN}══════════════════════════════════════════════════════════════════${NC}"
        echo ""
    fi
}

# ── Architecture detection ───────────────────────────────────────────────────

detect_architecture() {
    local ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  echo "amd64|Intel/AMD 64-bit" ;;
        aarch64) echo "arm64|ARM 64-bit (Pi 4/5)" ;;
        armv7l|armv6l) echo "arm|ARM 32-bit" ;;
        i386|i686) echo "386|Intel 32-bit" ;;
        *) log_error "Unbekannte Architektur: $ARCH"; exit 1 ;;
    esac
}

# ── Process / port cleanup ───────────────────────────────────────────────────

kill_all_modbridge_processes() {
    local PIDS
    PIDS=$(pgrep -x "modbridge" 2>/dev/null | grep -v "^$$\$" || true)
    [ -z "$PIDS" ] && [ -x "$INSTALL_DIR/modbridge" ] && PIDS=$(pidof modbridge 2>/dev/null || true)

    if [ -z "$PIDS" ]; then
        log "Keine laufenden Prozesse."
        return 0
    fi

    log "Beende Prozesse: $PIDS"
    kill $PIDS 2>/dev/null || true

    local count=0
    while [ $count -lt 10 ]; do
        sleep 0.5
        PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
        [ -z "$PIDS" ] && { log "Alle Prozesse beendet."; return 0; }
        count=$((count + 1))
    done

    PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
    if [ -n "$PIDS" ]; then
        kill -9 $PIDS 2>/dev/null || true
        sleep 1
        PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
        [ -n "$PIDS" ] && { log_error "Prozesse konnten nicht beendet werden: $PIDS"; return 1; }
    fi
    log "Alle Prozesse beendet."
    return 0
}

check_and_wait_for_ports() {
    local PORTS="8080 5020 5021 5022 5023 5024 5025 5026 5027 5028 5029 5030"
    local BLOCKED=()
    for port in $PORTS; do
        ss -tlnp 2>/dev/null | grep -q ":${port} " && BLOCKED+=("$port")
    done

    [ ${#BLOCKED[@]} -eq 0 ] && return 0

    log "Blockierte Ports: ${BLOCKED[*]}"
    local w=0
    while [ $w -lt 15 ]; do
        sleep 1; w=$((w + 1))
        local STILL=()
        for port in "${BLOCKED[@]}"; do
            ss -tlnp 2>/dev/null | grep -q ":${port} " && STILL+=("$port")
        done
        BLOCKED=("${STILL[@]}")
        [ ${#BLOCKED[@]} -eq 0 ] && { log "Ports freigegeben."; return 0; }
    done
    log_error "Ports werden nicht freigegeben: ${BLOCKED[*]}"
    return 1
}

cleanup_modbridge() {
    systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null && { systemctl stop "$SERVICE_NAME" 2>/dev/null || true; sleep 1; }
    kill_all_modbridge_processes || return 1
    check_and_wait_for_ports || return 1
    return 0
}

# ── Release download ─────────────────────────────────────────────────────────

fetch_available_versions() {
    local RESPONSE
    for attempt in 1 2 3; do
        RESPONSE=$(curl -s --connect-timeout 10 --max-time 30 "$RELEASES_API" 2>&1) && break
        [ $attempt -lt 3 ] && sleep 2
    done
    [ -z "$RESPONSE" ] && return 1

    local V
    V=$(echo "$RESPONSE" | jq -r '.[].tag_name' 2>/dev/null | head -10)
    [ -z "$V" ] && return 1
    echo "$V"
}

download_modbridge_binary() {
    local VERSION="$1" VARIANT="$2" ARCH="$3"
    local BINARY_NAME="modbridge-linux-${ARCH}"
    [ "$VARIANT" = "headless" ] && BINARY_NAME="${BINARY_NAME}-headless"

    local URL="${REPO_URL}/releases/download/${VERSION}/${BINARY_NAME}"
    local TEMP="/tmp/${BINARY_NAME}.$$"

    log "Lade herunter: $BINARY_NAME ($VERSION)"

    local ok=0
    for attempt in 1 2 3; do
        if curl -L -o "$TEMP" "$URL" --progress-bar --connect-timeout 10 --max-time 300 2>/dev/null; then
            ok=1; break
        fi
        [ $attempt -lt 3 ] && { log_warn "Download fehlgeschlagen, Versuch $attempt/3..."; sleep 3; rm -f "$TEMP"; }
    done

    [ $ok -eq 0 ] && { log_error "Download fehlgeschlagen"; rm -f "$TEMP"; return 1; }
    [ ! -s "$TEMP" ] && { log_error "Datei ist leer"; rm -f "$TEMP"; return 1; }

    if file "$TEMP" | grep -q "ELF"; then
        mv "$TEMP" "$INSTALL_DIR/modbridge"
        chmod +x "$INSTALL_DIR/modbridge"
        log "Binary erfolgreich heruntergeladen"
        return 0
    else
        log_error "Keine gueltige Binary: $(file "$TEMP")"
        rm -f "$TEMP"
        return 1
    fi
}

# ── Systemd service ──────────────────────────────────────────────────────────

create_systemd_service() {
    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=ModBridge - Modbus TCP Proxy Manager
Documentation=https://github.com/Xerolux/modbridge
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=60
StartLimitBurst=3

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/modbridge -config ${INSTALL_DIR}/config.json
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Hardening
ProtectSystem=strict
ReadWritePaths=${INSTALL_DIR} /var/log
PrivateTmp=true
NoNewPrivileges=false

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    log "systemd-Service erstellt und aktiviert (Autostart)"
}

# ── Installation ──────────────────────────────────────────────────────────────

install_modbridge() {
    check_root
    check_dependencies
    ensure_whiptail

    if [ $AUTO_INSTALL -eq 0 ] && is_modbridge_installed && [ "${MODBRIDGE_FORCE:-0}" != "1" ]; then
        local CUR LATEST STATUS
        CUR=$(get_current_version 2>/dev/null || echo "unbekannt")
        LATEST=$(get_latest_version 2>/dev/null || echo "unbekannt")
        STATUS=$(check_updates_available 2>/dev/null || echo "0")

        show_installation_status

        if [ "$STATUS" = "1" ]; then
            if whiptail --title "Update verfuegbar" \
                --yesno "ModBridge ist installiert ($CUR).\nNeuere Version: $LATEST\n\nAktualisieren?" \
                10 70 --yes-button "Ja" --no-button "Abbrechen" 3>&1 1>&2 2>&3; then
                update_modbridge
                return $?
            fi
            log_info "Abgebrochen. 'modbridge update' fuer manuelles Update."
            return 0
        else
            if whiptail --title "Bereits installiert" \
                --yesno "ModBridge $CUR ist installiert und aktuell.\n\nNeuinstallation erzwingen?" \
                10 70 --yes-button "Neu installieren" --no-button "Abbrechen" 3>&1 1>&2 2>&3; then
                :
            else
                return 0
            fi
        fi
    fi

    print_header

    IFS='|' read -r ARCH_INFO ARCH_NAME <<< "$(detect_architecture)"
    log "Architektur: ${BOLD}$ARCH_NAME${NC}"

    local WEBUI_VARIANT
    if [ $AUTO_INSTALL -eq 1 ]; then
        WEBUI_VARIANT="$DEFAULT_VARIANT"
    else
        whiptail --title "ModBridge Installer" \
            --yesno "Willkommen!\n\nArchitektur: $ARCH_NAME\n\nFortfahren?" \
            12 70 --yes-button "Ja" --no-button "Nein" 3>&1 1>&2 2>&3 || return 0

        local CHOICE
        CHOICE=$(whiptail --title "Variante waehlen" \
            --radiolist "ModBridge Variante:" 12 70 2 \
            "full"     "Mit WebUI (grafische Oberflaeche)" "ON" \
            "headless" "Ohne WebUI (nur Config-Datei)"     "OFF" \
            3>&1 1>&2 2>&3) || return 0

        WEBUI_VARIANT="${CHOICE:-full}"
    fi

    cleanup_modbridge || { whiptail --title "Fehler" --msgbox "Cleanup fehlgeschlagen." 8 50; exit 1; }

    mkdir -p "$INSTALL_DIR"
    self_install_script

    log "Frage Versionen ab..."
    local VERSIONS
    VERSIONS=$(fetch_available_versions) || { log_error "Versionen nicht abrufbar"; exit 1; }

    local SELECTED_VERSION
    if [ $AUTO_INSTALL -eq 1 ]; then
        SELECTED_VERSION=$(echo "$VERSIONS" | head -n 1)
    else
        local VERSION_LIST="" i=1
        while IFS= read -r v; do
            if [ $i -eq 1 ]; then
                VERSION_LIST="$VERSION_LIST \"$v\" \"(latest)\" \"ON\""
            else
                VERSION_LIST="$VERSION_LIST \"$v\" \"\" \"OFF\""
            fi
            i=$((i+1))
        done <<< "$VERSIONS"

        eval "SELECTED_VERSION=\$(whiptail --title \"Version waehlen\" \
            --radiolist \"Verfuegbare Versionen:\" 20 70 $i \
            $VERSION_LIST 3>&1 1>&2 2>&3)" || return 0
    fi

    [ -z "$SELECTED_VERSION" ] && { log_info "Abgebrochen."; return 0; }
    log "Version: $SELECTED_VERSION"

    download_modbridge_binary "$SELECTED_VERSION" "$WEBUI_VARIANT" "$ARCH_INFO" || {
        whiptail --title "Fehler" --msgbox "Download fehlgeschlagen.\nInternetverbindung pruefen." 8 50
        exit 1
    }

    if [ "$WEBUI_VARIANT" = "headless" ]; then
        if ! "$INSTALL_DIR/modbridge" -config > "$INSTALL_DIR/config.json" 2>/dev/null; then
            log_warn "Standard-Config konnte nicht erstellt werden"
        fi
    fi

    create_systemd_service
    systemctl start "$SERVICE_NAME"

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "${GREEN}Service laeuft${NC}"
    else
        log_error "Service konnte nicht gestartet werden"
        systemctl status "$SERVICE_NAME" --no-pager
        exit 1
    fi

    local IP
    IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "<IP>")

    echo ""
    log "${GREEN}Installation erfolgreich!${NC}"
    log "  Version:   $SELECTED_VERSION"
    log "  Variante:  $([ "$WEBUI_VARIANT" = "headless" ] && echo "Headless" || echo "Full (WebUI)")"
    log "  Binary:    $INSTALL_DIR/modbridge"
    log "  Config:    $INSTALL_DIR/config.json"
    log "  Service:   $SERVICE_NAME (Autostart aktiviert)"
    [ "$WEBUI_VARIANT" = "full" ] && log "  WebUI:     http://${IP}:8080"
    log "  CLI:       modbridge (ueberall verfuegbar)"
    echo ""

    if [ $AUTO_INSTALL -eq 0 ] && command -v whiptail &>/dev/null; then
        local MSG="ModBridge $SELECTED_VERSION installiert!\n\nService: Autostart aktiviert"
        [ "$WEBUI_VARIANT" = "full" ] && MSG="${MSG}\nWebUI: http://${IP}:8080"
        MSG="${MSG}\n\nCLI: modbridge {start|stop|restart|status|logs|...}"
        whiptail --title "Fertig" --msgbox "$MSG" 14 65 || true
    fi
}

# ── Update ────────────────────────────────────────────────────────────────────

update_modbridge() {
    check_root
    check_dependencies
    ensure_whiptail

    if ! is_modbridge_installed; then
        log_error "ModBridge ist nicht installiert. Bitte 'modbridge install' ausfuehren."
        exit 1
    fi

    print_header

    local CUR LATEST
    CUR=$(get_current_version 2>/dev/null || echo "unbekannt")
    LATEST=$(get_latest_version 2>/dev/null || echo "unbekannt")
    log "Aktuell: $CUR | Verfuegbar: $LATEST"

    IFS='|' read -r ARCH_INFO ARCH_NAME <<< "$(detect_architecture)"

    local SIZE
    SIZE=$(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || stat -f%z "$INSTALL_DIR/modbridge" 2>/dev/null || echo 0)
    local CURRENT_VARIANT="full"
    [ "$SIZE" -lt 8000000 ] && CURRENT_VARIANT="headless"

    local WEBUI_VARIANT
    if [ $AUTO_INSTALL -eq 1 ]; then
        WEBUI_VARIANT="$DEFAULT_VARIANT"
    else
        whiptail --title "ModBridge Update" \
            --yesno "Update durchfuehren?\n\nAktuell: $CUR\nNeu: $LATEST\nArchitektur: $ARCH_NAME" \
            12 70 --yes-button "Ja" --no-button "Nein" 3>&1 1>&2 2>&3 || return 0

        local CHOICE
        CHOICE=$(whiptail --title "Variante waehlen" \
            --radiolist "ModBridge Variante:" 12 70 2 \
            "full"     "Mit WebUI"       "$([ "$CURRENT_VARIANT" = "full" ] && echo "ON" || echo "OFF")" \
            "headless" "Ohne WebUI"      "$([ "$CURRENT_VARIANT" = "headless" ] && echo "ON" || echo "OFF")" \
            3>&1 1>&2 2>&3) || WEBUI_VARIANT="$CURRENT_VARIANT"
        WEBUI_VARIANT="${CHOICE:-$CURRENT_VARIANT}"
    fi

    cleanup_modbridge || { whiptail --title "Fehler" --msgbox "Cleanup fehlgeschlagen." 8 50; exit 1; }

    [ -f "$INSTALL_DIR/config.json" ] && backup_config
    [ -f "$INSTALL_DIR/modbridge" ] && cp "$INSTALL_DIR/modbridge" "$INSTALL_DIR/modbridge.backup.$(date +%Y%m%d_%H%M%S)"

    local VERSIONS
    VERSIONS=$(fetch_available_versions) || { log_error "Versionen nicht abrufbar"; exit 1; }

    local SELECTED_VERSION
    if [ $AUTO_INSTALL -eq 1 ]; then
        SELECTED_VERSION=$(echo "$VERSIONS" | head -n 1)
    else
        local VERSION_LIST="" i=1
        while IFS= read -r v; do
            if [ $i -eq 1 ]; then
                VERSION_LIST="$VERSION_LIST \"$v\" \"(latest)\" \"ON\""
            else
                VERSION_LIST="$VERSION_LIST \"$v\" \"\" \"OFF\""
            fi
            i=$((i+1))
        done <<< "$VERSIONS"
        eval "SELECTED_VERSION=\$(whiptail --title \"Version waehlen\" \
            --radiolist \"Version waehlen:\" 20 70 $i \
            $VERSION_LIST 3>&1 1>&2 2>&3)" || return 0
    fi

    [ -z "$SELECTED_VERSION" ] && { log_info "Abgebrochen."; return 0; }

    download_modbridge_binary "$SELECTED_VERSION" "$WEBUI_VARIANT" "$ARCH_INFO" || {
        whiptail --title "Fehler" --msgbox "Download fehlgeschlagen." 8 50
        exit 1
    }

    systemctl restart "$SERVICE_NAME"

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "${GREEN}Update erfolgreich - Service laeuft${NC}"
    else
        log_error "Service start fehlgeschlagen - versuche Rollback..."
        local LATEST_BAK
        LATEST_BAK=$(ls -t "$INSTALL_DIR"/modbridge.backup.* 2>/dev/null | head -n 1)
        if [ -n "$LATEST_BAK" ]; then
            cp "$LATEST_BAK" "$INSTALL_DIR/modbridge"
            chmod +x "$INSTALL_DIR/modbridge"
            systemctl restart "$SERVICE_NAME" 2>/dev/null || true
            systemctl is-active --quiet "$SERVICE_NAME" && log "Rollback erfolgreich" || log_error "Rollback fehlgeschlagen"
        fi
        exit 1
    fi

    local BACKUP_COUNT
    BACKUP_COUNT=$(ls "$INSTALL_DIR"/modbridge.backup.* 2>/dev/null | wc -l)
    [ "$BACKUP_COUNT" -gt 3 ] && { ls -t "$INSTALL_DIR"/modbridge.backup.* | tail -n +4 | xargs rm -f; log "Alte Backups aufgeraeumt."; }

    echo ""
    log "${GREEN}Update auf $SELECTED_VERSION erfolgreich!${NC}"

    [ $AUTO_INSTALL -eq 0 ] && command -v whiptail &>/dev/null && \
        whiptail --title "Update fertig" --msgbox "ModBridge $SELECTED_VERSION\nService laeuft." 8 40 || true
}

# ── Service management ────────────────────────────────────────────────────────

start_service() {
    check_root
    if ! is_modbridge_installed; then
        log_error "ModBridge nicht installiert. 'modbridge install' ausfuehren."
        exit 1
    fi
    cleanup_modbridge 2>/dev/null || true
    systemctl start "$SERVICE_NAME"
    log "Service gestartet"
}

stop_service() {
    check_root
    cleanup_modbridge
    log "Service gestoppt"
}

restart_service() {
    check_root
    if ! is_modbridge_installed; then
        log_error "ModBridge nicht installiert."
        exit 1
    fi
    cleanup_modbridge 2>/dev/null || true
    systemctl restart "$SERVICE_NAME"
    log "Service neugestartet"
}

status_service() {
    check_root
    show_installation_status
    echo "Service:"
    systemctl status "$SERVICE_NAME" --no-pager 2>/dev/null || echo "Service nicht gefunden"
    echo ""
    echo "Prozesse:"
    pgrep -a modbridge 2>/dev/null || echo "Keine Prozesse"
    echo ""
    echo "Ports:"
    for port in 8080 5020 5021 5022 5023; do
        if ss -tlnp 2>/dev/null | grep -q ":${port} "; then
            echo -e "  :$port  ${GREEN}BELEGT${NC}"
        else
            echo -e "  :$port  ${RED}FREI${NC}"
        fi
    done
}

logs_service() {
    check_root
    if [ "${1:-}" = "--follow" ] || [ "${1:-}" = "-f" ]; then
        journalctl -u "$SERVICE_NAME" -f
    else
        local LINES="${1:-50}"
        journalctl -u "$SERVICE_NAME" -n "$LINES" --no-pager
    fi
}

# ── Version ───────────────────────────────────────────────────────────────────

version_service() {
    echo "ModBridge Manager Script: v${SCRIPT_VERSION}"
    echo ""
    if is_modbridge_installed; then
        local VER SIZE SIZE_MB
        VER=$("$INSTALL_DIR/modbridge" -version 2>/dev/null || echo "unbekannt")
        SIZE=$(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || stat -f%z "$INSTALL_DIR/modbridge" 2>/dev/null || echo 0)
        SIZE_MB=$(awk "BEGIN {printf \"%.2f\", $SIZE/1024/1024}")
        echo "  Binary:      $INSTALL_DIR/modbridge"
        echo "  Version:     $VER"
        echo "  Groesse:     ${SIZE_MB} MB"
        [ "$SIZE" -lt 8000000 ] && echo "  Variante:    Headless" || echo "  Variante:    Full (WebUI)"

        if systemctl is-active --quiet "$SERVICE_NAME"; then
            echo -e "  Service:     ${GREEN}Aktiv${NC}"
        else
            echo -e "  Service:     ${RED}Inaktiv${NC}"
        fi
        if systemctl is-enabled --quiet "$SERVICE_NAME"; then
            echo -e "  Autostart:   ${GREEN}Ja${NC}"
        else
            echo -e "  Autostart:   ${RED}Nein${NC}"
        fi
    else
        echo "  ModBridge ist nicht installiert."
    fi
}

# ── Health check ──────────────────────────────────────────────────────────────

health_check() {
    local RC=0
    echo "ModBridge Health Check:"
    echo "======================="
    echo ""

    [ -f "$INSTALL_DIR/modbridge" ]  && echo -e "[${GREEN}OK${NC}] Binary"       || { echo -e "[${RED}!!${NC}] Binary fehlt"; RC=1; }
    [ -f "$INSTALL_DIR/config.json" ] && echo -e "[${GREEN}OK${NC}] Config"       || echo -e "[${YELLOW}??${NC}] Config fehlt"
    [ -f "$SERVICE_FILE" ]           && echo -e "[${GREEN}OK${NC}] Service-Datei" || { echo -e "[${RED}!!${NC}] Service-Datei fehlt"; RC=1; }

    systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null && echo -e "[${GREEN}OK${NC}] Service aktiv"   || { echo -e "[${RED}!!${NC}] Service inaktiv"; RC=1; }
    systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null && echo -e "[${GREEN}OK${NC}] Autostart"      || echo -e "[${YELLOW}??${NC}] Autostart deaktiviert"

    echo ""
    echo "Ports:"
    for port in 8080 5020 5021 5022 5023; do
        if ss -tlnp 2>/dev/null | grep -q ":${port} "; then
            echo -e "  :$port  [${GREEN}OFFEN${NC}]"
        else
            echo -e "  :$port  [${RED}ZU${NC}]"
        fi
    done

    echo ""
    [ $RC -eq 0 ] && echo -e "${GREEN}Status: OK${NC}" || echo -e "${RED}Status: PROBLEME GEFUNDEN${NC}"
    return $RC
}

# ── Config backup / restore ──────────────────────────────────────────────────

backup_config() {
    local BACKUP_DIR="$INSTALL_DIR/backups"
    mkdir -p "$BACKUP_DIR"

    local TS=$(date +%Y%m%d_%H%M%S)
    [ -f "$INSTALL_DIR/config.json" ] && cp "$INSTALL_DIR/config.json" "$BACKUP_DIR/config-${TS}.json"
    [ -f "$INSTALL_DIR/modbridge.db" ] && cp "$INSTALL_DIR/modbridge.db" "$BACKUP_DIR/db-${TS}.db" 2>/dev/null || true

    log "Backup nach $BACKUP_DIR/"
}

edit_config() {
    check_root
    if [ ! -f "$INSTALL_DIR/config.json" ]; then
        log_error "Config nicht gefunden: $INSTALL_DIR/config.json"
        return 1
    fi
    local EDITOR="${EDITOR:-nano}"
    command -v "$EDITOR" &>/dev/null || EDITOR=vi
    $EDITOR "$INSTALL_DIR/config.json"
    log_info "Aenderungen uebernommen. Neustart mit: modbridge restart"
}

# ── Uninstall ─────────────────────────────────────────────────────────────────

uninstall_modbridge() {
    check_root
    ensure_whiptail

    if [ ! -d "$INSTALL_DIR" ]; then
        log_error "ModBridge ist nicht installiert."
        exit 1
    fi

    whiptail --title "Deinstallieren" \
        --yesno "ModBridge vollstaendig entfernen?\n\n- Service stoppen\n- $INSTALL_DIR loeschen\n- Service-Datei entfernen\n\nWARNUNG: Alle Daten gehen verloren!" \
        14 70 --yes-button "Ja, entfernen" --no-button "Abbrechen" 3>&1 1>&2 2>&3 || return 0

    if whiptail --title "Backup?" \
        --yesno "Config + DB vorher sichern?" \
        8 50 --yes-button "Ja" --no-button "Nein" 3>&1 1>&2 2>&3; then
        backup_config
    fi

    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    cleanup_modbridge 2>/dev/null || true
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    rm -rf "$INSTALL_DIR"
    rm -f "$SCRIPT_TARGET"

    log "${GREEN}ModBridge deinstalliert${NC}"
}

# ── Whiptail TUI Menu ─────────────────────────────────────────────────────────

show_tui_menu() {
    ensure_whiptail

    while true; do
        local INSTALLED="nein"
        local SVC_STATUS="n/a"
        local VER="n/a"

        if is_modbridge_installed; then
            INSTALLED="ja"
            VER=$(get_current_version 2>/dev/null || echo "?")
            if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
                SVC_STATUS="${GREEN}laeuft${NC}"
            else
                SVC_STATUS="${RED}gestoppt${NC}"
            fi
        fi

        local CHOICE
        CHOICE=$(whiptail --title "ModBridge Manager v${SCRIPT_VERSION}" \
            --menu "ModBridge - Modbus TCP Proxy Manager\n\nInstalliert: $INSTALLED ($VER)\nService: $SVC_STATUS\n" \
            22 70 14 \
            "install"    "ModBridge installieren" \
            "update"     "ModBridge aktualisieren" \
            "start"      "Service starten" \
            "stop"       "Service stoppen" \
            "restart"    "Service neustarten" \
            "status"     "Status anzeigen" \
            "logs"       "Logs anzeigen" \
            "health"     "Health-Check" \
            "config"     "Konfiguration bearbeiten" \
            "backup"     "Backup erstellen" \
            "version"    "Version anzeigen" \
            "uninstall"  "ModBridge entfernen" \
            "quit"       "Beenden" \
            3>&1 1>&2 2>&3) || break

        case "$CHOICE" in
            install)   install_modbridge ;;
            update)    update_modbridge ;;
            start)     start_service ;;
            stop)      stop_service ;;
            restart)   restart_service ;;
            status)    status_service | less -R ;;
            logs)      logs_service "$(whiptail --title "Logs" --inputbox "Anzahl Zeilen (oder -f fuer live):" 8 50 "50" 3>&1 1>&2 2>&3)" ;;
            health)    health_check; echo ""; read -p "Enter druecken..." ;;
            config)    edit_config ;;
            backup)    backup_config ;;
            version)   version_service; echo ""; read -p "Enter druecken..." ;;
            uninstall) uninstall_modbridge ;;
            quit)      break ;;
        esac
    done
}

# ── Help ──────────────────────────────────────────────────────────────────────

show_help() {
    print_header
    cat <<EOF
Verwendung:
  modbridge                          Interaktives TUI-Menue
  modbridge install [--auto|--headless]   Installieren
  modbridge update [--auto]              Aktualisieren
  modbridge start                        Service starten
  modbridge stop                         Service stoppen
  modbridge restart                      Service neustarten
  modbridge status                       Status anzeigen
  modbridge logs [-f|N]                  Logs (live oder N Zeilen)
  modbridge health                       Health-Check
  modbridge config                       Config bearbeiten
  modbridge backup                       Backup erstellen
  modbridge version                      Version anzeigen
  modbridge uninstall                    Deinstallieren

Optionen:
  --force     Installation erzwingen
  --auto      Automatischer Modus (keine Dialoge)
  --headless  Automatischer Modus, Headless-Variante
  NO_UPDATE=1 Script-Update ueberspringen
EOF
}

# ═══════════════════════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════════════════════

self_update "$@"

for arg in "$@"; do
    case "$arg" in
        --force)    export MODBRIDGE_FORCE=1 ;;
        --auto)     AUTO_INSTALL=1; DEFAULT_VARIANT="full" ;;
        --headless) AUTO_INSTALL=1; DEFAULT_VARIANT="headless" ;;
    esac
done

case "${1:-}" in
    install)   install_modbridge ;;
    update)    update_modbridge ;;
    start)     start_service ;;
    stop)      stop_service ;;
    restart)   restart_service ;;
    status)    status_service ;;
    logs)      logs_service "${2:-50}" ;;
    health|--health) health_check ;;
    config)    edit_config ;;
    backup)    backup_config ;;
    version|--version|-v) version_service ;;
    uninstall) uninstall_modbridge ;;
    help|--help|-h) show_help ;;
    "")        show_tui_menu ;;
    *)         show_help ;;
esac
