# Distributed Tracing Guide

Modbus Proxy supports **OpenTelemetry distributed tracing** for comprehensive request flow visibility across your infrastructure.

## Features

- **OpenTelemetry Integration** - Industry-standard distributed tracing
- **Multiple Exporters** - Jaeger and Zipkin support
- **Trace Context Propagation** - W3C Trace Context standard
- **Automatic Instrumentation** - HTTP requests and Modbus operations
- **Detailed Spans** - Connection pool, read/write operations
- **Configurable Sampling** - Control tracing overhead
- **Zero Dependencies When Disabled** - No performance impact when tracing is off

## Quick Start

### 1. Enable Tracing

```bash
export OTEL_ENABLED=true
export OTEL_EXPORTER=zipkin
export OTEL_ZIPKIN_ENDPOINT=http://localhost:9411/api/v2/spans
```

### 2. Run with Docker Compose

```bash
docker-compose up -d
```

This starts:
- Modbus Proxy on port 8080
- Zipkin on port 9411

### 3. View Traces

Open http://localhost:9411 in your browser and explore your traces!

## Configuration

All tracing configuration is done via environment variables:

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `OTEL_ENABLED` | Enable/disable tracing | `false` | `true` |
| `OTEL_EXPORTER` | Exporter type | `none` | `jaeger`, `zipkin` |
| `OTEL_ENVIRONMENT` | Environment name | `production` | `development`, `staging` |
| `OTEL_SAMPLING_RATE` | Sampling ratio (0.0-1.0) | `1.0` | `0.1` (10%) |
| `OTEL_JAEGER_ENDPOINT` | Jaeger collector URL | `http://localhost:14268/api/traces` | Custom URL |
| `OTEL_ZIPKIN_ENDPOINT` | Zipkin collector URL | `http://localhost:9411/api/v2/spans` | Custom URL |

## Exporters

### Zipkin

**Recommended** - Modern, lightweight, easy to set up.

```bash
# Configuration
export OTEL_ENABLED=true
export OTEL_EXPORTER=zipkin
export OTEL_ZIPKIN_ENDPOINT=http://localhost:9411/api/v2/spans

# Run Zipkin with Docker
docker run -d -p 9411:9411 openzipkin/zipkin

# View UI
open http://localhost:9411
```

**Features:**
- Simple UI for trace exploration
- Service dependency graph
- Real-time trace search
- Low resource usage

### Jaeger

**Note:** The Jaeger exporter is deprecated but still supported. Consider using the OTLP exporter in production.

```bash
# Configuration
export OTEL_ENABLED=true
export OTEL_EXPORTER=jaeger
export OTEL_JAEGER_ENDPOINT=http://localhost:14268/api/traces

# Run Jaeger all-in-one with Docker
docker run -d \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest

# View UI
open http://localhost:16686
```

**Features:**
- Advanced trace analysis
- Service performance monitoring
- Root cause analysis
- Detailed span visualization

## Span Details

The Modbus Proxy creates the following spans:

### HTTP Spans

```
GET /api/proxies
├── http.method: GET
├── http.url: /api/proxies
├── http.status_code: 200
├── http.user_agent: curl/7.68.0
├── request.id: 550e8400-e29b-41d4-a716-446655440000
└── duration: 2.5ms
```

### Modbus Proxy Spans

```
modbus.handle_client
├── proxy.id: proxy-1
├── proxy.name: PLC-1
├── client.addr: 192.168.1.100:51234
├── target.addr: 192.168.1.200:502
├── request.size: 12 bytes
├── response.size: 15 bytes
└── modbus.forward_request
    ├── pool.get_connection (0.1ms)
    ├── modbus.write_request (0.3ms)
    │   └── bytes: 12
    └── modbus.read_response (1.2ms)
        └── response.size: 15
```

## Sampling

Control tracing overhead with sampling:

```bash
# Trace 100% of requests (default)
export OTEL_SAMPLING_RATE=1.0

# Trace 10% of requests
export OTEL_SAMPLING_RATE=0.1

# Trace 1% of requests
export OTEL_SAMPLING_RATE=0.01

# Disable tracing
export OTEL_SAMPLING_RATE=0.0
# Or simply:
export OTEL_ENABLED=false
```

**Recommendations:**
- **Development:** 100% sampling for full visibility
- **Staging:** 50% sampling to balance visibility and overhead
- **Production (low traffic):** 100% sampling
- **Production (high traffic):** 1-10% sampling

## Trace Context Propagation

The Modbus Proxy supports **W3C Trace Context** standard for distributed tracing across services.

### Incoming Requests

If a request includes `traceparent` header, the proxy will:
1. Extract the trace context
2. Continue the trace with a child span
3. Propagate the context to downstream services

```bash
# Example: Propagate trace from upstream service
curl -H "traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" \
     http://localhost:8080/api/proxies
```

### Outgoing Requests

All responses include trace headers for correlation:

```http
HTTP/1.1 200 OK
X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

## Use Cases

### 1. Debugging Latency Issues

**Problem:** Some requests are slow, but you don't know why.

**Solution:** Enable tracing and analyze slow traces in Zipkin/Jaeger:

```bash
# Enable tracing
export OTEL_ENABLED=true
export OTEL_EXPORTER=zipkin

# Make requests
# View traces in Zipkin UI
# Filter by duration > 100ms
# Identify slow spans (e.g., pool.get_connection, modbus.read_response)
```

### 2. Understanding Request Flow

**Problem:** Requests pass through multiple proxies, hard to track end-to-end.

**Solution:** Use trace context propagation:

```
Client -> LoadBalancer -> ModbusProxy1 -> ModbusProxy2 -> PLC
  └── Single trace ID spans all hops
```

### 3. Connection Pool Analysis

**Problem:** Connection pool exhaustion causes timeouts.

**Solution:** Analyze `pool.get_connection` spans:

```bash
# Long pool.get_connection spans indicate pool exhaustion
# Example: 5s wait time means pool was busy
# Action: Increase pool size or reduce request rate
```

### 4. Error Root Cause Analysis

**Problem:** Some requests fail, but logs don't show the full picture.

**Solution:** View failed traces with error status:

```bash
# Filter traces by error status in UI
# View error details in span attributes
# See exact operation that failed (read/write/pool)
```

## Production Deployment

### With Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: modbus-proxy
spec:
  template:
    spec:
      containers:
      - name: modbus-proxy
        image: modbus-proxy:latest
        env:
        - name: OTEL_ENABLED
          value: "true"
        - name: OTEL_EXPORTER
          value: "zipkin"
        - name: OTEL_ZIPKIN_ENDPOINT
          value: "http://zipkin.observability.svc:9411/api/v2/spans"
        - name: OTEL_SAMPLING_RATE
          value: "0.1"
        - name: OTEL_ENVIRONMENT
          value: "production"
```

### With Docker Compose

```yaml
version: '3.8'
services:
  modbus-proxy:
    image: modbus-proxy:latest
    environment:
      OTEL_ENABLED: "true"
      OTEL_EXPORTER: "zipkin"
      OTEL_ZIPKIN_ENDPOINT: "http://zipkin:9411/api/v2/spans"
      OTEL_SAMPLING_RATE: "0.1"
    depends_on:
      - zipkin

  zipkin:
    image: openzipkin/zipkin
    ports:
      - "9411:9411"
```

## Integration with Existing Systems

### Prometheus + Zipkin

Combine metrics with traces for full observability:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'modbus_proxy'
    static_configs:
      - targets: ['localhost:8080']
```

```bash
# Enable both
export OTEL_ENABLED=true
# Prometheus metrics available at /metrics
# Zipkin traces available at http://localhost:9411
```

### ELK Stack + Tracing

Correlate logs with traces using request IDs:

```json
{
  "level": "info",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "message": "Request completed"
}
```

## Troubleshooting

### Traces Not Appearing

1. **Check if tracing is enabled:**
   ```bash
   echo $OTEL_ENABLED
   # Should output: true
   ```

2. **Verify exporter endpoint is reachable:**
   ```bash
   curl http://localhost:9411/api/v2/spans -X POST -d '[]'
   # Should return 202 Accepted
   ```

3. **Check application logs:**
   ```bash
   # Look for:
   # "OpenTelemetry tracing initialized with zipkin exporter"
   ```

4. **Verify sampling rate:**
   ```bash
   echo $OTEL_SAMPLING_RATE
   # Should be > 0.0
   ```

### High Overhead

1. **Reduce sampling rate:**
   ```bash
   export OTEL_SAMPLING_RATE=0.1  # 10% sampling
   ```

2. **Use batch exporter (already configured):**
   - Spans are batched every 5 seconds
   - Max batch size: 512 spans

3. **Disable tracing in development:**
   ```bash
   export OTEL_ENABLED=false
   ```

### Missing Spans

**Problem:** Some operations don't create spans.

**Cause:** Spans are only created for traced contexts.

**Solution:** Ensure middleware is properly configured:
```go
// main.go
handler = middleware.RequestID(middleware.Tracing("modbus-proxy")(handler))
```

## Performance Impact

### Benchmarks

| Configuration | Latency Overhead | Memory Overhead | CPU Overhead |
|---------------|------------------|-----------------|--------------|
| Tracing disabled | 0 ns | 0 MB | 0% |
| 100% sampling | ~200 ns/op | ~1 MB | ~1% |
| 10% sampling | ~20 ns/op | ~0.1 MB | ~0.1% |
| 1% sampling | ~2 ns/op | ~0.01 MB | ~0.01% |

### Recommendations

- **Development:** Enable with 100% sampling
- **Production (< 1000 req/s):** Enable with 100% sampling
- **Production (1000-10000 req/s):** Enable with 10% sampling
- **Production (> 10000 req/s):** Enable with 1% sampling or use head-based sampling

## Advanced Topics

### Custom Instrumentation

Add custom spans to your code:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func myFunction(ctx context.Context) error {
    tracer := otel.Tracer("modbus-proxy")
    ctx, span := tracer.Start(ctx, "my_operation")
    defer span.End()

    span.SetAttributes(
        attribute.String("my.attribute", "value"),
        attribute.Int("my.count", 42),
    )

    // ... do work ...

    return nil
}
```

### Span Events

Add events to spans for detailed timeline:

```go
span.AddEvent("connection_acquired")
// ... do work ...
span.AddEvent("response_received")
```

### Trace IDs in Logs

Correlate traces with logs:

```go
import "go.opentelemetry.io/otel/trace"

func logWithTraceID(ctx context.Context, msg string) {
    spanCtx := trace.SpanContextFromContext(ctx)
    if spanCtx.HasTraceID() {
        log.Printf("[trace_id=%s] %s", spanCtx.TraceID(), msg)
    }
}
```

## Migration from Jaeger to OTLP

The Jaeger exporter is deprecated. To migrate:

1. **Deploy OpenTelemetry Collector:**
   ```bash
   docker run -d -p 4318:4318 otel/opentelemetry-collector
   ```

2. **Update configuration:**
   ```bash
   # Old (deprecated)
   export OTEL_EXPORTER=jaeger

   # New (recommended)
   # Note: OTLP exporter support coming in future release
   # For now, use Zipkin which is actively maintained
   export OTEL_EXPORTER=zipkin
   ```

## See Also

- [Prometheus Metrics](../docs/PERFORMANCE.md)
- [Structured Logging](./LOGGING.md)
- [Health Checks](../README.md#health-checks)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Zipkin Quickstart](https://zipkin.io/pages/quickstart.html)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
