package hooks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestExecutor_ExecuteHooks_Synchronous(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	executed := false
	hook := &Hook{
		Type:        schema.BeforeCreate,
		Transaction: true,
		Async:       false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			executed = true
			// Modify record to test mutation
			record["slug"] = "test-slug"
			return nil
		},
	}

	executor.Register(schema.BeforeCreate, hook)

	record := map[string]interface{}{
		"title": "Test Post",
	}

	err := executor.ExecuteHooks(context.Background(), resource, schema.BeforeCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	if !executed {
		t.Error("Hook was not executed")
	}

	if record["slug"] != "test-slug" {
		t.Errorf("Hook did not modify record, got slug: %v", record["slug"])
	}
}

func TestExecutor_ExecuteHooks_MultipleHooks(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	executionOrder := []string{}

	hook1 := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			executionOrder = append(executionOrder, "hook1")
			return nil
		},
	}

	hook2 := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			executionOrder = append(executionOrder, "hook2")
			return nil
		},
	}

	hook3 := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			executionOrder = append(executionOrder, "hook3")
			return nil
		},
	}

	executor.Register(schema.BeforeCreate, hook1)
	executor.Register(schema.BeforeCreate, hook2)
	executor.Register(schema.BeforeCreate, hook3)

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooks(context.Background(), resource, schema.BeforeCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 hooks to execute, got %d", len(executionOrder))
	}

	if executionOrder[0] != "hook1" || executionOrder[1] != "hook2" || executionOrder[2] != "hook3" {
		t.Errorf("Hooks executed in wrong order: %v", executionOrder)
	}
}

func TestExecutor_ExecuteHooks_Error(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	expectedErr := errors.New("validation failed")
	hook := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return expectedErr
		},
	}

	executor.Register(schema.BeforeCreate, hook)

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooks(context.Background(), resource, schema.BeforeCreate, record)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error to contain original error, got: %v", err)
	}
}

func TestExecutor_ExecuteHooks_ErrorStopsExecution(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	executed := map[string]bool{}

	hook1 := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			executed["hook1"] = true
			return nil
		},
	}

	hook2 := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			executed["hook2"] = true
			return errors.New("hook2 failed")
		},
	}

	hook3 := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			executed["hook3"] = true
			return nil
		},
	}

	executor.Register(schema.BeforeCreate, hook1)
	executor.Register(schema.BeforeCreate, hook2)
	executor.Register(schema.BeforeCreate, hook3)

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooks(context.Background(), resource, schema.BeforeCreate, record)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	if !executed["hook1"] {
		t.Error("hook1 should have executed")
	}

	if !executed["hook2"] {
		t.Error("hook2 should have executed")
	}

	if executed["hook3"] {
		t.Error("hook3 should not have executed after hook2 error")
	}
}

func TestExecutor_ExecuteHooks_Async(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	executed := make(chan bool, 1)
	hook := &Hook{
		Type:  schema.AfterCreate,
		Async: true,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			executed <- true
			return nil
		},
	}

	executor.Register(schema.AfterCreate, hook)

	record := map[string]interface{}{"title": "Test"}

	// Execute hooks - async should not block
	err := executor.ExecuteHooks(context.Background(), resource, schema.AfterCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	// Wait for async execution
	select {
	case <-executed:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Async hook did not execute within timeout")
	}
}

func TestExecutor_ExecuteHooks_AsyncError(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	hook := &Hook{
		Type:  schema.AfterCreate,
		Async: true,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return errors.New("async hook failed")
		},
	}

	executor.Register(schema.AfterCreate, hook)

	record := map[string]interface{}{"title": "Test"}

	// Async hook errors should not fail the operation
	err := executor.ExecuteHooks(context.Background(), resource, schema.AfterCreate, record)
	if err != nil {
		t.Fatalf("Async hook error should not fail ExecuteHooks: %v", err)
	}

	// Give async hook time to execute and log error
	time.Sleep(100 * time.Millisecond)
}

func TestExecutor_ExecuteHooks_MixedSyncAsync(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	executionOrder := []string{}
	asyncExecuted := make(chan bool, 1)

	syncHook := &Hook{
		Type:  schema.AfterCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			executionOrder = append(executionOrder, "sync")
			return nil
		},
	}

	asyncHook := &Hook{
		Type:  schema.AfterCreate,
		Async: true,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			asyncExecuted <- true
			return nil
		},
	}

	executor.Register(schema.AfterCreate, syncHook)
	executor.Register(schema.AfterCreate, asyncHook)

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooks(context.Background(), resource, schema.AfterCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	if len(executionOrder) != 1 || executionOrder[0] != "sync" {
		t.Errorf("Sync hook did not execute properly: %v", executionOrder)
	}

	// Wait for async execution
	select {
	case <-asyncExecuted:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Async hook did not execute within timeout")
	}
}

func TestExecutor_NoHooks(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooks(context.Background(), resource, schema.BeforeCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks with no hooks should not fail: %v", err)
	}
}

func TestExecutor_HasHooks(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)

	if executor.HasHooks(schema.BeforeCreate) {
		t.Error("Should not have hooks before registration")
	}

	hook := &Hook{
		Type:  schema.BeforeCreate,
		Async: false,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return nil
		},
	}

	executor.Register(schema.BeforeCreate, hook)

	if !executor.HasHooks(schema.BeforeCreate) {
		t.Error("Should have hooks after registration")
	}

	if executor.HasHooks(schema.BeforeUpdate) {
		t.Error("Should not have hooks for different type")
	}
}

func TestExecutor_RecordIsolation(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	asyncDone := make(chan bool, 1)

	asyncHook := &Hook{
		Type:  schema.AfterCreate,
		Async: true,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			// Sleep to ensure original record might be modified
			time.Sleep(50 * time.Millisecond)
			// This should have the original value
			if record["title"] != "Original" {
				t.Errorf("Async hook got modified record: %v", record["title"])
			}
			asyncDone <- true
			return nil
		},
	}

	executor.Register(schema.AfterCreate, asyncHook)

	record := map[string]interface{}{"title": "Original"}

	err := executor.ExecuteHooks(context.Background(), resource, schema.AfterCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	// Modify record after enqueuing
	record["title"] = "Modified"

	// Wait for async hook
	select {
	case <-asyncDone:
		// Success - async hook verified it has the original value
	case <-time.After(2 * time.Second):
		t.Error("Async hook did not execute within timeout")
	}
}

func TestExecutor_AsyncHookDeepCopyIsolation(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	// Test that async hooks receive a deep copy and don't see mutations
	record := map[string]interface{}{
		"title": "Test Post",
		"metadata": map[string]interface{}{
			"tags":   []string{"go", "test"},
			"counts": []int{1, 2, 3},
		},
		"settings": map[string]interface{}{
			"enabled": true,
			"nested": map[string]interface{}{
				"value": "original",
			},
		},
	}

	// Hook that verifies isolation
	hookCalled := make(chan bool, 1)
	hook := &Hook{
		Type:  schema.AfterCreate,
		Async: true,
		Fn: func(ctx *Context, data map[string]interface{}) error {
			// Verify nested map isolation
			metadata, ok := data["metadata"].(map[string]interface{})
			if !ok {
				t.Error("metadata is not a map")
				hookCalled <- false
				return nil
			}

			// Should NOT see the mutation made after enqueue
			if _, exists := metadata["author"]; exists {
				t.Error("async hook saw map mutation - deep copy failed")
				hookCalled <- false
				return nil
			}

			// Verify string slice isolation
			tags, ok := metadata["tags"].([]string)
			if !ok {
				t.Error("tags is not a string slice")
				hookCalled <- false
				return nil
			}
			if len(tags) != 2 {
				t.Errorf("tags length changed, expected 2, got %d", len(tags))
				hookCalled <- false
				return nil
			}
			if tags[0] != "go" || tags[1] != "test" {
				t.Errorf("tags values changed, got %v", tags)
				hookCalled <- false
				return nil
			}

			// Verify int slice isolation
			counts, ok := metadata["counts"].([]int)
			if !ok {
				t.Error("counts is not an int slice")
				hookCalled <- false
				return nil
			}
			if len(counts) != 3 {
				t.Errorf("counts length changed, expected 3, got %d", len(counts))
				hookCalled <- false
				return nil
			}

			// Verify deeply nested map isolation
			settings, ok := data["settings"].(map[string]interface{})
			if !ok {
				t.Error("settings is not a map")
				hookCalled <- false
				return nil
			}
			nested, ok := settings["nested"].(map[string]interface{})
			if !ok {
				t.Error("nested is not a map")
				hookCalled <- false
				return nil
			}
			if nested["value"] != "original" {
				t.Errorf("nested value changed, expected 'original', got %v", nested["value"])
				hookCalled <- false
				return nil
			}

			hookCalled <- true
			return nil
		},
	}

	executor.Register(schema.AfterCreate, hook)
	err := executor.ExecuteHooks(context.Background(), resource, schema.AfterCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	// Mutate the original record after enqueue to verify isolation
	// Add new key to nested map
	record["metadata"].(map[string]interface{})["author"] = "New Author"

	// Modify slice elements
	tags := record["metadata"].(map[string]interface{})["tags"].([]string)
	tags[0] = "modified"

	// Modify deeply nested value
	record["settings"].(map[string]interface{})["nested"].(map[string]interface{})["value"] = "mutated"

	// Wait for async hook
	select {
	case success := <-hookCalled:
		if !success {
			t.Fatal("async hook validation failed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("async hook not called within timeout")
	}
}
