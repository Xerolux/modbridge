// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"modbridge/pkg/api"
	"modbridge/pkg/auth"
	"modbridge/pkg/config"
	"modbridge/pkg/database"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
		if err := serverCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse server command: %v\n", err)
			os.Exit(1)
		}
		runServer(*configFile, *port)
	case "version":
		if err := versionCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse version command: %v\n", err)
			os.Exit(1)
		}
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

	if configFile == "" {
		configFile = "config.json"
	}
	fmt.Printf("Using config: %s\n", configFile)

	// Initialize database
	db, err := database.NewDB("modbridge.db")
	if err != nil {
		log.Printf("Warning: Failed to init database: %v. Database features will be disabled.", err)
		db = nil
	} else {
		defer db.Close()
	}

	// Initialize config
	cfgMgr := config.NewManager(configFile)
	if err := cfgMgr.Load(); err != nil {
		log.Printf("Starting with empty config: %v", err)
	}

	// Initialize logger
	l, err := logger.NewLogger("logs", 1000)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer l.Close()

	// Initialize proxy manager
	mgr := manager.NewManager(cfgMgr, l, db)
	mgr.Initialize()

	// Initialize authentication
	authenticator := auth.NewAuthenticator()
	authCtx, authCancel := context.WithCancel(context.Background())
	defer authCancel()
	go authenticator.CleanupExpiredSessions(authCtx)

	// Initialize API server
	apiServer := api.NewServer(cfgMgr, mgr, authenticator, l, db)

	// Setup HTTP router
	mux := http.NewServeMux()
	apiServer.Routes(mux)

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("Listening on %s", addr)
	l.Info("SYSTEM", "Starting Modbus Manager on "+addr)

	// Run server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	l.Info("SYSTEM", "Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mgr.StopAll()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
	l.Info("SYSTEM", "Server stopped")
}
