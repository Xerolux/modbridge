package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ValidationError represents a configuration validation error with details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// Error returns the error message
func (ve ValidationError) Error() string {
	if ve.Value != "" {
		return fmt.Sprintf("%s: %s (value: %s)", ve.Field, ve.Message, ve.Value)
	}
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// Error returns a combined error message
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("configuration validation failed with %d error(s):\n", len(ve)))
	for i, err := range ve {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Validator provides comprehensive configuration validation
type Validator struct {
	errors ValidationErrors
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// Validate validates the complete configuration
func (v *Validator) Validate(cfg *Config) error {
	v.errors = make(ValidationErrors, 0)

	// Validate global settings
	v.validateGlobalConfig(cfg)

	// Validate each proxy configuration
	for i, proxy := range cfg.Proxies {
		v.validateProxyConfig(&proxy, i)
	}

	// Validate TLS configuration
	if cfg.TLSEnabled {
		v.validateTLSConfig(cfg)
	}

	// Validate CORS settings
	v.validateCORSConfig(cfg)

	// Validate rate limiting
	v.validateRateLimitConfig(cfg)

	// Validate IP filtering
	v.validateIPFilterConfig(cfg)

	// Validate email configuration
	if cfg.EmailEnabled {
		v.validateEmailConfig(cfg)
	}

	// Validate backup configuration
	if cfg.BackupEnabled {
		v.validateBackupConfig(cfg)
	}

	// Validate metrics configuration
	v.validateMetricsConfig(cfg)

	if len(v.errors) > 0 {
		return v.errors
	}

	return nil
}

// validateGlobalConfig validates global configuration settings
func (v *Validator) validateGlobalConfig(cfg *Config) {
	// Validate web port
	if cfg.WebPort == "" {
		v.AddError("web_port", "cannot be empty", cfg.WebPort)
	} else if !v.IsValidPort(cfg.WebPort) {
		v.AddError("web_port", "must be a valid port number (1-65535) or :port format", cfg.WebPort)
	}

	// Validate session timeout
	if cfg.SessionTimeout < 1 {
		v.AddError("session_timeout", "must be at least 1 hour", strconv.Itoa(cfg.SessionTimeout))
	} else if cfg.SessionTimeout > 168 { // 1 week
		v.AddError("session_timeout", "must not exceed 168 hours (1 week)", strconv.Itoa(cfg.SessionTimeout))
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"DEBUG": true, "INFO": true, "WARN": true, "ERROR": true, "FATAL": true,
	}
	if cfg.LogLevel == "" {
		v.AddError("log_level", "cannot be empty", cfg.LogLevel)
	} else if !validLogLevels[strings.ToUpper(cfg.LogLevel)] {
		v.AddError("log_level", "must be one of: DEBUG, INFO, WARN, ERROR, FATAL", cfg.LogLevel)
	}

	// Validate log rotation settings
	if cfg.LogMaxSize < 1 {
		v.AddError("log_max_size", "must be at least 1 MB", strconv.Itoa(cfg.LogMaxSize))
	} else if cfg.LogMaxSize > 1000 {
		v.AddError("log_max_size", "must not exceed 1000 MB", strconv.Itoa(cfg.LogMaxSize))
	}

	if cfg.LogMaxFiles < 1 {
		v.AddError("log_max_files", "must be at least 1", strconv.Itoa(cfg.LogMaxFiles))
	} else if cfg.LogMaxFiles > 100 {
		v.AddError("log_max_files", "must not exceed 100", strconv.Itoa(cfg.LogMaxFiles))
	}

	if cfg.LogMaxAgeDays < 1 {
		v.AddError("log_max_age_days", "must be at least 1 day", strconv.Itoa(cfg.LogMaxAgeDays))
	} else if cfg.LogMaxAgeDays > 365 {
		v.AddError("log_max_age_days", "must not exceed 365 days", strconv.Itoa(cfg.LogMaxAgeDays))
	}

	// Validate max connections
	if cfg.MaxConnections < 1 {
		v.AddError("max_connections", "must be at least 1", strconv.Itoa(cfg.MaxConnections))
	} else if cfg.MaxConnections > 100000 {
		v.AddError("max_connections", "must not exceed 100000", strconv.Itoa(cfg.MaxConnections))
	}
}

// validateProxyConfig validates a single proxy configuration
func (v *Validator) validateProxyConfig(cfg *ProxyConfig, index int) {
	prefix := fmt.Sprintf("proxies[%d]", index)

	// Validate ID
	if cfg.ID == "" {
		v.AddError(prefix+".id", "cannot be empty", cfg.ID)
	} else if !v.IsValidID(cfg.ID) {
		v.AddError(prefix+".id", "must contain only alphanumeric characters, hyphens, and underscores", cfg.ID)
	}

	// Validate name
	if cfg.Name == "" {
		v.AddError(prefix+".name", "cannot be empty", cfg.Name)
	} else if len(cfg.Name) > 100 {
		v.AddError(prefix+".name", "must not exceed 100 characters", cfg.Name)
	}

	// Validate listen address
	if cfg.ListenAddr == "" {
		v.AddError(prefix+".listen_addr", "cannot be empty", cfg.ListenAddr)
	} else {
		host, port, err := v.ParseHostPort(cfg.ListenAddr)
		if err != nil {
			v.AddError(prefix+".listen_addr", "invalid format (expected host:port)", cfg.ListenAddr)
		} else {
			if host != "" && !v.IsValidIP(host) && host != "0.0.0.0" && host != "::" && host != "localhost" {
				v.AddError(prefix+".listen_addr", "invalid host address", cfg.ListenAddr)
			}
			if port < 1 || port > 65535 {
				v.AddError(prefix+".listen_addr", "port must be between 1 and 65535", cfg.ListenAddr)
			}
		}
	}

	// Validate target address
	if cfg.TargetAddr == "" {
		v.AddError(prefix+".target_addr", "cannot be empty", cfg.TargetAddr)
	} else {
		host, port, err := v.ParseHostPort(cfg.TargetAddr)
		if err != nil {
			v.AddError(prefix+".target_addr", "invalid format (expected host:port)", cfg.TargetAddr)
		} else {
			if !v.IsValidIP(host) && !v.IsValidHostname(host) {
				v.AddError(prefix+".target_addr", "invalid host address or hostname", cfg.TargetAddr)
			}
			if port < 1 || port > 65535 {
				v.AddError(prefix+".target_addr", "port must be between 1 and 65535", cfg.TargetAddr)
			}
		}
	}

	// Check for port conflicts (listen and target cannot be the same)
	if cfg.ListenAddr != "" && cfg.TargetAddr != "" && cfg.ListenAddr == cfg.TargetAddr {
		v.AddError(prefix, "listen_addr and target_addr cannot be the same", cfg.ListenAddr)
	}

	// Validate timeouts
	if cfg.ConnectionTimeout < 0 {
		v.AddError(prefix+".connection_timeout", "must be non-negative", strconv.Itoa(cfg.ConnectionTimeout))
	} else if cfg.ConnectionTimeout > 300 {
		v.AddError(prefix+".connection_timeout", "must not exceed 300 seconds", strconv.Itoa(cfg.ConnectionTimeout))
	}

	if cfg.ReadTimeout < 0 {
		v.AddError(prefix+".read_timeout", "must be non-negative", strconv.Itoa(cfg.ReadTimeout))
	} else if cfg.ReadTimeout > 600 {
		v.AddError(prefix+".read_timeout", "must not exceed 600 seconds", strconv.Itoa(cfg.ReadTimeout))
	}

	// Validate retries
	if cfg.MaxRetries < 0 {
		v.AddError(prefix+".max_retries", "must be non-negative", strconv.Itoa(cfg.MaxRetries))
	} else if cfg.MaxRetries > 10 {
		v.AddError(prefix+".max_retries", "must not exceed 10", strconv.Itoa(cfg.MaxRetries))
	}

	// Validate max read size
	if cfg.MaxReadSize < 0 {
		v.AddError(prefix+".max_read_size", "must be non-negative", strconv.Itoa(cfg.MaxReadSize))
	} else if cfg.MaxReadSize > 65535 { // Modbus limit
		v.AddError(prefix+".max_read_size", "must not exceed 65535", strconv.Itoa(cfg.MaxReadSize))
	}

	// Validate description length
	if len(cfg.Description) > 500 {
		v.AddError(prefix+".description", "must not exceed 500 characters", strconv.Itoa(len(cfg.Description)))
	}

	// Validate tags
	for i, tag := range cfg.Tags {
		if len(tag) > 50 {
			v.AddError(fmt.Sprintf("%s.tags[%d]", prefix, i), "must not exceed 50 characters", tag)
		}
		if !v.IsValidTag(tag) {
			v.AddError(fmt.Sprintf("%s.tags[%d]", prefix, i), "must contain only alphanumeric characters, hyphens, and underscores", tag)
		}
	}
}

// validateTLSConfig validates TLS configuration
func (v *Validator) validateTLSConfig(cfg *Config) {
	if cfg.TLSCertFile == "" {
		v.AddError("tls_cert_file", "required when TLS is enabled", cfg.TLSCertFile)
	} else if !v.FileExists(cfg.TLSCertFile) {
		v.AddError("tls_cert_file", "certificate file does not exist or is not readable", cfg.TLSCertFile)
	}

	if cfg.TLSKeyFile == "" {
		v.AddError("tls_key_file", "required when TLS is enabled", cfg.TLSKeyFile)
	} else if !v.FileExists(cfg.TLSKeyFile) {
		v.AddError("tls_key_file", "key file does not exist or is not readable", cfg.TLSKeyFile)
	}
}

// validateCORSConfig validates CORS configuration
func (v *Validator) validateCORSConfig(cfg *Config) {
	// Check if origins array is empty
	if len(cfg.CORSAllowedOrigins) == 0 {
		v.AddError("cors_allowed_origins", "must contain at least one origin", "")
		return
	}

	for i, origin := range cfg.CORSAllowedOrigins {
		if origin == "*" {
			continue // Allow wildcard
		}
		// IsValidOrigin already checks URL validity and scheme (http/https)
		if !v.IsValidOrigin(origin) {
			v.AddError(fmt.Sprintf("cors_allowed_origins[%d]", i), "invalid origin format (must be http:// or https:// URL)", origin)
		}
	}

	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "OPTIONS": true, "HEAD": true,
	}
	for i, method := range cfg.CORSAllowedMethods {
		if !validMethods[strings.ToUpper(method)] {
			v.AddError(fmt.Sprintf("cors_allowed_methods[%d]", i), "invalid HTTP method", method)
		}
	}

	if len(cfg.CORSAllowedHeaders) == 0 {
		v.AddError("cors_allowed_headers", "must contain at least one header", "")
	}
}

// validateRateLimitConfig validates rate limiting configuration
func (v *Validator) validateRateLimitConfig(cfg *Config) {
	if cfg.RateLimitEnabled {
		if cfg.RateLimitRequests < 1 {
			v.AddError("rate_limit_requests", "must be at least 1", strconv.Itoa(cfg.RateLimitRequests))
		} else if cfg.RateLimitRequests > 10000 {
			v.AddError("rate_limit_requests", "must not exceed 10000", strconv.Itoa(cfg.RateLimitRequests))
		}

		if cfg.RateLimitBurst < 1 {
			v.AddError("rate_limit_burst", "must be at least 1", strconv.Itoa(cfg.RateLimitBurst))
		} else if cfg.RateLimitBurst > 1000 {
			v.AddError("rate_limit_burst", "must not exceed 1000", strconv.Itoa(cfg.RateLimitBurst))
		}

		if cfg.RateLimitBurst < cfg.RateLimitRequests {
			v.AddError("rate_limit_burst", "should be greater than or equal to rate_limit_requests",
				fmt.Sprintf("%d < %d", cfg.RateLimitBurst, cfg.RateLimitRequests))
		}
	}
}

// validateIPFilterConfig validates IP filter configuration
func (v *Validator) validateIPFilterConfig(cfg *Config) {
	if cfg.IPWhitelistEnabled {
		for i, ip := range cfg.IPWhitelist {
			if !v.IsValidIPOrCIDR(ip) {
				v.AddError(fmt.Sprintf("ip_whitelist[%d]", i), "invalid IP address or CIDR range", ip)
			}
		}
	}

	if cfg.IPBlacklistEnabled {
		for i, ip := range cfg.IPBlacklist {
			if !v.IsValidIPOrCIDR(ip) {
				v.AddError(fmt.Sprintf("ip_blacklist[%d]", i), "invalid IP address or CIDR range", ip)
			}
		}
	}

	// Check if both are enabled (not recommended)
	if cfg.IPWhitelistEnabled && cfg.IPBlacklistEnabled {
		v.AddError("ip_filter", "both whitelist and blacklist enabled (not recommended)", "")
	}
}

// validateEmailConfig validates email configuration
func (v *Validator) validateEmailConfig(cfg *Config) {
	if cfg.EmailSMTPServer == "" {
		v.AddError("email_smtp_server", "required when email is enabled", cfg.EmailSMTPServer)
	} else if !v.IsValidHostname(cfg.EmailSMTPServer) && !v.IsValidIP(cfg.EmailSMTPServer) {
		v.AddError("email_smtp_server", "invalid hostname or IP address", cfg.EmailSMTPServer)
	}

	if cfg.EmailSMTPPort < 1 || cfg.EmailSMTPPort > 65535 {
		v.AddError("email_smtp_port", "must be between 1 and 65535", strconv.Itoa(cfg.EmailSMTPPort))
	}

	if cfg.EmailFrom == "" {
		v.AddError("email_from", "required when email is enabled", cfg.EmailFrom)
	} else if !v.IsValidEmail(cfg.EmailFrom) {
		v.AddError("email_from", "invalid email address", cfg.EmailFrom)
	}

	if cfg.EmailTo == "" {
		v.AddError("email_to", "required when email is enabled", cfg.EmailTo)
	} else {
		// Validate multiple email addresses
		emails := strings.Split(cfg.EmailTo, ",")
		for _, email := range emails {
			if !v.IsValidEmail(strings.TrimSpace(email)) {
				v.AddError("email_to", "invalid email address", cfg.EmailTo)
				break
			}
		}
	}
}

// validateBackupConfig validates backup configuration
func (v *Validator) validateBackupConfig(cfg *Config) {
	if cfg.BackupPath == "" {
		v.AddError("backup_path", "required when backup is enabled", cfg.BackupPath)
	} else if !v.IsValidPath(cfg.BackupPath) {
		v.AddError("backup_path", "invalid path", cfg.BackupPath)
	}

	validIntervals := map[string]bool{
		"hourly": true, "daily": true, "weekly": true, "monthly": true,
	}
	if !validIntervals[cfg.BackupInterval] {
		v.AddError("backup_interval", "must be one of: hourly, daily, weekly, monthly", cfg.BackupInterval)
	}

	if cfg.BackupRetention < 1 {
		v.AddError("backup_retention", "must be at least 1", strconv.Itoa(cfg.BackupRetention))
	} else if cfg.BackupRetention > 365 {
		v.AddError("backup_retention", "must not exceed 365", strconv.Itoa(cfg.BackupRetention))
	}

	if !cfg.BackupDatabase && !cfg.BackupConfig {
		v.AddError("backup", "at least one of backup_database or backup_config must be enabled", "")
	}
}

// validateMetricsConfig validates metrics configuration
func (v *Validator) validateMetricsConfig(cfg *Config) {
	if cfg.MetricsEnabled {
		if cfg.MetricsPort == "" {
			v.AddError("metrics_port", "required when metrics is enabled", cfg.MetricsPort)
		} else if !v.IsValidPort(cfg.MetricsPort) {
			v.AddError("metrics_port", "must be a valid port number (1-65535) or :port format", cfg.MetricsPort)
		}
	}
}

// AddError adds a validation error
func (v *Validator) AddError(field, message, value string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// IsValidID checks if a string is a valid ID
func (v *Validator) IsValidID(id string) bool {
	if id == "" {
		return false
	}
	// Alphanumeric, hyphens, and underscores only
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, id)
	return matched
}

// IsValidIP checks if a string is a valid IP address (v4 or v6)
func (v *Validator) IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsValidIPOrCIDR checks if a string is a valid IP address or CIDR range
func (v *Validator) IsValidIPOrCIDR(ip string) bool {
	// Try as IP first
	if net.ParseIP(ip) != nil {
		return true
	}

	// Try as CIDR
	_, _, err := net.ParseCIDR(ip)
	return err == nil
}

// IsValidHostname checks if a string is a valid hostname
func (v *Validator) IsValidHostname(hostname string) bool {
	if hostname == "" {
		return false
	}

	// Basic hostname validation
	// RFC 1123: hostname can contain letters, digits, hyphens, and dots
	// Must not start or end with hyphen
	// Must not have consecutive dots
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`, hostname)
	return matched
}

// IsValidPort checks if a string is a valid port specification
func (v *Validator) IsValidPort(portStr string) bool {
	// Allow :port format or just port number
	if strings.HasPrefix(portStr, ":") {
		portStr = portStr[1:]
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return false
	}

	return port >= 1 && port <= 65535
}

// ParseHostPort parses a host:port string
func (v *Validator) ParseHostPort(addr string) (host string, port int, err error) {
	// Split by last colon to handle IPv6 addresses
	lastColon := strings.LastIndex(addr, ":")
	if lastColon == -1 {
		return "", 0, errors.New("missing port")
	}

	host = addr[:lastColon]
	portStr := addr[lastColon+1:]

	port, err = strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %w", err)
	}

	return host, port, nil
}

// IsValidURL checks if a string is a valid URL
func (v *Validator) IsValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}

// IsValidOrigin checks if a string is a valid CORS origin
func (v *Validator) IsValidOrigin(origin string) bool {
	// Allow specific formats like http://localhost:3000
	if !v.IsValidURL(origin) {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// Only allow http and https
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	return true
}

// IsValidEmail checks if a string is a valid email address
func (v *Validator) IsValidEmail(email string) bool {
	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidTag checks if a string is a valid tag
func (v *Validator) IsValidTag(tag string) bool {
	if tag == "" {
		return false
	}
	// Alphanumeric, hyphens, underscores, and spaces allowed
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\- ]+$`, tag)
	return matched
}

// IsValidPath checks if a string is a valid file path
func (v *Validator) IsValidPath(path string) bool {
	// Check if path is absolute or relative
	if filepath.IsAbs(path) {
		return true
	}

	// Check for invalid characters
	if strings.ContainsAny(path, "<>:\"|?*") {
		return false
	}

	return true
}

// FileExists checks if a file exists and is readable
func (v *Validator) FileExists(path string) bool {
	// Check if absolute path
	if !filepath.IsAbs(path) {
		// Convert to absolute path
		abs, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		path = abs
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if it's a regular file
	return !info.IsDir()
}

// ValidatePortRange checks if a port range is valid
func (v *Validator) ValidatePortRange(start, end int) error {
	if start < 1 || start > 65535 {
		return fmt.Errorf("start port must be between 1 and 65535, got %d", start)
	}
	if end < 1 || end > 65535 {
		return fmt.Errorf("end port must be between 1 and 65535, got %d", end)
	}
	if start > end {
		return fmt.Errorf("start port (%d) cannot be greater than end port (%d)", start, end)
	}
	return nil
}

// ValidatePortAvailable checks if a port is available (not in use)
func (v *Validator) ValidatePortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("port %d is already in use", port)
	}
	ln.Close()
	return nil
}

// GetErrors returns the list of validation errors
func (v *Validator) GetErrors() ValidationErrors {
	return v.errors
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Clear clears all validation errors
func (v *Validator) Clear() {
	v.errors = make(ValidationErrors, 0)
}

// ValidateProxyConfigQuick is a quick validation for proxy creation/update
func ValidateProxyConfigQuick(cfg *ProxyConfig) error {
	v := NewValidator()
	v.validateProxyConfig(cfg, 0)
	if v.HasErrors() {
		return v.errors
	}
	return nil
}
