//go:build headless

package main

import (
	"net/http"
)

// getWebHandler returns nil in headless mode (no web UI)
// This file is ONLY compiled when building with -tags headless
// It overrides the getWebHandler() function from web.go
func getWebHandler() http.Handler {
	return nil
}
