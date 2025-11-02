package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TestNullableFieldJSONSerialization verifies that nullable fields serialize correctly to JSON
func TestNullableFieldJSONSerialization(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "TestUser",
		Fields: []*ast.FieldNode{
			{
				Name: "name",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
			{
				Name: "bio",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "text",
					Nullable: true,
				},
				Nullable: true,
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Verify nullable field uses pointer type
	if !strings.Contains(code, "Bio") || !strings.Contains(code, "*string") {
		t.Error("Nullable field should use *string type")
	}

	// Verify omitempty is present
	if !strings.Contains(code, "json:\"bio,omitempty\"") {
		t.Error("Nullable field should have omitempty in JSON tag")
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conduit-nullable-test-*")
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
	filePath := filepath.Join(modelsDir, "testuser.go")
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write generated code: %v", err)
	}

	// Create test file to verify JSON serialization
	testCode := `package models_test

import (
	"encoding/json"
	"testing"
	"` + tmpDir + `/models"
)

func TestJSONSerialization(t *testing.T) {
	// Test with nil bio
	user1 := &models.TestUser{
		ID: 1,
		Name: "John",
		Bio: nil,
	}

	data1, err := json.Marshal(user1)
	if err != nil {
		t.Fatalf("Failed to marshal user1: %v", err)
	}

	// Should not contain bio field or contain null
	jsonStr1 := string(data1)
	if !contains(jsonStr1, "\"name\":\"John\"") {
		t.Errorf("JSON should contain name field, got: %s", jsonStr1)
	}

	// bio should either be omitted or be null, not a Go struct representation
	if contains(jsonStr1, "String") || contains(jsonStr1, "Valid") {
		t.Errorf("JSON should not contain sql.Null* representation, got: %s", jsonStr1)
	}

	// Test with non-nil bio
	bioValue := "Software Developer"
	user2 := &models.TestUser{
		ID: 2,
		Name: "Jane",
		Bio: &bioValue,
	}

	data2, err := json.Marshal(user2)
	if err != nil {
		t.Fatalf("Failed to marshal user2: %v", err)
	}

	jsonStr2 := string(data2)
	if !contains(jsonStr2, "\"bio\":\"Software Developer\"") {
		t.Errorf("JSON should contain bio value, got: %s", jsonStr2)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
`

	testFilePath := filepath.Join(tmpDir, "models_test.go")
	if err := os.WriteFile(testFilePath, []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to write test code: %v", err)
	}

	// Create go.mod file
	goModContent := `module test-nullable

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

	// Run the test to verify JSON serialization
	cmd = exec.Command("go", "test", "-v", "./...")
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("Test output:\n%s", string(output))
		t.Fatalf("JSON serialization test failed: %v", err)
	}

	t.Logf("JSON serialization test passed:\n%s", string(output))
}

// TestNullableFieldValidation verifies that validation works on nullable fields with constraints
func TestNullableFieldValidation(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "TestPost",
		Fields: []*ast.FieldNode{
			{
				Name: "title",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
			{
				Name: "summary",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "text",
					Nullable: true,
				},
				Nullable: true,
				Constraints: []*ast.ConstraintNode{
					{
						Name: "min",
						Arguments: []ast.ExprNode{
							&ast.LiteralExpr{Value: int64(10)},
						},
						Error: "summary must be at least 10 characters",
					},
					{
						Name: "max",
						Arguments: []ast.ExprNode{
							&ast.LiteralExpr{Value: int64(100)},
						},
						Error: "summary must be at most 100 characters",
					},
				},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Verify nullable field uses pointer type
	if !strings.Contains(code, "Summary") || !strings.Contains(code, "*string") {
		t.Error("Nullable field should use *string type")
	}

	// Verify validation includes nil check
	if !strings.Contains(code, "if p.Summary != nil && len(*p.Summary) < 10") {
		t.Error("Validation should check for nil before dereferencing pointer for min constraint")
	}

	if !strings.Contains(code, "if p.Summary != nil && len(*p.Summary) > 100") {
		t.Error("Validation should check for nil before dereferencing pointer for max constraint")
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conduit-validation-test-*")
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
	filePath := filepath.Join(modelsDir, "testpost.go")
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write generated code: %v", err)
	}

	// Create go.mod file
	goModContent := `module test-validation

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
		t.Fatalf("Generated code with validation failed to compile: %v\nOutput:\n%s\nGenerated code:\n%s",
			err, string(output), code)
	}

	t.Log("Generated code with validation on nullable fields compiles successfully")
}
