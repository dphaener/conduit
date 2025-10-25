package docs

import (
	"testing"
)

func TestOpenAPIGenerator_CreateParameters(t *testing.T) {
	gen := &OpenAPIGenerator{}

	params := []*ParameterDoc{
		{
			Name:        "id",
			In:          "path",
			Type:        "string",
			Required:    true,
			Description: "Resource ID",
			Example:     "123",
		},
		{
			Name:        "page",
			In:          "query",
			Type:        "integer",
			Required:    false,
			Description: "Page number",
			Example:     1,
		},
	}

	result := gen.createParameters(params)

	if len(result) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(result))
	}

	// Check first parameter
	if result[0]["name"] != "id" {
		t.Errorf("First parameter name = %v, want 'id'", result[0]["name"])
	}

	if result[0]["in"] != "path" {
		t.Errorf("First parameter in = %v, want 'path'", result[0]["in"])
	}

	if result[0]["required"] != true {
		t.Errorf("First parameter required = %v, want true", result[0]["required"])
	}

	schema := result[0]["schema"].(map[string]interface{})
	if schema["type"] != "string" {
		t.Errorf("First parameter type = %v, want 'string'", schema["type"])
	}

	// Check second parameter with example
	if result[1]["example"] != 1 {
		t.Errorf("Second parameter example = %v, want 1", result[1]["example"])
	}
}

func TestOpenAPIGenerator_CreateRequestBody(t *testing.T) {
	gen := &OpenAPIGenerator{}

	body := &RequestBodyDoc{
		Description: "User data",
		Required:    true,
		ContentType: "application/json",
		Schema: &SchemaDoc{
			Type: "object",
			Properties: map[string]*PropertyDoc{
				"name": {
					Type:        "string",
					Description: "User name",
				},
			},
			Required: []string{"name"},
		},
		Example: map[string]interface{}{
			"name": "John Doe",
		},
	}

	result := gen.createRequestBody(body)

	if result["description"] != "User data" {
		t.Errorf("Description = %v, want 'User data'", result["description"])
	}

	if result["required"] != true {
		t.Errorf("Required = %v, want true", result["required"])
	}

	content := result["content"].(map[string]interface{})
	if content == nil {
		t.Fatal("Content is nil")
	}

	jsonContent := content["application/json"].(map[string]interface{})
	if jsonContent == nil {
		t.Fatal("JSON content is nil")
	}

	schema := jsonContent["schema"].(map[string]interface{})
	if schema["type"] != "object" {
		t.Errorf("Schema type = %v, want 'object'", schema["type"])
	}

	// Check that example is included
	if jsonContent["example"] == nil {
		t.Error("Example should be included")
	}
}

func TestOpenAPIGenerator_CreateSchemaObject(t *testing.T) {
	gen := &OpenAPIGenerator{}

	tests := []struct {
		name   string
		schema *SchemaDoc
		want   string
	}{
		{
			name:   "nil schema",
			schema: nil,
			want:   "",
		},
		{
			name: "string schema",
			schema: &SchemaDoc{
				Type: "string",
			},
			want: "string",
		},
		{
			name: "array schema",
			schema: &SchemaDoc{
				Type: "array",
				Items: &SchemaDoc{
					Type: "string",
				},
			},
			want: "array",
		},
		{
			name: "object schema",
			schema: &SchemaDoc{
				Type: "object",
				Properties: map[string]*PropertyDoc{
					"id": {
						Type:        "string",
						Description: "ID",
					},
					"name": {
						Type:        "string",
						Description: "Name",
					},
				},
				Required: []string{"id", "name"},
			},
			want: "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.createSchemaObject(tt.schema)

			if tt.schema == nil {
				if len(result) != 0 {
					t.Error("Expected empty map for nil schema")
				}
				return
			}

			if result["type"] != tt.want {
				t.Errorf("Type = %v, want %v", result["type"], tt.want)
			}

			// Check array items
			if tt.want == "array" {
				items := result["items"]
				if items == nil {
					t.Error("Array schema should have items")
				}
			}

			// Check object properties
			if tt.want == "object" && len(tt.schema.Properties) > 0 {
				props := result["properties"]
				if props == nil {
					t.Error("Object schema should have properties")
				}

				required := result["required"]
				if required == nil {
					t.Error("Object schema should have required fields")
				}
			}
		})
	}
}

func TestOpenAPIGenerator_CreatePropertyObject(t *testing.T) {
	gen := &OpenAPIGenerator{}

	tests := []struct {
		name     string
		prop     *PropertyDoc
		wantType string
	}{
		{
			name: "simple string property",
			prop: &PropertyDoc{
				Type:        "string",
				Description: "A string field",
			},
			wantType: "string",
		},
		{
			name: "property with format",
			prop: &PropertyDoc{
				Type:        "string",
				Description: "Email address",
				Format:      "email",
			},
			wantType: "string",
		},
		{
			name: "property with example",
			prop: &PropertyDoc{
				Type:        "integer",
				Description: "Age",
				Example:     25,
			},
			wantType: "integer",
		},
		{
			name: "property with enum",
			prop: &PropertyDoc{
				Type:        "string",
				Description: "Status",
				Enum:        []interface{}{"active", "inactive", "pending"},
			},
			wantType: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.createPropertyObject(tt.prop)

			if result["type"] != tt.wantType {
				t.Errorf("Type = %v, want %v", result["type"], tt.wantType)
			}

			if tt.prop.Description != "" {
				if result["description"] != tt.prop.Description {
					t.Errorf("Description = %v, want %v", result["description"], tt.prop.Description)
				}
			}

			if tt.prop.Format != "" {
				if result["format"] != tt.prop.Format {
					t.Errorf("Format = %v, want %v", result["format"], tt.prop.Format)
				}
			}

			if tt.prop.Example != nil {
				if result["example"] == nil {
					t.Error("Example should be included")
				}
			}

			if len(tt.prop.Enum) > 0 {
				if result["enum"] == nil {
					t.Error("Enum should be included")
				}
			}
		})
	}
}

func TestOpenAPIGenerator_CreateServers(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   int
	}{
		{
			name: "with base URL",
			config: &Config{
				BaseURL: "https://api.example.com",
			},
			want: 1,
		},
		{
			name: "with additional server URLs",
			config: &Config{
				BaseURL: "https://api.example.com",
				ServerURLs: []ServerURL{
					{URL: "https://staging.api.example.com", Description: "Staging"},
					{URL: "https://dev.api.example.com", Description: "Development"},
				},
			},
			want: 3,
		},
		{
			name:   "no servers configured",
			config: &Config{},
			want:   1, // Should have default localhost
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewOpenAPIGenerator(tt.config)
			result := gen.createServers()

			if len(result) != tt.want {
				t.Errorf("Server count = %v, want %v", len(result), tt.want)
			}

			// Check that each server has URL and description
			for i, server := range result {
				if server["url"] == nil {
					t.Errorf("Server %d missing URL", i)
				}
				if server["description"] == nil {
					t.Errorf("Server %d missing description", i)
				}
			}
		})
	}
}

func TestOpenAPIGenerator_CreateOperation(t *testing.T) {
	gen := &OpenAPIGenerator{}

	endpoint := &EndpointDoc{
		Method:      "POST",
		Path:        "/users",
		Summary:     "Create user",
		Description: "Create a new user",
		Parameters: []*ParameterDoc{
			{Name: "api_key", In: "header", Type: "string", Required: true},
		},
		RequestBody: &RequestBodyDoc{
			Description: "User data",
			Required:    true,
			ContentType: "application/json",
			Schema: &SchemaDoc{
				Type: "object",
			},
		},
		Responses: map[int]*ResponseDoc{
			201: {
				StatusCode:  201,
				Description: "Created",
				ContentType: "application/json",
			},
		},
	}

	result := gen.createOperation(endpoint, "User")

	if result["summary"] != "Create user" {
		t.Errorf("Summary = %v, want 'Create user'", result["summary"])
	}

	if result["description"] != "Create a new user" {
		t.Errorf("Description = %v, want 'Create a new user'", result["description"])
	}

	if result["operationId"] == nil {
		t.Error("Operation ID should be set")
	}

	tags := result["tags"].([]string)
	if len(tags) != 1 || tags[0] != "User" {
		t.Error("Tags should contain resource name")
	}

	if result["parameters"] == nil {
		t.Error("Parameters should be included")
	}

	if result["requestBody"] == nil {
		t.Error("Request body should be included")
	}

	if result["responses"] == nil {
		t.Error("Responses should be included")
	}
}
