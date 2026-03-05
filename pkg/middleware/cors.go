package middleware

import (
	"net/http"
	"sync"
)

// CORSMiddleware handles CORS headers with allowed origins whitelist
type CORSMiddleware struct {
	mu             sync.RWMutex
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

		m.mu.RLock()
		originAllowed := origin != "" && m.allowedOrigins[origin]
		m.mu.RUnlock()

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

// AddOrigin dynamically adds an origin to the allowed list
func (m *CORSMiddleware) AddOrigin(origin string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowedOrigins[origin] = true
}

// RemoveOrigin removes an origin from the allowed list
func (m *CORSMiddleware) RemoveOrigin(origin string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.allowedOrigins, origin)
}

// IsOriginAllowed checks if an origin is allowed
func (m *CORSMiddleware) IsOriginAllowed(origin string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.allowedOrigins[origin]
}
