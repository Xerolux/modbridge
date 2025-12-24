package rbac

import (
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role represents a user role.
type Role string

const (
	// RoleAdmin has full access to all resources.
	RoleAdmin Role = "admin"
	// RoleOperator can view and manage proxies but not users.
	RoleOperator Role = "operator"
	// RoleViewer can only view resources.
	RoleViewer Role = "viewer"
)

// Permission represents a specific permission.
type Permission string

const (
	PermissionViewProxies   Permission = "view_proxies"
	PermissionManageProxies Permission = "manage_proxies"
	PermissionViewDevices   Permission = "view_devices"
	PermissionManageDevices Permission = "manage_devices"
	PermissionViewUsers     Permission = "view_users"
	PermissionManageUsers   Permission = "manage_users"
	PermissionViewLogs      Permission = "view_logs"
	PermissionViewMetrics   Permission = "view_metrics"
	PermissionBackupConfig  Permission = "backup_config"
	PermissionRestoreConfig Permission = "restore_config"
)

// User represents a system user.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialize password hash
	Role         Role      `json:"role"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
}

// UserManager manages users and roles.
type UserManager struct {
	users map[string]*User // username -> user
	mu    sync.RWMutex
}

// NewUserManager creates a new user manager.
func NewUserManager() *UserManager {
	return &UserManager{
		users: make(map[string]*User),
	}
}

// CreateUser creates a new user.
func (um *UserManager) CreateUser(username, email, password string, role Role) (*User, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if _, exists := um.users[username]; exists {
		return nil, errors.New("user already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           generateID(),
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	um.users[username] = user
	return user, nil
}

// GetUser retrieves a user by username.
func (um *UserManager) GetUser(username string) (*User, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	user, exists := um.users[username]
	if !exists {
		return nil, errors.New("user not found")
	}

	return user, nil
}

// ListUsers returns all users.
func (um *UserManager) ListUsers() []*User {
	um.mu.RLock()
	defer um.mu.RUnlock()

	users := make([]*User, 0, len(um.users))
	for _, user := range um.users {
		users = append(users, user)
	}
	return users
}

// UpdateUser updates a user's information.
func (um *UserManager) UpdateUser(username string, email string, role Role, enabled bool) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[username]
	if !exists {
		return errors.New("user not found")
	}

	user.Email = email
	user.Role = role
	user.Enabled = enabled
	user.UpdatedAt = time.Now()

	return nil
}

// DeleteUser deletes a user.
func (um *UserManager) DeleteUser(username string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	if _, exists := um.users[username]; !exists {
		return errors.New("user not found")
	}

	delete(um.users, username)
	return nil
}

// ChangePassword changes a user's password.
func (um *UserManager) ChangePassword(username, oldPassword, newPassword string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[username]
	if !exists {
		return errors.New("user not found")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("invalid old password")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()

	return nil
}

// Authenticate authenticates a user.
func (um *UserManager) Authenticate(username, password string) (*User, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[username]
	if !exists {
		return nil, errors.New("invalid credentials")
	}

	if !user.Enabled {
		return nil, errors.New("user is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	user.LastLogin = time.Now()
	return user, nil
}

// HasPermission checks if a role has a specific permission.
func HasPermission(role Role, permission Permission) bool {
	permissions := rolePermissions[role]
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// rolePermissions maps roles to their permissions.
var rolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermissionViewProxies,
		PermissionManageProxies,
		PermissionViewDevices,
		PermissionManageDevices,
		PermissionViewUsers,
		PermissionManageUsers,
		PermissionViewLogs,
		PermissionViewMetrics,
		PermissionBackupConfig,
		PermissionRestoreConfig,
	},
	RoleOperator: {
		PermissionViewProxies,
		PermissionManageProxies,
		PermissionViewDevices,
		PermissionManageDevices,
		PermissionViewLogs,
		PermissionViewMetrics,
		PermissionBackupConfig,
	},
	RoleViewer: {
		PermissionViewProxies,
		PermissionViewDevices,
		PermissionViewLogs,
		PermissionViewMetrics,
	},
}

// generateID generates a unique ID for a user.
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
