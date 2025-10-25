package commands

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

//go:embed templates/*
var templatesFS embed.FS

var (
	newInteractive bool
	newDatabase    string
	newPort        int
)

// validateProjectName validates project name with security checks
func validateProjectName(name string) error {
	name = strings.TrimSpace(name)

	// Check length
	if len(name) == 0 || len(name) > 100 {
		return fmt.Errorf("project name must be 1-100 characters")
	}

	// Check for absolute paths
	if filepath.IsAbs(name) {
		return fmt.Errorf("project name cannot be an absolute path")
	}

	// Only allow alphanumeric, dash, and underscore
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	if !matched {
		return fmt.Errorf("project name can only contain letters, numbers, dashes, and underscores")
	}

	// Additional security checks
	if strings.Contains(name, "..") {
		return fmt.Errorf("project name cannot contain '..'")
	}

	return nil
}

// NewNewCommand creates the new command
func NewNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new Conduit project",
		Long: `Create a new Conduit project with directory structure and sample files.

If no project name is provided, you will be prompted to enter one.

Examples:
  conduit new my-blog
  conduit new my-api --database postgres
  conduit new --interactive`,
		RunE: runNew,
	}

	cmd.Flags().BoolVarP(&newInteractive, "interactive", "i", false, "Interactive project setup with prompts")
	cmd.Flags().StringVar(&newDatabase, "database", "postgresql", "Database type (postgresql)")
	cmd.Flags().IntVar(&newPort, "port", 3000, "Default server port")

	return cmd
}

func runNew(cmd *cobra.Command, args []string) error {
	var projectName string
	var dbURL string

	successColor := color.New(color.FgGreen, color.Bold)
	infoColor := color.New(color.FgCyan)
	promptColor := color.New(color.FgYellow)

	// Get project name from args or prompt
	if len(args) > 0 {
		projectName = args[0]
	} else if !newInteractive {
		// Prompt for project name if not provided and not in interactive mode
		prompt := &survey.Input{
			Message: "Project name:",
		}
		if err := survey.AskOne(prompt, &projectName, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// Interactive mode
	if newInteractive || len(args) == 0 {
		questions := []*survey.Question{
			{
				Name: "projectName",
				Prompt: &survey.Input{
					Message: "Project name:",
					Default: projectName,
				},
				Validate: survey.Required,
			},
			{
				Name: "database",
				Prompt: &survey.Select{
					Message: "Database:",
					Options: []string{"PostgreSQL", "MySQL (coming soon)", "SQLite (coming soon)"},
					Default: "PostgreSQL",
				},
			},
			{
				Name: "port",
				Prompt: &survey.Input{
					Message: "Server port:",
					Default: "3000",
				},
			},
			{
				Name: "dbURL",
				Prompt: &survey.Input{
					Message: "Database URL (optional):",
					Default: "",
					Help:    "Leave empty to set via DATABASE_URL environment variable",
				},
			},
		}

		answers := struct {
			ProjectName string
			Database    string
			Port        string
			DbURL       string
		}{}

		if err := survey.Ask(questions, &answers); err != nil {
			return err
		}

		projectName = answers.ProjectName
		dbURL = answers.DbURL

		// Parse port
		fmt.Sscanf(answers.Port, "%d", &newPort)
	}

	// Validate project name with security checks
	if err := validateProjectName(projectName); err != nil {
		return err
	}

	// Create project directory
	projectPath := filepath.Join(".", projectName)
	if _, err := os.Stat(projectPath); err == nil {
		return fmt.Errorf("directory %s already exists", projectName)
	}

	infoColor.Printf("Creating project: %s\n\n", projectName)

	// Create directory structure
	dirs := []string{
		projectPath,
		filepath.Join(projectPath, "app"),
		filepath.Join(projectPath, "migrations"),
		filepath.Join(projectPath, "build"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Template data
	data := map[string]interface{}{
		"ProjectName": projectName,
		"Port":        newPort,
		"DatabaseURL": dbURL,
	}

	// Create files from templates
	files := map[string]string{
		"app/main.cdt": "templates/app.cdt.tmpl",
		".gitignore":   "templates/gitignore.tmpl",
		"conduit.yaml": "templates/config.tmpl",
	}

	for destPath, tmplPath := range files {
		destFullPath := filepath.Join(projectPath, destPath)

		// Read template
		tmplContent, err := templatesFS.ReadFile(tmplPath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tmplPath, err)
		}

		// Parse and execute template
		tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(tmplContent))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tmplPath, err)
		}

		// Create destination file
		f, err := os.Create(destFullPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", destFullPath, err)
		}

		if err := tmpl.Execute(f, data); err != nil {
			f.Close()
			return fmt.Errorf("failed to execute template %s: %w", tmplPath, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close file %s: %w", destFullPath, err)
		}

		infoColor.Printf("  ✓ Created %s\n", destPath)
	}

	// Create README
	readmePath := filepath.Join(projectPath, "README.md")
	readmeContent := fmt.Sprintf(`# %s

A Conduit web application.

## Getting Started

1. Set up your database:
   `+"`"+`bash
   export DATABASE_URL="postgresql://user:password@localhost:5432/%s"
   `+"`"+`

2. Run migrations:
   `+"`"+`bash
   conduit migrate up
   `+"`"+`

3. Build and run your application:
   `+"`"+`bash
   conduit run
   `+"`"+`

Your API will be available at http://localhost:%d

## Project Structure

- `+"`app/`"+` - Conduit source files (`+"`\\.cdt`"+`)
- `+"`migrations/`"+` - Database migration SQL files
- `+"`build/`"+` - Compiled output (auto-generated)
- `+"`conduit.yaml`"+` - Project configuration

## Documentation

Learn more at https://conduit-lang.org/docs
`, projectName, projectName, newPort)

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	infoColor.Println("  ✓ Created README.md")

	// Print success message
	fmt.Println()
	successColor.Printf("✓ Created project: %s\n\n", projectName)

	promptColor.Println("Get started:")
	fmt.Printf("  cd %s\n", projectName)
	if dbURL == "" {
		fmt.Println("  export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
	}
	fmt.Println("  conduit migrate up")
	fmt.Println("  conduit run")
	fmt.Println()

	return nil
}
