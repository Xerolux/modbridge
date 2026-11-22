# ModBridge auf Home Assistant OS

1. Das ZIP mit `make ha-app` oder `python scripts/package_ha_app.py` im ModBridge-Checkout bauen.
2. Das ZIP auf Home Assistant OS unter `/addons` entpacken. Die App-Konfiguration muss unter `/addons/modbridge/config.yaml` liegen.
3. In **Einstellungen → Apps** den App-Store öffnen und über das Menü neu laden. Unter **Lokale Apps** ModBridge installieren. Der Supervisor baut dabei das Container-Image für AMD64 oder ARM64.
4. Die App starten, **In Seitenleiste anzeigen** einschalten und **ModBridge** öffnen. Das Panel ist für HA-Administratoren sichtbar. Bei einer frischen Installation steht das einmalige ModBridge-Administratorkennwort im App-Protokoll. Benutzername ist `admin`.

Die App übernimmt ModBridge-Anmeldung, Benutzerrollen, Proxy-Konfiguration, Dashboard und Live-Ansichten. Ingress stellt die Weboberfläche über die Home-Assistant-Adresse bereit. Die SQLite-Datenbank, Konfiguration und Logs liegen im persistenten App-Verzeichnis `/data/modbridge`.

Für Modbus-Clients im LAN unter **Konfiguration → Netzwerk** die gewünschten TCP-Ports zwischen 5020 und 5030 veröffentlichen. Anschließend in ModBridge einen Proxy mit der entsprechenden Container-Portnummer einrichten, z. B. `:5020`. Der Client verbindet sich mit der IP von Home Assistant und der eingestellten Host-Portnummer.

Für App-Updates ein ZIP mit neuer Version entpacken, den App-Store neu laden und ModBridge dort aktualisieren.
