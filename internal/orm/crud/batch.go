package crud

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// CreateMany creates multiple records in a single transaction
// This is optimized for bulk inserts with better performance than individual creates
func (o *Operations) CreateMany(
	ctx context.Context,
	records []map[string]interface{},
) ([]map[string]interface{}, error) {
	if len(records) == 0 {
		return []map[string]interface{}{}, nil
	}

	var results []map[string]interface{}

	// Use transaction if txManager is available
	if o.txManager != nil {
		err := o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			inserted, err := o.createManyInTx(ctx, tx, records)
			if err != nil {
				return err
			}
			results = inserted
			return nil
		})
		return results, err
	}

	// Fall back to direct execution without transaction
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	results, err = o.createManyInTx(ctx, tx, records)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return results, nil
}

// createManyInTx creates multiple records within a transaction
func (o *Operations) createManyInTx(
	ctx context.Context,
	tx *sql.Tx,
	records []map[string]interface{},
) ([]map[string]interface{}, error) {
	// For now, we'll use individual inserts within the same transaction
	// This ensures hooks and validation run for each record
	// A future optimization could use COPY or multi-value INSERT for better performance

	var results []map[string]interface{}

	for _, data := range records {
		// Make a copy to avoid mutating input
		record := make(map[string]interface{})
		for k, v := range data {
			record[k] = v
		}

		// 1. Auto-populate @auto fields
		if err := o.populateAutoFields(record, OperationCreate); err != nil {
			return nil, fmt.Errorf("failed to populate auto fields: %w", err)
		}

		// 2. Execute before hooks
		if o.hooks != nil {
			if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.BeforeCreate, record); err != nil {
				return nil, fmt.Errorf("before_create hook failed: %w", err)
			}
			if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.BeforeSave, record); err != nil {
				return nil, fmt.Errorf("before_save hook failed: %w", err)
			}
		}

		// 3. Validate
		if o.validator != nil {
			if err := o.validator.Validate(ctx, o.resource, record, OperationCreate); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrValidationFailed, err)
			}
		}

		// 4. Insert
		inserted, err := o.insertRecord(ctx, tx, record)
		if err != nil {
			return nil, ConvertDBError(err)
		}

		// 5. Execute after hooks
		if o.hooks != nil {
			if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.AfterCreate, inserted); err != nil {
				return nil, fmt.Errorf("after_create hook failed: %w", err)
			}
			if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.AfterSave, inserted); err != nil {
				return nil, fmt.Errorf("after_save hook failed: %w", err)
			}
		}

		results = append(results, inserted)
	}

	return results, nil
}

// BulkInsert performs a bulk insert without hooks or validation
// This is the fastest way to insert many records but bypasses safety features
// Use with caution and only when performance is critical
func (o *Operations) BulkInsert(
	ctx context.Context,
	records []map[string]interface{},
) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	var count int

	// Use transaction if txManager is available
	if o.txManager != nil {
		err := o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			n, err := o.bulkInsertInTx(ctx, tx, records)
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

	count, err = o.bulkInsertInTx(ctx, tx, records)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

// bulkInsertInTx performs bulk insert within a transaction using multi-value INSERT
func (o *Operations) bulkInsertInTx(
	ctx context.Context,
	tx *sql.Tx,
	records []map[string]interface{},
) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	tableName := toTableName(o.resource.Name)

	// Determine fields from first record (all records must have same structure)
	var fields []string
	for fieldName := range o.resource.Fields {
		if _, ok := records[0][fieldName]; ok {
			fields = append(fields, fieldName)
		}
	}

	if len(fields) == 0 {
		return 0, fmt.Errorf("no fields to insert")
	}

	// Build multi-value INSERT statement
	var valuePlaceholders []string
	var values []interface{}
	counter := 1

	for _, record := range records {
		var recordPlaceholders []string
		for _, field := range fields {
			recordPlaceholders = append(recordPlaceholders, fmt.Sprintf("$%d", counter))
			values = append(values, record[field])
			counter++
		}
		valuePlaceholders = append(valuePlaceholders, fmt.Sprintf("(%s)", strings.Join(recordPlaceholders, ", ")))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(valuePlaceholders, ", "),
	)

	result, err := tx.ExecContext(ctx, query, values...)
	if err != nil {
		return 0, ConvertDBError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rows), nil
}

// Upsert performs an INSERT ... ON CONFLICT DO UPDATE (PostgreSQL)
// This allows atomic insert-or-update operations
func (o *Operations) Upsert(
	ctx context.Context,
	data map[string]interface{},
	conflictFields []string,
) (map[string]interface{}, error) {
	var result map[string]interface{}

	// Use transaction if txManager is available
	if o.txManager != nil {
		err := o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			record, err := o.upsertInTx(ctx, tx, data, conflictFields)
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

	result, err = o.upsertInTx(ctx, tx, data, conflictFields)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// upsertInTx performs upsert within a transaction
func (o *Operations) upsertInTx(
	ctx context.Context,
	tx *sql.Tx,
	data map[string]interface{},
	conflictFields []string,
) (map[string]interface{}, error) {
	tableName := toTableName(o.resource.Name)

	// Build INSERT statement
	var fields []string
	var placeholders []string
	var values []interface{}
	counter := 1

	for fieldName := range o.resource.Fields {
		if value, ok := data[fieldName]; ok {
			fields = append(fields, fieldName)
			placeholders = append(placeholders, fmt.Sprintf("$%d", counter))
			values = append(values, value)
			counter++
		}
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to insert")
	}

	// Build UPDATE SET clause (exclude conflict fields and id)
	var updateSets []string
	for _, field := range fields {
		if field == "id" || field == "created_at" || contains(conflictFields, field) {
			continue
		}
		updateSets = append(updateSets, fmt.Sprintf("%s = EXCLUDED.%s", field, field))
	}

	// Build conflict target
	conflictTarget := strings.Join(conflictFields, ", ")

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s RETURNING *",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
		conflictTarget,
		strings.Join(updateSets, ", "),
	)

	row := tx.QueryRowContext(ctx, query, values...)
	record, err := scanSingleRow(row, o.resource)
	if err != nil {
		return nil, ConvertDBError(err)
	}

	return record, nil
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
