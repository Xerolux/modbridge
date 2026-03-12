package audit

import (
	"encoding/json"
	"log"
	"modbridge/pkg/database"
	"sync"
	"time"
)

// Auditor handles audit logging
type Auditor struct {
	db  *database.DB
	mu  sync.Mutex
	buf chan *database.AuditLogEntry
}

// NewAuditor creates a new auditor
func NewAuditor(db *database.DB) *Auditor {
	a := &Auditor{
		db:  db,
		buf: make(chan *database.AuditLogEntry, 1000),
	}
	go a.processBuffer()
	return a
}

// LogAction logs an action
func (a *Auditor) LogAction(action, resourceType, resourceID, userID, username, details, ipAddress, userAgent string, success bool, errorMsg string) {
	entry := &database.AuditLogEntry{
		Timestamp:    time.Now(),
		UserID:       userID,
		Username:     username,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Success:      success,
		ErrorMsg:     errorMsg,
	}

	select {
	case a.buf <- entry:
	default:
		log.Printf("WARNING: Audit log buffer full, dropping entry")
	}
}

// LogLogin logs a login attempt
func (a *Auditor) LogLogin(username, ipAddress, userAgent string, success bool) {
	a.LogAction("user.login", "user", username, "", username, "", ipAddress, userAgent, success, "")
}

// LogLogout logs a logout
func (a *Auditor) LogLogout(userID, username, ipAddress, userAgent string) {
	a.LogAction("user.logout", "user", userID, userID, username, "", ipAddress, userAgent, true, "")
}

// LogProxyAction logs a proxy action
func (a *Auditor) LogProxyAction(action, proxyID, userID, username, details, ipAddress, userAgent string, success bool) {
	a.LogAction(action, "proxy", proxyID, userID, username, details, ipAddress, userAgent, success, "")
}

// LogConfigChange logs a configuration change
func (a *Auditor) LogConfigChange(action, userID, username, details, ipAddress, userAgent string, success bool) {
	a.LogAction(action, "config", "", userID, username, details, ipAddress, userAgent, success, "")
}

// LogUserAction logs a user management action
func (a *Auditor) LogUserAction(action, targetUserID, userID, username, details, ipAddress, userAgent string, success bool) {
	a.LogAction(action, "user", targetUserID, userID, username, details, ipAddress, userAgent, success, "")
}

// processBuffer processes buffered audit log entries
func (a *Auditor) processBuffer() {
	for entry := range a.buf {
		if err := a.db.AddAuditLog(entry); err != nil {
			log.Printf("ERROR: Failed to write audit log: %v", err)
		}
	}
}

// GetLogs retrieves audit logs
func (a *Auditor) GetLogs(limit, offset int) ([]*database.AuditLogEntry, error) {
	return a.db.GetAuditLogs(limit, offset)
}

// ExportLogsJSON exports audit logs as JSON
func (a *Auditor) ExportLogsJSON(limit int) (string, error) {
	logs, err := a.GetLogs(limit, 0)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Close closes the auditor
func (a *Auditor) Close() {
	close(a.buf)
}
