package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireHAIngress(t *testing.T) {
	const prefix = "/api/hassio_ingress/example-token"
	handler := RequireHAIngress(NewSecurityMiddleware().Middleware(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "test", Value: "ok", Path: CookiePath(r), Secure: SecureCookie(r, false)})
		w.WriteHeader(http.StatusNoContent)
	}))
	tests := []struct {
		name, peer, path string
		status           int
	}{
		{"supervisor", "172.30.32.2:12345", prefix + "/", http.StatusNoContent},
		{"direct", "172.30.32.3:12345", prefix, http.StatusForbidden},
		{"missing prefix", "172.30.32.2:12345", "", http.StatusBadRequest},
		{"invalid prefix", "172.30.32.2:12345", prefix + "/../", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.peer
			r.Header.Set("X-Ingress-Path", tt.path)
			r.Header.Set("X-Forwarded-Proto", "https")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d", w.Code, tt.status)
			}
			if w.Code != http.StatusNoContent {
				return
			}
			cookies := w.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Path != prefix || !cookies[0].Secure {
				t.Fatalf("unexpected cookies: %v", cookies)
			}
			if w.Header().Get("X-Frame-Options") != "SAMEORIGIN" ||
				!strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'self'") {
				t.Fatal("expected same-origin framing")
			}
		})
	}
}
