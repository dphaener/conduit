// Package stdlib provides a static registry of standard library functions
// available in Conduit. This registry is used for introspection, documentation,
// and type checking without requiring a full compilation.
package stdlib

import "sort"

// FunctionDef represents a function signature in the standard library
type FunctionDef struct {
	Name        string // Function name (without namespace)
	Signature   string // Full signature: Name(params) -> returnType
	Description string // One-line description of what the function does
}

// StdlibRegistry contains all standard library functions organized by namespace.
// This registry is manually maintained and must be kept in sync with the actual
// type checker implementation in internal/compiler/typechecker/stdlib.go.
// TODO: Consider auto-generating this from the typechecker in a future iteration
// to reduce maintenance burden and eliminate drift possibility.
var StdlibRegistry = map[string][]FunctionDef{
	"String": {
		{
			Name:        "length",
			Signature:   "length(s: string!) -> int!",
			Description: "Returns the length of a string in characters",
		},
		{
			Name:        "slugify",
			Signature:   "slugify(s: string!) -> string!",
			Description: "Converts a string to a URL-friendly slug (lowercase, hyphens)",
		},
		{
			Name:        "upcase",
			Signature:   "upcase(s: string!) -> string!",
			Description: "Converts a string to uppercase",
		},
		{
			Name:        "downcase",
			Signature:   "downcase(s: string!) -> string!",
			Description: "Converts a string to lowercase",
		},
		{
			Name:        "trim",
			Signature:   "trim(s: string!) -> string!",
			Description: "Removes leading and trailing whitespace from a string",
		},
		{
			Name:        "contains",
			Signature:   "contains(s: string!, substr: string!) -> bool!",
			Description: "Checks if a string contains a substring",
		},
		{
			Name:        "replace",
			Signature:   "replace(s: string!, old: string!, new: string!) -> string!",
			Description: "Replaces all occurrences of old with new in the string",
		},
	},
	"Time": {
		{
			Name:        "now",
			Signature:   "now() -> timestamp!",
			Description: "Returns the current timestamp",
		},
		{
			Name:        "format",
			Signature:   "format(t: timestamp!, layout: string!) -> string!",
			Description: "Formats a timestamp as a string using the specified layout",
		},
		{
			Name:        "parse",
			Signature:   "parse(s: string!, layout: string!) -> timestamp?",
			Description: "Parses a string as a timestamp using the specified layout (returns null on error)",
		},
		{
			Name:        "add_days",
			Signature:   "add_days(t: timestamp!, days: int!) -> timestamp!",
			Description: "Adds the specified number of days to a timestamp",
		},
	},
	"Array": {
		{
			Name:        "length",
			Signature:   "length(arr: array!) -> int!",
			Description: "Returns the number of elements in an array",
		},
		{
			Name:        "contains",
			Signature:   "contains(arr: array!, value: any!) -> bool!",
			Description: "Checks if an array contains a specific value",
		},
	},
	"Hash": {
		{
			Name:        "has_key",
			Signature:   "has_key(h: hash!, key: any!) -> bool!",
			Description: "Checks if a hash contains a specific key",
		},
	},
	"UUID": {
		{
			Name:        "generate",
			Signature:   "generate() -> uuid!",
			Description: "Generates a new random UUID (v4)",
		},
	},
}

// GetNamespaces returns a sorted list of all available namespaces
func GetNamespaces() []string {
	namespaces := make([]string, 0, len(StdlibRegistry))
	for namespace := range StdlibRegistry {
		namespaces = append(namespaces, namespace)
	}
	// Sort for consistent output
	sort.Strings(namespaces)
	return namespaces
}

// GetFunctions returns all functions for a given namespace
// Returns nil if the namespace doesn't exist
func GetFunctions(namespace string) []FunctionDef {
	return StdlibRegistry[namespace]
}

// GetAllFunctions returns all functions across all namespaces
func GetAllFunctions() map[string][]FunctionDef {
	return StdlibRegistry
}

// TotalFunctionCount returns the total number of functions across all namespaces
func TotalFunctionCount() int {
	total := 0
	for _, funcs := range StdlibRegistry {
		total += len(funcs)
	}
	return total
}
