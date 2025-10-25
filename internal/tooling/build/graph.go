package build

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// NodeType represents the type of dependency node
type NodeType int

const (
	NodeTypeSource NodeType = iota
	NodeTypeGenerated
	NodeTypeAsset
)

// Node represents a node in the dependency graph
type Node struct {
	Path         string
	Type         NodeType
	Hash         string
	LastModified time.Time
	Dependencies []string
}

// DependencyGraph tracks dependencies between files
type DependencyGraph struct {
	nodes map[string]*Node
	edges map[string][]string // file -> dependencies
	mu    sync.RWMutex
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*Node),
		edges: make(map[string][]string),
	}
}

// AddNode adds a node to the graph
func (dg *DependencyGraph) AddNode(path string, dependencies []string) error {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	// Compute file hash
	hash, err := computeFileHash(path)
	if err != nil {
		return fmt.Errorf("failed to hash %s: %w", path, err)
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}

	// Create node
	node := &Node{
		Path:         path,
		Type:         NodeTypeSource,
		Hash:         hash,
		LastModified: info.ModTime(),
		Dependencies: dependencies,
	}

	dg.nodes[path] = node
	dg.edges[path] = dependencies

	return nil
}

// GetNode retrieves a node by path
func (dg *DependencyGraph) GetNode(path string) (*Node, bool) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	node, ok := dg.nodes[path]
	return node, ok
}

// FindAffected finds all files affected by changes to the given files.
// This method is thread-safe for concurrent reads using RLock, but should not be
// called concurrently with AddNode or other mutation operations.
func (dg *DependencyGraph) FindAffected(changedFiles []string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	affected := make(map[string]struct{})

	// Recursive function to find dependents
	var visit func(string)
	visit = func(file string) {
		if _, ok := affected[file]; ok {
			return // Already visited
		}

		affected[file] = struct{}{}

		// Find all files that depend on this file
		for dependent, deps := range dg.edges {
			for _, dep := range deps {
				if dep == file {
					visit(dependent)
					break
				}
			}
		}
	}

	// Visit all changed files
	for _, file := range changedFiles {
		visit(file)
	}

	// Convert to slice
	result := make([]string, 0, len(affected))
	for file := range affected {
		result = append(result, file)
	}

	return result
}

// TopologicalSort returns files in build order (dependencies first)
func (dg *DependencyGraph) TopologicalSort(files []string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	// Calculate in-degree for each file
	// In our representation, edges[A] = [B, C] means "A depends on B and C"
	// So B and C must be built before A
	// Therefore, A has an in-degree based on how many dependencies it has
	inDegree := make(map[string]int)
	for _, file := range files {
		deps, ok := dg.edges[file]
		if !ok {
			inDegree[file] = 0
		} else {
			// Count only dependencies that are in our file list
			count := 0
			for _, dep := range deps {
				for _, f := range files {
					if f == dep {
						count++
						break
					}
				}
			}
			inDegree[file] = count
		}
	}

	// Find files with no incoming edges
	queue := make([]string, 0)
	for file, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, file)
		}
	}

	// Process queue
	result := make([]string, 0, len(files))
	for len(queue) > 0 {
		// Pop from queue
		file := queue[0]
		queue = queue[1:]
		result = append(result, file)

		// Find all files that depend on this file and reduce their in-degree
		for _, otherFile := range files {
			deps, ok := dg.edges[otherFile]
			if !ok {
				continue
			}

			// Check if otherFile depends on file
			for _, dep := range deps {
				if dep == file {
					if _, exists := inDegree[otherFile]; exists {
						inDegree[otherFile]--
						if inDegree[otherFile] == 0 {
							queue = append(queue, otherFile)
						}
					}
					break
				}
			}
		}
	}

	// If result doesn't contain all files, there's a cycle
	if len(result) != len(files) {
		// Find which files are missing (likely involved in cycle)
		missing := make([]string, 0)
		for _, file := range files {
			found := false
			for _, sorted := range result {
				if sorted == file {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, file)
			}
		}

		log.Printf("Warning: Circular dependency detected. Files involved: %v", missing)
		log.Printf("Returning original order. Compilation may fail.")

		// Return original order - let compiler detect the cycle
		return files
	}

	return result
}

// HasCycle detects if the dependency graph has a cycle
func (dg *DependencyGraph) HasCycle() bool {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(file string) bool {
		visited[file] = true
		recStack[file] = true

		deps, ok := dg.edges[file]
		if !ok {
			recStack[file] = false
			return false
		}

		for _, dep := range deps {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[file] = false
		return false
	}

	for file := range dg.nodes {
		if !visited[file] {
			if hasCycle(file) {
				return true
			}
		}
	}

	return false
}

// GetDependencies returns the dependencies of a file
func (dg *DependencyGraph) GetDependencies(file string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	deps, ok := dg.edges[file]
	if !ok {
		return []string{}
	}

	// Return a copy to avoid data races
	result := make([]string, len(deps))
	copy(result, deps)
	return result
}

// GetDependents returns all files that depend on the given file
func (dg *DependencyGraph) GetDependents(file string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	dependents := make([]string, 0)

	for dependent, deps := range dg.edges {
		for _, dep := range deps {
			if dep == file {
				dependents = append(dependents, dependent)
				break
			}
		}
	}

	return dependents
}

// computeFileHash computes SHA-256 hash of a file
func computeFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
