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

	gen := NewGenerator(config)
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

	generator := NewGenerator(config)

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

	err := generator.Generate(program)
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

	generator := NewGenerator(config)

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

	err := generator.Generate(program)
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

	generator := NewGenerator(config)

	program := &ast.Program{
		Resources: []*ast.ResourceNode{},
	}

	err := generator.Generate(program)
	if err != nil {
		t.Fatalf("Generate failed for empty program: %v", err)
	}

	// Should still generate documentation with no resources
	mdReadme := filepath.Join(tmpDir, "markdown", "README.md")
	if _, err := os.Stat(mdReadme); os.IsNotExist(err) {
		t.Error("README not generated for empty program")
	}
}
