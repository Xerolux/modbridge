package rbac

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrUserNotFound is returned when a user doesn't exist
	ErrUserNotFound = errors.New("user not found")
	// ErrUserExists is returned when trying to create a duplicate user
	ErrUserExists = errors.New("user already exists")
	// ErrInvalidCredentials is returned when credentials are invalid
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrAccessDenied is returned when access is denied
	ErrAccessDenied = errors.New("access denied")
	// ErrInvalidToken is returned when a token is invalid
	ErrInvalidToken = errors.New("invalid token")
)

// User represents a system user
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	Active       bool      `json:"active"`
	APITokens    []string  `json:"api_tokens,omitempty"`
	Permissions  []Permission `json:"permissions,omitempty"`
}

// UserStore manages users
type UserStore struct {
	users       map[string]*User
	usersByName map[string]*User
	mu          sync.RWMutex
}

// NewUserStore creates a new user store
func NewUserStore() *UserStore {
	store := &UserStore{
		users:       make(map[string]*User),
		usersByName: make(map[string]*User),
	}

	// Create default admin user
	admin := &User{
		ID:        "admin",
		Username:  "admin",
		Email:     "admin@modbridge.local",
		Role:      RoleAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	store.users[admin.ID] = admin
	store.usersByName[admin.Username] = admin

	return store
}

// CreateUser creates a new user
func (s *UserStore) CreateUser(username, email string, role Role) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if username already exists
	if _, exists := s.usersByName[username]; exists {
		return nil, ErrUserExists
	}

	// Check if email already exists
	for _, user := range s.users {
		if user.Email == email {
			return nil, ErrUserExists
		}
	}

	// Generate user ID
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	user := &User{
		ID:          id,
		Username:    username,
		Email:       email,
		Role:        role,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Active:      true,
		Permissions: RolePermissions[role],
	}

	s.users[id] = user
	s.usersByName[username] = user

	return user, nil
}

// GetUser retrieves a user by ID
func (s *UserStore) GetUser(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	// Return a copy to avoid concurrent modification
	copy := *user
	return &copy, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserStore) GetUserByUsername(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.usersByName[username]
	if !ok {
		return nil, ErrUserNotFound
	}

	copy := *user
	return &copy, nil
}

// UpdateUser updates an existing user
func (s *UserStore) UpdateUser(id string, updates func(*User) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}

	if err := updates(user); err != nil {
		return err
	}

	user.UpdatedAt = time.Now()

	// Update permissions based on role
	user.Permissions = RolePermissions[user.Role]

	return nil
}

// DeleteUser deletes a user
func (s *UserStore) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}

	// Prevent deletion of default admin
	if user.ID == "admin" {
		return errors.New("cannot delete default admin user")
	}

	delete(s.users, id)
	delete(s.usersByName, user.Username)

	return nil
}

// ListUsers returns all users
func (s *UserStore) ListUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		copy := *user
		users = append(users, &copy)
	}

	return users
}

// AuthenticateUser authenticates a user by username/password
func (s *UserStore) AuthenticateUser(username, password string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.usersByName[username]
	if !ok {
		return nil, ErrInvalidCredentials
	}

	if !user.Active {
		return nil, ErrInvalidCredentials
	}

	// TODO: Implement proper password verification
	// For now, we'll just check if user exists
	// In production, you'd verify bcrypt hash

	return user, nil
}

// ChangeUserRole changes a user's role
func (s *UserStore) ChangeUserRole(userID string, newRole Role) error {
	return s.UpdateUser(userID, func(u *User) error {
		u.Role = newRole
		u.Permissions = RolePermissions[newRole]
		return nil
	})
}

// SetUserActive sets a user's active status
func (s *UserStore) SetUserActive(userID string, active bool) error {
	return s.UpdateUser(userID, func(u *User) error {
		u.Active = active
		return nil
	})
}

// UpdateLastLogin updates the last login time for a user
func (s *UserStore) UpdateLastLogin(userID string) error {
	return s.UpdateUser(userID, func(u *User) error {
		now := time.Now()
		u.LastLogin = &now
		return nil
	})
}

// GenerateAPIToken generates a new API token for a user
func (s *UserStore) GenerateAPIToken(userID string) (string, error) {
	token, err := generateSecureToken()
	if err != nil {
		return "", err
	}

	err = s.UpdateUser(userID, func(u *User) error {
		u.APITokens = append(u.APITokens, token)
		return nil
	})

	if err != nil {
		return "", err
	}

	return token, nil
}

// RevokeAPIToken revokes an API token
func (s *UserStore) RevokeAPIToken(userID, token string) error {
	return s.UpdateUser(userID, func(u *User) error {
		for i, t := range u.APITokens {
			if t == token {
				u.APITokens = append(u.APITokens[:i], u.APITokens[i+1:]...)
				break
			}
		}
		return nil
	})
}

// ValidateAPIToken validates an API token and returns the associated user
func (s *UserStore) ValidateAPIToken(token string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if !user.Active {
			continue
		}

		for _, t := range user.APITokens {
			if t == token {
				copy := *user
				return &copy, nil
			}
		}
	}

	return nil, ErrInvalidToken
}

// CheckPermission checks if a user has a specific permission
func (s *UserStore) CheckPermission(userID string, permission Permission) bool {
	user, err := s.GetUser(userID)
	if err != nil {
		return false
	}

	if !user.Active {
		return false
	}

	return HasPermission(user.Role, permission)
}

// GetPermissions returns all permissions for a user
func (s *UserStore) GetPermissions(userID string) ([]Permission, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}

	return user.Permissions, nil
}

// generateID generates a unique ID
func generateID() (string, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generateSecureToken generates a secure random token
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// UserContextKey is the context key for user information
type UserContextKey struct{}

// UserFromContext extracts the user from context
func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(UserContextKey{}).(*User)
	return user, ok
}

// ContextWithUser adds a user to the context
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, UserContextKey{}, user)
}

// PermissionCheckResult represents the result of a permission check
type PermissionCheckResult struct {
	Allowed   bool     `json:"allowed"`
	Permission Permission `json:"permission"`
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Role      Role     `json:"role"`
	DeniedAt  time.Time `json:"denied_at,omitempty"`
}

// RequirePermission creates a permission check middleware
type RequirePermission struct {
	store      *UserStore
	permission Permission
}

// NewRequirePermission creates a new permission requirement
func NewRequirePermission(store *UserStore, permission Permission) *RequirePermission {
	return &RequirePermission{
		store:      store,
		permission: permission,
	}
}

// Check checks if the given user has the required permission
func (rp *RequirePermission) Check(user *User) bool {
	if !user.Active {
		return false
	}

	for _, p := range user.Permissions {
		if p == rp.permission {
			return true
		}
	}

	return false
}

// CheckPermission is a convenience function to check permissions
func CheckPermission(store *UserStore, userID string, permission Permission) (*PermissionCheckResult, error) {
	user, err := store.GetUser(userID)
	if err != nil {
		return nil, err
	}

	result := &PermissionCheckResult{
		Permission: permission,
		UserID:     user.ID,
		Username:   user.Username,
		Role:       user.Role,
	}

	hasPerm := false
	for _, p := range user.Permissions {
		if p == permission {
			hasPerm = true
			break
		}
	}

	result.Allowed = hasPerm

	if !hasPerm {
		now := time.Now()
		result.DeniedAt = now
	}

	return result, nil
}

// RoleDescription returns a description of a role
func RoleDescription(role Role) string {
	switch role {
	case RoleAdmin:
		return "Full system access including user management"
	case RoleOperator:
		return "Operational access to control proxies and devices"
	case RoleViewer:
		return "Read-only access to view system state"
	case RoleAuditor:
		return "Read-only access with audit log export capabilities"
	default:
		return "Unknown role"
	}
}

// PermissionDescription returns a description of a permission
func PermissionDescription(permission Permission) string {
	descriptions := map[Permission]string{
		PermProxyView:    "View proxy configurations and status",
		PermProxyCreate:  "Create new proxy configurations",
		PermProxyEdit:    "Edit existing proxy configurations",
		PermProxyDelete:  "Delete proxy configurations",
		PermProxyControl: "Start, stop, and control proxy instances",

		PermDeviceView:   "View device information and history",
		PermDeviceEdit:   "Edit device names and properties",
		PermDeviceDelete: "Delete device records",

		PermConfigView:   "View system configuration",
		PermConfigEdit:   "Modify system configuration",
		PermConfigExport: "Export configuration to file",
		PermConfigImport: "Import configuration from file",

		PermSystemView:    "View system status and metrics",
		PermSystemManage:  "Manage system settings",
		PermSystemRestart: "Restart the system",

		PermUserView:   "View user accounts",
		PermUserCreate: "Create new user accounts",
		PermUserEdit:   "Edit user accounts",
		PermUserDelete: "Delete user accounts",

		PermAuditView:   "View audit logs",
		PermAuditExport: "Export audit logs",

		PermLogsView:   "View system logs",
		PermLogsExport: "Export system logs",
	}

	if desc, ok := descriptions[permission]; ok {
		return desc
	}

	return "Unknown permission"
}

// GetRolePermissions returns all permissions for a role
func GetRolePermissions(role Role) []Permission {
	if perms, ok := RolePermissions[role]; ok {
		result := make([]Permission, len(perms))
		copy(result, perms)
		return result
	}
	return []Permission{}
}

// UserStats represents statistics about users
type UserStats struct {
	TotalUsers    int            `json:"total_users"`
	ActiveUsers   int            `json:"active_users"`
	InactiveUsers int            `json:"inactive_users"`
	UsersByRole   map[string]int `json:"users_by_role"`
}

// GetStats returns user statistics
func (s *UserStore) GetStats() *UserStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &UserStats{
		TotalUsers:  len(s.users),
		UsersByRole: make(map[string]int),
	}

	for _, user := range s.users {
		if user.Active {
			stats.ActiveUsers++
		} else {
			stats.InactiveUsers++
		}

		stats.UsersByRole[string(user.Role)]++
	}

	return stats
}
