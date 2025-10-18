// Package runtime provides runtime implementations of Conduit standard library functions.
// These functions are called by generated Go code and map directly to the type signatures
// defined in internal/compiler/typechecker/stdlib.go.
package runtime

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// String Namespace Functions (7 functions)

// StringLength returns the length of a string in characters (runes, not bytes).
// Maps to: String.length(s: string!) -> int!
func StringLength(s string) int {
	return len([]rune(s))
}

// StringSlugify converts a string to a URL-friendly slug.
// Maps to: String.slugify(s: string!) -> string!
//
// Example:
//   StringSlugify("Hello World!") => "hello-world"
//   StringSlugify("  Multiple   Spaces  ") => "multiple-spaces"
func StringSlugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace non-alphanumeric characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	// Trim leading and trailing hyphens
	s = strings.Trim(s, "-")

	return s
}

// StringUpcase converts a string to uppercase.
// Maps to: String.upcase(s: string!) -> string!
func StringUpcase(s string) string {
	return strings.ToUpper(s)
}

// StringDowncase converts a string to lowercase.
// Maps to: String.downcase(s: string!) -> string!
func StringDowncase(s string) string {
	return strings.ToLower(s)
}

// StringTrim removes leading and trailing whitespace from a string.
// Maps to: String.trim(s: string!) -> string!
func StringTrim(s string) string {
	return strings.TrimSpace(s)
}

// StringContains checks if a string contains a substring.
// Maps to: String.contains(s: string!, substr: string!) -> bool!
func StringContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// StringReplace replaces all occurrences of old with new in string s.
// Maps to: String.replace(s: string!, old: string!, new: string!) -> string!
func StringReplace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

// Time Namespace Functions (4 functions)

// TimeNow returns the current time as a timestamp.
// Maps to: Time.now() -> timestamp!
func TimeNow() time.Time {
	return time.Now()
}

// TimeFormat formats a timestamp according to the specified layout string.
// Maps to: Time.format(t: timestamp!, layout: string!) -> string!
//
// Uses Go's standard time formatting layout. Common layouts:
//   "2006-01-02" => "2025-10-17"
//   "2006-01-02 15:04:05" => "2025-10-17 14:30:00"
//   "Jan 2, 2006" => "Oct 17, 2025"
func TimeFormat(t time.Time, layout string) string {
	return t.Format(layout)
}

// TimeParse parses a string into a timestamp using the specified layout.
// Maps to: Time.parse(s: string!, layout: string!) -> timestamp?
//
// Returns nil if parsing fails, making this a nullable return type.
func TimeParse(s, layout string) *time.Time {
	t, err := time.Parse(layout, s)
	if err != nil {
		return nil
	}
	return &t
}

// TimeAddDays adds the specified number of days to a timestamp.
// Maps to: Time.add_days(t: timestamp!, days: int!) -> timestamp!
//
// Negative values subtract days.
func TimeAddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// Array Namespace Functions (2 functions)

// ArrayLength returns the length of an array.
// Maps to: Array.length(arr: T[]!) -> int!
//
// Generic function - works with any slice type.
func ArrayLength(arr interface{}) int {
	// In generated code, this will be type-safe
	// For runtime, we need to handle the interface
	switch v := arr.(type) {
	case []interface{}:
		return len(v)
	case []string:
		return len(v)
	case []int:
		return len(v)
	case []float64:
		return len(v)
	case []bool:
		return len(v)
	default:
		// Fallback - should not happen in generated code
		return 0
	}
}

// ArrayContains checks if an array contains a specific value.
// Maps to: Array.contains(arr: T[]!, value: T!) -> bool!
//
// Generic function - performs equality check on elements.
func ArrayContains(arr interface{}, value interface{}) bool {
	switch v := arr.(type) {
	case []interface{}:
		for _, item := range v {
			if item == value {
				return true
			}
		}
	case []string:
		if strVal, ok := value.(string); ok {
			for _, item := range v {
				if item == strVal {
					return true
				}
			}
		}
	case []int:
		if intVal, ok := value.(int); ok {
			for _, item := range v {
				if item == intVal {
					return true
				}
			}
		}
	case []float64:
		if floatVal, ok := value.(float64); ok {
			for _, item := range v {
				if item == floatVal {
					return true
				}
			}
		}
	case []bool:
		if boolVal, ok := value.(bool); ok {
			for _, item := range v {
				if item == boolVal {
					return true
				}
			}
		}
	}
	return false
}

// Hash Namespace Functions (1 function)

// HashHasKey checks if a hash/map contains a specific key.
// Maps to: Hash.has_key(h: hash{K!, V!}!, key: K!) -> bool!
//
// Generic function - works with any map type.
func HashHasKey(h interface{}, key interface{}) bool {
	switch m := h.(type) {
	case map[string]interface{}:
		if strKey, ok := key.(string); ok {
			_, exists := m[strKey]
			return exists
		}
	case map[string]string:
		if strKey, ok := key.(string); ok {
			_, exists := m[strKey]
			return exists
		}
	case map[string]int:
		if strKey, ok := key.(string); ok {
			_, exists := m[strKey]
			return exists
		}
	case map[int]interface{}:
		if intKey, ok := key.(int); ok {
			_, exists := m[intKey]
			return exists
		}
	case map[int]string:
		if intKey, ok := key.(int); ok {
			_, exists := m[intKey]
			return exists
		}
	case map[int]int:
		if intKey, ok := key.(int); ok {
			_, exists := m[intKey]
			return exists
		}
	}
	return false
}

// UUID Namespace Functions (1 function)

// UUIDGenerate generates a new UUID v4.
// Maps to: UUID.generate() -> uuid!
func UUIDGenerate() string {
	return uuid.New().String()
}
