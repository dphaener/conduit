package build

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestExtractVersionFromFilename(t *testing.T) {
	tests := []struct {
		filename    string
		wantVersion int64
		wantName    string
		wantErr     bool
	}{
		{
			filename:    "001_create_posts.up.sql",
			wantVersion: 1,
			wantName:    "create_posts",
			wantErr:     false,
		},
		{
			filename:    "123_add_users_table.up.sql",
			wantVersion: 123,
			wantName:    "add_users_table",
			wantErr:     false,
		},
		{
			filename:    "001_create_posts.down.sql",
			wantVersion: 1,
			wantName:    "create_posts",
			wantErr:     false,
		},
		{
			filename:    "invalid.sql",
			wantVersion: 0,
			wantName:    "",
			wantErr:     true,
		},
		{
			filename:    "abc_invalid.up.sql",
			wantVersion: 0,
			wantName:    "",
			wantErr:     true,
		},
		{
			filename:    "001.up.sql",
			wantVersion: 1,
			wantName:    "001.up.sql",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			version, name, err := extractVersionFromFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractVersionFromFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if version != tt.wantVersion {
					t.Errorf("extractVersionFromFilename() version = %v, want %v", version, tt.wantVersion)
				}
				if name != tt.wantName {
					t.Errorf("extractVersionFromFilename() name = %v, want %v", name, tt.wantName)
				}
			}
		})
	}
}

func TestIsBreakingMigration(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"001_create_posts.up.sql", false},
		{"002_drop_posts.up.sql", true},
		{"003_remove_column.up.sql", true},
		{"004_breaking_change.up.sql", true},
		{"005_add_index.up.sql", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := isBreakingMigration(tt.filename); got != tt.want {
				t.Errorf("isBreakingMigration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDataLossMigration(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"001_create_posts.up.sql", false},
		{"002_drop_posts.up.sql", true},
		{"003_delete_old_data.up.sql", true},
		{"004_truncate_logs.up.sql", true},
		{"005_add_index.up.sql", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := isDataLossMigration(tt.filename); got != tt.want {
				t.Errorf("isDataLossMigration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckMigrationStatus_NoDatabaseURL(t *testing.T) {
	// Save original DATABASE_URL
	origURL := os.Getenv("DATABASE_URL")
	defer func() {
		if origURL != "" {
			os.Setenv("DATABASE_URL", origURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}()

	// Unset DATABASE_URL
	os.Unsetenv("DATABASE_URL")

	status, err := CheckMigrationStatus()
	if err != nil {
		t.Fatalf("CheckMigrationStatus() error = %v, want nil", err)
	}

	if !status.DatabaseSkipped {
		t.Error("Expected DatabaseSkipped to be true when DATABASE_URL not set")
	}

	if status.DatabaseError != nil {
		t.Errorf("Expected no DatabaseError, got %v", status.DatabaseError)
	}

	if len(status.Pending) != 0 {
		t.Errorf("Expected 0 pending migrations, got %d", len(status.Pending))
	}
}

func TestCheckMigrationStatus_InvalidDatabaseURL(t *testing.T) {
	// Save original DATABASE_URL
	origURL := os.Getenv("DATABASE_URL")
	defer func() {
		if origURL != "" {
			os.Setenv("DATABASE_URL", origURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}()

	// Set invalid DATABASE_URL
	os.Setenv("DATABASE_URL", "postgresql://invalid:5432/nonexistent")

	status, err := CheckMigrationStatus()
	if err != nil {
		t.Fatalf("CheckMigrationStatus() should not return error on connection failure, got: %v", err)
	}

	if status.DatabaseSkipped {
		t.Error("DatabaseSkipped should be false when DATABASE_URL is set")
	}

	if status.DatabaseError == nil {
		t.Error("Expected DatabaseError when database is unreachable")
	}
}

func TestGetAppliedMigrationVersions_NoTable(t *testing.T) {
	// Skip if test database not available
	db, err := sql.Open("pgx", "postgresql://localhost:5432/conduit_test?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not reachable:", err)
	}

	// Clean up
	db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")
	defer db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")

	ctx := context.Background()
	applied, err := getAppliedMigrationVersions(ctx, db)
	if err != nil {
		t.Fatalf("getAppliedMigrationVersions() error = %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("Expected 0 applied migrations when table doesn't exist, got %d", len(applied))
	}
}

func TestGetAppliedMigrationVersions_WithMigrations(t *testing.T) {
	// Skip if test database not available
	db, err := sql.Open("pgx", "postgresql://localhost:5432/conduit_test?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not reachable:", err)
	}

	// Set up test table
	db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")
	defer db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")

	_, err = db.Exec(`
		CREATE TABLE schema_migrations (
			version BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			breaking BOOLEAN NOT NULL DEFAULT FALSE,
			data_loss BOOLEAN NOT NULL DEFAULT FALSE,
			up_sql TEXT,
			down_sql TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert test migrations
	_, err = db.Exec("INSERT INTO schema_migrations (version, name) VALUES (1, 'first'), (2, 'second'), (3, 'third')")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	ctx := context.Background()
	applied, err := getAppliedMigrationVersions(ctx, db)
	if err != nil {
		t.Fatalf("getAppliedMigrationVersions() error = %v", err)
	}

	expectedVersions := []int64{1, 2, 3}
	if len(applied) != len(expectedVersions) {
		t.Errorf("Expected %d applied migrations, got %d", len(expectedVersions), len(applied))
	}

	for _, v := range expectedVersions {
		if !applied[v] {
			t.Errorf("Expected version %d to be applied", v)
		}
	}
}

func TestCheckMigrationStatus_Integration(t *testing.T) {
	// Skip if test database not available
	db, err := sql.Open("pgx", "postgresql://localhost:5432/conduit_test?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not reachable:", err)
	}

	// Save original DATABASE_URL
	origURL := os.Getenv("DATABASE_URL")
	defer func() {
		if origURL != "" {
			os.Setenv("DATABASE_URL", origURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}()

	// Set test DATABASE_URL
	os.Setenv("DATABASE_URL", "postgresql://localhost:5432/conduit_test?sslmode=disable")

	// Set up test table
	db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")
	defer db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")

	_, err = db.Exec(`
		CREATE TABLE schema_migrations (
			version BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			breaking BOOLEAN NOT NULL DEFAULT FALSE,
			data_loss BOOLEAN NOT NULL DEFAULT FALSE,
			up_sql TEXT,
			down_sql TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Create temporary migrations directory
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	err = os.Mkdir("migrations", 0755)
	if err != nil {
		t.Fatalf("Failed to create migrations directory: %v", err)
	}

	// Create test migration files
	migrationFiles := []string{
		"001_create_posts.up.sql",
		"002_add_users.up.sql",
		"003_drop_old_table.up.sql",
	}

	for _, file := range migrationFiles {
		path := filepath.Join("migrations", file)
		if err := os.WriteFile(path, []byte("SELECT 1;"), 0644); err != nil {
			t.Fatalf("Failed to create migration file %s: %v", file, err)
		}
	}

	// Mark first migration as applied
	_, err = db.Exec("INSERT INTO schema_migrations (version, name) VALUES (1, 'create_posts')")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Run the check
	status, err := CheckMigrationStatus()
	if err != nil {
		t.Fatalf("CheckMigrationStatus() error = %v", err)
	}

	// Should have 2 pending migrations (002 and 003)
	if len(status.Pending) != 2 {
		t.Errorf("Expected 2 pending migrations, got %d", len(status.Pending))
	}

	// Check that we detected the breaking migration (003_drop_old_table)
	if !status.HasBreaking {
		t.Error("Expected HasBreaking to be true (003_drop_old_table)")
	}

	// Verify pending migration details
	foundVersion2 := false
	foundVersion3 := false
	for _, m := range status.Pending {
		if m.Version == 2 {
			foundVersion2 = true
		}
		if m.Version == 3 {
			foundVersion3 = true
			if !m.Breaking {
				t.Error("Migration 003_drop_old_table should be marked as breaking")
			}
		}
	}

	if !foundVersion2 {
		t.Error("Expected to find pending migration with version 2")
	}
	if !foundVersion3 {
		t.Error("Expected to find pending migration with version 3")
	}
}

func TestPrintMigrationWarning(t *testing.T) {
	// Test with database skipped
	status := &MigrationStatus{
		DatabaseSkipped: true,
	}
	// Should not panic
	PrintMigrationWarning(status)

	// Test with database error
	status = &MigrationStatus{
		DatabaseError: sql.ErrConnDone,
	}
	// Should not panic
	PrintMigrationWarning(status)

	// Test with no pending migrations
	status = &MigrationStatus{}
	// Should not panic
	PrintMigrationWarning(status)

	// Test with pending migrations
	status = &MigrationStatus{
		Pending: []MigrationInfo{
			{Version: 1, Name: "create_posts", Breaking: false},
			{Version: 2, Name: "drop_old_table", Breaking: true},
		},
		HasBreaking: true,
	}
	// Should not panic
	PrintMigrationWarning(status)
}
