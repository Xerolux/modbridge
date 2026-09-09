// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// LogLevel defines log levels.
type LogLevel string

const (
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
	DEBUG LogLevel = "DEBUG"
)

// LogEntry represents a structured log line.
type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	Level     LogLevel `json:"level"`
	ProxyID   string   `json:"proxy_id,omitempty"`
	Message   string   `json:"message"`
}

// RotationConfig controls how log files are rotated.
//
// A zero MaxSizeMB disables rotation entirely, which keeps the previous
// behaviour for callers that do not configure it.
type RotationConfig struct {
	// MaxSizeMB is the size at which the active file is rotated.
	MaxSizeMB int
	// MaxFiles is how many rotated files to keep per log, not counting the
	// active one.
	MaxFiles int
	// MaxAgeDays deletes rotated files older than this. Zero keeps them until
	// MaxFiles pushes them out.
	MaxAgeDays int
}

// logFile is an open log file plus the size bookkeeping rotation needs, so the
// hot path does not have to stat the file on every line.
type logFile struct {
	f    *os.File
	path string
	size int64
}

// Logger manages logging.
type Logger struct {
	mu          sync.Mutex
	logDir      string
	files       map[string]*logFile
	rotation    RotationConfig
	ringBuffer  []LogEntry
	ringSize    int
	ringStart   int
	ringCount   int
	subscribers map[chan LogEntry]struct{}
	minLevel    atomic.Int32 // Minimum log level to output (levelPriority value)
}

// NewLogger creates a new logger.
func NewLogger(logDir string, bufferSize int) (*Logger, error) {
	if logDir != "" {
		if info, err := os.Stat(logDir); err == nil {
			if !info.IsDir() {
				// An older install may have written a plain file here. Keep it:
				// moving it aside preserves the log, deleting it would not.
				moved := logDir + ".old"
				if err := os.Rename(logDir, moved); err != nil {
					return nil, fmt.Errorf("failed to move existing file at log path %s aside: %w", logDir, err)
				}
			}
		}
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	l := &Logger{
		logDir:      logDir,
		files:       make(map[string]*logFile),
		ringBuffer:  make([]LogEntry, 0, bufferSize),
		ringSize:    bufferSize,
		subscribers: make(map[chan LogEntry]struct{}),
	}
	l.minLevel.Store(levelPriority(INFO))
	return l, nil
}

// NewNullLogger creates a logger that discards file output.
func NewNullLogger(bufferSize int) *Logger {
	l := &Logger{
		logDir:      "",
		files:       make(map[string]*logFile),
		ringBuffer:  make([]LogEntry, 0, bufferSize),
		ringSize:    bufferSize,
		subscribers: make(map[chan LogEntry]struct{}),
	}
	l.minLevel.Store(levelPriority(INFO))
	return l
}

// SetRotation updates the rotation policy. It takes effect on the next write.
func (l *Logger) SetRotation(cfg RotationConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotation = cfg
}

// fileIDFor maps a proxy ID onto the log file that receives its entries.
func fileIDFor(proxyID string) string {
	if proxyID == "" || proxyID == "SYSTEM" || proxyID == "API" {
		return "system"
	}
	return proxyID
}

func (l *Logger) getLogFile(proxyID string) (*logFile, error) {
	if l.logDir == "" {
		return nil, nil
	}

	fileID := fileIDFor(proxyID)

	if lf, exists := l.files[fileID]; exists {
		return lf, nil
	}

	filePath := filepath.Join(l.logDir, fmt.Sprintf("proxy_%s.log", fileID))
	if fileID == "system" {
		filePath = filepath.Join(l.logDir, "system.log")
	}
	filePath = filepath.Clean(filePath)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	// Pick up the size of a file that already exists, so a restart does not
	// reset the rotation threshold.
	var size int64
	if info, statErr := f.Stat(); statErr == nil {
		size = info.Size()
	}

	lf := &logFile{f: f, path: filePath, size: size}
	l.files[fileID] = lf
	return lf, nil
}

// writeLine appends one line and rotates first when the line would push the
// file past the configured size. Callers must hold l.mu.
func (l *Logger) writeLine(fileID string, lf *logFile, line []byte) {
	maxBytes := int64(l.rotation.MaxSizeMB) * 1024 * 1024
	if maxBytes > 0 && lf.size > 0 && lf.size+int64(len(line)) > maxBytes {
		if err := l.rotate(fileID, lf); err != nil {
			// Rotation failed; keep logging to the current file rather than
			// dropping the line.
			fmt.Fprintf(os.Stderr, "log rotation failed for %s: %v\n", lf.path, err)
		}
	}

	n, err := lf.f.Write(line)
	if err == nil {
		lf.size += int64(n)
	}
}

// rotate closes the active file, shifts the numbered backups up by one and
// reopens a fresh file. Callers must hold l.mu.
func (l *Logger) rotate(fileID string, lf *logFile) error {
	if err := lf.f.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	keep := l.rotation.MaxFiles
	if keep < 1 {
		// Nothing to keep: the active file is simply truncated by reopening.
		if err := os.Remove(lf.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove log file: %w", err)
		}
	} else {
		// The oldest backup falls off the end, the rest shift up.
		oldest := fmt.Sprintf("%s.%d", lf.path, keep)
		if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove oldest log file: %w", err)
		}
		for i := keep - 1; i >= 1; i-- {
			from := fmt.Sprintf("%s.%d", lf.path, i)
			to := fmt.Sprintf("%s.%d", lf.path, i+1)
			if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to rotate %s: %w", from, err)
			}
		}
		if err := os.Rename(lf.path, lf.path+".1"); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to rotate active log file: %w", err)
		}
	}

	f, err := os.OpenFile(lf.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		// The map entry would otherwise hand out a closed file.
		delete(l.files, fileID)
		return fmt.Errorf("failed to reopen log file: %w", err)
	}
	lf.f = f
	lf.size = 0

	l.pruneAged(lf.path)
	return nil
}

// pruneAged deletes rotated files older than MaxAgeDays. Callers must hold
// l.mu. The active file is never pruned.
func (l *Logger) pruneAged(basePath string) {
	if l.rotation.MaxAgeDays <= 0 {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -l.rotation.MaxAgeDays)
	for i := 1; i <= l.rotation.MaxFiles; i++ {
		path := fmt.Sprintf("%s.%d", basePath, i)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(path)
		}
	}
}

// shouldLog returns true if the given level should be logged.
func (l *Logger) shouldLog(level LogLevel) bool {
	return levelPriority(level) >= l.minLevel.Load()
}

func levelPriority(level LogLevel) int32 {
	switch level {
	case DEBUG:
		return 0
	case INFO:
		return 1
	case WARN:
		return 2
	case ERROR:
		return 3
	default:
		// Unknown levels should be treated conservatively and logged.
		return 3
	}
}

func priorityToLevel(priority int32) LogLevel {
	switch priority {
	case 0:
		return DEBUG
	case 1:
		return INFO
	case 2:
		return WARN
	case 3:
		return ERROR
	default:
		return ERROR
	}
}

// Log writes a log entry.
func (l *Logger) Log(level LogLevel, proxyID, msg string) {
	// Check if we should log this level
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		ProxyID:   proxyID,
		Message:   msg,
	}

	// Single critical section:
	// 1) write file, 2) append ring, 3) snapshot subscribers.
	l.mu.Lock()
	if lf, _ := l.getLogFile(proxyID); lf != nil {
		if jsonBytes, err := json.Marshal(entry); err == nil {
			l.writeLine(fileIDFor(proxyID), lf, append(jsonBytes, '\n'))
		}
	}
	l.appendRingEntry(entry)
	subscribers := make([]chan LogEntry, 0, len(l.subscribers))
	for ch := range l.subscribers {
		subscribers = append(subscribers, ch)
	}
	l.mu.Unlock()

	// Broadcast to subscribers (outside of lock)
	for _, ch := range subscribers {
		select {
		case ch <- entry:
		default:
			// Drop if channel full to avoid blocking logger
		}
	}

	// Print to stdout for debug
	fmt.Printf("[%s] [%s] %s: %s\n", entry.Timestamp, entry.Level, entry.ProxyID, entry.Message)
}

// Subscribe returns a channel for live logs.
func (l *Logger) Subscribe() chan LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan LogEntry, 100)
	l.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber.
func (l *Logger) Unsubscribe(ch chan LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.subscribers, ch)
	close(ch)
}

// GetRecent returns recent logs.
func (l *Logger) GetRecent(limit int) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limit <= 0 || l.ringCount == 0 {
		return nil
	}

	if limit > l.ringCount {
		limit = l.ringCount
	}

	out := make([]LogEntry, limit)
	start := l.ringCount - limit
	for i := 0; i < limit; i++ {
		idx := (l.ringStart + start + i) % l.ringSize
		out[i] = l.ringBuffer[idx]
	}
	return out
}

func (l *Logger) appendRingEntry(entry LogEntry) {
	if l.ringSize <= 0 {
		return
	}

	if len(l.ringBuffer) < l.ringSize {
		l.ringBuffer = append(l.ringBuffer, entry)
		l.ringCount++
		return
	}

	l.ringBuffer[l.ringStart] = entry
	l.ringStart = (l.ringStart + 1) % l.ringSize
	if l.ringCount < l.ringSize {
		l.ringCount++
	}
}

func (l *Logger) Info(proxyID, msg string) {
	l.Log(INFO, proxyID, msg)
}

func (l *Logger) Error(proxyID, msg string) {
	l.Log(ERROR, proxyID, msg)
}

func (l *Logger) Debug(proxyID, msg string) {
	l.Log(DEBUG, proxyID, msg)
}

func (l *Logger) Warn(proxyID, msg string) {
	l.Log(WARN, proxyID, msg)
}

// SetLogLevel changes the minimum log level.
func (l *Logger) SetLogLevel(level LogLevel) {
	l.minLevel.Store(levelPriority(level))
}

// GetLogLevel returns the current minimum log level.
func (l *Logger) GetLogLevel() LogLevel {
	return priorityToLevel(l.minLevel.Load())
}

// IsDebugEnabled reports whether DEBUG-level messages will actually be
// emitted. Intended for hot paths where the message argument is expensive to
// build (e.g. fmt.Sprintf of a binary frame) — wrap the call in
// `if l.IsDebugEnabled() { l.Debug(...) }` to skip the formatting cost
// entirely when DEBUG is off (the default in production).
func (l *Logger) IsDebugEnabled() bool {
	return l.minLevel.Load() <= levelPriority(DEBUG)
}

// IsInfoEnabled reports whether INFO-level messages will be emitted.
func (l *Logger) IsInfoEnabled() bool {
	return l.minLevel.Load() <= levelPriority(INFO)
}

// Close closes all logger files.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var lastErr error
	for id, lf := range l.files {
		if err := lf.f.Close(); err != nil {
			lastErr = err
		}
		delete(l.files, id)
	}
	return lastErr
}
