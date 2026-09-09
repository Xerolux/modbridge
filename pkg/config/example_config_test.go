// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package config

import "testing"

// TestExampleConfigIsValid keeps config.example.json honest: it is the file
// operators copy, so a key that no longer matches the struct has to fail here
// rather than at their first start.
func TestExampleConfigIsValid(t *testing.T) {
	m := NewManager("../../config.example.json")
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("config.example.json is not valid: %v", err)
	}

	cfg := m.Get()
	if len(cfg.Proxies) != 1 {
		t.Fatalf("expected 1 example proxy, got %d", len(cfg.Proxies))
	}
	if cfg.Proxies[0].ListenAddr == "" || cfg.Proxies[0].TargetAddr == "" {
		t.Error("example proxy is missing listen_addr or target_addr")
	}
	if cfg.WebPort != ":8080" {
		t.Errorf("WebPort = %q, want \":8080\"", cfg.WebPort)
	}
}
