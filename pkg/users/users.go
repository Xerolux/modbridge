// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package users

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"modbridge/pkg/auth"
	"modbridge/pkg/database"
	"modbridge/pkg/rbac"
	"strings"
	"time"
)

type Manager struct {
	db *database.DB
}

func NewManager(db *database.DB) *Manager {
	return &Manager{db: db}
}

type CreateUserRequest struct {
	Username           string `json:"username"`
	FullName           string `json:"full_name"`
	Email              string `json:"email"`
	Password           string `json:"password"`
	Role               string `json:"role"`
	Enabled            bool   `json:"enabled"`
	AutoDeactivateDays int    `json:"auto_deactivate_days"`
	MustChangePassword bool   `json:"must_change_password"`
	Description        string `json:"description"`
}

type UpdateUserRequest struct {
	Username           *string `json:"username"`
	FullName           *string `json:"full_name"`
	Email              *string `json:"email"`
	Role               *string `json:"role"`
	Enabled            *bool   `json:"enabled"`
	AutoDeactivateDays *int    `json:"auto_deactivate_days"`
	ExpiresAt          *string `json:"expires_at"`
	Password           *string `json:"password,omitempty"`
	MustChangePassword *bool   `json:"must_change_password"`
	Description        *string `json:"description"`
}

func (m *Manager) CreateUser(req *CreateUserRequest, createdBy string) (*database.User, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.FullName = strings.TrimSpace(req.FullName)
	req.Email = strings.TrimSpace(req.Email)
	req.Description = strings.TrimSpace(req.Description)

	if req.Username == "" {
		return nil, errors.New("username is required")
	}
	if req.FullName == "" {
		return nil, errors.New("full name is required")
	}
	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if req.Password == "" {
		return nil, errors.New("password is required")
	}

	existing, _ := m.db.GetUserByUsername(req.Username)
	if existing != nil {
		return nil, errors.New("username already exists")
	}

	_, err := rbac.ParseRole(req.Role)
	if err != nil {
		return nil, err
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	id, _ := generateID()

	var expiresAt *time.Time
	if req.AutoDeactivateDays > 0 {
		t := time.Now().AddDate(0, 0, req.AutoDeactivateDays)
		expiresAt = &t
	}

	user := &database.User{
		ID:                 id,
		Username:           req.Username,
		FullName:           req.FullName,
		Email:              req.Email,
		PasswordHash:       passwordHash,
		Role:               req.Role,
		Enabled:            req.Enabled,
		AutoDeactivateDays: req.AutoDeactivateDays,
		ExpiresAt:          expiresAt,
		MustChangePassword: req.MustChangePassword,
		CreatedBy:          createdBy,
		Description:        req.Description,
	}

	err = m.db.CreateUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

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

	if user.ExpiresAt != nil && time.Now().After(*user.ExpiresAt) {
		user.Enabled = false
		if err := m.db.UpdateUser(user); err != nil {
			return nil, fmt.Errorf("failed to disable expired user: %w", err)
		}
		return nil, errors.New("user account has expired")
	}

	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	if err := m.db.UpdateUserLastLogin(user.ID); err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	return user, nil
}

func (m *Manager) GetAllUsers() ([]*database.User, error) {
	return m.db.GetAllUsers()
}

func (m *Manager) GetUser(id string) (*database.User, error) {
	if id == "" {
		return nil, errors.New("user id is required")
	}
	return m.db.GetUser(id)
}

func (m *Manager) UpdateUser(id string, req *UpdateUserRequest) error {
	user, err := m.db.GetUser(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Capture admin state before applying updates so we can enforce that the
	// last enabled administrator cannot be removed or disabled (lockout guard).
	wasEnabledAdmin := user.Enabled && user.Role == string(rbac.RoleAdmin)

	if req.Username != nil {
		trimmedUsername := strings.TrimSpace(*req.Username)
		if trimmedUsername == "" {
			return errors.New("username is required")
		}
		if trimmedUsername != user.Username {
			existing, err := m.db.GetUserByUsername(trimmedUsername)
			if err != nil {
				return err
			}
			if existing != nil && existing.ID != id {
				return errors.New("username already exists")
			}
		}
		user.Username = trimmedUsername
	}

	if req.FullName != nil {
		trimmedFullName := strings.TrimSpace(*req.FullName)
		if trimmedFullName == "" {
			return errors.New("full name is required")
		}
		user.FullName = trimmedFullName
	}

	if req.Email != nil {
		trimmedEmail := strings.TrimSpace(*req.Email)
		if trimmedEmail == "" {
			return errors.New("email is required")
		}
		user.Email = trimmedEmail
	}

	if req.Role != nil {
		_, err := rbac.ParseRole(*req.Role)
		if err != nil {
			return err
		}
		user.Role = *req.Role
	}

	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}

	if req.AutoDeactivateDays != nil {
		user.AutoDeactivateDays = *req.AutoDeactivateDays
	}

	if req.Description != nil {
		user.Description = strings.TrimSpace(*req.Description)
	}

	if req.ExpiresAt != nil {
		expiresAtValue := strings.TrimSpace(*req.ExpiresAt)
		if expiresAtValue != "" {
			t, err := time.Parse(time.RFC3339, expiresAtValue)
			if err != nil {
				return errors.New("invalid expires_at format, use RFC3339")
			}
			user.ExpiresAt = &t
		} else if req.AutoDeactivateDays != nil {
			if *req.AutoDeactivateDays > 0 {
				t := time.Now().AddDate(0, 0, *req.AutoDeactivateDays)
				user.ExpiresAt = &t
			} else {
				user.ExpiresAt = nil
			}
		}
	} else if req.AutoDeactivateDays != nil {
		if *req.AutoDeactivateDays > 0 {
			t := time.Now().AddDate(0, 0, *req.AutoDeactivateDays)
			user.ExpiresAt = &t
		} else {
			user.ExpiresAt = nil
		}
	}

	if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			return err
		}
		user.PasswordHash = hash
		// When an admin sets a password, default to requiring a change on next
		// login unless the request explicitly clears the flag.
		if req.MustChangePassword == nil {
			user.MustChangePassword = true
		}
	}

	if req.MustChangePassword != nil {
		user.MustChangePassword = *req.MustChangePassword
	}

	// Lockout guard: never let the last enabled admin be demoted or disabled.
	willBeEnabledAdmin := user.Enabled && user.Role == string(rbac.RoleAdmin)
	if wasEnabledAdmin && !willBeEnabledAdmin {
		count, err := m.CountEnabledAdmins()
		if err != nil {
			return err
		}
		if count <= 1 {
			return errors.New("cannot remove or disable the last administrator")
		}
	}

	return m.db.UpdateUser(user)
}

func (m *Manager) DeleteUser(id string) error {
	user, err := m.db.GetUser(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Lockout guard: never delete the last enabled admin.
	if user.Enabled && user.Role == string(rbac.RoleAdmin) {
		count, err := m.CountEnabledAdmins()
		if err != nil {
			return err
		}
		if count <= 1 {
			return errors.New("cannot delete the last administrator")
		}
	}

	return m.db.DeleteUser(id)
}

func (m *Manager) SetUserEnabled(id string, enabled bool) error {
	user, err := m.db.GetUser(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Lockout guard: never disable the last enabled administrator.
	if !enabled && user.Enabled && user.Role == string(rbac.RoleAdmin) {
		count, err := m.CountEnabledAdmins()
		if err != nil {
			return err
		}
		if count <= 1 {
			return errors.New("cannot disable the last administrator")
		}
	}

	user.Enabled = enabled
	return m.db.UpdateUser(user)
}

func (m *Manager) ChangePassword(userID, oldPassword, newPassword string) error {
	user, err := m.db.GetUser(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	if !auth.CheckPasswordHash(oldPassword, user.PasswordHash) {
		return errors.New("invalid old password")
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return m.db.UpdateUserPassword(userID, newHash)
}

func (m *Manager) DeactivateExpiredUsers() (int, error) {
	return m.db.DeactivateExpiredUsers()
}

// CountEnabledAdmins returns the number of currently enabled administrators.
func (m *Manager) CountEnabledAdmins() (int, error) {
	return m.db.CountEnabledAdmins()
}

// AdminResetPassword sets a new password for a user. When mustChange is true,
// the user is forced to pick a new password on next login. This bypasses the
// old-password check (admin-only operation).
func (m *Manager) AdminResetPassword(userID, newPassword string, mustChange bool) error {
	user, err := m.db.GetUser(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return m.db.AdminResetUserPassword(userID, hash, mustChange)
}

// EnsureDefaultAdmin creates the initial administrator account when the user
// store is empty (first run in multi-user mode). It returns the generated
// plaintext password so the caller can log and display it; the account is
// marked must_change_password so the operator sets a personal password ASAP.
// The provided hashPassword function is used so bootstrap can reuse the
// application's crypto/rand-based generator without import cycles.
func (m *Manager) EnsureDefaultAdmin(username, password, createdBy string) (bool, error) {
	existing, err := m.db.GetAllUsers()
	if err != nil {
		return false, err
	}
	if len(existing) > 0 {
		return false, nil // already populated
	}

	username = strings.TrimSpace(username)
	if username == "" {
		username = "admin"
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return false, err
	}

	id, _ := generateID()
	admin := &database.User{
		ID:                 id,
		Username:           username,
		FullName:           "Administrator",
		Email:              "admin@modbridge.local",
		PasswordHash:       hash,
		Role:               string(rbac.RoleAdmin),
		Enabled:            true,
		MustChangePassword: true,
		CreatedBy:          createdBy,
		Description:        "Initial administrator account",
	}
	if err := m.db.CreateUser(admin); err != nil {
		return false, err
	}
	return true, nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
