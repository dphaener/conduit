package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockPermissionChecker is a mock implementation of PermissionChecker for testing
type mockPermissionChecker struct {
	permissions map[string]map[string]map[Permission]bool // userID -> resource -> permission -> hasPermission
	checkError  error
}

func newMockPermissionChecker() *mockPermissionChecker {
	return &mockPermissionChecker{
		permissions: make(map[string]map[string]map[Permission]bool),
	}
}

func (m *mockPermissionChecker) HasPermission(ctx context.Context, userID string, resource string, permission Permission) (bool, error) {
	if m.checkError != nil {
		return false, m.checkError
	}

	if userPerms, ok := m.permissions[userID]; ok {
		if resourcePerms, ok := userPerms[resource]; ok {
			if hasPermission, ok := resourcePerms[permission]; ok {
				return hasPermission, nil
			}
		}
	}
	return false, nil
}

func (m *mockPermissionChecker) grantPermission(userID, resource string, permission Permission) {
	if _, ok := m.permissions[userID]; !ok {
		m.permissions[userID] = make(map[string]map[Permission]bool)
	}
	if _, ok := m.permissions[userID][resource]; !ok {
		m.permissions[userID][resource] = make(map[Permission]bool)
	}
	m.permissions[userID][resource][permission] = true
}

func TestNewAuthorizer(t *testing.T) {
	checker := newMockPermissionChecker()
	authorizer := NewAuthorizer(checker)

	if authorizer == nil {
		t.Fatal("NewAuthorizer() returned nil")
	}
	if authorizer.permissionChecker != checker {
		t.Error("NewAuthorizer() did not set permissionChecker correctly")
	}
}

func TestAuthorizerMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupChecker   func(*mockPermissionChecker)
		setupRequest   func(*http.Request)
		resource       string
		permission     Permission
		expectedStatus int
	}{
		{
			name: "allows user with permission",
			setupChecker: func(checker *mockPermissionChecker) {
				checker.grantPermission("user-123", "posts", PermissionCreate)
			},
			setupRequest: func(req *http.Request) {
				ctx := SetCurrentUser(req.Context(), "user-123")
				*req = *req.WithContext(ctx)
			},
			resource:       "posts",
			permission:     PermissionCreate,
			expectedStatus: http.StatusOK,
		},
		{
			name: "denies user without permission",
			setupChecker: func(checker *mockPermissionChecker) {
				// User has read permission, but not create
				checker.grantPermission("user-456", "posts", PermissionRead)
			},
			setupRequest: func(req *http.Request) {
				ctx := SetCurrentUser(req.Context(), "user-456")
				*req = *req.WithContext(ctx)
			},
			resource:       "posts",
			permission:     PermissionCreate,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:         "rejects unauthenticated user",
			setupChecker: func(checker *mockPermissionChecker) {},
			setupRequest: func(req *http.Request) {
				// No user in context
			},
			resource:       "posts",
			permission:     PermissionRead,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "denies user with no permissions",
			setupChecker: func(checker *mockPermissionChecker) {},
			setupRequest: func(req *http.Request) {
				ctx := SetCurrentUser(req.Context(), "user-no-perms")
				*req = *req.WithContext(ctx)
			},
			resource:       "posts",
			permission:     PermissionDelete,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock checker
			checker := newMockPermissionChecker()
			tt.setupChecker(checker)

			// Create authorizer
			authorizer := NewAuthorizer(checker)

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with authorization middleware
			middleware := authorizer.Middleware(tt.resource, tt.permission)
			wrappedHandler := middleware(handler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)

			// Record response
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Middleware() status = %v, want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestPermissionTypes(t *testing.T) {
	permissions := []Permission{
		PermissionCreate,
		PermissionRead,
		PermissionUpdate,
		PermissionDelete,
	}

	expectedValues := []string{"create", "read", "update", "delete"}

	for i, perm := range permissions {
		if string(perm) != expectedValues[i] {
			t.Errorf("Permission %d value = %v, want %v", i, perm, expectedValues[i])
		}
	}
}

func TestMultipleResourcePermissions(t *testing.T) {
	checker := newMockPermissionChecker()

	// Grant different permissions for different resources
	checker.grantPermission("user-multi", "posts", PermissionCreate)
	checker.grantPermission("user-multi", "posts", PermissionRead)
	checker.grantPermission("user-multi", "comments", PermissionRead)

	tests := []struct {
		resource       string
		permission     Permission
		expectedAccess bool
	}{
		{"posts", PermissionCreate, true},
		{"posts", PermissionRead, true},
		{"posts", PermissionUpdate, false},
		{"posts", PermissionDelete, false},
		{"comments", PermissionRead, true},
		{"comments", PermissionCreate, false},
	}

	for _, tt := range tests {
		hasPermission, err := checker.HasPermission(
			context.Background(),
			"user-multi",
			tt.resource,
			tt.permission,
		)
		if err != nil {
			t.Fatalf("HasPermission() error = %v", err)
		}
		if hasPermission != tt.expectedAccess {
			t.Errorf("HasPermission(%s, %s) = %v, want %v",
				tt.resource, tt.permission, hasPermission, tt.expectedAccess)
		}
	}
}

func TestAuthorizerWithCheckerError(t *testing.T) {
	checker := newMockPermissionChecker()
	checker.checkError = errors.New("database connection failed")

	authorizer := NewAuthorizer(checker)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := authorizer.Middleware("posts", PermissionRead)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetCurrentUser(req.Context(), "user-error")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Middleware() with checker error status = %v, want %v",
			rr.Code, http.StatusInternalServerError)
	}
}

func TestRequirePermission(t *testing.T) {
	checker := newMockPermissionChecker()
	checker.grantPermission("user-helper", "articles", PermissionUpdate)

	// Use the helper function
	middleware := RequirePermission(checker, "articles", PermissionUpdate)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Test with permission
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetCurrentUser(req.Context(), "user-helper")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("RequirePermission() with permission status = %v, want %v",
			rr.Code, http.StatusOK)
	}

	// Test without permission
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx2 := SetCurrentUser(req2.Context(), "user-no-perm")
	req2 = req2.WithContext(ctx2)

	rr2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusForbidden {
		t.Errorf("RequirePermission() without permission status = %v, want %v",
			rr2.Code, http.StatusForbidden)
	}
}

func TestAuthorizerWithAllPermissions(t *testing.T) {
	checker := newMockPermissionChecker()
	userID := "admin-user"
	resource := "documents"

	// Grant all permissions
	checker.grantPermission(userID, resource, PermissionCreate)
	checker.grantPermission(userID, resource, PermissionRead)
	checker.grantPermission(userID, resource, PermissionUpdate)
	checker.grantPermission(userID, resource, PermissionDelete)

	authorizer := NewAuthorizer(checker)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test each permission
	permissions := []Permission{
		PermissionCreate,
		PermissionRead,
		PermissionUpdate,
		PermissionDelete,
	}

	for _, perm := range permissions {
		t.Run(string(perm), func(t *testing.T) {
			middleware := authorizer.Middleware(resource, perm)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := SetCurrentUser(req.Context(), userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Middleware() for %s permission status = %v, want %v",
					perm, rr.Code, http.StatusOK)
			}
		})
	}
}

func TestPermissionCheckerInterface(t *testing.T) {
	// Verify that mockPermissionChecker implements PermissionChecker
	var _ PermissionChecker = (*mockPermissionChecker)(nil)
}
