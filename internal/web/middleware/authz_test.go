package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conduit-lang/conduit/internal/web/auth"
	webcontext "github.com/conduit-lang/conduit/internal/web/context"
)

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name           string
		permission     auth.RBACPermission
		userRoles      []string
		expectedStatus int
	}{
		{
			name:           "allows admin to create posts",
			permission:     auth.PostsCreate,
			userRoles:      []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "allows editor to create posts",
			permission:     auth.PostsCreate,
			userRoles:      []string{"editor"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "denies viewer from creating posts",
			permission:     auth.PostsCreate,
			userRoles:      []string{"viewer"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "allows viewer to read posts",
			permission:     auth.PostsRead,
			userRoles:      []string{"viewer"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "denies editor from deleting posts",
			permission:     auth.PostsDelete,
			userRoles:      []string{"editor"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "allows admin to delete posts",
			permission:     auth.PostsDelete,
			userRoles:      []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "denies user with no roles",
			permission:     auth.PostsRead,
			userRoles:      []string{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "allows user with multiple roles if any has permission",
			permission:     auth.PostsCreate,
			userRoles:      []string{"viewer", "editor"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequirePermission(tt.permission)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := webcontext.SetUserRoles(req.Context(), tt.userRoles)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("RequirePermission() status = %v, want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		requiredRole   string
		userRoles      []string
		expectedStatus int
	}{
		{
			name:           "allows user with required role",
			requiredRole:   "admin",
			userRoles:      []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "denies user without required role",
			requiredRole:   "admin",
			userRoles:      []string{"editor"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "allows user with multiple roles including required",
			requiredRole:   "editor",
			userRoles:      []string{"viewer", "editor", "admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "denies user with no roles",
			requiredRole:   "viewer",
			userRoles:      []string{},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "case sensitive role check",
			requiredRole:   "admin",
			userRoles:      []string{"Admin"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequireRole(tt.requiredRole)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := webcontext.SetUserRoles(req.Context(), tt.userRoles)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("RequireRole() status = %v, want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRequireAnyRole(t *testing.T) {
	tests := []struct {
		name           string
		requiredRoles  []string
		userRoles      []string
		expectedStatus int
	}{
		{
			name:           "allows user with one of required roles",
			requiredRoles:  []string{"admin", "editor"},
			userRoles:      []string{"editor"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "allows user with multiple matching roles",
			requiredRoles:  []string{"admin", "editor"},
			userRoles:      []string{"admin", "editor"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "denies user without any required role",
			requiredRoles:  []string{"admin", "editor"},
			userRoles:      []string{"viewer"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "denies user with no roles",
			requiredRoles:  []string{"admin", "editor"},
			userRoles:      []string{},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "allows user with first required role",
			requiredRoles:  []string{"admin", "editor", "viewer"},
			userRoles:      []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "allows user with last required role",
			requiredRoles:  []string{"admin", "editor", "viewer"},
			userRoles:      []string{"viewer"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequireAnyRole(tt.requiredRoles...)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := webcontext.SetUserRoles(req.Context(), tt.userRoles)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("RequireAnyRole() status = %v, want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRequirePermissionAllPermissions(t *testing.T) {
	permissions := []auth.RBACPermission{
		auth.PostsRead,
		auth.PostsCreate,
		auth.PostsUpdate,
		auth.PostsDelete,
		auth.UsersRead,
		auth.UsersCreate,
		auth.UsersUpdate,
		auth.UsersDelete,
		auth.SystemAdmin,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Admin should have all permissions
	for _, perm := range permissions {
		t.Run(string(perm)+" with admin", func(t *testing.T) {
			middleware := RequirePermission(perm)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := webcontext.SetUserRoles(req.Context(), []string{"admin"})
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Admin should have permission %v, got status %v", perm, rr.Code)
			}
		})
	}
}

func TestRequirePermissionWithoutContext(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequirePermission(auth.PostsRead)
	wrappedHandler := middleware(handler)

	// Request without roles in context
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("RequirePermission() should deny request without roles, got status %v", rr.Code)
	}
}

func TestAuthorizationMiddlewareChaining(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Chain multiple authorization middlewares
	chain := NewChain(
		RequireRole("admin"),
		RequirePermission(auth.SystemAdmin),
	)
	wrappedHandler := chain.Then(handler)

	tests := []struct {
		name           string
		userRoles      []string
		expectedStatus int
	}{
		{
			name:           "passes all checks with admin role",
			userRoles:      []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "fails role check",
			userRoles:      []string{"editor"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "fails with no roles",
			userRoles:      []string{},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := webcontext.SetUserRoles(req.Context(), tt.userRoles)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Chained middleware status = %v, want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRequirePermissionErrorMessages(t *testing.T) {
	tests := []struct {
		name               string
		permission         auth.RBACPermission
		userRoles          []string
		expectedStatus     int
		expectedBodyPrefix string
	}{
		{
			name:               "no roles returns unauthorized with auth message",
			permission:         auth.PostsRead,
			userRoles:          []string{},
			expectedStatus:     http.StatusUnauthorized,
			expectedBodyPrefix: "Unauthorized: authentication required",
		},
		{
			name:               "insufficient permissions returns forbidden message",
			permission:         auth.PostsDelete,
			userRoles:          []string{"viewer"},
			expectedStatus:     http.StatusForbidden,
			expectedBodyPrefix: "Forbidden: insufficient permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequirePermission(tt.permission)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := webcontext.SetUserRoles(req.Context(), tt.userRoles)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("RequirePermission() status = %v, want %v", rr.Code, tt.expectedStatus)
			}

			body := rr.Body.String()
			if !contains(body, tt.expectedBodyPrefix) {
				t.Errorf("Response body = %q, should contain %q", body, tt.expectedBodyPrefix)
			}
		})
	}
}

func TestRequireRoleWithEmptyRoleName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireRole("")
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := webcontext.SetUserRoles(req.Context(), []string{"admin"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("RequireRole('') should deny all requests, got status %v", rr.Code)
	}
}

func TestRequireAnyRoleWithNoRequiredRoles(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireAnyRole()
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := webcontext.SetUserRoles(req.Context(), []string{"admin"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("RequireAnyRole() with no required roles should deny all requests, got status %v", rr.Code)
	}
}

func BenchmarkRequirePermission(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequirePermission(auth.PostsCreate)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := webcontext.SetUserRoles(req.Context(), []string{"editor"})
	req = req.WithContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}

func BenchmarkRequireRole(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireRole("admin")
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := webcontext.SetUserRoles(req.Context(), []string{"admin"})
	req = req.WithContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && containsHelper(s[1:], substr)
}

func containsHelper(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsHelper(s[1:], substr)
}
