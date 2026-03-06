//go:build headless
// +build headless

package main

import (
	"net/http"
)

// getWebHandler returns nil in headless mode (no web UI)
func getWebHandler() http.Handler {
	return nil
}
