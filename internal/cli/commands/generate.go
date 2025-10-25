package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewGenerateCommand creates the generate command
func NewGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"g"},
		Short:   "Code generation commands",
		Long: `Generate boilerplate code for resources, controllers, and migrations.

Available generators:
  resource   - Generate a new resource definition
  controller - Generate a controller (stub)
  migration  - Generate a database migration

Examples:
  conduit generate resource User
  conduit generate migration create_users
  conduit g resource Post`,
	}

	cmd.AddCommand(newGenerateResourceCommand())
	cmd.AddCommand(newGenerateControllerCommand())
	cmd.AddCommand(newGenerateMigrationCommand())

	return cmd
}

func newGenerateResourceCommand() *cobra.Command {
	var interactive bool

	cmd := &cobra.Command{
		Use:   "resource [name]",
		Short: "Generate a new resource",
		Long: `Generate a new resource definition in the app/ directory.

This is a stub implementation. Actual resource generation will be
implemented in the tooling milestone.

Examples:
  conduit generate resource User
  conduit generate resource Post --interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			successColor := color.New(color.FgGreen, color.Bold)
			infoColor := color.New(color.FgCyan)
			warningColor := color.New(color.FgYellow)

			var resourceName string

			if len(args) > 0 {
				resourceName = args[0]
			} else if interactive {
				prompt := &survey.Input{
					Message: "Resource name (singular, CamelCase):",
				}
				if err := survey.AskOne(prompt, &resourceName, survey.WithValidator(survey.Required)); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("resource name required\n\nUsage: conduit generate resource <name>")
			}

			// Validate resource name
			if resourceName == "" {
				return fmt.Errorf("resource name cannot be empty")
			}

			// Check if app directory exists
			if _, err := os.Stat("app"); os.IsNotExist(err) {
				return fmt.Errorf("app/ directory not found - are you in a Conduit project?")
			}

			filename := fmt.Sprintf("app/%s.cdt", strings.ToLower(resourceName))

			// Check if file already exists
			if _, err := os.Stat(filename); err == nil {
				return fmt.Errorf("file %s already exists", filename)
			}

			warningColor.Println("\nNote: Full resource generation is not yet implemented (coming in tooling milestone)")
			infoColor.Println("Creating basic resource template...")

			// Create basic resource template (stub)
			content := fmt.Sprintf(`/// %s resource
resource %s {
  id: uuid! @primary @auto
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Add your fields here
}
`, resourceName, resourceName)

			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			successColor.Printf("✓ Created %s\n", filename)
			infoColor.Println("\nNext steps:")
			fmt.Println("  1. Add fields to your resource")
			fmt.Println("  2. Run 'conduit build' to compile")
			fmt.Println("  3. Generate migration with 'conduit generate migration create_" + strings.ToLower(resourceName) + "s'")
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive resource generation")

	return cmd
}

func newGenerateControllerCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "controller [name]",
		Short: "Generate a controller (stub)",
		Long: `Generate a controller (stub implementation).

This is a placeholder for future controller generation functionality
that will be implemented in the tooling milestone.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			warningColor := color.New(color.FgYellow, color.Bold)
			infoColor := color.New(color.FgCyan)

			warningColor.Println("\nController generation is not yet implemented.")
			infoColor.Println("This feature will be added in the tooling milestone.")
			infoColor.Println("Conduit auto-generates REST controllers from resources.")
			infoColor.Println("Custom controllers will be supported in a future release.")

			return nil
		},
	}
}

func newGenerateMigrationCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migration [name]",
		Short: "Generate a database migration",
		Long: `Generate up and down migration SQL files in the migrations/ directory.

Migration files are named with a timestamp prefix:
  {timestamp}_{name}.up.sql
  {timestamp}_{name}.down.sql

Examples:
  conduit generate migration create_users
  conduit generate migration add_email_to_users`,
		RunE: func(cmd *cobra.Command, args []string) error {
			successColor := color.New(color.FgGreen, color.Bold)
			infoColor := color.New(color.FgCyan)

			if len(args) == 0 {
				return fmt.Errorf("migration name required\n\nUsage: conduit generate migration <name>")
			}

			migrationName := args[0]

			// Validate migration name
			if migrationName == "" {
				return fmt.Errorf("migration name cannot be empty")
			}

			// Check if migrations directory exists, create if not
			if err := os.MkdirAll("migrations", 0755); err != nil {
				return fmt.Errorf("failed to create migrations directory: %w", err)
			}

			// Generate timestamp-based version
			version := time.Now().Unix()

			// Sanitize migration name (replace spaces with underscores, lowercase)
			sanitizedName := strings.ToLower(strings.ReplaceAll(migrationName, " ", "_"))
			sanitizedName = strings.ReplaceAll(sanitizedName, "-", "_")

			// Generate filenames
			upFile := filepath.Join("migrations", fmt.Sprintf("%d_%s.up.sql", version, sanitizedName))
			downFile := filepath.Join("migrations", fmt.Sprintf("%d_%s.down.sql", version, sanitizedName))

			// Create up migration file
			upContent := fmt.Sprintf(`-- Migration: %s
-- Created: %s
--
-- Add your SQL here to create/modify database schema

`, migrationName, time.Now().Format("2006-01-02 15:04:05"))

			if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
				return fmt.Errorf("failed to write up migration: %w", err)
			}

			// Create down migration file
			downContent := fmt.Sprintf(`-- Rollback migration: %s
-- Created: %s
--
-- Add your SQL here to rollback the changes from the up migration

`, migrationName, time.Now().Format("2006-01-02 15:04:05"))

			if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
				return fmt.Errorf("failed to write down migration: %w", err)
			}

			successColor.Println("✓ Generated migration files:")
			infoColor.Printf("  %s\n", upFile)
			infoColor.Printf("  %s\n", downFile)
			fmt.Println()
			infoColor.Println("Next steps:")
			fmt.Println("  1. Edit the migration files to add your SQL")
			fmt.Println("  2. Run 'conduit migrate up' to apply")
			fmt.Println()

			return nil
		},
	}
}
