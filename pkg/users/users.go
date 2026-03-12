package users

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"modbridge/pkg/database"
	"modbridge/pkg/rbac"
	"time"
)

// Manager manages users
type Manager struct {
	db *database.DB
}

// NewManager creates a new user manager
func NewManager(db *database.DB) *Manager {
	return &Manager{db: db}
}

// CreateUser creates a new user
func (m *Manager) CreateUser(username, email, password, role, createdBy, description string) (*database.User, error) {
	// Check if username already exists
	existing, _ := m.db.GetUserByUsername(username)
	if existing != nil {
		return nil, errors.New("username already exists")
	}

	// Validate role
	_, err := rbac.ParseRole(role)
	if err != nil {
		return nil, err
	}

	// Hash password
	passwordHash, err := rbac.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Generate ID
	id, _ := generateID()

	user := &database.User{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		Enabled:      true,
		CreatedBy:    createdBy,
		Description:  description,
	}

	err = m.db.CreateUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// AuthenticateUser authenticates a user
func (m *Manager) AuthenticateUser(username, password string) (*database.User, error) {
	user, err := m.db.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.Enabled {
		return nil, errors.New("user account is disabled")
	}

	if !rbac.CheckPasswordHash(password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	m.db.UpdateUserLastLogin(user.ID)

	return user, nil
}

// GetAllUsers returns all users
func (m *Manager) GetAllUsers() ([]*database.User, error) {
	return m.db.GetAllUsers()
}

// UpdateUser updates a user
func (m *Manager) UpdateUser(id, username, email, role, description string, enabled bool) error {
	user, err := m.db.GetUser(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	user.Username = username
	user.Email = email
	user.Role = role
	user.Description = description
	user.Enabled = enabled

	return m.db.UpdateUser(user)
}

// DeleteUser deletes a user
func (m *Manager) DeleteUser(id string) error {
	return m.db.DeleteUser(id)
}

// ChangePassword changes a user's password
func (m *Manager) ChangePassword(userID, oldPassword, newPassword string) error {
	user, err := m.db.GetUser(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Verify old password
	if !rbac.CheckPasswordHash(oldPassword, user.PasswordHash) {
		return errors.New("invalid old password")
	}

	// Hash new password
	newHash, err := rbac.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return m.db.UpdateUserPassword(userID, newHash)
}

func generateID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
