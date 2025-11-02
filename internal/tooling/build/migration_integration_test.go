package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
)

// TestIntegration_FirstBuildGeneratesInitialMigration tests that the first build
// generates 001_init.sql
//
// NOTE: This test focuses on migration generation only, using parsed AST directly
func TestIntegration_FirstBuildGeneratesInitialMigration(t *testing.T) {
	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	buildDir := filepath.Join(tmpDir, "build")

	// Parse a simple resource
	source := `resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
  created_at: timestamp! @auto
}`

	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	program, _ := p.Parse()

	// Create compiled file
	compiled := []*CompiledFile{
		{
			Path:    "user.cdt",
			Hash:    "test",
			Program: program,
		},
	}

	// Extract schemas
	extractor := NewSchemaExtractor()
	schemas, err := extractor.ExtractSchemas(compiled)
	if err != nil {
		t.Fatalf("Failed to extract schemas: %v", err)
	}

	// Create migration builder and generate initial migration
	migrationBuilder := NewMigrationBuilder()
	shouldGenerate, err := migrationBuilder.ShouldGenerateInitialMigration(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}

	if !shouldGenerate {
		t.Fatal("Should want to generate initial migration")
	}

	initialSQL, err := migrationBuilder.GenerateInitialMigration(schemas)
	if err != nil {
		t.Fatalf("Failed to generate initial migration: %v", err)
	}

	// Write migration
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	initPath := filepath.Join(migrationsDir, "001_init.sql")
	if err := os.WriteFile(initPath, []byte(initialSQL), 0644); err != nil {
		t.Fatal(err)
	}

	// Save snapshot
	snapshotManager := NewSnapshotManager(buildDir)
	if err := snapshotManager.Save(schemas, 123456); err != nil {
		t.Fatal(err)
	}

	// Verify 001_init.sql was created
	initMigration := filepath.Join(tmpDir, "migrations", "001_init.sql")
	if _, err := os.Stat(initMigration); os.IsNotExist(err) {
		t.Error("001_init.sql should have been created")
	}

	// Verify migration content
	content, err := os.ReadFile(initMigration)
	if err != nil {
		t.Fatalf("Failed to read migration: %v", err)
	}

	sql := string(content)
	if !strings.Contains(sql, "CREATE TABLE") {
		t.Error("Migration should contain CREATE TABLE")
	}
	if !strings.Contains(sql, "user") {
		t.Error("Migration should create user table")
	}

	// Verify snapshot was created
	snapshotPath := filepath.Join(tmpDir, ".conduit", "schema-snapshot.json")
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		t.Error("Schema snapshot should have been created")
	}
}

// TestIntegration_ModifyResourceGeneratesNewMigration tests that modifying a resource
// generates a new versioned migration
func TestIntegration_ModifyResourceGeneratesNewMigration(t *testing.T) {
	tmpDir := t.TempDir()
	buildDir := filepath.Join(tmpDir, "build")
	migrationsDir := filepath.Join(tmpDir, "migrations")

	// Parse initial resource
	initialSource := `resource User {
  id: uuid! @primary @auto
  email: string! @unique
}`

	lex := lexer.New(initialSource)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	initialProgram, _ := p.Parse()

	compiled1 := []*CompiledFile{{Path: "user.cdt", Hash: "test1", Program: initialProgram}}

	// Extract and save initial schemas
	extractor := NewSchemaExtractor()
	initialSchemas, err := extractor.ExtractSchemas(compiled1)
	if err != nil {
		t.Fatal(err)
	}

	// Generate and write initial migration
	migrationBuilder := NewMigrationBuilder()
	initialSQL, err := migrationBuilder.GenerateInitialMigration(initialSchemas)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	initPath := filepath.Join(migrationsDir, "001_init.sql")
	if err := os.WriteFile(initPath, []byte(initialSQL), 0644); err != nil {
		t.Fatal(err)
	}

	// Save initial snapshot
	snapshotManager := NewSnapshotManager(buildDir)
	if err := snapshotManager.Save(initialSchemas, 111111); err != nil {
		t.Fatal(err)
	}

	// Count migrations before modification
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	migrationsBefore := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrationsBefore++
		}
	}

	// Parse modified resource (add new fields)
	modifiedSource := `resource User {
  id: uuid! @primary @auto
  email: string! @unique
  name: string!
  created_at: timestamp! @auto
}`

	lex2 := lexer.New(modifiedSource)
	tokens2, _ := lex2.ScanTokens()
	p2 := parser.New(tokens2)
	modifiedProgram, _ := p2.Parse()

	compiled2 := []*CompiledFile{{Path: "user.cdt", Hash: "test2", Program: modifiedProgram}}

	// Extract modified schemas
	modifiedSchemas, err := extractor.ExtractSchemas(compiled2)
	if err != nil {
		t.Fatal(err)
	}

	// Load previous schemas
	previousSchemas, err := snapshotManager.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Generate versioned migration
	result, err := migrationBuilder.GenerateVersionedMigration(previousSchemas, modifiedSchemas, migrationsDir)
	if err != nil {
		t.Fatalf("Failed to generate migration: %v", err)
	}

	if !result.MigrationGenerated {
		t.Fatal("Migration should have been generated")
	}

	// Save new snapshot
	if err := snapshotManager.Save(modifiedSchemas, 222222); err != nil {
		t.Fatal(err)
	}

	// Count migrations after modification
	entries, err = os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	migrationsAfter := 0
	var newMigrationFile string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrationsAfter++
			if entry.Name() != "001_init.sql" {
				newMigrationFile = entry.Name()
			}
		}
	}

	// Should have one more migration
	if migrationsAfter != migrationsBefore+1 {
		t.Errorf("Expected %d migrations, got %d", migrationsBefore+1, migrationsAfter)
	}

	// Verify new migration was created with correct format
	if newMigrationFile == "" {
		t.Fatal("New migration file should have been created")
	}

	// Verify filename format: {timestamp}_{seq}_{name}.sql
	parts := strings.Split(newMigrationFile, "_")
	if len(parts) < 3 {
		t.Errorf("Migration filename %s should have format {timestamp}_{seq}_{name}.sql", newMigrationFile)
	}

	// Verify migration content
	migrationPath := filepath.Join(migrationsDir, newMigrationFile)
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("Failed to read new migration: %v", err)
	}

	sql := string(content)

	// Should contain ALTER TABLE ADD COLUMN for new fields
	if !strings.Contains(sql, "ALTER TABLE") {
		t.Error("New migration should contain ALTER TABLE")
	}
	if !strings.Contains(sql, "ADD COLUMN") {
		t.Error("New migration should contain ADD COLUMN")
	}
	if !strings.Contains(sql, "name") || !strings.Contains(sql, "created_at") {
		t.Error("New migration should add 'name' and 'created_at' columns")
	}

	// 001_init.sql should not have been modified
	initContent, err := os.ReadFile(initPath)
	if err != nil {
		t.Fatal(err)
	}
	initSQL := string(initContent)
	if strings.Contains(initSQL, "name") || strings.Contains(initSQL, "created_at") {
		t.Error("001_init.sql should not have been modified")
	}
}

// TestIntegration_NoChangesNoMigration tests that rebuilding without changes
// does not generate a new migration
func TestIntegration_NoChangesNoMigration(t *testing.T) {
	tmpDir := t.TempDir()
	buildDir := filepath.Join(tmpDir, "build")
	migrationsDir := filepath.Join(tmpDir, "migrations")

	// Parse resource
	source := `resource User {
  id: uuid! @primary @auto
  email: string! @unique
}`

	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	program, _ := p.Parse()

	compiled := []*CompiledFile{{Path: "user.cdt", Hash: "test", Program: program}}

	// Extract schemas
	extractor := NewSchemaExtractor()
	schemas, err := extractor.ExtractSchemas(compiled)
	if err != nil {
		t.Fatal(err)
	}

	// Generate initial migration
	migrationBuilder := NewMigrationBuilder()
	initialSQL, err := migrationBuilder.GenerateInitialMigration(schemas)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	initPath := filepath.Join(migrationsDir, "001_init.sql")
	if err := os.WriteFile(initPath, []byte(initialSQL), 0644); err != nil {
		t.Fatal(err)
	}

	// Save snapshot
	snapshotManager := NewSnapshotManager(buildDir)
	if err := snapshotManager.Save(schemas, 111111); err != nil {
		t.Fatal(err)
	}

	// Count migrations
	entries1, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	count1 := 0
	for _, entry := range entries1 {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			count1++
		}
	}

	// Load previous schemas
	previousSchemas, err := snapshotManager.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Try to generate migration with same schemas
	result, err := migrationBuilder.GenerateVersionedMigration(previousSchemas, schemas, migrationsDir)
	if err != nil {
		t.Fatalf("GenerateVersionedMigration failed: %v", err)
	}

	// Count migrations again
	entries2, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	count2 := 0
	for _, entry := range entries2 {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			count2++
		}
	}

	// Should not generate a migration
	if result.MigrationGenerated {
		t.Error("No migration should be generated when schemas are identical")
	}
}
