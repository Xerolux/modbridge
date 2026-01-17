package middleware

import (
	"net/http"
)

// CORSMiddleware handles CORS headers with allowed origins whitelist
type CORSMiddleware struct {
	allowedOrigins map[string]bool
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(allowedOrigins []string) *CORSMiddleware {
	origins := make(map[string]bool)
	for _, origin := range allowedOrigins {
		origins[origin] = true
	}

	// Always allow localhost for development
	origins["http://localhost:8080"] = true
	origins["http://localhost:3000"] = true
	origins["http://127.0.0.1:8080"] = true
	origins["http://127.0.0.1:3000"] = true

	return &CORSMiddleware{
		allowedOrigins: origins,
	}
}

// Middleware returns a CORS middleware
func (m *CORSMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" && m.allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			// Same-origin requests, no CORS needed
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// AddOrigin dynamically adds an origin to the allowed list
func (m *CORSMiddleware) AddOrigin(origin string) {
	m.allowedOrigins[origin] = true
}

// RemoveOrigin removes an origin from the allowed list
func (m *CORSMiddleware) RemoveOrigin(origin string) {
	delete(m.allowedOrigins, origin)
}

// IsOriginAllowed checks if an origin is allowed
func (m *CORSMiddleware) IsOriginAllowed(origin string) bool {
	return m.allowedOrigins[origin]
}
