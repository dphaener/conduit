package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestMigrationBuilder_GenerateInitialMigration(t *testing.T) {
	builder := NewMigrationBuilder()

	// Create test schemas
	schemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:   "User",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
					Constraints: []schema.Constraint{
						{Type: schema.ConstraintPrimary},
					},
				},
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: false,
					},
					Constraints: []schema.Constraint{
						{Type: schema.ConstraintUnique},
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	sql, err := builder.GenerateInitialMigration(schemas)
	if err != nil {
		t.Fatalf("GenerateInitialMigration() error = %v", err)
	}

	// Verify SQL contains CREATE TABLE
	if !strings.Contains(sql, "CREATE TABLE") {
		t.Error("Initial migration should contain CREATE TABLE statement")
	}

	// Verify table name is snake_case
	if !strings.Contains(sql, "user") {
		t.Error("Initial migration should create 'user' table")
	}
}

func TestMigrationBuilder_GenerateVersionedMigration(t *testing.T) {
	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")

	builder := NewMigrationBuilder()

	// Create old schema
	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: false,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	// Create new schema with added field
	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: false,
					},
				},
				"name": {
					Name: "name",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: false,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	result, err := builder.GenerateVersionedMigration(oldSchemas, newSchemas, migrationsDir)
	if err != nil {
		t.Fatalf("GenerateVersionedMigration() error = %v", err)
	}

	if !result.MigrationGenerated {
		t.Fatal("Expected migration to be generated")
	}

	// Verify migration file was created
	if _, err := os.Stat(result.MigrationPath); os.IsNotExist(err) {
		t.Errorf("Migration file should exist at %s", result.MigrationPath)
	}

	// Read and verify migration content
	content, err := os.ReadFile(result.MigrationPath)
	if err != nil {
		t.Fatalf("Failed to read migration file: %v", err)
	}

	sqlContent := string(content)

	// Should contain ALTER TABLE ADD COLUMN
	if !strings.Contains(sqlContent, "ALTER TABLE") {
		t.Error("Migration should contain ALTER TABLE statement")
	}

	if !strings.Contains(sqlContent, "ADD COLUMN") {
		t.Error("Migration should contain ADD COLUMN statement")
	}

	// Should mention the new field
	if !strings.Contains(sqlContent, "name") {
		t.Error("Migration should mention the 'name' field")
	}
}

func TestMigrationBuilder_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")

	builder := NewMigrationBuilder()

	// Create identical schemas
	schemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	result, err := builder.GenerateVersionedMigration(schemas, schemas, migrationsDir)
	if err != nil {
		t.Fatalf("GenerateVersionedMigration() error = %v", err)
	}

	if result.MigrationGenerated {
		t.Error("No migration should be generated when schemas are identical")
	}
}

func TestMigrationBuilder_ShouldGenerateInitialMigration(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(string)
		wantGenerate   bool
	}{
		{
			name: "no migrations directory",
			setupFunc: func(dir string) {
				// Do nothing - directory doesn't exist
			},
			wantGenerate: true,
		},
		{
			name: "empty migrations directory",
			setupFunc: func(dir string) {
				os.MkdirAll(dir, 0755)
			},
			wantGenerate: true,
		},
		{
			name: "001_init.sql exists",
			setupFunc: func(dir string) {
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "001_init.sql"), []byte("-- test"), 0644)
			},
			wantGenerate: false,
		},
		{
			name: "other migration exists",
			setupFunc: func(dir string) {
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "002_add_users.sql"), []byte("-- test"), 0644)
			},
			wantGenerate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			migrationsDir := filepath.Join(tmpDir, "migrations")

			tt.setupFunc(migrationsDir)

			builder := NewMigrationBuilder()
			shouldGenerate, err := builder.ShouldGenerateInitialMigration(migrationsDir)
			if err != nil {
				t.Fatalf("ShouldGenerateInitialMigration() error = %v", err)
			}

			if shouldGenerate != tt.wantGenerate {
				t.Errorf("ShouldGenerateInitialMigration() = %v, want %v", shouldGenerate, tt.wantGenerate)
			}
		})
	}
}

func TestMigrationBuilder_SequenceNumbering(t *testing.T) {
	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	builder := NewMigrationBuilder()

	// Create schemas that will trigger migrations
	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name:          "User",
			Fields:        map[string]*schema.Field{},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	newSchemas1 := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"field1": {
					Name: "field1",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: true,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	// Generate first migration
	result1, err := builder.GenerateVersionedMigration(oldSchemas, newSchemas1, migrationsDir)
	if err != nil {
		t.Fatalf("First migration error = %v", err)
	}

	if !result1.MigrationGenerated {
		t.Fatal("First migration should be generated")
	}

	// Verify filename contains sequence number
	filename1 := filepath.Base(result1.MigrationPath)
	if !strings.Contains(filename1, "_001_") {
		t.Errorf("First migration filename %s should contain _001_", filename1)
	}
}

func TestMigrationBuilder_BreakingAndDataLoss(t *testing.T) {
	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")

	builder := NewMigrationBuilder()

	// Old schema with nullable field
	oldSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: true,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	// New schema making field non-nullable (breaking change)
	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"email": {
					Name: "email",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeString,
						Nullable: false,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	result, err := builder.GenerateVersionedMigration(oldSchemas, newSchemas, migrationsDir)
	if err != nil {
		t.Fatalf("GenerateVersionedMigration() error = %v", err)
	}

	if !result.MigrationGenerated {
		t.Fatal("Migration should be generated")
	}

	if !result.Breaking {
		t.Error("Migration should be marked as breaking")
	}
}

func TestMigrationBuilder_ReadOnlyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")

	// Create directory with read-only permissions
	if err := os.MkdirAll(migrationsDir, 0444); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}
	defer os.Chmod(migrationsDir, 0755) // Restore permissions for cleanup

	builder := NewMigrationBuilder()

	oldSchemas := map[string]*schema.ResourceSchema{}
	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	// Should fail due to read-only directory
	_, err := builder.GenerateVersionedMigration(oldSchemas, newSchemas, migrationsDir)
	if err == nil {
		t.Error("Expected error when writing to read-only directory")
	}
}

func TestMigrationBuilder_MalformedMigrationFiles(t *testing.T) {
	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create malformed migration files
	malformedFiles := []string{
		"invalid.sql",
		"123_noseq.sql",
		"timestamp_abc_name.sql",
		"_001_name.sql",
	}

	for _, filename := range malformedFiles {
		os.WriteFile(filepath.Join(migrationsDir, filename), []byte("-- test"), 0644)
	}

	builder := NewMigrationBuilder()

	// Create schemas that will generate a migration
	oldSchemas := map[string]*schema.ResourceSchema{}
	newSchemas := map[string]*schema.ResourceSchema{
		"User": {
			Name: "User",
			Fields: map[string]*schema.Field{
				"id": {
					Name: "id",
					Type: &schema.TypeSpec{
						BaseType: schema.TypeUUID,
						Nullable: false,
					},
				},
			},
			Relationships: map[string]*schema.Relationship{},
		},
	}

	// Should succeed despite malformed files
	result, err := builder.GenerateVersionedMigration(oldSchemas, newSchemas, migrationsDir)
	if err != nil {
		t.Fatalf("Should succeed despite malformed files: %v", err)
	}

	if !result.MigrationGenerated {
		t.Error("Migration should be generated")
	}
}
