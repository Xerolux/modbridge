// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package openapi

import (
	"encoding/json"
)

// Spec represents an OpenAPI 3.0 specification
type Spec struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Servers    []Server            `json:"servers"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components"`
}

// Info provides metadata about the API
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// Server represents an API server
type Server struct {
	URL string `json:"url"`
}

// PathItem describes the operations available on a single path
type PathItem map[string]Operation

// Operation describes a single API operation
type Operation struct {
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	OperationID string              `json:"operationId"`
	Responses   map[string]Response `json:"responses"`
}

// Response describes an API response
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content"`
}

// MediaType represents a media type
type MediaType struct {
	Schema map[string]interface{} `json:"schema"`
}

// Components holds reusable objects
type Components struct {
	Schemas map[string]Schema `json:"schemas"`
}

// Schema describes a data schema
type Schema struct {
	Type       string            `json:"type"`
	Properties map[string]Schema `json:"properties,omitempty"`
	Required   []string          `json:"required,omitempty"`
}

// Version is the version reported in the generated spec. It is overridden at
// build time with -ldflags "-X modbridge/pkg/openapi.Version=...".
var Version = "dev"

// Generate generates an OpenAPI spec for ModBridge.
//
// The path set mirrors the routes registered in pkg/api. A test in this
// package compares the two, so a route added there without a matching entry
// here fails the build rather than silently leaving the spec wrong.
func Generate() *Spec {
	return &Spec{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "ModBridge API",
			Description: "Modbus TCP Proxy Manager API. All endpoints except the health, readiness, metrics, login, setup and recovery routes require an authenticated session; state-changing requests additionally require a CSRF token.",
			Version:     Version,
		},
		Servers: []Server{
			{URL: "http://localhost:8080"},
		},
		Paths: map[string]PathItem{
			"/api/health": {
				"get": {
					Summary:     "Liveness probe",
					Description: "Returns 200 while the process is running.",
					OperationID: "getHealth",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
			"/api/ready": {
				"get": {
					Summary:     "Readiness probe",
					Description: "Reports whether the proxies and their dependencies are ready to serve.",
					OperationID: "getReady",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
			"/api/status": {
				"get": {
					Summary:     "Session status",
					Description: "Returns the proxy inventory for the current session.",
					OperationID: "getStatus",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/login": {
				"post": {
					Summary:     "Log in",
					Description: "Exchanges credentials for a session cookie. Rate limited.",
					OperationID: "postLogin",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
			"/api/logout": {
				"post": {
					Summary:     "Log out",
					Description: "Invalidates the current session.",
					OperationID: "postLogout",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/setup": {
				"post": {
					Summary:     "Initial setup",
					Description: "Creates the first administrator on a fresh install.",
					OperationID: "postSetup",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
			"/api/account-recovery": {
				"get": {
					Summary:     "Account recovery",
					Description: "Completes the recovery flow started with --enable-account-recovery.",
					OperationID: "getAccountRecovery",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
				"post": {
					Summary:     "Account recovery",
					Description: "Completes the recovery flow started with --enable-account-recovery.",
					OperationID: "postAccountRecovery",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
			"/api/me": {
				"get": {
					Summary:     "Current user",
					Description: "Returns the authenticated user and its permissions.",
					OperationID: "getMe",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/users": {
				"get": {
					Summary:     "List or create users",
					Description: "",
					OperationID: "getUsers",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
				"post": {
					Summary:     "List or create users",
					Description: "",
					OperationID: "postUsers",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/users/": {
				"get": {
					Summary:     "Read, update or delete a user",
					Description: "The user ID is the trailing path segment.",
					OperationID: "getUserByID",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
				"put": {
					Summary:     "Read, update or delete a user",
					Description: "The user ID is the trailing path segment.",
					OperationID: "putUserByID",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
				"delete": {
					Summary:     "Read, update or delete a user",
					Description: "The user ID is the trailing path segment.",
					OperationID: "deleteUserByID",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/proxies": {
				"get": {
					Summary:     "List, create, update or delete proxies",
					Description: "",
					OperationID: "getProxies",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
				"post": {
					Summary:     "List, create, update or delete proxies",
					Description: "",
					OperationID: "postProxies",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
				"put": {
					Summary:     "List, create, update or delete proxies",
					Description: "",
					OperationID: "putProxies",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
				"delete": {
					Summary:     "List, create, update or delete proxies",
					Description: "",
					OperationID: "deleteProxies",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/proxies/stream": {
				"get": {
					Summary:     "Proxy event stream",
					Description: "Server-sent events carrying live proxy state.",
					OperationID: "getProxiesStream",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/proxies/control": {
				"post": {
					Summary:     "Control a proxy",
					Description: "Starts, stops, pauses or restarts a proxy.",
					OperationID: "postProxiesControl",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/proxies/calibrate": {
				"post": {
					Summary:     "Calibrate a proxy",
					Description: "Measures the target device and proposes timing settings.",
					OperationID: "postProxiesCalibrate",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/devices": {
				"get": {
					Summary:     "List or rename tracked devices",
					Description: "",
					OperationID: "getDevices",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
				"put": {
					Summary:     "List or rename tracked devices",
					Description: "",
					OperationID: "putDevices",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/devices/history": {
				"get": {
					Summary:     "Device connection history",
					Description: "",
					OperationID: "getDevicesHistory",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/logs": {
				"get": {
					Summary:     "Read log entries",
					Description: "",
					OperationID: "getLogs",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/logs/download": {
				"get": {
					Summary:     "Download log files",
					Description: "",
					OperationID: "getLogsDownload",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/logs/stream": {
				"get": {
					Summary:     "Log event stream",
					Description: "Server-sent events carrying live log entries.",
					OperationID: "getLogsStream",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/audit/logs": {
				"get": {
					Summary:     "Read audit log entries",
					Description: "",
					OperationID: "getAuditLogs",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/audit/logs/export": {
				"get": {
					Summary:     "Export the audit log",
					Description: "Returns CSV or JSON.",
					OperationID: "getAuditLogsExport",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/config/export": {
				"get": {
					Summary:     "Export the configuration",
					Description: "",
					OperationID: "getConfigExport",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/config/import": {
				"post": {
					Summary:     "Import a configuration",
					Description: "The imported configuration is validated before it is applied.",
					OperationID: "postConfigImport",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/config/rollback": {
				"post": {
					Summary:     "Roll back to the previous configuration",
					Description: "",
					OperationID: "postConfigRollback",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/config/system": {
				"get": {
					Summary:     "Read or update system settings",
					Description: "",
					OperationID: "getConfigSystem",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
				"put": {
					Summary:     "Read or update system settings",
					Description: "",
					OperationID: "putConfigSystem",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/config/webport": {
				"get": {
					Summary:     "Read or change the web listen address",
					Description: "",
					OperationID: "getConfigWebport",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
				"put": {
					Summary:     "Read or change the web listen address",
					Description: "",
					OperationID: "putConfigWebport",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/config/password": {
				"post": {
					Summary:     "Change the current user's password",
					Description: "",
					OperationID: "postConfigPassword",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/metrics": {
				"get": {
					Summary:     "Prometheus metrics",
					Description: "Text exposition format.",
					OperationID: "getMetrics",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
			"/api/system/info": {
				"get": {
					Summary:     "System information",
					Description: "Version, build time, runtime and uptime.",
					OperationID: "getSystemInfo",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/system/restart": {
				"post": {
					Summary:     "Restart the service",
					Description: "",
					OperationID: "postSystemRestart",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/system/ports/check": {
				"get": {
					Summary:     "Check the configured proxy ports",
					Description: "",
					OperationID: "getSystemPortsCheck",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/system/ports/diagnostics": {
				"post": {
					Summary:     "Diagnose a port",
					Description: "Reports what is listening on a port.",
					OperationID: "postSystemPortsDiagnostics",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/system/ports/release": {
				"post": {
					Summary:     "Release a port",
					Description: "Terminates the process holding a port.",
					OperationID: "postSystemPortsRelease",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/system/diagnostics/connectivity": {
				"get": {
					Summary:     "Check target connectivity",
					Description: "Dials every configured target.",
					OperationID: "getSystemDiagnosticsConnectivity",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/update/check": {
				"get": {
					Summary:     "Check for a new release",
					Description: "",
					OperationID: "getUpdateCheck",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/update/perform": {
				"post": {
					Summary:     "Install the available update",
					Description: "",
					OperationID: "postUpdatePerform",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
			"/api/update/status": {
				"get": {
					Summary:     "Report update progress",
					Description: "",
					OperationID: "getUpdateStatus",
					Responses: map[string]Response{
						"200": {Description: "Success"},
						"401": {Description: "Not authenticated"},
						"403": {Description: "Not permitted"},
					},
				},
			},
		},
		Components: Components{
			Schemas: map[string]Schema{
				"Proxy": {
					Type: "object",
					Properties: map[string]Schema{
						"id":          {Type: "string"},
						"name":        {Type: "string"},
						"listen_addr": {Type: "string"},
						"target_addr": {Type: "string"},
						"enabled":     {Type: "boolean"},
					},
					Required: []string{"id", "listen_addr", "target_addr"},
				},
				"Device": {
					Type: "object",
					Properties: map[string]Schema{
						"ip":   {Type: "string"},
						"mac":  {Type: "string"},
						"name": {Type: "string"},
					},
					Required: []string{"ip"},
				},
				"Error": {
					Type: "object",
					Properties: map[string]Schema{
						"error": {Type: "string"},
					},
					Required: []string{"error"},
				},
			},
		},
	}
}

// ToJSON converts the spec to JSON
func (s *Spec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}
