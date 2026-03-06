//go:build !headless

package main

import (
	"modbridge/pkg/web"
	"net/http"
)

// getWebHandler returns the web UI handler
// This file is used when NOT building with 'headless' tag
func getWebHandler() http.Handler {
	return web.Handler()
}
