package build

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDependencyGraph(t *testing.T) {
	dg := NewDependencyGraph()

	if dg == nil {
		t.Fatal("Expected non-nil dependency graph")
	}

	if dg.nodes == nil {
		t.Error("Expected non-nil nodes map")
	}

	if dg.edges == nil {
		t.Error("Expected non-nil edges map")
	}
}

func TestAddNode(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	dg := NewDependencyGraph()
	deps := []string{"dep1.txt", "dep2.txt"}

	err := dg.AddNode(tmpFile, deps)
	if err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	node, ok := dg.GetNode(tmpFile)
	if !ok {
		t.Fatal("Expected node to exist")
	}

	if node.Path != tmpFile {
		t.Errorf("Expected path %q, got %q", tmpFile, node.Path)
	}

	if node.Hash == "" {
		t.Error("Expected non-empty hash")
	}

	if len(node.Dependencies) != len(deps) {
		t.Errorf("Expected %d dependencies, got %d", len(deps), len(node.Dependencies))
	}
}

func TestGetNode(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	dg := NewDependencyGraph()
	if err := dg.AddNode(tmpFile, nil); err != nil {
		t.Fatal(err)
	}

	// Get existing node
	node, ok := dg.GetNode(tmpFile)
	if !ok {
		t.Fatal("Expected node to exist")
	}

	if node == nil {
		t.Fatal("Expected non-nil node")
	}

	// Get non-existing node
	_, ok = dg.GetNode("nonexistent.txt")
	if ok {
		t.Error("Expected node not to exist")
	}
}

func TestFindAffected(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := make([]string, 4)
	for i := 0; i < 4; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
		files[i] = path
	}

	// Create dependency graph:
	// file0 -> file1
	// file1 -> file2
	// file2 -> file3
	dg := NewDependencyGraph()
	if err := dg.AddNode(files[0], []string{files[1]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[1], []string{files[2]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[2], []string{files[3]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[3], nil); err != nil {
		t.Fatal(err)
	}

	// Find affected files when file3 changes
	affected := dg.FindAffected([]string{files[3]})

	// All files depend on file3 (directly or transitively)
	if len(affected) < 4 {
		t.Errorf("Expected at least 4 affected files, got %d", len(affected))
	}
}

func TestTopologicalSort(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := make([]string, 3)
	for i := 0; i < 3; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
		files[i] = path
	}

	// Create dependency graph:
	// file0 depends on file1
	// file1 depends on file2
	// file2 has no dependencies
	dg := NewDependencyGraph()
	if err := dg.AddNode(files[0], []string{files[1]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[1], []string{files[2]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[2], nil); err != nil {
		t.Fatal(err)
	}

	// Topological sort should put dependencies first
	sorted := dg.TopologicalSort(files)

	if len(sorted) != len(files) {
		t.Errorf("Expected %d files in sorted list, got %d", len(files), len(sorted))
	}

	// file2 should come before file1, file1 before file0
	file2Idx := -1
	file1Idx := -1
	file0Idx := -1

	for i, f := range sorted {
		switch f {
		case files[2]:
			file2Idx = i
		case files[1]:
			file1Idx = i
		case files[0]:
			file0Idx = i
		}
	}

	// file0 depends on file1, file1 depends on file2
	// Build order should be: file2, then file1, then file0
	// (dependencies come before their dependents)

	if file2Idx >= file1Idx {
		t.Errorf("Expected file2 (idx %d) before file1 (idx %d)", file2Idx, file1Idx)
	}

	if file1Idx >= file0Idx {
		t.Errorf("Expected file1 (idx %d) before file0 (idx %d)", file1Idx, file0Idx)
	}
}

func TestHasCycle(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := make([]string, 3)
	for i := 0; i < 3; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
		files[i] = path
	}

	// Test without cycle
	dg := NewDependencyGraph()
	if err := dg.AddNode(files[0], []string{files[1]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[1], []string{files[2]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[2], nil); err != nil {
		t.Fatal(err)
	}

	if dg.HasCycle() {
		t.Error("Expected no cycle, but got one")
	}

	// Test with cycle
	dg2 := NewDependencyGraph()
	if err := dg2.AddNode(files[0], []string{files[1]}); err != nil {
		t.Fatal(err)
	}
	if err := dg2.AddNode(files[1], []string{files[2]}); err != nil {
		t.Fatal(err)
	}
	if err := dg2.AddNode(files[2], []string{files[0]}); err != nil {
		t.Fatal(err)
	}

	if !dg2.HasCycle() {
		t.Error("Expected cycle, but got none")
	}
}

func TestGetDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	for _, f := range []string{file1, file2, file3} {
		if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	dg := NewDependencyGraph()
	deps := []string{file2, file3}
	if err := dg.AddNode(file1, deps); err != nil {
		t.Fatal(err)
	}

	gotDeps := dg.GetDependencies(file1)

	if len(gotDeps) != len(deps) {
		t.Errorf("Expected %d dependencies, got %d", len(deps), len(gotDeps))
	}

	// Verify we got a copy (not the same slice)
	if &gotDeps[0] == &deps[0] {
		t.Error("Expected copy of dependencies, got same slice")
	}
}

func TestGetDependents(t *testing.T) {
	tmpDir := t.TempDir()

	files := make([]string, 4)
	for i := 0; i < 4; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
		files[i] = path
	}

	// file0, file1, file2 all depend on file3
	dg := NewDependencyGraph()
	if err := dg.AddNode(files[0], []string{files[3]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[1], []string{files[3]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[2], []string{files[3]}); err != nil {
		t.Fatal(err)
	}
	if err := dg.AddNode(files[3], nil); err != nil {
		t.Fatal(err)
	}

	dependents := dg.GetDependents(files[3])

	if len(dependents) != 3 {
		t.Errorf("Expected 3 dependents, got %d", len(dependents))
	}
}

func TestComputeFileHash(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	content := "test content for hashing"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatalf("computeFileHash failed: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	if len(hash) != 64 {
		t.Errorf("Expected hash length 64 (SHA-256), got %d", len(hash))
	}

	// Verify same file produces same hash
	hash2, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if hash != hash2 {
		t.Error("Expected same hash for same file")
	}

	// Verify different content produces different hash
	if err := os.WriteFile(tmpFile, []byte("different content"), 0644); err != nil {
		t.Fatal(err)
	}

	hash3, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if hash == hash3 {
		t.Error("Expected different hash for different content")
	}
}

func BenchmarkAddNode(b *testing.B) {
	tmpDir := b.TempDir()
	dg := NewDependencyGraph()

	// Create test files
	files := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			b.Fatal(err)
		}
		files[i] = path
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := dg.AddNode(files[i], nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTopologicalSort(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test files
	numFiles := 100
	files := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			b.Fatal(err)
		}
		files[i] = path
	}

	// Create dependency graph
	dg := NewDependencyGraph()
	for i := 0; i < numFiles; i++ {
		var deps []string
		if i > 0 {
			deps = []string{files[i-1]}
		}
		if err := dg.AddNode(files[i], deps); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dg.TopologicalSort(files)
	}
}
