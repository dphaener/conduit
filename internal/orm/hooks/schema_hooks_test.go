package hooks

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestExecutor_ExecuteHooksFromSchema(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)

	// Create resource with schema hooks
	resource := schema.NewResourceSchema("Post")
	resource.Hooks[schema.BeforeCreate] = []*schema.Hook{
		{
			Type:        schema.BeforeCreate,
			Transaction: true,
			Async:       false,
			Body:        []ast.StmtNode{}, // Empty body for now
		},
	}

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooksFromSchema(
		context.Background(),
		resource,
		schema.BeforeCreate,
		record,
		nil,
		nil,
	)

	if err != nil {
		t.Errorf("ExecuteHooksFromSchema failed: %v", err)
	}
}

func TestExecutor_ExecuteHooksFromSchema_NoHooks(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooksFromSchema(
		context.Background(),
		resource,
		schema.BeforeCreate,
		record,
		nil,
		nil,
	)

	if err != nil {
		t.Errorf("ExecuteHooksFromSchema with no hooks should not fail: %v", err)
	}
}

func TestExecutor_ExecuteHooksFromSchema_WithTransaction(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)

	resource := schema.NewResourceSchema("Post")
	resource.Hooks[schema.BeforeCreate] = []*schema.Hook{
		{
			Type:        schema.BeforeCreate,
			Transaction: true,
			Async:       false,
			Body:        []ast.StmtNode{},
		},
	}

	record := map[string]interface{}{"title": "Test"}

	// Mock transaction
	tx := &sql.Tx{}
	db := &sql.DB{}

	err := executor.ExecuteHooksFromSchema(
		context.Background(),
		resource,
		schema.BeforeCreate,
		record,
		db,
		tx,
	)

	if err != nil {
		t.Errorf("ExecuteHooksFromSchema with transaction failed: %v", err)
	}
}

func TestExecutor_ExecuteHooksFromSchema_Async(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)

	resource := schema.NewResourceSchema("Post")
	resource.Hooks[schema.AfterCreate] = []*schema.Hook{
		{
			Type:  schema.AfterCreate,
			Async: true,
			Body:  []ast.StmtNode{},
		},
	}

	record := map[string]interface{}{"title": "Test"}
	db := &sql.DB{}

	err := executor.ExecuteHooksFromSchema(
		context.Background(),
		resource,
		schema.AfterCreate,
		record,
		db,
		nil,
	)

	if err != nil {
		t.Errorf("ExecuteHooksFromSchema async failed: %v", err)
	}

	// Wait for async execution
	time.Sleep(100 * time.Millisecond)
}

func TestExecutor_ExecuteHooksFromSchema_Multiple(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)

	resource := schema.NewResourceSchema("Post")
	resource.Hooks[schema.BeforeCreate] = []*schema.Hook{
		{
			Type: schema.BeforeCreate,
			Body: []ast.StmtNode{},
		},
		{
			Type: schema.BeforeCreate,
			Body: []ast.StmtNode{},
		},
	}

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooksFromSchema(
		context.Background(),
		resource,
		schema.BeforeCreate,
		record,
		nil,
		nil,
	)

	if err != nil {
		t.Errorf("ExecuteHooksFromSchema with multiple hooks failed: %v", err)
	}
}

func TestExecutor_GetRegistry(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	registry := executor.GetRegistry()

	if registry == nil {
		t.Error("GetRegistry returned nil")
	}

	// Should be the same registry instance
	if registry != executor.registry {
		t.Error("GetRegistry returned different instance")
	}
}

func TestExecutor_NewExecutorWithRegistry(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	registry := NewRegistry()
	hook := &Hook{
		Type: schema.BeforeCreate,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			return nil
		},
	}
	registry.Register(schema.BeforeCreate, hook)

	executor := NewExecutorWithRegistry(registry, queue)

	if executor.GetRegistry() != registry {
		t.Error("Executor not using provided registry")
	}

	if !executor.HasHooks(schema.BeforeCreate) {
		t.Error("Executor should have hooks from registry")
	}
}

func TestExecutor_ExecuteHooks_WithContextValues(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	executed := false
	hook := &Hook{
		Type: schema.BeforeCreate,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			// Verify context has transaction
			if ctx.HasTransaction() {
				executed = true
			}
			return nil
		},
	}

	executor.Register(schema.BeforeCreate, hook)

	// Create context with transaction
	tx := &sql.Tx{}
	ctx := context.WithValue(context.Background(), ContextKeyTransaction, tx)

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooks(ctx, resource, schema.BeforeCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	if !executed {
		t.Error("Hook did not see transaction in context")
	}
}

func TestExecutor_ExecuteHooks_WithDBInContext(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	var capturedDB *sql.DB
	hook := &Hook{
		Type: schema.BeforeCreate,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			capturedDB = ctx.DB()
			return nil
		},
	}

	executor.Register(schema.BeforeCreate, hook)

	// Create context with DB
	db := &sql.DB{}
	ctx := context.WithValue(context.Background(), ContextKeyDB, db)

	record := map[string]interface{}{"title": "Test"}

	err := executor.ExecuteHooks(ctx, resource, schema.BeforeCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	if capturedDB != db {
		t.Error("Hook did not receive DB from context")
	}
}

func TestExecutor_AsyncHookRecordCopy(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executor := NewExecutor(queue)
	resource := schema.NewResourceSchema("Post")

	asyncDone := make(chan map[string]interface{}, 1)
	hook := &Hook{
		Type:  schema.AfterCreate,
		Async: true,
		Fn: func(ctx *Context, record map[string]interface{}) error {
			asyncDone <- record
			return nil
		},
	}

	executor.Register(schema.AfterCreate, hook)

	record := map[string]interface{}{"title": "Original", "id": 1}

	err := executor.ExecuteHooks(context.Background(), resource, schema.AfterCreate, record)
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	// Modify original record
	record["title"] = "Modified"
	record["id"] = 999

	// Get async record
	var asyncRecord map[string]interface{}
	select {
	case asyncRecord = <-asyncDone:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Async hook did not execute")
	}

	// Async hook should have original values
	if asyncRecord["title"] != "Original" {
		t.Errorf("Async hook got modified title: %v", asyncRecord["title"])
	}

	if asyncRecord["id"] != 1 {
		t.Errorf("Async hook got modified id: %v", asyncRecord["id"])
	}
}
