# Structured Logging Guide

Modbus Proxy uses **zerolog** for high-performance structured logging with multiple log levels and contextual fields.

## Features

- **Structured JSON Logging** - Machine-readable logs for log aggregation systems
- **Multiple Log Levels** - DEBUG, INFO, WARN, ERROR, FATAL
- **Contextual Fields** - Add structured data to log entries
- **Request Tracing** - Correlation IDs for distributed tracing
- **Pretty Console Output** - Human-readable logs for development
- **Zero Allocation** - Extremely fast logging with minimal overhead
- **Dual Output** - Logs to both file and stdout

## Log Levels

| Level | Description | Use Case |
|-------|-------------|----------|
| `DEBUG` | Detailed diagnostic information | Development, troubleshooting |
| `INFO` | General informational messages | Normal operations, state changes |
| `WARN` | Warning messages, potential issues | Degraded performance, retries |
| `ERROR` | Error messages, operation failures | Failed operations, exceptions |
| `FATAL` | Critical errors, application exits | Unrecoverable errors |

## Configuration

Set log level via environment variable:

```bash
export LOG_LEVEL=debug    # Development
export LOG_LEVEL=info     # Production (default)
export LOG_LEVEL=warn     # Production (quiet)
export LOG_LEVEL=error    # Production (errors only)
```

Enable pretty console output for development:

```bash
export LOG_PRETTY=true    # Pretty console output
export LOG_PRETTY=false   # JSON output (default)
```

## Usage Examples

### Basic Logging

```go
import "modbusproxy/pkg/logger"

// Create structured logger
slog, err := logger.NewStructuredLogger("app.log", "info", false)
if err != nil {
    log.Fatal(err)
}
defer slog.Close()

// Simple messages
slog.Info("Application started")
slog.Debug("Processing request")
slog.Warn("Connection pool nearly exhausted")
slog.Error("Failed to connect", err)
```

### Contextual Fields

```go
// Add fields to log entries
slog.Info("User login", map[string]interface{}{
    "user_id": "12345",
    "ip": "192.168.1.100",
    "method": "oauth",
})

// Output:
// {"level":"info","time":"2025-12-23T21:00:00Z","service":"modbus-proxy","user_id":"12345","ip":"192.168.1.100","method":"oauth","message":"User login"}
```

### Proxy-Specific Logging

```go
// Create proxy-scoped logger
proxyLog := slog.WithProxyID("proxy-1")
proxyLog.Info().Msg("Proxy started")
proxyLog.Warn().
    Int("connections", 95).
    Int("max_connections", 100).
    Msg("Connection pool nearly full")

// Output includes proxy_id automatically
// {"level":"info","proxy_id":"proxy-1","message":"Proxy started"}
```

### Request Tracing

```go
import "modbusproxy/pkg/middleware"

// Middleware adds request_id to all requests
mux := http.NewServeMux()
handler := middleware.RequestID(mux)

// In handler, get request ID from context
requestID := middleware.GetRequestID(r.Context())
reqLog := slog.WithRequestID(requestID)
reqLog.Info().Msg("Processing request")

// All logs for this request share the same request_id
// {"level":"info","request_id":"550e8400-e29b-41d4-a716-446655440000","message":"Processing request"}
```

### HTTP Request Logging

```go
import (
    "modbusproxy/pkg/middleware"
    "github.com/rs/zerolog"
)

zlog := zerolog.New(os.Stdout).With().Timestamp().Logger()

// Add logging middleware
mux := http.NewServeMux()
handler := middleware.RequestID(
    middleware.Logging(&zlog)(mux),
)

// Automatic request/response logging
// {"level":"info","request_id":"...","method":"GET","path":"/api/health","status":200,"duration":2.5,"bytes":45,"message":"Request completed"}
```

### Error Logging

```go
// Log errors with full context
err := proxy.Connect()
if err != nil {
    slog.Error("Failed to connect to target", err, map[string]interface{}{
        "proxy_id": "proxy-1",
        "target": "192.168.1.100:502",
        "retry_count": 3,
    })
}

// Output:
// {"level":"error","error":"connection refused","proxy_id":"proxy-1","target":"192.168.1.100:502","retry_count":3,"message":"Failed to connect to target"}
```

### Performance Monitoring

```go
import "time"

start := time.Now()
// ... do work ...
duration := time.Since(start)

slog.Info("Operation completed", map[string]interface{}{
    "operation": "modbus_read",
    "duration_ms": duration.Milliseconds(),
    "bytes_read": 256,
    "success": true,
})
```

## Log Format

### JSON Format (Production)

```json
{
  "level": "info",
  "service": "modbus-proxy",
  "time": "2025-12-23T21:00:00Z",
  "proxy_id": "proxy-1",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/api/proxies",
  "status": 200,
  "duration": 2.5,
  "bytes": 1024,
  "message": "Request completed"
}
```

### Pretty Format (Development)

```
2025-12-23T21:00:00Z INF Request completed
    service=modbus-proxy
    proxy_id=proxy-1
    request_id=550e8400-e29b-41d4-a716-446655440000
    method=GET
    path=/api/proxies
    status=200
    duration=2.5ms
    bytes=1024
```

## Log Aggregation

### Elasticsearch / ELK Stack

```bash
# Use filebeat to ship logs to Elasticsearch
filebeat -e -c filebeat.yml
```

**filebeat.yml:**
```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/modbus-proxy/*.log
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "modbus-proxy-%{+yyyy.MM.dd}"
```

### Fluentd

```bash
# Use fluentd to forward logs
fluentd -c fluent.conf
```

**fluent.conf:**
```
<source>
  @type tail
  path /var/log/modbus-proxy/*.log
  pos_file /var/log/td-agent/modbus-proxy.pos
  tag modbus.proxy
  format json
</source>

<match modbus.**>
  @type elasticsearch
  host localhost
  port 9200
  index_name modbus-proxy
  type_name _doc
</match>
```

### Grafana Loki

```yaml
# promtail config for Loki
server:
  http_listen_port: 9080

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: modbus-proxy
    static_configs:
      - targets:
          - localhost
        labels:
          job: modbus-proxy
          __path__: /var/log/modbus-proxy/*.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            proxy_id: proxy_id
            request_id: request_id
      - labels:
          level:
          proxy_id:
```

## Querying Logs

### Search by Request ID (trace all logs for a request)

```bash
# Grep JSON logs
grep '"request_id":"550e8400"' app.log | jq .

# Elasticsearch query
GET /modbus-proxy-*/_search
{
  "query": {
    "term": { "request_id": "550e8400-e29b-41d4-a716-446655440000" }
  }
}

# Loki query
{request_id="550e8400-e29b-41d4-a716-446655440000"}
```

### Search by Proxy

```bash
# Find all logs for a specific proxy
grep '"proxy_id":"proxy-1"' app.log | jq .

# Elasticsearch
GET /modbus-proxy-*/_search
{
  "query": {
    "term": { "proxy_id": "proxy-1" }
  }
}
```

### Search Errors

```bash
# Find all errors
grep '"level":"error"' app.log | jq .

# Elasticsearch - errors in last hour
GET /modbus-proxy-*/_search
{
  "query": {
    "bool": {
      "must": [
        { "term": { "level": "error" } },
        { "range": { "time": { "gte": "now-1h" } } }
      ]
    }
  }
}
```

## Best Practices

1. **Use Appropriate Log Levels**
   - DEBUG: Detailed flow, variable values
   - INFO: State changes, lifecycle events
   - WARN: Recoverable issues, retries
   - ERROR: Operation failures, exceptions

2. **Add Context**
   - Always include relevant IDs (proxy_id, device_id, request_id)
   - Add error details and error codes
   - Include timing information for performance issues

3. **Avoid Sensitive Data**
   - Never log passwords, tokens, or API keys
   - Mask or hash PII (IP addresses, device IDs)
   - Sanitize user input before logging

4. **Use Structured Fields**
   - Prefer structured fields over string interpolation
   - Bad: `fmt.Sprintf("User %s logged in from %s", user, ip)`
   - Good: `slog.Info("User login", map[string]interface{}{"user": user, "ip": ip})`

5. **Performance Considerations**
   - Use DEBUG level for high-frequency logs
   - Avoid logging in tight loops
   - Use sampling for high-volume events

## Dynamic Log Level Changes

Change log level at runtime:

```go
// Via API endpoint
POST /api/admin/log-level
{
  "level": "debug"
}

// Programmatically
slog.SetLevel("debug")

// Via signal (SIGUSR1 to increase, SIGUSR2 to decrease)
kill -SIGUSR1 <pid>  # debug -> info -> warn -> error
kill -SIGUSR2 <pid>  # error -> warn -> info -> debug
```

## Troubleshooting

### High Log Volume

```bash
# Rotate logs daily
logrotate -f /etc/logrotate.d/modbus-proxy

# Compress old logs
find /var/log/modbus-proxy -name "*.log.*" -exec gzip {} \;

# Clean up old logs (keep 30 days)
find /var/log/modbus-proxy -name "*.log.*.gz" -mtime +30 -delete
```

### Performance Impact

Zerolog is designed for zero-allocation logging:

```
BenchmarkZerolog-8    20000000    85.7 ns/op    0 B/op    0 allocs/op
```

For high-throughput scenarios, consider:
- Reduce log level (INFO or WARN in production)
- Use async logging (log to buffered channel)
- Sample high-frequency events

### Log Correlation

To trace a request across multiple services:

1. Propagate `X-Request-ID` header
2. Include request_id in all logs
3. Use distributed tracing (OpenTelemetry) for complex flows

Example:
```go
// Service A
requestID := uuid.New().String()
req.Header.Set("X-Request-ID", requestID)

// Service B
requestID := req.Header.Get("X-Request-ID")
ctx := context.WithValue(ctx, RequestIDKey, requestID)
```

## See Also

- [Zerolog Documentation](https://github.com/rs/zerolog)
- [Prometheus Metrics](../PERFORMANCE.md)
- [Distributed Tracing](./TRACING.md)
- [Health Checks](../README.md#health-checks)
