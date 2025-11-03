package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/conduit-lang/conduit/runtime/metadata"
)

// TestLoadMetadataFromFile_SecurityVulnerabilities tests that the path validation
// properly blocks all known attack vectors for directory traversal and symlink attacks.
func TestLoadMetadataFromFile_SecurityVulnerabilities(t *testing.T) {
	// Setup: Create a valid metadata file in build/introspection/
	err := os.MkdirAll("build/introspection", 0755)
	require.NoError(t, err)
	defer os.RemoveAll("build")

	validMetadata := &metadata.Metadata{
		Version:   "1.0.0",
		Generated: time.Now(),
		Resources: []metadata.ResourceMetadata{},
	}
	data, err := json.Marshal(validMetadata)
	require.NoError(t, err)

	validPath := "build/introspection/metadata.json"
	err = os.WriteFile(validPath, data, 0644)
	require.NoError(t, err)

	t.Run("allows valid default path", func(t *testing.T) {
		metadata.Reset()
		metadataFile = ""
		err := loadMetadataFromFile()
		assert.NoError(t, err, "should allow default path: build/introspection/metadata.json")
	})

	t.Run("allows valid custom path within build directory", func(t *testing.T) {
		metadata.Reset()
		err := os.MkdirAll("build/test", 0755)
		require.NoError(t, err)
		testPath := "build/test/metadata.json"
		err = os.WriteFile(testPath, data, 0644)
		require.NoError(t, err)

		metadataFile = testPath
		err = loadMetadataFromFile()
		assert.NoError(t, err, "should allow valid path: build/test/metadata.json")
	})

	t.Run("blocks directory traversal with ../", func(t *testing.T) {
		metadata.Reset()
		metadataFile = "../../../etc/passwd"
		err := loadMetadataFromFile()
		assert.Error(t, err, "should block directory traversal")
		assert.Contains(t, err.Error(), "must be in build/ directory")
	})

	t.Run("blocks absolute paths outside build directory", func(t *testing.T) {
		metadata.Reset()
		metadataFile = "/etc/passwd"
		err := loadMetadataFromFile()
		assert.Error(t, err, "should block absolute path outside build/")
		assert.Contains(t, err.Error(), "must be in build/ directory")
	})

	t.Run("blocks traversal with build prefix", func(t *testing.T) {
		metadata.Reset()
		metadataFile = "build/../../etc/passwd"
		err := loadMetadataFromFile()
		assert.Error(t, err, "should block traversal even with build/ prefix")
		assert.Contains(t, err.Error(), "must be in build/ directory")
	})

	t.Run("blocks symlink to file outside build directory", func(t *testing.T) {
		metadata.Reset()

		// Create a symlink in build/ pointing to /etc/passwd
		symlinkPath := "build/introspection/evil.json"
		err := os.Symlink("/etc/passwd", symlinkPath)
		require.NoError(t, err)
		defer os.Remove(symlinkPath)

		metadataFile = symlinkPath
		err = loadMetadataFromFile()
		assert.Error(t, err, "should block symlink")
		assert.Contains(t, err.Error(), "must be a regular file")
	})

	t.Run("blocks directory path", func(t *testing.T) {
		metadata.Reset()
		metadataFile = "build/introspection"
		err := loadMetadataFromFile()
		assert.Error(t, err, "should block directory path")
		assert.Contains(t, err.Error(), "must be a regular file")
	})

	t.Run("returns proper error for non-existent file", func(t *testing.T) {
		metadata.Reset()
		metadataFile = "build/introspection/nonexistent.json"
		err := loadMetadataFromFile()
		assert.Error(t, err, "should error on non-existent file")
		assert.Contains(t, err.Error(), "metadata file not found")
	})

	// Cleanup
	metadataFile = ""
}

// TestLoadMetadataFromFile_EdgeCases tests edge cases in path validation
func TestLoadMetadataFromFile_EdgeCases(t *testing.T) {
	// Setup
	err := os.MkdirAll("build/introspection", 0755)
	require.NoError(t, err)
	defer os.RemoveAll("build")

	validMetadata := &metadata.Metadata{
		Version:   "1.0.0",
		Generated: time.Now(),
		Resources: []metadata.ResourceMetadata{},
	}
	data, err := json.Marshal(validMetadata)
	require.NoError(t, err)

	t.Run("handles nested directories correctly", func(t *testing.T) {
		metadata.Reset()
		nestedPath := "build/a/b/c/metadata.json"
		err := os.MkdirAll(filepath.Dir(nestedPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(nestedPath, data, 0644)
		require.NoError(t, err)

		metadataFile = nestedPath
		err = loadMetadataFromFile()
		assert.NoError(t, err, "should allow deeply nested paths within build/")
	})

	t.Run("blocks path with . in middle", func(t *testing.T) {
		metadata.Reset()
		metadataFile = "build/./../../etc/passwd"
		err := loadMetadataFromFile()
		assert.Error(t, err, "should block path with . components")
		assert.Contains(t, err.Error(), "must be in build/ directory")
	})

	t.Run("handles relative path correctly", func(t *testing.T) {
		metadata.Reset()
		metadataFile = "./build/introspection/metadata.json"
		err = os.WriteFile("build/introspection/metadata.json", data, 0644)
		require.NoError(t, err)

		err = loadMetadataFromFile()
		assert.NoError(t, err, "should handle relative path with ./")
	})

	// Cleanup
	metadataFile = ""
}
