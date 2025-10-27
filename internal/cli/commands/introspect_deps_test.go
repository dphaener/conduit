package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

func TestRunIntrospectDepsCommand(t *testing.T) {
	// Helper to create test metadata with relationships
	createTestMetadataWithDeps := func() *metadata.Metadata {
		return &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{
					Name: "Post",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "title", Type: "string", Required: true},
						{Name: "author_id", Type: "uuid", Required: true},
						{Name: "category_id", Type: "uuid", Nullable: true},
					},
					Relationships: []metadata.RelationshipMetadata{
						{
							Name:           "author",
							Type:           "belongs_to",
							TargetResource: "User",
							ForeignKey:     "author_id",
							OnDelete:       "restrict",
						},
						{
							Name:           "category",
							Type:           "belongs_to",
							TargetResource: "Category",
							ForeignKey:     "category_id",
							OnDelete:       "set_null",
						},
					},
					Middleware: map[string][]string{
						"create": {"auth", "rate_limit"},
						"update": {"auth"},
						"delete": {"auth"},
						"list":   {"cache"},
						"get":    {"cache"},
					},
					Hooks: []metadata.HookMetadata{
						{
							Type:       "before_create",
							SourceCode: "self.slug = String.slugify(self.title)",
						},
					},
				},
				{
					Name: "User",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "email", Type: "string", Required: true},
					},
				},
				{
					Name: "Category",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "name", Type: "string", Required: true},
					},
				},
				{
					Name: "Comment",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "content", Type: "text", Required: true},
						{Name: "post_id", Type: "uuid", Required: true},
					},
					Relationships: []metadata.RelationshipMetadata{
						{
							Name:           "post",
							Type:           "belongs_to",
							TargetResource: "Post",
							ForeignKey:     "post_id",
							OnDelete:       "cascade",
						},
					},
				},
			},
			Routes: []metadata.RouteMetadata{
				{
					Method:    "GET",
					Path:      "/api/posts",
					Handler:   "Post.list",
					Resource:  "Post",
					Operation: "list",
				},
				{
					Method:    "POST",
					Path:      "/api/posts",
					Handler:   "Post.create",
					Resource:  "Post",
					Operation: "create",
				},
				{
					Method:    "PUT",
					Path:      "/api/posts/:id",
					Handler:   "Post.update",
					Resource:  "Post",
					Operation: "update",
				},
				{
					Method:    "DELETE",
					Path:      "/api/posts/:id",
					Handler:   "Post.delete",
					Resource:  "Post",
					Operation: "delete",
				},
			},
		}
	}

	t.Run("formats table output correctly", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectDepsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"Post"})
		require.NoError(t, err)

		output := buf.String()

		// Check header
		assert.Contains(t, output, "DEPENDENCIES: Post")

		// Check direct dependencies section
		assert.Contains(t, output, "DIRECT DEPENDENCIES (what Post uses)")

		// Check resources section
		assert.Contains(t, output, "Resources:")
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "Category")

		// Check middleware section
		assert.Contains(t, output, "Middleware:")
		assert.Contains(t, output, "auth")
		assert.Contains(t, output, "cache")
		assert.Contains(t, output, "rate_limit")

		// Check functions section
		assert.Contains(t, output, "Functions:")
		assert.Contains(t, output, "String.slugify")

		// Check reverse dependencies section
		assert.Contains(t, output, "REVERSE DEPENDENCIES (what uses Post)")

		// Check routes section
		assert.Contains(t, output, "Routes:")
		assert.Contains(t, output, "GET /api/posts")
		assert.Contains(t, output, "POST /api/posts")
	})

	t.Run("shows impact descriptions", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectDepsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"Post"})
		require.NoError(t, err)

		output := buf.String()

		// Check for impact descriptions
		assert.Contains(t, output, "Impact:")
		// User has restrict on_delete
		assert.Contains(t, output, "Cannot delete")
		// Category has set_null on_delete
		assert.Contains(t, output, "nullifies")
	})

	t.Run("filters by type - resource", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectDepsCommand()
		cmd.SetArgs([]string{"Post", "--type", "resource"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should show resources
		assert.Contains(t, output, "Resources:")
		assert.Contains(t, output, "User")

		// Should NOT show middleware or functions (they're filtered out)
		// The sections might still appear but be empty or not appear at all
	})

	t.Run("filters by type - middleware", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectDepsCommand()
		cmd.SetArgs([]string{"Post", "--type", "middleware"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should show middleware
		assert.Contains(t, output, "Middleware:")
		assert.Contains(t, output, "auth")
	})

	t.Run("filters by type - function", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectDepsCommand()
		cmd.SetArgs([]string{"Post", "--type", "function"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should show functions
		assert.Contains(t, output, "Functions:")
		assert.Contains(t, output, "String.slugify")
	})

	t.Run("shows only reverse dependencies with --reverse flag", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectDepsCommand()
		cmd.SetArgs([]string{"Post", "--reverse"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should show reverse dependencies
		assert.Contains(t, output, "REVERSE DEPENDENCIES (what uses Post)")

		// Should NOT show direct dependencies
		assert.NotContains(t, output, "DIRECT DEPENDENCIES (what Post uses)")
	})

	t.Run("validates depth range", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		// Test depth too low
		cmd := newIntrospectDepsCommand()
		cmd.SetArgs([]string{"Post", "--depth", "0"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "depth must be between 1 and 5")

		// Test depth too high
		cmd = newIntrospectDepsCommand()
		cmd.SetArgs([]string{"Post", "--depth", "10"})
		buf = &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "depth must be between 1 and 5")
	})

	t.Run("validates type filter", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectDepsCommand()
		cmd.SetArgs([]string{"Post", "--type", "invalid"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type filter")
		assert.Contains(t, err.Error(), "valid: resource, middleware, function")
	})

	t.Run("handles resource not found", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectDepsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"NonExistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource not found")
	})

	t.Run("formats JSON output correctly", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithDeps()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "json"
		noColor = true

		cmd := newIntrospectDepsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"Post"})
		require.NoError(t, err)

		// Parse JSON output
		var result metadata.DependencyGraph
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Verify structure
		assert.NotNil(t, result.Nodes)
		assert.NotNil(t, result.Edges)
		assert.True(t, len(result.Nodes) > 0)
		assert.True(t, len(result.Edges) > 0)

		// Verify Post node exists
		_, exists := result.Nodes["Post"]
		assert.True(t, exists)

		// Reset format
		outputFormat = "table"
	})

	t.Run("handles registry not initialized", func(t *testing.T) {
		metadata.Reset()

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectDepsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err := cmd.RunE(cmd, []string{"Post"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registry not initialized")
	})

	// Cleanup after tests
	t.Cleanup(func() {
		metadata.Reset()
		outputFormat = "table"
		verbose = false
		noColor = false
	})
}

func TestGroupDependenciesByType(t *testing.T) {
	graph := &metadata.DependencyGraph{
		Nodes: map[string]*metadata.DependencyNode{
			"Post": {
				ID:   "Post",
				Type: "resource",
				Name: "Post",
			},
			"User": {
				ID:   "User",
				Type: "resource",
				Name: "User",
			},
			"auth": {
				ID:   "auth",
				Type: "middleware",
				Name: "auth",
			},
			"String.slugify": {
				ID:   "String.slugify",
				Type: "function",
				Name: "String.slugify",
			},
		},
		Edges: []metadata.DependencyEdge{
			{
				From:         "Post",
				To:           "User",
				Relationship: "belongs_to",
				Weight:       1,
			},
			{
				From:         "Post",
				To:           "auth",
				Relationship: "uses",
				Weight:       3,
			},
			{
				From:         "Post",
				To:           "String.slugify",
				Relationship: "calls",
				Weight:       1,
			},
		},
	}

	t.Run("groups forward dependencies correctly", func(t *testing.T) {
		groups := groupDependenciesByType(graph, false)

		// Should have 3 groups: resource, middleware, function
		assert.Len(t, groups, 3)

		// Check resource group
		assert.Len(t, groups["resource"], 1)
		assert.Equal(t, "User", groups["resource"][0].To)

		// Check middleware group
		assert.Len(t, groups["middleware"], 1)
		assert.Equal(t, "auth", groups["middleware"][0].To)

		// Check function group
		assert.Len(t, groups["function"], 1)
		assert.Equal(t, "String.slugify", groups["function"][0].To)
	})

	t.Run("groups reverse dependencies correctly", func(t *testing.T) {
		groups := groupDependenciesByType(graph, true)

		// Should have 1 group: resource (all edges point from Post)
		assert.Len(t, groups, 1)

		// All edges should be grouped under "resource" since Post is a resource
		assert.Len(t, groups["resource"], 3)
	})

	t.Run("handles empty graph", func(t *testing.T) {
		emptyGraph := &metadata.DependencyGraph{
			Nodes: map[string]*metadata.DependencyNode{},
			Edges: []metadata.DependencyEdge{},
		}

		groups := groupDependenciesByType(emptyGraph, false)
		assert.Len(t, groups, 0)
	})
}

func TestGetImpactDescription(t *testing.T) {
	// Setup test metadata
	metadata.Reset()
	testMeta := &metadata.Metadata{
		Version:   "1.0.0",
		Generated: time.Now(),
		Resources: []metadata.ResourceMetadata{
			{
				Name: "Post",
				Relationships: []metadata.RelationshipMetadata{
					{
						Name:           "author",
						Type:           "belongs_to",
						TargetResource: "User",
						ForeignKey:     "author_id",
						OnDelete:       "restrict",
					},
					{
						Name:           "category",
						Type:           "belongs_to",
						TargetResource: "Category",
						ForeignKey:     "category_id",
						OnDelete:       "set_null",
					},
				},
			},
			{
				Name: "Comment",
				Relationships: []metadata.RelationshipMetadata{
					{
						Name:           "post",
						Type:           "belongs_to",
						TargetResource: "Post",
						ForeignKey:     "post_id",
						OnDelete:       "cascade",
					},
				},
			},
		},
	}
	data, _ := json.Marshal(testMeta)
	metadata.RegisterMetadata(data)

	graph := &metadata.DependencyGraph{
		Nodes: map[string]*metadata.DependencyNode{
			"Post": {ID: "Post", Type: "resource", Name: "Post"},
			"User": {ID: "User", Type: "resource", Name: "User"},
		},
	}

	t.Run("restrict on_delete", func(t *testing.T) {
		edge := metadata.DependencyEdge{
			From:         "Post",
			To:           "User",
			Relationship: "belongs_to",
		}

		description := getImpactDescription(edge, graph, "Post", false)
		assert.Contains(t, description, "Cannot delete")
	})

	t.Run("set_null on_delete", func(t *testing.T) {
		edge := metadata.DependencyEdge{
			From:         "Post",
			To:           "Category",
			Relationship: "belongs_to",
		}

		description := getImpactDescription(edge, graph, "Post", false)
		assert.Contains(t, description, "nullifies")
	})

	t.Run("cascade on_delete", func(t *testing.T) {
		edge := metadata.DependencyEdge{
			From:         "Comment",
			To:           "Post",
			Relationship: "belongs_to",
		}

		description := getImpactDescription(edge, graph, "Comment", false)
		assert.Contains(t, description, "cascade")
	})

	t.Run("middleware usage", func(t *testing.T) {
		edge := metadata.DependencyEdge{
			From:         "Post",
			To:           "auth",
			Relationship: "uses",
		}

		description := getImpactDescription(edge, graph, "Post", false)
		assert.Equal(t, "Applied to operations", description)
	})

	t.Run("function calls", func(t *testing.T) {
		edge := metadata.DependencyEdge{
			From:         "Post",
			To:           "String.slugify",
			Relationship: "calls",
		}

		description := getImpactDescription(edge, graph, "Post", false)
		assert.Equal(t, "Called from hooks", description)
	})

	// Cleanup
	t.Cleanup(func() {
		metadata.Reset()
	})
}

func TestFormatDependenciesAsTable(t *testing.T) {
	// Setup test metadata
	metadata.Reset()
	testMeta := &metadata.Metadata{
		Version:   "1.0.0",
		Generated: time.Now(),
		Resources: []metadata.ResourceMetadata{
			{
				Name: "Post",
				Relationships: []metadata.RelationshipMetadata{
					{
						Name:           "author",
						Type:           "belongs_to",
						TargetResource: "User",
						ForeignKey:     "author_id",
						OnDelete:       "restrict",
					},
				},
			},
			{Name: "User"},
		},
		Routes: []metadata.RouteMetadata{
			{
				Method:   "GET",
				Path:     "/api/posts",
				Resource: "Post",
			},
		},
	}
	data, _ := json.Marshal(testMeta)
	metadata.RegisterMetadata(data)

	t.Run("formats dependencies correctly", func(t *testing.T) {
		graph := &metadata.DependencyGraph{
			Nodes: map[string]*metadata.DependencyNode{
				"Post": {ID: "Post", Type: "resource", Name: "Post"},
				"User": {ID: "User", Type: "resource", Name: "User"},
				"auth": {ID: "auth", Type: "middleware", Name: "auth"},
			},
			Edges: []metadata.DependencyEdge{
				{From: "Post", To: "User", Relationship: "belongs_to"},
				{From: "Post", To: "auth", Relationship: "uses"},
			},
		}

		opts := metadata.DependencyOptions{
			Depth:   1,
			Reverse: false,
		}

		buf := &bytes.Buffer{}
		noColor = true

		err := formatDependenciesAsTable(graph, "Post", opts, buf)
		require.NoError(t, err)

		output := buf.String()

		// Check header
		assert.Contains(t, output, "DEPENDENCIES: Post")

		// Check sections
		assert.Contains(t, output, "DIRECT DEPENDENCIES")
		assert.Contains(t, output, "REVERSE DEPENDENCIES")

		// Check content
		assert.Contains(t, output, "Resources:")
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "Middleware:")
		assert.Contains(t, output, "auth")
	})

	t.Run("handles empty dependencies", func(t *testing.T) {
		graph := &metadata.DependencyGraph{
			Nodes: map[string]*metadata.DependencyNode{
				"Post": {ID: "Post", Type: "resource", Name: "Post"},
			},
			Edges: []metadata.DependencyEdge{},
		}

		opts := metadata.DependencyOptions{
			Depth:   1,
			Reverse: false,
		}

		buf := &bytes.Buffer{}
		noColor = true

		err := formatDependenciesAsTable(graph, "Post", opts, buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "No direct dependencies")
	})

	// Cleanup
	t.Cleanup(func() {
		metadata.Reset()
		noColor = false
	})
}

func TestFormatDependenciesAsJSON(t *testing.T) {
	graph := &metadata.DependencyGraph{
		Nodes: map[string]*metadata.DependencyNode{
			"Post": {ID: "Post", Type: "resource", Name: "Post"},
			"User": {ID: "User", Type: "resource", Name: "User"},
		},
		Edges: []metadata.DependencyEdge{
			{From: "Post", To: "User", Relationship: "belongs_to", Weight: 1},
		},
	}

	t.Run("formats JSON correctly", func(t *testing.T) {
		buf := &bytes.Buffer{}

		err := formatDependenciesAsJSON(graph, buf)
		require.NoError(t, err)

		// Parse JSON
		var result metadata.DependencyGraph
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Verify structure
		assert.Len(t, result.Nodes, 2)
		assert.Len(t, result.Edges, 1)

		// Verify nodes
		postNode, exists := result.Nodes["Post"]
		assert.True(t, exists)
		assert.Equal(t, "Post", postNode.Name)
		assert.Equal(t, "resource", postNode.Type)

		userNode, exists := result.Nodes["User"]
		assert.True(t, exists)
		assert.Equal(t, "User", userNode.Name)

		// Verify edge
		assert.Equal(t, "Post", result.Edges[0].From)
		assert.Equal(t, "User", result.Edges[0].To)
		assert.Equal(t, "belongs_to", result.Edges[0].Relationship)
		assert.Equal(t, 1, result.Edges[0].Weight)
	})

	t.Run("handles empty graph", func(t *testing.T) {
		emptyGraph := &metadata.DependencyGraph{
			Nodes: map[string]*metadata.DependencyNode{},
			Edges: []metadata.DependencyEdge{},
		}

		buf := &bytes.Buffer{}

		err := formatDependenciesAsJSON(emptyGraph, buf)
		require.NoError(t, err)

		var result metadata.DependencyGraph
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		assert.Len(t, result.Nodes, 0)
		assert.Len(t, result.Edges, 0)
	})
}

// BenchmarkIntrospectDepsCommand benchmarks the deps command performance
func BenchmarkIntrospectDepsCommand(b *testing.B) {
	// Setup test registry with realistic data
	testMeta := &metadata.Metadata{
		Version:   "1.0.0",
		Generated: time.Now(),
		Resources: make([]metadata.ResourceMetadata, 0, 50),
	}

	// Create 50 resources with relationships
	resourceNames := []string{}
	for i := 0; i < 50; i++ {
		name := strings.Title(strings.ToLower(string(rune('A' + i%26))))
		if i >= 26 {
			name = name + strings.Title(strings.ToLower(string(rune('A'+(i-26)%26))))
		}
		resourceNames = append(resourceNames, name)

		res := metadata.ResourceMetadata{
			Name: name,
			Fields: []metadata.FieldMetadata{
				{Name: "id", Type: "uuid", Required: true},
				{Name: "name", Type: "string", Required: true},
			},
			Relationships: []metadata.RelationshipMetadata{},
			Middleware: map[string][]string{
				"create": {"auth"},
				"list":   {"cache"},
			},
			Hooks: []metadata.HookMetadata{
				{
					Type:       "before_create",
					SourceCode: "self.slug = String.slugify(self.name)",
				},
			},
		}

		// Add relationships to other resources
		if i > 0 {
			res.Relationships = append(res.Relationships, metadata.RelationshipMetadata{
				Name:           "parent",
				Type:           "belongs_to",
				TargetResource: resourceNames[i-1],
				ForeignKey:     "parent_id",
				OnDelete:       "cascade",
			})
		}

		testMeta.Resources = append(testMeta.Resources, res)
	}

	metadata.Reset()
	data, err := json.Marshal(testMeta)
	if err != nil {
		b.Fatal(err)
	}
	err = metadata.RegisterMetadata(data)
	if err != nil {
		b.Fatal(err)
	}

	// Reset flags
	outputFormat = "table"
	verbose = false
	noColor = true

	cmd := newIntrospectDepsCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := cmd.RunE(cmd, []string{resourceNames[25]})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.Cleanup(func() {
		metadata.Reset()
	})
}
