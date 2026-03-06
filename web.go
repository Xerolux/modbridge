//go:build !headless

package main

import (
	"modbridge/pkg/web"
	"net/http"
)

// getWebHandler returns the web UI handler
// Built when using -tags webui or any tag except headless
func getWebHandler() http.Handler {
	return web.Handler()
}
