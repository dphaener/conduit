package commands

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/internal/cli/config"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

var (
	migrateVerbose bool
)

// validateMigrationSQL validates migration SQL for dangerous operations
func validateMigrationSQL(sql string) error {
	// Check for dangerous operations
	dangerous := []string{
		"DROP DATABASE",
		"DROP SCHEMA",
		"TRUNCATE",
		"GRANT",
		"REVOKE",
	}

	upperSQL := strings.ToUpper(sql)
	for _, pattern := range dangerous {
		if strings.Contains(upperSQL, pattern) {
			return fmt.Errorf("migration contains potentially dangerous operation: %s", pattern)
		}
	}
	return nil
}

// categorizeDatabaseError returns a user-friendly error message based on the database error
// In verbose mode, it returns the full error; otherwise, it returns a categorized message
func categorizeDatabaseError(err error, verbose bool) string {
	if verbose {
		return err.Error()
	}

	errStr := strings.ToLower(err.Error())

	// Categorize common database errors
	if strings.Contains(errStr, "syntax") {
		return "SQL syntax error - use --verbose for details"
	}
	if strings.Contains(errStr, "constraint") || strings.Contains(errStr, "violates") {
		return "constraint violation - use --verbose for details"
	}
	if strings.Contains(errStr, "does not exist") {
		return "referenced object does not exist - use --verbose for details"
	}
	if strings.Contains(errStr, "already exists") {
		return "object already exists - use --verbose for details"
	}
	if strings.Contains(errStr, "permission denied") || strings.Contains(errStr, "access denied") {
		return "permission denied - check database user privileges"
	}

	// Generic error for everything else
	return "migration failed - use --verbose for details"
}

// NewMigrateCommand creates the migrate command
func NewMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
		Long: `Run and manage database migrations.

Migrations are stored in the migrations/ directory as SQL files.
Each migration should have an up and down file:
  001_create_users.up.sql
  001_create_users.down.sql

Available subcommands:
  up       - Apply all pending migrations
  down     - Rollback the last migration
  status   - Show migration status
  rollback - Rollback to a specific version`,
	}

	cmd.AddCommand(newMigrateUpCommand())
	cmd.AddCommand(newMigrateDownCommand())
	cmd.AddCommand(newMigrateStatusCommand())
	cmd.AddCommand(newMigrateRollbackCommand())

	return cmd
}

func newMigrateUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		Long:  "Apply all pending database migrations from the migrations/ directory",
		RunE:  runMigrateUp,
	}

	cmd.Flags().BoolVarP(&migrateVerbose, "verbose", "v", false, "Show detailed error messages")

	return cmd
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	successColor := color.New(color.FgGreen, color.Bold)
	infoColor := color.New(color.FgCyan)
	errorColor := color.New(color.FgRed, color.Bold)

	// Get DATABASE_URL
	dbURL := config.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not set\n\nExample:\n  export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
	}

	// Connect to database
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Find migration files
	migrationFiles, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return fmt.Errorf("failed to find migration files: %w", err)
	}

	if len(migrationFiles) == 0 {
		infoColor.Println("No migration files found in migrations/")
		return nil
	}

	// Sort migration files
	sort.Strings(migrationFiles)

	// Apply pending migrations
	pending := 0
	for _, file := range migrationFiles {
		filename := filepath.Base(file)

		// Skip down migrations
		if strings.Contains(filename, ".down.sql") {
			continue
		}

		// Extract version and name from filename
		version, name, err := extractVersionFromFilename(filename)
		if err != nil {
			errorColor.Printf("  Skipping invalid migration file: %s (%v)\n", filename, err)
			continue
		}

		// Skip if already applied
		if applied[version] {
			continue
		}

		pending++
		infoColor.Printf("Applying migration: %s\n", filename)

		// Read migration file
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		// Validate migration file size (prevent DoS)
		if len(content) > 10*1024*1024 { // 10MB limit
			return fmt.Errorf("migration %s exceeds maximum size", filename)
		}

		upSQL := string(content)

		// Validate migration SQL for dangerous operations
		if err := validateMigrationSQL(upSQL); err != nil {
			return fmt.Errorf("migration validation failed: %w", err)
		}

		// Try to find corresponding down migration
		downFile := strings.Replace(file, ".up.sql", ".down.sql", 1)
		var downSQL string
		if downContent, err := os.ReadFile(downFile); err == nil {
			downSQL = string(downContent)
			// Validate down migration as well
			if err := validateMigrationSQL(downSQL); err != nil {
				return fmt.Errorf("down migration validation failed: %w", err)
			}
		}

		// Execute migration in a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// Execute migration SQL
		if _, err := tx.Exec(upSQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s failed: %s", filename, categorizeDatabaseError(err, migrateVerbose))
		}

		// Record migration with up and down SQL
		recordQuery := `
			INSERT INTO schema_migrations (version, name, up_sql, down_sql)
			VALUES ($1, $2, $3, $4)
		`
		if _, err := tx.Exec(recordQuery, version, name, upSQL, downSQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
		}

		successColor.Printf("  ✓ Applied %s\n", filename)
	}

	if pending == 0 {
		infoColor.Println("No pending migrations")
	} else {
		successColor.Printf("\n✓ Applied %d migration(s)\n", pending)
	}

	return nil
}

func newMigrateDownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		Long:  "Rollback the most recently applied migration",
		RunE:  runMigrateDown,
	}

	cmd.Flags().BoolVarP(&migrateVerbose, "verbose", "v", false, "Show detailed error messages")

	return cmd
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	successColor := color.New(color.FgGreen, color.Bold)
	infoColor := color.New(color.FgCyan)

	// Get DATABASE_URL
	dbURL := config.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not set\n\nExample:\n  export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
	}

	// Connect to database
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get last applied migration
	var version int64
	var name string
	var appliedAt string
	var downSQL sql.NullString
	err = db.QueryRow(`
		SELECT version, name, applied_at, down_sql
		FROM schema_migrations
		ORDER BY version DESC
		LIMIT 1
	`).Scan(&version, &name, &appliedAt, &downSQL)
	if err == sql.ErrNoRows {
		infoColor.Println("No migrations to rollback")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	infoColor.Printf("Rolling back migration: %s (version %d, applied %s)\n", name, version, appliedAt)

	// Check if we have down SQL stored
	if !downSQL.Valid || downSQL.String == "" {
		return fmt.Errorf("migration has no rollback SQL stored in database")
	}

	// Validate down migration SQL
	if err := validateMigrationSQL(downSQL.String); err != nil {
		return fmt.Errorf("down migration validation failed: %w", err)
	}

	// Execute rollback in a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Execute down migration SQL
	if _, err := tx.Exec(downSQL.String); err != nil {
		tx.Rollback()
		return fmt.Errorf("rollback failed: %s", categorizeDatabaseError(err, migrateVerbose))
	}

	// Remove migration record
	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	successColor.Printf("✓ Rolled back migration: %s (version %d)\n", name, version)
	return nil
}

func newMigrateStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Long:  "Display which migrations have been applied",
		RunE:  runMigrateStatus,
	}
}

func runMigrateStatus(cmd *cobra.Command, args []string) error {
	infoColor := color.New(color.FgCyan)
	successColor := color.New(color.FgGreen)
	warningColor := color.New(color.FgYellow)

	// Get DATABASE_URL
	dbURL := config.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not set\n\nExample:\n  export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
	}

	// Connect to database
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Find migration files
	migrationFiles, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return fmt.Errorf("failed to find migration files: %w", err)
	}

	if len(migrationFiles) == 0 {
		warningColor.Println("No migration files found in migrations/")
		return nil
	}

	// Sort migration files
	sort.Strings(migrationFiles)

	// Display status
	infoColor.Println("Migration Status:")
	fmt.Println(strings.Repeat("-", 60))

	uniqueVersions := make(map[int64]string)
	for _, file := range migrationFiles {
		filename := filepath.Base(file)

		// Skip down migrations
		if strings.Contains(filename, ".down.sql") {
			continue
		}

		version, _, err := extractVersionFromFilename(filename)
		if err != nil {
			warningColor.Printf("  Invalid migration file: %s (%v)\n", filename, err)
			continue
		}

		uniqueVersions[version] = filename
	}

	// Sort versions for display
	var versions []int64
	for v := range uniqueVersions {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] < versions[j]
	})

	for _, version := range versions {
		filename := uniqueVersions[version]
		status := "pending"
		icon := "○"

		if applied[version] {
			status = "applied"
			icon = successColor.Sprint("✓")
		} else {
			icon = warningColor.Sprint(icon)
		}

		fmt.Printf("%s %s [%s]\n", icon, filename, status)
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total: %d migrations (%d applied, %d pending)\n",
		len(uniqueVersions),
		len(applied),
		len(uniqueVersions)-len(applied))

	return nil
}

func newMigrateRollbackCommand() *cobra.Command {
	var rollbackSteps int

	cmd := &cobra.Command{
		Use:   "rollback [--steps N]",
		Short: "Rollback N migrations",
		Long:  "Rollback the last N migrations (default: 1)",
		RunE: func(cmdInner *cobra.Command, args []string) error {
			successColor := color.New(color.FgGreen, color.Bold)
			infoColor := color.New(color.FgCyan)

			// Get DATABASE_URL
			dbURL := config.GetDatabaseURL()
			if dbURL == "" {
				return fmt.Errorf("DATABASE_URL not set\n\nExample:\n  export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
			}

			// Connect to database
			db, err := sql.Open("pgx", dbURL)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer db.Close()

			// Test connection
			if err := db.Ping(); err != nil {
				return fmt.Errorf("failed to ping database: %w", err)
			}

			// Create migrations table if it doesn't exist
			if err := createMigrationsTable(db); err != nil {
				return fmt.Errorf("failed to create migrations table: %w", err)
			}

			// Rollback N migrations
			for i := 0; i < rollbackSteps; i++ {
				// Get last applied migration
				var version int64
				var name string
				var downSQL sql.NullString
				err = db.QueryRow(`
					SELECT version, name, down_sql
					FROM schema_migrations
					ORDER BY version DESC
					LIMIT 1
				`).Scan(&version, &name, &downSQL)

				if err == sql.ErrNoRows {
					if i == 0 {
						infoColor.Println("No migrations to rollback")
					} else {
						successColor.Printf("\n✓ Rolled back %d migration(s)\n", i)
					}
					return nil
				}
				if err != nil {
					return fmt.Errorf("failed to get last migration: %w", err)
				}

				infoColor.Printf("Rolling back migration %d/%d: %s (version %d)\n", i+1, rollbackSteps, name, version)

				// Check if we have down SQL stored
				if !downSQL.Valid || downSQL.String == "" {
					return fmt.Errorf("migration has no rollback SQL stored in database")
				}

				// Validate down migration SQL
				if err := validateMigrationSQL(downSQL.String); err != nil {
					return fmt.Errorf("down migration validation failed: %w", err)
				}

				// Execute rollback in a transaction
				tx, err := db.Begin()
				if err != nil {
					return fmt.Errorf("failed to start transaction: %w", err)
				}

				// Execute down migration SQL
				if _, err := tx.Exec(downSQL.String); err != nil {
					tx.Rollback()
					return fmt.Errorf("rollback failed: %s", categorizeDatabaseError(err, migrateVerbose))
				}

				// Remove migration record
				if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to remove migration record: %w", err)
				}

				if err := tx.Commit(); err != nil {
					return fmt.Errorf("failed to commit rollback: %w", err)
				}

				successColor.Printf("  ✓ Rolled back %s\n", name)
			}

			successColor.Printf("\n✓ Rolled back %d migration(s)\n", rollbackSteps)
			return nil
		},
	}

	cmd.Flags().IntVarP(&rollbackSteps, "steps", "n", 1, "Number of migrations to rollback")
	cmd.Flags().BoolVarP(&migrateVerbose, "verbose", "v", false, "Show detailed error messages")

	return cmd
}

// Helper functions

func createMigrationsTable(db *sql.DB) error {
	query := `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    breaking BOOLEAN NOT NULL DEFAULT FALSE,
    data_loss BOOLEAN NOT NULL DEFAULT FALSE,
    up_sql TEXT,
    down_sql TEXT
);

CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at
ON schema_migrations(applied_at);
`

	_, err := db.Exec(query)
	return err
}

func getAppliedMigrations(db *sql.DB) (map[int64]bool, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func extractVersionFromFilename(filename string) (int64, string, error) {
	// Remove .sql extension
	name := strings.TrimSuffix(filename, ".sql")

	// Remove .up or .down suffix
	name = strings.TrimSuffix(name, ".up")
	name = strings.TrimSuffix(name, ".down")

	// Split on first underscore to separate version from name
	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 1 {
		return 0, "", fmt.Errorf("invalid migration filename format: %s", filename)
	}

	// Parse version number
	var version int64
	_, err := fmt.Sscanf(parts[0], "%d", &version)
	if err != nil {
		return 0, "", fmt.Errorf("invalid version number in filename %s: %w", filename, err)
	}

	migrationName := filename
	if len(parts) > 1 {
		migrationName = parts[1]
	}

	return version, migrationName, nil
}
