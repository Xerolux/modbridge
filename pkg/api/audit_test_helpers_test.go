// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package api

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"modbridge/pkg/auth"
	"modbridge/pkg/config"
	"modbridge/pkg/database"
	"modbridge/pkg/logger"
	"modbridge/pkg/manager"
	"modbridge/pkg/users"
)

// dbCounter ensures each test gets a unique temp DB file, avoiding collisions
// when the full suite runs in parallel. We use a file (not :memory:) because
// Go's database/sql pool may hand a handler a different connection than the
// one that ran initExtendedSchema — and an un-shared :memory: DB is per-
// connection, so the handler would see "no such table".
var dbCounter int64

// auditedTestServer builds a Server backed by a unique temp SQLite DB file so
// that auditor.GetLogs() works and userMgr.CreateUser succeeds (the existing
// proxyTestServer passes db=nil, leaving both auditor and userMgr nil).
// Returns the server and a cleanup func that closes the auditor + DB + log
// and removes the temp file.
func auditedTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	log, err := logger.NewLogger("test.log", 100)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	n := atomic.AddInt64(&dbCounter, 1)
	dbPath := filepath.Join(t.TempDir(), fmt.Sprintf("audit-test-%d.db", n))
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	cfgMgr := config.NewManager("test.json")
	mgr := manager.NewManager(cfgMgr, log, db)
	authenticator := auth.NewAuthenticator()
	server := NewServer(cfgMgr, mgr, authenticator, log, db, "test", "unknown")

	cleanup := func() {
		if server.auditor != nil {
			server.auditor.Close()
		}
		db.Close()
		log.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}
	return server, cleanup
}

// sessionFor creates a DB-backed user with the given role and returns a valid
// session token for that user. Used to drive RBAC tests with non-admin roles.
func sessionFor(t *testing.T, server *Server, role, username string) string {
	t.Helper()
	if server.userMgr == nil {
		t.Fatal("server.userMgr is nil; use auditedTestServer not proxyTestServer")
	}
	user, err := server.userMgr.CreateUser(&users.CreateUserRequest{
		Username: username,
		FullName: username,
		Email:    username + "@example.com",
		Password: "TestPass123!",
		Role:     role,
		Enabled:  true,
	}, "test")
	if err != nil {
		t.Fatalf("CreateUser(%s, %s): %v", username, role, err)
	}
	token, err := server.auth.CreateSession(user.ID, user.Username, user.Role, 24*time.Hour, false)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return token
}
