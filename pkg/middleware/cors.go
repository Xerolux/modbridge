// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package middleware

import (
	"net/http"
	"sync"
)

// CORSMiddleware handles CORS headers with allowed origins whitelist
type CORSMiddleware struct {
	mu             sync.RWMutex
	allowedOrigins map[string]bool
	allowAny       bool
}

// NewCORSMiddleware creates a new CORS middleware with the given allowed origins.
// Pass an empty slice for no additional origins (all requests without an Origin
// header are still served — CORS headers are only added for recognised origins).
// The wildcard "*" allows any origin; because credentials are enabled the
// requesting origin is echoed back instead of a literal "*".
func NewCORSMiddleware(allowedOrigins []string) *CORSMiddleware {
	origins := make(map[string]bool, len(allowedOrigins))
	allowAny := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAny = true
			continue
		}
		if origin != "" {
			origins[origin] = true
		}
	}
	return &CORSMiddleware{
		allowedOrigins: origins,
		allowAny:       allowAny,
	}
}

// Middleware returns a CORS middleware
func (m *CORSMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		m.mu.RLock()
		originAllowed := origin != "" && (m.allowAny || m.allowedOrigins[origin])
		m.mu.RUnlock()

		// The response body depends on the Origin header whenever the
		// allowed-origin header is derived from it, so a shared cache must key
		// on it. Set unconditionally: a cached response from a request without
		// an Origin must not be replayed for one that has it.
		w.Header().Add("Vary", "Origin")

		if originAllowed {
			// Only set CORS headers for allowed origins
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		// If no Origin header (same-origin request), no CORS headers needed.
		// Do NOT set wildcard "*" with credentials - browsers reject this.

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// AddOrigin dynamically adds an origin to the allowed list. "*" switches the
// middleware to allowing any origin.
func (m *CORSMiddleware) AddOrigin(origin string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if origin == "*" {
		m.allowAny = true
		return
	}
	m.allowedOrigins[origin] = true
}

// RemoveOrigin removes an origin from the allowed list. "*" revokes the
// allow-any setting.
func (m *CORSMiddleware) RemoveOrigin(origin string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if origin == "*" {
		m.allowAny = false
		return
	}
	delete(m.allowedOrigins, origin)
}

// IsOriginAllowed checks if an origin is allowed
func (m *CORSMiddleware) IsOriginAllowed(origin string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.allowAny || m.allowedOrigins[origin]
}
