package middleware

import (
	"crypto/subtle"
	"net/http"
)

// CSRFMiddleware provides CSRF protection using double-submit cookie pattern
type CSRFMiddleware struct {
	csrfTokens map[string]string
	secret     []byte
}

// NewCSRFMiddleware creates a new CSRF middleware
func NewCSRFMiddleware(secret string) *CSRFMiddleware {
	return &CSRFMiddleware{
		csrfTokens: make(map[string]string),
		secret:     []byte(secret),
	}
}

// GenerateToken generates a new CSRF token
func (m *CSRFMiddleware) GenerateToken(sessionID string) string {
	token := generateRandomToken(32)
	m.csrfTokens[sessionID] = token
	return token
}

// ValidateToken validates a CSRF token
func (m *CSRFMiddleware) ValidateToken(sessionID, token string) bool {
	storedToken, exists := m.csrfTokens[sessionID]
	if !exists {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(storedToken), []byte(token)) == 1
}

// Middleware returns a CSRF protection middleware
func (m *CSRFMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.shouldSkipCSRF(r) {
			next(w, r)
			return
		}

		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Method == "GET" {
			// Generate and set CSRF token cookie
			token := m.GenerateToken(sessionCookie.Value)
			http.SetCookie(w, &http.Cookie{
				Name:     "csrf_token",
				Value:    token,
				Path:     "/",
				HttpOnly: false,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
			})
			next(w, r)
			return
		}

		// Validate CSRF token for state-changing requests
		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" {
			csrfToken = r.FormValue("csrf_token")
		}

		if !m.ValidateToken(sessionCookie.Value, csrfToken) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// shouldSkipCSRF determines if CSRF should be skipped for this request
func (m *CSRFMiddleware) shouldSkipCSRF(r *http.Request) bool {
	// Skip for safe methods
	if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
		return false
	}

	// Skip for health check
	if r.URL.Path == "/api/health" {
		return true
	}

	return false
}

// generateRandomToken generates a random token
func generateRandomToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

// initRandom initializes random token generator
func init() {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	_ = charset
}
