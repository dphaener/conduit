package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
)

// CompilationMetrics tracks performance metrics for compilation
type CompilationMetrics struct {
	TotalFiles      int
	CacheHits       int
	CacheMisses     int
	FilesCompiled   int
	ParallelBatches int
	TotalDuration   time.Duration
	LexingDuration  time.Duration
	ParsingDuration time.Duration
	CachingDuration time.Duration
	StartTime       time.Time
	EndTime         time.Time
}

// CacheHitRate returns the cache hit rate as a percentage
func (cm *CompilationMetrics) CacheHitRate() float64 {
	if cm.TotalFiles == 0 {
		return 0.0
	}
	return float64(cm.CacheHits) / float64(cm.TotalFiles) * 100.0
}

// CompilationResult represents the result of compiling a single file
type CompilationResult struct {
	Path    string
	Program *ast.Program
	Hash    string
	Err     error
	Cached  bool
}

// CompilationCoordinator manages incremental compilation with caching
type CompilationCoordinator struct {
	astCache *ASTCache
	depGraph *DependencyGraph
	hasher   *FileHasher
	metrics  *CompilationMetrics
	mu       sync.Mutex
}

// NewCompilationCoordinator creates a new compilation coordinator
func NewCompilationCoordinator() *CompilationCoordinator {
	return &CompilationCoordinator{
		astCache: NewASTCache(),
		depGraph: NewDependencyGraph(),
		hasher:   NewFileHasher(),
		metrics:  &CompilationMetrics{},
	}
}

// CompileFiles compiles multiple files with incremental compilation and caching
func (cc *CompilationCoordinator) CompileFiles(paths []string, parallel bool) ([]*CompilationResult, *CompilationMetrics, error) {
	cc.mu.Lock()
	cc.metrics = &CompilationMetrics{
		TotalFiles: len(paths),
		StartTime:  time.Now(),
	}
	cc.mu.Unlock()

	results := make([]*CompilationResult, len(paths))

	if parallel {
		// Compile files in parallel based on dependency graph
		results = cc.compileParallel(paths)
	} else {
		// Compile files sequentially
		results = cc.compileSequential(paths)
	}

	cc.mu.Lock()
	cc.metrics.EndTime = time.Now()
	cc.metrics.TotalDuration = cc.metrics.EndTime.Sub(cc.metrics.StartTime)
	metrics := cc.metrics
	cc.mu.Unlock()

	return results, metrics, nil
}

// compileSequential compiles files one by one
func (cc *CompilationCoordinator) compileSequential(paths []string) []*CompilationResult {
	results := make([]*CompilationResult, len(paths))

	for i, path := range paths {
		results[i] = cc.compileFile(path)
	}

	return results
}

// compileParallel compiles files in parallel respecting dependencies
func (cc *CompilationCoordinator) compileParallel(paths []string) []*CompilationResult {
	// Get topological order
	order, err := cc.depGraph.GetTopologicalOrder()
	if err != nil {
		// If there's a cycle or error, fall back to sequential
		return cc.compileSequential(paths)
	}

	// Build a map for quick lookup
	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}

	// Filter order to only include requested paths
	orderedPaths := make([]string, 0, len(paths))
	for _, p := range order {
		if pathSet[p] {
			orderedPaths = append(orderedPaths, p)
		}
	}

	// Add any paths not in the dependency graph
	for _, p := range paths {
		found := false
		for _, op := range orderedPaths {
			if op == p {
				found = true
				break
			}
		}
		if !found {
			orderedPaths = append(orderedPaths, p)
		}
	}

	// Compile in batches of independent files
	results := make([]*CompilationResult, len(orderedPaths))
	resultMap := make(map[string]*CompilationResult)
	var resultMu sync.Mutex

	compiled := make(map[string]bool)
	batchNum := 0

	for len(compiled) < len(orderedPaths) {
		// Find files ready to compile (all dependencies compiled)
		batch := make([]string, 0)
		for _, path := range orderedPaths {
			if compiled[path] {
				continue
			}

			// Check if all dependencies are compiled
			deps := cc.depGraph.GetDependencies(path)
			ready := true
			for _, dep := range deps {
				if !compiled[dep] {
					ready = false
					break
				}
			}

			if ready {
				batch = append(batch, path)
			}
		}

		if len(batch) == 0 {
			// No progress possible, break to avoid infinite loop
			break
		}

		// Compile batch in parallel
		batchNum++
		cc.mu.Lock()
		cc.metrics.ParallelBatches = batchNum
		cc.mu.Unlock()

		var wg sync.WaitGroup
		for _, path := range batch {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				result := cc.compileFile(p)

				// Protect map write with mutex
				resultMu.Lock()
				resultMap[p] = result
				resultMu.Unlock()
			}(path)
		}
		wg.Wait()

		// Mark as compiled
		for _, path := range batch {
			compiled[path] = true
		}
	}

	// Build results in original order
	for i, path := range orderedPaths {
		if result, exists := resultMap[path]; exists {
			results[i] = result
		} else {
			results[i] = &CompilationResult{
				Path: path,
				Err:  fmt.Errorf("file not compiled: %s", path),
			}
		}
	}

	return results
}

// compileFile compiles a single file with caching
func (cc *CompilationCoordinator) compileFile(path string) *CompilationResult {
	// Compute file hash
	hash, err := cc.hasher.HashFile(path)
	if err != nil {
		return &CompilationResult{
			Path: path,
			Err:  fmt.Errorf("failed to hash file: %w", err),
		}
	}

	// Check cache by path
	if cached, exists := cc.astCache.Get(path); exists {
		if cached.Hash == hash {
			// Cache hit
			cc.mu.Lock()
			cc.metrics.CacheHits++
			cc.mu.Unlock()

			return &CompilationResult{
				Path:    path,
				Program: cached.Program,
				Hash:    hash,
				Cached:  true,
			}
		}
		// Hash mismatch, invalidate cache
		cc.astCache.Invalidate(path)
	}

	// Check cache by hash (in case file was moved/renamed)
	if cached, exists := cc.astCache.GetByHash(hash); exists {
		// Cache hit by hash
		cc.mu.Lock()
		cc.metrics.CacheHits++
		cc.mu.Unlock()

		// Update cache with new path
		cc.astCache.Set(path, cached.Program, hash)

		return &CompilationResult{
			Path:    path,
			Program: cached.Program,
			Hash:    hash,
			Cached:  true,
		}
	}

	// Cache miss - compile the file
	cc.mu.Lock()
	cc.metrics.CacheMisses++
	cc.metrics.FilesCompiled++
	cc.mu.Unlock()

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return &CompilationResult{
			Path: path,
			Err:  fmt.Errorf("failed to read file: %w", err),
		}
	}

	// Lex
	lexStart := time.Now()
	lex := lexer.New(string(content))
	tokens, lexErrors := lex.ScanTokens()
	lexDuration := time.Since(lexStart)

	cc.mu.Lock()
	cc.metrics.LexingDuration += lexDuration
	cc.mu.Unlock()

	// Check for lexing errors
	if len(lexErrors) > 0 {
		return &CompilationResult{
			Path: path,
			Err:  fmt.Errorf("lexing errors: %d errors", len(lexErrors)),
		}
	}

	// Parse
	parseStart := time.Now()
	p := parser.New(tokens)
	program, parseErrors := p.Parse()
	parseDuration := time.Since(parseStart)

	cc.mu.Lock()
	cc.metrics.ParsingDuration += parseDuration
	cc.mu.Unlock()

	if len(parseErrors) > 0 {
		return &CompilationResult{
			Path: path,
			Err:  fmt.Errorf("parse errors: %d errors", len(parseErrors)),
		}
	}

	// Cache the result
	cacheStart := time.Now()
	cc.astCache.Set(path, program, hash)
	cacheDuration := time.Since(cacheStart)

	cc.mu.Lock()
	cc.metrics.CachingDuration += cacheDuration
	cc.mu.Unlock()

	// Build dependencies
	cc.depGraph.BuildDependencies(path, program)

	return &CompilationResult{
		Path:    path,
		Program: program,
		Hash:    hash,
		Cached:  false,
	}
}

// InvalidateFile invalidates a file and all its dependents
func (cc *CompilationCoordinator) InvalidateFile(path string) []string {
	// Get transitive dependents
	dependents := cc.depGraph.GetTransitiveDependents(path)

	// Invalidate cache for the file and all dependents
	cc.astCache.Invalidate(path)
	for _, dep := range dependents {
		cc.astCache.Invalidate(dep)
	}

	// Return list of invalidated files
	invalidated := append([]string{path}, dependents...)
	return invalidated
}

// GetMetrics returns the current compilation metrics
func (cc *CompilationCoordinator) GetMetrics() *CompilationMetrics {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Return a copy
	metrics := *cc.metrics
	return &metrics
}

// GetCacheStats returns cache statistics
func (cc *CompilationCoordinator) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_size":     cc.astCache.Size(),
		"dep_graph_size": cc.depGraph.Size(),
	}
}

// Clear clears all caches and dependency graph
func (cc *CompilationCoordinator) Clear() {
	cc.astCache.InvalidateAll()
	cc.depGraph.Clear()
	cc.mu.Lock()
	cc.metrics = &CompilationMetrics{}
	cc.mu.Unlock()
}

// WatchModeCompile is optimized for watch mode - keeps ASTs in memory
func (cc *CompilationCoordinator) WatchModeCompile(changedFiles []string) ([]*CompilationResult, *CompilationMetrics, error) {
	// Invalidate changed files and their dependents
	allInvalidated := make(map[string]bool)
	for _, path := range changedFiles {
		invalidated := cc.InvalidateFile(path)
		for _, inv := range invalidated {
			allInvalidated[inv] = true
		}
	}

	// Convert to slice
	filesToCompile := make([]string, 0, len(allInvalidated))
	for path := range allInvalidated {
		filesToCompile = append(filesToCompile, path)
	}

	// Compile with parallelization
	return cc.CompileFiles(filesToCompile, true)
}

// ScanDirectory scans a directory for .cdt files
func ScanDirectory(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".cdt" {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
