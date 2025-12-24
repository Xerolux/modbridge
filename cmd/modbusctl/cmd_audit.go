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
	rootCmd.AddCommand(auditCmd)
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditSearchCmd)
	auditCmd.AddCommand(auditExportCmd)

	// List flags
	auditListCmd.Flags().String("since", "24h", "Time range (e.g., 1h, 24h, 7d)")
	auditListCmd.Flags().Int("limit", 50, "Maximum number of entries")
	auditListCmd.Flags().String("user", "", "Filter by user")
	auditListCmd.Flags().String("action", "", "Filter by action")

	// Search flags
	auditSearchCmd.Flags().String("query", "", "Search query (required)")
	auditSearchCmd.Flags().String("since", "24h", "Time range")
	auditSearchCmd.Flags().Int("limit", 50, "Maximum number of entries")
	_ = auditSearchCmd.MarkFlagRequired("query")

	// Export flags
	auditExportCmd.Flags().StringP("output", "o", "", "Output file path (required)")
	auditExportCmd.Flags().String("since", "24h", "Time range")
	auditExportCmd.Flags().String("format", "json", "Export format (json, csv)")
	_ = auditExportCmd.MarkFlagRequired("output")
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit logs",
	Long:  `View and search audit logs for security and compliance`,
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent audit log entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		limit, _ := cmd.Flags().GetInt("limit")
		user, _ := cmd.Flags().GetString("user")
		action, _ := cmd.Flags().GetString("action")
		apiURL := viper.GetString("api-url")

		// Build URL with query parameters
		url := fmt.Sprintf("%s/api/audit?since=%s&limit=%d", apiURL, since, limit)
		if user != "" {
			url += "&user=" + user
		}
		if action != "" {
			url += "&action=" + action
		}

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to get audit logs: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get audit logs: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var logs []map[string]interface{}
		if err := json.Unmarshal(body, &logs); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(logs) == 0 {
			fmt.Println("No audit logs found")
			return nil
		}

		// Print as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TIMESTAMP\tUSER\tACTION\tRESOURCE\tSTATUS\tIP ADDRESS")
		fmt.Fprintln(w, strings.Repeat("─", 100))

		for _, log := range logs {
			timestamp := getStr(log, "timestamp")
			if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
				timestamp = t.Format("2006-01-02 15:04:05")
			}

			username := getStr(log, "user")
			actionStr := getStr(log, "action")
			resource := getStr(log, "resource")
			status := getStr(log, "status")
			ipAddr := getStr(log, "ip_address")

			// Truncate long resource names
			if len(resource) > 30 {
				resource = resource[:27] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				timestamp, username, actionStr, resource, status, ipAddr)
		}

		w.Flush()
		fmt.Printf("\nShowing %d entries from the last %s\n", len(logs), since)
		return nil
	},
}

var auditSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search audit logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		since, _ := cmd.Flags().GetString("since")
		limit, _ := cmd.Flags().GetInt("limit")
		apiURL := viper.GetString("api-url")

		url := fmt.Sprintf("%s/api/audit/search?q=%s&since=%s&limit=%d",
			apiURL, query, since, limit)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to search audit logs: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to search audit logs: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var logs []map[string]interface{}
		if err := json.Unmarshal(body, &logs); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(logs) == 0 {
			fmt.Printf("No audit logs found matching '%s'\n", query)
			return nil
		}

		// Print as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TIMESTAMP\tUSER\tACTION\tRESOURCE\tDETAILS")
		fmt.Fprintln(w, strings.Repeat("─", 100))

		for _, log := range logs {
			timestamp := getStr(log, "timestamp")
			if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
				timestamp = t.Format("2006-01-02 15:04:05")
			}

			username := getStr(log, "user")
			action := getStr(log, "action")
			resource := getStr(log, "resource")
			details := getStr(log, "details")

			// Truncate long details
			if len(details) > 40 {
				details = details[:37] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				timestamp, username, action, resource, details)
		}

		w.Flush()
		fmt.Printf("\nFound %d matching entries\n", len(logs))
		return nil
	},
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit logs to file",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile, _ := cmd.Flags().GetString("output")
		since, _ := cmd.Flags().GetString("since")
		format, _ := cmd.Flags().GetString("format")
		apiURL := viper.GetString("api-url")

		url := fmt.Sprintf("%s/api/audit/export?since=%s&format=%s",
			apiURL, since, format)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to export audit logs: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to export audit logs: %s", string(body))
		}

		// Create output file
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		// Copy response to file
		written, err := io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Printf("✅ Audit logs exported to '%s' (%s, %s format)\n",
			outputFile, formatBytes(written), format)
		return nil
	},
}
