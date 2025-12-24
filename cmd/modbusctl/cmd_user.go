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
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userGetCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userUpdateCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userPasswordCmd)

	// Create flags
	userCreateCmd.Flags().String("username", "", "Username (required)")
	userCreateCmd.Flags().String("password", "", "Password (required)")
	userCreateCmd.Flags().String("email", "", "Email address")
	userCreateCmd.Flags().String("role", "operator", "User role (admin, operator, viewer)")
	_ = userCreateCmd.MarkFlagRequired("username")
	_ = userCreateCmd.MarkFlagRequired("password")

	// Update flags
	userUpdateCmd.Flags().String("email", "", "New email address")
	userUpdateCmd.Flags().String("role", "", "New role (admin, operator, viewer)")
	userUpdateCmd.Flags().Bool("enabled", true, "Enable/disable user")

	// Password flags
	userPasswordCmd.Flags().String("password", "", "New password (required)")
	_ = userPasswordCmd.MarkFlagRequired("password")
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users (admin only)",
	Long:  `Create, update, and manage user accounts. Requires admin privileges.`,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api-url")
		resp, err := http.Get(apiURL + "/api/users")
		if err != nil {
			return fmt.Errorf("failed to get users: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get users: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var users []map[string]interface{}
		if err := json.Unmarshal(body, &users); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Print as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "USERNAME\tEMAIL\tROLE\tSTATUS\tCREATED\tLAST LOGIN")
		fmt.Fprintln(w, strings.Repeat("─", 90))

		for _, user := range users {
			username := getStr(user, "username")
			email := getStr(user, "email")
			role := getStr(user, "role")
			enabled := getBool(user, "enabled")

			status := "enabled"
			if !enabled {
				status = "disabled"
			}

			created := getStr(user, "created_at")
			lastLogin := getStr(user, "last_login")
			if lastLogin == "" {
				lastLogin = "never"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				username, email, role, status, created, lastLogin)
		}

		w.Flush()
		return nil
	},
}

var userGetCmd = &cobra.Command{
	Use:   "get <username>",
	Short: "Get user details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		apiURL := viper.GetString("api-url")

		resp, err := http.Get(fmt.Sprintf("%s/api/users/%s", apiURL, username))
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get user: %s", string(body))
		}

		if outputFmt == "json" {
			fmt.Println(string(body))
			return nil
		}

		var user map[string]interface{}
		if err := json.Unmarshal(body, &user); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Print details
		fmt.Printf("User: %s\n", getStr(user, "username"))
		fmt.Printf("  Email:        %s\n", getStr(user, "email"))
		fmt.Printf("  Role:         %s\n", getStr(user, "role"))
		fmt.Printf("  Status:       %s\n", func() string {
			if getBool(user, "enabled") {
				return "enabled"
			}
			return "disabled"
		}())
		fmt.Printf("  Created:      %s\n", getStr(user, "created_at"))
		fmt.Printf("  Last Login:   %s\n", getStr(user, "last_login"))

		if perms := getMap(user, "permissions"); len(perms) > 0 {
			fmt.Println("\nPermissions:")
			for key, value := range perms {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}

		return nil
	},
}

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		email, _ := cmd.Flags().GetString("email")
		role, _ := cmd.Flags().GetString("role")
		apiURL := viper.GetString("api-url")

		payload := map[string]interface{}{
			"username": username,
			"password": password,
			"role":     role,
			"enabled":  true,
		}

		if email != "" {
			payload["email"] = email
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		resp, err := http.Post(apiURL+"/api/users", "application/json",
			strings.NewReader(string(data)))
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to create user: %s", string(body))
		}

		fmt.Printf("✅ User '%s' created successfully with role '%s'\n", username, role)
		return nil
	},
}

var userUpdateCmd = &cobra.Command{
	Use:   "update <username>",
	Short: "Update user details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		apiURL := viper.GetString("api-url")

		payload := make(map[string]interface{})

		if cmd.Flags().Changed("email") {
			email, _ := cmd.Flags().GetString("email")
			payload["email"] = email
		}

		if cmd.Flags().Changed("role") {
			role, _ := cmd.Flags().GetString("role")
			payload["role"] = role
		}

		if cmd.Flags().Changed("enabled") {
			enabled, _ := cmd.Flags().GetBool("enabled")
			payload["enabled"] = enabled
		}

		if len(payload) == 0 {
			return fmt.Errorf("no fields to update specified")
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		req, err := http.NewRequest("PATCH",
			fmt.Sprintf("%s/api/users/%s", apiURL, username),
			strings.NewReader(string(data)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to update user: %s", string(body))
		}

		fmt.Printf("✅ User '%s' updated successfully\n", username)
		return nil
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <username>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		apiURL := viper.GetString("api-url")

		req, err := http.NewRequest("DELETE",
			fmt.Sprintf("%s/api/users/%s", apiURL, username), nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to delete user: %s", string(body))
		}

		fmt.Printf("✅ User '%s' deleted successfully\n", username)
		return nil
	},
}

var userPasswordCmd = &cobra.Command{
	Use:   "password <username>",
	Short: "Change user password",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		password, _ := cmd.Flags().GetString("password")
		apiURL := viper.GetString("api-url")

		payload := map[string]interface{}{
			"password": password,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/api/users/%s/password", apiURL, username),
			"application/json",
			strings.NewReader(string(data)))
		if err != nil {
			return fmt.Errorf("failed to change password: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to change password: %s", string(body))
		}

		fmt.Printf("✅ Password changed successfully for user '%s'\n", username)
		return nil
	},
}
