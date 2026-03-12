package main

import (
	"modbridge/pkg/web"
	"net/http"
)

// getWebHandler returns the web UI handler
func getWebHandler() http.Handler {
	return web.Handler()
}
