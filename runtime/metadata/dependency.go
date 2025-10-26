package metadata

import (
	"fmt"
)

// DependencyOptions configures dependency graph queries
type DependencyOptions struct {
	Depth   int      // Maximum traversal depth (0 = unlimited)
	Reverse bool     // Reverse traversal (find what depends on this)
	Types   []string // Filter by edge types (e.g., ["belongs_to", "has_many"])
}

// BuildDependencyGraph constructs a complete dependency graph from metadata
func BuildDependencyGraph(meta *Metadata) *DependencyGraph {
	graph := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make([]DependencyEdge, 0),
	}

	if meta == nil {
		return graph
	}

	// Add all resources as nodes
	for _, resource := range meta.Resources {
		node := &DependencyNode{
			ID:       resource.Name,
			Type:     "resource",
			Name:     resource.Name,
			FilePath: resource.FilePath,
		}
		graph.Nodes[resource.Name] = node

		// Add edges for relationships
		for _, rel := range resource.Relationships {
			edge := DependencyEdge{
				From:         resource.Name,
				To:           rel.TargetResource,
				Relationship: rel.Type,
				Weight:       1,
			}
			graph.Edges = append(graph.Edges, edge)

			// Ensure target resource node exists
			if _, exists := graph.Nodes[rel.TargetResource]; !exists {
				graph.Nodes[rel.TargetResource] = &DependencyNode{
					ID:       rel.TargetResource,
					Type:     "resource",
					Name:     rel.TargetResource,
					FilePath: "", // Will be filled in when we process that resource
				}
			}
		}
	}

	return graph
}

// QueryDependencies finds dependencies of a resource with configurable options
func QueryDependencies(resourceName string, opts DependencyOptions) (*DependencyGraph, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	if !globalRegistry.initialized.Load() {
		return nil, fmt.Errorf("registry not initialized")
	}

	// Check if resource exists
	if _, ok := globalRegistry.resourcesByName[resourceName]; !ok {
		return nil, fmt.Errorf("resource not found: %s", resourceName)
	}

	// Check cache first
	cacheKey := fmt.Sprintf("deps:%s:%d:%v:%v", resourceName, opts.Depth, opts.Reverse, opts.Types)
	if cached := globalRegistry.getCached(cacheKey); cached != nil {
		return cached.(*DependencyGraph), nil
	}

	// Build full dependency graph from metadata
	fullGraph := BuildDependencyGraph(globalRegistry.metadata)

	// Extract subgraph starting from resourceName
	result := extractSubgraph(fullGraph, resourceName, opts)

	// Cache the result
	globalRegistry.setCached(cacheKey, result)

	return result, nil
}

// extractSubgraph extracts a subgraph using BFS traversal
func extractSubgraph(fullGraph *DependencyGraph, startNode string, opts DependencyOptions) *DependencyGraph {
	result := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make([]DependencyEdge, 0),
	}

	visited := make(map[string]bool)
	queue := []depthNode{{id: startNode, depth: 0}}

	// Always add the start node
	if node, exists := fullGraph.Nodes[startNode]; exists {
		result.Nodes[startNode] = node
	}
	visited[startNode] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Find relevant edges
		var edges []DependencyEdge
		if opts.Reverse {
			edges = findIncomingEdges(fullGraph, current.id)
		} else {
			edges = findOutgoingEdges(fullGraph, current.id)
		}

		// Filter by type if specified
		if len(opts.Types) > 0 {
			edges = filterEdgesByType(edges, opts.Types)
		}

		// Add edges and queue next nodes
		for _, edge := range edges {
			result.Edges = append(result.Edges, edge)

			// Determine next node
			nextNode := edge.To
			if opts.Reverse {
				nextNode = edge.From
			}

			// Add node if not visited
			if !visited[nextNode] {
				visited[nextNode] = true

				// Add the node to result
				if node, exists := fullGraph.Nodes[nextNode]; exists {
					result.Nodes[nextNode] = node
				}

				// Check depth limit for next level before queuing
				if opts.Depth == 0 || current.depth+1 < opts.Depth {
					// Add to queue for further traversal
					queue = append(queue, depthNode{id: nextNode, depth: current.depth + 1})
				}
			}
		}
	}

	return result
}

// depthNode tracks a node and its depth during traversal
type depthNode struct {
	id    string
	depth int
}

// findOutgoingEdges finds all edges from a node
func findOutgoingEdges(graph *DependencyGraph, nodeID string) []DependencyEdge {
	var result []DependencyEdge
	for _, edge := range graph.Edges {
		if edge.From == nodeID {
			result = append(result, edge)
		}
	}
	return result
}

// findIncomingEdges finds all edges to a node
func findIncomingEdges(graph *DependencyGraph, nodeID string) []DependencyEdge {
	var result []DependencyEdge
	for _, edge := range graph.Edges {
		if edge.To == nodeID {
			result = append(result, edge)
		}
	}
	return result
}

// filterEdgesByType filters edges by relationship type
func filterEdgesByType(edges []DependencyEdge, types []string) []DependencyEdge {
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	var result []DependencyEdge
	for _, edge := range edges {
		if typeSet[edge.Relationship] {
			result = append(result, edge)
		}
	}
	return result
}

// DetectCycles detects circular dependencies in the graph
func DetectCycles(graph *DependencyGraph) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	for nodeID := range graph.Nodes {
		if !visited[nodeID] {
			findCycles(graph, nodeID, visited, recStack, path, &cycles)
		}
	}

	return cycles
}

// findCycles performs DFS to find cycles
func findCycles(graph *DependencyGraph, nodeID string, visited, recStack map[string]bool, path []string, cycles *[][]string) {
	visited[nodeID] = true
	recStack[nodeID] = true
	path = append(path, nodeID)

	// Find all outgoing edges
	for _, edge := range graph.Edges {
		if edge.From != nodeID {
			continue
		}

		nextNode := edge.To

		// If node is in recursion stack, we found a cycle
		if recStack[nextNode] {
			// Extract the cycle from path
			cycleStart := -1
			for i, n := range path {
				if n == nextNode {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := make([]string, len(path)-cycleStart)
				copy(cycle, path[cycleStart:])
				cycle = append(cycle, nextNode) // Close the cycle
				*cycles = append(*cycles, cycle)
			}
		} else if !visited[nextNode] {
			findCycles(graph, nextNode, visited, recStack, path, cycles)
		}
	}

	recStack[nodeID] = false
}

// GetDependencyDepth calculates the maximum dependency depth for a resource
func GetDependencyDepth(resourceName string) (int, error) {
	opts := DependencyOptions{
		Depth:   0, // Unlimited
		Reverse: false,
	}

	graph, err := QueryDependencies(resourceName, opts)
	if err != nil {
		return 0, err
	}

	// Calculate maximum depth using BFS
	maxDepth := 0
	visited := make(map[string]int)
	queue := []depthNode{{id: resourceName, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if depth, ok := visited[current.id]; ok && depth >= current.depth {
			continue
		}
		visited[current.id] = current.depth

		if current.depth > maxDepth {
			maxDepth = current.depth
		}

		// Find outgoing edges
		edges := findOutgoingEdges(graph, current.id)
		for _, edge := range edges {
			queue = append(queue, depthNode{id: edge.To, depth: current.depth + 1})
		}
	}

	return maxDepth, nil
}

// CountDependents counts how many resources depend on a given resource
func CountDependents(resourceName string) (int, error) {
	refs := QueryRelationshipsTo(resourceName)
	return len(refs), nil
}

// CountDependencies counts how many resources this resource depends on
func CountDependencies(resourceName string) (int, error) {
	rels, err := QueryRelationshipsFrom(resourceName)
	if err != nil {
		return 0, err
	}
	return len(rels), nil
}
