package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyGenerator(t *testing.T) {
	kg := DefaultKeyGenerator()
	assert.NotNil(t, kg)
	assert.False(t, kg.IncludeHost)
	assert.True(t, kg.IncludeQuery)
	assert.NotEmpty(t, kg.IncludeHeaders)
	assert.NotEmpty(t, kg.Prefix)
}

func TestKeyGenerator_GenerateKey(t *testing.T) {
	kg := DefaultKeyGenerator()

	r := httptest.NewRequest("GET", "/path", nil)
	key := kg.GenerateKey(r)

	assert.NotEmpty(t, key)
	assert.Contains(t, key, kg.Prefix)

	// Same request should produce same key
	key2 := kg.GenerateKey(r)
	assert.Equal(t, key, key2)
}

func TestKeyGenerator_GenerateKey_WithQuery(t *testing.T) {
	kg := DefaultKeyGenerator()

	r1 := httptest.NewRequest("GET", "/path?foo=bar", nil)
	r2 := httptest.NewRequest("GET", "/path?foo=baz", nil)

	key1 := kg.GenerateKey(r1)
	key2 := kg.GenerateKey(r2)

	// Different query should produce different keys
	assert.NotEqual(t, key1, key2)
}

func TestKeyGenerator_GenerateKey_QueryOrder(t *testing.T) {
	kg := DefaultKeyGenerator()

	r1 := httptest.NewRequest("GET", "/path?foo=bar&baz=qux", nil)
	r2 := httptest.NewRequest("GET", "/path?baz=qux&foo=bar", nil)

	key1 := kg.GenerateKey(r1)
	key2 := kg.GenerateKey(r2)

	// Same query parameters in different order should produce same key
	assert.Equal(t, key1, key2)
}

func TestKeyGenerator_GenerateKey_WithHost(t *testing.T) {
	kg := &KeyGenerator{
		IncludeHost:    true,
		IncludeQuery:   true,
		IncludeHeaders: []string{},
		Prefix:         "test:",
	}

	r1 := httptest.NewRequest("GET", "http://example.com/path", nil)
	r2 := httptest.NewRequest("GET", "http://other.com/path", nil)

	key1 := kg.GenerateKey(r1)
	key2 := kg.GenerateKey(r2)

	// Different hosts should produce different keys
	assert.NotEqual(t, key1, key2)
}

func TestKeyGenerator_GenerateKey_WithHeaders(t *testing.T) {
	kg := &KeyGenerator{
		IncludeHost:    false,
		IncludeQuery:   true,
		IncludeHeaders: []string{"Accept", "Accept-Language"},
		Prefix:         "test:",
	}

	r1 := httptest.NewRequest("GET", "/path", nil)
	r1.Header.Set("Accept", "application/json")

	r2 := httptest.NewRequest("GET", "/path", nil)
	r2.Header.Set("Accept", "text/html")

	key1 := kg.GenerateKey(r1)
	key2 := kg.GenerateKey(r2)

	// Different headers should produce different keys
	assert.NotEqual(t, key1, key2)
}

func TestKeyGenerator_GenerateKey_IgnoreQuery(t *testing.T) {
	kg := &KeyGenerator{
		IncludeHost:    false,
		IncludeQuery:   false,
		IncludeHeaders: []string{},
		Prefix:         "test:",
	}

	r1 := httptest.NewRequest("GET", "/path?foo=bar", nil)
	r2 := httptest.NewRequest("GET", "/path?foo=baz", nil)

	key1 := kg.GenerateKey(r1)
	key2 := kg.GenerateKey(r2)

	// Different query should produce same key when IncludeQuery is false
	assert.Equal(t, key1, key2)
}

func TestKeyGenerator_GenerateKey_DifferentMethods(t *testing.T) {
	kg := DefaultKeyGenerator()

	r1 := httptest.NewRequest("GET", "/path", nil)
	r2 := httptest.NewRequest("POST", "/path", nil)

	key1 := kg.GenerateKey(r1)
	key2 := kg.GenerateKey(r2)

	// Different methods should produce different keys
	assert.NotEqual(t, key1, key2)
}

func TestKeyGenerator_GenerateKey_DifferentPaths(t *testing.T) {
	kg := DefaultKeyGenerator()

	r1 := httptest.NewRequest("GET", "/path1", nil)
	r2 := httptest.NewRequest("GET", "/path2", nil)

	key1 := kg.GenerateKey(r1)
	key2 := kg.GenerateKey(r2)

	// Different paths should produce different keys
	assert.NotEqual(t, key1, key2)
}

func TestGenerateKeySimple(t *testing.T) {
	key := GenerateKeySimple("GET", "/path")
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "http:")
	assert.Contains(t, key, "GET")
	assert.Contains(t, key, "/path")

	// Same method and path should produce same key
	key2 := GenerateKeySimple("GET", "/path")
	assert.Equal(t, key, key2)

	// Different method should produce different key
	key3 := GenerateKeySimple("POST", "/path")
	assert.NotEqual(t, key, key3)
}

func TestGenerateKeyWithQuery(t *testing.T) {
	key := GenerateKeyWithQuery("GET", "/path", "foo=bar")
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "http:")
	assert.Contains(t, key, "GET")
	assert.Contains(t, key, "/path")
	assert.Contains(t, key, "foo=bar")

	// Empty query should match simple key
	keyNoQuery := GenerateKeyWithQuery("GET", "/path", "")
	keySimple := GenerateKeySimple("GET", "/path")
	assert.Equal(t, keySimple, keyNoQuery)
}

func TestGenerateKeyFromRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "/path?foo=bar", nil)
	r.Header.Set("Accept", "application/json")

	key := GenerateKeyFromRequest(r)
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "http:")

	// Same request should produce same key
	key2 := GenerateKeyFromRequest(r)
	assert.Equal(t, key, key2)
}

func TestKeyGenerator_Prefix(t *testing.T) {
	kg := &KeyGenerator{
		IncludeHost:    false,
		IncludeQuery:   false,
		IncludeHeaders: []string{},
		Prefix:         "custom-prefix:",
	}

	r := httptest.NewRequest("GET", "/path", nil)
	key := kg.GenerateKey(r)

	assert.Contains(t, key, "custom-prefix:")
}

func TestKeyGenerator_MultipleQueryValues(t *testing.T) {
	kg := DefaultKeyGenerator()

	r, _ := http.NewRequest("GET", "/path?tag=a&tag=b&tag=c", nil)
	key := kg.GenerateKey(r)

	assert.NotEmpty(t, key)

	// Order within same parameter should be consistent
	r2, _ := http.NewRequest("GET", "/path?tag=c&tag=b&tag=a", nil)
	key2 := kg.GenerateKey(r2)

	// After sorting, should produce same key
	assert.Equal(t, key, key2)
}

func TestKeyGenerator_HeaderOrder(t *testing.T) {
	kg := &KeyGenerator{
		IncludeHost:    false,
		IncludeQuery:   false,
		IncludeHeaders: []string{"X-Custom", "Accept"},
		Prefix:         "test:",
	}

	r1 := httptest.NewRequest("GET", "/path", nil)
	r1.Header.Set("Accept", "text/html")
	r1.Header.Set("X-Custom", "value")

	r2 := httptest.NewRequest("GET", "/path", nil)
	r2.Header.Set("X-Custom", "value")
	r2.Header.Set("Accept", "text/html")

	key1 := kg.GenerateKey(r1)
	key2 := kg.GenerateKey(r2)

	// Same headers in different order should produce same key
	assert.Equal(t, key1, key2)
}
