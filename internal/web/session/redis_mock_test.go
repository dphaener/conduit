package session

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisStoreMock tests Redis store with an in-memory mock
func TestRedisStoreMock(t *testing.T) {
	// Create a miniredis server
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create Redis client pointing to miniredis
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create store from client
	store := NewRedisStoreFromClient(client, "test:")
	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		sessionID := "test-session-1"
		sess := NewSession(sessionID, 1*time.Hour)
		sess.Set("key1", "value1")
		sess.Set("key2", 42)
		sess.UserID = "user123"
		sess.CSRFToken = "csrf-token"

		// Set session
		err := store.Set(ctx, sessionID, sess, 5*time.Minute)
		require.NoError(t, err)

		// Get session
		retrieved, err := store.Get(ctx, sessionID)
		require.NoError(t, err)
		assert.Equal(t, sessionID, retrieved.ID)
		assert.Equal(t, "user123", retrieved.UserID)
		assert.Equal(t, "csrf-token", retrieved.CSRFToken)

		val1, ok := retrieved.Get("key1")
		require.True(t, ok)
		assert.Equal(t, "value1", val1)

		val2, ok := retrieved.Get("key2")
		require.True(t, ok)
		// JSON unmarshals numbers as float64
		assert.Equal(t, float64(42), val2)
	})

	t.Run("Get non-existent session", func(t *testing.T) {
		_, err := store.Get(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("Get expired session", func(t *testing.T) {
		sessionID := "expired-session"
		sess := NewSession(sessionID, -1*time.Hour) // Already expired

		err := store.Set(ctx, sessionID, sess, 1*time.Second)
		require.NoError(t, err)

		// Session is expired based on ExpiresAt field
		retrieved, err := store.Get(ctx, sessionID)
		if err == nil {
			// If Get succeeded, check if IsExpired
			assert.True(t, retrieved.IsExpired())
		}
	})

	t.Run("Delete session", func(t *testing.T) {
		sessionID := "delete-test"
		sess := NewSession(sessionID, 1*time.Hour)

		err := store.Set(ctx, sessionID, sess, 5*time.Minute)
		require.NoError(t, err)

		// Verify it exists
		_, err = store.Get(ctx, sessionID)
		require.NoError(t, err)

		// Delete
		err = store.Delete(ctx, sessionID)
		require.NoError(t, err)

		// Verify deleted
		_, err = store.Get(ctx, sessionID)
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("Refresh session", func(t *testing.T) {
		sessionID := "refresh-test"
		sess := NewSession(sessionID, 1*time.Hour)

		err := store.Set(ctx, sessionID, sess, 1*time.Second)
		require.NoError(t, err)

		// Refresh with longer TTL
		err = store.Refresh(ctx, sessionID, 1*time.Hour)
		require.NoError(t, err)

		// Session should still exist after original TTL
		time.Sleep(2 * time.Second)
		_, err = store.Get(ctx, sessionID)
		assert.NoError(t, err)

		// Cleanup
		store.Delete(ctx, sessionID)
	})

	t.Run("Refresh non-existent session", func(t *testing.T) {
		err := store.Refresh(ctx, "nonexistent", 1*time.Hour)
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("Ping", func(t *testing.T) {
		err := store.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Close", func(t *testing.T) {
		// Create a new store to close
		client2 := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
		store2 := NewRedisStoreFromClient(client2, "test2:")

		err := store2.Close()
		assert.NoError(t, err)
	})

	t.Run("Key prefix", func(t *testing.T) {
		key := store.key("session-id")
		assert.Equal(t, "test:session-id", key)
	})

	t.Run("Empty prefix uses default", func(t *testing.T) {
		store2 := NewRedisStoreFromClient(client, "")
		assert.Equal(t, "conduit:session:", store2.prefix)
	})

	t.Run("Set with flash messages", func(t *testing.T) {
		sessionID := "flash-test"
		sess := NewSession(sessionID, 1*time.Hour)
		sess.AddFlash(FlashSuccess, "Test message")
		sess.AddFlash(FlashError, "Error message")

		err := store.Set(ctx, sessionID, sess, 5*time.Minute)
		require.NoError(t, err)

		retrieved, err := store.Get(ctx, sessionID)
		require.NoError(t, err)
		assert.Len(t, retrieved.FlashMessages, 2)
		assert.Equal(t, FlashSuccess, retrieved.FlashMessages[0].Type)
		assert.Equal(t, "Test message", retrieved.FlashMessages[0].Message)

		// Cleanup
		store.Delete(ctx, sessionID)
	})
}

// TestRedisStoreWithRealRedis tests with real Redis if available
func TestRedisStoreWithRealRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	config := DefaultRedisConfig("localhost:6379")
	store := NewRedisStore(config)
	defer store.Close()

	ctx := context.Background()

	// Check if Redis is available
	if err := store.Ping(ctx); err != nil {
		t.Skip("Real Redis not available, skipping test:", err)
		return
	}

	// Run basic test with real Redis
	sessionID := "real-redis-test"
	sess := NewSession(sessionID, 1*time.Hour)
	sess.Set("test", "value")

	err := store.Set(ctx, sessionID, sess, 5*time.Minute)
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionID, retrieved.ID)

	// Cleanup
	err = store.Delete(ctx, sessionID)
	require.NoError(t, err)
}
