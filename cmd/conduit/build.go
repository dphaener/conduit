package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/compiler/errors"
	"github.com/conduit-lang/conduit/internal/tooling/build"
)

var (
	buildJSON      bool
	buildVerbose   bool
	buildMode      string
	buildOutput    string
	buildWatch     bool
	buildNoCache   bool
	buildMinify    bool
	buildTreeShake bool
	buildJobs      int
)

func init() {
	buildCmd.Flags().BoolVar(&buildJSON, "json", false, "Output errors in JSON format")
	buildCmd.Flags().BoolVar(&buildVerbose, "verbose", false, "Show detailed build output")
	buildCmd.Flags().StringVar(&buildMode, "mode", "development", "Build mode (development|production|test)")
	buildCmd.Flags().StringVar(&buildOutput, "output", "build/app", "Output path for binary")
	buildCmd.Flags().BoolVar(&buildWatch, "watch", false, "Watch for changes and rebuild")
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Disable build cache")
	buildCmd.Flags().BoolVar(&buildMinify, "minify", false, "Minify assets")
	buildCmd.Flags().BoolVar(&buildTreeShake, "tree-shake", false, "Enable tree shaking (EXPERIMENTAL - not yet implemented)")
	buildCmd.Flags().IntVar(&buildJobs, "jobs", 0, "Number of parallel jobs (0 = number of CPUs)")
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile Conduit source to Go and build binary",
	Long:  "Compile all .cdt files in the app/ directory and generate a native executable",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if app directory exists
		if _, err := os.Stat("app"); os.IsNotExist(err) {
			return fmt.Errorf("app/ directory not found - are you in a Conduit project?")
		}

		// Parse build mode
		var mode build.BuildMode
		switch buildMode {
		case "development", "dev":
			mode = build.ModeDevelopment
		case "production", "prod":
			mode = build.ModeProduction
		case "test":
			mode = build.ModeTest
		default:
			return fmt.Errorf("invalid build mode: %s (valid: development, production, test)", buildMode)
		}

		// Configure build options
		opts := &build.BuildOptions{
			Mode:       mode,
			OutputPath: buildOutput,
			SourceDir:  "app",
			BuildDir:   "build",
			Parallel:   true,
			MaxJobs:    buildJobs,
			Verbose:    buildVerbose,
			Watch:      buildWatch,
			UseCache:   !buildNoCache,
			Minify:     buildMinify,
			TreeShake:  buildTreeShake,
		}

		// Progress callback
		if buildVerbose {
			opts.ProgressFunc = func(current, total int, message string) {
				fmt.Printf("[%d/%d] %s\n", current, total, message)
			}
		} else {
			opts.ProgressFunc = func(current, total int, message string) {
				if current == total {
					fmt.Printf("✓ %s\n", message)
				}
			}
		}

		// Create build system
		sys, err := build.NewSystem(opts)
		if err != nil {
			return fmt.Errorf("failed to create build system: %w", err)
		}

		// Run build
		ctx := context.Background()
		result, err := sys.Build(ctx)
		if err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		// Check for compilation errors
		if !result.Success {
			if buildJSON {
				outputBuildErrorsJSON(result.Errors)
			} else {
				outputBuildErrorsTerminal(result.Errors)
			}
			return fmt.Errorf("compilation failed with %d error(s)", len(result.Errors))
		}

		// Print success message
		if !buildJSON {
			fmt.Printf("\n✓ Build successful in %.2fs\n", result.Duration.Seconds())
			fmt.Printf("  Binary: %s\n", result.OutputPath)
			fmt.Printf("  Metadata: %s\n", result.MetadataPath)
			fmt.Printf("  Files compiled: %d\n", result.FilesCompiled)
			if result.CacheHits > 0 {
				fmt.Printf("  Cache hits: %d\n", result.CacheHits)
			}
		} else {
			outputBuildSuccessJSON(result)
		}

		return nil
	},
}

func outputBuildErrorsJSON(errs []build.BuildError) {
	output := struct {
		Success bool               `json:"success"`
		Errors  []build.BuildError `json:"errors"`
	}{
		Success: false,
		Errors:  errs,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
}

func outputBuildErrorsTerminal(errs []build.BuildError) {
	fmt.Fprintf(os.Stderr, "\nCompilation failed with %d error(s):\n\n", len(errs))

	for i, err := range errs {
		fmt.Fprintf(os.Stderr, "%d. [%s] %s:%d:%d\n",
			i+1, err.Phase, err.File, err.Line, err.Column)
		fmt.Fprintf(os.Stderr, "   %s\n", err.Message)

		if i < len(errs)-1 {
			fmt.Fprintln(os.Stderr, strings.Repeat("-", 60))
		}
	}
	fmt.Fprintln(os.Stderr)
}

func outputBuildSuccessJSON(result *build.BuildResult) {
	output := struct {
		Success       bool    `json:"success"`
		OutputPath    string  `json:"output_path"`
		MetadataPath  string  `json:"metadata_path"`
		Duration      float64 `json:"duration_seconds"`
		FilesCompiled int     `json:"files_compiled"`
		CacheHits     int     `json:"cache_hits"`
	}{
		Success:       true,
		OutputPath:    result.OutputPath,
		MetadataPath:  result.MetadataPath,
		Duration:      result.Duration.Seconds(),
		FilesCompiled: result.FilesCompiled,
		CacheHits:     result.CacheHits,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
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
