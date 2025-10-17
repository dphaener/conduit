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
			name := filepath.Base(file)

			// Skip if already applied
			if applied[name] {
				continue
			}

			pending++
			fmt.Printf("Applying migration: %s\n", name)

			// Read migration file
			content, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("failed to read migration %s: %w", name, err)
			}

			// Validate migration file size (prevent DoS)
			if len(content) > 10*1024*1024 { // 10MB limit
				return fmt.Errorf("migration %s exceeds maximum size", name)
			}

			// Execute migration in a transaction
			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("failed to start transaction: %w", err)
			}

			// Execute migration SQL
			if _, err := tx.Exec(string(content)); err != nil {
				tx.Rollback()
				// Don't expose SQL content in error message
				return fmt.Errorf("migration %s failed (check database logs for details)", name)
			}

			// Record migration
			if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", name); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to record migration %s: %w", name, err)
			}

			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit migration %s: %w", name, err)
			}

			fmt.Printf("  ✓ Applied %s\n", name)
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

		for _, file := range migrationFiles {
			name := filepath.Base(file)
			status := "pending"
			icon := "○"

			if applied[name] {
				status = "applied"
				icon = "✓"
			}

			fmt.Printf("%s %s [%s]\n", icon, name, status)
		}

		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Total: %d migrations (%d applied, %d pending)\n",
			len(migrationFiles),
			len(applied),
			len(migrationFiles)-len(applied))

		return nil
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist
func createMigrationsTable(db *sql.DB) error {
	query := `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
)`

	_, err := db.Exec(query)
	return err
}

// getAppliedMigrations returns a map of applied migration names
func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}
