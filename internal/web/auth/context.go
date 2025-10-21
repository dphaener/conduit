package auth

import (
	"context"

	webcontext "github.com/conduit-lang/conduit/internal/web/context"
)

// GetCurrentUser retrieves the current user ID from the context
// Returns an empty string if no user is authenticated
func GetCurrentUser(ctx context.Context) string {
	return webcontext.GetCurrentUser(ctx)
}

// GetUserID is an alias for GetCurrentUser for backwards compatibility
func GetUserID(ctx context.Context) string {
	return GetCurrentUser(ctx)
}

// SetCurrentUser adds the user ID to the context
// Returns a new context with the user ID set
func SetCurrentUser(ctx context.Context, userID string) context.Context {
	return webcontext.SetCurrentUser(ctx, userID)
}
