package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/compiler/errors"
	"github.com/conduit-lang/conduit/internal/cli/config"
	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/codegen"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/compiler/typechecker"
	"github.com/conduit-lang/conduit/internal/utils"
)

var (
	buildJSON    bool
	buildVerbose bool
	buildOutput  string
)

// NewBuildCommand creates the build command
func NewBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Compile Conduit source to Go and build binary",
		Long: `Compile all .cdt files in the app/ directory and generate a native executable.

The build process:
  1. Lexical analysis - tokenize .cdt files
  2. Parsing - generate AST
  3. Type checking - verify type safety
  4. Code generation - produce Go source
  5. Go compilation - build native binary`,
		Example: `  # Build with default settings
  conduit build

  # Build with verbose output to see each compilation step
  conduit build --verbose

  # Build and output errors in JSON format (useful for tooling)
  conduit build --json

  # Build to a custom output location
  conduit build --output dist/myapp

  # Build with verbose output and custom location
  conduit build -v -o bin/production`,
		RunE: runBuild,
	}

	cmd.Flags().BoolVar(&buildJSON, "json", false, "Output errors in JSON format")
	cmd.Flags().BoolVarP(&buildVerbose, "verbose", "v", false, "Show detailed build output")
	cmd.Flags().StringVarP(&buildOutput, "output", "o", "", "Output binary path (default: build/app)")

	return cmd
}

func runBuild(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	successColor := color.New(color.FgGreen, color.Bold)
	errorColor := color.New(color.FgRed, color.Bold)
	infoColor := color.New(color.FgCyan)
	warningColor := color.New(color.FgYellow)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		if buildVerbose {
			warningColor.Printf("Warning: %v\n", err)
		}
	}

	// Determine output path
	outputPath := buildOutput
	if outputPath == "" {
		if cfg != nil && cfg.Build.Output != "" {
			outputPath = cfg.Build.Output
		} else {
			outputPath = "build/app"
		}
	}

	// Determine generated directory
	generatedDir := "build/generated"
	if cfg != nil && cfg.Build.GeneratedDir != "" {
		generatedDir = cfg.Build.GeneratedDir
	}

	// Check if app directory exists
	if _, err := os.Stat("app"); os.IsNotExist(err) {
		return fmt.Errorf("app/ directory not found - are you in a Conduit project?")
	}

	// Find all .cdt files (recursively)
	cdtFiles, err := utils.FindCdtFiles("app")
	if err != nil {
		return fmt.Errorf("failed to find .cdt files: %w", err)
	}

	if len(cdtFiles) == 0 {
		return fmt.Errorf("no .cdt files found in app/ directory")
	}

	if buildVerbose {
		infoColor.Printf("Found %d .cdt file(s)\n", len(cdtFiles))
	}

	// Combine all resources from all files
	allResources := make([]*ast.ResourceNode, 0)
	var allErrors []errors.CompilerError

	for _, file := range cdtFiles {
		if buildVerbose {
			infoColor.Printf("Compiling %s...\n", file)
		}

		// Read source
		source, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
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
			continue
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
			continue
		}

		// Add resources to combined list
		allResources = append(allResources, program.Resources...)
	}

	// Stop if there were errors
	if len(allErrors) > 0 {
		if buildJSON {
			outputErrorsJSON(allErrors)
		} else {
			outputErrorsTerminal(allErrors, errorColor)
		}
		return fmt.Errorf("compilation failed")
	}

	// Create combined program
	program := &ast.Program{
		Resources: allResources,
	}

	// Type check
	tc := typechecker.NewTypeChecker()
	typeErrors := tc.CheckProgram(program)

	if len(typeErrors) > 0 {
		// Convert type checker errors to compiler errors
		for _, typeErr := range typeErrors {
			allErrors = append(allErrors, errors.CompilerError{
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

		if buildJSON {
			outputErrorsJSON(allErrors)
		} else {
			outputErrorsTerminal(allErrors, errorColor)
		}
		return fmt.Errorf("type checking failed")
	}

	// Generate Go code
	if buildVerbose {
		infoColor.Println("Generating Go code...")
	}

	// Derive module name from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	moduleName := filepath.Base(cwd)

	// Find conduit source path
	// Priority: 1. CONDUIT_ROOT env var, 2. Traverse from executable, 3. Error
	conduitPath := os.Getenv("CONDUIT_ROOT")

	if conduitPath == "" {
		// Try to find from executable location
		conduitExec, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		// Resolve symlinks (in case conduit is a symlink)
		if resolvedExec, err := filepath.EvalSymlinks(conduitExec); err == nil {
			conduitExec = resolvedExec
		}

		conduitDir := filepath.Dir(conduitExec)

		// Try to find conduit source root by looking for go.mod + pkg/runtime/stdlib.go
		checkDir := conduitDir
		for i := 0; i < 10; i++ { // Check up to 10 levels up
			goModPath := filepath.Join(checkDir, "go.mod")
			stdlibPath := filepath.Join(checkDir, "pkg", "runtime", "stdlib.go")

			if _, err := os.Stat(goModPath); err == nil {
				if _, err := os.Stat(stdlibPath); err == nil {
					// Found it! This is the conduit source directory
					conduitPath, _ = filepath.Abs(checkDir)
					if buildVerbose {
						infoColor.Printf("Found conduit source at: %s\n", conduitPath)
					}
					break
				}
			}

			parent := filepath.Dir(checkDir)
			if parent == checkDir {
				break // Reached root
			}
			checkDir = parent
		}
	} else {
		if buildVerbose {
			infoColor.Printf("Using CONDUIT_ROOT: %s\n", conduitPath)
		}
	}

	// If we still haven't found it, error out with helpful message
	if conduitPath == "" {
		return fmt.Errorf(`could not locate conduit source directory

Please set the CONDUIT_ROOT environment variable to point to your conduit source:
  export CONDUIT_ROOT=/path/to/conduit

Or run conduit build from the conduit source directory.`)
	}

	// Verify the path actually contains conduit runtime
	stdlibCheck := filepath.Join(conduitPath, "pkg", "runtime", "stdlib.go")
	if _, err := os.Stat(stdlibCheck); err != nil {
		return fmt.Errorf("CONDUIT_ROOT (%s) does not contain pkg/runtime/stdlib.go - is this the correct path?", conduitPath)
	}

	// Get API prefix from config (default to empty string if not set)
	apiPrefix := ""
	if cfg != nil {
		apiPrefix = cfg.Server.APIPrefix
	}

	gen := codegen.NewGenerator()
	files, err := gen.GenerateProgram(program, moduleName, conduitPath, apiPrefix)
	if err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	// Create build/generated directory
	if err := os.MkdirAll(generatedDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Write generated files
	for filename, content := range files {
		fullPath := filepath.Join(generatedDir, filename)

		// Create subdirectories if needed
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", filename, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}

		if buildVerbose {
			infoColor.Printf("  Generated %s\n", filename)
		}
	}

	// Copy migration files to root migrations/ directory so conduit migrate can find them
	migrationsDir := "migrations"
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create root migrations directory: %w", err)
	}

	migrationCount := 0
	for filename, content := range files {
		if strings.HasPrefix(filename, "migrations/") {
			migrationName := filepath.Base(filename)
			destPath := filepath.Join(migrationsDir, migrationName)

			if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write migration %s: %w", migrationName, err)
			}

			migrationCount++
			if buildVerbose {
				infoColor.Printf("  Copied migration to %s\n", destPath)
			}
		}
	}

	if migrationCount > 0 && !buildVerbose {
		infoColor.Printf("  Copied %d migration(s) to migrations/\n", migrationCount)
	}

	// Download Go dependencies and create go.sum
	if buildVerbose {
		infoColor.Println("Downloading dependencies...")
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = generatedDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr

	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	// Build Go binary
	if buildVerbose {
		infoColor.Println("Building binary...")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Convert output path to absolute path
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute output path: %w", err)
	}

	// Run go build from the generated directory
	buildCmd := exec.Command("go", "build", "-o", absOutputPath)
	buildCmd.Dir = generatedDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Println()
	successColor.Printf("✓ Build successful in %.2fs\n", elapsed.Seconds())
	infoColor.Printf("  Binary: %s\n", outputPath)
	if apiPrefix != "" {
		infoColor.Printf("  API Prefix: %s\n", apiPrefix)
	}

	return nil
}

func outputErrorsJSON(errs []errors.CompilerError) {
	output := struct {
		Success bool                   `json:"success"`
		Errors  []errors.CompilerError `json:"errors"`
	}{
		Success: false,
		Errors:  errs,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
}

func outputErrorsTerminal(errs []errors.CompilerError, errorColor *color.Color) {
	errorColor.Fprintf(os.Stderr, "\nCompilation failed with %d error(s):\n\n", len(errs))

	for i, err := range errs {
		fmt.Fprintf(os.Stderr, "%d. [%s] %s:%d:%d\n",
			i+1, err.Phase, err.Location.File, err.Location.Line, err.Location.Column)
		fmt.Fprintf(os.Stderr, "   %s\n", err.Message)

		if i < len(errs)-1 {
			fmt.Fprintln(os.Stderr, strings.Repeat("-", 60))
		}
	}
	fmt.Fprintln(os.Stderr)
}
