// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

// SecurityMiddleware adds security headers
type SecurityMiddleware struct{}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware() *SecurityMiddleware {
	return &SecurityMiddleware{}
}

// Middleware returns a security headers middleware
func (m *SecurityMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// HSTS (HTTP Strict Transport Security)
		// Only add if the request is over HTTPS
		if r.URL.Scheme == "https" || strings.HasPrefix(r.Proto, "HTTPS") ||
			r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// X-Content-Type-Options
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// X-Frame-Options
		w.Header().Set("X-Frame-Options", "DENY")

		// X-XSS-Protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content-Security-Policy
		// unsafe-eval removed from script-src; a pre-built Vue 3/Vite SPA does not need it.
		// unsafe-inline is kept for style-src only (PrimeVue/TailwindCSS inject inline styles).
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: https:; font-src 'self' data: https://fonts.gstatic.com; connect-src 'self' ws: wss:;")

		// Referrer-Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy (formerly Feature-Policy)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Cross-Origin-Opener-Policy helps prevent cross-origin attacks
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")

		// Disable DNS prefetching to reduce information leakage
		w.Header().Set("X-DNS-Prefetch-Control", "off")

		// X-Request-ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)

		// Server header (remove or obscure server info)
		w.Header().Set("Server", "")

		next(w, r)
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a cryptographically secure random string
func randomString(length int) string {
	bytes := make([]byte, length/2+1) // hex encoding doubles the length
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID (less secure but better than nothing)
		return time.Now().Format("20060102150405.000000")[:length]
	}
	return hex.EncodeToString(bytes)[:length]
}
