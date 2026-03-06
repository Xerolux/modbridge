//go:build !headless

package main

import (
	"modbridge/pkg/web"
	"net/http"
)

// getWebHandler returns the web UI handler
// This file is ONLY compiled when NOT building with -tags headless
func getWebHandler() http.Handler {
	return web.Handler()
}
