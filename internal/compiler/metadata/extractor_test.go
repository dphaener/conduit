package metadata

import (
	"encoding/json"
	"strings"
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
	expectedRoutes := map[string]struct {
		handler   string
		operation string
	}{
		"GET /users":        {handler: "User.list", operation: "list"},
		"GET /users/:id":    {handler: "User.get", operation: "get"},
		"POST /users":       {handler: "User.create", operation: "create"},
		"PUT /users/:id":    {handler: "User.update", operation: "update"},
		"DELETE /users/:id": {handler: "User.delete", operation: "delete"},
	}

	for _, route := range meta.Routes {
		key := route.Method + " " + route.Path
		expected, ok := expectedRoutes[key]
		if !ok {
			t.Errorf("Unexpected route: %v", key)
			continue
		}
		if route.Handler != expected.handler {
			t.Errorf("Route %v handler = %v, want %v", key, route.Handler, expected.handler)
		}
		if route.Operation != expected.operation {
			t.Errorf("Route %v operation = %v, want %v", key, route.Operation, expected.operation)
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
func TestExtractor_GenerateRoutes_StandardRoutes(t *testing.T) {
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
				Middleware: []string{"auth", "cache(300)"},
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

	// Verify routes are generated with correct structure
	expectedRoutes := map[string]struct {
		method    string
		path      string
		handler   string
		operation string
	}{
		"list": {
			method:    "GET",
			path:      "/posts",
			handler:   "Post.list",
			operation: "list",
		},
		"get": {
			method:    "GET",
			path:      "/posts/:id",
			handler:   "Post.get",
			operation: "get",
		},
		"create": {
			method:    "POST",
			path:      "/posts",
			handler:   "Post.create",
			operation: "create",
		},
		"update": {
			method:    "PUT",
			path:      "/posts/:id",
			handler:   "Post.update",
			operation: "update",
		},
		"delete": {
			method:    "DELETE",
			path:      "/posts/:id",
			handler:   "Post.delete",
			operation: "delete",
		},
	}

	for _, route := range meta.Routes {
		expected, ok := expectedRoutes[route.Operation]
		if !ok {
			t.Errorf("Unexpected route operation: %v", route.Operation)
			continue
		}

		if route.Method != expected.method {
			t.Errorf("Route %v method = %v, want %v", route.Operation, route.Method, expected.method)
		}
		if route.Path != expected.path {
			t.Errorf("Route %v path = %v, want %v", route.Operation, route.Path, expected.path)
		}
		if route.Handler != expected.handler {
			t.Errorf("Route %v handler = %v, want %v", route.Operation, route.Handler, expected.handler)
		}
		if route.Resource != "Post" {
			t.Errorf("Route %v resource = %v, want Post", route.Operation, route.Resource)
		}

		// Check middleware is applied
		if len(route.Middleware) != 2 {
			t.Errorf("Route %v middleware count = %v, want 2", route.Operation, len(route.Middleware))
		}
	}
}

func TestExtractor_GenerateRoutes_WithOperationsRestriction(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name:       "Post",
				Operations: []string{"list", "get"}, // Only allow list and get
				Fields: []*ast.FieldNode{
					{
						Name:     "title",
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

	// Should only generate 2 routes (list and get)
	if len(meta.Routes) != 2 {
		t.Fatalf("Routes count = %v, want 2", len(meta.Routes))
	}

	operations := make(map[string]bool)
	for _, route := range meta.Routes {
		operations[route.Operation] = true
	}

	// Verify only list and get operations exist
	if !operations["list"] {
		t.Error("Missing list operation")
	}
	if !operations["get"] {
		t.Error("Missing get operation")
	}
	if operations["create"] || operations["update"] || operations["delete"] {
		t.Error("Should not have create, update, or delete operations")
	}
}

func TestExtractor_GenerateRoutes_NestedResources(t *testing.T) {
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
						Name: "comments",
						Type: "Comment",
						Kind: ast.RelationshipHasMany,
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

	// Should generate 5 standard routes + 1 nested route
	if len(meta.Routes) != 6 {
		t.Fatalf("Routes count = %v, want 6", len(meta.Routes))
	}

	// Find the nested route
	var nestedRoute *RouteMetadata
	for i := range meta.Routes {
		if meta.Routes[i].Operation == "list_comments" {
			nestedRoute = &meta.Routes[i]
			break
		}
	}

	if nestedRoute == nil {
		t.Fatal("Nested route not found")
	}

	if nestedRoute.Method != "GET" {
		t.Errorf("Nested route method = %v, want GET", nestedRoute.Method)
	}
	if nestedRoute.Path != "/posts/:id/comments" {
		t.Errorf("Nested route path = %v, want /posts/:id/comments", nestedRoute.Path)
	}
	if nestedRoute.Handler != "Post.comments.list" {
		t.Errorf("Nested route handler = %v, want Post.comments.list", nestedRoute.Handler)
	}
	if nestedRoute.Resource != "Post" {
		t.Errorf("Nested route resource = %v, want Post", nestedRoute.Resource)
	}
}

func TestExtractor_ToPlural(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""}, // empty string
		{"Post", "posts"},
		{"Comment", "comments"},
		{"Category", "categories"},
		{"Person", "people"},
		{"Child", "children"},
		{"Man", "men"},
		{"Woman", "women"},
		{"Box", "boxes"},
		{"Class", "classes"},
		{"Dish", "dishes"},
		{"Church", "churches"},
		{"Status", "statuses"},
		{"Day", "days"}, // vowel before 'y'
	}

	extractor := NewExtractor("1.0.0")

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractor.toPlural(strings.ToLower(tt.input))
			want := tt.want
			if got != want {
				t.Errorf("toPlural(%v) = %v, want %v", tt.input, got, want)
			}
		})
	}
}

func TestExtractor_GenerateRoutes_NoMiddleware(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
				Fields: []*ast.FieldNode{
					{
						Name:     "username",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
				// No middleware specified
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	// Verify routes have no middleware
	for _, route := range meta.Routes {
		if len(route.Middleware) != 0 {
			t.Errorf("Route %v should have no middleware, got %v", route.Operation, route.Middleware)
		}
	}
}

func TestExtractor_GenerateRoutes_MultipleResources(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
				Fields: []*ast.FieldNode{
					{
						Name:     "username",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
			},
			{
				Name: "Post",
				Fields: []*ast.FieldNode{
					{
						Name:     "title",
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

	// Should generate 5 routes per resource = 10 total
	if len(meta.Routes) != 10 {
		t.Fatalf("Routes count = %v, want 10", len(meta.Routes))
	}

	// Count routes by resource
	resourceCounts := make(map[string]int)
	for _, route := range meta.Routes {
		resourceCounts[route.Resource]++
	}

	if resourceCounts["User"] != 5 {
		t.Errorf("User routes count = %v, want 5", resourceCounts["User"])
	}
	if resourceCounts["Post"] != 5 {
		t.Errorf("Post routes count = %v, want 5", resourceCounts["Post"])
	}
}
