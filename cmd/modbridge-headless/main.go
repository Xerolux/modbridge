// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

// Package main is the entry point for the headless ModBridge variant: a
// stripped-down binary that runs proxies from a config file with NO Web UI,
// NO REST API mutations, NO authentication, and NO database. It is intended
// for resource-constrained deployments, sidecars, and pure forwarding tasks.
//
// An optional read-only HTTP surface (default on) exposes /health, /metrics
// (Prometheus), and /status (per-proxy stats) so the headless instance can be
// monitored, health-checked, and scraped. None of these endpoints mutate state
// or expose credentials. Disable with -no-http.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"modbridge/pkg/config"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
	"modbridge/pkg/metrics"
)

var version = "headless-dev"

// startTime records process start for uptime reporting in /health and /status.
var startTime = time.Now()

func main() {
	configFile := flag.String("config", "config.json", "Path to configuration file")
	verbose := flag.Bool("v", false, "Verbose logging (debug level)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	httpAddr := flag.String("http-addr", "", "Override the read-only HTTP listen address (defaults to config web_port)")
	noHTTP := flag.Bool("no-http", false, "Disable the read-only HTTP server (no /health, /metrics, /status)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("modbridge-headless %s\n", version)
		return
	}

	log.Printf("ModBridge Headless v%s", version)
	log.Printf("Config: %s", *configFile)

	cfgMgr := config.NewManager(*configFile)
	if err := cfgMgr.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfg := cfgMgr.Get()
	if len(cfg.Proxies) == 0 {
		log.Fatal("No proxies defined in config. Exiting.")
	}

	logLevel := "INFO"
	if *verbose {
		logLevel = "DEBUG"
	}
	_ = logLevel

	l, err := logger.NewLogger("proxy.log", 1000)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer l.Close()

	mgr := manager.NewManager(cfgMgr, l, nil)

	log.Printf("Starting %d proxy(ies) from config...", len(cfg.Proxies))
	l.Info("HEADLESS", fmt.Sprintf("Initializing %d proxies from %s", len(cfg.Proxies), *configFile))

	mgr.Initialize()

	proxies := mgr.GetProxies()
	running := 0
	for _, p := range proxies {
		if status, ok := p["status"].(string); ok && status == "Running" {
			running++
		}
	}
	log.Printf("%d/%d proxies running", running, len(proxies))
	l.Info("HEADLESS", fmt.Sprintf("%d/%d proxies running", running, len(proxies)))

	// Read-only HTTP surface for monitoring. Default listen address is the
	// config's web_port; override with -http-addr. Set -no-http to disable.
	httpEnabled := !*noHTTP
	httpListen := *httpAddr
	if httpListen == "" {
		httpListen = cfg.WebPort
	}

	var httpSrv *http.Server
	if httpEnabled {
		httpSrv = startReadOnlyHTTP(httpListen, mgr, l)
		l.Info("HEADLESS", fmt.Sprintf("Read-only HTTP surface on %s (/health, /metrics, /status)", httpListen))
	} else {
		l.Info("HEADLESS", "HTTP surface disabled (-no-http)")
	}

	ready := make(chan os.Signal, 1)
	signal.Notify(ready, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ready

	log.Printf("Received %s, shutting down...", sig)
	l.Info("HEADLESS", fmt.Sprintf("Received signal %s, shutting down...", sig))

	if httpSrv != nil {
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := httpSrv.Shutdown(shutCtx); err != nil {
			l.Error("HEADLESS", fmt.Sprintf("HTTP shutdown error: %v", err))
		}
		shutCancel()
	}

	mgr.StopAll()

	log.Println("Stopped.")
	l.Info("HEADLESS", "All proxies stopped. Goodbye.")
}

// startReadOnlyHTTP launches a minimal HTTP server exposing three read-only
// endpoints. It returns immediately after starting the listener; the server
// runs in a goroutine and is stopped via Shutdown in main's signal handler.
//
// Endpoints:
//   - GET /health  -> {"status":"ok","uptime_seconds":N}  (200; for k8s liveness)
//   - GET /metrics -> Prometheus text exposition
//   - GET /status  -> JSON snapshot of all proxies (id, name, status, stats)
//
// No authentication is required. The surface exposes no secrets and supports
// no state mutations. If a deploy needs network-layer protection, bind to
// localhost or use a reverse proxy / firewall.
func startReadOnlyHTTP(addr string, mgr *manager.Manager, l *logger.Logger) *http.Server {
	mux := http.NewServeMux()
	registerReadOnlyHandlers(mux, mgr, l)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Error("HEADLESS", fmt.Sprintf("HTTP server error: %v", err))
		}
	}()
	return srv
}

// registerReadOnlyHandlers wires the /health, /metrics, /status, and
// /favicon.ico endpoints onto mux. Extracted from startReadOnlyHTTP so tests
// can attach the same handlers to an httptest.Server without binding a port.
//
// Endpoints:
//   - GET /health  -> {"status":"ok","uptime_seconds":N}  (200; for k8s liveness)
//   - GET /metrics -> Prometheus text exposition
//   - GET /status  -> JSON snapshot of all proxies (id, name, status, stats)
//
// No authentication, no mutations. The surface exposes no secrets. Each handler
// rejects non-GET methods with 405 so the surface cannot be abused for command
// channels via POST/PUT.
func registerReadOnlyHandlers(mux *http.ServeMux, mgr *manager.Manager, _ *logger.Logger) {
	// getOnly wraps h so that any non-GET method returns 405 Method Not Allowed.
	getOnly := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			h(w, r)
		}
	}

	mux.HandleFunc("/health", getOnly(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "ok",
			"version":        version,
			"uptime_seconds": int(time.Since(startTime).Seconds()),
		})
	}))

	// A single shared Metrics registry collects nothing automatically; proxies
	// report via their own stats. We expose a minimal process-level scrape so
	// Prometheus has something to pull even before per-proxy instrumentation
	// is wired. This keeps /metrics non-empty and useful out of the box.
	_ = metrics.NewMetrics()
	mux.HandleFunc("/metrics", getOnly(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		fmt.Fprintf(w, "# HELP modbridge_headless_uptime_seconds Process uptime in seconds.\n")
		fmt.Fprintf(w, "# TYPE modbridge_headless_uptime_seconds counter\n")
		fmt.Fprintf(w, "modbridge_headless_uptime_seconds %d\n", int(time.Since(startTime).Seconds()))
		fmt.Fprintf(w, "# HELP modbridge_headless_goroutines Current number of goroutines.\n")
		fmt.Fprintf(w, "# TYPE modbridge_headless_goroutines gauge\n")
		fmt.Fprintf(w, "modbridge_headless_goroutines %d\n", runtime.NumGoroutine())
		fmt.Fprintf(w, "# HELP modbridge_headless_alloc_bytes Heap allocation in bytes.\n")
		fmt.Fprintf(w, "# TYPE modbridge_headless_alloc_bytes gauge\n")
		fmt.Fprintf(w, "modbridge_headless_alloc_bytes %d\n", mem.Alloc)
		fmt.Fprintf(w, "# HELP modbridge_headless_proxies_total Total proxies configured.\n")
		fmt.Fprintf(w, "# TYPE modbridge_headless_proxies_total gauge\n")
		fmt.Fprintf(w, "modbridge_headless_proxies_total %d\n", len(mgr.GetProxies()))
	}))

	mux.HandleFunc("/status", getOnly(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"version":        version,
			"uptime_seconds": int(time.Since(startTime).Seconds()),
			"proxies":        mgr.GetProxies(),
		})
	}))

	// /favicon.ico: return 204 to keep browser probes out of the logs.
	mux.HandleFunc("/favicon.ico", getOnly(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}
