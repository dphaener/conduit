package hooks

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/orm/schema"
	_ "github.com/lib/pq"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use environment variables or defaults for test DB
	dsn := "postgres://postgres:postgres@localhost:5432/conduit_test?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
		return nil
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping integration test (database not available): %v", err)
		return nil
	}

	return db
}

// cleanupTestDB drops test tables
func cleanupTestDB(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()
	_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName))
	if err != nil {
		t.Logf("Failed to cleanup table %s: %v", tableName, err)
	}
}

func TestIntegration_HooksWithDatabase(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	tableName := "test_posts"
	cleanupTestDB(t, db, tableName)
	defer cleanupTestDB(t, db, tableName)

	// Create test table
	_, err := db.Exec(`
		CREATE TABLE test_posts (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			slug TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Create resource schema
	resource := schema.NewResourceSchema("TestPost")
	resource.TableName = tableName
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeInt, Nullable: false},
	}
	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: false},
	}
	resource.Fields["slug"] = &schema.Field{
		Name: "slug",
		Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: true},
	}

	// Setup hooks
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)

	// Before create hook: auto-generate slug
	beforeCreateHook := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			if title, ok := record["title"].(string); ok {
				// Simple slugify
				slug := title
				record["slug"] = slug
			}
			return nil
		},
	}
	executor.Register(schema.BeforeCreate, beforeCreateHook)

	// Test hook execution during insert
	ctx := context.Background()
	record := map[string]interface{}{
		"title": "Test Post Title",
	}

	// Execute before create hooks
	err = executor.ExecuteHooks(ctx, resource, schema.BeforeCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	// Verify slug was set
	if record["slug"] != "Test Post Title" {
		t.Errorf("Before create hook did not set slug: %v", record["slug"])
	}

	// Insert record
	var id int
	err = db.QueryRow(
		"INSERT INTO test_posts (title, slug) VALUES ($1, $2) RETURNING id",
		record["title"], record["slug"],
	).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	// Verify record was inserted
	var title, slug string
	err = db.QueryRow(
		"SELECT title, slug FROM test_posts WHERE id = $1",
		id,
	).Scan(&title, &slug)
	if err != nil {
		t.Fatalf("Failed to query record: %v", err)
	}

	if title != "Test Post Title" {
		t.Errorf("Expected title 'Test Post Title', got '%s'", title)
	}

	if slug != "Test Post Title" {
		t.Errorf("Expected slug 'Test Post Title', got '%s'", slug)
	}
}

func TestIntegration_HooksWithTransaction(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	tableName := "test_transactions"
	cleanupTestDB(t, db, tableName)
	defer cleanupTestDB(t, db, tableName)

	// Create test table
	_, err := db.Exec(`
		CREATE TABLE test_transactions (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			counter INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Create resource schema
	resource := schema.NewResourceSchema("TestTransaction")
	resource.TableName = tableName

	// Setup hooks
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)

	// Hook that increments counter
	hook := &Hook{
		Type:        schema.BeforeCreate,
		Transaction: true,
		Async:       false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			if ctx.HasTransaction() {
				record["counter"] = 42
			}
			return nil
		},
	}
	executor.Register(schema.BeforeCreate, hook)

	// Test with transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Create context with transaction
	ctx := context.WithValue(context.Background(), ContextKeyTransaction, tx)
	ctx = context.WithValue(ctx, ContextKeyDB, db)

	record := map[string]interface{}{
		"title": "Test",
	}

	err = executor.ExecuteHooks(ctx, resource, schema.BeforeCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	// Verify counter was set by transactional hook
	if record["counter"] != 42 {
		t.Errorf("Expected counter 42, got %v", record["counter"])
	}

	// Insert within transaction
	_, err = tx.Exec(
		"INSERT INTO test_transactions (title, counter) VALUES ($1, $2)",
		record["title"], record["counter"],
	)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify record exists
	var counter int
	err = db.QueryRow("SELECT counter FROM test_transactions WHERE title = $1", "Test").Scan(&counter)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if counter != 42 {
		t.Errorf("Expected counter 42 in DB, got %d", counter)
	}
}

func TestIntegration_AsyncHooksWithDatabase(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	tableName := "test_async"
	auditTableName := "test_audit_log"
	cleanupTestDB(t, db, tableName)
	cleanupTestDB(t, db, auditTableName)
	defer cleanupTestDB(t, db, tableName)
	defer cleanupTestDB(t, db, auditTableName)

	// Create test tables
	_, err := db.Exec(`
		CREATE TABLE test_async (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE test_audit_log (
			id SERIAL PRIMARY KEY,
			action TEXT NOT NULL,
			record_id INTEGER,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit table: %v", err)
	}

	// Create resource schema
	resource := schema.NewResourceSchema("TestAsync")
	resource.TableName = tableName

	// Setup hooks with async audit logging
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)

	// Async after create hook: log to audit table
	asyncHook := &Hook{
		Type:  schema.AfterCreate,
		Async: true,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			// This runs asynchronously after the main operation
			id, ok := record["id"].(int)
			if !ok {
				return fmt.Errorf("invalid id type")
			}

			_, err := ctx.DB().Exec(
				"INSERT INTO test_audit_log (action, record_id) VALUES ($1, $2)",
				"create", id,
			)
			return err
		},
	}
	executor.Register(schema.AfterCreate, asyncHook)

	// Insert record
	var id int
	err = db.QueryRow(
		"INSERT INTO test_async (title) VALUES ($1) RETURNING id",
		"Test Title",
	).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Create context and execute after create hooks
	ctx := context.WithValue(context.Background(), ContextKeyDB, db)
	record := map[string]interface{}{
		"id":    id,
		"title": "Test Title",
	}

	err = executor.ExecuteHooks(ctx, resource, schema.AfterCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	// Wait for async hook to execute
	time.Sleep(500 * time.Millisecond)

	// Verify audit log entry was created
	var auditID int
	var action string
	var recordID int
	err = db.QueryRow(
		"SELECT id, action, record_id FROM test_audit_log WHERE record_id = $1",
		id,
	).Scan(&auditID, &action, &recordID)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}

	if action != "create" {
		t.Errorf("Expected action 'create', got '%s'", action)
	}

	if recordID != id {
		t.Errorf("Expected record_id %d, got %d", id, recordID)
	}
}

func TestIntegration_HookErrorRollback(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	tableName := "test_rollback"
	cleanupTestDB(t, db, tableName)
	defer cleanupTestDB(t, db, tableName)

	// Create test table
	_, err := db.Exec(`
		CREATE TABLE test_rollback (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Create resource schema
	resource := schema.NewResourceSchema("TestRollback")
	resource.TableName = tableName

	// Setup hooks
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)

	// Hook that fails
	failingHook := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return fmt.Errorf("validation failed")
		},
	}
	executor.Register(schema.BeforeCreate, failingHook)

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Create context
	ctx := context.WithValue(context.Background(), ContextKeyTransaction, tx)
	record := map[string]interface{}{
		"title": "Test",
	}

	// Execute hooks - should fail
	err = executor.ExecuteHooks(ctx, resource, schema.BeforeCreate, record)
	if err == nil {
		t.Fatal("Expected hook to fail")
	}

	// Rollback transaction
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify no records were inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_rollback").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query count: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 records after rollback, got %d", count)
	}
}
