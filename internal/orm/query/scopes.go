// Package query provides scope implementation for reusable query fragments
package query

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// ScopeCompiler compiles scope definitions into executable scopes
type ScopeCompiler struct {
	resource *schema.ResourceSchema
}

// NewScopeCompiler creates a new scope compiler
func NewScopeCompiler(resource *schema.ResourceSchema) *ScopeCompiler {
	return &ScopeCompiler{
		resource: resource,
	}
}

// CompileScope compiles a scope definition
func (c *ScopeCompiler) CompileScope(scope *schema.Scope) (*CompiledScope, error) {
	compiled := &CompiledScope{
		Name:       scope.Name,
		Arguments:  scope.Arguments,
		Conditions: make([]*Condition, 0),
		OrderBy:    make([]string, 0),
	}

	// Compile where conditions from the scope's Where map
	// Note: In a real implementation, this would parse the Where map
	// and convert it to Condition objects. For now, this is simplified.

	// Set order by if specified
	if scope.OrderBy != "" {
		compiled.OrderBy = []string{scope.OrderBy}
	}

	// Set limit if specified
	if scope.Limit != nil {
		compiled.Limit = scope.Limit
	}

	// Set offset if specified
	if scope.Offset != nil {
		compiled.Offset = scope.Offset
	}

	return compiled, nil
}

// CompiledScope represents a compiled scope ready for execution
type CompiledScope struct {
	Name       string
	Arguments  []*schema.ScopeArgument
	Conditions []*Condition
	OrderBy    []string
	Limit      *int
	Offset     *int
}

// Bind binds arguments to the scope and returns a bound scope
func (cs *CompiledScope) Bind(args []interface{}) (*BoundScope, error) {
	if len(args) != len(cs.Arguments) {
		return nil, fmt.Errorf("scope %s expects %d arguments, got %d",
			cs.Name, len(cs.Arguments), len(args))
	}

	// Validate argument types
	for i, arg := range args {
		expectedType := cs.Arguments[i].Type
		if err := validateArgumentType(arg, expectedType); err != nil {
			return nil, fmt.Errorf("argument %s: %w", cs.Arguments[i].Name, err)
		}
	}

	bound := &BoundScope{
		Name:       cs.Name,
		Conditions: cs.Conditions,
		OrderBy:    cs.OrderBy,
		Limit:      cs.Limit,
		Offset:     cs.Offset,
		Arguments:  make(map[string]interface{}),
	}

	// Bind arguments
	for i, arg := range cs.Arguments {
		bound.Arguments[arg.Name] = args[i]
	}

	// Substitute argument values in conditions
	bound.Conditions = substituteArguments(cs.Conditions, bound.Arguments)

	return bound, nil
}

// BoundScope represents a scope with bound arguments
type BoundScope struct {
	Name       string
	Conditions []*Condition
	OrderBy    []string
	Limit      *int
	Offset     *int
	Arguments  map[string]interface{}
}

// validateArgumentType validates that an argument matches the expected type
func validateArgumentType(value interface{}, typeSpec *schema.TypeSpec) error {
	// This is a simplified validation
	// In practice, this would do comprehensive type checking
	if value == nil {
		if !typeSpec.Nullable {
			return fmt.Errorf("expected non-null value")
		}
		return nil
	}

	// Basic type checking
	switch typeSpec.BaseType {
	case schema.TypeString, schema.TypeText:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case schema.TypeInt:
		if _, ok := value.(int); !ok {
			if _, ok := value.(int64); !ok {
				return fmt.Errorf("expected int, got %T", value)
			}
		}
	case schema.TypeFloat:
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("expected float64, got %T", value)
		}
	case schema.TypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	}

	return nil
}

// substituteArguments substitutes argument placeholders with actual values
func substituteArguments(conditions []*Condition, args map[string]interface{}) []*Condition {
	// In a real implementation, this would scan conditions for argument placeholders
	// and replace them with actual values. For now, return conditions as-is.
	return conditions
}

// ScopeRegistry manages all compiled scopes for a resource
type ScopeRegistry struct {
	scopes map[string]*CompiledScope
}

// NewScopeRegistry creates a new scope registry
func NewScopeRegistry() *ScopeRegistry {
	return &ScopeRegistry{
		scopes: make(map[string]*CompiledScope),
	}
}

// Register registers a compiled scope
func (sr *ScopeRegistry) Register(scope *CompiledScope) {
	sr.scopes[scope.Name] = scope
}

// Get retrieves a compiled scope by name
func (sr *ScopeRegistry) Get(name string) (*CompiledScope, error) {
	scope, ok := sr.scopes[name]
	if !ok {
		return nil, fmt.Errorf("unknown scope: %s", name)
	}
	return scope, nil
}

// Has checks if a scope exists
func (sr *ScopeRegistry) Has(name string) bool {
	_, ok := sr.scopes[name]
	return ok
}

// List returns all registered scope names
func (sr *ScopeRegistry) List() []string {
	names := make([]string, 0, len(sr.scopes))
	for name := range sr.scopes {
		names = append(names, name)
	}
	return names
}

// ScopeChain allows chaining multiple scopes together
type ScopeChain struct {
	scopes []*BoundScope
}

// NewScopeChain creates a new scope chain
func NewScopeChain() *ScopeChain {
	return &ScopeChain{
		scopes: make([]*BoundScope, 0),
	}
}

// Add adds a scope to the chain
func (sc *ScopeChain) Add(scope *BoundScope) {
	sc.scopes = append(sc.scopes, scope)
}

// Apply applies all scopes in the chain to a query builder
func (sc *ScopeChain) Apply(qb *QueryBuilder) error {
	for _, scope := range sc.scopes {
		// Add conditions
		qb.conditions = append(qb.conditions, scope.Conditions...)

		// Add order by
		qb.orderBy = append(qb.orderBy, scope.OrderBy...)

		// Apply limit (use the most restrictive)
		if scope.Limit != nil {
			if qb.limit == nil || *scope.Limit < *qb.limit {
				qb.limit = scope.Limit
			}
		}

		// Apply offset (accumulate)
		if scope.Offset != nil {
			if qb.offset == nil {
				qb.offset = scope.Offset
			} else {
				combined := *qb.offset + *scope.Offset
				qb.offset = &combined
			}
		}
	}

	return nil
}

// Merge merges multiple scopes into a single scope
func (sc *ScopeChain) Merge() *BoundScope {
	if len(sc.scopes) == 0 {
		return &BoundScope{
			Conditions: make([]*Condition, 0),
			OrderBy:    make([]string, 0),
			Arguments:  make(map[string]interface{}),
		}
	}

	merged := &BoundScope{
		Name:       "merged",
		Conditions: make([]*Condition, 0),
		OrderBy:    make([]string, 0),
		Arguments:  make(map[string]interface{}),
	}

	// Merge all conditions
	for _, scope := range sc.scopes {
		merged.Conditions = append(merged.Conditions, scope.Conditions...)
		merged.OrderBy = append(merged.OrderBy, scope.OrderBy...)

		// Merge arguments
		for k, v := range scope.Arguments {
			merged.Arguments[k] = v
		}

		// Use most restrictive limit
		if scope.Limit != nil {
			if merged.Limit == nil || *scope.Limit < *merged.Limit {
				merged.Limit = scope.Limit
			}
		}

		// Accumulate offsets
		if scope.Offset != nil {
			if merged.Offset == nil {
				merged.Offset = scope.Offset
			} else {
				combined := *merged.Offset + *scope.Offset
				merged.Offset = &combined
			}
		}
	}

	return merged
}

// DefaultScopes provides commonly used scopes
type DefaultScopes struct {
	resource *schema.ResourceSchema
}

// NewDefaultScopes creates default scopes for a resource
func NewDefaultScopes(resource *schema.ResourceSchema) *DefaultScopes {
	return &DefaultScopes{
		resource: resource,
	}
}

// Recent returns a scope for recent records (ordered by created_at DESC)
func (ds *DefaultScopes) Recent(limit int) *BoundScope {
	limitVal := limit
	return &BoundScope{
		Name:       "recent",
		Conditions: make([]*Condition, 0),
		OrderBy:    []string{"created_at DESC"},
		Limit:      &limitVal,
		Arguments:  make(map[string]interface{}),
	}
}

// Active returns a scope for active records
func (ds *DefaultScopes) Active() *BoundScope {
	return &BoundScope{
		Name: "active",
		Conditions: []*Condition{
			{
				Field:    "status",
				Operator: OpEqual,
				Value:    "active",
				Or:       false,
			},
		},
		OrderBy:   make([]string, 0),
		Arguments: make(map[string]interface{}),
	}
}

// Archived returns a scope for archived records
func (ds *DefaultScopes) Archived() *BoundScope {
	return &BoundScope{
		Name: "archived",
		Conditions: []*Condition{
			{
				Field:    "archived_at",
				Operator: OpIsNotNull,
				Value:    nil,
				Or:       false,
			},
		},
		OrderBy:   make([]string, 0),
		Arguments: make(map[string]interface{}),
	}
}

// NotArchived returns a scope for non-archived records
func (ds *DefaultScopes) NotArchived() *BoundScope {
	return &BoundScope{
		Name: "not_archived",
		Conditions: []*Condition{
			{
				Field:    "archived_at",
				Operator: OpIsNull,
				Value:    nil,
				Or:       false,
			},
		},
		OrderBy:   make([]string, 0),
		Arguments: make(map[string]interface{}),
	}
}

// CreatedAfter returns a scope for records created after a specific time
func (ds *DefaultScopes) CreatedAfter(timestamp interface{}) *BoundScope {
	return &BoundScope{
		Name: "created_after",
		Conditions: []*Condition{
			{
				Field:    "created_at",
				Operator: OpGreaterThan,
				Value:    timestamp,
				Or:       false,
			},
		},
		OrderBy:   make([]string, 0),
		Arguments: map[string]interface{}{"timestamp": timestamp},
	}
}

// CreatedBefore returns a scope for records created before a specific time
func (ds *DefaultScopes) CreatedBefore(timestamp interface{}) *BoundScope {
	return &BoundScope{
		Name: "created_before",
		Conditions: []*Condition{
			{
				Field:    "created_at",
				Operator: OpLessThan,
				Value:    timestamp,
				Or:       false,
			},
		},
		OrderBy:   make([]string, 0),
		Arguments: map[string]interface{}{"timestamp": timestamp},
	}
}

// UpdatedAfter returns a scope for records updated after a specific time
func (ds *DefaultScopes) UpdatedAfter(timestamp interface{}) *BoundScope {
	return &BoundScope{
		Name: "updated_after",
		Conditions: []*Condition{
			{
				Field:    "updated_at",
				Operator: OpGreaterThan,
				Value:    timestamp,
				Or:       false,
			},
		},
		OrderBy:   make([]string, 0),
		Arguments: map[string]interface{}{"timestamp": timestamp},
	}
}

// Paginate returns a scope for pagination
func (ds *DefaultScopes) Paginate(page, perPage int) *BoundScope {
	offset := (page - 1) * perPage
	return &BoundScope{
		Name:       "paginate",
		Conditions: make([]*Condition, 0),
		OrderBy:    make([]string, 0),
		Limit:      &perPage,
		Offset:     &offset,
		Arguments: map[string]interface{}{
			"page":     page,
			"per_page": perPage,
		},
	}
}
