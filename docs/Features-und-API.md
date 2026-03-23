# Features & API

## Kern-Funktionen

* **Multiplexing:** Mehrere Modbus-Clients können gleichzeitig auf ein einzelnes physisches Gerät zugreifen, das nur eine Verbindung erlaubt.
* **Headless-Modus:** Kann komplett ohne Web-Interface kompiliert und betrieben werden (geringerer Ressourcenverbrauch).
* **Connection Pooling & Keep-Alive:** Intelligentes Wiederverwenden von Verbindungen zum Zielgerät, reduziert Latenz und Overhead.
* **Latenz-Optimierung:** Effizientes Zusammenfassen und Pipelining von Anfragen.
* **Intelligentes Polling:** Anfragen nach den gleichen Registern können zusammengefasst werden.
* **Modbus TCP Validierung:** Blockiert ungültige oder fehlerhafte Frames, bevor sie das Zielgerät erreichen.

## Sicherheit

* **Authentifizierung:** Die Web-UI ist durch eine passwortbasierte Authentifizierung geschützt (Bcrypt).
* **Session-Management:** Sichere, zeitgesteuerte Sessions mit automatischem Timeout.
* **Rate Limiting:** Schutz vor Brute-Force- und DoS-Angriffen durch IP-basiertes Rate-Limiting.
* **CSRF-Schutz:** Alle zustandsverändernden API-Endpunkte sind gegen Cross-Site Request Forgery geschützt.
* **Sichere Header:** Implementierung gängiger Security-Header (HSTS, X-Content-Type-Options, etc.).
* **Passwortrichtlinien:** Erzwingung komplexer Passwörter beim Setup.

## API-Endpunkte

| Endpunkt | Methode | Beschreibung |
|----------|---------|--------------|
| `/api/health` | GET | Health Check (kein Login erforderlich) |
| `/api/status` | GET | Server-Status |
| `/api/login` | POST | Anmelden |
| `/api/logout` | POST | Abmelden |
| `/api/proxies` | GET | Alle Proxies auflisten |
| `/api/proxies` | POST | Neuen Proxy anlegen |
| `/api/proxies` | PUT | Proxy aktualisieren (ID im Body) |
| `/api/proxies?id={id}` | DELETE | Proxy löschen |
| `/api/proxies/control` | POST | Proxy steuern (`{id, action: start\|stop\|restart\|pause\|resume}`) |
| `/api/proxies/control` | POST | Alle Proxies steuern (`{action: start_all\|stop_all}`) |
| `/api/proxies/stream` | GET | Live-Proxy-Updates (SSE) |
| `/api/config/system` | GET | Systemkonfiguration abrufen |
| `/api/config/system` | PUT | Systemkonfiguration speichern |
| `/api/config/password` | POST | Passwort ändern |
| `/api/logs` | GET | Log-Einträge abrufen |
| `/api/logs/stream` | GET | Live-Log-Stream (SSE) |
| `/api/devices` | GET | Verbundene Geräte auflisten |
| `/api/system/info` | GET | Systeminformationen & Metriken |
| `/api/system/diagnostics/connectivity` | GET | Verbindbarkeit aller Proxy-Ziele prüfen |
| `/api/metrics` | GET | Prometheus-Metriken (Port `:9090`) |

## Skripte

### scripts/modbridge.sh

Das Haupt-Verwaltungsskript für Installation, Updates und Service-Management.

```bash
sudo bash scripts/modbridge.sh <Befehl>
```

| Befehl | Beschreibung |
|--------|--------------|
| `install` | Modbridge installieren (Binary-Download oder Quellcode-Build, systemd-Setup) |
| `update` | Auf neue Version aktualisieren (mit automatischem Rollback bei Fehler) |
| `uninstall` | Vollständig deinstallieren (Service + Dateien) |
| `start` | systemd-Service starten |
| `stop` | systemd-Service stoppen |
| `restart` | systemd-Service neu starten |
| `status` | Service-Status anzeigen |
| `logs [N]` | Letzte N Log-Einträge anzeigen (Standard: 50) |

**Beispiele:**
```bash
sudo bash scripts/modbridge.sh install    # Erstinstallation
sudo bash scripts/modbridge.sh update     # Update auf neue Version
sudo bash scripts/modbridge.sh status     # Status prüfen
sudo bash scripts/modbridge.sh logs 100   # Letzte 100 Log-Zeilen
sudo bash scripts/modbridge.sh uninstall  # Deinstallieren
```

**Update-Ablauf:**
1. Service stoppen
2. Altes Binary sichern (Backup)
3. Neues Binary herunterladen
4. Service starten
5. Bei Fehler: automatischer Rollback auf Backup (die letzten 3 Backups werden behalten)

---

### scripts/go-updater.sh

Hält die lokale Go-Installation aktuell. Nützlich, wenn ModBridge aus dem Quellcode gebaut wird.

```bash
sudo bash scripts/go-updater.sh <Befehl>
```

| Befehl | Beschreibung |
|--------|--------------|
| `update` | Go auf die neueste stabile Version aktualisieren |
| `install` | systemd-Service einrichten (Update bei jedem Systemstart) |
| `uninstall` | systemd-Service entfernen |
| `start` | Service manuell ausführen |
| `stop` | Laufenden Service stoppen |
| `status` | Status und letzte Log-Einträge anzeigen |

```bash
sudo bash scripts/go-updater.sh update    # Go jetzt aktualisieren
sudo bash scripts/go-updater.sh install   # Autostart einrichten
```

---

### build.sh

Schnelles lokales Build-Skript (Frontend + Go-Binary).

```bash
./build.sh
```

Führt folgende Schritte aus:
1. `npm install` + `npm run build` im `frontend/`-Verzeichnis
2. Kopiert das Build-Ergebnis nach `pkg/web/dist/`

---
