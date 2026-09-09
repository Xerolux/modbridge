// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"
)

// FlexibleTags is a custom type that can unmarshal both string and array for tags
type FlexibleTags []string

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleTags
func (ft *FlexibleTags) UnmarshalJSON(data []byte) error {
	// Handle null
	if string(data) == "null" {
		*ft = FlexibleTags{}
		return nil
	}

	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		// Split comma-separated string into array
		if str == "" {
			*ft = FlexibleTags{}
		} else {
			tags := strings.Split(str, ",")
			result := make([]string, 0, len(tags))
			for _, tag := range tags {
				trimmed := strings.TrimSpace(tag)
				if trimmed != "" {
					result = append(result, trimmed)
				}
			}
			*ft = FlexibleTags(result)
		}
		return nil
	}

	// Try to unmarshal as array
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*ft = FlexibleTags(arr)
	return nil
}

// MarshalJSON marshals FlexibleTags as JSON array
func (ft FlexibleTags) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(ft))
}

// ProxyConfig defines the configuration for a single proxy instance.
type ProxyConfig struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	ListenAddr        string `json:"listen_addr"`
	TargetAddr        string `json:"target_addr"`
	Enabled           bool   `json:"enabled"`
	Paused            bool   `json:"paused"`             // Paused state (different from Enabled)
	ConnectionTimeout int    `json:"connection_timeout"` // Connection timeout in seconds (default: 10)
	ReadTimeout       int    `json:"read_timeout"`       // Read timeout in seconds (default: 30)
	MaxRetries        int    `json:"max_retries"`        // Max retry attempts (default: 3)
	Description       string `json:"description"`        // User description
	MaxReadSize       int    `json:"max_read_size"`
	ConnectDelayMs    int    `json:"connect_delay_ms"`   // Delay after TCP connect before first request (ms). For slow devices like Huawei inverters/sDongles.
	MaxTargetConns    int    `json:"max_target_conns"`   // Max simultaneous connections to the target (0 = default 10). Set to 1 for single-session devices like SolarEdge/SunSpec inverters.
	MinRequestGapMs   int    `json:"min_request_gap_ms"` // Minimum pause between two requests to the target (ms). For devices that drop back-to-back requests.
	RequestTimeoutMs  int    `json:"request_timeout_ms"` // Hard cap for one client request incl. retries (ms, 0 = derived). Keep below the client's own timeout.
	CacheEnabled      bool   `json:"cache_enabled"`      // Serve repeated reads from a cache instead of asking the target every time
	CacheTTLMs        int    `json:"cache_ttl_ms"`       // Lifetime of a cached read (ms, 0 = 5000). A cached value is not the live value.
	PollIntervalMs    int    `json:"poll_interval_ms"`   // Refresh cached reads in the background at this interval (ms, 0 = passive cache only)
	CalibratedAt      string `json:"calibrated_at"`      // When this proxy was last measured (RFC3339). Informational: tells you how old the tuned values are.
	DeviceProfile     string `json:"device_profile"`     // Device profile last applied in the UI. Purely informational: it records which preset the settings came from, the proxy behaviour follows the individual fields.
	// LastCalibration is the full report of the most recent measurement, kept
	// verbatim so it can be read again later — a run takes up to 90 seconds and
	// locks out every client, so nobody should have to repeat one just to see
	// what it said. Held as raw JSON: the shape belongs to pkg/proxy, and
	// copying the type here would only create a second one to keep in step.
	// Recording a measurement is not applying it; the tuning fields above are
	// only ever changed by a deliberate act.
	LastCalibration json.RawMessage `json:"last_calibration,omitempty"`
	Tags            FlexibleTags    `json:"tags"`
	// Protocol controls the wire format used when talking to the target.
	// "tcp"     – standard Modbus TCP (MBAP header, default)
	// "rtu-tcp" – Modbus RTU over TCP: client sends TCP frames, proxy strips
	//             the MBAP header, appends CRC-16, forwards raw RTU frames to
	//             the target, then wraps the RTU response back in a TCP frame.
	Protocol string `json:"protocol"`
}

// Config holds the global configuration.
type Config struct {
	WebPort             string        `json:"web_port"`
	AdminPassHash       string        `json:"admin_pass_hash"`       // Empty means first-run
	ForcePasswordChange bool          `json:"force_password_change"` // Forces password change on next login
	MultiUser           bool          `json:"multi_user"`            // Enable DB-backed multi-user authentication
	Proxies             []ProxyConfig `json:"proxies"`

	LogLevel      string `json:"log_level"`
	LogMaxSize    int    `json:"log_max_size"`
	LogMaxFiles   int    `json:"log_max_files"`
	LogMaxAgeDays int    `json:"log_max_age_days"`

	TLSEnabled     bool   `json:"tls_enabled"`
	TLSCertFile    string `json:"tls_cert_file"`
	TLSKeyFile     string `json:"tls_key_file"`
	SessionTimeout int    `json:"session_timeout"`

	CORSAllowedOrigins []string `json:"cors_allowed_origins"`
	CORSAllowedMethods []string `json:"cors_allowed_methods"`
	CORSAllowedHeaders []string `json:"cors_allowed_headers"`

	RateLimitEnabled  bool `json:"rate_limit_enabled"`
	RateLimitRequests int  `json:"rate_limit_requests"`
	RateLimitBurst    int  `json:"rate_limit_burst"`

	IPWhitelistEnabled bool     `json:"ip_whitelist_enabled"`
	IPWhitelist        []string `json:"ip_whitelist"`
	IPBlacklistEnabled bool     `json:"ip_blacklist_enabled"`
	IPBlacklist        []string `json:"ip_blacklist"`

	EmailEnabled        bool   `json:"email_enabled"`
	EmailSMTPServer     string `json:"email_smtp_server"`
	EmailSMTPPort       int    `json:"email_smtp_port"`
	EmailFrom           string `json:"email_from"`
	EmailTo             string `json:"email_to"`
	EmailUsername       string `json:"email_username"`
	EmailPassword       string `json:"email_password"`
	EmailAlertOnError   bool   `json:"email_alert_on_error"`
	EmailAlertOnWarning bool   `json:"email_alert_on_warning"`

	BackupEnabled   bool   `json:"backup_enabled"`
	BackupInterval  string `json:"backup_interval"`
	BackupRetention int    `json:"backup_retention"`
	BackupPath      string `json:"backup_path"`
	BackupDatabase  bool   `json:"backup_database"`
	BackupConfig    bool   `json:"backup_config"`

	MetricsEnabled bool   `json:"metrics_enabled"`
	MetricsPort    string `json:"metrics_port"`

	DebugMode      bool `json:"debug_mode"`
	MaxConnections int  `json:"max_connections"`
}

// Manager handles config persistence.
type Manager struct {
	mu       sync.RWMutex
	path     string
	cfg      Config
	previous *Config // snapshot before the last Update call, enables Rollback
}

// NewManager creates a config manager.
func NewManager(path string) *Manager {
	return &Manager{
		path: path,
		cfg:  DefaultConfig(),
	}
}

// DefaultConfig returns the compiled-in defaults. Load starts from these, so a
// config.json that omits a key keeps the default instead of falling back to
// the zero value.
func DefaultConfig() Config {
	return Config{
		WebPort:             ":8080",
		Proxies:             []ProxyConfig{},
		LogLevel:            "INFO",
		LogMaxSize:          100,
		LogMaxFiles:         10,
		LogMaxAgeDays:       30,
		TLSEnabled:          false,
		SessionTimeout:      24,
		CORSAllowedOrigins:  []string{"http://localhost:3000", "http://localhost:8080"}, // Default local dev origins
		CORSAllowedMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowedHeaders:  []string{"Content-Type", "Authorization"},
		RateLimitEnabled:    true,
		RateLimitRequests:   60,
		RateLimitBurst:      100,
		IPWhitelistEnabled:  false,
		IPBlacklistEnabled:  false,
		EmailEnabled:        false,
		EmailAlertOnError:   true,
		EmailAlertOnWarning: false,
		BackupEnabled:       true,
		BackupInterval:      "daily",
		BackupRetention:     7,
		BackupPath:          "./backups",
		BackupDatabase:      true,
		BackupConfig:        true,
		MetricsEnabled:      true,
		MetricsPort:         ":9090",
		DebugMode:           false,
		MaxConnections:      1000,
		// Multi-user (DB-backed auth) is the default mode. Operators can opt
		// out explicitly via {"multi_user": false} in config.json.
		MultiUser: true,
	}
}

// Load reads config from disk.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	f, err := os.Open(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// Unmarshal into the defaults rather than into a zero Config: a key that
	// config.json omits then keeps its default instead of becoming 0 or "".
	// This also settles the bool case — an explicit {"multi_user": false}
	// overwrites the default, an absent key does not.
	cfg := DefaultConfig()
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}

	m.cfg = cfg
	return nil
}

// writeConfigFile persists the config atomically: it writes to a temp file,
// fsyncs it, and renames it over the target. A crash mid-write would
// otherwise truncate config.json in place, and the next successful save
// would permanently overwrite the operator's config with compiled defaults.
//
// The rename can fail on a path that is not a plain file in a plain
// directory. Bind-mounting config.json into a container is the case that
// shows up in practice: the mount point cannot be replaced, and rename
// returns EBUSY (or EXDEV when the temp file and the target end up on
// different filesystems). Falling back to an in-place write keeps the UI
// working there; it gives up atomicity, which is the lesser problem.
func (m *Manager) writeConfigFile() error {
	data, err := json.MarshalIndent(m.cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := m.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	if err := os.Rename(tmp, m.path); err != nil {
		os.Remove(tmp)
		if !isReplaceUnsupported(err) {
			return err
		}
		return m.writeInPlace(data)
	}
	return nil
}

// writeInPlace overwrites the target without replacing the directory entry.
func (m *Manager) writeInPlace(data []byte) error {
	f, err := os.OpenFile(m.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}

// isReplaceUnsupported reports whether a rename failed because the target
// cannot be replaced, rather than because of a genuine I/O problem.
func isReplaceUnsupported(err error) bool {
	return errors.Is(err, syscall.EBUSY) ||
		errors.Is(err, syscall.EXDEV) ||
		errors.Is(err, syscall.EPERM) ||
		errors.Is(err, syscall.EACCES) ||
		errors.Is(err, syscall.EISDIR) ||
		errors.Is(err, syscall.ENOTDIR)
}

// Save writes config to disk.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.writeConfigFile()
}

func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.deepCopyConfig(m.cfg)
}

// deepCopyConfig creates a deep copy of the config.
// This is optimized to only copy slices, not individual structs.
func (m *Manager) deepCopyConfig(c Config) Config {
	result := c

	// Deep copy all slices to prevent concurrent modification issues
	if c.Proxies != nil {
		result.Proxies = make([]ProxyConfig, len(c.Proxies))
		for i := range c.Proxies {
			// Copy struct by value (shallow copy is fine for struct)
			// Then copy the Tags slice if present
			result.Proxies[i] = c.Proxies[i]
			if c.Proxies[i].Tags != nil {
				result.Proxies[i].Tags = make(FlexibleTags, len(c.Proxies[i].Tags))
				copy(result.Proxies[i].Tags, c.Proxies[i].Tags)
			}
		}
	}
	if c.CORSAllowedOrigins != nil {
		result.CORSAllowedOrigins = make([]string, len(c.CORSAllowedOrigins))
		copy(result.CORSAllowedOrigins, c.CORSAllowedOrigins)
	}
	if c.CORSAllowedMethods != nil {
		result.CORSAllowedMethods = make([]string, len(c.CORSAllowedMethods))
		copy(result.CORSAllowedMethods, c.CORSAllowedMethods)
	}
	if c.CORSAllowedHeaders != nil {
		result.CORSAllowedHeaders = make([]string, len(c.CORSAllowedHeaders))
		copy(result.CORSAllowedHeaders, c.CORSAllowedHeaders)
	}
	if c.IPWhitelist != nil {
		result.IPWhitelist = make([]string, len(c.IPWhitelist))
		copy(result.IPWhitelist, c.IPWhitelist)
	}
	if c.IPBlacklist != nil {
		result.IPBlacklist = make([]string, len(c.IPBlacklist))
		copy(result.IPBlacklist, c.IPBlacklist)
	}

	return result
}

func (m *Manager) Update(fn func(*Config) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Snapshot current config for potential rollback.
	prev := m.deepCopyConfig(m.cfg)

	// Create a deep copy to modify
	newCfg := m.deepCopyConfig(m.cfg)

	if err := fn(&newCfg); err != nil {
		return err
	}

	// Reject an update that would make the config invalid. The comparison
	// against the current config matters: an install whose config.json is
	// already invalid must still be able to change proxy state, so only newly
	// introduced validation errors are refused.
	if err := NewValidator().Validate(&newCfg); err != nil {
		if NewValidator().Validate(&m.cfg) == nil {
			return fmt.Errorf("refusing invalid configuration: %w", err)
		}
	}

	m.previous = &prev
	m.cfg = newCfg

	// Save to disk immediately
	return m.writeConfigFile()
}

// CanRollback reports whether a previous config snapshot is available.
func (m *Manager) CanRollback() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.previous != nil
}

// Rollback restores the configuration to the state before the last Update call.
// Returns an error if no previous snapshot exists or the save fails.
func (m *Manager) Rollback() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.previous == nil {
		return fmt.Errorf("no previous configuration to roll back to")
	}

	restored := m.deepCopyConfig(*m.previous)
	m.previous = nil // consume the snapshot
	m.cfg = restored

	return m.writeConfigFile()
}

// Validate validates the current configuration
func (m *Manager) Validate() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	validator := NewValidator()
	return validator.Validate(&m.cfg)
}

// ValidateWithErrors validates the configuration and returns detailed errors
func (m *Manager) ValidateWithErrors() (*Validator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	validator := NewValidator()
	err := validator.Validate(&m.cfg)
	return validator, err
}

// AddProxy validates and adds a new proxy configuration
func (m *Manager) AddProxy(proxy ProxyConfig) error {
	// Validate the proxy configuration first
	if err := ValidateProxyConfigQuick(&proxy); err != nil {
		return err
	}

	return m.Update(func(c *Config) error {
		// Check for duplicate ID
		for _, p := range c.Proxies {
			if p.ID == proxy.ID {
				return fmt.Errorf("proxy with ID %s already exists", proxy.ID)
			}
		}
		c.Proxies = append(c.Proxies, proxy)
		return nil
	})
}

// UpdateProxy validates and updates an existing proxy configuration
func (m *Manager) UpdateProxy(id string, updateFn func(*ProxyConfig) error) error {
	return m.Update(func(c *Config) error {
		for i, p := range c.Proxies {
			if p.ID == id {
				// Create a copy to validate
				updated := p
				if err := updateFn(&updated); err != nil {
					return err
				}

				// Validate the updated configuration
				if err := ValidateProxyConfigQuick(&updated); err != nil {
					return err
				}

				c.Proxies[i] = updated
				return nil
			}
		}
		return fmt.Errorf("proxy with ID %s not found", id)
	})
}
