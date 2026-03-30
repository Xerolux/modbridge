// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package auth

import (
	"errors"
	"sync"
	"time"
)

// UserRole defines user permission levels.
type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleEditor   UserRole = "editor"
	RoleViewer   UserRole = "viewer"
	RoleReadOnly UserRole = "readonly"
)

// User represents a user in the system.
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"` // Never returned in JSON
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// UserStore manages users.
type UserStore struct {
	mu              sync.RWMutex
	users           map[string]*User
	usersByUsername map[string]*User
}

var roleHierarchy = map[UserRole]int{
	RoleAdmin:    4,
	RoleEditor:   3,
	RoleViewer:   2,
	RoleReadOnly: 1,
}

// NewUserStore creates a new user store.
func NewUserStore() *UserStore {
	return &UserStore{
		users:           make(map[string]*User),
		usersByUsername: make(map[string]*User),
	}
}

// AddUser adds a new user.
func (s *UserStore) AddUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; exists {
		return errors.New("user already exists")
	}
	if _, exists := s.usersByUsername[user.Username]; exists {
		return errors.New("user already exists")
	}

	s.users[user.ID] = user
	s.usersByUsername[user.Username] = user
	return nil
}

// RemoveUser removes a user.
func (s *UserStore) RemoveUser(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if user, exists := s.users[id]; exists {
		delete(s.usersByUsername, user.Username)
		delete(s.users, id)
	}
}

// GetUser retrieves a user by ID.
func (s *UserStore) GetUser(id string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	return user, exists
}

// GetByUsername retrieves a user by username.
func (s *UserStore) GetByUsername(username string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.usersByUsername[username]
	return user, exists
}

// GetAllUsers returns all users.
func (s *UserStore) GetAllUsers() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, *user)
	}
	return users
}

// UpdateUser updates a user.
func (s *UserStore) UpdateUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; !exists {
		return errors.New("user not found")
	}

	existing, exists := s.users[user.ID]
	if !exists {
		return errors.New("user not found")
	}

	if existing.Username != user.Username {
		if _, taken := s.usersByUsername[user.Username]; taken {
			return errors.New("user already exists")
		}
		delete(s.usersByUsername, existing.Username)
	}

	s.users[user.ID] = user
	s.usersByUsername[user.Username] = user
	return nil
}

// HasPermission checks if a user has a specific permission.
func (u *User) HasPermission(requiredRole UserRole) bool {
	if u.Role == RoleAdmin {
		return true
	}

	return roleHierarchy[u.Role] >= roleHierarchy[requiredRole]
}
