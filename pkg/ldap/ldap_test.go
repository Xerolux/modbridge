package ldap

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	config := &Config{
		URL:    "ldap://localhost:389",
		BaseDN: "dc=example,dc=com",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

func TestClient_Authenticate(t *testing.T) {
	client := &Client{config: &Config{}}
	user, err := client.Authenticate("testuser", "password")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if user == nil {
		t.Error("Expected non-nil user")
	}
}

func TestClient_Authenticate_Empty(t *testing.T) {
	client := &Client{config: &Config{}}
	_, err := client.Authenticate("", "")
	if err == nil {
		t.Error("Expected error for empty credentials")
	}
}
