package metadata_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// ExampleGetRegistry demonstrates basic usage of the registry API
func ExampleGetRegistry() {
	// Setup: Register test metadata
	setupExampleMetadata()
	defer metadata.Reset()

	// Get the registry singleton
	registry := metadata.GetRegistry()

	// Query all resources
	resources := registry.Resources()
	fmt.Printf("Total resources: %d\n", len(resources))

	// Query a single resource
	post, err := registry.Resource("Post")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Post has %d fields\n", len(post.Fields))

	// Output:
	// Total resources: 2
	// Post has 3 fields
}

// ExampleRegistryAPI_Resources demonstrates querying all resources
func ExampleRegistryAPI_Resources() {
	// Setup: Register test metadata
	setupExampleMetadata()
	defer metadata.Reset()

	registry := metadata.GetRegistry()
	resources := registry.Resources()

	for _, res := range resources {
		fmt.Printf("%s: %d fields, %d relationships\n",
			res.Name, len(res.Fields), len(res.Relationships))
	}

	// Output:
	// Post: 3 fields, 1 relationships
	// User: 2 fields, 0 relationships
}

// ExampleRegistryAPI_Resource demonstrates querying a single resource
func ExampleRegistryAPI_Resource() {
	// Setup: Register test metadata
	setupExampleMetadata()
	defer metadata.Reset()

	registry := metadata.GetRegistry()
	post, err := registry.Resource("Post")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Resource: %s\n", post.Name)
	fmt.Printf("Fields:\n")
	for _, field := range post.Fields {
		fmt.Printf("  - %s: %s\n", field.Name, field.Type)
	}

	// Output:
	// Resource: Post
	// Fields:
	//   - id: uuid
	//   - title: string
	//   - content: text
}

// ExampleRegistryAPI_Routes demonstrates querying routes with filters
func ExampleRegistryAPI_Routes() {
	// Setup: Register test metadata
	setupExampleMetadata()
	defer metadata.Reset()

	registry := metadata.GetRegistry()

	// Get all GET routes
	routes := registry.Routes(metadata.RouteFilter{Method: "GET"})
	fmt.Printf("GET routes: %d\n", len(routes))

	// Get all routes for Post resource
	routes = registry.Routes(metadata.RouteFilter{Resource: "Post"})
	fmt.Printf("Post routes: %d\n", len(routes))

	// Get specific route
	routes = registry.Routes(metadata.RouteFilter{
		Method:   "GET",
		Resource: "Post",
	})
	for _, route := range routes {
		fmt.Printf("%s %s -> %s\n", route.Method, route.Path, route.Operation)
	}

	// Output:
	// GET routes: 2
	// Post routes: 3
	// GET /posts -> list
	// GET /posts/:id -> show
}

// ExampleRegistryAPI_Patterns demonstrates querying patterns by category
func ExampleRegistryAPI_Patterns() {
	// Setup: Register test metadata
	setupExampleMetadata()
	defer metadata.Reset()

	registry := metadata.GetRegistry()

	// Get all hook patterns
	patterns := registry.Patterns("hook")
	fmt.Printf("Hook patterns: %d\n", len(patterns))

	for _, p := range patterns {
		fmt.Printf("  - %s (used %d times)\n", p.Name, p.Frequency)
	}

	// Output:
	// Hook patterns: 2
	//   - slug-generation (used 5 times)
	//   - timestamp-tracking (used 10 times)
}

// ExampleRegistryAPI_Dependencies demonstrates querying dependency graphs
func ExampleRegistryAPI_Dependencies() {
	// Setup: Register test metadata
	setupExampleMetadata()
	defer metadata.Reset()

	registry := metadata.GetRegistry()

	// Get forward dependencies (what Post depends on)
	deps, err := registry.Dependencies("Post", metadata.DependencyOptions{
		Depth:   0,
		Reverse: false,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Post depends on %d nodes\n", len(deps.Nodes)-1)

	// Get reverse dependencies (what depends on User)
	deps, err = registry.Dependencies("User", metadata.DependencyOptions{
		Depth:   0,
		Reverse: true,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("%d nodes depend on User\n", len(deps.Nodes)-1)

	// Output:
	// Post depends on 1 nodes
	// 1 nodes depend on User
}

// ExampleRegistryAPI_GetSchema demonstrates retrieving the complete schema
func ExampleRegistryAPI_GetSchema() {
	// Setup: Register test metadata
	setupExampleMetadata()
	defer metadata.Reset()

	registry := metadata.GetRegistry()
	schema := registry.GetSchema()

	fmt.Printf("Schema version: %s\n", schema.Version)
	fmt.Printf("Resources: %d\n", len(schema.Resources))
	fmt.Printf("Routes: %d\n", len(schema.Routes))
	fmt.Printf("Patterns: %d\n", len(schema.Patterns))

	// Output:
	// Schema version: 1.0
	// Resources: 2
	// Routes: 3
	// Patterns: 3
}

// Example_buildingIntrospectionTool demonstrates building a complete introspection tool
func Example_buildingIntrospectionTool() {
	// Setup: Register test metadata
	setupExampleMetadata()
	defer metadata.Reset()

	// This example shows how to build a simple introspection tool
	// that analyzes the application schema

	registry := metadata.GetRegistry()

	// 1. Get overview statistics
	schema := registry.GetSchema()
	fmt.Println("=== Application Overview ===")
	fmt.Printf("Resources: %d\n", len(schema.Resources))
	fmt.Printf("Routes: %d\n", len(schema.Routes))
	fmt.Printf("Patterns: %d\n", len(schema.Patterns))
	fmt.Println()

	// 2. Analyze resources
	fmt.Println("=== Resource Details ===")
	resources := registry.Resources()
	for _, res := range resources {
		fmt.Printf("%s:\n", res.Name)
		fmt.Printf("  Fields: %d\n", len(res.Fields))
		fmt.Printf("  Relationships: %d\n", len(res.Relationships))

		// Get dependency information
		deps, err := registry.Dependencies(res.Name, metadata.DependencyOptions{
			Depth:   1,
			Reverse: false,
		})
		if err == nil {
			fmt.Printf("  Direct dependencies: %d\n", len(deps.Nodes)-1)
		}
	}
	fmt.Println()

	// 3. Analyze routes
	fmt.Println("=== Route Analysis ===")
	getMethods := registry.Routes(metadata.RouteFilter{Method: "GET"})
	postMethods := registry.Routes(metadata.RouteFilter{Method: "POST"})
	fmt.Printf("GET endpoints: %d\n", len(getMethods))
	fmt.Printf("POST endpoints: %d\n", len(postMethods))
	fmt.Println()

	// 4. Analyze patterns
	fmt.Println("=== Pattern Discovery ===")
	hookPatterns := registry.Patterns("hook")
	validationPatterns := registry.Patterns("validation")
	fmt.Printf("Hook patterns: %d\n", len(hookPatterns))
	fmt.Printf("Validation patterns: %d\n", len(validationPatterns))

	// Output:
	// === Application Overview ===
	// Resources: 2
	// Routes: 3
	// Patterns: 3
	//
	// === Resource Details ===
	// Post:
	//   Fields: 3
	//   Relationships: 1
	//   Direct dependencies: 1
	// User:
	//   Fields: 2
	//   Relationships: 0
	//   Direct dependencies: 0
	//
	// === Route Analysis ===
	// GET endpoints: 2
	// POST endpoints: 1
	//
	// === Pattern Discovery ===
	// Hook patterns: 2
	// Validation patterns: 1
}

// setupExampleMetadata registers test metadata for examples
func setupExampleMetadata() {
	meta := metadata.Metadata{
		Version:    "1.0",
		Generated:  time.Now(),
		SourceHash: "example-hash",
		Resources: []metadata.ResourceMetadata{
			{
				Name:     "Post",
				FilePath: "/example/post.cdt",
				Fields: []metadata.FieldMetadata{
					{Name: "id", Type: "uuid", Required: true},
					{Name: "title", Type: "string", Required: true},
					{Name: "content", Type: "text", Required: true},
				},
				Relationships: []metadata.RelationshipMetadata{
					{
						Name:           "author",
						Type:           "belongs_to",
						TargetResource: "User",
						ForeignKey:     "author_id",
					},
				},
			},
			{
				Name:     "User",
				FilePath: "/example/user.cdt",
				Fields: []metadata.FieldMetadata{
					{Name: "id", Type: "uuid", Required: true},
					{Name: "email", Type: "string", Required: true},
				},
			},
		},
		Routes: []metadata.RouteMetadata{
			{Method: "GET", Path: "/posts", Resource: "Post", Operation: "list"},
			{Method: "GET", Path: "/posts/:id", Resource: "Post", Operation: "show"},
			{Method: "POST", Path: "/posts", Resource: "Post", Operation: "create"},
		},
		Patterns: []metadata.PatternMetadata{
			{
				ID:          "pattern-1",
				Name:        "slug-generation",
				Category:    "hook",
				Description: "Generate slug from title",
				Frequency:   5,
			},
			{
				ID:          "pattern-2",
				Name:        "timestamp-tracking",
				Category:    "hook",
				Description: "Track created_at and updated_at",
				Frequency:   10,
			},
			{
				ID:          "pattern-3",
				Name:        "email-validation",
				Category:    "validation",
				Description: "Validate email format",
				Frequency:   3,
			},
		},
	}

	// Build dependency graph
	meta.Dependencies = *metadata.BuildDependencyGraph(&meta)

	// Marshal and register
	data, _ := json.Marshal(meta)
	_ = metadata.RegisterMetadata(data)
}
