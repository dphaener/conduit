package cache

import (
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestASTCache_SetAndGet(t *testing.T) {
	cache := NewASTCache()

	// Create a test program
	program := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
			},
		},
	}

	path := "/test/user.cdt"
	hash := "abc123"

	// Set
	cache.Set(path, program, hash)

	// Get by path
	cached, exists := cache.Get(path)
	if !exists {
		t.Errorf("Get() returned false for existing entry")
	}

	if cached == nil {
		t.Fatalf("Get() returned nil cached entry")
	}

	if cached.Hash != hash {
		t.Errorf("Get() hash = %s, want %s", cached.Hash, hash)
	}

	if cached.Program == nil {
		t.Errorf("Get() program is nil")
	}

	if len(cached.Program.Resources) != 1 {
		t.Errorf("Get() program has %d resources, want 1", len(cached.Program.Resources))
	}
}

func TestASTCache_GetByHash(t *testing.T) {
	cache := NewASTCache()

	program := &ast.Program{
		Resources: []*ast.ResourceNode{
			{Name: "Post"},
		},
	}

	path := "/test/post.cdt"
	hash := "def456"

	cache.Set(path, program, hash)

	// Get by hash
	cached, exists := cache.GetByHash(hash)
	if !exists {
		t.Errorf("GetByHash() returned false for existing hash")
	}

	if cached.Path != path {
		t.Errorf("GetByHash() path = %s, want %s", cached.Path, path)
	}
}

func TestASTCache_Invalidate(t *testing.T) {
	cache := NewASTCache()

	program := &ast.Program{
		Resources: []*ast.ResourceNode{{Name: "User"}},
	}

	path := "/test/user.cdt"
	hash := "abc123"

	cache.Set(path, program, hash)

	// Verify it exists
	if _, exists := cache.Get(path); !exists {
		t.Fatalf("Entry should exist before invalidation")
	}

	// Invalidate
	cache.Invalidate(path)

	// Verify it's gone
	if _, exists := cache.Get(path); exists {
		t.Errorf("Entry should not exist after invalidation")
	}
}

func TestASTCache_InvalidateAll(t *testing.T) {
	cache := NewASTCache()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		program := &ast.Program{
			Resources: []*ast.ResourceNode{{Name: "Resource"}},
		}
		cache.Set("/test/file"+string(rune(i))+".cdt", program, "hash"+string(rune(i)))
	}

	if cache.Size() != 5 {
		t.Fatalf("Cache should have 5 entries, has %d", cache.Size())
	}

	// Invalidate all
	cache.InvalidateAll()

	if cache.Size() != 0 {
		t.Errorf("Cache should be empty after InvalidateAll(), has %d entries", cache.Size())
	}
}

func TestASTCache_Size(t *testing.T) {
	cache := NewASTCache()

	if cache.Size() != 0 {
		t.Errorf("New cache should have size 0, has %d", cache.Size())
	}

	program := &ast.Program{
		Resources: []*ast.ResourceNode{{Name: "User"}},
	}

	cache.Set("/test/user.cdt", program, "hash1")
	if cache.Size() != 1 {
		t.Errorf("Cache should have size 1, has %d", cache.Size())
	}

	cache.Set("/test/post.cdt", program, "hash2")
	if cache.Size() != 2 {
		t.Errorf("Cache should have size 2, has %d", cache.Size())
	}

	cache.Invalidate("/test/user.cdt")
	if cache.Size() != 1 {
		t.Errorf("Cache should have size 1 after invalidation, has %d", cache.Size())
	}
}

func TestASTCache_GetAll(t *testing.T) {
	cache := NewASTCache()

	program := &ast.Program{
		Resources: []*ast.ResourceNode{{Name: "User"}},
	}

	cache.Set("/test/user.cdt", program, "hash1")
	cache.Set("/test/post.cdt", program, "hash2")

	all := cache.GetAll()

	if len(all) != 2 {
		t.Errorf("GetAll() returned %d entries, want 2", len(all))
	}

	// Verify we got a copy (modifying shouldn't affect cache)
	for k := range all {
		delete(all, k)
	}

	if cache.Size() != 2 {
		t.Errorf("Cache size should still be 2 after modifying GetAll() result, has %d", cache.Size())
	}
}

func TestASTCache_Prune(t *testing.T) {
	cache := NewASTCache()

	program := &ast.Program{
		Resources: []*ast.ResourceNode{{Name: "User"}},
	}

	// Add entries with different timestamps
	cache.Set("/test/old.cdt", program, "hash1")
	time.Sleep(10 * time.Millisecond)
	cache.Set("/test/new.cdt", program, "hash2")

	// Prune entries older than 5ms (should remove old entry only)
	pruned := cache.Prune(5 * time.Millisecond)

	if pruned != 1 {
		t.Errorf("Prune() removed %d entries, expected 1 (the old entry)", pruned)
	}

	if cache.Size() != 1 {
		t.Errorf("Cache should have 1 entry after pruning, has %d", cache.Size())
	}

	// Sleep and prune again - should remove the remaining entry
	time.Sleep(20 * time.Millisecond)
	pruned = cache.Prune(10 * time.Millisecond)

	if pruned != 1 {
		t.Errorf("Prune() removed %d entries, expected 1", pruned)
	}

	if cache.Size() != 0 {
		t.Errorf("Cache should be empty after pruning, has %d entries", cache.Size())
	}
}

func TestASTCache_ConcurrentAccess(t *testing.T) {
	cache := NewASTCache()

	program := &ast.Program{
		Resources: []*ast.ResourceNode{{Name: "User"}},
	}

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			cache.Set("/test/file"+string(rune(idx))+".cdt", program, "hash"+string(rune(idx)))
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(idx int) {
			cache.Get("/test/file" + string(rune(idx)) + ".cdt")
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 entries
	if cache.Size() != 10 {
		t.Errorf("Cache should have 10 entries after concurrent access, has %d", cache.Size())
	}
}

func TestASTCache_UpdateExistingEntry(t *testing.T) {
	cache := NewASTCache()

	program1 := &ast.Program{
		Resources: []*ast.ResourceNode{{Name: "User"}},
	}

	program2 := &ast.Program{
		Resources: []*ast.ResourceNode{{Name: "UpdatedUser"}},
	}

	path := "/test/user.cdt"

	// Set initial
	cache.Set(path, program1, "hash1")

	cached, _ := cache.Get(path)
	if cached.Hash != "hash1" {
		t.Errorf("Initial hash = %s, want hash1", cached.Hash)
	}

	// Update
	cache.Set(path, program2, "hash2")

	cached, _ = cache.Get(path)
	if cached.Hash != "hash2" {
		t.Errorf("Updated hash = %s, want hash2", cached.Hash)
	}

	if len(cached.Program.Resources) == 0 || cached.Program.Resources[0].Name != "UpdatedUser" {
		t.Errorf("Program was not updated")
	}
}
