package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(deviceCmd)
	deviceCmd.AddCommand(deviceListCmd)
	deviceCmd.AddCommand(deviceGetCmd)
	deviceCmd.AddCommand(deviceRenameCmd)
	deviceCmd.AddCommand(deviceHistoryCmd)

	// Rename flags
	deviceRenameCmd.Flags().String("name", "", "New device name (required)")
	_ = deviceRenameCmd.MarkFlagRequired("name")

	// History flags
	deviceHistoryCmd.Flags().String("since", "24h", "Time range (e.g., 1h, 24h, 7d)")
	deviceHistoryCmd.Flags().Int("limit", 100, "Maximum number of entries")
}

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Manage Modbus devices",
	Long:  `List, view, and manage Modbus devices connected through proxies`,
}

var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all discovered devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api-url")
		resp, err := http.Get(apiURL + "/api/devices")
		if err != nil {
			return fmt.Errorf("failed to get devices: %w", err)
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

		var devices []map[string]interface{}
		if err := json.Unmarshal(body, &devices); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Print as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "DEVICE ID\tNAME\tADDRESS\tPROXY\tSTATUS\tLAST SEEN\tREQUESTS")
		fmt.Fprintln(w, strings.Repeat("─", 90))

		for _, device := range devices {
			deviceID := getStr(device, "device_id")
			name := getStr(device, "name")
			if name == "" {
				name = deviceID
			}
			address := getStr(device, "address")
			proxyID := getStr(device, "proxy_id")
			status := getStr(device, "status")
			lastSeen := getStr(device, "last_seen")

			stats := getMap(device, "stats")
			requests := getInt(stats, "requests")

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
				deviceID, name, address, proxyID, status, lastSeen, requests)
		}

		w.Flush()
		return nil
	},
}

var deviceGetCmd = &cobra.Command{
	Use:   "get <device-id>",
	Short: "Get device details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceID := args[0]
		apiURL := viper.GetString("api-url")

		resp, err := http.Get(fmt.Sprintf("%s/api/devices/%s", apiURL, deviceID))
		if err != nil {
			return fmt.Errorf("failed to get device: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get device: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var device map[string]interface{}
		if err := json.Unmarshal(body, &device); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Print details
		fmt.Printf("Device: %s\n", getStr(device, "device_id"))
		fmt.Printf("  Name:         %s\n", getStr(device, "name"))
		fmt.Printf("  Address:      %s\n", getStr(device, "address"))
		fmt.Printf("  Proxy ID:     %s\n", getStr(device, "proxy_id"))
		fmt.Printf("  Status:       %s\n", getStr(device, "status"))
		fmt.Printf("  First Seen:   %s\n", getStr(device, "first_seen"))
		fmt.Printf("  Last Seen:    %s\n", getStr(device, "last_seen"))

		stats := getMap(device, "stats")
		if len(stats) > 0 {
			fmt.Println("\nStatistics:")
			fmt.Printf("  Requests:     %d\n", getInt(stats, "requests"))
			fmt.Printf("  Errors:       %d\n", getInt(stats, "errors"))
			fmt.Printf("  Avg Latency:  %.2f ms\n", getFloat(stats, "avg_latency_ms"))
		}

		return nil
	},
}

var deviceRenameCmd = &cobra.Command{
	Use:   "rename <device-id>",
	Short: "Rename a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceID := args[0]
		newName, _ := cmd.Flags().GetString("name")
		apiURL := viper.GetString("api-url")

		payload := map[string]interface{}{
			"name": newName,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		req, err := http.NewRequest("PATCH",
			fmt.Sprintf("%s/api/devices/%s", apiURL, deviceID),
			strings.NewReader(string(data)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to rename device: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to rename device: %s", string(body))
		}

		fmt.Printf("✅ Device '%s' renamed to '%s'\n", deviceID, newName)
		return nil
	},
}

var deviceHistoryCmd = &cobra.Command{
	Use:   "history <device-id>",
	Short: "View device request history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceID := args[0]
		since, _ := cmd.Flags().GetString("since")
		limit, _ := cmd.Flags().GetInt("limit")
		apiURL := viper.GetString("api-url")

		url := fmt.Sprintf("%s/api/devices/%s/history?since=%s&limit=%d",
			apiURL, deviceID, since, limit)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to get history: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get history: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var history []map[string]interface{}
		if err := json.Unmarshal(body, &history); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Print as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TIMESTAMP\tFUNCTION\tADDRESS\tCOUNT\tSTATUS\tLATENCY")
		fmt.Fprintln(w, strings.Repeat("─", 80))

		for _, entry := range history {
			timestamp := getStr(entry, "timestamp")
			if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
				timestamp = t.Format("2006-01-02 15:04:05")
			}

			functionCode := getInt(entry, "function_code")
			address := getInt(entry, "address")
			count := getInt(entry, "count")
			status := getStr(entry, "status")
			latency := getFloat(entry, "latency_ms")

			fmt.Fprintf(w, "%s\t0x%02X\t%d\t%d\t%s\t%.2f ms\n",
				timestamp, functionCode, address, count, status, latency)
		}

		w.Flush()
		fmt.Printf("\nShowing %d entries from the last %s\n", len(history), since)
		return nil
	},
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.0
}
