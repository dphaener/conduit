package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/conduit-lang/conduit/internal/templates"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	templateValidateVerbose bool
)

// NewTemplateCommand creates the template command
func NewTemplateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage project templates",
		Long: `Manage Conduit project templates.

Templates provide scaffolding for different types of projects:
- api: RESTful API with resources
- web: Full-stack web application
- microservice: Event-driven microservice

Examples:
  conduit template list
  conduit template validate api
  conduit new my-project --template api`,
	}

	cmd.AddCommand(NewTemplateListCommand())
	cmd.AddCommand(NewTemplateValidateCommand())

	return cmd
}

// NewTemplateListCommand creates the template list command
func NewTemplateListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available project templates",
		Long:  `Display all available project templates with their descriptions.`,
		RunE:  runTemplateList,
	}

	return cmd
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	// Initialize built-in templates
	if err := templates.RegisterBuiltinTemplates(); err != nil {
		return fmt.Errorf("failed to register templates: %w", err)
	}

	registry := templates.DefaultRegistry()
	tmplList := registry.List()

	if len(tmplList) == 0 {
		fmt.Println("No templates available")
		return nil
	}

	successColor := color.New(color.FgGreen, color.Bold)
	infoColor := color.New(color.FgCyan)
	metaColor := color.New(color.FgYellow)

	fmt.Println()
	successColor.Println("Available Templates:")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION")
	fmt.Fprintln(w, "----\t-------\t-----------")

	for _, tmpl := range tmplList {
		fmt.Fprintf(w, "%s\t%s\t%s\n", tmpl.Name, tmpl.Version, tmpl.Description)
	}
	w.Flush()

	fmt.Println()
	infoColor.Println("Template Details:")
	fmt.Println()

	for _, tmpl := range tmplList {
		fmt.Printf("• %s\n", color.New(color.Bold).Sprint(tmpl.Name))
		fmt.Printf("  Description: %s\n", tmpl.Description)
		fmt.Printf("  Version: %s\n", tmpl.Version)

		if len(tmpl.Variables) > 0 {
			fmt.Printf("  Variables:\n")
			for _, v := range tmpl.Variables {
				required := ""
				if v.Required {
					required = " (required)"
				}
				fmt.Printf("    - %s: %s%s\n", v.Name, v.Description, required)
			}
		}

		if tmpl.Metadata != nil {
			if category, ok := tmpl.Metadata["category"].(string); ok {
				fmt.Printf("  Category: %s\n", category)
			}
			if tags, ok := tmpl.Metadata["tags"].([]string); ok && len(tags) > 0 {
				fmt.Printf("  Tags: %v\n", tags)
			}
		}
		fmt.Println()
	}

	metaColor.Println("Use 'conduit new <project-name> --template <name>' to create a project from a template")
	fmt.Println()

	return nil
}

// NewTemplateValidateCommand creates the template validate command
func NewTemplateValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [template-name]",
		Short: "Validate a template",
		Long:  `Validate the structure and configuration of a template.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runTemplateValidate,
	}

	cmd.Flags().BoolVarP(&templateValidateVerbose, "verbose", "v", false, "Show detailed validation output")

	return cmd
}

func runTemplateValidate(cmd *cobra.Command, args []string) error {
	templateName := args[0]

	// Initialize built-in templates
	if err := templates.RegisterBuiltinTemplates(); err != nil {
		return fmt.Errorf("failed to register templates: %w", err)
	}

	registry := templates.DefaultRegistry()
	tmpl, err := registry.Get(templateName)
	if err != nil {
		return fmt.Errorf("template not found: %s", templateName)
	}

	successColor := color.New(color.FgGreen, color.Bold)
	errorColor := color.New(color.FgRed, color.Bold)
	infoColor := color.New(color.FgCyan)

	if templateValidateVerbose {
		infoColor.Printf("Validating template: %s\n\n", templateName)
	}

	// Validate template
	if err := tmpl.Validate(); err != nil {
		errorColor.Printf("✗ Template validation failed: %v\n", err)
		return err
	}

	successColor.Printf("✓ Template '%s' is valid\n\n", templateName)

	if templateValidateVerbose {
		fmt.Printf("Template Details:\n")
		fmt.Printf("  Name: %s\n", tmpl.Name)
		fmt.Printf("  Description: %s\n", tmpl.Description)
		fmt.Printf("  Version: %s\n", tmpl.Version)
		fmt.Printf("  Variables: %d\n", len(tmpl.Variables))
		fmt.Printf("  Files: %d\n", len(tmpl.Files))
		fmt.Printf("  Directories: %d\n", len(tmpl.Directories))

		if len(tmpl.Variables) > 0 {
			fmt.Println("\nVariables:")
			for _, v := range tmpl.Variables {
				requiredStr := ""
				if v.Required {
					requiredStr = " (required)"
				}
				defaultStr := ""
				if v.Default != nil {
					defaultStr = fmt.Sprintf(" [default: %v]", v.Default)
				}
				fmt.Printf("  • %s (%s)%s%s\n", v.Name, v.Type, requiredStr, defaultStr)
				if v.Description != "" {
					fmt.Printf("    %s\n", v.Description)
				}
				if len(v.Options) > 0 {
					fmt.Printf("    Options: %v\n", v.Options)
				}
			}
		}

		if len(tmpl.Files) > 0 {
			fmt.Println("\nFiles:")
			for _, f := range tmpl.Files {
				templateStr := ""
				if f.Template {
					templateStr = " (templated)"
				}
				conditionStr := ""
				if f.Condition != "" {
					conditionStr = fmt.Sprintf(" [conditional: %s]", f.Condition)
				}
				fmt.Printf("  • %s%s%s\n", f.TargetPath, templateStr, conditionStr)
			}
		}

		if len(tmpl.Directories) > 0 {
			fmt.Println("\nDirectories:")
			for _, dir := range tmpl.Directories {
				fmt.Printf("  • %s\n", dir)
			}
		}

		if tmpl.Hooks != nil {
			if len(tmpl.Hooks.BeforeCreate) > 0 {
				fmt.Println("\nBefore Create Hooks:")
				for _, hook := range tmpl.Hooks.BeforeCreate {
					fmt.Printf("  • %s\n", hook)
				}
			}
			if len(tmpl.Hooks.AfterCreate) > 0 {
				fmt.Println("\nAfter Create Hooks:")
				for _, hook := range tmpl.Hooks.AfterCreate {
					fmt.Printf("  • %s\n", hook)
				}
			}
		}

		fmt.Println()
	}

	return nil
}
