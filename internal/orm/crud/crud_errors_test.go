package crud

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestConvertDBErrorWithPgErrors(t *testing.T) {
	// Test unique violation
	pgErr := &pgconn.PgError{Code: "23505", Detail: "Key (email)=(test@test.com) already exists."}
	err := ConvertDBError(pgErr)
	assert.ErrorIs(t, err, ErrUniqueViolation)
	assert.Contains(t, err.Error(), "Key (email)")

	// Test foreign key violation
	pgErr = &pgconn.PgError{Code: "23503", Detail: "Key (author_id)=(123) is not present in table users."}
	err = ConvertDBError(pgErr)
	assert.ErrorIs(t, err, ErrForeignKeyViolation)
	assert.Contains(t, err.Error(), "Key (author_id)")

	// Test check violation
	pgErr = &pgconn.PgError{Code: "23514", Detail: "Check constraint failed"}
	err = ConvertDBError(pgErr)
	assert.ErrorIs(t, err, ErrCheckViolation)

	// Test not null violation
	pgErr = &pgconn.PgError{Code: "23502", ColumnName: "title"}
	err = ConvertDBError(pgErr)
	assert.ErrorIs(t, err, ErrNotNullViolation)
	assert.Contains(t, err.Error(), "title")

	// Test unknown pg error
	pgErr = &pgconn.PgError{Code: "99999", Message: "Unknown error"}
	err = ConvertDBError(pgErr)
	assert.Error(t, err)

	// Test generic error
	genericErr := errors.New("generic error")
	err = ConvertDBError(genericErr)
	assert.Equal(t, genericErr, err)
}
