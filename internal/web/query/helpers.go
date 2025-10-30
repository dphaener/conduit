package query

import "sort"

// toSnakeCase converts a string from camelCase or PascalCase to snake_case.
// This is a simple implementation that handles common ASCII cases.
// Limitations:
//   - Only handles ASCII uppercase letters (A-Z)
//   - Does not special-case acronyms (HTTPServer -> h_t_t_p_server)
//   - Preserves non-ASCII characters as-is
func toSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+32) // Convert to lowercase
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// sortKeys sorts a slice of strings in place for deterministic ordering.
// Uses the standard library's sort.Strings which implements quicksort.
func sortKeys(keys []string) {
	sort.Strings(keys)
}
