# Development & CI/CD

## Makefile commands

```bash
make help           # Show all available commands
```

| Command | Description |
|---------|-------------|
| `make build` | Build the frontend and compile the Go binary |
| `make build-frontend` | Build only the frontend and copy it to pkg/web/dist/ |
| `make build-all` | Binaries for all platforms (Linux, Windows, macOS) |
| `make run` | Build the binary and start it directly |
| `make test` | Run tests with race detector and coverage |
| `make coverage` | Generate a coverage report as HTML (`coverage.html`) |
| `make lint` | Run the linter (requires `golangci-lint`) |
| `make fmt` | Format the code (`go fmt` + `gofmt`) |
| `make vet` | Run `go vet` |
| `make clean` | Remove build artifacts |
| `make deps` | Download and tidy Go dependencies |
| `make update-deps` | Update Go dependencies to the latest versions |
| `make install` | Install the binary system-wide (`go install`) |
| `make dev` | Development mode with live reload (requires `air`) |
| `make docker-build` | Build the Docker image locally |
| `make docker-run` | Build the Docker image and start the container |
| `make docker-stop` | Stop the Docker container |
| `make docker-logs` | Show Docker container logs |

**Examples:**
```bash
make build          # full build
make test           # run tests
make coverage       # check test coverage
make build-all      # build for all platforms
```

---

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEB_PORT` | `:8080` | Web UI port (overrides `web_port` from config.json) |
| `LOG_LEVEL` | `INFO` | Log level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `TZ` | `UTC` | Timezone for the container |

For Docker deployments: copy `.env.example` to `.env` and adjust the values.

```bash
cp .env.example .env
# edit .env
docker-compose up -d
```

---


## Automated build (GitHub Actions)

The project uses GitHub Actions for automated builds and releases:

1. Frontend is built with Node.js 22
2. Go binaries for Linux (AMD64/ARM64) and Windows (AMD64)
3. Docker images with multi-arch support
4. Automatic releases on tags (`v*`)

**Create a release:**
```bash
git tag v1.0.0
git push origin v1.0.0
```

**Required GitHub secrets for Docker push:**
```
DOCKER_USERNAME   = Docker Hub username
DOCKER_PASSWORD   = Docker Hub access token
```

---
