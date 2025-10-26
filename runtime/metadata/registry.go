package metadata

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry holds the runtime metadata for introspection queries.
// It is initialized at application startup via the generated init() function.
// The registry provides fast indexed access to metadata with sub-millisecond query times.
type Registry struct {
	mu       sync.RWMutex
	metadata *Metadata

	// Pre-computed indexes for fast queries (built at initialization)
	resourcesByName   map[string]*ResourceMetadata
	routesByPath      map[string][]*RouteMetadata
	routesByMethod    map[string][]*RouteMetadata
	patternsByName    map[string]*PatternMetadata
	relationshipIndex map[string][]*RelationshipRef // resource name -> relationships

	// Query result cache (metadata never changes at runtime)
	cache      map[string]interface{}
	cacheMutex sync.RWMutex

	// Lazy initialization state
	initialized atomic.Bool
	initMutex   sync.Mutex
}

// RelationshipRef references a relationship and its source resource
type RelationshipRef struct {
	SourceResource string
	Relationship   *RelationshipMetadata
}

// Global registry instance
var globalRegistry = &Registry{
	resourcesByName:   make(map[string]*ResourceMetadata),
	routesByPath:      make(map[string][]*RouteMetadata),
	routesByMethod:    make(map[string][]*RouteMetadata),
	patternsByName:    make(map[string]*PatternMetadata),
	relationshipIndex: make(map[string][]*RelationshipRef),
	cache:             make(map[string]interface{}),
}

// RegisterMetadata registers metadata in the global registry.
// This is called from the generated init() function at application startup.
// Builds all indexes for fast query performance (<1ms for typical queries).
func RegisterMetadata(data []byte) error {
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.metadata = &meta

	// Build indexes for fast queries
	globalRegistry.buildIndexes()
	globalRegistry.initialized.Store(true)

	return nil
}

// buildIndexes builds all pre-computed indexes for fast queries.
// This is called once during RegisterMetadata.
// Target time: <10ms for typical applications (50 resources).
func (r *Registry) buildIndexes() {
	if r.metadata == nil {
		return
	}

	// Index resources by name
	for i := range r.metadata.Resources {
		res := &r.metadata.Resources[i]
		r.resourcesByName[res.Name] = res

		// Index relationships for this resource
		for j := range res.Relationships {
			rel := &res.Relationships[j]
			r.relationshipIndex[rel.TargetResource] = append(
				r.relationshipIndex[rel.TargetResource],
				&RelationshipRef{
					SourceResource: res.Name,
					Relationship:   rel,
				},
			)
		}
	}

	// Index routes by path and method
	for i := range r.metadata.Routes {
		route := &r.metadata.Routes[i]
		r.routesByPath[route.Path] = append(r.routesByPath[route.Path], route)
		r.routesByMethod[route.Method] = append(r.routesByMethod[route.Method], route)
	}

	// Index patterns by name
	for i := range r.metadata.Patterns {
		pattern := &r.metadata.Patterns[i]
		r.patternsByName[pattern.Name] = pattern
	}
}

// GetMetadata returns the registered metadata.
// Returns nil if no metadata has been registered.
func GetMetadata() *Metadata {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	return globalRegistry.metadata
}

// QueryResources returns all registered resources.
// Returns a copy to prevent external mutation.
func QueryResources() []ResourceMetadata {
	meta := GetMetadata()
	if meta == nil {
		return nil
	}
	// Return a copy to prevent external mutation
	resources := make([]ResourceMetadata, len(meta.Resources))
	copy(resources, meta.Resources)
	return resources
}

// QueryResource finds a resource by name using the pre-computed index.
// This is optimized for O(1) lookup time (<1ms).
// Uses double-check locking pattern: fast path checks initialized atomically,
// slow path acquires lock only if initialization is needed.
func QueryResource(name string) (*ResourceMetadata, error) {
	// Fast path: check if already initialized (no locks)
	if !globalRegistry.initialized.Load() {
		// Slow path: initialize if needed
		globalRegistry.initMutex.Lock()
		if !globalRegistry.initialized.Load() {
			// TODO: Load embedded metadata here (waiting for CON-51)
			// For now, return error if not manually registered
			globalRegistry.initMutex.Unlock()
			return nil, fmt.Errorf("registry not initialized")
		}
		globalRegistry.initMutex.Unlock()
	}

	// Now safe to read
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	if res, ok := globalRegistry.resourcesByName[name]; ok {
		// Return a copy to prevent external mutation
		resCopy := *res
		return &resCopy, nil
	}

	return nil, fmt.Errorf("resource not found: %s", name)
}

// QueryPatterns returns all registered patterns.
// Returns a copy to prevent external mutation.
func QueryPatterns() []PatternMetadata {
	meta := GetMetadata()
	if meta == nil {
		return nil
	}
	// Return a copy to prevent external mutation
	patterns := make([]PatternMetadata, len(meta.Patterns))
	copy(patterns, meta.Patterns)
	return patterns
}

// QueryRoutes returns all registered routes.
// Returns a copy to prevent external mutation.
func QueryRoutes() []RouteMetadata {
	meta := GetMetadata()
	if meta == nil {
		return nil
	}
	// Return a copy to prevent external mutation
	routes := make([]RouteMetadata, len(meta.Routes))
	copy(routes, meta.Routes)
	return routes
}

// Reset clears the registry (used for testing).
func Reset() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.metadata = nil
	globalRegistry.resourcesByName = make(map[string]*ResourceMetadata)
	globalRegistry.routesByPath = make(map[string][]*RouteMetadata)
	globalRegistry.routesByMethod = make(map[string][]*RouteMetadata)
	globalRegistry.patternsByName = make(map[string]*PatternMetadata)
	globalRegistry.relationshipIndex = make(map[string][]*RelationshipRef)
	globalRegistry.cache = make(map[string]interface{})
	globalRegistry.initialized.Store(false)
}

// QueryRoutesByMethod returns all routes for a specific HTTP method.
// Uses pre-computed index for O(1) lookup.
// Uses double-check locking pattern: fast path checks initialized atomically,
// slow path acquires lock only if initialization is needed.
func QueryRoutesByMethod(method string) []RouteMetadata {
	// Fast path: check if already initialized (no locks)
	if !globalRegistry.initialized.Load() {
		// Slow path: initialize if needed
		globalRegistry.initMutex.Lock()
		if !globalRegistry.initialized.Load() {
			// TODO: Load embedded metadata here (waiting for CON-51)
			// For now, return nil if not manually registered
			globalRegistry.initMutex.Unlock()
			return nil
		}
		globalRegistry.initMutex.Unlock()
	}

	// Now safe to read
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	routes, ok := globalRegistry.routesByMethod[strings.ToUpper(method)]
	if !ok {
		return []RouteMetadata{}
	}

	// Return copies to prevent external mutation
	result := make([]RouteMetadata, len(routes))
	for i, route := range routes {
		result[i] = *route
	}
	return result
}

// QueryRoutesByPath returns all routes for a specific path.
// Uses pre-computed index for O(1) lookup.
// Uses double-check locking pattern: fast path checks initialized atomically,
// slow path acquires lock only if initialization is needed.
func QueryRoutesByPath(path string) []RouteMetadata {
	// Fast path: check if already initialized (no locks)
	if !globalRegistry.initialized.Load() {
		// Slow path: initialize if needed
		globalRegistry.initMutex.Lock()
		if !globalRegistry.initialized.Load() {
			// TODO: Load embedded metadata here (waiting for CON-51)
			// For now, return nil if not manually registered
			globalRegistry.initMutex.Unlock()
			return nil
		}
		globalRegistry.initMutex.Unlock()
	}

	// Now safe to read
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	routes, ok := globalRegistry.routesByPath[path]
	if !ok {
		return []RouteMetadata{}
	}

	// Return copies to prevent external mutation
	result := make([]RouteMetadata, len(routes))
	for i, route := range routes {
		result[i] = *route
	}
	return result
}

// QueryPattern finds a pattern by name.
// Uses pre-computed index for O(1) lookup.
// Uses double-check locking pattern: fast path checks initialized atomically,
// slow path acquires lock only if initialization is needed.
func QueryPattern(name string) (*PatternMetadata, error) {
	// Fast path: check if already initialized (no locks)
	if !globalRegistry.initialized.Load() {
		// Slow path: initialize if needed
		globalRegistry.initMutex.Lock()
		if !globalRegistry.initialized.Load() {
			// TODO: Load embedded metadata here (waiting for CON-51)
			// For now, return error if not manually registered
			globalRegistry.initMutex.Unlock()
			return nil, fmt.Errorf("registry not initialized")
		}
		globalRegistry.initMutex.Unlock()
	}

	// Now safe to read
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	if pattern, ok := globalRegistry.patternsByName[name]; ok {
		// Return a copy to prevent external mutation
		patternCopy := *pattern
		return &patternCopy, nil
	}

	return nil, fmt.Errorf("pattern not found: %s", name)
}

// QueryRelationshipsTo returns all relationships pointing to a resource.
// This finds reverse dependencies (what depends on this resource).
// Uses double-check locking pattern: fast path checks initialized atomically,
// slow path acquires lock only if initialization is needed.
func QueryRelationshipsTo(resourceName string) []RelationshipRef {
	// Fast path: check if already initialized (no locks)
	if !globalRegistry.initialized.Load() {
		// Slow path: initialize if needed
		globalRegistry.initMutex.Lock()
		if !globalRegistry.initialized.Load() {
			// TODO: Load embedded metadata here (waiting for CON-51)
			// For now, return nil if not manually registered
			globalRegistry.initMutex.Unlock()
			return nil
		}
		globalRegistry.initMutex.Unlock()
	}

	// Now safe to read
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	refs, ok := globalRegistry.relationshipIndex[resourceName]
	if !ok {
		return []RelationshipRef{}
	}

	// Return copies to prevent external mutation
	result := make([]RelationshipRef, len(refs))
	for i, ref := range refs {
		relCopy := *ref.Relationship
		result[i] = RelationshipRef{
			SourceResource: ref.SourceResource,
			Relationship:   &relCopy,
		}
	}
	return result
}

// QueryRelationshipsFrom returns all relationships from a resource.
// This finds forward dependencies (what this resource depends on).
func QueryRelationshipsFrom(resourceName string) ([]RelationshipMetadata, error) {
	res, err := QueryResource(resourceName)
	if err != nil {
		return nil, err
	}

	// Return a copy to prevent external mutation
	relationships := make([]RelationshipMetadata, len(res.Relationships))
	copy(relationships, res.Relationships)
	return relationships, nil
}

// QueryResourcesByPattern searches resources matching a pattern.
// Pattern supports wildcards: "*" matches any characters.
// Uses double-check locking pattern: fast path checks initialized atomically,
// slow path acquires lock only if initialization is needed.
func QueryResourcesByPattern(pattern string) []ResourceMetadata {
	// Fast path: check if already initialized (no locks)
	if !globalRegistry.initialized.Load() {
		// Slow path: initialize if needed
		globalRegistry.initMutex.Lock()
		if !globalRegistry.initialized.Load() {
			// TODO: Load embedded metadata here (waiting for CON-51)
			// For now, return nil if not manually registered
			globalRegistry.initMutex.Unlock()
			return nil
		}
		globalRegistry.initMutex.Unlock()
	}

	// Now safe to read
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	// Check cache first
	cacheKey := "pattern:" + pattern
	if cached := globalRegistry.getCached(cacheKey); cached != nil {
		return cached.([]ResourceMetadata)
	}

	var result []ResourceMetadata
	for _, res := range globalRegistry.metadata.Resources {
		if matchPattern(res.Name, pattern) {
			result = append(result, res)
		}
	}

	// Cache result
	globalRegistry.setCached(cacheKey, result)
	return result
}

// QueryFieldsByType returns all fields of a specific type across all resources.
// Uses double-check locking pattern: fast path checks initialized atomically,
// slow path acquires lock only if initialization is needed.
func QueryFieldsByType(typeName string) []FieldReference {
	// Fast path: check if already initialized (no locks)
	if !globalRegistry.initialized.Load() {
		// Slow path: initialize if needed
		globalRegistry.initMutex.Lock()
		if !globalRegistry.initialized.Load() {
			// TODO: Load embedded metadata here (waiting for CON-51)
			// For now, return nil if not manually registered
			globalRegistry.initMutex.Unlock()
			return nil
		}
		globalRegistry.initMutex.Unlock()
	}

	// Now safe to read
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	// Check cache first
	cacheKey := "fields_by_type:" + typeName
	if cached := globalRegistry.getCached(cacheKey); cached != nil {
		return cached.([]FieldReference)
	}

	var result []FieldReference
	for _, res := range globalRegistry.metadata.Resources {
		for _, field := range res.Fields {
			if strings.HasPrefix(field.Type, typeName) {
				result = append(result, FieldReference{
					ResourceName: res.Name,
					Field:        field,
				})
			}
		}
	}

	// Cache result
	globalRegistry.setCached(cacheKey, result)
	return result
}

// FieldReference references a field and its containing resource
type FieldReference struct {
	ResourceName string
	Field        FieldMetadata
}

// getCached retrieves a value from the cache
func (r *Registry) getCached(key string) interface{} {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()
	return r.cache[key]
}

// setCached stores a value in the cache
func (r *Registry) setCached(key string, value interface{}) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	r.cache[key] = value
}

// matchPattern matches a string against a pattern with wildcards
func matchPattern(s, pattern string) bool {
	// Exact match
	if pattern == s {
		return true
	}

	// Wildcard match
	if pattern == "*" {
		return true
	}

	// Prefix match (pattern ends with *)
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(s, prefix)
	}

	// Suffix match (pattern starts with *)
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(s, suffix)
	}

	// Contains match (pattern has * in the middle)
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(s, parts[0]) && strings.HasSuffix(s, parts[1])
		}
	}

	return false
}
