#!/bin/sh
set -eu

mkdir -p "$MODBRIDGE_LOG_DIR"
chown appuser:appuser "$MODBRIDGE_DATA_DIR" "$MODBRIDGE_LOG_DIR"
chmod 0700 "$MODBRIDGE_DATA_DIR"
exec su-exec appuser:appuser /app/modbridge "$@"
