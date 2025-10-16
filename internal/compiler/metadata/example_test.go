package metadata_test

import (
	"encoding/json"
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/metadata"
)

// ExampleExtractor_Extract demonstrates extracting metadata from an AST
func ExampleExtractor_Extract() {
	// Create a simple program with a User resource
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
						Constraints: []*ast.ConstraintNode{
							{Name: "unique"},
						},
					},
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
						Timing:        "after",
						Event:         "create",
						IsTransaction: true,
						Middleware:    []string{"auth"},
					},
				},
			},
		},
	}

	// Extract metadata
	extractor := metadata.NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Convert to JSON
	jsonStr, err := meta.ToJSON()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Pretty print the JSON (simplified for example)
	var prettyJSON map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &prettyJSON)

	fmt.Printf("Version: %s\n", prettyJSON["version"])
	resources := prettyJSON["resources"].([]interface{})
	fmt.Printf("Resources: %d\n", len(resources))

	resource := resources[0].(map[string]interface{})
	fmt.Printf("Resource Name: %s\n", resource["name"])

	fields := resource["fields"].([]interface{})
	fmt.Printf("Fields: %d\n", len(fields))

	hooks := resource["hooks"].([]interface{})
	fmt.Printf("Hooks: %d\n", len(hooks))

	patterns := prettyJSON["patterns"].([]interface{})
	fmt.Printf("Patterns: %d\n", len(patterns))

	routes := prettyJSON["routes"].([]interface{})
	fmt.Printf("Routes: %d\n", len(routes))

	// Output:
	// Version: 1.0.0
	// Resources: 1
	// Resource Name: User
	// Fields: 3
	// Hooks: 1
	// Patterns: 3
	// Routes: 5
}

// ExampleMetadata_ToJSON demonstrates converting metadata to JSON
func ExampleMetadata_ToJSON() {
	meta := &metadata.Metadata{
		Version: "1.0.0",
		Resources: []metadata.ResourceMetadata{
			{
				Name: "Post",
				Fields: []metadata.FieldMetadata{
					{Name: "title", Type: "string!", Nullable: false},
					{Name: "content", Type: "text!", Nullable: false},
				},
			},
		},
		Patterns: []metadata.PatternMetadata{},
		Routes:   []metadata.RouteMetadata{},
	}

	jsonStr, err := meta.ToJSON()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Check if valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		fmt.Printf("Invalid JSON: %v\n", err)
		return
	}

	fmt.Println("Valid JSON generated")

	// Output:
	// Valid JSON generated
}
