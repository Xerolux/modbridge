package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	version = "1.0.0"
)

func main() {
	// Define subcommands
	serverCmd := flag.NewFlagSet("server", flag.ExitOnError)
	configFile := serverCmd.String("config", "", "Configuration file")
	port := serverCmd.Int("port", 8080, "Server port")

	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		serverCmd.Parse(os.Args[2:])
		runServer(*configFile, *port)
	case "version":
		versionCmd.Parse(os.Args[2:])
		fmt.Printf("ModBridge CLI v%s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("ModBridge CLI - Modbus TCP Proxy Manager")
	fmt.Println("\nUsage:")
	fmt.Println("  cli server [options]    Start the ModBridge server")
	fmt.Println("  cli version             Show version information")
	fmt.Println("\nServer Options:")
	fmt.Println("  -config string")
	fmt.Println("        Configuration file path")
	fmt.Println("  -port int")
	fmt.Println("        Server port (default 8080)")
}

func runServer(configFile string, port int) {
	fmt.Printf("Starting ModBridge server on port %d\n", port)
	if configFile != "" {
		fmt.Printf("Using config: %s\n", configFile)
	}
	// TODO: Implement actual server startup
}
