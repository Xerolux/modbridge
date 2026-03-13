package config

import (
	"testing"
)

// getValidBaseConfig returns a valid base configuration for testing
func getValidBaseConfig() Config {
	return Config{
		WebPort:             ":8080",
		Proxies:             []ProxyConfig{},
		LogLevel:            "INFO",
		LogMaxSize:          100,
		LogMaxFiles:         10,
		LogMaxAgeDays:       30,
		SessionTimeout:      24,
		RateLimitEnabled:    true,
		RateLimitRequests:   60,
		RateLimitBurst:      100,
		MetricsEnabled:      true,
		MetricsPort:         ":9090",
		MaxConnections:      1000,
		CORSAllowedOrigins:  []string{"http://localhost:3000"},
		CORSAllowedMethods:  []string{"GET", "POST"},
		CORSAllowedHeaders:  []string{"Content-Type"},
	}
}

func TestValidator_ValidConfig(t *testing.T) {
	v := NewValidator()
	cfg := getValidBaseConfig()

	err := v.Validate(&cfg)
	if err != nil {
		t.Errorf("Expected no errors for valid config, got: %v", err)
	}
}

func TestValidator_InvalidWebPort(t *testing.T) {
	tests := []struct {
		name    string
		webPort string
		wantErr bool
	}{
		{"empty port", "", true},
		{"invalid port", ":abc", true},
		{"port out of range", ":99999", true},
		{"zero port", ":0", true},
		{"negative port", ":-1", true},
		{"valid port", ":8080", false},
		{"valid port without colon", "8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.WebPort = tt.webPort

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_InvalidProxyConfig(t *testing.T) {
	tests := []struct {
		name    string
		proxy   ProxyConfig
		wantErr bool
	}{
		{
			name: "empty ID",
			proxy: ProxyConfig{
				ID:         "",
				Name:       "Test Proxy",
				ListenAddr: ":8080",
				TargetAddr: "localhost:502",
			},
			wantErr: true,
		},
		{
			name: "invalid ID with spaces",
			proxy: ProxyConfig{
				ID:         "test proxy",
				Name:       "Test Proxy",
				ListenAddr: ":8080",
				TargetAddr: "localhost:502",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			proxy: ProxyConfig{
				ID:         "test-proxy",
				Name:       "",
				ListenAddr: ":8080",
				TargetAddr: "localhost:502",
			},
			wantErr: true,
		},
		{
			name: "name too long",
			proxy: ProxyConfig{
				ID:         "test-proxy",
				Name:        string(make([]byte, 101)), // 101 characters
				ListenAddr:  ":8080",
				TargetAddr:  "localhost:502",
			},
			wantErr: true,
		},
		{
			name: "empty listen address",
			proxy: ProxyConfig{
				ID:         "test-proxy",
				Name:       "Test Proxy",
				ListenAddr: "",
				TargetAddr: "localhost:502",
			},
			wantErr: true,
		},
		{
			name: "invalid listen address",
			proxy: ProxyConfig{
				ID:         "test-proxy",
				Name:       "Test Proxy",
				ListenAddr: "invalid",
				TargetAddr: "localhost:502",
			},
			wantErr: true,
		},
		{
			name: "empty target address",
			proxy: ProxyConfig{
				ID:         "test-proxy",
				Name:       "Test Proxy",
				ListenAddr: ":8080",
				TargetAddr: "",
			},
			wantErr: true,
		},
		{
			name: "same listen and target address",
			proxy: ProxyConfig{
				ID:         "test-proxy",
				Name:       "Test Proxy",
				ListenAddr: ":8080",
				TargetAddr: ":8080",
			},
			wantErr: true,
		},
		{
			name: "valid proxy config",
			proxy: ProxyConfig{
				ID:         "test-proxy",
				Name:       "Test Proxy",
				ListenAddr: ":8080",
				TargetAddr: "localhost:502",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.Proxies = []ProxyConfig{tt.proxy}

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_TimeoutValidation(t *testing.T) {
	tests := []struct {
		name              string
		connectionTimeout int
		readTimeout       int
		wantErr           bool
	}{
		{"negative connection timeout", -1, 30, true},
		{"connection timeout too high", 301, 30, true},
		{"negative read timeout", 10, -1, true},
		{"read timeout too high", 10, 601, true},
		{"valid timeouts", 10, 30, false},
		{"zero timeout (allowed)", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.Proxies = []ProxyConfig{
				{
					ID:                "test-proxy",
					Name:              "Test Proxy",
					ListenAddr:        ":8080",
					TargetAddr:        "localhost:502",
					ConnectionTimeout: tt.connectionTimeout,
					ReadTimeout:       tt.readTimeout,
				},
			}

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_RetryValidation(t *testing.T) {
	tests := []struct {
		name      string
		maxRetries int
		wantErr   bool
	}{
		{"negative retries", -1, true},
		{"too many retries", 11, true},
		{"valid retries", 3, false},
		{"zero retries", 0, false},
		{"max retries", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.Proxies = []ProxyConfig{
				{
					ID:         "test-proxy",
					Name:       "Test Proxy",
					ListenAddr: ":8080",
					TargetAddr: "localhost:502",
					MaxRetries: tt.maxRetries,
				},
			}

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_IPValidation(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		valid   bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv4 localhost", "127.0.0.1", true},
		{"valid IPv6", "::1", true},
		{"valid IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"valid CIDR", "192.168.1.0/24", true},
		{"valid CIDR IPv6", "2001:db8::/32", true},
		{"invalid IP", "256.256.256.256", false},
		{"invalid IP format", "192.168.1", false},
		{"invalid IP with text", "192.168.1.abc", false},
		{"empty string", "", false},
	}

	v := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.IsValidIPOrCIDR(tt.ip)
			if result != tt.valid {
				t.Errorf("IsValidIPOrCIDR(%q) = %v, want %v", tt.ip, result, tt.valid)
			}
		})
	}
}

func TestValidator_HostnameValidation(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		valid    bool
	}{
		{"valid hostname", "example.com", true},
		{"valid hostname with subdomain", "sub.example.com", true},
		{"valid hostname with hyphen", "my-server.example.com", true},
		{"localhost", "localhost", true},
		{"starts with hyphen", "-example.com", false},
		{"ends with hyphen", "example-.com", false},
		{"consecutive dots", "example..com", false},
		{"starts with dot", ".example.com", false},
		{"ends with dot", "example.com.", false},
		{"empty string", "", false},
		{"contains invalid chars", "example$.com", false},
	}

	v := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.IsValidHostname(tt.hostname)
			if result != tt.valid {
				t.Errorf("IsValidHostname(%q) = %v, want %v", tt.hostname, result, tt.valid)
			}
		})
	}
}

func TestValidator_EmailValidation(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"valid email", "user@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"valid email with plus", "user+tag@example.com", true},
		{"valid email with numbers", "user123@example.com", true},
		{"missing @", "userexample.com", false},
		{"missing domain", "user@", false},
		{"missing user", "@example.com", false},
		{"invalid chars", "user$@example.com", false},
		{"empty string", "", false},
	}

	v := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.IsValidEmail(tt.email)
			if result != tt.valid {
				t.Errorf("IsValidEmail(%q) = %v, want %v", tt.email, result, tt.valid)
			}
		})
	}
}

func TestValidator_RateLimitValidation(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		requests int
		burst    int
		wantErr  bool
	}{
		{"valid rate limiting", true, 60, 100, false},
		{"disabled rate limiting", false, 0, 0, false},
		{"zero requests", true, 0, 10, true},
		{"requests too high", true, 10001, 100, true},
		{"burst too high", true, 60, 1001, true},
		{"burst less than requests", true, 100, 60, true},
		{"burst equals requests", true, 60, 60, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.RateLimitEnabled = tt.enabled
			cfg.RateLimitRequests = tt.requests
			cfg.RateLimitBurst = tt.burst

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_SessionTimeoutValidation(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		wantErr bool
	}{
		{"valid timeout", 24, false},
		{"minimum timeout", 1, false},
		{"zero timeout", 0, true},
		{"negative timeout", -1, true},
		{"too high timeout", 169, true},
		{"max valid timeout", 168, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.SessionTimeout = tt.timeout

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_LogLevelValidation(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		wantErr  bool
	}{
		{"valid DEBUG", "DEBUG", false},
		{"valid INFO", "INFO", false},
		{"valid WARN", "WARN", false},
		{"valid ERROR", "ERROR", false},
		{"valid FATAL", "FATAL", false},
		{"lowercase info", "info", false},
		{"mixed case Info", "Info", false},
		{"invalid level", "TRACE", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.LogLevel = tt.logLevel

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_TagValidation(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name  string
		tag   string
		valid bool
	}{
		{"valid tag", "production", true},
		{"valid tag with hyphen", "test-server", true},
		{"valid tag with underscore", "server_1", true},
		{"valid tag with space", "server 1", true},
		{"tag too long", string(make([]byte, 51)), false},
		{"tag with special chars", "server$", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.IsValidTag(tt.tag)
			if result != tt.valid {
				t.Errorf("IsValidTag(%q) = %v, want %v", tt.tag, result, tt.valid)
			}
		})
	}
}

func TestValidator_URLValidation(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name  string
		url   string
		valid bool
	}{
		{"valid HTTP URL", "http://example.com", true},
		{"valid HTTPS URL", "https://example.com", true},
		{"valid URL with port", "http://localhost:8080", true},
		{"valid URL with path", "https://example.com/path", true},
		{"missing scheme", "example.com", false},
		{"empty string", "", false},
		{"invalid URL", "://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.IsValidURL(tt.url)
			if result != tt.valid {
				t.Errorf("IsValidURL(%q) = %v, want %v", tt.url, result, tt.valid)
			}
		})
	}
}

func TestValidator_CORSValidation(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
		wantErr bool
	}{
		{"valid origins", []string{"http://localhost:3000", "https://example.com"}, false},
		{"wildcard origin", []string{"*"}, false},
		{"invalid origin", []string{"not-a-url"}, true},
		{"ftp origin", []string{"ftp://example.com"}, true},
		{"empty origins", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.CORSAllowedOrigins = tt.origins

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_BackupValidation(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		path        string
		interval    string
		retention   int
		backupDB    bool
		backupCfg   bool
		wantErr     bool
	}{
		{"valid backup config", true, "./backups", "daily", 7, true, true, false},
		{"valid hourly", true, "./backups", "hourly", 24, true, true, false},
		{"valid weekly", true, "./backups", "weekly", 4, true, true, false},
		{"valid monthly", true, "./backups", "monthly", 12, true, true, false},
		{"invalid interval", true, "./backups", "invalid", 7, true, true, true},
		{"retention too low", true, "./backups", "daily", 0, true, true, true},
		{"retention too high", true, "./backups", "daily", 400, true, true, true},
		{"nothing enabled", true, "./backups", "daily", 7, false, false, true},
		{"only database", true, "./backups", "daily", 7, true, false, false},
		{"only config", true, "./backups", "daily", 7, false, true, false},
		{"disabled backup", false, "", "", 0, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.BackupEnabled = tt.enabled
			cfg.BackupPath = tt.path
			cfg.BackupInterval = tt.interval
			cfg.BackupRetention = tt.retention
			cfg.BackupDatabase = tt.backupDB
			cfg.BackupConfig = tt.backupCfg

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_MaxConnectionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		maxConn int
		wantErr bool
	}{
		{"valid connections", 1000, false},
		{"minimum connections", 1, false},
		{"zero connections", 0, true},
		{"negative connections", -1, true},
		{"too many connections", 100001, true},
		{"max valid connections", 100000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			cfg := getValidBaseConfig()
			cfg.MaxConnections = tt.maxConn

			err := v.Validate(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateProxyConfigQuick(t *testing.T) {
	tests := []struct {
		name    string
		proxy   ProxyConfig
		wantErr bool
	}{
		{
			name: "valid proxy",
			proxy: ProxyConfig{
				ID:         "test-proxy",
				Name:       "Test Proxy",
				ListenAddr: ":8080",
				TargetAddr: "localhost:502",
			},
			wantErr: false,
		},
		{
			name: "invalid proxy - empty ID",
			proxy: ProxyConfig{
				ID:         "",
				Name:       "Test Proxy",
				ListenAddr: ":8080",
				TargetAddr: "localhost:502",
			},
			wantErr: true,
		},
		{
			name: "invalid proxy - empty name",
			proxy: ProxyConfig{
				ID:         "test-proxy",
				Name:       "",
				ListenAddr: ":8080",
				TargetAddr: "localhost:502",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProxyConfigQuick(&tt.proxy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProxyConfigQuick() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_PortRangeValidation(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		start   int
		end     int
		wantErr bool
	}{
		{"valid range", 8000, 9000, false},
		{"single port", 8080, 8080, false},
		{"start below minimum", 0, 8080, true},
		{"end above maximum", 8080, 70000, true},
		{"start greater than end", 9000, 8000, true},
		{"minimum valid range", 1, 1, false},
		{"maximum valid range", 65535, 65535, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePortRange(tt.start, tt.end)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePortRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errors := ValidationErrors{
		{Field: "field1", Message: "error 1"},
		{Field: "field2", Message: "error 2", Value: "bad_value"},
	}

	errStr := errors.Error()
	if !containsString(errStr, "2 error(s)") {
		t.Errorf("Error() should contain error count, got: %s", errStr)
	}
	if !containsString(errStr, "field1") {
		t.Errorf("Error() should contain field1, got: %s", errStr)
	}
	if !containsString(errStr, "field2") {
		t.Errorf("Error() should contain field2, got: %s", errStr)
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     ValidationError
		contains []string
	}{
		{
			name: "error without value",
			err: ValidationError{Field: "test_field", Message: "is invalid"},
			contains: []string{"test_field", "is invalid"},
		},
		{
			name: "error with value",
			err: ValidationError{Field: "test_field", Message: "is invalid", Value: "bad"},
			contains: []string{"test_field", "is invalid", "bad"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.contains {
				if !containsString(errStr, substr) {
					t.Errorf("Error() should contain %q, got: %s", substr, errStr)
				}
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (
		s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
