package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateETag(t *testing.T) {
	content := []byte("test content")
	etag := GenerateETag(content)

	assert.NotEmpty(t, etag)
	assert.Contains(t, etag, `"`)

	// Same content should produce same ETag
	etag2 := GenerateETag(content)
	assert.Equal(t, etag, etag2)

	// Different content should produce different ETag
	etag3 := GenerateETag([]byte("different content"))
	assert.NotEqual(t, etag, etag3)
}

func TestGenerateWeakETag(t *testing.T) {
	content := []byte("test content")
	etag := GenerateWeakETag(content)

	assert.NotEmpty(t, etag)
	assert.Contains(t, etag, `W/"`)

	// Same content should produce same weak ETag
	etag2 := GenerateWeakETag(content)
	assert.Equal(t, etag, etag2)
}

func TestGenerateLastModified(t *testing.T) {
	now := time.Now()
	lastModified := GenerateLastModified(now)

	assert.NotEmpty(t, lastModified)

	// Parse back
	parsed, err := http.ParseTime(lastModified)
	require.NoError(t, err)

	// Should be equal when truncated to second precision
	assert.True(t, now.Truncate(time.Second).Equal(parsed.Truncate(time.Second)))
}

func TestParseIfNoneMatch(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected []string
	}{
		{
			name:     "empty header",
			header:   "",
			expected: nil,
		},
		{
			name:     "single etag",
			header:   `"abc123"`,
			expected: []string{`"abc123"`},
		},
		{
			name:     "multiple etags",
			header:   `"abc123", "def456"`,
			expected: []string{`"abc123"`, `"def456"`},
		},
		{
			name:     "weak etag",
			header:   `W/"abc123"`,
			expected: []string{`W/"abc123"`},
		},
		{
			name:     "mixed etags",
			header:   `"abc123", W/"def456"`,
			expected: []string{`"abc123"`, `W/"def456"`},
		},
		{
			name:     "wildcard",
			header:   "*",
			expected: []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseIfNoneMatch(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseIfModifiedSince(t *testing.T) {
	now := time.Now().UTC()
	formatted := now.Format(http.TimeFormat)

	parsed, err := ParseIfModifiedSince(formatted)
	require.NoError(t, err)
	assert.True(t, now.Truncate(time.Second).Equal(parsed.Truncate(time.Second)))

	// Test empty header
	_, err = ParseIfModifiedSince("")
	assert.Error(t, err)

	// Test invalid format
	_, err = ParseIfModifiedSince("invalid")
	assert.Error(t, err)
}

func TestMatchesETag(t *testing.T) {
	tests := []struct {
		name     string
		etag     string
		etags    []string
		expected bool
	}{
		{
			name:     "exact match",
			etag:     `"abc123"`,
			etags:    []string{`"abc123"`},
			expected: true,
		},
		{
			name:     "no match",
			etag:     `"abc123"`,
			etags:    []string{`"def456"`},
			expected: false,
		},
		{
			name:     "match in list",
			etag:     `"abc123"`,
			etags:    []string{`"def456"`, `"abc123"`, `"ghi789"`},
			expected: true,
		},
		{
			name:     "wildcard match",
			etag:     `"abc123"`,
			etags:    []string{"*"},
			expected: true,
		},
		{
			name:     "weak etag match",
			etag:     `W/"abc123"`,
			etags:    []string{`W/"abc123"`},
			expected: true,
		},
		{
			name:     "weak vs strong match",
			etag:     `"abc123"`,
			etags:    []string{`W/"abc123"`},
			expected: true,
		},
		{
			name:     "empty list",
			etag:     `"abc123"`,
			etags:    []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesETag(tt.etag, tt.etags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckConditionalRequest_IfNoneMatch(t *testing.T) {
	etag := `"abc123"`
	lastModified := time.Now()

	tests := []struct {
		name           string
		ifNoneMatch    string
		expectedStatus int
		expectedResult bool
	}{
		{
			name:           "matching etag",
			ifNoneMatch:    `"abc123"`,
			expectedStatus: http.StatusNotModified,
			expectedResult: true,
		},
		{
			name:           "non-matching etag",
			ifNoneMatch:    `"def456"`,
			expectedStatus: 0,
			expectedResult: false,
		},
		{
			name:           "wildcard",
			ifNoneMatch:    "*",
			expectedStatus: http.StatusNotModified,
			expectedResult: true,
		},
		{
			name:           "weak etag match",
			ifNoneMatch:    `W/"abc123"`,
			expectedStatus: http.StatusNotModified,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			r.Header.Set("If-None-Match", tt.ifNoneMatch)

			result := CheckConditionalRequest(w, r, etag, lastModified)
			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedResult {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestCheckConditionalRequest_IfModifiedSince(t *testing.T) {
	// Test with empty ETag so If-Modified-Since is checked
	// Use UTC times to avoid timezone issues with HTTP date parsing
	etag := ""
	lastModified := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)

	tests := []struct {
		name            string
		ifModifiedSince time.Time
		expectedStatus  int
		expectedResult  bool
	}{
		{
			name:            "not modified - client has newer timestamp",
			ifModifiedSince: lastModified.Add(30 * time.Minute),
			expectedStatus:  http.StatusNotModified,
			expectedResult:  true,
		},
		{
			name:            "not modified - exact match",
			ifModifiedSince: lastModified,
			expectedStatus:  http.StatusNotModified,
			expectedResult:  true,
		},
		{
			name:            "modified - client has older timestamp",
			ifModifiedSince: lastModified.Add(-30 * time.Minute),
			expectedStatus:  0,
			expectedResult:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			r.Header.Set("If-Modified-Since", tt.ifModifiedSince.Format(http.TimeFormat))

			result := CheckConditionalRequest(w, r, etag, lastModified)
			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedResult {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestCheckConditionalRequest_Priority(t *testing.T) {
	// If-None-Match takes precedence over If-Modified-Since
	etag := `"abc123"`
	lastModified := time.Now().Add(-1 * time.Hour)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("If-None-Match", `"def456"`)                                              // Doesn't match
	r.Header.Set("If-Modified-Since", time.Now().Add(1*time.Hour).Format(http.TimeFormat)) // Would match

	result := CheckConditionalRequest(w, r, etag, lastModified)
	assert.False(t, result) // Should be false because If-None-Match doesn't match
}

func TestSetCacheHeaders(t *testing.T) {
	etag := `"abc123"`
	lastModified := time.Now()
	cacheControl := "public, max-age=300"

	w := httptest.NewRecorder()
	SetCacheHeaders(w, etag, lastModified, cacheControl)

	assert.Equal(t, etag, w.Header().Get("ETag"))
	assert.NotEmpty(t, w.Header().Get("Last-Modified"))
	assert.Equal(t, cacheControl, w.Header().Get("Cache-Control"))
}

func TestSetCacheHeaders_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	SetCacheHeaders(w, "", time.Time{}, "")

	assert.Empty(t, w.Header().Get("ETag"))
	assert.Empty(t, w.Header().Get("Last-Modified"))
	assert.Empty(t, w.Header().Get("Cache-Control"))
}
