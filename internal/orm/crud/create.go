package crud

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/conduit-lang/conduit/internal/orm/schema"
	"github.com/google/uuid"
)

// Create creates a new record with validation and lifecycle hooks
func (o *Operations) Create(
	ctx context.Context,
	data map[string]interface{},
) (map[string]interface{}, error) {
	var result map[string]interface{}

	// Use transaction if txManager is available
	if o.txManager != nil {
		err := o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			record, err := o.createInTx(ctx, tx, data)
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

	result, err = o.createInTx(ctx, tx, data)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// createInTx creates a record within a transaction
func (o *Operations) createInTx(
	ctx context.Context,
	tx *sql.Tx,
	data map[string]interface{},
) (map[string]interface{}, error) {
	// Make a copy to avoid mutating input
	record := make(map[string]interface{})
	for k, v := range data {
		record[k] = v
	}

	// 1. Auto-populate @auto fields
	if err := o.populateAutoFields(record, OperationCreate); err != nil {
		return nil, fmt.Errorf("failed to populate auto fields: %w", err)
	}

	// 2. Execute before create hooks
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

	// 4. Insert into database
	inserted, err := o.insertRecord(ctx, tx, record)
	if err != nil {
		return nil, fmt.Errorf("failed to insert record: %w", ConvertDBError(err))
	}

	// 5. Execute after create hooks
	if o.hooks != nil {
		if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.AfterCreate, inserted); err != nil {
			return nil, fmt.Errorf("after_create hook failed: %w", err)
		}
		if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.AfterSave, inserted); err != nil {
			return nil, fmt.Errorf("after_save hook failed: %w", err)
		}
	}

	return inserted, nil
}

// insertRecord inserts a record into the database
func (o *Operations) insertRecord(
	ctx context.Context,
	tx *sql.Tx,
	data map[string]interface{},
) (map[string]interface{}, error) {
	tableName := toTableName(o.resource.Name)

	// Build INSERT statement
	var fields []string
	var placeholders []string
	var values []interface{}
	counter := 1

	// Collect fields and values, excluding relationships
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

	// Build explicit RETURNING clause with all fields in sorted order for determinism
	var returnFields []string
	for fieldName := range o.resource.Fields {
		returnFields = append(returnFields, fieldName)
	}
	sort.Strings(returnFields)

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(returnFields, ", "),
	)

	row := tx.QueryRowContext(ctx, query, values...)
	return scanRowWithColumns(row, returnFields)
}

// populateAutoFields populates @auto fields like id, created_at, updated_at
func (o *Operations) populateAutoFields(record map[string]interface{}, operation Operation) error {
	now := time.Now()

	for fieldName, field := range o.resource.Fields {
		// Check for @auto annotation
		hasAuto := false
		hasPrimary := false
		hasAutoUpdate := false

		for _, annotation := range field.Annotations {
			switch annotation.Name {
			case "auto":
				hasAuto = true
			case "primary":
				hasPrimary = true
			case "auto_update":
				hasAutoUpdate = true
			}
		}

		// Auto-generate primary key if @auto and @primary
		if hasAuto && hasPrimary && operation == OperationCreate {
			if _, exists := record[fieldName]; !exists {
				// Generate UUID for primary key
				if field.Type.BaseType == schema.TypeUUID {
					record[fieldName] = uuid.New()
				}
			}
		}

		// Auto-populate timestamp fields
		if operation == OperationCreate {
			if fieldName == "created_at" && field.Type.BaseType == schema.TypeTimestamp {
				if _, exists := record[fieldName]; !exists {
					record[fieldName] = now
				}
			}
			if fieldName == "updated_at" && field.Type.BaseType == schema.TypeTimestamp {
				if _, exists := record[fieldName]; !exists {
					record[fieldName] = now
				}
			}
		}

		if operation == OperationUpdate {
			// Auto-update timestamp fields with @auto_update
			if hasAutoUpdate && field.Type.BaseType == schema.TypeTimestamp {
				record[fieldName] = now
			}
			// Always update updated_at on update
			if fieldName == "updated_at" && field.Type.BaseType == schema.TypeTimestamp {
				record[fieldName] = now
			}
		}
	}

	return nil
}

// toTableName converts a resource name to a table name (snake_case plural)
func toTableName(resourceName string) string {
	snake := toSnakeCase(resourceName)
	return pluralize(snake)
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result []rune
	runes := []rune(s)

	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			if prev >= 'a' && prev <= 'z' {
				result = append(result, '_')
			} else if i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
				result = append(result, '_')
			}
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+('a'-'A'))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// pluralize adds simple pluralization
func pluralize(s string) string {
	if strings.HasSuffix(s, "s") ||
		strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

