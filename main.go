package main

import (
	"context"
	"log"
	"modbusproxy/pkg/api"
	"modbusproxy/pkg/auth"
	"modbusproxy/pkg/config"
	"modbusproxy/pkg/logger"
	"modbusproxy/pkg/manager"
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

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
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
