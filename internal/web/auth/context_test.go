package auth

import (
	"context"
	"testing"

	webcontext "github.com/conduit-lang/conduit/internal/web/context"
)

func TestGetCurrentUser(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "returns user ID when present",
			ctx:      webcontext.SetCurrentUser(context.Background(), "user-123"),
			expected: "user-123",
		},
		{
			name:     "returns empty string when not present",
			ctx:      context.Background(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCurrentUser(tt.ctx)
			if result != tt.expected {
				t.Errorf("GetCurrentUser() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "returns user ID when present",
			ctx:      webcontext.SetCurrentUser(context.Background(), "user-456"),
			expected: "user-456",
		},
		{
			name:     "returns empty string when not present",
			ctx:      context.Background(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserID(tt.ctx)
			if result != tt.expected {
				t.Errorf("GetUserID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSetCurrentUser(t *testing.T) {
	tests := []struct {
		name   string
		userID string
	}{
		{
			name:   "sets user ID in context",
			userID: "user-789",
		},
		{
			name:   "sets empty user ID",
			userID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := SetCurrentUser(context.Background(), tt.userID)
			result := GetCurrentUser(ctx)
			if result != tt.userID {
				t.Errorf("SetCurrentUser() then GetCurrentUser() = %v, want %v", result, tt.userID)
			}
		})
	}
}

func TestContextKeyIsolation(t *testing.T) {
	// Test that our custom context key doesn't conflict with string keys
	ctx := context.Background()
	ctx = context.WithValue(ctx, "current_user", "wrong-user")
	ctx = webcontext.SetCurrentUser(ctx, "correct-user")

	result := webcontext.GetCurrentUser(ctx)
	if result != "correct-user" {
		t.Errorf("Context key isolation failed: got %v, want %v", result, "correct-user")
	}

	// Verify the string key is still accessible
	if stringVal := ctx.Value("current_user"); stringVal != "wrong-user" {
		t.Errorf("String key was overwritten: got %v, want %v", stringVal, "wrong-user")
	}
}
