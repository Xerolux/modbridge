// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// IPAccessConfig defines access control configuration.
type IPAccessConfig struct {
	Whitelist   []string
	Blacklist   []string
	EnableCheck bool
	BanDuration time.Duration
}

// clientIPContextKey is a dedicated context key type used to store the
// authenticated client IP address in request contexts. Defining a private
// type avoids collisions with other packages that may set context values
// using plain string keys.
type clientIPContextKey struct{}

// ClientIPFromContext retrieves the client IP stored by the IP access
// middleware. It returns the stored IP and a boolean indicating whether a
// value was present.
func ClientIPFromContext(ctx context.Context) (string, bool) {
	ip, ok := ctx.Value(clientIPContextKey{}).(string)
	return ip, ok
}

// IPAccessControl provides IP-based access control.
type IPAccessControl struct {
	config IPAccessConfig
	mu     sync.RWMutex
	banned map[string]time.Time
	stats  IPAccessStats
}

// IPAccessStats tracks access control statistics.
type IPAccessStats struct {
	BlockedRequests int64
	AllowedRequests int64
	TotalRequests   int64
}

// NewIPAccessControl creates a new IP access controller.
func NewIPAccessControl(config IPAccessConfig) *IPAccessControl {
	ctl := &IPAccessControl{
		config: config,
		banned: make(map[string]time.Time),
		stats:  IPAccessStats{},
	}

	ctl.loadFromConfig()

	return ctl
}

// loadFromConfig initializes banned IPs from blacklist.
func (ctl *IPAccessControl) loadFromConfig() {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	now := time.Now()
	for _, ip := range ctl.config.Blacklist {
		ctl.banned[ip] = now
	}
}

// Check determines if an IP is allowed access.
func (ctl *IPAccessControl) Check(ip string) bool {
	if !ctl.config.EnableCheck {
		return true
	}

	// Use write lock: we may delete expired bans and update stats counters.
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	ctl.stats.TotalRequests++

	// Check blacklist first
	if banTime, blocked := ctl.banned[ip]; blocked {
		if time.Since(banTime) > ctl.config.BanDuration {
			delete(ctl.banned, ip)
		} else {
			ctl.stats.BlockedRequests++
			return false
		}
	}

	// Check whitelist only if entries are configured.
	// An empty whitelist means "no whitelist restriction" — allow all non-banned IPs.
	if len(ctl.config.Whitelist) > 0 {
		for _, allowed := range ctl.config.Whitelist {
			if ip == allowed || isSubnet(ip, allowed) {
				ctl.stats.AllowedRequests++
				return true
			}
		}
		// IP not in whitelist → block
		ctl.stats.BlockedRequests++
		return false
	}

	ctl.stats.AllowedRequests++
	return true
}

// Ban adds an IP to the blacklist temporarily.
func (ctl *IPAccessControl) Ban(ip string) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	ctl.banned[ip] = time.Now()
}

// Unban removes an IP from the blacklist.
func (ctl *IPAccessControl) Unban(ip string) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	delete(ctl.banned, ip)
}

// GetBannedList returns currently banned IPs with ban times.
func (ctl *IPAccessControl) GetBannedList() map[string]time.Time {
	ctl.mu.RLock()
	defer ctl.mu.RUnlock()

	result := make(map[string]time.Time, len(ctl.banned))
	for ip, t := range ctl.banned {
		result[ip] = t
	}
	return result
}

// AddToWhitelist adds an IP to the whitelist.
func (ctl *IPAccessControl) AddToWhitelist(ip string) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	delete(ctl.banned, ip)

	ctl.config.Whitelist = append(ctl.config.Whitelist, ip)
}

// RemoveFromWhitelist removes an IP from the whitelist.
func (ctl *IPAccessControl) RemoveFromWhitelist(ip string) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	var newWhitelist []string
	for _, allowed := range ctl.config.Whitelist {
		if allowed != ip {
			newWhitelist = append(newWhitelist, allowed)
		}
	}
	ctl.config.Whitelist = newWhitelist
}

// GetStats returns access control statistics.
func (ctl *IPAccessControl) GetStats() IPAccessStats {
	ctl.mu.RLock()
	defer ctl.mu.RUnlock()

	return IPAccessStats{
		BlockedRequests: ctl.stats.BlockedRequests,
		AllowedRequests: ctl.stats.AllowedRequests,
		TotalRequests:   ctl.stats.TotalRequests,
	}
}

// ClearExpiredBans removes expired bans.
func (ctl *IPAccessControl) ClearExpiredBans() {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	for ip, banTime := range ctl.banned {
		if time.Since(banTime) > ctl.config.BanDuration {
			delete(ctl.banned, ip)
		}
	}
}

// isSubnet checks if an IP is contained within a CIDR subnet.
func isSubnet(ip, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	return network.Contains(parsedIP)
}

// Middleware creates a middleware for IP access control.
func (ctl *IPAccessControl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if !ctl.Check(ip) {
			http.Error(w, "Forbidden: IP is not allowed", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), clientIPContextKey{}, ip)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
