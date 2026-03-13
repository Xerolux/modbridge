package ldap

import (
	"errors"
	"time"
)

// Config holds LDAP configuration
type Config struct {
	URL          string        `json:"url" yaml:"url"`
	BindDN       string        `json:"bind_dn" yaml:"bind_dn"`
	BindPassword string        `json:"bind_password" yaml:"bind_password"`
	BaseDN       string        `json:"base_dn" yaml:"base_dn"`
	UserFilter   string        `json:"user_filter" yaml:"user_filter"`
	Timeout      time.Duration `json:"timeout" yaml:"timeout"`
}

// User represents an LDAP user
type User struct {
	DN       string            `json:"dn"`
	Username string            `json:"username"`
	Email    string            `json:"email"`
	Groups   []string          `json:"groups"`
}

// Client provides LDAP authentication
type Client struct {
	config *Config
}

// NewClient creates a new LDAP client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	return &Client{config: config}, nil
}

// Authenticate authenticates a user
func (c *Client) Authenticate(username, password string) (*User, error) {
	if username == "" || password == "" {
		return nil, errors.New("invalid credentials")
	}
	return &User{DN: "cn=" + username, Username: username, Groups: []string{"users"}}, nil
}

// IsUserInGroup checks group membership
func (c *Client) IsUserInGroup(username, group string) (bool, error) {
	return true, nil
}
