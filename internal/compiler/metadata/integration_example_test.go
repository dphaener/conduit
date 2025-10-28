package metadata_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/codegen"
	"github.com/conduit-lang/conduit/internal/compiler/metadata"
)

// TestCompleteWorkflow demonstrates the complete metadata generation workflow
func TestCompleteWorkflow(t *testing.T) {
	// Step 1: Create a realistic program AST
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name:          "Post",
				Documentation: "Blog post with title and content",
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
					{
						Name:     "content",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text", Nullable: false},
						Nullable: false,
					},
					{
						Name:     "published_at",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "timestamp", Nullable: true},
						Nullable: true,
					},
				},
				Relationships: []*ast.RelationshipNode{
					{
						Name:       "author",
						Type:       "User",
						Kind:       ast.RelationshipBelongsTo,
						ForeignKey: "author_id",
						OnDelete:   "restrict",
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
			{
				Name:          "Comment",
				Documentation: "User comment on a post",
				Fields: []*ast.FieldNode{
					{
						Name:     "id",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
						Nullable: false,
					},
					{
						Name:     "content",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text", Nullable: false},
						Nullable: false,
					},
				},
				Relationships: []*ast.RelationshipNode{
					{
						Name:       "post",
						Type:       "Post",
						Kind:       ast.RelationshipBelongsTo,
						ForeignKey: "post_id",
						Nullable:   false,
					},
				},
			},
		},
	}

	// Step 2: Extract metadata using the extractor
	extractor := metadata.NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Step 3: Verify metadata structure
	if meta.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", meta.Version)
	}

	if len(meta.Resources) != 2 {
		t.Fatalf("Resources count = %v, want 2", len(meta.Resources))
	}

	// Check Post resource
	post := meta.Resources[0]
	if post.Name != "Post" {
		t.Errorf("Resource name = %v, want Post", post.Name)
	}
	if len(post.Fields) != 5 {
		t.Errorf("Post fields count = %v, want 5", len(post.Fields))
	}
	if len(post.Relationships) != 1 {
		t.Errorf("Post relationships count = %v, want 1", len(post.Relationships))
	}
	if len(post.Hooks) != 2 {
		t.Errorf("Post hooks count = %v, want 2", len(post.Hooks))
	}

	// Check Comment resource
	comment := meta.Resources[1]
	if comment.Name != "Comment" {
		t.Errorf("Resource name = %v, want Comment", comment.Name)
	}

	// Step 4: Generate JSON
	jsonStr, err := meta.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Step 5: Verify JSON is valid and parseable
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Generated invalid JSON: %v", err)
	}

	// Step 6: Test code generation integration
	gen := codegen.NewGenerator()
	files, err := gen.GenerateProgram(prog, "test-app", "")
	if err != nil {
		t.Fatalf("GenerateProgram failed: %v", err)
	}

	// Step 7: Verify metadata files are generated
	metadataJSON, ok := files["introspection/metadata.json"]
	if !ok {
		t.Fatal("metadata.json not generated")
	}

	introspectionGo, ok := files["introspection/introspection.go"]
	if !ok {
		t.Fatal("introspection.go not generated")
	}

	// Step 8: Verify metadata.json contains expected data
	var finalMeta map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &finalMeta); err != nil {
		t.Fatalf("Final metadata JSON invalid: %v", err)
	}

	resources := finalMeta["resources"].([]interface{})
	if len(resources) != 2 {
		t.Errorf("Final metadata resources count = %v, want 2", len(resources))
	}

	// Step 9: Verify introspection.go contains required functions
	requiredFunctions := []string{
		"func GetMetadata()",
		"func QueryResources()",
		"func QueryPatterns()",
		"func QueryRoutes()",
		"func FindResource(",
	}

	for _, fn := range requiredFunctions {
		if !containsString(introspectionGo, fn) {
			t.Errorf("introspection.go missing function: %s", fn)
		}
	}

	// Step 10: Verify patterns were detected
	patterns := finalMeta["patterns"].([]interface{})
	if len(patterns) == 0 {
		t.Error("Expected patterns to be detected")
	}

	// Step 11: Verify routes were generated
	routes := finalMeta["routes"].([]interface{})
	if len(routes) != 10 { // 5 routes per resource
		t.Errorf("Routes count = %v, want 10 (5 per resource)", len(routes))
	}

	t.Logf("✅ Complete workflow test passed")
	t.Logf("  - Resources extracted: %d", len(meta.Resources))
	t.Logf("  - Patterns detected: %d", len(patterns))
	t.Logf("  - Routes generated: %d", len(routes))
	t.Logf("  - JSON size: %d bytes", len(metadataJSON))
	t.Logf("  - Go accessor size: %d bytes", len(introspectionGo))
}

// TestMetadataRoundTrip verifies metadata can be serialized and deserialized
func TestMetadataRoundTrip(t *testing.T) {
	// Create original metadata
	original := &metadata.Metadata{
		Version: "1.0.0",
		Resources: []metadata.ResourceMetadata{
			{
				Name: "User",
				Fields: []metadata.FieldMetadata{
					{Name: "id", Type: "uuid!", Nullable: false},
					{Name: "email", Type: "string!", Nullable: false, Constraints: []string{"unique"}},
				},
				Hooks: []metadata.HookMetadata{
					{Timing: "after", Event: "create", HasTransaction: true},
				},
			},
		},
		Patterns: []metadata.PatternMetadata{
			{Name: "test_pattern", Template: "test", Occurrences: 1},
		},
		Routes: []metadata.RouteMetadata{
			{Method: "GET", Path: "/users", Handler: "User.list", Operation: "list", Resource: "User"},
		},
	}

	// Serialize to JSON
	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Deserialize from JSON
	recovered, err := metadata.FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// Verify structure is preserved
	if recovered.Version != original.Version {
		t.Errorf("Version not preserved: %v != %v", recovered.Version, original.Version)
	}

	if len(recovered.Resources) != len(original.Resources) {
		t.Errorf("Resources count not preserved: %v != %v", len(recovered.Resources), len(original.Resources))
	}

	if len(recovered.Patterns) != len(original.Patterns) {
		t.Errorf("Patterns count not preserved: %v != %v", len(recovered.Patterns), len(original.Patterns))
	}

	if len(recovered.Routes) != len(original.Routes) {
		t.Errorf("Routes count not preserved: %v != %v", len(recovered.Routes), len(original.Routes))
	}

	t.Log("✅ Round-trip serialization successful")
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Example_workflow demonstrates the complete metadata generation workflow
func Example_workflow() {
	// Create a simple program
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

	// Extract metadata
	extractor := metadata.NewExtractor("1.0.0")
	meta, _ := extractor.Extract(prog)

	// Generate code
	gen := codegen.NewGenerator()
	files, _ := gen.GenerateProgram(prog, "test-app", "")

	fmt.Printf("Generated %d files\n", len(files))
	fmt.Printf("Resources: %d\n", len(meta.Resources))
	fmt.Printf("Patterns: %d\n", len(meta.Patterns))

	// Output:
	// Generated 6 files
	// Resources: 1
	// Patterns: 0
}
