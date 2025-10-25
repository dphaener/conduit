package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/conduit-lang/conduit/internal/format"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	formatWrite  bool
	formatCheck  bool
	formatConfig string
)

// NewFormatCommand creates the format command
func NewFormatCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "format [files...]",
		Short: "Format Conduit source files",
		Long: `Format Conduit source files (.cdt) using the configured style rules.

By default, shows a diff preview of what would change without modifying files.
Use --write to apply formatting changes, or --check to verify formatting.

Examples:
  conduit format                    # Show diff for all .cdt files
  conduit format --write            # Format and save all files
  conduit format --check            # Exit with error if not formatted
  conduit format file.cdt           # Format specific file
  conduit format src/*.cdt          # Format files matching pattern`,
		RunE: runFormat,
	}

	cmd.Flags().BoolVarP(&formatWrite, "write", "w", false, "Write formatted output to files")
	cmd.Flags().BoolVarP(&formatCheck, "check", "c", false, "Check if files are formatted (exit 1 if not)")
	cmd.Flags().StringVar(&formatConfig, "config", ".conduit-format.yml", "Path to formatting config file")

	return cmd
}

func runFormat(cmd *cobra.Command, args []string) error {
	// Load config
	config, err := format.LoadConfig(formatConfig)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Find files to format
	files, err := findConduitFiles(args)
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no .cdt files found")
	}

	// Process files
	hasChanges := false
	errorCount := 0

	titleColor := color.New(color.FgCyan, color.Bold)
	successColor := color.New(color.FgGreen)
	errorColor := color.New(color.FgRed, color.Bold)

	for _, file := range files {
		// Read original content
		original, err := os.ReadFile(file)
		if err != nil {
			errorColor.Fprintf(cmd.ErrOrStderr(), "Error reading %s: %v\n", file, err)
			errorCount++
			continue
		}

		// Format
		formatter := format.New(config)
		formatted, err := formatter.Format(string(original))
		if err != nil {
			errorColor.Fprintf(cmd.ErrOrStderr(), "Error formatting %s: %v\n", file, err)
			errorCount++
			continue
		}

		// Check if changed
		diff := format.Diff(string(original), formatted)
		if !diff.Changed {
			if !formatCheck {
				successColor.Fprintf(cmd.OutOrStdout(), "✓ %s (no changes)\n", file)
			}
			continue
		}

		hasChanges = true

		// Handle different modes
		if formatCheck {
			errorColor.Fprintf(cmd.ErrOrStderr(), "✗ %s needs formatting\n", file)
		} else if formatWrite {
			// Write formatted content
			if err := os.WriteFile(file, []byte(formatted), 0644); err != nil {
				errorColor.Fprintf(cmd.ErrOrStderr(), "Error writing %s: %v\n", file, err)
				errorCount++
				continue
			}
			successColor.Fprintf(cmd.OutOrStdout(), "✓ %s formatted\n", file)
		} else {
			// Show diff
			titleColor.Fprintf(cmd.OutOrStdout(), "\n=== %s ===\n", file)
			fmt.Fprintln(cmd.OutOrStdout(), diff.String())
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", diff.Stats())
		}
	}

	// Summary
	if !formatWrite && !formatCheck && hasChanges {
		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		titleColor.Fprintf(cmd.OutOrStdout(), "Run 'conduit format --write' to apply changes\n")
	}

	// Exit with error if in check mode and there are changes
	if formatCheck && hasChanges {
		return fmt.Errorf("files need formatting")
	}

	if errorCount > 0 {
		return fmt.Errorf("%d files had errors", errorCount)
	}

	return nil
}

// findConduitFiles finds all .cdt files to format
func findConduitFiles(patterns []string) ([]string, error) {
	var files []string

	// Get current working directory as base
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// If no patterns provided, find all .cdt files in current directory
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	for _, pattern := range patterns {
		// Resolve to absolute path
		absPattern, err := filepath.Abs(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid path %s: %w", pattern, err)
		}

		// Validate path is within or equal to cwd
		relPath, err := filepath.Rel(cwd, absPattern)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return nil, fmt.Errorf("path %s is outside working directory", pattern)
		}

		// Check if it's a directory
		info, err := os.Stat(absPattern)
		if err == nil && info.IsDir() {
			// Walk directory to find .cdt files
			err := filepath.Walk(absPattern, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Skip hidden directories and build directories
				if info.IsDir() && (strings.HasPrefix(info.Name(), ".") || info.Name() == "build" || info.Name() == "node_modules") {
					return filepath.SkipDir
				}

				// Add .cdt files
				if !info.IsDir() && strings.HasSuffix(path, ".cdt") {
					files = append(files, path)
				}

				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			// It's a file or pattern
			matches, err := filepath.Glob(absPattern)
			if err != nil {
				return nil, err
			}

			// Filter for .cdt files and validate each match
			for _, match := range matches {
				// Validate match is within cwd
				absMatch, err := filepath.Abs(match)
				if err != nil {
					continue
				}
				relMatch, err := filepath.Rel(cwd, absMatch)
				if err != nil || strings.HasPrefix(relMatch, "..") {
					continue
				}

				if strings.HasSuffix(match, ".cdt") {
					files = append(files, match)
				}
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := []string{}
	for _, file := range files {
		if !seen[file] {
			seen[file] = true
			unique = append(unique, file)
		}
	}

	return unique, nil
}
