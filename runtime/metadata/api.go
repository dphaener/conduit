package metadata

// GetRegistry returns the global registry singleton.
// This is the primary entry point for runtime introspection.
//
// The registry is initialized at application startup via RegisterMetadata
// and provides fast indexed access to metadata with sub-millisecond query times.
//
// Example usage:
//
//	registry := metadata.GetRegistry()
//	resources := registry.Resources()
//	for _, res := range resources {
//		fmt.Printf("Resource: %s\n", res.Name)
//	}
func GetRegistry() *RegistryAPI {
	return &RegistryAPI{}
}

// RegistryAPI provides an ergonomic public API for runtime introspection.
// It wraps the internal registry implementation with type-safe, error-safe methods.
//
// All query methods leverage pre-computed indexes for fast O(1) or O(log n) lookups.
// The underlying metadata never changes at runtime, so results are cached for performance.
//
// Example usage:
//
//	registry := metadata.GetRegistry()
//
//	// Query all resources
//	resources := registry.Resources()
//
//	// Query single resource
//	post, err := registry.Resource("Post")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Query routes with filters
//	routes := registry.Routes(metadata.RouteFilter{
//		Method: "GET",
//	})
//
//	// Query patterns
//	patterns := registry.Patterns("hook")
//
//	// Query dependencies
//	deps, err := registry.Dependencies("Post", metadata.DependencyOptions{
//		Depth: 2,
//		Reverse: false,
//	})
type RegistryAPI struct{}

// RouteFilter provides optional filters for route queries.
// All fields are optional - empty string means no filtering on that field.
//
// Example usage:
//
//	// Filter by HTTP method
//	routes := registry.Routes(metadata.RouteFilter{Method: "GET"})
//
//	// Filter by resource
//	routes := registry.Routes(metadata.RouteFilter{Resource: "Post"})
//
//	// Filter by both method and resource
//	routes := registry.Routes(metadata.RouteFilter{
//		Method: "GET",
//		Resource: "Post",
//	})
//
//	// Get all routes (no filtering)
//	routes := registry.Routes(metadata.RouteFilter{})
type RouteFilter struct {
	Method   string // Optional: filter by HTTP method (GET, POST, PUT, DELETE, etc.)
	Path     string // Optional: filter by exact path pattern
	Resource string // Optional: filter by resource name
}

// Resources returns all registered resources.
//
// Returns a copy of the resource metadata to prevent external mutation.
// This is a fast operation (<1ms) that reads from pre-computed indexes.
//
// Example usage:
//
//	registry := metadata.GetRegistry()
//	resources := registry.Resources()
//	for _, res := range resources {
//		fmt.Printf("Resource: %s (%d fields)\n", res.Name, len(res.Fields))
//	}
func (r *RegistryAPI) Resources() []ResourceMetadata {
	return QueryResources()
}

// Resource returns metadata for a single resource by name.
//
// This is an O(1) lookup using pre-computed indexes. Returns an error
// if the resource is not found or if the registry is not initialized.
//
// Example usage:
//
//	registry := metadata.GetRegistry()
//	post, err := registry.Resource("Post")
//	if err != nil {
//		log.Fatalf("Resource not found: %v", err)
//	}
//	fmt.Printf("Post has %d fields\n", len(post.Fields))
func (r *RegistryAPI) Resource(name string) (*ResourceMetadata, error) {
	return QueryResource(name)
}

// Routes returns routes filtered by the provided criteria.
//
// If filter is empty (all fields are empty strings), returns all routes.
// Multiple filter criteria are combined with AND logic.
//
// This leverages pre-computed indexes for fast lookups:
//   - Method filtering: O(1) lookup
//   - Path filtering: O(1) lookup
//   - Resource filtering: O(n) scan
//
// Example usage:
//
//	registry := metadata.GetRegistry()
//
//	// Get all GET routes
//	routes := registry.Routes(metadata.RouteFilter{Method: "GET"})
//
//	// Get all routes for Post resource
//	routes := registry.Routes(metadata.RouteFilter{Resource: "Post"})
//
//	// Get specific route
//	routes := registry.Routes(metadata.RouteFilter{
//		Method: "GET",
//		Path:   "/posts",
//	})
func (r *RegistryAPI) Routes(filter RouteFilter) []RouteMetadata {
	// If no filters, return all routes
	if filter.Method == "" && filter.Path == "" && filter.Resource == "" {
		return QueryRoutes()
	}

	// Start with all routes from the most specific filter
	var routes []RouteMetadata
	if filter.Method != "" {
		routes = QueryRoutesByMethod(filter.Method)
	} else if filter.Path != "" {
		routes = QueryRoutesByPath(filter.Path)
	} else {
		routes = QueryRoutes()
	}

	// Apply additional filters
	var result []RouteMetadata
	for _, route := range routes {
		if filter.Method != "" && route.Method != filter.Method {
			continue
		}
		if filter.Path != "" && route.Path != filter.Path {
			continue
		}
		if filter.Resource != "" && route.Resource != filter.Resource {
			continue
		}
		result = append(result, route)
	}

	return result
}

// Patterns returns patterns filtered by category.
//
// If category is an empty string, returns all patterns.
// Category matching is case-sensitive and exact.
//
// Common categories: "hook", "validation", "middleware", "query", "relationship"
//
// Example usage:
//
//	registry := metadata.GetRegistry()
//
//	// Get all hook patterns
//	patterns := registry.Patterns("hook")
//
//	// Get all patterns
//	patterns := registry.Patterns("")
//
//	// Print pattern usage
//	for _, p := range patterns {
//		fmt.Printf("Pattern: %s (used %d times)\n", p.Name, p.Frequency)
//	}
func (r *RegistryAPI) Patterns(category string) []PatternMetadata {
	allPatterns := QueryPatterns()
	if category == "" {
		return allPatterns
	}

	// Filter by category
	var result []PatternMetadata
	for _, pattern := range allPatterns {
		if pattern.Category == category {
			result = append(result, pattern)
		}
	}
	return result
}

// Dependencies returns a dependency graph starting from the specified resource.
//
// The graph includes all nodes and edges reachable from the starting resource
// according to the provided options:
//
//   - Depth: Maximum traversal depth (0 = unlimited)
//   - Reverse: If true, finds what depends on this resource (reverse dependencies)
//   - Types: Filter edges by relationship type (e.g., ["belongs_to", "has_many"])
//
// Results are cached for performance since metadata never changes at runtime.
//
// Example usage:
//
//	registry := metadata.GetRegistry()
//
//	// Get all forward dependencies (what Post depends on)
//	deps, err := registry.Dependencies("Post", metadata.DependencyOptions{
//		Depth:   0,
//		Reverse: false,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Post depends on %d resources\n", len(deps.Nodes)-1)
//
//	// Get all reverse dependencies (what depends on User)
//	deps, err := registry.Dependencies("User", metadata.DependencyOptions{
//		Depth:   0,
//		Reverse: true,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("%d resources depend on User\n", len(deps.Nodes)-1)
//
//	// Get direct dependencies only (depth 1)
//	deps, err := registry.Dependencies("Post", metadata.DependencyOptions{
//		Depth:   1,
//		Reverse: false,
//	})
func (r *RegistryAPI) Dependencies(resource string, opts DependencyOptions) (*DependencyGraph, error) {
	return QueryDependencies(resource, opts)
}

// GetSchema returns the complete metadata schema.
//
// This returns the entire Metadata structure containing all resources,
// routes, patterns, and the full dependency graph. Returns nil if the
// registry has not been initialized.
//
// Use this when you need the complete schema for serialization or
// comprehensive analysis. For targeted queries, prefer the specific
// query methods (Resources, Resource, Routes, etc.) which are faster.
//
// Example usage:
//
//	registry := metadata.GetRegistry()
//	schema := registry.GetSchema()
//	if schema == nil {
//		log.Fatal("Registry not initialized")
//	}
//	fmt.Printf("Schema version: %s\n", schema.Version)
//	fmt.Printf("Total resources: %d\n", len(schema.Resources))
//	fmt.Printf("Total routes: %d\n", len(schema.Routes))
//	fmt.Printf("Total patterns: %d\n", len(schema.Patterns))
func (r *RegistryAPI) GetSchema() *Metadata {
	return GetMetadata()
}
