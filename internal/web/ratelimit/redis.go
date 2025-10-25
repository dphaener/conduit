package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements a Redis-backed sliding window rate limiter
type RedisRateLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
	prefix string
}

// RedisRateLimiterConfig holds configuration for the Redis rate limiter
type RedisRateLimiterConfig struct {
	// Client is the Redis client to use
	Client *redis.Client
	// Limit is the maximum number of requests allowed in the window
	Limit int
	// Window is the time window for rate limiting
	Window time.Duration
	// Prefix is the key prefix for Redis keys
	Prefix string
}

// DefaultRedisRateLimiterConfig returns a default Redis rate limiter configuration
// Allows 100 requests per minute
func DefaultRedisRateLimiterConfig(client *redis.Client) RedisRateLimiterConfig {
	return RedisRateLimiterConfig{
		Client: client,
		Limit:  100,
		Window: time.Minute,
		Prefix: "ratelimit:",
	}
}

// NewRedisRateLimiter creates a new Redis rate limiter with custom configuration
func NewRedisRateLimiter(config RedisRateLimiterConfig) (*RedisRateLimiter, error) {
	if config.Client == nil {
		return nil, errors.New("redis client is required")
	}
	if config.Limit <= 0 {
		return nil, errors.New("limit must be greater than 0")
	}
	if config.Window <= 0 {
		return nil, errors.New("window must be greater than 0")
	}

	return &RedisRateLimiter{
		client: config.Client,
		limit:  config.Limit,
		window: config.Window,
		prefix: config.Prefix,
	}, nil
}

// Allow checks if a request should be allowed for the given key using sliding window
func (r *RedisRateLimiter) Allow(ctx context.Context, key string) (*RateLimitInfo, error) {
	redisKey := r.prefix + key
	now := time.Now()
	windowStart := now.Add(-r.window)

	// Use Lua script for atomic operations
	script := redis.NewScript(`
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window = tonumber(ARGV[4])

		-- Remove old entries
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)

		-- Count current entries
		local current = redis.call('ZCARD', key)

		-- Check if limit exceeded
		if current < limit then
			-- Add new entry
			redis.call('ZADD', key, now, now)
			-- Set expiration
			redis.call('EXPIRE', key, window)
			return {1, current + 1}
		else
			return {0, current}
		end
	`)

	result, err := script.Run(ctx, r.client, []string{redisKey},
		now.UnixNano(),
		windowStart.UnixNano(),
		r.limit,
		int(r.window.Seconds()),
	).Result()

	if err != nil {
		return nil, fmt.Errorf("redis rate limit check failed: %w", err)
	}

	// Parse result
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 2 {
		return nil, errors.New("unexpected redis script result")
	}

	allowed, ok := resultSlice[0].(int64)
	if !ok {
		return nil, errors.New("invalid allowed value from redis")
	}

	count, ok := resultSlice[1].(int64)
	if !ok {
		return nil, errors.New("invalid count value from redis")
	}

	remaining := r.limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitInfo{
		Limit:     r.limit,
		Remaining: remaining,
		ResetAt:   now.Add(r.window),
		Allowed:   allowed == 1,
	}, nil
}

// Reset removes all rate limit data for the given key
func (r *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := r.prefix + key
	return r.client.Del(ctx, redisKey).Err()
}

// GetCount returns the current count for the given key
func (r *RedisRateLimiter) GetCount(ctx context.Context, key string) (int, error) {
	redisKey := r.prefix + key
	now := time.Now()
	windowStart := now.Add(-r.window)

	// Remove old entries and count
	pipe := r.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart.UnixNano(), 10))
	countCmd := pipe.ZCard(ctx, redisKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %w", err)
	}

	return int(countCmd.Val()), nil
}
