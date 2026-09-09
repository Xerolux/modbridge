// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package openapi

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// handleFuncPattern matches the route patterns registered in pkg/api.
var handleFuncPattern = regexp.MustCompile(`mux\.HandleFunc\("([^"]+)"`)

// registeredRoutes reads the routes pkg/api actually registers. The spec is
// hand-maintained, so this is what keeps it from drifting: net/http's ServeMux
// does not expose its patterns, and parsing the registration calls is the
// cheapest reliable source.
func registeredRoutes(t *testing.T) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("..", "api", "*.go"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}

	seen := map[string]struct{}{}
	for _, file := range files {
		if filepath.Base(file) == "server_test.go" {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("ReadFile %s: %v", file, err)
		}
		for _, m := range handleFuncPattern.FindAllStringSubmatch(string(src), -1) {
			seen[m[1]] = struct{}{}
		}
	}

	if len(seen) == 0 {
		t.Fatal("no routes found in pkg/api; the extraction pattern is stale")
	}

	routes := make([]string, 0, len(seen))
	for r := range seen {
		routes = append(routes, r)
	}
	sort.Strings(routes)
	return routes
}

func TestSpecCoversEveryRegisteredRoute(t *testing.T) {
	spec := Generate()

	for _, route := range registeredRoutes(t) {
		if _, ok := spec.Paths[route]; !ok {
			t.Errorf("route %s is registered in pkg/api but missing from the OpenAPI spec", route)
		}
	}
}

func TestSpecHasNoUnknownRoutes(t *testing.T) {
	registered := map[string]struct{}{}
	for _, route := range registeredRoutes(t) {
		registered[route] = struct{}{}
	}

	for path := range Generate().Paths {
		if _, ok := registered[path]; !ok {
			t.Errorf("OpenAPI spec documents %s, which pkg/api does not register", path)
		}
	}
}

func TestSpecOperationsAreWellFormed(t *testing.T) {
	for path, item := range Generate().Paths {
		if len(item) == 0 {
			t.Errorf("%s has no operations", path)
		}
		for method, op := range item {
			switch method {
			case "get", "post", "put", "delete", "patch":
			default:
				t.Errorf("%s: unexpected method %q", path, method)
			}
			if op.Summary == "" {
				t.Errorf("%s %s: missing summary", method, path)
			}
			if op.OperationID == "" {
				t.Errorf("%s %s: missing operationId", method, path)
			}
			if len(op.Responses) == 0 {
				t.Errorf("%s %s: no responses documented", method, path)
			}
		}
	}
}

func TestOperationIDsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for path, item := range Generate().Paths {
		for method, op := range item {
			if where, dup := seen[op.OperationID]; dup {
				t.Errorf("operationId %q used by both %s and %s %s", op.OperationID, where, method, path)
			}
			seen[op.OperationID] = method + " " + path
		}
	}
}
