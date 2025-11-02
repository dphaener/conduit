package build

import (
	"os"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/migrate"
)

// TestCheckProductionEnvironment tests environment detection logic
func TestCheckProductionEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		shouldBlock bool
		description string
	}{
		{
			name:        "development environment",
			envVars:     map[string]string{"ENV": "development"},
			shouldBlock: false,
			description: "development environment should not block",
		},
		{
			name:        "staging environment",
			envVars:     map[string]string{"ENV": "staging"},
			shouldBlock: false,
			description: "staging environment should not block",
		},
		{
			name:        "production via ENV",
			envVars:     map[string]string{"ENV": "production"},
			shouldBlock: true,
			description: "ENV=production should block",
		},
		{
			name:        "production via RAILS_ENV",
			envVars:     map[string]string{"RAILS_ENV": "production"},
			shouldBlock: true,
			description: "RAILS_ENV=production should block",
		},
		{
			name:        "production via NODE_ENV",
			envVars:     map[string]string{"NODE_ENV": "production"},
			shouldBlock: true,
			description: "NODE_ENV=production should block",
		},
		{
			name:        "production via CONDUIT_ENV",
			envVars:     map[string]string{"CONDUIT_ENV": "production"},
			shouldBlock: true,
			description: "CONDUIT_ENV=production should block",
		},
		{
			name:        "production short form (prod)",
			envVars:     map[string]string{"ENV": "prod"},
			shouldBlock: true,
			description: "ENV=prod should block",
		},
		{
			name:        "production short form (prd)",
			envVars:     map[string]string{"ENV": "prd"},
			shouldBlock: true,
			description: "ENV=prd should block",
		},
		{
			name:        "production in DATABASE_URL",
			envVars:     map[string]string{"DATABASE_URL": "postgres://user:pass@production.db.example.com/db"},
			shouldBlock: true,
			description: "production in DATABASE_URL hostname should block",
		},
		{
			name:        "prod subdomain in DATABASE_URL",
			envVars:     map[string]string{"DATABASE_URL": "postgres://user:pass@prod.db.example.com/db"},
			shouldBlock: true,
			description: "prod subdomain in DATABASE_URL should block",
		},
		{
			name:        "prd in DATABASE_URL",
			envVars:     map[string]string{"DATABASE_URL": "postgres://user:pass@prd-db.example.com/db"},
			shouldBlock: true,
			description: "prd in DATABASE_URL should block",
		},
		{
			name:        "localhost database",
			envVars:     map[string]string{"DATABASE_URL": "postgres://user:pass@localhost/mydb"},
			shouldBlock: false,
			description: "localhost DATABASE_URL should not block",
		},
		{
			name:        "no environment variables",
			envVars:     map[string]string{},
			shouldBlock: false,
			description: "no env vars should not block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env vars
			originalEnv := make(map[string]string)
			envVarNames := []string{"ENV", "RAILS_ENV", "NODE_ENV", "CONDUIT_ENV", "ENVIRONMENT", "DATABASE_URL"}
			for _, name := range envVarNames {
				originalEnv[name] = os.Getenv(name)
				os.Unsetenv(name)
			}
			defer func() {
				for name, val := range originalEnv {
					if val != "" {
						os.Setenv(name, val)
					} else {
						os.Unsetenv(name)
					}
				}
			}()

			// Set test env vars
			for name, val := range tt.envVars {
				os.Setenv(name, val)
			}

			// Create auto-migrator and check environment with SkipConfirm to avoid prompts
			am := NewAutoMigrator(AutoMigrateOptions{SkipConfirm: true})
			err := am.checkProductionEnvironment()

			if tt.shouldBlock {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: expected no error but got: %v", tt.description, err)
				}
			}
		})
	}
}

// TestAutoMigratorIsBreakingMigration tests migration safety classification
func TestAutoMigratorIsBreakingMigration(t *testing.T) {
	tests := []struct {
		name        string
		migration   *migrate.Migration
		isBreaking  bool
		description string
	}{
		{
			name: "CREATE TABLE is safe",
			migration: &migrate.Migration{
				Name: "create_users",
				Up:   "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(255));",
			},
			isBreaking:  false,
			description: "CREATE TABLE should be safe",
		},
		{
			name: "ADD COLUMN is safe",
			migration: &migrate.Migration{
				Name: "add_email_to_users",
				Up:   "ALTER TABLE users ADD COLUMN email VARCHAR(255);",
			},
			isBreaking:  false,
			description: "ADD COLUMN should be safe",
		},
		{
			name: "CREATE INDEX is safe",
			migration: &migrate.Migration{
				Name: "add_index_on_email",
				Up:   "CREATE INDEX idx_users_email ON users(email);",
			},
			isBreaking:  false,
			description: "CREATE INDEX should be safe",
		},
		{
			name: "DROP TABLE is breaking",
			migration: &migrate.Migration{
				Name: "drop_old_table",
				Up:   "DROP TABLE old_table;",
			},
			isBreaking:  true,
			description: "DROP TABLE should be breaking",
		},
		{
			name: "DROP COLUMN is breaking",
			migration: &migrate.Migration{
				Name: "remove_deprecated_field",
				Up:   "ALTER TABLE users DROP COLUMN deprecated_field;",
			},
			isBreaking:  true,
			description: "DROP COLUMN should be breaking",
		},
		{
			name: "DROP INDEX is breaking",
			migration: &migrate.Migration{
				Name: "remove_old_index",
				Up:   "DROP INDEX idx_old;",
			},
			isBreaking:  true,
			description: "DROP INDEX should be breaking",
		},
		{
			name: "DROP CONSTRAINT is breaking",
			migration: &migrate.Migration{
				Name: "remove_constraint",
				Up:   "ALTER TABLE users DROP CONSTRAINT fk_users_org;",
			},
			isBreaking:  true,
			description: "DROP CONSTRAINT should be breaking",
		},
		{
			name: "TRUNCATE is breaking",
			migration: &migrate.Migration{
				Name: "clear_old_data",
				Up:   "TRUNCATE TABLE logs;",
			},
			isBreaking:  true,
			description: "TRUNCATE should be breaking",
		},
		{
			name: "DELETE FROM is breaking",
			migration: &migrate.Migration{
				Name: "delete_old_records",
				Up:   "DELETE FROM users WHERE created_at < '2020-01-01';",
			},
			isBreaking:  true,
			description: "DELETE FROM should be breaking",
		},
		{
			name: "ALTER COLUMN is breaking",
			migration: &migrate.Migration{
				Name: "change_column_type",
				Up:   "ALTER TABLE users ALTER COLUMN age TYPE INTEGER;",
			},
			isBreaking:  true,
			description: "ALTER COLUMN should be breaking",
		},
		{
			name: "ALTER TYPE is breaking",
			migration: &migrate.Migration{
				Name: "change_enum",
				Up:   "ALTER TYPE status_enum ADD VALUE 'archived';",
			},
			isBreaking:  true,
			description: "ALTER TYPE should be breaking",
		},
		{
			name: "Migration marked as breaking",
			migration: &migrate.Migration{
				Name:     "some_migration",
				Up:       "CREATE TABLE test (id SERIAL);",
				Breaking: true,
			},
			isBreaking:  true,
			description: "Migration with Breaking=true should be breaking",
		},
		{
			name: "Migration marked as data loss",
			migration: &migrate.Migration{
				Name:     "some_migration",
				Up:       "CREATE TABLE test (id SERIAL);",
				DataLoss: true,
			},
			isBreaking:  true,
			description: "Migration with DataLoss=true should be breaking",
		},
		{
			name: "Multiple statements (mixed)",
			migration: &migrate.Migration{
				Name: "complex_migration",
				Up: `
					CREATE TABLE new_table (id SERIAL PRIMARY KEY);
					DROP TABLE old_table;
				`,
			},
			isBreaking:  true,
			description: "Migration with any breaking statement should be breaking",
		},
		{
			name: "ADD COLUMN NOT NULL without DEFAULT is breaking",
			migration: &migrate.Migration{
				Name: "add_required_column",
				Up:   "ALTER TABLE users ADD COLUMN required_field VARCHAR(255) NOT NULL;",
			},
			isBreaking:  true,
			description: "ADD COLUMN NOT NULL without DEFAULT should be breaking (BLOCKER 1)",
		},
		{
			name: "ADD COLUMN NOT NULL with DEFAULT is safe",
			migration: &migrate.Migration{
				Name: "add_required_column_with_default",
				Up:   "ALTER TABLE users ADD COLUMN required_field VARCHAR(255) NOT NULL DEFAULT 'value';",
			},
			isBreaking:  false,
			description: "ADD COLUMN NOT NULL with DEFAULT should be safe",
		},
		{
			name: "ADD COLUMN nullable is safe",
			migration: &migrate.Migration{
				Name: "add_optional_column",
				Up:   "ALTER TABLE users ADD COLUMN optional_field VARCHAR(255);",
			},
			isBreaking:  false,
			description: "ADD COLUMN without NOT NULL should be safe",
		},
		{
			name: "ADD COLUMN with NULL explicit is safe",
			migration: &migrate.Migration{
				Name: "add_null_column",
				Up:   "ALTER TABLE users ADD COLUMN null_field VARCHAR(255) NULL;",
			},
			isBreaking:  false,
			description: "ADD COLUMN with explicit NULL should be safe",
		},
		{
			name: "Multiple ADD COLUMN, first lacks DEFAULT (CRITICAL BUG)",
			migration: &migrate.Migration{
				Name: "add_multiple_columns",
				Up: `ALTER TABLE users
					ADD COLUMN first_name VARCHAR(255) NOT NULL,
					ADD COLUMN last_name VARCHAR(255) NOT NULL DEFAULT '';`,
			},
			isBreaking:  true,
			description: "Multiple ADD COLUMN where first has NOT NULL without DEFAULT should be breaking",
		},
		{
			name: "Multiple ADD COLUMN, second lacks DEFAULT",
			migration: &migrate.Migration{
				Name: "add_multiple_columns_v2",
				Up: `ALTER TABLE users
					ADD COLUMN first_name VARCHAR(255) NOT NULL DEFAULT '',
					ADD COLUMN last_name VARCHAR(255) NOT NULL;`,
			},
			isBreaking:  true,
			description: "Multiple ADD COLUMN where second has NOT NULL without DEFAULT should be breaking",
		},
		{
			name: "Multiple ADD COLUMN, both have DEFAULT",
			migration: &migrate.Migration{
				Name: "add_multiple_columns_safe",
				Up: `ALTER TABLE users
					ADD COLUMN first_name VARCHAR(255) NOT NULL DEFAULT '',
					ADD COLUMN last_name VARCHAR(255) NOT NULL DEFAULT '';`,
			},
			isBreaking:  false,
			description: "Multiple ADD COLUMN where both have DEFAULT should be safe",
		},
		{
			name: "ADD COLUMN with DEFAULT before NOT NULL is safe",
			migration: &migrate.Migration{
				Name: "add_status_column",
				Up:   "ALTER TABLE users ADD COLUMN status VARCHAR(50) DEFAULT 'active' NOT NULL;",
			},
			isBreaking:  false,
			description: "DEFAULT can appear before NOT NULL in PostgreSQL",
		},
		{
			name: "ADD COLUMN with quoted column name and NOT NULL without DEFAULT",
			migration: &migrate.Migration{
				Name: "add_quoted_column",
				Up:   `ALTER TABLE users ADD COLUMN "first_name" VARCHAR(255) NOT NULL;`,
			},
			isBreaking:  true,
			description: "Quoted column names should be handled correctly",
		},
		{
			name: "ADD COLUMN with quoted column name and DEFAULT",
			migration: &migrate.Migration{
				Name: "add_quoted_column_safe",
				Up:   `ALTER TABLE users ADD COLUMN "first_name" VARCHAR(255) NOT NULL DEFAULT 'test';`,
			},
			isBreaking:  false,
			description: "Quoted column names with DEFAULT should be safe",
		},
		{
			name: "Multiple ADD COLUMN with mixed spacing",
			migration: &migrate.Migration{
				Name: "add_columns_spacing",
				Up: `ALTER TABLE users
					ADD COLUMN   first_name   VARCHAR(255)   NOT NULL,
					ADD COLUMN last_name VARCHAR(255) NOT NULL DEFAULT   '';`,
			},
			isBreaking:  true,
			description: "Should handle different whitespace variations",
		},
		{
			name: "Three ADD COLUMN statements, middle one lacks DEFAULT",
			migration: &migrate.Migration{
				Name: "add_three_columns",
				Up: `ALTER TABLE users
					ADD COLUMN first_name VARCHAR(255) NOT NULL DEFAULT '',
					ADD COLUMN middle_name VARCHAR(255) NOT NULL,
					ADD COLUMN last_name VARCHAR(255) NOT NULL DEFAULT '';`,
			},
			isBreaking:  true,
			description: "Should detect NOT NULL without DEFAULT in any position",
		},
		{
			name: "ADD COLUMN with complex DEFAULT expression",
			migration: &migrate.Migration{
				Name: "add_timestamp",
				Up:   "ALTER TABLE users ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;",
			},
			isBreaking:  false,
			description: "Complex DEFAULT expressions should be recognized",
		},
		{
			name: "Multiple ADD COLUMN across multiple ALTER TABLE statements",
			migration: &migrate.Migration{
				Name: "multiple_alter_tables",
				Up: `ALTER TABLE users ADD COLUMN first_name VARCHAR(255) NOT NULL;
					ALTER TABLE posts ADD COLUMN title VARCHAR(255) NOT NULL DEFAULT '';`,
			},
			isBreaking:  true,
			description: "Should check each ALTER TABLE statement independently",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAutoMigrator(AutoMigrateOptions{})
			result := am.isBreakingMigration(tt.migration)

			if result != tt.isBreaking {
				t.Errorf("%s: expected isBreaking=%v but got %v", tt.description, tt.isBreaking, result)
			}
		})
	}
}

// TestClassifyMigrations tests separation of safe and breaking migrations
func TestClassifyMigrations(t *testing.T) {
	migrations := []*migrate.Migration{
		{
			Name: "create_users",
			Up:   "CREATE TABLE users (id SERIAL PRIMARY KEY);",
		},
		{
			Name: "drop_old_table",
			Up:   "DROP TABLE old_table;",
		},
		{
			Name: "add_index",
			Up:   "CREATE INDEX idx_users ON users(email);",
		},
		{
			Name: "remove_column",
			Up:   "ALTER TABLE users DROP COLUMN deprecated;",
		},
	}

	am := NewAutoMigrator(AutoMigrateOptions{})
	safe, breaking := am.classifyMigrations(migrations)

	if len(safe) != 2 {
		t.Errorf("expected 2 safe migrations but got %d", len(safe))
	}

	if len(breaking) != 2 {
		t.Errorf("expected 2 breaking migrations but got %d", len(breaking))
	}

	// Verify safe migrations
	safeNames := make(map[string]bool)
	for _, m := range safe {
		safeNames[m.Name] = true
	}
	if !safeNames["create_users"] || !safeNames["add_index"] {
		t.Errorf("expected 'create_users' and 'add_index' to be safe")
	}

	// Verify breaking migrations
	breakingNames := make(map[string]bool)
	for _, m := range breaking {
		breakingNames[m.Name] = true
	}
	if !breakingNames["drop_old_table"] || !breakingNames["remove_column"] {
		t.Errorf("expected 'drop_old_table' and 'remove_column' to be breaking")
	}
}

// TestParseMigrationFile tests migration file parsing
func TestParseMigrationFile(t *testing.T) {
	tests := []struct {
		name            string
		filepath        string
		expectedVersion int64
		expectedName    string
		shouldError     bool
	}{
		{
			name:            "standard format",
			filepath:        "migrations/001_create_users.up.sql",
			expectedVersion: 1,
			expectedName:    "create_users",
			shouldError:     false,
		},
		{
			name:            "longer version number",
			filepath:        "migrations/20231215120000_add_posts.up.sql",
			expectedVersion: 20231215120000,
			expectedName:    "add_posts",
			shouldError:     false,
		},
		{
			name:            "without .up suffix",
			filepath:        "migrations/002_create_comments.sql",
			expectedVersion: 2,
			expectedName:    "create_comments",
			shouldError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, name, err := extractVersionFromFilename(tt.filepath[len("migrations/"):])

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if version != tt.expectedVersion {
				t.Errorf("expected version %d but got %d", tt.expectedVersion, version)
			}

			if name != tt.expectedName {
				t.Errorf("expected name %s but got %s", tt.expectedName, name)
			}
		})
	}
}

// TestAutoMigrateOptions tests option configuration
func TestAutoMigrateOptions(t *testing.T) {
	t.Run("dry-run mode", func(t *testing.T) {
		opts := AutoMigrateOptions{
			Mode: AutoMigrateDryRun,
		}
		am := NewAutoMigrator(opts)

		if am.opts.Mode != AutoMigrateDryRun {
			t.Errorf("expected dry-run mode")
		}
	})

	t.Run("apply mode", func(t *testing.T) {
		opts := AutoMigrateOptions{
			Mode: AutoMigrateApply,
		}
		am := NewAutoMigrator(opts)

		if am.opts.Mode != AutoMigrateApply {
			t.Errorf("expected apply mode")
		}
	})

	t.Run("force production", func(t *testing.T) {
		// Save original env
		originalEnv := os.Getenv("ENV")
		defer func() {
			if originalEnv != "" {
				os.Setenv("ENV", originalEnv)
			} else {
				os.Unsetenv("ENV")
			}
		}()

		opts := AutoMigrateOptions{
			ForceProduction: true,
		}
		am := NewAutoMigrator(opts)

		// Should not block in production when forced
		os.Setenv("ENV", "production")

		// When ForceProduction is true, checkProductionEnvironment is skipped in Run()
		// So we verify the flag is set correctly
		if !am.opts.ForceProduction {
			t.Errorf("expected ForceProduction to be true")
		}
	})
}
