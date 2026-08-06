package web

import (
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
	if !strings.Contains(w.Body.String(), `src="/theme-bootstrap.js"`) {
		t.Fatal("index.html does not load the external theme bootstrap")
	}
	if strings.Contains(w.Body.String(), "localStorage.getItem") {
		t.Fatal("index.html still contains an inline theme bootstrap")
	}
}
