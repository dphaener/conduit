// Package schema provides a registry for managing resource schemas
package schema

import (
	"fmt"
	"sync"
)

// Registry manages all resource schemas in the application
type Registry struct {
	schemas   map[string]*ResourceSchema
	validator *SchemaValidator
	mu        sync.RWMutex
}

// NewRegistry creates a new schema registry
func NewRegistry() *Registry {
	return &Registry{
		schemas:   make(map[string]*ResourceSchema),
		validator: NewSchemaValidator(),
	}
}

// Register registers a new resource schema
func (r *Registry) Register(schema *ResourceSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate
	if _, exists := r.schemas[schema.Name]; exists {
		return fmt.Errorf("resource %s is already registered", schema.Name)
	}

	// Store schema FIRST so it's available for validation
	r.schemas[schema.Name] = schema

	// Validate schema with access to all registered schemas
	// Note: We skip relationship validation here to allow forward references
	// Full relationship validation happens in ValidateAll()
	if err := r.validator.ValidateStructural(schema); err != nil {
		// Rollback on validation failure
		delete(r.schemas, schema.Name)
		return fmt.Errorf("schema validation failed for %s: %w", schema.Name, err)
	}

	return nil
}

// Get retrieves a resource schema by name
func (r *Registry) Get(name string) (*ResourceSchema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[name]
	return schema, exists
}

// All returns a copy of all registered schemas
func (r *Registry) All() map[string]*ResourceSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return copy to prevent external modification
	result := make(map[string]*ResourceSchema, len(r.schemas))
	for k, v := range r.schemas {
		result[k] = v
	}
	return result
}

// List returns a list of all resource names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.schemas))
	for name := range r.schemas {
		names = append(names, name)
	}
	return names
}

// ValidateAll performs comprehensive validation on all registered schemas
func (r *Registry) ValidateAll() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Build relationship graph
	graph := NewRelationshipGraph(r.schemas)

	// Check for cycles
	cycles := graph.DetectCycles()
	if len(cycles) > 0 {
		return fmt.Errorf("circular dependencies detected:\n%s", formatCycles(cycles))
	}

	// Validate relationships across all schemas
	relValidator := NewRelationshipValidator(r.schemas)
	if err := relValidator.Validate(); err != nil {
		return fmt.Errorf("relationship validation failed: %w", err)
	}

	return nil
}

// GetDependencyOrder returns resources in dependency order (safe for migrations)
func (r *Registry) GetDependencyOrder() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	graph := NewRelationshipGraph(r.schemas)
	return graph.TopologicalSort()
}

// AnalyzeDependencies returns a comprehensive dependency analysis report
func (r *Registry) AnalyzeDependencies() (*DependencyReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	analyzer := NewDependencyAnalyzer(r.schemas)
	return analyzer.Analyze()
}

// Clear removes all registered schemas (useful for testing)
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.schemas = make(map[string]*ResourceSchema)
}

// Count returns the number of registered schemas
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.schemas)
}

// Exists checks if a resource schema exists
func (r *Registry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.schemas[name]
	return exists
}

// GetRelationships returns all relationships for a resource
func (r *Registry) GetRelationships(resourceName string) (map[string]*Relationship, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[resourceName]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", resourceName)
	}

	return schema.Relationships, nil
}

// GetFields returns all fields for a resource
func (r *Registry) GetFields(resourceName string) (map[string]*Field, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[resourceName]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", resourceName)
	}

	return schema.Fields, nil
}

// GetHooks returns all hooks for a resource
func (r *Registry) GetHooks(resourceName string) (map[HookType][]*Hook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[resourceName]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", resourceName)
	}

	return schema.Hooks, nil
}

// Stats returns statistics about the registry
type RegistryStats struct {
	TotalResources       int
	TotalFields          int
	TotalRelationships   int
	TotalHooks           int
	TotalConstraints     int
	TotalScopes          int
	TotalComputedFields  int
	TotalValidators      int
	ResourcesWithHooks   int
	ResourcesWithScopes  int
	CircularDependencies bool
}

// GetStats returns statistics about the registry
func (r *Registry) GetStats() *RegistryStats {
	r.mu.RLock()
	schemasCopy := make(map[string]*ResourceSchema, len(r.schemas))
	for k, v := range r.schemas {
		schemasCopy[k] = v
	}
	r.mu.RUnlock()

	// Now work with the snapshot without holding the lock
	stats := &RegistryStats{}
	stats.TotalResources = len(schemasCopy)

	for _, schema := range schemasCopy {
		stats.TotalFields += len(schema.Fields)
		stats.TotalRelationships += len(schema.Relationships)
		stats.TotalConstraints += len(schema.ConstraintBlocks)
		stats.TotalScopes += len(schema.Scopes)
		stats.TotalComputedFields += len(schema.Computed)
		stats.TotalValidators += len(schema.Validators)

		for _, hooks := range schema.Hooks {
			stats.TotalHooks += len(hooks)
		}

		if len(schema.Hooks) > 0 {
			stats.ResourcesWithHooks++
		}
		if len(schema.Scopes) > 0 {
			stats.ResourcesWithScopes++
		}
	}

	// Check for circular dependencies using snapshot
	graph := NewRelationshipGraph(schemasCopy)
	cycles := graph.DetectCycles()
	stats.CircularDependencies = len(cycles) > 0

	return stats
}
