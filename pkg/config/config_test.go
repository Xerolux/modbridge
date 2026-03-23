// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package config

import (
	"os"
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager("test_config.json")
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.path != "test_config.json" {
		t.Errorf("Expected path 'test_config.json', got '%s'", mgr.path)
	}
}

func TestLoadSaveConfig(t *testing.T) {
	testFile := "test_config_temp.json"
	defer os.Remove(testFile)

	mgr := NewManager(testFile)

	// Test initial config
	cfg := mgr.Get()
	if cfg.WebPort != ":8080" {
		t.Errorf("Expected default WebPort ':8080', got '%s'", cfg.WebPort)
	}

	// Update and save
	err := mgr.Update(func(c *Config) error {
		c.WebPort = ":9090"
		c.AdminPassHash = "test_hash"
		c.Proxies = []ProxyConfig{
			{
				ID:         "test-1",
				Name:       "Test Proxy",
				ListenAddr: ":5020",
				TargetAddr: "localhost:502",
				Enabled:    true,
			},
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Load in new manager
	mgr2 := NewManager(testFile)
	err = mgr2.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	cfg2 := mgr2.Get()
	if cfg2.WebPort != ":9090" {
		t.Errorf("Expected WebPort ':9090', got '%s'", cfg2.WebPort)
	}
	if cfg2.AdminPassHash != "test_hash" {
		t.Errorf("Expected AdminPassHash 'test_hash', got '%s'", cfg2.AdminPassHash)
	}
	if len(cfg2.Proxies) != 1 {
		t.Errorf("Expected 1 proxy, got %d", len(cfg2.Proxies))
	}
}

func TestRollback(t *testing.T) {
	testFile := "test_rollback_temp.json"
	defer os.Remove(testFile)

	mgr := NewManager(testFile)

	// No rollback available initially.
	if mgr.CanRollback() {
		t.Fatal("CanRollback should be false before any Update")
	}
	if err := mgr.Rollback(); err == nil {
		t.Fatal("Rollback should return error when no snapshot exists")
	}

	// Apply first change.
	if err := mgr.Update(func(c *Config) error {
		c.WebPort = ":9090"
		return nil
	}); err != nil {
		t.Fatalf("first Update failed: %v", err)
	}

	// Rollback should now be available.
	if !mgr.CanRollback() {
		t.Fatal("CanRollback should be true after Update")
	}

	// Apply second change.
	if err := mgr.Update(func(c *Config) error {
		c.WebPort = ":7777"
		return nil
	}); err != nil {
		t.Fatalf("second Update failed: %v", err)
	}
	if mgr.Get().WebPort != ":7777" {
		t.Fatalf("expected :7777, got %s", mgr.Get().WebPort)
	}

	// Rollback should restore the first change value.
	if err := mgr.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	if mgr.Get().WebPort != ":9090" {
		t.Fatalf("after rollback expected :9090, got %s", mgr.Get().WebPort)
	}

	// After rollback, snapshot is consumed.
	if mgr.CanRollback() {
		t.Fatal("CanRollback should be false after Rollback consumes snapshot")
	}
}

func TestConfigDeepCopy(t *testing.T) {
	mgr := NewManager("test.json")

	err := mgr.Update(func(c *Config) error {
		c.Proxies = []ProxyConfig{
			{ID: "1", Name: "Test"},
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	cfg := mgr.Get()
	cfg.Proxies[0].Name = "Modified"

	// Original should not be affected
	cfg2 := mgr.Get()
	if cfg2.Proxies[0].Name != "Test" {
		t.Errorf("Deep copy failed, original was modified")
	}
}