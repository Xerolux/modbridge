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
