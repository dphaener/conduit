package docs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAPIGenerator_Generate(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	config := &Config{
		ProjectName:        "TestAPI",
		ProjectVersion:     "1.0.0",
		ProjectDescription: "Test API",
		OutputDir:          tmpDir,
		BaseURL:            "https://api.example.com",
	}

	generator := NewOpenAPIGenerator(config)

	doc := &Documentation{
		ProjectInfo: &ProjectInfo{
			Name:        "TestAPI",
			Version:     "1.0.0",
			Description: "Test API",
		},
		Resources: []*ResourceDoc{
			{
				Name:          "User",
				Documentation: "User resource",
				Fields: []*FieldDoc{
					{
						Name:     "id",
						Type:     "uuid!",
						Required: true,
						Example:  "550e8400-e29b-41d4-a716-446655440000",
					},
					{
						Name:     "email",
						Type:     "email!",
						Required: true,
						Example:  "user@example.com",
					},
				},
				Endpoints: []*EndpointDoc{
					{
						Method:      "GET",
						Path:        "/users",
						Summary:     "List users",
						Description: "Get all users",
						Responses: map[int]*ResponseDoc{
							200: {
								StatusCode:  200,
								Description: "Success",
								ContentType: "application/json",
							},
						},
					},
				},
			},
		},
	}

	err := generator.Generate(doc)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify file was created
	outputPath := filepath.Join(tmpDir, "openapi.json")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("OpenAPI file was not created")
	}

	// Read and parse the file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read OpenAPI file: %v", err)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("Failed to parse OpenAPI JSON: %v", err)
	}

	// Verify structure
	if spec["openapi"] != "3.0.3" {
		t.Errorf("Expected OpenAPI version 3.0.3, got %v", spec["openapi"])
	}

	info := spec["info"].(map[string]interface{})
	if info["title"] != "TestAPI" {
		t.Errorf("Expected title 'TestAPI', got %v", info["title"])
	}

	if info["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", info["version"])
	}

	// Verify servers
	servers := spec["servers"].([]interface{})
	if len(servers) == 0 {
		t.Error("Expected at least one server")
	}

	// Verify paths
	paths := spec["paths"].(map[string]interface{})
	if len(paths) == 0 {
		t.Error("Expected paths to be generated")
	}

	// Verify components
	components := spec["components"].(map[string]interface{})
	schemas := components["schemas"].(map[string]interface{})
	if len(schemas) == 0 {
		t.Error("Expected schemas to be generated")
	}
}

func TestOpenAPIGenerator_CreateSpec(t *testing.T) {
	config := &Config{
		ProjectName:    "TestAPI",
		ProjectVersion: "1.0.0",
		BaseURL:        "https://api.example.com",
	}

	generator := NewOpenAPIGenerator(config)

	doc := &Documentation{
		ProjectInfo: &ProjectInfo{
			Name:    "TestAPI",
			Version: "1.0.0",
		},
		Resources: []*ResourceDoc{},
	}

	spec := generator.createSpec(doc)

	if spec["openapi"] != "3.0.3" {
		t.Errorf("Expected OpenAPI 3.0.3, got %v", spec["openapi"])
	}

	info := spec["info"].(map[string]interface{})
	if info["title"] != "TestAPI" {
		t.Errorf("Expected title 'TestAPI', got %v", info["title"])
	}

	if spec["paths"] == nil {
		t.Error("Expected paths to be present")
	}

	if spec["components"] == nil {
		t.Error("Expected components to be present")
	}
}

func TestOpenAPIGenerator_MapTypeToOpenAPI(t *testing.T) {
	generator := &OpenAPIGenerator{}

	tests := []struct {
		conduitType string
		expected    string
	}{
		{"int!", "integer"},
		{"integer!", "integer"},
		{"bigint!", "integer"},
		{"float!", "number"},
		{"decimal!", "number"},
		{"bool!", "boolean"},
		{"string!", "string"},
		{"uuid!", "string"},
		{"email!", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.conduitType, func(t *testing.T) {
			result := generator.mapTypeToOpenAPI(tt.conduitType)
			if result != tt.expected {
				t.Errorf("mapTypeToOpenAPI(%s) = %s, expected %s", tt.conduitType, result, tt.expected)
			}
		})
	}
}
