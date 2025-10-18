package migrate

import (
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestRunner_Initialize(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	err := runner.Initialize()
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Verify table was created
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'schema_migrations')").Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}

	if !exists {
		t.Error("schema_migrations table was not created")
	}
}

func TestRunner_MigrateUp(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Define test migrations
	migrations := []*Migration{
		{
			Version: 1000,
			Name:    "create_test_table",
			Up:      "CREATE TABLE test_table (id SERIAL PRIMARY KEY, name VARCHAR(255));",
			Down:    "DROP TABLE IF EXISTS test_table;",
		},
		{
			Version: 2000,
			Name:    "add_email_column",
			Up:      "ALTER TABLE test_table ADD COLUMN email VARCHAR(255);",
			Down:    "ALTER TABLE test_table DROP COLUMN IF EXISTS email;",
		},
	}

	// Apply migrations
	err := runner.MigrateUp(migrations)
	if err != nil {
		t.Fatalf("MigrateUp() failed: %v", err)
	}

	// Verify migrations were applied
	count, err := runner.tracker.GetCount()
	if err != nil {
		t.Fatalf("GetCount() failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 migrations applied, got %d", count)
	}

	// Verify table and column were created
	var tableExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'test_table')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}

	if !tableExists {
		t.Error("test_table was not created")
	}

	var columnExists bool
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'test_table' AND column_name = 'email'
		)
	`).Scan(&columnExists)
	if err != nil {
		t.Fatalf("Failed to check column existence: %v", err)
	}

	if !columnExists {
		t.Error("email column was not created")
	}

	// Cleanup
	db.Exec("DROP TABLE IF EXISTS test_table CASCADE")
}

func TestRunner_MigrateUp_NoPendingMigrations(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Apply migrations
	migrations := []*Migration{
		{
			Version: 1000,
			Name:    "test_migration",
			Up:      "CREATE TABLE test_table (id SERIAL PRIMARY KEY);",
			Down:    "DROP TABLE IF EXISTS test_table;",
		},
	}

	err := runner.MigrateUp(migrations)
	if err != nil {
		t.Fatalf("First MigrateUp() failed: %v", err)
	}

	// Apply same migrations again (should be no-op)
	err = runner.MigrateUp(migrations)
	if err != nil {
		t.Fatalf("Second MigrateUp() failed: %v", err)
	}

	// Verify only one migration was recorded
	count, err := runner.tracker.GetCount()
	if err != nil {
		t.Fatalf("GetCount() failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 migration applied, got %d", count)
	}

	// Cleanup
	db.Exec("DROP TABLE IF EXISTS test_table CASCADE")
}

func TestRunner_MigrateUp_TransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Define migration with invalid SQL
	migrations := []*Migration{
		{
			Version: 1000,
			Name:    "invalid_migration",
			Up:      "CREATE TABLE test_table (id SERIAL PRIMARY KEY);",
			Down:    "DROP TABLE IF EXISTS test_table;",
		},
		{
			Version: 2000,
			Name:    "failing_migration",
			Up:      "INVALID SQL THAT WILL FAIL;",
			Down:    "-- Nothing",
		},
	}

	// Apply migrations - should fail on second migration
	err := runner.MigrateUp(migrations)
	if err == nil {
		t.Fatal("Expected MigrateUp() to fail with invalid SQL")
	}

	// Verify first migration was applied
	count, err := runner.tracker.GetCount()
	if err != nil {
		t.Fatalf("GetCount() failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 migration applied, got %d", count)
	}

	// Verify second migration was NOT applied (transaction rolled back)
	applied, err := runner.tracker.IsApplied(2000)
	if err != nil {
		t.Fatalf("IsApplied() failed: %v", err)
	}

	if applied {
		t.Error("Failed migration was incorrectly recorded as applied")
	}

	// Cleanup
	db.Exec("DROP TABLE IF EXISTS test_table CASCADE")
}

func TestRunner_MigrateDown(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Apply migrations
	migrations := []*Migration{
		{
			Version: 1000,
			Name:    "create_test_table",
			Up:      "CREATE TABLE test_table (id SERIAL PRIMARY KEY);",
			Down:    "DROP TABLE IF EXISTS test_table;",
		},
	}

	err := runner.MigrateUp(migrations)
	if err != nil {
		t.Fatalf("MigrateUp() failed: %v", err)
	}

	// Verify table was created
	var tableExists bool
	db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'test_table')").Scan(&tableExists)
	if !tableExists {
		t.Fatal("test_table was not created")
	}

	// Rollback last migration
	err = runner.MigrateDown()
	if err != nil {
		t.Fatalf("MigrateDown() failed: %v", err)
	}

	// Verify migration was removed from tracker
	count, err := runner.tracker.GetCount()
	if err != nil {
		t.Fatalf("GetCount() failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 migrations applied after rollback, got %d", count)
	}

	// Verify table was dropped
	db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'test_table')").Scan(&tableExists)
	if tableExists {
		t.Error("test_table was not dropped")
	}
}

func TestRunner_MigrateDown_NoMigrations(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Try to rollback with no migrations
	err := runner.MigrateDown()
	if err == nil {
		t.Error("Expected error when rolling back with no migrations")
	}
}

func TestRunner_MigrateDown_NoDownSQL(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Apply migration without down SQL
	migration := &Migration{
		Version: 1000,
		Name:    "no_down_migration",
		Up:      "CREATE TABLE test_table (id SERIAL PRIMARY KEY);",
		Down:    "", // No down SQL
	}

	err := runner.MigrateUp([]*Migration{migration})
	if err != nil {
		t.Fatalf("MigrateUp() failed: %v", err)
	}

	// Try to rollback
	err = runner.MigrateDown()
	if err == nil {
		t.Error("Expected error when rolling back migration without down SQL")
	}

	// Cleanup
	db.Exec("DROP TABLE IF EXISTS test_table CASCADE")
}

func TestRunner_MigrateDownTo(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Apply multiple migrations
	migrations := []*Migration{
		{
			Version: 1000,
			Name:    "first_migration",
			Up:      "CREATE TABLE table1 (id SERIAL PRIMARY KEY);",
			Down:    "DROP TABLE IF EXISTS table1;",
		},
		{
			Version: 2000,
			Name:    "second_migration",
			Up:      "CREATE TABLE table2 (id SERIAL PRIMARY KEY);",
			Down:    "DROP TABLE IF EXISTS table2;",
		},
		{
			Version: 3000,
			Name:    "third_migration",
			Up:      "CREATE TABLE table3 (id SERIAL PRIMARY KEY);",
			Down:    "DROP TABLE IF EXISTS table3;",
		},
	}

	err := runner.MigrateUp(migrations)
	if err != nil {
		t.Fatalf("MigrateUp() failed: %v", err)
	}

	// Verify all migrations were applied
	count, err := runner.tracker.GetCount()
	if err != nil {
		t.Fatalf("GetCount() failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 migrations applied, got %d", count)
	}

	// Rollback to version 1000
	err = runner.MigrateDownTo(1000)
	if err != nil {
		t.Fatalf("MigrateDownTo() failed: %v", err)
	}

	// Verify only first migration remains
	count, err = runner.tracker.GetCount()
	if err != nil {
		t.Fatalf("GetCount() failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 migration applied after rollback, got %d", count)
	}

	// Verify first migration is still applied
	applied, err := runner.tracker.IsApplied(1000)
	if err != nil {
		t.Fatalf("IsApplied() failed: %v", err)
	}

	if !applied {
		t.Error("First migration should still be applied")
	}

	// Cleanup
	db.Exec("DROP TABLE IF EXISTS table1 CASCADE")
	db.Exec("DROP TABLE IF EXISTS table2 CASCADE")
	db.Exec("DROP TABLE IF EXISTS table3 CASCADE")
}

func TestRunner_Status(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	allMigrations := []*Migration{
		{
			Version: 1000,
			Name:    "first_migration",
			Up:      "CREATE TABLE table1 (id SERIAL PRIMARY KEY);",
			Down:    "DROP TABLE IF EXISTS table1;",
		},
		{
			Version: 2000,
			Name:    "second_migration",
			Up:      "CREATE TABLE table2 (id SERIAL PRIMARY KEY);",
			Down:    "DROP TABLE IF EXISTS table2;",
		},
	}

	// Apply first migration only
	err := runner.MigrateUp(allMigrations[:1])
	if err != nil {
		t.Fatalf("MigrateUp() failed: %v", err)
	}

	// Get status
	status, err := runner.Status(allMigrations)
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	if status.Total != 2 {
		t.Errorf("Expected total 2 migrations, got %d", status.Total)
	}

	if len(status.Applied) != 1 {
		t.Errorf("Expected 1 applied migration, got %d", len(status.Applied))
	}

	if len(status.Pending) != 1 {
		t.Errorf("Expected 1 pending migration, got %d", len(status.Pending))
	}

	if status.LastApplied == nil {
		t.Fatal("Expected LastApplied to be set")
	}

	if status.LastApplied.Version != 1000 {
		t.Errorf("Expected last applied version 1000, got %d", status.LastApplied.Version)
	}

	// Cleanup
	db.Exec("DROP TABLE IF EXISTS table1 CASCADE")
}

func TestRunner_Validate(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Test valid migration
	validMigration := &Migration{
		Version: 1000,
		Name:    "valid_migration",
		Up:      "CREATE TABLE test_table (id SERIAL PRIMARY KEY);",
	}

	err := runner.Validate(validMigration)
	if err != nil {
		t.Errorf("Validate() failed for valid migration: %v", err)
	}

	// Test migration with empty SQL
	emptyMigration := &Migration{
		Version: 2000,
		Name:    "empty_migration",
		Up:      "",
	}

	err = runner.Validate(emptyMigration)
	if err == nil {
		t.Error("Expected error for migration with empty SQL")
	}
}

func TestRunner_BreakingAndDataLossWarnings(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Test migration with breaking changes
	migration := &Migration{
		Version:  time.Now().UnixMilli(),
		Name:     "breaking_migration",
		Up:       "CREATE TABLE test_table (id SERIAL PRIMARY KEY);",
		Down:     "DROP TABLE IF EXISTS test_table;",
		Breaking: true,
		DataLoss: true,
	}

	err := runner.MigrateUp([]*Migration{migration})
	if err != nil {
		t.Fatalf("MigrateUp() failed: %v", err)
	}

	// Verify migration was recorded with flags
	last, err := runner.tracker.GetLast()
	if err != nil {
		t.Fatalf("GetLast() failed: %v", err)
	}

	if last == nil {
		t.Fatal("Expected last migration to be set")
	}

	if !last.Breaking {
		t.Error("Expected breaking flag to be true")
	}

	if !last.DataLoss {
		t.Error("Expected data_loss flag to be true")
	}

	// Cleanup
	db.Exec("DROP TABLE IF EXISTS test_table CASCADE")
}

func TestRunner_MigrationStoredSQL(t *testing.T) {
	db := setupTestDB(t)
	runner := NewRunner(db)

	if err := runner.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	upSQL := "CREATE TABLE test_table (id SERIAL PRIMARY KEY, name VARCHAR(255));"
	downSQL := "DROP TABLE IF EXISTS test_table;"

	migration := &Migration{
		Version: time.Now().UnixMilli(),
		Name:    "test_stored_sql",
		Up:      upSQL,
		Down:    downSQL,
	}

	err := runner.MigrateUp([]*Migration{migration})
	if err != nil {
		t.Fatalf("MigrateUp() failed: %v", err)
	}

	// Retrieve migration and verify SQL was stored
	last, err := runner.tracker.GetLast()
	if err != nil {
		t.Fatalf("GetLast() failed: %v", err)
	}

	if last == nil {
		t.Fatal("Expected last migration to be set")
	}

	if last.Up != upSQL {
		t.Errorf("Up SQL not stored correctly.\nExpected: %s\nGot: %s", upSQL, last.Up)
	}

	if last.Down != downSQL {
		t.Errorf("Down SQL not stored correctly.\nExpected: %s\nGot: %s", downSQL, last.Down)
	}

	// Cleanup
	db.Exec("DROP TABLE IF EXISTS test_table CASCADE")
}
