package build

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadState_NewFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Load state from non-existent file
	state, err := LoadState(tmpDir)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Should return empty state
	if state.FileHashes == nil {
		t.Error("FileHashes should be initialized")
	}
	if len(state.FileHashes) != 0 {
		t.Errorf("Expected empty FileHashes, got %d entries", len(state.FileHashes))
	}
}

func TestSaveAndLoadState(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a build state
	originalState := &BuildState{
		FileHashes: map[string]string{
			"/path/to/file1.cdt": "hash1",
			"/path/to/file2.cdt": "hash2",
		},
		BinaryPath: "build/app",
		BuildOptions: BuildOptionsSnapshot{
			Mode:       "development",
			OutputPath: "build/app",
			Minify:     false,
			TreeShake:  false,
			UseCache:   true,
		},
		LastBuildTime: time.Now(),
		Version:       "0.1.0",
	}

	// Save state
	if err := originalState.SaveState(tmpDir); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Load state
	loadedState, err := LoadState(tmpDir)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Verify loaded state matches original
	if len(loadedState.FileHashes) != len(originalState.FileHashes) {
		t.Errorf("Expected %d file hashes, got %d", len(originalState.FileHashes), len(loadedState.FileHashes))
	}

	for path, hash := range originalState.FileHashes {
		if loadedState.FileHashes[path] != hash {
			t.Errorf("Hash mismatch for %s: expected %s, got %s", path, hash, loadedState.FileHashes[path])
		}
	}

	if loadedState.BinaryPath != originalState.BinaryPath {
		t.Errorf("BinaryPath mismatch: expected %s, got %s", originalState.BinaryPath, loadedState.BinaryPath)
	}

	if loadedState.BuildOptions != originalState.BuildOptions {
		t.Errorf("BuildOptions mismatch: expected %+v, got %+v", originalState.BuildOptions, loadedState.BuildOptions)
	}

	if loadedState.Version != originalState.Version {
		t.Errorf("Version mismatch: expected %s, got %s", originalState.Version, loadedState.Version)
	}
}

func TestNeedsRebuild_BinaryMissing(t *testing.T) {
	state := &BuildState{
		FileHashes: make(map[string]string),
	}

	opts := &BuildOptions{
		Mode:       ModeDevelopment,
		OutputPath: "/nonexistent/binary",
	}

	needsRebuild, changedFiles, reason := state.NeedsRebuild([]string{}, opts)

	if !needsRebuild {
		t.Error("Expected rebuild needed when binary is missing")
	}

	if reason != "binary does not exist" {
		t.Errorf("Expected reason 'binary does not exist', got '%s'", reason)
	}

	if len(changedFiles) != 0 {
		t.Errorf("Expected 0 changed files, got %d", len(changedFiles))
	}
}

func TestNeedsRebuild_BuildOptionsChanged(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "app")

	// Create a dummy binary
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("Failed to create dummy binary: %v", err)
	}

	state := &BuildState{
		FileHashes: make(map[string]string),
		BuildOptions: BuildOptionsSnapshot{
			Mode:       "development",
			OutputPath: binaryPath,
			Minify:     false,
		},
	}

	opts := &BuildOptions{
		Mode:       ModeProduction, // Changed from development
		OutputPath: binaryPath,
		Minify:     false,
	}

	needsRebuild, _, reason := state.NeedsRebuild([]string{}, opts)

	if !needsRebuild {
		t.Error("Expected rebuild needed when build options change")
	}

	if reason != "build options changed" {
		t.Errorf("Expected reason 'build options changed', got '%s'", reason)
	}
}

func TestNeedsRebuild_FileChanged(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "app")
	testFile := filepath.Join(tmpDir, "test.cdt")

	// Create a dummy binary
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("Failed to create dummy binary: %v", err)
	}

	// Create a test file
	if err := os.WriteFile(testFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute original hash
	originalHash, err := computeHash(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	state := &BuildState{
		FileHashes: map[string]string{
			testFile: originalHash,
		},
		BuildOptions: BuildOptionsSnapshot{
			Mode:       "development",
			OutputPath: binaryPath,
		},
	}

	opts := &BuildOptions{
		Mode:       ModeDevelopment,
		OutputPath: binaryPath,
	}

	// Modify the file
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	needsRebuild, changedFiles, reason := state.NeedsRebuild([]string{testFile}, opts)

	if !needsRebuild {
		t.Error("Expected rebuild needed when file changes")
	}

	if len(changedFiles) != 1 {
		t.Errorf("Expected 1 changed file, got %d", len(changedFiles))
	}

	if len(changedFiles) > 0 && changedFiles[0] != testFile {
		t.Errorf("Expected changed file to be %s, got %s", testFile, changedFiles[0])
	}

	if reason != "1 file(s) changed" {
		t.Errorf("Expected reason '1 file(s) changed', got '%s'", reason)
	}
}

func TestNeedsRebuild_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "app")
	testFile := filepath.Join(tmpDir, "test.cdt")

	// Create a dummy binary
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("Failed to create dummy binary: %v", err)
	}

	// Create a test file
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	state := &BuildState{
		FileHashes: make(map[string]string), // Empty - file is new
		BuildOptions: BuildOptionsSnapshot{
			Mode:       "development",
			OutputPath: binaryPath,
		},
	}

	opts := &BuildOptions{
		Mode:       ModeDevelopment,
		OutputPath: binaryPath,
	}

	needsRebuild, changedFiles, reason := state.NeedsRebuild([]string{testFile}, opts)

	if !needsRebuild {
		t.Error("Expected rebuild needed when new file is added")
	}

	if len(changedFiles) != 1 {
		t.Errorf("Expected 1 changed file, got %d", len(changedFiles))
	}

	if reason != "1 file(s) changed" {
		t.Errorf("Expected reason '1 file(s) changed', got '%s'", reason)
	}
}

func TestNeedsRebuild_FileDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "app")

	// Create a dummy binary
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("Failed to create dummy binary: %v", err)
	}

	state := &BuildState{
		FileHashes: map[string]string{
			"/deleted/file.cdt": "somehash",
		},
		BuildOptions: BuildOptionsSnapshot{
			Mode:       "development",
			OutputPath: binaryPath,
		},
	}

	opts := &BuildOptions{
		Mode:       ModeDevelopment,
		OutputPath: binaryPath,
	}

	needsRebuild, _, reason := state.NeedsRebuild([]string{}, opts)

	if !needsRebuild {
		t.Error("Expected rebuild needed when file is deleted")
	}

	if reason != "file deleted: /deleted/file.cdt" {
		t.Errorf("Expected reason about deleted file, got '%s'", reason)
	}
}

func TestNeedsRebuild_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "app")
	testFile := filepath.Join(tmpDir, "test.cdt")

	// Create a dummy binary
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("Failed to create dummy binary: %v", err)
	}

	// Create a test file
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute hash
	hash, err := computeHash(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	state := &BuildState{
		FileHashes: map[string]string{
			testFile: hash,
		},
		BuildOptions: BuildOptionsSnapshot{
			Mode:       "development",
			OutputPath: binaryPath,
		},
	}

	opts := &BuildOptions{
		Mode:       ModeDevelopment,
		OutputPath: binaryPath,
	}

	needsRebuild, changedFiles, reason := state.NeedsRebuild([]string{testFile}, opts)

	if needsRebuild {
		t.Errorf("Expected no rebuild needed, but got rebuild with reason: %s", reason)
	}

	if len(changedFiles) != 0 {
		t.Errorf("Expected 0 changed files, got %d", len(changedFiles))
	}

	if reason != "" {
		t.Errorf("Expected empty reason, got '%s'", reason)
	}
}

func TestUpdateFromBuild(t *testing.T) {
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.cdt")
	testFile2 := filepath.Join(tmpDir, "test2.cdt")

	// Create test files
	if err := os.WriteFile(testFile1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	state := &BuildState{
		FileHashes: make(map[string]string),
	}

	opts := &BuildOptions{
		Mode:       ModeProduction,
		OutputPath: "build/app",
		Minify:     true,
		TreeShake:  true,
		UseCache:   true,
	}

	sourceFiles := []string{testFile1, testFile2}

	if err := state.UpdateFromBuild(sourceFiles, opts); err != nil {
		t.Fatalf("UpdateFromBuild failed: %v", err)
	}

	// Verify file hashes were computed
	if len(state.FileHashes) != 2 {
		t.Errorf("Expected 2 file hashes, got %d", len(state.FileHashes))
	}

	if _, exists := state.FileHashes[testFile1]; !exists {
		t.Error("Expected hash for testFile1")
	}
	if _, exists := state.FileHashes[testFile2]; !exists {
		t.Error("Expected hash for testFile2")
	}

	// Verify build options were saved
	if state.BuildOptions.Mode != "production" {
		t.Errorf("Expected mode 'production', got '%s'", state.BuildOptions.Mode)
	}
	if !state.BuildOptions.Minify {
		t.Error("Expected Minify to be true")
	}
	if !state.BuildOptions.TreeShake {
		t.Error("Expected TreeShake to be true")
	}

	// Verify binary path was saved
	if state.BinaryPath != "build/app" {
		t.Errorf("Expected binary path 'build/app', got '%s'", state.BinaryPath)
	}

	// Verify timestamp was set
	if state.LastBuildTime.IsZero() {
		t.Error("Expected LastBuildTime to be set")
	}

	// Verify version was set
	if state.Version == "" {
		t.Error("Expected Version to be set")
	}
}

func TestComputeHash_Consistency(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.cdt")

	content := []byte("test content for hashing")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute hash twice
	hash1, err := computeHash(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash 1: %v", err)
	}

	hash2, err := computeHash(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash 2: %v", err)
	}

	// Hashes should be identical
	if hash1 != hash2 {
		t.Errorf("Hash inconsistency: %s != %s", hash1, hash2)
	}

	// Hash should be non-empty
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}

	// Hash should be hex string (64 chars for SHA-256)
	if len(hash1) != 64 {
		t.Errorf("Expected 64-character SHA-256 hash, got %d characters", len(hash1))
	}
}

func TestComputeHash_DifferentContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.cdt")

	// Create file with first content
	if err := os.WriteFile(testFile, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash1, err := computeHash(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash 1: %v", err)
	}

	// Modify file content
	if err := os.WriteFile(testFile, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	hash2, err := computeHash(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash 2: %v", err)
	}

	// Hashes should be different
	if hash1 == hash2 {
		t.Error("Expected different hashes for different content")
	}
}
