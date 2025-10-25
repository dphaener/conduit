package docs

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestExtractor_Extract(t *testing.T) {
	extractor := NewExtractor()

	program := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name:          "User",
				Documentation: "/// User resource for authentication",
				Fields: []*ast.FieldNode{
					{
						Name:     "id",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
						Nullable: false,
					},
					{
						Name:     "email",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "email", Nullable: false},
						Nullable: false,
						Constraints: []*ast.ConstraintNode{
							{Name: "unique", Arguments: []ast.ExprNode{}},
						},
					},
					{
						Name:     "name",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: true},
						Nullable: true,
					},
				},
				Hooks: []*ast.HookNode{
					{
						Timing:        "before",
						Event:         "create",
						IsAsync:       false,
						IsTransaction: false,
					},
				},
			},
		},
	}

	doc := extractor.Extract(program, "TestApp", "1.0.0", "Test application")

	// Test project info
	if doc.ProjectInfo.Name != "TestApp" {
		t.Errorf("Expected project name 'TestApp', got '%s'", doc.ProjectInfo.Name)
	}

	if doc.ProjectInfo.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", doc.ProjectInfo.Version)
	}

	// Test resources
	if len(doc.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(doc.Resources))
	}

	resource := doc.Resources[0]
	if resource.Name != "User" {
		t.Errorf("Expected resource name 'User', got '%s'", resource.Name)
	}

	// Test fields
	if len(resource.Fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(resource.Fields))
	}

	emailField := resource.Fields[1]
	if emailField.Name != "email" {
		t.Errorf("Expected field name 'email', got '%s'", emailField.Name)
	}

	if !emailField.Required {
		t.Error("Expected email field to be required")
	}

	if emailField.Type != "email!" {
		t.Errorf("Expected type 'email!', got '%s'", emailField.Type)
	}

	if len(emailField.Constraints) != 1 {
		t.Fatalf("Expected 1 constraint, got %d", len(emailField.Constraints))
	}

	// Test hooks
	if len(resource.Hooks) != 1 {
		t.Fatalf("Expected 1 hook, got %d", len(resource.Hooks))
	}

	hook := resource.Hooks[0]
	if hook.Timing != "before" || hook.Event != "create" {
		t.Errorf("Expected before create hook, got %s %s", hook.Timing, hook.Event)
	}

	// Test endpoints
	if len(resource.Endpoints) != 5 {
		t.Fatalf("Expected 5 endpoints, got %d", len(resource.Endpoints))
	}

	// Check GET list endpoint
	listEndpoint := resource.Endpoints[0]
	if listEndpoint.Method != "GET" || listEndpoint.Path != "/users" {
		t.Errorf("Expected GET /users, got %s %s", listEndpoint.Method, listEndpoint.Path)
	}

	// Check POST endpoint
	createEndpoint := resource.Endpoints[2]
	if createEndpoint.Method != "POST" {
		t.Errorf("Expected POST method, got %s", createEndpoint.Method)
	}

	if createEndpoint.RequestBody == nil {
		t.Error("Expected request body for POST endpoint")
	}
}

func TestExtractor_FormatType(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		typeNode *ast.TypeNode
		expected string
	}{
		{
			name:     "required string",
			typeNode: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
			expected: "string!",
		},
		{
			name:     "optional int",
			typeNode: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int", Nullable: true},
			expected: "int?",
		},
		{
			name: "array of strings",
			typeNode: &ast.TypeNode{
				Kind:        ast.TypeArray,
				Nullable:    false,
				ElementType: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
			},
			expected: "array<string!>!",
		},
		{
			name: "hash",
			typeNode: &ast.TypeNode{
				Kind:      ast.TypeHash,
				Nullable:  false,
				KeyType:   &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
				ValueType: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int", Nullable: false},
			},
			expected: "hash<string!,int!>!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.formatType(tt.typeNode)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "users"},
		{"post", "posts"},
		{"category", "categories"},
		{"box", "boxes"},
		{"church", "churches"},
		{"dish", "dishes"},
		{"day", "days"},
		{"news", "newses"}, // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pluralize(tt.input)
			if result != tt.expected {
				t.Errorf("pluralize(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCleanDocumentation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line",
			input:    "/// User resource",
			expected: "User resource",
		},
		{
			name:     "multiple lines",
			input:    "/// First line\n/// Second line\n/// Third line",
			expected: "First line Second line Third line",
		},
		{
			name:     "with whitespace",
			input:    "///   Trimmed   ",
			expected: "Trimmed",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanDocumentation(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestExtractor_GenerateEndpoints(t *testing.T) {
	extractor := NewExtractor()

	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "id",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
				Nullable: false,
			},
			{
				Name:     "title",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
				Nullable: false,
			},
		},
	}

	endpoints := extractor.generateEndpoints(resource)

	// Should generate 5 REST endpoints
	if len(endpoints) != 5 {
		t.Fatalf("Expected 5 endpoints, got %d", len(endpoints))
	}

	// Test GET list
	if endpoints[0].Method != "GET" || endpoints[0].Path != "/posts" {
		t.Errorf("Expected GET /posts, got %s %s", endpoints[0].Method, endpoints[0].Path)
	}

	// Test GET single
	if endpoints[1].Method != "GET" || endpoints[1].Path != "/posts/:id" {
		t.Errorf("Expected GET /posts/:id, got %s %s", endpoints[1].Method, endpoints[1].Path)
	}

	// Test POST
	if endpoints[2].Method != "POST" || endpoints[2].Path != "/posts" {
		t.Errorf("Expected POST /posts, got %s %s", endpoints[2].Method, endpoints[2].Path)
	}

	if endpoints[2].RequestBody == nil {
		t.Error("POST endpoint should have request body")
	}

	// Test PUT
	if endpoints[3].Method != "PUT" || endpoints[3].Path != "/posts/:id" {
		t.Errorf("Expected PUT /posts/:id, got %s %s", endpoints[3].Method, endpoints[3].Path)
	}

	// Test DELETE
	if endpoints[4].Method != "DELETE" || endpoints[4].Path != "/posts/:id" {
		t.Errorf("Expected DELETE /posts/:id, got %s %s", endpoints[4].Method, endpoints[4].Path)
	}

	// Test responses
	if endpoints[0].Responses[200] == nil {
		t.Error("GET list endpoint should have 200 response")
	}

	if endpoints[2].Responses[201] == nil {
		t.Error("POST endpoint should have 201 response")
	}

	if endpoints[4].Responses[204] == nil {
		t.Error("DELETE endpoint should have 204 response")
	}
}

func TestExtractor_CreateSchema(t *testing.T) {
	extractor := NewExtractor()

	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{
				Name:     "id",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
				Nullable: false,
			},
			{
				Name:     "email",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "email", Nullable: false},
				Nullable: false,
			},
			{
				Name:     "age",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int", Nullable: true},
				Nullable: true,
			},
		},
	}

	schema := extractor.createObjectSchema(resource)

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if len(schema.Properties) != 3 {
		t.Fatalf("Expected 3 properties, got %d", len(schema.Properties))
	}

	if schema.Properties["id"].Type != "string" {
		t.Errorf("Expected id type 'string', got '%s'", schema.Properties["id"].Type)
	}

	if schema.Properties["id"].Format != "uuid" {
		t.Errorf("Expected id format 'uuid', got '%s'", schema.Properties["id"].Format)
	}

	if schema.Properties["age"].Type != "integer" {
		t.Errorf("Expected age type 'integer', got '%s'", schema.Properties["age"].Type)
	}

	// Check required fields
	if len(schema.Required) != 2 {
		t.Fatalf("Expected 2 required fields, got %d", len(schema.Required))
	}

	requiredMap := make(map[string]bool)
	for _, req := range schema.Required {
		requiredMap[req] = true
	}

	if !requiredMap["id"] || !requiredMap["email"] {
		t.Error("Expected id and email to be required")
	}

	if requiredMap["age"] {
		t.Error("Expected age to be optional")
	}
}
