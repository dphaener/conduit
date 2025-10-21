package context

import "context"

// contextKey is a custom type for context keys to avoid collisions
type contextKey int

const (
	requestIDKey contextKey = iota
	currentUserKey
	userRolesKey
)

// GetRequestID extracts the request ID from the context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// SetRequestID adds the request ID to the context
func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// GetCurrentUser extracts the current user ID from the context
func GetCurrentUser(ctx context.Context) string {
	if user, ok := ctx.Value(currentUserKey).(string); ok {
		return user
	}
	return ""
}

// SetCurrentUser adds the current user ID to the context
func SetCurrentUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, currentUserKey, user)
}

// GetUserRoles extracts the user roles from the context
func GetUserRoles(ctx context.Context) []string {
	if roles, ok := ctx.Value(userRolesKey).([]string); ok {
		return roles
	}
	return nil
}

// SetUserRoles adds the user roles to the context
func SetUserRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, userRolesKey, roles)
}
