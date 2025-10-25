package ratelimit

import (
	"context"
	"sync"
	"time"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TokenBucket implements an in-memory token bucket rate limiter
type TokenBucket struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	capacity int
	refillRate time.Duration
	cleanup  *time.Ticker
	done     chan struct{}
}

// bucket represents a single token bucket for a key
type bucket struct {
	tokens     int
	lastRefill time.Time
}

// TokenBucketConfig holds configuration for the token bucket rate limiter
type TokenBucketConfig struct {
	// Capacity is the maximum number of tokens in the bucket
	Capacity int
	// RefillRate is how often tokens are refilled
	RefillRate time.Duration
	// CleanupInterval is how often to clean up expired buckets
	CleanupInterval time.Duration
}

// DefaultTokenBucketConfig returns a default token bucket configuration
// Allows 100 requests per minute
func DefaultTokenBucketConfig() TokenBucketConfig {
	return TokenBucketConfig{
		Capacity:        100,
		RefillRate:      time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
}

// NewTokenBucket creates a new token bucket rate limiter with default configuration
func NewTokenBucket() *TokenBucket {
	return NewTokenBucketWithConfig(DefaultTokenBucketConfig())
}

// NewTokenBucketWithConfig creates a new token bucket rate limiter with custom configuration
func NewTokenBucketWithConfig(config TokenBucketConfig) *TokenBucket {
	tb := &TokenBucket{
		buckets:    make(map[string]*bucket),
		capacity:   config.Capacity,
		refillRate: config.RefillRate,
		done:       make(chan struct{}),
	}

	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		tb.cleanup = time.NewTicker(config.CleanupInterval)
		go tb.cleanupLoop()
	}

	return tb
}

// Allow checks if a request should be allowed for the given key
func (tb *TokenBucket) Allow(ctx context.Context, key string) (*RateLimitInfo, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()

	// Get or create bucket
	b, exists := tb.buckets[key]
	if !exists {
		b = &bucket{
			tokens:     tb.capacity - 1, // Consume one token immediately
			lastRefill: now,
		}
		tb.buckets[key] = b

		return &RateLimitInfo{
			Limit:     tb.capacity,
			Remaining: b.tokens,
			ResetAt:   now.Add(tb.refillRate),
			Allowed:   true,
		}, nil
	}

	// Calculate tokens to add based on elapsed time
	elapsed := now.Sub(b.lastRefill)
	if elapsed > 0 {
		// Add tokens proportionally to elapsed time
		// Rate: capacity tokens per refillRate duration
		tokensToAdd := int(float64(tb.capacity) * elapsed.Seconds() / tb.refillRate.Seconds())
		if tokensToAdd > 0 {
			b.tokens = min(tb.capacity, b.tokens+tokensToAdd)
			b.lastRefill = now
		}
	}

	// Check if token available
	if b.tokens > 0 {
		b.tokens--
		return &RateLimitInfo{
			Limit:     tb.capacity,
			Remaining: b.tokens,
			ResetAt:   b.lastRefill.Add(tb.refillRate),
			Allowed:   true,
		}, nil
	}

	// Rate limit exceeded
	return &RateLimitInfo{
		Limit:     tb.capacity,
		Remaining: 0,
		ResetAt:   b.lastRefill.Add(tb.refillRate),
		Allowed:   false,
	}, nil
}

// cleanupLoop removes old buckets that haven't been used recently
func (tb *TokenBucket) cleanupLoop() {
	for {
		select {
		case <-tb.cleanup.C:
			tb.cleanupOldBuckets()
		case <-tb.done:
			return
		}
	}
}

// cleanupOldBuckets removes buckets that haven't been accessed in 2x refill rate
func (tb *TokenBucket) cleanupOldBuckets() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	threshold := 2 * tb.refillRate

	for key, b := range tb.buckets {
		if now.Sub(b.lastRefill) > threshold {
			delete(tb.buckets, key)
		}
	}
}

// Close stops the cleanup goroutine
func (tb *TokenBucket) Close() error {
	close(tb.done)
	if tb.cleanup != nil {
		tb.cleanup.Stop()
	}
	return nil
}
