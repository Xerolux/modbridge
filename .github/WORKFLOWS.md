# GitHub Actions Workflows

Dieses Repository nutzt GitHub Actions für automatische Builds, Tests und Releases.

---

## Workflows Übersicht

### 1. CI (Continuous Integration)
**Datei**: `.github/workflows/ci.yml`

**Trigger**:
- Push auf `main`, `master`, `develop`
- Pull Requests auf `main`, `master`, `develop`

**Was es macht**:
- **Tests**: Führt alle Go-Tests aus (mit Go 1.21, 1.22, 1.23)
- **Lint**: Führt golangci-lint aus
- **Build**: Kompiliert das Binary
- **Coverage**: Lädt Test-Coverage zu Codecov hoch

**Status**: ✅ Aktiv

---

### 2. Release
**Datei**: `.github/workflows/release.yml`

**Trigger**:
- Push eines Tags (z.B. `v0.1.0`)

**Was es macht**:
1. **Tests**: Führt alle Tests aus
2. **Binaries**: Baut für alle Plattformen:
   - Linux (AMD64, ARM64)
   - Windows (AMD64)
   - macOS (AMD64, ARM64)
3. **.deb Pakete**: Baut Debian/Ubuntu Pakete:
   - AMD64 (x86_64)
   - ARM64 (Raspberry Pi)
4. **Docker Image**: Baut und pusht Multi-Arch Image zu `ghcr.io`:
   - `ghcr.io/xerolux/modbridge:latest`
   - `ghcr.io/xerolux/modbridge:v0.1.0`
   - `ghcr.io/xerolux/modbridge:0.1`
   - `ghcr.io/xerolux/modbridge:0`
5. **GitHub Release**: Erstellt Release mit:
   - Allen Binaries
   - .deb Paketen
   - Automatischen Release Notes
   - Installations-Anleitung

**Status**: ✅ Aktiv

**Beispiel-Nutzung**:
```bash
# Release erstellen
git tag v0.2.0
git push origin v0.2.0

# Warten, bis Workflow fertig ist (~5-10 Minuten)
# Release ist dann verfügbar unter:
# https://github.com/Xerolux/modbridge/releases/tag/v0.2.0
```

---

### 3. Docker Publish
**Datei**: `.github/workflows/docker-publish.yml`

**Trigger**:
- Push auf `main`, `master`, `develop`
- Pull Requests auf `main`, `master`

**Was es macht**:
1. **Docker Image**: Baut Multi-Arch Image (AMD64 + ARM64)
2. **Push zu ghcr.io**: Nur bei Push (nicht bei PRs)
3. **Tags**:
   - `ghcr.io/xerolux/modbridge:main` - Main branch
   - `ghcr.io/xerolux/modbridge:edge` - Neueste Version
   - `ghcr.io/xerolux/modbridge:main-abc1234` - Commit SHA

**Status**: ✅ Aktiv

**Verwendung**:
```bash
# Neuestes Image pullen
docker pull ghcr.io/xerolux/modbridge:edge

# Spezifischen Branch pullen
docker pull ghcr.io/xerolux/modbridge:main
```

---

### 4. PR Check
**Datei**: `.github/workflows/pr-check.yml`

**Trigger**:
- Pull Requests

**Was es macht**:
- Prüft Code-Qualität
- Führt Tests aus
- Validiert Änderungen

**Status**: ✅ Aktiv

---

### 5. Security
**Datei**: `.github/workflows/security.yml`

**Trigger**:
- Schedule (wöchentlich)
- Manuell

**Was es macht**:
- Scannt Code auf Sicherheitslücken
- Prüft Abhängigkeiten
- CodeQL-Analyse

**Status**: ✅ Aktiv

---

### 6. Stale Issues/PRs
**Datei**: `.github/workflows/stale.yml`

**Trigger**:
- Schedule (täglich)

**Was es macht**:
- Markiert inaktive Issues/PRs als "stale"
- Schließt sie nach Wartezeit

**Status**: ✅ Aktiv

---

### 7. Labeler
**Datei**: `.github/workflows/labeler.yml`

**Trigger**:
- Pull Requests

**Was es macht**:
- Fügt automatisch Labels basierend auf geänderten Dateien hinzu

**Status**: ✅ Aktiv

---

## Veröffentlichungs-Prozess

### Release erstellen (automatisch)

1. **Version vorbereiten**:
   ```bash
   # Version in version.txt setzen
   echo "0.2.0" > version.txt
   git add version.txt
   git commit -m "Bump version to 0.2.0"
   git push
   ```

2. **Tag erstellen**:
   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```

3. **Warten**: Workflow läuft automatisch (~5-10 Minuten)

4. **Fertig**: Release ist verfügbar unter:
   - GitHub Releases: `https://github.com/Xerolux/modbridge/releases`
   - Docker: `ghcr.io/xerolux/modbridge:v0.2.0`

---

## Docker Image Registry

### GitHub Container Registry (ghcr.io)

**Public Registry**: Jeder kann Images pullen (kein Login erforderlich)

**Verfügbare Images**:
```bash
# Stabile Releases
ghcr.io/xerolux/modbridge:latest        # Neueste Version
ghcr.io/xerolux/modbridge:v0.1.0        # Spezifische Version
ghcr.io/xerolux/modbridge:0.1           # Major.Minor
ghcr.io/xerolux/modbridge:0             # Major

# Development
ghcr.io/xerolux/modbridge:main          # Main branch
ghcr.io/xerolux/modbridge:edge          # Bleeding edge
ghcr.io/xerolux/modbridge:develop       # Develop branch
```

**Architektur-Support**:
- ✅ AMD64 (x86_64)
- ✅ ARM64 (aarch64)

**Image-Details**:
```bash
# Image-Info anzeigen
docker image inspect ghcr.io/xerolux/modbridge:latest

# Unterstützte Plattformen
docker manifest inspect ghcr.io/xerolux/modbridge:latest
```

---

## Secrets und Permissions

### Benötigte Secrets

**GITHUB_TOKEN**:
- ✅ Automatisch verfügbar
- Keine Konfiguration nötig
- Wird für folgendes verwendet:
  - GitHub Releases erstellen
  - Docker Images zu ghcr.io pushen
  - Code scannen

### Permissions

Die Workflows benötigen folgende Permissions (bereits konfiguriert):

**release.yml**:
- `contents: write` - Release erstellen
- `packages: write` - Docker Images pushen

**docker-publish.yml**:
- `contents: read` - Code auschecken
- `packages: write` - Docker Images pushen

**ci.yml**:
- `contents: read` - Code auschecken

---

## Lokal testen

### Release-Workflow lokal simulieren

```bash
# .deb Pakete bauen
make deb-all

# Docker Image bauen
docker build -t modbus-proxy-manager:test .

# Multi-Arch Build (erfordert buildx)
docker buildx build --platform linux/amd64,linux/arm64 -t modbus-proxy-manager:test .
```

### CI-Workflow lokal simulieren

```bash
# Tests ausführen
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Lint
golangci-lint run --timeout=5m

# Build
go build -v -o modbusmanager ./main.go
```

---

## Workflow-Logs ansehen

1. **GitHub Actions Tab öffnen**:
   `https://github.com/Xerolux/modbridge/actions`

2. **Workflow auswählen**:
   - Release
   - Docker
   - CI

3. **Run auswählen**: Klicke auf einen spezifischen Workflow-Run

4. **Logs ansehen**: Klicke auf einzelne Steps

---

## Troubleshooting

### Docker Push schlägt fehl

**Problem**: `permission denied while trying to connect to the Docker daemon socket`

**Lösung**:
- Permissions in Repository-Settings überprüfen
- `packages: write` Permission aktivieren

---

### .deb Paket-Build schlägt fehl

**Problem**: `make: *** [deb-all] Error 1`

**Lösung**:
1. `version.txt` existiert
2. Alle DEBIAN-Scripts sind vorhanden
3. dpkg-deb ist installiert (Ubuntu/Debian Runner)

---

### Release wird nicht erstellt

**Problem**: Tag wurde gepusht, aber kein Release

**Lösung**:
1. Tag-Format prüfen: `v*` (z.B. `v0.1.0`)
2. Workflow-Logs prüfen
3. Permissions überprüfen

---

## Best Practices

### Version Tagging

**Format**: `vMAJOR.MINOR.PATCH`

**Beispiele**:
- ✅ `v0.1.0` - Korrektes Format
- ✅ `v1.0.0` - Major Release
- ✅ `v0.2.1` - Patch Release
- ❌ `0.1.0` - Fehlt 'v' Prefix
- ❌ `v0.1` - Fehlt Patch-Version

### Commit Messages für Releases

```bash
# Gute Commit-Message
git commit -m "Release v0.2.0: Add feature X, fix bug Y"

# Release-Tag
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

### Docker Image Tags

- **Stabil**: Verwende `latest` oder spezifische Version (z.B. `v0.1.0`)
- **Testing**: Verwende `edge` oder `main`
- **Development**: Verwende Branch-Namen

---

## Weitere Ressourcen

- **GitHub Actions Docs**: https://docs.github.com/actions
- **Docker Buildx**: https://docs.docker.com/buildx/
- **GitHub Container Registry**: https://docs.github.com/packages/working-with-a-github-packages-registry/working-with-the-container-registry

---

**Version**: 0.1.0
**Letzte Aktualisierung**: 31. Dezember 2025
