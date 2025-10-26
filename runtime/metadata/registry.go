package metadata

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Registry holds the runtime metadata for introspection queries.
// It is initialized at application startup via the generated init() function.
type Registry struct {
	mu       sync.RWMutex
	metadata *Metadata
}

// Global registry instance
var globalRegistry = &Registry{}

// RegisterMetadata registers metadata in the global registry.
// This is called from the generated init() function at application startup.
func RegisterMetadata(data []byte) error {
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.metadata = &meta

	return nil
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

// QueryResource finds a resource by name.
func QueryResource(name string) (*ResourceMetadata, error) {
	resources := QueryResources()
	for i := range resources {
		if resources[i].Name == name {
			return &resources[i], nil
		}
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
}
