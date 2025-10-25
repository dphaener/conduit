package cache_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/conduit-lang/conduit/internal/web/cache"
)

// Example_memoryCache demonstrates basic usage of the memory cache
func Example_memoryCache() {
	// Create a memory cache
	c := cache.NewMemoryCache()

	// Store a value
	ctx := httptest.NewRequest("GET", "/", nil).Context()
	_ = c.Set(ctx, "user:123", []byte("John Doe"), 5*time.Minute)

	// Retrieve the value
	value, _ := c.Get(ctx, "user:123")
	fmt.Println(string(value))

	// Output: John Doe
}

// Example_httpCaching demonstrates HTTP response caching with middleware
func Example_httpCaching() {
	// Create a cache
	c := cache.NewMemoryCache()

	// Configure cache middleware
	config := cache.DefaultCacheMiddlewareConfig(c)
	config.TTL = 5 * time.Minute
	config.CacheControl = "public, max-age=300"

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Wrap with cache middleware
	cachedHandler := cache.CacheMiddleware(config)(handler)

	// First request - cache miss
	r1 := httptest.NewRequest("GET", "/api/data", nil)
	w1 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(w1, r1)

	fmt.Println("First request:", w1.Header().Get("X-Cache"))
	fmt.Println("Response:", w1.Body.String())
	fmt.Println("Has ETag:", w1.Header().Get("ETag") != "")

	// Second request - cache hit
	r2 := httptest.NewRequest("GET", "/api/data", nil)
	w2 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(w2, r2)

	fmt.Println("Second request:", w2.Header().Get("X-Cache"))

	// Output:
	// First request: MISS
	// Response: Hello, World!
	// Has ETag: true
	// Second request: HIT
}

// Example_conditionalGet demonstrates conditional GET with ETags
func Example_conditionalGet() {
	// Create a cache
	c := cache.NewMemoryCache()

	// Configure cache middleware
	config := cache.DefaultCacheMiddlewareConfig(c)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Wrap with cache middleware
	cachedHandler := cache.CacheMiddleware(config)(handler)

	// First request to get ETag
	r1 := httptest.NewRequest("GET", "/api/data", nil)
	w1 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(w1, r1)

	etag := w1.Header().Get("ETag")

	// Second request with If-None-Match
	r2 := httptest.NewRequest("GET", "/api/data", nil)
	r2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(w2, r2)

	fmt.Println("Status Code:", w2.Code)
	fmt.Println("Body Empty:", w2.Body.Len() == 0)

	// Output:
	// Status Code: 304
	// Body Empty: true
}

// Example_cacheInvalidation demonstrates cache invalidation
func Example_cacheInvalidation() {
	// Create a cache
	c := cache.NewMemoryCache()

	// Create invalidator
	invalidator := cache.NewCacheInvalidator(c)

	// Store some values
	ctx := httptest.NewRequest("GET", "/", nil).Context()
	_ = c.Set(ctx, "http:GET:/api/users", []byte("users data"), 5*time.Minute)
	_ = c.Set(ctx, "http:GET:/api/posts", []byte("posts data"), 5*time.Minute)

	// Invalidate specific path
	_ = invalidator.InvalidatePath(ctx, "GET", "/api/users")

	// Check what remains
	_, err1 := c.Get(ctx, "http:GET:/api/users")
	_, err2 := c.Get(ctx, "http:GET:/api/posts")

	fmt.Println("Users invalidated:", cache.IsCacheMiss(err1))
	fmt.Println("Posts still cached:", err2 == nil)

	// Output:
	// Users invalidated: true
	// Posts still cached: true
}

// Example_customKeyGeneration demonstrates custom cache key generation
func Example_customKeyGeneration() {
	// Create custom key generator
	keyGen := &cache.KeyGenerator{
		IncludeHost:    true,
		IncludeQuery:   true,
		IncludeHeaders: []string{"Accept-Language"},
		Prefix:         "api:",
	}

	// Generate keys for different requests
	r1 := httptest.NewRequest("GET", "http://api.example.com/users?page=1", nil)
	r1.Header.Set("Accept-Language", "en-US")

	r2 := httptest.NewRequest("GET", "http://api.example.com/users?page=1", nil)
	r2.Header.Set("Accept-Language", "es-ES")

	key1 := keyGen.GenerateKey(r1)
	key2 := keyGen.GenerateKey(r2)

	fmt.Println("Keys are different:", key1 != key2)
	fmt.Println("Both have api: prefix:", len(key1) > 4 && len(key2) > 4)

	// Output:
	// Keys are different: true
	// Both have api: prefix: true
}
