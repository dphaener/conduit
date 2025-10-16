# Incremental Compilation and Build Caching

This package implements incremental compilation and build caching for the Conduit compiler, achieving sub-300ms compilation times for single file changes in large projects.

## Overview

The incremental compilation system consists of four main components:

1. **File Content Hashing** (`hash.go`) - SHA-256 based cache keys
2. **AST Caching** (`ast_cache.go`) - In-memory storage of parsed ASTs
3. **Dependency Tracking** (`dependencies.go`) - File and resource dependency graph
4. **Compilation Coordinator** (`coordinator.go`) - Orchestrates incremental compilation with parallelization

## Features

### File Content Hashing

Computes SHA-256 hashes of file contents to detect changes:

```go
hasher := cache.NewFileHasher()
hash, err := hasher.HashFile("/path/to/file.cdt")
```

- Deterministic hashing for reliable cache keys
- Support for file paths, byte arrays, and strings
- 64-character hex-encoded hashes

### AST Caching

Thread-safe in-memory cache for parsed ASTs:

```go
astCache := cache.NewASTCache()
astCache.Set(path, program, hash)

if cached, exists := astCache.Get(path); exists {
    // Use cached AST
}
```

- **Cache by path**: Fast lookups for known files
- **Cache by hash**: Handles file moves/renames
- **Automatic pruning**: Remove stale entries based on age
- **Thread-safe**: Concurrent access from multiple goroutines

### Dependency Tracking

Tracks relationships between files to enable smart invalidation:

```go
depGraph := cache.NewDependencyGraph()
depGraph.AddFile(path, resourceName)
depGraph.AddDependency(fileA, fileB) // fileA depends on fileB

// Get files that need recompilation when fileA changes
dependents := depGraph.GetTransitiveDependents(fileA)
```

Features:
- **Transitive dependencies**: Automatically invalidate dependent files
- **Topological ordering**: Compile files in correct dependency order
- **Cycle detection**: Detect and report circular dependencies
- **Parallel batching**: Group independent files for parallel compilation

### Compilation Coordinator

Orchestrates incremental compilation with caching and parallelization:

```go
coordinator := cache.NewCompilationCoordinator()

// Compile files with caching and parallelization
results, metrics, err := coordinator.CompileFiles(paths, parallel=true)

// Watch mode: compile only changed files and their dependents
results, metrics, err := coordinator.WatchModeCompile(changedFiles)
```

## Performance Metrics

The system tracks detailed performance metrics:

```go
type CompilationMetrics struct {
    TotalFiles       int           // Total files requested
    CacheHits        int           // Files served from cache
    CacheMisses      int           // Files compiled
    FilesCompiled    int           // Actual compilations performed
    ParallelBatches  int           // Number of parallel batches
    TotalDuration    time.Duration // Total compilation time
    LexingDuration   time.Duration // Time spent lexing
    ParsingDuration  time.Duration // Time spent parsing
    CachingDuration  time.Duration // Time spent caching
}
```

### Performance Targets

Based on ticket CON-9 requirements:

- **First compilation**: 3 seconds for 50 resources
- **Incremental compilation**: < 300ms for single file change
- **Cache hit rate**: > 80% in watch mode

## Usage Examples

### Basic Compilation with Caching

```go
coordinator := cache.NewCompilationCoordinator()

// First compilation - builds cache
files := []string{"/src/user.cdt", "/src/post.cdt"}
results, metrics, err := coordinator.CompileFiles(files, false)

fmt.Printf("Compiled %d files in %v\n", metrics.FilesCompiled, metrics.TotalDuration)
fmt.Printf("Cache hit rate: %.2f%%\n", metrics.CacheHitRate())

// Second compilation - uses cache
results2, metrics2, _ := coordinator.CompileFiles(files, false)
fmt.Printf("Cache hits: %d, misses: %d\n", metrics2.CacheHits, metrics2.CacheMisses)
```

### Parallel Compilation

```go
// Compile independent files in parallel
results, metrics, err := coordinator.CompileFiles(files, true)

fmt.Printf("Compiled in %d parallel batches\n", metrics.ParallelBatches)
```

### Watch Mode

```go
// Initial compilation
coordinator.CompileFiles(allFiles, true)

// User modifies user.cdt
changedFiles := []string{"/src/user.cdt"}

// Automatically recompile changed file and dependents
results, metrics, err := coordinator.WatchModeCompile(changedFiles)

fmt.Printf("Recompiled %d files in %v\n", metrics.FilesCompiled, metrics.TotalDuration)
```

### Manual Cache Invalidation

```go
// Invalidate a file and all its dependents
invalidated := coordinator.InvalidateFile("/src/user.cdt")
fmt.Printf("Invalidated %d files\n", len(invalidated))

// Clear entire cache
coordinator.Clear()
```

## Architecture

### Compilation Flow

```
┌─────────────────┐
│  Source Files   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Hash Files     │ ◄── FileHasher
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Check Cache    │ ◄── ASTCache
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
Cache Hit  Cache Miss
    │         │
    │         ▼
    │    ┌─────────────────┐
    │    │  Lex & Parse    │
    │    └────────┬────────┘
    │             │
    │             ▼
    │    ┌─────────────────┐
    │    │  Update Cache   │
    │    └────────┬────────┘
    │             │
    └─────┬───────┘
          │
          ▼
   ┌─────────────────┐
   │  Build Deps     │ ◄── DependencyGraph
   └────────┬────────┘
            │
            ▼
     ┌─────────────────┐
     │  Return ASTs    │
     └─────────────────┘
```

### Parallel Compilation Flow

```
1. Build dependency graph
2. Compute topological order
3. Group files into independent batches:
   Batch 1: [A, B, C] (no dependencies)
   Batch 2: [D, E]    (depend on A, B)
   Batch 3: [F]       (depends on D)
4. Compile each batch in parallel
5. Wait for batch completion before next batch
```

## Testing

Comprehensive test coverage across all components:

- **Unit tests**: Hash, cache, and dependency operations
- **Integration tests**: End-to-end compilation scenarios
- **Performance tests**: Verify < 300ms incremental compilation

Run tests:

```bash
go test ./internal/compiler/cache/...
```

Run with coverage:

```bash
go test -cover ./internal/compiler/cache/...
```

## Thread Safety

All components are designed for concurrent access:

- **ASTCache**: RWMutex for thread-safe reads and writes
- **DependencyGraph**: RWMutex for thread-safe graph operations
- **CompilationCoordinator**: Mutex for metrics updates

## Future Enhancements

Potential improvements not in the current scope:

1. **Persistent cache**: Save ASTs to disk for cross-session caching
2. **Distributed caching**: Share cache across multiple machines
3. **Smart dependency detection**: Analyze AST to detect which changes require recompilation
4. **Profile-guided optimization**: Use compilation profiles to optimize batch sizes
5. **Incremental type checking**: Only type-check changed portions of the AST

## Related Documentation

- `IMPLEMENTATION-COMPILER.md` - Compiler implementation guide
- `ARCHITECTURE.md` - System architecture overview
- Ticket CON-9 - Original requirements

## License

Part of the Conduit compiler, same license applies.
