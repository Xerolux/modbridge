# ModBridge auf Home Assistant OS

1. [Repository in Home Assistant hinzufügen](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2FXerolux%2Fmodbridge). Alternativ im App-Store unter **Repositories** die URL `https://github.com/Xerolux/modbridge` eintragen.
2. Im App-Store unter **Xerolux Apps** ModBridge installieren. Der Supervisor baut das Container-Image für AMD64 oder ARM64. Der erste Build kann einige Minuten dauern.
3. Die App starten, **In Seitenleiste anzeigen** einschalten und **ModBridge** öffnen. Das Panel ist für HA-Administratoren sichtbar. Bei einer frischen Installation steht das einmalige ModBridge-Administratorkennwort im App-Protokoll. Benutzername ist `admin`.

Die App übernimmt ModBridge-Anmeldung, Benutzerrollen, Proxy-Konfiguration, Dashboard und Live-Ansichten. Ingress stellt die Weboberfläche über die Home-Assistant-Adresse bereit. Die SQLite-Datenbank, Konfiguration und Logs liegen im persistenten App-Verzeichnis `/data/modbridge`.

Für Modbus-Clients im LAN unter **Konfiguration → Netzwerk** die gewünschten TCP-Ports zwischen 5020 und 5030 veröffentlichen. Anschließend in ModBridge einen Proxy mit der entsprechenden Container-Portnummer einrichten, z. B. `:5020`. Der Client verbindet sich mit der IP von Home Assistant und der eingestellten Host-Portnummer.

App-Updates werden über den Home-Assistant-App-Store installiert. Für einen lokalen Build steht weiterhin `make ha-app` zur Verfügung.
