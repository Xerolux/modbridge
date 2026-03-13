package cluster

import (
	"context"
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

// Cluster manages cluster membership
type Cluster struct {
	mu    sync.RWMutex
	nodes map[string]*Node
	self  *Node
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
	}
	c.nodes[nodeID] = c.self
	return c
}

// Join joins the cluster
func (c *Cluster) Join(ctx context.Context, discoveryAddrs []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// TODO: Implement actual cluster join logic
	return nil
}

// Leave leaves the cluster gracefully
func (c *Cluster) Leave() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.self.State = NodeStateLeaving
	return nil
}

// Members returns all cluster members
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

// IsLeader checks if this node is the leader
func (c *Cluster) IsLeader() bool {
	// Simplified: first node is leader
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	for _, node := range c.nodes {
		if node.State == NodeStateReady {
			return node.ID == c.self.ID
		}
	}
	return false
}
