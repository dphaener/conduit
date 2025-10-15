package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestGeneratedCode_Compiles(t *testing.T) {
	// Create a simple resource
	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{
				Name: "username",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
			{
				Name: "email",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conduit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create models directory
	modelsDir := filepath.Join(tmpDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatalf("Failed to create models dir: %v", err)
	}

	// Write generated code to file
	filePath := filepath.Join(modelsDir, "user.go")
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write generated code: %v", err)
	}

	// Create go.mod file
	goModContent := `module test-conduit

go 1.23

require github.com/google/uuid v1.6.0
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Try to compile the generated code
	cmd := exec.Command("go", "build", "./models")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Generated code failed to compile: %v\nOutput:\n%s\nGenerated code:\n%s",
			err, string(output), code)
	}
}

func TestGeneratedCode_GoFmt(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{
				Name: "username",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write code to file
	if _, err := tmpFile.Write([]byte(code)); err != nil {
		t.Fatalf("Failed to write code: %v", err)
	}
	tmpFile.Close()

	// Run gofmt
	cmd := exec.Command("gofmt", "-l", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt failed: %v\nOutput: %s", err, string(output))
	}

	// If gofmt outputs the filename, it means the code is not formatted
	if strings.TrimSpace(string(output)) != "" {
		// Run gofmt -d to show the diff
		diffCmd := exec.Command("gofmt", "-d", tmpFile.Name())
		diffOutput, _ := diffCmd.CombinedOutput()
		t.Errorf("Generated code is not properly formatted according to gofmt\nDiff:\n%s\nGenerated code:\n%s",
			string(diffOutput), code)
	}
}

func TestGeneratedSQL_Valid(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "User",
			Fields: []*ast.FieldNode{
				{
					Name: "username",
					Type: &ast.TypeNode{
						Kind:     ast.TypePrimitive,
						Name:     "string",
						Nullable: false,
					},
					Nullable: false,
					Constraints: []*ast.ConstraintNode{
						{
							Name: "min",
							Arguments: []ast.ExprNode{
								&ast.LiteralExpr{Value: int64(3)},
							},
						},
						{
							Name: "max",
							Arguments: []ast.ExprNode{
								&ast.LiteralExpr{Value: int64(50)},
							},
						},
						{Name: "unique"},
					},
				},
				{
					Name: "email",
					Type: &ast.TypeNode{
						Kind:     ast.TypePrimitive,
						Name:     "string",
						Nullable: false,
					},
					Nullable: false,
				},
			},
		},
	}

	gen := NewGenerator()
	sql, err := gen.GenerateMigrations(resources)
	if err != nil {
		t.Fatalf("GenerateMigrations failed: %v", err)
	}

	// Basic SQL validation - check for required elements
	requiredElements := []string{
		"CREATE TABLE users",
		"username VARCHAR",
		"email VARCHAR",
		"NOT NULL",
		"CHECK (length(username) >= 3)",
		"UNIQUE",
	}

	for _, element := range requiredElements {
		if !strings.Contains(sql, element) {
			t.Errorf("Generated SQL missing required element: %s\nGenerated SQL:\n%s",
				element, sql)
		}
	}

	// Verify SQL syntax is valid (basic check)
	if !strings.Contains(sql, ");") {
		t.Error("Generated SQL should end table definitions with );")
	}
}
