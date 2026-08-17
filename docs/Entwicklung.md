# Entwicklung & CI/CD

## Makefile-Befehle

```bash
make help           # Alle verfügbaren Befehle anzeigen
```

| Befehl | Beschreibung |
|--------|--------------|
| `make build` | Frontend bauen und Go-Binary kompilieren |
| `make build-frontend` | Nur Frontend bauen und nach pkg/web/dist/ kopieren |
| `make build-all` | Binaries für alle Plattformen (Linux, Windows, macOS) |
| `make run` | Binary bauen und direkt starten |
| `make test` | Tests mit Race-Detector und Coverage ausführen |
| `make coverage` | Coverage-Report als HTML generieren (`coverage.html`) |
| `make lint` | Linter ausführen (benötigt `golangci-lint`) |
| `make fmt` | Code formatieren (`go fmt` + `gofmt`) |
| `make vet` | `go vet` ausführen |
| `make clean` | Build-Artefakte entfernen |
| `make deps` | Go-Abhängigkeiten herunterladen und aufräumen |
| `make update-deps` | Go-Abhängigkeiten auf neueste Versionen aktualisieren |
| `make install` | Binary systemweit installieren (`go install`) |
| `make dev` | Entwicklungsmodus mit Live-Reload (benötigt `air`) |
| `make docker-build` | Docker Image lokal bauen |
| `make docker-run` | Docker Image bauen und Container starten |
| `make docker-stop` | Docker Container stoppen |
| `make docker-logs` | Docker Container Logs anzeigen |

**Beispiele:**
```bash
make build          # Komplett-Build
make test           # Tests ausführen
make coverage       # Testabdeckung prüfen
make build-all      # Für alle Plattformen bauen
```

---

## Umgebungsvariablen

| Variable | Standard | Beschreibung |
|----------|----------|--------------|
| `WEB_PORT` | `:8080` | Web-UI Port (überschreibt `web_port` aus config.json) |
| `LOG_LEVEL` | `INFO` | Log-Level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `TZ` | `UTC` | Zeitzone für den Container |

Für Docker-Deployments: Kopieren Sie `.env.example` nach `.env` und passen Sie die Werte an.

```bash
cp .env.example .env
# .env anpassen
docker-compose up -d
```

---


## End-to-End-Tests (WebUI)

Unit-Tests und ein grüner Build sagen nichts darüber aus, ob eine Seite auf
Klicks reagiert. Genau dort sind zwei Fehler durchgerutscht — ein nicht
funktionierendes Verschieben von Karten und eine Auswahl, die beim erneuten
Öffnen leer war. Dagegen gibt es jetzt eine Playwright-Suite in
`frontend/e2e/`.

Die Tests starten ein echtes ModBridge-Binary in einem temporären Verzeichnis,
durchlaufen den erzwungenen Passwortwechsel des ersten Starts und bedienen
danach die Oberfläche wie ein Benutzer.

```bash
# WebUI bauen und ins Binary einbetten
npm --prefix frontend run build
rm -rf pkg/web/dist && cp -r frontend/dist pkg/web/dist
CGO_ENABLED=1 go build -o modbridge .

# Tests ausführen
npm --prefix frontend run test:e2e
```

Nützliche Umgebungsvariablen:

| Variable | Zweck |
|----------|-------|
| `MODBRIDGE_BIN` | Pfad zum Binary (Standard: `../modbridge`) |
| `MODBRIDGE_URL` | Adresse der Instanz (Standard: `http://localhost:8080`) |
| `PLAYWRIGHT_CHROMIUM_PATH` | Vorhandenes Chromium nutzen statt herunterzuladen |

In CI läuft das als Job **E2E (WebUI)**; das automatische Release hängt davon
ab. Sollte sich die Suite als unzuverlässig erweisen, nimm `e2e` aus dem
`needs`-Block des `auto-release`-Jobs heraus — dann bleibt der rote Haken am
PR sichtbar, ohne Releases zu blockieren.

**Ein neuer Test ist erst fertig, wenn er auch fehlschlägt.** Beide vorhandenen
Tests wurden gegen den kaputten Stand geprüft und melden dort genau das, was
der Benutzer gemeldet hat.

## Automatisierter Build (GitHub Actions)

Das Projekt nutzt GitHub Actions für automatisierte Builds und Releases:

1. Frontend wird mit Node.js 22 gebaut
2. Go-Binaries für Linux (AMD64/ARM64) und Windows (AMD64)
3. Docker-Images mit Multi-Arch-Support
4. Automatische Releases bei Tags (`v*`)

**Release erstellen:**
```bash
git tag v1.0.0
git push origin v1.0.0
```

**Benötigte GitHub Secrets für Docker-Push:**
```
DOCKER_USERNAME   = Docker Hub Username
DOCKER_PASSWORD   = Docker Hub Access Token
```

---
