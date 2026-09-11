package api

import (
	"encoding/json"
	"modbridge/pkg/config"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConnectivityTargetsOnlySelectedProxy(t *testing.T) {
	server, token := updateTestServer(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := server.cfgMgr.AddProxy(config.ProxyConfig{ID: "selected", Name: "Selected", ListenAddr: ":5901", TargetAddr: listener.Addr().String()}); err != nil {
		t.Fatal(err)
	}
	if err := server.cfgMgr.AddProxy(config.ProxyConfig{ID: "other", Name: "Other", ListenAddr: ":5902", TargetAddr: "127.0.0.1:1"}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"selected", "missing"} {
		req := httptest.NewRequest(http.MethodGet, "/api/system/diagnostics/connectivity?proxy_id="+id, nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		w := httptest.NewRecorder()
		server.handleProxyConnectivityCheck(w, req)
		if id == "missing" {
			if w.Code != 404 {
				t.Fatalf("missing proxy: %d", w.Code)
			}
			continue
		}
		var result map[string]struct {
			Reachable bool `json:"reachable"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || !result[id].Reachable {
			t.Fatalf("unexpected probe result: %s", w.Body.String())
		}
	}
}
