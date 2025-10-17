package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/compiler/errors"
	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/codegen"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/compiler/typechecker"
)

var (
	buildJSON    bool
	buildVerbose bool
)

func init() {
	buildCmd.Flags().BoolVar(&buildJSON, "json", false, "Output errors in JSON format")
	buildCmd.Flags().BoolVar(&buildVerbose, "verbose", false, "Show detailed build output")
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile Conduit source to Go and build binary",
	Long:  "Compile all .cdt files in the app/ directory and generate a native executable",
	RunE: func(cmd *cobra.Command, args []string) error {
		startTime := time.Now()

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
			fmt.Printf("Found %d .cdt file(s)\n", len(cdtFiles))
		}

		// Combine all resources from all files
		allResources := make([]*ast.ResourceNode, 0)
		var allErrors []errors.CompilerError

		for _, file := range cdtFiles {
			if buildVerbose {
				fmt.Printf("Compiling %s...\n", file)
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
				outputErrorsTerminal(allErrors)
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
				outputErrorsTerminal(allErrors)
			}
			return fmt.Errorf("type checking failed")
		}

		// Generate Go code
		if buildVerbose {
			fmt.Println("Generating Go code...")
		}

		gen := codegen.NewGenerator()
		files, err := gen.GenerateProgram(program)
		if err != nil {
			return fmt.Errorf("code generation failed: %w", err)
		}

		// Create build/generated directory
		generatedDir := "build/generated"
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
				fmt.Printf("  Generated %s\n", filename)
			}
		}

		// Build Go binary
		if buildVerbose {
			fmt.Println("Building binary...")
		}

		buildCmd := exec.Command("go", "build", "-o", "build/app", "./build/generated")
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("go build failed: %w", err)
		}

		elapsed := time.Since(startTime)
		fmt.Printf("\n✓ Build successful in %.2fs\n", elapsed.Seconds())
		fmt.Println("  Binary: build/app")

		return nil
	},
}

func outputErrorsJSON(errs []errors.CompilerError) {
	output := struct {
		Success bool                    `json:"success"`
		Errors  []errors.CompilerError  `json:"errors"`
	}{
		Success: false,
		Errors:  errs,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
}

func outputErrorsTerminal(errs []errors.CompilerError) {
	fmt.Fprintf(os.Stderr, "\nCompilation failed with %d error(s):\n\n", len(errs))

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
