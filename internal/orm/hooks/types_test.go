package hooks

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestRegistry_NewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}

	if registry.hooks == nil {
		t.Fatal("Registry hooks map is nil")
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	hook := &Hook{
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return nil
		},
	}

	registry.Register(schema.BeforeCreate, hook)

	hooks := registry.GetHooks(schema.BeforeCreate)
	if len(hooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(hooks))
	}

	if hooks[0] != hook {
		t.Error("Retrieved hook does not match registered hook")
	}

	// Verify type was set on the hook
	if hooks[0].Type != schema.BeforeCreate {
		t.Errorf("Expected hook type %v, got %v", schema.BeforeCreate, hooks[0].Type)
	}
}

func TestRegistry_RegisterMultiple(t *testing.T) {
	registry := NewRegistry()

	hook1 := &Hook{
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return nil
		},
	}

	hook2 := &Hook{
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return nil
		},
	}

	registry.Register(schema.BeforeCreate, hook1)
	registry.Register(schema.BeforeCreate, hook2)

	hooks := registry.GetHooks(schema.BeforeCreate)
	if len(hooks) != 2 {
		t.Errorf("Expected 2 hooks, got %d", len(hooks))
	}
}

func TestRegistry_GetHooks_NoHooks(t *testing.T) {
	registry := NewRegistry()

	hooks := registry.GetHooks(schema.BeforeCreate)
	if hooks != nil {
		t.Errorf("Expected nil for no hooks, got %v", hooks)
	}
}

func TestRegistry_HasHooks(t *testing.T) {
	registry := NewRegistry()

	if registry.HasHooks(schema.BeforeCreate) {
		t.Error("Should not have hooks initially")
	}

	hook := &Hook{
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return nil
		},
	}

	registry.Register(schema.BeforeCreate, hook)

	if !registry.HasHooks(schema.BeforeCreate) {
		t.Error("Should have hooks after registration")
	}

	if registry.HasHooks(schema.AfterCreate) {
		t.Error("Should not have hooks for different type")
	}
}

func TestRegistry_DifferentHookTypes(t *testing.T) {
	registry := NewRegistry()

	beforeHook := &Hook{
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return nil
		},
	}

	afterHook := &Hook{
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return nil
		},
	}

	registry.Register(schema.BeforeCreate, beforeHook)
	registry.Register(schema.AfterCreate, afterHook)

	beforeHooks := registry.GetHooks(schema.BeforeCreate)
	afterHooks := registry.GetHooks(schema.AfterCreate)

	if len(beforeHooks) != 1 {
		t.Errorf("Expected 1 before hook, got %d", len(beforeHooks))
	}

	if len(afterHooks) != 1 {
		t.Errorf("Expected 1 after hook, got %d", len(afterHooks))
	}
}

func TestHook_Fields(t *testing.T) {
	hook := &Hook{
		Type:        schema.BeforeCreate,
		Transaction: true,
		Async:       false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return nil
		},
	}

	if hook.Type != schema.BeforeCreate {
		t.Error("Hook type not set correctly")
	}

	if !hook.Transaction {
		t.Error("Hook transaction flag not set correctly")
	}

	if hook.Async {
		t.Error("Hook async flag should be false")
	}

	if hook.Fn == nil {
		t.Error("Hook function should not be nil")
	}
}

func TestContextKey_String(t *testing.T) {
	key := ContextKeyTransaction

	// ContextKey should be usable as string
	str := string(key)
	if str != "transaction" {
		t.Errorf("Expected 'transaction', got '%s'", str)
	}
}
