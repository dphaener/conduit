package migrate

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Runner executes migrations with transaction support
type Runner struct {
	db      *sql.DB
	tracker *Tracker
}

// NewRunner creates a new migration runner
func NewRunner(db *sql.DB) *Runner {
	return &Runner{
		db:      db,
		tracker: NewTracker(db),
	}
}

// Initialize sets up the migration tracking table
func (r *Runner) Initialize() error {
	return r.tracker.Initialize()
}

// RecordMigration records a migration in a transaction
// This method exposes the tracker's Record method for external callers
func (r *Runner) RecordMigration(tx *sql.Tx, m *Migration) error {
	return r.tracker.Record(tx, m)
}

// MigrateUp applies all pending migrations
func (r *Runner) MigrateUp(migrations []*Migration) error {
	pending, err := r.tracker.GetPending(migrations)
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(pending) == 0 {
		log.Println("No pending migrations")
		return nil
	}

	log.Printf("Found %d pending migration(s)", len(pending))

	for _, migration := range pending {
		if err := r.applyMigration(migration); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration.Name, err)
		}
		log.Printf("✓ Applied migration: %s", migration.Name)
	}

	log.Printf("✓ Successfully applied %d migration(s)", len(pending))
	return nil
}

// MigrateDown rolls back the last migration
func (r *Runner) MigrateDown() error {
	last, err := r.tracker.GetLast()
	if err != nil {
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	if last == nil {
		return fmt.Errorf("no migrations to rollback")
	}

	if last.Down == "" {
		return fmt.Errorf("migration %s has no down migration", last.Name)
	}

	log.Printf("Rolling back migration: %s", last.Name)

	if err := r.rollbackMigration(last); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	log.Printf("✓ Rolled back migration: %s", last.Name)
	return nil
}

// MigrateDownTo rolls back migrations down to a specific version
func (r *Runner) MigrateDownTo(targetVersion int64) error {
	applied, err := r.tracker.GetApplied()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Find migrations to rollback (in reverse order)
	var toRollback []*Migration
	for i := len(applied) - 1; i >= 0; i-- {
		if applied[i].Version > targetVersion {
			toRollback = append(toRollback, applied[i])
		}
	}

	if len(toRollback) == 0 {
		log.Println("No migrations to rollback")
		return nil
	}

	log.Printf("Rolling back %d migration(s)", len(toRollback))

	for _, migration := range toRollback {
		if migration.Down == "" {
			return fmt.Errorf("migration %s has no down migration", migration.Name)
		}

		if err := r.rollbackMigration(migration); err != nil {
			return fmt.Errorf("rollback of %s failed: %w", migration.Name, err)
		}

		log.Printf("✓ Rolled back migration: %s", migration.Name)
	}

	log.Printf("✓ Successfully rolled back %d migration(s)", len(toRollback))
	return nil
}

// applyMigration applies a single migration in a transaction
func (r *Runner) applyMigration(migration *Migration) error {
	start := time.Now()

	// Validate migration
	if migration.Up == "" {
		return fmt.Errorf("migration has no up SQL")
	}

	// Check for breaking changes
	if migration.Breaking {
		log.Printf("⚠ Warning: Migration %s contains breaking changes", migration.Name)
	}

	if migration.DataLoss {
		log.Printf("⚠ Warning: Migration %s may cause data loss", migration.Name)
	}

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	// Execute migration SQL
	if _, err := tx.Exec(migration.Up); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	if err := r.tracker.Record(tx, migration); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(start)
	log.Printf("  Migration took %v", duration)

	return nil
}

// rollbackMigration rolls back a single migration in a transaction
func (r *Runner) rollbackMigration(migration *Migration) error {
	start := time.Now()

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	// Execute down migration SQL
	if _, err := tx.Exec(migration.Down); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Remove migration record
	if err := r.tracker.Remove(tx, migration.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(start)
	log.Printf("  Rollback took %v", duration)

	return nil
}

// Status returns the current migration status
func (r *Runner) Status(allMigrations []*Migration) (*MigrationStatus, error) {
	applied, err := r.tracker.GetApplied()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	pending, err := r.tracker.GetPending(allMigrations)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending migrations: %w", err)
	}

	var lastApplied *Migration
	if len(applied) > 0 {
		lastApplied = applied[len(applied)-1]
	}

	return &MigrationStatus{
		Total:       len(allMigrations),
		Applied:     applied,
		Pending:     pending,
		LastApplied: lastApplied,
	}, nil
}

// Validate checks if a migration can be safely applied
func (r *Runner) Validate(migration *Migration) error {
	// Check for empty SQL
	if migration.Up == "" {
		return fmt.Errorf("migration has no up SQL")
	}

	// Validate SQL syntax using EXPLAIN (without executing)
	// This is a basic check - it won't catch all errors
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	// Try to prepare the statement (validates syntax)
	// Note: This doesn't execute, just checks syntax
	stmt, err := tx.Prepare(migration.Up)
	if err != nil {
		return fmt.Errorf("invalid SQL syntax: %w", err)
	}
	stmt.Close()

	return nil
}

// MigrationStatus represents the current state of migrations
type MigrationStatus struct {
	Total       int
	Applied     []*Migration
	Pending     []*Migration
	LastApplied *Migration
}

// Summary returns a human-readable summary
func (s *MigrationStatus) Summary() string {
	return fmt.Sprintf("Total: %d migrations (%d applied, %d pending)",
		s.Total,
		len(s.Applied),
		len(s.Pending))
}
