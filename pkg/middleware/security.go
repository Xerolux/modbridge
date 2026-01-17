package middleware

import (
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

		// Content-Security-Policy (basic)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' ws: wss:;")

		// Referrer-Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy (formerly Feature-Policy)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

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

// randomString generates a random string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
