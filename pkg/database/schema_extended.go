package database

import (
	"database/sql"
	"time"
)

// initExtendedSchema creates additional tables for multi-user, audit, versioning, etc.
func (db *DB) initExtendedSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer',
		enabled BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login DATETIME,
		created_by TEXT,
		description TEXT
	);

	CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		user_id TEXT,
		username TEXT,
		action TEXT NOT NULL,
		resource_type TEXT,
		resource_id TEXT,
		details TEXT,
		ip_address TEXT,
		user_agent TEXT,
		success BOOLEAN DEFAULT 1,
		error_message TEXT
	);

	CREATE TABLE IF NOT EXISTS config_versions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version INTEGER NOT NULL,
		config_hash TEXT NOT NULL,
		config_data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		change_description TEXT,
		is_rollback BOOLEAN DEFAULT 0,
		parent_version INTEGER,
		FOREIGN KEY (created_by) REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id);
	CREATE INDEX IF NOT EXISTS idx_config_versions_version ON config_versions(version DESC);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// User represents a user in the system
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	Enabled      bool       `json:"enabled"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	CreatedBy    string     `json:"created_by,omitempty"`
	Description  string     `json:"description,omitempty"`
}

// CreateUser creates a new user
func (db *DB) CreateUser(user *User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, role, enabled, created_by, description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.conn.Exec(query,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.Role, user.Enabled, user.CreatedBy, user.Description)
	return err
}

// GetUser retrieves a user by ID
func (db *DB) GetUser(id string) (*User, error) {
	query := `SELECT id, username, email, password_hash, role, enabled, created_at, updated_at, last_login, created_by, description FROM users WHERE id = ?`
	var user User
	var lastLogin sql.NullTime
	err := db.conn.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt,
		&lastLogin, &user.CreatedBy, &user.Description)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (db *DB) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, email, password_hash, role, enabled, created_at, updated_at, last_login, created_by, description FROM users WHERE username = ?`
	var user User
	var lastLogin sql.NullTime
	err := db.conn.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt,
		&lastLogin, &user.CreatedBy, &user.Description)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	return &user, nil
}

// GetAllUsers retrieves all users
func (db *DB) GetAllUsers() ([]*User, error) {
	query := `SELECT id, username, email, password_hash, role, enabled, created_at, updated_at, last_login, created_by, description FROM users ORDER BY username`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		var lastLogin sql.NullTime
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt,
			&lastLogin, &user.CreatedBy, &user.Description)
		if err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}
		users = append(users, &user)
	}
	return users, nil
}

// UpdateUser updates a user
func (db *DB) UpdateUser(user *User) error {
	query := `
		UPDATE users SET username = ?, email = ?, role = ?, enabled = ?, description = ?
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, user.Username, user.Email, user.Role, user.Enabled, user.Description, user.ID)
	return err
}

// UpdateUserPassword updates a user's password
func (db *DB) UpdateUserPassword(id, passwordHash string) error {
	query := `UPDATE users SET password_hash = ? WHERE id = ?`
	_, err := db.conn.Exec(query, passwordHash, id)
	return err
}

// DeleteUser deletes a user
func (db *DB) DeleteUser(id string) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID           int64     `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	UserID       string    `json:"user_id,omitempty"`
	Username     string    `json:"username,omitempty"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type,omitempty"`
	ResourceID   string    `json:"resource_id,omitempty"`
	Details      string    `json:"details,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	Success      bool      `json:"success"`
	ErrorMsg     string    `json:"error_message,omitempty"`
}

// AddAuditLog adds an audit log entry
func (db *DB) AddAuditLog(entry *AuditLogEntry) error {
	query := `
		INSERT INTO audit_log (user_id, username, action, resource_type, resource_id, details, ip_address, user_agent, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.conn.Exec(query,
		entry.UserID, entry.Username, entry.Action, entry.ResourceType,
		entry.ResourceID, entry.Details, entry.IPAddress, entry.UserAgent,
		entry.Success, entry.ErrorMsg)
	return err
}

// GetAuditLogs retrieves audit logs with pagination
func (db *DB) GetAuditLogs(limit, offset int) ([]*AuditLogEntry, error) {
	query := `
		SELECT id, timestamp, user_id, username, action, resource_type, resource_id, details, ip_address, user_agent, success, error_message
		FROM audit_log
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`
	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*AuditLogEntry
	for rows.Next() {
		var entry AuditLogEntry
		err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.UserID, &entry.Username,
			&entry.Action, &entry.ResourceType, &entry.ResourceID, &entry.Details,
			&entry.IPAddress, &entry.UserAgent, &entry.Success, &entry.ErrorMsg)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
	}
	return entries, nil
}

// SaveConfigVersion saves a configuration version
func (db *DB) SaveConfigVersion(version int, configHash, configData, createdBy, description string, isRollback bool, parentVersion int) error {
	query := `
		INSERT INTO config_versions (version, config_hash, config_data, created_by, change_description, is_rollback, parent_version)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.conn.Exec(query, version, configHash, configData, createdBy, description, isRollback, parentVersion)
	return err
}

// GetConfigVersions retrieves all config versions
func (db *DB) GetConfigVersions(limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, version, config_hash, created_at, created_by, change_description, is_rollback, parent_version
		FROM config_versions
		ORDER BY version DESC
		LIMIT ?
	`
	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []map[string]interface{}
	for rows.Next() {
		var id, version, configHash, createdBy, description string
		var createdAt time.Time
		var isRollback bool
		var parentVersion sql.NullInt64

		err := rows.Scan(&id, &version, &configHash, &createdAt, &createdBy, &description, &isRollback, &parentVersion)
		if err != nil {
			return nil, err
		}

		v := map[string]interface{}{
			"id":                 id,
			"version":            version,
			"config_hash":        configHash,
			"created_at":         createdAt,
			"created_by":         createdBy,
			"change_description": description,
			"is_rollback":        isRollback,
		}
		if parentVersion.Valid {
			v["parent_version"] = parentVersion.Int64
		}
		versions = append(versions, v)
	}
	return versions, nil
}

// GetConfigVersionData retrieves the config data for a specific version
func (db *DB) GetConfigVersionData(version int) (string, error) {
	query := `SELECT config_data FROM config_versions WHERE version = ?`
	var configData string
	err := db.conn.QueryRow(query, version).Scan(&configData)
	if err != nil {
		return "", err
	}
	return configData, nil
}

// UpdateUserLastLogin updates the last login time
func (db *DB) UpdateUserLastLogin(userID string) error {
	query := `UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.Exec(query, userID)
	return err
}
