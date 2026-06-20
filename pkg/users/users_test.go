// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package users

import (
	"path/filepath"
	"testing"

	"modbridge/pkg/database"
)

const strongPassword = "Str0ngP@ssw0rd!"

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	db, err := database.NewDB(filepath.Join(t.TempDir(), "users_test.db"))
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewManager(db)
}

func TestEnsureDefaultAdmin_CreatesWhenEmpty(t *testing.T) {
	m := newTestManager(t)

	created, err := m.EnsureDefaultAdmin("admin", strongPassword, "system")
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin error: %v", err)
	}
	if !created {
		t.Fatal("expected admin to be created on empty store")
	}

	user, err := m.db.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if user == nil {
		t.Fatal("default admin not found after bootstrap")
	}
	if user.Role != "admin" {
		t.Errorf("expected role admin, got %s", user.Role)
	}
	if !user.Enabled {
		t.Error("default admin should be enabled")
	}
	if !user.MustChangePassword {
		t.Error("default admin must be flagged for password change on first login")
	}
}

func TestEnsureDefaultAdmin_Idempotent(t *testing.T) {
	m := newTestManager(t)

	if _, err := m.EnsureDefaultAdmin("admin", strongPassword, "system"); err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	// Second call must NOT overwrite or create another admin.
	created, err := m.EnsureDefaultAdmin("admin", "X9diff!pas", "system")
	if err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	if created {
		t.Fatal("expected created=false on already-populated store")
	}

	all, err := m.GetAllUsers()
	if err != nil {
		t.Fatalf("GetAllUsers: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected exactly 1 user after idempotent bootstrap, got %d", len(all))
	}
}

func TestAuthenticateUser_Roundtrip(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.EnsureDefaultAdmin("operator", strongPassword, "system"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// Correct credentials → success.
	user, err := m.AuthenticateUser("operator", strongPassword)
	if err != nil || user == nil {
		t.Fatalf("expected successful auth, got err=%v user=%v", err, user)
	}

	// Wrong password → failure.
	if _, err := m.AuthenticateUser("operator", "Wr0ngP@ssw0rd!XX"); err == nil {
		t.Fatal("expected auth failure for wrong password")
	}

	// Unknown user → failure.
	if _, err := m.AuthenticateUser("ghost", strongPassword); err == nil {
		t.Fatal("expected auth failure for unknown user")
	}
}

func TestLastAdminProtection_Delete(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.EnsureDefaultAdmin("admin", strongPassword, "system"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	admin, _ := m.db.GetUserByUsername("admin")

	// Deleting the only admin must be refused.
	if err := m.DeleteUser(admin.ID); err == nil {
		t.Fatal("expected error when deleting the last administrator")
	}

	// Add a second admin, then deletion of one must succeed.
	secondID, _ := generateID()
	m.db.CreateUser(&database.User{
		ID: secondID, Username: "admin2", FullName: "Admin Two",
		Email: "a2@modbridge.local", PasswordHash: "x", Role: "admin", Enabled: true,
	})
	if err := m.DeleteUser(admin.ID); err != nil {
		t.Fatalf("expected deletion to succeed with a second admin present, got: %v", err)
	}
}

func TestLastAdminProtection_DemoteOrDisable(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.EnsureDefaultAdmin("admin", strongPassword, "system"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	admin, _ := m.db.GetUserByUsername("admin")

	// Demoting the last admin must be refused.
	demote := "viewer"
	if err := m.UpdateUser(admin.ID, &UpdateUserRequest{Role: &demote}); err == nil {
		t.Fatal("expected error when demoting the last administrator")
	}

	// Disabling the last admin must be refused.
	disable := false
	if err := m.UpdateUser(admin.ID, &UpdateUserRequest{Enabled: &disable}); err == nil {
		t.Fatal("expected error when disabling the last administrator")
	}
}

func TestUpdateUser_AdminSetPasswordForcesChange(t *testing.T) {
	m := newTestManager(t)
	// Create a non-admin user.
	_, err := m.CreateUser(&CreateUserRequest{
		Username: "viewer", FullName: "Viewer", Email: "v@modbridge.local",
		Password: strongPassword, Role: "viewer", Enabled: true,
	}, "admin")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	user, _ := m.db.GetUserByUsername("viewer")

	// Admin resets the password without explicitly setting MustChangePassword.
	newPwd := "An0ther!Strong1"
	if err := m.UpdateUser(user.ID, &UpdateUserRequest{Password: &newPwd}); err != nil {
		t.Fatalf("UpdateUser password reset failed: %v", err)
	}

	updated, _ := m.db.GetUserByUsername("viewer")
	if !updated.MustChangePassword {
		t.Fatal("expected must_change_password=true after admin password reset")
	}

	// Explicitly clearing the flag must be honored.
	off := false
	if err := m.UpdateUser(user.ID, &UpdateUserRequest{MustChangePassword: &off}); err != nil {
		t.Fatalf("UpdateUser clear flag failed: %v", err)
	}
	updated, _ = m.db.GetUserByUsername("viewer")
	if updated.MustChangePassword {
		t.Fatal("expected must_change_password=false after explicit clear")
	}
}

func TestCountEnabledAdmins(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.EnsureDefaultAdmin("admin", strongPassword, "system"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	count, err := m.CountEnabledAdmins()
	if err != nil {
		t.Fatalf("CountEnabledAdmins: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 admin, got %d", count)
	}
}
