package commands

import (
	"fmt"
	"os"
	"testing"
)

func TestNewMigrateCommand(t *testing.T) {
	cmd := NewMigrateCommand()

	if cmd.Use != "migrate" {
		t.Errorf("expected Use to be 'migrate', got %s", cmd.Use)
	}

	// Check subcommands are registered
	expectedSubcommands := []string{
		"up",
		"down",
		"status",
		"rollback",
	}

	for _, expected := range expectedSubcommands {
		found := false
		for _, cmd := range cmd.Commands() {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %s to be registered", expected)
		}
	}
}

func TestCategorizeDatabaseError(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		verbose        bool
		expectedSubstr string
	}{
		{
			name:           "syntax error in verbose mode",
			err:            fmt.Errorf("syntax error at or near \"CRATE\""),
			verbose:        true,
			expectedSubstr: "syntax error",
		},
		{
			name:           "syntax error in non-verbose mode",
			err:            fmt.Errorf("syntax error at or near \"CRATE\""),
			verbose:        false,
			expectedSubstr: "SQL syntax error",
		},
		{
			name:           "constraint violation",
			err:            fmt.Errorf("violates foreign key constraint"),
			verbose:        false,
			expectedSubstr: "constraint violation",
		},
		{
			name:           "does not exist error",
			err:            fmt.Errorf("relation \"users\" does not exist"),
			verbose:        false,
			expectedSubstr: "does not exist",
		},
		{
			name:           "already exists error",
			err:            fmt.Errorf("relation \"users\" already exists"),
			verbose:        false,
			expectedSubstr: "already exists",
		},
		{
			name:           "permission denied",
			err:            fmt.Errorf("permission denied for table users"),
			verbose:        false,
			expectedSubstr: "permission denied",
		},
		{
			name:           "generic error",
			err:            fmt.Errorf("some other database error"),
			verbose:        false,
			expectedSubstr: "migration failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := categorizeDatabaseError(tc.err, tc.verbose)

			if !containsMigrateStr(result, tc.expectedSubstr) {
				t.Errorf("expected result to contain %q, got %q", tc.expectedSubstr, result)
			}

			// In verbose mode, should return the full error
			if tc.verbose && result != tc.err.Error() {
				t.Errorf("in verbose mode, expected full error %q, got %q", tc.err.Error(), result)
			}
		})
	}
}

func TestRunMigrateUp_NoDatabaseURL(t *testing.T) {
	// Save and clear DATABASE_URL
	oldURL := os.Getenv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", oldURL)

	cmd := newMigrateUpCommand()
	err := runMigrateUp(cmd, []string{})

	if err == nil {
		t.Error("expected error when DATABASE_URL not set, got nil")
	}
	if err != nil && !containsMigrateStr(err.Error(), "DATABASE_URL") {
		t.Errorf("expected error about DATABASE_URL, got: %v", err)
	}
}

func TestRunMigrateUp_NoMigrationFiles(t *testing.T) {
	// This test requires a valid database connection, so we skip it if DATABASE_URL is not set
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database test")
	}

	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create migrations directory but no files
	if err := os.MkdirAll("migrations", 0755); err != nil {
		t.Fatalf("failed to create migrations directory: %v", err)
	}

	cmd := newMigrateUpCommand()
	err := runMigrateUp(cmd, []string{})

	// Should not error, just report no migrations found
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewMigrateUpCommand_VerboseFlag(t *testing.T) {
	cmd := newMigrateUpCommand()

	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected --verbose flag to be registered")
	}
}

func TestNewMigrateDownCommand_VerboseFlag(t *testing.T) {
	cmd := newMigrateDownCommand()

	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected --verbose flag to be registered")
	}
}

func TestNewMigrateRollbackCommand_Flags(t *testing.T) {
	cmd := newMigrateRollbackCommand()

	if cmd.Flags().Lookup("steps") == nil {
		t.Error("expected --steps flag to be registered")
	}

	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected --verbose flag to be registered")
	}
}

// Helper function
func containsMigrateStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findMigrateSubStr(s, substr)))
}

func findMigrateSubStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidateMigrationSQL(t *testing.T) {
	testCases := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "safe CREATE TABLE",
			sql:         "CREATE TABLE users (id SERIAL PRIMARY KEY);",
			expectError: false,
		},
		{
			name:        "safe ALTER TABLE",
			sql:         "ALTER TABLE users ADD COLUMN email VARCHAR(255);",
			expectError: false,
		},
		{
			name:        "safe INSERT",
			sql:         "INSERT INTO users (name) VALUES ('test');",
			expectError: false,
		},
		{
			name:        "dangerous DROP DATABASE",
			sql:         "DROP DATABASE production;",
			expectError: true,
		},
		{
			name:        "dangerous DROP SCHEMA",
			sql:         "DROP SCHEMA public;",
			expectError: true,
		},
		{
			name:        "dangerous TRUNCATE",
			sql:         "TRUNCATE TABLE users;",
			expectError: true,
		},
		{
			name:        "dangerous GRANT",
			sql:         "GRANT ALL PRIVILEGES ON DATABASE mydb TO user;",
			expectError: true,
		},
		{
			name:        "dangerous REVOKE",
			sql:         "REVOKE ALL PRIVILEGES ON DATABASE mydb FROM user;",
			expectError: true,
		},
		{
			name:        "lowercase dangerous command",
			sql:         "drop database test;",
			expectError: true,
		},
		{
			name:        "mixed case dangerous command",
			sql:         "TrUnCaTe TABLE users;",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMigrationSQL(tc.sql)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error for SQL: %q", tc.sql)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for SQL: %q, got %v", tc.sql, err)
				}
			}
		})
	}
}

func TestExtractVersionFromFilename(t *testing.T) {
	testCases := []struct {
		name            string
		filename        string
		expectedVersion int64
		expectedName    string
		expectError     bool
	}{
		{
			name:            "valid up migration",
			filename:        "001_create_users.up.sql",
			expectedVersion: 1,
			expectedName:    "create_users",
			expectError:     false,
		},
		{
			name:            "valid down migration",
			filename:        "002_add_email.down.sql",
			expectedVersion: 2,
			expectedName:    "add_email",
			expectError:     false,
		},
		{
			name:            "timestamp version",
			filename:        "20250101120000_init.up.sql",
			expectedVersion: 20250101120000,
			expectedName:    "init",
			expectError:     false,
		},
		{
			name:            "no extension",
			filename:        "001_create_users",
			expectedVersion: 1,
			expectedName:    "create_users",
			expectError:     false,
		},
		{
			name:        "invalid no version",
			filename:    "create_users.sql",
			expectError: true,
		},
		{
			name:            "invalid no underscore",
			filename:        "001.sql",
			expectedVersion: 1,
			expectedName:    "001.sql", // When no underscore, entire filename is used as name
			expectError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			version, name, err := extractVersionFromFilename(tc.filename)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error for filename: %q", tc.filename)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for filename: %q, got %v", tc.filename, err)
				}
				if version != tc.expectedVersion {
					t.Errorf("expected version %d, got %d", tc.expectedVersion, version)
				}
				if name != tc.expectedName && tc.expectedName != "" {
					t.Errorf("expected name %q, got %q", tc.expectedName, name)
				}
			}
		})
	}
}
