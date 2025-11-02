package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/conduit-lang/conduit/compiler/errors"
	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/codegen"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/compiler/typechecker"
	"github.com/conduit-lang/conduit/internal/tooling/build"
	"github.com/conduit-lang/conduit/internal/utils"
)

// IncrementalCompiler handles incremental compilation of changed files
type IncrementalCompiler struct {
	// Cache of parsed resources by file
	resourceCache map[string][]*ast.ResourceNode

	// Last successful compile time
	lastCompile time.Time
}

// NewIncrementalCompiler creates a new incremental compiler
func NewIncrementalCompiler() *IncrementalCompiler {
	return &IncrementalCompiler{
		resourceCache: make(map[string][]*ast.ResourceNode),
	}
}

// CompileResult holds the result of a compilation
type CompileResult struct {
	Success       bool
	Errors        []errors.CompilerError
	Duration      time.Duration
	ChangedFiles  []string
	GeneratedFile map[string]string
}

// IncrementalBuild compiles only changed files
func (ic *IncrementalCompiler) IncrementalBuild(changedFiles []string) (*CompileResult, error) {
	start := time.Now()

	result := &CompileResult{
		Success:       false,
		Errors:        make([]errors.CompilerError, 0),
		ChangedFiles:  changedFiles,
		GeneratedFile: make(map[string]string),
	}

	// Filter for .cdt files only
	cdtFiles := make([]string, 0)
	for _, file := range changedFiles {
		if filepath.Ext(file) == ".cdt" {
			cdtFiles = append(cdtFiles, file)
		}
	}

	if len(cdtFiles) == 0 {
		// No .cdt files changed, return success
		result.Success = true
		result.Duration = time.Since(start)
		return result, nil
	}

	// Compile changed files
	newResources := make(map[string][]*ast.ResourceNode)

	for _, file := range cdtFiles {
		resources, errs := ic.compileFile(file)
		if len(errs) > 0 {
			result.Errors = append(result.Errors, errs...)
			continue
		}
		newResources[file] = resources
	}

	// If there were errors, return early
	if len(result.Errors) > 0 {
		result.Duration = time.Since(start)
		return result, fmt.Errorf("compilation failed with %d error(s)", len(result.Errors))
	}

	// Update cache with new resources
	for file, resources := range newResources {
		ic.resourceCache[file] = resources
	}

	// Gather all resources (changed + cached)
	allResources := make([]*ast.ResourceNode, 0)
	for _, resources := range ic.resourceCache {
		allResources = append(allResources, resources...)
	}

	// Type check all resources
	program := &ast.Program{
		Resources: allResources,
	}

	tc := typechecker.NewTypeChecker()
	typeErrors := tc.CheckProgram(program)

	if len(typeErrors) > 0 {
		for _, typeErr := range typeErrors {
			result.Errors = append(result.Errors, errors.CompilerError{
				Phase:    "type_checker",
				Code:     "TYPE001",
				Message:  typeErr.Error(),
				Severity: errors.Error,
				Location: errors.SourceLocation{
					File:   "<source>",
					Line:   0,
					Column: 0,
				},
			})
		}
		result.Duration = time.Since(start)
		return result, fmt.Errorf("type checking failed")
	}

	// Generate Go code
	// Derive module name from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return result, fmt.Errorf("failed to get current directory: %w", err)
	}
	moduleName := filepath.Base(cwd)

	gen := codegen.NewGenerator()
	files, err := gen.GenerateProgram(program, moduleName, "")
	if err != nil {
		result.Errors = append(result.Errors, errors.CompilerError{
			Phase:    "codegen",
			Code:     "CODEGEN001",
			Message:  err.Error(),
			Severity: errors.Error,
			Location: errors.SourceLocation{},
		})
		result.Duration = time.Since(start)
		return result, fmt.Errorf("code generation failed: %w", err)
	}

	// Write generated files
	generatedDir := "build/generated"
	if err := os.MkdirAll(generatedDir, 0755); err != nil {
		return result, fmt.Errorf("failed to create build directory: %w", err)
	}

	for filename, content := range files {
		fullPath := filepath.Join(generatedDir, filename)

		// Create subdirectories if needed
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return result, fmt.Errorf("failed to create directory for %s: %w", filename, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return result, fmt.Errorf("failed to write %s: %w", filename, err)
		}

		result.GeneratedFile[filename] = fullPath
	}

	result.Success = true
	result.Duration = time.Since(start)
	ic.lastCompile = time.Now()

	return result, nil
}

// compileFile compiles a single .cdt file and returns its resources
func (ic *IncrementalCompiler) compileFile(file string) ([]*ast.ResourceNode, []errors.CompilerError) {
	var allErrors []errors.CompilerError

	// Read source
	source, err := os.ReadFile(file)
	if err != nil {
		return nil, []errors.CompilerError{{
			Phase:    "io",
			Code:     "IO001",
			Message:  fmt.Sprintf("failed to read file: %v", err),
			Severity: errors.Error,
			Location: errors.SourceLocation{
				File: file,
			},
		}}
	}

	// Lex
	lex := lexer.New(string(source))
	tokens, lexErrors := lex.ScanTokens()

	if len(lexErrors) > 0 {
		for _, lexErr := range lexErrors {
			allErrors = append(allErrors, errors.CompilerError{
				Phase:    "lexer",
				Code:     "LEX001",
				Message:  lexErr.Message,
				Severity: errors.Error,
				Location: errors.SourceLocation{
					File:   file,
					Line:   lexErr.Line,
					Column: lexErr.Column,
				},
			})
		}
		return nil, allErrors
	}

	// Parse
	p := parser.New(tokens)
	program, parseErrors := p.Parse()

	if len(parseErrors) > 0 {
		for _, parseErr := range parseErrors {
			allErrors = append(allErrors, errors.CompilerError{
				Phase:    "parser",
				Code:     "PARSE001",
				Message:  parseErr.Message,
				Severity: errors.Error,
				Location: errors.SourceLocation{
					File:   file,
					Line:   parseErr.Token.Line,
					Column: parseErr.Token.Column,
				},
			})
		}
		return nil, allErrors
	}

	return program.Resources, nil
}

// FullBuild performs a full rebuild of all .cdt files
func (ic *IncrementalCompiler) FullBuild() (*CompileResult, error) {
	// Clear cache
	ic.resourceCache = make(map[string][]*ast.ResourceNode)

	// Find all .cdt files
	cdtFiles, err := utils.FindCdtFiles("app")
	if err != nil {
		return nil, fmt.Errorf("failed to find .cdt files: %w", err)
	}

	if len(cdtFiles) == 0 {
		return nil, fmt.Errorf("no .cdt files found in app/ directory")
	}

	// Build all files
	return ic.IncrementalBuild(cdtFiles)
}

// ClearCache clears the resource cache
func (ic *IncrementalCompiler) ClearCache() {
	ic.resourceCache = make(map[string][]*ast.ResourceNode)
}

// HandleMigrations checks for schema changes and generates migrations if needed
func (ic *IncrementalCompiler) HandleMigrations() error {
	// Gather all resources from cache
	allResources := make([]*ast.ResourceNode, 0)
	for _, resources := range ic.resourceCache {
		allResources = append(allResources, resources...)
	}

	if len(allResources) == 0 {
		return nil
	}

	// Create a minimal compiled file structure for migration handling
	program := &ast.Program{
		Resources: allResources,
	}

	// Extract current schemas
	extractor := build.NewSchemaExtractor()
	compiled := []*build.CompiledFile{
		{
			Program: program,
		},
	}
	currentSchemas, err := extractor.ExtractSchemas(compiled)
	if err != nil {
		return fmt.Errorf("failed to extract schemas: %w", err)
	}

	if len(currentSchemas) == 0 {
		return nil
	}

	// Load previous schema snapshot
	snapshotManager := build.NewSnapshotManager("build")
	previousSchemas, err := snapshotManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load schema snapshot: %w", err)
	}

	// Initialize migration builder
	migrationBuilder := build.NewMigrationBuilder()
	migrationsDir := "migrations"

	if previousSchemas == nil {
		// First build: check if we should generate initial migration
		shouldGenerate, err := migrationBuilder.ShouldGenerateInitialMigration(migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to check initial migration status: %w", err)
		}

		if shouldGenerate {
			// Generate initial migration
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
		}
	} else {
		// Generate versioned migration if schema changed
		result, err := migrationBuilder.GenerateVersionedMigration(
			previousSchemas,
			currentSchemas,
			migrationsDir,
		)
		if err != nil {
			return fmt.Errorf("failed to generate versioned migration: %w", err)
		}

		// Result is available but we don't need to report it here
		// The caller will check for pending migrations
		_ = result
	}

	// Save current schema snapshot for next build
	timestamp := time.Now().Unix()
	if err := snapshotManager.Save(currentSchemas, timestamp); err != nil {
		return fmt.Errorf("failed to save schema snapshot: %w", err)
	}

	return nil
}
