package crud

import (
	"database/sql"
	"sort"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// scanRowWithColumns scans a single row with known column order
func scanRowWithColumns(row *sql.Row, columns []string) (map[string]interface{}, error) {
	// Create value holders for each column
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan the row
	if err := row.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	// Build the result map
	record := make(map[string]interface{})
	for i, col := range columns {
		record[col] = values[i]
	}

	return record, nil
}

// scanSingleRow scans a single row into a map using all resource fields
// This is a fallback for SELECT * queries - prefer scanRowWithColumns for explicit column lists
func scanSingleRow(row *sql.Row, resource *schema.ResourceSchema) (map[string]interface{}, error) {
	// Build column list from all resource fields in sorted order for determinism
	var columns []string
	for fieldName := range resource.Fields {
		columns = append(columns, fieldName)
	}
	sort.Strings(columns)

	return scanRowWithColumns(row, columns)
}

// scanRows scans multiple rows into a slice of maps
func scanRows(rows *sql.Rows, resource *schema.ResourceSchema) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		record := make(map[string]interface{})
		for i, col := range columns {
			record[col] = values[i]
		}

		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
