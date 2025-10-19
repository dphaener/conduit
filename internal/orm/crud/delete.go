package crud

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Delete deletes a record by its primary key
// Supports both hard delete and soft delete (if deleted_at field exists)
func (o *Operations) Delete(
	ctx context.Context,
	id interface{},
) error {
	// Use transaction if txManager is available
	if o.txManager != nil {
		return o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			return o.deleteInTx(ctx, tx, id)
		})
	}

	// Fall back to direct execution without transaction
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := o.deleteInTx(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// deleteInTx deletes a record within a transaction
func (o *Operations) deleteInTx(
	ctx context.Context,
	tx *sql.Tx,
	id interface{},
) error {
	// 1. Load existing record
	record, err := o.findByID(ctx, tx, id)
	if err != nil {
		return err
	}

	// 2. Execute before delete hooks
	if o.hooks != nil {
		if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.BeforeDelete, record); err != nil {
			return fmt.Errorf("before_delete hook failed: %w", err)
		}
	}

	// 3. Check if soft delete is supported (deleted_at field exists)
	if o.supportsSoftDelete() {
		// Soft delete: set deleted_at timestamp
		if err := o.softDelete(ctx, tx, id); err != nil {
			return fmt.Errorf("failed to soft delete record: %w", ConvertDBError(err))
		}
	} else {
		// Hard delete: remove from database
		if err := o.hardDelete(ctx, tx, id); err != nil {
			return fmt.Errorf("failed to hard delete record: %w", ConvertDBError(err))
		}
	}

	// 4. Execute after delete hooks
	if o.hooks != nil {
		if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.AfterDelete, record); err != nil {
			return fmt.Errorf("after_delete hook failed: %w", err)
		}
	}

	return nil
}

// DeleteMany deletes multiple records matching the conditions
func (o *Operations) DeleteMany(
	ctx context.Context,
	conditions map[string]interface{},
) (int, error) {
	var count int

	// Use transaction if txManager is available
	if o.txManager != nil {
		err := o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			n, err := o.deleteManyInTx(ctx, tx, conditions)
			if err != nil {
				return err
			}
			count = n
			return nil
		})
		return count, err
	}

	// Fall back to direct execution without transaction
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	count, err = o.deleteManyInTx(ctx, tx, conditions)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

// deleteManyInTx deletes multiple records within a transaction
func (o *Operations) deleteManyInTx(
	ctx context.Context,
	tx *sql.Tx,
	conditions map[string]interface{},
) (int, error) {
	tableName := toTableName(o.resource.Name)

	// Build WHERE clause
	var whereClauses []string
	var values []interface{}
	counter := 1

	for field, value := range conditions {
		// Validate field is a column, not a relationship
		if err := o.validateFieldIsColumn(field); err != nil {
			return 0, fmt.Errorf("invalid field %s: %w", field, err)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, counter))
		values = append(values, value)
		counter++
	}

	var query string
	var result sql.Result
	var err error

	if o.supportsSoftDelete() {
		// Soft delete: set deleted_at
		if len(whereClauses) > 0 {
			query = fmt.Sprintf(
				"UPDATE %s SET deleted_at = $%d WHERE %s AND deleted_at IS NULL",
				tableName,
				counter,
				joinWithAnd(whereClauses),
			)
			values = append(values, time.Now())
		} else {
			query = fmt.Sprintf(
				"UPDATE %s SET deleted_at = $1 WHERE deleted_at IS NULL",
				tableName,
			)
			values = []interface{}{time.Now()}
		}
		result, err = tx.ExecContext(ctx, query, values...)
	} else {
		// Hard delete
		if len(whereClauses) > 0 {
			query = fmt.Sprintf(
				"DELETE FROM %s WHERE %s",
				tableName,
				joinWithAnd(whereClauses),
			)
		} else {
			query = fmt.Sprintf("DELETE FROM %s", tableName)
		}
		result, err = tx.ExecContext(ctx, query, values...)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to delete records: %w", ConvertDBError(err))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rows), nil
}

// HardDelete forces a hard delete even if soft delete is supported
func (o *Operations) HardDelete(
	ctx context.Context,
	id interface{},
) error {
	// Use transaction if txManager is available
	if o.txManager != nil {
		return o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			return o.hardDelete(ctx, tx, id)
		})
	}

	// Fall back to direct execution without transaction
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := o.hardDelete(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Restore restores a soft-deleted record
func (o *Operations) Restore(
	ctx context.Context,
	id interface{},
) (map[string]interface{}, error) {
	if !o.supportsSoftDelete() {
		return nil, fmt.Errorf("soft delete not supported for resource %s", o.resource.Name)
	}

	var result map[string]interface{}

	// Use transaction if txManager is available
	if o.txManager != nil {
		err := o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			record, err := o.restoreInTx(ctx, tx, id)
			if err != nil {
				return err
			}
			result = record
			return nil
		})
		return result, err
	}

	// Fall back to direct execution without transaction
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err = o.restoreInTx(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// restoreInTx restores a soft-deleted record within a transaction
func (o *Operations) restoreInTx(
	ctx context.Context,
	tx *sql.Tx,
	id interface{},
) (map[string]interface{}, error) {
	tableName := toTableName(o.resource.Name)

	// Build explicit RETURNING clause with sorted fields for determinism
	var returnFields []string
	for fieldName := range o.resource.Fields {
		returnFields = append(returnFields, fieldName)
	}
	sort.Strings(returnFields)

	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = NULL WHERE id = $1 RETURNING %s",
		tableName,
		strings.Join(returnFields, ", "),
	)

	row := tx.QueryRowContext(ctx, query, id)
	record, err := scanRowWithColumns(row, returnFields)
	if err != nil {
		return nil, ConvertDBError(err)
	}

	return record, nil
}

// supportsSoftDelete returns true if the resource has a deleted_at field
func (o *Operations) supportsSoftDelete() bool {
	_, hasDeletedAt := o.resource.Fields["deleted_at"]
	return hasDeletedAt
}

// softDelete performs a soft delete by setting deleted_at
func (o *Operations) softDelete(
	ctx context.Context,
	tx *sql.Tx,
	id interface{},
) error {
	tableName := toTableName(o.resource.Name)
	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL",
		tableName,
	)

	result, err := tx.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// hardDelete performs a hard delete by removing the record
func (o *Operations) hardDelete(
	ctx context.Context,
	tx *sql.Tx,
	id interface{},
) error {
	tableName := toTableName(o.resource.Name)
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", tableName)

	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
