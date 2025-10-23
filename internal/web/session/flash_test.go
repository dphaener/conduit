package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddFlash(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := AddFlash(r.Context(), FlashSuccess, "Operation successful")
		require.NoError(t, err)

		err = AddFlash(r.Context(), FlashError, "An error occurred")
		require.NoError(t, err)

		sess := GetSession(r.Context())
		require.NotNil(t, sess)
		assert.Len(t, sess.FlashMessages, 2)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAddFlashHelpers(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := AddFlashSuccess(r.Context(), "Success message")
		require.NoError(t, err)

		err = AddFlashError(r.Context(), "Error message")
		require.NoError(t, err)

		err = AddFlashWarning(r.Context(), "Warning message")
		require.NoError(t, err)

		err = AddFlashInfo(r.Context(), "Info message")
		require.NoError(t, err)

		sess := GetSession(r.Context())
		require.NotNil(t, sess)
		assert.Len(t, sess.FlashMessages, 4)

		assert.Equal(t, FlashSuccess, sess.FlashMessages[0].Type)
		assert.Equal(t, FlashError, sess.FlashMessages[1].Type)
		assert.Equal(t, FlashWarning, sess.FlashMessages[2].Type)
		assert.Equal(t, FlashInfo, sess.FlashMessages[3].Type)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetFlashes(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add flashes
		err := AddFlashSuccess(r.Context(), "Success message")
		require.NoError(t, err)

		err = AddFlashError(r.Context(), "Error message")
		require.NoError(t, err)

		// Get flashes
		flashes := GetFlashes(r.Context())
		assert.Len(t, flashes, 2)
		assert.Equal(t, FlashSuccess, flashes[0].Type)
		assert.Equal(t, "Success message", flashes[0].Message)

		// Flashes should be cleared
		sess := GetSession(r.Context())
		assert.Len(t, sess.FlashMessages, 0)

		// Getting again should return empty
		flashes = GetFlashes(r.Context())
		assert.Len(t, flashes, 0)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetFlashesByType(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add multiple types
		AddFlashSuccess(r.Context(), "Success 1")
		AddFlashSuccess(r.Context(), "Success 2")
		AddFlashError(r.Context(), "Error 1")
		AddFlashWarning(r.Context(), "Warning 1")

		// Get only success flashes
		flashes := GetFlashesByType(r.Context(), FlashSuccess)
		assert.Len(t, flashes, 2)
		assert.Equal(t, "Success 1", flashes[0].Message)
		assert.Equal(t, "Success 2", flashes[1].Message)

		// All flashes should be cleared
		sess := GetSession(r.Context())
		assert.Len(t, sess.FlashMessages, 0)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHasFlashes(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Initially no flashes
		assert.False(t, HasFlashes(r.Context()))

		// Add a flash
		AddFlashSuccess(r.Context(), "Success message")

		// Now should have flashes
		assert.True(t, HasFlashes(r.Context()))

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHasFlashesByType(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add success flash only
		AddFlashSuccess(r.Context(), "Success message")

		// Should have success flash
		assert.True(t, HasFlashesByType(r.Context(), FlashSuccess))

		// Should not have error flash
		assert.False(t, HasFlashesByType(r.Context(), FlashError))

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPeekFlashes(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add flashes
		AddFlashSuccess(r.Context(), "Success message")
		AddFlashError(r.Context(), "Error message")

		// Peek flashes (should not clear)
		flashes := PeekFlashes(r.Context())
		assert.Len(t, flashes, 2)

		// Flashes should still be there
		sess := GetSession(r.Context())
		assert.Len(t, sess.FlashMessages, 2)

		// Peek again should return same
		flashes = PeekFlashes(r.Context())
		assert.Len(t, flashes, 2)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestClearFlashes(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add flashes
		AddFlashSuccess(r.Context(), "Success message")
		AddFlashError(r.Context(), "Error message")

		sess := GetSession(r.Context())
		assert.Len(t, sess.FlashMessages, 2)

		// Clear flashes
		ClearFlashes(r.Context())

		assert.Len(t, sess.FlashMessages, 0)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestFlashPersistence(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	config := DefaultConfig(store)
	config.Secure = false

	middleware := Middleware(config)

	var sessionCookie *http.Cookie

	// First request: add flash
	handler1 := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddFlashSuccess(r.Context(), "Persisted message")
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()

	handler1.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie = cookies[0]

	// Second request: retrieve flash
	handler2 := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flashes := PeekFlashes(r.Context())
		require.Len(t, flashes, 1)
		assert.Equal(t, FlashSuccess, flashes[0].Type)
		assert.Equal(t, "Persisted message", flashes[0].Message)
		w.WriteHeader(http.StatusOK)
	}))

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(sessionCookie)
	rec2 := httptest.NewRecorder()

	handler2.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
}
