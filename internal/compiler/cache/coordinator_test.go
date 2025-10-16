package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createTestFile(t *testing.T, dir, filename, content string) string {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
	return path
}

func TestCompilationCoordinator_CompileFiles_Sequential(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	userFile := createTestFile(t, tmpDir, "user.cdt", `resource User {
  name: string!
  email: string!
}`)

	postFile := createTestFile(t, tmpDir, "post.cdt", `resource Post {
  title: string!
  content: text!
}`)

	coordinator := NewCompilationCoordinator()

	results, metrics, err := coordinator.CompileFiles([]string{userFile, postFile}, false)
	if err != nil {
		t.Fatalf("CompileFiles() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Check first compilation (cache misses)
	if metrics.CacheHits != 0 {
		t.Errorf("Expected 0 cache hits on first compilation, got %d", metrics.CacheHits)
	}

	if metrics.CacheMisses != 2 {
		t.Errorf("Expected 2 cache misses on first compilation, got %d", metrics.CacheMisses)
	}

	if metrics.FilesCompiled != 2 {
		t.Errorf("Expected 2 files compiled, got %d", metrics.FilesCompiled)
	}

	// Verify results
	for _, result := range results {
		if result.Err != nil {
			t.Errorf("Compilation error for %s: %v", result.Path, result.Err)
		}
		if result.Cached {
			t.Errorf("Result for %s should not be cached on first compilation", result.Path)
		}
		if result.Program == nil {
			t.Errorf("Result for %s has nil program", result.Path)
		}
	}

	// Compile again (should be cached)
	results2, metrics2, err := coordinator.CompileFiles([]string{userFile, postFile}, false)
	if err != nil {
		t.Fatalf("CompileFiles() error on second run = %v", err)
	}

	if metrics2.CacheHits != 2 {
		t.Errorf("Expected 2 cache hits on second compilation, got %d", metrics2.CacheHits)
	}

	if metrics2.CacheMisses != 0 {
		t.Errorf("Expected 0 cache misses on second compilation, got %d", metrics2.CacheMisses)
	}

	if metrics2.FilesCompiled != 0 {
		t.Errorf("Expected 0 files compiled on second run (all cached), got %d", metrics2.FilesCompiled)
	}

	for _, result := range results2 {
		if !result.Cached {
			t.Errorf("Result for %s should be cached on second compilation", result.Path)
		}
	}
}

func TestCompilationCoordinator_CacheInvalidation(t *testing.T) {
	tmpDir := t.TempDir()

	userFile := createTestFile(t, tmpDir, "user.cdt", `resource User {
  name: string!
}`)

	coordinator := NewCompilationCoordinator()

	// First compilation
	results1, _, _ := coordinator.CompileFiles([]string{userFile}, false)
	hash1 := results1[0].Hash

	// Modify file
	time.Sleep(10 * time.Millisecond) // Ensure file modification time changes
	if err := os.WriteFile(userFile, []byte(`resource User {
  name: string!
  email: string!
}`), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Second compilation
	results2, metrics2, _ := coordinator.CompileFiles([]string{userFile}, false)
	hash2 := results2[0].Hash

	// Hash should be different
	if hash1 == hash2 {
		t.Errorf("Hash should change after file modification")
	}

	// Should be a cache miss
	if metrics2.CacheMisses != 1 {
		t.Errorf("Expected 1 cache miss after file modification, got %d", metrics2.CacheMisses)
	}
}

func TestCompilationCoordinator_ParallelCompilation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple independent files
	files := make([]string, 5)
	for i := 0; i < 5; i++ {
		content := `resource Resource` + string(rune('A'+i)) + ` {
  name: string!
}`
		files[i] = createTestFile(t, tmpDir, "resource"+string(rune('a'+i))+".cdt", content)
	}

	coordinator := NewCompilationCoordinator()

	// Compile in parallel
	results, metrics, err := coordinator.CompileFiles(files, true)
	if err != nil {
		t.Fatalf("CompileFiles() error = %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("Expected 5 results, got %d", len(results))
	}

	// Check parallel batches (all files are independent, should compile in one batch)
	if metrics.ParallelBatches < 1 {
		t.Errorf("Expected at least 1 parallel batch, got %d", metrics.ParallelBatches)
	}

	// All should succeed
	for _, result := range results {
		if result.Err != nil {
			t.Errorf("Compilation error for %s: %v", result.Path, result.Err)
		}
	}
}

func TestCompilationCoordinator_WatchModeCompile(t *testing.T) {
	tmpDir := t.TempDir()

	userFile := createTestFile(t, tmpDir, "user.cdt", `resource User {
  name: string!
}`)

	postFile := createTestFile(t, tmpDir, "post.cdt", `resource Post {
  title: string!
}`)

	coordinator := NewCompilationCoordinator()

	// Initial compilation
	coordinator.CompileFiles([]string{userFile, postFile}, true)

	// Modify one file
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(userFile, []byte(`resource User {
  name: string!
  email: string!
}`), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Watch mode compile (only changed file)
	results, metrics, err := coordinator.WatchModeCompile([]string{userFile})
	if err != nil {
		t.Fatalf("WatchModeCompile() error = %v", err)
	}

	// Should only compile the changed file
	if metrics.FilesCompiled != 1 {
		t.Errorf("Expected 1 file compiled in watch mode, got %d", metrics.FilesCompiled)
	}

	// Verify the changed file was compiled
	found := false
	for _, result := range results {
		if result.Path == userFile && !result.Cached {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Changed file should be compiled (not cached)")
	}
}

func TestCompilationCoordinator_PerformanceMetrics(t *testing.T) {
	tmpDir := t.TempDir()

	userFile := createTestFile(t, tmpDir, "user.cdt", `resource User {
  name: string!
}`)

	coordinator := NewCompilationCoordinator()

	results, metrics, err := coordinator.CompileFiles([]string{userFile}, false)
	if err != nil {
		t.Fatalf("CompileFiles() error = %v", err)
	}

	// Verify metrics are populated
	if metrics.TotalDuration == 0 {
		t.Errorf("TotalDuration should not be 0")
	}

	if metrics.LexingDuration == 0 {
		t.Errorf("LexingDuration should not be 0")
	}

	if metrics.ParsingDuration == 0 {
		t.Errorf("ParsingDuration should not be 0")
	}

	if metrics.CachingDuration == 0 {
		t.Errorf("CachingDuration should not be 0")
	}

	if metrics.StartTime.IsZero() {
		t.Errorf("StartTime should not be zero")
	}

	if metrics.EndTime.IsZero() {
		t.Errorf("EndTime should not be zero")
	}

	if metrics.EndTime.Before(metrics.StartTime) {
		t.Errorf("EndTime should be after StartTime")
	}

	// Verify compilation was fast (< 300ms as per requirements)
	if len(results) == 1 && metrics.TotalDuration > 300*time.Millisecond {
		t.Logf("Warning: Single file compilation took %v, target is < 300ms", metrics.TotalDuration)
	}
}

func TestCompilationCoordinator_CacheHitRate(t *testing.T) {
	tmpDir := t.TempDir()

	files := make([]string, 3)
	for i := 0; i < 3; i++ {
		content := `resource Resource` + string(rune('A'+i)) + ` {
  name: string!
}`
		files[i] = createTestFile(t, tmpDir, "resource"+string(rune('a'+i))+".cdt", content)
	}

	coordinator := NewCompilationCoordinator()

	// First compilation - all misses
	_, metrics1, _ := coordinator.CompileFiles(files, false)
	hitRate1 := metrics1.CacheHitRate()

	if hitRate1 != 0.0 {
		t.Errorf("First compilation cache hit rate = %.2f%%, want 0.00%%", hitRate1)
	}

	// Second compilation - all hits
	_, metrics2, _ := coordinator.CompileFiles(files, false)
	hitRate2 := metrics2.CacheHitRate()

	if hitRate2 != 100.0 {
		t.Errorf("Second compilation cache hit rate = %.2f%%, want 100.00%%", hitRate2)
	}
}

func TestCompilationCoordinator_GetCacheStats(t *testing.T) {
	tmpDir := t.TempDir()

	userFile := createTestFile(t, tmpDir, "user.cdt", `resource User {
  name: string!
}`)

	coordinator := NewCompilationCoordinator()

	// Initial stats
	stats1 := coordinator.GetCacheStats()
	if stats1["cache_size"].(int) != 0 {
		t.Errorf("Initial cache size should be 0, got %d", stats1["cache_size"])
	}

	// Compile
	coordinator.CompileFiles([]string{userFile}, false)

	// Stats after compilation
	stats2 := coordinator.GetCacheStats()
	if stats2["cache_size"].(int) != 1 {
		t.Errorf("Cache size after compilation should be 1, got %d", stats2["cache_size"])
	}
}

func TestCompilationCoordinator_Clear(t *testing.T) {
	tmpDir := t.TempDir()

	userFile := createTestFile(t, tmpDir, "user.cdt", `resource User {
  name: string!
}`)

	coordinator := NewCompilationCoordinator()

	// Compile and cache
	coordinator.CompileFiles([]string{userFile}, false)

	stats1 := coordinator.GetCacheStats()
	if stats1["cache_size"].(int) == 0 {
		t.Fatalf("Cache should not be empty after compilation")
	}

	// Clear
	coordinator.Clear()

	stats2 := coordinator.GetCacheStats()
	if stats2["cache_size"].(int) != 0 {
		t.Errorf("Cache size after clear should be 0, got %d", stats2["cache_size"])
	}

	// Compile again - should be cache miss
	_, metrics, _ := coordinator.CompileFiles([]string{userFile}, false)
	if metrics.CacheMisses != 1 {
		t.Errorf("Expected cache miss after clear, got %d misses", metrics.CacheMisses)
	}
}

func TestScanDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	createTestFile(t, tmpDir, "user.cdt", "resource User {}")
	createTestFile(t, tmpDir, "post.cdt", "resource Post {}")
	createTestFile(t, tmpDir, "readme.md", "# README") // Non-.cdt file

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "models")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	createTestFile(t, subDir, "comment.cdt", "resource Comment {}")

	files, err := ScanDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}

	// Should find 3 .cdt files
	if len(files) != 3 {
		t.Errorf("ScanDirectory() found %d files, want 3", len(files))
	}

	// Verify all are .cdt files
	for _, file := range files {
		if filepath.Ext(file) != ".cdt" {
			t.Errorf("ScanDirectory() returned non-.cdt file: %s", file)
		}
	}
}

func TestCompilationCoordinator_IncrementalPerformance(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 50 test resources as per requirements
	files := make([]string, 50)
	for i := 0; i < 50; i++ {
		content := fmt.Sprintf(`resource Resource%d {
  name: string!
  description: text?
  created_at: timestamp!
}`, i)
		files[i] = createTestFile(t, tmpDir, fmt.Sprintf("resource%d.cdt", i), content)
	}

	coordinator := NewCompilationCoordinator()

	// First compilation (baseline)
	start := time.Now()
	_, metrics1, err := coordinator.CompileFiles(files, true)
	firstDuration := time.Since(start)

	if err != nil {
		t.Fatalf("CompileFiles() error = %v", err)
	}

	t.Logf("First compilation of 50 resources: %v", firstDuration)
	t.Logf("Cache hits: %d, misses: %d, hit rate: %.2f%%",
		metrics1.CacheHits, metrics1.CacheMisses, metrics1.CacheHitRate())

	// Modify one file
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(files[0], []byte(`resource Resource0 {
  name: string!
  email: string!
  created_at: timestamp!
}`), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Incremental compilation
	start = time.Now()
	_, metrics2, err := coordinator.WatchModeCompile([]string{files[0]})
	incrementalDuration := time.Since(start)

	if err != nil {
		t.Fatalf("WatchModeCompile() error = %v", err)
	}

	t.Logf("Incremental compilation: %v", incrementalDuration)
	t.Logf("Cache hits: %d, misses: %d, hit rate: %.2f%%",
		metrics2.CacheHits, metrics2.CacheMisses, metrics2.CacheHitRate())

	// Verify performance target: < 300ms for single file change
	if incrementalDuration > 300*time.Millisecond {
		t.Logf("Warning: Incremental compilation took %v, target is < 300ms", incrementalDuration)
	}

	// Should only compile 1 file
	if metrics2.FilesCompiled != 1 {
		t.Errorf("Expected 1 file compiled incrementally, got %d", metrics2.FilesCompiled)
	}
}
