package docs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestNewGenerator(t *testing.T) {
	config := &Config{
		ProjectName:    "TestProject",
		ProjectVersion: "1.0.0",
		OutputDir:      "/tmp/test",
		Formats:        []Format{FormatHTML, FormatMarkdown, FormatOpenAPI},
	}

	gen, err := NewGenerator(config)
	if err != nil {
		t.Fatalf("NewGenerator returned error: %v", err)
	}
	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}

	if gen.config != config {
		t.Error("Generator config not set correctly")
	}

	if gen.extractor == nil {
		t.Error("Generator extractor not initialized")
	}
}

func TestGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		ProjectName:        "TestAPI",
		ProjectVersion:     "1.0.0",
		ProjectDescription: "Test API",
		OutputDir:          tmpDir,
		Formats:            []Format{FormatHTML, FormatMarkdown, FormatOpenAPI},
		BaseURL:            "https://api.example.com",
	}

	generator, err := NewGenerator(config)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	program := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name:          "User",
				Documentation: "User resource",
				Fields: []*ast.FieldNode{
					{
						Name:     "id",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
						Nullable: false,
					},
					{
						Name:     "email",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "email", Nullable: false},
						Nullable: false,
						Constraints: []*ast.ConstraintNode{
							{Name: "unique"},
						},
					},
				},
			},
		},
	}

	err = generator.Generate(program)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify HTML was generated
	htmlIndex := filepath.Join(tmpDir, "html", "index.html")
	if _, err := os.Stat(htmlIndex); os.IsNotExist(err) {
		t.Error("HTML documentation not generated")
	}

	// Verify Markdown was generated
	mdReadme := filepath.Join(tmpDir, "markdown", "README.md")
	if _, err := os.Stat(mdReadme); os.IsNotExist(err) {
		t.Error("Markdown documentation not generated")
	}

	// Verify OpenAPI was generated
	openAPISpec := filepath.Join(tmpDir, "openapi.json")
	if _, err := os.Stat(openAPISpec); os.IsNotExist(err) {
		t.Error("OpenAPI spec not generated")
	}
}

func TestGenerator_GenerateSingleFormat(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		ProjectName:    "TestAPI",
		ProjectVersion: "1.0.0",
		OutputDir:      tmpDir,
		Formats:        []Format{FormatOpenAPI},
	}

	generator, err := NewGenerator(config)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	program := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Post",
				Fields: []*ast.FieldNode{
					{
						Name:     "title",
						Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Nullable: false,
					},
				},
			},
		},
	}

	err = generator.Generate(program)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify only OpenAPI was generated
	openAPISpec := filepath.Join(tmpDir, "openapi.json")
	if _, err := os.Stat(openAPISpec); os.IsNotExist(err) {
		t.Error("OpenAPI spec not generated")
	}

	// HTML should not be generated
	htmlIndex := filepath.Join(tmpDir, "html", "index.html")
	if _, err := os.Stat(htmlIndex); err == nil {
		t.Error("HTML should not be generated when not requested")
	}
}

func TestGenerator_EmptyProgram(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		ProjectName:    "EmptyAPI",
		ProjectVersion: "1.0.0",
		OutputDir:      tmpDir,
		Formats:        []Format{FormatMarkdown},
	}

	generator, err := NewGenerator(config)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	program := &ast.Program{
		Resources: []*ast.ResourceNode{},
	}

	err = generator.Generate(program)
	if err != nil {
		t.Fatalf("Generate failed for empty program: %v", err)
	}

	// Should still generate documentation with no resources
	mdReadme := filepath.Join(tmpDir, "markdown", "README.md")
	if _, err := os.Stat(mdReadme); os.IsNotExist(err) {
		t.Error("README not generated for empty program")
	}
}

func TestNewGenerator_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		shouldError bool
		errorMsg    string
	}{
		{
			name: "empty project name",
			config: &Config{
				ProjectName: "",
				OutputDir:   "/tmp/test",
			},
			shouldError: true,
			errorMsg:    "project name is required",
		},
		{
			name: "project name too long",
			config: &Config{
				ProjectName: "ThisIsAReallyLongProjectNameThatExceedsTheMaximumAllowedLengthOf100CharactersAndShouldFailValidationX",
				OutputDir:   "/tmp/test",
			},
			shouldError: true,
			errorMsg:    "project name too long",
		},
		{
			name: "project description too long",
			config: &Config{
				ProjectName:        "TestProject",
				ProjectDescription: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
				OutputDir:          "/tmp/test",
			},
			shouldError: true,
			errorMsg:    "project description too long",
		},
		{
			name: "valid config with empty version",
			config: &Config{
				ProjectName:    "TestProject",
				ProjectVersion: "",
				OutputDir:      "/tmp/test",
			},
			shouldError: false,
		},
		{
			name: "valid config",
			config: &Config{
				ProjectName:    "TestProject",
				ProjectVersion: "1.0.0",
				OutputDir:      "/tmp/test",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator(tt.config)
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if gen != nil {
					t.Error("Expected nil generator when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if gen == nil {
					t.Error("Expected non-nil generator")
				}
				// Check that empty version gets defaulted
				if tt.config.ProjectVersion == "" && gen.config.ProjectVersion != "0.0.0" {
					t.Errorf("Expected version to be defaulted to '0.0.0', got '%s'", gen.config.ProjectVersion)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
