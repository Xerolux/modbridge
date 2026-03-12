package openapi

import (
	"encoding/json"
)

// Spec represents the OpenAPI specification
type Spec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Servers    []Server               `json:"servers"`
	Paths      map[string]PathItem    `json:"paths"`
	Components Components             `json:"components"`
}

// Info contains API information
type Info struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Version        string `json:"version"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
}

// Contact contains contact information
type Contact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// License contains license information
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// PathItem represents a path item
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation represents an operation
type Operation struct {
	Tags        []string            `json:"tags"`
	Summary     string              `json:"summary"`
	Description string              `json:"description,omitempty"`
	OperationID string              `json:"operationId"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

// Parameter represents a parameter
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody represents a request body
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required"`
	Content     map[string]MediaType `json:"content"`
}

// MediaType represents a media type
type MediaType struct {
	Schema *Schema `json:"schema"`
}

// Response represents a response
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// Schema represents a schema
type Schema struct {
	Type                 string               `json:"type,omitempty"`
	Format               string               `json:"format,omitempty"`
	Description          string               `json:"description,omitempty"`
	Properties           map[string]*Schema   `json:"properties,omitempty"`
	Required             []string             `json:"required,omitempty"`
	Items                *Schema              `json:"items,omitempty"`
	Ref                  string               `json:"$ref,omitempty"`
	Enum                 []interface{}        `json:"enum,omitempty"`
	AdditionalProperties *Schema              `json:"additionalProperties,omitempty"`
}

// Components contains reusable components
type Components struct {
	Schemas         map[string]Schema    `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme represents a security scheme
type SecurityScheme struct {
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

// Generator generates OpenAPI specification
type Generator struct {
	spec *Spec
}

// NewGenerator creates a new OpenAPI generator
func NewGenerator(title, version string) *Generator {
	spec := &Spec{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       title,
			Description: "ModBridge API - Modbus TCP Proxy Manager",
			Version:     version,
			Contact: &Contact{
				Name:  "Xerolux",
				Email: "support@example.com",
			},
			License: &License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []Server{
			{
				URL:         "http://localhost:8080",
				Description: "Development server",
			},
		},
		Paths:      make(map[string]PathItem),
		Components: Components{
			Schemas: make(map[string]Schema),
			SecuritySchemes: map[string]SecurityScheme{
				"cookieAuth": {
					Type:        "apiKey",
					In:          "cookie",
					Name:        "session_token",
					Description: "Session cookie for authentication",
				},
			},
		},
	}

	return &Generator{spec: spec}
}

// AddPath adds a path to the specification
func (g *Generator) AddPath(path string, item PathItem) {
	g.spec.Paths[path] = item
}

// AddSchema adds a schema to the components
func (g *Generator) AddSchema(name string, schema Schema) {
	g.spec.Components.Schemas[name] = schema
}

// Generate generates the OpenAPI specification as JSON
func (g *Generator) Generate() (string, error) {
	data, err := json.MarshalIndent(g.spec, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateYAML generates the OpenAPI specification as YAML (simplified)
func (g *Generator) GenerateYAML() (string, error) {
	// For simplicity, just return JSON for now
	return g.Generate()
}
