package crud

import (
	"context"
	"database/sql"
	"fmt"
)

// Find retrieves a record by its primary key
func (o *Operations) Find(
	ctx context.Context,
	id interface{},
) (map[string]interface{}, error) {
	return o.findByID(ctx, o.db, id)
}

// FindBy retrieves a single record by a specific field value
func (o *Operations) FindBy(
	ctx context.Context,
	field string,
	value interface{},
) (map[string]interface{}, error) {
	// Validate field is a column, not a relationship
	if err := o.validateFieldIsColumn(field); err != nil {
		return nil, fmt.Errorf("invalid field %s: %w", field, err)
	}

	tableName := toTableName(o.resource.Name)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1 LIMIT 1", tableName, field)

	row := o.db.QueryRowContext(ctx, query, value)
	record, err := scanSingleRow(row, o.resource)
	if err != nil {
		return nil, fmt.Errorf("failed to find record by %s: %w", field, ConvertDBError(err))
	}

	return record, nil
}

// FindAll retrieves all records matching the given conditions
func (o *Operations) FindAll(
	ctx context.Context,
	conditions map[string]interface{},
) ([]map[string]interface{}, error) {
	tableName := toTableName(o.resource.Name)

	// Build WHERE clause
	var whereClauses []string
	var values []interface{}
	counter := 1

	for field, value := range conditions {
		// Validate field is a column, not a relationship
		if err := o.validateFieldIsColumn(field); err != nil {
			return nil, fmt.Errorf("invalid field %s: %w", field, err)
		}

		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, counter))
		values = append(values, value)
		counter++
	}

	var query string
	if len(whereClauses) > 0 {
		query = fmt.Sprintf("SELECT * FROM %s WHERE %s",
			tableName,
			joinWithAnd(whereClauses))
	} else {
		query = fmt.Sprintf("SELECT * FROM %s", tableName)
	}

	rows, err := o.db.QueryContext(ctx, query, values...)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", ConvertDBError(err))
	}
	defer rows.Close()

	results, err := scanRows(rows, o.resource)
	if err != nil {
		return nil, fmt.Errorf("failed to scan query results: %w", ConvertDBError(err))
	}

	return results, nil
}

// Exists checks if a record exists by its primary key
func (o *Operations) Exists(
	ctx context.Context,
	id interface{},
) (bool, error) {
	tableName := toTableName(o.resource.Name)
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", tableName)

	var exists bool
	err := o.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if record exists: %w", ConvertDBError(err))
	}

	return exists, nil
}

// Count returns the total number of records
func (o *Operations) Count(
	ctx context.Context,
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
	if len(whereClauses) > 0 {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s",
			tableName,
			joinWithAnd(whereClauses))
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	}

	var count int
	err := o.db.QueryRowContext(ctx, query, values...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count records: %w", ConvertDBError(err))
	}

	return count, nil
}

// findByID retrieves a record by its primary key (internal)
func (o *Operations) findByID(
	ctx context.Context,
	db interface{ QueryRowContext(context.Context, string, ...interface{}) *sql.Row },
	id interface{},
) (map[string]interface{}, error) {
	tableName := toTableName(o.resource.Name)
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", tableName)

	row := db.QueryRowContext(ctx, query, id)
	record, err := scanSingleRow(row, o.resource)
	if err != nil {
		return nil, fmt.Errorf("failed to find record by id: %w", ConvertDBError(err))
	}

	return record, nil
}


// joinWithAnd joins a slice of strings with " AND "
func joinWithAnd(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " AND "
		}
		result += part
	}
	return result
}
