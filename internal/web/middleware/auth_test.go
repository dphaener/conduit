package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/web/auth"
)

func TestAuth(t *testing.T) {
	// Create auth service
	authService := auth.NewAuthService("test-secret-key", time.Hour)

	// Generate a valid token
	validToken, err := authService.GenerateToken("user-123", "user@example.com", []string{"admin", "editor"})
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedUserID string
		expectedRoles  []string
	}{
		{
			name:           "allows request with valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedUserID: "user-123",
			expectedRoles:  []string{"admin", "editor"},
		},
		{
			name:           "rejects request without authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
			expectedRoles:  nil,
		},
		{
			name:           "rejects request with invalid authorization format",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
			expectedRoles:  nil,
		},
		{
			name:           "rejects request without Bearer prefix",
			authHeader:     "Basic " + validToken,
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
			expectedRoles:  nil,
		},
		{
			name:           "rejects request with invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
			expectedRoles:  nil,
		},
		{
			name:           "rejects request with malformed Bearer header",
			authHeader:     "Bearer",
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
			expectedRoles:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			var capturedUserID string
			var capturedRoles []string
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedUserID = GetUserID(r.Context())
				capturedRoles = GetUserRoles(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			// Create auth middleware
			middleware := Auth(authService)
			wrappedHandler := middleware(handler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Record response
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Auth middleware status = %v, want %v", rr.Code, tt.expectedStatus)
			}

			// Check user ID in context (only for successful requests)
			if tt.expectedStatus == http.StatusOK {
				if capturedUserID != tt.expectedUserID {
					t.Errorf("User ID in context = %v, want %v", capturedUserID, tt.expectedUserID)
				}

				// Check roles in context
				if len(capturedRoles) != len(tt.expectedRoles) {
					t.Errorf("Roles count = %v, want %v", len(capturedRoles), len(tt.expectedRoles))
				} else {
					for i, role := range tt.expectedRoles {
						if capturedRoles[i] != role {
							t.Errorf("Role[%d] = %v, want %v", i, capturedRoles[i], role)
						}
					}
				}
			}
		})
	}
}

func TestAuthWithConfig(t *testing.T) {
	authService := auth.NewAuthService("test-secret-key", time.Hour)
	validToken, _ := authService.GenerateToken("user-456", "user@example.com", []string{"viewer"})

	tests := []struct {
		name           string
		skipPaths      []string
		requestPath    string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "skips authentication for configured paths",
			skipPaths:      []string{"/health", "/public"},
			requestPath:    "/health",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "requires authentication for non-skipped paths",
			skipPaths:      []string{"/health"},
			requestPath:    "/api/posts",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "allows authenticated request on non-skipped path",
			skipPaths:      []string{"/health"},
			requestPath:    "/api/posts",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			config := AuthConfig{
				AuthService: authService,
				SkipPaths:   tt.skipPaths,
			}
			middleware := AuthWithConfig(config)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Auth middleware status = %v, want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestAuthWithExpiredToken(t *testing.T) {
	// Create auth service with very short TTL
	authService := auth.NewAuthService("test-secret-key", -time.Hour) // Already expired
	expiredToken, _ := authService.GenerateToken("user-expired", "user@example.com", []string{"admin"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(authService)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Auth middleware should reject expired token, got status %v", rr.Code)
	}
}

func TestAuthWithTokenWithoutUserID(t *testing.T) {
	authService := auth.NewAuthService("test-secret-key", time.Hour)

	// Manually create a token without user_id claim
	token, _ := authService.GenerateToken("", "user@example.com", []string{"admin"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(authService)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Auth middleware should reject token without user_id, got status %v", rr.Code)
	}
}

func TestAuthWithTokenWithoutRoles(t *testing.T) {
	authService := auth.NewAuthService("test-secret-key", time.Hour)
	tokenWithoutRoles, _ := authService.GenerateToken("user-no-roles", "user@example.com", []string{})

	var capturedUserID string
	var capturedRoles []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = GetUserID(r.Context())
		capturedRoles = GetUserRoles(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(authService)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenWithoutRoles)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Auth middleware status = %v, want %v", rr.Code, http.StatusOK)
	}

	if capturedUserID != "user-no-roles" {
		t.Errorf("User ID in context = %v, want user-no-roles", capturedUserID)
	}

	if len(capturedRoles) != 0 {
		t.Errorf("Roles should be empty, got %v", capturedRoles)
	}
}

func TestGetUserID(t *testing.T) {
	authService := auth.NewAuthService("test-secret-key", time.Hour)
	token, _ := authService.GenerateToken("test-user-id", "user@example.com", []string{"admin"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		if userID != "test-user-id" {
			t.Errorf("GetUserID() = %v, want test-user-id", userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(authService)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)
}

func TestGetUserRoles(t *testing.T) {
	authService := auth.NewAuthService("test-secret-key", time.Hour)
	expectedRoles := []string{"admin", "editor", "viewer"}
	token, _ := authService.GenerateToken("test-user", "user@example.com", expectedRoles)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles := GetUserRoles(r.Context())
		if len(roles) != len(expectedRoles) {
			t.Errorf("GetUserRoles() count = %v, want %v", len(roles), len(expectedRoles))
		}
		for i, role := range expectedRoles {
			if roles[i] != role {
				t.Errorf("GetUserRoles()[%d] = %v, want %v", i, roles[i], role)
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(authService)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)
}

func TestAuthWithDifferentSecretKeys(t *testing.T) {
	// Create token with one secret
	authService1 := auth.NewAuthService("secret-key-1", time.Hour)
	token, _ := authService1.GenerateToken("user-123", "user@example.com", []string{"admin"})

	// Try to validate with different secret
	authService2 := auth.NewAuthService("secret-key-2", time.Hour)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(authService2)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Auth middleware should reject token signed with different secret, got status %v", rr.Code)
	}
}

func BenchmarkAuthMiddleware(b *testing.B) {
	authService := auth.NewAuthService("test-secret-key", time.Hour)
	token, _ := authService.GenerateToken("bench-user", "user@example.com", []string{"admin"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(authService)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}
