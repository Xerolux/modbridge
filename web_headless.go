// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

//go:build headless

package main

import "net/http"

// getWebHandler returns nil in the headless build: the embedded web assets
// are excluded from the binary entirely. Everything else — REST API,
// authentication, database, proxies, calibration — is identical to the web
// build; the UI is simply served by a different instance (or not at all).
func getWebHandler() http.Handler {
	return nil
}
