package build

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSystem(t *testing.T) {
	opts := DefaultBuildOptions()
	opts.BuildDir = t.TempDir()

	sys, err := NewSystem(opts)
	if err != nil {
		t.Fatalf("NewSystem failed: %v", err)
	}

	if sys == nil {
		t.Fatal("Expected non-nil system")
	}

	if sys.options.Mode != ModeDevelopment {
		t.Errorf("Expected development mode, got %v", sys.options.Mode)
	}
}

func TestBuildMode_String(t *testing.T) {
	tests := []struct {
		mode BuildMode
		want string
	}{
		{ModeDevelopment, "development"},
		{ModeProduction, "production"},
		{ModeTest, "test"},
		{BuildMode(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.mode.String()
		if got != tt.want {
			t.Errorf("BuildMode.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestFindSourceFiles(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	testFiles := []string{"user.cdt", "post.cdt", "comment.cdt"}
	for _, name := range testFiles {
		path := filepath.Join(sourceDir, name)
		if err := os.WriteFile(path, []byte("resource Test {}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create build system
	opts := DefaultBuildOptions()
	opts.SourceDir = sourceDir
	opts.BuildDir = filepath.Join(tmpDir, "build")

	sys, err := NewSystem(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Find files
	files, err := sys.findSourceFiles()
	if err != nil {
		t.Fatalf("findSourceFiles failed: %v", err)
	}

	if len(files) != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), len(files))
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test file
	testFile := filepath.Join(sourceDir, "test.cdt")
	if err := os.WriteFile(testFile, []byte("resource Test {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create build system
	opts := DefaultBuildOptions()
	opts.SourceDir = sourceDir
	opts.BuildDir = filepath.Join(tmpDir, "build")

	sys, err := NewSystem(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Build dependency graph
	files := []string{testFile}
	if err := sys.buildDependencyGraph(files); err != nil {
		t.Fatalf("buildDependencyGraph failed: %v", err)
	}

	// Verify node was added
	node, ok := sys.depGraph.GetNode(testFile)
	if !ok {
		t.Fatal("Expected node to be in graph")
	}

	if node.Path != testFile {
		t.Errorf("Expected path %q, got %q", testFile, node.Path)
	}
}

func TestCompileFile(t *testing.T) {
	// Create temporary file
	tmpFile := filepath.Join(t.TempDir(), "test.cdt")
	source := `resource User {
		id: uuid! @primary @auto
		name: string!
	}`

	if err := os.WriteFile(tmpFile, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	// Create build system
	opts := DefaultBuildOptions()
	opts.BuildDir = t.TempDir()

	sys, err := NewSystem(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Compile file
	compiled, err := sys.compileFile(tmpFile)
	if err != nil {
		t.Fatalf("compileFile failed: %v", err)
	}

	if compiled == nil {
		t.Fatal("Expected non-nil compiled file")
	}

	if compiled.Path != tmpFile {
		t.Errorf("Expected path %q, got %q", tmpFile, compiled.Path)
	}

	if compiled.Hash == "" {
		t.Error("Expected non-empty hash")
	}

	if compiled.Program == nil {
		t.Fatal("Expected non-nil program")
	}

	if len(compiled.Program.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(compiled.Program.Resources))
	}
}

func TestBuildOptions_DefaultValues(t *testing.T) {
	opts := DefaultBuildOptions()

	if opts.Mode != ModeDevelopment {
		t.Errorf("Expected development mode, got %v", opts.Mode)
	}

	if opts.OutputPath != "build/app" {
		t.Errorf("Expected output path 'build/app', got %q", opts.OutputPath)
	}

	if opts.SourceDir != "app" {
		t.Errorf("Expected source dir 'app', got %q", opts.SourceDir)
	}

	if opts.BuildDir != "build" {
		t.Errorf("Expected build dir 'build', got %q", opts.BuildDir)
	}

	if !opts.Parallel {
		t.Error("Expected parallel to be true")
	}

	if !opts.UseCache {
		t.Error("Expected UseCache to be true")
	}

	if opts.Minify {
		t.Error("Expected Minify to be false")
	}

	if opts.TreeShake {
		t.Error("Expected TreeShake to be false")
	}
}

func TestBuildResult_SuccessFields(t *testing.T) {
	result := &BuildResult{
		Success:       true,
		OutputPath:    "build/app",
		MetadataPath:  "build/app.meta.json",
		Duration:      1500 * time.Millisecond,
		FilesCompiled: 5,
		CacheHits:     2,
		Errors:        nil,
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.Duration < time.Second {
		t.Errorf("Expected duration >= 1s, got %v", result.Duration)
	}

	if result.FilesCompiled != 5 {
		t.Errorf("Expected 5 files compiled, got %d", result.FilesCompiled)
	}

	if result.CacheHits != 2 {
		t.Errorf("Expected 2 cache hits, got %d", result.CacheHits)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(result.Errors))
	}
}

func TestBuildError_Fields(t *testing.T) {
	err := BuildError{
		Phase:   "compilation",
		File:    "test.cdt",
		Line:    10,
		Column:  5,
		Message: "unexpected token",
	}

	if err.Phase != "compilation" {
		t.Errorf("Expected phase 'compilation', got %q", err.Phase)
	}

	if err.File != "test.cdt" {
		t.Errorf("Expected file 'test.cdt', got %q", err.File)
	}

	if err.Line != 10 {
		t.Errorf("Expected line 10, got %d", err.Line)
	}

	if err.Column != 5 {
		t.Errorf("Expected column 5, got %d", err.Column)
	}

	if err.Message != "unexpected token" {
		t.Errorf("Expected message 'unexpected token', got %q", err.Message)
	}
}

func TestExtractDependencies(t *testing.T) {
	opts := DefaultBuildOptions()
	opts.BuildDir = t.TempDir()

	sys, err := NewSystem(opts)
	if err != nil {
		t.Fatal(err)
	}

	source := `resource User {
		id: uuid! @primary @auto
	}`

	deps := sys.extractDependencies(source)

	// Currently returns empty since Conduit doesn't have explicit imports
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}
}

func TestProgressCallback(t *testing.T) {
	called := false
	var lastCurrent, lastTotal int
	var lastMessage string

	progressFunc := func(current, total int, message string) {
		called = true
		lastCurrent = current
		lastTotal = total
		lastMessage = message
	}

	opts := DefaultBuildOptions()
	opts.ProgressFunc = progressFunc
	opts.BuildDir = t.TempDir()

	if opts.ProgressFunc == nil {
		t.Fatal("Expected progress func to be set")
	}

	// Call it
	opts.ProgressFunc(5, 10, "test message")

	if !called {
		t.Error("Expected progress func to be called")
	}

	if lastCurrent != 5 {
		t.Errorf("Expected current = 5, got %d", lastCurrent)
	}

	if lastTotal != 10 {
		t.Errorf("Expected total = 10, got %d", lastTotal)
	}

	if lastMessage != "test message" {
		t.Errorf("Expected message 'test message', got %q", lastMessage)
	}
}

func BenchmarkCompileFile(b *testing.B) {
	tmpFile := filepath.Join(b.TempDir(), "test.cdt")
	source := `resource User {
		id: uuid! @primary @auto
		name: string!
		email: email! @unique
	}`

	if err := os.WriteFile(tmpFile, []byte(source), 0644); err != nil {
		b.Fatal(err)
	}

	opts := DefaultBuildOptions()
	opts.BuildDir = b.TempDir()

	sys, err := NewSystem(opts)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sys.compileFile(tmpFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildDependencyGraph(b *testing.B) {
	tmpDir := b.TempDir()
	sourceDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		b.Fatal(err)
	}

	// Create test files
	files := make([]string, 10)
	for i := 0; i < 10; i++ {
		path := filepath.Join(sourceDir, fmt.Sprintf("test%d.cdt", i))
		if err := os.WriteFile(path, []byte("resource Test {}"), 0644); err != nil {
			b.Fatal(err)
		}
		files[i] = path
	}

	opts := DefaultBuildOptions()
	opts.SourceDir = sourceDir
	opts.BuildDir = filepath.Join(tmpDir, "build")

	sys, err := NewSystem(opts)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := sys.buildDependencyGraph(files); err != nil {
			b.Fatal(err)
		}
	}
}
