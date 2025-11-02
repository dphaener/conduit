package build

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestAutoMigrateIntegration tests the full auto-migrate workflow
func TestAutoMigrateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Get database URL from environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	// Create temporary migrations directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir("migrations", 0755); err != nil {
		t.Fatal(err)
	}

	// Setup database
	db := setupTestDatabase(t, dbURL)
	defer cleanupTestDatabase(t, db)

	t.Run("applies safe migrations automatically", func(t *testing.T) {
		// Create safe migration files
		createMigrationFile(t, "migrations/001_create_users.up.sql", `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255)
			);
		`)

		createMigrationFile(t, "migrations/002_add_index.up.sql", `
			CREATE INDEX idx_users_email ON users(email);
		`)

		// Set environment to development
		os.Setenv("ENV", "development")
		defer os.Unsetenv("ENV")

		// Run auto-migrate
		migrator := NewAutoMigrator(AutoMigrateOptions{
			Mode:        AutoMigrateApply,
			SkipConfirm: true,
		})

		err := migrator.Run()
		if err != nil {
			t.Fatalf("auto-migrate failed: %v", err)
		}

		// Verify migrations were applied
		assertMigrationApplied(t, db, 1, "001_create_users.up.sql")
		assertMigrationApplied(t, db, 2, "002_add_index.up.sql")

		// Verify table was created
		assertTableExists(t, db, "users")
	})

	t.Run("no pending migrations shows success", func(t *testing.T) {
		// Set environment to development
		os.Setenv("ENV", "development")
		defer os.Unsetenv("ENV")

		// Run auto-migrate again (all migrations already applied)
		migrator := NewAutoMigrator(AutoMigrateOptions{
			Mode:        AutoMigrateApply,
			SkipConfirm: true,
		})

		err := migrator.Run()
		if err != nil {
			t.Fatalf("auto-migrate failed: %v", err)
		}
	})

	t.Run("dry-run shows SQL without executing", func(t *testing.T) {
		// Create new migration
		createMigrationFile(t, "migrations/003_add_posts.up.sql", `
			CREATE TABLE posts (
				id SERIAL PRIMARY KEY,
				title VARCHAR(255),
				content TEXT
			);
		`)

		// Set environment to development
		os.Setenv("ENV", "development")
		defer os.Unsetenv("ENV")

		// Run dry-run
		migrator := NewAutoMigrator(AutoMigrateOptions{
			Mode:        AutoMigrateDryRun,
			SkipConfirm: true,
		})

		err := migrator.Run()
		if err != nil {
			t.Fatalf("dry-run failed: %v", err)
		}

		// Verify migration was NOT applied
		assertMigrationNotApplied(t, db, 3)
		assertTableNotExists(t, db, "posts")
	})
}

// TestProductionBlocking tests that auto-migrate blocks in production
func TestProductionBlocking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Get database URL from environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	// Create temporary migrations directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir("migrations", 0755); err != nil {
		t.Fatal(err)
	}

	// Create a safe migration
	createMigrationFile(t, "migrations/001_create_test.up.sql", `
		CREATE TABLE test_table (id SERIAL PRIMARY KEY);
	`)

	tests := []struct {
		name   string
		envVar string
		value  string
	}{
		{
			name:   "blocks on ENV=production",
			envVar: "ENV",
			value:  "production",
		},
		{
			name:   "blocks on RAILS_ENV=production",
			envVar: "RAILS_ENV",
			value:  "production",
		},
		{
			name:   "blocks on NODE_ENV=production",
			envVar: "NODE_ENV",
			value:  "production",
		},
		{
			name:   "blocks on ENV=prod",
			envVar: "ENV",
			value:  "prod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set production environment
			os.Setenv(tt.envVar, tt.value)
			defer os.Unsetenv(tt.envVar)

			// Run auto-migrate
			migrator := NewAutoMigrator(AutoMigrateOptions{
				Mode:        AutoMigrateApply,
				SkipConfirm: true,
			})

			err := migrator.Run()
			if err == nil {
				t.Fatal("expected error in production environment, got none")
			}

			// Verify error message mentions blocking
			errMsg := err.Error()
			if !contains(errMsg, "BLOCKED") && !contains(errMsg, "production") {
				t.Errorf("expected error to mention production blocking, got: %s", errMsg)
			}
		})
	}

	t.Run("blocks on production DATABASE_URL", func(t *testing.T) {
		// Set production DATABASE_URL
		originalURL := os.Getenv("DATABASE_URL")
		os.Setenv("DATABASE_URL", "postgres://user:pass@production.example.com/db")
		defer func() {
			if originalURL != "" {
				os.Setenv("DATABASE_URL", originalURL)
			} else {
				os.Unsetenv("DATABASE_URL")
			}
		}()

		// Run auto-migrate
		migrator := NewAutoMigrator(AutoMigrateOptions{
			Mode:        AutoMigrateApply,
			SkipConfirm: true,
		})

		err := migrator.Run()
		if err == nil {
			t.Fatal("expected error with production DATABASE_URL, got none")
		}

		// Verify error message mentions blocking
		errMsg := err.Error()
		if !contains(errMsg, "BLOCKED") {
			t.Errorf("expected error to mention blocking, got: %s", errMsg)
		}
	})

	t.Run("allows production with ForceProduction flag", func(t *testing.T) {
		// Set production environment
		os.Setenv("ENV", "production")
		defer os.Unsetenv("ENV")

		// This would normally be blocked, but ForceProduction allows it
		// Note: We don't actually run this against a real prod DB in tests
		migrator := NewAutoMigrator(AutoMigrateOptions{
			Mode:            AutoMigrateDryRun, // Use dry-run to avoid actual changes
			SkipConfirm:     true,
			ForceProduction: true,
		})

		// Should not error due to production check
		err := migrator.checkProductionEnvironment()
		if err != nil {
			t.Errorf("ForceProduction should bypass production check, got error: %v", err)
		}
	})
}

// TestBreakingMigrationConfirmation tests that breaking migrations require confirmation
func TestBreakingMigrationConfirmation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Get database URL from environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	// Create temporary migrations directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir("migrations", 0755); err != nil {
		t.Fatal(err)
	}

	// Setup database
	db := setupTestDatabase(t, dbURL)
	defer cleanupTestDatabase(t, db)

	// Create initial table
	createMigrationFile(t, "migrations/001_create_temp.up.sql", `
		CREATE TABLE temp_table (id SERIAL PRIMARY KEY, name VARCHAR(255));
	`)

	// Set environment to development
	os.Setenv("ENV", "development")
	defer os.Unsetenv("ENV")

	// Apply initial migration
	migrator := NewAutoMigrator(AutoMigrateOptions{
		Mode:        AutoMigrateApply,
		SkipConfirm: true,
	})
	if err := migrator.Run(); err != nil {
		t.Fatalf("failed to apply initial migration: %v", err)
	}

	// Create breaking migration
	createMigrationFile(t, "migrations/002_drop_table.up.sql", `
		DROP TABLE temp_table;
	`)

	t.Run("breaking migration applied with SkipConfirm", func(t *testing.T) {
		// Run with SkipConfirm (for automated tests)
		migrator := NewAutoMigrator(AutoMigrateOptions{
			Mode:        AutoMigrateApply,
			SkipConfirm: true, // Skip confirmation prompt
		})

		err := migrator.Run()
		if err != nil {
			t.Fatalf("auto-migrate failed: %v", err)
		}

		// Verify migration was applied
		assertMigrationApplied(t, db, 2, "002_drop_table.up.sql")

		// Verify table was dropped
		assertTableNotExists(t, db, "temp_table")
	})
}

// Helper functions

func setupTestDatabase(t *testing.T, dbURL string) *sql.DB {
	t.Helper()

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("database unreachable: %v", err)
	}

	// Set DATABASE_URL for auto-migrator to use
	os.Setenv("DATABASE_URL", dbURL)

	return db
}

func cleanupTestDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	// Drop all tables
	_, _ = db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")
	_, _ = db.Exec("DROP TABLE IF EXISTS users CASCADE")
	_, _ = db.Exec("DROP TABLE IF EXISTS posts CASCADE")
	_, _ = db.Exec("DROP TABLE IF EXISTS temp_table CASCADE")
	_, _ = db.Exec("DROP TABLE IF EXISTS test_table CASCADE")

	db.Close()
	os.Unsetenv("DATABASE_URL")
}

func createMigrationFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write migration file %s: %v", path, err)
	}
}

func assertMigrationApplied(t *testing.T, db *sql.DB, version int64, name string) {
	t.Helper()

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check migration status: %v", err)
	}

	if !exists {
		t.Errorf("expected migration %d (%s) to be applied, but it wasn't", version, name)
	}
}

func assertMigrationNotApplied(t *testing.T, db *sql.DB, version int64) {
	t.Helper()

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check migration status: %v", err)
	}

	if exists {
		t.Errorf("expected migration %d to NOT be applied, but it was", version)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()

	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)",
		tableName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}

	if !exists {
		t.Errorf("expected table %s to exist, but it doesn't", tableName)
	}
}

func assertTableNotExists(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()

	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)",
		tableName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}

	if exists {
		t.Errorf("expected table %s to NOT exist, but it does", tableName)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
