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
	rootCmd.AddCommand(metricsCmd)
	metricsCmd.AddCommand(metricsOverviewCmd)
	metricsCmd.AddCommand(metricsProxyCmd)
	metricsCmd.AddCommand(metricsSystemCmd)
	metricsCmd.AddCommand(metricsHealthCmd)

	// Proxy metrics flags
	metricsProxyCmd.Flags().String("proxy", "", "Proxy ID (optional)")
}

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "View metrics and statistics",
	Long:  `View system metrics, proxy statistics, and performance data`,
}

var metricsOverviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Show overall system metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api-url")
		resp, err := http.Get(apiURL + "/api/metrics")
		if err != nil {
			return fmt.Errorf("failed to get metrics: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get metrics: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var metrics map[string]interface{}
		if err := json.Unmarshal(body, &metrics); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Println("=== System Overview ===")
		fmt.Printf("Uptime:              %s\n", getStr(metrics, "uptime"))
		fmt.Printf("Total Requests:      %d\n", getInt(metrics, "total_requests"))
		fmt.Printf("Total Errors:        %d\n", getInt(metrics, "total_errors"))
		fmt.Printf("Error Rate:          %.2f%%\n", getFloat(metrics, "error_rate")*100)
		fmt.Printf("Active Connections:  %d\n", getInt(metrics, "active_connections"))
		fmt.Printf("Active Proxies:      %d\n", getInt(metrics, "active_proxies"))

		if perf := getMap(metrics, "performance"); len(perf) > 0 {
			fmt.Println("\n=== Performance ===")
			fmt.Printf("Avg Latency:         %.2f ms\n", getFloat(perf, "avg_latency_ms"))
			fmt.Printf("P50 Latency:         %.2f ms\n", getFloat(perf, "p50_latency_ms"))
			fmt.Printf("P95 Latency:         %.2f ms\n", getFloat(perf, "p95_latency_ms"))
			fmt.Printf("P99 Latency:         %.2f ms\n", getFloat(perf, "p99_latency_ms"))
			fmt.Printf("Requests/sec:        %.2f\n", getFloat(perf, "requests_per_sec"))
		}

		if resources := getMap(metrics, "resources"); len(resources) > 0 {
			fmt.Println("\n=== Resource Usage ===")
			fmt.Printf("CPU Usage:           %.1f%%\n", getFloat(resources, "cpu_percent"))
			fmt.Printf("Memory Usage:        %s / %s (%.1f%%)\n",
				formatBytes(int64(getInt(resources, "memory_used_bytes"))),
				formatBytes(int64(getInt(resources, "memory_total_bytes"))),
				getFloat(resources, "memory_percent"))
			fmt.Printf("Goroutines:          %d\n", getInt(resources, "goroutines"))
		}

		return nil
	},
}

var metricsProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Show proxy-specific metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyID, _ := cmd.Flags().GetString("proxy")
		apiURL := viper.GetString("api-url")

		var url string
		if proxyID != "" {
			url = fmt.Sprintf("%s/api/proxies/%s/metrics", apiURL, proxyID)
		} else {
			url = apiURL + "/api/metrics/proxies"
		}

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to get proxy metrics: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get proxy metrics: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		if proxyID != "" {
			// Single proxy metrics
			var metrics map[string]interface{}
			if err := json.Unmarshal(body, &metrics); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("=== Proxy: %s ===\n", getStr(metrics, "name"))
			fmt.Printf("Status:              %s\n", getStr(metrics, "status"))
			fmt.Printf("Uptime:              %s\n", getStr(metrics, "uptime"))
			fmt.Printf("Total Requests:      %d\n", getInt(metrics, "total_requests"))
			fmt.Printf("Total Errors:        %d\n", getInt(metrics, "total_errors"))
			fmt.Printf("Error Rate:          %.2f%%\n", getFloat(metrics, "error_rate")*100)
			fmt.Printf("Active Connections:  %d\n", getInt(metrics, "active_connections"))
			fmt.Printf("Avg Latency:         %.2f ms\n", getFloat(metrics, "avg_latency_ms"))
			fmt.Printf("Requests/sec:        %.2f\n", getFloat(metrics, "requests_per_sec"))
		} else {
			// All proxies metrics
			var proxies []map[string]interface{}
			if err := json.Unmarshal(body, &proxies); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROXY ID\tNAME\tSTATUS\tREQUESTS\tERRORS\tERROR%\tLATENCY\tREQ/S")
			fmt.Fprintln(w, strings.Repeat("─", 100))

			for _, proxy := range proxies {
				id := getStr(proxy, "id")
				name := getStr(proxy, "name")
				status := getStr(proxy, "status")
				requests := getInt(proxy, "total_requests")
				errors := getInt(proxy, "total_errors")
				errorRate := getFloat(proxy, "error_rate") * 100
				latency := getFloat(proxy, "avg_latency_ms")
				reqPerSec := getFloat(proxy, "requests_per_sec")

				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%.1f%%\t%.1f ms\t%.1f\n",
					id, name, status, requests, errors, errorRate, latency, reqPerSec)
			}

			w.Flush()
		}

		return nil
	},
}

var metricsSystemCmd = &cobra.Command{
	Use:   "system",
	Short: "Show system resource metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api-url")
		resp, err := http.Get(apiURL + "/api/metrics/system")
		if err != nil {
			return fmt.Errorf("failed to get system metrics: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get system metrics: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var metrics map[string]interface{}
		if err := json.Unmarshal(body, &metrics); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Println("=== System Resources ===")
		fmt.Printf("Hostname:            %s\n", getStr(metrics, "hostname"))
		fmt.Printf("OS:                  %s\n", getStr(metrics, "os"))
		fmt.Printf("Architecture:        %s\n", getStr(metrics, "arch"))
		fmt.Printf("Go Version:          %s\n", getStr(metrics, "go_version"))
		fmt.Printf("Uptime:              %s\n", getStr(metrics, "uptime"))

		fmt.Println("\n=== CPU ===")
		fmt.Printf("CPU Cores:           %d\n", getInt(metrics, "cpu_cores"))
		fmt.Printf("CPU Usage:           %.1f%%\n", getFloat(metrics, "cpu_percent"))

		fmt.Println("\n=== Memory ===")
		fmt.Printf("Total Memory:        %s\n", formatBytes(int64(getInt(metrics, "memory_total"))))
		fmt.Printf("Used Memory:         %s\n", formatBytes(int64(getInt(metrics, "memory_used"))))
		fmt.Printf("Free Memory:         %s\n", formatBytes(int64(getInt(metrics, "memory_free"))))
		fmt.Printf("Memory Usage:        %.1f%%\n", getFloat(metrics, "memory_percent"))

		fmt.Println("\n=== Go Runtime ===")
		fmt.Printf("Goroutines:          %d\n", getInt(metrics, "goroutines"))
		fmt.Printf("Heap Alloc:          %s\n", formatBytes(int64(getInt(metrics, "heap_alloc"))))
		fmt.Printf("Heap Objects:        %d\n", getInt(metrics, "heap_objects"))
		fmt.Printf("GC Runs:             %d\n", getInt(metrics, "gc_runs"))
		fmt.Printf("Last GC:             %s\n", getStr(metrics, "last_gc"))

		return nil
	},
}

var metricsHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show health check status",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api-url")
		resp, err := http.Get(apiURL + "/health")
		if err != nil {
			return fmt.Errorf("failed to get health status: %w", err)
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

		var health map[string]interface{}
		if err := json.Unmarshal(body, &health); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		status := getStr(health, "status")
		statusIcon := "✅"
		if status != "healthy" {
			statusIcon = "⚠️"
		}

		fmt.Printf("%s System Status: %s\n", statusIcon, strings.ToUpper(status))
		fmt.Printf("Version:             %s\n", getStr(health, "version"))
		fmt.Printf("Uptime:              %s\n", getStr(health, "uptime"))

		if checks := getMap(health, "checks"); len(checks) > 0 {
			fmt.Println("\n=== Component Health ===")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "COMPONENT\tSTATUS\tMESSAGE")
			fmt.Fprintln(w, strings.Repeat("─", 60))

			for component, checkData := range checks {
				if checkMap, ok := checkData.(map[string]interface{}); ok {
					checkStatus := getStr(checkMap, "status")
					message := getStr(checkMap, "message")

					icon := "✅"
					if checkStatus != "healthy" {
						icon = "❌"
					}

					fmt.Fprintf(w, "%s %s\t%s\t%s\n",
						icon, component, checkStatus, message)
				}
			}

			w.Flush()
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
		}

		return nil
	},
}
