// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package cluster

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewCluster(t *testing.T) {
	c := NewCluster("node1", "127.0.0.1:9001")
	if c == nil {
		t.Fatal("expected non-nil cluster")
	}
	if c.self.ID != "node1" {
		t.Fatalf("expected node1, got %s", c.self.ID)
	}
	if c.self.State != NodeStateReady {
		t.Fatal("expected NodeStateReady")
	}
}

func TestCluster_Members(t *testing.T) {
	c := NewCluster("node1", "127.0.0.1:9001")
	members := c.Members()
	if len(members) != 1 {
		t.Fatalf("expected 1 member (self), got %d", len(members))
	}
}

func TestHandleJoinAndMembers(t *testing.T) {
	c := NewCluster("node1", "127.0.0.1:9001")

	peer := &Node{ID: "node2", Address: "127.0.0.1:9002", State: NodeStateReady}
	body, _ := json.Marshal(peer)
	req := httptest.NewRequest(http.MethodPost, "/cluster/join", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	c.HandleJoin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleJoin returned %d", w.Code)
	}

	members := c.Members()
	found := false
	for _, m := range members {
		if m.ID == "node2" {
			found = true
		}
	}
	if !found {
		t.Fatal("node2 not found in members after HandleJoin")
	}
}

func TestHandleMembers(t *testing.T) {
	c := NewCluster("node1", "127.0.0.1:9001")
	req := httptest.NewRequest(http.MethodGet, "/cluster/members", nil)
	w := httptest.NewRecorder()
	c.HandleMembers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleMembers returned %d", w.Code)
	}
	var members []*Node
	if err := json.NewDecoder(w.Body).Decode(&members); err != nil {
		t.Fatalf("failed to decode members: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
}

func TestHandleJoinBadBody(t *testing.T) {
	c := NewCluster("node1", "127.0.0.1:9001")
	req := httptest.NewRequest(http.MethodPost, "/cluster/join", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	c.HandleJoin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleJoinWrongMethod(t *testing.T) {
	c := NewCluster("node1", "127.0.0.1:9001")
	req := httptest.NewRequest(http.MethodGet, "/cluster/join", nil)
	w := httptest.NewRecorder()
	c.HandleJoin(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestIsLeaderSingleNode(t *testing.T) {
	c := NewCluster("node1", "127.0.0.1:9001")
	if !c.IsLeader() {
		t.Fatal("single node should always be leader")
	}
}

func TestIsLeaderMultipleNodes(t *testing.T) {
	c := NewCluster("node_b", "127.0.0.1:9002")
	c.mu.Lock()
	c.nodes["node_a"] = &Node{ID: "node_a", Address: "127.0.0.1:9001", State: NodeStateReady}
	c.mu.Unlock()

	if c.IsLeader() {
		t.Fatal("node_b should not be leader when node_a has smaller ID")
	}
}

func TestLeave(t *testing.T) {
	c := NewCluster("node1", "127.0.0.1:9001")
	if err := c.Leave(); err != nil {
		t.Fatalf("Leave returned error: %v", err)
	}
	if c.self.State != NodeStateLeaving {
		t.Fatal("expected NodeStateLeaving after Leave()")
	}
	members := c.Members()
	if len(members) != 0 {
		t.Fatalf("expected 0 ready members after leave, got %d", len(members))
	}
}

func TestJoinWithNoSeeds(t *testing.T) {
	c := NewCluster("node1", "127.0.0.1:9001")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.Join(ctx, []string{}); err != nil {
		t.Fatalf("Join with no seeds returned error: %v", err)
	}
}

func TestJoinAnnounce(t *testing.T) {
	seed := NewCluster("seed", "127.0.0.1:9999")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cluster/join":
			seed.HandleJoin(w, r)
		case "/cluster/members":
			seed.HandleMembers(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	c := NewCluster("node2", "127.0.0.1:9002")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.Join(ctx, []string{addr}); err != nil {
		t.Fatalf("Join returned error: %v", err)
	}

	members := seed.Members()
	found := false
	for _, m := range members {
		if m.ID == "node2" {
			found = true
		}
	}
	if !found {
		t.Fatal("seed does not know about node2 after join")
	}
}
