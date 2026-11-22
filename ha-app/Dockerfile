FROM alpine:3.24 AS source
ADD https://codeload.github.com/Xerolux/modbridge/tar.gz/b34bd3cb4e374e8336de5ee582956a5cecb03820 /tmp/modbridge.tar.gz
RUN mkdir /source && tar -xzf /tmp/modbridge.tar.gz -C /source --strip-components=1

FROM node:24-alpine AS frontend
WORKDIR /frontend
COPY --from=source /source/frontend/package*.json ./
RUN npm ci
COPY --from=source /source/frontend/ ./
RUN npm run build

FROM golang:1.27.4-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /source
COPY --from=source /source/go.mod /source/go.sum ./
RUN go mod download && go mod verify
COPY --from=source /source/ ./
RUN rm -rf pkg/web/dist
COPY --from=frontend /frontend/dist/ ./pkg/web/dist/
ARG BUILD_VERSION=3.0.0.10
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w -X main.Version=${BUILD_VERSION}" -o /modbridge .

FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata su-exec && adduser -D -u 1000 appuser
WORKDIR /app
COPY --from=builder /modbridge /app/modbridge
COPY run.sh /usr/local/bin/run-modbridge-ha
RUN chmod 0555 /usr/local/bin/run-modbridge-ha
ENV WEB_PORT=:8080 \
    MODBRIDGE_HA_INGRESS=1 \
    MODBRIDGE_DATA_DIR=/data/modbridge \
    MODBRIDGE_LOG_DIR=/data/modbridge/logs \
    MODBRIDGE_CONFIG=/data/modbridge/config.json
CMD ["/bin/sh", "/usr/local/bin/run-modbridge-ha"]
