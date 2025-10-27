package metadata

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestBuildDependencyGraph(t *testing.T) {
	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
					{Name: "category", TargetResource: "Category", Type: "belongs_to"},
				},
			},
			{
				Name: "Comment",
				Relationships: []RelationshipMetadata{
					{Name: "post", TargetResource: "Post", Type: "belongs_to"},
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "User",
			},
			{
				Name: "Category",
			},
		},
	}

	graph := BuildDependencyGraph(meta)

	// Verify all resources are nodes
	if len(graph.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(graph.Nodes))
	}

	// Verify resource nodes exist
	for _, name := range []string{"Post", "Comment", "User", "Category"} {
		if node, ok := graph.Nodes[name]; !ok {
			t.Errorf("Missing node: %s", name)
		} else if node.Type != "resource" {
			t.Errorf("Node %s has wrong type: %s", name, node.Type)
		}
	}

	// Verify edges (4 relationships)
	if len(graph.Edges) != 4 {
		t.Errorf("Expected 4 edges, got %d", len(graph.Edges))
	}

	// Verify specific edge
	foundPostToUser := false
	for _, edge := range graph.Edges {
		if edge.From == "Post" && edge.To == "User" && edge.Relationship == "belongs_to" {
			foundPostToUser = true
			break
		}
	}
	if !foundPostToUser {
		t.Error("Missing edge from Post to User")
	}
}

func TestQueryDependencies_Forward(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "User",
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Query forward dependencies (what Post depends on)
	opts := DependencyOptions{
		Depth:   1,
		Reverse: false,
	}

	graph, err := QueryDependencies("Post", opts)
	if err != nil {
		t.Fatalf("QueryDependencies failed: %v", err)
	}

	// Should have Post and User nodes
	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
	}

	// Should have edge from Post to User
	if len(graph.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(graph.Edges))
	}

	if graph.Edges[0].From != "Post" || graph.Edges[0].To != "User" {
		t.Errorf("Wrong edge: %s -> %s", graph.Edges[0].From, graph.Edges[0].To)
	}
}

func TestQueryDependencies_Reverse(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "Comment",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "User",
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Query reverse dependencies (what depends on User)
	opts := DependencyOptions{
		Depth:   1,
		Reverse: true,
	}

	graph, err := QueryDependencies("User", opts)
	if err != nil {
		t.Fatalf("QueryDependencies failed: %v", err)
	}

	// Should have User, Post, and Comment nodes
	if len(graph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(graph.Nodes))
	}

	// Should have 2 edges pointing to User
	if len(graph.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(graph.Edges))
	}

	// Verify both edges point to User
	for _, edge := range graph.Edges {
		if edge.To != "User" {
			t.Errorf("Edge should point to User, got: %s -> %s", edge.From, edge.To)
		}
	}
}

func TestQueryDependencies_DepthLimit(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Comment",
				Relationships: []RelationshipMetadata{
					{Name: "post", TargetResource: "Post", Type: "belongs_to"},
				},
			},
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "User",
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Query with depth=1 (should only get Post, not User)
	opts := DependencyOptions{
		Depth:   1,
		Reverse: false,
	}

	graph, err := QueryDependencies("Comment", opts)
	if err != nil {
		t.Fatalf("QueryDependencies failed: %v", err)
	}

	// Should have Comment and Post, but not User (depth limit)
	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes with depth=1, got %d", len(graph.Nodes))
	}

	if _, hasUser := graph.Nodes["User"]; hasUser {
		t.Error("Should not include User with depth=1")
	}

	// Query with depth=2 (should get all three)
	opts.Depth = 2
	graph, err = QueryDependencies("Comment", opts)
	if err != nil {
		t.Fatalf("QueryDependencies failed: %v", err)
	}

	if len(graph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes with depth=2, got %d", len(graph.Nodes))
	}

	if _, hasUser := graph.Nodes["User"]; !hasUser {
		t.Error("Should include User with depth=2")
	}
}

func TestQueryDependencies_TypeFilter(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
					{Name: "tags", TargetResource: "Tag", Type: "has_many"},
				},
			},
			{
				Name: "User",
			},
			{
				Name: "Tag",
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Query only belongs_to relationships
	opts := DependencyOptions{
		Depth:   1,
		Reverse: false,
		Types:   []string{"belongs_to"},
	}

	graph, err := QueryDependencies("Post", opts)
	if err != nil {
		t.Fatalf("QueryDependencies failed: %v", err)
	}

	// Should only have edge to User, not Tag
	if len(graph.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(graph.Edges))
	}

	if graph.Edges[0].To != "User" {
		t.Errorf("Expected edge to User, got edge to %s", graph.Edges[0].To)
	}

	// Should have Post and User nodes, but not Tag
	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
	}

	if _, hasTag := graph.Nodes["Tag"]; hasTag {
		t.Error("Should not include Tag when filtering by belongs_to")
	}
}

func TestQueryDependencies_NotFound(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version:   "1.0.0",
		Resources: []ResourceMetadata{{Name: "User"}},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	opts := DependencyOptions{Depth: 1}
	_, err := QueryDependencies("NonExistent", opts)
	if err == nil {
		t.Error("Expected error for non-existent resource")
	}
}

func TestDetectCycles(t *testing.T) {
	// Create a graph with a cycle: A -> B -> C -> A
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"A": {ID: "A", Type: "resource", Name: "A"},
			"B": {ID: "B", Type: "resource", Name: "B"},
			"C": {ID: "C", Type: "resource", Name: "C"},
		},
		Edges: []DependencyEdge{
			{From: "A", To: "B", Relationship: "depends_on"},
			{From: "B", To: "C", Relationship: "depends_on"},
			{From: "C", To: "A", Relationship: "depends_on"},
		},
	}

	cycles := DetectCycles(graph)
	if len(cycles) == 0 {
		t.Error("Expected to detect cycles")
	}

	// Verify the cycle contains A, B, C
	if len(cycles) > 0 {
		cycle := cycles[0]
		if len(cycle) < 3 {
			t.Errorf("Expected cycle of length >= 3, got %d", len(cycle))
		}
	}
}

func TestDetectCycles_NoCycle(t *testing.T) {
	// Create a graph without cycles: A -> B -> C
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"A": {ID: "A", Type: "resource", Name: "A"},
			"B": {ID: "B", Type: "resource", Name: "B"},
			"C": {ID: "C", Type: "resource", Name: "C"},
		},
		Edges: []DependencyEdge{
			{From: "A", To: "B", Relationship: "depends_on"},
			{From: "B", To: "C", Relationship: "depends_on"},
		},
	}

	cycles := DetectCycles(graph)
	if len(cycles) != 0 {
		t.Errorf("Expected no cycles, found %d", len(cycles))
	}
}

func TestGetDependencyDepth(t *testing.T) {
	defer Reset()

	// Create a linear dependency chain: Comment -> Post -> User
	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Comment",
				Relationships: []RelationshipMetadata{
					{Name: "post", TargetResource: "Post", Type: "belongs_to"},
				},
			},
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "User",
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Comment has depth 2 (Comment -> Post -> User)
	depth, err := GetDependencyDepth("Comment")
	if err != nil {
		t.Fatalf("GetDependencyDepth failed: %v", err)
	}
	if depth != 2 {
		t.Errorf("Expected depth 2 for Comment, got %d", depth)
	}

	// Post has depth 1 (Post -> User)
	depth, err = GetDependencyDepth("Post")
	if err != nil {
		t.Fatalf("GetDependencyDepth failed: %v", err)
	}
	if depth != 1 {
		t.Errorf("Expected depth 1 for Post, got %d", depth)
	}

	// User has depth 0 (no dependencies)
	depth, err = GetDependencyDepth("User")
	if err != nil {
		t.Fatalf("GetDependencyDepth failed: %v", err)
	}
	if depth != 0 {
		t.Errorf("Expected depth 0 for User, got %d", depth)
	}
}

func TestCountDependents(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "Comment",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "User",
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// User has 2 dependents (Post and Comment)
	count, err := CountDependents("User")
	if err != nil {
		t.Fatalf("CountDependents failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 dependents for User, got %d", count)
	}

	// Post has 0 dependents
	count, err = CountDependents("Post")
	if err != nil {
		t.Fatalf("CountDependents failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 dependents for Post, got %d", count)
	}
}

func TestCountDependencies(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
					{Name: "category", TargetResource: "Category", Type: "belongs_to"},
				},
			},
			{
				Name: "User",
			},
			{
				Name: "Category",
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Post has 2 dependencies (User and Category)
	count, err := CountDependencies("Post")
	if err != nil {
		t.Fatalf("CountDependencies failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 dependencies for Post, got %d", count)
	}

	// User has 0 dependencies
	count, err = CountDependencies("User")
	if err != nil {
		t.Fatalf("CountDependencies failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 dependencies for User, got %d", count)
	}
}

func TestDependencyCaching(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "User",
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	opts := DependencyOptions{Depth: 1}

	// First query - should populate cache
	graph1, err := QueryDependencies("Post", opts)
	if err != nil {
		t.Fatalf("QueryDependencies failed: %v", err)
	}

	// Second query - should use cache
	graph2, err := QueryDependencies("Post", opts)
	if err != nil {
		t.Fatalf("QueryDependencies failed: %v", err)
	}

	// Results should be identical
	if len(graph1.Nodes) != len(graph2.Nodes) {
		t.Errorf("Cached result differs: nodes %d vs %d", len(graph1.Nodes), len(graph2.Nodes))
	}

	if len(graph1.Edges) != len(graph2.Edges) {
		t.Errorf("Cached result differs: edges %d vs %d", len(graph1.Edges), len(graph2.Edges))
	}
}

func TestBuildDependencyGraph_WithMiddleware(t *testing.T) {
	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Middleware: map[string][]string{
					"list":   {"AuthMiddleware", "RateLimitMiddleware"},
					"create": {"AuthMiddleware"},
				},
			},
		},
	}

	graph := BuildDependencyGraph(meta)

	// Verify middleware nodes were created
	if _, ok := graph.Nodes["AuthMiddleware"]; !ok {
		t.Error("AuthMiddleware node not created")
	}

	if _, ok := graph.Nodes["RateLimitMiddleware"]; !ok {
		t.Error("RateLimitMiddleware node not created")
	}

	// Verify middleware nodes have correct type
	if graph.Nodes["AuthMiddleware"].Type != "middleware" {
		t.Errorf("AuthMiddleware has wrong type: %s", graph.Nodes["AuthMiddleware"].Type)
	}

	// Verify edges exist (should be 2 deduplicated edges with correct weights)
	// Post->AuthMiddleware (weight=2, used in list and create)
	// Post->RateLimitMiddleware (weight=1, used in list)
	if len(graph.Edges) != 2 {
		t.Errorf("Expected 2 edges (deduplicated), got %d", len(graph.Edges))
	}

	foundAuthEdge := false
	foundRateLimitEdge := false

	for _, edge := range graph.Edges {
		if edge.From == "Post" && edge.Relationship == "uses" {
			if edge.To == "AuthMiddleware" {
				foundAuthEdge = true
				if edge.Weight != 2 {
					t.Errorf("Expected AuthMiddleware edge weight=2 (used in 2 operations), got %d", edge.Weight)
				}
			}
			if edge.To == "RateLimitMiddleware" {
				foundRateLimitEdge = true
				if edge.Weight != 1 {
					t.Errorf("Expected RateLimitMiddleware edge weight=1, got %d", edge.Weight)
				}
			}
		}
	}

	if !foundAuthEdge {
		t.Error("Expected edge from Post to AuthMiddleware")
	}

	if !foundRateLimitEdge {
		t.Error("Expected edge from Post to RateLimitMiddleware")
	}
}

func TestExtractFunctionCalls(t *testing.T) {
	sourceCode := `
        self.slug = String.slugify(self.title)
        timestamp = Time.now()
    `

	functions := extractFunctionCalls(sourceCode)

	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	expected := map[string]bool{
		"String.slugify": true,
		"Time.now":       true,
	}

	for _, fn := range functions {
		if !expected[fn] {
			t.Errorf("Unexpected function: %s", fn)
		}
		delete(expected, fn)
	}

	if len(expected) > 0 {
		t.Errorf("Missing functions: %v", expected)
	}
}

func TestExtractFunctionCalls_Empty(t *testing.T) {
	functions := extractFunctionCalls("")
	if functions != nil {
		t.Errorf("Expected nil for empty source code, got %v", functions)
	}
}

func TestExtractFunctionCalls_NoMatches(t *testing.T) {
	sourceCode := `
        self.title = "Hello"
        x = 5 + 3
    `

	functions := extractFunctionCalls(sourceCode)

	if len(functions) != 0 {
		t.Errorf("Expected 0 functions, got %d: %v", len(functions), functions)
	}
}

func TestExtractFunctionCalls_AllCapsNamespace(t *testing.T) {
	sourceCode := `
        id = UUID.generate()
        response = HTTP.request()
        data = JSON.parse()
    `

	functions := extractFunctionCalls(sourceCode)

	expected := []string{"UUID.generate", "HTTP.request", "JSON.parse"}
	if len(functions) != len(expected) {
		t.Errorf("Expected %d functions, got %d", len(expected), len(functions))
	}

	for _, exp := range expected {
		found := false
		for _, fn := range functions {
			if fn == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find function call: %s", exp)
		}
	}
}

func TestExtractFunctionCalls_CamelCaseFunction(t *testing.T) {
	sourceCode := `
        text = String.toUpperCase(input)
        result = Array.forEach(items)
    `

	functions := extractFunctionCalls(sourceCode)

	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	expected := map[string]bool{
		"String.toUpperCase": true,
		"Array.forEach":      true,
	}

	for _, fn := range functions {
		if !expected[fn] {
			t.Errorf("Unexpected function: %s", fn)
		}
	}
}

func TestBuildDependencyGraph_BlogExample(t *testing.T) {
	// Integration test with User, Post, Comment
	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "User",
			},
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
				Middleware: map[string][]string{
					"create": {"AuthMiddleware"},
				},
				Hooks: []HookMetadata{
					{
						Type:       "before_create",
						SourceCode: "self.slug = String.slugify(self.title)",
					},
				},
			},
			{
				Name: "Comment",
				Relationships: []RelationshipMetadata{
					{Name: "post", TargetResource: "Post", Type: "belongs_to"},
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
		},
	}

	graph := BuildDependencyGraph(meta)

	// Verify all resource nodes
	for _, name := range []string{"User", "Post", "Comment"} {
		if node, ok := graph.Nodes[name]; !ok {
			t.Errorf("Missing resource node: %s", name)
		} else if node.Type != "resource" {
			t.Errorf("Node %s has wrong type: %s", name, node.Type)
		}
	}

	// Verify middleware node
	if node, ok := graph.Nodes["AuthMiddleware"]; !ok {
		t.Error("Missing AuthMiddleware node")
	} else if node.Type != "middleware" {
		t.Errorf("AuthMiddleware has wrong type: %s", node.Type)
	}

	// Verify function node
	if node, ok := graph.Nodes["String.slugify"]; !ok {
		t.Error("Missing String.slugify function node")
	} else if node.Type != "function" {
		t.Errorf("String.slugify has wrong type: %s", node.Type)
	}

	// Verify relationship edges
	foundPostToUser := false
	foundCommentToPost := false
	foundCommentToUser := false

	for _, edge := range graph.Edges {
		if edge.From == "Post" && edge.To == "User" && edge.Relationship == "belongs_to" {
			foundPostToUser = true
		}
		if edge.From == "Comment" && edge.To == "Post" && edge.Relationship == "belongs_to" {
			foundCommentToPost = true
		}
		if edge.From == "Comment" && edge.To == "User" && edge.Relationship == "belongs_to" {
			foundCommentToUser = true
		}
	}

	if !foundPostToUser {
		t.Error("Missing relationship edge: Post -> User")
	}
	if !foundCommentToPost {
		t.Error("Missing relationship edge: Comment -> Post")
	}
	if !foundCommentToUser {
		t.Error("Missing relationship edge: Comment -> User")
	}

	// Verify middleware edge
	foundMiddlewareEdge := false
	for _, edge := range graph.Edges {
		if edge.From == "Post" && edge.To == "AuthMiddleware" && edge.Relationship == "uses" {
			foundMiddlewareEdge = true
			break
		}
	}
	if !foundMiddlewareEdge {
		t.Error("Missing middleware edge: Post -> AuthMiddleware")
	}

	// Verify function call edge
	foundFunctionEdge := false
	for _, edge := range graph.Edges {
		if edge.From == "Post" && edge.To == "String.slugify" && edge.Relationship == "calls" {
			foundFunctionEdge = true
			break
		}
	}
	if !foundFunctionEdge {
		t.Error("Missing function call edge: Post -> String.slugify")
	}

	// Verify total node count (3 resources + 1 middleware + 1 function)
	expectedNodeCount := 5
	if len(graph.Nodes) != expectedNodeCount {
		t.Errorf("Expected %d nodes, got %d", expectedNodeCount, len(graph.Nodes))
	}

	// Verify total edge count (3 relationships + 1 middleware + 1 function call)
	expectedEdgeCount := 5
	if len(graph.Edges) != expectedEdgeCount {
		t.Errorf("Expected %d edges, got %d", expectedEdgeCount, len(graph.Edges))
	}
}

func TestBuildDependencyGraph_Performance(t *testing.T) {
	// Create metadata with 50 resources, each with 2 relationships
	meta := &Metadata{
		Version:   "1.0.0",
		Resources: make([]ResourceMetadata, 50),
	}

	// Generate 50 resources with relationships
	for i := 0; i < 50; i++ {
		resource := ResourceMetadata{
			Name: fmt.Sprintf("Resource%d", i),
		}

		// Add relationships to previous resources
		if i > 0 {
			resource.Relationships = []RelationshipMetadata{
				{Name: "rel1", TargetResource: fmt.Sprintf("Resource%d", i-1), Type: "belongs_to"},
			}
		}
		if i > 1 {
			resource.Relationships = append(resource.Relationships,
				RelationshipMetadata{Name: "rel2", TargetResource: fmt.Sprintf("Resource%d", i-2), Type: "has_many"},
			)
		}

		meta.Resources[i] = resource
	}

	// Measure execution time
	start := time.Now()
	BuildDependencyGraph(meta)
	duration := time.Since(start)

	// Should be < 50ms
	if duration > 50*time.Millisecond {
		t.Errorf("Performance requirement not met: took %v (expected < 50ms)", duration)
	}
}

func BenchmarkBuildDependencyGraph_50Resources(b *testing.B) {
	// Create metadata with 50 resources, each with 2 relationships
	meta := &Metadata{
		Version:   "1.0.0",
		Resources: make([]ResourceMetadata, 50),
	}

	// Generate 50 resources with relationships
	for i := 0; i < 50; i++ {
		resource := ResourceMetadata{
			Name: fmt.Sprintf("Resource%d", i),
		}

		// Add relationships to previous resources
		if i > 0 {
			resource.Relationships = []RelationshipMetadata{
				{Name: "rel1", TargetResource: fmt.Sprintf("Resource%d", i-1), Type: "belongs_to"},
			}
		}
		if i > 1 {
			resource.Relationships = append(resource.Relationships,
				RelationshipMetadata{Name: "rel2", TargetResource: fmt.Sprintf("Resource%d", i-2), Type: "has_many"},
			)
		}

		meta.Resources[i] = resource
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildDependencyGraph(meta)
	}
}

func TestWarnCircularDependencies_NoCycles(t *testing.T) {
	// Create a graph without cycles
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"A": {ID: "A", Type: "resource", Name: "A"},
			"B": {ID: "B", Type: "resource", Name: "B"},
		},
		Edges: []DependencyEdge{
			{From: "A", To: "B", Relationship: "depends_on"},
		},
	}

	// Should not panic or error - just logs warnings if cycles exist
	WarnCircularDependencies(graph)
}

func TestWarnCircularDependencies_WithCycles(t *testing.T) {
	// Create a graph with a cycle
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"A": {ID: "A", Type: "resource", Name: "A"},
			"B": {ID: "B", Type: "resource", Name: "B"},
		},
		Edges: []DependencyEdge{
			{From: "A", To: "B", Relationship: "depends_on"},
			{From: "B", To: "A", Relationship: "depends_on"},
		},
	}

	// Should log warning but not panic
	WarnCircularDependencies(graph)
}

func TestBuildDependencyGraph_EdgeWeights(t *testing.T) {
	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Middleware: map[string][]string{
					"list":   {"AuthMiddleware"},
					"create": {"AuthMiddleware"},
					"update": {"AuthMiddleware"},
					"delete": {"AuthMiddleware"},
				},
				Hooks: []HookMetadata{
					{
						Type:       "before_create",
						SourceCode: "self.slug = String.slugify(self.title)",
					},
					{
						Type:       "after_create",
						SourceCode: "self.slug = String.slugify(self.description)",
					},
				},
			},
		},
	}

	graph := BuildDependencyGraph(meta)

	// Should have 2 edges: Post->AuthMiddleware and Post->String.slugify
	if len(graph.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(graph.Edges))
	}

	// AuthMiddleware should have weight=4 (used in 4 operations)
	foundAuthEdge := false
	foundSlugifyEdge := false
	for _, edge := range graph.Edges {
		if edge.To == "AuthMiddleware" {
			foundAuthEdge = true
			if edge.Weight != 4 {
				t.Errorf("Expected AuthMiddleware weight=4, got %d", edge.Weight)
			}
		}
		if edge.To == "String.slugify" {
			foundSlugifyEdge = true
			if edge.Weight != 2 {
				t.Errorf("Expected String.slugify weight=2, got %d", edge.Weight)
			}
		}
	}

	if !foundAuthEdge {
		t.Error("Expected edge to AuthMiddleware")
	}
	if !foundSlugifyEdge {
		t.Error("Expected edge to String.slugify")
	}
}

// BenchmarkQueryDependencies_Depth1_Cold benchmarks depth=1 queries (cold cache)
func BenchmarkQueryDependencies_Depth1_Cold(b *testing.B) {
	meta := generateDeepDependencyChain(10)
	data, _ := json.Marshal(meta)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		Reset()
		RegisterMetadata(data)
		b.StartTimer()

		opts := DependencyOptions{Depth: 1}
		_, err := QueryDependencies("Resource0", opts)
		if err != nil {
			b.Fatalf("QueryDependencies failed: %v", err)
		}
	}
}

// BenchmarkQueryDependencies_Depth1_Cached benchmarks depth=1 queries (cached)
func BenchmarkQueryDependencies_Depth1_Cached(b *testing.B) {
	defer Reset()

	meta := generateDeepDependencyChain(10)
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	opts := DependencyOptions{Depth: 1}

	// Warm up cache
	QueryDependencies("Resource0", opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := QueryDependencies("Resource0", opts)
		if err != nil {
			b.Fatalf("QueryDependencies failed: %v", err)
		}
	}
}

// BenchmarkQueryDependencies_Depth3_Cold benchmarks depth=3 queries (cold cache)
func BenchmarkQueryDependencies_Depth3_Cold(b *testing.B) {
	meta := generateDeepDependencyChain(10)
	data, _ := json.Marshal(meta)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		Reset()
		RegisterMetadata(data)
		b.StartTimer()

		opts := DependencyOptions{Depth: 3}
		_, err := QueryDependencies("Resource0", opts)
		if err != nil {
			b.Fatalf("QueryDependencies failed: %v", err)
		}
	}
}

// BenchmarkQueryDependencies_Depth3_Cached benchmarks depth=3 queries (cached)
func BenchmarkQueryDependencies_Depth3_Cached(b *testing.B) {
	defer Reset()

	meta := generateDeepDependencyChain(10)
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	opts := DependencyOptions{Depth: 3}

	// Warm up cache
	QueryDependencies("Resource0", opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := QueryDependencies("Resource0", opts)
		if err != nil {
			b.Fatalf("QueryDependencies failed: %v", err)
		}
	}
}

// BenchmarkQueryDependencies_Depth5_Cold benchmarks depth=5 queries (cold cache)
func BenchmarkQueryDependencies_Depth5_Cold(b *testing.B) {
	meta := generateDeepDependencyChain(20)
	data, _ := json.Marshal(meta)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		Reset()
		RegisterMetadata(data)
		b.StartTimer()

		opts := DependencyOptions{Depth: 5}
		_, err := QueryDependencies("Resource0", opts)
		if err != nil {
			b.Fatalf("QueryDependencies failed: %v", err)
		}
	}
}

// BenchmarkQueryDependencies_Depth5_Cached benchmarks depth=5 queries (cached)
func BenchmarkQueryDependencies_Depth5_Cached(b *testing.B) {
	defer Reset()

	meta := generateDeepDependencyChain(20)
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	opts := DependencyOptions{Depth: 5}

	// Warm up cache
	QueryDependencies("Resource0", opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := QueryDependencies("Resource0", opts)
		if err != nil {
			b.Fatalf("QueryDependencies failed: %v", err)
		}
	}
}

// BenchmarkBuildDependencyGraph_WithAdjacencyLists measures adjacency list construction overhead
func BenchmarkBuildDependencyGraph_WithAdjacencyLists(b *testing.B) {
	meta := generateDeepDependencyChain(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		graph := BuildDependencyGraph(meta)
		if graph.outgoingEdges == nil {
			b.Fatal("Adjacency lists not built")
		}
	}
}

// generateDeepDependencyChain creates a chain of resources with deep dependencies
func generateDeepDependencyChain(n int) *Metadata {
	meta := &Metadata{
		Version:   "1.0.0",
		Resources: make([]ResourceMetadata, n),
	}

	for i := 0; i < n; i++ {
		resource := ResourceMetadata{
			Name: fmt.Sprintf("Resource%d", i),
		}

		// Create a dependency chain: Resource0 -> Resource1 -> Resource2 -> ...
		if i > 0 {
			resource.Relationships = []RelationshipMetadata{
				{
					Name:           "dependency",
					TargetResource: fmt.Sprintf("Resource%d", i-1),
					Type:           "belongs_to",
				},
			}
		}

		// Also add a few branching relationships for more complexity
		if i > 1 && i%3 == 0 {
			resource.Relationships = append(resource.Relationships,
				RelationshipMetadata{
					Name:           "alternate",
					TargetResource: fmt.Sprintf("Resource%d", i-2),
					Type:           "belongs_to",
				},
			)
		}

		meta.Resources[i] = resource
	}

	return meta
}

func TestFindEdges_Fallback(t *testing.T) {
	// Test fallback when adjacency lists are nil
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"A": {ID: "A", Type: "resource", Name: "A"},
			"B": {ID: "B", Type: "resource", Name: "B"},
			"C": {ID: "C", Type: "resource", Name: "C"},
		},
		Edges: []DependencyEdge{
			{From: "A", To: "B", Relationship: "belongs_to"},
			{From: "A", To: "C", Relationship: "has_many"},
		},
		// Do NOT initialize adjacency lists
		outgoingEdges: nil,
		incomingEdges: nil,
	}

	// Test findOutgoingEdges fallback
	edges := findOutgoingEdges(graph, "A")
	if len(edges) != 2 {
		t.Errorf("Fallback findOutgoingEdges failed: expected 2 edges, got %d", len(edges))
	}

	// Test findIncomingEdges fallback
	edges = findIncomingEdges(graph, "B")
	if len(edges) != 1 {
		t.Errorf("Fallback findIncomingEdges failed: expected 1 edge, got %d", len(edges))
	}

	edges = findIncomingEdges(graph, "C")
	if len(edges) != 1 {
		t.Errorf("Fallback findIncomingEdges failed: expected 1 edge, got %d", len(edges))
	}
}
