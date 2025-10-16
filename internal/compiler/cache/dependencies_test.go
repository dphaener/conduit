package cache

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestDependencyGraph_AddFile(t *testing.T) {
	dg := NewDependencyGraph()

	dg.AddFile("/test/user.cdt", "User")

	if dg.Size() != 1 {
		t.Errorf("Size() = %d, want 1", dg.Size())
	}

	deps := dg.GetDependencies("/test/user.cdt")
	if len(deps) != 0 {
		t.Errorf("GetDependencies() = %d, want 0", len(deps))
	}
}

func TestDependencyGraph_AddDependency(t *testing.T) {
	dg := NewDependencyGraph()

	dg.AddFile("/test/post.cdt", "Post")
	dg.AddFile("/test/user.cdt", "User")

	// Post depends on User
	dg.AddDependency("/test/post.cdt", "/test/user.cdt")

	deps := dg.GetDependencies("/test/post.cdt")
	if len(deps) != 1 {
		t.Fatalf("Post should have 1 dependency, has %d", len(deps))
	}
	if deps[0] != "/test/user.cdt" {
		t.Errorf("Post dependency = %s, want /test/user.cdt", deps[0])
	}

	dependents := dg.GetDependents("/test/user.cdt")
	if len(dependents) != 1 {
		t.Fatalf("User should have 1 dependent, has %d", len(dependents))
	}
	if dependents[0] != "/test/post.cdt" {
		t.Errorf("User dependent = %s, want /test/post.cdt", dependents[0])
	}
}

func TestDependencyGraph_GetTransitiveDependents(t *testing.T) {
	dg := NewDependencyGraph()

	// Build a chain: A <- B <- C <- D
	dg.AddFile("/test/a.cdt", "A")
	dg.AddFile("/test/b.cdt", "B")
	dg.AddFile("/test/c.cdt", "C")
	dg.AddFile("/test/d.cdt", "D")

	dg.AddDependency("/test/b.cdt", "/test/a.cdt")
	dg.AddDependency("/test/c.cdt", "/test/b.cdt")
	dg.AddDependency("/test/d.cdt", "/test/c.cdt")

	// Changing A should invalidate B, C, D
	transitive := dg.GetTransitiveDependents("/test/a.cdt")

	if len(transitive) != 3 {
		t.Errorf("GetTransitiveDependents() = %d, want 3", len(transitive))
	}

	// Check all are present
	found := make(map[string]bool)
	for _, dep := range transitive {
		found[dep] = true
	}

	if !found["/test/b.cdt"] || !found["/test/c.cdt"] || !found["/test/d.cdt"] {
		t.Errorf("GetTransitiveDependents() missing expected files")
	}
}

func TestDependencyGraph_GetIndependentFiles(t *testing.T) {
	dg := NewDependencyGraph()

	// A and B are independent, C depends on A
	dg.AddFile("/test/a.cdt", "A")
	dg.AddFile("/test/b.cdt", "B")
	dg.AddFile("/test/c.cdt", "C")

	dg.AddDependency("/test/c.cdt", "/test/a.cdt")

	independent := dg.GetIndependentFiles()

	if len(independent) != 2 {
		t.Errorf("GetIndependentFiles() = %d, want 2", len(independent))
	}

	// Check A and B are present
	found := make(map[string]bool)
	for _, file := range independent {
		found[file] = true
	}

	if !found["/test/a.cdt"] || !found["/test/b.cdt"] {
		t.Errorf("GetIndependentFiles() missing expected files")
	}

	if found["/test/c.cdt"] {
		t.Errorf("GetIndependentFiles() should not include C (depends on A)")
	}
}

func TestDependencyGraph_GetTopologicalOrder(t *testing.T) {
	dg := NewDependencyGraph()

	// Build dependencies: A, B are independent; C depends on A; D depends on B and C
	dg.AddFile("/test/a.cdt", "A")
	dg.AddFile("/test/b.cdt", "B")
	dg.AddFile("/test/c.cdt", "C")
	dg.AddFile("/test/d.cdt", "D")

	dg.AddDependency("/test/c.cdt", "/test/a.cdt")
	dg.AddDependency("/test/d.cdt", "/test/b.cdt")
	dg.AddDependency("/test/d.cdt", "/test/c.cdt")

	order, err := dg.GetTopologicalOrder()
	if err != nil {
		t.Fatalf("GetTopologicalOrder() error = %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("GetTopologicalOrder() returned %d files, want 4", len(order))
	}

	// Create position map
	pos := make(map[string]int)
	for i, file := range order {
		pos[file] = i
	}

	// Verify dependencies come before dependents
	if pos["/test/a.cdt"] >= pos["/test/c.cdt"] {
		t.Errorf("A should come before C in topological order")
	}
	if pos["/test/b.cdt"] >= pos["/test/d.cdt"] {
		t.Errorf("B should come before D in topological order")
	}
	if pos["/test/c.cdt"] >= pos["/test/d.cdt"] {
		t.Errorf("C should come before D in topological order")
	}
}

func TestDependencyGraph_GetTopologicalOrder_Cycle(t *testing.T) {
	dg := NewDependencyGraph()

	// Create a cycle: A -> B -> C -> A
	dg.AddFile("/test/a.cdt", "A")
	dg.AddFile("/test/b.cdt", "B")
	dg.AddFile("/test/c.cdt", "C")

	dg.AddDependency("/test/a.cdt", "/test/b.cdt")
	dg.AddDependency("/test/b.cdt", "/test/c.cdt")
	dg.AddDependency("/test/c.cdt", "/test/a.cdt")

	_, err := dg.GetTopologicalOrder()
	if err == nil {
		t.Errorf("GetTopologicalOrder() should return error for cycle")
	}

	if _, ok := err.(*CycleError); !ok {
		t.Errorf("GetTopologicalOrder() should return CycleError, got %T", err)
	}
}

func TestDependencyGraph_RemoveFile(t *testing.T) {
	dg := NewDependencyGraph()

	dg.AddFile("/test/a.cdt", "A")
	dg.AddFile("/test/b.cdt", "B")
	dg.AddFile("/test/c.cdt", "C")

	dg.AddDependency("/test/b.cdt", "/test/a.cdt")
	dg.AddDependency("/test/c.cdt", "/test/b.cdt")

	// Remove B
	dg.RemoveFile("/test/b.cdt")

	if dg.Size() != 2 {
		t.Errorf("Size() = %d after removal, want 2", dg.Size())
	}

	// A should have no dependents now
	dependents := dg.GetDependents("/test/a.cdt")
	if len(dependents) != 0 {
		t.Errorf("A should have 0 dependents after removing B, has %d", len(dependents))
	}

	// C should have no dependencies now
	deps := dg.GetDependencies("/test/c.cdt")
	if len(deps) != 0 {
		t.Errorf("C should have 0 dependencies after removing B, has %d", len(deps))
	}
}

func TestDependencyGraph_Clear(t *testing.T) {
	dg := NewDependencyGraph()

	dg.AddFile("/test/a.cdt", "A")
	dg.AddFile("/test/b.cdt", "B")
	dg.AddDependency("/test/b.cdt", "/test/a.cdt")

	if dg.Size() != 2 {
		t.Fatalf("Size() = %d before clear, want 2", dg.Size())
	}

	dg.Clear()

	if dg.Size() != 0 {
		t.Errorf("Size() = %d after clear, want 0", dg.Size())
	}
}

func TestDependencyGraph_BuildDependencies(t *testing.T) {
	dg := NewDependencyGraph()

	// Create a program with relationships
	program := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Post",
				Relationships: []*ast.RelationshipNode{
					{
						Name: "author",
						Type: "User",
					},
					{
						Name: "category",
						Type: "Category",
					},
				},
				Fields: []*ast.FieldNode{
					{
						Name: "title",
						Type: &ast.TypeNode{
							Kind: ast.TypePrimitive,
							Name: "string",
						},
					},
				},
			},
		},
	}

	dg.BuildDependencies("/test/post.cdt", program)

	// Should add the file
	if dg.Size() != 1 {
		t.Errorf("Size() = %d after BuildDependencies, want 1", dg.Size())
	}

	// Note: BuildDependencies doesn't create edges because it doesn't have
	// a resource-to-file mapping. That would be done by the coordinator.
}

func TestDependencyGraph_NoDuplicateDependencies(t *testing.T) {
	dg := NewDependencyGraph()

	dg.AddFile("/test/a.cdt", "A")
	dg.AddFile("/test/b.cdt", "B")

	// Add dependency twice
	dg.AddDependency("/test/b.cdt", "/test/a.cdt")
	dg.AddDependency("/test/b.cdt", "/test/a.cdt")

	deps := dg.GetDependencies("/test/b.cdt")
	if len(deps) != 1 {
		t.Errorf("GetDependencies() = %d, want 1 (no duplicates)", len(deps))
	}

	dependents := dg.GetDependents("/test/a.cdt")
	if len(dependents) != 1 {
		t.Errorf("GetDependents() = %d, want 1 (no duplicates)", len(dependents))
	}
}

func TestDependencyGraph_ComplexGraph(t *testing.T) {
	dg := NewDependencyGraph()

	// Build a more complex dependency graph
	//     A     B
	//    / \   / \
	//   C   D E   F
	//    \ /   \ /
	//     G     H

	files := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	for _, f := range files {
		dg.AddFile("/test/"+f+".cdt", f)
	}

	dg.AddDependency("/test/C.cdt", "/test/A.cdt")
	dg.AddDependency("/test/D.cdt", "/test/A.cdt")
	dg.AddDependency("/test/E.cdt", "/test/B.cdt")
	dg.AddDependency("/test/F.cdt", "/test/B.cdt")
	dg.AddDependency("/test/G.cdt", "/test/C.cdt")
	dg.AddDependency("/test/G.cdt", "/test/D.cdt")
	dg.AddDependency("/test/H.cdt", "/test/E.cdt")
	dg.AddDependency("/test/H.cdt", "/test/F.cdt")

	// Get topological order
	order, err := dg.GetTopologicalOrder()
	if err != nil {
		t.Fatalf("GetTopologicalOrder() error = %v", err)
	}

	if len(order) != 8 {
		t.Fatalf("GetTopologicalOrder() returned %d files, want 8", len(order))
	}

	// Create position map
	pos := make(map[string]int)
	for i, file := range order {
		pos[file] = i
	}

	// Verify all dependency constraints
	if pos["/test/A.cdt"] >= pos["/test/C.cdt"] {
		t.Errorf("A should come before C")
	}
	if pos["/test/A.cdt"] >= pos["/test/D.cdt"] {
		t.Errorf("A should come before D")
	}
	if pos["/test/C.cdt"] >= pos["/test/G.cdt"] {
		t.Errorf("C should come before G")
	}
	if pos["/test/D.cdt"] >= pos["/test/G.cdt"] {
		t.Errorf("D should come before G")
	}
}
