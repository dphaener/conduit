package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// KeyGenerator generates cache keys from HTTP requests
type KeyGenerator struct {
	// IncludeHost includes the request host in the cache key
	IncludeHost bool
	// IncludeQuery includes query parameters in the cache key
	IncludeQuery bool
	// IncludeHeaders includes specified headers in the cache key
	IncludeHeaders []string
	// Prefix is prepended to all cache keys
	Prefix string
}

// DefaultKeyGenerator returns a default key generator
func DefaultKeyGenerator() *KeyGenerator {
	return &KeyGenerator{
		IncludeHost:    false,
		IncludeQuery:   true,
		IncludeHeaders: []string{"Accept", "Accept-Encoding"},
		Prefix:         "http:",
	}
}

// GenerateKey generates a cache key for the given request
func (kg *KeyGenerator) GenerateKey(r *http.Request) string {
	var parts []string

	// Add host if configured
	if kg.IncludeHost {
		parts = append(parts, r.Host)
	}

	// Add method and path
	parts = append(parts, r.Method)
	parts = append(parts, r.URL.Path)

	// Add query if configured
	if kg.IncludeQuery && r.URL.RawQuery != "" {
		// Sort query parameters for consistent keys
		query := r.URL.Query()
		var queryParts []string
		for key, values := range query {
			sort.Strings(values)
			for _, value := range values {
				queryParts = append(queryParts, fmt.Sprintf("%s=%s", key, value))
			}
		}
		sort.Strings(queryParts)
		parts = append(parts, strings.Join(queryParts, "&"))
	}

	// Add headers if configured
	if len(kg.IncludeHeaders) > 0 {
		var headerParts []string
		for _, header := range kg.IncludeHeaders {
			value := r.Header.Get(header)
			if value != "" {
				headerParts = append(headerParts, fmt.Sprintf("%s=%s", header, value))
			}
		}
		if len(headerParts) > 0 {
			sort.Strings(headerParts)
			parts = append(parts, strings.Join(headerParts, "|"))
		}
	}

	// Join parts and hash for a shorter key
	key := strings.Join(parts, ":")
	hash := sha256.Sum256([]byte(key))
	// Truncate to 16 bytes for shorter cache keys (still 128-bit security)
	return kg.Prefix + hex.EncodeToString(hash[:16])
}

// GenerateKeySimple generates a simple cache key from method and path
func GenerateKeySimple(method, path string) string {
	return fmt.Sprintf("http:%s:%s", method, path)
}

// GenerateKeyWithQuery generates a cache key from method, path, and query
func GenerateKeyWithQuery(method, path, query string) string {
	if query == "" {
		return GenerateKeySimple(method, path)
	}
	return fmt.Sprintf("http:%s:%s?%s", method, path, query)
}

// GenerateKeyFromRequest generates a cache key from an HTTP request using default settings
func GenerateKeyFromRequest(r *http.Request) string {
	kg := DefaultKeyGenerator()
	return kg.GenerateKey(r)
}
