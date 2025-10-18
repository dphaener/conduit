// Package migrate provides migration management for database schema evolution
package migrate

import (
	"database/sql"
	"fmt"
	"time"
)

// Migration represents a single database migration
type Migration struct {
	Version   int64     // Unix timestamp for ordering
	Name      string    // Human-readable name
	Up        string    // SQL to apply
	Down      string    // SQL to rollback
	Applied   bool      // Whether this migration has been applied
	AppliedAt time.Time // When the migration was applied
	Breaking  bool      // Requires manual review
	DataLoss  bool      // May cause data loss
}

// Tracker manages migration history in the database
type Tracker struct {
	db *sql.DB
}

// NewTracker creates a new migration tracker
func NewTracker(db *sql.DB) *Tracker {
	return &Tracker{db: db}
}

// Initialize ensures the schema_migrations table exists
func (t *Tracker) Initialize() error {
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
	_, err := t.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to initialize migrations table: %w", err)
	}

	return nil
}

// GetApplied returns all applied migrations sorted by version
func (t *Tracker) GetApplied() ([]*Migration, error) {
	query := `
SELECT version, name, applied_at, breaking, data_loss, up_sql, down_sql
FROM schema_migrations
ORDER BY version ASC
`
	rows, err := t.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []*Migration
	for rows.Next() {
		m := &Migration{Applied: true}
		var upSQL, downSQL sql.NullString
		if err := rows.Scan(&m.Version, &m.Name, &m.AppliedAt, &m.Breaking, &m.DataLoss, &upSQL, &downSQL); err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		if upSQL.Valid {
			m.Up = upSQL.String
		}
		if downSQL.Valid {
			m.Down = downSQL.String
		}
		migrations = append(migrations, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migrations: %w", err)
	}

	return migrations, nil
}

// GetLast returns the most recently applied migration, or nil if none exist
func (t *Tracker) GetLast() (*Migration, error) {
	query := `
SELECT version, name, applied_at, breaking, data_loss, up_sql, down_sql
FROM schema_migrations
ORDER BY version DESC
LIMIT 1
`
	m := &Migration{Applied: true}
	var upSQL, downSQL sql.NullString
	err := t.db.QueryRow(query).Scan(&m.Version, &m.Name, &m.AppliedAt, &m.Breaking, &m.DataLoss, &upSQL, &downSQL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last migration: %w", err)
	}

	if upSQL.Valid {
		m.Up = upSQL.String
	}
	if downSQL.Valid {
		m.Down = downSQL.String
	}

	return m, nil
}

// IsApplied checks if a migration version has been applied
func (t *Tracker) IsApplied(version int64) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)"
	var exists bool
	err := t.db.QueryRow(query, version).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check migration status: %w", err)
	}
	return exists, nil
}

// Record marks a migration as applied in a transaction
func (t *Tracker) Record(tx *sql.Tx, m *Migration) error {
	query := `
INSERT INTO schema_migrations (version, name, breaking, data_loss, up_sql, down_sql)
VALUES ($1, $2, $3, $4, $5, $6)
`
	_, err := tx.Exec(query, m.Version, m.Name, m.Breaking, m.DataLoss, m.Up, m.Down)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	return nil
}

// Remove removes a migration record in a transaction
func (t *Tracker) Remove(tx *sql.Tx, version int64) error {
	query := "DELETE FROM schema_migrations WHERE version = $1"
	result, err := tx.Exec(query, version)
	if err != nil {
		return fmt.Errorf("failed to remove migration: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("migration version %d not found", version)
	}

	return nil
}

// GetPending returns migrations that haven't been applied yet
func (t *Tracker) GetPending(all []*Migration) ([]*Migration, error) {
	applied, err := t.GetApplied()
	if err != nil {
		return nil, err
	}

	// Build set of applied versions
	appliedSet := make(map[int64]bool)
	for _, m := range applied {
		appliedSet[m.Version] = true
	}

	// Filter out applied migrations
	var pending []*Migration
	for _, m := range all {
		if !appliedSet[m.Version] {
			pending = append(pending, m)
		}
	}

	return pending, nil
}

// GetCount returns the total number of applied migrations
func (t *Tracker) GetCount() (int, error) {
	query := "SELECT COUNT(*) FROM schema_migrations"
	var count int
	err := t.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get migration count: %w", err)
	}
	return count, nil
}
