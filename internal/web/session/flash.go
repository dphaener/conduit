package session

import (
	"context"
)

// Flash message types
const (
	FlashSuccess = "success"
	FlashError   = "error"
	FlashWarning = "warning"
	FlashInfo    = "info"
)

// AddFlash adds a flash message to the current session
func AddFlash(ctx context.Context, messageType, message string) error {
	sess := GetSession(ctx)
	if sess == nil {
		return ErrSessionNotFound
	}
	sess.AddFlash(messageType, message)
	return nil
}

// AddFlashSuccess adds a success flash message
func AddFlashSuccess(ctx context.Context, message string) error {
	return AddFlash(ctx, FlashSuccess, message)
}

// AddFlashError adds an error flash message
func AddFlashError(ctx context.Context, message string) error {
	return AddFlash(ctx, FlashError, message)
}

// AddFlashWarning adds a warning flash message
func AddFlashWarning(ctx context.Context, message string) error {
	return AddFlash(ctx, FlashWarning, message)
}

// AddFlashInfo adds an info flash message
func AddFlashInfo(ctx context.Context, message string) error {
	return AddFlash(ctx, FlashInfo, message)
}

// GetFlashes retrieves all flash messages from the current session and clears them
func GetFlashes(ctx context.Context) []FlashMessage {
	sess := GetSession(ctx)
	if sess == nil {
		return []FlashMessage{}
	}
	return sess.GetFlashes()
}

// GetFlashesByType retrieves flash messages of a specific type and clears all flashes
func GetFlashesByType(ctx context.Context, messageType string) []FlashMessage {
	allFlashes := GetFlashes(ctx)
	var filtered []FlashMessage
	for _, flash := range allFlashes {
		if flash.Type == messageType {
			filtered = append(filtered, flash)
		}
	}
	return filtered
}

// HasFlashes checks if there are any flash messages in the current session
func HasFlashes(ctx context.Context) bool {
	sess := GetSession(ctx)
	if sess == nil {
		return false
	}
	return len(sess.FlashMessages) > 0
}

// HasFlashesByType checks if there are flash messages of a specific type
func HasFlashesByType(ctx context.Context, messageType string) bool {
	sess := GetSession(ctx)
	if sess == nil {
		return false
	}
	for _, flash := range sess.FlashMessages {
		if flash.Type == messageType {
			return true
		}
	}
	return false
}

// PeekFlashes retrieves all flash messages without clearing them
func PeekFlashes(ctx context.Context) []FlashMessage {
	sess := GetSession(ctx)
	if sess == nil {
		return []FlashMessage{}
	}
	return sess.FlashMessages
}

// ClearFlashes removes all flash messages from the session
func ClearFlashes(ctx context.Context) {
	sess := GetSession(ctx)
	if sess != nil {
		sess.FlashMessages = []FlashMessage{}
	}
}
