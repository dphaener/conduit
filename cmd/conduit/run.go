package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/internal/tooling/build"
)

var (
	runPort             int
	requireMigrations   bool
	autoMigrate         string
)

func init() {
	runCmd.Flags().IntVar(&runPort, "port", 3000, "Port to run the server on")
	runCmd.Flags().BoolVar(&requireMigrations, "require-migrations", false, "Block startup if migrations are pending")
	runCmd.Flags().StringVar(&autoMigrate, "auto-migrate", "", "Automatically apply migrations before startup (use 'dry-run' to preview)")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Build and run the application",
	Long:  "Build the Conduit application and start the server",
	RunE: func(cmd *cobra.Command, args []string) error {
		startTime := time.Now()

		// Check if app directory exists
		if _, err := os.Stat("app"); os.IsNotExist(err) {
			return fmt.Errorf("app/ directory not found - are you in a Conduit project?")
		}

		// Configure build options
		opts := &build.BuildOptions{
			Mode:       build.ModeDevelopment,
			OutputPath: "build/app",
			SourceDir:  "app",
			BuildDir:   "build",
			Parallel:   true,
			MaxJobs:    runtime.NumCPU(),
			Verbose:    buildVerbose,
			UseCache:   !buildNoCache,
			Minify:     buildMinify,
			TreeShake:  buildTreeShake,
		}

		// Create build system
		sys, err := build.NewSystem(opts)
		if err != nil {
			return fmt.Errorf("failed to create build system: %w", err)
		}

		// Load build state
		state, err := build.LoadState(opts.BuildDir)
		if err != nil {
			// Non-fatal - just do a full build
			if buildVerbose {
				fmt.Printf("Warning: failed to load build state: %v\n", err)
			}
			state = &build.BuildState{
				FileHashes: make(map[string]string),
			}
		}

		// Find source files
		sourceFiles, err := findSourceFiles(opts.SourceDir)
		if err != nil {
			return fmt.Errorf("failed to find source files: %w", err)
		}

		if len(sourceFiles) == 0 {
			return fmt.Errorf("no .cdt files found in %s", opts.SourceDir)
		}

		// Check if rebuild is needed
		needsRebuild, changedFiles, reason := state.NeedsRebuild(sourceFiles, opts)

		var result *build.BuildResult
		ctx := context.Background()

		if !needsRebuild {
			// Use cached binary
			fmt.Printf("✓ Using cached binary (<%dms)\n", time.Since(startTime).Milliseconds())
		} else {
			// Determine build strategy
			if len(changedFiles) > 0 && len(changedFiles) < len(sourceFiles) {
				// Incremental build - only some files changed
				fmt.Printf("🔨 Rebuilding (%s: %d/%d files)...\n", reason, len(changedFiles), len(sourceFiles))

				result, err = sys.IncrementalBuild(ctx, changedFiles)
			} else {
				// Full build needed
				fmt.Printf("🔨 Building (%s)...\n", reason)

				result, err = sys.Build(ctx)
			}

			if err != nil {
				return fmt.Errorf("build failed: %w", err)
			}

			// Check for compilation errors
			if !result.Success {
				fmt.Fprintf(os.Stderr, "\nCompilation failed with %d error(s):\n\n", len(result.Errors))
				for i, err := range result.Errors {
					fmt.Fprintf(os.Stderr, "%d. [%s] %s:%d:%d\n",
						i+1, err.Phase, err.File, err.Line, err.Column)
					fmt.Fprintf(os.Stderr, "   %s\n", err.Message)
				}
				return fmt.Errorf("compilation failed")
			}

			// Update build state
			if err := state.UpdateFromBuild(sourceFiles, opts); err != nil {
				// Non-fatal - just log warning
				if buildVerbose {
					fmt.Printf("Warning: failed to update build state: %v\n", err)
				}
			} else {
				// Save build state
				if err := state.SaveState(opts.BuildDir); err != nil {
					// Non-fatal - just log warning
					if buildVerbose {
						fmt.Printf("Warning: failed to save build state: %v\n", err)
					}
				}
			}

			// Print build summary
			fmt.Printf("✓ Build completed in %.2fs\n", result.Duration.Seconds())
			if buildVerbose {
				fmt.Printf("  Files compiled: %d\n", result.FilesCompiled)
				if result.CacheHits > 0 {
					fmt.Printf("  Cache hits: %d\n", result.CacheHits)
				}
			}
		}

		// Check if binary exists
		if _, err := os.Stat(opts.OutputPath); os.IsNotExist(err) {
			return fmt.Errorf("%s not found - build may have failed", opts.OutputPath)
		}

		// Handle auto-migrate if requested
		if autoMigrate != "" {
			mode := build.AutoMigrateApply
			if autoMigrate == "dry-run" {
				mode = build.AutoMigrateDryRun
			} else if autoMigrate != "true" && autoMigrate != "1" {
				return fmt.Errorf("invalid --auto-migrate value: %s (use 'dry-run' or leave empty)", autoMigrate)
			}

			migrator := build.NewAutoMigrator(build.AutoMigrateOptions{
				Mode: mode,
			})

			if err := migrator.Run(); err != nil {
				return fmt.Errorf("auto-migrate failed: %w", err)
			}

			// If dry-run, don't start the server
			if mode == build.AutoMigrateDryRun {
				return nil
			}
		}

		// Check migration status before starting server
		migrationStatus, err := build.CheckMigrationStatus()
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		// Print warning if there are pending migrations
		if len(migrationStatus.Pending) > 0 {
			build.PrintMigrationWarning(migrationStatus)

			// Block startup if --require-migrations flag is set
			if requireMigrations {
				return fmt.Errorf("startup blocked: %d pending migration(s) detected (use --require-migrations=false to override)", len(migrationStatus.Pending))
			}
		}

		// Run the application
		fmt.Printf("\nStarting server on port %d...\n", runPort)

		app := exec.Command("./build/app")
		app.Stdout = os.Stdout
		app.Stderr = os.Stderr
		app.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", runPort))

		// Start the application
		if err := app.Start(); err != nil {
			return fmt.Errorf("failed to start application: %w", err)
		}

		// Handle Ctrl+C gracefully
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Println("\nShutting down server...")
			if app.Process != nil {
				app.Process.Kill()
			}
			os.Exit(0)
		}()

		// Wait for the application to finish
		if err := app.Wait(); err != nil {
			// Check if it was killed by signal
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == -1 {
					// Process was killed by signal, this is expected
					return nil
				}
			}
			return fmt.Errorf("application exited with error: %w", err)
		}

		return nil
	},
}

// findSourceFiles finds all .cdt files in the source directory
func findSourceFiles(sourceDir string) ([]string, error) {
	// This is a simplified version - in a real implementation,
	// you'd want to use the same function as the build system
	return findCdtFilesRecursive(sourceDir)
}

// findCdtFilesRecursive finds all .cdt files recursively
func findCdtFilesRecursive(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		path := dir + "/" + entry.Name()

		if entry.IsDir() {
			subFiles, err := findCdtFilesRecursive(path)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else if len(entry.Name()) > 4 && entry.Name()[len(entry.Name())-4:] == ".cdt" {
			// Convert to absolute path
			absPath, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			files = append(files, absPath+"/"+path)
		}
	}

	return files, nil
}
