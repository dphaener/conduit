package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestSnapshotManager_SaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	buildDir := filepath.Join(tmpDir, "build")

	manager := NewSnapshotManager(buildDir)

	// Create test schemas
	schemas := map[string]*schema.ResourceSchema{
		"User": schema.NewResourceSchema("User"),
		"Post": schema.NewResourceSchema("Post"),
	}

	// Add a field to User
	schemas["User"].Fields["email"] = &schema.Field{
		Name: "email",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
	}

	// Save snapshot
	timestamp := int64(1234567890)
	err := manager.Save(schemas, timestamp)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if !manager.Exists() {
		t.Error("Snapshot file should exist after Save()")
	}

	// Load snapshot
	loadedSchemas, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded schemas match
	if len(loadedSchemas) != len(schemas) {
		t.Errorf("Loaded %d schemas, want %d", len(loadedSchemas), len(schemas))
	}

	if _, ok := loadedSchemas["User"]; !ok {
		t.Error("User schema not found in loaded snapshot")
	}

	if _, ok := loadedSchemas["Post"]; !ok {
		t.Error("Post schema not found in loaded snapshot")
	}

	// Verify field was preserved
	if userSchema, ok := loadedSchemas["User"]; ok {
		if emailField, ok := userSchema.Fields["email"]; ok {
			if emailField.Type.BaseType != schema.TypeString {
				t.Errorf("email field type = %v, want String", emailField.Type.BaseType)
			}
		} else {
			t.Error("email field not found in loaded User schema")
		}
	}
}

func TestSnapshotManager_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	buildDir := filepath.Join(tmpDir, "build")

	manager := NewSnapshotManager(buildDir)

	// Load from non-existent file should return nil, nil
	schemas, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if schemas != nil {
		t.Error("Load() should return nil for non-existent snapshot")
	}

	if manager.Exists() {
		t.Error("Exists() should return false for non-existent snapshot")
	}
}

func TestSnapshotManager_SaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	buildDir := filepath.Join(tmpDir, "build")

	manager := NewSnapshotManager(buildDir)

	// Save should create .conduit directory if it doesn't exist
	schemas := map[string]*schema.ResourceSchema{
		"User": schema.NewResourceSchema("User"),
	}

	err := manager.Save(schemas, 123456)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify .conduit directory was created
	conduitDir := filepath.Join(tmpDir, ".conduit")
	if _, err := os.Stat(conduitDir); os.IsNotExist(err) {
		t.Error(".conduit directory should have been created")
	}

	// Verify snapshot file exists
	if !manager.Exists() {
		t.Error("Snapshot file should exist after Save()")
	}
}

func TestSnapshotManager_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	buildDir := filepath.Join(tmpDir, "build")

	manager := NewSnapshotManager(buildDir)

	// Save initial snapshot
	schemas1 := map[string]*schema.ResourceSchema{
		"User": schema.NewResourceSchema("User"),
	}

	err := manager.Save(schemas1, 111111)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Save updated snapshot
	schemas2 := map[string]*schema.ResourceSchema{
		"User": schema.NewResourceSchema("User"),
		"Post": schema.NewResourceSchema("Post"),
	}

	err = manager.Save(schemas2, 222222)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load and verify we got the updated version
	loadedSchemas, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loadedSchemas) != 2 {
		t.Errorf("Expected 2 schemas after update, got %d", len(loadedSchemas))
	}

	// Verify no .tmp file left behind
	tmpFile := manager.Path() + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("Temporary file should have been cleaned up")
	}
}
