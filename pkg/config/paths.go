// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package config

import (
	"os"
	"path/filepath"
)

// Environment variables that relocate the files ModBridge writes. They exist
// because a container mounts its volumes somewhere other than the working
// directory, and a database written next to the binary is lost when the
// container is recreated.
const (
	// EnvDataDir holds the database and anything else that must survive a
	// restart.
	EnvDataDir = "MODBRIDGE_DATA_DIR"
	// EnvLogDir holds the per-proxy log files.
	EnvLogDir = "MODBRIDGE_LOG_DIR"
	// EnvConfigFile points at config.json.
	EnvConfigFile = "MODBRIDGE_CONFIG"
)

// Defaults keep the historical layout: everything relative to the working
// directory, so an existing bare-metal install keeps finding its files.
const (
	defaultDataDir    = "."
	defaultLogDir     = "proxy.log"
	defaultConfigFile = "config.json"
	databaseFileName  = "modbridge.db"
)

// DataDir returns the directory for persistent state.
func DataDir() string {
	if dir := os.Getenv(EnvDataDir); dir != "" {
		return dir
	}
	return defaultDataDir
}

// DatabasePath returns the path of the SQLite database.
func DatabasePath() string {
	return filepath.Join(DataDir(), databaseFileName)
}

// LogDir returns the directory that receives the log files.
func LogDir() string {
	if dir := os.Getenv(EnvLogDir); dir != "" {
		return dir
	}
	return defaultLogDir
}

// ConfigPath returns the path of config.json.
func ConfigPath() string {
	if path := os.Getenv(EnvConfigFile); path != "" {
		return path
	}
	return defaultConfigFile
}
