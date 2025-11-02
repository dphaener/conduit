package build

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/conduit-lang/conduit/internal/orm/migrate"
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// MigrationBuilder generates versioned migrations based on schema changes
type MigrationBuilder struct {
	generator *migrate.Generator
}

// NewMigrationBuilder creates a new migration builder
func NewMigrationBuilder() *MigrationBuilder {
	return &MigrationBuilder{
		generator: migrate.NewGenerator(),
	}
}

// MigrationResult contains information about generated migrations
type MigrationResult struct {
	MigrationGenerated bool
	MigrationPath      string
	MigrationName      string
	Breaking           bool
	DataLoss           bool
	IsFirstBuild       bool
}

// GenerateInitialMigration generates the initial 001_init.sql migration
func (b *MigrationBuilder) GenerateInitialMigration(schemas map[string]*schema.ResourceSchema) (string, error) {
	// Use the generator to create the migration SQL
	// For initial migration, we pass empty old schemas
	emptySchemas := make(map[string]*schema.ResourceSchema)
	generatedMigration, err := b.generator.GenerateMigration(emptySchemas, schemas)
	if err != nil {
		return "", fmt.Errorf("failed to generate initial migration: %w", err)
	}

	if generatedMigration == nil {
		return "", fmt.Errorf("no migration generated for initial schema")
	}

	return generatedMigration.Up, nil
}

// GenerateVersionedMigration generates a new versioned migration based on schema changes
func (b *MigrationBuilder) GenerateVersionedMigration(
	oldSchemas, newSchemas map[string]*schema.ResourceSchema,
	migrationsDir string,
) (*MigrationResult, error) {
	result := &MigrationResult{
		MigrationGenerated: false,
	}

	// Generate migration using the differ
	migration, err := b.generator.GenerateMigration(oldSchemas, newSchemas)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migration: %w", err)
	}

	// If no changes detected, return early
	if migration == nil {
		return result, nil
	}

	result.MigrationGenerated = true
	result.Breaking = migration.Breaking
	result.DataLoss = migration.DataLoss

	// Generate migration filename with timestamp and sequence
	timestamp := time.Now().UnixMilli()
	seq, err := b.getNextSequence(migrationsDir, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to determine migration sequence: %w", err)
	}

	filename := fmt.Sprintf("%d_%03d_%s.sql", timestamp, seq, migration.Name)
	result.MigrationPath = filepath.Join(migrationsDir, filename)
	result.MigrationName = migration.Name

	// Write migration file
	if err := b.writeMigrationFile(result.MigrationPath, migration); err != nil {
		// Try to remove the file if it was partially written
		os.Remove(result.MigrationPath)
		return nil, fmt.Errorf("failed to write migration file: %w", err)
	}

	return result, nil
}

// getNextSequence determines the next sequence number for migrations with the same timestamp
func (b *MigrationBuilder) getNextSequence(migrationsDir string, timestamp int64) (int, error) {
	// List existing migrations
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil // No migrations directory yet
		}
		return 0, err
	}

	// Find migrations with the same timestamp prefix
	prefix := fmt.Sprintf("%d_", timestamp)
	maxSeq := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) < len(prefix) {
			continue
		}

		if name[:len(prefix)] == prefix {
			// Parse sequence number
			var seq int
			_, err := fmt.Sscanf(name[len(prefix):], "%d_", &seq)
			if err != nil {
				// Log warning but continue - don't fail the build
				fmt.Fprintf(os.Stderr, "Warning: malformed migration filename: %s\n", name)
				continue
			}
			if seq > maxSeq {
				maxSeq = seq
			}
		}
	}

	return maxSeq + 1, nil
}

// writeMigrationFile writes a migration to a SQL file
func (b *MigrationBuilder) writeMigrationFile(path string, migration *migrate.Migration) error {
	content := fmt.Sprintf(`-- Migration: %s
-- Generated at: %s
-- Version: %d
`, migration.Name, time.Now().Format(time.RFC3339), migration.Version)

	if migration.Breaking {
		content += "-- WARNING: This migration contains breaking changes\n"
	}
	if migration.DataLoss {
		content += "-- WARNING: This migration may cause data loss\n"
	}

	content += "\n-- Up Migration\n"
	content += migration.Up

	if migration.Down != "" {
		content += "\n-- Down Migration (Rollback)\n"
		content += "/*\n"
		content += migration.Down
		content += "*/\n"
	}

	// Ensure migrations directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	return nil
}

// ShouldGenerateInitialMigration determines if we should generate the initial migration
func (b *MigrationBuilder) ShouldGenerateInitialMigration(migrationsDir string) (bool, error) {
	// Check if 001_init.sql exists
	initPath := filepath.Join(migrationsDir, "001_init.sql")
	_, err := os.Stat(initPath)
	if err == nil {
		return false, nil // File exists, don't regenerate
	}
	if !os.IsNotExist(err) {
		return false, err // Some other error
	}

	// Check if any migration files exist
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // No migrations directory, this is first build
		}
		return false, err
	}

	// Count SQL files
	sqlFileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			sqlFileCount++
		}
	}

	// If no SQL files exist, this is the first build
	return sqlFileCount == 0, nil
}
