// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeUntilRotated writes entries until the active file has been rotated at
// least once, so the test does not depend on the exact size of a log line.
func writeUntilRotated(t *testing.T, l *Logger, path string) {
	t.Helper()
	for i := 0; i < 5000; i++ {
		l.Log(ERROR, "SYSTEM", strings.Repeat("x", 200))
		if _, err := os.Stat(path + ".1"); err == nil {
			return
		}
	}
	t.Fatalf("log was never rotated")
}

func TestLoggerRotatesAtMaxSize(t *testing.T) {
	dir := t.TempDir()
	l, err := NewLogger(dir, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	l.SetRotation(RotationConfig{MaxSizeMB: 1, MaxFiles: 3})

	active := filepath.Join(dir, "system.log")
	writeUntilRotated(t, l, active)

	info, err := os.Stat(active)
	if err != nil {
		t.Fatalf("active log missing after rotation: %v", err)
	}
	if info.Size() > 1024*1024 {
		t.Errorf("active log is %d bytes, expected it to be rotated below 1 MiB", info.Size())
	}
}

func TestLoggerKeepsAtMostMaxFiles(t *testing.T) {
	dir := t.TempDir()
	l, err := NewLogger(dir, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	const keep = 2
	l.SetRotation(RotationConfig{MaxSizeMB: 1, MaxFiles: keep})

	active := filepath.Join(dir, "system.log")
	for r := 0; r < keep+2; r++ {
		writeUntilRotated(t, l, active)
		// Force the next round to rotate again rather than exit immediately.
		_ = os.Remove(active + ".1")
		l.Log(ERROR, "SYSTEM", strings.Repeat("x", 200))
	}

	if _, err := os.Stat(fmt.Sprintf("%s.%d", active, keep+1)); !os.IsNotExist(err) {
		t.Errorf("backup beyond MaxFiles=%d exists, expected it to be pruned", keep)
	}
}

func TestLoggerNoRotationWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	l, err := NewLogger(dir, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	// MaxSizeMB zero means the previous behaviour: one file, no rotation.
	for i := 0; i < 2000; i++ {
		l.Log(ERROR, "SYSTEM", strings.Repeat("x", 200))
	}

	if _, err := os.Stat(filepath.Join(dir, "system.log.1")); !os.IsNotExist(err) {
		t.Errorf("rotated file exists although rotation is disabled")
	}
}

func TestLoggerPrunesAgedBackups(t *testing.T) {
	dir := t.TempDir()
	l, err := NewLogger(dir, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	l.SetRotation(RotationConfig{MaxSizeMB: 1, MaxFiles: 3, MaxAgeDays: 7})

	active := filepath.Join(dir, "system.log")
	writeUntilRotated(t, l, active)

	// Backdate the backup past the retention window, then rotate again so the
	// prune runs.
	old := time.Now().AddDate(0, 0, -30)
	if err := os.Chtimes(active+".1", old, old); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	writeUntilRotated(t, l, active)

	if _, err := os.Stat(active + ".2"); !os.IsNotExist(err) {
		t.Errorf("aged backup still present, expected it to be pruned")
	}
}

func TestNewLoggerMovesExistingFileAside(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxy.log")
	if err := os.WriteFile(path, []byte("previous install\n"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	l, err := NewLogger(path, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	saved, err := os.ReadFile(path + ".old")
	if err != nil {
		t.Fatalf("previous log was not preserved: %v", err)
	}
	if string(saved) != "previous install\n" {
		t.Errorf("preserved log content = %q, want %q", saved, "previous install\n")
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Errorf("log path is not a directory after NewLogger")
	}
}
