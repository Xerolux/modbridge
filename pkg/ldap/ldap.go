package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

// Config represents LDAP configuration
type Config struct {
	Enabled          bool              `json:"enabled"`
	ServerURL        string            `json:"server_url"`
	BaseDN           string            `json:"base_dn"`
	BindDN           string            `json:"bind_dn"`
	BindPassword     string            `json:"bind_password"`
	UserSearchFilter string            `json:"user_search_filter"`
	UserSearchBase   string            `json:"user_search_base"`
	GroupSearchBase  string            `json:"group_search_base"`
	Port             int               `json:"port"`
	UseSSL           bool              `json:"use_ssl"`
	UseStartTLS      bool              `json:"use_start_tls"`
	SkipCertVerify   bool              `json:"skip_cert_verify"`
	RoleMapping      map[string]string `json:"role_mapping"`
}

// Client represents an LDAP client
type Client struct {
	config *Config
	conn   *ldap.Conn
}

// NewClient creates a new LDAP client
func NewClient(config *Config) *Client {
	return &Client{
		config: config,
	}
}

// Connect connects to the LDAP server
func (c *Client) Connect() error {
	if !c.config.Enabled {
		return errors.New("LDAP is not enabled")
	}

	address := fmt.Sprintf("%s:%d", c.config.ServerURL, c.config.Port)

	var err error
	if c.config.UseSSL {
		c.conn, err = ldap.DialTLS("tcp", address, &tls.Config{
			InsecureSkipVerify: c.config.SkipCertVerify,
		})
	} else {
		c.conn, err = ldap.Dial("tcp", address)
		if err == nil && c.config.UseStartTLS {
			err = c.conn.StartTLS(&tls.Config{
				InsecureSkipVerify: c.config.SkipCertVerify,
			})
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	// Bind with service account
	if c.config.BindDN != "" {
		err = c.conn.Bind(c.config.BindDN, c.config.BindPassword)
		if err != nil {
			c.conn.Close()
			return fmt.Errorf("failed to bind to LDAP server: %w", err)
		}
	}

	return nil
}

// Close closes the LDAP connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// AuthenticateUser authenticates a user against LDAP
func (c *Client) AuthenticateUser(username, password string) (bool, string, error) {
	if !c.config.Enabled {
		return false, "", errors.New("LDAP is not enabled")
	}

	// Search for user
	searchFilter := fmt.Sprintf(c.config.UserSearchFilter, username)
	searchReq := ldap.NewSearchRequest(
		c.config.UserSearchBase,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		searchFilter,
		[]string{"dn", "cn", "mail", "memberOf"},
		nil,
	)

	sr, err := c.conn.Search(searchReq)
	if err != nil {
		return false, "", fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(sr.Entries) == 0 {
		return false, "", errors.New("user not found")
	}

	if len(sr.Entries) > 1 {
		return false, "", errors.New("multiple users found")
	}

	userDN := sr.Entries[0].DN

	// Authenticate user
	userConn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", c.config.ServerURL, c.config.Port))
	if err != nil {
		return false, "", err
	}
	defer userConn.Close()

	if c.config.UseSSL {
		err = userConn.Bind(userDN, password)
	} else if c.config.UseStartTLS {
		err = userConn.StartTLS(&tls.Config{InsecureSkipVerify: c.config.SkipCertVerify})
		if err != nil {
			return false, "", err
		}
		err = userConn.Bind(userDN, password)
	} else {
		err = userConn.Bind(userDN, password)
	}

	if err != nil {
		return false, "", errors.New("invalid credentials")
	}

	// Determine role based on group membership
	role := c.determineRole(sr.Entries[0].GetAttributeValues("memberOf"))

	return true, role, nil
}

// determineRole determines user role based on group membership
func (c *Client) determineRole(groups []string) string {
	if c.config.RoleMapping == nil {
		return "viewer" // Default role
	}

	// Check each group mapping
	for group, role := range c.config.RoleMapping {
		for _, userGroup := range groups {
			if userGroup == group || containsCN(userGroup, group) {
				return role
			}
		}
	}

	return "viewer" // Default role
}

// containsCN checks if a group DN contains a specific CN
func containsCN(dn, cn string) bool {
	// Simple check - can be improved with proper DN parsing
	return contains(dn, "CN="+cn)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetUserAttributes retrieves user attributes from LDAP
func (c *Client) GetUserAttributes(username string) (map[string]string, error) {
	searchFilter := fmt.Sprintf(c.config.UserSearchFilter, username)
	searchReq := ldap.NewSearchRequest(
		c.config.UserSearchBase,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		searchFilter,
		[]string{"cn", "mail", "displayName", "telephoneNumber"},
		nil,
	)

	sr, err := c.conn.Search(searchReq)
	if err != nil {
		return nil, err
	}

	if len(sr.Entries) == 0 {
		return nil, errors.New("user not found")
	}

	entry := sr.Entries[0]
	attrs := make(map[string]string)

	if attr := entry.GetAttributeValue("cn"); attr != "" {
		attrs["cn"] = attr
	}
	if attr := entry.GetAttributeValue("mail"); attr != "" {
		attrs["email"] = attr
	}
	if attr := entry.GetAttributeValue("displayName"); attr != "" {
		attrs["display_name"] = attr
	}
	if attr := entry.GetAttributeValue("telephoneNumber"); attr != "" {
		attrs["phone"] = attr
	}

	return attrs, nil
}

// TestConnection tests the LDAP connection
func (c *Client) TestConnection() error {
	if !c.config.Enabled {
		return errors.New("LDAP is not enabled")
	}

	err := c.Connect()
	if err != nil {
		return err
	}
	defer c.Close()

	return nil
}
