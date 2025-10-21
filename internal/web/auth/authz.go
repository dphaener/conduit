package auth

import (
	"context"
	"net/http"
)

// Permission represents a resource permission type
type Permission string

const (
	// PermissionCreate allows creating new resources
	PermissionCreate Permission = "create"
	// PermissionRead allows reading resources
	PermissionRead Permission = "read"
	// PermissionUpdate allows updating resources
	PermissionUpdate Permission = "update"
	// PermissionDelete allows deleting resources
	PermissionDelete Permission = "delete"
)

// PermissionChecker defines the interface for checking user permissions
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID string, resource string, permission Permission) (bool, error)
}

// Authorizer handles authorization checks
type Authorizer struct {
	permissionChecker PermissionChecker
}

// NewAuthorizer creates a new authorizer with the given permission checker
func NewAuthorizer(permissionChecker PermissionChecker) *Authorizer {
	return &Authorizer{
		permissionChecker: permissionChecker,
	}
}

// Middleware returns a middleware function that checks permissions for a specific resource and action
func (a *Authorizer) Middleware(resource string, permission Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context
			userID := GetCurrentUser(r.Context())
			if userID == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check permission
			hasPermission, err := a.permissionChecker.HasPermission(
				r.Context(),
				userID,
				resource,
				permission,
			)
			if err != nil {
				http.Error(w, "Error checking permissions", http.StatusInternalServerError)
				return
			}

			if !hasPermission {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission is a helper function that creates an authorization middleware
// for a specific resource and permission
func RequirePermission(checker PermissionChecker, resource string, permission Permission) func(http.Handler) http.Handler {
	authorizer := NewAuthorizer(checker)
	return authorizer.Middleware(resource, permission)
}
