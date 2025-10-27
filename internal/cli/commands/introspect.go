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

	"github.com/conduit-lang/conduit/runtime/metadata"
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
		RunE: runIntrospectResourcesCommand,
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
		RunE: runIntrospectResourceCommand,
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

  # Filter by resource
  conduit introspect routes --resource Post

  # Output in JSON format
  conduit introspect routes --format json`,
		RunE: runIntrospectRoutesCommand,
	}

	// Add command-specific flags
	cmd.Flags().String("method", "", "Filter by HTTP method (GET, POST, PUT, DELETE)")
	cmd.Flags().String("middleware", "", "Filter by middleware name")
	cmd.Flags().String("resource", "", "Filter by resource name")

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
		RunE: runIntrospectDepsCommand,
	}

	// Add command-specific flags
	cmd.Flags().Int("depth", 1, "Traversal depth for dependency tree (max: 5)")
	cmd.Flags().Bool("reverse", false, "Show only reverse dependencies")
	cmd.Flags().String("type", "", "Filter by type: resource, middleware, or function")

	return cmd
}

// runIntrospectDepsCommand executes the 'introspect deps <resource>' command
func runIntrospectDepsCommand(cmd *cobra.Command, args []string) error {
	resourceName := args[0]

	// Get flag values
	depth, _ := cmd.Flags().GetInt("depth")
	reverse, _ := cmd.Flags().GetBool("reverse")
	typeFilter, _ := cmd.Flags().GetString("type")

	// Validate depth
	if depth < 1 || depth > 5 {
		return fmt.Errorf("depth must be between 1 and 5, got: %d", depth)
	}

	// Validate type filter if specified
	validTypes := map[string]bool{
		"resource":   true,
		"middleware": true,
		"function":   true,
	}
	var types []string
	if typeFilter != "" {
		if !validTypes[typeFilter] {
			return fmt.Errorf("invalid type filter: %s (valid: resource, middleware, function)", typeFilter)
		}
		// Map CLI type names to relationship types
		switch typeFilter {
		case "resource":
			types = []string{"belongs_to", "has_many", "has_many_through"}
		case "middleware":
			types = []string{"uses"}
		case "function":
			types = []string{"calls"}
		}
	}

	// Build dependency options
	opts := metadata.DependencyOptions{
		Depth:   depth,
		Reverse: reverse,
		Types:   types,
	}

	// Query dependencies
	graph, err := metadata.QueryDependencies(resourceName, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return handleResourceNotFound(resourceName, cmd.OutOrStdout())
		}
		if strings.Contains(err.Error(), "not initialized") {
			return fmt.Errorf("registry not initialized - run 'conduit build' first to generate metadata")
		}
		return err
	}

	// Get the output writer
	writer := cmd.OutOrStdout()

	// Format output based on the format flag
	if outputFormat == "json" {
		return formatDependenciesAsJSON(graph, writer)
	}

	// Default: table format
	return formatDependenciesAsTable(graph, resourceName, opts, writer)
}

// DependencyGroup groups dependencies by their type
type DependencyGroup struct {
	Type  string
	Edges []metadata.DependencyEdge
}

// groupDependenciesByType groups dependency edges by node type
func groupDependenciesByType(graph *metadata.DependencyGraph, reverse bool) map[string][]metadata.DependencyEdge {
	groups := make(map[string][]metadata.DependencyEdge)

	for _, edge := range graph.Edges {
		// Determine the target node type
		var targetNodeID string
		if reverse {
			targetNodeID = edge.From
		} else {
			targetNodeID = edge.To
		}

		if node, exists := graph.Nodes[targetNodeID]; exists {
			groups[node.Type] = append(groups[node.Type], edge)
		}
	}

	return groups
}

// getImpactDescription generates a human-readable impact description for a dependency
func getImpactDescription(edge metadata.DependencyEdge, graph *metadata.DependencyGraph, resourceName string, reverse bool) string {
	// Get source resource metadata
	var sourceRes *metadata.ResourceMetadata

	if reverse {
		sourceRes, _ = metadata.QueryResource(edge.From)
	} else {
		sourceRes, _ = metadata.QueryResource(resourceName)
	}

	// Handle relationship-based impacts
	if edge.Relationship == "belongs_to" || edge.Relationship == "has_many" || edge.Relationship == "has_many_through" {
		// Find the relationship metadata to get on_delete behavior
		var relMeta *metadata.RelationshipMetadata
		if sourceRes != nil {
			for i := range sourceRes.Relationships {
				rel := &sourceRes.Relationships[i]
				// Match both the target resource AND relationship type
				if rel.TargetResource == edge.To && rel.Type == edge.Relationship {
					relMeta = rel
					break
				}
			}
		}

		if relMeta != nil {
			switch relMeta.OnDelete {
			case "cascade":
				if reverse {
					return fmt.Sprintf("Deleting %s cascades to %s", resourceName, edge.From)
				}
				return fmt.Sprintf("Deleting %s cascades to %s", edge.To, resourceName)
			case "restrict":
				if reverse {
					return fmt.Sprintf("Cannot delete %s with existing %s", resourceName, edge.From)
				}
				return fmt.Sprintf("Cannot delete %s with existing %s", edge.To, resourceName)
			case "set_null":
				if reverse {
					return fmt.Sprintf("Deleting %s nullifies %s.%s", resourceName, edge.From, relMeta.ForeignKey)
				}
				return fmt.Sprintf("Deleting %s nullifies %s.%s", edge.To, resourceName, relMeta.ForeignKey)
			}
		}

		// Default for relationships without explicit on_delete
		if edge.Relationship == "belongs_to" {
			if reverse {
				return fmt.Sprintf("Deleting %s affects %s records", resourceName, edge.From)
			}
			return fmt.Sprintf("%s requires %s", resourceName, edge.To)
		}
		return fmt.Sprintf("%s relationship", edge.Relationship)
	}

	// Handle middleware usage
	if edge.Relationship == "uses" {
		return "Applied to operations"
	}

	// Handle function calls
	if edge.Relationship == "calls" {
		return "Called from hooks"
	}

	return ""
}

// formatDependenciesAsTable formats dependency graph as a human-readable table
func formatDependenciesAsTable(graph *metadata.DependencyGraph, resourceName string, opts metadata.DependencyOptions, writer io.Writer) error {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)

	// Header
	bold.Fprintf(writer, "DEPENDENCIES: %s\n\n", resourceName)

	// Show direct dependencies (what resource uses) unless --reverse is specified
	if !opts.Reverse {
		cyan.Fprintln(writer, "━━━ DIRECT DEPENDENCIES (what "+resourceName+" uses) ━━━━━━")
		fmt.Fprintln(writer)

		// Group dependencies by type
		groups := groupDependenciesByType(graph, false)

		// Show resources
		if resourceEdges, ok := groups["resource"]; ok && len(resourceEdges) > 0 {
			bold.Fprintln(writer, "Resources:")
			for _, edge := range resourceEdges {
				targetNode := graph.Nodes[edge.To]
				fmt.Fprintf(writer, "└─ %s (%s)\n", targetNode.Name, edge.Relationship)

				impact := getImpactDescription(edge, graph, resourceName, false)
				if impact != "" {
					yellow.Fprintf(writer, "   Impact: %s\n", impact)
				}
			}
			fmt.Fprintln(writer)
		}

		// Show middleware
		if middlewareEdges, ok := groups["middleware"]; ok && len(middlewareEdges) > 0 {
			bold.Fprintln(writer, "Middleware:")
			for _, edge := range middlewareEdges {
				targetNode := graph.Nodes[edge.To]
				fmt.Fprintf(writer, "└─ %s\n", targetNode.Name)
			}
			fmt.Fprintln(writer)
		}

		// Show functions
		if functionEdges, ok := groups["function"]; ok && len(functionEdges) > 0 {
			bold.Fprintln(writer, "Functions:")
			for _, edge := range functionEdges {
				targetNode := graph.Nodes[edge.To]
				fmt.Fprintf(writer, "└─ %s\n", targetNode.Name)
			}
			fmt.Fprintln(writer)
		}

		if len(groups) == 0 {
			fmt.Fprintln(writer, "No direct dependencies")
			fmt.Fprintln(writer)
		}
	}

	// Show reverse dependencies (what uses resource)
	// Always shown: in default mode (both directions) or when --reverse is specified
	{
		// Query reverse dependencies
		reverseOpts := metadata.DependencyOptions{
			Depth:   opts.Depth,
			Reverse: true,
			Types:   opts.Types,
		}

		reverseGraph, err := metadata.QueryDependencies(resourceName, reverseOpts)
		if err != nil {
			return err
		}

		cyan.Fprintln(writer, "━━━ REVERSE DEPENDENCIES (what uses "+resourceName+") ━━━━━━")
		fmt.Fprintln(writer)

		// Group reverse dependencies by type
		reverseGroups := groupDependenciesByType(reverseGraph, true)

		// Show resources
		if resourceEdges, ok := reverseGroups["resource"]; ok && len(resourceEdges) > 0 {
			bold.Fprintln(writer, "Resources:")
			for _, edge := range resourceEdges {
				sourceNode := reverseGraph.Nodes[edge.From]
				fmt.Fprintf(writer, "└─ %s (via %s to %s)\n", sourceNode.Name, edge.Relationship, resourceName)

				impact := getImpactDescription(edge, reverseGraph, resourceName, true)
				if impact != "" {
					yellow.Fprintf(writer, "   Impact: %s\n", impact)
				}
			}
			fmt.Fprintln(writer)
		}

		// Show routes that use this resource
		allRoutes := metadata.QueryRoutes()
		resourceRoutes := []metadata.RouteMetadata{}
		for _, route := range allRoutes {
			if route.Resource == resourceName {
				resourceRoutes = append(resourceRoutes, route)
			}
		}

		if len(resourceRoutes) > 0 {
			bold.Fprintln(writer, "Routes:")
			for _, route := range resourceRoutes {
				fmt.Fprintf(writer, "└─ %s %s\n", route.Method, route.Path)
			}
			fmt.Fprintln(writer)
		}

		if len(reverseGroups) == 0 && len(resourceRoutes) == 0 {
			fmt.Fprintln(writer, "No reverse dependencies")
			fmt.Fprintln(writer)
		}
	}

	return nil
}

// formatDependenciesAsJSON formats dependency graph as JSON
func formatDependenciesAsJSON(graph *metadata.DependencyGraph, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(graph)
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

// runIntrospectResourcesCommand executes the 'introspect resources' command
func runIntrospectResourcesCommand(cmd *cobra.Command, args []string) error {
	// Get resources from the registry
	resources := metadata.QueryResources()
	if resources == nil {
		return fmt.Errorf("registry not initialized - run 'conduit build' first to generate metadata")
	}

	// Get the output writer
	writer := cmd.OutOrStdout()

	// Format output based on the format flag
	if outputFormat == "json" {
		return formatResourcesAsJSON(resources, writer)
	}

	// Default: table format
	return formatResourcesAsTable(resources, writer, verbose)
}

// ResourceCategory represents a category of resources
type ResourceCategory struct {
	Name      string
	Resources []ResourceSummary
}

// ResourceSummary contains summary information about a resource
type ResourceSummary struct {
	Name              string
	FieldCount        int
	RelationshipCount int
	HookCount         int
	AuthRequired      bool
	Cached            bool
	Nested            bool
}

// categorizeResources groups resources into categories
func categorizeResources(resources []metadata.ResourceMetadata) []ResourceCategory {
	categories := make(map[string][]ResourceSummary)

	for _, res := range resources {
		summary := ResourceSummary{
			Name:              res.Name,
			FieldCount:        len(res.Fields),
			RelationshipCount: len(res.Relationships),
			HookCount:         len(res.Hooks),
		}

		// Analyze middleware to determine flags
		for op, middlewares := range res.Middleware {
			_ = op // unused for now
			for _, mw := range middlewares {
				if strings.Contains(strings.ToLower(mw), "auth") {
					summary.AuthRequired = true
				}
				if strings.Contains(strings.ToLower(mw), "cache") {
					summary.Cached = true
				}
			}
		}

		// Determine if resource is nested (has parent relationship)
		for _, rel := range res.Relationships {
			if rel.Type == "belongs_to" {
				summary.Nested = true
				break
			}
		}

		// Categorize based on resource name patterns
		category := categorizeResource(res.Name)
		categories[category] = append(categories[category], summary)
	}

	// Convert to ordered list
	result := make([]ResourceCategory, 0)
	categoryOrder := []string{"Core Resources", "Administrative", "System"}

	for _, catName := range categoryOrder {
		if resources, ok := categories[catName]; ok {
			result = append(result, ResourceCategory{
				Name:      catName,
				Resources: resources,
			})
		}
	}

	// Add any remaining categories
	for catName, resources := range categories {
		found := false
		for _, orderName := range categoryOrder {
			if catName == orderName {
				found = true
				break
			}
		}
		if !found {
			result = append(result, ResourceCategory{
				Name:      catName,
				Resources: resources,
			})
		}
	}

	return result
}

// categorizeResource determines the category for a resource
func categorizeResource(name string) string {
	// Common patterns for categorization
	corePatterns := []string{"User", "Post", "Comment", "Article", "Page", "Product", "Order"}
	adminPatterns := []string{"Category", "Tag", "Setting", "Config"}
	systemPatterns := []string{"Log", "Audit", "Session", "Token", "Job"}

	for _, pattern := range corePatterns {
		if name == pattern {
			return "Core Resources"
		}
	}

	for _, pattern := range adminPatterns {
		if name == pattern {
			return "Administrative"
		}
	}

	for _, pattern := range systemPatterns {
		if name == pattern {
			return "System"
		}
	}

	// Default to Core Resources
	return "Core Resources"
}

// formatResourcesAsTable formats resources as a human-readable table
func formatResourcesAsTable(resources []metadata.ResourceMetadata, writer io.Writer, verbose bool) error {
	if len(resources) == 0 {
		fmt.Fprintln(writer, "No resources found.")
		return nil
	}

	// Categorize resources
	categories := categorizeResources(resources)

	// Print header
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)

	bold.Fprintf(writer, "RESOURCES (%d total)\n\n", len(resources))

	// Print each category
	for _, category := range categories {
		if len(category.Resources) == 0 {
			continue
		}

		cyan.Fprintf(writer, "%s:\n", category.Name)

		if verbose {
			// Verbose mode: show detailed information
			for _, res := range category.Resources {
				fmt.Fprintf(writer, "  %s\n", res.Name)
				fmt.Fprintf(writer, "    Fields: %d\n", res.FieldCount)
				fmt.Fprintf(writer, "    Relationships: %d\n", res.RelationshipCount)
				fmt.Fprintf(writer, "    Hooks: %d\n", res.HookCount)

				flags := []string{}
				if res.AuthRequired {
					flags = append(flags, "auth required")
				}
				if res.Cached {
					flags = append(flags, "cached")
				}
				if res.Nested {
					flags = append(flags, "nested")
				}

				if len(flags) > 0 {
					fmt.Fprintf(writer, "    Flags: %s\n", strings.Join(flags, ", "))
				}
				fmt.Fprintln(writer)
			}
		} else {
			// Default mode: compact summary
			for _, res := range category.Resources {
				// Format: "  User        8 fields  2 relationships  1 hook  ✓ auth required"
				fmt.Fprintf(writer, "  %-12s", res.Name)

				// Fields
				if res.FieldCount > 0 {
					fmt.Fprintf(writer, "%d fields  ", res.FieldCount)
				} else {
					fmt.Fprintf(writer, "-           ")
				}

				// Relationships
				if res.RelationshipCount > 0 {
					if res.RelationshipCount == 1 {
						fmt.Fprintf(writer, "%d relationship   ", res.RelationshipCount)
					} else {
						fmt.Fprintf(writer, "%d relationships  ", res.RelationshipCount)
					}
				} else {
					fmt.Fprintf(writer, "-                 ")
				}

				// Hooks
				if res.HookCount > 0 {
					if res.HookCount == 1 {
						fmt.Fprintf(writer, "%d hook   ", res.HookCount)
					} else {
						fmt.Fprintf(writer, "%d hooks  ", res.HookCount)
					}
				} else {
					fmt.Fprintf(writer, "-        ")
				}

				// Flags
				flags := []string{}
				if res.AuthRequired {
					flags = append(flags, green.Sprint("✓ auth required"))
				}
				if res.Cached {
					flags = append(flags, green.Sprint("✓ cached"))
				}
				if res.Nested {
					flags = append(flags, green.Sprint("✓ nested"))
				}

				if len(flags) > 0 {
					fmt.Fprintf(writer, "%s", strings.Join(flags, "  "))
				}

				fmt.Fprintln(writer)
			}
		}

		fmt.Fprintln(writer)
	}

	return nil
}

// formatResourcesAsJSON formats resources as JSON
func formatResourcesAsJSON(resources []metadata.ResourceMetadata, writer io.Writer) error {
	// Create summary data for JSON output
	type JSONResourceSummary struct {
		Name              string              `json:"name"`
		FieldCount        int                 `json:"field_count"`
		RelationshipCount int                 `json:"relationship_count"`
		HookCount         int                 `json:"hook_count"`
		ValidationCount   int                 `json:"validation_count"`
		ConstraintCount   int                 `json:"constraint_count"`
		Middleware        map[string][]string `json:"middleware,omitempty"`
		Category          string              `json:"category"`
		Flags             []string            `json:"flags,omitempty"`
	}

	type JSONOutput struct {
		TotalCount int                   `json:"total_count"`
		Resources  []JSONResourceSummary `json:"resources"`
	}

	output := JSONOutput{
		TotalCount: len(resources),
		Resources:  make([]JSONResourceSummary, 0, len(resources)),
	}

	for _, res := range resources {
		summary := JSONResourceSummary{
			Name:              res.Name,
			FieldCount:        len(res.Fields),
			RelationshipCount: len(res.Relationships),
			HookCount:         len(res.Hooks),
			ValidationCount:   len(res.Validations),
			ConstraintCount:   len(res.Constraints),
			Middleware:        res.Middleware,
			Category:          categorizeResource(res.Name),
			Flags:             []string{},
		}

		// Determine flags
		for _, middlewares := range res.Middleware {
			for _, mw := range middlewares {
				if strings.Contains(strings.ToLower(mw), "auth") {
					summary.Flags = append(summary.Flags, "auth_required")
				}
				if strings.Contains(strings.ToLower(mw), "cache") {
					summary.Flags = append(summary.Flags, "cached")
				}
			}
		}

		for _, rel := range res.Relationships {
			if rel.Type == "belongs_to" {
				summary.Flags = append(summary.Flags, "nested")
				break
			}
		}

		output.Resources = append(output.Resources, summary)
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// runIntrospectResourceCommand executes the 'introspect resource <name>' command
func runIntrospectResourceCommand(cmd *cobra.Command, args []string) error {
	resourceName := args[0]

	// Get resource from the registry
	resource, err := metadata.QueryResource(resourceName)
	if err != nil {
		// Try to suggest similar resource names
		return handleResourceNotFound(resourceName, cmd.OutOrStdout())
	}

	// Get the output writer
	writer := cmd.OutOrStdout()

	// Format output based on the format flag
	if outputFormat == "json" {
		return formatResourceAsJSON(resource, writer)
	}

	// Default: table format
	return formatResourceAsTable(resource, writer, verbose)
}

// handleResourceNotFound handles the case when a resource is not found
// and suggests similar resource names using fuzzy search
func handleResourceNotFound(name string, writer io.Writer) error {
	// Get all resources for fuzzy matching
	resources := metadata.QueryResources()
	if resources == nil {
		return fmt.Errorf("registry not initialized - run 'conduit build' first to generate metadata")
	}

	// Find similar resource names
	suggestions := findSimilarResourceNames(name, resources)

	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)

	red.Fprintf(writer, "Error: Resource '%s' not found\n\n", name)

	if len(suggestions) > 0 {
		yellow.Fprintln(writer, "Did you mean:")
		for _, suggestion := range suggestions {
			fmt.Fprintf(writer, "  - %s\n", suggestion)
		}
		fmt.Fprintln(writer)
	}

	fmt.Fprintln(writer, "Available resources:")
	for _, res := range resources {
		fmt.Fprintf(writer, "  - %s\n", res.Name)
	}

	return fmt.Errorf("resource not found: %s", name)
}

// findSimilarResourceNames finds resource names similar to the given name
// using Levenshtein distance algorithm
func findSimilarResourceNames(name string, resources []metadata.ResourceMetadata) []string {
	const maxDistance = 3 // Maximum edit distance to consider
	const maxSuggestions = 3

	type suggestion struct {
		name     string
		distance int
	}

	var suggestions []suggestion

	for _, res := range resources {
		dist := levenshteinDistance(strings.ToLower(name), strings.ToLower(res.Name))
		if dist <= maxDistance {
			suggestions = append(suggestions, suggestion{
				name:     res.Name,
				distance: dist,
			})
		}
	}

	// Sort by distance (closest first)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].distance < suggestions[j].distance
	})

	// Return top suggestions
	result := make([]string, 0, maxSuggestions)
	for i := 0; i < len(suggestions) && i < maxSuggestions; i++ {
		result = append(result, suggestions[i].name)
	}

	return result
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first column and row
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// formatResourceAsTable formats a single resource as a human-readable table
func formatResourceAsTable(resource *metadata.ResourceMetadata, writer io.Writer, verbose bool) error {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	// Header section
	bold.Fprintf(writer, "RESOURCE: %s\n", resource.Name)
	if resource.FilePath != "" {
		fmt.Fprintf(writer, "File: %s\n", resource.FilePath)
	}
	if resource.Documentation != "" {
		fmt.Fprintf(writer, "Docs: %s\n", resource.Documentation)
	}
	fmt.Fprintln(writer)

	// Schema section
	cyan.Fprintln(writer, "━━━ SCHEMA ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(writer)

	// Fields
	bold.Fprintf(writer, "FIELDS (%d):\n", len(resource.Fields))

	// Group fields by required/optional
	requiredFields := []metadata.FieldMetadata{}
	optionalFields := []metadata.FieldMetadata{}

	for _, field := range resource.Fields {
		if field.Required {
			requiredFields = append(requiredFields, field)
		} else {
			optionalFields = append(optionalFields, field)
		}
	}

	if len(requiredFields) > 0 {
		fmt.Fprintf(writer, "Required (%d):\n", len(requiredFields))
		for _, field := range requiredFields {
			fmt.Fprintf(writer, "  %s  %s", field.Name, field.Type)
			if len(field.Constraints) > 0 {
				fmt.Fprintf(writer, "  %s", strings.Join(field.Constraints, " "))
			}
			if field.DefaultValue != "" {
				fmt.Fprintf(writer, "  (default: %s)", field.DefaultValue)
			}
			fmt.Fprintln(writer)
			if verbose && field.Documentation != "" {
				fmt.Fprintf(writer, "    %s\n", field.Documentation)
			}
		}
		fmt.Fprintln(writer)
	}

	if len(optionalFields) > 0 {
		fmt.Fprintf(writer, "Optional (%d):\n", len(optionalFields))
		for _, field := range optionalFields {
			fmt.Fprintf(writer, "  %s  %s", field.Name, field.Type)
			if len(field.Constraints) > 0 {
				fmt.Fprintf(writer, "  %s", strings.Join(field.Constraints, " "))
			}
			if field.DefaultValue != "" {
				fmt.Fprintf(writer, "  (default: %s)", field.DefaultValue)
			}
			fmt.Fprintln(writer)
			if verbose && field.Documentation != "" {
				fmt.Fprintf(writer, "    %s\n", field.Documentation)
			}
		}
		fmt.Fprintln(writer)
	}

	// Relationships
	if len(resource.Relationships) > 0 {
		bold.Fprintf(writer, "RELATIONSHIPS (%d):\n", len(resource.Relationships))
		for _, rel := range resource.Relationships {
			fmt.Fprintf(writer, "  → %s (%s %s)\n", rel.Name, rel.Type, rel.TargetResource)
			if rel.ForeignKey != "" {
				fmt.Fprintf(writer, "    Foreign key: %s\n", rel.ForeignKey)
			}
			if rel.ThroughTable != "" {
				fmt.Fprintf(writer, "    Through: %s\n", rel.ThroughTable)
			}
			if rel.OnDelete != "" {
				fmt.Fprintf(writer, "    On delete: %s\n", rel.OnDelete)
			}
			if rel.OnUpdate != "" {
				fmt.Fprintf(writer, "    On update: %s\n", rel.OnUpdate)
			}
		}
		fmt.Fprintln(writer)
	}

	// Behavior section
	if len(resource.Hooks) > 0 || len(resource.Constraints) > 0 || len(resource.Validations) > 0 {
		cyan.Fprintln(writer, "━━━ BEHAVIOR ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Fprintln(writer)

		// Lifecycle Hooks
		if len(resource.Hooks) > 0 {
			bold.Fprintln(writer, "LIFECYCLE HOOKS:")

			// Group hooks by type
			hooksByType := make(map[string][]metadata.HookMetadata)
			for _, hook := range resource.Hooks {
				hooksByType[hook.Type] = append(hooksByType[hook.Type], hook)
			}

			// Sort hook types for consistent output
			hookTypes := make([]string, 0, len(hooksByType))
			for hookType := range hooksByType {
				hookTypes = append(hookTypes, hookType)
			}
			sort.Strings(hookTypes)

			for _, hookType := range hookTypes {
				hooks := hooksByType[hookType]
				if len(hooks) == 0 {
					continue
				}

				fmt.Fprintf(writer, "  @%s", hookType)

				// Gather flags from all hooks of this type
				flags := make(map[string]bool)
				for _, hook := range hooks {
					if hook.Transaction {
						flags["transaction"] = true
					}
					if hook.Async {
						flags["async"] = true
					}
				}

				if len(flags) > 0 {
					flagList := make([]string, 0, len(flags))
					for flag := range flags {
						flagList = append(flagList, flag)
					}
					sort.Strings(flagList) // For consistent output
					fmt.Fprintf(writer, " [%s]", strings.Join(flagList, ", "))
				}
				fmt.Fprintln(writer, ":")

				// Show source code for all hooks in verbose mode
				if verbose {
					for idx, hook := range hooks {
						if hook.SourceCode != "" {
							if len(hooks) > 1 {
								fmt.Fprintf(writer, "    Hook %d:\n", idx+1)
							}
							lines := strings.Split(hook.SourceCode, "\n")
							for _, line := range lines {
								if len(hooks) > 1 {
									fmt.Fprintf(writer, "      %s\n", line)
								} else {
									fmt.Fprintf(writer, "    %s\n", line)
								}
							}
						}
					}
				}
			}
			fmt.Fprintln(writer)
		}

		// Constraints
		if len(resource.Constraints) > 0 {
			bold.Fprintf(writer, "CONSTRAINTS (%d):\n", len(resource.Constraints))
			for _, constraint := range resource.Constraints {
				green.Fprintf(writer, "  ✓ %s\n", constraint.Name)
				if verbose {
					fmt.Fprintf(writer, "    Operations: %s\n", strings.Join(constraint.Operations, ", "))
					if constraint.When != "" {
						fmt.Fprintf(writer, "    When: %s\n", constraint.When)
					}
					fmt.Fprintf(writer, "    Condition: %s\n", constraint.Condition)
					fmt.Fprintf(writer, "    Error: %s\n", constraint.Error)
				}
			}
			fmt.Fprintln(writer)
		}

		// Validations
		if len(resource.Validations) > 0 && verbose {
			bold.Fprintf(writer, "VALIDATIONS (%d):\n", len(resource.Validations))
			for _, validation := range resource.Validations {
				fmt.Fprintf(writer, "  %s: %s", validation.Field, validation.Type)
				if validation.Value != "" {
					fmt.Fprintf(writer, "(%s)", validation.Value)
				}
				if validation.Message != "" {
					fmt.Fprintf(writer, " - %s", validation.Message)
				}
				fmt.Fprintln(writer)
			}
			fmt.Fprintln(writer)
		}
	}

	// API Endpoints section
	cyan.Fprintln(writer, "━━━ API ENDPOINTS ━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(writer)

	// Get all routes and filter by this resource
	allRoutes := metadata.QueryRoutes()
	resourceRoutes := []metadata.RouteMetadata{}
	for _, route := range allRoutes {
		if route.Resource == resource.Name {
			resourceRoutes = append(resourceRoutes, route)
		}
	}

	if len(resourceRoutes) > 0 {
		for _, route := range resourceRoutes {
			fmt.Fprintf(writer, "%s %s → %s", route.Method, route.Path, route.Operation)
			if len(route.Middleware) > 0 {
				yellow.Fprintf(writer, " [%s]", strings.Join(route.Middleware, ", "))
			}
			fmt.Fprintln(writer)
		}
	} else {
		fmt.Fprintln(writer, "No auto-generated routes for this resource.")
	}

	// Show middleware summary
	if len(resource.Middleware) > 0 && verbose {
		fmt.Fprintln(writer)
		bold.Fprintln(writer, "MIDDLEWARE BY OPERATION:")

		// Sort operations for consistent output
		operations := make([]string, 0, len(resource.Middleware))
		for op := range resource.Middleware {
			operations = append(operations, op)
		}
		sort.Strings(operations)

		for _, op := range operations {
			middlewares := resource.Middleware[op]
			fmt.Fprintf(writer, "  %s: %s\n", op, strings.Join(middlewares, ", "))
		}
	}

	fmt.Fprintln(writer)
	return nil
}

// formatResourceAsJSON formats a single resource as JSON
func formatResourceAsJSON(resource *metadata.ResourceMetadata, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(resource)
}

// runIntrospectRoutesCommand executes the 'introspect routes' command
func runIntrospectRoutesCommand(cmd *cobra.Command, args []string) error {
	// Get all routes from the registry
	routes := metadata.QueryRoutes()
	if routes == nil {
		return fmt.Errorf("registry not initialized - run 'conduit build' first to generate metadata")
	}

	// Get filter flags
	methodFilter, _ := cmd.Flags().GetString("method")
	middlewareFilter, _ := cmd.Flags().GetString("middleware")
	resourceFilter, _ := cmd.Flags().GetString("resource")

	// Apply filtering
	filteredRoutes := filterRoutes(routes, methodFilter, middlewareFilter, resourceFilter)

	// Sort routes alphabetically by path
	sort.Slice(filteredRoutes, func(i, j int) bool {
		return filteredRoutes[i].Path < filteredRoutes[j].Path
	})

	// Get the output writer
	writer := cmd.OutOrStdout()

	// Format output based on the format flag
	if outputFormat == "json" {
		return formatRoutesAsJSON(filteredRoutes, writer)
	}

	// Default: table format
	return formatRoutesAsTable(filteredRoutes, writer)
}

// filterRoutes applies filtering logic to routes based on the provided filters
func filterRoutes(routes []metadata.RouteMetadata, methodFilter, middlewareFilter, resourceFilter string) []metadata.RouteMetadata {
	if methodFilter == "" && middlewareFilter == "" && resourceFilter == "" {
		return routes
	}

	filtered := make([]metadata.RouteMetadata, 0, len(routes))
	for _, route := range routes {
		// Check method filter (case-insensitive)
		if methodFilter != "" && !strings.EqualFold(route.Method, methodFilter) {
			continue
		}

		// Check middleware filter (substring match)
		if middlewareFilter != "" {
			found := false
			for _, mw := range route.Middleware {
				if strings.Contains(strings.ToLower(mw), strings.ToLower(middlewareFilter)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check resource filter (exact match)
		if resourceFilter != "" && route.Resource != resourceFilter {
			continue
		}

		filtered = append(filtered, route)
	}

	return filtered
}

// formatRoutesAsTable formats routes as a human-readable table
func formatRoutesAsTable(routes []metadata.RouteMetadata, writer io.Writer) error {
	if len(routes) == 0 {
		fmt.Fprintln(writer, "No routes found.")
		return nil
	}

	// Define colors for different HTTP methods
	getColor := color.New(color.FgGreen)
	postColor := color.New(color.FgBlue)
	putColor := color.New(color.FgYellow)
	deleteColor := color.New(color.FgRed)
	defaultColor := color.New(color.Reset)

	for _, route := range routes {
		// Colorize method based on HTTP verb
		var methodColor *color.Color
		switch strings.ToUpper(route.Method) {
		case "GET":
			methodColor = getColor
		case "POST":
			methodColor = postColor
		case "PUT":
			methodColor = putColor
		case "DELETE":
			methodColor = deleteColor
		default:
			methodColor = defaultColor
		}

		// Format: METHOD PATH -> HANDLER [MIDDLEWARE]
		methodColor.Fprintf(writer, "%-6s", route.Method)
		fmt.Fprintf(writer, " %-30s -> ", route.Path)
		fmt.Fprintf(writer, "%-20s", route.Handler)

		// Show middleware if present
		if len(route.Middleware) > 0 {
			yellow := color.New(color.FgYellow)
			yellow.Fprintf(writer, " [%s]", strings.Join(route.Middleware, ", "))
		}

		fmt.Fprintln(writer)
	}

	return nil
}

// formatRoutesAsJSON formats routes as JSON
func formatRoutesAsJSON(routes []metadata.RouteMetadata, writer io.Writer) error {
	type JSONOutput struct {
		TotalCount int                      `json:"total_count"`
		Routes     []metadata.RouteMetadata `json:"routes"`
	}

	output := JSONOutput{
		TotalCount: len(routes),
		Routes:     routes,
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
