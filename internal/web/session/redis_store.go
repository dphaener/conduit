package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a Redis-backed session store
type RedisStore struct {
	client *redis.Client
	prefix string
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	// Addr is the Redis server address (host:port)
	Addr string

	// Password is the Redis password (empty if no auth)
	Password string

	// DB is the Redis database number
	DB int

	// PoolSize is the connection pool size
	PoolSize int

	// MinIdleConns is the minimum number of idle connections
	MinIdleConns int

	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int

	// KeyPrefix is the prefix for all session keys
	KeyPrefix string
}

// DefaultRedisConfig returns default Redis configuration
func DefaultRedisConfig(addr string) *RedisConfig {
	return &RedisConfig{
		Addr:         addr,
		Password:     "",
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
		MaxIdleConns: 20,
		KeyPrefix:    "conduit:session:",
	}
}

// NewRedisStore creates a new Redis session store
func NewRedisStore(config *RedisConfig) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,

		// Connection pooling
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxIdleConns: config.MaxIdleConns,

		// Timeouts
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	return &RedisStore{
		client: client,
		prefix: config.KeyPrefix,
	}
}

// NewRedisStoreFromClient creates a new Redis store from an existing client
func NewRedisStoreFromClient(client *redis.Client, keyPrefix string) *RedisStore {
	if keyPrefix == "" {
		keyPrefix = "conduit:session:"
	}
	return &RedisStore{
		client: client,
		prefix: keyPrefix,
	}
}

// Get retrieves a session from Redis
func (s *RedisStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := s.key(sessionID)

	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}

	// Check expiration
	if session.IsExpired() {
		s.client.Del(ctx, key)
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// Set stores a session in Redis
func (s *RedisStore) Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error {
	key := s.key(sessionID)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// Delete removes a session from Redis
func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	key := s.key(sessionID)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del error: %w", err)
	}

	return nil
}

// Refresh updates the expiration time of a session
func (s *RedisStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := s.key(sessionID)

	// Check if key exists
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("redis exists error: %w", err)
	}
	if exists == 0 {
		return ErrSessionNotFound
	}

	// Update expiration
	if err := s.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("redis expire error: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Ping checks the Redis connection
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// key generates the full Redis key for a session
func (s *RedisStore) key(sessionID string) string {
	return s.prefix + sessionID
}
