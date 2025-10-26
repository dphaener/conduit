package metadata

import (
	"encoding/json"
	"testing"
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
