package audit

import (
	"encoding/json"
	"fmt"
	"log"
	"modbridge/pkg/database"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	// Authentication events
	EventAuthLogin    EventType = "auth.login"
	EventAuthLogout   EventType = "auth.logout"
	EventAuthFailed   EventType = "auth.failed"
	EventAuthTokenGen EventType = "auth.token_generated"
	EventAuthTokenRev EventType = "auth.token_revoked"

	// Authorization events
	EventAccessGranted EventType = "access.granted"
	EventAccessDenied  EventType = "access.denied"

	// User management events
	EventUserCreated    EventType = "user.created"
	EventUserUpdated    EventType = "user.updated"
	EventUserDeleted    EventType = "user.deleted"
	EventUserRoleChange EventType = "user.role_changed"

	// Proxy management events
	EventProxyCreated EventType = "proxy.created"
	EventProxyUpdated EventType = "proxy.updated"
	EventProxyDeleted EventType = "proxy.deleted"
	EventProxyStarted EventType = "proxy.started"
	EventProxyStopped EventType = "proxy.stopped"

	// Configuration events
	EventConfigUpdated  EventType = "config.updated"
	EventConfigImported EventType = "config.imported"
	EventConfigExported EventType = "config.exported"

	// Data access events
	EventDataRead  EventType = "data.read"
	EventDataWrite EventType = "data.written"

	// System events
	EventSystemStarted EventType = "system.started"
	EventSystemStopped EventType = "system.stopped"
	EventSystemRestart EventType = "system.restarted"
	EventSystemError   EventType = "system.error"
)

// Event represents a comprehensive audit log entry
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id,omitempty"`
	Username  string                 `json:"username,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Resource  string                 `json:"resource,omitempty"`
	Action    string                 `json:"action"`
	Outcome   string                 `json:"outcome"` // "success" or "failure"
	Details   map[string]interface{} `json:"details,omitempty"`
	Reason    string                 `json:"reason,omitempty"`
}

// FileAuditLogger manages file-based audit logging
type FileAuditLogger struct {
	mu          sync.RWMutex
	file        *os.File
	filePath    string
	maxFileSize int64
	currentSize int64
	enabled     bool
	eventChan   chan *Event
	wg          sync.WaitGroup
	stopChan    chan struct{}
}

// FileLoggerConfig holds file audit logger configuration
type FileLoggerConfig struct {
	FilePath    string
	MaxFileSize int64 // in bytes, 0 = unlimited
	BufferSize  int   // number of events to buffer
}

// DefaultFileLoggerConfig returns default file audit logger configuration
func DefaultFileLoggerConfig() FileLoggerConfig {
	return FileLoggerConfig{
		FilePath:    "logs/audit.log",
		MaxFileSize: 100 * 1024 * 1024, // 100 MB
		BufferSize:  1000,
	}
}

// NewFileAuditLogger creates a new file-based audit logger
func NewFileAuditLogger(cfg FileLoggerConfig) (*FileAuditLogger, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}

	fal := &FileAuditLogger{
		file:        file,
		filePath:    cfg.FilePath,
		maxFileSize: cfg.MaxFileSize,
		currentSize: info.Size(),
		enabled:     true,
		eventChan:   make(chan *Event, cfg.BufferSize),
		stopChan:    make(chan struct{}),
	}

	// Start background writer
	fal.wg.Add(1)
	go fal.backgroundWriter()

	return fal, nil
}

// Log logs an audit event
func (fal *FileAuditLogger) Log(event *Event) error {
	if !fal.enabled {
		return nil
	}

	// Set timestamp and ID if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.ID == "" {
		event.ID = generateEventID()
	}

	// Send to background writer
	select {
	case fal.eventChan <- event:
		return nil
	default:
		// Buffer full, write synchronously
		return fal.writeEvent(event)
	}
}

// Logf logs an audit event with formatted details
func (fal *FileAuditLogger) Logf(eventType EventType, action, outcome string, details map[string]interface{}) error {
	event := &Event{
		Type:    eventType,
		Action:  action,
		Outcome: outcome,
		Details: details,
	}
	return fal.Log(event)
}

// LogAuth logs authentication events
func (fal *FileAuditLogger) LogAuth(eventType EventType, userID, username, ipAddress string, success bool, reason string) error {
	outcome := "success"
	if !success {
		outcome = "failure"
	}

	event := &Event{
		Type:      eventType,
		UserID:    userID,
		Username:  username,
		IPAddress: ipAddress,
		Action:    string(eventType),
		Outcome:   outcome,
		Reason:    reason,
	}
	return fal.Log(event)
}

// LogAccess logs access control events
func (fal *FileAuditLogger) LogAccess(userID, username, ipAddress, resource, action string, granted bool, reason string) error {
	outcome := "success"
	if !granted {
		outcome = "failure"
	}

	event := &Event{
		Type:      EventAccessGranted,
		UserID:    userID,
		Username:  username,
		IPAddress: ipAddress,
		Resource:  resource,
		Action:    action,
		Outcome:   outcome,
		Reason:    reason,
	}

	if !granted {
		event.Type = EventAccessDenied
	}

	return fal.Log(event)
}

// LogUserAction logs user management actions
func (fal *FileAuditLogger) LogUserAction(eventType EventType, targetUserID, targetUsername string, actorID, actorName string, details map[string]interface{}) error {
	event := &Event{
		Type:     eventType,
		UserID:   actorID,
		Username: actorName,
		Action:   string(eventType),
		Outcome:  "success",
		Details:  details,
	}

	if details == nil {
		event.Details = make(map[string]interface{})
	}
	event.Details["target_user_id"] = targetUserID
	event.Details["target_username"] = targetUsername

	return fal.Log(event)
}

// LogConfigChange logs configuration changes
func (fal *FileAuditLogger) LogConfigChange(userID, username string, changes map[string]interface{}) error {
	event := &Event{
		Type:     EventConfigUpdated,
		UserID:   userID,
		Username: username,
		Action:   "configuration_changed",
		Outcome:  "success",
		Details:  changes,
	}
	return fal.Log(event)
}

// LogSystemEvent logs system-level events
func (fal *FileAuditLogger) LogSystemEvent(eventType EventType, details map[string]interface{}) error {
	event := &Event{
		Type:    eventType,
		Action:  string(eventType),
		Outcome: "success",
		Details: details,
	}
	return fal.Log(event)
}

// backgroundWriter writes events from the channel to the file
func (fal *FileAuditLogger) backgroundWriter() {
	defer fal.wg.Done()

	for {
		select {
		case event := <-fal.eventChan:
			if err := fal.writeEvent(event); err != nil {
				// Log to stderr as fallback
				fmt.Fprintf(os.Stderr, "Failed to write audit log: %v\n", err)
			}
		case <-fal.stopChan:
			// Drain remaining events
			for len(fal.eventChan) > 0 {
				event := <-fal.eventChan
				if err := fal.writeEvent(event); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to write audit log: %v\n", err)
				}
			}
			return
		}
	}
}

// writeEvent writes a single event to the log file
func (fal *FileAuditLogger) writeEvent(event *Event) error {
	fal.mu.Lock()
	defer fal.mu.Unlock()

	// Check file rotation
	if fal.maxFileSize > 0 && fal.currentSize > fal.maxFileSize {
		if err := fal.rotateLogFile(); err != nil {
			return fmt.Errorf("failed to rotate log file: %w", err)
		}
	}

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write to file
	data = append(data, '\n')
	n, err := fal.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	fal.currentSize += int64(n)

	// Sync to disk for durability
	if err := fal.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync log file: %w", err)
	}

	return nil
}

// rotateLogFile rotates the log file
func (fal *FileAuditLogger) rotateLogFile() error {
	// Close current file
	if err := fal.file.Close(); err != nil {
		return err
	}

	// Rename current file
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", fal.filePath, timestamp)
	if err := os.Rename(fal.filePath, rotatedPath); err != nil {
		return err
	}

	// Open new file
	file, err := os.OpenFile(fal.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	fal.file = file
	fal.currentSize = 0

	return nil
}

// Enable enables audit logging
func (fal *FileAuditLogger) Enable() {
	fal.mu.Lock()
	defer fal.mu.Unlock()
	fal.enabled = true
}

// Disable disables audit logging
func (fal *FileAuditLogger) Disable() {
	fal.mu.Lock()
	defer fal.mu.Unlock()
	fal.enabled = false
}

// Close closes the file audit logger
func (fal *FileAuditLogger) Close() error {
	// Signal stop to background writer
	close(fal.stopChan)

	// Wait for writer to finish
	fal.wg.Wait()

	// Close file
	fal.mu.Lock()
	defer fal.mu.Unlock()

	if fal.file != nil {
		return fal.file.Close()
	}

	return nil
}

// Export exports audit logs to a file in JSON format
func (fal *FileAuditLogger) Export(outputPath string, filter EventFilter) error {
	fal.mu.RLock()
	defer fal.mu.RUnlock()

	// Read log file
	events, err := fal.readEventsFromFile(fal.filePath)
	if err != nil {
		return err
	}

	// Apply filter
	if filter.StartTime != nil || filter.EndTime != nil || filter.UserID != "" || filter.Type != "" || filter.Outcome != "" {
		events = filter.Apply(events)
	}

	// Write to output file
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}

// readEventsFromFile reads all events from a log file
func (fal *FileAuditLogger) readEventsFromFile(filePath string) ([]*Event, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var events []*Event
	lines := splitLines(data)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			continue // Skip malformed lines
		}

		events = append(events, &event)
	}

	return events, nil
}

// splitLines splits data into lines
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0

	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}

	if start < len(data) {
		lines = append(lines, data[start:])
	}

	return lines
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// EventFilter is used to filter audit events
type EventFilter struct {
	StartTime *time.Time
	EndTime   *time.Time
	UserID    string
	Type      EventType
	Outcome   string
}

// Apply applies the filter to events
func (f *EventFilter) Apply(events []*Event) []*Event {
	var filtered []*Event

	for _, event := range events {
		if f.StartTime != nil && event.Timestamp.Before(*f.StartTime) {
			continue
		}
		if f.EndTime != nil && event.Timestamp.After(*f.EndTime) {
			continue
		}
		if f.UserID != "" && event.UserID != f.UserID {
			continue
		}
		if f.Type != "" && event.Type != f.Type {
			continue
		}
		if f.Outcome != "" && event.Outcome != f.Outcome {
			continue
		}

		filtered = append(filtered, event)
	}

	return filtered
}

// Auditor handles audit logging (legacy database-backed implementation)
type Auditor struct {
	db             *database.DB
	mu             sync.Mutex
	buf            chan *database.AuditLogEntry
	fileLogger     *FileAuditLogger
	enableFileLog  bool
}

// NewAuditor creates a new auditor with database backing
func NewAuditor(db *database.DB) *Auditor {
	a := &Auditor{
		db:  db,
		buf: make(chan *database.AuditLogEntry, 1000),
	}
	go a.processBuffer()
	return a
}

// NewAuditorWithFile creates a new auditor with both database and file logging
func NewAuditorWithFile(db *database.DB, fileLoggerCfg FileLoggerConfig) (*Auditor, error) {
	fileLogger, err := NewFileAuditLogger(fileLoggerCfg)
	if err != nil {
		return nil, err
	}

	a := &Auditor{
		db:            db,
		buf:           make(chan *database.AuditLogEntry, 1000),
		fileLogger:    fileLogger,
		enableFileLog: true,
	}
	go a.processBuffer()
	return a, nil
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

	// Also log to file if enabled
	if a.fileLogger != nil && a.enableFileLog {
		eventType := mapActionToEventType(action)
		outcome := "success"
		if !success {
			outcome = "failure"
		}

		fileEvent := &Event{
			Type:      eventType,
			Timestamp: entry.Timestamp,
			UserID:    userID,
			Username:  username,
			IPAddress: ipAddress,
			UserAgent: userAgent,
			Resource:  resourceType + ":" + resourceID,
			Action:    action,
			Outcome:   outcome,
			Reason:    errorMsg,
		}

		if details != "" {
			fileEvent.Details = map[string]interface{}{
				"details": details,
			}
		}

		a.fileLogger.Log(fileEvent)
	}

	select {
	case a.buf <- entry:
	default:
		log.Printf("WARNING: Audit log buffer full, dropping entry")
	}
}

// mapActionToEventType maps action strings to EventTypes
func mapActionToEventType(action string) EventType {
	switch action {
	case "user.login":
		return EventAuthLogin
	case "user.logout":
		return EventAuthLogout
	case "proxy.created":
		return EventProxyCreated
	case "proxy.started":
		return EventProxyStarted
	case "proxy.stopped":
		return EventProxyStopped
	case "config.updated":
		return EventConfigUpdated
	default:
		return EventType(action)
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

// GetFileLogger returns the file audit logger if enabled
func (a *Auditor) GetFileLogger() *FileAuditLogger {
	return a.fileLogger
}

// Close closes the auditor
func (a *Auditor) Close() {
	close(a.buf)
	if a.fileLogger != nil {
		a.fileLogger.Close()
	}
}
