package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.AddCommand(validateConfigCmd)
	validateCmd.AddCommand(validateProxyCmd)

	// Config validation flags
	validateConfigCmd.Flags().StringP("file", "f", "config.json", "Config file to validate")
	validateConfigCmd.Flags().Bool("strict", false, "Enable strict validation")

	// Proxy validation flags
	validateProxyCmd.Flags().String("listen", "", "Listen address to validate (required)")
	validateProxyCmd.Flags().String("target", "", "Target address to validate (required)")
	validateProxyCmd.Flags().Duration("timeout", 5*time.Second, "Connection timeout")
	_ = validateProxyCmd.MarkFlagRequired("listen")
	_ = validateProxyCmd.MarkFlagRequired("target")
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configurations",
	Long:  `Validate configuration files and proxy settings before deployment`,
}

var validateConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Validate configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, _ := cmd.Flags().GetString("file")
		strict, _ := cmd.Flags().GetBool("strict")

		fmt.Printf("Validating configuration file: %s\n\n", configFile)

		// Read config file
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("❌ failed to read config file: %w", err)
		}

		// Parse JSON
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("❌ invalid JSON: %w", err)
		}

		fmt.Println("✅ JSON syntax is valid")

		// Validate structure
		errors := 0
		warnings := 0

		// Validate proxies
		if proxies, ok := config["proxies"].([]interface{}); ok {
			fmt.Printf("\nValidating %d proxies...\n", len(proxies))

			usedPorts := make(map[string]bool)

			for i, p := range proxies {
				if proxy, ok := p.(map[string]interface{}); ok {
					proxyID := getStr(proxy, "id")
					if proxyID == "" {
						proxyID = fmt.Sprintf("proxy-%d", i)
					}

					fmt.Printf("\nProxy '%s':\n", proxyID)

					// Validate required fields
					if getStr(proxy, "listen_addr") == "" {
						fmt.Println("  ❌ Missing listen_addr")
						errors++
					} else {
						listenAddr := getStr(proxy, "listen_addr")
						fmt.Printf("  ✅ Listen address: %s\n", listenAddr)

						// Check for port conflicts
						if usedPorts[listenAddr] {
							fmt.Printf("  ⚠️  Port conflict: %s already in use\n", listenAddr)
							warnings++
						}
						usedPorts[listenAddr] = true

						// Validate address format
						if err := validateAddress(listenAddr); err != nil {
							fmt.Printf("  ❌ Invalid listen address: %v\n", err)
							errors++
						}
					}

					if getStr(proxy, "target_addr") == "" {
						fmt.Println("  ❌ Missing target_addr")
						errors++
					} else {
						targetAddr := getStr(proxy, "target_addr")
						fmt.Printf("  ✅ Target address: %s\n", targetAddr)

						// Validate address format
						if err := validateAddress(targetAddr); err != nil {
							fmt.Printf("  ❌ Invalid target address: %v\n", err)
							errors++
						}
					}

					// Validate pool settings
					poolSize := getInt(proxy, "pool_size")
					poolMin := getInt(proxy, "pool_min_size")

					if poolSize > 0 {
						fmt.Printf("  ✅ Connection pool: size=%d, min=%d\n", poolSize, poolMin)

						if poolMin > poolSize {
							fmt.Println("  ❌ pool_min_size cannot be greater than pool_size")
							errors++
						}

						if strict && poolSize > 100 {
							fmt.Printf("  ⚠️  Large pool size (%d) may consume significant resources\n", poolSize)
							warnings++
						}
					}

					// Validate enabled field
					if proxy["enabled"] != nil {
						enabled := getBool(proxy, "enabled")
						fmt.Printf("  ✅ Enabled: %v\n", enabled)
					}
				}
			}
		} else {
			fmt.Println("\n⚠️  No proxies defined")
			warnings++
		}

		// Validate server settings
		if server, ok := config["server"].(map[string]interface{}); ok {
			fmt.Println("\nValidating server settings...")

			if addr := getStr(server, "addr"); addr != "" {
				fmt.Printf("  ✅ API server address: %s\n", addr)
				if err := validateAddress(addr); err != nil {
					fmt.Printf("  ❌ Invalid server address: %v\n", err)
					errors++
				}
			}

			if tlsEnabled := getBool(server, "tls_enabled"); tlsEnabled {
				fmt.Println("  ✅ TLS enabled")

				certFile := getStr(server, "cert_file")
				keyFile := getStr(server, "key_file")

				if certFile == "" {
					fmt.Println("  ❌ TLS enabled but cert_file not specified")
					errors++
				} else if _, err := os.Stat(certFile); os.IsNotExist(err) {
					fmt.Printf("  ❌ Certificate file not found: %s\n", certFile)
					errors++
				} else {
					fmt.Printf("  ✅ Certificate file: %s\n", certFile)
				}

				if keyFile == "" {
					fmt.Println("  ❌ TLS enabled but key_file not specified")
					errors++
				} else if _, err := os.Stat(keyFile); os.IsNotExist(err) {
					fmt.Printf("  ❌ Key file not found: %s\n", keyFile)
					errors++
				} else {
					fmt.Printf("  ✅ Key file: %s\n", keyFile)
				}
			}
		}

		// Validate logging settings
		if logging, ok := config["logging"].(map[string]interface{}); ok {
			fmt.Println("\nValidating logging settings...")

			level := getStr(logging, "level")
			validLevels := map[string]bool{
				"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
			}

			if level == "" {
				fmt.Println("  ⚠️  Log level not specified, will use default")
				warnings++
			} else if !validLevels[strings.ToLower(level)] {
				fmt.Printf("  ❌ Invalid log level: %s\n", level)
				errors++
			} else {
				fmt.Printf("  ✅ Log level: %s\n", level)
			}
		}

		// Summary
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("Validation Summary:")
		fmt.Printf("  Errors:   %d\n", errors)
		fmt.Printf("  Warnings: %d\n", warnings)

		if errors > 0 {
			fmt.Println("\n❌ Configuration validation FAILED")
			return fmt.Errorf("configuration has %d error(s)", errors)
		}

		if warnings > 0 {
			fmt.Println("\n⚠️  Configuration is valid but has warnings")
		} else {
			fmt.Println("\n✅ Configuration is valid")
		}

		return nil
	},
}

var validateProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Validate proxy connection settings",
	Long: `Test if proxy can bind to listen address and connect to target.
This performs actual connection tests without starting a full proxy.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		listenAddr, _ := cmd.Flags().GetString("listen")
		targetAddr, _ := cmd.Flags().GetString("target")
		timeout, _ := cmd.Flags().GetDuration("timeout")

		fmt.Println("Validating proxy configuration...\n")

		// Validate listen address
		fmt.Printf("Testing listen address: %s\n", listenAddr)
		if err := validateAddress(listenAddr); err != nil {
			return fmt.Errorf("❌ invalid listen address: %w", err)
		}

		// Try to bind to listen address
		listener, err := net.Listen("tcp", listenAddr)
		if err != nil {
			return fmt.Errorf("❌ cannot bind to %s: %w", listenAddr, err)
		}
		listener.Close()
		fmt.Printf("  ✅ Can bind to %s\n", listenAddr)

		// Validate target address
		fmt.Printf("\nTesting target address: %s\n", targetAddr)
		if err := validateAddress(targetAddr); err != nil {
			return fmt.Errorf("❌ invalid target address: %w", err)
		}

		// Try to connect to target
		fmt.Printf("  Attempting connection (timeout: %s)...\n", timeout)
		conn, err := net.DialTimeout("tcp", targetAddr, timeout)
		if err != nil {
			return fmt.Errorf("❌ cannot connect to %s: %w", targetAddr, err)
		}
		conn.Close()
		fmt.Printf("  ✅ Successfully connected to %s\n", targetAddr)

		// Check if listen and target are the same
		if listenAddr == targetAddr {
			fmt.Println("\n⚠️  WARNING: listen and target addresses are the same!")
			fmt.Println("   This will create an infinite loop!")
		}

		fmt.Println("\n✅ Proxy configuration is valid")
		return nil
	},
}

// validateAddress validates a TCP address (host:port format).
func validateAddress(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format (expected host:port): %w", err)
	}

	// Validate host
	if host != "" && host != "0.0.0.0" && host != "::" {
		// Try to parse as IP
		if ip := net.ParseIP(host); ip == nil {
			// Not an IP, try to resolve as hostname
			if _, err := net.LookupHost(host); err != nil {
				return fmt.Errorf("cannot resolve hostname '%s': %w", host, err)
			}
		}
	}

	// Validate port
	if port == "" {
		return fmt.Errorf("port not specified")
	}

	return nil
}
