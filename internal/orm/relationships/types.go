// Package relationships provides efficient relationship loading for the Conduit ORM
package relationships

import (
	"context"
	"database/sql"
	"sync"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Querier is an interface for executing SQL queries, allowing for testing and instrumentation
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// LoadStrategy defines the relationship loading strategy
type LoadStrategy int

const (
	// EagerLoad loads relationships upfront in batched queries
	EagerLoad LoadStrategy = iota
	// LazyLoad loads relationships on-demand
	LazyLoad
)

// Loader handles efficient relationship loading with N+1 prevention
type Loader struct {
	db      Querier
	schemas map[string]*schema.ResourceSchema
	mu      sync.RWMutex
}

// NewLoader creates a new relationship loader
func NewLoader(db Querier, schemas map[string]*schema.ResourceSchema) *Loader {
	return &Loader{
		db:      db,
		schemas: schemas,
	}
}

// getSchema safely retrieves a schema from the map (thread-safe)
func (l *Loader) getSchema(name string) (*schema.ResourceSchema, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	schema, ok := l.schemas[name]
	return schema, ok
}

// LoadContext tracks loading state to prevent circular references
type LoadContext struct {
	visited  map[string]bool
	depth    int
	maxDepth int
	mu       sync.RWMutex
}

// NewLoadContext creates a new load context with the given max depth
func NewLoadContext(maxDepth int) *LoadContext {
	return &LoadContext{
		visited:  make(map[string]bool),
		depth:    0,
		maxDepth: maxDepth,
	}
}

// MarkVisited marks a resource as visited in the load context
func (lc *LoadContext) MarkVisited(resourceKey string) bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if lc.visited[resourceKey] {
		return false // Already visited
	}
	lc.visited[resourceKey] = true
	return true
}

// IncrementDepth increments the depth counter
func (lc *LoadContext) IncrementDepth() error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.depth++
	if lc.depth > lc.maxDepth {
		return ErrMaxDepthExceeded
	}
	return nil
}

// DecrementDepth decrements the depth counter
func (lc *LoadContext) DecrementDepth() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.depth--
}

// LazyRelation represents a lazy-loaded relationship
type LazyRelation struct {
	loader     *Loader
	ctx        context.Context
	parentID   interface{}
	relation   *schema.Relationship
	resource   *schema.ResourceSchema
	loaded     bool
	value      interface{}
	err        error
	mu         sync.Mutex
}

// NewLazyRelation creates a new lazy relation
func NewLazyRelation(
	loader *Loader,
	ctx context.Context,
	parentID interface{},
	relation *schema.Relationship,
	resource *schema.ResourceSchema,
) *LazyRelation {
	return &LazyRelation{
		loader:   loader,
		ctx:      ctx,
		parentID: parentID,
		relation: relation,
		resource: resource,
		loaded:   false,
	}
}

// Get loads the relationship value on first access (thread-safe)
func (lr *LazyRelation) Get() (interface{}, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.loaded {
		return lr.value, lr.err
	}

	// Load on demand
	lr.value, lr.err = lr.loader.LoadSingle(lr.ctx, lr.parentID, lr.relation, lr.resource)
	lr.loaded = true

	return lr.value, lr.err
}

// IsLoaded returns true if the relationship has been loaded
func (lr *LazyRelation) IsLoaded() bool {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	return lr.loaded
}

// BatchRequest represents a request to batch load relationships
type BatchRequest struct {
	ParentIDs  []interface{}
	Relation   *schema.Relationship
	Resource   *schema.ResourceSchema
	ResultChan chan BatchResult
}

// BatchResult represents the result of a batch load
type BatchResult struct {
	Data  map[string]interface{} // parent_id -> related data
	Error error
}
