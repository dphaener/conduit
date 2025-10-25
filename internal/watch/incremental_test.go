package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIncrementalCompiler_IncrementalBuild(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "incremental-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app directory
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create test file
	testFile := filepath.Join(appDir, "post.cdt")
	content := `
resource Post {
  id: uuid! @primary @auto
  title: string!
  content: text!
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	compiler := NewIncrementalCompiler()

	// First build
	result, err := compiler.IncrementalBuild([]string{testFile})
	if err != nil {
		t.Fatalf("First build failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected first build to succeed")
	}

	if result.Duration == 0 {
		t.Error("Expected duration to be set")
	}

	// Check cache
	if len(compiler.resourceCache) != 1 {
		t.Errorf("Expected 1 cached file, got %d", len(compiler.resourceCache))
	}

	// Second build (same file)
	result2, err := compiler.IncrementalBuild([]string{testFile})
	if err != nil {
		t.Fatalf("Second build failed: %v", err)
	}

	if !result2.Success {
		t.Error("Expected second build to succeed")
	}
}

func TestIncrementalCompiler_CompileError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "incremental-error-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	appDir := filepath.Join(tmpDir, "app")
	os.MkdirAll(appDir, 0755)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create file with syntax error
	testFile := filepath.Join(appDir, "bad.cdt")
	badContent := `
resource Post {
  id: uuid! @primary
  title: // missing type
}
`
	os.WriteFile(testFile, []byte(badContent), 0644)

	compiler := NewIncrementalCompiler()

	result, err := compiler.IncrementalBuild([]string{testFile})
	if err == nil {
		t.Error("Expected build to fail with syntax error")
	}

	if result.Success {
		t.Error("Expected result.Success to be false")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors to be reported")
	}
}

func TestIncrementalCompiler_FullBuild(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "full-build-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	appDir := filepath.Join(tmpDir, "app")
	os.MkdirAll(appDir, 0755)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create multiple files
	files := []string{"post.cdt", "user.cdt", "comment.cdt"}
	for _, file := range files {
		content := `
resource Test {
  id: uuid! @primary @auto
  name: string!
}
`
		os.WriteFile(filepath.Join(appDir, file), []byte(content), 0644)
	}

	compiler := NewIncrementalCompiler()

	result, err := compiler.FullBuild()
	if err != nil {
		t.Fatalf("Full build failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected full build to succeed")
	}

	// Should have cached all files
	if len(compiler.resourceCache) != len(files) {
		t.Errorf("Expected %d cached files, got %d", len(files), len(compiler.resourceCache))
	}
}

func TestIncrementalCompiler_ClearCache(t *testing.T) {
	compiler := NewIncrementalCompiler()

	// Add some dummy cache entries
	compiler.resourceCache["file1.cdt"] = nil
	compiler.resourceCache["file2.cdt"] = nil

	if len(compiler.resourceCache) != 2 {
		t.Fatalf("Expected 2 cache entries, got %d", len(compiler.resourceCache))
	}

	compiler.ClearCache()

	if len(compiler.resourceCache) != 0 {
		t.Errorf("Expected cache to be cleared, got %d entries", len(compiler.resourceCache))
	}
}

func TestIncrementalCompiler_NonCdtFiles(t *testing.T) {
	compiler := NewIncrementalCompiler()

	// Should ignore non-.cdt files
	result, err := compiler.IncrementalBuild([]string{"test.css", "test.js"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("Expected success for non-.cdt files")
	}

	if len(result.ChangedFiles) != 2 {
		t.Errorf("Expected 2 changed files, got %d", len(result.ChangedFiles))
	}
}

func TestCompileResult_Duration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "duration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	appDir := filepath.Join(tmpDir, "app")
	os.MkdirAll(appDir, 0755)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	testFile := filepath.Join(appDir, "post.cdt")
	content := `
resource Post {
  id: uuid! @primary @auto
  title: string!
}
`
	os.WriteFile(testFile, []byte(content), 0644)

	compiler := NewIncrementalCompiler()

	start := time.Now()
	result, _ := compiler.IncrementalBuild([]string{testFile})
	elapsed := time.Since(start)

	if result.Duration == 0 {
		t.Error("Expected duration to be set")
	}

	if result.Duration > elapsed+time.Millisecond {
		t.Error("Result duration should not exceed actual elapsed time")
	}
}

func BenchmarkIncrementalCompiler_Build(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-test-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	appDir := filepath.Join(tmpDir, "app")
	os.MkdirAll(appDir, 0755)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	testFile := filepath.Join(appDir, "post.cdt")
	content := `
resource Post {
  id: uuid! @primary @auto
  title: string!
  content: text!
}
`
	os.WriteFile(testFile, []byte(content), 0644)

	compiler := NewIncrementalCompiler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compiler.IncrementalBuild([]string{testFile})
	}
}
