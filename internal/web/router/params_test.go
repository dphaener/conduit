package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParamExtractor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	extractor := NewParamExtractor(req)

	assert.NotNil(t, extractor)
	assert.Equal(t, req, extractor.req)
}

func TestPathParam(t *testing.T) {
	router := chi.NewRouter()
	var extractedID string

	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := NewParamExtractor(r)
		extractedID = params.PathParam("id")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "123", extractedID)
}

func TestPathParamUUID(t *testing.T) {
	router := chi.NewRouter()
	validUUID := uuid.New()
	var extractedUUID uuid.UUID
	var extractedErr error

	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := NewParamExtractor(r)
		extractedUUID, extractedErr = params.PathParamUUID("id")
		w.WriteHeader(http.StatusOK)
	})

	// Test valid UUID
	req := httptest.NewRequest(http.MethodGet, "/posts/"+validUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.NoError(t, extractedErr)
	assert.Equal(t, validUUID, extractedUUID)

	// Test invalid UUID
	req = httptest.NewRequest(http.MethodGet, "/posts/invalid-uuid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Error(t, extractedErr)
	assert.Contains(t, extractedErr.Error(), "invalid UUID")
}

func TestPathParamInt(t *testing.T) {
	router := chi.NewRouter()
	var extractedInt int
	var extractedErr error

	router.Get("/posts/{page}", func(w http.ResponseWriter, r *http.Request) {
		params := NewParamExtractor(r)
		extractedInt, extractedErr = params.PathParamInt("page")
		w.WriteHeader(http.StatusOK)
	})

	// Test valid integer
	req := httptest.NewRequest(http.MethodGet, "/posts/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.NoError(t, extractedErr)
	assert.Equal(t, 42, extractedInt)

	// Test invalid integer
	req = httptest.NewRequest(http.MethodGet, "/posts/invalid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Error(t, extractedErr)
	assert.Contains(t, extractedErr.Error(), "invalid integer")
}

func TestPathParamInt64(t *testing.T) {
	router := chi.NewRouter()
	var extractedInt64 int64
	var extractedErr error

	router.Get("/posts/{count}", func(w http.ResponseWriter, r *http.Request) {
		params := NewParamExtractor(r)
		extractedInt64, extractedErr = params.PathParamInt64("count")
		w.WriteHeader(http.StatusOK)
	})

	// Test valid int64
	req := httptest.NewRequest(http.MethodGet, "/posts/9223372036854775807", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.NoError(t, extractedErr)
	assert.Equal(t, int64(9223372036854775807), extractedInt64)

	// Test invalid int64
	req = httptest.NewRequest(http.MethodGet, "/posts/invalid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Error(t, extractedErr)
	assert.Contains(t, extractedErr.Error(), "invalid int64")
}

func TestQueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts?search=golang", nil)
	params := NewParamExtractor(req)

	result := params.QueryParam("search")
	assert.Equal(t, "golang", result)

	// Test non-existent parameter
	result = params.QueryParam("nonexistent")
	assert.Equal(t, "", result)
}

func TestQueryParamWithDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts?status=published", nil)
	params := NewParamExtractor(req)

	// Test with existing parameter
	result := params.QueryParamWithDefault("status", "draft")
	assert.Equal(t, "published", result)

	// Test with non-existent parameter
	result = params.QueryParamWithDefault("category", "uncategorized")
	assert.Equal(t, "uncategorized", result)
}

func TestQueryParamInt(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		param        string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			url:          "/posts?page=5",
			param:        "page",
			defaultValue: 1,
			expected:     5,
		},
		{
			name:         "missing parameter",
			url:          "/posts",
			param:        "page",
			defaultValue: 1,
			expected:     1,
		},
		{
			name:         "invalid integer",
			url:          "/posts?page=invalid",
			param:        "page",
			defaultValue: 1,
			expected:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			params := NewParamExtractor(req)

			result := params.QueryParamInt(tt.param, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryParamInt64(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts?offset=1000000", nil)
	params := NewParamExtractor(req)

	result := params.QueryParamInt64("offset", 0)
	assert.Equal(t, int64(1000000), result)

	// Test default
	result = params.QueryParamInt64("nonexistent", 42)
	assert.Equal(t, int64(42), result)
}

func TestQueryParamBool(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		param        string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "true value",
			url:          "/posts?published=true",
			param:        "published",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "false value",
			url:          "/posts?published=false",
			param:        "published",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "1 as true",
			url:          "/posts?published=1",
			param:        "published",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "0 as false",
			url:          "/posts?published=0",
			param:        "published",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "missing parameter",
			url:          "/posts",
			param:        "published",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "invalid value",
			url:          "/posts?published=invalid",
			param:        "published",
			defaultValue: false,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			params := NewParamExtractor(req)

			result := params.QueryParamBool(tt.param, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryParamArray(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts?tags=golang&tags=web&tags=api", nil)
	params := NewParamExtractor(req)

	result := params.QueryParamArray("tags")
	assert.Equal(t, []string{"golang", "web", "api"}, result)

	// Test non-existent parameter
	result = params.QueryParamArray("nonexistent")
	assert.Empty(t, result)
}

func TestHeaderParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	req.Header.Set("Authorization", "Bearer token123")
	params := NewParamExtractor(req)

	result := params.HeaderParam("Authorization")
	assert.Equal(t, "Bearer token123", result)

	// Test non-existent header
	result = params.HeaderParam("X-Custom-Header")
	assert.Equal(t, "", result)
}

func TestHeaderParamWithDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	req.Header.Set("Content-Type", "application/json")
	params := NewParamExtractor(req)

	// Test with existing header
	result := params.HeaderParamWithDefault("Content-Type", "text/plain")
	assert.Equal(t, "application/json", result)

	// Test with non-existent header
	result = params.HeaderParamWithDefault("Accept", "application/json")
	assert.Equal(t, "application/json", result)
}

func TestExtractPagination(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		defaultPerPage  int
		maxPerPage      int
		expectedPage    int
		expectedPerPage int
		expectedOffset  int
	}{
		{
			name:            "default values",
			url:             "/posts",
			defaultPerPage:  50,
			maxPerPage:      100,
			expectedPage:    1,
			expectedPerPage: 50,
			expectedOffset:  0,
		},
		{
			name:            "custom page and per_page",
			url:             "/posts?page=3&per_page=20",
			defaultPerPage:  50,
			maxPerPage:      100,
			expectedPage:    3,
			expectedPerPage: 20,
			expectedOffset:  40,
		},
		{
			name:            "exceeds max per_page",
			url:             "/posts?per_page=200",
			defaultPerPage:  50,
			maxPerPage:      100,
			expectedPage:    1,
			expectedPerPage: 100,
			expectedOffset:  0,
		},
		{
			name:            "negative page defaults to 1",
			url:             "/posts?page=-1",
			defaultPerPage:  50,
			maxPerPage:      100,
			expectedPage:    1,
			expectedPerPage: 50,
			expectedOffset:  0,
		},
		{
			name:            "zero per_page uses default",
			url:             "/posts?per_page=0",
			defaultPerPage:  50,
			maxPerPage:      100,
			expectedPage:    1,
			expectedPerPage: 50,
			expectedOffset:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			params := NewParamExtractor(req)

			result := params.ExtractPagination(tt.defaultPerPage, tt.maxPerPage)
			assert.Equal(t, tt.expectedPage, result.Page)
			assert.Equal(t, tt.expectedPerPage, result.PerPage)
			assert.Equal(t, tt.expectedOffset, result.Offset)
		})
	}
}

func TestExtractSort(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		defaultField  string
		defaultOrder  string
		allowedFields []string
		expectedField string
		expectedOrder string
	}{
		{
			name:          "default values",
			url:           "/posts",
			defaultField:  "created_at",
			defaultOrder:  "desc",
			allowedFields: []string{"created_at", "updated_at", "title"},
			expectedField: "created_at",
			expectedOrder: "desc",
		},
		{
			name:          "custom sort and order",
			url:           "/posts?sort=title&order=asc",
			defaultField:  "created_at",
			defaultOrder:  "desc",
			allowedFields: []string{"created_at", "updated_at", "title"},
			expectedField: "title",
			expectedOrder: "asc",
		},
		{
			name:          "invalid field falls back to default",
			url:           "/posts?sort=invalid&order=asc",
			defaultField:  "created_at",
			defaultOrder:  "desc",
			allowedFields: []string{"created_at", "updated_at", "title"},
			expectedField: "created_at",
			expectedOrder: "asc",
		},
		{
			name:          "invalid order falls back to default",
			url:           "/posts?sort=title&order=invalid",
			defaultField:  "created_at",
			defaultOrder:  "desc",
			allowedFields: []string{"created_at", "updated_at", "title"},
			expectedField: "title",
			expectedOrder: "desc",
		},
		{
			name:          "no allowed fields restriction",
			url:           "/posts?sort=anything&order=asc",
			defaultField:  "created_at",
			defaultOrder:  "desc",
			allowedFields: nil,
			expectedField: "anything",
			expectedOrder: "asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			params := NewParamExtractor(req)

			result := params.ExtractSort(tt.defaultField, tt.defaultOrder, tt.allowedFields)
			assert.Equal(t, tt.expectedField, result.Field)
			assert.Equal(t, tt.expectedOrder, result.Order)
		})
	}
}

func TestExtractFilters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts?status=published&author=john&page=1", nil)
	params := NewParamExtractor(req)

	allowedFilters := []string{"status", "author", "category"}
	result := params.ExtractFilters(allowedFilters)

	assert.Equal(t, "published", result["status"])
	assert.Equal(t, "john", result["author"])
	assert.NotContains(t, result, "page")     // page is not in allowed filters
	assert.NotContains(t, result, "category") // category not provided
}

func TestGetPathParam(t *testing.T) {
	router := chi.NewRouter()
	var extractedID string

	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		extractedID = GetPathParam(r, "id")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "abc123", extractedID)
}

func TestGetPathParamUUID(t *testing.T) {
	router := chi.NewRouter()
	validUUID := uuid.New()
	var extractedUUID uuid.UUID
	var extractedErr error

	router.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		extractedUUID, extractedErr = GetPathParamUUID(r, "id")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/"+validUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.NoError(t, extractedErr)
	assert.Equal(t, validUUID, extractedUUID)
}

func TestGetQueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts?search=golang", nil)

	result := GetQueryParam(req, "search", "")
	assert.Equal(t, "golang", result)

	result = GetQueryParam(req, "nonexistent", "default")
	assert.Equal(t, "default", result)
}

func TestGetQueryParamInt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts?limit=25", nil)

	result := GetQueryParamInt(req, "limit", 10)
	assert.Equal(t, 25, result)

	result = GetQueryParamInt(req, "nonexistent", 10)
	assert.Equal(t, 10, result)
}
