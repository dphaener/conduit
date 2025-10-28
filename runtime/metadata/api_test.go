package metadata

import (
	"encoding/json"
	"testing"
	"time"
)

// TestGetRegistry verifies that GetRegistry returns a non-nil API instance
func TestGetRegistry(t *testing.T) {
	registry := GetRegistry()
	if registry == nil {
		t.Fatal("GetRegistry() returned nil")
	}
}

// TestRegistryResources tests querying all resources
func TestRegistryResources(t *testing.T) {
	// Setup: Register test metadata
	setupTestMetadata(t)
	defer Reset()

	registry := GetRegistry()
	resources := registry.Resources()

	if len(resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(resources))
	}

	// Verify resource names
	names := make(map[string]bool)
	for _, res := range resources {
		names[res.Name] = true
	}
	if !names["Post"] || !names["User"] {
		t.Errorf("expected Post and User resources, got: %v", names)
	}
}

// TestRegistryResource tests querying a single resource by name
func TestRegistryResource(t *testing.T) {
	// Setup: Register test metadata
	setupTestMetadata(t)
	defer Reset()

	registry := GetRegistry()

	tests := []struct {
		name      string
		resource  string
		expectErr bool
		validate  func(*testing.T, *ResourceMetadata)
	}{
		{
			name:      "existing resource",
			resource:  "Post",
			expectErr: false,
			validate: func(t *testing.T, res *ResourceMetadata) {
				if res.Name != "Post" {
					t.Errorf("expected name Post, got %s", res.Name)
				}
				if len(res.Fields) != 3 {
					t.Errorf("expected 3 fields, got %d", len(res.Fields))
				}
			},
		},
		{
			name:      "non-existent resource",
			resource:  "NonExistent",
			expectErr: true,
		},
		{
			name:      "empty name",
			resource:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := registry.Resource(tt.resource)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, res)
				}
			}
		})
	}
}

// TestRegistryRoutes tests querying routes with various filters
func TestRegistryRoutes(t *testing.T) {
	// Setup: Register test metadata
	setupTestMetadata(t)
	defer Reset()

	registry := GetRegistry()

	tests := []struct {
		name     string
		filter   RouteFilter
		expected int
		validate func(*testing.T, []RouteMetadata)
	}{
		{
			name:     "all routes",
			filter:   RouteFilter{},
			expected: 5,
		},
		{
			name:     "filter by GET method",
			filter:   RouteFilter{Method: "GET"},
			expected: 2,
			validate: func(t *testing.T, routes []RouteMetadata) {
				for _, route := range routes {
					if route.Method != "GET" {
						t.Errorf("expected GET, got %s", route.Method)
					}
				}
			},
		},
		{
			name:     "filter by POST method",
			filter:   RouteFilter{Method: "POST"},
			expected: 2,
			validate: func(t *testing.T, routes []RouteMetadata) {
				for _, route := range routes {
					if route.Method != "POST" {
						t.Errorf("expected POST, got %s", route.Method)
					}
				}
			},
		},
		{
			name:     "filter by resource",
			filter:   RouteFilter{Resource: "Post"},
			expected: 4, // GET /posts, GET /posts/:id, POST /posts, DELETE /posts/:id
			validate: func(t *testing.T, routes []RouteMetadata) {
				for _, route := range routes {
					if route.Resource != "Post" {
						t.Errorf("expected Post resource, got %s", route.Resource)
					}
				}
			},
		},
		{
			name:     "filter by path",
			filter:   RouteFilter{Path: "/posts"},
			expected: 2,
			validate: func(t *testing.T, routes []RouteMetadata) {
				for _, route := range routes {
					if route.Path != "/posts" {
						t.Errorf("expected /posts path, got %s", route.Path)
					}
				}
			},
		},
		{
			name:     "filter by method and resource",
			filter:   RouteFilter{Method: "GET", Resource: "Post"},
			expected: 2,
			validate: func(t *testing.T, routes []RouteMetadata) {
				for _, route := range routes {
					if route.Method != "GET" {
						t.Errorf("expected GET, got %s", route.Method)
					}
					if route.Resource != "Post" {
						t.Errorf("expected Post resource, got %s", route.Resource)
					}
				}
			},
		},
		{
			name:     "filter with no matches",
			filter:   RouteFilter{Method: "DELETE", Resource: "NonExistent"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := registry.Routes(tt.filter)
			if len(routes) != tt.expected {
				t.Errorf("expected %d routes, got %d", tt.expected, len(routes))
			}
			if tt.validate != nil {
				tt.validate(t, routes)
			}
		})
	}
}

// TestRegistryPatterns tests querying patterns with category filtering
func TestRegistryPatterns(t *testing.T) {
	// Setup: Register test metadata
	setupTestMetadata(t)
	defer Reset()

	registry := GetRegistry()

	tests := []struct {
		name     string
		category string
		expected int
		validate func(*testing.T, []PatternMetadata)
	}{
		{
			name:     "all patterns",
			category: "",
			expected: 3,
		},
		{
			name:     "hook patterns",
			category: "hook",
			expected: 2,
			validate: func(t *testing.T, patterns []PatternMetadata) {
				for _, p := range patterns {
					if p.Category != "hook" {
						t.Errorf("expected hook category, got %s", p.Category)
					}
				}
			},
		},
		{
			name:     "validation patterns",
			category: "validation",
			expected: 1,
			validate: func(t *testing.T, patterns []PatternMetadata) {
				for _, p := range patterns {
					if p.Category != "validation" {
						t.Errorf("expected validation category, got %s", p.Category)
					}
				}
			},
		},
		{
			name:     "non-existent category",
			category: "nonexistent",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := registry.Patterns(tt.category)
			if len(patterns) != tt.expected {
				t.Errorf("expected %d patterns, got %d", tt.expected, len(patterns))
			}
			if tt.validate != nil {
				tt.validate(t, patterns)
			}
		})
	}
}

// TestRegistryDependencies tests querying dependency graphs
func TestRegistryDependencies(t *testing.T) {
	// Setup: Register test metadata
	setupTestMetadata(t)
	defer Reset()

	registry := GetRegistry()

	tests := []struct {
		name      string
		resource  string
		opts      DependencyOptions
		expectErr bool
		validate  func(*testing.T, *DependencyGraph)
	}{
		{
			name:      "forward dependencies unlimited depth",
			resource:  "Post",
			opts:      DependencyOptions{Depth: 0, Reverse: false},
			expectErr: false,
			validate: func(t *testing.T, graph *DependencyGraph) {
				if len(graph.Nodes) < 1 {
					t.Error("expected at least 1 node (Post itself)")
				}
				if _, exists := graph.Nodes["Post"]; !exists {
					t.Error("expected Post node in graph")
				}
			},
		},
		{
			name:      "reverse dependencies unlimited depth",
			resource:  "User",
			opts:      DependencyOptions{Depth: 0, Reverse: true},
			expectErr: false,
			validate: func(t *testing.T, graph *DependencyGraph) {
				if len(graph.Nodes) < 1 {
					t.Error("expected at least 1 node (User itself)")
				}
				if _, exists := graph.Nodes["User"]; !exists {
					t.Error("expected User node in graph")
				}
			},
		},
		{
			name:      "limited depth",
			resource:  "Post",
			opts:      DependencyOptions{Depth: 1, Reverse: false},
			expectErr: false,
			validate: func(t *testing.T, graph *DependencyGraph) {
				if _, exists := graph.Nodes["Post"]; !exists {
					t.Error("expected Post node in graph")
				}
			},
		},
		{
			name:      "filter by relationship type",
			resource:  "Post",
			opts:      DependencyOptions{Depth: 0, Reverse: false, Types: []string{"belongs_to"}},
			expectErr: false,
			validate: func(t *testing.T, graph *DependencyGraph) {
				for _, edge := range graph.Edges {
					if edge.Relationship != "belongs_to" {
						t.Errorf("expected only belongs_to edges, got %s", edge.Relationship)
					}
				}
			},
		},
		{
			name:      "non-existent resource",
			resource:  "NonExistent",
			opts:      DependencyOptions{Depth: 0, Reverse: false},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := registry.Dependencies(tt.resource, tt.opts)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, graph)
				}
			}
		})
	}
}

// TestRegistryGetSchema tests retrieving the complete schema
func TestRegistryGetSchema(t *testing.T) {
	// Setup: Register test metadata
	setupTestMetadata(t)
	defer Reset()

	registry := GetRegistry()
	schema := registry.GetSchema()

	if schema == nil {
		t.Fatal("GetSchema() returned nil")
	}

	if schema.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", schema.Version)
	}

	if len(schema.Resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(schema.Resources))
	}

	if len(schema.Routes) != 5 {
		t.Errorf("expected 5 routes, got %d", len(schema.Routes))
	}

	if len(schema.Patterns) != 3 {
		t.Errorf("expected 3 patterns, got %d", len(schema.Patterns))
	}
}

// TestRegistryUninitializedRegistry tests behavior when registry is not initialized
func TestRegistryUninitializedRegistry(t *testing.T) {
	// Ensure registry is reset
	Reset()

	registry := GetRegistry()

	// Resources() should return nil when not initialized
	resources := registry.Resources()
	if resources != nil {
		t.Error("expected nil resources from uninitialized registry")
	}

	// Resource() should return error when not initialized
	_, err := registry.Resource("Post")
	if err == nil {
		t.Error("expected error from uninitialized registry")
	}

	// Routes() should return nil when not initialized
	routes := registry.Routes(RouteFilter{})
	if routes != nil {
		t.Error("expected nil routes from uninitialized registry")
	}

	// Patterns() should return nil when not initialized
	patterns := registry.Patterns("")
	if patterns != nil {
		t.Error("expected nil patterns from uninitialized registry")
	}

	// Dependencies() should return error when not initialized
	_, err = registry.Dependencies("Post", DependencyOptions{})
	if err == nil {
		t.Error("expected error from uninitialized registry")
	}

	// GetSchema() should return nil when not initialized
	schema := registry.GetSchema()
	if schema != nil {
		t.Error("expected nil schema from uninitialized registry")
	}
}

// setupTestMetadata registers test metadata for use in tests
func setupTestMetadata(t *testing.T) {
	t.Helper()

	meta := Metadata{
		Version:    "1.0",
		Generated:  time.Now(),
		SourceHash: "test-hash",
		Resources: []ResourceMetadata{
			{
				Name:     "Post",
				FilePath: "/test/post.cdt",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid", Required: true},
					{Name: "title", Type: "string", Required: true},
					{Name: "content", Type: "text", Required: true},
				},
				Relationships: []RelationshipMetadata{
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
				FilePath: "/test/user.cdt",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid", Required: true},
					{Name: "email", Type: "string", Required: true},
				},
			},
		},
		Routes: []RouteMetadata{
			{Method: "GET", Path: "/posts", Resource: "Post", Operation: "list"},
			{Method: "GET", Path: "/posts/:id", Resource: "Post", Operation: "show"},
			{Method: "POST", Path: "/posts", Resource: "Post", Operation: "create"},
			{Method: "POST", Path: "/users", Resource: "User", Operation: "create"},
			{Method: "DELETE", Path: "/posts/:id", Resource: "Post", Operation: "delete"},
		},
		Patterns: []PatternMetadata{
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
	meta.Dependencies = *BuildDependencyGraph(&meta)

	// Marshal and register
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("failed to marshal test metadata: %v", err)
	}

	if err := RegisterMetadata(data); err != nil {
		t.Fatalf("failed to register test metadata: %v", err)
	}
}
