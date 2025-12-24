package audit

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Action represents an audit action type.
type Action string

const (
	ActionLogin          Action = "login"
	ActionLogout         Action = "logout"
	ActionCreateProxy    Action = "create_proxy"
	ActionUpdateProxy    Action = "update_proxy"
	ActionDeleteProxy    Action = "delete_proxy"
	ActionStartProxy     Action = "start_proxy"
	ActionStopProxy      Action = "stop_proxy"
	ActionCreateUser     Action = "create_user"
	ActionUpdateUser     Action = "update_user"
	ActionDeleteUser     Action = "delete_user"
	ActionChangePassword Action = "change_password"
	ActionBackupConfig   Action = "backup_config"
	ActionRestoreConfig  Action = "restore_config"
	ActionRenameDevice   Action = "rename_device"
)

// Entry represents an audit log entry.
type Entry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id"`
	Username  string                 `json:"username"`
	Action    Action                 `json:"action"`
	Resource  string                 `json:"resource,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
}

// Logger handles audit logging.
type Logger struct {
	file    *os.File
	entries []Entry
	mu      sync.RWMutex
	maxSize int
}

// NewLogger creates a new audit logger.
func NewLogger(filePath string, maxSize int) (*Logger, error) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	logger := &Logger{
		file:    f,
		entries: make([]Entry, 0, maxSize),
		maxSize: maxSize,
	}

	// Load existing entries from file
	_ = logger.loadEntries(filePath)

	return logger, nil
}

// Log creates a new audit log entry.
func (l *Logger) Log(userID, username string, action Action, resource string, details map[string]interface{}, ipAddress string, success bool, err error) {
	entry := Entry{
		ID:        generateEntryID(),
		Timestamp: time.Now(),
		UserID:    userID,
		Username:  username,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
		Success:   success,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Add to memory (ring buffer)
	if len(l.entries) >= l.maxSize {
		l.entries = l.entries[1:] // Remove oldest
	}
	l.entries = append(l.entries, entry)

	// Write to file
	data, _ := json.Marshal(entry)
	_, _ = l.file.Write(append(data, '\n'))
	_ = l.file.Sync()
}

// GetEntries returns recent audit log entries.
func (l *Logger) GetEntries(limit int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit <= 0 || limit > len(l.entries) {
		limit = len(l.entries)
	}

	// Return most recent entries
	start := len(l.entries) - limit
	entries := make([]Entry, limit)
	copy(entries, l.entries[start:])

	return entries
}

// Search searches audit log entries by criteria.
func (l *Logger) Search(userID string, action Action, startTime, endTime time.Time, limit int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	results := make([]Entry, 0)

	for i := len(l.entries) - 1; i >= 0; i-- {
		entry := l.entries[i]

		// Filter by user ID
		if userID != "" && entry.UserID != userID {
			continue
		}

		// Filter by action
		if action != "" && entry.Action != action {
			continue
		}

		// Filter by time range
		if !startTime.IsZero() && entry.Timestamp.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && entry.Timestamp.After(endTime) {
			continue
		}

		results = append(results, entry)

		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results
}

// Close closes the audit logger.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// loadEntries loads existing entries from file (most recent ones).
func (l *Logger) loadEntries(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := splitLines(data)
	entries := make([]Entry, 0, len(lines))

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	// Keep only the most recent entries
	if len(entries) > l.maxSize {
		entries = entries[len(entries)-l.maxSize:]
	}

	l.entries = entries
	return nil
}

// splitLines splits data into lines.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0

	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}

	if start < len(data) {
		lines = append(lines, data[start:])
	}

	return lines
}

// generateEntryID generates a unique ID for an audit entry.
func generateEntryID() string {
	return time.Now().Format("20060102150405.000000")
}
