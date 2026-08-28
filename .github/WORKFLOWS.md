# GitHub Actions Workflows

This repository uses GitHub Actions for automated builds, tests, and releases.

---

## Workflow Overview

### 1. CI (Continuous Integration)
**File**: `.github/workflows/ci.yml`

**Triggers**:
- Push to `main`
- Pull requests to `main`
- Manual dispatch

**What it does**:
- **Code quality**: format check, `go vet`, linting
- **Tests**: runs all Go tests
- **Build**: compiles the binaries and Docker image
- **Auto-release**: bumps the version and tags releases from `main`

**Status**: ✅ Active

---

### 2. Release
**File**: `.github/workflows/release.yml`

**Triggers**:
- Push of a tag (e.g. `v0.1.0`)
- Manual dispatch

**What it does**:
1. **Tests**: runs all tests
2. **Binaries**: builds for all platforms:
   - Linux (AMD64, ARM64)
   - Windows (AMD64)
   - macOS (AMD64, ARM64)
3. **Docker image**: builds and pushes a multi-arch image to `ghcr.io`:
   - `ghcr.io/xerolux/modbridge:latest`
   - `ghcr.io/xerolux/modbridge:v0.1.0`
   - `ghcr.io/xerolux/modbridge:0.1`
   - `ghcr.io/xerolux/modbridge:0`
4. **GitHub Release**: creates a release with:
   - All binaries
   - Automatic release notes
   - Installation instructions

**Status**: ✅ Active

**Example usage**:
```bash
# Create a release
git tag v0.2.0
git push origin v0.2.0

# Wait for the workflow to finish (~5-10 minutes)
# The release is then available at:
# https://github.com/Xerolux/modbridge/releases/tag/v0.2.0
```

---

### 3. GitHub Pages
**File**: `.github/workflows/pages.yml`

**Triggers**:
- Push to `main`
- Manual dispatch

**What it does**:
- Builds and deploys the documentation site to GitHub Pages

**Status**: ✅ Active

---

### 4. Wiki Sync
**File**: `.github/workflows/wiki-sync.yml`

**Triggers**:
- Push to `main` with changes under `docs/**`
- Manual dispatch

**What it does**:
- Copies all markdown files from `docs/` to the GitHub Wiki

**Status**: ✅ Active

---

## Release Process

### Create a release (automatically)

1. **Prepare the version**:
   ```bash
   # Set the version in version.txt
   echo "0.2.0" > version.txt
   git add version.txt
   git commit -m "Bump version to 0.2.0"
   git push
   ```

2. **Create the tag**:
   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```

3. **Wait**: the workflow runs automatically (~5-10 minutes)

4. **Done**: the release is available at:
   - GitHub Releases: `https://github.com/Xerolux/modbridge/releases`
   - Docker: `ghcr.io/xerolux/modbridge:v0.2.0`

---

## Docker Image Registry

### GitHub Container Registry (ghcr.io)

**Public registry**: anyone can pull the images (no login required)

**Available images**:
```bash
# Stable releases
ghcr.io/xerolux/modbridge:latest        # newest version
ghcr.io/xerolux/modbridge:v0.1.0        # specific version
ghcr.io/xerolux/modbridge:0.1           # major.minor
ghcr.io/xerolux/modbridge:0             # major

# Development
ghcr.io/xerolux/modbridge:main          # main branch
```

**Architecture support**:
- ✅ AMD64 (x86_64)
- ✅ ARM64 (aarch64)

**Image details**:
```bash
# Show image info
docker image inspect ghcr.io/xerolux/modbridge:latest

# Supported platforms
docker manifest inspect ghcr.io/xerolux/modbridge:latest
```

---

## Secrets and Permissions

### Required secrets

**GITHUB_TOKEN**:
- ✅ Automatically available
- No configuration needed
- Used for:
  - Creating GitHub releases
  - Pushing Docker images to ghcr.io
  - Scanning code

### Permissions

The workflows require the following permissions (already configured):

**release.yml**:
- `contents: write` - create releases
- `packages: write` - push Docker images

**ci.yml**:
- `contents: read` - check out code

---

## Testing locally

### Simulate the release workflow locally

```bash
# Build the Docker image
docker build -t modbus-proxy-manager:test .

# Multi-arch build (requires buildx)
docker buildx build --platform linux/amd64,linux/arm64 -t modbus-proxy-manager:test .
```

### Simulate the CI workflow locally

```bash
# Run tests
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Lint
golangci-lint run --timeout=5m

# Build
go build -v -o modbridge .
```

---

## Viewing workflow logs

1. **Open the GitHub Actions tab**:
   `https://github.com/Xerolux/modbridge/actions`

2. **Select the workflow**:
   - Release
   - CI

3. **Select a run**: click a specific workflow run

4. **View logs**: click the individual steps

---

## Troubleshooting

### Docker push fails

**Problem**: `permission denied while trying to connect to the Docker daemon socket`

**Solution**:
- Check permissions in the repository settings
- Enable the `packages: write` permission

---

### Release is not created

**Problem**: the tag was pushed, but no release appeared

**Solution**:
1. Check the tag format: `v*` (e.g. `v0.1.0`)
2. Check the workflow logs
3. Verify the permissions

---

## Best Practices

### Version tagging

**Format**: `vMAJOR.MINOR.PATCH`

**Examples**:
- ✅ `v0.1.0` - correct format
- ✅ `v1.0.0` - major release
- ✅ `v0.2.1` - patch release
- ❌ `0.1.0` - missing 'v' prefix
- ❌ `v0.1` - missing patch version

### Commit messages for releases

```bash
# Good commit message
git commit -m "Release v0.2.0: Add feature X, fix bug Y"

# Release tag
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

### Docker image tags

- **Stable**: use `latest` or a specific version (e.g. `v0.1.0`)
- **Testing**: use `main`
- **Development**: use branch names

---

## Additional resources

- **GitHub Actions docs**: https://docs.github.com/actions
- **Docker Buildx**: https://docs.docker.com/buildx/
- **GitHub Container Registry**: https://docs.github.com/packages/working-with-a-github-packages-registry/working-with-the-container-registry

---

**Version**: 0.1.0
**Last updated**: August 2026
