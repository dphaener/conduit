package metadata

import (
	"encoding/json"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestExtractor_Extract_SimpleResource(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name:          "User",
				Documentation: "User represents a user account",
				Fields: []*ast.FieldNode{
					{
						Name:     "id",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
						Nullable: false,
					},
					{
						Name:     "username",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
					{
						Name:     "email",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if meta.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", meta.Version)
	}

	if len(meta.Resources) != 1 {
		t.Fatalf("Resources count = %v, want 1", len(meta.Resources))
	}

	resource := meta.Resources[0]
	if resource.Name != "User" {
		t.Errorf("Resource name = %v, want User", resource.Name)
	}

	if resource.Documentation != "User represents a user account" {
		t.Errorf("Resource documentation = %v, want 'User represents a user account'", resource.Documentation)
	}

	if len(resource.Fields) != 3 {
		t.Fatalf("Fields count = %v, want 3", len(resource.Fields))
	}

	// Check field types
	expectedFields := map[string]string{
		"id":       "uuid!",
		"username": "string!",
		"email":    "string!",
	}

	for _, field := range resource.Fields {
		expectedType, ok := expectedFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %v", field.Name)
			continue
		}
		if field.Type != expectedType {
			t.Errorf("Field %v type = %v, want %v", field.Name, field.Type, expectedType)
		}
	}
}

func TestExtractor_Extract_WithRelationships(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Post",
				Fields: []*ast.FieldNode{
					{
						Name:     "title",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
				Relationships: []*ast.RelationshipNode{
					{
						Name:       "author",
						Type:       "User",
						Kind:       ast.RelationshipBelongsTo,
						ForeignKey: "author_id",
						OnDelete:   "cascade",
						Nullable:   false,
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(meta.Resources) != 1 {
		t.Fatalf("Resources count = %v, want 1", len(meta.Resources))
	}

	resource := meta.Resources[0]
	if len(resource.Relationships) != 1 {
		t.Fatalf("Relationships count = %v, want 1", len(resource.Relationships))
	}

	rel := resource.Relationships[0]
	if rel.Name != "author" {
		t.Errorf("Relationship name = %v, want author", rel.Name)
	}
	if rel.Type != "User" {
		t.Errorf("Relationship type = %v, want User", rel.Type)
	}
	if rel.Kind != "belongs_to" {
		t.Errorf("Relationship kind = %v, want belongs_to", rel.Kind)
	}
	if rel.ForeignKey != "author_id" {
		t.Errorf("ForeignKey = %v, want author_id", rel.ForeignKey)
	}
	if rel.OnDelete != "cascade" {
		t.Errorf("OnDelete = %v, want cascade", rel.OnDelete)
	}
}

func TestExtractor_Extract_WithHooks(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Post",
				Fields: []*ast.FieldNode{
					{
						Name:     "title",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
				Hooks: []*ast.HookNode{
					{
						Timing:        "after",
						Event:         "create",
						IsTransaction: true,
						IsAsync:       false,
						Middleware:    []string{"auth"},
					},
					{
						Timing:        "before",
						Event:         "delete",
						IsTransaction: false,
						IsAsync:       true,
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	resource := meta.Resources[0]
	if len(resource.Hooks) != 2 {
		t.Fatalf("Hooks count = %v, want 2", len(resource.Hooks))
	}

	// Check first hook
	hook1 := resource.Hooks[0]
	if hook1.Timing != "after" {
		t.Errorf("Hook1 timing = %v, want after", hook1.Timing)
	}
	if hook1.Event != "create" {
		t.Errorf("Hook1 event = %v, want create", hook1.Event)
	}
	if !hook1.HasTransaction {
		t.Error("Hook1 should have transaction")
	}
	if hook1.HasAsync {
		t.Error("Hook1 should not be async")
	}
	if len(hook1.Middleware) != 1 || hook1.Middleware[0] != "auth" {
		t.Errorf("Hook1 middleware = %v, want [auth]", hook1.Middleware)
	}

	// Check second hook
	hook2 := resource.Hooks[1]
	if hook2.Timing != "before" {
		t.Errorf("Hook2 timing = %v, want before", hook2.Timing)
	}
	if hook2.Event != "delete" {
		t.Errorf("Hook2 event = %v, want delete", hook2.Event)
	}
	if hook2.HasTransaction {
		t.Error("Hook2 should not have transaction")
	}
	if !hook2.HasAsync {
		t.Error("Hook2 should be async")
	}
}

func TestExtractor_Extract_Patterns(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Post",
				Fields: []*ast.FieldNode{
					{
						Name:     "slug",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
						Constraints: []*ast.ConstraintNode{
							{Name: "unique"},
						},
					},
				},
				Hooks: []*ast.HookNode{
					{
						Timing:        "after",
						Event:         "create",
						IsTransaction: true,
						Middleware:    []string{"auth"},
					},
				},
			},
			{
				Name: "Comment",
				Fields: []*ast.FieldNode{
					{
						Name:     "email",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
						Constraints: []*ast.ConstraintNode{
							{Name: "unique"},
						},
					},
				},
				Hooks: []*ast.HookNode{
					{
						Timing:  "before",
						Event:   "create",
						IsAsync: true,
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	// Check patterns
	if len(meta.Patterns) == 0 {
		t.Fatal("Expected patterns to be extracted")
	}

	// Build pattern map for easier testing
	patterns := make(map[string]PatternMetadata)
	for _, p := range meta.Patterns {
		patterns[p.Name] = p
	}

	// Check for authenticated_handler pattern
	if p, ok := patterns["authenticated_handler"]; ok {
		if p.Occurrences != 1 {
			t.Errorf("authenticated_handler occurrences = %v, want 1", p.Occurrences)
		}
	}

	// Check for transactional_hook pattern
	if p, ok := patterns["transactional_hook"]; ok {
		if p.Occurrences != 1 {
			t.Errorf("transactional_hook occurrences = %v, want 1", p.Occurrences)
		}
	}

	// Check for async_operation pattern
	if p, ok := patterns["async_operation"]; ok {
		if p.Occurrences != 1 {
			t.Errorf("async_operation occurrences = %v, want 1", p.Occurrences)
		}
	}

	// Check for unique_field pattern
	if p, ok := patterns["unique_field"]; ok {
		if p.Occurrences != 2 {
			t.Errorf("unique_field occurrences = %v, want 2", p.Occurrences)
		}
	}
}

func TestExtractor_Extract_Routes(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name:       "User",
				Middleware: []string{"cors", "auth"},
				Fields: []*ast.FieldNode{
					{
						Name:     "username",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	// Should generate 5 standard REST routes
	if len(meta.Routes) != 5 {
		t.Fatalf("Routes count = %v, want 5", len(meta.Routes))
	}

	// Check route patterns
	expectedRoutes := map[string]string{
		"GET /users":        "Index",
		"GET /users/:id":    "Show",
		"POST /users":       "Create",
		"PUT /users/:id":    "Update",
		"DELETE /users/:id": "Delete",
	}

	for _, route := range meta.Routes {
		key := route.Method + " " + route.Path
		expectedHandler, ok := expectedRoutes[key]
		if !ok {
			t.Errorf("Unexpected route: %v", key)
			continue
		}
		if route.Handler != expectedHandler {
			t.Errorf("Route %v handler = %v, want %v", key, route.Handler, expectedHandler)
		}
		if route.Resource != "User" {
			t.Errorf("Route %v resource = %v, want User", key, route.Resource)
		}
		if len(route.Middleware) != 2 {
			t.Errorf("Route %v middleware count = %v, want 2", key, len(route.Middleware))
		}
	}
}

func TestMetadata_ToJSON(t *testing.T) {
	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "User",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid!", Nullable: false},
					{Name: "email", Type: "string!", Nullable: false},
				},
			},
		},
		Patterns: []PatternMetadata{
			{Name: "test_pattern", Template: "test", Occurrences: 1},
		},
		Routes: []RouteMetadata{
			{Method: "GET", Path: "/users", Handler: "Index", Resource: "User"},
		},
	}

	jsonStr, err := meta.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Generated invalid JSON: %v", err)
	}

	// Check structure
	if version, ok := parsed["version"].(string); !ok || version != "1.0.0" {
		t.Errorf("Version in JSON = %v, want 1.0.0", parsed["version"])
	}

	resources, ok := parsed["resources"].([]interface{})
	if !ok || len(resources) != 1 {
		t.Errorf("Resources in JSON not correct")
	}
}

func TestFromJSON(t *testing.T) {
	jsonStr := `{
		"version": "1.0.0",
		"resources": [
			{
				"name": "User",
				"fields": [
					{"name": "id", "type": "uuid!", "nullable": false}
				]
			}
		],
		"patterns": [],
		"routes": []
	}`

	meta, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if meta.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", meta.Version)
	}

	if len(meta.Resources) != 1 {
		t.Fatalf("Resources count = %v, want 1", len(meta.Resources))
	}

	if meta.Resources[0].Name != "User" {
		t.Errorf("Resource name = %v, want User", meta.Resources[0].Name)
	}
}

func TestExtractor_ComplexTypes(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Post",
				Fields: []*ast.FieldNode{
					{
						Name: "tags",
						Type: &ast.TypeNode{
							Kind: ast.TypeArray,
							ElementType: &ast.TypeNode{
								Kind:     ast.TypePrimitive,
								Name:     "string",
								Nullable: false,
							},
							Nullable: false,
						},
						Nullable: false,
					},
					{
						Name: "metadata",
						Type: &ast.TypeNode{
							Kind: ast.TypeHash,
							KeyType: &ast.TypeNode{
								Kind:     ast.TypePrimitive,
								Name:     "string",
								Nullable: false,
							},
							ValueType: &ast.TypeNode{
								Kind:     ast.TypePrimitive,
								Name:     "string",
								Nullable: false,
							},
							Nullable: true,
						},
						Nullable: true,
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	resource := meta.Resources[0]
	if len(resource.Fields) != 2 {
		t.Fatalf("Fields count = %v, want 2", len(resource.Fields))
	}

	// Check array type
	tagsField := resource.Fields[0]
	if tagsField.Type != "array<string!>!" {
		t.Errorf("Tags type = %v, want array<string!>!", tagsField.Type)
	}

	// Check hash type
	metadataField := resource.Fields[1]
	if metadataField.Type != "hash<string!,string!>?" {
		t.Errorf("Metadata type = %v, want hash<string!,string!>?", metadataField.Type)
	}
}

func TestExtractor_RelationshipKinds(t *testing.T) {
	tests := []struct {
		kind     ast.RelationshipKind
		expected string
	}{
		{ast.RelationshipBelongsTo, "belongs_to"},
		{ast.RelationshipHasMany, "has_many"},
		{ast.RelationshipHasManyThrough, "has_many_through"},
		{ast.RelationshipHasOne, "has_one"},
	}

	extractor := NewExtractor("1.0.0")
	for _, tt := range tests {
		result := extractor.formatRelationshipKind(tt.kind)
		if result != tt.expected {
			t.Errorf("formatRelationshipKind(%v) = %q, want %q", tt.kind, result, tt.expected)
		}
	}
}

func TestExtractor_StructTypes(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Config",
				Fields: []*ast.FieldNode{
					{
						Name: "settings",
						Type: &ast.TypeNode{
							Kind: ast.TypeStruct,
							StructFields: []*ast.FieldNode{
								{
									Name: "theme",
									Type: &ast.TypeNode{
										Kind:     ast.TypePrimitive,
										Name:     "string",
										Nullable: false,
									},
								},
								{
									Name: "notifications",
									Type: &ast.TypeNode{
										Kind:     ast.TypePrimitive,
										Name:     "bool",
										Nullable: false,
									},
								},
							},
							Nullable: false,
						},
						Nullable: false,
					},
					{
						Name: "emptyStruct",
						Type: &ast.TypeNode{
							Kind:         ast.TypeStruct,
							StructFields: []*ast.FieldNode{},
							Nullable:     true,
						},
						Nullable: true,
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	resource := meta.Resources[0]
	if len(resource.Fields) != 2 {
		t.Fatalf("Fields count = %v, want 2", len(resource.Fields))
	}

	// Check struct type with fields
	settingsField := resource.Fields[0]
	expectedType := "struct{theme: string!, notifications: bool!}!"
	if settingsField.Type != expectedType {
		t.Errorf("Settings type = %v, want %v", settingsField.Type, expectedType)
	}

	// Check empty struct type
	emptyStructField := resource.Fields[1]
	if emptyStructField.Type != "struct{}?" {
		t.Errorf("EmptyStruct type = %v, want struct{}?", emptyStructField.Type)
	}
}
