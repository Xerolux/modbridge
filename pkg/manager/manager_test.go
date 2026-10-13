// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package manager

import (
	"net"
	"path/filepath"
	"testing"

	"modbridge/pkg/config"
	"modbridge/pkg/logger"
)

func TestNewManager(t *testing.T) {
	log, err := logger.NewLogger("test_logs", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	m := NewManager(nil, log, nil)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.proxies == nil {
		t.Error("proxies map not initialized")
	}

	if m.broadcaster == nil {
		t.Error("broadcaster not initialized")
	}
}

func TestAddProxy(t *testing.T) {
	log, err := logger.NewLogger("test_logs", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	cfgMgr := config.NewManager("test.json")
	m := NewManager(cfgMgr, log, nil)

	cfg := config.ProxyConfig{
		ID:         "test-id",
		Name:       "Test Proxy",
		ListenAddr: ":5020",
		TargetAddr: "192.168.1.100:502",
		Enabled:    true,
	}

	err = m.AddProxy(cfg, false)
	if err != nil {
		t.Errorf("AddProxy failed: %v", err)
	}

	if len(m.proxies) != 1 {
		t.Errorf("Expected 1 proxy, got %d", len(m.proxies))
	}

	p, ok := m.proxies["test-id"]
	if !ok {
		t.Error("Proxy not added to map")
	}

	if p.ID != "test-id" {
		t.Errorf("Expected ID test-id, got %s", p.ID)
	}
}

func TestRemoveProxy(t *testing.T) {
	log, err := logger.NewLogger("test_logs", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	cfgMgr := config.NewManager("test.json")
	m := NewManager(cfgMgr, log, nil)

	cfg := config.ProxyConfig{
		ID:         "test-id",
		Name:       "Test Proxy",
		ListenAddr: ":5020",
		TargetAddr: "192.168.1.100:502",
		Enabled:    true,
	}

	// Add proxy to config first
	err = m.AddProxy(cfg, true)
	if err != nil {
		t.Fatalf("AddProxy failed: %v", err)
	}

	err = m.RemoveProxy("test-id")
	if err != nil {
		t.Errorf("RemoveProxy failed: %v", err)
	}

	if len(m.proxies) != 0 {
		t.Errorf("Expected 0 proxies, got %d", len(m.proxies))
	}
}

func TestRemoveProxyNotFound(t *testing.T) {
	log, err := logger.NewLogger("test_logs", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	cfgMgr := config.NewManager("test.json")
	m := NewManager(cfgMgr, log, nil)

	err = m.RemoveProxy("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent proxy")
	}
}

func TestGetProxies(t *testing.T) {
	log, err := logger.NewLogger("test_logs", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	cfgMgr := config.NewManager("test.json")
	m := NewManager(cfgMgr, log, nil)

	cfg := config.ProxyConfig{
		ID:         "test-id",
		Name:       "Test Proxy",
		ListenAddr: ":5020",
		TargetAddr: "192.168.1.100:502",
		Enabled:    true,
	}

	if err := m.AddProxy(cfg, false); err != nil {
		t.Fatalf("Failed to add proxy: %v", err)
	}

	proxies := m.GetProxies()
	if len(proxies) != 1 {
		t.Errorf("Expected 1 proxy in list, got %d", len(proxies))
	}

	if proxies[0]["id"] != "test-id" {
		t.Errorf("Expected ID test-id, got %v", proxies[0]["id"])
	}
}

func TestGetProxyStatus(t *testing.T) {
	log, err := logger.NewLogger("test_logs", 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	cfgMgr := config.NewManager("test.json")
	m := NewManager(cfgMgr, log, nil)

	cfg := config.ProxyConfig{
		ID:         "test-id",
		Name:       "Test Proxy",
		ListenAddr: ":5020",
		TargetAddr: "192.168.1.100:502",
		Enabled:    true,
		Paused:     false,
	}

	if err := m.AddProxy(cfg, true); err != nil {
		t.Fatalf("Failed to add proxy: %v", err)
	}

	status := m.getProxyStatus("test-id")
	if status == nil {
		t.Error("getProxyStatus returned nil")
	}

	if status["id"] != "test-id" {
		t.Errorf("Expected ID test-id, got %v", status["id"])
	}

	if status["paused"] != false {
		t.Errorf("Expected paused false, got %v", status["paused"])
	}

	if status["enabled"] != true {
		t.Errorf("Expected enabled true, got %v", status["enabled"])
	}
}

// TestUpdateProxyPersistsConfigWhenStartFails verifies that a failed start
// after an update does not leave config.json describing the old proxy: the
// runtime already holds the new instance, the health monitor retries it, and
// skipping the persist would silently revert the update on the next restart.
func TestUpdateProxyPersistsConfigWhenStartFails(t *testing.T) {
	dir := t.TempDir()
	log, err := logger.NewLogger(dir, 100)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	cfgMgr := config.NewManager(filepath.Join(dir, "config.json"))
	m := NewManager(cfgMgr, log, nil)

	initial := config.ProxyConfig{
		ID:         "update-fail",
		Name:       "Initial",
		ListenAddr: "127.0.0.1:47891",
		TargetAddr: "10.255.255.1:502",
		Enabled:    false,
	}
	if err := m.AddProxy(initial, true); err != nil {
		t.Fatalf("AddProxy failed: %v", err)
	}

	// Occupy the port the updated proxy wants, so its Start() fails.
	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("blocker listen: %v", err)
	}
	defer blocker.Close()

	updated := initial
	updated.Name = "Updated"
	updated.ListenAddr = blocker.Addr().String()
	updated.TargetAddr = "192.168.178.196:502"
	updated.Enabled = true

	if err := m.UpdateProxy(updated); err == nil {
		t.Fatal("UpdateProxy should report the start failure")
	}

	got := cfgMgr.Get().Proxies
	if len(got) != 1 {
		t.Fatalf("config holds %d proxies, want 1", len(got))
	}
	if got[0].Name != "Updated" || got[0].TargetAddr != updated.TargetAddr {
		t.Errorf("config not persisted after failed start: %+v", got[0])
	}
}
