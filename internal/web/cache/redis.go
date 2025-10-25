package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements a Redis-backed cache
type RedisCache struct {
	client *redis.Client
	config CacheConfig
}

// RedisConfig holds Redis-specific configuration
type RedisConfig struct {
	// Addr is the Redis server address (host:port)
	Addr string
	// Password is the Redis password (optional)
	Password string
	// DB is the Redis database number
	DB int
	// CacheConfig holds common cache configuration
	CacheConfig CacheConfig
}

// DefaultRedisConfig returns a default Redis configuration
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:        "localhost:6379",
		Password:    "",
		DB:          0,
		CacheConfig: DefaultCacheConfig(),
	}
}

// NewRedisCache creates a new Redis cache with default configuration
func NewRedisCache() (*RedisCache, error) {
	return NewRedisCacheWithConfig(DefaultRedisConfig())
}

// NewRedisCacheWithConfig creates a new Redis cache with custom configuration
func NewRedisCacheWithConfig(config RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
		config: config.CacheConfig,
	}, nil
}

// NewRedisCacheWithClient creates a new Redis cache with an existing client
func NewRedisCacheWithClient(client *redis.Client, config CacheConfig) *RedisCache {
	return &RedisCache{
		client: client,
		config: config,
	}
}

// Get retrieves a value from the cache
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := r.config.Prefix + key

	value, err := r.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss{Key: key}
		}
		return nil, err
	}

	return value, nil
}

// Set stores a value in the cache with a TTL
func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := r.config.Prefix + key

	// Use default TTL if none provided
	if ttl == 0 {
		ttl = r.config.DefaultTTL
	}

	return r.client.Set(ctx, fullKey, value, ttl).Err()
}

// Delete removes a value from the cache
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := r.config.Prefix + key
	return r.client.Del(ctx, fullKey).Err()
}

// Clear removes all values from the cache
func (r *RedisCache) Clear(ctx context.Context) error {
	// Use SCAN to find all keys with our prefix
	iter := r.client.Scan(ctx, 0, r.config.Prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Exists checks if a key exists in the cache
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.config.Prefix + key

	count, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}
