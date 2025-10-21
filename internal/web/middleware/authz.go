package middleware

import (
	"net/http"

	"github.com/conduit-lang/conduit/internal/web/auth"
)

// RequirePermission creates a middleware that checks if the user has a specific permission
// The permission should be in the format "resource.action" (e.g., "posts.create")
func RequirePermission(permission auth.RBACPermission) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user roles from context
			roles := GetUserRoles(r.Context())

			// Check if user is authenticated
			if len(roles) == 0 {
				http.Error(w, "Unauthorized: authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user has the required permission
			if !auth.UserHasPermission(roles, permission) {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates a middleware that checks if the user has a specific role
func RequireRole(roleName string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user roles from context
			roles := GetUserRoles(r.Context())

			// Check if user has the required role
			hasRole := false
			for _, role := range roles {
				if role == roleName {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, "Forbidden: role required", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole creates a middleware that checks if the user has any of the specified roles
func RequireAnyRole(roleNames ...string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user roles from context
			roles := GetUserRoles(r.Context())

			// Check if user has any of the required roles
			hasRole := false
			for _, userRole := range roles {
				for _, requiredRole := range roleNames {
					if userRole == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				http.Error(w, "Forbidden: one of the required roles needed", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
