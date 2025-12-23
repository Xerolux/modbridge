package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// ExporterType defines the type of trace exporter to use.
type ExporterType string

const (
	// ExporterJaeger uses Jaeger exporter (deprecated but still supported).
	ExporterJaeger ExporterType = "jaeger"
	// ExporterZipkin uses Zipkin exporter.
	ExporterZipkin ExporterType = "zipkin"
	// ExporterNone disables tracing.
	ExporterNone ExporterType = "none"
)

// Config holds OpenTelemetry configuration.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	ExporterType   ExporterType

	// Jaeger configuration
	JaegerEndpoint string // e.g., "http://localhost:14268/api/traces"

	// Zipkin configuration
	ZipkinEndpoint string // e.g., "http://localhost:9411/api/v2/spans"

	// Sampling
	SamplingRate float64 // 0.0 to 1.0 (default: 1.0 = 100%)
}

// TracerProvider manages the OpenTelemetry tracer provider.
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	config   Config
}

// NewTracerProvider creates and initializes a new OpenTelemetry tracer provider.
func NewTracerProvider(cfg Config) (*TracerProvider, error) {
	// Set defaults
	if cfg.ServiceName == "" {
		cfg.ServiceName = "modbus-proxy"
	}
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "1.0.0"
	}
	if cfg.Environment == "" {
		cfg.Environment = "production"
	}
	if cfg.SamplingRate == 0 {
		cfg.SamplingRate = 1.0
	}

	// If tracing is disabled, return early
	if cfg.ExporterType == ExporterNone {
		return &TracerProvider{config: cfg}, nil
	}

	// Create exporter based on type
	var exporter sdktrace.SpanExporter
	var err error

	switch cfg.ExporterType {
	case ExporterJaeger:
		exporter, err = createJaegerExporter(cfg)
	case ExporterZipkin:
		exporter, err = createZipkinExporter(cfg)
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", cfg.ExporterType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if cfg.SamplingRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SamplingRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SamplingRate)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for distributed tracing
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return &TracerProvider{
		provider: tp,
		config:   cfg,
	}, nil
}

// createJaegerExporter creates a Jaeger exporter.
func createJaegerExporter(cfg Config) (sdktrace.SpanExporter, error) {
	endpoint := cfg.JaegerEndpoint
	if endpoint == "" {
		endpoint = "http://localhost:14268/api/traces"
	}

	return jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(endpoint),
		),
	)
}

// createZipkinExporter creates a Zipkin exporter.
func createZipkinExporter(cfg Config) (sdktrace.SpanExporter, error) {
	endpoint := cfg.ZipkinEndpoint
	if endpoint == "" {
		endpoint = "http://localhost:9411/api/v2/spans"
	}

	return zipkin.New(endpoint)
}

// Tracer returns a tracer with the given name.
func (tp *TracerProvider) Tracer(name string) trace.Tracer {
	if tp.provider == nil {
		return otel.Tracer(name)
	}
	return tp.provider.Tracer(name)
}

// Shutdown gracefully shuts down the tracer provider.
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}
	return tp.provider.Shutdown(ctx)
}

// GetTracer returns a tracer for the modbus-proxy service.
// This is a convenience function for getting a tracer with the service name.
func GetTracer() trace.Tracer {
	return otel.Tracer("modbus-proxy")
}
