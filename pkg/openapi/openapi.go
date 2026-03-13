package openapi

import (
	"encoding/json"
)

// Spec represents an OpenAPI 3.0 specification
type Spec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Servers    []Server               `json:"servers"`
	Paths      map[string]PathItem    `json:"paths"`
	Components Components             `json:"components"`
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
	Summary     string               `json:"summary"`
	Description string               `json:"description"`
	OperationID string               `json:"operationId"`
	Responses   map[string]Response  `json:"responses"`
}

// Response describes an API response
type Response struct {
	Description string                 `json:"description"`
	Content     map[string]MediaType  `json:"content"`
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
	Type       string                    `json:"type"`
	Properties map[string]Schema         `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

// Generate generates an OpenAPI spec for ModBridge
func Generate() *Spec {
	return &Spec{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "ModBridge API",
			Description: "Modbus TCP Proxy Manager API",
			Version:     "1.0.0",
		},
		Servers: []Server{
			{URL: "http://localhost:8080"},
		},
		Paths: map[string]PathItem{
			"/api/v1/devices": {
				"get": Operation{
					Summary:     "List all devices",
					OperationID: "listDevices",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
		Components: Components{
			Schemas: map[string]Schema{
				"Device": {
					Type: "object",
					Properties: map[string]Schema{
						"id":   {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
		},
	}
}

// ToJSON converts the spec to JSON
func (s *Spec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}
