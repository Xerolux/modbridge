// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type csrfEntry struct {
	token     string
	createdAt time.Time
}

// CSRFMiddleware provides CSRF protection using double-submit cookie pattern
type CSRFMiddleware struct {
	mu         sync.Mutex
	csrfTokens map[string]csrfEntry
	secret     []byte
	maxAge     time.Duration
	stop       chan struct{}
	wg         sync.WaitGroup
}

// NewCSRFMiddleware creates a new CSRF middleware
func NewCSRFMiddleware(secret string) *CSRFMiddleware {
	m := &CSRFMiddleware{
		csrfTokens: make(map[string]csrfEntry),
		secret:     []byte(secret),
		maxAge:     24 * time.Hour,
		stop:       make(chan struct{}),
	}
	m.wg.Add(1)
	go m.cleanup()
	return m
}

// Stop gracefully shuts down the background cleanup goroutine.
func (m *CSRFMiddleware) Stop() {
	close(m.stop)
	m.wg.Wait()
}

// GenerateToken generates a new CSRF token, or returns the still-valid token
// already bound to the session. Reusing the existing token avoids invalidating
// tokens held by other concurrent tabs/requests for the same session.
func (m *CSRFMiddleware) GenerateToken(sessionID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, ok := m.csrfTokens[sessionID]; ok && time.Since(entry.createdAt) <= m.maxAge {
		return entry.token
	}

	token := m.deriveToken(sessionID)
	if token == "" {
		// Retry once — if crypto/rand fails twice the system has a serious problem.
		token = m.deriveToken(sessionID)
	}
	// If both attempts failed, return an empty string without storing it; the
	// caller must treat that as an error.
	if token == "" {
		return ""
	}
	m.csrfTokens[sessionID] = csrfEntry{
		token:     token,
		createdAt: time.Now(),
	}
	return token
}

// deriveToken derives a token from the configured secret, the session ID and a
// fresh random nonce, so MODBRIDGE_CSRF_SECRET actually influences the issued
// tokens instead of being ignored. Returns "" when crypto/rand fails.
func (m *CSRFMiddleware) deriveToken(sessionID string) string {
	nonce := generateRandomToken(32)
	if nonce == "" {
		return ""
	}
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(sessionID))
	mac.Write([]byte{0})
	mac.Write([]byte(nonce))
	return hex.EncodeToString(mac.Sum(nil))
}

// ClearSession drops the CSRF token bound to a session. Call it when the
// session ends so logged-out sessions leave no reusable token behind.
func (m *CSRFMiddleware) ClearSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.csrfTokens, sessionID)
}

// ValidateToken validates a CSRF token
func (m *CSRFMiddleware) ValidateToken(sessionID, token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if token == "" {
		return false
	}
	entry, exists := m.csrfTokens[sessionID]
	if !exists {
		return false
	}

	if time.Since(entry.createdAt) > m.maxAge {
		delete(m.csrfTokens, sessionID)
		return false
	}

	return subtle.ConstantTimeCompare([]byte(entry.token), []byte(token)) == 1
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
			if token == "" {
				http.Error(w, "Failed to issue CSRF token", http.StatusInternalServerError)
				return
			}
			// Only set Secure flag if connection is HTTPS
			isSecure := r.TLS != nil
			http.SetCookie(w, &http.Cookie{
				Name:     "csrf_token",
				Value:    token,
				Path:     "/",
				HttpOnly: false,
				Secure:   isSecure,
				SameSite: http.SameSiteStrictMode,
			})
			next(w, r)
			return
		}

		// Validate CSRF token for state-changing requests
		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
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

// cleanup periodically removes expired CSRF tokens
func (m *CSRFMiddleware) cleanup() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.mu.Lock()
			for sessionID, entry := range m.csrfTokens {
				if time.Since(entry.createdAt) > m.maxAge {
					delete(m.csrfTokens, sessionID)
				}
			}
			m.mu.Unlock()
		}
	}
}

// generateRandomToken generates a cryptographically secure random token.
// Returns a hex string of length*2 characters (each byte encodes to 2 hex chars).
func generateRandomToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b) // full hex encoding — no truncation
}
