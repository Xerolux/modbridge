// Copyright (c) 2026 Xerolux. All rights reserved.

package middleware

import (
	"context"
	"net"
	"net/http"
	"regexp"
)

type ingressPathKey struct{}

var haIngressPath = regexp.MustCompile(`^(/api/hassio_ingress/[A-Za-z0-9_-]+)/?$`)

// RequireHAIngress restricts the app's HTTP listener to Supervisor ingress.
func RequireHAIngress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || peer != "172.30.32.2" {
			http.Error(w, "Home Assistant Supervisor required", http.StatusForbidden)
			return
		}
		match := haIngressPath.FindStringSubmatch(r.Header.Get("X-Ingress-Path"))
		if match == nil {
			http.Error(w, "Invalid Home Assistant ingress path", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), ingressPathKey{}, match[1])
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HAIngressPath returns the validated request prefix without a trailing slash.
func HAIngressPath(r *http.Request) string {
	prefix, _ := r.Context().Value(ingressPathKey{}).(string)
	return prefix
}

// CookiePath scopes ModBridge cookies to the app when it shares the HA origin.
func CookiePath(r *http.Request) string {
	if prefix := HAIngressPath(r); prefix != "" {
		return prefix
	}
	return "/"
}

// SecureCookie uses the original HTTPS scheme when the Supervisor proxies HTTP.
func SecureCookie(r *http.Request, directTLS bool) bool {
	if HAIngressPath(r) != "" {
		return r.Header.Get("X-Forwarded-Proto") == "https"
	}
	return directTLS
}
