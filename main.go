package main

import (
	"context"
	"fmt"
	"log"
	"modbusproxy/pkg/api"
	"modbusproxy/pkg/auth"
	"modbusproxy/pkg/config"
	"modbusproxy/pkg/logger"
	"modbusproxy/pkg/manager"
	"modbusproxy/pkg/middleware"
	"modbusproxy/pkg/tracing"
	"modbusproxy/pkg/web"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 0. Runtime Settings for better profiling
	runtime.SetBlockProfileRate(1)    // Enable blocking profile
	runtime.SetMutexProfileFraction(1) // Enable mutex profile

	// 1. Config
	cfgMgr := config.NewManager("config.json")
	if err := cfgMgr.Load(); err != nil {
		log.Printf("Starting with empty config: %v", err)
	}

	// 2. Logger
	l, err := logger.NewLogger("proxy.log", 1000)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer l.Close()

	// 2.5. OpenTelemetry Tracing
	// Configure via environment variables:
	//   OTEL_ENABLED=true/false (default: false)
	//   OTEL_EXPORTER=jaeger/zipkin/none (default: none)
	//   OTEL_JAEGER_ENDPOINT=http://localhost:14268/api/traces
	//   OTEL_ZIPKIN_ENDPOINT=http://localhost:9411/api/v2/spans
	//   OTEL_SAMPLING_RATE=0.0-1.0 (default: 1.0)
	var tracerProvider *tracing.TracerProvider
	if os.Getenv("OTEL_ENABLED") == "true" {
		exporterType := tracing.ExporterType(os.Getenv("OTEL_EXPORTER"))
		if exporterType == "" {
			exporterType = tracing.ExporterNone
		}

		tracingCfg := tracing.Config{
			ServiceName:    "modbus-proxy",
			ServiceVersion: "1.0.0",
			Environment:    getEnvOrDefault("OTEL_ENVIRONMENT", "production"),
			ExporterType:   exporterType,
			JaegerEndpoint: os.Getenv("OTEL_JAEGER_ENDPOINT"),
			ZipkinEndpoint: os.Getenv("OTEL_ZIPKIN_ENDPOINT"),
			SamplingRate:   getEnvFloat("OTEL_SAMPLING_RATE", 1.0),
		}

		tracerProvider, err = tracing.NewTracerProvider(tracingCfg)
		if err != nil {
			log.Printf("Warning: Failed to initialize tracing: %v", err)
		} else {
			l.Info("SYSTEM", "OpenTelemetry tracing initialized with "+string(exporterType)+" exporter")
			log.Printf("OpenTelemetry tracing initialized with %s exporter", exporterType)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tracerProvider.Shutdown(ctx); err != nil {
					log.Printf("Error shutting down tracer provider: %v", err)
				}
			}()
		}
	}

	// 3. Manager
	mgr := manager.NewManager(cfgMgr, l)
	mgr.Initialize()

	// 4. Auth
	authenticator := auth.NewAuthenticator()
	go authenticator.CleanupExpiredSessions()

	// 5. API Server
	apiServer := api.NewServer(cfgMgr, mgr, authenticator, l)

	// 6. Router
	mux := http.NewServeMux()

	// API Routes
	apiServer.Routes(mux)

	// Web Routes
	mux.Handle("/", web.Handler())

	// Profiling endpoints (available at /debug/pprof/)
	// Usage:
	//   go tool pprof http://localhost:8080/debug/pprof/heap
	//   go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

	// Prometheus metrics endpoint
	// Usage: curl http://localhost:8080/metrics
	//        Or configure Prometheus to scrape this endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Start server
	addr := cfgMgr.Get().WebPort
	if addr == "" {
		addr = ":8080"
	}
	// Override with environment variable if set
	if envPort := os.Getenv("WEB_PORT"); envPort != "" {
		addr = envPort
	}

	// Wrap handler with middleware (order matters: outermost first)
	var handler http.Handler = mux

	// Add OpenTelemetry tracing middleware if enabled
	if tracerProvider != nil {
		handler = middleware.Tracing("modbus-proxy")(handler)
	}

	// Add request ID middleware (always enabled)
	handler = middleware.RequestID(handler)

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	l.Info("SYSTEM", "Starting Modbus Manager on "+addr)
	log.Printf("Listening on %s", addr)

	// Run server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	l.Info("SYSTEM", "Shutting down server...")
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop all proxies
	mgr.StopAll()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	l.Info("SYSTEM", "Server stopped")
	log.Println("Server stopped")
}

// getEnvOrDefault returns the value of an environment variable or a default value if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvFloat returns a float64 value from an environment variable or a default value.
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err == nil {
			return f
		}
	}
	return defaultValue
}
