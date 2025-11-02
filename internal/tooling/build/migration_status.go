// Package build provides build tooling including migration status detection
package build

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

// MigrationInfo represents information about a migration
type MigrationInfo struct {
	Version  int64
	Name     string
	Breaking bool
	DataLoss bool
}

// MigrationStatus contains the status of migrations
type MigrationStatus struct {
	Pending         []MigrationInfo
	HasBreaking     bool
	HasDataLoss     bool
	DatabaseError   error
	DatabaseSkipped bool
}

// CheckMigrationStatus checks for pending migrations before server startup
// Returns status information about pending migrations
func CheckMigrationStatus() (*MigrationStatus, error) {
	status := &MigrationStatus{
		Pending: []MigrationInfo{},
	}

	// Get DATABASE_URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Gracefully degrade if DATABASE_URL not set
		status.DatabaseSkipped = true
		return status, nil
	}

	// Create context with timeout to avoid hanging startup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Connect to database
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		status.DatabaseError = fmt.Errorf("failed to connect: %w", err)
		return status, nil
	}
	defer db.Close()
	db.SetMaxIdleConns(0) // Prevent connection pooling for this short-lived check

	// Test connection with context
	if err := db.PingContext(ctx); err != nil {
		status.DatabaseError = fmt.Errorf("database unreachable: %w", err)
		return status, nil
	}

	// Get applied migrations from database
	applied, err := getAppliedMigrationVersions(ctx, db)
	if err != nil {
		status.DatabaseError = fmt.Errorf("failed to query migrations: %w", err)
		return status, nil
	}

	// Find migration files
	migrationFiles, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("failed to find migration files: %w", err)
	}

	// No migrations directory or files - nothing to check
	if len(migrationFiles) == 0 {
		return status, nil
	}

	// Sort migration files
	sort.Strings(migrationFiles)

	// Check each migration file
	for _, file := range migrationFiles {
		filename := filepath.Base(file)

		// Skip down migrations
		if strings.Contains(filename, ".down.sql") {
			continue
		}

		// Extract version from filename
		version, name, err := extractVersionFromFilename(filename)
		if err != nil {
			// Skip invalid migration files
			continue
		}

		// Check if migration is pending
		if !applied[version] {
			info := MigrationInfo{
				Version:  version,
				Name:     name,
				Breaking: isBreakingMigration(filename),
				DataLoss: isDataLossMigration(filename),
			}

			status.Pending = append(status.Pending, info)

			if info.Breaking {
				status.HasBreaking = true
			}
			if info.DataLoss {
				status.HasDataLoss = true
			}
		}
	}

	return status, nil
}

// getAppliedMigrationVersions returns a map of applied migration versions
func getAppliedMigrationVersions(ctx context.Context, db *sql.DB) (map[int64]bool, error) {
	// Check if schema_migrations table exists
	var exists bool
	err := db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)",
		"schema_migrations").Scan(&exists)
	if err != nil {
		return nil, err
	}

	// If table doesn't exist, no migrations have been applied
	if !exists {
		return make(map[int64]bool), nil
	}

	// Query applied migrations
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations ORDER BY version")
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

// extractVersionFromFilename extracts the numeric version and name from a migration filename
// Expected format: {version}_{name}.{up|down}.sql
// Example: 001_create_posts.up.sql -> 1, "create_posts"
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

// isBreakingMigration checks if a migration is marked as breaking
// Heuristic: filename contains "breaking" or file content has special marker
func isBreakingMigration(filename string) bool {
	// Simple heuristic based on filename
	lower := strings.ToLower(filename)
	return strings.Contains(lower, "breaking") ||
		strings.Contains(lower, "drop_") ||
		strings.Contains(lower, "remove_")
}

// isDataLossMigration checks if a migration may cause data loss
// Heuristic: filename indicates destructive operations
func isDataLossMigration(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.Contains(lower, "drop_") ||
		strings.Contains(lower, "delete_") ||
		strings.Contains(lower, "truncate_")
}

// PrintMigrationWarning prints a formatted warning about pending migrations
func PrintMigrationWarning(status *MigrationStatus) {
	if status.DatabaseSkipped {
		fmt.Println("Warning: DATABASE_URL not set - skipping migration check")
		return
	}

	if status.DatabaseError != nil {
		fmt.Printf("Warning: Could not check migration status: %v\n", status.DatabaseError)
		return
	}

	if len(status.Pending) == 0 {
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("WARNING: PENDING MIGRATIONS DETECTED")
	fmt.Println(strings.Repeat("=", 70))

	// Separate breaking/data-loss migrations from safe ones
	var breaking, dataLoss, safe []MigrationInfo
	for _, m := range status.Pending {
		if m.Breaking {
			breaking = append(breaking, m)
		} else if m.DataLoss {
			dataLoss = append(dataLoss, m)
		} else {
			safe = append(safe, m)
		}
	}

	// Print breaking migrations first (most critical)
	if len(breaking) > 0 {
		fmt.Printf("\n  BREAKING MIGRATIONS (%d):\n", len(breaking))
		for _, m := range breaking {
			fmt.Printf("    - %s\n", m.Name)
		}
	}

	// Print data-loss migrations
	if len(dataLoss) > 0 {
		fmt.Printf("\n  DATA LOSS RISK (%d):\n", len(dataLoss))
		for _, m := range dataLoss {
			fmt.Printf("    - %s\n", m.Name)
		}
	}

	// Print safe migrations
	if len(safe) > 0 {
		fmt.Printf("\n  PENDING MIGRATIONS (%d):\n", len(safe))
		for _, m := range safe {
			fmt.Printf("    - %s\n", m.Name)
		}
	}

	fmt.Println("\n  ACTION REQUIRED:")
	fmt.Println("    Run: conduit migrate up")
	fmt.Println()

	if status.HasBreaking || status.HasDataLoss {
		fmt.Println("  Note: Some migrations may require manual review before applying.")
	}

	fmt.Println(strings.Repeat("=", 70) + "\n")
}
