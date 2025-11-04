package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationSQLGenerator_GenerateFromResource(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Create a test resource file
	resourceContent := `/// User resource
resource User {
  id: uuid! @primary @auto
  email: string! @unique @max(255)
  username: string! @unique @max(30)
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
}
`
	resourceFile := filepath.Join(appDir, "user.cdt")
	if err := os.WriteFile(resourceFile, []byte(resourceContent), 0644); err != nil {
		t.Fatalf("Failed to write test resource file: %v", err)
	}

	// Change to the temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test the generator
	generator := NewMigrationSQLGenerator()
	upSQL, downSQL, err := generator.GenerateFromResource("User")
	if err != nil {
		t.Fatalf("GenerateFromResource failed: %v", err)
	}

	// Print the generated SQL for debugging
	t.Logf("Generated UP SQL:\n%s", upSQL)
	t.Logf("Generated DOWN SQL:\n%s", downSQL)

	// Verify up migration contains expected SQL
	if !strings.Contains(upSQL, "CREATE TABLE IF NOT EXISTS") {
		t.Error("Up migration should contain CREATE TABLE IF NOT EXISTS")
	}

	if !strings.Contains(upSQL, "\"user\"") {
		t.Error("Up migration should reference the user table")
	}

	if !strings.Contains(upSQL, "id") {
		t.Error("Up migration should contain id field")
	}

	if !strings.Contains(upSQL, "email") {
		t.Error("Up migration should contain email field")
	}

	if !strings.Contains(upSQL, "username") {
		t.Error("Up migration should contain username field")
	}

	if !strings.Contains(upSQL, "UUID") {
		t.Error("Up migration should contain UUID type")
	}

	if !strings.Contains(upSQL, "VARCHAR") {
		t.Error("Up migration should contain VARCHAR type")
	}

	if !strings.Contains(upSQL, "TIMESTAMP") {
		t.Error("Up migration should contain TIMESTAMP type")
	}

	if !strings.Contains(upSQL, "NOT NULL") {
		t.Error("Up migration should contain NOT NULL constraints")
	}

	if !strings.Contains(upSQL, "PRIMARY KEY") {
		t.Error("Up migration should contain PRIMARY KEY constraint")
	}

	if !strings.Contains(upSQL, "gen_random_uuid()") {
		t.Error("Up migration should contain gen_random_uuid() for @auto UUID")
	}

	if !strings.Contains(upSQL, "CURRENT_TIMESTAMP") {
		t.Error("Up migration should contain CURRENT_TIMESTAMP for @auto timestamp")
	}

	// Verify indexes are generated for @unique fields
	if !strings.Contains(upSQL, "CREATE UNIQUE INDEX") {
		t.Error("Up migration should contain CREATE UNIQUE INDEX for @unique fields")
	}

	if !strings.Contains(upSQL, "idx_") {
		t.Error("Up migration should contain index names with idx_ prefix")
	}

	// Verify down migration contains expected SQL
	if !strings.Contains(downSQL, "DROP TABLE IF EXISTS") {
		t.Error("Down migration should contain DROP TABLE IF EXISTS")
	}

	if !strings.Contains(downSQL, "CASCADE") {
		t.Error("Down migration should contain CASCADE for safe drops")
	}
}

func TestMigrationSQLGenerator_FindResourceFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Create test resource files
	if err := os.WriteFile(filepath.Join(appDir, "user.cdt"), []byte("resource User {}"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(appDir, "Post.cdt"), []byte("resource Post {}"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Change to the temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	generator := NewMigrationSQLGenerator()

	// Test lowercase file (user.cdt)
	file, err := generator.findResourceFile("User")
	if err != nil {
		t.Errorf("Should find user.cdt for 'User': %v", err)
	}
	if !strings.HasSuffix(file, "user.cdt") {
		t.Errorf("Expected user.cdt, got %s", file)
	}

	// Test exact case file (Post.cdt) - note: findResourceFile returns full path
	file, err = generator.findResourceFile("Post")
	if err != nil {
		t.Errorf("Should find Post.cdt for 'Post': %v", err)
	}
	if !strings.Contains(file, "post.cdt") && !strings.Contains(file, "Post.cdt") {
		t.Errorf("Expected post.cdt or Post.cdt in path, got %s", file)
	}

	// Test non-existent file
	_, err = generator.findResourceFile("NonExistent")
	if err == nil {
		t.Error("Should return error for non-existent resource")
	}
}

func TestMigrationSQLGenerator_ComplexTypes(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Create a test resource file with various types
	resourceContent := `resource Product {
  id: uuid! @primary @auto
  name: string! @max(255)
  description: text?
  price: int!
  in_stock: bool!
  metadata: json?
  created_at: timestamp! @auto
}
`
	resourceFile := filepath.Join(appDir, "product.cdt")
	if err := os.WriteFile(resourceFile, []byte(resourceContent), 0644); err != nil {
		t.Fatalf("Failed to write test resource file: %v", err)
	}

	// Change to the temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test the generator
	generator := NewMigrationSQLGenerator()
	upSQL, downSQL, err := generator.GenerateFromResource("Product")
	if err != nil {
		t.Fatalf("GenerateFromResource failed: %v", err)
	}

	// Verify various types are correctly mapped
	if !strings.Contains(upSQL, "VARCHAR(255)") {
		t.Error("Should map string with @max to VARCHAR(N)")
	}

	if !strings.Contains(upSQL, "TEXT") {
		t.Error("Should map text to TEXT")
	}

	if !strings.Contains(upSQL, "INTEGER") {
		t.Error("Should map int to INTEGER")
	}

	if !strings.Contains(upSQL, "BOOLEAN") {
		t.Error("Should map boolean to BOOLEAN")
	}

	if !strings.Contains(upSQL, "JSON") && !strings.Contains(upSQL, "JSONB") {
		t.Error("Should map json to JSON or JSONB")
	}

	// Verify nullable fields
	if !strings.Contains(upSQL, "NULL") {
		t.Error("Should handle nullable fields")
	}

	// Verify down migration
	if !strings.Contains(downSQL, "DROP TABLE") {
		t.Error("Down migration should drop the table")
	}
}

func TestMigrationSQLGenerator_PathTraversalPrevention(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	generator := NewMigrationSQLGenerator()

	// Test path traversal attempts
	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"../../secrets",
		"../internal/config",
		"./../../passwords",
		"User/../../etc",
		"User/../../../root",
	}

	for _, attempt := range pathTraversalAttempts {
		t.Run(attempt, func(t *testing.T) {
			_, _, err := generator.GenerateFromResource(attempt)
			if err == nil {
				t.Errorf("Expected error for path traversal attempt: %s", attempt)
			}

			// Should reject with validation error
			if !strings.Contains(err.Error(), "invalid resource name") &&
				!strings.Contains(err.Error(), "path traversal") {
				t.Errorf("Expected validation/security error, got: %v", err)
			}
		})
	}

	// Valid resource names should still work
	validNames := []string{"User", "user", "UserProfile", "user_profile"}
	for _, validName := range validNames {
		t.Run(validName+"_valid", func(t *testing.T) {
			// Create a valid test file
			resourceContent := `resource TestResource {
  id: uuid! @primary @auto
}`
			testFile := filepath.Join(appDir, strings.ToLower(validName)+".cdt")
			if err := os.WriteFile(testFile, []byte(resourceContent), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			_, _, err := generator.GenerateFromResource(validName)
			// Should not reject valid names
			if err != nil && strings.Contains(err.Error(), "invalid resource name") {
				t.Errorf("Valid resource name rejected: %s, error: %v", validName, err)
			}
		})
	}
}

func TestMigrationSQLGenerator_SymlinkPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	secretsDir := filepath.Join(tmpDir, "secrets")

	// Create directories
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Create a "secret" file outside app directory
	secretFile := filepath.Join(secretsDir, "secret.cdt")
	secretContent := `resource Secret { password: string! }`
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("Failed to write secret file: %v", err)
	}

	// Create symlink inside app/ pointing to secret file
	symlinkPath := filepath.Join(appDir, "malicious.cdt")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Skipf("Skipping symlink test (symlinks not supported): %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	generator := NewMigrationSQLGenerator()

	// Should reject symlink that points outside app directory
	_, _, err := generator.GenerateFromResource("malicious")
	if err == nil {
		t.Error("Expected error for symlink-based path traversal")
	}

	if !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("Expected 'path traversal' error, got: %v", err)
	}
}
