// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package rbac

import (
	"encoding/json"
	"errors"
	"strings"
)

// Role represents a user role with specific permissions
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleTechniker Role = "techniker"
	RoleBenutzer  Role = "benutzer"
	RoleAuditor   Role = "auditor"
)

// Permission represents a specific action
type Permission string

const (
	// Proxy permissions
	PermProxyView    Permission = "proxy:view"
	PermProxyCreate  Permission = "proxy:create"
	PermProxyEdit    Permission = "proxy:edit"
	PermProxyDelete  Permission = "proxy:delete"
	PermProxyControl Permission = "proxy:control"

	// Device permissions
	PermDeviceView   Permission = "device:view"
	PermDeviceEdit   Permission = "device:edit"
	PermDeviceDelete Permission = "device:delete"

	// Config permissions
	PermConfigView   Permission = "config:view"
	PermConfigEdit   Permission = "config:edit"
	PermConfigExport Permission = "config:export"
	PermConfigImport Permission = "config:import"

	// System permissions
	PermSystemView    Permission = "system:view"
	PermSystemManage  Permission = "system:manage"
	PermSystemRestart Permission = "system:restart"

	// User management permissions
	PermUserView   Permission = "user:view"
	PermUserCreate Permission = "user:create"
	PermUserEdit   Permission = "user:edit"
	PermUserDelete Permission = "user:delete"

	// Audit log permissions
	PermAuditView   Permission = "audit:view"
	PermAuditExport Permission = "audit:export"

	// Logs permissions
	PermLogsView   Permission = "logs:view"
	PermLogsExport Permission = "logs:export"
)

// RolePermissions defines the permissions for each role
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermProxyView, PermProxyCreate, PermProxyEdit, PermProxyDelete, PermProxyControl,
		PermDeviceView, PermDeviceEdit, PermDeviceDelete,
		PermConfigView, PermConfigEdit, PermConfigExport, PermConfigImport,
		PermSystemView, PermSystemManage, PermSystemRestart,
		PermUserView, PermUserCreate, PermUserEdit, PermUserDelete,
		PermAuditView, PermAuditExport,
		PermLogsView, PermLogsExport,
	},
	RoleTechniker: {
		PermProxyView, PermProxyCreate, PermProxyEdit, PermProxyDelete, PermProxyControl,
		PermDeviceView, PermDeviceEdit,
		PermConfigView,
		PermSystemView,
		PermLogsView,
	},
	RoleBenutzer: {
		PermProxyView, PermProxyControl,
		PermDeviceView,
		PermConfigView,
		PermSystemView,
		PermLogsView,
	},
	RoleAuditor: {
		PermProxyView,
		PermDeviceView,
		PermConfigView,
		PermSystemView,
		PermAuditView, PermAuditExport,
		PermLogsView, PermLogsExport,
	},
}

var rolePermissionSets = func() map[Role]map[Permission]struct{} {
	sets := make(map[Role]map[Permission]struct{}, len(RolePermissions))
	for role, permissions := range RolePermissions {
		set := make(map[Permission]struct{}, len(permissions))
		for _, permission := range permissions {
			set[permission] = struct{}{}
		}
		sets[role] = set
	}
	return sets
}()

// HasPermission checks if a role has a specific permission
func HasPermission(role Role, permission Permission) bool {
	perms, ok := rolePermissionSets[role]
	if !ok {
		return false
	}
	_, ok = perms[permission]
	return ok
}

// GetRolePermissions returns a fresh copy of all permissions granted to a role.
// A copy (rather than the package-level slice) is returned so callers cannot
// mutate the shared role definition.
func GetRolePermissions(role Role) []Permission {
	if perms, ok := RolePermissions[role]; ok {
		result := make([]Permission, len(perms))
		copy(result, perms)
		return result
	}
	return []Permission{}
}

// ParseRole parses a role from string
func ParseRole(roleStr string) (Role, error) {
	role := Role(strings.ToLower(roleStr))
	switch role {
	case RoleAdmin, RoleTechniker, RoleBenutzer, RoleAuditor:
		return role, nil
	case "operator":
		return RoleTechniker, nil
	case "viewer":
		return RoleBenutzer, nil
	default:
		return "", errors.New("invalid role")
	}
}

// MarshalJSON implements custom JSON marshaling for Role
func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

// UnmarshalJSON implements custom JSON unmarshaling for Role
func (r *Role) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	role, err := ParseRole(s)
	if err != nil {
		return err
	}
	*r = role
	return nil
}
