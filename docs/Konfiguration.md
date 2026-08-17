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
| `connect_delay_ms` | int | Pause nach TCP-Connect vor der ersten Anfrage (ms, 0 = aus). Für langsame Geräte wie Huawei-Wechselrichter/sDongles |
| `max_target_conns` | int | Max. gleichzeitige Verbindungen zum Zielgerät (0 = Standard 10). `1` für Geräte mit nur einer Modbus-Sitzung, z.B. SolarEdge/SunSpec |
| `min_request_gap_ms` | int | Mindestabstand zwischen zwei Anfragen an das Zielgerät (ms, 0 = aus) |
| `request_timeout_ms` | int | Hartes Zeitbudget für eine Client-Anfrage inkl. Wiederholungen (ms, 0 = automatisch aus `read_timeout` und `max_retries`) |
| `device_profile` | string | Zuletzt angewendetes Geräte-Profil. Rein informativ — merkt sich, aus welchem Preset die Werte stammen; das Verhalten richtet sich nach den Einzelfeldern |
| `cache_enabled` | bool | Wiederholte Lesezugriffe aus einem Cache bedienen (Standard: aus) |
| `cache_ttl_ms` | int | Gültigkeit eines Cache-Eintrags (ms, 0 = 5000) |
| `poll_interval_ms` | int | Abgefragte Register im Hintergrund aktualisieren (ms, 0 = aus). Setzt `cache_enabled` voraus |
| `protocol` | string | `tcp` (Standard) oder `rtu-tcp` für serielle Adapter, die rohe RTU-Frames erwarten |
| `description` | string | Optionale Beschreibung |
| `tags` | array | Optionale Tags zur Kategorisierung |


### Cache und Hintergrund-Abfrage

Manche Geräte lassen sich nicht beschleunigen: ein SolarEdge-Leader holt
Follower-Register erst über die RS485-Kette, ein Heizungsregler braucht
schlicht Sekunden. Wenn der Client (z.B. Home Assistant mit 3 s Timeout)
schneller aufgibt als das Gerät antwortet, hilft kein Timeout-Tuning mehr.

Dafür gibt es zwei zusammengehörige Optionen:

- **`cache_enabled`** — wiederholte Lesezugriffe werden aus dem Cache bedient,
  statt das Gerät erneut zu fragen.
- **`poll_interval_ms`** — ein Hintergrund-Poller aktualisiert genau die
  Register, die Clients tatsächlich abfragen, in seinem eigenen Takt. Der
  Client bekommt dadurch sofort eine Antwort und wartet nie auf das Gerät.

```json
"cache_enabled": true,
"cache_ttl_ms": 20000,
"poll_interval_ms": 5000
```

**TTL und Intervall gehören zusammen.** `cache_ttl_ms` ist eine Obergrenze für
das Alter eines Werts, kein Aktualisierungsplan — der Poller hält die Einträge
deutlich frischer. Ist die Gültigkeit nicht **mehrfach so groß** wie das
Intervall, verfallen Einträge zwischen zwei Runden, der Client fällt in dieser
Lücke wieder auf das Gerät durch und wartet doch. ModBridge protokolliert beim
Start eine Warnung, wenn die beiden Werte in diesem Verhältnis stehen.

**Was du dabei in Kauf nimmst:** Ein zwischengespeicherter Wert ist per
Definition nicht der Live-Wert — er ist bis zu `cache_ttl_ms` alt. Für
Dashboards und Energiedaten ist das unproblematisch, für Regelkreise nicht.
Deshalb ist die Funktion standardmäßig **aus** und muss bewusst aktiviert
werden.

Was der Cache **nicht** tut:

- Schreibzugriffe werden nie zwischengespeichert und immer durchgereicht.
- Nach einem Schreibzugriff werden alle Cache-Einträge dieser Unit-ID
  verworfen — ein gerade geänderter Wert wäre nicht nur alt, sondern falsch.
- Modbus-Exceptions landen nie im Cache.
- Der Poller fragt nur Register ab, die ein Client vorher angefragt hat, und
  vergisst sie wieder, wenn längere Zeit niemand danach fragt.

Im Proxy-Status stehen `cache_hits`, `cache_misses`, `cache_entries` und
`polled_requests` zum Nachprüfen.

### Geräte-Profile

Im Proxy-Dialog des Web-Interface füllt ein Geräte-Profil die Felder oben mit
Werten, die zum Verhalten des Zielgeräts passen. Die Auswahl ist nach
Kategorien gruppiert und durchsuchbar.

**Wichtig zur Einordnung:** Die Profile beschreiben eine *Verhaltensklasse*,
keine Hersteller-Spezifikation. Die Werte stecken in neun Klassen; die knapp 60
Geräteeinträge bilden nur ab, in welche Klasse ein Gerät fällt — wie viele
Modbus-Sitzungen es bedient, wie viel Luft es zwischen Anfragen braucht, wie
lange eine Antwort dauern darf und ob es Modbus TCP oder rohes RTU spricht.
Gemessene Timing-Werte pro Modell sind das ausdrücklich nicht. Ein Profil ist
ein Startpunkt, der ein Gerät stabil hält — Feintuning bleibt deine Sache.

#### Verhaltensklassen

| Klasse | Verhalten | Kern-Einstellung |
|--------|-----------|------------------|
| `standard` | ModBridge-Voreinstellungen | keine Begrenzung |
| `multiSession` | schnell, viele Clients | keine Begrenzung, kein Abstand |
| `fewSessions` | einige parallele Sitzungen | 2 Verbindungen, 50 ms Abstand |
| `singleSession` | nur eine Modbus-Sitzung | 1 Verbindung, 100 ms Abstand |
| `singleSessionFast` | eine Sitzung, Client mit kurzem Timeout | + Budget 2,5 s, Lese-Timeout 2 s |
| `singleSessionSlow` | eine Sitzung, träge Steuerung | 250 ms Abstand, 10 s Lese-Timeout |
| `connectDelay` | verwirft Anfragen direkt nach dem Connect | 3 s Pause, 1 Verbindung |
| `serialGateway` | serielle Leitung hinter TCP | 1 Verbindung, 50 ms Abstand, Reads auf 125 Register |
| `rtuOverTcp` | rohe RTU-Frames ohne MBAP-Header | Protokoll `rtu-tcp`, 1 Verbindung |

#### Kategorien und Geräte

| Kategorie | Geräte |
|-----------|--------|
| Allgemein | Standard, SPS/PLC, Modbus-TCP→RTU-Gateway, serieller Adapter |
| Wechselrichter / PV | SolarEdge (einzeln), SolarEdge Leader+Follower, SMA, Fronius, Kostal, Huawei, Sungrow, GoodWe, Growatt, SolaX, Deye/Sunsynk, Sofar, Delta, KACO, FIMER/ABB, E3/DC, Victron, SunSpec allgemein |
| Wärmepumpen / Heizung | IDM (Navigator 2.0 / Navigator 10), Stiebel Eltron ISG, Tecalor ISG, NIBE S-Serie, NIBE MODBUS 40, Lambda, Waterkotte, Ochsner, Nilan, Daikin Altherma, Panasonic Aquarea, LG Therma V, Mitsubishi Ecodan |
| Lüftung / Klima | Helios KWL, Zehnder ComfoAir Q, Vallox, Pluggit, Wolf CWL |
| Energiezähler | Eastron SDM, Carlo Gavazzi EM24/EM340, Janitza UMG, Schneider iEM3000, Siemens SENTRON PAC, ABB B23/B24, Finder 7M, Iskra WM3, Shelly Pro EM |
| Batteriespeicher | BYD Battery-Box, Pylontech, VARTA |
| Wallboxen / Ladeinfrastruktur | KEBA, Alfen Eve, go-e, Wallbox Pulsar Plus, MENNEKES AMTRON, Webasto, ABL eMH, Heidelberg Energy Control |

Geräte, die Modbus nur über einen separaten Adapter erreichen (z.B. NIBE
MODBUS 40, LG PI485, viele Zähler), sind im Hinweistext entsprechend
gekennzeichnet — dort gelten die Werte für den Adapter, nicht für das Gerät.

Ein neues Gerät ist ein Eintrag in `frontend/src/deviceProfiles.js` mit Label
und Klasse; Kategorien, Klassen-Hinweise und Notizen liegen in
`frontend/src/i18n.js` unter `control.profiles.*`.

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
      "connect_delay_ms": 0,
      "max_target_conns": 1,
      "min_request_gap_ms": 100,
      "request_timeout_ms": 2500,
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
