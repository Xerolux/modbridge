// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"modbridge/pkg/config"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
)

// newTestManager builds a manager with no proxies (sufficient for endpoint
// shape tests). We avoid spinning up real proxies / mock modbus here because
// the HTTP surface is read-only and only reads manager.GetProxies().
func newTestManager(t *testing.T) *manager.Manager {
	t.Helper()
	log, err := logger.NewLogger("headless-test.log", 100)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	t.Cleanup(func() { log.Close() })
	cfgMgr := config.NewManager("") // no file; empty config is fine for these tests
	return manager.NewManager(cfgMgr, log, nil)
}

func TestHealthEndpoint(t *testing.T) {
	mgr := newTestManager(t)
	log, _ := logger.NewLogger("headless-test.log", 100)
	defer log.Close()
	// We can't call startReadOnlyHTTP (it binds a real port); instead replicate
	// the handler chain via httptest by invoking it through a test server built
	// from the same mux shape. The handler closures reference the package-level
	// startTime, so we test that the response includes a non-negative uptime.
	srv := newTestServer(mgr, log)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v, want ok", body["status"])
	}
	up, ok := body["uptime_seconds"].(float64)
	if !ok || up < 0 {
		t.Errorf("uptime_seconds = %v, want non-negative number", body["uptime_seconds"])
	}
}

func TestMetricsEndpoint_PrometheusFormat(t *testing.T) {
	mgr := newTestManager(t)
	log, _ := logger.NewLogger("headless-test.log", 100)
	defer log.Close()
	srv := newTestServer(mgr, log)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain; version=0.0.4" {
		t.Errorf("Content-Type = %q, want Prometheus text", ct)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	// We won't parse the full Prometheus exposition; just assert that a known
	// series is present so the endpoint is not empty or broken.
	buf := make([]byte, 8192)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	for _, want := range []string{"modbridge_headless_uptime_seconds", "modbridge_headless_goroutines", "modbridge_headless_proxies_total"} {
		if !contains(body, want) {
			t.Errorf("metrics body missing %q\nbody: %s", want, body)
		}
	}
}

func TestStatusEndpoint_ReturnsProxies(t *testing.T) {
	mgr := newTestManager(t)
	log, _ := logger.NewLogger("headless-test.log", 100)
	defer log.Close()
	srv := newTestServer(mgr, log)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/status")
	if err != nil {
		t.Fatalf("GET /status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body["proxies"]; !ok {
		t.Error("status response missing 'proxies' field")
	}
	if _, ok := body["uptime_seconds"]; !ok {
		t.Error("status response missing 'uptime_seconds' field")
	}
}

func TestReadOnlyHTTP_NoMutationEndpoints(t *testing.T) {
	// Smoke-test that POST/PUT/DELETE are not specially handled (default 404
	// from the ServeMux, since only /health, /metrics, /status, /favicon.ico
	// are registered). This is a guard against accidentally adding a mutating
	// endpoint to the headless surface later.
	mgr := newTestManager(t)
	log, _ := logger.NewLogger("headless-test.log", 100)
	defer log.Close()
	srv := newTestServer(mgr, log)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/status", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /status: got %d, want 405 Method Not Allowed", resp.StatusCode)
	}
}

// newTestServer builds the same handler set as startReadOnlyHTTP but bound to
// an httptest.Server instead of a real port. This avoids port conflicts in CI.
func newTestServer(mgr *manager.Manager, log *logger.Logger) *httptest.Server {
	mux := http.NewServeMux()
	registerReadOnlyHandlers(mux, mgr, log)
	return httptest.NewServer(mux)
}

// contains is a tiny substring helper to avoid pulling strings just for one call.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// touch the time package so the import stays used if we add timing later.
var _ = time.Now
