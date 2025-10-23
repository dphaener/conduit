package session

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSRFMiddleware(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	csrfConfig := DefaultCSRFConfig()

	sessionMiddleware := Middleware(sessionConfig)
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Wrap with session middleware
	handler = sessionMiddleware(handler)

	t.Run("GET request should pass without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("POST request without token should fail", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("POST request with invalid token should fail", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-CSRF-Token", "invalid-token")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("POST request with valid token should pass", func(t *testing.T) {
		// First request to get a session and CSRF token
		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		rec1 := httptest.NewRecorder()

		testHandler := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Token should be generated automatically
			token := GetCSRFToken(r.Context())
			assert.NotEmpty(t, token)
			w.WriteHeader(http.StatusOK)
		}))

		testHandler.ServeHTTP(rec1, req1)

		// Extract session cookie
		cookies := rec1.Result().Cookies()
		require.Len(t, cookies, 1)
		sessionCookie := cookies[0]

		// Get the CSRF token from the session
		var csrfToken string
		testHandler2 := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			csrfToken = GetCSRFToken(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.AddCookie(sessionCookie)
		rec2 := httptest.NewRecorder()

		testHandler2.ServeHTTP(rec2, req2)
		require.NotEmpty(t, csrfToken)

		// Now make POST request with valid token
		req3 := httptest.NewRequest(http.MethodPost, "/", nil)
		req3.AddCookie(sessionCookie)
		req3.Header.Set("X-CSRF-Token", csrfToken)
		rec3 := httptest.NewRecorder()

		handler.ServeHTTP(rec3, req3)

		assert.Equal(t, http.StatusOK, rec3.Code)
	})
}

func TestCSRFTokenInFormField(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	csrfConfig := DefaultCSRFConfig()

	sessionMiddleware := Middleware(sessionConfig)
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	// Get session and CSRF token
	var sessionCookie *http.Cookie
	var csrfToken string

	setupHandler := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csrfToken = GetCSRFToken(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	setupHandler.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie = cookies[0]
	require.NotEmpty(t, csrfToken)

	// Test POST with token in form field
	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler = sessionMiddleware(handler)

	formData := url.Values{}
	formData.Set("csrf_token", csrfToken)
	formData.Set("username", "testuser")

	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.AddCookie(sessionCookie)
	rec2 := httptest.NewRecorder()

	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestCSRFSafeMethods(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	csrfConfig := DefaultCSRFConfig()

	sessionMiddleware := Middleware(sessionConfig)
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler = sessionMiddleware(handler)

	safeMethods := []string{"GET", "HEAD", "OPTIONS", "TRACE"}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestCSRFSkipPaths(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	csrfConfig := DefaultCSRFConfig()
	csrfConfig.SkipPaths = []string{"/api/webhook", "/api/public"}

	sessionMiddleware := Middleware(sessionConfig)
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler = sessionMiddleware(handler)

	t.Run("skipped path should not require CSRF token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/webhook", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("non-skipped path should require CSRF token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/protected", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestRegenerateCSRFToken(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	sessionMiddleware := Middleware(sessionConfig)

	handler := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get initial token
		token1 := GetCSRFToken(r.Context())
		require.NotEmpty(t, token1)

		// Regenerate token
		err := RegenerateCSRFToken(r.Context())
		require.NoError(t, err)

		// Get new token
		token2 := GetCSRFToken(r.Context())
		require.NotEmpty(t, token2)

		// Tokens should be different
		assert.NotEqual(t, token1, token2)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCSRFToken(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	sessionMiddleware := Middleware(sessionConfig)

	handler := sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First call should generate a token
		token1 := GetCSRFToken(r.Context())
		require.NotEmpty(t, token1)

		// Second call should return the same token
		token2 := GetCSRFToken(r.Context())
		assert.Equal(t, token1, token2)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRFCustomErrorHandler(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	sessionConfig := DefaultConfig(store)
	sessionConfig.Secure = false

	csrfConfig := DefaultCSRFConfig()
	customErrorCalled := false
	csrfConfig.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		customErrorCalled = true
		http.Error(w, "Custom CSRF error", http.StatusTeapot)
	}

	sessionMiddleware := Middleware(sessionConfig)
	csrfMiddleware := CSRFMiddleware(csrfConfig)

	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler = sessionMiddleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, customErrorCalled)
	assert.Equal(t, http.StatusTeapot, rec.Code)
}

func TestValidateCSRFToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
		valid    bool
	}{
		{
			name:     "matching tokens",
			token:    "abc123",
			expected: "abc123",
			valid:    true,
		},
		{
			name:     "mismatched tokens",
			token:    "abc123",
			expected: "xyz789",
			valid:    false,
		},
		{
			name:     "empty token",
			token:    "",
			expected: "abc123",
			valid:    false,
		},
		{
			name:     "empty expected",
			token:    "abc123",
			expected: "",
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCSRFToken(tt.token, tt.expected)
			assert.Equal(t, tt.valid, result)
		})
	}
}
