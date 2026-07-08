package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	_ "expvar"
	"fmt"
	"log"
	"modbridge/pkg/api"
	"modbridge/pkg/auth"
	"modbridge/pkg/config"
	"modbridge/pkg/database"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
	"modbridge/pkg/users"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	// Version holds the current version of the application
	Version = "dev"
	// BuildTime holds the timestamp when the application was built
	BuildTime = "unknown"
)

// generateSecurePassword generates a cryptographically secure random password.
// It produces exactly `length` URL-safe base64 characters without truncating entropy:
// (length+3)/4*3 bytes → base64 → at least `length` characters, no wasted randomness.
func generateSecurePassword(length int) string {
	// Calculate the minimum bytes needed so base64 output is >= length chars.
	// base64: 3 bytes → 4 chars, so ceil(length/4)*3 bytes.
	byteCount := (length + 3) / 4 * 3
	b := make([]byte, byteCount)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random password: " + err.Error())
	}
	encoded := base64.RawURLEncoding.EncodeToString(b) // no padding
	return encoded[:length]
}

// bootstrapUsers ensures a usable admin account exists in multi-user mode.
//   - Fresh install (no users, no prior AdminPassHash): creates admin/admin with
//     MustChangePassword=true (change forced on first login).
//   - Migration (no users, but a prior single-user AdminPassHash): creates the
//     admin from the EXISTING bcrypt hash so the operator's old password stays
//     valid; MustChangePassword mirrors the prior ForcePasswordChange flag.
//   - Already populated: no-op.
//
// Safe to call when multi-user is disabled (it still seeds the store if a DB is
// present so the operator can switch to multi-user later without losing state).
func bootstrapUsers(userMgr *users.Manager, cfg config.Config, l *logger.Logger) {
	if userMgr == nil {
		return
	}
	existing, err := userMgr.GetAllUsers()
	if err != nil {
		l.Error("SYSTEM", fmt.Sprintf("bootstrap: cannot read users: %v", err))
		return
	}
	if len(existing) > 0 {
		return // already populated
	}

	if strings.TrimSpace(cfg.AdminPassHash) == "" {
		// Fresh install: default admin/admin.
		if _, err := userMgr.EnsureDefaultAdmin("admin", "admin", "system"); err != nil {
			l.Error("SYSTEM", fmt.Sprintf("bootstrap: failed to create default admin: %v", err))
			return
		}
		l.Info("SYSTEM", "Default-Login erstellt — Benutzername: admin / Passwort: admin — BITTE beim ersten Login ändern.")
		log.Println("SYSTEM: Default admin created (username: admin, password: admin). Change it on first login.")
		return
	}

	// Migration: reuse the existing single-user password hash so the operator
	// does not have to reconfigure their password.
	if _, err := userMgr.EnsureDefaultAdminFromHash("admin", cfg.AdminPassHash, "system"); err != nil {
		l.Error("SYSTEM", fmt.Sprintf("bootstrap: migration failed: %v", err))
		return
	}
	if cfg.ForcePasswordChange {
		// Carry over the prior "must change" requirement.
		if err := userMgr.SetMustChangePasswordByUsername("admin", true); err != nil {
			l.Error("SYSTEM", fmt.Sprintf("bootstrap: could not set MustChangePassword: %v", err))
		}
	}
	l.Info("SYSTEM", "Bestehendes Admin-Passwort migriert (Benutzername: admin).")
	log.Println("SYSTEM: Existing admin password migrated (username: admin).")
}

func main() {
	// 1. Database
	db, err := database.NewDB("modbridge.db")
	if err != nil {
		log.Printf("Warning: Failed to init database: %v. Database features will be disabled.", err)
		db = nil
	} else {
		defer db.Close()
		log.Println("Database initialized successfully")
	}

	// 2. Config
	cfgMgr := config.NewManager("config.json")
	if err := cfgMgr.Load(); err != nil {
		log.Printf("Starting with empty config: %v", err)
	}

	// NOTE: The legacy single-user password bootstrap (random password written
	// to AdminPassHash) has been removed. In the new default multi-user mode the
	// admin account is seeded in the user store by bootstrapUsers() below. The
	// single-user fallback path in handleLogin still honours AdminPassHash if a
	// pre-existing config.json carries one.

	// 3. Logger
	l, err := logger.NewLogger("proxy.log", 1000)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer l.Close()

	// 4. Manager
	mgr := manager.NewManager(cfgMgr, l, db)
	mgr.Initialize()

	// 4. Auth
	authenticator := auth.NewAuthenticator()
	authCtx, authCancel := context.WithCancel(context.Background())
	defer authCancel()
	go authenticator.CleanupExpiredSessions(authCtx)

	// 5. API Server
	apiServer := api.NewServer(cfgMgr, mgr, authenticator, l, db)

	// 6. Auto-deactivate expired users periodically + bootstrap multi-user store
	if db != nil {
		userMgr := users.NewManager(db)

		// Bootstrap the initial admin account. Multi-user is the default mode:
		//   - Fresh install -> admin/admin (MustChangePassword forced)
		//   - Migration from single-user -> existing password hash reused
		//   - Already populated -> no-op
		bootstrapUsers(userMgr, cfgMgr.Get(), l)

		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-authCtx.Done():
					return
				case <-ticker.C:
					count, err := userMgr.DeactivateExpiredUsers()
					if err != nil {
						l.Error("SYSTEM", fmt.Sprintf("Failed to deactivate expired users: %v", err))
					} else if count > 0 {
						l.Info("SYSTEM", fmt.Sprintf("Auto-deactivated %d expired user(s)", count))
					}
				}
			}
		}()
	}

	// 6. Router
	mux := http.NewServeMux()

	// API Routes
	apiServer.Routes(mux)

	// Web Routes (only if not headless build)
	if handler := getWebHandler(); handler != nil {
		mux.Handle("/", handler)
	}

	// Start server
	addr := cfgMgr.Get().WebPort
	if addr == "" {
		addr = ":8080"
	}
	// Override with environment variable if set
	if envPort := os.Getenv("WEB_PORT"); envPort != "" {
		addr = envPort
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB
	}

	// Apply TLS configuration when enabled.
	tlsEnabled := cfgMgr.Get().TLSEnabled
	tlsCertFile := cfgMgr.Get().TLSCertFile
	tlsKeyFile := cfgMgr.Get().TLSKeyFile
	if tlsEnabled {
		server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			},
		}
	}

	l.Info("SYSTEM", "Starting Modbus Manager on "+addr)
	log.Printf("Listening on %s", addr)

	// Run server in goroutine
	go func() {
		var err error
		if tlsEnabled {
			err = server.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown — wait for either OS signal or API restart request.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
	case <-apiServer.RestartSignal():
	}

	l.Info("SYSTEM", "Shutting down server...")
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop all proxies
	mgr.StopAll()

	// Stop API background goroutines
	apiServer.Stop()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("Server close error: %v", closeErr)
		}
	}

	l.Info("SYSTEM", "Server stopped")
	log.Println("Server stopped")
}
