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
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupExportCmd)
	backupCmd.AddCommand(backupImportCmd)

	// Create flags
	backupCreateCmd.Flags().String("name", "", "Backup name/description")

	// Restore flags
	backupRestoreCmd.Flags().Bool("confirm", false, "Confirm restore operation")
	_ = backupRestoreCmd.MarkFlagRequired("confirm")

	// Export flags
	backupExportCmd.Flags().StringP("output", "o", "", "Output file path (required)")
	_ = backupExportCmd.MarkFlagRequired("output")

	// Import flags
	backupImportCmd.Flags().StringP("file", "f", "", "Backup file to import (required)")
	_ = backupImportCmd.MarkFlagRequired("file")
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage configuration backups",
	Long:  `Create, restore, and manage configuration backups`,
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backups",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api-url")
		resp, err := http.Get(apiURL + "/api/backup")
		if err != nil {
			return fmt.Errorf("failed to get backups: %w", err)
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

		var backups []map[string]interface{}
		if err := json.Unmarshal(body, &backups); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(backups) == 0 {
			fmt.Println("No backups found")
			return nil
		}

		// Print as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tCREATED\tSIZE\tTYPE\tSTATUS")
		fmt.Fprintln(w, strings.Repeat("─", 80))

		for _, backup := range backups {
			id := getStr(backup, "id")
			name := getStr(backup, "name")
			created := getStr(backup, "created_at")
			if t, err := time.Parse(time.RFC3339, created); err == nil {
				created = t.Format("2006-01-02 15:04:05")
			}

			size := getInt(backup, "size_bytes")
			sizeStr := formatBytes(int64(size))
			backupType := getStr(backup, "type")
			status := getStr(backup, "status")

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				id, name, created, sizeStr, backupType, status)
		}

		w.Flush()
		return nil
	},
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		apiURL := viper.GetString("api-url")

		payload := map[string]interface{}{}
		if name != "" {
			payload["name"] = name
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		resp, err := http.Post(apiURL+"/api/backup", "application/json",
			strings.NewReader(string(data)))
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to create backup: %s", string(body))
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			backupID := getStr(result, "id")
			fmt.Printf("✅ Backup created successfully (ID: %s)\n", backupID)
		} else {
			fmt.Println("✅ Backup created successfully")
		}

		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-id>",
	Short: "Restore from a backup",
	Long: `Restore configuration from a backup. This will overwrite current configuration.
Use --confirm flag to proceed with the restore operation.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupID := args[0]
		confirm, _ := cmd.Flags().GetBool("confirm")

		if !confirm {
			return fmt.Errorf("restore requires --confirm flag to proceed")
		}

		apiURL := viper.GetString("api-url")

		resp, err := http.Post(
			fmt.Sprintf("%s/api/backup/%s/restore", apiURL, backupID),
			"application/json", nil)
		if err != nil {
			return fmt.Errorf("failed to restore backup: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to restore backup: %s", string(body))
		}

		fmt.Printf("✅ Configuration restored from backup '%s'\n", backupID)
		fmt.Println("⚠️  Please restart the service for changes to take effect")
		return nil
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete <backup-id>",
	Short: "Delete a backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupID := args[0]
		apiURL := viper.GetString("api-url")

		req, err := http.NewRequest("DELETE",
			fmt.Sprintf("%s/api/backup/%s", apiURL, backupID), nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to delete backup: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to delete backup: %s", string(body))
		}

		fmt.Printf("✅ Backup '%s' deleted successfully\n", backupID)
		return nil
	},
}

var backupExportCmd = &cobra.Command{
	Use:   "export <backup-id>",
	Short: "Export a backup to file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupID := args[0]
		outputFile, _ := cmd.Flags().GetString("output")
		apiURL := viper.GetString("api-url")

		resp, err := http.Get(fmt.Sprintf("%s/api/backup/%s/download", apiURL, backupID))
		if err != nil {
			return fmt.Errorf("failed to download backup: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to download backup: %s", string(body))
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
			return fmt.Errorf("failed to write backup file: %w", err)
		}

		fmt.Printf("✅ Backup exported to '%s' (%s)\n", outputFile, formatBytes(written))
		return nil
	},
}

var backupImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a backup from file",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		apiURL := viper.GetString("api-url")

		// Open backup file
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open backup file: %w", err)
		}
		defer file.Close()

		// Get file info for size
		fileInfo, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}

		resp, err := http.Post(apiURL+"/api/backup/import", "application/octet-stream", file)
		if err != nil {
			return fmt.Errorf("failed to import backup: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to import backup: %s", string(body))
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			backupID := getStr(result, "id")
			fmt.Printf("✅ Backup imported successfully (ID: %s, Size: %s)\n",
				backupID, formatBytes(fileInfo.Size()))
		} else {
			fmt.Printf("✅ Backup imported successfully (Size: %s)\n",
				formatBytes(fileInfo.Size()))
		}

		return nil
	},
}

// formatBytes converts bytes to human-readable format.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
