// Package schema provides relationship graph analysis for circular dependency detection
package schema

import (
	"fmt"
	"strings"
)

// RelationshipGraph represents the dependency graph between resources
type RelationshipGraph struct {
	nodes map[string]*ResourceSchema
	edges map[string][]string // resource -> dependencies
}

// NewRelationshipGraph creates a new relationship graph
func NewRelationshipGraph(schemas map[string]*ResourceSchema) *RelationshipGraph {
	graph := &RelationshipGraph{
		nodes: schemas,
		edges: make(map[string][]string),
	}

	// Build edges from belongs_to relationships
	for name, schema := range schemas {
		for _, rel := range schema.Relationships {
			if rel.Type == RelationshipBelongsTo {
				// This resource depends on target resource
				graph.edges[name] = append(graph.edges[name], rel.TargetResource)
			}
		}
	}

	return graph
}

// DetectCycles detects circular dependencies in the relationship graph
func (g *RelationshipGraph) DetectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var dfs func(node string, path []string) bool
	dfs = func(node string, path []string) bool {
		visited[node] = true
		recursionStack[node] = true
		path = append(path, node)

		for _, neighbor := range g.edges[node] {
			if !visited[neighbor] {
				if dfs(neighbor, path) {
					return true
				}
			} else if recursionStack[neighbor] {
				// Found cycle
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
				return true
			}
		}

		recursionStack[node] = false
		return false
	}

	for node := range g.nodes {
		if !visited[node] {
			dfs(node, []string{})
		}
	}

	return cycles
}

// TopologicalSort returns resources in dependency order (dependencies first)
func (g *RelationshipGraph) TopologicalSort() ([]string, error) {
	// Use out-degree: nodes with no dependencies should come first
	outDegree := make(map[string]int)
	for node := range g.nodes {
		outDegree[node] = len(g.edges[node])
	}

	// Build reverse edges for updating
	reverseEdges := make(map[string][]string)
	for source, targets := range g.edges {
		for _, target := range targets {
			reverseEdges[target] = append(reverseEdges[target], source)
		}
	}

	// Process nodes with no outgoing edges first (they don't depend on anything)
	queue := []string{}
	for node, degree := range outDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	result := []string{}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// For each resource that depends on this node
		for _, dependent := range reverseEdges[node] {
			outDegree[dependent]--
			if outDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(g.nodes) {
		cycles := g.DetectCycles()
		if len(cycles) > 0 {
			return nil, fmt.Errorf("circular dependency detected: %s", formatCycles(cycles))
		}
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

// GetDependencies returns all direct dependencies of a resource
func (g *RelationshipGraph) GetDependencies(resource string) []string {
	deps, exists := g.edges[resource]
	if !exists {
		return []string{}
	}
	return deps
}

// GetDependents returns all resources that depend on the given resource
func (g *RelationshipGraph) GetDependents(resource string) []string {
	dependents := []string{}
	for node, deps := range g.edges {
		for _, dep := range deps {
			if dep == resource {
				dependents = append(dependents, node)
				break
			}
		}
	}
	return dependents
}

// ValidateGraph validates the relationship graph
func (g *RelationshipGraph) ValidateGraph() error {
	// Check for cycles
	cycles := g.DetectCycles()
	if len(cycles) > 0 {
		return fmt.Errorf("circular dependencies detected:\n%s",
			formatCycles(cycles))
	}

	// Check for missing resources
	for _, schema := range g.nodes {
		for _, rel := range schema.Relationships {
			if rel.Type == RelationshipBelongsTo || rel.Type == RelationshipHasOne {
				if _, exists := g.nodes[rel.TargetResource]; !exists {
					return fmt.Errorf("resource %s references unknown resource %s in relationship %s",
						schema.Name, rel.TargetResource, rel.FieldName)
				}
			}
		}
	}

	return nil
}

// formatCycles formats cycle information for error messages
func formatCycles(cycles [][]string) string {
	var b strings.Builder
	for i, cycle := range cycles {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("  Cycle %d: %s -> %s",
			i+1,
			strings.Join(cycle, " -> "),
			cycle[0])) // Complete the cycle
	}
	return b.String()
}

// DependencyAnalyzer analyzes relationship dependencies
type DependencyAnalyzer struct {
	graph *RelationshipGraph
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(schemas map[string]*ResourceSchema) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		graph: NewRelationshipGraph(schemas),
	}
}

// Analyze performs comprehensive dependency analysis
func (a *DependencyAnalyzer) Analyze() (*DependencyReport, error) {
	report := &DependencyReport{
		TotalResources:  len(a.graph.nodes),
		Dependencies:    make(map[string][]string),
		Dependents:      make(map[string][]string),
		CircularDeps:    make([][]string, 0),
	}

	// Get dependencies and dependents for each resource
	for name := range a.graph.nodes {
		report.Dependencies[name] = a.graph.GetDependencies(name)
		report.Dependents[name] = a.graph.GetDependents(name)
	}

	// Check for cycles
	cycles := a.graph.DetectCycles()
	if len(cycles) > 0 {
		report.CircularDeps = cycles
		report.HasCycles = true
	}

	// Get topological order
	order, err := a.graph.TopologicalSort()
	if err == nil {
		report.TopologicalOrder = order
	}

	return report, nil
}

// DependencyReport contains the results of dependency analysis
type DependencyReport struct {
	TotalResources    int
	Dependencies      map[string][]string // resource -> direct dependencies
	Dependents        map[string][]string // resource -> resources that depend on it
	CircularDeps      [][]string          // list of circular dependency cycles
	HasCycles         bool
	TopologicalOrder  []string            // dependency-ordered list of resources
}

// String formats the dependency report
func (r *DependencyReport) String() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Dependency Analysis Report\n"))
	b.WriteString(fmt.Sprintf("Total Resources: %d\n\n", r.TotalResources))

	if r.HasCycles {
		b.WriteString("ERRORS:\n")
		b.WriteString("Circular dependencies detected:\n")
		b.WriteString(formatCycles(r.CircularDeps))
		b.WriteString("\n\n")
	}

	if len(r.TopologicalOrder) > 0 {
		b.WriteString("Dependency Order (safe creation order):\n")
		for i, resource := range r.TopologicalOrder {
			deps := r.Dependencies[resource]
			if len(deps) > 0 {
				b.WriteString(fmt.Sprintf("  %d. %s (depends on: %s)\n",
					i+1, resource, strings.Join(deps, ", ")))
			} else {
				b.WriteString(fmt.Sprintf("  %d. %s (no dependencies)\n", i+1, resource))
			}
		}
	}

	return b.String()
}

// RelationshipValidator validates relationships across resources
type RelationshipValidator struct {
	schemas map[string]*ResourceSchema
	errors  []error
}

// NewRelationshipValidator creates a new relationship validator
func NewRelationshipValidator(schemas map[string]*ResourceSchema) *RelationshipValidator {
	return &RelationshipValidator{
		schemas: schemas,
		errors:  make([]error, 0),
	}
}

// Validate validates all relationships
func (v *RelationshipValidator) Validate() error {
	// Build relationship graph
	graph := NewRelationshipGraph(v.schemas)

	// Validate the graph
	if err := graph.ValidateGraph(); err != nil {
		return err
	}

	// Validate foreign key types match
	for _, schema := range v.schemas {
		for _, rel := range schema.Relationships {
			if rel.Type == RelationshipBelongsTo {
				if err := v.validateForeignKeyType(schema, rel); err != nil {
					v.errors = append(v.errors, err)
				}
			}
		}
	}

	if len(v.errors) > 0 {
		return fmt.Errorf("relationship validation failed with %d errors", len(v.errors))
	}

	return nil
}

// validateForeignKeyType ensures foreign key type matches target primary key
func (v *RelationshipValidator) validateForeignKeyType(schema *ResourceSchema, rel *Relationship) error {
	targetSchema, exists := v.schemas[rel.TargetResource]
	if !exists {
		return fmt.Errorf("resource %s: relationship %s references unknown resource %s",
			schema.Name, rel.FieldName, rel.TargetResource)
	}

	targetPK, err := targetSchema.GetPrimaryKey()
	if err != nil {
		return fmt.Errorf("resource %s: relationship %s target resource %s has no primary key",
			schema.Name, rel.FieldName, rel.TargetResource)
	}

	// For now, we assume all primary keys are UUIDs
	// In the future, we could validate the actual types match
	if targetPK.Type.BaseType != TypeUUID {
		return fmt.Errorf("resource %s: relationship %s target primary key must be UUID",
			schema.Name, rel.FieldName)
	}

	return nil
}

// Errors returns all validation errors
func (v *RelationshipValidator) Errors() []error {
	return v.errors
}
