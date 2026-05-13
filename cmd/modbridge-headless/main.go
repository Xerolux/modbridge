package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"modbridge/pkg/config"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	version = "headless-dev"
)

func main() {
	configFile := flag.String("config", "config.json", "Path to configuration file")
	verbose := flag.Bool("v", false, "Verbose logging (debug level)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("modbridge-headless %s\n", version)
		return
	}

	log.Printf("ModBridge Headless v%s", version)
	log.Printf("Config: %s", *configFile)

	cfgMgr := config.NewManager(*configFile)
	if err := cfgMgr.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfg := cfgMgr.Get()
	if len(cfg.Proxies) == 0 {
		log.Fatal("No proxies defined in config. Exiting.")
	}

	logLevel := "INFO"
	if *verbose {
		logLevel = "DEBUG"
	}
	_ = logLevel

	l, err := logger.NewLogger("proxy.log", 1000)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer l.Close()

	mgr := manager.NewManager(cfgMgr, l, nil)

	log.Printf("Starting %d proxy(ies) from config...", len(cfg.Proxies))
	l.Info("HEADLESS", fmt.Sprintf("Initializing %d proxies from %s", len(cfg.Proxies), *configFile))

	mgr.Initialize()

	proxies := mgr.GetProxies()
	running := 0
	for _, p := range proxies {
		if status, ok := p["status"].(string); ok && status == "Running" {
			running++
		}
	}
	log.Printf("%d/%d proxies running", running, len(proxies))
	l.Info("HEADLESS", fmt.Sprintf("%d/%d proxies running", running, len(proxies)))

	ready := make(chan os.Signal, 1)
	signal.Notify(ready, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ready

	log.Printf("Received %s, shutting down...", sig)
	l.Info("HEADLESS", fmt.Sprintf("Received signal %s, shutting down...", sig))

	mgr.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = ctx

	log.Println("Stopped.")
	l.Info("HEADLESS", "All proxies stopped. Goodbye.")
}
