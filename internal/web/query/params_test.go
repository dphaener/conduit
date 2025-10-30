package query

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestParseInclude(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected []string
	}{
		{
			name:     "empty when not present",
			url:      "/api/posts",
			expected: []string{},
		},
		{
			name:     "single relationship",
			url:      "/api/posts?include=author",
			expected: []string{"author"},
		},
		{
			name:     "multiple relationships",
			url:      "/api/posts?include=author,comments",
			expected: []string{"author", "comments"},
		},
		{
			name:     "nested relationships",
			url:      "/api/posts?include=author,comments.author",
			expected: []string{"author", "comments.author"},
		},
		{
			name:     "trims whitespace",
			url:      "/api/posts?include=author,%20comments%20,%20tags",
			expected: []string{"author", "comments", "tags"},
		},
		{
			name:     "empty string parameter",
			url:      "/api/posts?include=",
			expected: []string{},
		},
		{
			name:     "multiple commas ignored",
			url:      "/api/posts?include=author,,comments",
			expected: []string{"author", "comments"},
		},
		{
			name:     "only whitespace ignored",
			url:      "/api/posts?include=%20,%20,%20author",
			expected: []string{"author"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			result := ParseInclude(req)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseInclude() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseFields(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected map[string][]string
	}{
		{
			name:     "empty when not present",
			url:      "/api/posts",
			expected: map[string][]string{},
		},
		{
			name: "single type with single field",
			url:  "/api/posts?fields[users]=name",
			expected: map[string][]string{
				"users": {"name"},
			},
		},
		{
			name: "single type with multiple fields",
			url:  "/api/posts?fields[users]=name,email",
			expected: map[string][]string{
				"users": {"name", "email"},
			},
		},
		{
			name: "multiple types",
			url:  "/api/posts?fields[users]=name,email&fields[posts]=title,body",
			expected: map[string][]string{
				"users": {"name", "email"},
				"posts": {"title", "body"},
			},
		},
		{
			name: "trims whitespace",
			url:  "/api/posts?fields[users]=name,%20email%20,%20bio",
			expected: map[string][]string{
				"users": {"name", "email", "bio"},
			},
		},
		{
			name: "empty string parameter",
			url:  "/api/posts?fields[users]=",
			expected: map[string][]string{
				"users": {},
			},
		},
		{
			name: "multiple commas ignored",
			url:  "/api/posts?fields[users]=name,,email",
			expected: map[string][]string{
				"users": {"name", "email"},
			},
		},
		{
			name: "complex type names",
			url:  "/api/posts?fields[blog-posts]=title&fields[comment_replies]=text",
			expected: map[string][]string{
				"blog-posts":      {"title"},
				"comment_replies": {"text"},
			},
		},
		{
			name:     "ignores malformed parameters",
			url:      "/api/posts?fields=invalid&fields[]=empty&fields[users]=name",
			expected: map[string][]string{
				"users": {"name"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			result := ParseFields(req)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseFields() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected map[string]string
	}{
		{
			name:     "empty when not present",
			url:      "/api/posts",
			expected: map[string]string{},
		},
		{
			name: "single filter",
			url:  "/api/posts?filter[status]=published",
			expected: map[string]string{
				"status": "published",
			},
		},
		{
			name: "multiple filters",
			url:  "/api/posts?filter[status]=published&filter[author_id]=123",
			expected: map[string]string{
				"status":    "published",
				"author_id": "123",
			},
		},
		{
			name: "empty value",
			url:  "/api/posts?filter[tag]=",
			expected: map[string]string{
				"tag": "",
			},
		},
		{
			name: "special characters in key",
			url:  "/api/posts?filter[created-at]=2024-01-01&filter[author_id]=123",
			expected: map[string]string{
				"created-at": "2024-01-01",
				"author_id":  "123",
			},
		},
		{
			name: "value with spaces",
			url:  "/api/posts?filter[title]=Hello%20World",
			expected: map[string]string{
				"title": "Hello World",
			},
		},
		{
			name: "value with special characters",
			url:  "/api/posts?filter[email]=user%40example.com",
			expected: map[string]string{
				"email": "user@example.com",
			},
		},
		{
			name:     "ignores malformed parameters",
			url:      "/api/posts?filter=invalid&filter[]=empty&filter[status]=active",
			expected: map[string]string{
				"status": "active",
			},
		},
		{
			name: "numeric values",
			url:  "/api/posts?filter[id]=42&filter[rating]=4.5",
			expected: map[string]string{
				"id":     "42",
				"rating": "4.5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			result := ParseFilter(req)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseSort(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected []string
	}{
		{
			name:     "empty when not present",
			url:      "/api/posts",
			expected: []string{},
		},
		{
			name:     "single field ascending",
			url:      "/api/posts?sort=title",
			expected: []string{"title"},
		},
		{
			name:     "single field descending",
			url:      "/api/posts?sort=-created_at",
			expected: []string{"-created_at"},
		},
		{
			name:     "multiple fields",
			url:      "/api/posts?sort=-created_at,title",
			expected: []string{"-created_at", "title"},
		},
		{
			name:     "mixed ascending and descending",
			url:      "/api/posts?sort=-priority,created_at,-updated_at",
			expected: []string{"-priority", "created_at", "-updated_at"},
		},
		{
			name:     "trims whitespace",
			url:      "/api/posts?sort=-created_at,%20title%20,%20-rating",
			expected: []string{"-created_at", "title", "-rating"},
		},
		{
			name:     "empty string parameter",
			url:      "/api/posts?sort=",
			expected: []string{},
		},
		{
			name:     "multiple commas ignored",
			url:      "/api/posts?sort=title,,-created_at",
			expected: []string{"title", "-created_at"},
		},
		{
			name:     "only whitespace ignored",
			url:      "/api/posts?sort=%20,%20,%20title",
			expected: []string{"title"},
		},
		{
			name:     "field names with underscores and hyphens",
			url:      "/api/posts?sort=-created_at,author-name,comment_count",
			expected: []string{"-created_at", "author-name", "comment_count"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			result := ParseSort(req)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseSort() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMultipleParameters tests parsing multiple different parameter types in one request
func TestMultipleParameters(t *testing.T) {
	url := "/api/posts?include=author,comments&fields[posts]=title,body&fields[users]=name&filter[status]=published&sort=-created_at,title"
	req := httptest.NewRequest(http.MethodGet, url, nil)

	include := ParseInclude(req)
	expectedInclude := []string{"author", "comments"}
	if !reflect.DeepEqual(include, expectedInclude) {
		t.Errorf("ParseInclude() = %v, want %v", include, expectedInclude)
	}

	fields := ParseFields(req)
	expectedFields := map[string][]string{
		"posts": {"title", "body"},
		"users": {"name"},
	}
	if !reflect.DeepEqual(fields, expectedFields) {
		t.Errorf("ParseFields() = %v, want %v", fields, expectedFields)
	}

	filter := ParseFilter(req)
	expectedFilter := map[string]string{
		"status": "published",
	}
	if !reflect.DeepEqual(filter, expectedFilter) {
		t.Errorf("ParseFilter() = %v, want %v", filter, expectedFilter)
	}

	sort := ParseSort(req)
	expectedSort := []string{"-created_at", "title"}
	if !reflect.DeepEqual(sort, expectedSort) {
		t.Errorf("ParseSort() = %v, want %v", sort, expectedSort)
	}
}
