package cluster

import (
	"testing"
)

func TestNewCluster(t *testing.T) {
	c := NewCluster("node1", "localhost:8080")
	if c == nil {
		t.Fatal("Expected non-nil cluster")
	}
	if c.self.ID != "node1" {
		t.Errorf("Expected ID node1, got %s", c.self.ID)
	}
}

func TestCluster_Members(t *testing.T) {
	c := NewCluster("node1", "localhost:8080")
	members := c.Members()
	if len(members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(members))
	}
}
