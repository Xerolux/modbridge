package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	apiURL    string
	verbose   bool
	outputFmt string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "modbusctl",
	Short: "Modbridge CLI - Manage Modbus proxies from the command line",
	Long: `modbusctl is a command-line interface for managing Modbridge,
the enterprise-grade Modbus TCP proxy.

Features:
  • Manage proxies (list, create, update, delete, start, stop)
  • Monitor devices and connections
  • View metrics and statistics
  • Manage users and permissions
  • Backup and restore configurations
  • View audit logs
  • Validate configurations
`,
	Version: "1.0.0",
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.modbusctl.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "http://localhost:8080", "Modbridge API URL")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "output format (table, json, yaml)")

	// Bind flags to viper
	_ = viper.BindPFlag("api-url", rootCmd.PersistentFlags().Lookup("api-url"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".modbusctl")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
