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
  5. Go compilation - build native binary

Examples:
  conduit build
  conduit build --verbose
  conduit build --json
  conduit build --output dist/myapp`,
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

	// Find all .cdt files
	cdtFiles, err := filepath.Glob("app/*.cdt")
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

	gen := codegen.NewGenerator()
	files, err := gen.GenerateProgram(program)
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

	// Build Go binary
	if buildVerbose {
		infoColor.Println("Building binary...")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	buildCmd := exec.Command("go", "build", "-o", outputPath, "./"+generatedDir)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Println()
	successColor.Printf("✓ Build successful in %.2fs\n", elapsed.Seconds())
	infoColor.Printf("  Binary: %s\n", outputPath)

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
