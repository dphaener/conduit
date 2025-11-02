package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// SchemaSnapshot represents a snapshot of the application's schema
type SchemaSnapshot struct {
	Version   int                              `json:"version"`   // Snapshot format version
	Timestamp int64                            `json:"timestamp"` // Unix timestamp when snapshot was created
	Resources map[string]*schema.ResourceSchema `json:"resources"` // All resource schemas
}

// SnapshotManager handles schema snapshot persistence
type SnapshotManager struct {
	snapshotPath string
	mu           sync.Mutex // Protects concurrent access to snapshot file
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(buildDir string) *SnapshotManager {
	// buildDir is typically "build" - get its parent (project root)
	absBuildDir, err := filepath.Abs(buildDir)
	if err != nil {
		// Fallback to relative path if Abs fails
		absBuildDir = buildDir
	}
	projectRoot := filepath.Dir(absBuildDir)
	conduitDir := filepath.Join(projectRoot, ".conduit")
	snapshotPath := filepath.Join(conduitDir, "schema-snapshot.json")

	return &SnapshotManager{
		snapshotPath: snapshotPath,
	}
}

// Load loads the schema snapshot from disk
// Returns nil if the snapshot doesn't exist (first build)
func (m *SnapshotManager) Load() (map[string]*schema.ResourceSchema, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(m.snapshotPath); os.IsNotExist(err) {
		return nil, nil // No snapshot exists yet
	}

	data, err := os.ReadFile(m.snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot file: %w", err)
	}

	var snapshot SchemaSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot JSON: %w", err)
	}

	return snapshot.Resources, nil
}

// Save saves the schema snapshot to disk
func (m *SnapshotManager) Save(schemas map[string]*schema.ResourceSchema, timestamp int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot := SchemaSnapshot{
		Version:   1,
		Timestamp: timestamp,
		Resources: schemas,
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// Ensure .conduit directory exists
	dir := filepath.Dir(m.snapshotPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create .conduit directory: %w", err)
	}

	// Write atomically using a temporary file
	tmpPath := m.snapshotPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary snapshot file: %w", err)
	}

	if err := os.Rename(tmpPath, m.snapshotPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file on error
		return fmt.Errorf("failed to rename snapshot file: %w", err)
	}

	return nil
}

// Exists checks if a snapshot file exists
func (m *SnapshotManager) Exists() bool {
	_, err := os.Stat(m.snapshotPath)
	return err == nil
}

// Path returns the path to the snapshot file
func (m *SnapshotManager) Path() string {
	return m.snapshotPath
}
