package cache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCacheMiddlewareConfig(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)

	assert.NotNil(t, config.Cache)
	assert.NotNil(t, config.KeyGenerator)
	assert.NotZero(t, config.TTL)
	assert.NotEmpty(t, config.OnlyMethods)
	assert.NotEmpty(t, config.CacheControl)
}

func TestCacheMiddleware_CacheHit(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)

	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	middleware := CacheMiddleware(config)(handler)

	// First request - cache miss
	r1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, r1)

	assert.Equal(t, 1, callCount)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "response body", w1.Body.String())
	assert.Equal(t, "MISS", w1.Header().Get("X-Cache"))

	// Second request - cache hit
	r2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, r2)

	assert.Equal(t, 1, callCount) // Handler should not be called again
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "response body", w2.Body.String())
	assert.Equal(t, "HIT", w2.Header().Get("X-Cache"))
}

func TestCacheMiddleware_OnlyGET(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)

	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	middleware := CacheMiddleware(config)(handler)

	// POST request - should not cache
	r1 := httptest.NewRequest("POST", "/test", nil)
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, r1)

	assert.Equal(t, 1, callCount)
	assert.Empty(t, w1.Header().Get("X-Cache"))

	// Second POST request - should call handler again
	r2 := httptest.NewRequest("POST", "/test", nil)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, r2)

	assert.Equal(t, 2, callCount)
	assert.Empty(t, w2.Header().Get("X-Cache"))
}

func TestCacheMiddleware_SkipPaths(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)
	config.SkipPaths = []string{"/skip"}

	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	middleware := CacheMiddleware(config)(handler)

	// Request to skipped path
	r1 := httptest.NewRequest("GET", "/skip", nil)
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, r1)

	assert.Equal(t, 1, callCount)
	assert.Empty(t, w1.Header().Get("X-Cache"))

	// Second request to skipped path - should call handler again
	r2 := httptest.NewRequest("GET", "/skip", nil)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, r2)

	assert.Equal(t, 2, callCount)
	assert.Empty(t, w2.Header().Get("X-Cache"))
}

func TestCacheMiddleware_OnlySuccessfulResponses(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)

	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})

	middleware := CacheMiddleware(config)(handler)

	// First request - should not cache error response
	r1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, r1)

	assert.Equal(t, 1, callCount)

	// Second request - should call handler again
	r2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, r2)

	assert.Equal(t, 2, callCount)
}

func TestCacheMiddleware_ETagGeneration(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	middleware := CacheMiddleware(config)(handler)

	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, r)

	assert.NotEmpty(t, w.Header().Get("ETag"))
	assert.NotEmpty(t, w.Header().Get("Last-Modified"))
	assert.NotEmpty(t, w.Header().Get("Cache-Control"))
}

func TestCacheMiddleware_ConditionalRequest(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	middleware := CacheMiddleware(config)(handler)

	// First request to populate cache
	r1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, r1)

	etag := w1.Header().Get("ETag")
	assert.NotEmpty(t, etag)

	// Second request with If-None-Match
	r2 := httptest.NewRequest("GET", "/test", nil)
	r2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, r2)

	assert.Equal(t, http.StatusNotModified, w2.Code)
	assert.Empty(t, w2.Body.String())
}

func TestCacheMiddleware_DifferentQueryParams(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)

	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response " + r.URL.Query().Get("id")))
	})

	middleware := CacheMiddleware(config)(handler)

	// First request
	r1 := httptest.NewRequest("GET", "/test?id=1", nil)
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, r1)

	assert.Equal(t, 1, callCount)
	assert.Equal(t, "response 1", w1.Body.String())

	// Second request with different query param
	r2 := httptest.NewRequest("GET", "/test?id=2", nil)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, r2)

	assert.Equal(t, 2, callCount)
	assert.Equal(t, "response 2", w2.Body.String())

	// Third request with same query as first
	r3 := httptest.NewRequest("GET", "/test?id=1", nil)
	w3 := httptest.NewRecorder()
	middleware.ServeHTTP(w3, r3)

	assert.Equal(t, 2, callCount) // Should use cache
	assert.Equal(t, "response 1", w3.Body.String())
	assert.Equal(t, "HIT", w3.Header().Get("X-Cache"))
}

func TestCacheMiddleware_CustomTTL(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)
	config.TTL = 50 * time.Millisecond

	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	middleware := CacheMiddleware(config)(handler)

	// First request
	r1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, r1)

	assert.Equal(t, 1, callCount)

	// Second request immediately - should hit cache
	r2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, r2)

	assert.Equal(t, 1, callCount)
	assert.Equal(t, "HIT", w2.Header().Get("X-Cache"))

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Third request after expiration - should miss cache
	r3 := httptest.NewRequest("GET", "/test", nil)
	w3 := httptest.NewRecorder()
	middleware.ServeHTTP(w3, r3)

	assert.Equal(t, 2, callCount)
	assert.Equal(t, "MISS", w3.Header().Get("X-Cache"))
}

func TestNewResponseRecorder(t *testing.T) {
	w := httptest.NewRecorder()
	recorder := newResponseRecorder(w)

	assert.NotNil(t, recorder)
	assert.Equal(t, http.StatusOK, recorder.statusCode)
	assert.NotNil(t, recorder.body)
}

func TestResponseRecorder_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	recorder := newResponseRecorder(w)

	recorder.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, recorder.statusCode)

	// Second call should not change status code
	recorder.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusCreated, recorder.statusCode)
}

func TestResponseRecorder_Write(t *testing.T) {
	w := httptest.NewRecorder()
	recorder := newResponseRecorder(w)

	data := []byte("test data")
	n, err := recorder.Write(data)

	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, recorder.body.Bytes())
}

func TestResponseRecorder_MultipleWrites(t *testing.T) {
	w := httptest.NewRecorder()
	recorder := newResponseRecorder(w)

	// First write
	data1 := []byte("first ")
	n1, err1 := recorder.Write(data1)
	require.NoError(t, err1)
	assert.Equal(t, len(data1), n1)

	// Second write
	data2 := []byte("second")
	n2, err2 := recorder.Write(data2)
	require.NoError(t, err2)
	assert.Equal(t, len(data2), n2)

	// Verify both writes are in the buffer
	assert.Equal(t, "first second", recorder.body.String())

	// Verify underlying response writer got all the data
	assert.Equal(t, "first second", w.Body.String())

	// Verify status code was set correctly and only once
	assert.Equal(t, http.StatusOK, recorder.statusCode)
	assert.True(t, recorder.wroteHeader)
}

func TestNewCacheInvalidator(t *testing.T) {
	cache := NewMemoryCache()
	invalidator := NewCacheInvalidator(cache)

	assert.NotNil(t, invalidator)
	assert.NotNil(t, invalidator.cache)
	assert.NotNil(t, invalidator.keyGenerator)
}

func TestCacheInvalidator_InvalidatePath(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Set a value
	key := GenerateKeySimple("GET", "/test")
	err := cache.Set(ctx, key, []byte("value"), 1*time.Minute)
	require.NoError(t, err)

	// Create invalidator
	invalidator := NewCacheInvalidator(cache)

	// Invalidate path
	err = invalidator.InvalidatePath(ctx, "GET", "/test")
	require.NoError(t, err)

	// Verify value is deleted
	_, err = cache.Get(ctx, key)
	assert.Error(t, err)
	assert.True(t, IsCacheMiss(err))
}

func TestCacheInvalidator_InvalidateRequest(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	invalidator := NewCacheInvalidator(cache)

	// Create request and generate key
	r := httptest.NewRequest("GET", "/test", nil)
	key := invalidator.keyGenerator.GenerateKey(r)

	// Set a value
	err := cache.Set(ctx, key, []byte("value"), 1*time.Minute)
	require.NoError(t, err)

	// Invalidate request
	err = invalidator.InvalidateRequest(ctx, r)
	require.NoError(t, err)

	// Verify value is deleted
	_, err = cache.Get(ctx, key)
	assert.Error(t, err)
	assert.True(t, IsCacheMiss(err))
}

func TestCacheInvalidator_InvalidateAll(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Set multiple values
	err := cache.Set(ctx, "key1", []byte("value1"), 1*time.Minute)
	require.NoError(t, err)
	err = cache.Set(ctx, "key2", []byte("value2"), 1*time.Minute)
	require.NoError(t, err)

	// Create invalidator
	invalidator := NewCacheInvalidator(cache)

	// Invalidate all
	err = invalidator.InvalidateAll(ctx)
	require.NoError(t, err)

	// Verify all values deleted
	_, err = cache.Get(ctx, "key1")
	assert.Error(t, err)
	_, err = cache.Get(ctx, "key2")
	assert.Error(t, err)
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	assert.True(t, contains(slice, "a"))
	assert.True(t, contains(slice, "b"))
	assert.True(t, contains(slice, "c"))
	assert.False(t, contains(slice, "d"))
	assert.False(t, contains([]string{}, "a"))
}

func TestCacheMiddleware_HeadersPreserved(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	middleware := CacheMiddleware(config)(handler)

	// First request
	r1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, r1)

	// Second request - headers should be preserved from cache
	r2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, r2)

	assert.Equal(t, "value", w2.Header().Get("X-Custom"))
	assert.Equal(t, "application/json", w2.Header().Get("Content-Type"))
	assert.Equal(t, "HIT", w2.Header().Get("X-Cache"))
}

func TestCacheMiddleware_CustomCacheControl(t *testing.T) {
	cache := NewMemoryCache()
	config := DefaultCacheMiddlewareConfig(cache)
	config.CacheControl = "private, max-age=600"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	middleware := CacheMiddleware(config)(handler)

	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, r)

	assert.Equal(t, "private, max-age=600", w.Header().Get("Cache-Control"))
}
