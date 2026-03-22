# Erste Schritte

## 1. Anwendung starten

Wenn Sie den systemd-Service installiert haben:
```bash
sudo systemctl start modbridge.service
```

Oder manuell:
```bash
./modbridge
```

## 2. Erstes Login

Beim ersten Start generiert ModBridge automatisch ein sicheres Admin-Passwort. Dieses finden Sie in der Konsolenausgabe oder im systemd-Log.

```bash
# Passwort im Log finden
sudo journalctl -u modbridge.service -n 50 | grep "Initial admin password"
```

## 3. Web-Interface öffnen

Öffnen Sie einen Browser und navigieren Sie zu:
`http://<ihre-server-ip>:8080`

Melden Sie sich an mit:
* **Benutzername:** admin
* **Passwort:** (Das generierte Passwort aus Schritt 2)

Nach dem ersten Login werden Sie aufgefordert, das Passwort zu ändern.

## 4. Proxy anlegen

1. Gehen Sie zu **Proxies** im linken Menü
2. Klicken Sie auf **Neuer Proxy**
3. Füllen Sie die Details aus:
   * **Name:** z.B. "Wechselrichter 1"
   * **Lokaler Port:** z.B. `:5020` (Der Port, auf dem ModBridge lauschen soll)
   * **Zieladresse:** z.B. `192.168.1.100:502` (IP-Adresse Ihres Modbus-Geräts)
4. Klicken Sie auf **Speichern**
5. Aktivieren Sie den Proxy über den Schalter in der Liste

## 5. Modbus-Client verbinden

Konfigurieren Sie Ihr SCADA-System, Home Assistant oder Node-RED so, dass es sich nicht direkt mit dem Gerät, sondern mit ModBridge verbindet:

* **IP:** `<IP-des-ModBridge-Servers>`
* **Port:** `5020` (oder den in Schritt 4 konfigurierten lokalen Port)

Das war's! ModBridge leitet nun die Anfragen weiter und loggt den Datenverkehr.
