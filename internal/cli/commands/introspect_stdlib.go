package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/internal/compiler/stdlib"
)

// newIntrospectStdlibCommand creates the 'introspect stdlib' command
func newIntrospectStdlibCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stdlib [namespace]",
		Short: "List standard library functions",
		Long: `List standard library functions organized by namespace.

The stdlib introspection command provides a complete reference of all available
standard library functions in Conduit. This helps LLMs and developers discover
stdlib functions without hallucinating names.

Available before build - does not require compilation or metadata.`,
		Example: `  # List all stdlib functions
  conduit introspect stdlib

  # List functions in a specific namespace
  conduit introspect stdlib String

  # Output in JSON format for tooling
  conduit introspect stdlib --format json

  # List only Time namespace functions as JSON
  conduit introspect stdlib Time --format json`,
		Args: cobra.MaximumNArgs(1),
		RunE: runIntrospectStdlibCommand,
	}

	return cmd
}

// runIntrospectStdlibCommand executes the 'introspect stdlib [namespace]' command
func runIntrospectStdlibCommand(cmd *cobra.Command, args []string) error {
	writer := cmd.OutOrStdout()

	// Check if a specific namespace was requested
	var namespaceFilter string
	if len(args) > 0 {
		namespaceFilter = args[0]
	}

	// Validate namespace if specified
	if namespaceFilter != "" {
		funcs := stdlib.GetFunctions(namespaceFilter)
		if funcs == nil {
			return handleStdlibNamespaceNotFound(namespaceFilter, writer)
		}
	}

	// Format output based on the format flag
	if outputFormat == "json" {
		return formatStdlibAsJSON(namespaceFilter, writer)
	}

	// Default: human-readable format
	return formatStdlibAsTable(namespaceFilter, writer)
}

// handleStdlibNamespaceNotFound handles the case when a namespace is not found
func handleStdlibNamespaceNotFound(namespace string, writer io.Writer) error {
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)

	red.Fprintf(writer, "Error: Namespace '%s' not found\n\n", namespace)

	// Show available namespaces
	yellow.Fprintln(writer, "Available namespaces:")
	for _, ns := range stdlib.GetNamespaces() {
		fmt.Fprintf(writer, "  • %s\n", ns)
	}

	return fmt.Errorf("namespace not found: %s", namespace)
}

// formatStdlibAsTable formats stdlib functions as human-readable output
func formatStdlibAsTable(namespaceFilter string, writer io.Writer) error {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)
	gray := color.New(color.Faint)

	// Determine which namespaces to display
	var namespaces []string
	if namespaceFilter != "" {
		namespaces = []string{namespaceFilter}
	} else {
		namespaces = stdlib.GetNamespaces()
	}

	// Print header
	if namespaceFilter == "" {
		bold.Fprintf(writer, "STANDARD LIBRARY FUNCTIONS (%d total)\n\n", stdlib.TotalFunctionCount())
		gray.Fprintln(writer, "All stdlib functions are namespaced. Use Namespace.function() syntax.")
		fmt.Fprintln(writer)
	}

	// Print each namespace
	for i, namespace := range namespaces {
		funcs := stdlib.GetFunctions(namespace)
		if funcs == nil {
			continue
		}

		// Namespace header
		cyan.Fprintf(writer, "%s Functions (%d):\n", namespace, len(funcs))

		// Print each function
		for _, fn := range funcs {
			// Function signature in green
			green.Fprintf(writer, "  %s.%s\n", namespace, fn.Signature)

			// Description in gray
			gray.Fprintf(writer, "    %s\n", fn.Description)
		}

		// Add spacing between namespaces (but not after the last one)
		if i < len(namespaces)-1 {
			fmt.Fprintln(writer)
		}
	}

	return nil
}

// formatStdlibAsJSON formats stdlib functions as JSON
func formatStdlibAsJSON(namespaceFilter string, writer io.Writer) error {
	type FunctionJSON struct {
		Name        string `json:"name"`
		Signature   string `json:"signature"`
		Description string `json:"description"`
	}

	type NamespaceJSON struct {
		Namespace string         `json:"namespace"`
		Functions []FunctionJSON `json:"functions"`
	}

	type OutputJSON struct {
		TotalCount int             `json:"total_count"`
		Namespaces []NamespaceJSON `json:"namespaces"`
	}

	output := OutputJSON{
		Namespaces: []NamespaceJSON{},
	}

	// Determine which namespaces to include
	var namespaces []string
	if namespaceFilter != "" {
		namespaces = []string{namespaceFilter}
	} else {
		namespaces = stdlib.GetNamespaces()
	}

	// Build output structure
	totalCount := 0
	for _, namespace := range namespaces {
		funcs := stdlib.GetFunctions(namespace)
		if funcs == nil {
			continue
		}

		nsJSON := NamespaceJSON{
			Namespace: namespace,
			Functions: make([]FunctionJSON, len(funcs)),
		}

		for i, fn := range funcs {
			nsJSON.Functions[i] = FunctionJSON{
				Name:        fn.Name,
				Signature:   fn.Signature,
				Description: fn.Description,
			}
			totalCount++
		}

		output.Namespaces = append(output.Namespaces, nsJSON)
	}

	output.TotalCount = totalCount

	// Encode as JSON
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
