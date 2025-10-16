package cache

import (
	"sync"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// FileDependency represents a dependency between files
type FileDependency struct {
	Path         string   // The file path
	DependsOn    []string // Files this file depends on
	DependedBy   []string // Files that depend on this file
	ResourceName string   // Primary resource name in this file (if any)
}

// DependencyGraph tracks dependencies between source files
type DependencyGraph struct {
	nodes map[string]*FileDependency
	mu    sync.RWMutex
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*FileDependency),
	}
}

// AddFile adds a file to the dependency graph
func (dg *DependencyGraph) AddFile(path string, resourceName string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	if _, exists := dg.nodes[path]; !exists {
		dg.nodes[path] = &FileDependency{
			Path:         path,
			DependsOn:    make([]string, 0),
			DependedBy:   make([]string, 0),
			ResourceName: resourceName,
		}
	} else {
		dg.nodes[path].ResourceName = resourceName
	}
}

// AddDependency adds a dependency relationship: from depends on to
func (dg *DependencyGraph) AddDependency(from, to string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	// Ensure both nodes exist
	if _, exists := dg.nodes[from]; !exists {
		dg.nodes[from] = &FileDependency{
			Path:       from,
			DependsOn:  make([]string, 0),
			DependedBy: make([]string, 0),
		}
	}
	if _, exists := dg.nodes[to]; !exists {
		dg.nodes[to] = &FileDependency{
			Path:       to,
			DependsOn:  make([]string, 0),
			DependedBy: make([]string, 0),
		}
	}

	// Add the dependency (avoid duplicates)
	if !contains(dg.nodes[from].DependsOn, to) {
		dg.nodes[from].DependsOn = append(dg.nodes[from].DependsOn, to)
	}
	if !contains(dg.nodes[to].DependedBy, from) {
		dg.nodes[to].DependedBy = append(dg.nodes[to].DependedBy, from)
	}
}

// GetDependencies returns the files that the given file depends on
func (dg *DependencyGraph) GetDependencies(path string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	if node, exists := dg.nodes[path]; exists {
		result := make([]string, len(node.DependsOn))
		copy(result, node.DependsOn)
		return result
	}
	return []string{}
}

// GetDependents returns the files that depend on the given file
func (dg *DependencyGraph) GetDependents(path string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	if node, exists := dg.nodes[path]; exists {
		result := make([]string, len(node.DependedBy))
		copy(result, node.DependedBy)
		return result
	}
	return []string{}
}

// GetTransitiveDependents returns all files that transitively depend on the given file
func (dg *DependencyGraph) GetTransitiveDependents(path string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	visited := make(map[string]bool)
	result := make([]string, 0)

	var visit func(string)
	visit = func(p string) {
		if visited[p] {
			return
		}
		visited[p] = true

		if node, exists := dg.nodes[p]; exists {
			for _, dependent := range node.DependedBy {
				result = append(result, dependent)
				visit(dependent)
			}
		}
	}

	visit(path)
	return result
}

// GetIndependentFiles returns files that have no dependencies
// These can be compiled in parallel
func (dg *DependencyGraph) GetIndependentFiles() []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	result := make([]string, 0)
	for path, node := range dg.nodes {
		if len(node.DependsOn) == 0 {
			result = append(result, path)
		}
	}
	return result
}

// GetTopologicalOrder returns files in topological order for compilation
// Files with no dependencies come first
func (dg *DependencyGraph) GetTopologicalOrder() ([]string, error) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	// Make a copy of the graph for Kahn's algorithm
	inDegree := make(map[string]int)
	for path, node := range dg.nodes {
		inDegree[path] = len(node.DependsOn)
	}

	// Queue of nodes with no dependencies
	queue := make([]string, 0)
	for path, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, path)
		}
	}

	result := make([]string, 0, len(dg.nodes))

	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree for dependents
		if node, exists := dg.nodes[current]; exists {
			for _, dependent := range node.DependedBy {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					queue = append(queue, dependent)
				}
			}
		}
	}

	// Check for cycles
	if len(result) != len(dg.nodes) {
		return nil, &CycleError{
			Message: "circular dependency detected in resource graph",
		}
	}

	return result, nil
}

// RemoveFile removes a file and its dependencies from the graph
func (dg *DependencyGraph) RemoveFile(path string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	if node, exists := dg.nodes[path]; exists {
		// Remove this file from dependents' DependsOn lists
		for _, dependent := range node.DependedBy {
			if depNode, exists := dg.nodes[dependent]; exists {
				depNode.DependsOn = removeString(depNode.DependsOn, path)
			}
		}

		// Remove this file from dependencies' DependedBy lists
		for _, dependency := range node.DependsOn {
			if depNode, exists := dg.nodes[dependency]; exists {
				depNode.DependedBy = removeString(depNode.DependedBy, path)
			}
		}

		delete(dg.nodes, path)
	}
}

// Clear removes all entries from the dependency graph
func (dg *DependencyGraph) Clear() {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	dg.nodes = make(map[string]*FileDependency)
}

// Size returns the number of files in the graph
func (dg *DependencyGraph) Size() int {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	return len(dg.nodes)
}

// BuildDependencies analyzes an AST to extract dependencies
func (dg *DependencyGraph) BuildDependencies(path string, program *ast.Program) {
	// Extract resource name
	var resourceName string
	if len(program.Resources) > 0 {
		resourceName = program.Resources[0].Name
	}

	dg.AddFile(path, resourceName)

	// Track relationships to other resources
	resourceDeps := make(map[string]bool)

	for _, resource := range program.Resources {
		// Check relationships
		for _, rel := range resource.Relationships {
			if rel.Type != "" {
				resourceDeps[rel.Type] = true
			}
		}

		// Check field types (for resource references)
		for _, field := range resource.Fields {
			if field.Type.Kind == ast.TypeResource && field.Type.Name != "" {
				resourceDeps[field.Type.Name] = true
			}
		}
	}

	// Note: We can't map resource names to file paths without a registry
	// This would be done by the compiler coordinator
	// For now, we just track the resource dependencies
}

// CycleError represents a circular dependency error
type CycleError struct {
	Message string
}

func (e *CycleError) Error() string {
	return e.Message
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
