package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

//go:embed templates/*
var templatesFS embed.FS

var newCmd = &cobra.Command{
	Use:   "new <project-name>",
	Short: "Create a new Conduit project",
	Long:  "Create a new Conduit project with directory structure and sample files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		// Validate project name
		if projectName == "" || strings.TrimSpace(projectName) == "" {
			return fmt.Errorf("project name cannot be empty")
		}
		if strings.Contains(projectName, "..") {
			return fmt.Errorf("project name cannot contain '..'")
		}
		if strings.Contains(projectName, "/") || strings.Contains(projectName, "\\") {
			return fmt.Errorf("project name cannot contain path separators")
		}
		if strings.HasPrefix(projectName, ".") {
			return fmt.Errorf("project name cannot start with '.'")
		}

		// Create project directory
		projectPath := filepath.Join(".", projectName)
		if _, err := os.Stat(projectPath); err == nil {
			return fmt.Errorf("directory %s already exists", projectName)
		}

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
		data := map[string]string{
			"ProjectName": projectName,
		}

		// Create files from templates
		files := map[string]string{
			"app/main.cdt":    "templates/app.cdt.tmpl",
			".gitignore":      "templates/gitignore.tmpl",
			"conduit.yaml":    "templates/config.tmpl",
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
		}

		// Create README
		readmePath := filepath.Join(projectPath, "README.md")
		readmeContent := fmt.Sprintf(`# %s

A Conduit web application.

## Getting Started

1. Set up your database:
   ` + "```bash" + `
   export DATABASE_URL="postgresql://user:password@localhost:5432/%s"
   ` + "```" + `

2. Run migrations:
   ` + "```bash" + `
   conduit migrate up
   ` + "```" + `

3. Build and run your application:
   ` + "```bash" + `
   conduit run
   ` + "```" + `

Your API will be available at http://localhost:3000

## Project Structure

- ` + "`app/`" + ` - Conduit source files (` + "`.cdt`" + `)
- ` + "`migrations/`" + ` - Database migration SQL files
- ` + "`build/`" + ` - Compiled output (auto-generated)
- ` + "`conduit.yaml`" + ` - Project configuration

## Documentation

Learn more at https://conduit-lang.org/docs
`, projectName, projectName)

		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return fmt.Errorf("failed to create README: %w", err)
		}

		// Print success message
		fmt.Printf("\n✓ Created project: %s\n\n", projectName)
		fmt.Println("Get started:")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
		fmt.Println("  conduit migrate up")
		fmt.Println("  conduit run")
		fmt.Println()

		return nil
	},
}
