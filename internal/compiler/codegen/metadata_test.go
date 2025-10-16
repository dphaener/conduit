package codegen

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestGenerator_GenerateMetadata(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name:          "User",
				Documentation: "User account",
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

	gen := NewGenerator()
	metadataJSON, err := gen.GenerateMetadata(prog)

	if err != nil {
		t.Fatalf("GenerateMetadata() error = %v", err)
	}

	if metadataJSON == "" {
		t.Fatal("GenerateMetadata() returned empty string")
	}

	// Verify it's valid JSON
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		t.Fatalf("GenerateMetadata() produced invalid JSON: %v", err)
	}

	// Check structure
	if version, ok := meta["version"].(string); !ok || version == "" {
		t.Error("Metadata missing version")
	}

	resources, ok := meta["resources"].([]interface{})
	if !ok {
		t.Fatal("Metadata missing resources")
	}

	if len(resources) != 1 {
		t.Errorf("Resources count = %v, want 1", len(resources))
	}
}

func TestGenerator_GenerateMetadataAccessor(t *testing.T) {
	metadataJSON := `{"version":"1.0.0","resources":[],"patterns":[],"routes":[]}`

	gen := NewGenerator()
	code, err := gen.GenerateMetadataAccessor(metadataJSON)

	if err != nil {
		t.Fatalf("GenerateMetadataAccessor() error = %v", err)
	}

	if code == "" {
		t.Fatal("GenerateMetadataAccessor() returned empty string")
	}

	// Check for required components
	requiredStrings := []string{
		"package introspection",
		"const Metadata =",
		"func GetMetadata()",
		"func QueryResources()",
		"func QueryPatterns()",
		"func QueryRoutes()",
		"func FindResource(",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(code, required) {
			t.Errorf("Generated code missing: %s", required)
		}
	}
}

func TestGenerator_GenerateProgram_IncludesMetadata(t *testing.T) {
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
		},
	}

	gen := NewGenerator()
	files, err := gen.GenerateProgram(prog)

	if err != nil {
		t.Fatalf("GenerateProgram() error = %v", err)
	}

	// Check that metadata files are generated
	if _, ok := files["introspection/metadata.json"]; !ok {
		t.Error("GenerateProgram() did not generate introspection/metadata.json")
	}

	if _, ok := files["introspection/introspection.go"]; !ok {
		t.Error("GenerateProgram() did not generate introspection/introspection.go")
	}

	// Verify metadata.json is valid
	metadataJSON := files["introspection/metadata.json"]
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		t.Errorf("Generated metadata.json is invalid: %v", err)
	}

	// Verify introspection.go is valid Go code
	introspectionCode := files["introspection/introspection.go"]
	if !strings.Contains(introspectionCode, "package introspection") {
		t.Error("Generated introspection.go missing package declaration")
	}
}

func TestEscapeBackticks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no backticks",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "single backtick",
			input:    "hello`world",
			expected: "hello` + \"`\" + `world",
		},
		{
			name:     "multiple backticks",
			input:    "`test`data`",
			expected: "` + \"`\" + `test` + \"`\" + `data` + \"`\" + `",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeBackticks(tt.input)
			if result != tt.expected {
				t.Errorf("escapeBackticks(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerator_MetadataWithAllFeatures(t *testing.T) {
	// Create a program with all features
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name:          "Post",
				Documentation: "Blog post",
				Fields: []*ast.FieldNode{
					{
						Name:     "title",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
						Constraints: []*ast.ConstraintNode{
							{Name: "min", Arguments: []ast.ExprNode{&ast.LiteralExpr{Value: 5}}},
							{Name: "max", Arguments: []ast.ExprNode{&ast.LiteralExpr{Value: 200}}},
						},
					},
					{
						Name:     "slug",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
						Constraints: []*ast.ConstraintNode{
							{Name: "unique"},
						},
					},
				},
				Relationships: []*ast.RelationshipNode{
					{
						Name:       "author",
						Type:       "User",
						Kind:       ast.RelationshipBelongsTo,
						ForeignKey: "author_id",
						Nullable:   false,
					},
				},
				Hooks: []*ast.HookNode{
					{
						Timing:        "before",
						Event:         "create",
						IsTransaction: false,
						IsAsync:       false,
					},
					{
						Timing:        "after",
						Event:         "create",
						IsTransaction: true,
						IsAsync:       true,
						Middleware:    []string{"auth"},
					},
				},
				Validations: []*ast.ValidationNode{
					{
						Name:      "title_not_empty",
						Condition: &ast.LiteralExpr{Value: true},
						Error:     "Title cannot be empty",
					},
				},
			},
		},
	}

	gen := NewGenerator()
	metadataJSON, err := gen.GenerateMetadata(prog)

	if err != nil {
		t.Fatalf("GenerateMetadata() error = %v", err)
	}

	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Verify resources
	resources, ok := meta["resources"].([]interface{})
	if !ok || len(resources) != 1 {
		t.Fatal("Resources not properly structured")
	}

	resource := resources[0].(map[string]interface{})

	// Check fields
	fields, ok := resource["fields"].([]interface{})
	if !ok || len(fields) != 2 {
		t.Errorf("Fields not properly structured, got %v", fields)
	}

	// Check relationships
	relationships, ok := resource["relationships"].([]interface{})
	if !ok || len(relationships) != 1 {
		t.Errorf("Relationships not properly structured")
	}

	// Check hooks
	hooks, ok := resource["hooks"].([]interface{})
	if !ok || len(hooks) != 2 {
		t.Errorf("Hooks not properly structured")
	}

	// Check validations
	validations, ok := resource["validations"].([]interface{})
	if !ok || len(validations) != 1 {
		t.Errorf("Validations not properly structured")
	}

	// Check patterns are extracted
	patterns, ok := meta["patterns"].([]interface{})
	if !ok {
		t.Error("Patterns not properly structured")
	}

	if len(patterns) == 0 {
		t.Error("Expected patterns to be extracted")
	}

	// Check routes
	routes, ok := meta["routes"].([]interface{})
	if !ok || len(routes) != 5 {
		t.Errorf("Routes not properly structured, got %d routes", len(routes))
	}
}

// BenchmarkGenerateMetadata tests metadata generation performance
func BenchmarkGenerateMetadata(b *testing.B) {
	// Create a realistic program with multiple resources
	prog := &ast.Program{
		Resources: make([]*ast.ResourceNode, 10),
	}

	for i := 0; i < 10; i++ {
		prog.Resources[i] = &ast.ResourceNode{
			Name: "Resource" + string(rune('A'+i)),
			Fields: []*ast.FieldNode{
				{
					Name:     "id",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
					Nullable: false,
				},
				{
					Name:     "name",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
					Nullable: false,
				},
				{
					Name:     "created_at",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "timestamp", Nullable: false},
					Nullable: false,
				},
			},
			Hooks: []*ast.HookNode{
				{
					Timing: "after",
					Event:  "create",
				},
			},
		}
	}

	gen := NewGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.GenerateMetadata(prog)
		if err != nil {
			b.Fatalf("GenerateMetadata() error = %v", err)
		}
	}
}

// TestMetadataGenerationPerformance verifies < 50ms requirement
func TestMetadataGenerationPerformance(t *testing.T) {
	// Create a realistic program
	prog := &ast.Program{
		Resources: make([]*ast.ResourceNode, 20),
	}

	for i := 0; i < 20; i++ {
		prog.Resources[i] = &ast.ResourceNode{
			Name: "Resource" + string(rune('A'+i)),
			Fields: []*ast.FieldNode{
				{
					Name:     "id",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
					Nullable: false,
				},
				{
					Name:     "name",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
					Nullable: false,
				},
			},
		}
	}

	gen := NewGenerator()

	start := time.Now()
	_, err := gen.GenerateMetadata(prog)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GenerateMetadata() error = %v", err)
	}

	// Check performance requirement: < 50ms
	if duration > 50*time.Millisecond {
		t.Errorf("GenerateMetadata() took %v, want < 50ms", duration)
	} else {
		t.Logf("GenerateMetadata() took %v (< 50ms requirement met)", duration)
	}
}
