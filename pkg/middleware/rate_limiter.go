package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	clients map[string]*clientLimiter
	mu      sync.RWMutex
	rate    int
	burst   int
}

type clientLimiter struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientLimiter),
		rate:    rate,
		burst:   burst,
	}
	go rl.cleanup()
	return rl
}

// Middleware returns a rate limiting middleware
func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := rl.getClientIP(r)

		if !rl.allow(ip) {
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// allow checks if the request is allowed
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	limiter, exists := rl.clients[ip]
	if !exists {
		limiter = &clientLimiter{
			tokens:     rl.burst,
			lastRefill: time.Now(),
		}
		rl.clients[ip] = limiter
	}
	rl.mu.Unlock()

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(limiter.lastRefill).Seconds()
	tokensToAdd := int(elapsed * float64(rl.rate))

	limiter.tokens += tokensToAdd
	if limiter.tokens > rl.burst {
		limiter.tokens = rl.burst
	}
	limiter.lastRefill = now

	if limiter.tokens > 0 {
		limiter.tokens--
		return true
	}

	return false
}

// getClientIP extracts the client IP address
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	return r.RemoteAddr
}

// cleanup removes inactive clients
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, limiter := range rl.clients {
			if now.Sub(limiter.lastRefill) > 10*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}
