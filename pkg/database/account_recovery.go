package database

import (
	"database/sql"
	"errors"
	"time"
)

// CreateAccountRecovery replaces any prior recovery authorization with a new,
// short-lived token for one administrator account.
func (db *DB) CreateAccountRecovery(userID, tokenHash string, expiresAt time.Time) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM account_recovery`); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO account_recovery (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		tokenHash, userID, expiresAt.UTC(),
	); err != nil {
		return err
	}
	return tx.Commit()
}

// AccountRecoveryAvailable reports whether a non-expired recovery authorization exists.
func (db *DB) AccountRecoveryAvailable() (bool, error) {
	var count int
	err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM account_recovery WHERE expires_at > ?`,
		time.Now().UTC(),
	).Scan(&count)
	return count > 0, err
}

// RecoverAdminAccount atomically consumes a recovery token and changes the
// selected administrator's username and password.
func (db *DB) RecoverAdminAccount(tokenHash, username, passwordHash string) (string, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var userID string
	err = tx.QueryRow(
		`SELECT user_id FROM account_recovery WHERE token_hash = ? AND expires_at > ?`,
		tokenHash, time.Now().UTC(),
	).Scan(&userID)
	if err != nil {
		return "", errors.New("invalid or expired recovery code")
	}

	var existingID string
	err = tx.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&existingID)
	if err == nil && existingID != userID {
		return "", errors.New("username already exists")
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	result, err := tx.Exec(`
		UPDATE users
		SET username = ?, password_hash = ?, enabled = 1, expires_at = NULL,
			must_change_password = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND role = 'admin'
	`, username, passwordHash, userID)
	if err != nil {
		return "", errors.New("could not update account")
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		return "", errors.New("recovery account is no longer available")
	}

	if _, err := tx.Exec(`DELETE FROM account_recovery`); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return userID, nil
}
