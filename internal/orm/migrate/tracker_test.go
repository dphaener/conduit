package migrate

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use PostgreSQL test database
	// For unit tests, you'd typically use a test container or mock
	// For this example, we'll skip if DATABASE_URL is not set
	db, err := sql.Open("pgx", "postgresql://localhost:5432/conduit_test?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}

	if err := db.Ping(); err != nil {
		t.Skip("Test database not reachable:", err)
	}

	// Clean up any existing migrations table
	db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")

	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")
		db.Close()
	})

	return db
}

func TestTracker_Initialize(t *testing.T) {
	db := setupTestDB(t)
	tracker := NewTracker(db)

	err := tracker.Initialize()
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

	// Test idempotency - should not error if called again
	err = tracker.Initialize()
	if err != nil {
		t.Errorf("Initialize() should be idempotent, got error: %v", err)
	}
}

func TestTracker_Record(t *testing.T) {
	db := setupTestDB(t)
	tracker := NewTracker(db)

	if err := tracker.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	migration := &Migration{
		Version:  time.Now().Unix(),
		Name:     "test_migration",
		Breaking: false,
		DataLoss: false,
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() failed: %v", err)
	}
	defer tx.Rollback()

	err = tracker.Record(tx, migration)
	if err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() failed: %v", err)
	}

	// Verify migration was recorded
	applied, err := tracker.IsApplied(migration.Version)
	if err != nil {
		t.Fatalf("IsApplied() failed: %v", err)
	}

	if !applied {
		t.Error("Migration was not recorded")
	}
}

func TestTracker_GetApplied(t *testing.T) {
	db := setupTestDB(t)
	tracker := NewTracker(db)

	if err := tracker.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Record some migrations
	migrations := []*Migration{
		{Version: 1000, Name: "first_migration"},
		{Version: 2000, Name: "second_migration"},
		{Version: 3000, Name: "third_migration"},
	}

	for _, m := range migrations {
		tx, _ := db.Begin()
		tracker.Record(tx, m)
		tx.Commit()
	}

	// Get applied migrations
	applied, err := tracker.GetApplied()
	if err != nil {
		t.Fatalf("GetApplied() failed: %v", err)
	}

	if len(applied) != 3 {
		t.Errorf("Expected 3 applied migrations, got %d", len(applied))
	}

	// Verify order (should be ascending by version)
	for i, m := range applied {
		if m.Version != migrations[i].Version {
			t.Errorf("Migration at index %d has wrong version: got %d, want %d",
				i, m.Version, migrations[i].Version)
		}
	}
}

func TestTracker_GetLast(t *testing.T) {
	db := setupTestDB(t)
	tracker := NewTracker(db)

	if err := tracker.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Test with no migrations
	last, err := tracker.GetLast()
	if err != nil {
		t.Fatalf("GetLast() failed: %v", err)
	}

	if last != nil {
		t.Error("Expected nil for empty migration list")
	}

	// Record some migrations
	migrations := []*Migration{
		{Version: 1000, Name: "first_migration"},
		{Version: 3000, Name: "third_migration"},
		{Version: 2000, Name: "second_migration"},
	}

	for _, m := range migrations {
		tx, _ := db.Begin()
		tracker.Record(tx, m)
		tx.Commit()
	}

	// Get last migration
	last, err = tracker.GetLast()
	if err != nil {
		t.Fatalf("GetLast() failed: %v", err)
	}

	if last == nil {
		t.Fatal("Expected last migration, got nil")
	}

	// Should be the one with highest version
	if last.Version != 3000 {
		t.Errorf("Expected version 3000, got %d", last.Version)
	}

	if last.Name != "third_migration" {
		t.Errorf("Expected name 'third_migration', got %s", last.Name)
	}
}

func TestTracker_Remove(t *testing.T) {
	db := setupTestDB(t)
	tracker := NewTracker(db)

	if err := tracker.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	migration := &Migration{
		Version: time.Now().Unix(),
		Name:    "test_migration",
	}

	// Record migration
	tx, _ := db.Begin()
	tracker.Record(tx, migration)
	tx.Commit()

	// Verify it's applied
	applied, _ := tracker.IsApplied(migration.Version)
	if !applied {
		t.Fatal("Migration was not recorded")
	}

	// Remove migration
	tx, _ = db.Begin()
	err := tracker.Remove(tx, migration.Version)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}
	tx.Commit()

	// Verify it's no longer applied
	applied, _ = tracker.IsApplied(migration.Version)
	if applied {
		t.Error("Migration was not removed")
	}
}

func TestTracker_GetPending(t *testing.T) {
	db := setupTestDB(t)
	tracker := NewTracker(db)

	if err := tracker.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Define all migrations
	allMigrations := []*Migration{
		{Version: 1000, Name: "first_migration"},
		{Version: 2000, Name: "second_migration"},
		{Version: 3000, Name: "third_migration"},
		{Version: 4000, Name: "fourth_migration"},
	}

	// Apply first two migrations
	for i := 0; i < 2; i++ {
		tx, _ := db.Begin()
		tracker.Record(tx, allMigrations[i])
		tx.Commit()
	}

	// Get pending migrations
	pending, err := tracker.GetPending(allMigrations)
	if err != nil {
		t.Fatalf("GetPending() failed: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending migrations, got %d", len(pending))
	}

	// Verify correct migrations are pending
	expectedVersions := []int64{3000, 4000}
	for i, m := range pending {
		if m.Version != expectedVersions[i] {
			t.Errorf("Pending migration %d has wrong version: got %d, want %d",
				i, m.Version, expectedVersions[i])
		}
	}
}

func TestTracker_GetCount(t *testing.T) {
	db := setupTestDB(t)
	tracker := NewTracker(db)

	if err := tracker.Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Initial count should be 0
	count, err := tracker.GetCount()
	if err != nil {
		t.Fatalf("GetCount() failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Record some migrations
	migrations := []*Migration{
		{Version: 1000, Name: "first_migration"},
		{Version: 2000, Name: "second_migration"},
	}

	for _, m := range migrations {
		tx, _ := db.Begin()
		tracker.Record(tx, m)
		tx.Commit()
	}

	// Count should be 2
	count, err = tracker.GetCount()
	if err != nil {
		t.Fatalf("GetCount() failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}
