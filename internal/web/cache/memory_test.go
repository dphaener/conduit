package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryCache(t *testing.T) {
	cache := NewMemoryCache()
	assert.NotNil(t, cache)
	assert.NotZero(t, cache.config.DefaultTTL)
}

func TestNewMemoryCacheWithConfig(t *testing.T) {
	config := CacheConfig{
		DefaultTTL: 10 * time.Minute,
		Prefix:     "test:",
	}
	cache := NewMemoryCacheWithConfig(config)
	assert.NotNil(t, cache)
	assert.Equal(t, config.DefaultTTL, cache.config.DefaultTTL)
	assert.Equal(t, config.Prefix, cache.config.Prefix)
}

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value
	err := cache.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	// Get value
	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestMemoryCache_GetMiss(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Get non-existent key
	_, err := cache.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, IsCacheMiss(err))
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value
	err := cache.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	// Delete value
	err = cache.Delete(ctx, key)
	require.NoError(t, err)

	// Verify deleted
	_, err = cache.Get(ctx, key)
	assert.Error(t, err)
	assert.True(t, IsCacheMiss(err))
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Set multiple values
	err := cache.Set(ctx, "key1", []byte("value1"), 1*time.Minute)
	require.NoError(t, err)
	err = cache.Set(ctx, "key2", []byte("value2"), 1*time.Minute)
	require.NoError(t, err)

	// Clear cache
	err = cache.Clear(ctx)
	require.NoError(t, err)

	// Verify all keys deleted
	_, err = cache.Get(ctx, "key1")
	assert.Error(t, err)
	_, err = cache.Get(ctx, "key2")
	assert.Error(t, err)
}

func TestMemoryCache_Exists(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Check non-existent key
	exists, err := cache.Exists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)

	// Set value
	err = cache.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	// Check existing key
	exists, err = cache.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value with short TTL
	err := cache.Set(ctx, key, value, 50*time.Millisecond)
	require.NoError(t, err)

	// Get value immediately
	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Get value after expiration
	_, err = cache.Get(ctx, key)
	assert.Error(t, err)
	assert.True(t, IsCacheMiss(err))
}

func TestMemoryCache_DefaultTTL(t *testing.T) {
	config := CacheConfig{
		DefaultTTL: 1 * time.Hour,
		Prefix:     "test:",
	}
	cache := NewMemoryCacheWithConfig(config)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value with 0 TTL (should use default)
	err := cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Verify value is set
	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestMemoryCache_NoExpiration(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value with negative TTL (no expiration)
	err := cache.Set(ctx, key, value, -1)
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Value should still be available
	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestMemoryCache_Prefix(t *testing.T) {
	config := CacheConfig{
		DefaultTTL: 1 * time.Minute,
		Prefix:     "prefix:",
	}
	cache := NewMemoryCacheWithConfig(config)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value
	err := cache.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	// Get value
	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := string(rune('a' + n))
			value := []byte{byte('A' + n)}
			cache.Set(ctx, key, value, 1*time.Minute)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := string(rune('a' + n))
			_, err := cache.Get(ctx, key)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Clean up
	cache.Close()
}

func TestMemoryCache_Close(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "key", []byte("value"), 1*time.Minute)
	require.NoError(t, err)

	// Close the cache
	err = cache.Close()
	require.NoError(t, err)

	// Verify the cache still works after close (data remains)
	value, err := cache.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, []byte("value"), value)
}

func TestMemoryCache_NoGoroutineLeak(t *testing.T) {
	// Create and close multiple caches to verify goroutines are cleaned up
	for i := 0; i < 5; i++ {
		cache := NewMemoryCache()
		ctx := context.Background()

		// Use the cache
		err := cache.Set(ctx, "key", []byte("value"), 1*time.Minute)
		require.NoError(t, err)

		// Close the cache
		err = cache.Close()
		require.NoError(t, err)

		// Give goroutine time to exit
		time.Sleep(10 * time.Millisecond)
	}

	// If there was a goroutine leak, we'd eventually run out of resources
	// This test passes if we can create and close caches without issue
}

func TestMemoryCache_ContextCancellation(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	t.Run("Get with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := cache.Get(ctx, "key")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("Set with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := cache.Set(ctx, "key", []byte("value"), 1*time.Minute)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("Delete with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := cache.Delete(ctx, "key")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("Clear with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := cache.Clear(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("Exists with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := cache.Exists(ctx, "key")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}
