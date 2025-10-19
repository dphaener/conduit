package crud

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/conduit-lang/conduit/internal/orm/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindBy(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()
	testID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM posts WHERE title`).
		WithArgs("Test Post").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "created_at", "updated_at"}).
			AddRow("Test content", now, testID, "Test Post", now))

	result, err := ops.FindBy(ctx, "title", "Test Post")
	require.NoError(t, err)
	assert.Equal(t, "Test Post", result["title"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByInvalidField(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()

	_, err = ops.FindBy(ctx, "nonexistent", "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field not found")
}

func TestFindAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM posts`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "created_at", "updated_at"}).
			AddRow(uuid.New(), "Post 1", "Content 1", now, now).
			AddRow(uuid.New(), "Post 2", "Content 2", now, now))

	results, err := ops.FindAll(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindAllWithConditions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM posts WHERE title`).
		WithArgs("Test Post").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "created_at", "updated_at"}).
			AddRow(uuid.New(), "Test Post", "Content 1", now, now))

	results, err := ops.FindAll(ctx, map[string]interface{}{"title": "Test Post"})
	require.NoError(t, err)
	assert.Len(t, results, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMany(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE posts SET`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 5))
	mock.ExpectCommit()

	count, err := ops.UpdateMany(ctx, map[string]interface{}{"title": "Old"}, map[string]interface{}{"title": "New"})
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteMany(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM posts WHERE title`).
		WithArgs("Test Post").
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	count, err := ops.DeleteMany(ctx, map[string]interface{}{"title": "Test Post"})
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHardDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	// Add soft delete support
	resource.Fields["deleted_at"] = &schema.Field{
		Name: "deleted_at",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeTimestamp,
			Nullable: true,
		},
	}

	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()
	testID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM posts WHERE id`).
		WithArgs(testID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = ops.HardDelete(ctx, testID)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRestore(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	// Add soft delete support
	resource.Fields["deleted_at"] = &schema.Field{
		Name: "deleted_at",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeTimestamp,
			Nullable: true,
		},
	}

	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()
	testID := uuid.New()
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE posts SET deleted_at = NULL`).
		WithArgs(testID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "created_at", "updated_at", "deleted_at"}).
			AddRow(testID, "Test Post", "Test content", now, now, nil))
	mock.ExpectCommit()

	result, err := ops.Restore(ctx, testID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRestoreWithoutSoftDelete(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()
	testID := uuid.New()

	_, err = ops.Restore(ctx, testID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "soft delete not supported")
}

func TestBulkInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()

	records := []map[string]interface{}{
		{"id": uuid.New(), "title": "Post 1", "content": "Content 1"},
		{"id": uuid.New(), "title": "Post 2", "content": "Content 2"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO posts`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	count, err := ops.BulkInsert(ctx, records)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()
	testID := uuid.New()
	now := time.Now()

	data := map[string]interface{}{
		"id":      testID,
		"title":   "Test Post",
		"content": "Test content",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO posts .* ON CONFLICT`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "created_at", "updated_at"}).
			AddRow("Test content", now, testID, "Test Post", now))
	mock.ExpectCommit()

	result, err := ops.Upsert(ctx, data, []string{"id"})
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOperationString(t *testing.T) {
	assert.Equal(t, "create", OperationCreate.String())
	assert.Equal(t, "read", OperationRead.String())
	assert.Equal(t, "update", OperationUpdate.String())
	assert.Equal(t, "delete", OperationDelete.String())
	assert.Equal(t, "unknown", Operation(999).String())
}

func TestResourceAndDBGetters(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	assert.Equal(t, resource, ops.Resource())
	assert.Equal(t, db, ops.DB())
}

func TestErrorHelpers(t *testing.T) {
	assert.True(t, IsNotFound(ErrNotFound))
	assert.False(t, IsNotFound(nil))

	assert.True(t, IsUniqueViolation(ErrUniqueViolation))
	assert.False(t, IsUniqueViolation(nil))

	assert.True(t, IsForeignKeyViolation(ErrForeignKeyViolation))
	assert.False(t, IsForeignKeyViolation(nil))

	assert.True(t, IsOptimisticLockFailed(ErrOptimisticLockFailed))
	assert.False(t, IsOptimisticLockFailed(nil))

	assert.True(t, IsValidationFailed(ErrValidationFailed))
	assert.True(t, IsValidationFailed(&ValidationError{}))
	assert.False(t, IsValidationFailed(nil))
}

func TestValidationErrorMultiple(t *testing.T) {
	err := &ValidationError{
		Errors: []FieldError{
			{Field: "title", Message: "title is required"},
			{Field: "content", Message: "content is required"},
		},
	}

	assert.Contains(t, err.Error(), "2 errors")
}

func TestValidationErrorSingle(t *testing.T) {
	err := &ValidationError{
		Errors: []FieldError{
			{Field: "title", Message: "title is required"},
		},
	}

	assert.Contains(t, err.Error(), "title: title is required")
}

func TestValidationErrorEmpty(t *testing.T) {
	err := &ValidationError{
		Errors: []FieldError{},
	}

	assert.Equal(t, "validation failed", err.Error())
}

func TestCheckOptimisticLock(t *testing.T) {
	resource := createTestResource()
	// Add version field
	resource.Fields["version"] = &schema.Field{
		Name: "version",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeInt,
			Nullable: false,
		},
	}

	ops := NewOperations(resource, nil, nil, nil, nil)

	// Test matching versions
	updates := map[string]interface{}{"version": 1}
	existing := map[string]interface{}{"version": 1}
	err := ops.checkOptimisticLock(updates, existing)
	assert.NoError(t, err)

	// Test mismatched versions
	updates = map[string]interface{}{"version": 2}
	existing = map[string]interface{}{"version": 1}
	err = ops.checkOptimisticLock(updates, existing)
	assert.ErrorIs(t, err, ErrOptimisticLockFailed)
}

func TestCountWithConditions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resource := createTestResource()
	ops := NewOperations(resource, db, nil, nil, nil)

	ctx := context.Background()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE title`).
		WithArgs("Test Post").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := ops.Count(ctx, map[string]interface{}{"title": "Test Post"})
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestJoinWithAnd(t *testing.T) {
	assert.Equal(t, "a", joinWithAnd([]string{"a"}))
	assert.Equal(t, "a AND b", joinWithAnd([]string{"a", "b"}))
	assert.Equal(t, "a AND b AND c", joinWithAnd([]string{"a", "b", "c"}))
}

func TestContains(t *testing.T) {
	assert.True(t, contains([]string{"a", "b", "c"}, "b"))
	assert.False(t, contains([]string{"a", "b", "c"}, "d"))
	assert.False(t, contains([]string{}, "a"))
}
