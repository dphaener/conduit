package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisStoreConfig tests Redis configuration
func TestRedisStoreConfig(t *testing.T) {
	config := DefaultRedisConfig("localhost:6379")

	assert.Equal(t, "localhost:6379", config.Addr)
	assert.Equal(t, "", config.Password)
	assert.Equal(t, 0, config.DB)
	assert.Equal(t, 100, config.PoolSize)
	assert.Equal(t, 10, config.MinIdleConns)
	assert.Equal(t, 20, config.MaxIdleConns)
	assert.Equal(t, "conduit:session:", config.KeyPrefix)
}

// TestNewRedisStore tests Redis store creation
func TestNewRedisStore(t *testing.T) {
	config := DefaultRedisConfig("localhost:6379")
	store := NewRedisStore(config)

	assert.NotNil(t, store)
	assert.NotNil(t, store.client)
	assert.Equal(t, "conduit:session:", store.prefix)

	// Close to prevent resource leak
	store.Close()
}

// TestRedisStoreKey tests key generation
func TestRedisStoreKey(t *testing.T) {
	config := DefaultRedisConfig("localhost:6379")
	store := NewRedisStore(config)
	defer store.Close()

	key := store.key("test-session-id")
	assert.Equal(t, "conduit:session:test-session-id", key)
}

// Note: The following tests require a running Redis instance
// They are skipped if Redis is not available

func TestRedisStoreIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration tests in short mode")
	}

	config := DefaultRedisConfig("localhost:6379")
	store := NewRedisStore(config)
	defer store.Close()

	ctx := context.Background()

	// Check if Redis is available
	if err := store.Ping(ctx); err != nil {
		t.Skip("Redis not available, skipping integration tests:", err)
		return
	}

	t.Run("Get/Set/Delete", func(t *testing.T) {
		sessionID := "redis-test-session"
		ttl := 1 * time.Hour

		// Create session
		sess := NewSession(sessionID, ttl)
		sess.Set("key1", "value1")
		sess.UserID = "user-123"

		// Set
		err := store.Set(ctx, sessionID, sess, ttl)
		require.NoError(t, err)

		// Get
		retrieved, err := store.Get(ctx, sessionID)
		require.NoError(t, err)
		assert.Equal(t, sessionID, retrieved.ID)
		assert.Equal(t, "user-123", retrieved.UserID)

		val, ok := retrieved.Get("key1")
		require.True(t, ok)
		assert.Equal(t, "value1", val)

		// Delete
		err = store.Delete(ctx, sessionID)
		require.NoError(t, err)

		// Verify deleted
		_, err = store.Get(ctx, sessionID)
		assert.Error(t, err)
	})

	t.Run("Refresh", func(t *testing.T) {
		sessionID := "redis-test-refresh"
		ttl := 1 * time.Second

		sess := NewSession(sessionID, ttl)
		err := store.Set(ctx, sessionID, sess, ttl)
		require.NoError(t, err)

		// Refresh
		err = store.Refresh(ctx, sessionID, 1*time.Hour)
		require.NoError(t, err)

		// Wait past original TTL
		time.Sleep(2 * time.Second)

		// Session should still exist
		_, err = store.Get(ctx, sessionID)
		assert.NoError(t, err)

		// Cleanup
		store.Delete(ctx, sessionID)
	})

	t.Run("Get Not Found", func(t *testing.T) {
		_, err := store.Get(ctx, "nonexistent-redis-session")
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("Refresh Not Found", func(t *testing.T) {
		err := store.Refresh(ctx, "nonexistent-redis-session", 1*time.Hour)
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})
}
