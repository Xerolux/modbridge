package database_test

import (
	"path/filepath"
	"testing"
	"time"

	"modbridge/pkg/auth"
	"modbridge/pkg/database"
)

func TestAccountRecoveryChangesLoginAndConsumesToken(t *testing.T) {
	db, err := database.NewDB(filepath.Join(t.TempDir(), "recovery.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	user := &database.User{
		ID:           "admin-id",
		Username:     "forgotten-admin",
		FullName:     "Administrator",
		Email:        "admin@example.invalid",
		PasswordHash: "old-hash",
		Role:         "admin",
		Enabled:      false,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token := "local-one-time-recovery-token"
	if err := db.CreateAccountRecovery(user.ID, auth.HashRecoveryToken(token), time.Now().Add(15*time.Minute)); err != nil {
		t.Fatalf("CreateAccountRecovery: %v", err)
	}
	available, err := db.AccountRecoveryAvailable()
	if err != nil || !available {
		t.Fatalf("AccountRecoveryAvailable = %v, %v", available, err)
	}

	newHash, err := auth.HashPassword("NewSecurePassword42!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	userID, err := db.RecoverAdminAccount(auth.HashRecoveryToken(token), "new-admin", newHash)
	if err != nil {
		t.Fatalf("RecoverAdminAccount: %v", err)
	}
	if userID != user.ID {
		t.Fatalf("recovered user ID = %q, want %q", userID, user.ID)
	}

	recovered, err := db.GetUserByUsername("new-admin")
	if err != nil || recovered == nil {
		t.Fatalf("GetUserByUsername: %v, %v", recovered, err)
	}
	if !recovered.Enabled || recovered.MustChangePassword || recovered.ExpiresAt != nil {
		t.Fatalf("recovered account state is incorrect: %+v", recovered)
	}
	if !auth.CheckPasswordHash("NewSecurePassword42!", recovered.PasswordHash) {
		t.Fatal("new password was not stored")
	}

	available, err = db.AccountRecoveryAvailable()
	if err != nil || available {
		t.Fatalf("recovery token was not consumed: available=%v err=%v", available, err)
	}
	if _, err := db.RecoverAdminAccount(auth.HashRecoveryToken(token), "another-name", newHash); err == nil {
		t.Fatal("consumed recovery token was accepted a second time")
	}
}

func TestExpiredAccountRecoveryIsRejected(t *testing.T) {
	db, err := database.NewDB(filepath.Join(t.TempDir(), "expired-recovery.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	user := &database.User{
		ID:           "admin-id",
		Username:     "admin",
		FullName:     "Administrator",
		Email:        "admin@example.invalid",
		PasswordHash: "old-hash",
		Role:         "admin",
		Enabled:      true,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := db.CreateAccountRecovery(user.ID, auth.HashRecoveryToken("expired"), time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("CreateAccountRecovery: %v", err)
	}
	if _, err := db.RecoverAdminAccount(auth.HashRecoveryToken("expired"), "admin", "new-hash"); err == nil {
		t.Fatal("expired recovery token was accepted")
	}
}
