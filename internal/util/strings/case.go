package strings

import (
	"strings"
	"unicode"
)

// ToSnakeCase converts CamelCase to snake_case
// Handles acronyms properly (HTTPRequest -> http_request)
func ToSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				// Add underscore before uppercase letter if:
				// 1. Previous char is lowercase
				// 2. Next char is lowercase (for acronyms like HTTPRequest -> http_request)
				if unicode.IsLower(prev) {
					result.WriteRune('_')
				} else if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
