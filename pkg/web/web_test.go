package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerAppliesSecurityHeaders(t *testing.T) {
	handler, err := Handler()
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self'") {
		t.Fatalf("CSP missing self-only script policy: %q", csp)
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", w.Header().Get("X-Frame-Options"))
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") || !strings.Contains(csp, "object-src 'none'") {
		t.Fatalf("CSP missing hardening directives: %q", csp)
	}
	if w.Header().Get("Cross-Origin-Resource-Policy") != "same-origin" {
		t.Fatalf("Cross-Origin-Resource-Policy = %q, want same-origin", w.Header().Get("Cross-Origin-Resource-Policy"))
	}
	if !strings.Contains(w.Body.String(), `src="/theme-bootstrap.js"`) {
		t.Fatal("index.html does not load the external theme bootstrap")
	}
	if strings.Contains(w.Body.String(), "localStorage.getItem") {
		t.Fatal("index.html still contains an inline theme bootstrap")
	}
}

func TestHandlerCachePolicy(t *testing.T) {
	handler, err := Handler()
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	assets, err := fs.Glob(distFS, "dist/assets/*")
	if err != nil || len(assets) == 0 {
		t.Fatalf("embedded assets unavailable: %v", err)
	}
	assetPath := strings.TrimPrefix(assets[0], "dist")

	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/", want: "no-cache"},
		{path: "/theme-bootstrap.js", want: "no-cache"},
		{path: assetPath, want: "public, max-age=31536000, immutable"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if got := w.Header().Get("Cache-Control"); got != tc.want {
				t.Fatalf("Cache-Control = %q, want %q", got, tc.want)
			}
		})
	}
}
