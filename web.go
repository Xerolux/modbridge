package main

import (
	"log"
	"modbridge/pkg/web"
	"net/http"
)

// getWebHandler returns the web UI handler, or nil if the embedded assets are unavailable.
func getWebHandler() http.Handler {
	h, err := web.Handler()
	if err != nil {
		log.Printf("Warning: web UI unavailable: %v", err)
		return nil
	}
	return h
}
