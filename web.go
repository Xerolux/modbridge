//go:build !headless
// +build !headless

package main

import (
	"modbridge/pkg/web"
	"net/http"
)

// getWebHandler returns the web UI handler (default implementation)
func getWebHandler() http.Handler {
	return web.Handler()
}
