package query

import (
	"fmt"
	"strings"
)

// BuildFilterClause generates a SQL WHERE clause from a filter map.
// It validates fields against a whitelist and returns parameterized query components.
//
// Parameters:
//   - filters: Map of field names to filter values
//   - tableName: Database table name to prefix columns with (MUST be from code generation, not user input)
//   - validFields: Whitelist of allowed field names for filtering
//
// Returns:
//   - whereClause: SQL WHERE clause (empty string if no filters)
//   - args: Slice of arguments for parameterized query
//   - err: Error if validation fails
//
// SECURITY NOTE: tableName MUST be a trusted value from code generation, never from user input.
// It is not parameterized because SQL does not support parameterized table/column names.
// Field names are validated against validFields whitelist, and values are parameterized.
//
// Example:
//
//	filters := map[string]string{"status": "published", "author_id": "123"}
//	clause, args, err := BuildFilterClause(filters, "posts", []string{"status", "author_id"})
//	// Returns: "WHERE posts.status = $1 AND posts.author_id = $2", ["published", "123"], nil
func BuildFilterClause(filters map[string]string, tableName string, validFields []string) (string, []interface{}, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	// Validate all filter fields first
	if err := ValidateFilterFields(filters, validFields); err != nil {
		return "", nil, err
	}

	// Build WHERE clause with parameterized queries
	var conditions []string
	var args []interface{}
	paramIndex := 1

	// Iterate in deterministic order for testing consistency
	// Extract keys and sort them
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	// Sort keys for deterministic output
	sortKeys(keys)

	for _, field := range keys {
		value := filters[field]
		// Convert field name to snake_case and prefix with table name
		columnName := fmt.Sprintf("%s.%s", tableName, toSnakeCase(field))
		condition := fmt.Sprintf("%s = $%d", columnName, paramIndex)
		conditions = append(conditions, condition)
		args = append(args, value)
		paramIndex++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")
	return whereClause, args, nil
}

// ValidateFilterFields checks if all filter fields are in the validFields whitelist.
// Returns an error listing any invalid fields found.
func ValidateFilterFields(filters map[string]string, validFields []string) error {
	if len(filters) == 0 {
		return nil
	}

	// Create a set of valid fields for O(1) lookup
	validSet := make(map[string]bool, len(validFields))
	for _, field := range validFields {
		validSet[field] = true
	}

	// Check each filter field (convert to snake_case for validation)
	var invalidFields []string
	for field := range filters {
		snakeCaseField := toSnakeCase(field)
		if !validSet[snakeCaseField] {
			invalidFields = append(invalidFields, snakeCaseField)
		}
	}

	if len(invalidFields) > 0 {
		// Sort for deterministic error messages
		sortKeys(invalidFields)
		return fmt.Errorf("invalid filter fields: %s", strings.Join(invalidFields, ", "))
	}

	return nil
}
