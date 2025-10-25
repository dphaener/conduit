package cache

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// CacheMiddlewareConfig holds configuration for the cache middleware
type CacheMiddlewareConfig struct {
	// Cache is the cache backend to use
	Cache Cache
	// KeyGenerator generates cache keys from requests
	KeyGenerator *KeyGenerator
	// TTL is the time-to-live for cached responses
	TTL time.Duration
	// OnlyMethods specifies which HTTP methods to cache (defaults to GET)
	OnlyMethods []string
	// SkipPaths is a list of paths to skip caching
	SkipPaths []string
	// CacheControl is the Cache-Control header to set on cached responses
	CacheControl string
}

// DefaultCacheMiddlewareConfig returns a default cache middleware configuration
func DefaultCacheMiddlewareConfig(cache Cache) CacheMiddlewareConfig {
	return CacheMiddlewareConfig{
		Cache:        cache,
		KeyGenerator: DefaultKeyGenerator(),
		TTL:          5 * time.Minute,
		OnlyMethods:  []string{http.MethodGet},
		SkipPaths:    []string{},
		CacheControl: "public, max-age=300",
	}
}

// CacheMiddleware creates a cache middleware with the given configuration
func CacheMiddleware(config CacheMiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if method not in OnlyMethods
			if !contains(config.OnlyMethods, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			// Skip if path in SkipPaths
			for _, skipPath := range config.SkipPaths {
				if r.URL.Path == skipPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Generate cache key
			cacheKey := config.KeyGenerator.GenerateKey(r)

			// Try to get from cache
			// Note: There is a small time window between cache lookup and response serving
			// where the cache might be invalidated. The cached response will still be served
			// from memory in this case. For strict consistency requirements, use cache
			// versioning or implement application-level validation.
			ctx := r.Context()
			cachedData, err := config.Cache.Get(ctx, cacheKey)
			if err == nil {
				// Cache hit - deserialize and send response
				var cached cachedResponse
				if err := json.Unmarshal(cachedData, &cached); err == nil {
					// Check conditional request
					if CheckConditionalRequest(w, r, cached.ETag, cached.LastModified) {
						return
					}

					// Set headers
					for key, values := range cached.Headers {
						for _, value := range values {
							w.Header().Add(key, value)
						}
					}

					// Set cache headers
					SetCacheHeaders(w, cached.ETag, cached.LastModified, config.CacheControl)
					w.Header().Set("X-Cache", "HIT")

					// Write status and body
					w.WriteHeader(cached.StatusCode)
					w.Write(cached.Body)
					return
				}
			}

			// Cache miss - record response
			recorder := newResponseRecorder(w)
			next.ServeHTTP(recorder, r)

			// Only cache successful responses
			if recorder.statusCode >= 200 && recorder.statusCode < 300 {
				// Generate ETag from response body
				etag := GenerateETag(recorder.body.Bytes())
				lastModified := time.Now()

				// Create cached response
				cached := cachedResponse{
					StatusCode:   recorder.statusCode,
					Headers:      recorder.Header().Clone(),
					Body:         recorder.body.Bytes(),
					ETag:         etag,
					LastModified: lastModified,
				}

				// Serialize and store in cache
				if data, err := json.Marshal(cached); err == nil {
					config.Cache.Set(ctx, cacheKey, data, config.TTL)
				}

				// Set cache headers on original response
				SetCacheHeaders(recorder.ResponseWriter, etag, lastModified, config.CacheControl)
			}

			recorder.ResponseWriter.Header().Set("X-Cache", "MISS")
		})
	}
}

// cachedResponse represents a cached HTTP response
type cachedResponse struct {
	StatusCode   int
	Headers      http.Header
	Body         []byte
	ETag         string
	LastModified time.Time
}

// responseRecorder records an HTTP response for caching
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	body        *bytes.Buffer
	wroteHeader bool
}

// newResponseRecorder creates a new response recorder
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           new(bytes.Buffer),
	}
}

// WriteHeader records the status code
func (r *responseRecorder) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		r.statusCode = statusCode
		r.wroteHeader = true
	}
}

// Write records the response body and writes to the underlying writer
func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.ResponseWriter.WriteHeader(r.statusCode)
		r.wroteHeader = true
	}

	// Write to buffer for caching
	r.body.Write(b)

	// Write to underlying response writer
	return r.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker interface
func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not support hijacking")
}

// Flush implements http.Flusher interface
func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// CacheInvalidator provides methods to invalidate cache entries
type CacheInvalidator struct {
	cache        Cache
	keyGenerator *KeyGenerator
}

// NewCacheInvalidator creates a new cache invalidator
func NewCacheInvalidator(cache Cache) *CacheInvalidator {
	return &CacheInvalidator{
		cache:        cache,
		keyGenerator: DefaultKeyGenerator(),
	}
}

// NewCacheInvalidatorWithKeyGen creates a new cache invalidator with a custom key generator
func NewCacheInvalidatorWithKeyGen(cache Cache, keyGenerator *KeyGenerator) *CacheInvalidator {
	return &CacheInvalidator{
		cache:        cache,
		keyGenerator: keyGenerator,
	}
}

// InvalidatePath invalidates cache entries for a specific path
func (ci *CacheInvalidator) InvalidatePath(ctx context.Context, method, path string) error {
	key := GenerateKeySimple(method, path)
	return ci.cache.Delete(ctx, key)
}

// InvalidateRequest invalidates cache entries for a specific request
func (ci *CacheInvalidator) InvalidateRequest(ctx context.Context, r *http.Request) error {
	key := ci.keyGenerator.GenerateKey(r)
	return ci.cache.Delete(ctx, key)
}

// InvalidateAll clears all cache entries
func (ci *CacheInvalidator) InvalidateAll(ctx context.Context) error {
	return ci.cache.Clear(ctx)
}
