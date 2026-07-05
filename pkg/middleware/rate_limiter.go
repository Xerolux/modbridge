// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package middleware

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// maxClients limits the number of tracked IPs to prevent unbounded memory growth.
const maxClients = 10000

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	clients        map[string]*clientLimiter
	mu             sync.RWMutex
	rate           int
	burst          int
	trustedProxies []*net.IPNet
	stop           chan struct{}
	wg             sync.WaitGroup
}

type clientLimiter struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter. If MODBRIDGE_TRUSTED_PROXIES is
// set (comma-separated IPs or CIDRs), X-Forwarded-For/X-Real-IP headers are
// only trusted when the direct connection comes from one of those addresses.
// Without trusted proxies the limiter falls back to RemoteAddr to prevent
// clients from spoofing their IP and bypassing the rate limit.
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients:        make(map[string]*clientLimiter),
		rate:           rate,
		burst:          burst,
		trustedProxies: parseTrustedProxies(),
		stop:           make(chan struct{}),
	}
	rl.wg.Add(1)
	go rl.cleanup()
	return rl
}

// Stop gracefully shuts down the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stop)
	rl.wg.Wait()
}

// parseTrustedProxies parses the MODBRIDGE_TRUSTED_PROXIES environment variable.
func parseTrustedProxies() []*net.IPNet {
	raw := strings.TrimSpace(os.Getenv("MODBRIDGE_TRUSTED_PROXIES"))
	if raw == "" {
		return nil
	}

	var result []*net.IPNet
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if ip := net.ParseIP(part); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			result = append(result, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}

		if _, cidr, err := net.ParseCIDR(part); err == nil {
			result = append(result, cidr)
		}
	}
	return result
}

// isTrustedProxy reports whether addr belongs to a configured trusted proxy.
func (rl *RateLimiter) isTrustedProxy(addr string) bool {
	if len(rl.trustedProxies) == 0 {
		return false
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	for _, n := range rl.trustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
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
		// Reject new clients when the map is at capacity to prevent memory exhaustion.
		if len(rl.clients) >= maxClients {
			rl.mu.Unlock()
			return false
		}
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

// getClientIP extracts the client IP address. Proxy headers are only used
// when the direct connection originates from a trusted proxy.
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Always fall back to the direct connection address when the request is not
	// coming from a trusted proxy. This prevents clients from spoofing their IP.
	if !rl.isTrustedProxy(r.RemoteAddr) {
		if ip := normalizeIP(r.RemoteAddr); ip != "" {
			return ip
		}
		return "unknown"
	}

	// Check X-Forwarded-For header (right-most untrusted address is safest;
	// left-most can be spoofed by the client).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for i := len(ips) - 1; i >= 0; i-- {
			if ip := normalizeIP(ips[i]); ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := normalizeIP(xri); ip != "" {
			return ip
		}
	}

	// Fallback to RemoteAddr
	if ip := normalizeIP(r.RemoteAddr); ip != "" {
		return ip
	}

	return "unknown"
}

func normalizeIP(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return ""
	}

	if ip := net.ParseIP(candidate); ip != nil {
		return ip.String()
	}

	host, _, err := net.SplitHostPort(candidate)
	if err != nil {
		return candidate
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}

	return host
}

// cleanup removes inactive clients
func (rl *RateLimiter) cleanup() {
	defer rl.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			now := time.Now()
			// Collect stale IPs first (under limiter lock), then delete them.
			var stale []string
			rl.mu.RLock()
			for ip, limiter := range rl.clients {
				limiter.mu.Lock()
				idle := now.Sub(limiter.lastRefill)
				limiter.mu.Unlock()
				if idle > 10*time.Minute {
					stale = append(stale, ip)
				}
			}
			rl.mu.RUnlock()

			if len(stale) > 0 {
				rl.mu.Lock()
				for _, ip := range stale {
					delete(rl.clients, ip)
				}
				rl.mu.Unlock()
			}
		}
	}
}
