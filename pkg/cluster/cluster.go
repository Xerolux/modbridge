// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// NodeState represents the state of a cluster node
type NodeState int

const (
	NodeStateStarting NodeState = iota
	NodeStateReady
	NodeStateLeaving
	NodeStateLeft
)

// Node represents a cluster member
type Node struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	State    NodeState `json:"state"`
	LastSeen time.Time `json:"last_seen"`
}

// Cluster manages cluster membership via HTTP-based peer discovery.
// Each node announces itself to a list of seed addresses and periodically
// exchanges heartbeats so all members stay in sync.
type Cluster struct {
	mu    sync.RWMutex
	nodes map[string]*Node
	self  *Node

	client *http.Client
}

// NewCluster creates a new cluster
func NewCluster(nodeID, address string) *Cluster {
	c := &Cluster{
		nodes: make(map[string]*Node),
		self: &Node{
			ID:      nodeID,
			Address: address,
			State:   NodeStateReady,
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}
	c.nodes[nodeID] = c.self
	return c
}

// Join announces this node to each discovery address and fetches their member
// lists.  It then starts a background heartbeat loop that keeps all peers
// updated until ctx is cancelled.
func (c *Cluster) Join(ctx context.Context, discoveryAddrs []string) error {
	c.mu.Lock()
	c.self.State = NodeStateReady
	c.mu.Unlock()

	// Announce to all seed nodes and collect their known members.
	for _, addr := range discoveryAddrs {
		if err := c.announceToNode(addr); err != nil {
			// Non-fatal: seed may be temporarily unavailable.
			continue
		}
		if peers, err := c.fetchMembers(addr); err == nil {
			c.mu.Lock()
			for _, peer := range peers {
				if _, exists := c.nodes[peer.ID]; !exists {
					c.nodes[peer.ID] = peer
				}
			}
			c.mu.Unlock()
		}
	}

	// Start heartbeat loop.
	go c.heartbeatLoop(ctx)

	return nil
}

// announceToNode sends a POST with this node's info to addr/cluster/join.
func (c *Cluster) announceToNode(addr string) error {
	c.mu.RLock()
	self := *c.self
	c.mu.RUnlock()

	body, err := json.Marshal(self)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s/cluster/join", addr)
	resp, err := c.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("peer %s returned %d", addr, resp.StatusCode)
	}
	return nil
}

// fetchMembers fetches the member list from addr/cluster/members.
func (c *Cluster) fetchMembers(addr string) ([]*Node, error) {
	url := fmt.Sprintf("http://%s/cluster/members", addr)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var members []*Node
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, err
	}
	return members, nil
}

// heartbeatLoop periodically announces this node to all known peers and evicts
// peers that haven't been seen within 3× the heartbeat interval.
func (c *Cluster) heartbeatLoop(ctx context.Context) {
	const interval = 15 * time.Second
	const ttl = 3 * interval

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.mu.Lock()
			c.self.State = NodeStateLeft
			c.mu.Unlock()
			return
		case <-ticker.C:
			// Update our own LastSeen.
			c.mu.Lock()
			c.self.LastSeen = time.Now()
			c.mu.Unlock()

			// Announce to peers.
			c.mu.RLock()
			peers := make([]*Node, 0, len(c.nodes))
			for _, n := range c.nodes {
				if n.ID != c.self.ID {
					peers = append(peers, n)
				}
			}
			c.mu.RUnlock()

			for _, peer := range peers {
				_ = c.announceToNode(peer.Address) //nolint:errcheck // best effort
			}

			// Evict stale nodes.
			c.mu.Lock()
			cutoff := time.Now().Add(-ttl)
			for id, n := range c.nodes {
				if id != c.self.ID && n.LastSeen.Before(cutoff) {
					delete(c.nodes, id)
				}
			}
			c.mu.Unlock()
		}
	}
}

// HandleJoin processes an inbound join announcement from a peer.
// Register this as an HTTP handler at POST /cluster/join.
func (c *Cluster) HandleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var peer Node
	if err := json.NewDecoder(r.Body).Decode(&peer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	peer.LastSeen = time.Now()
	peer.State = NodeStateReady

	c.mu.Lock()
	c.nodes[peer.ID] = &peer
	c.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// HandleMembers returns the current cluster member list as JSON.
// Register this as an HTTP handler at GET /cluster/members.
func (c *Cluster) HandleMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	members := c.Members()
	if err := json.NewEncoder(w).Encode(members); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Leave leaves the cluster gracefully
func (c *Cluster) Leave() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.self.State = NodeStateLeaving
	return nil
}

// Members returns all ready cluster members
func (c *Cluster) Members() []*Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	members := make([]*Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		if node.State == NodeStateReady {
			members = append(members, node)
		}
	}
	return members
}

// IsLeader checks if this node is the leader.
// Leadership is assigned to the node with the lexicographically smallest ID
// to ensure a deterministic, stable election without external coordination.
func (c *Cluster) IsLeader() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, node := range c.nodes {
		if node.State == NodeStateReady && node.ID < c.self.ID {
			return false
		}
	}
	return true
}
