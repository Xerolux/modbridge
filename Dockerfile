# Build stage for frontend
FROM node:24-alpine AS frontend-builder

WORKDIR /frontend

# Copy frontend files
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Build stage for backend
FROM golang:1.26-alpine AS builder

# Install build dependencies including GCC for CGO/sqlite3
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev \
    sqlite-dev \
    curl \
    bash

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Copy frontend build to pkg/web/dist (this is what go:embed serves)
COPY --from=frontend-builder /frontend/dist ./pkg/web/dist

# Build the application. CGO is required for sqlite3.
RUN CGO_ENABLED=1 GOFLAGS=-trimpath \
    go build \
    -ldflags="-s -w -X main.Version=docker -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o modbridge .

# Final stage
FROM alpine:3.23

LABEL org.opencontainers.image.title="ModBridge" \
      org.opencontainers.image.description="Modbus TCP Proxy Manager" \
      org.opencontainers.image.source="https://github.com/Xerolux/modbridge" \
      org.opencontainers.image.licenses="MIT"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget \
    sqlite-libs \
    curl && \
    adduser -D -u 1000 -g appuser appuser

WORKDIR /app

# Copy binary from builder (frontend is already embedded)
COPY --from=builder /build/modbridge .

# Create directory for logs and config with correct permissions
RUN mkdir -p /app/data /app/logs && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose ports: 8080 = Web UI/API, 5020-5030 = Modbus proxy ports
EXPOSE 8080
EXPOSE 5020-5030

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

CMD ["./modbridge"]
