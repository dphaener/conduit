package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/templates"
)

func TestTemplateListCommand(t *testing.T) {
	// Create a fresh registry for this test
	oldRegistry := templates.DefaultRegistry()
	defer func() {
		// Restore old registry after test
		templates.SetDefaultRegistry(oldRegistry)
	}()

	freshRegistry := templates.NewRegistry()
	templates.SetDefaultRegistry(freshRegistry)

	cmd := NewTemplateListCommand()

	// Capture output
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("template list command failed: %v", err)
	}

	// Just verify the command ran without error
	// (output goes to stdout which isn't captured in tests)
}

func TestTemplateValidateCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid template - api",
			args:    []string{"api"},
			wantErr: false,
		},
		{
			name:    "valid template - web",
			args:    []string{"web"},
			wantErr: false,
		},
		{
			name:    "valid template - microservice",
			args:    []string{"microservice"},
			wantErr: false,
		},
		{
			name:        "non-existent template",
			args:        []string{"non-existent"},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "no args",
			args:        []string{},
			wantErr:     true,
			errContains: "accepts 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh registry for each test
			oldRegistry := templates.DefaultRegistry()
			defer templates.SetDefaultRegistry(oldRegistry)

			freshRegistry := templates.NewRegistry()
			templates.SetDefaultRegistry(freshRegistry)

			cmd := NewTemplateValidateCommand()
			cmd.SetArgs(tt.args)

			output := &bytes.Buffer{}
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate command error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain %q, got %v", tt.errContains, err)
				}
			}

			// For successful cases, just verify no error
			// (output goes to stdout which isn't captured in tests)
		})
	}
}

func TestTemplateValidateVerbose(t *testing.T) {
	// Create a fresh registry
	oldRegistry := templates.DefaultRegistry()
	defer templates.SetDefaultRegistry(oldRegistry)

	freshRegistry := templates.NewRegistry()
	templates.SetDefaultRegistry(freshRegistry)

	cmd := NewTemplateValidateCommand()
	cmd.SetArgs([]string{"api", "--verbose"})

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Validate command failed: %v", err)
	}

	// Just verify the command ran without error
	// (verbose output goes to stdout which isn't easily captured in tests)
}

func TestTemplateCommandHelp(t *testing.T) {
	cmd := NewTemplateCommand()
	cmd.SetArgs([]string{"--help"})

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	outputStr := output.String()

	// Verify help text contains important information
	expectedContent := []string{
		"template",
		"list",
		"validate",
	}

	for _, content := range expectedContent {
		if !strings.Contains(outputStr, content) {
			t.Errorf("Help output missing: %s", content)
		}
	}
}

func TestBuiltinTemplatesIntegrity(t *testing.T) {
	// Create a fresh registry
	oldRegistry := templates.DefaultRegistry()
	defer templates.SetDefaultRegistry(oldRegistry)

	freshRegistry := templates.NewRegistry()
	templates.SetDefaultRegistry(freshRegistry)

	// Ensure built-in templates are registered
	err := templates.RegisterBuiltinTemplates()
	if err != nil {
		t.Fatalf("Failed to register built-in templates: %v", err)
	}

	registry := templates.DefaultRegistry()

	// Test each built-in template
	templateNames := []string{"api", "web", "microservice"}

	for _, name := range templateNames {
		t.Run(name, func(t *testing.T) {
			tmpl, err := registry.Get(name)
			if err != nil {
				t.Fatalf("Failed to get template %s: %v", name, err)
			}

			// Validate template structure
			if err := tmpl.Validate(); err != nil {
				t.Errorf("Template %s is invalid: %v", name, err)
			}

			// Check metadata
			if tmpl.Name != name {
				t.Errorf("Template name mismatch: got %s, want %s", tmpl.Name, name)
			}

			if tmpl.Version == "" {
				t.Error("Template version is empty")
			}

			if tmpl.Description == "" {
				t.Error("Template description is empty")
			}

			// Check variables
			hasProjectName := false
			for _, v := range tmpl.Variables {
				if v.Name == "project_name" {
					hasProjectName = true
					if !v.Required {
						t.Error("project_name should be required")
					}
				}

				// Verify variable has valid type
				validTypes := []templates.VariableType{
					templates.VariableTypeString,
					templates.VariableTypeInt,
					templates.VariableTypeBool,
					templates.VariableTypeConfirm,
					templates.VariableTypeSelect,
				}

				validType := false
				for _, vt := range validTypes {
					if v.Type == vt {
						validType = true
						break
					}
				}

				if !validType {
					t.Errorf("Variable %s has invalid type: %s", v.Name, v.Type)
				}
			}

			if !hasProjectName {
				t.Error("Template missing project_name variable")
			}

			// Check files
			if len(tmpl.Files) == 0 {
				t.Error("Template has no files")
			}

			hasReadme := false
			hasGitignore := false
			hasConfig := false

			for _, f := range tmpl.Files {
				if strings.Contains(f.TargetPath, "README") {
					hasReadme = true
				}
				if strings.Contains(f.TargetPath, ".gitignore") {
					hasGitignore = true
				}
				if strings.Contains(f.TargetPath, "conduit.yaml") {
					hasConfig = true
				}

				// Verify file has content
				if f.Content == "" {
					t.Errorf("File %s has no content", f.TargetPath)
				}
			}

			if !hasReadme {
				t.Error("Template missing README file")
			}

			if !hasGitignore {
				t.Error("Template missing .gitignore file")
			}

			if !hasConfig {
				t.Error("Template missing conduit.yaml config")
			}

			// Check directories
			if len(tmpl.Directories) == 0 {
				t.Error("Template has no directories")
			}
		})
	}
}
