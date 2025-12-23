package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// StructuredLogger provides structured logging with zerolog.
type StructuredLogger struct {
	zlog   zerolog.Logger
	level  zerolog.Level
	legacy *Logger // Keep legacy logger for compatibility
}

// NewStructuredLogger creates a new structured logger.
func NewStructuredLogger(filePath string, level string, prettyPrint bool) (*StructuredLogger, error) {
	// Open log file
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Create multi-writer (file + stdout)
	var output io.Writer
	if prettyPrint {
		// Pretty console output for development
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		output = zerolog.MultiLevelWriter(consoleWriter, file)
	} else {
		// JSON output for production
		output = zerolog.MultiLevelWriter(os.Stdout, file)
	}

	// Parse log level
	logLevel := parseLevel(level)

	// Create zerolog logger
	zlog := zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Str("service", "modbus-proxy").
		Logger()

	// Create legacy logger for compatibility
	legacyLogger, err := NewLogger(filePath+".legacy", 1000)
	if err != nil {
		return nil, err
	}

	return &StructuredLogger{
		zlog:   zlog,
		level:  logLevel,
		legacy: legacyLogger,
	}, nil
}

// parseLevel parses string log level to zerolog.Level.
func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug", "DEBUG":
		return zerolog.DebugLevel
	case "info", "INFO":
		return zerolog.InfoLevel
	case "warn", "WARN", "warning", "WARNING":
		return zerolog.WarnLevel
	case "error", "ERROR":
		return zerolog.ErrorLevel
	case "fatal", "FATAL":
		return zerolog.FatalLevel
	case "panic", "PANIC":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithField returns a logger with an additional field.
func (sl *StructuredLogger) WithField(key string, value interface{}) *zerolog.Logger {
	logger := sl.zlog.With().Interface(key, value).Logger()
	return &logger
}

// WithFields returns a logger with multiple fields.
func (sl *StructuredLogger) WithFields(fields map[string]interface{}) *zerolog.Logger {
	ctx := sl.zlog.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	logger := ctx.Logger()
	return &logger
}

// WithProxyID returns a logger with proxy_id field.
func (sl *StructuredLogger) WithProxyID(proxyID string) *zerolog.Logger {
	logger := sl.zlog.With().Str("proxy_id", proxyID).Logger()
	return &logger
}

// WithRequestID returns a logger with request_id (correlation ID) field.
func (sl *StructuredLogger) WithRequestID(requestID string) *zerolog.Logger {
	logger := sl.zlog.With().Str("request_id", requestID).Logger()
	return &logger
}

// Debug logs a debug message.
func (sl *StructuredLogger) Debug(msg string, fields ...map[string]interface{}) {
	event := sl.zlog.Debug()
	if len(fields) > 0 {
		event = addFields(event, fields[0])
	}
	event.Msg(msg)
}

// Info logs an info message.
func (sl *StructuredLogger) Info(msg string, fields ...map[string]interface{}) {
	event := sl.zlog.Info()
	if len(fields) > 0 {
		event = addFields(event, fields[0])
	}
	event.Msg(msg)

	// Also log to legacy logger for compatibility
	if sl.legacy != nil {
		proxyID := ""
		if len(fields) > 0 {
			if id, ok := fields[0]["proxy_id"].(string); ok {
				proxyID = id
			}
		}
		sl.legacy.Info(proxyID, msg)
	}
}

// Warn logs a warning message.
func (sl *StructuredLogger) Warn(msg string, fields ...map[string]interface{}) {
	event := sl.zlog.Warn()
	if len(fields) > 0 {
		event = addFields(event, fields[0])
	}
	event.Msg(msg)
}

// Error logs an error message.
func (sl *StructuredLogger) Error(msg string, err error, fields ...map[string]interface{}) {
	event := sl.zlog.Error()
	if err != nil {
		event = event.Err(err)
	}
	if len(fields) > 0 {
		event = addFields(event, fields[0])
	}
	event.Msg(msg)

	// Also log to legacy logger for compatibility
	if sl.legacy != nil {
		proxyID := ""
		if len(fields) > 0 {
			if id, ok := fields[0]["proxy_id"].(string); ok {
				proxyID = id
			}
		}
		errorMsg := msg
		if err != nil {
			errorMsg = msg + ": " + err.Error()
		}
		sl.legacy.Error(proxyID, errorMsg)
	}
}

// Fatal logs a fatal message and exits.
func (sl *StructuredLogger) Fatal(msg string, err error, fields ...map[string]interface{}) {
	event := sl.zlog.Fatal()
	if err != nil {
		event = event.Err(err)
	}
	if len(fields) > 0 {
		event = addFields(event, fields[0])
	}
	event.Msg(msg)
}

// addFields adds map fields to zerolog event.
func addFields(event *zerolog.Event, fields map[string]interface{}) *zerolog.Event {
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	return event
}

// SetLevel changes the log level dynamically.
func (sl *StructuredLogger) SetLevel(level string) {
	sl.level = parseLevel(level)
	sl.zlog = sl.zlog.Level(sl.level)
}

// GetLevel returns the current log level.
func (sl *StructuredLogger) GetLevel() string {
	return sl.level.String()
}

// GetLegacyLogger returns the legacy logger for compatibility.
func (sl *StructuredLogger) GetLegacyLogger() *Logger {
	return sl.legacy
}

// Close closes the structured logger.
func (sl *StructuredLogger) Close() error {
	if sl.legacy != nil {
		return sl.legacy.Close()
	}
	return nil
}
