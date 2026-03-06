package main

import (
	"modbridge/pkg/web"
	"net/http"
)

// getWebHandler returns the web UI handler
// This is the DEFAULT implementation
// When building with -tags headless, web_headless.go is used instead
func getWebHandler() http.Handler {
	return web.Handler()
}
