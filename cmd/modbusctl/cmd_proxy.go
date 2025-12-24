package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(proxyCmd)
	proxyCmd.AddCommand(proxyListCmd)
	proxyCmd.AddCommand(proxyGetCmd)
	proxyCmd.AddCommand(proxyCreateCmd)
	proxyCmd.AddCommand(proxyUpdateCmd)
	proxyCmd.AddCommand(proxyDeleteCmd)
	proxyCmd.AddCommand(proxyStartCmd)
	proxyCmd.AddCommand(proxyStopCmd)
	proxyCmd.AddCommand(proxyRestartCmd)

	// Create flags
	proxyCreateCmd.Flags().String("name", "", "Proxy name (required)")
	proxyCreateCmd.Flags().String("listen", "", "Listen address (required)")
	proxyCreateCmd.Flags().String("target", "", "Target address (required)")
	proxyCreateCmd.Flags().Int("pool-size", 10, "Connection pool size")
	proxyCreateCmd.Flags().Int("pool-min", 2, "Minimum pool size")
	_ = proxyCreateCmd.MarkFlagRequired("name")
	_ = proxyCreateCmd.MarkFlagRequired("listen")
	_ = proxyCreateCmd.MarkFlagRequired("target")
}

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage Modbus proxies",
	Long:  `Create, list, update, delete, start, and stop Modbus proxies`,
}

var proxyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all proxies",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api-url")
		resp, err := http.Get(apiURL + "/api/proxies")
		if err != nil {
			return fmt.Errorf("failed to get proxies: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var proxies []map[string]interface{}
		if err := json.Unmarshal(body, &proxies); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Print as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tLISTEN\tTARGET\tSTATUS\tREQUESTS\tERRORS")
		fmt.Fprintln(w, strings.Repeat("─", 80))

		for _, proxy := range proxies {
			id := getStr(proxy, "id")
			name := getStr(proxy, "name")
			listen := getStr(proxy, "listen_addr")
			target := getStr(proxy, "target_addr")
			status := getStr(proxy, "status")

			stats := getMap(proxy, "stats")
			requests := getInt(stats, "requests")
			errors := getInt(stats, "errors")

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
				id, name, listen, target, status, requests, errors)
		}

		w.Flush()
		return nil
	},
}

var proxyGetCmd = &cobra.Command{
	Use:   "get <proxy-id>",
	Short: "Get proxy details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyID := args[0]
		apiURL := viper.GetString("api-url")

		resp, err := http.Get(fmt.Sprintf("%s/api/proxies/%s", apiURL, proxyID))
		if err != nil {
			return fmt.Errorf("failed to get proxy: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get proxy: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var proxy map[string]interface{}
		if err := json.Unmarshal(body, &proxy); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Print details
		fmt.Printf("Proxy: %s\n", getStr(proxy, "name"))
		fmt.Printf("  ID:           %s\n", getStr(proxy, "id"))
		fmt.Printf("  Listen Addr:  %s\n", getStr(proxy, "listen_addr"))
		fmt.Printf("  Target Addr:  %s\n", getStr(proxy, "target_addr"))
		fmt.Printf("  Status:       %s\n", getStr(proxy, "status"))
		fmt.Printf("  Enabled:      %v\n", getBool(proxy, "enabled"))

		stats := getMap(proxy, "stats")
		if len(stats) > 0 {
			fmt.Println("\nStatistics:")
			fmt.Printf("  Requests:     %d\n", getInt(stats, "requests"))
			fmt.Printf("  Errors:       %d\n", getInt(stats, "errors"))
			fmt.Printf("  Uptime:       %d seconds\n", getInt(stats, "uptime"))
		}

		return nil
	},
}

var proxyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		listen, _ := cmd.Flags().GetString("listen")
		target, _ := cmd.Flags().GetString("target")
		poolSize, _ := cmd.Flags().GetInt("pool-size")
		poolMin, _ := cmd.Flags().GetInt("pool-min")

		// Generate ID from name
		id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

		config := map[string]interface{}{
			"id":            id,
			"name":          name,
			"listen_addr":   listen,
			"target_addr":   target,
			"enabled":       true,
			"pool_size":     poolSize,
			"pool_min_size": poolMin,
		}

		data, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		apiURL := viper.GetString("api-url")
		resp, err := http.Post(apiURL+"/api/proxies", "application/json", strings.NewReader(string(data)))
		if err != nil {
			return fmt.Errorf("failed to create proxy: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to create proxy: %s", string(body))
		}

		fmt.Printf("✅ Proxy '%s' created successfully\n", name)
		return nil
	},
}

var proxyUpdateCmd = &cobra.Command{
	Use:   "update <proxy-id>",
	Short: "Update a proxy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Proxy update not yet implemented")
		return nil
	},
}

var proxyDeleteCmd = &cobra.Command{
	Use:   "delete <proxy-id>",
	Short: "Delete a proxy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyID := args[0]
		apiURL := viper.GetString("api-url")

		req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/proxies/%s", apiURL, proxyID), nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to delete proxy: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to delete proxy: %s", string(body))
		}

		fmt.Printf("✅ Proxy '%s' deleted successfully\n", proxyID)
		return nil
	},
}

var proxyStartCmd = &cobra.Command{
	Use:   "start <proxy-id>",
	Short: "Start a proxy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyID := args[0]
		apiURL := viper.GetString("api-url")

		resp, err := http.Post(fmt.Sprintf("%s/api/proxies/%s/start", apiURL, proxyID), "application/json", nil)
		if err != nil {
			return fmt.Errorf("failed to start proxy: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to start proxy: %s", string(body))
		}

		fmt.Printf("✅ Proxy '%s' started successfully\n", proxyID)
		return nil
	},
}

var proxyStopCmd = &cobra.Command{
	Use:   "stop <proxy-id>",
	Short: "Stop a proxy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyID := args[0]
		apiURL := viper.GetString("api-url")

		resp, err := http.Post(fmt.Sprintf("%s/api/proxies/%s/stop", apiURL, proxyID), "application/json", nil)
		if err != nil {
			return fmt.Errorf("failed to stop proxy: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to stop proxy: %s", string(body))
		}

		fmt.Printf("✅ Proxy '%s' stopped successfully\n", proxyID)
		return nil
	},
}

var proxyRestartCmd = &cobra.Command{
	Use:   "restart <proxy-id>",
	Short: "Restart a proxy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyID := args[0]

		// Stop
		if err := proxyStopCmd.RunE(cmd, args); err != nil {
			return err
		}

		// Start
		if err := proxyStartCmd.RunE(cmd, args); err != nil {
			return err
		}

		fmt.Printf("✅ Proxy '%s' restarted successfully\n", proxyID)
		return nil
	},
}

// Helper functions
func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		if i, ok := v.(float64); ok {
			return int(i)
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return make(map[string]interface{})
}
