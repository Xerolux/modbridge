package rbac

import (
	"testing"
)

func TestNewUserStore(t *testing.T) {
	store := NewUserStore()

	// Check that default admin exists
	admin, err := store.GetUser("admin")
	if err != nil {
		t.Fatalf("Expected admin user, got error: %v", err)
	}

	if admin.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", admin.Username)
	}

	if admin.Role != RoleAdmin {
		t.Errorf("Expected role '%s', got '%s'", RoleAdmin, admin.Role)
	}
}

func TestUserStore_CreateUser(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
	}

	if user.Role != RoleOperator {
		t.Errorf("Expected role '%s', got '%s'", RoleOperator, user.Role)
	}

	// Check user can be retrieved
	retrieved, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if retrieved.ID != user.ID {
		t.Errorf("Expected ID '%s', got '%s'", user.ID, retrieved.ID)
	}
}

func TestUserStore_CreateDuplicateUser(t *testing.T) {
	store := NewUserStore()

	_, err := store.CreateUser("testuser", "test1@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Try to create user with same username
	_, err = store.CreateUser("testuser", "test2@example.com", RoleViewer)
	if err != ErrUserExists {
		t.Errorf("Expected ErrUserExists, got: %v", err)
	}

	// Try to create user with same email
	_, err = store.CreateUser("testuser2", "test1@example.com", RoleViewer)
	if err != ErrUserExists {
		t.Errorf("Expected ErrUserExists for duplicate email, got: %v", err)
	}
}

func TestUserStore_GetUserNotFound(t *testing.T) {
	store := NewUserStore()

	_, err := store.GetUser("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got: %v", err)
	}
}

func TestUserStore_UpdateUser(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update email
	err = store.UpdateUser(user.ID, func(u *User) error {
		u.Email = "newemail@example.com"
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify update
	updated, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated user: %v", err)
	}

	if updated.Email != "newemail@example.com" {
		t.Errorf("Expected email 'newemail@example.com', got '%s'", updated.Email)
	}
}

func TestUserStore_DeleteUser(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	err = store.DeleteUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify deletion
	_, err = store.GetUser(user.ID)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got: %v", err)
	}
}

func TestUserStore_DeleteAdmin(t *testing.T) {
	store := NewUserStore()

	err := store.DeleteUser("admin")
	if err == nil {
		t.Error("Expected error when deleting admin user")
	}
}

func TestUserStore_ListUsers(t *testing.T) {
	store := NewUserStore()

	// Create some test users
	store.CreateUser("user1", "user1@example.com", RoleOperator)
	store.CreateUser("user2", "user2@example.com", RoleViewer)

	users := store.ListUsers()

	// Should have admin + 2 created users = 3 total
	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}
}

func TestUserStore_AuthenticateUser(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Authenticate with correct credentials
	authUser, err := store.AuthenticateUser("testuser", "password")
	if err != nil {
		t.Errorf("Authentication failed: %v", err)
	}

	if authUser.ID != user.ID {
		t.Errorf("Expected user ID '%s', got '%s'", user.ID, authUser.ID)
	}

	// Try to authenticate non-existent user
	_, err = store.AuthenticateUser("nonexistent", "password")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestUserStore_ChangeUserRole(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	err = store.ChangeUserRole(user.ID, RoleViewer)
	if err != nil {
		t.Fatalf("Failed to change role: %v", err)
	}

	updated, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if updated.Role != RoleViewer {
		t.Errorf("Expected role '%s', got '%s'", RoleViewer, updated.Role)
	}
}

func TestUserStore_SetUserActive(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Deactivate user
	err = store.SetUserActive(user.ID, false)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	updated, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if updated.Active {
		t.Error("Expected user to be inactive")
	}

	// Reactivate user
	err = store.SetUserActive(user.ID, true)
	if err != nil {
		t.Fatalf("Failed to reactivate user: %v", err)
	}

	updated, err = store.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if !updated.Active {
		t.Error("Expected user to be active")
	}
}

func TestUserStore_GenerateAPIToken(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	token, err := store.GenerateAPIToken(user.ID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Verify token exists
	updated, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if len(updated.APITokens) != 1 {
		t.Errorf("Expected 1 token, got %d", len(updated.APITokens))
	}
}

func TestUserStore_ValidateAPIToken(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	token, err := store.GenerateAPIToken(user.ID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate valid token
	validUser, err := store.ValidateAPIToken(token)
	if err != nil {
		t.Errorf("Token validation failed: %v", err)
	}

	if validUser.ID != user.ID {
		t.Errorf("Expected user ID '%s', got '%s'", user.ID, validUser.ID)
	}

	// Try invalid token
	_, err = store.ValidateAPIToken("invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got: %v", err)
	}
}

func TestUserStore_RevokeAPIToken(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	token, err := store.GenerateAPIToken(user.ID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	err = store.RevokeAPIToken(user.ID, token)
	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// Verify token no longer works
	_, err = store.ValidateAPIToken(token)
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got: %v", err)
	}
}

func TestUserStore_CheckPermission(t *testing.T) {
	store := NewUserStore()

	adminUser, _ := store.GetUser("admin")
	viewerUser, err := store.CreateUser("viewer", "viewer@example.com", RoleViewer)
	if err != nil {
		t.Fatalf("Failed to create viewer: %v", err)
	}

	// Admin should have all permissions
	if !store.CheckPermission(adminUser.ID, PermProxyCreate) {
		t.Error("Expected admin to have PermProxyCreate")
	}

	// Viewer should only have view permissions
	if store.CheckPermission(viewerUser.ID, PermProxyCreate) {
		t.Error("Expected viewer to not have PermProxyCreate")
	}

	if !store.CheckPermission(viewerUser.ID, PermProxyView) {
		t.Error("Expected viewer to have PermProxyView")
	}
}

func TestUserStore_GetStats(t *testing.T) {
	store := NewUserStore()

	// Create some users
	_, _ = store.CreateUser("user1", "user1@example.com", RoleOperator)
	user2, _ := store.CreateUser("user2", "user2@example.com", RoleViewer)

	// Deactivate user2
	store.SetUserActive(user2.ID, false)

	stats := store.GetStats()

	if stats.TotalUsers != 3 { // admin + 2 created
		t.Errorf("Expected 3 total users, got %d", stats.TotalUsers)
	}

	if stats.ActiveUsers != 2 { // admin + user1
		t.Errorf("Expected 2 active users, got %d", stats.ActiveUsers)
	}

	if stats.InactiveUsers != 1 { // user2
		t.Errorf("Expected 1 inactive user, got %d", stats.InactiveUsers)
	}
}

func TestUserStore_UpdateLastLogin(t *testing.T) {
	store := NewUserStore()

	user, err := store.CreateUser("testuser", "test@example.com", RoleOperator)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.LastLogin != nil {
		t.Error("Expected LastLogin to be nil initially")
	}

	err = store.UpdateLastLogin(user.ID)
	if err != nil {
		t.Fatalf("Failed to update last login: %v", err)
	}

	updated, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if updated.LastLogin == nil {
		t.Error("Expected LastLogin to be set")
	}
}

func TestGetRolePermissions(t *testing.T) {
	perms := GetRolePermissions(RoleAdmin)

	if len(perms) == 0 {
		t.Error("Expected admin to have permissions")
	}

	// Admin should have all permissions
	expectedPerms := []Permission{
		PermProxyView, PermProxyCreate, PermProxyEdit, PermProxyDelete, PermProxyControl,
		PermDeviceView, PermDeviceEdit, PermDeviceDelete,
		PermConfigView, PermConfigEdit, PermConfigExport, PermConfigImport,
		PermSystemView, PermSystemManage, PermSystemRestart,
		PermUserView, PermUserCreate, PermUserEdit, PermUserDelete,
		PermAuditView, PermAuditExport,
		PermLogsView, PermLogsExport,
	}

	for _, expected := range expectedPerms {
		found := false
		for _, perm := range perms {
			if perm == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected admin to have %s permission", expected)
		}
	}
}

func TestRoleDescription(t *testing.T) {
	tests := []struct {
		role           Role
		hasDescription bool
	}{
		{RoleAdmin, true},
		{RoleOperator, true},
		{RoleViewer, true},
		{RoleAuditor, true},
		{Role("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			desc := RoleDescription(tt.role)
			hasDesc := desc != "Unknown role"

			if tt.hasDescription && !hasDesc {
				t.Error("Expected description for role")
			}

			if !tt.hasDescription && hasDesc {
				t.Error("Expected no description for invalid role")
			}
		})
	}
}

func TestPermissionDescription(t *testing.T) {
	desc := PermissionDescription(PermProxyView)

	if desc == "" {
		t.Error("Expected non-empty description")
	}

	if desc == "Unknown permission" {
		t.Error("Expected valid description for PermProxyView")
	}
}

func TestCheckPermission(t *testing.T) {
	// Test admin has all permissions
	for _, perm := range GetRolePermissions(RoleAdmin) {
		if !HasPermission(RoleAdmin, perm) {
			t.Errorf("Expected admin to have %s", perm)
		}
	}

	// Test viewer has limited permissions
	if HasPermission(RoleViewer, PermProxyCreate) {
		t.Error("Expected viewer to not have PermProxyCreate")
	}

	if !HasPermission(RoleViewer, PermProxyView) {
		t.Error("Expected viewer to have PermProxyView")
	}
}

func TestParseRole(t *testing.T) {
	tests := []struct {
		roleStr    string
		expected   Role
		shouldFail bool
	}{
		{"admin", RoleAdmin, false},
		{"ADMIN", RoleAdmin, false},
		{"operator", RoleOperator, false},
		{"viewer", RoleViewer, false},
		{"auditor", RoleAuditor, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.roleStr, func(t *testing.T) {
			role, err := ParseRole(tt.roleStr)

			if tt.shouldFail {
				if err == nil {
					t.Error("Expected error for invalid role")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if role != tt.expected {
					t.Errorf("Expected role %s, got %s", tt.expected, role)
				}
			}
		})
	}
}
