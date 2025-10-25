package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	// Create a mock Redis server
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create cache
	cache := NewRedisCacheWithClient(client, DefaultCacheConfig())
	return cache, mr
}

func TestNewRedisCacheWithConfig(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	config := RedisConfig{
		Addr:        mr.Addr(),
		Password:    "",
		DB:          0,
		CacheConfig: DefaultCacheConfig(),
	}

	cache, err := NewRedisCacheWithConfig(config)
	require.NoError(t, err)
	assert.NotNil(t, cache)
	defer cache.Close()
}

func TestNewRedisCacheWithConfig_ConnectionError(t *testing.T) {
	config := RedisConfig{
		Addr:        "localhost:99999", // Invalid port
		Password:    "",
		DB:          0,
		CacheConfig: DefaultCacheConfig(),
	}

	_, err := NewRedisCacheWithConfig(config)
	assert.Error(t, err)
}

func TestRedisCache_SetAndGet(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

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

func TestRedisCache_GetMiss(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Get non-existent key
	_, err := cache.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, IsCacheMiss(err))
}

func TestRedisCache_Delete(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

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

func TestRedisCache_Clear(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

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

func TestRedisCache_Exists(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

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

func TestRedisCache_TTLExpiration(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

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

	// Fast-forward time in miniredis
	mr.FastForward(100 * time.Millisecond)

	// Get value after expiration
	_, err = cache.Get(ctx, key)
	assert.Error(t, err)
	assert.True(t, IsCacheMiss(err))
}

func TestRedisCache_DefaultTTL(t *testing.T) {
	config := CacheConfig{
		DefaultTTL: 1 * time.Hour,
		Prefix:     "test:",
	}
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cache := NewRedisCacheWithClient(client, config)
	defer cache.Close()

	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value with 0 TTL (should use default)
	err = cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Verify value is set
	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestRedisCache_Prefix(t *testing.T) {
	config := CacheConfig{
		DefaultTTL: 1 * time.Minute,
		Prefix:     "prefix:",
	}
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cache := NewRedisCacheWithClient(client, config)
	defer cache.Close()

	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value
	err = cache.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	// Get value
	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Verify key has prefix in Redis
	keys := mr.Keys()
	assert.Len(t, keys, 1)
	assert.Equal(t, "prefix:test-key", keys[0])
}

func TestDefaultRedisConfig(t *testing.T) {
	config := DefaultRedisConfig()
	assert.Equal(t, "localhost:6379", config.Addr)
	assert.Equal(t, "", config.Password)
	assert.Equal(t, 0, config.DB)
	assert.NotZero(t, config.CacheConfig.DefaultTTL)
}
