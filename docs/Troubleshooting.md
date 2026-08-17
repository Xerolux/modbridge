# Troubleshooting

## Port bereits in Verwendung

```bash
# Prozess finden, der den Port belegt
sudo lsof -i :8080

# Prozess beenden
sudo kill -9 <PID>

# Oder anderen Port verwenden (Umgebungsvariable)
WEB_PORT=:9090 ./modbridge
```

## Keine Verbindung zum Zielgerät

```bash
# Erreichbarkeit prüfen
ping 192.168.1.100

# Port prüfen
nc -zv 192.168.1.100 502
```

## Timeouts trotz erreichbarem Zielgerät (Home Assistant, SolarEdge & Co.)

Typisches Bild im Client-Log: einzelne Anfragen laufen in ein Timeout, danach
schlägt jede weitere Abfrage fehl, bis der Client die Verbindung neu aufbaut.
In `pymodbus`-basierten Integrationen sieht das so aus:

```
Error reading inverter ID 4 at InverterCommon:
Response timeout after 3 seconds for transaction with ID 0x23
```

Drei Ursachen, die ModBridge gezielt abfängt:

1. **Mehrere Sitzungen zum Gerät.** Viele Wechselrichter (SolarEdge/SunSpec,
   kleine RTU-Gateways) beantworten nur eine Modbus-Verbindung und lassen
   weitere still ins Leere laufen. `max_target_conns: 1` erzwingt genau eine
   Verbindung zum Zielgerät; Anfragen mehrerer Clients werden davor
   eingereiht.
2. **Anfragen zu dicht hintereinander.** Manche Geräte verwerfen Anfragen, die
   ohne Pause aufeinander folgen. `min_request_gap_ms` (z.B. `100`) setzt einen
   Mindestabstand.
3. **Antwort kommt, nachdem der Client aufgegeben hat.** Läuft die
   Weiterleitung inklusive Wiederholungen länger als das Timeout des Clients,
   trifft die späte Antwort auf dessen nächste Anfrage — ab da passt keine
   Transaktions-ID mehr und jede Abfrage schlägt fehl. `request_timeout_ms`
   deckelt die gesamte Anfrage; ist das Budget aufgebraucht, antwortet
   ModBridge mit einer regulären Modbus-Exception (`0x0B`,
   *Gateway Target Device Failed To Respond*) statt verspätet mit Nutzdaten.

Zusätzlich vergibt ModBridge zum Zielgerät eigene Transaktions-IDs und
verwirft Antworten, die nicht zur laufenden Anfrage gehören. Wie oft das
passiert, steht als `stale_responses` im Proxy-Status — dauerhaft steigende
Werte bedeuten, dass das Gerät langsamer antwortet als die Timeouts erlauben.

### Wenn das Gerät grundsätzlich langsamer ist als der Client wartet

Bei einem SolarEdge-Leader mit Followern (Unit-IDs 2, 3, 4 …) laufen die
Follower-Register über die RS485-Kette und brauchen oft mehr als die 3 s, die
Home Assistant wartet. Typisches Muster: der Leader antwortet zuverlässig, die
Follower laufen in Timeouts.

Hier hilft kein kürzeres Budget, sondern der umgekehrte Weg — Cache plus
Hintergrund-Abfrage:

```json
"max_target_conns": 1,
"min_request_gap_ms": 250,
"read_timeout": 15,
"max_retries": 1,
"request_timeout_ms": 0,
"cache_enabled": true,
"cache_ttl_ms": 5000,
"poll_interval_ms": 5000
```

ModBridge fragt die Register dann selbstständig alle 5 s ab und bedient Home
Assistant sofort aus dem Cache. Der Wert ist dadurch bis zu 5 s alt — für
PV-Daten unkritisch. Das Profil **SolarEdge Leader + Follower** setzt genau
das.

Im Proxy-Dialog des Web-Interface setzt das Geräte-Profil
**SolarEdge / SunSpec** diese Werte mit einem Klick; für Huawei-Wechselrichter
und sDongles gibt es ein eigenes Profil. Die Profile füllen nur das Formular —
bestehende Proxys bleiben unverändert, bis dort ein Profil gewählt wird.

Empfohlener Startpunkt für einen SolarEdge-Wechselrichter mit mehreren
Unit-IDs, abgefragt aus Home Assistant (Client-Timeout dort: 3 s):

```json
{
  "max_target_conns": 1,
  "min_request_gap_ms": 100,
  "request_timeout_ms": 2500,
  "max_retries": 1,
  "read_timeout": 2
}
```

## Admin-Passwort vergessen

### Benutzername und Passwort über die WebUI neu vergeben

Stoppen Sie ModBridge und schalten Sie die WebUI-Wiederherstellung einmalig frei:

```bash
sudo systemctl stop modbridge.service
sudo -u modbridge ./modbridge --enable-account-recovery
sudo systemctl start modbridge.service
```

Öffnen Sie anschließend die Anmeldung und wählen Sie **„Zugangsdaten vergessen?“**. Mit dem ausgegebenen, 15 Minuten gültigen Wiederherstellungscode können Sie einen neuen Benutzernamen und ein neues Passwort vergeben. Der Code ist nur einmal verwendbar.

Existieren mehrere Administratoren, geben Sie das Zielkonto explizit an:

```bash
sudo -u modbridge ./modbridge --enable-account-recovery --recovery-user bisheriger-name
```

### Nur das Passwort per Konsole zurücksetzen

Stoppen Sie ModBridge und erzeugen Sie lokal ein neues einmaliges Passwort für den Admin-Benutzer:

```bash
sudo systemctl stop modbridge.service
sudo -u modbridge ./modbridge --reset-password admin
sudo systemctl start modbridge.service
```

Der Befehl gibt ein zufälliges Einmalpasswort aus. Nach dem Login muss es sofort geändert werden. Führen Sie den Befehl nur lokal auf dem ModBridge-Host aus.

## Docker Container startet nicht

```bash
docker logs modbridge
docker ps -a
```

## systemd-Service Probleme

```bash
sudo bash scripts/modbridge.sh status
journalctl -u modbridge.service -n 100
```
