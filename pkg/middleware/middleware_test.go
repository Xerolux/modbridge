package middleware

import (
	"testing"

	"modbusproxy/pkg/config"
)

func TestValidateProxyConfig(t *testing.T) {
	v := NewValidator()

	t.Run("Valid config", func(t *testing.T) {
		cfg := config.ProxyConfig{
			ID:                "test-id",
			Name:              "Test Proxy",
			ListenAddr:        ":5020",
			TargetAddr:        "192.168.1.100:502",
			ConnectionTimeout: 10,
			ReadTimeout:       30,
			MaxRetries:        3,
			MaxReadSize:       100,
		}

		err := v.ValidateProxyConfig(cfg)
		if err != nil {
			t.Errorf("Valid config failed validation: %v", err)
		}
	})

	t.Run("Empty name", func(t *testing.T) {
		cfg := config.ProxyConfig{
			ID:         "test-id",
			ListenAddr: ":5020",
			TargetAddr: "192.168.1.100:502",
		}

		err := v.ValidateProxyConfig(cfg)
		if err == nil {
			t.Error("Expected error for empty name")
		}
	})

	t.Run("Invalid port", func(t *testing.T) {
		cfg := config.ProxyConfig{
			ID:         "test-id",
			Name:       "Test Proxy",
			ListenAddr: ":99999",
			TargetAddr: "192.168.1.100:502",
		}

		err := v.ValidateProxyConfig(cfg)
		if err == nil {
			t.Error("Expected error for invalid port")
		}
	})

	t.Run("Timeout too high", func(t *testing.T) {
		cfg := config.ProxyConfig{
			ID:                "test-id",
			Name:              "Test Proxy",
			ListenAddr:        ":5020",
			TargetAddr:        "192.168.1.100:502",
			ConnectionTimeout: 500,
			ReadTimeout:       600,
		}

		err := v.ValidateProxyConfig(cfg)
		if err == nil {
			t.Error("Expected error for timeout too high")
		}
	})

	t.Run("Too many retries", func(t *testing.T) {
		cfg := config.ProxyConfig{
			ID:         "test-id",
			Name:       "Test Proxy",
			ListenAddr: ":5020",
			TargetAddr: "192.168.1.100:502",
			MaxRetries: 20,
		}

		err := v.ValidateProxyConfig(cfg)
		if err == nil {
			t.Error("Expected error for too many retries")
		}
	})

	t.Run("Invalid address format", func(t *testing.T) {
		cfg := config.ProxyConfig{
			ID:         "test-id",
			Name:       "Test Proxy",
			ListenAddr: "invalid-address",
			TargetAddr: "192.168.1.100:502",
		}

		err := v.ValidateProxyConfig(cfg)
		if err == nil {
			t.Error("Expected error for invalid address format")
		}
	})
}

func TestValidateAddress(t *testing.T) {
	v := NewValidator()

	t.Run("Valid port only", func(t *testing.T) {
		err := v.validateAddress(":5020")
		if err != nil {
			t.Errorf("Valid port address failed: %v", err)
		}
	})

	t.Run("Valid IP:Port", func(t *testing.T) {
		err := v.validateAddress("192.168.1.100:502")
		if err != nil {
			t.Errorf("Valid IP:Port address failed: %v", err)
		}
	})

	t.Run("Valid Hostname:Port", func(t *testing.T) {
		err := v.validateAddress("localhost:5020")
		if err != nil {
			t.Errorf("Valid Hostname:Port address failed: %v", err)
		}
	})

	t.Run("Empty address", func(t *testing.T) {
		err := v.validateAddress("")
		if err == nil {
			t.Error("Expected error for empty address")
		}
	})

	t.Run("Invalid hostname", func(t *testing.T) {
		err := v.validateAddress("invalid host name:5020")
		if err == nil {
			t.Error("Expected error for invalid hostname")
		}
	})
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, 20)

	t.Run("Allow within limit", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			if !rl.allow("test-ip") {
				t.Errorf("Request %d should be allowed", i)
			}
		}
	})

	t.Run("Exceed limit", func(t *testing.T) {
		allowed := true
		for i := 0; i < 30; i++ {
			if !rl.allow("test-ip2") {
				allowed = false
				break
			}
		}
		if allowed {
			t.Error("Expected rate limit to be exceeded")
		}
	})

	t.Run("Different IPs independent", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			if !rl.allow("ip1") {
				t.Error("IP1 should be allowed")
			}
			if !rl.allow("ip2") {
				t.Error("IP2 should be allowed")
			}
		}
	})
}

func TestCache(t *testing.T) {
	cache := NewCache(1)

	t.Run("Set and Get", func(t *testing.T) {
		cache.Set("key1", "value1")
		val, found := cache.Get("key1")
		if !found {
			t.Error("Value not found in cache")
		}
		if val != "value1" {
			t.Errorf("Expected value1, got %v", val)
		}
	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, found := cache.Get("non-existent")
		if found {
			t.Error("Non-existent key should not be found")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		cache.Set("key2", "value2")
		cache.Delete("key2")
		_, found := cache.Get("key2")
		if found {
			t.Error("Deleted key should not be found")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		cache.Set("key3", "value3")
		cache.Clear()
		if cache.Size() != 0 {
			t.Errorf("Expected cache size 0, got %d", cache.Size())
		}
	})
}
