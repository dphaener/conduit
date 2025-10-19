package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
)

var (
	// ErrTransactionAborted is returned when a transaction is explicitly aborted
	ErrTransactionAborted = errors.New("transaction aborted")
	// ErrDeadlock is returned when a deadlock is detected
	ErrDeadlock = errors.New("deadlock detected")
	// ErrTransactionTimeout is returned when a transaction times out
	ErrTransactionTimeout = errors.New("transaction timeout")
	// ErrNestedTransactionNotSupported is returned when nested transactions are not supported
	ErrNestedTransactionNotSupported = errors.New("nested transactions require an existing transaction")
)

// savepointCounter provides guaranteed unique savepoint IDs across all transactions
var savepointCounter atomic.Uint64

// IsolationLevel represents the transaction isolation level
type IsolationLevel int

const (
	// ReadUncommitted allows dirty reads
	ReadUncommitted IsolationLevel = iota
	// ReadCommitted prevents dirty reads (PostgreSQL default)
	ReadCommitted
	// RepeatableRead prevents non-repeatable reads
	RepeatableRead
	// Serializable provides full isolation
	Serializable
)

// String returns the string representation of the isolation level
func (l IsolationLevel) String() string {
	switch l {
	case ReadUncommitted:
		return "READ UNCOMMITTED"
	case ReadCommitted:
		return "READ COMMITTED"
	case RepeatableRead:
		return "REPEATABLE READ"
	case Serializable:
		return "SERIALIZABLE"
	default:
		return "READ COMMITTED"
	}
}

// ToSQLOptions converts IsolationLevel to sql.TxOptions
func (l IsolationLevel) ToSQLOptions() *sql.TxOptions {
	var level sql.IsolationLevel
	switch l {
	case ReadUncommitted:
		level = sql.LevelReadUncommitted
	case ReadCommitted:
		level = sql.LevelReadCommitted
	case RepeatableRead:
		level = sql.LevelRepeatableRead
	case Serializable:
		level = sql.LevelSerializable
	default:
		level = sql.LevelReadCommitted
	}
	return &sql.TxOptions{Isolation: level}
}

// Transaction represents a database transaction with support for nesting
type Transaction struct {
	db             *sql.DB
	tx             *sql.Tx
	ctx            context.Context
	level          int  // Nesting level (0 = top-level, 1+ = savepoint)
	savepointName  string
	committed      atomic.Bool
	rolledBack     atomic.Bool
	isolationLevel IsolationLevel
	cancelFunc     context.CancelFunc // Optional cancel function for timeout/deadline
}

// Manager manages database transactions
type Manager struct {
	db *sql.DB
}

// NewManager creates a new transaction manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// Begin starts a new transaction with default isolation level
func (m *Manager) Begin(ctx context.Context) (*Transaction, error) {
	return m.BeginWithIsolation(ctx, ReadCommitted)
}

// BeginTx implements the TransactionManager interface from crud package
// This is required for compatibility with existing CRUD operations
func (m *Manager) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return m.db.BeginTx(ctx, nil)
}

// BeginWithIsolation starts a new transaction with the specified isolation level
func (m *Manager) BeginWithIsolation(ctx context.Context, level IsolationLevel) (*Transaction, error) {
	tx, err := m.db.BeginTx(ctx, level.ToSQLOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &Transaction{
		db:             m.db,
		tx:             tx,
		ctx:            ctx,
		level:          0,
		isolationLevel: level,
	}, nil
}

// WithTransaction executes a function within a transaction
// Automatically commits on success or rolls back on error
func (m *Manager) WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return m.WithTransactionIsolation(ctx, ReadCommitted, fn)
}

// WithTransactionIsolation executes a function within a transaction with specified isolation level
func (m *Manager) WithTransactionIsolation(ctx context.Context, level IsolationLevel, fn func(tx *sql.Tx) error) error {
	tx, err := m.BeginWithIsolation(ctx, level)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-throw panic after rollback
		}
	}()

	if err := fn(tx.tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %w, rollback failed: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// Context returns a context with the transaction embedded
func (t *Transaction) Context() context.Context {
	return context.WithValue(t.ctx, contextKeyTransaction, t)
}

// DB returns the underlying database connection
func (t *Transaction) DB() *sql.DB {
	return t.db
}

// Tx returns the underlying sql.Tx
func (t *Transaction) Tx() *sql.Tx {
	return t.tx
}

// Level returns the nesting level of the transaction
func (t *Transaction) Level() int {
	return t.level
}

// IsolationLevel returns the isolation level of the transaction
func (t *Transaction) IsolationLevel() IsolationLevel {
	return t.isolationLevel
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	// Call cancel function if it exists (for timeout/deadline transactions)
	if t.cancelFunc != nil {
		defer t.cancelFunc()
	}

	if t.committed.Load() {
		return errors.New("transaction already committed")
	}
	if t.rolledBack.Load() {
		return errors.New("transaction already rolled back")
	}

	// If this is a nested transaction (savepoint), release the savepoint
	if t.level > 0 {
		if _, err := t.tx.ExecContext(t.ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", t.savepointName)); err != nil {
			return fmt.Errorf("failed to release savepoint: %w", err)
		}
		t.committed.Store(true)
		return nil
	}

	// Top-level transaction - commit the entire transaction
	if err := t.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	t.committed.Store(true)
	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	// Call cancel function if it exists (for timeout/deadline transactions)
	if t.cancelFunc != nil {
		defer t.cancelFunc()
	}

	if t.committed.Load() {
		return errors.New("transaction already committed")
	}
	if t.rolledBack.Load() {
		return nil // Already rolled back, no-op
	}

	// If this is a nested transaction (savepoint), rollback to savepoint
	if t.level > 0 {
		if _, err := t.tx.ExecContext(t.ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", t.savepointName)); err != nil {
			return fmt.Errorf("failed to rollback to savepoint: %w", err)
		}
		t.rolledBack.Store(true)
		return nil
	}

	// Top-level transaction - rollback the entire transaction
	if err := t.tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	t.rolledBack.Store(true)
	return nil
}

// BeginNested creates a nested transaction using a PostgreSQL savepoint
func (t *Transaction) BeginNested(ctx context.Context) (*Transaction, error) {
	if t.tx == nil {
		return nil, ErrNestedTransactionNotSupported
	}

	// Generate unique savepoint name using atomic counter for guaranteed uniqueness
	savepointName := fmt.Sprintf("sp_%d_%d", savepointCounter.Add(1), t.level+1)

	// Create savepoint
	if _, err := t.tx.ExecContext(ctx, fmt.Sprintf("SAVEPOINT %s", savepointName)); err != nil {
		return nil, fmt.Errorf("failed to create savepoint: %w", err)
	}

	return &Transaction{
		db:             t.db,
		tx:             t.tx,
		ctx:            ctx,
		level:          t.level + 1,
		savepointName:  savepointName,
		isolationLevel: t.isolationLevel,
	}, nil
}

// Exec executes a query that doesn't return rows
func (t *Transaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(t.ctx, query, args...)
}

// Query executes a query that returns rows
func (t *Transaction) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(t.ctx, query, args...)
}

// QueryRow executes a query that returns at most one row
func (t *Transaction) QueryRow(query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(t.ctx, query, args...)
}

// IsCommitted returns true if the transaction has been committed
func (t *Transaction) IsCommitted() bool {
	return t.committed.Load()
}

// IsRolledBack returns true if the transaction has been rolled back
func (t *Transaction) IsRolledBack() bool {
	return t.rolledBack.Load()
}
