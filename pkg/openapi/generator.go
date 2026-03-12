package openapi

import (
	"fmt"
	"modbridge/pkg/openapi"
	"net/http"
)

// GenerateForModBridge generates OpenAPI spec for ModBridge
func GenerateForModBridge(version string) (string, error) {
	gen := openapi.NewGenerator("ModBridge API", version)

	// Add common schemas
	gen.AddSchema("Proxy", Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"id": {Type: "string", Description: "Unique identifier"},
			"name": {Type: "string", Description: "Proxy name"},
			"listen_addr": {Type: "string", Description: "Listen address"},
			"target_addr": {Type: "string", Description: "Target address"},
			"enabled": {Type: "boolean", Description: "Whether proxy is enabled"},
			"paused": {Type: "boolean", Description: "Whether proxy is paused"},
			"connection_timeout": {Type: "integer", Description: "Connection timeout in seconds"},
			"read_timeout": {Type: "integer", Description: "Read timeout in seconds"},
			"max_retries": {Type: "integer", Description: "Maximum retries"},
		},
		Required: []string{"id", "name", "listen_addr", "target_addr"},
	})

	gen.AddSchema("User", Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"id": {Type: "string"},
			"username": {Type: "string"},
			"email": {Type: "string"},
			"role": {Type: "string", Enum: []interface{}{"admin", "operator", "viewer", "auditor"}},
			"enabled": {Type: "boolean"},
		},
	})

	gen.AddSchema("Device", Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"ip": {Type: "string"},
			"mac": {Type: "string"},
			"name": {Type: "string"},
			"first_seen": {Type: "string", Format: "date-time"},
			"last_connect": {Type: "string", Format: "date-time"},
			"request_count": {Type: "integer"},
			"proxy_id": {Type: "string"},
		},
	})

	// Health endpoint
	gen.AddPath("/api/health", PathItem{
		Get: &Operation{
			Tags:        []string{"Health"},
			Summary:     "Health check endpoint",
			OperationID: "getHealth",
			Responses: map[string]Response{
				"200": {
					Description: "Server is healthy",
					Content: map[string]MediaType{
						"application/json": {Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"status": {Type: "string", Example: "ok"},
							},
						}},
					},
				},
			},
		},
	})

	// Login endpoint
	gen.AddPath("/api/login", PathItem{
		Post: &Operation{
			Tags:        []string{"Authentication"},
			Summary:     "Login to the application",
			OperationID: "login",
			RequestBody: &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{
						Type: "object",
						Properties: map[string]*Schema{
							"password": {Type: "string"},
						},
						Required: []string{"password"},
					}},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Login successful",
					Content: map[string]MediaType{
						"application/json": {Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"token": {Type: "string"},
							},
						}},
					},
				},
				"401": {Description: "Unauthorized"},
			},
		},
	})

	// Proxies endpoints
	gen.AddPath("/api/proxies", PathItem{
		Get: &Operation{
			Tags:        []string{"Proxies"},
			Summary:     "Get all proxies",
			OperationID: "getProxies",
			Security:    []map[string][]string{{"cookieAuth": {}}},
			Responses: map[string]Response{
				"200": {
					Description: "List of proxies",
					Content: map[string]MediaType{
						"application/json": {Schema: &Schema{
							Type: "array",
							Items: &Schema{Ref: "#/components/schemas/Proxy"},
						}},
					},
				},
			},
		},
		Post: &Operation{
			Tags:        []string{"Proxies"},
			Summary:     "Create a new proxy",
			OperationID: "createProxy",
			Security:    []map[string][]string{{"cookieAuth": {}}},
			RequestBody: &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{Ref: "#/components/schemas/Proxy"}},
				},
			},
			Responses: map[string]Response{
				"201": {Description: "Proxy created"},
				"400": {Description: "Invalid request"},
			},
		},
	})

	// Devices endpoint
	gen.AddPath("/api/devices", PathItem{
		Get: &Operation{
			Tags:        []string{"Devices"},
			Summary:     "Get all connected devices",
			OperationID: "getDevices",
			Security:    []map[string][]string{{"cookieAuth": {}}},
			Responses: map[string]Response{
				"200": {
					Description: "List of devices",
					Content: map[string]MediaType{
						"application/json": {Schema: &Schema{
							Type: "array",
							Items: &Schema{Ref: "#/components/schemas/Device"},
						}},
					},
				},
			},
		},
	})

	return gen.Generate()
}

// RegisterOpenAPIHandler registers the OpenAPI spec endpoint
func RegisterOpenAPIHandler(mux *http.ServeMux, path string) {
	spec, _ := GenerateForModBridge("1.0.0")
	
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(spec))
	})
}
