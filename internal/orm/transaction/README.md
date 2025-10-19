# Transaction Management

This package provides robust ACID transaction support for the Conduit ORM with nested transactions, deadlock handling, and timeout support.

## Features

- **Explicit Transaction Control:** `DB.Transaction()` for explicit transaction blocks
- **Automatic Transaction Wrapping:** Integration with `@transaction` hooks
- **Nested Transactions:** PostgreSQL savepoints for nested transaction support
- **Configurable Isolation Levels:** ReadUncommitted, ReadCommitted, RepeatableRead, Serializable
- **Automatic Rollback:** Rollback on error or panic
- **Manual Rollback:** `tx.Rollback()` method for explicit control
- **Context Propagation:** Transaction stored in context for easy access
- **Deadlock Detection:** Automatic retry with exponential backoff
- **Transaction Timeouts:** Built-in timeout support with context cancellation

## Usage

### Basic Transaction

```go
mgr := transaction.NewManager(db)

err := mgr.WithTransaction(ctx, func(tx *sql.Tx) error {
    _, err := tx.Exec("INSERT INTO posts (title) VALUES (?)", "New Post")
    return err // Automatic commit on nil, rollback on error
})
```

### With Isolation Level

```go
err := mgr.WithTransactionIsolation(ctx, transaction.Serializable, func(tx *sql.Tx) error {
    // Your transaction code here
    return nil
})
```

### Nested Transactions (Savepoints)

```go
tx, err := mgr.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback()

// Do some work
_, err = tx.Exec("INSERT INTO posts (title) VALUES (?)", "Parent")
if err != nil {
    return err
}

// Create nested transaction
nested, err := tx.BeginNested(ctx)
if err != nil {
    return err
}

// Do nested work
_, err = nested.Exec("INSERT INTO comments (body) VALUES (?)", "Child")
if err != nil {
    nested.Rollback() // Rollback to savepoint
} else {
    nested.Commit() // Release savepoint
}

// Commit parent transaction
return tx.Commit()
```

### Deadlock Retry

```go
config := &transaction.RetryConfig{
    MaxRetries:  3,
    BaseBackoff: 100 * time.Millisecond,
}

err := mgr.WithRetryConfig(ctx, config, func(tx *sql.Tx) error {
    // Your transaction code here
    // Automatically retries on deadlock errors (40P01)
    return nil
})
```

### Transaction Timeout

```go
// Timeout after 5 seconds
err := mgr.WithTimeout(ctx, 5*time.Second, func(tx *sql.Tx) error {
    // Your transaction code here
    return nil
})

// Or begin with timeout
tx, err := mgr.BeginWithTimeout(ctx, 5*time.Second)
if err != nil {
    return err
}
defer tx.Rollback()

// Transaction must complete within 5 seconds
_, err = tx.Exec("INSERT INTO posts (title) VALUES (?)", "Timeout Test")
if err != nil {
    return err
}

return tx.Commit()
```

### Context Propagation

```go
tx, err := mgr.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback()

// Get context with transaction
txCtx := tx.Context()

// Pass to other functions
processWithTransaction(txCtx)

func processWithTransaction(ctx context.Context) error {
    // Retrieve transaction from context
    tx, ok := transaction.FromContext(ctx)
    if !ok {
        return errors.New("no transaction in context")
    }

    // Use transaction
    _, err := tx.Exec("UPDATE posts SET views = views + 1")
    return err
}
```

### Manual Control

```go
tx, err := mgr.Begin(ctx)
if err != nil {
    return err
}

// Do work
_, err = tx.Exec("INSERT INTO posts (title) VALUES (?)", "Manual Control")
if err != nil {
    tx.Rollback() // Explicit rollback
    return err
}

// Explicit commit
if err := tx.Commit(); err != nil {
    return err
}
```

## Isolation Levels

```go
const (
    ReadUncommitted // Allows dirty reads
    ReadCommitted   // Prevents dirty reads (PostgreSQL default)
    RepeatableRead  // Prevents non-repeatable reads
    Serializable    // Full isolation
)
```

## Error Handling

The package provides specific error types:

- `ErrTransactionAborted`: Transaction was explicitly aborted
- `ErrDeadlock`: Deadlock detected
- `ErrTransactionTimeout`: Transaction timed out
- `ErrNestedTransactionNotSupported`: Nested transaction not supported

Check for retriable errors:

```go
if transaction.IsRetryableError(err) {
    // This is a deadlock or serialization error
    // Safe to retry the transaction
}
```

## Integration with CRUD Operations

The Manager implements the `TransactionManager` interface required by CRUD operations:

```go
type TransactionManager interface {
    WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error
    BeginTx(ctx context.Context) (*sql.Tx, error)
}
```

This allows seamless integration with existing CRUD operations:

```go
ops := crud.NewOperations(
    resource,
    db,
    validator,
    hooks,
    transaction.NewManager(db), // Transaction manager
)
```

## Performance

- Transaction overhead: <1ms
- Deadlock detection: O(1)
- Retry with exponential backoff: 100ms base, 2^n multiplier
- Context cancellation: Immediate rollback

## Best Practices

1. **Keep transactions short**: Hold locks for minimal time
2. **Use appropriate isolation levels**: Higher isolation = lower concurrency
3. **Handle errors properly**: Always check error returns
4. **Use defer for rollback**: Ensures cleanup on panic
5. **Avoid nested transactions when possible**: Use them only when necessary
6. **Set reasonable timeouts**: Prevent hanging transactions
7. **Retry on deadlocks**: Use `WithRetry` for high-contention operations

## Testing

Run tests with coverage:

```bash
go test ./internal/orm/transaction/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Current coverage: **90.7%**

## Implementation Notes

- Uses PostgreSQL savepoints (`SAVEPOINT`, `RELEASE SAVEPOINT`, `ROLLBACK TO SAVEPOINT`) for nested transactions
- Detects PostgreSQL deadlock error code `40P01` and serialization error code `40001`
- Supports context cancellation for transaction timeout
- Thread-safe with atomic operations for state tracking
- Panic recovery with automatic rollback
