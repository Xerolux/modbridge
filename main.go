package main

import (
	"log"
	"modbusproxy/pkg/api"
	"modbusproxy/pkg/auth"
	"modbusproxy/pkg/config"
	"modbusproxy/pkg/logger"
	"modbusproxy/pkg/manager"
	"modbusproxy/pkg/web"
	"net/http"
)

func main() {
	// 1. Config
	cfgMgr := config.NewManager("config.json")
	if err := cfgMgr.Load(); err != nil {
		log.Printf("Starting with empty config: %v", err)
	}

	// 2. Logger
	l, err := logger.NewLogger("proxy.log", 1000)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}

	// 3. Manager
	mgr := manager.NewManager(cfgMgr, l)
	mgr.Initialize()

	// 4. Auth
	authenticator := auth.NewAuthenticator()

	// 5. API Server
	apiServer := api.NewServer(cfgMgr, mgr, authenticator, l)

	// 6. Router
	mux := http.NewServeMux()
	
	// API Routes
	apiServer.Routes(mux)

	// Web Routes
	mux.Handle("/", web.Handler())

	// Start
	addr := cfgMgr.Get().WebPort
	if addr == "" {
		addr = ":8080"
	}
	
	l.Info("SYSTEM", "Starting Modbus Manager on "+addr)
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
