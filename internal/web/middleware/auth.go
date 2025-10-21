package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/conduit-lang/conduit/internal/web/auth"
	webcontext "github.com/conduit-lang/conduit/internal/web/context"
)

// AuthConfig holds configuration for authentication middleware
type AuthConfig struct {
	// AuthService is used to validate tokens
	AuthService *auth.AuthService
	// SkipPaths is a list of paths to skip authentication
	SkipPaths []string
}

// Auth creates an authentication middleware with the given auth service
func Auth(authService *auth.AuthService) Middleware {
	return AuthWithConfig(AuthConfig{
		AuthService: authService,
		SkipPaths:   []string{},
	})
}

// AuthWithConfig creates an authentication middleware with custom configuration
func AuthWithConfig(config AuthConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range config.SkipPaths {
				if r.URL.Path == skipPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization required", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := config.AuthService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Extract user ID from claims
			userID, ok := claims["user_id"].(string)
			if !ok || userID == "" {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			// Extract roles from claims (optional)
			var roles []string
			if rolesInterface, ok := claims["roles"].([]interface{}); ok {
				for _, role := range rolesInterface {
					if roleStr, ok := role.(string); ok {
						roles = append(roles, roleStr)
					}
				}
			}

			// Add user ID and roles to context
			ctx := webcontext.SetCurrentUser(r.Context(), userID)
			if len(roles) > 0 {
				ctx = webcontext.SetUserRoles(ctx, roles)
			}
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) string {
	return webcontext.GetCurrentUser(ctx)
}

// GetUserRoles extracts the user roles from the request context
func GetUserRoles(ctx context.Context) []string {
	roles := webcontext.GetUserRoles(ctx)
	if roles == nil {
		return []string{}
	}
	return roles
}
