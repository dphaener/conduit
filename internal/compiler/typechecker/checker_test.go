package typechecker

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TestPrimitiveTypes tests basic primitive type checking
func TestPrimitiveTypes(t *testing.T) {
	tests := []struct {
		name     string
		typ1     Type
		typ2     Type
		assignOk bool
	}{
		{
			name:     "string! to string! - OK",
			typ1:     NewPrimitiveType("string", false),
			typ2:     NewPrimitiveType("string", false),
			assignOk: true,
		},
		{
			name:     "string! to string? - OK",
			typ1:     NewPrimitiveType("string", true),
			typ2:     NewPrimitiveType("string", false),
			assignOk: true,
		},
		{
			name:     "string? to string! - FAIL (nullability violation)",
			typ1:     NewPrimitiveType("string", false),
			typ2:     NewPrimitiveType("string", true),
			assignOk: false,
		},
		{
			name:     "int! to string! - FAIL (type mismatch)",
			typ1:     NewPrimitiveType("string", false),
			typ2:     NewPrimitiveType("int", false),
			assignOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typ1.IsAssignableFrom(tt.typ2)
			if result != tt.assignOk {
				t.Errorf("Expected IsAssignableFrom to be %v, got %v", tt.assignOk, result)
			}
		})
	}
}

// TestArrayTypes tests array type checking
func TestArrayTypes(t *testing.T) {
	intRequired := NewPrimitiveType("int", false)
	intNullable := NewPrimitiveType("int", true)
	stringRequired := NewPrimitiveType("string", false)

	tests := []struct {
		name     string
		typ1     Type
		typ2     Type
		assignOk bool
	}{
		{
			name:     "array<int!>! to array<int!>! - OK",
			typ1:     NewArrayType(intRequired, false),
			typ2:     NewArrayType(intRequired, false),
			assignOk: true,
		},
		{
			name:     "array<int!>! to array<int!>? - FAIL",
			typ1:     NewArrayType(intRequired, false),
			typ2:     NewArrayType(intRequired, true),
			assignOk: false,
		},
		{
			name:     "array<int?>! to array<int!>! - FAIL (element nullability)",
			typ1:     NewArrayType(intRequired, false),
			typ2:     NewArrayType(intNullable, false),
			assignOk: false,
		},
		{
			name:     "array<string!>! to array<int!>! - FAIL (element type)",
			typ1:     NewArrayType(intRequired, false),
			typ2:     NewArrayType(stringRequired, false),
			assignOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typ1.IsAssignableFrom(tt.typ2)
			if result != tt.assignOk {
				t.Errorf("Expected IsAssignableFrom to be %v, got %v", tt.assignOk, result)
			}
		})
	}
}

// TestHashTypes tests hash type checking
func TestHashTypes(t *testing.T) {
	stringRequired := NewPrimitiveType("string", false)
	intRequired := NewPrimitiveType("int", false)
	intNullable := NewPrimitiveType("int", true)

	tests := []struct {
		name     string
		typ1     Type
		typ2     Type
		assignOk bool
	}{
		{
			name:     "hash<string!, int!>! to hash<string!, int!>! - OK",
			typ1:     NewHashType(stringRequired, intRequired, false),
			typ2:     NewHashType(stringRequired, intRequired, false),
			assignOk: true,
		},
		{
			name:     "hash<string!, int?>! to hash<string!, int!>! - FAIL",
			typ1:     NewHashType(stringRequired, intRequired, false),
			typ2:     NewHashType(stringRequired, intNullable, false),
			assignOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typ1.IsAssignableFrom(tt.typ2)
			if result != tt.assignOk {
				t.Errorf("Expected IsAssignableFrom to be %v, got %v", tt.assignOk, result)
			}
		})
	}
}

// TestStructTypes tests struct type checking
func TestStructTypes(t *testing.T) {
	stringRequired := NewPrimitiveType("string", false)
	intRequired := NewPrimitiveType("int", false)

	struct1 := NewStructType([]StructField{
		{Name: "name", Type: stringRequired},
		{Name: "age", Type: intRequired},
	}, false)

	struct2 := NewStructType([]StructField{
		{Name: "name", Type: stringRequired},
		{Name: "age", Type: intRequired},
	}, false)

	struct3 := NewStructType([]StructField{
		{Name: "name", Type: stringRequired},
	}, false)

	tests := []struct {
		name     string
		typ1     Type
		typ2     Type
		assignOk bool
	}{
		{
			name:     "identical structs - OK",
			typ1:     struct1,
			typ2:     struct2,
			assignOk: true,
		},
		{
			name:     "different field count - FAIL",
			typ1:     struct1,
			typ2:     struct3,
			assignOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typ1.IsAssignableFrom(tt.typ2)
			if result != tt.assignOk {
				t.Errorf("Expected IsAssignableFrom to be %v, got %v", tt.assignOk, result)
			}
		})
	}
}

// TestEnumTypes tests enum type checking
func TestEnumTypes(t *testing.T) {
	enum1 := NewEnumType([]string{"draft", "published"}, false)
	enum2 := NewEnumType([]string{"draft", "published"}, false)
	enum3 := NewEnumType([]string{"active", "inactive"}, false)

	tests := []struct {
		name     string
		typ1     Type
		typ2     Type
		assignOk bool
	}{
		{
			name:     "identical enums - OK",
			typ1:     enum1,
			typ2:     enum2,
			assignOk: true,
		},
		{
			name:     "different values - FAIL",
			typ1:     enum1,
			typ2:     enum3,
			assignOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typ1.IsAssignableFrom(tt.typ2)
			if result != tt.assignOk {
				t.Errorf("Expected IsAssignableFrom to be %v, got %v", tt.assignOk, result)
			}
		})
	}
}

// TestResourceTypes tests resource type checking
func TestResourceTypes(t *testing.T) {
	user1 := NewResourceType("User", false)
	user2 := NewResourceType("User", false)
	post := NewResourceType("Post", false)
	userNullable := NewResourceType("User", true)

	tests := []struct {
		name     string
		typ1     Type
		typ2     Type
		assignOk bool
	}{
		{
			name:     "User! to User! - OK",
			typ1:     user1,
			typ2:     user2,
			assignOk: true,
		},
		{
			name:     "User? to User! - FAIL",
			typ1:     user1,
			typ2:     userNullable,
			assignOk: false,
		},
		{
			name:     "Post! to User! - FAIL",
			typ1:     user1,
			typ2:     post,
			assignOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typ1.IsAssignableFrom(tt.typ2)
			if result != tt.assignOk {
				t.Errorf("Expected IsAssignableFrom to be %v, got %v", tt.assignOk, result)
			}
		})
	}
}

// TestNullabilityViolation tests that nullable-to-required assignments are caught
func TestNullabilityViolation(t *testing.T) {
	tc := NewTypeChecker()

	// Create a simple resource with two fields
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "title",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 2, Column: 3},
			},
			{
				Name:     "bio",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"},
				Nullable: true,
				Loc:      ast.SourceLocation{Line: 3, Column: 3},
			},
		},
		Hooks: []*ast.HookNode{
			{
				Timing: "before",
				Event:  "create",
				Body: []ast.StmtNode{
					&ast.AssignmentStmt{
						Target: &ast.FieldAccessExpr{
							Object: &ast.SelfExpr{},
							Field:  "title",
						},
						Value: &ast.FieldAccessExpr{
							Object: &ast.SelfExpr{},
							Field:  "bio",
						},
						Loc: ast.SourceLocation{Line: 6, Column: 5},
					},
				},
				Loc: ast.SourceLocation{Line: 5, Column: 3},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	prog := &ast.Program{
		Resources: []*ast.ResourceNode{resource},
	}

	errors := tc.CheckProgram(prog)

	if len(errors) == 0 {
		t.Fatal("Expected nullability violation error, got none")
	}

	// Check that we got a TYP101 error
	foundNullabilityError := false
	for _, err := range errors {
		if err.Code == ErrNullabilityViolation {
			foundNullabilityError = true
			break
		}
	}

	if !foundNullabilityError {
		t.Errorf("Expected TYP101 nullability violation, got: %v", errors)
	}
}

// TestTypeMismatch tests that type mismatches are caught
func TestTypeMismatch(t *testing.T) {
	tc := NewTypeChecker()

	// Create a resource with int and string fields
	resource := &ast.ResourceNode{
		Name: "Product",
		Fields: []*ast.FieldNode{
			{
				Name:     "price",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "float"},
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 2, Column: 3},
			},
			{
				Name:     "name",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 3, Column: 3},
			},
		},
		Hooks: []*ast.HookNode{
			{
				Timing: "before",
				Event:  "create",
				Body: []ast.StmtNode{
					&ast.AssignmentStmt{
						Target: &ast.FieldAccessExpr{
							Object: &ast.SelfExpr{},
							Field:  "price",
						},
						Value: &ast.FieldAccessExpr{
							Object: &ast.SelfExpr{},
							Field:  "name",
						},
						Loc: ast.SourceLocation{Line: 6, Column: 5},
					},
				},
				Loc: ast.SourceLocation{Line: 5, Column: 3},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	prog := &ast.Program{
		Resources: []*ast.ResourceNode{resource},
	}

	errors := tc.CheckProgram(prog)

	if len(errors) == 0 {
		t.Fatal("Expected type mismatch error, got none")
	}

	// Check that we got a TYP102 error
	foundTypeMismatch := false
	for _, err := range errors {
		if err.Code == ErrTypeMismatch {
			foundTypeMismatch = true
			break
		}
	}

	if !foundTypeMismatch {
		t.Errorf("Expected TYP102 type mismatch, got: %v", errors)
	}
}

// TestUnwrapOperator tests the unwrap operator (!)
func TestUnwrapOperator(t *testing.T) {
	tc := NewTypeChecker()

	// Create a resource that uses unwrap
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "title",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 2, Column: 3},
			},
			{
				Name:     "bio",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"},
				Nullable: true,
				Loc:      ast.SourceLocation{Line: 3, Column: 3},
			},
		},
		Hooks: []*ast.HookNode{
			{
				Timing: "before",
				Event:  "create",
				Body: []ast.StmtNode{
					&ast.AssignmentStmt{
						Target: &ast.FieldAccessExpr{
							Object: &ast.SelfExpr{},
							Field:  "title",
						},
						Value: &ast.UnaryExpr{
							Operator: "!",
							Operand: &ast.FieldAccessExpr{
								Object: &ast.SelfExpr{},
								Field:  "bio",
							},
							Loc: ast.SourceLocation{Line: 6, Column: 5},
						},
						Loc: ast.SourceLocation{Line: 6, Column: 5},
					},
				},
				Loc: ast.SourceLocation{Line: 5, Column: 3},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	prog := &ast.Program{
		Resources: []*ast.ResourceNode{resource},
	}

	errors := tc.CheckProgram(prog)

	// This should succeed - unwrap makes nullable into required
	if len(errors) > 0 {
		// Filter out warnings
		var realErrors []*TypeError
		for _, err := range errors {
			if err.Severity == SeverityError {
				realErrors = append(realErrors, err)
			}
		}
		if len(realErrors) > 0 {
			t.Errorf("Expected no errors with unwrap operator, got: %v", realErrors)
		}
	}
}

// TestNilCoalescing tests the nil coalescing operator (??)
func TestNilCoalescing(t *testing.T) {
	tc := NewTypeChecker()

	// Create a resource that uses nil coalescing
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "title",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 2, Column: 3},
			},
			{
				Name:     "bio",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"},
				Nullable: true,
				Loc:      ast.SourceLocation{Line: 3, Column: 3},
			},
		},
		Hooks: []*ast.HookNode{
			{
				Timing: "before",
				Event:  "create",
				Body: []ast.StmtNode{
					&ast.AssignmentStmt{
						Target: &ast.FieldAccessExpr{
							Object: &ast.SelfExpr{},
							Field:  "title",
						},
						Value: &ast.NullCoalesceExpr{
							Left: &ast.FieldAccessExpr{
								Object: &ast.SelfExpr{},
								Field:  "bio",
							},
							Right: &ast.LiteralExpr{
								Value: "Default",
							},
							Loc: ast.SourceLocation{Line: 6, Column: 5},
						},
						Loc: ast.SourceLocation{Line: 6, Column: 5},
					},
				},
				Loc: ast.SourceLocation{Line: 5, Column: 3},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	prog := &ast.Program{
		Resources: []*ast.ResourceNode{resource},
	}

	errors := tc.CheckProgram(prog)

	// This should succeed - ?? makes nullable into required
	if len(errors) > 0 {
		// Filter out warnings
		var realErrors []*TypeError
		for _, err := range errors {
			if err.Severity == SeverityError {
				realErrors = append(realErrors, err)
			}
		}
		if len(realErrors) > 0 {
			t.Errorf("Expected no errors with nil coalescing, got: %v", realErrors)
		}
	}
}

// TestConstraintTypeValidation tests that constraints are validated against field types
func TestConstraintTypeValidation(t *testing.T) {
	tc := NewTypeChecker()

	// Create a resource with @min on a bool field (invalid)
	resource := &ast.ResourceNode{
		Name: "Test",
		Fields: []*ast.FieldNode{
			{
				Name:     "active",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "bool"},
				Nullable: false,
				Constraints: []*ast.ConstraintNode{
					{
						Name: "min",
						Arguments: []ast.ExprNode{
							&ast.LiteralExpr{Value: 1},
						},
						Loc: ast.SourceLocation{Line: 2, Column: 20},
					},
				},
				Loc: ast.SourceLocation{Line: 2, Column: 3},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	prog := &ast.Program{
		Resources: []*ast.ResourceNode{resource},
	}

	errors := tc.CheckProgram(prog)

	if len(errors) == 0 {
		t.Fatal("Expected constraint type error, got none")
	}

	// Check that we got a TYP400 error
	foundConstraintError := false
	for _, err := range errors {
		if err.Code == ErrInvalidConstraintType {
			foundConstraintError = true
			break
		}
	}

	if !foundConstraintError {
		t.Errorf("Expected TYP400 invalid constraint type, got: %v", errors)
	}
}

// TestStdlibFunctionCall tests standard library function type checking
func TestStdlibFunctionCall(t *testing.T) {
	tc := NewTypeChecker()

	// Create a resource that calls String.slugify
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "title",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 2, Column: 3},
			},
			{
				Name:     "slug",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 3, Column: 3},
			},
		},
		Hooks: []*ast.HookNode{
			{
				Timing: "before",
				Event:  "create",
				Body: []ast.StmtNode{
					&ast.AssignmentStmt{
						Target: &ast.FieldAccessExpr{
							Object: &ast.SelfExpr{},
							Field:  "slug",
						},
						Value: &ast.CallExpr{
							Namespace: "String",
							Function:  "slugify",
							Arguments: []ast.ExprNode{
								&ast.FieldAccessExpr{
									Object: &ast.SelfExpr{},
									Field:  "title",
								},
							},
							Loc: ast.SourceLocation{Line: 6, Column: 5},
						},
						Loc: ast.SourceLocation{Line: 6, Column: 5},
					},
				},
				Loc: ast.SourceLocation{Line: 5, Column: 3},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	prog := &ast.Program{
		Resources: []*ast.ResourceNode{resource},
	}

	errors := tc.CheckProgram(prog)

	// This should succeed
	if len(errors) > 0 {
		// Filter out warnings
		var realErrors []*TypeError
		for _, err := range errors {
			if err.Severity == SeverityError {
				realErrors = append(realErrors, err)
			}
		}
		if len(realErrors) > 0 {
			t.Errorf("Expected no errors for valid stdlib call, got: %v", realErrors)
		}
	}
}

// TestInvalidFunctionCall tests that invalid function calls are caught
func TestInvalidFunctionCall(t *testing.T) {
	tc := NewTypeChecker()

	// Create a resource that calls an undefined function
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "slug",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 2, Column: 3},
			},
		},
		Hooks: []*ast.HookNode{
			{
				Timing: "before",
				Event:  "create",
				Body: []ast.StmtNode{
					&ast.AssignmentStmt{
						Target: &ast.FieldAccessExpr{
							Object: &ast.SelfExpr{},
							Field:  "slug",
						},
						Value: &ast.CallExpr{
							Namespace: "String",
							Function:  "nonexistent",
							Arguments: []ast.ExprNode{},
							Loc:       ast.SourceLocation{Line: 5, Column: 5},
						},
						Loc: ast.SourceLocation{Line: 5, Column: 5},
					},
				},
				Loc: ast.SourceLocation{Line: 4, Column: 3},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	prog := &ast.Program{
		Resources: []*ast.ResourceNode{resource},
	}

	errors := tc.CheckProgram(prog)

	if len(errors) == 0 {
		t.Fatal("Expected undefined function error, got none")
	}

	// Check that we got a TYP300 error
	foundFunctionError := false
	for _, err := range errors {
		if err.Code == ErrUndefinedFunction {
			foundFunctionError = true
			break
		}
	}

	if !foundFunctionError {
		t.Errorf("Expected TYP300 undefined function, got: %v", errors)
	}
}

// TestTypeString tests the String() method for types
func TestTypeString(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		expected string
	}{
		{
			name:     "string!",
			typ:      NewPrimitiveType("string", false),
			expected: "string!",
		},
		{
			name:     "string?",
			typ:      NewPrimitiveType("string", true),
			expected: "string?",
		},
		{
			name:     "array<int!>!",
			typ:      NewArrayType(NewPrimitiveType("int", false), false),
			expected: "array<int!>!",
		},
		{
			name:     "hash<string!, int!>!",
			typ:      NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("int", false), false),
			expected: "hash<string!, int!>!",
		},
		{
			name:     "enum",
			typ:      NewEnumType([]string{"a", "b"}, false),
			expected: `enum ["a", "b"]!`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.typ.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
