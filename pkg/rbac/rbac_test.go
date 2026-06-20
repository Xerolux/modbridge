// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package rbac

import "testing"

func TestHasPermission(t *testing.T) {
	cases := []struct {
		role Role
		perm Permission
		want bool
	}{
		{RoleAdmin, PermUserDelete, true},
		{RoleAdmin, PermSystemRestart, true},
		{RoleBenutzer, PermProxyView, true},
		{RoleBenutzer, PermProxyEdit, false},   // viewer-tier cannot edit
		{RoleBenutzer, PermUserDelete, false},  // no user management
		{RoleAuditor, PermAuditExport, true},
		{RoleAuditor, PermProxyControl, false}, // auditor is read-only
		{RoleTechniker, PermProxyDelete, true},
		{Role("unknown"), PermProxyView, false}, // unknown role → deny all
	}
	for _, c := range cases {
		got := HasPermission(c.role, c.perm)
		if got != c.want {
			t.Errorf("HasPermission(%s,%s)=%v want %v", c.role, c.perm, got, c.want)
		}
	}
}

func TestParseRole(t *testing.T) {
	cases := []struct {
		in     string
		want   Role
		hasErr bool
	}{
		{"admin", RoleAdmin, false},
		{"ADMIN", RoleAdmin, false}, // case-insensitive
		{"operator", RoleTechniker, false}, // alias
		{"viewer", RoleBenutzer, false},    // alias
		{"auditor", RoleAuditor, false},
		{"superuser", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, err := ParseRole(c.in)
		if (err != nil) != c.hasErr {
			t.Errorf("ParseRole(%q) err=%v wantErr=%v", c.in, err, c.hasErr)
			continue
		}
		if !c.hasErr && got != c.want {
			t.Errorf("ParseRole(%q)=%s want %s", c.in, got, c.want)
		}
	}
}

func TestGetRolePermissions(t *testing.T) {
	perms := GetRolePermissions(RoleAdmin)
	if len(perms) == 0 {
		t.Fatal("admin must have permissions")
	}
	// Must return an independent copy: mutating it must not affect the
	// package-level definition nor subsequent calls.
	perms[0] = Permission("tampered")
	again := GetRolePermissions(RoleAdmin)
	if again[0] == "tampered" {
		t.Fatal("GetRolePermissions must return a copy, not the shared slice")
	}

	if got := GetRolePermissions(Role("nonsense")); len(got) != 0 {
		t.Errorf("unknown role should yield no permissions, got %d", len(got))
	}
}

func TestRoleJSONRoundtrip(t *testing.T) {
	// Marshal then Unmarshal should preserve a valid role.
	in := RoleTechniker
	data, err := in.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var out Role
	if err := out.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if out != in {
		t.Errorf("roundtrip mismatch: got %s want %s", out, in)
	}

	// Invalid role JSON must error.
	var bad Role
	if err := bad.UnmarshalJSON([]byte(`"godmode"`)); err == nil {
		t.Fatal("expected error unmarshalling invalid role")
	}
}
