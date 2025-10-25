package build

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestNewCache(t *testing.T) {
	cacheDir := t.TempDir()

	cache, err := NewCache(cacheDir)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}

	if cache.cacheDir != cacheDir {
		t.Errorf("Expected cache dir %q, got %q", cacheDir, cache.cacheDir)
	}

	if cache.entries == nil {
		t.Error("Expected non-nil entries map")
	}

	// Verify cache directory was created
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Expected cache directory to be created")
	}
}

func TestCachePutGet(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create test file
	tmpFile := filepath.Join(t.TempDir(), "test.cdt")
	content := "resource Test {}"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Compute hash
	hash, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	// Create compiled file
	compiled := &CompiledFile{
		Path: tmpFile,
		Hash: hash,
		Program: &ast.Program{
			Resources: []*ast.ResourceNode{
				{Name: "Test"},
			},
		},
	}

	// Put in cache
	if err := cache.Put(tmpFile, compiled); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Give it a moment for async persist
	time.Sleep(100 * time.Millisecond)

	// Get from cache
	got, err := cache.Get(tmpFile)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got == nil {
		t.Fatal("Expected non-nil result")
	}

	if got.Path != tmpFile {
		t.Errorf("Expected path %q, got %q", tmpFile, got.Path)
	}

	if got.Hash != hash {
		t.Errorf("Expected hash %q, got %q", hash, got.Hash)
	}
}

func TestCacheGetInvalidated(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create test file
	tmpFile := filepath.Join(t.TempDir(), "test.cdt")
	content := "resource Test {}"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Compute initial hash
	hash, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	// Put in cache
	compiled := &CompiledFile{
		Path: tmpFile,
		Hash: hash,
		Program: &ast.Program{
			Resources: []*ast.ResourceNode{{Name: "Test"}},
		},
	}

	if err := cache.Put(tmpFile, compiled); err != nil {
		t.Fatal(err)
	}

	// Modify file (invalidate cache)
	if err := os.WriteFile(tmpFile, []byte("resource Updated {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get should fail since file changed
	_, err = cache.Get(tmpFile)
	if err == nil {
		t.Error("Expected error getting invalidated cache entry")
	}
}

func TestCacheInvalidate(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create test file
	tmpFile := filepath.Join(t.TempDir(), "test.cdt")
	if err := os.WriteFile(tmpFile, []byte("resource Test {}"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	// Put in cache
	compiled := &CompiledFile{
		Path:    tmpFile,
		Hash:    hash,
		Program: &ast.Program{},
	}

	if err := cache.Put(tmpFile, compiled); err != nil {
		t.Fatal(err)
	}

	// Invalidate
	cache.Invalidate(tmpFile)

	// Get should fail
	_, err = cache.Get(tmpFile)
	if err == nil {
		t.Error("Expected error after invalidation")
	}
}

func TestCacheClear(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	// Add some entries
	for i := 0; i < 5; i++ {
		tmpFile := filepath.Join(t.TempDir(), fmt.Sprintf("test%d.cdt", i))
		if err := os.WriteFile(tmpFile, []byte("resource Test {}"), 0644); err != nil {
			t.Fatal(err)
		}

		hash, err := computeFileHash(tmpFile)
		if err != nil {
			t.Fatal(err)
		}

		compiled := &CompiledFile{
			Path:    tmpFile,
			Hash:    hash,
			Program: &ast.Program{},
		}

		if err := cache.Put(tmpFile, compiled); err != nil {
			t.Fatal(err)
		}
	}

	// Verify entries exist
	stats := cache.Stats()
	if stats.TotalEntries != 5 {
		t.Errorf("Expected 5 entries, got %d", stats.TotalEntries)
	}

	// Clear cache
	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify entries were cleared
	stats = cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.TotalEntries)
	}

	// Verify cache directory still exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Expected cache directory to exist after clear")
	}
}

func TestCacheStats(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	// Initially empty
	stats := cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries, got %d", stats.TotalEntries)
	}

	// Add entry
	tmpFile := filepath.Join(t.TempDir(), "test.cdt")
	if err := os.WriteFile(tmpFile, []byte("resource Test {}"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	compiled := &CompiledFile{
		Path:    tmpFile,
		Hash:    hash,
		Program: &ast.Program{},
	}

	if err := cache.Put(tmpFile, compiled); err != nil {
		t.Fatal(err)
	}

	// Check stats
	stats = cache.Stats()
	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 entry, got %d", stats.TotalEntries)
	}

	if stats.TotalSize == 0 {
		t.Error("Expected non-zero total size")
	}
}

func TestCachePersistAndLoad(t *testing.T) {
	cacheDir := t.TempDir()

	// Create cache and add entry
	cache1, err := NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.cdt")
	if err := os.WriteFile(tmpFile, []byte("resource Test {}"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	compiled := &CompiledFile{
		Path:    tmpFile,
		Hash:    hash,
		Program: &ast.Program{},
	}

	if err := cache1.Put(tmpFile, compiled); err != nil {
		t.Fatal(err)
	}

	// Force persist
	if err := cache1.persist(); err != nil {
		t.Fatalf("Persist failed: %v", err)
	}

	// Create new cache from same directory
	cache2, err := NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify entry was loaded
	stats := cache2.Stats()
	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 entry after load, got %d", stats.TotalEntries)
	}

	// Try to get the entry (it should fail since file hash check)
	// but at least verify it's in the cache
	cache2.mu.RLock()
	_, exists := cache2.entries[tmpFile]
	cache2.mu.RUnlock()

	if !exists {
		t.Error("Expected entry to be loaded from cache")
	}
}

func TestCacheEntry_Fields(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.cdt")
	if err := os.WriteFile(tmpFile, []byte("resource Test {}"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	entry := &CacheEntry{
		FilePath: tmpFile,
		FileHash: hash,
		Compiled: &CompiledFile{
			Path:    tmpFile,
			Hash:    hash,
			Program: &ast.Program{},
		},
		Dependencies: []string{"dep1.cdt", "dep2.cdt"},
		CachedAt:     time.Now(),
	}

	if entry.FilePath != tmpFile {
		t.Errorf("Expected file path %q, got %q", tmpFile, entry.FilePath)
	}

	if entry.FileHash != hash {
		t.Errorf("Expected hash %q, got %q", hash, entry.FileHash)
	}

	if len(entry.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(entry.Dependencies))
	}

	if entry.Compiled == nil {
		t.Error("Expected non-nil compiled file")
	}

	if entry.CachedAt.IsZero() {
		t.Error("Expected non-zero cached time")
	}
}

func BenchmarkCachePut(b *testing.B) {
	cacheDir := b.TempDir()
	cache, err := NewCache(cacheDir)
	if err != nil {
		b.Fatal(err)
	}

	tmpFile := filepath.Join(b.TempDir(), "test.cdt")
	if err := os.WriteFile(tmpFile, []byte("resource Test {}"), 0644); err != nil {
		b.Fatal(err)
	}

	hash, err := computeFileHash(tmpFile)
	if err != nil {
		b.Fatal(err)
	}

	compiled := &CompiledFile{
		Path:    tmpFile,
		Hash:    hash,
		Program: &ast.Program{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cache.Put(tmpFile, compiled); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cacheDir := b.TempDir()
	cache, err := NewCache(cacheDir)
	if err != nil {
		b.Fatal(err)
	}

	tmpFile := filepath.Join(b.TempDir(), "test.cdt")
	if err := os.WriteFile(tmpFile, []byte("resource Test {}"), 0644); err != nil {
		b.Fatal(err)
	}

	hash, err := computeFileHash(tmpFile)
	if err != nil {
		b.Fatal(err)
	}

	compiled := &CompiledFile{
		Path:    tmpFile,
		Hash:    hash,
		Program: &ast.Program{},
	}

	if err := cache.Put(tmpFile, compiled); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := cache.Get(tmpFile); err != nil {
			b.Fatal(err)
		}
	}
}
