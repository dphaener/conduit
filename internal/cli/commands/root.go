package commands

import (
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Version information - set at build time
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = "unknown"
)

// NewRootCommand creates the root command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "conduit",
		Short: "Conduit programming language compiler and tooling",
		Long: color.CyanString(`Conduit - LLM-First Programming Language

Conduit is a programming language for building web applications.
It compiles to Go and provides explicit syntax optimized for AI-assisted development.

Features:
  • Explicit nullability (type! vs type?)
  • Namespaced standard library
  • Built-in ORM and web framework
  • Single binary deployment
  • Sub-second compilation`),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add subcommands
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewNewCommand())
	rootCmd.AddCommand(NewBuildCommand())
	rootCmd.AddCommand(NewRunCommand())
	rootCmd.AddCommand(NewWatchCommand())
	rootCmd.AddCommand(NewMigrateCommand())
	rootCmd.AddCommand(NewGenerateCommand())
	rootCmd.AddCommand(NewLSPCommand())
	rootCmd.AddCommand(NewDebugCommand())
	rootCmd.AddCommand(NewFormatCommand())
	rootCmd.AddCommand(NewTemplateCommand())

	return rootCmd
}

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display the Conduit compiler version, Git commit, build date, and Go version",
		Run: func(cmd *cobra.Command, args []string) {
			// Set GoVersion to actual runtime if not set at build time
			goVer := GoVersion
			if goVer == "unknown" {
				goVer = runtime.Version()
			}

			titleColor := color.New(color.FgCyan, color.Bold)
			valueColor := color.New(color.FgWhite)

			titleColor.Print("Conduit version: ")
			valueColor.Println(Version)

			titleColor.Print("Git commit: ")
			valueColor.Println(GitCommit)

			titleColor.Print("Build date: ")
			valueColor.Println(BuildDate)

			titleColor.Print("Go version: ")
			valueColor.Println(goVer)
		},
	}
}

// Execute runs the root command
func Execute() error {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		errorColor := color.New(color.FgRed, color.Bold)
		errorColor.Fprintf(rootCmd.ErrOrStderr(), "Error: %v\n", err)
		return err
	}
	return nil
}
