package query

import (
	"fmt"
	"strings"
)

// BuildSortClause generates an SQL ORDER BY clause from a JSON:API sort array.
// Fields prefixed with '-' are sorted descending, others ascending.
// All field names are prefixed with the table name.
// Returns empty string if sorts slice is empty.
//
// SECURITY NOTE: tableName MUST be a trusted value from code generation, never from user input.
// It is not parameterized because SQL does not support parameterized table/column names.
// Field names are validated against validFields whitelist.
//
// Example: ["-created_at", "title"] -> "ORDER BY posts.created_at DESC, posts.title ASC"
func BuildSortClause(sorts []string, tableName string, validFields []string) (string, error) {
	if len(sorts) == 0 {
		return "", nil
	}

	if err := ValidateSortFields(sorts, validFields); err != nil {
		return "", err
	}

	var sortExpressions []string
	for _, sort := range sorts {
		direction := "ASC"
		fieldName := sort

		// Check for descending sort prefix
		if strings.HasPrefix(sort, "-") {
			direction = "DESC"
			fieldName = sort[1:] // Remove the '-' prefix
		}

		// Convert to snake_case if needed
		fieldName = toSnakeCase(fieldName)

		// Build the sort expression with table prefix
		sortExpr := fmt.Sprintf("%s.%s %s", tableName, fieldName, direction)
		sortExpressions = append(sortExpressions, sortExpr)
	}

	return "ORDER BY " + strings.Join(sortExpressions, ", "), nil
}

// ValidateSortFields checks that all sort fields (without '-' prefix) exist in the validFields list.
// Returns an error listing any invalid fields.
func ValidateSortFields(sorts []string, validFields []string) error {
	// Create a map for O(1) lookups
	validFieldsMap := make(map[string]bool, len(validFields))
	for _, field := range validFields {
		validFieldsMap[field] = true
	}

	var invalidFields []string
	for _, sort := range sorts {
		// Remove the '-' prefix if present
		fieldName := strings.TrimPrefix(sort, "-")

		// Convert to snake_case for validation
		fieldName = toSnakeCase(fieldName)

		if !validFieldsMap[fieldName] {
			invalidFields = append(invalidFields, fieldName)
		}
	}

	if len(invalidFields) > 0 {
		return fmt.Errorf("invalid sort fields: %s", strings.Join(invalidFields, ", "))
	}

	return nil
}
