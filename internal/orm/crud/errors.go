package crud

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// Common CRUD error types
var (
	// ErrNotFound is returned when a record is not found
	ErrNotFound = errors.New("record not found")

	// ErrOptimisticLockFailed is returned when a record was modified by another transaction
	ErrOptimisticLockFailed = errors.New("record was modified by another transaction")

	// ErrUniqueViolation is returned when a unique constraint is violated
	ErrUniqueViolation = errors.New("unique constraint violation")

	// ErrForeignKeyViolation is returned when a foreign key constraint is violated
	ErrForeignKeyViolation = errors.New("foreign key constraint violation")

	// ErrValidationFailed is returned when validation fails
	ErrValidationFailed = errors.New("validation failed")

	// ErrCheckViolation is returned when a check constraint is violated
	ErrCheckViolation = errors.New("check constraint violation")

	// ErrNotNullViolation is returned when a NOT NULL constraint is violated
	ErrNotNullViolation = errors.New("not null constraint violation")

	// ErrFieldNotFound is returned when a field does not exist on a resource
	ErrFieldNotFound = errors.New("field not found")

	// ErrRelationshipField is returned when trying to use a relationship field in a WHERE clause
	ErrRelationshipField = errors.New("relationship field cannot be used directly in queries")
)

// ValidationError contains multiple validation errors for a record
type ValidationError struct {
	Errors []FieldError
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}
	if len(ve.Errors) == 1 {
		return fmt.Sprintf("validation failed: %s: %s", ve.Errors[0].Field, ve.Errors[0].Message)
	}
	return fmt.Sprintf("validation failed: %d errors", len(ve.Errors))
}

// FieldError represents a validation error on a specific field
type FieldError struct {
	Field   string
	Message string
}

// ConvertDBError converts database-specific errors to CRUD errors
func ConvertDBError(err error) error {
	if err == nil {
		return nil
	}

	// Check for sql.ErrNoRows
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	// Check for PostgreSQL errors (pgx)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%w: %s", ErrUniqueViolation, pgErr.Detail)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %s", ErrForeignKeyViolation, pgErr.Detail)
		case "23514": // check_violation
			return fmt.Errorf("%w: %s", ErrCheckViolation, pgErr.Detail)
		case "23502": // not_null_violation
			return fmt.Errorf("%w: column %s", ErrNotNullViolation, pgErr.ColumnName)
		}
	}

	return err
}

// IsNotFound returns true if the error is ErrNotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUniqueViolation returns true if the error is ErrUniqueViolation
func IsUniqueViolation(err error) bool {
	return errors.Is(err, ErrUniqueViolation)
}

// IsForeignKeyViolation returns true if the error is ErrForeignKeyViolation
func IsForeignKeyViolation(err error) bool {
	return errors.Is(err, ErrForeignKeyViolation)
}

// IsOptimisticLockFailed returns true if the error is ErrOptimisticLockFailed
func IsOptimisticLockFailed(err error) bool {
	return errors.Is(err, ErrOptimisticLockFailed)
}

// IsValidationFailed returns true if the error is a validation error
func IsValidationFailed(err error) bool {
	if errors.Is(err, ErrValidationFailed) {
		return true
	}
	var valErr *ValidationError
	return errors.As(err, &valErr)
}
