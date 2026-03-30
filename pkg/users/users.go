// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package users

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	Description        string `json:"description"`
}

type UpdateUserRequest struct {
	Username           string `json:"username"`
	FullName           string `json:"full_name"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	Enabled            bool   `json:"enabled"`
	AutoDeactivateDays int    `json:"auto_deactivate_days"`
	ExpiresAt          string `json:"expires_at"`
	Password           string `json:"password,omitempty"`
	Description        string `json:"description"`
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
		m.db.UpdateUser(user)
		return nil, errors.New("user account has expired")
	}

	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	m.db.UpdateUserLastLogin(user.ID)

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

	if req.Username != "" {
		req.Username = strings.TrimSpace(req.Username)
		if req.Username != user.Username {
			existing, err := m.db.GetUserByUsername(req.Username)
			if err != nil {
				return err
			}
			if existing != nil && existing.ID != id {
				return errors.New("username already exists")
			}
		}
		user.Username = req.Username
	}
	user.FullName = strings.TrimSpace(req.FullName)
	user.Email = strings.TrimSpace(req.Email)
	if req.Role != "" {
		_, err := rbac.ParseRole(req.Role)
		if err != nil {
			return err
		}
		user.Role = req.Role
	}
	user.Enabled = req.Enabled
	user.AutoDeactivateDays = req.AutoDeactivateDays
	user.Description = req.Description

	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return errors.New("invalid expires_at format, use RFC3339")
		}
		user.ExpiresAt = &t
	} else if req.ExpiresAt == "" && req.AutoDeactivateDays > 0 {
		t := time.Now().AddDate(0, 0, req.AutoDeactivateDays)
		user.ExpiresAt = &t
	}

	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			return err
		}
		user.PasswordHash = hash
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

func generateID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
