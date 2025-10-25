package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/web/ratelimit"
	webcontext "github.com/conduit-lang/conduit/internal/web/context"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit_Allow(t *testing.T) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        10,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	middleware := RateLimit(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "success", rec.Body.String())
	assert.Equal(t, "10", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "9", rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
}

func TestRateLimit_Deny(t *testing.T) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        2,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	middleware := RateLimit(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Make requests until denied
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Third request should be denied
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestRateLimit_DifferentIPs(t *testing.T) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        2,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	middleware := RateLimit(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust IP1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// IP1 should be denied
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// IP2 should still work
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_XForwardedFor(t *testing.T) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        2,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	middleware := RateLimit(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust using X-Forwarded-For
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Should be denied
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_BypassFunc(t *testing.T) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        1,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	bypassFunc := func(r *http.Request) bool {
		return r.Header.Get("X-Admin") == "true"
	}

	middleware := RateLimitWithConfig(RateLimitConfig{
		Limiter:    limiter,
		KeyFunc:    IPKeyFunc,
		BypassFunc: bypassFunc,
		FailOpen:   false,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Should be denied
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Should bypass with admin header
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("X-Admin", "true")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_UserKeyFunc(t *testing.T) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        2,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	middleware := RateLimitWithConfig(RateLimitConfig{
		Limiter: limiter,
		KeyFunc: UserKeyFunc,
		FailOpen: false,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create context with user ID
	ctx := webcontext.SetCurrentUser(context.Background(), "user123")

	// Exhaust limit for user123
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Should be denied
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Different user should work
	ctx2 := webcontext.SetCurrentUser(context.Background(), "user456")
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(ctx2)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_EndpointKeyFunc(t *testing.T) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        2,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	middleware := RateLimitWithConfig(RateLimitConfig{
		Limiter: limiter,
		KeyFunc: EndpointKeyFunc,
		FailOpen: false,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust /api/users
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Should be denied
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Different endpoint should work
	req = httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_UserEndpointKeyFunc(t *testing.T) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        2,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	middleware := RateLimitWithConfig(RateLimitConfig{
		Limiter: limiter,
		KeyFunc: UserEndpointKeyFunc,
		FailOpen: false,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ctx := webcontext.SetCurrentUser(context.Background(), "user123")

	// Exhaust user123 + /api/users
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Should be denied for user123 + /api/users
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Same user, different endpoint should work
	req = httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_FailOpen(t *testing.T) {
	// Mock limiter that always errors
	mockLimiter := &mockRateLimiter{shouldError: true}

	middleware := RateLimitWithConfig(RateLimitConfig{
		Limiter:  mockLimiter,
		KeyFunc:  IPKeyFunc,
		FailOpen: true,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should allow when fail open
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_FailClosed(t *testing.T) {
	// Mock limiter that always errors
	mockLimiter := &mockRateLimiter{shouldError: true}

	middleware := RateLimitWithConfig(RateLimitConfig{
		Limiter:  mockLimiter,
		KeyFunc:  IPKeyFunc,
		FailOpen: false,
		ErrorHandler: DefaultRateLimitErrorHandler,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should deny when fail closed
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestIPKeyFunc(t *testing.T) {
	tests := []struct {
		name         string
		remoteAddr   string
		xForwardedFor string
		xRealIP      string
		expected     string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For single IP",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "10.0.0.1",
			expected:      "10.0.0.1",
		},
		{
			name:          "X-Forwarded-For multiple IPs",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "10.0.0.1, 10.0.0.2, 10.0.0.3",
			expected:      "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "192.168.1.1:12345",
			xRealIP:    "10.0.0.1",
			expected:   "10.0.0.1",
		},
		{
			name:          "X-Forwarded-For takes precedence",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "10.0.0.1",
			xRealIP:       "10.0.0.2",
			expected:      "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			result := IPKeyFunc(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdminBypassFunc(t *testing.T) {
	tests := []struct {
		name     string
		roles    []string
		expected bool
	}{
		{
			name:     "admin role",
			roles:    []string{"admin"},
			expected: true,
		},
		{
			name:     "superadmin role",
			roles:    []string{"superadmin"},
			expected: true,
		},
		{
			name:     "regular user",
			roles:    []string{"user"},
			expected: false,
		},
		{
			name:     "no roles",
			roles:    []string{},
			expected: false,
		},
		{
			name:     "multiple roles with admin",
			roles:    []string{"user", "admin", "editor"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := webcontext.SetUserRoles(context.Background(), tt.roles)
			req = req.WithContext(ctx)

			result := AdminBypassFunc(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInternalBypassFunc(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected bool
	}{
		{
			name:     "internal request",
			header:   "true",
			expected: true,
		},
		{
			name:     "external request",
			header:   "false",
			expected: false,
		},
		{
			name:     "no header",
			header:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.header != "" {
				req.Header.Set("X-Internal", tt.header)
			}

			result := InternalBypassFunc(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCombinedBypassFunc(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := webcontext.SetUserRoles(context.Background(), []string{"user"})
	req = req.WithContext(ctx)

	// Neither bypass should trigger
	combined := CombinedBypassFunc(AdminBypassFunc, InternalBypassFunc)
	assert.False(t, combined(req))

	// Set admin role
	ctx = webcontext.SetUserRoles(context.Background(), []string{"admin"})
	req = req.WithContext(ctx)
	assert.True(t, combined(req))

	// Or internal header
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Internal", "true")
	assert.True(t, combined(req))
}

func TestRateLimit_Concurrent(t *testing.T) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        100,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	middleware := RateLimit(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup
	results := make(chan int, 150)

	// Launch 150 concurrent requests
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			results <- rec.Code
		}()
	}

	wg.Wait()
	close(results)

	// Count status codes
	okCount := 0
	tooManyCount := 0
	for code := range results {
		if code == http.StatusOK {
			okCount++
		} else if code == http.StatusTooManyRequests {
			tooManyCount++
		}
	}

	// Should allow exactly 100 requests
	assert.Equal(t, 100, okCount)
	assert.Equal(t, 50, tooManyCount)
}

// Mock rate limiter for testing error handling
type mockRateLimiter struct {
	shouldError bool
}

func (m *mockRateLimiter) Allow(ctx context.Context, key string) (*ratelimit.RateLimitInfo, error) {
	if m.shouldError {
		return nil, assert.AnError
	}
	return &ratelimit.RateLimitInfo{
		Limit:     10,
		Remaining: 5,
		ResetAt:   time.Now().Add(time.Minute),
		Allowed:   true,
	}, nil
}

func BenchmarkRateLimit_Middleware(b *testing.B) {
	limiter := ratelimit.NewTokenBucketWithConfig(ratelimit.TokenBucketConfig{
		Capacity:        1000000,
		RefillRate:      time.Minute,
		CleanupInterval: 0,
	})
	defer limiter.Close()

	middleware := RateLimit(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
