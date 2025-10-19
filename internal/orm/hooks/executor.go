package hooks

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Executor executes lifecycle hooks for resources
type Executor struct {
	registry   *Registry
	asyncQueue *AsyncQueue
}

// NewExecutor creates a new hook executor
func NewExecutor(asyncQueue *AsyncQueue) *Executor {
	return &Executor{
		registry:   NewRegistry(),
		asyncQueue: asyncQueue,
	}
}

// NewExecutorWithRegistry creates a new hook executor with an existing registry
func NewExecutorWithRegistry(registry *Registry, asyncQueue *AsyncQueue) *Executor {
	return &Executor{
		registry:   registry,
		asyncQueue: asyncQueue,
	}
}

// Register registers a hook
func (e *Executor) Register(hookType schema.HookType, hook *Hook) {
	e.registry.Register(hookType, hook)
}

// ExecuteHooks executes all hooks for a given type
// This method implements the HookExecutor interface expected by CRUD operations
func (e *Executor) ExecuteHooks(
	ctx context.Context,
	resource *schema.ResourceSchema,
	hookType schema.HookType,
	record map[string]interface{},
) error {
	// Get hooks from registry
	hooks := e.registry.GetHooks(hookType)
	if len(hooks) == 0 {
		return nil
	}

	// Create hook context
	hookCtx := NewContext(ctx, nil, resource)

	// Extract transaction from context if available
	if tx, ok := ctx.Value(ContextKeyTransaction).(*sql.Tx); ok {
		hookCtx = hookCtx.WithTransaction(tx)
	}

	// Extract database from context if available
	if db, ok := ctx.Value(ContextKeyDB).(*sql.DB); ok {
		hookCtx.db = db
	}

	// Execute hooks in order
	for _, hook := range hooks {
		if hook.Async {
			// Enqueue async hooks to execute after response
			if err := e.enqueueAsyncHook(hookCtx, hook, record); err != nil {
				// Log error but don't fail the operation for async hooks
				log.Printf("failed to enqueue async hook %s: %v", hookType.String(), err)
			}
		} else {
			// Execute synchronous hooks immediately
			if err := hook.Fn(hookCtx, record); err != nil {
				return fmt.Errorf("hook %s failed: %w", hookType.String(), err)
			}
		}
	}

	return nil
}

// deepCopyRecord creates a deep copy of a record map to ensure
// async hooks have fully isolated data
func deepCopyRecord(record map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{}, len(record))
	for k, v := range record {
		copy[k] = deepCopyValue(v)
	}
	return copy
}

// deepCopyValue recursively copies values to ensure full isolation
func deepCopyValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]interface{}:
		// Recursively copy nested maps
		return deepCopyRecord(val)
	case []interface{}:
		// Copy slices
		copySlice := make([]interface{}, len(val))
		for i, item := range val {
			copySlice[i] = deepCopyValue(item)
		}
		return copySlice
	case []string:
		// Common case - copy string slices
		copySlice := make([]string, len(val))
		copy(copySlice, val)
		return copySlice
	case []int:
		// Copy int slices
		copySlice := make([]int, len(val))
		copy(copySlice, val)
		return copySlice
	case []int64:
		// Copy int64 slices
		copySlice := make([]int64, len(val))
		copy(copySlice, val)
		return copySlice
	case []float64:
		// Copy float64 slices
		copySlice := make([]float64, len(val))
		copy(copySlice, val)
		return copySlice
	case []bool:
		// Copy bool slices
		copySlice := make([]bool, len(val))
		copy(copySlice, val)
		return copySlice
	default:
		// Primitive types (string, int, bool, time.Time, uuid.UUID, etc.)
		// are copied by value, so just return them
		return v
	}
}

// enqueueAsyncHook queues an async hook for later execution
func (e *Executor) enqueueAsyncHook(
	hookCtx *Context,
	hook *Hook,
	record map[string]interface{},
) error {
	if e.asyncQueue == nil {
		return fmt.Errorf("async queue not configured")
	}

	// Make a deep copy of the record to avoid mutation issues
	recordCopy := deepCopyRecord(record)

	task := AsyncTask{
		Name: fmt.Sprintf("%s_hook", hook.Type.String()),
		Fn: func(ctx context.Context) error {
			// Create a new context for async execution
			asyncCtx := NewContext(ctx, hookCtx.db, hookCtx.resource)

			// Execute the hook
			if err := hook.Fn(asyncCtx, recordCopy); err != nil {
				// Log error but don't propagate (async hooks don't affect operation)
				log.Printf("async hook %s failed: %v", hook.Type.String(), err)
				return err
			}
			return nil
		},
	}

	return e.asyncQueue.Enqueue(task)
}

// ExecuteHooksFromSchema executes hooks defined in the resource schema
// This is used when hooks are defined in Conduit source files
func (e *Executor) ExecuteHooksFromSchema(
	ctx context.Context,
	resource *schema.ResourceSchema,
	hookType schema.HookType,
	record map[string]interface{},
	db *sql.DB,
	tx *sql.Tx,
) error {
	// Get hooks from schema
	schemaHooks := resource.Hooks[hookType]
	if len(schemaHooks) == 0 {
		return nil
	}

	// Create hook context
	hookCtx := NewContext(ctx, db, resource)
	if tx != nil {
		hookCtx = hookCtx.WithTransaction(tx)
	}

	// Execute hooks in order
	for _, schemaHook := range schemaHooks {
		if schemaHook.Async {
			// Enqueue async hooks
			if err := e.enqueueSchemaAsyncHook(hookCtx, schemaHook, hookType, record); err != nil {
				log.Printf("failed to enqueue async schema hook %s: %v", hookType.String(), err)
			}
		} else {
			// Execute synchronous hooks
			// Note: Actual execution of schema hook body would be done by
			// generated code or interpreter - this is a stub for now
			if err := e.executeSchemaHook(hookCtx, schemaHook, record); err != nil {
				return fmt.Errorf("schema hook %s failed: %w", hookType.String(), err)
			}
		}
	}

	return nil
}

// executeSchemaHook executes a hook from the schema
// This is a stub - actual implementation would execute the AST body
func (e *Executor) executeSchemaHook(
	ctx *Context,
	hook *schema.Hook,
	record map[string]interface{},
) error {
	// TODO: This would be implemented by the compiler/runtime
	// For now, this is a placeholder that validates the hook structure
	if len(hook.Body) == 0 {
		return nil
	}

	// In the full implementation, this would:
	// 1. Create an execution context with 'self' bound to the record
	// 2. Execute each statement in hook.Body
	// 3. Handle errors and return them

	return nil
}

// enqueueSchemaAsyncHook queues an async schema hook
func (e *Executor) enqueueSchemaAsyncHook(
	hookCtx *Context,
	hook *schema.Hook,
	hookType schema.HookType,
	record map[string]interface{},
) error {
	if e.asyncQueue == nil {
		return fmt.Errorf("async queue not configured")
	}

	// Make a deep copy of the record to avoid mutation issues
	recordCopy := deepCopyRecord(record)

	task := AsyncTask{
		Name: fmt.Sprintf("%s_schema_hook", hookType.String()),
		Fn: func(ctx context.Context) error {
			asyncCtx := NewContext(ctx, hookCtx.db, hookCtx.resource)
			if err := e.executeSchemaHook(asyncCtx, hook, recordCopy); err != nil {
				log.Printf("async schema hook %s failed: %v", hookType.String(), err)
				return err
			}
			return nil
		},
	}

	return e.asyncQueue.Enqueue(task)
}

// HasHooks returns true if there are any hooks registered for the given type
func (e *Executor) HasHooks(hookType schema.HookType) bool {
	return e.registry.HasHooks(hookType)
}

// GetRegistry returns the hook registry
func (e *Executor) GetRegistry() *Registry {
	return e.registry
}
