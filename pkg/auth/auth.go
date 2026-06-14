// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

type Session struct {
	Token     string
	UserID    string
	Username  string
	Role      string
	ExpiresAt time.Time
}

type Authenticator struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewAuthenticator() *Authenticator {
	return &Authenticator{
		sessions: make(map[string]Session),
	}
}

func HashPassword(password string) (string, error) {
	if err := ValidatePasswordStrength(password); err != nil {
		return "", err
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if len(password) > 128 {
		return errors.New("password must not exceed 128 characters")
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	typeCount := 0
	if hasUpper {
		typeCount++
	}
	if hasLower {
		typeCount++
	}
	if hasNumber {
		typeCount++
	}
	if hasSpecial {
		typeCount++
	}

	if typeCount < 3 {
		return errors.New("password must contain at least 3 of: uppercase, lowercase, numbers, special characters")
	}

	weakPasswords := []string{
		"password", "Password1!", "12345678", "qwerty",
		"admin", "letmein", "welcome", "monkey",
	}

	lowerPassword := strings.ToLower(password)
	for _, weak := range weakPasswords {
		if strings.Contains(lowerPassword, weak) {
			return errors.New("password is too common or weak")
		}
	}

	return nil
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (a *Authenticator) CreateSession(userID, username, role string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.StdEncoding.EncodeToString(b)

	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions[token] = Session{
		Token:     token,
		UserID:    userID,
		Username:  username,
		Role:      role,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	return token, nil
}

func (a *Authenticator) GetSession(token string) *Session {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if session, ok := a.sessions[token]; ok {
		copy := session
		return &copy
	}
	return nil
}

func (a *Authenticator) ValidateSession(token string) bool {
	a.mu.RLock()
	session, ok := a.sessions[token]
	a.mu.RUnlock()

	if !ok {
		return false
	}
	if time.Now().After(session.ExpiresAt) {
		a.mu.Lock()
		delete(a.sessions, token)
		a.mu.Unlock()
		return false
	}
	return true
}

func (a *Authenticator) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !a.ValidateSession(c.Value) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (a *Authenticator) CleanupExpiredSessions(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			now := time.Now()
			for token, session := range a.sessions {
				if now.After(session.ExpiresAt) {
					delete(a.sessions, token)
				}
			}
			a.mu.Unlock()
		}
	}
}
