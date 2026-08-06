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
