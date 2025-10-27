package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Global flags for introspect commands
	outputFormat string
	verbose      bool
	noColor      bool
)

// NewIntrospectCommand creates the introspect command group
func NewIntrospectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "introspect",
		Short: "Introspect the runtime system",
		Long: `Introspect the Conduit runtime system.

The introspect command provides access to the runtime registry, allowing you
to explore resources, routes, patterns, and dependencies in your application.

This is useful for:
  • Understanding the structure of your application
  • Debugging resource relationships
  • Discovering common patterns
  • Generating documentation
  • Building tooling and integrations

The introspection system reads metadata from your compiled binary to provide
accurate, up-to-date information about your application's structure.`,
		Example: `  # List all resources in the application
  conduit introspect resources

  # View detailed information about a specific resource
  conduit introspect resource Post

  # List all HTTP routes
  conduit introspect routes

  # Show dependencies of a resource
  conduit introspect deps Post

  # Discover common patterns
  conduit introspect patterns

  # Output in JSON format for tooling
  conduit introspect resources --format json

  # Verbose output with all details
  conduit introspect resource Post --verbose`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Disable color output if requested
			if noColor {
				color.NoColor = true
			}
		},
	}

	// Add global flags
	cmd.PersistentFlags().StringVar(&outputFormat, "format", "table", "Output format: json or table")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show all details")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Add subcommands (placeholders for now - will be implemented in future tickets)
	cmd.AddCommand(newIntrospectResourcesCommand())
	cmd.AddCommand(newIntrospectResourceCommand())
	cmd.AddCommand(newIntrospectRoutesCommand())
	cmd.AddCommand(newIntrospectDepsCommand())
	cmd.AddCommand(newIntrospectPatternsCommand())

	return cmd
}

// newIntrospectResourcesCommand creates the 'introspect resources' command
func newIntrospectResourcesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "resources",
		Short: "List all resources in the application",
		Long: `List all resources in the application.

Shows a summary of all resources including their fields, relationships, and hooks.
Use the 'introspect resource <name>' command to view detailed information about
a specific resource.`,
		Example: `  # List all resources
  conduit introspect resources

  # List resources in JSON format
  conduit introspect resources --format json

  # Show verbose output with all details
  conduit introspect resources --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented - requires runtime registry")
		},
	}
}

// newIntrospectResourceCommand creates the 'introspect resource' command
func newIntrospectResourceCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "resource <name>",
		Short: "Show detailed information about a specific resource",
		Long: `Show detailed information about a specific resource.

Displays all fields, relationships, hooks, constraints, and middleware
associated with the resource.`,
		Example: `  # View details of the Post resource
  conduit introspect resource Post

  # View details in JSON format
  conduit introspect resource Post --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented - requires runtime registry")
		},
	}
}

// newIntrospectRoutesCommand creates the 'introspect routes' command
func newIntrospectRoutesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "List all HTTP routes",
		Long: `List all HTTP routes in the application.

Shows the HTTP method, path, handler, and middleware for each route.`,
		Example: `  # List all routes
  conduit introspect routes

  # Filter by HTTP method
  conduit introspect routes --method GET

  # Filter by middleware
  conduit introspect routes --middleware auth

  # Output in JSON format
  conduit introspect routes --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented - requires runtime registry")
		},
	}

	// Add command-specific flags
	cmd.Flags().String("method", "", "Filter by HTTP method (GET, POST, PUT, DELETE)")
	cmd.Flags().String("middleware", "", "Filter by middleware name")

	return cmd
}

// newIntrospectDepsCommand creates the 'introspect deps' command
func newIntrospectDepsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps <resource>",
		Short: "Show dependencies of a resource",
		Long: `Show dependencies of a resource.

Displays both direct dependencies (what the resource uses) and reverse
dependencies (what uses the resource). This includes relationships to other
resources, middleware, and routes.`,
		Example: `  # Show dependencies of Post resource
  conduit introspect deps Post

  # Show reverse dependencies only
  conduit introspect deps Post --reverse

  # Traverse deeper dependency tree
  conduit introspect deps Post --depth 2

  # Filter by dependency type
  conduit introspect deps Post --type resource`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented - requires runtime registry")
		},
	}

	// Add command-specific flags
	cmd.Flags().Int("depth", 1, "Traversal depth for dependency tree")
	cmd.Flags().Bool("reverse", false, "Show only reverse dependencies")
	cmd.Flags().String("type", "", "Filter by type: resource, middleware, or function")

	return cmd
}

// newIntrospectPatternsCommand creates the 'introspect patterns' command
func newIntrospectPatternsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patterns [category]",
		Short: "Show discovered patterns",
		Long: `Show discovered patterns in the application.

The pattern discovery system analyzes your codebase to identify common
patterns and conventions. This helps with:
  • Understanding coding standards
  • Maintaining consistency
  • Generating documentation
  • Training LLMs on project-specific patterns`,
		Example: `  # Show all patterns
  conduit introspect patterns

  # Show patterns for a specific category
  conduit introspect patterns authentication

  # Filter by minimum frequency
  conduit introspect patterns --min-frequency 3

  # Output in JSON format
  conduit introspect patterns --format json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented - requires runtime registry")
		},
	}

	// Add command-specific flags
	cmd.Flags().Int("min-frequency", 1, "Minimum number of occurrences for a pattern")

	return cmd
}

// Formatter is an interface for formatting output
type Formatter interface {
	Format(data interface{}) error
}

// TableFormatter formats output as human-readable tables
type TableFormatter struct {
	writer io.Writer
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(w io.Writer) *TableFormatter {
	if w == nil {
		w = os.Stdout
	}
	return &TableFormatter{writer: w}
}

// Format formats data as a table
func (f *TableFormatter) Format(data interface{}) error {
	// This is a simple implementation - can be enhanced with proper table formatting
	fmt.Fprintln(f.writer, formatAsTable(data))
	return nil
}

// formatAsTable converts data to a simple table format
func formatAsTable(data interface{}) string {
	// Handle maps
	if m, ok := data.(map[string]interface{}); ok {
		var lines []string
		// Sort keys for consistent output
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("%-20s %v", k+":", m[k]))
		}
		return strings.Join(lines, "\n")
	}

	// Handle slices
	if s, ok := data.([]interface{}); ok {
		var lines []string
		for i, item := range s {
			lines = append(lines, fmt.Sprintf("%d. %v", i+1, item))
		}
		return strings.Join(lines, "\n")
	}

	// Fallback
	return fmt.Sprintf("%+v", data)
}

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	writer io.Writer
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(w io.Writer) *JSONFormatter {
	if w == nil {
		w = os.Stdout
	}
	return &JSONFormatter{writer: w}
}

// Format formats data as JSON
func (f *JSONFormatter) Format(data interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// GetFormatter returns the appropriate formatter based on the format parameter
func GetFormatter(format string, writer io.Writer) (Formatter, error) {
	if writer == nil {
		writer = os.Stdout
	}
	f := strings.ToLower(format)
	switch f {
	case "json":
		return NewJSONFormatter(writer), nil
	case "table":
		return NewTableFormatter(writer), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: json, table)", format)
	}
}

