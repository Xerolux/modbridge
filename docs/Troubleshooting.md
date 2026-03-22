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

Das Passwort-Hash steht in `config.json` unter `admin_pass_hash`. Löschen Sie den Wert, um beim nächsten Start ein neues zufälliges Passwort zu generieren:

```bash
# Wert leeren (ModBridge muss gestoppt sein)
# In config.json: "admin_pass_hash": ""

# Wenn systemd-Service verwendet wird:
sudo bash scripts/modbridge.sh restart

# Neues Passwort erscheint in den Logs
sudo bash scripts/modbridge.sh logs
```

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
