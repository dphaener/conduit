package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/internal/testing/llm"
	"github.com/conduit-lang/conduit/runtime/metadata"
)

// NewTestPatternsCommand creates the 'conduit test-patterns' command.
func NewTestPatternsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test-patterns",
		Short: "Run LLM pattern validation iterations",
		Long: `Run iterative pattern validation against LLMs.

Validates that extracted patterns can be successfully used by LLMs for code generation.
Runs multiple iterations, analyzing failures and tracking progress toward success criteria.

Success Criteria:
  - Claude Opus: 80%+ pattern adherence
  - GPT-4: 70%+ pattern adherence
  - No critical patterns with <50% adherence

The command will:
1. Load resources metadata from introspection
2. Extract patterns from resources
3. Generate test cases from patterns
4. Validate with configured LLM providers
5. Analyze failures and suggest improvements
6. Repeat until success criteria met or max iterations reached`,
		Example: `  # Run with default settings
  conduit test-patterns

  # Run with custom iteration limit
  conduit test-patterns --max-iterations 5

  # Use mock LLM for testing (no API calls)
  conduit test-patterns --mock

  # Export results to JSON
  conduit test-patterns --format json --output results.json

  # Verbose output with detailed progress
  conduit test-patterns --verbose`,
		RunE: runTestPatternsCommand,
	}

	cmd.Flags().Int("max-iterations", 4, "Maximum number of iterations to run")
	cmd.Flags().Bool("mock", false, "Use mock LLM for testing (no API calls)")
	cmd.Flags().String("format", "text", "Output format: text or json")
	cmd.Flags().String("output", "", "Output file (default: stdout)")
	cmd.Flags().Bool("verbose", false, "Show detailed output")
	cmd.Flags().Bool("no-fail", false, "Don't exit with error if criteria not met (for exploration)")

	return cmd
}

// runTestPatternsCommand executes the 'test-patterns' command.
func runTestPatternsCommand(cmd *cobra.Command, args []string) error {
	// Get flag values
	maxIterations, _ := cmd.Flags().GetInt("max-iterations")
	useMock, _ := cmd.Flags().GetBool("mock")
	format, _ := cmd.Flags().GetString("format")
	outputPath, _ := cmd.Flags().GetString("output")
	verbose, _ := cmd.Flags().GetBool("verbose")
	noFail, _ := cmd.Flags().GetBool("no-fail")

	// Validate flags
	if maxIterations < 1 || maxIterations > 10 {
		return fmt.Errorf("max-iterations must be between 1 and 10, got: %d", maxIterations)
	}

	if format != "text" && format != "json" {
		return fmt.Errorf("format must be 'text' or 'json', got: %s", format)
	}

	// Get output writer
	var writer *os.File
	if outputPath != "" {
		var err error
		writer, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	// Load resources from registry
	if verbose {
		fmt.Fprintln(cmd.OutOrStdout(), "Loading resources from registry...")
	}

	resources := metadata.QueryResources()
	if resources == nil {
		return fmt.Errorf("registry not initialized - run 'conduit build' first to generate metadata")
	}

	if len(resources) == 0 {
		return fmt.Errorf(`no resources found in registry

To create your first resource:

  mkdir -p app/resources
  cat > app/resources/todo.cdt << 'EOF'
  resource Todo {
    id: uuid! @primary @auto
    title: string!
    created_at: timestamp! @auto
  }
  EOF

Then run: conduit build`)
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Loaded %d resources\n\n", len(resources))
	}

	// Create iteration config
	config := llm.DefaultIterationConfig()
	config.MaxIterations = maxIterations

	if verbose {
		fmt.Fprintln(cmd.OutOrStdout(), "Success criteria:")
		for provider, target := range config.TargetSuccessRate {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s: %.0f%%\n", provider, target*100)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  - Minimum pattern success: %.0f%%\n\n", config.MinimumPatternSuccess*100)
	}

	// Set up LLM harness
	var harness *llm.Harness
	var err error

	if useMock {
		if verbose {
			fmt.Fprintln(cmd.OutOrStdout(), "Using mock LLM (no API calls)")
			fmt.Fprintln(cmd.OutOrStdout())
		}
		// Create mock config with fake providers
		mockConfig := &llm.Config{
			Providers: []llm.ProviderConfig{
				{
					Type:       llm.ProviderClaude,
					Model:      "mock-claude",
					APIKey:     "mock-key",
					Timeout:    60 * time.Second,
					MaxRetries: 3,
					Enabled:    true,
				},
			},
			MaxConcurrentRequests: 3,
			DefaultTimeout:        60 * time.Second,
			RateLimitDelay:        0, // No delay for mock
		}
		harness, err = llm.NewHarness(mockConfig)
		if err != nil {
			return fmt.Errorf("failed to create harness: %w", err)
		}
	} else {
		// Create real config with API keys from environment
		harnessConfig := llm.NewDefaultConfig()
		harness, err = llm.NewHarness(harnessConfig)
		if err != nil {
			yellow := color.New(color.FgYellow)
			yellow.Fprintln(cmd.OutOrStdout(), "Warning: Failed to initialize LLM providers")
			yellow.Fprintln(cmd.OutOrStdout(), "Make sure you have set the required API keys:")
			yellow.Fprintln(cmd.OutOrStdout(), "  - ANTHROPIC_API_KEY for Claude")
			yellow.Fprintln(cmd.OutOrStdout(), "  - OPENAI_API_KEY for GPT models")
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Tip: Use --mock flag to test without API calls")
			return fmt.Errorf("failed to create harness: %w", err)
		}
	}

	harness.SetVerbose(verbose)

	// Create failure analyzer
	analyzer := llm.NewFailureAnalyzer()

	// Create iteration runner
	runner := llm.NewIterationRunner(harness, analyzer, config)
	runner.SetVerbose(verbose)

	// Set up mock if requested
	if useMock {
		mockFn := func(prompt string) string {
			// Simple mock that returns a valid middleware declaration
			return "@on create: [auth]"
		}
		runner.SetMockLLM(mockFn)
	}

	// Run iterations
	ctx := context.Background()

	if verbose {
		fmt.Fprintln(cmd.OutOrStdout(), "Starting pattern validation iterations...")
		fmt.Fprintln(cmd.OutOrStdout())
	}

	results, err := runner.Run(ctx, resources)
	if err != nil {
		return fmt.Errorf("iteration run failed: %w", err)
	}

	// Create reporter
	reporter := llm.NewIterationReporter(writer)

	// Output results based on format
	if format == "json" {
		jsonReport, err := reporter.ExportIterationReport(results, config)
		if err != nil {
			return fmt.Errorf("failed to export report: %w", err)
		}
		fmt.Fprintln(writer, jsonReport)
	} else {
		// Text format
		reporter.PrintIterationResults(results, config)
	}

	// Return error if success criteria not met (for CI/CD)
	if !noFail && len(results) > 0 && !results[len(results)-1].MetCriteria {
		return fmt.Errorf("success criteria not met after %d iterations", len(results))
	}

	return nil
}
