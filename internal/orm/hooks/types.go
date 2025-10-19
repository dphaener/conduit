package hooks

import (
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// HookFunc represents a hook function that can be executed
// It receives the hook context and the resource record as a map
type HookFunc func(ctx *Context, record map[string]interface{}) error

// Hook represents a registered lifecycle hook
type Hook struct {
	Type        schema.HookType
	Fn          HookFunc
	Transaction bool // Execute in transaction context
	Async       bool // Execute asynchronously after response
}

// Registry manages all registered hooks for a resource
type Registry struct {
	hooks map[schema.HookType][]*Hook
}

// NewRegistry creates a new hook registry
func NewRegistry() *Registry {
	return &Registry{
		hooks: make(map[schema.HookType][]*Hook),
	}
}

// Register adds a hook to the registry
func (r *Registry) Register(hookType schema.HookType, hook *Hook) {
	hook.Type = hookType
	r.hooks[hookType] = append(r.hooks[hookType], hook)
}

// GetHooks returns all hooks for a given type
func (r *Registry) GetHooks(hookType schema.HookType) []*Hook {
	return r.hooks[hookType]
}

// HasHooks returns true if there are any hooks registered for the given type
func (r *Registry) HasHooks(hookType schema.HookType) bool {
	hooks := r.hooks[hookType]
	return len(hooks) > 0
}

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

const (
	// ContextKeyTransaction is the key for the transaction in the context
	ContextKeyTransaction ContextKey = "transaction"
	// ContextKeyDB is the key for the database connection in the context
	ContextKeyDB ContextKey = "database"
	// ContextKeyResource is the key for the resource schema in the context
	ContextKeyResource ContextKey = "resource"
)
