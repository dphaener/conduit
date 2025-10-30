package query

import (
	"net/http"
	"regexp"
	"strings"
)

// fieldsPattern matches query parameters like fields[typename]
var fieldsPattern = regexp.MustCompile(`^fields\[([^\]]+)\]$`)

// filterPattern matches query parameters like filter[key]
var filterPattern = regexp.MustCompile(`^filter\[([^\]]+)\]$`)

// ParseInclude parses the include query parameter into a slice of relationship names.
// Example: ?include=author,comments returns ["author", "comments"]
// Returns an empty slice if the include parameter is not present.
func ParseInclude(r *http.Request) []string {
	include := r.URL.Query().Get("include")
	if include == "" {
		return []string{}
	}

	parts := strings.Split(include, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// ParseFields parses the fields query parameters into a map of resource types to field names.
// Example: ?fields[users]=name,email&fields[posts]=title
// Returns: {"users": ["name", "email"], "posts": ["title"]}
// Returns an empty map if no fields parameters are present.
func ParseFields(r *http.Request) map[string][]string {
	result := make(map[string][]string)

	for key, values := range r.URL.Query() {
		matches := fieldsPattern.FindStringSubmatch(key)
		if len(matches) != 2 {
			continue
		}

		typeName := matches[1]
		if len(values) == 0 || values[0] == "" {
			result[typeName] = []string{}
			continue
		}

		fields := strings.Split(values[0], ",")
		fieldList := make([]string, 0, len(fields))
		for _, field := range fields {
			trimmed := strings.TrimSpace(field)
			if trimmed != "" {
				fieldList = append(fieldList, trimmed)
			}
		}
		result[typeName] = fieldList
	}

	return result
}

// ParseFilter parses the filter query parameters into a map of filter keys to values.
// Example: ?filter[status]=published&filter[author_id]=123
// Returns: {"status": "published", "author_id": "123"}
// Returns an empty map if no filter parameters are present.
func ParseFilter(r *http.Request) map[string]string {
	result := make(map[string]string)

	for key, values := range r.URL.Query() {
		matches := filterPattern.FindStringSubmatch(key)
		if len(matches) != 2 {
			continue
		}

		filterKey := matches[1]
		if len(values) > 0 {
			result[filterKey] = values[0]
		}
	}

	return result
}

// ParseSort parses the sort query parameter into a slice of sort fields.
// Example: ?sort=-created_at,title returns ["-created_at", "title"]
// The "-" prefix indicates descending sort order.
// Returns an empty slice if the sort parameter is not present.
func ParseSort(r *http.Request) []string {
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		return []string{}
	}

	parts := strings.Split(sort, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
