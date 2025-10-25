package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBucket_Allow_FirstRequest(t *testing.T) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        10,
		RefillRate:      time.Minute,
		CleanupInterval: 0, // Disable cleanup for tests
	})
	defer tb.Close()

	ctx := context.Background()
	info, err := tb.Allow(ctx, "test-key")

	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, 10, info.Limit)
	assert.Equal(t, 9, info.Remaining) // One token consumed
}

func TestTokenBucket_Allow_ExceedLimit(t *testing.T) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        3,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer tb.Close()

	ctx := context.Background()

	// Consume all tokens
	for i := 0; i < 3; i++ {
		info, err := tb.Allow(ctx, "test-key")
		require.NoError(t, err)
		assert.True(t, info.Allowed, "request %d should be allowed", i)
		assert.Equal(t, 3-i-1, info.Remaining)
	}

	// Fourth request should be denied
	info, err := tb.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.False(t, info.Allowed)
	assert.Equal(t, 0, info.Remaining)
}

func TestTokenBucket_Allow_DifferentKeys(t *testing.T) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        5,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer tb.Close()

	ctx := context.Background()

	// Consume tokens for key1
	for i := 0; i < 5; i++ {
		info, err := tb.Allow(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, info.Allowed)
	}

	// key1 should be exhausted
	info, err := tb.Allow(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, info.Allowed)

	// key2 should still have tokens
	info, err = tb.Allow(ctx, "key2")
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, 4, info.Remaining)
}

func TestTokenBucket_Allow_Refill(t *testing.T) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        5,
		RefillRate:      100 * time.Millisecond,
		CleanupInterval: 0,
	})
	defer tb.Close()

	ctx := context.Background()

	// Consume all tokens
	for i := 0; i < 5; i++ {
		info, err := tb.Allow(ctx, "test-key")
		require.NoError(t, err)
		assert.True(t, info.Allowed)
	}

	// Should be denied
	info, err := tb.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.False(t, info.Allowed)

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	info, err = tb.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, 4, info.Remaining) // Full refill, minus one
}

func TestTokenBucket_Allow_Concurrent(t *testing.T) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        100,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer tb.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	allowed := make(chan bool, 150)

	// Launch 150 concurrent requests
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			info, err := tb.Allow(ctx, "test-key")
			require.NoError(t, err)
			allowed <- info.Allowed
		}()
	}

	wg.Wait()
	close(allowed)

	// Count allowed and denied
	allowedCount := 0
	deniedCount := 0
	for a := range allowed {
		if a {
			allowedCount++
		} else {
			deniedCount++
		}
	}

	// Should allow exactly 100 requests
	assert.Equal(t, 100, allowedCount)
	assert.Equal(t, 50, deniedCount)
}

func TestTokenBucket_ResetAt(t *testing.T) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        10,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer tb.Close()

	ctx := context.Background()
	before := time.Now()

	info, err := tb.Allow(ctx, "test-key")
	require.NoError(t, err)

	after := time.Now().Add(time.Minute)

	// ResetAt should be approximately now + refill rate
	assert.True(t, info.ResetAt.After(before))
	assert.True(t, info.ResetAt.Before(after.Add(time.Second))) // Allow 1 second tolerance
}

func TestTokenBucket_Cleanup(t *testing.T) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        10,
		RefillRate:      50 * time.Millisecond,
		CleanupInterval: 100 * time.Millisecond,
	})
	defer tb.Close()

	ctx := context.Background()

	// Create some buckets
	_, err := tb.Allow(ctx, "key1")
	require.NoError(t, err)
	_, err = tb.Allow(ctx, "key2")
	require.NoError(t, err)

	// Verify buckets exist
	tb.mu.RLock()
	assert.Len(t, tb.buckets, 2)
	tb.mu.RUnlock()

	// Wait for cleanup (2x refill rate + cleanup interval)
	time.Sleep(250 * time.Millisecond)

	// Buckets should be cleaned up
	tb.mu.RLock()
	assert.Len(t, tb.buckets, 0)
	tb.mu.RUnlock()
}

func TestTokenBucket_MultipleRefills(t *testing.T) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        5,
		RefillRate:      50 * time.Millisecond,
		CleanupInterval: 0,
	})
	defer tb.Close()

	ctx := context.Background()

	// Consume all tokens
	for i := 0; i < 5; i++ {
		info, err := tb.Allow(ctx, "test-key")
		require.NoError(t, err)
		assert.True(t, info.Allowed)
	}

	// Wait for multiple refill periods (3x)
	time.Sleep(160 * time.Millisecond)

	// Should have full capacity again
	info, err := tb.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, 4, info.Remaining)
}

func TestTokenBucket_DefaultConfig(t *testing.T) {
	tb := NewTokenBucket()
	defer tb.Close()

	assert.Equal(t, 100, tb.capacity)
	assert.Equal(t, time.Minute, tb.refillRate)
}

func TestTokenBucket_Close(t *testing.T) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        10,
		RefillRate:      time.Minute,
		CleanupInterval: time.Second,
	})

	err := tb.Close()
	assert.NoError(t, err)

	// Verify cleanup ticker is stopped and channel is closed
	select {
	case <-tb.done:
		// Channel should be closed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("done channel should be closed")
	}
}

func BenchmarkTokenBucket_Allow_SingleKey(b *testing.B) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        1000000,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer tb.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow(ctx, "test-key")
	}
}

func BenchmarkTokenBucket_Allow_MultipleKeys(b *testing.B) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        1000000,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer tb.Close()

	ctx := context.Background()
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = string(rune('a' + i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow(ctx, keys[i%len(keys)])
	}
}

func BenchmarkTokenBucket_Allow_Concurrent(b *testing.B) {
	tb := NewTokenBucketWithConfig(TokenBucketConfig{
		Capacity:        1000000,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer tb.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tb.Allow(ctx, "test-key")
			i++
		}
	})
}
