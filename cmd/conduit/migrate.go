package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
	Long:  "Run and manage database migrations",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	Long:  "Apply all pending database migrations from the migrations/ directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get DATABASE_URL from environment
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return fmt.Errorf("DATABASE_URL environment variable not set\n\nExample:\n  export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
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
			fmt.Println("No migration files found in migrations/")
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
				fmt.Printf("  Skipping invalid migration file: %s (%v)\n", filename, err)
				continue
			}

			// Skip if already applied
			if applied[version] {
				continue
			}

			pending++
			fmt.Printf("Applying migration: %s\n", filename)

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

			// Try to find corresponding down migration
			downFile := strings.Replace(file, ".up.sql", ".down.sql", 1)
			var downSQL string
			if downContent, err := os.ReadFile(downFile); err == nil {
				downSQL = string(downContent)
			}

			// Execute migration in a transaction
			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("failed to start transaction: %w", err)
			}

			// Execute migration SQL
			if _, err := tx.Exec(upSQL); err != nil {
				tx.Rollback()
				// Don't expose SQL content in error message
				return fmt.Errorf("migration %s failed (check database logs for details)", filename)
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

			fmt.Printf("  ✓ Applied %s\n", filename)
		}

		if pending == 0 {
			fmt.Println("No pending migrations")
		} else {
			fmt.Printf("\n✓ Applied %d migration(s)\n", pending)
		}

		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long:  "Display which migrations have been applied",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get DATABASE_URL from environment
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return fmt.Errorf("DATABASE_URL environment variable not set\n\nExample:\n  export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
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
			fmt.Println("No migration files found in migrations/")
			return nil
		}

		// Sort migration files
		sort.Strings(migrationFiles)

		// Display status
		fmt.Println("Migration Status:")
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
				fmt.Printf("  Invalid migration file: %s (%v)\n", filename, err)
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
				icon = "✓"
			}

			fmt.Printf("%s %s [%s]\n", icon, filename, status)
		}

		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Total: %d migrations (%d applied, %d pending)\n",
			len(uniqueVersions),
			len(applied),
			len(uniqueVersions)-len(applied))

		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the last migration",
	Long:  "Rollback the most recently applied migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get DATABASE_URL from environment
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return fmt.Errorf("DATABASE_URL environment variable not set\n\nExample:\n  export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
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
			fmt.Println("No migrations to rollback")
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to get last migration: %w", err)
		}

		fmt.Printf("Rolling back migration: %s (version %d, applied %s)\n", name, version, appliedAt)

		// Check if we have down SQL stored
		if !downSQL.Valid || downSQL.String == "" {
			return fmt.Errorf("migration has no rollback SQL stored in database")
		}

		// Execute rollback in a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// Execute down migration SQL
		if _, err := tx.Exec(downSQL.String); err != nil {
			tx.Rollback()
			return fmt.Errorf("rollback failed (check database logs for details): %w", err)
		}

		// Remove migration record
		if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to remove migration record: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit rollback: %w", err)
		}

		fmt.Printf("✓ Rolled back migration: %s (version %d)\n", name, version)
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist
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

// getAppliedMigrations returns a map of applied migration versions
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

// extractVersionFromFilename extracts the numeric version from a migration filename
// Expected format: {version}_{name}.{up|down}.sql
// Example: 001_create_posts.up.sql -> 1
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
