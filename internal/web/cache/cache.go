package cache

import (
	"context"
	"time"
)

// Cache defines the interface for all cache backends
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in the cache with a TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error

	// Clear removes all values from the cache
	Clear(ctx context.Context) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)
}

// CacheConfig holds common configuration for cache backends
type CacheConfig struct {
	// DefaultTTL is the default time-to-live for cached items
	DefaultTTL time.Duration
	// Prefix is prepended to all cache keys
	Prefix string
}

// DefaultCacheConfig returns a default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		DefaultTTL: 5 * time.Minute,
		Prefix:     "conduit:",
	}
}

// ErrCacheMiss is returned when a key is not found in the cache
type ErrCacheMiss struct {
	Key string
}

func (e ErrCacheMiss) Error() string {
	return "cache miss: " + e.Key
}

// IsCacheMiss checks if an error is a cache miss
func IsCacheMiss(err error) bool {
	_, ok := err.(ErrCacheMiss)
	return ok
}
