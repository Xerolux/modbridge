# Konfiguration

ModBridge speichert seine Konfiguration standardmäßig in der Datei `config.json`.

Die Konfiguration kann über die **Web-Oberfläche** (empfohlen) oder durch **direktes Editieren der `config.json`** im Headless-Modus erfolgen.

## Headless-Betrieb (ohne WebUI)

ModBridge kann komplett ohne grafische Oberfläche betrieben werden. Dies ist ideal für ressourcenschonende Server, Edge-Devices (wie Raspberry Pi) oder die Automatisierung über Konfigurations-Management-Tools (Ansible, Puppet, etc.).

### Konfigurationsdatei erstellen

Wenn noch keine `config.json` existiert, erstellt ModBridge beim ersten Start automatisch eine Standardkonfiguration.

```bash
# ModBridge einmal kurz starten, um config.json zu generieren
./modbridge-linux-amd64-headless &
sleep 2
kill $!
```

### Konfiguration bearbeiten

Bearbeiten Sie die Datei mit einem Texteditor Ihrer Wahl:

```bash
# z.B. vi config.json
```

### Konfigurations-Beispiel

Eine typische `config.json` für den Headless-Betrieb mit zwei konfigurierten Proxies:

```json
{
  "web_port": ":8080",
  "admin_pass_hash": "$2a$10$xyz...",
  "force_password_change": false,
  "session_timeout": 24,

  "proxies": [
    {
      "id": "proxy-1-solar",
      "name": "Solar Wechselrichter",
      "listen_addr": ":5020",
      "target_addr": "192.168.1.100:502",
      "enabled": true,
      "paused": false,
      "connection_timeout": 5,
      "read_timeout": 5,
      "max_retries": 3,
      "description": "Dach-Solaranlage",
      "max_read_size": 0,
      "tags": ["solar", "roof"]
    },
    {
      "id": "proxy-2-hvac",
      "name": "Klimaanlage",
      "listen_addr": ":5021",
      "target_addr": "192.168.1.101:502",
      "enabled": true,
      "paused": false,
      "connection_timeout": 10,
      "read_timeout": 10,
      "max_retries": 1,
      "description": "Klimasteuerung Gebäude A",
      "max_read_size": 256,
      "tags": ["hvac", "building-a"]
    }
  ],

  "log_level": "INFO",
  "log_max_size": 100,
  "log_max_files": 10,
  "log_max_age_days": 30,

  "metrics_enabled": true,
  "metrics_port": ":9090",

  "debug_mode": false,
  "max_connections": 1000
}
```

### Headless-Konfigurationsoptionen

Wichtige Einstellungen für den Server-Betrieb:

| Feld | Beschreibung | Empfehlung für Headless |
|------|--------------|-------------------------|
| `log_level` | Detailgrad der Logs (`DEBUG`, `INFO`, `WARN`, `ERROR`) | `INFO` für Produktion, `WARN` für minimale Logs |
| `metrics_enabled` | Aktiviert den Prometheus-Metrik-Endpunkt | `true` (Sehr nützlich für Server-Monitoring) |
| `metrics_port` | Port für den `/metrics` Endpunkt | `:9090` (Standard) |
| `max_connections` | Maximale Anzahl gleichzeitiger TCP-Verbindungen | Entsprechend der Server-Kapazität anpassen (z.B. `5000`) |

### Proxy-Konfiguration (Headless)

Um einen neuen Proxy im Headless-Modus hinzuzufügen, erweitern Sie das `proxies`-Array in der `config.json`.

**Wichtig:**
* Jeder Proxy benötigt eine eindeutige `id` (kann ein beliebiger String sein, UUID wird empfohlen).
* `listen_addr` muss eindeutig sein und das Format `:PORT` oder `IP:PORT` haben.
* Setzen Sie `"enabled": true`, damit der Proxy beim Start automatisch geladen wird.

### Service nach Konfigurationsänderung neu starten

Damit Änderungen an der `config.json` wirksam werden, muss der ModBridge-Prozess neu gestartet werden:

**Mit systemd:**
```bash
sudo systemctl restart modbridge
```

**Docker:**
```bash
docker restart modbridge
```

### Service-Status und Logs prüfen

Um sicherzustellen, dass Ihre Headless-Konfiguration korrekt geladen wurde:

```bash
# systemd Status prüfen
sudo systemctl status modbridge

# Logs auf Fehler prüfen
sudo journalctl -u modbridge -f
```

## Proxy-Felder (Referenz)

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `id` | string | UUID (wird automatisch vergeben) |
| `name` | string | Anzeigename im Web-Interface |
| `listen_addr` | string | Lokaler Port, z.B. `:5020` |
| `target_addr` | string | Zieladresse, z.B. `192.168.1.100:502` |
| `enabled` | bool | Proxy aktiviert/deaktiviert |
| `paused` | bool | Proxy pausiert (Verbindungen werden abgelehnt) |
| `connection_timeout` | int | Verbindungs-Timeout in Sekunden |
| `read_timeout` | int | Lese-Timeout in Sekunden |
| `max_retries` | int | Maximale Wiederholungsversuche bei Fehler |
| `max_read_size` | int | Max. Modbus-Read-Größe (0 = unbegrenzt) |
| `description` | string | Optionale Beschreibung |
| `tags` | array | Optionale Tags zur Kategorisierung |


### Vollständige config.json (Beispiel mit allen Optionen)

```json
{
  "web_port": ":8080",
  "admin_pass_hash": "",
  "force_password_change": true,
  "session_timeout": 24,

  "proxies": [
    {
      "id": "21e71152-3866-43ac-891d-c5ec85fa1e98",
      "name": "SolarEdge Proxy",
      "listen_addr": ":5020",
      "target_addr": "192.168.1.100:502",
      "enabled": true,
      "paused": false,
      "connection_timeout": 10,
      "read_timeout": 10,
      "max_retries": 3,
      "description": "Verbindet sich mit SolarEdge Anlage",
      "max_read_size": 0,
      "tags": []
    }
  ],

  "log_level": "INFO",
  "log_max_size": 100,
  "log_max_files": 10,
  "log_max_age_days": 30,

  "tls_enabled": false,
  "tls_cert_file": "",
  "tls_key_file": "",

  "cors_allowed_origins": ["*"],
  "cors_allowed_methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
  "cors_allowed_headers": ["Content-Type", "Authorization"],

  "rate_limit_enabled": true,
  "rate_limit_requests": 60,
  "rate_limit_burst": 100,

  "ip_whitelist_enabled": false,
  "ip_whitelist": [],
  "ip_blacklist_enabled": false,
  "ip_blacklist": [],

  "metrics_enabled": true,
  "metrics_port": ":9090",

  "email_enabled": false,
  "email_smtp_server": "",
  "email_smtp_port": 587,
  "email_from": "",
  "email_to": "",
  "email_username": "",
  "email_password": "",
  "email_alert_on_error": true,
  "email_alert_on_warning": false,

  "backup_enabled": true,
  "backup_interval": "daily",
  "backup_retention": 7,
  "backup_path": "./backups",
  "backup_database": true,
  "backup_config": true,

  "debug_mode": false,
  "max_connections": 1000
}
```
