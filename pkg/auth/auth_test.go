// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHashPassword(t *testing.T) {
	password := "S7r0ngP@ssw0rd!2024"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Error("Hash is empty")
	}
	if hash == password {
		t.Error("Hash should not equal password")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "S7r0ngP@ssw0rd!2024"
	hash, _ := HashPassword(password)

	if !CheckPasswordHash(password, hash) {
		t.Error("Valid password check failed")
	}

	if CheckPasswordHash("wrongpassword", hash) {
		t.Error("Invalid password check should fail")
	}
}

func TestCreateSession(t *testing.T) {
	a := NewAuthenticator()
	token, err := a.CreateSession("user1", "testuser", "admin", 24*time.Hour, false)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if token == "" {
		t.Error("Token is empty")
	}

	if !a.ValidateSession(token) {
		t.Error("Created session should be valid")
	}

	session := a.GetSession(token)
	if session == nil {
		t.Fatal("GetSession returned nil")
	}
	if session.UserID != "user1" {
		t.Errorf("Expected UserID user1, got %s", session.UserID)
	}
	if session.Username != "testuser" {
		t.Errorf("Expected Username testuser, got %s", session.Username)
	}
	if session.Role != "admin" {
		t.Errorf("Expected Role admin, got %s", session.Role)
	}
}

func TestValidateSession(t *testing.T) {
	a := NewAuthenticator()

	if a.ValidateSession("invalid-token") {
		t.Error("Invalid token should not validate")
	}

	token, _ := a.CreateSession("u1", "user", "admin", 24*time.Hour, false)
	if !a.ValidateSession(token) {
		t.Error("Valid token should validate")
	}

	a.mu.Lock()
	a.sessions[token] = Session{
		Token:     token,
		UserID:    "u1",
		Username:  "user",
		Role:      "admin",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	a.mu.Unlock()

	if a.ValidateSession(token) {
		t.Error("Expired token should not validate")
	}
}

func TestHashPasswordUncheckedAllowsWeak(t *testing.T) {
	// "admin" violates ValidatePasswordStrength, but the unchecked variant must
	// accept it so we can seed the default admin/admin login on first run.
	hash, err := HashPasswordUnchecked("admin")
	if err != nil {
		t.Fatalf("HashPasswordUnchecked returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if !CheckPasswordHash("admin", hash) {
		t.Fatal("CheckPasswordHash failed for unchecked hash of 'admin'")
	}
}

func TestInvalidateSessionRemovesSession(t *testing.T) {
	a := NewAuthenticator()
	token, err := a.CreateSession("u1", "user", "admin", time.Hour, false)
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if a.GetSession(token) == nil {
		t.Fatal("expected session to exist")
	}
	if ok := a.InvalidateSession(token); !ok {
		t.Fatal("expected InvalidateSession to return true for existing token")
	}
	if a.GetSession(token) != nil {
		t.Fatal("expected session to be removed after InvalidateSession")
	}
	if ok := a.InvalidateSession("nonexistent"); ok {
		t.Fatal("expected InvalidateSession to return false for missing token")
	}
}

func TestCreateSessionStoresMustChangePassword(t *testing.T) {
	a := NewAuthenticator()
	token, _ := a.CreateSession("u1", "user", "admin", time.Hour, true)
	s := a.GetSession(token)
	if s == nil {
		t.Fatal("expected session")
	}
	if !s.MustChangePassword {
		t.Fatal("expected MustChangePassword=true when requested")
	}

	token2, _ := a.CreateSession("u2", "user2", "admin", time.Hour, false)
	s2 := a.GetSession(token2)
	if s2 == nil {
		t.Fatal("expected second session")
	}
	if s2.MustChangePassword {
		t.Fatal("expected MustChangePassword=false when not requested")
	}
}

func TestMiddleware(t *testing.T) {
	a := NewAuthenticator()
	token, _ := a.CreateSession("u1", "user", "admin", 24*time.Hour, false)

	handler := a.Middleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected %d, got %d", http.StatusUnauthorized, w.Code)
	}

	req = httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected %d, got %d", http.StatusOK, w.Code)
	}

	req = httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "invalid"})
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
