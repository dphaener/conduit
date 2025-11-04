package metadata

import (
	"container/list"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	defaultMaxCacheEntries = 1000
	maxCacheMemoryBytes    = 10 * 1024 * 1024 // 10MB
)

// lruCache implements an LRU cache with memory limits and hit/miss tracking
type lruCache struct {
	maxEntries   int
	maxMemory    int64
	entries      map[string]*list.Element
	evictionList *list.List
	currentSize  int64
	hits         int64
	misses       int64
}

// cacheEntry represents a single cache entry
type cacheEntry struct {
	key   string
	value interface{}
	size  int64
}

// newLRUCache creates a new LRU cache with default settings
func newLRUCache() *lruCache {
	return &lruCache{
		maxEntries:   defaultMaxCacheEntries,
		maxMemory:    maxCacheMemoryBytes,
		entries:      make(map[string]*list.Element),
		evictionList: list.New(),
		currentSize:  0,
		hits:         0,
		misses:       0,
	}
}

// get retrieves a value from the cache
func (c *lruCache) get(key string) (interface{}, bool) {
	if elem, ok := c.entries[key]; ok {
		c.evictionList.MoveToFront(elem)
		atomic.AddInt64(&c.hits, 1)
		return elem.Value.(*cacheEntry).value, true
	}
	atomic.AddInt64(&c.misses, 1)
	return nil, false
}

// set adds or updates a value in the cache
func (c *lruCache) set(key string, value interface{}) {
	// Estimate size (rough approximation)
	size := estimateSize(value)

	// If entry exists, update it
	if elem, ok := c.entries[key]; ok {
		entry := elem.Value.(*cacheEntry)
		c.currentSize -= entry.size
		entry.value = value
		entry.size = size
		c.currentSize += size
		c.evictionList.MoveToFront(elem)
		c.evictIfNeeded()
		return
	}

	// Add new entry
	entry := &cacheEntry{
		key:   key,
		value: value,
		size:  size,
	}
	elem := c.evictionList.PushFront(entry)
	c.entries[key] = elem
	c.currentSize += size

	// Evict if necessary
	c.evictIfNeeded()
}

// evictIfNeeded removes entries if limits are exceeded
func (c *lruCache) evictIfNeeded() {
	for c.evictionList.Len() > c.maxEntries || c.currentSize > c.maxMemory {
		if c.evictionList.Len() == 0 {
			break
		}
		elem := c.evictionList.Back()
		if elem != nil {
			c.removeElement(elem)
		}
	}
}

// removeElement removes an element from the cache
func (c *lruCache) removeElement(elem *list.Element) {
	c.evictionList.Remove(elem)
	entry := elem.Value.(*cacheEntry)
	delete(c.entries, entry.key)
	c.currentSize -= entry.size
}

// clear removes all entries from the cache
func (c *lruCache) clear() {
	c.entries = make(map[string]*list.Element)
	c.evictionList = list.New()
	c.currentSize = 0
	c.hits = 0
	c.misses = 0
}

// stats returns cache statistics
func (c *lruCache) stats() (hits, misses int64, hitRate float64) {
	hits = atomic.LoadInt64(&c.hits)
	misses = atomic.LoadInt64(&c.misses)
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	return hits, misses, hitRate
}

// estimateSize estimates the memory size of a cached value
func estimateSize(value interface{}) int64 {
	// Rough estimation based on type
	switch v := value.(type) {
	case *DependencyGraph:
		size := int64(len(v.Nodes) * 200)        // ~200 bytes per node
		size += int64(len(v.Edges) * 100)        // ~100 bytes per edge
		size += int64(len(v.outgoingEdges) * 50) // overhead for maps
		size += int64(len(v.incomingEdges) * 50)
		return size
	case []ResourceMetadata:
		return int64(len(v) * 500) // ~500 bytes per resource
	case []FieldReference:
		return int64(len(v) * 150) // ~150 bytes per field ref
	default:
		return 1024 // Default 1KB estimate
	}
}

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

	// LRU cache for query results (metadata never changes at runtime)
	cache      *lruCache
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
	cache:             newLRUCache(),
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
			return nil, fmt.Errorf(`registry not initialized - no resources found yet

To create your first resource:

  mkdir -p app/resources
  cat > app/resources/todo.cdt << 'EOF'
  resource Todo {
    id: uuid! @primary @auto
    title: string!
    created_at: timestamp! @auto
  }
  EOF

Then run: conduit build`)
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
	globalRegistry.cache.clear()
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
			return nil, fmt.Errorf(`registry not initialized - no resources found yet

To create your first resource:

  mkdir -p app/resources
  cat > app/resources/todo.cdt << 'EOF'
  resource Todo {
    id: uuid! @primary @auto
    title: string!
    created_at: timestamp! @auto
  }
  EOF

Then run: conduit build`)
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
	if val, ok := r.cache.get(key); ok {
		return val
	}
	return nil
}

// setCached stores a value in the cache
func (r *Registry) setCached(key string, value interface{}) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	r.cache.set(key, value)
}

// GetCacheStats returns cache hit/miss statistics
func GetCacheStats() (hits, misses int64, hitRate float64) {
	globalRegistry.cacheMutex.RLock()
	defer globalRegistry.cacheMutex.RUnlock()
	return globalRegistry.cache.stats()
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
