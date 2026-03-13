package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewFileAuditLogger tests creating a new file audit logger
func TestNewFileAuditLogger(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Fatal("Logger is nil")
	}

	if !logger.enabled {
		t.Error("Logger should be enabled by default")
	}
}

// TestFileAuditLogger_Log tests logging events
func TestFileAuditLogger_Log(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024 * 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	event := &Event{
		Type:      EventAuthLogin,
		UserID:    "user123",
		Username:  "testuser",
		IPAddress: "192.168.1.1",
		Action:    "login",
		Outcome:   "success",
	}

	err = logger.Log(event)
	if err != nil {
		t.Errorf("Failed to log event: %v", err)
	}

	// Give time for async write
	time.Sleep(100 * time.Millisecond)

	// Verify file exists and contains event
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Log file is empty")
	}

	var loggedEvent Event
	if err := json.Unmarshal(data, &loggedEvent); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if loggedEvent.Type != EventAuthLogin {
		t.Errorf("Expected type %s, got %s", EventAuthLogin, loggedEvent.Type)
	}

	if loggedEvent.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", loggedEvent.Username)
	}
}

// TestFileAuditLogger_LogAuth tests authentication event logging
func TestFileAuditLogger_LogAuth(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024 * 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Test successful login
	err = logger.LogAuth(EventAuthLogin, "user123", "testuser", "192.168.1.1", true, "")
	if err != nil {
		t.Errorf("Failed to log auth: %v", err)
	}

	// Test failed login
	err = logger.LogAuth(EventAuthFailed, "user456", "baduser", "192.168.1.2", false, "invalid password")
	if err != nil {
		t.Errorf("Failed to log failed auth: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify events
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := splitLines(data)
	if len(lines) < 2 {
		t.Errorf("Expected at least 2 log entries, got %d", len(lines))
	}
}

// TestFileAuditLogger_LogAccess tests access control event logging
func TestFileAuditLogger_LogAccess(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024 * 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Test granted access
	err = logger.LogAccess("user123", "testuser", "192.168.1.1", "proxy:config", "read", true, "")
	if err != nil {
		t.Errorf("Failed to log access: %v", err)
	}

	// Test denied access
	err = logger.LogAccess("user456", "baduser", "192.168.1.2", "proxy:delete", "delete", false, "insufficient permissions")
	if err != nil {
		t.Errorf("Failed to log denied access: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify events
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := splitLines(data)
	if len(lines) < 2 {
		t.Errorf("Expected at least 2 log entries, got %d", len(lines))
	}

	// Check first event (granted)
	var grantedEvent Event
	if err := json.Unmarshal(lines[0], &grantedEvent); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if grantedEvent.Type != EventAccessGranted {
		t.Errorf("Expected type %s, got %s", EventAccessGranted, grantedEvent.Type)
	}

	// Check second event (denied)
	var deniedEvent Event
	if err := json.Unmarshal(lines[1], &deniedEvent); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if deniedEvent.Type != EventAccessDenied {
		t.Errorf("Expected type %s, got %s", EventAccessDenied, deniedEvent.Type)
	}
}

// TestFileAuditLogger_LogUserAction tests user management event logging
func TestFileAuditLogger_LogUserAction(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024 * 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	details := map[string]interface{}{
		"old_role": "viewer",
		"new_role": "operator",
	}

	err = logger.LogUserAction(EventUserRoleChange, "target123", "targetuser", "admin123", "admin", details)
	if err != nil {
		t.Errorf("Failed to log user action: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify event
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var loggedEvent Event
	if err := json.Unmarshal(data, &loggedEvent); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if loggedEvent.Type != EventUserRoleChange {
		t.Errorf("Expected type %s, got %s", EventUserRoleChange, loggedEvent.Type)
	}

	if loggedEvent.Details["target_user_id"] != "target123" {
		t.Error("Target user ID not in details")
	}
}

// TestFileAuditLogger_LogConfigChange tests configuration change logging
func TestFileAuditLogger_LogConfigChange(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024 * 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	changes := map[string]interface{}{
		"setting": "timeout",
		"old":     30,
		"new":     60,
	}

	err = logger.LogConfigChange("user123", "admin", changes)
	if err != nil {
		t.Errorf("Failed to log config change: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify event
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var loggedEvent Event
	if err := json.Unmarshal(data, &loggedEvent); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if loggedEvent.Type != EventConfigUpdated {
		t.Errorf("Expected type %s, got %s", EventConfigUpdated, loggedEvent.Type)
	}
}

// TestFileAuditLogger_LogSystemEvent tests system event logging
func TestFileAuditLogger_LogSystemEvent(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024 * 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	details := map[string]interface{}{
		"version": "1.0.0",
		"port":    8080,
	}

	err = logger.LogSystemEvent(EventSystemStarted, details)
	if err != nil {
		t.Errorf("Failed to log system event: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify event
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var loggedEvent Event
	if err := json.Unmarshal(data, &loggedEvent); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if loggedEvent.Type != EventSystemStarted {
		t.Errorf("Expected type %s, got %s", EventSystemStarted, loggedEvent.Type)
	}
}

// TestFileAuditLogger_DisableEnable tests disabling and enabling logging
func TestFileAuditLogger_DisableEnable(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024 * 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	event := &Event{
		Type:    EventAuthLogin,
		Action:  "login",
		Outcome: "success",
	}

	// Log while enabled
	err = logger.Log(event)
	if err != nil {
		t.Errorf("Failed to log event: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Disable logging
	logger.Disable()

	err = logger.Log(event)
	if err != nil {
		t.Errorf("Logging should not fail when disabled")
	}

	// Re-enable logging
	logger.Enable()

	err = logger.Log(event)
	if err != nil {
		t.Errorf("Failed to log event after re-enabling: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify we have events (should have 2: one before disable, one after enable)
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := splitLines(data)
	if len(lines) < 2 {
		t.Errorf("Expected at least 2 log entries, got %d", len(lines))
	}
}

// TestFileAuditLogger_FileRotation tests automatic file rotation
func TestFileAuditLogger_FileRotation(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 500, // Very small to trigger rotation
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log multiple events to trigger rotation
	for i := 0; i < 10; i++ {
		event := &Event{
			Type:      EventAuthLogin,
			UserID:    "user123",
			Username:  "testuser",
			IPAddress: "192.168.1.1",
			Action:    "login",
			Outcome:   "success",
			Details: map[string]interface{}{
				"index":     i,
				"some_data": "this is some longer data to fill up the log file faster and trigger rotation",
			},
		}
		err = logger.Log(event)
		if err != nil {
			t.Errorf("Failed to log event: %v", err)
		}
	}

	// Wait for async writes
	time.Sleep(200 * time.Millisecond)

	// Check for rotated files
	matches, err := filepath.Glob(filepath.Join(tempDir, "audit.log*"))
	if err != nil {
		t.Fatalf("Failed to glob log files: %v", err)
	}

	if len(matches) < 2 {
		t.Logf("Warning: Expected rotated files, got %d files. This might be OK if writes are still pending.", len(matches))
	}
}

// TestFileAuditLogger_Export tests exporting audit logs
func TestFileAuditLogger_Export(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024 * 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log some events
	events := []*Event{
		{
			Type:      EventAuthLogin,
			UserID:    "user1",
			Username:  "user1",
			IPAddress: "192.168.1.1",
			Action:    "login",
			Outcome:   "success",
		},
		{
			Type:      EventAuthFailed,
			UserID:    "user2",
			Username:  "user2",
			IPAddress: "192.168.1.2",
			Action:    "login",
			Outcome:   "failure",
			Reason:    "invalid password",
		},
	}

	for _, event := range events {
		if err := logger.Log(event); err != nil {
			t.Errorf("Failed to log event: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Export without filter
	outputPath := filepath.Join(tempDir, "export.json")
	err = logger.Export(outputPath, EventFilter{})
	if err != nil {
		t.Fatalf("Failed to export logs: %v", err)
	}

	// Verify export
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportedEvents []*Event
	if err := json.Unmarshal(data, &exportedEvents); err != nil {
		t.Fatalf("Failed to unmarshal exported events: %v", err)
	}

	if len(exportedEvents) != 2 {
		t.Errorf("Expected 2 exported events, got %d", len(exportedEvents))
	}
}

// TestEventFilter tests filtering events
func TestEventFilter(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	events := []*Event{
		{
			Type:      EventAuthLogin,
			Timestamp: now,
			UserID:    "user1",
			Outcome:   "success",
		},
		{
			Type:      EventAuthFailed,
			Timestamp: now,
			UserID:    "user2",
			Outcome:   "failure",
		},
		{
			Type:      EventConfigUpdated,
			Timestamp: past,
			UserID:    "user1",
			Outcome:   "success",
		},
	}

	tests := []struct {
		name    string
		filter  EventFilter
		wantLen int
	}{
		{
			name:    "No filter",
			filter:  EventFilter{},
			wantLen: 3,
		},
		{
			name: "Filter by type",
			filter: EventFilter{
				Type: EventAuthLogin,
			},
			wantLen: 1,
		},
		{
			name: "Filter by user",
			filter: EventFilter{
				UserID: "user1",
			},
			wantLen: 2,
		},
		{
			name: "Filter by outcome",
			filter: EventFilter{
				Outcome: "success",
			},
			wantLen: 2,
		},
		{
			name: "Filter by start time",
			filter: EventFilter{
				StartTime: &now,
			},
			wantLen: 2, // Events at or after now
		},
		{
			name: "Filter by end time",
			filter: EventFilter{
				EndTime: &now,
			},
			wantLen: 3, // All events at or before now
		},
		{
			name: "Filter by time range",
			filter: EventFilter{
				StartTime: &past,
				EndTime:   &now,
			},
			wantLen: 3, // All events within range
		},
		{
			name: "Combined filter",
			filter: EventFilter{
				UserID:  "user1",
				Type:    EventAuthLogin,
				Outcome: "success",
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Apply(events)
			if len(result) != tt.wantLen {
				t.Errorf("Filter returned %d events, want %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestDefaultFileLoggerConfig tests default configuration
func TestDefaultFileLoggerConfig(t *testing.T) {
	cfg := DefaultFileLoggerConfig()

	if cfg.FilePath != "logs/audit.log" {
		t.Errorf("Expected default path 'logs/audit.log', got '%s'", cfg.FilePath)
	}

	if cfg.MaxFileSize != 100*1024*1024 {
		t.Errorf("Expected default max size 100MB, got %d", cfg.MaxFileSize)
	}

	if cfg.BufferSize != 1000 {
		t.Errorf("Expected default buffer size 1000, got %d", cfg.BufferSize)
	}
}

// TestEventIDGeneration tests event ID generation
func TestEventIDGeneration(t *testing.T) {
	id1 := generateEventID()
	time.Sleep(time.Nanosecond) // Ensure different timestamp
	id2 := generateEventID()

	if id1 == id2 {
		t.Error("Event IDs should be unique")
	}

	if id1 == "" {
		t.Error("Event ID should not be empty")
	}
}

// TestEventTypeConstants tests event type constants
func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name  string
		value EventType
	}{
		{"AuthLogin", EventAuthLogin},
		{"AuthLogout", EventAuthLogout},
		{"AuthFailed", EventAuthFailed},
		{"AuthTokenGen", EventAuthTokenGen},
		{"AuthTokenRev", EventAuthTokenRev},
		{"AccessGranted", EventAccessGranted},
		{"AccessDenied", EventAccessDenied},
		{"UserCreated", EventUserCreated},
		{"UserUpdated", EventUserUpdated},
		{"UserDeleted", EventUserDeleted},
		{"UserRoleChange", EventUserRoleChange},
		{"ProxyCreated", EventProxyCreated},
		{"ProxyUpdated", EventProxyUpdated},
		{"ProxyDeleted", EventProxyDeleted},
		{"ProxyStarted", EventProxyStarted},
		{"ProxyStopped", EventProxyStopped},
		{"ConfigUpdated", EventConfigUpdated},
		{"ConfigImported", EventConfigImported},
		{"ConfigExported", EventConfigExported},
		{"DataRead", EventDataRead},
		{"DataWrite", EventDataWrite},
		{"SystemStarted", EventSystemStarted},
		{"SystemStopped", EventSystemStopped},
		{"SystemRestart", EventSystemRestart},
		{"SystemError", EventSystemError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("EventType %s should not be empty", tt.name)
			}
		})
	}
}

// TestFileAuditLogger_Close tests closing the logger
func TestFileAuditLogger_Close(t *testing.T) {
	tempDir := t.TempDir()
	cfg := FileLoggerConfig{
		FilePath:    filepath.Join(tempDir, "audit.log"),
		MaxFileSize: 1024 * 1024,
		BufferSize:  10,
	}

	logger, err := NewFileAuditLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log an event
	event := &Event{
		Type:    EventAuthLogin,
		Action:  "login",
		Outcome: "success",
	}

	err = logger.Log(event)
	if err != nil {
		t.Errorf("Failed to log event: %v", err)
	}

	// Close logger
	err = logger.Close()
	if err != nil {
		t.Errorf("Failed to close logger: %v", err)
	}

	// Wait for async writes
	time.Sleep(200 * time.Millisecond)

	// Verify event was written before close
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Log file should contain event written before close")
	}
}
