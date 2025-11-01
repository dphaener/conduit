package build

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BuildState tracks file hashes and build metadata for incremental builds
type BuildState struct {
	// FileHashes maps absolute file paths to their SHA-256 hashes
	FileHashes map[string]string `json:"file_hashes"`
	// BinaryPath is the path to the built binary
	BinaryPath string `json:"binary_path"`
	// BuildOptions tracks the build configuration that affects output
	BuildOptions BuildOptionsSnapshot `json:"build_options"`
	// LastBuildTime records when the build completed
	LastBuildTime time.Time `json:"last_build_time"`
	// Version tracks the conduit version used for the build
	Version string `json:"version"`
}

// BuildOptionsSnapshot captures build options that affect the output
type BuildOptionsSnapshot struct {
	Mode         string `json:"mode"`
	OutputPath   string `json:"output_path"`
	Minify       bool   `json:"minify"`
	TreeShake    bool   `json:"tree_shake"`
	UseCache     bool   `json:"use_cache"`
}

// LoadState loads build state from .conduit/build-state.json
func LoadState(buildDir string) (*BuildState, error) {
	// Get project root (where build/ directory lives)
	projectRoot := filepath.Dir(buildDir)
	statePath := filepath.Join(projectRoot, ".conduit", "build-state.json")

	file, err := os.Open(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty state if file doesn't exist
			return &BuildState{
				FileHashes: make(map[string]string),
			}, nil
		}
		return nil, fmt.Errorf("failed to open build state: %w", err)
	}
	defer file.Close()

	var state BuildState
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode build state: %w", err)
	}

	// Initialize map if nil (backwards compatibility)
	if state.FileHashes == nil {
		state.FileHashes = make(map[string]string)
	}

	return &state, nil
}

// SaveState persists build state to .conduit/build-state.json
func (s *BuildState) SaveState(buildDir string) error {
	// Get project root (where build/ directory lives)
	projectRoot := filepath.Dir(buildDir)
	conduitDir := filepath.Join(projectRoot, ".conduit")

	// Ensure .conduit directory exists
	if err := os.MkdirAll(conduitDir, 0755); err != nil {
		return fmt.Errorf("failed to create .conduit directory: %w", err)
	}

	statePath := filepath.Join(conduitDir, "build-state.json")

	// Create temporary file for atomic write
	tmpPath := statePath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp state file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to encode build state: %w", err)
	}

	file.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, statePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save build state: %w", err)
	}

	return nil
}

// NeedsRebuild determines if a rebuild is needed based on current state
// Returns: (needsRebuild bool, changedFiles []string, reason string)
func (s *BuildState) NeedsRebuild(sourceFiles []string, opts *BuildOptions) (bool, []string, string) {
	// Check if binary exists
	if _, err := os.Stat(opts.OutputPath); os.IsNotExist(err) {
		return true, sourceFiles, "binary does not exist"
	}

	// Check if build options changed
	currentOpts := BuildOptionsSnapshot{
		Mode:       opts.Mode.String(),
		OutputPath: opts.OutputPath,
		Minify:     opts.Minify,
		TreeShake:  opts.TreeShake,
		UseCache:   opts.UseCache,
	}

	if s.BuildOptions != currentOpts {
		return true, sourceFiles, "build options changed"
	}

	// Check for new or modified files
	changedFiles := make([]string, 0)

	for _, file := range sourceFiles {
		currentHash, err := computeHash(file)
		if err != nil {
			// If we can't compute hash, assume file changed
			changedFiles = append(changedFiles, file)
			continue
		}

		cachedHash, exists := s.FileHashes[file]
		if !exists || cachedHash != currentHash {
			changedFiles = append(changedFiles, file)
		}
	}

	// Check for deleted files - O(n) instead of O(n²)
	sourceFileSet := make(map[string]bool, len(sourceFiles))
	for _, file := range sourceFiles {
		sourceFileSet[file] = true
	}

	for cachedFile := range s.FileHashes {
		if !sourceFileSet[cachedFile] {
			return true, sourceFiles, fmt.Sprintf("file deleted: %s", cachedFile)
		}
	}

	if len(changedFiles) > 0 {
		return true, changedFiles, fmt.Sprintf("%d file(s) changed", len(changedFiles))
	}

	// No changes detected
	return false, nil, ""
}

// UpdateFromBuild updates the build state after a successful build
func (s *BuildState) UpdateFromBuild(sourceFiles []string, opts *BuildOptions) error {
	// Update file hashes
	s.FileHashes = make(map[string]string)
	for _, file := range sourceFiles {
		hash, err := computeHash(file)
		if err != nil {
			return fmt.Errorf("failed to compute hash for %s: %w", file, err)
		}
		s.FileHashes[file] = hash
	}

	// Update build metadata
	s.BinaryPath = opts.OutputPath
	s.BuildOptions = BuildOptionsSnapshot{
		Mode:       opts.Mode.String(),
		OutputPath: opts.OutputPath,
		Minify:     opts.Minify,
		TreeShake:  opts.TreeShake,
		UseCache:   opts.UseCache,
	}
	s.LastBuildTime = time.Now()
	s.Version = "0.1.0" // TODO: Get from version package when available

	return nil
}

// computeHash computes SHA-256 hash of a file
func computeHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
