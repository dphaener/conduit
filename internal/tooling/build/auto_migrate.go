package build

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/conduit-lang/conduit/internal/orm/migrate"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

// AutoMigrateMode determines how migrations are applied
type AutoMigrateMode string

const (
	AutoMigrateApply  AutoMigrateMode = "apply"  // Actually apply migrations
	AutoMigrateDryRun AutoMigrateMode = "dry-run" // Show SQL without executing
)

// AutoMigrateOptions configures the auto-migration behavior
type AutoMigrateOptions struct {
	Mode            AutoMigrateMode
	SkipConfirm     bool // Skip confirmation prompts (for tests)
	ForceProduction bool // Allow running in production (dangerous, for tests only)
}

// AutoMigrator handles automatic migration application
type AutoMigrator struct {
	opts AutoMigrateOptions
}

// NewAutoMigrator creates a new auto-migrator
func NewAutoMigrator(opts AutoMigrateOptions) *AutoMigrator {
	return &AutoMigrator{opts: opts}
}

// Run executes the auto-migration workflow
func (am *AutoMigrator) Run() error {
	// Multi-layer production safety checks
	if !am.opts.ForceProduction {
		if err := am.checkProductionEnvironment(); err != nil {
			return err
		}
	}

	// Get DATABASE_URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable not set")
	}

	// Create database connection
	ctx := context.Background()
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database unreachable: %w", err)
	}

	// HIGH 5 FIX: Validate that the database is not read-only (e.g., a read replica)
	// For PostgreSQL, check if it's in recovery mode (read-only replica)
	var inRecovery bool
	err = db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery)
	if err == nil && inRecovery {
		return fmt.Errorf("DATABASE_URL points to a read-only replica (recovery mode) - migrations cannot be applied")
	}
	// Note: We ignore errors here since some databases might not support this function

	// Create migration runner
	runner := migrate.NewRunner(db)

	// Initialize migration tracking table
	if err := runner.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize migration tracking: %w", err)
	}

	// Load all migrations from disk
	migrations, err := am.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	if len(migrations) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	// Get migration status
	tracker := migrate.NewTracker(db)
	pending, err := tracker.GetPending(migrations)
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(pending) == 0 {
		fmt.Println("✓ All migrations up to date")
		return nil
	}

	// Classify migrations by safety
	safe, breaking := am.classifyMigrations(pending)

	// Print status
	fmt.Printf("\nFound %d pending migration(s):\n", len(pending))
	if len(safe) > 0 {
		fmt.Printf("  %d safe (additions)\n", len(safe))
	}
	if len(breaking) > 0 {
		fmt.Printf("  %d breaking (drops/modifications)\n", len(breaking))
	}
	fmt.Print("\n")

	// Handle dry-run mode
	if am.opts.Mode == AutoMigrateDryRun {
		return am.printDryRun(pending)
	}

	// Handle breaking migrations - require confirmation
	if len(breaking) > 0 {
		if !am.opts.SkipConfirm {
			fmt.Println("WARNING: The following migrations contain breaking changes:")
			fmt.Print("\n")
			for _, m := range breaking {
				fmt.Printf("  - %s\n", m.Name)
			}
			fmt.Print("\n")

			if !am.confirmAction("Apply these breaking migrations?") {
				return fmt.Errorf("migration cancelled by user")
			}
		}
	}

	// Apply migrations
	fmt.Printf("\nApplying %d migration(s)...\n", len(pending))

	// HIGH 4 FIX: Wrap ALL migrations in a single transaction for atomicity
	// PostgreSQL supports DDL in transactions, so if any migration fails,
	// all migrations will be rolled back automatically
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback() // Safe to call even after commit

	for _, migration := range pending {
		if err := am.applyMigrationInTx(tx, runner, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w (all migrations rolled back)", migration.Name, err)
		}
	}

	// Commit all migrations atomically
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	fmt.Printf("\n✓ Successfully applied %d migration(s)\n", len(pending))
	return nil
}

// checkProductionEnvironment performs multi-layer checks to detect production
func (am *AutoMigrator) checkProductionEnvironment() error {
	var warnings []string

	// Check 1: DATABASE_URL hostname
	// HIGH 3 FIX: Case-insensitive checking is intentional to catch variations like
	// "Production", "PRODUCTION", "Prod", etc. that might appear in hostnames
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		lower := strings.ToLower(dbURL)
		if strings.Contains(lower, "production") ||
			strings.Contains(lower, "prod.") ||
			strings.Contains(lower, ".prod") ||
			strings.Contains(lower, "prd.") ||
			strings.Contains(lower, ".prd") ||
			strings.Contains(lower, "prd-") ||
			strings.Contains(lower, "-prd") {
			return fmt.Errorf("auto-migrate is BLOCKED: DATABASE_URL appears to be production (contains 'prod'/'production')")
		}
	}

	// Check 2: Environment variables
	// HIGH 3 FIX: Case-insensitive checking is intentional to catch "Production", "PROD", etc.
	envVars := []string{"ENV", "RAILS_ENV", "NODE_ENV", "CONDUIT_ENV", "ENVIRONMENT"}
	for _, envVar := range envVars {
		val := os.Getenv(envVar)
		if val != "" {
			lower := strings.ToLower(val)
			if lower == "production" || lower == "prod" || lower == "prd" {
				return fmt.Errorf("auto-migrate is BLOCKED: %s=%s indicates production environment", envVar, val)
			}
		}
	}

	// Check 3: Git branch (warning with confirmation required)
	// HIGH 3 FIX: Case-insensitive checking is intentional to catch "Main", "MASTER", etc.
	// HIGH 7 FIX: Require confirmation when on production branches
	branch, err := am.getCurrentGitBranch()
	if err == nil {
		lower := strings.ToLower(branch)
		if lower == "main" || lower == "master" || lower == "production" {
			if !am.opts.SkipConfirm {
				fmt.Printf("\nWARNING: Running on production branch '%s'\n", branch)
				fmt.Println("This is typically used for production deployments.")
				if !am.confirmAction("Continue auto-migration?") {
					return fmt.Errorf("migration cancelled by user")
				}
			} else {
				warnings = append(warnings, fmt.Sprintf("running on '%s' branch (typically used for production)", branch))
			}
		}
	}

	// Print warnings if any (non-blocking warnings only)
	if len(warnings) > 0 {
		fmt.Println("\nWARNING: Production environment indicators detected:")
		for _, w := range warnings {
			fmt.Printf("  - %s\n", w)
		}
		fmt.Println()
	}

	return nil
}

// getCurrentGitBranch returns the current git branch name
func (am *AutoMigrator) getCurrentGitBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// classifyMigrations separates safe and breaking migrations
func (am *AutoMigrator) classifyMigrations(migrations []*migrate.Migration) (safe, breaking []*migrate.Migration) {
	for _, m := range migrations {
		if am.isBreakingMigration(m) {
			breaking = append(breaking, m)
		} else {
			safe = append(safe, m)
		}
	}
	return
}

// isBreakingMigration determines if a migration contains breaking changes
func (am *AutoMigrator) isBreakingMigration(m *migrate.Migration) bool {
	// Check migration flags
	if m.Breaking || m.DataLoss {
		return true
	}

	// Check SQL content for breaking operations
	sql := strings.ToUpper(m.Up)

	// Breaking operations: DROP, DELETE, TRUNCATE, ALTER TYPE changes
	breakingKeywords := []string{
		"DROP TABLE",
		"DROP COLUMN",
		"DROP INDEX",
		"DROP CONSTRAINT",
		"TRUNCATE",
		"DELETE FROM",
		"ALTER COLUMN",
		"ALTER TYPE", // PostgreSQL-specific type changes
	}

	for _, keyword := range breakingKeywords {
		if strings.Contains(sql, keyword) {
			return true
		}
	}

	// BLOCKER 1 FIX: Check for ADD COLUMN NOT NULL without DEFAULT
	// This will cause existing INSERT statements to fail
	// CRITICAL: Must check each ADD COLUMN clause independently to avoid false negatives
	// when multiple columns are added in a single ALTER TABLE statement
	if strings.Contains(sql, "ADD COLUMN") && strings.Contains(sql, "NOT NULL") {
		// Split by semicolon to check each SQL statement
		statements := strings.Split(sql, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			// Pattern to capture each ADD COLUMN clause and its definition
			// Matches: ADD COLUMN <name> <type> [constraints] until comma, semicolon, or end
			// Supports quoted column names like "first_name" or unquoted like first_name
			addColPattern := regexp.MustCompile(`ADD\s+COLUMN\s+(?:"[^"]+"|[a-zA-Z_][a-zA-Z0-9_]*)\s+([^,;]+?)(?:,|;|\s*$)`)
			matches := addColPattern.FindAllStringSubmatch(stmt, -1)

			for _, match := range matches {
				if len(match) >= 2 {
					columnDef := strings.TrimSpace(match[1])

					// Check if THIS specific column has NOT NULL
					if strings.Contains(columnDef, "NOT NULL") {
						// Check if DEFAULT appears in THIS column's definition
						// DEFAULT can appear before OR after NOT NULL in PostgreSQL
						// e.g., "VARCHAR(50) DEFAULT 'active' NOT NULL" is valid
						// e.g., "VARCHAR(50) NOT NULL DEFAULT 'active'" is also valid
						if !strings.Contains(columnDef, "DEFAULT") {
							// This column has NOT NULL but no DEFAULT - breaking change
							return true
						}
					}
				}
			}
		}
	}

	// Safe operations: CREATE, ADD (with nullable)
	// Note: We're conservative - only explicitly safe operations are non-breaking
	return false
}

// loadMigrations loads all migration files from the migrations directory
func (am *AutoMigrator) loadMigrations() ([]*migrate.Migration, error) {
	// Find migration files
	migrationFiles, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("failed to find migration files: %w", err)
	}

	// Filter to only .up.sql files (skip .down.sql)
	var upFiles []string
	for _, file := range migrationFiles {
		if strings.HasSuffix(file, ".up.sql") {
			upFiles = append(upFiles, file)
		} else if !strings.HasSuffix(file, ".down.sql") {
			// Include files without .up/.down suffix
			upFiles = append(upFiles, file)
		}
	}

	if len(upFiles) == 0 {
		return nil, nil
	}

	// Sort files
	sort.Strings(upFiles)

	// Parse each migration file
	var migrations []*migrate.Migration
	// HIGH 6 FIX: Track seen versions to detect duplicates
	seen := make(map[int64]string)

	for _, file := range upFiles {
		m, err := am.parseMigrationFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration %s: %w", file, err)
		}

		// HIGH 6 FIX: Check for duplicate version numbers
		if existing, exists := seen[m.Version]; exists {
			return nil, fmt.Errorf("duplicate migration version %d: %s and %s", m.Version, existing, file)
		}
		seen[m.Version] = file

		migrations = append(migrations, m)
	}

	return migrations, nil
}

// parseMigrationFile parses a migration file into a Migration struct
func (am *AutoMigrator) parseMigrationFile(filepath string) (*migrate.Migration, error) {
	// Extract version and name from filename
	filename := filepath[len("migrations/"):]

	// Parse version
	version, name, err := extractVersionFromFilename(filename)
	if err != nil {
		return nil, err
	}

	// Read SQL content
	upSQL, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	// Try to find corresponding down migration
	downPath := strings.Replace(filepath, ".up.sql", ".down.sql", 1)
	if !strings.HasSuffix(filepath, ".up.sql") {
		// If file doesn't have .up suffix, try adding .down
		downPath = strings.TrimSuffix(filepath, ".sql") + ".down.sql"
	}

	var downSQL []byte
	if _, err := os.Stat(downPath); err == nil {
		downSQL, _ = os.ReadFile(downPath)
	}

	// Check if migration is marked as breaking/data-loss based on filename
	breaking := isBreakingMigration(filename)
	dataLoss := isDataLossMigration(filename)

	return &migrate.Migration{
		Version:  version,
		Name:     name,
		Up:       string(upSQL),
		Down:     string(downSQL),
		Breaking: breaking,
		DataLoss: dataLoss,
	}, nil
}

// applyMigrationInTx applies a single migration within an existing transaction
// HIGH 4 FIX: This method now takes a transaction parameter instead of creating its own
func (am *AutoMigrator) applyMigrationInTx(tx *sql.Tx, runner *migrate.Runner, m *migrate.Migration) error {
	fmt.Printf("  Applying: %s...", m.Name)

	start := time.Now()

	// Execute migration SQL
	if _, err := tx.Exec(m.Up); err != nil {
		fmt.Println(" FAILED")
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	// Record migration using runner's tracker (BLOCKER 2 FIX: reuse existing tracker)
	if err := runner.RecordMigration(tx, m); err != nil {
		fmt.Println(" FAILED")
		return fmt.Errorf("failed to record migration: %w", err)
	}

	duration := time.Since(start)
	fmt.Printf(" ✓ (%v)\n", duration.Round(time.Millisecond))

	return nil
}

// printDryRun prints the SQL that would be executed without actually running it
func (am *AutoMigrator) printDryRun(pending []*migrate.Migration) error {
	fmt.Println("\n=== DRY RUN MODE ===")
	fmt.Println("The following SQL would be executed:")
	fmt.Print("\n")

	for i, m := range pending {
		fmt.Printf("-- Migration %d: %s (version %d)\n", i+1, m.Name, m.Version)
		fmt.Println(strings.Repeat("-", 70))
		fmt.Println(m.Up)
		fmt.Println()
	}

	fmt.Printf("Total migrations: %d\n", len(pending))
	fmt.Println("\nTo apply these migrations, run: conduit run --auto-migrate")
	return nil
}

// confirmAction prompts the user for confirmation
func (am *AutoMigrator) confirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [y/N]: ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
