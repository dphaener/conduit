package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryCache implements an in-memory cache with TTL support
type MemoryCache struct {
	data   sync.Map
	config CacheConfig
	cancel context.CancelFunc
}

// cacheItem represents an item stored in the cache
type cacheItem struct {
	value      []byte
	expiration time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	return NewMemoryCacheWithConfig(DefaultCacheConfig())
}

// NewMemoryCacheWithConfig creates a new in-memory cache with custom configuration
func NewMemoryCacheWithConfig(config CacheConfig) *MemoryCache {
	ctx, cancel := context.WithCancel(context.Background())
	mc := &MemoryCache{
		config: config,
		cancel: cancel,
	}

	// Start background goroutine to clean up expired items
	go mc.cleanupExpired(ctx)

	return mc
}

// Get retrieves a value from the cache
func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullKey := m.config.Prefix + key

	value, ok := m.data.Load(fullKey)
	if !ok {
		return nil, ErrCacheMiss{Key: key}
	}

	item := value.(cacheItem)

	// Check if item has expired
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		m.data.Delete(fullKey)
		return nil, ErrCacheMiss{Key: key}
	}

	return item.value, nil
}

// Set stores a value in the cache with a TTL
func (m *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullKey := m.config.Prefix + key

	// Use default TTL if none provided
	if ttl == 0 {
		ttl = m.config.DefaultTTL
	}

	item := cacheItem{
		value: value,
	}

	// Set expiration if TTL is positive
	if ttl > 0 {
		item.expiration = time.Now().Add(ttl)
	}

	m.data.Store(fullKey, item)
	return nil
}

// Delete removes a value from the cache
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullKey := m.config.Prefix + key
	m.data.Delete(fullKey)
	return nil
}

// Clear removes all values from the cache
func (m *MemoryCache) Clear(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.data.Range(func(key, value interface{}) bool {
		m.data.Delete(key)
		return true
	})
	return nil
}

// Exists checks if a key exists in the cache
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	fullKey := m.config.Prefix + key

	value, ok := m.data.Load(fullKey)
	if !ok {
		return false, nil
	}

	item := value.(cacheItem)

	// Check if item has expired
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		m.data.Delete(fullKey)
		return false, nil
	}

	return true, nil
}

// Close stops the background cleanup goroutine
func (m *MemoryCache) Close() error {
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}

// cleanupExpired periodically removes expired items from the cache
func (m *MemoryCache) cleanupExpired(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			m.data.Range(func(key, value interface{}) bool {
				item := value.(cacheItem)
				if !item.expiration.IsZero() && now.After(item.expiration) {
					m.data.Delete(key)
				}
				return true
			})
		}
	}
}
