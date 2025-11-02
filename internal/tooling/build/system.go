package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/codegen"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/compiler/typechecker"
	"github.com/conduit-lang/conduit/internal/utils"
)

// BuildMode represents the compilation mode
type BuildMode int

const (
	// Development mode with debug symbols and no optimizations
	ModeDevelopment BuildMode = iota
	// Production mode with optimizations and stripped symbols
	ModeProduction
	// Test mode for running tests
	ModeTest
)

func (m BuildMode) String() string {
	switch m {
	case ModeDevelopment:
		return "development"
	case ModeProduction:
		return "production"
	case ModeTest:
		return "test"
	default:
		return "unknown"
	}
}

// BuildOptions configures the build process
type BuildOptions struct {
	Mode         BuildMode
	OutputPath   string
	SourceDir    string
	BuildDir     string
	Parallel     bool
	MaxJobs      int
	Verbose      bool
	Watch        bool
	UseCache     bool
	Minify       bool
	TreeShake    bool
	ProgressFunc func(current, total int, message string)
}

// DefaultBuildOptions returns sensible defaults
func DefaultBuildOptions() *BuildOptions {
	return &BuildOptions{
		Mode:       ModeDevelopment,
		OutputPath: "build/app",
		SourceDir:  "app",
		BuildDir:   "build",
		Parallel:   true,
		MaxJobs:    runtime.NumCPU(),
		Verbose:    false,
		Watch:      false,
		UseCache:   true,
		Minify:     false,
		TreeShake:  false,
	}
}

// BuildResult contains information about the build
type BuildResult struct {
	Success       bool
	OutputPath    string
	MetadataPath  string
	Duration      time.Duration
	FilesCompiled int
	CacheHits     int
	Errors        []BuildError
}

// BuildError represents a build-time error
type BuildError struct {
	Phase    string
	File     string
	Line     int
	Column   int
	Message  string
}

// System coordinates the entire build process.
// Thread-safety: The System is not designed for concurrent access.
// The embedded components (Cache, DependencyGraph) have their own
// internal synchronization where needed.
type System struct {
	options   *BuildOptions
	depGraph  *DependencyGraph
	cache     *Cache
	assets    *AssetCompiler
	optimizer *Optimizer
}

// NewSystem creates a new build system
func NewSystem(opts *BuildOptions) (*System, error) {
	if opts == nil {
		opts = DefaultBuildOptions()
	}

	// Initialize cache
	cacheDir := filepath.Join(opts.BuildDir, ".cache")
	buildCache, err := NewCache(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	// Initialize dependency graph
	depGraph := NewDependencyGraph()

	// Initialize asset compiler
	assetCompiler := NewAssetCompiler()

	// Initialize optimizer
	optimizer := NewOptimizer(opts.Mode)

	return &System{
		options:   opts,
		depGraph:  depGraph,
		cache:     buildCache,
		assets:    assetCompiler,
		optimizer: optimizer,
	}, nil
}

// Build performs a full build
func (s *System) Build(ctx context.Context) (*BuildResult, error) {
	startTime := time.Now()

	result := &BuildResult{
		Success: false,
		Errors:  make([]BuildError, 0),
	}

	// Find all source files
	sourceFiles, err := s.findSourceFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find source files: %w", err)
	}

	if len(sourceFiles) == 0 {
		return nil, fmt.Errorf("no .cdt files found in %s", s.options.SourceDir)
	}

	// Progress reporting
	if s.options.ProgressFunc != nil {
		s.options.ProgressFunc(0, len(sourceFiles), "Starting build...")
	}

	// Build dependency graph
	if err := s.buildDependencyGraph(sourceFiles); err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Determine build order (topological sort)
	buildOrder := s.depGraph.TopologicalSort(sourceFiles)

	// Compile files
	compiled, cacheHits, errors := s.compileFiles(ctx, buildOrder)

	if len(errors) > 0 {
		result.Errors = errors
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Handle schema-based migration generation
	if err := s.handleMigrations(compiled); err != nil {
		return nil, fmt.Errorf("migration generation failed: %w", err)
	}

	// Generate Go code
	generatedFiles, err := s.generateGoCode(compiled)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	// Write generated files
	if err := s.writeGeneratedFiles(generatedFiles); err != nil {
		return nil, fmt.Errorf("failed to write generated files: %w", err)
	}

	// Compile assets
	if err := s.compileAssets(); err != nil {
		return nil, fmt.Errorf("asset compilation failed: %w", err)
	}

	// Build binary
	outputPath, err := s.buildBinary()
	if err != nil {
		return nil, fmt.Errorf("binary build failed: %w", err)
	}

	// Generate metadata
	metadataPath, err := s.generateMetadata(compiled)
	if err != nil {
		return nil, fmt.Errorf("metadata generation failed: %w", err)
	}

	result.Success = true
	result.OutputPath = outputPath
	result.MetadataPath = metadataPath
	result.Duration = time.Since(startTime)
	result.FilesCompiled = len(sourceFiles)
	result.CacheHits = cacheHits

	return result, nil
}

// IncrementalBuild performs an incremental build of changed files.
//
// LIMITATION: This implementation loads ALL source file ASTs from cache to generate
// complete Go code. While it only recompiles changed/affected files, the code generation
// step requires the full program AST. This means:
// - Recompilation is fast (only affected files)
// - Code generation still processes all files (from cache)
// - Memory usage scales with total project size
//
// A future optimization could implement partial code generation or maintain
// a persistent program state to avoid loading all cached ASTs.
func (s *System) IncrementalBuild(ctx context.Context, changedFiles []string) (*BuildResult, error) {
	startTime := time.Now()

	result := &BuildResult{
		Success: false,
		Errors:  make([]BuildError, 0),
	}

	// Find affected files
	affected := s.depGraph.FindAffected(changedFiles)

	if len(affected) == 0 {
		// No files affected - this is a success
		result.Success = true
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Progress reporting
	if s.options.ProgressFunc != nil {
		s.options.ProgressFunc(0, len(affected), fmt.Sprintf("Incremental build: %d files", len(affected)))
	}

	// Determine build order
	buildOrder := s.depGraph.TopologicalSort(affected)

	// Compile affected files
	compiled, cacheHits, errors := s.compileFiles(ctx, buildOrder)

	if len(errors) > 0 {
		result.Errors = errors
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Load cached ASTs for non-affected files
	// NOTE: This loads ALL files from cache to enable full code generation.
	// See function documentation for details on this limitation.
	allFiles, err := s.findSourceFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find source files: %w", err)
	}

	for _, file := range allFiles {
		isAffected := false
		for _, af := range affected {
			if af == file {
				isAffected = true
				break
			}
		}
		if !isAffected {
			// Try to load from cache
			if cached, err := s.cache.Get(file); err == nil {
				compiled = append(compiled, cached)
			}
		}
	}

	// Handle schema-based migration generation
	if err := s.handleMigrations(compiled); err != nil {
		return nil, fmt.Errorf("migration generation failed: %w", err)
	}

	// Generate Go code for all files
	generatedFiles, err := s.generateGoCode(compiled)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	// Write generated files
	if err := s.writeGeneratedFiles(generatedFiles); err != nil {
		return nil, fmt.Errorf("failed to write generated files: %w", err)
	}

	// Build binary
	outputPath, err := s.buildBinary()
	if err != nil {
		return nil, fmt.Errorf("binary build failed: %w", err)
	}

	// Generate metadata
	metadataPath, err := s.generateMetadata(compiled)
	if err != nil {
		return nil, fmt.Errorf("metadata generation failed: %w", err)
	}

	result.Success = true
	result.OutputPath = outputPath
	result.MetadataPath = metadataPath
	result.Duration = time.Since(startTime)
	result.FilesCompiled = len(affected)
	result.CacheHits = cacheHits

	return result, nil
}

// findSourceFiles finds all .cdt files in the source directory
func (s *System) findSourceFiles() ([]string, error) {
	// Find all .cdt files recursively
	files, err := utils.FindCdtFiles(s.options.SourceDir)
	if err != nil {
		return nil, err
	}

	// Convert to absolute paths
	absFiles := make([]string, len(files))
	for i, file := range files {
		abs, err := filepath.Abs(file)
		if err != nil {
			return nil, err
		}
		absFiles[i] = abs
	}

	return absFiles, nil
}

// buildDependencyGraph builds the dependency graph for all files
func (s *System) buildDependencyGraph(files []string) error {
	for _, file := range files {
		// Parse file to extract dependencies
		source, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Quick parse to find imports/dependencies
		deps := s.extractDependencies(string(source))

		// Add to graph
		s.depGraph.AddNode(file, deps)
	}

	return nil
}

// extractDependencies extracts dependencies from source code
func (s *System) extractDependencies(source string) []string {
	// For now, Conduit doesn't have explicit imports
	// Dependencies are implicit through resource relationships
	// This will be enhanced when we add module system
	return []string{}
}

// compileFiles compiles a list of files in order
func (s *System) compileFiles(ctx context.Context, files []string) ([]*CompiledFile, int, []BuildError) {
	compiled := make([]*CompiledFile, 0, len(files))
	errors := make([]BuildError, 0)
	cacheHits := 0

	if s.options.Parallel {
		// Parallel compilation
		return s.compileFilesParallel(ctx, files)
	}

	// Sequential compilation
	for i, file := range files {
		// Progress reporting
		if s.options.ProgressFunc != nil {
			s.options.ProgressFunc(i+1, len(files), fmt.Sprintf("Compiling %s", filepath.Base(file)))
		}

		// Check cache
		if s.options.UseCache {
			if cached, err := s.cache.Get(file); err == nil {
				compiled = append(compiled, cached)
				cacheHits++
				continue
			}
		}

		// Compile file
		result, err := s.compileFile(file)
		if err != nil {
			errors = append(errors, BuildError{
				Phase:   "compilation",
				File:    file,
				Message: err.Error(),
			})
			continue
		}

		compiled = append(compiled, result)

		// Store in cache
		if s.options.UseCache {
			if err := s.cache.Put(file, result); err != nil {
				// Cache failure is non-fatal
				if s.options.Verbose {
					fmt.Printf("Warning: failed to cache %s: %v\n", file, err)
				}
			}
		}
	}

	return compiled, cacheHits, errors
}

// compileFilesParallel compiles files in parallel
func (s *System) compileFilesParallel(ctx context.Context, files []string) ([]*CompiledFile, int, []BuildError) {
	type result struct {
		compiled  *CompiledFile
		err       error
		cacheHit  bool
		fileIndex int
	}

	jobs := make(chan string, len(files))
	results := make(chan result, len(files))

	// Start workers
	numWorkers := s.options.MaxJobs
	if numWorkers > len(files) {
		numWorkers = len(files)
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				// Find file index
				fileIndex := -1
				for idx, f := range files {
					if f == file {
						fileIndex = idx
						break
					}
				}

				// Check cache
				if s.options.UseCache {
					if cached, err := s.cache.Get(file); err == nil {
						results <- result{
							compiled:  cached,
							cacheHit:  true,
							fileIndex: fileIndex,
						}
						continue
					}
				}

				// Compile
				compiled, err := s.compileFile(file)
				results <- result{
					compiled:  compiled,
					err:       err,
					cacheHit:  false,
					fileIndex: fileIndex,
				}

				// Store in cache
				if err == nil && s.options.UseCache {
					s.cache.Put(file, compiled)
				}
			}
		}()
	}

	// Send jobs
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// Wait for workers
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results using map to preserve ordering
	compiledMap := make(map[int]*CompiledFile)
	errors := make([]BuildError, 0)
	cacheHits := 0
	completed := 0

	for res := range results {
		completed++

		// Progress reporting
		if s.options.ProgressFunc != nil {
			s.options.ProgressFunc(completed, len(files), fmt.Sprintf("Compiled %d/%d files", completed, len(files)))
		}

		if res.err != nil {
			errors = append(errors, BuildError{
				Phase:   "compilation",
				File:    files[res.fileIndex],
				Message: res.err.Error(),
			})
			continue
		}

		if res.cacheHit {
			cacheHits++
		}

		compiledMap[res.fileIndex] = res.compiled
	}

	// Convert map to ordered slice, preserving original file order
	compiled := make([]*CompiledFile, 0, len(compiledMap))
	for i := 0; i < len(files); i++ {
		if cf, ok := compiledMap[i]; ok {
			compiled = append(compiled, cf)
		}
	}

	return compiled, cacheHits, errors
}

// compileFile compiles a single file
func (s *System) compileFile(file string) (*CompiledFile, error) {
	// Read source
	source, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Lex
	lex := lexer.New(string(source))
	tokens, lexErrors := lex.ScanTokens()
	if len(lexErrors) > 0 {
		return nil, fmt.Errorf("lexer error: %s", lexErrors[0].Message)
	}

	// Parse
	p := parser.New(tokens)
	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		return nil, fmt.Errorf("parse error: %s", parseErrors[0].Message)
	}

	// Type check
	tc := typechecker.NewTypeChecker()
	typeErrors := tc.CheckProgram(program)
	if len(typeErrors) > 0 {
		return nil, fmt.Errorf("type error: %v", typeErrors[0])
	}

	// Compute file hash for cache invalidation
	hash, err := computeFileHash(file)
	if err != nil {
		return nil, fmt.Errorf("failed to compute hash: %w", err)
	}

	return &CompiledFile{
		Path:    file,
		Hash:    hash,
		Program: program,
	}, nil
}

// generateGoCode generates Go source code from compiled files
func (s *System) generateGoCode(compiled []*CompiledFile) (map[string]string, error) {
	// Combine all programs
	allResources := make([]*ast.ResourceNode, 0)
	for _, cf := range compiled {
		allResources = append(allResources, cf.Program.Resources...)
	}

	program := &ast.Program{
		Resources: allResources,
	}

	// Derive module name from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	moduleName := filepath.Base(cwd)

	// Generate code (empty conduitPath for now - will be resolved by go mod tidy)
	gen := codegen.NewGenerator()
	files, err := gen.GenerateProgram(program, moduleName, "")
	if err != nil {
		return nil, err
	}

	return files, nil
}

// writeGeneratedFiles writes generated Go files to disk atomically.
// Uses a temporary directory to ensure consistency - either all files are
// written successfully or the old state is preserved.
func (s *System) writeGeneratedFiles(files map[string]string) error {
	generatedDir := filepath.Join(s.options.BuildDir, "generated")
	tmpDir := generatedDir + ".tmp"

	// Clean up any existing temp directory from previous failed builds
	if err := os.RemoveAll(tmpDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean temp directory: %w", err)
	}

	// Create temporary directory
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Write all files to temporary directory
	for filename, content := range files {
		fullPath := filepath.Join(tmpDir, filename)

		// Create subdirectories
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			os.RemoveAll(tmpDir) // Clean up on failure
			return fmt.Errorf("failed to create directory for %s: %w", filename, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			os.RemoveAll(tmpDir) // Clean up on failure
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// All files written successfully - now atomically swap directories
	// Remove old generated directory if it exists
	if err := os.RemoveAll(generatedDir); err != nil && !os.IsNotExist(err) {
		os.RemoveAll(tmpDir) // Clean up on failure
		return fmt.Errorf("failed to remove old generated directory: %w", err)
	}

	// Rename temp directory to final location
	if err := os.Rename(tmpDir, generatedDir); err != nil {
		return fmt.Errorf("failed to move temp directory to final location: %w", err)
	}

	// Copy migration files to root migrations/ directory so conduit migrate can find them
	if err := s.copyMigrationsToRoot(files); err != nil {
		return fmt.Errorf("failed to copy migrations to root: %w", err)
	}

	return nil
}

// copyMigrationsToRoot copies generated migration files to the root migrations/ directory
// so that conduit migrate commands can find them
func (s *System) copyMigrationsToRoot(files map[string]string) error {
	// Ensure root migrations directory exists
	migrationsDir := "migrations"
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create root migrations directory: %w", err)
	}

	copiedCount := 0
	// Copy each migration file from generated files to root
	for filename, content := range files {
		if strings.HasPrefix(filename, "migrations/") {
			// Extract just the migration filename (e.g., "001_init.sql")
			migrationName := filepath.Base(filename)
			destPath := filepath.Join(migrationsDir, migrationName)

			// Write migration file to root migrations/ directory
			if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write migration %s: %w", migrationName, err)
			}

			copiedCount++
			if s.options.Verbose {
				fmt.Printf("Copied migration to %s\n", destPath)
			}
		}
	}

	// Always report if migrations were copied (not just in verbose mode)
	if copiedCount > 0 && !s.options.Verbose {
		fmt.Printf("✓ Copied %d migration(s) to %s/\n", copiedCount, migrationsDir)
	}

	return nil
}

// compileAssets compiles CSS, JS, and copies static assets
func (s *System) compileAssets() error {
	// This will be implemented when we add frontend support
	return nil
}

// buildBinary builds the final Go binary
func (s *System) buildBinary() (string, error) {
	generatedDir := filepath.Join(s.options.BuildDir, "generated")
	outputPath := s.options.OutputPath

	// Resolve to absolute path to prevent command injection
	absGenDir, err := filepath.Abs(generatedDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve generated directory: %w", err)
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve output path: %w", err)
	}

	// Build command
	args := []string{"build", "-o", absOutputPath}

	// Add build flags based on mode
	if s.options.Mode == ModeProduction {
		// Production optimizations
		args = append(args, "-ldflags", "-s -w") // Strip debug symbols
	}

	args = append(args, absGenDir)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build failed: %w", err)
	}

	return absOutputPath, nil
}

// generateMetadata generates the introspection metadata file
func (s *System) generateMetadata(compiled []*CompiledFile) (string, error) {
	// This will be implemented when we add introspection
	metadataPath := s.options.OutputPath + ".meta.json"

	// For now, create an empty metadata file
	if err := os.WriteFile(metadataPath, []byte("{}"), 0644); err != nil {
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}

	return metadataPath, nil
}

// CompiledFile represents a compiled source file
type CompiledFile struct {
	Path    string
	Hash    string
	Program *ast.Program
}

// handleMigrations manages schema-based migration generation
func (s *System) handleMigrations(compiled []*CompiledFile) error {
	// Extract current schemas from compiled AST
	extractor := NewSchemaExtractor()
	currentSchemas, err := extractor.ExtractSchemas(compiled)
	if err != nil {
		return fmt.Errorf("failed to extract schemas: %w", err)
	}

	// If no resources, nothing to do
	if len(currentSchemas) == 0 {
		return nil
	}

	// Load previous schema snapshot
	snapshotManager := NewSnapshotManager(s.options.BuildDir)
	previousSchemas, err := snapshotManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load schema snapshot: %w", err)
	}

	// Initialize migration builder
	migrationBuilder := NewMigrationBuilder()
	migrationsDir := "migrations"

	if previousSchemas == nil {
		// First build: check if we should generate initial migration
		shouldGenerate, err := migrationBuilder.ShouldGenerateInitialMigration(migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to check initial migration status: %w", err)
		}

		if shouldGenerate {
			// Generate initial 001_init.sql
			initialSQL, err := migrationBuilder.GenerateInitialMigration(currentSchemas)
			if err != nil {
				return fmt.Errorf("failed to generate initial migration: %w", err)
			}

			// Write to migrations/001_init.sql
			if err := os.MkdirAll(migrationsDir, 0755); err != nil {
				return fmt.Errorf("failed to create migrations directory: %w", err)
			}

			initPath := filepath.Join(migrationsDir, "001_init.sql")
			if err := os.WriteFile(initPath, []byte(initialSQL), 0644); err != nil {
				return fmt.Errorf("failed to write initial migration: %w", err)
			}

			if s.options.Verbose {
				fmt.Printf("Generated initial migration: %s\n", initPath)
			} else {
				fmt.Printf("✓ Generated initial migration: 001_init.sql\n")
			}
		}
	} else {
		// Subsequent build: generate versioned migration if schema changed
		result, err := migrationBuilder.GenerateVersionedMigration(
			previousSchemas,
			currentSchemas,
			migrationsDir,
		)
		if err != nil {
			// DO NOT save snapshot if migration generation failed
			return fmt.Errorf("failed to generate versioned migration: %w", err)
		}

		if result.MigrationGenerated {
			// Report the generated migration
			if s.options.Verbose {
				fmt.Printf("Generated migration: %s\n", result.MigrationPath)
				if result.Breaking {
					fmt.Printf("  WARNING: Contains breaking changes\n")
				}
				if result.DataLoss {
					fmt.Printf("  WARNING: May cause data loss\n")
				}
			} else {
				warnings := ""
				if result.Breaking {
					warnings += " [BREAKING]"
				}
				if result.DataLoss {
					warnings += " [DATA LOSS]"
				}
				fmt.Printf("✓ Generated migration: %s%s\n", filepath.Base(result.MigrationPath), warnings)
			}
		} else if s.options.Verbose {
			fmt.Printf("No schema changes detected, skipping migration generation\n")
		}
	}

	// Validate schemas before saving
	for name, s := range currentSchemas {
		if s.Name == "" {
			return fmt.Errorf("invalid schema: missing name")
		}
		if name != s.Name {
			return fmt.Errorf("schema name mismatch: key=%s, schema.Name=%s", name, s.Name)
		}
	}

	// Save current schema snapshot for next build
	// Only save after successful migration generation (or no migration needed)
	timestamp := time.Now().Unix()
	if err := snapshotManager.Save(currentSchemas, timestamp); err != nil {
		return fmt.Errorf("failed to save schema snapshot: %w", err)
	}

	return nil
}
