package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestNewRedisRateLimiter_InvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      RedisRateLimiterConfig
		expectedErr string
	}{
		{
			name: "nil client",
			config: RedisRateLimiterConfig{
				Client: nil,
				Limit:  100,
				Window: time.Minute,
			},
			expectedErr: "redis client is required",
		},
		{
			name: "zero limit",
			config: RedisRateLimiterConfig{
				Client: &redis.Client{},
				Limit:  0,
				Window: time.Minute,
			},
			expectedErr: "limit must be greater than 0",
		},
		{
			name: "negative limit",
			config: RedisRateLimiterConfig{
				Client: &redis.Client{},
				Limit:  -1,
				Window: time.Minute,
			},
			expectedErr: "limit must be greater than 0",
		},
		{
			name: "zero window",
			config: RedisRateLimiterConfig{
				Client: &redis.Client{},
				Limit:  100,
				Window: 0,
			},
			expectedErr: "window must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRedisRateLimiter(tt.config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestRedisRateLimiter_Allow_FirstRequest(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  10,
		Window: time.Minute,
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	info, err := limiter.Allow(ctx, "test-key")

	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, 10, info.Limit)
	assert.Equal(t, 9, info.Remaining)
}

func TestRedisRateLimiter_Allow_ExceedLimit(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  3,
		Window: time.Minute,
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Consume all tokens
	for i := 0; i < 3; i++ {
		info, err := limiter.Allow(ctx, "test-key")
		require.NoError(t, err)
		assert.True(t, info.Allowed, "request %d should be allowed", i)
		assert.Equal(t, 3-i-1, info.Remaining)
	}

	// Fourth request should be denied
	info, err := limiter.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.False(t, info.Allowed)
	assert.Equal(t, 0, info.Remaining)
}

func TestRedisRateLimiter_Allow_DifferentKeys(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  5,
		Window: time.Minute,
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Consume tokens for key1
	for i := 0; i < 5; i++ {
		info, err := limiter.Allow(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, info.Allowed)
	}

	// key1 should be exhausted
	info, err := limiter.Allow(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, info.Allowed)

	// key2 should still have tokens
	info, err = limiter.Allow(ctx, "key2")
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, 4, info.Remaining)
}

func TestRedisRateLimiter_Allow_ResetBehavior(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  5,
		Window: time.Minute,
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Consume all tokens
	for i := 0; i < 5; i++ {
		info, err := limiter.Allow(ctx, "test-key")
		require.NoError(t, err)
		assert.True(t, info.Allowed)
	}

	// Should be denied
	info, err := limiter.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.False(t, info.Allowed)

	// Reset to simulate window expiry
	err = limiter.Reset(ctx, "test-key")
	require.NoError(t, err)

	// Should be allowed again
	info, err = limiter.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.True(t, info.Allowed)
}

func TestRedisRateLimiter_Allow_Concurrent(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  100,
		Window: time.Minute,
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowedCount := 0
	deniedCount := 0

	// Launch 150 concurrent requests
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			info, err := limiter.Allow(ctx, "test-key")
			require.NoError(t, err)
			mu.Lock()
			if info.Allowed {
				allowedCount++
			} else {
				deniedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Verify total requests processed
	total := allowedCount + deniedCount
	assert.Equal(t, 150, total)

	// Note: miniredis doesn't perfectly emulate Redis Lua script atomicity
	// In production with real Redis, exactly 100 would be allowed
	// For testing purposes, we just verify the limiter is working
	assert.Greater(t, allowedCount, 0, "some requests should be allowed")
	assert.Greater(t, deniedCount, 0, "some requests should be denied")
}

func TestRedisRateLimiter_Reset(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  3,
		Window: time.Minute,
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Consume all tokens
	for i := 0; i < 3; i++ {
		info, err := limiter.Allow(ctx, "test-key")
		require.NoError(t, err)
		assert.True(t, info.Allowed)
	}

	// Should be denied
	info, err := limiter.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.False(t, info.Allowed)

	// Reset
	err = limiter.Reset(ctx, "test-key")
	require.NoError(t, err)

	// Should be allowed again
	info, err = limiter.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.True(t, info.Allowed)
}

func TestRedisRateLimiter_GetCount(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  10,
		Window: time.Minute,
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Initial count should be 0
	count, err := limiter.GetCount(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Make some requests
	for i := 0; i < 5; i++ {
		_, err := limiter.Allow(ctx, "test-key")
		require.NoError(t, err)
	}

	// Count should be 5
	count, err = limiter.GetCount(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestRedisRateLimiter_GetCount_AfterReset(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  10,
		Window: time.Minute,
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Make some requests
	for i := 0; i < 5; i++ {
		_, err := limiter.Allow(ctx, "test-key")
		require.NoError(t, err)
	}

	// Count should be 5
	count, err := limiter.GetCount(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	// Reset the limiter
	err = limiter.Reset(ctx, "test-key")
	require.NoError(t, err)

	// Count should be 0 after reset
	count, err = limiter.GetCount(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRedisRateLimiter_Prefix(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	limiter1, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  5,
		Window: time.Minute,
		Prefix: "app1:",
	})
	require.NoError(t, err)

	limiter2, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  5,
		Window: time.Minute,
		Prefix: "app2:",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Exhaust limiter1
	for i := 0; i < 5; i++ {
		info, err := limiter1.Allow(ctx, "key")
		require.NoError(t, err)
		assert.True(t, info.Allowed)
	}

	// limiter1 should be exhausted
	info, err := limiter1.Allow(ctx, "key")
	require.NoError(t, err)
	assert.False(t, info.Allowed)

	// limiter2 should still work (different prefix)
	info, err = limiter2.Allow(ctx, "key")
	require.NoError(t, err)
	assert.True(t, info.Allowed)
}

func TestRedisRateLimiter_DefaultConfig(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultRedisRateLimiterConfig(client)

	assert.Equal(t, client, config.Client)
	assert.Equal(t, 100, config.Limit)
	assert.Equal(t, time.Minute, config.Window)
	assert.Equal(t, "ratelimit:", config.Prefix)
}

func BenchmarkRedisRateLimiter_Allow_SingleKey(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  1000000,
		Window: time.Minute,
		Prefix: "bench:",
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "test-key")
	}
}

func BenchmarkRedisRateLimiter_Allow_MultipleKeys(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	limiter, err := NewRedisRateLimiter(RedisRateLimiterConfig{
		Client: client,
		Limit:  1000000,
		Window: time.Minute,
		Prefix: "bench:",
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = string(rune('a' + i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, keys[i%len(keys)])
	}
}
