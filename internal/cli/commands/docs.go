package commands

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/docs"
	"github.com/conduit-lang/conduit/internal/utils"
	"github.com/conduit-lang/conduit/internal/watch"
)

var (
	docsFormat     string
	docsOutput     string
	docsBaseURL    string
	docsWatch      bool
	docsServePort  int
	docsProjectName string
	docsProjectDesc string
	docsVersion    string
)

// NewDocsCommand creates the docs command
func NewDocsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate and serve API documentation",
		Long: `Generate comprehensive API documentation from Conduit source code.

Supports multiple output formats:
  - openapi: OpenAPI 3.0 specification (JSON)
  - markdown: Markdown documentation files
  - html: Interactive HTML documentation site

Examples:
  conduit docs generate
  conduit docs generate --format=openapi
  conduit docs generate --format=markdown,html
  conduit docs serve
  conduit docs serve --port=8080
  conduit docs generate --watch`,
	}

	cmd.AddCommand(NewDocsGenerateCommand())
	cmd.AddCommand(NewDocsServeCommand())

	return cmd
}

// NewDocsGenerateCommand creates the docs generate subcommand
func NewDocsGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate API documentation",
		Long: `Generate API documentation in one or more formats.

The generate command parses all .cdt files in your project and produces
comprehensive API documentation including:
  - Resource definitions and fields
  - REST API endpoints
  - Request/response schemas
  - Code examples
  - Validation rules and constraints

Output formats:
  - openapi: OpenAPI 3.0 JSON specification
  - markdown: Markdown documentation files
  - html: Interactive HTML documentation site`,
		RunE: runDocsGenerate,
	}

	cmd.Flags().StringVar(&docsFormat, "format", "html", "Output format(s): openapi, markdown, html (comma-separated)")
	cmd.Flags().StringVarP(&docsOutput, "output", "o", "docs", "Output directory")
	cmd.Flags().StringVar(&docsBaseURL, "base-url", "", "Base URL for the API")
	cmd.Flags().BoolVarP(&docsWatch, "watch", "w", false, "Watch for changes and regenerate")
	cmd.Flags().StringVar(&docsProjectName, "name", "", "Project name (defaults to directory name)")
	cmd.Flags().StringVar(&docsProjectDesc, "description", "", "Project description")
	cmd.Flags().StringVar(&docsVersion, "version", "1.0.0", "API version")

	return cmd
}

// NewDocsServeCommand creates the docs serve subcommand
func NewDocsServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve documentation locally",
		Long: `Start a local HTTP server to view the generated HTML documentation.

The serve command starts a development server on localhost and automatically
opens your browser to view the documentation. The server supports hot reload
when combined with the --watch flag.

Examples:
  conduit docs serve
  conduit docs serve --port=8080
  conduit docs serve --watch`,
		RunE: runDocsServe,
	}

	cmd.Flags().IntVarP(&docsServePort, "port", "p", 8000, "Port to serve on")
	cmd.Flags().StringVarP(&docsOutput, "output", "o", "docs", "Documentation directory")
	cmd.Flags().BoolVarP(&docsWatch, "watch", "w", false, "Watch for changes and regenerate")

	return cmd
}

func runDocsGenerate(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	successColor := color.New(color.FgGreen, color.Bold)
	errorColor := color.New(color.FgRed, color.Bold)
	infoColor := color.New(color.FgCyan)

	infoColor.Println("Generating documentation...")

	// Parse formats
	formats := parseFormats(docsFormat)

	// Determine project name
	projectName := docsProjectName
	if projectName == "" {
		cwd, _ := os.Getwd()
		projectName = filepath.Base(cwd)
	}

	// Parse Conduit source files
	program, err := parseConduitSources()
	if err != nil {
		errorColor.Printf("Error: %v\n", err)
		return err
	}

	// Create documentation generator
	config := &docs.Config{
		ProjectName:        projectName,
		ProjectVersion:     docsVersion,
		ProjectDescription: docsProjectDesc,
		OutputDir:          docsOutput,
		Formats:            formats,
		BaseURL:            docsBaseURL,
	}

	generator, err := docs.NewGenerator(config)
	if err != nil {
		errorColor.Printf("Error: %v\n", err)
		return err
	}

	// Generate documentation
	if err := generator.Generate(program); err != nil {
		errorColor.Printf("Error: %v\n", err)
		return err
	}

	elapsed := time.Since(startTime)
	successColor.Printf("✓ Documentation generated in %v\n", elapsed)
	infoColor.Printf("Output: %s\n", docsOutput)

	// Watch mode
	if docsWatch {
		infoColor.Println("Watching for changes...")
		return watchAndRegenerate(program, config)
	}

	return nil
}

func runDocsServe(cmd *cobra.Command, args []string) error {
	infoColor := color.New(color.FgCyan)
	errorColor := color.New(color.FgRed, color.Bold)
	successColor := color.New(color.FgGreen, color.Bold)

	// Check if HTML docs exist
	htmlDir := filepath.Join(docsOutput, "html")
	if _, err := os.Stat(htmlDir); os.IsNotExist(err) {
		errorColor.Println("Error: HTML documentation not found")
		infoColor.Println("Run 'conduit docs generate --format=html' first")
		return fmt.Errorf("HTML documentation not found")
	}

	// Start HTTP server
	fs := http.FileServer(http.Dir(htmlDir))
	http.Handle("/", fs)

	addr := fmt.Sprintf("localhost:%d", docsServePort)
	url := fmt.Sprintf("http://%s", addr)

	successColor.Printf("✓ Documentation server running at %s\n", url)
	infoColor.Println("Press Ctrl+C to stop")

	// Watch mode
	if docsWatch {
		go func() {
			infoColor.Println("Watching for changes...")
			// Parse and regenerate on changes
			program, err := parseConduitSources()
			if err != nil {
				return
			}

			// Determine project name
			projectName := docsProjectName
			if projectName == "" {
				cwd, _ := os.Getwd()
				projectName = filepath.Base(cwd)
			}

			config := &docs.Config{
				ProjectName:        projectName,
				ProjectVersion:     docsVersion,
				ProjectDescription: docsProjectDesc,
				OutputDir:          docsOutput,
				Formats:            []docs.Format{docs.FormatHTML},
				BaseURL:            docsBaseURL,
			}

			watchAndRegenerate(program, config)
		}()
	}

	return http.ListenAndServe(addr, nil)
}

// Helper functions

func parseFormats(formatStr string) []docs.Format {
	formats := make([]docs.Format, 0)
	parts := splitAndTrim(formatStr, ",")

	for _, part := range parts {
		switch part {
		case "openapi":
			formats = append(formats, docs.FormatOpenAPI)
		case "markdown":
			formats = append(formats, docs.FormatMarkdown)
		case "html":
			formats = append(formats, docs.FormatHTML)
		}
	}

	if len(formats) == 0 {
		formats = append(formats, docs.FormatHTML)
	}

	return formats
}

func parseConduitSources() (*ast.Program, error) {
	const maxResources = 1000 // reasonable limit

	// Find all .cdt files (recursively)
	files, err := utils.FindCdtFiles("app")
	if err != nil {
		return nil, fmt.Errorf("failed to find .cdt files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .cdt files found in app/ directory")
	}

	// Parse all files
	program := &ast.Program{
		Resources: make([]*ast.ResourceNode, 0),
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Lex and parse
		l := lexer.New(string(content))
		tokens, lexErrors := l.ScanTokens()
		if len(lexErrors) > 0 {
			return nil, fmt.Errorf("failed to lex %s: %d errors", file, len(lexErrors))
		}

		p := parser.New(tokens)
		fileProgram, parseErrors := p.Parse()
		if len(parseErrors) > 0 {
			return nil, fmt.Errorf("failed to parse %s: %d errors", file, len(parseErrors))
		}

		// Check resource limit before merging
		if len(program.Resources)+len(fileProgram.Resources) > maxResources {
			return nil, fmt.Errorf("resource limit exceeded: maximum %d resources allowed", maxResources)
		}

		// Merge resources
		program.Resources = append(program.Resources, fileProgram.Resources...)
	}

	return program, nil
}

func watchAndRegenerate(program *ast.Program, config *docs.Config) error {
	infoColor := color.New(color.FgCyan)
	successColor := color.New(color.FgGreen, color.Bold)
	errorColor := color.New(color.FgRed, color.Bold)

	// Create file watcher
	watcher, err := watch.NewFileWatcher(
		[]string{"*.cdt"},
		[]string{"build", "docs", ".git"},
		func(files []string) error {
			infoColor.Printf("Change detected: %d files\n", len(files))

			// Reparse sources
			newProgram, err := parseConduitSources()
			if err != nil {
				errorColor.Printf("Parse error: %v\n", err)
				return err
			}

			// Regenerate documentation
			generator, err := docs.NewGenerator(config)
			if err != nil {
				errorColor.Printf("Generator creation error: %v\n", err)
				return err
			}
			if err := generator.Generate(newProgram); err != nil {
				errorColor.Printf("Generation error: %v\n", err)
				return err
			}

			successColor.Println("✓ Documentation regenerated")
			// Note: program variable is not used after this point, so no need to update it
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Stop()

	// Start watching
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Block forever
	select {}
}

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimString(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	result := make([]string, 0)
	current := ""

	for _, char := range s {
		if string(char) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing whitespace
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
