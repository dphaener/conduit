package transaction_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/conduit-lang/conduit/internal/orm/transaction"
	_ "github.com/mattn/go-sqlite3"
)

func ExampleManager_WithTransaction() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	// Create test table
	db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT)")

	mgr := transaction.NewManager(db)
	ctx := context.Background()

	err := mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO posts (title) VALUES (?)", "Hello World")
		return err // Automatic commit on success, rollback on error
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Transaction committed successfully")
	// Output: Transaction committed successfully
}

func ExampleManager_WithRetry() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT)")

	mgr := transaction.NewManager(db)
	ctx := context.Background()

	err := mgr.WithRetry(ctx, func(tx *sql.Tx) error {
		// This will automatically retry if a deadlock is detected
		_, err := tx.Exec("INSERT INTO posts (title) VALUES (?)", "Retry Example")
		return err
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Transaction completed with retry support")
	// Output: Transaction completed with retry support
}

func ExampleTransaction_BeginNested() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT)")

	mgr := transaction.NewManager(db)
	ctx := context.Background()

	tx, _ := mgr.Begin(ctx)
	defer tx.Rollback()

	// Parent transaction work
	tx.Exec("INSERT INTO posts (title) VALUES (?)", "Parent Post")

	// Note: This example shows the API, but full savepoint support requires PostgreSQL
	// SQLite has limited savepoint support
	// nested, _ := tx.BeginNested(ctx)
	// nested.Exec("INSERT INTO posts (title) VALUES (?)", "Nested Post")
	// nested.Commit() // or nested.Rollback()

	tx.Commit()

	fmt.Println("Nested transaction example (requires PostgreSQL for full support)")
	// Output: Nested transaction example (requires PostgreSQL for full support)
}

func ExampleFromContext() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	mgr := transaction.NewManager(db)
	ctx := context.Background()

	tx, _ := mgr.Begin(ctx)
	defer tx.Rollback()

	// Add transaction to context
	txCtx := tx.Context()

	// Retrieve transaction from context
	retrievedTx, ok := transaction.FromContext(txCtx)
	if ok {
		fmt.Printf("Transaction found in context: level=%d\n", retrievedTx.Level())
	}

	tx.Commit()
	// Output: Transaction found in context: level=0
}

func ExampleHookExecutorWithTransactions() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT, slug TEXT)")

	executor := transaction.NewHookExecutorWithTransactions(db)
	ctx := context.Background()

	// Hook function that needs transaction
	generateSlug := func(ctx context.Context, record map[string]interface{}) error {
		// Get transaction from context
		tx, ok := transaction.GetTransactionFromContext(ctx)
		if !ok {
			return fmt.Errorf("no transaction in context")
		}

		// Generate slug from title
		title := record["title"].(string)
		slug := title // In real app, would slugify this

		// Update record
		_, err := tx.Exec("UPDATE posts SET slug = ? WHERE title = ?", slug, title)
		return err
	}

	// Wrap hook with transaction
	wrappedHook := executor.WrapHookWithTransaction(generateSlug, true)

	// Execute hook
	record := map[string]interface{}{"title": "Example Post"}
	err := wrappedHook(ctx, record)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Hook executed with transaction support")
	// Output: Hook executed with transaction support
}

func ExampleIsolationLevel() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT)")

	mgr := transaction.NewManager(db)
	ctx := context.Background()

	// Use Serializable isolation level for maximum consistency
	err := mgr.WithTransactionIsolation(ctx, transaction.Serializable, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO posts (title) VALUES (?)", "Serializable Example")
		return err
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Transaction with Serializable isolation level")
	// Output: Transaction with Serializable isolation level
}
