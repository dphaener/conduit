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

// Update updates a record by its primary key with optimistic locking support
func (o *Operations) Update(
	ctx context.Context,
	id interface{},
	data map[string]interface{},
) (map[string]interface{}, error) {
	var result map[string]interface{}

	// Use transaction if txManager is available
	if o.txManager != nil {
		err := o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			record, err := o.updateInTx(ctx, tx, id, data)
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

	result, err = o.updateInTx(ctx, tx, id, data)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// updateInTx updates a record within a transaction
func (o *Operations) updateInTx(
	ctx context.Context,
	tx *sql.Tx,
	id interface{},
	data map[string]interface{},
) (map[string]interface{}, error) {
	// 1. Load existing record (for change tracking and optimistic locking)
	existing, err := o.findByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	// 2. Check optimistic locking (version field)
	if err := o.checkOptimisticLock(data, existing); err != nil {
		return nil, err
	}

	// 3. Merge changes
	record := o.mergeChanges(existing, data)

	// 4. Set up change tracking
	changeTracker := NewChangeTracker(existing, record)
	record["__changes__"] = changeTracker

	// 5. Auto-populate fields like updated_at
	if err := o.populateAutoFields(record, OperationUpdate); err != nil {
		return nil, fmt.Errorf("failed to populate auto fields: %w", err)
	}

	// 6. Execute before hooks
	if o.hooks != nil {
		if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.BeforeUpdate, record); err != nil {
			return nil, fmt.Errorf("before_update hook failed: %w", err)
		}
		if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.BeforeSave, record); err != nil {
			return nil, fmt.Errorf("before_save hook failed: %w", err)
		}
	}

	// 7. Validate
	if o.validator != nil {
		if err := o.validator.Validate(ctx, o.resource, record, OperationUpdate); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidationFailed, err)
		}
	}

	// 8. Update in database
	updated, err := o.updateRecord(ctx, tx, id, record, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", ConvertDBError(err))
	}

	// 9. Execute after hooks
	updated["__changes__"] = changeTracker
	if o.hooks != nil {
		if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.AfterUpdate, updated); err != nil {
			return nil, fmt.Errorf("after_update hook failed: %w", err)
		}
		if err := o.hooks.ExecuteHooks(ctx, o.resource, schema.AfterSave, updated); err != nil {
			return nil, fmt.Errorf("after_save hook failed: %w", err)
		}
	}

	// Remove internal tracking
	delete(updated, "__changes__")

	return updated, nil
}

// UpdateMany updates multiple records matching the conditions
func (o *Operations) UpdateMany(
	ctx context.Context,
	conditions map[string]interface{},
	updates map[string]interface{},
) (int, error) {
	var count int

	// Use transaction if txManager is available
	if o.txManager != nil {
		err := o.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			n, err := o.updateManyInTx(ctx, tx, conditions, updates)
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

	count, err = o.updateManyInTx(ctx, tx, conditions, updates)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

// updateManyInTx updates multiple records within a transaction
func (o *Operations) updateManyInTx(
	ctx context.Context,
	tx *sql.Tx,
	conditions map[string]interface{},
	updates map[string]interface{},
) (int, error) {
	tableName := toTableName(o.resource.Name)

	// Build WHERE clause
	var whereClauses []string
	var whereValues []interface{}
	counter := 1

	for field, value := range conditions {
		// Validate field is a column, not a relationship
		if err := o.validateFieldIsColumn(field); err != nil {
			return 0, fmt.Errorf("invalid field %s in conditions: %w", field, err)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, counter))
		whereValues = append(whereValues, value)
		counter++
	}

	// Build SET clause
	var sets []string
	var setValues []interface{}
	for field, value := range updates {
		if field == "id" || field == "created_at" || field == "__changes__" {
			continue // Skip immutable fields
		}
		// Skip validation for updated_at (auto-managed timestamp)
		if field != "updated_at" {
			// Validate field is a column, not a relationship
			if err := o.validateFieldIsColumn(field); err != nil {
				return 0, fmt.Errorf("invalid field %s in updates: %w", field, err)
			}
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", field, counter))
		setValues = append(setValues, value)
		counter++
	}

	// Auto-update updated_at
	sets = append(sets, fmt.Sprintf("updated_at = $%d", counter))
	setValues = append(setValues, time.Now())
	counter++

	if len(sets) == 0 {
		return 0, fmt.Errorf("no fields to update")
	}

	values := append(whereValues, setValues...)

	var query string
	if len(whereClauses) > 0 {
		query = fmt.Sprintf(
			"UPDATE %s SET %s WHERE %s",
			tableName,
			strings.Join(sets, ", "),
			joinWithAnd(whereClauses),
		)
	} else {
		query = fmt.Sprintf(
			"UPDATE %s SET %s",
			tableName,
			strings.Join(sets, ", "),
		)
	}

	result, err := tx.ExecContext(ctx, query, values...)
	if err != nil {
		return 0, fmt.Errorf("failed to update records: %w", ConvertDBError(err))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rows), nil
}

// updateRecord updates a single record in the database
func (o *Operations) updateRecord(
	ctx context.Context,
	tx *sql.Tx,
	id interface{},
	data map[string]interface{},
	existing map[string]interface{},
) (map[string]interface{}, error) {
	tableName := toTableName(o.resource.Name)

	// Build UPDATE statement
	var sets []string
	var values []interface{}
	counter := 1

	for field, value := range data {
		if field == "id" || field == "created_at" || field == "__changes__" {
			continue // Skip immutable fields and internal tracking
		}
		// Only include fields that exist in the resource schema or are timestamps
		if _, exists := o.resource.Fields[field]; exists || field == "updated_at" {
			sets = append(sets, fmt.Sprintf("%s = $%d", field, counter))
			values = append(values, value)
			counter++
		}
	}

	if len(sets) == 0 {
		// No changes, return existing record
		return existing, nil
	}

	// Auto-increment version field for optimistic locking
	if _, hasVersion := o.resource.Fields["version"]; hasVersion {
		sets = append(sets, fmt.Sprintf("version = $%d", counter))
		// Increment the existing version by 1
		existingVersion := 0
		if v, ok := existing["version"]; ok {
			if vi, ok := v.(int); ok {
				existingVersion = vi
			} else if vi64, ok := v.(int64); ok {
				existingVersion = int(vi64)
			}
		}
		values = append(values, existingVersion+1)
		counter++
	}

	// Add WHERE clause with optimistic locking if version field exists
	values = append(values, id)
	whereClause := fmt.Sprintf("id = $%d", counter)
	counter++

	// Check for version field for optimistic locking
	if _, hasVersion := o.resource.Fields["version"]; hasVersion {
		if existingVersion, ok := existing["version"]; ok {
			whereClause += fmt.Sprintf(" AND version = $%d", counter)
			values = append(values, existingVersion)
			counter++
		}
	}

	// Build explicit RETURNING clause with sorted fields for determinism
	var returnFields []string
	for fieldName := range o.resource.Fields {
		returnFields = append(returnFields, fieldName)
	}
	sort.Strings(returnFields)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s RETURNING %s",
		tableName,
		strings.Join(sets, ", "),
		whereClause,
		strings.Join(returnFields, ", "),
	)

	row := tx.QueryRowContext(ctx, query, values...)
	updated, err := scanRowWithColumns(row, returnFields)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOptimisticLockFailed
		}
		return nil, err
	}

	return updated, nil
}

// checkOptimisticLock checks if the version field matches for optimistic locking
func (o *Operations) checkOptimisticLock(updates, existing map[string]interface{}) error {
	// Check if resource has a version field
	if _, hasVersion := o.resource.Fields["version"]; !hasVersion {
		return nil // No version field, skip optimistic locking
	}

	// If update includes version, check it matches
	if updateVersion, ok := updates["version"]; ok {
		if existingVersion, ok := existing["version"]; ok {
			if updateVersion != existingVersion {
				return ErrOptimisticLockFailed
			}
		}
	}

	return nil
}

// mergeChanges merges updates into existing record
func (o *Operations) mergeChanges(
	existing, updates map[string]interface{},
) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy existing
	for k, v := range existing {
		result[k] = v
	}

	// Apply updates
	for k, v := range updates {
		result[k] = v
	}

	return result
}

// ChangeTracker tracks field changes in a record
type ChangeTracker struct {
	before map[string]interface{}
	after  map[string]interface{}
}

// NewChangeTracker creates a new change tracker
func NewChangeTracker(before, after map[string]interface{}) *ChangeTracker {
	return &ChangeTracker{
		before: before,
		after:  after,
	}
}

// Changed returns true if the field changed
func (ct *ChangeTracker) Changed(field string) bool {
	beforeVal, beforeOk := ct.before[field]
	afterVal, afterOk := ct.after[field]

	if beforeOk != afterOk {
		return true
	}

	return beforeVal != afterVal
}

// WasChanged returns the list of changed fields
func (ct *ChangeTracker) WasChanged() []string {
	var changed []string
	for field := range ct.after {
		if ct.Changed(field) {
			changed = append(changed, field)
		}
	}
	return changed
}

// PreviousValue returns the previous value of a field
func (ct *ChangeTracker) PreviousValue(field string) interface{} {
	return ct.before[field]
}
