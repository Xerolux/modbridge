// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig drops a config.json with the given body into a temp dir.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestLoadKeepsDefaultsForAbsentKeys(t *testing.T) {
	// A minimal config that only sets the web port must not reset everything
	// else to the zero value.
	m := NewManager(writeConfig(t, `{"web_port": ":9999"}`))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	cfg := m.Get()
	defaults := DefaultConfig()

	if cfg.WebPort != ":9999" {
		t.Errorf("WebPort = %q, want %q", cfg.WebPort, ":9999")
	}
	if cfg.LogMaxSize != defaults.LogMaxSize {
		t.Errorf("LogMaxSize = %d, want default %d", cfg.LogMaxSize, defaults.LogMaxSize)
	}
	if cfg.RateLimitRequests != defaults.RateLimitRequests {
		t.Errorf("RateLimitRequests = %d, want default %d", cfg.RateLimitRequests, defaults.RateLimitRequests)
	}
	if cfg.SessionTimeout != defaults.SessionTimeout {
		t.Errorf("SessionTimeout = %d, want default %d", cfg.SessionTimeout, defaults.SessionTimeout)
	}
	if cfg.MaxConnections != defaults.MaxConnections {
		t.Errorf("MaxConnections = %d, want default %d", cfg.MaxConnections, defaults.MaxConnections)
	}
	if cfg.MetricsPort != defaults.MetricsPort {
		t.Errorf("MetricsPort = %q, want default %q", cfg.MetricsPort, defaults.MetricsPort)
	}
	if !cfg.RateLimitEnabled {
		t.Error("RateLimitEnabled = false, want the default true")
	}
	if !cfg.MultiUser {
		t.Error("MultiUser = false, want the default true")
	}
}

func TestLoadHonorsExplicitFalse(t *testing.T) {
	// An explicitly disabled bool must survive the defaults.
	m := NewManager(writeConfig(t, `{"multi_user": false, "rate_limit_enabled": false, "metrics_enabled": false}`))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	cfg := m.Get()
	if cfg.MultiUser {
		t.Error("MultiUser = true, want the explicit false")
	}
	if cfg.RateLimitEnabled {
		t.Error("RateLimitEnabled = true, want the explicit false")
	}
	if cfg.MetricsEnabled {
		t.Error("MetricsEnabled = true, want the explicit false")
	}
}

func TestLoadHonorsExplicitValues(t *testing.T) {
	m := NewManager(writeConfig(t, `{"log_max_size": 5, "session_timeout": 1, "cors_allowed_origins": ["https://example.com"]}`))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	cfg := m.Get()
	if cfg.LogMaxSize != 5 {
		t.Errorf("LogMaxSize = %d, want 5", cfg.LogMaxSize)
	}
	if cfg.SessionTimeout != 1 {
		t.Errorf("SessionTimeout = %d, want 1", cfg.SessionTimeout)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "https://example.com" {
		t.Errorf("CORSAllowedOrigins = %v, want [https://example.com]", cfg.CORSAllowedOrigins)
	}
}

func TestUpdateRejectsInvalidChange(t *testing.T) {
	m := NewManager(writeConfig(t, `{}`))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// A proxy without addresses is invalid and must not be persisted.
	err := m.Update(func(c *Config) error {
		c.Proxies = append(c.Proxies, ProxyConfig{ID: "1", Name: "Broken"})
		return nil
	})
	if err == nil {
		t.Fatal("Update accepted an invalid configuration, want an error")
	}
	if len(m.Get().Proxies) != 0 {
		t.Errorf("invalid proxy was stored anyway: %+v", m.Get().Proxies)
	}
}

func TestUpdateAcceptsValidChange(t *testing.T) {
	m := NewManager(writeConfig(t, `{}`))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	err := m.Update(func(c *Config) error {
		c.Proxies = append(c.Proxies, ProxyConfig{
			ID:         "1",
			Name:       "Valid",
			ListenAddr: ":5020",
			TargetAddr: "192.0.2.10:502",
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Update rejected a valid configuration: %v", err)
	}
	if len(m.Get().Proxies) != 1 {
		t.Errorf("valid proxy was not stored")
	}
}
