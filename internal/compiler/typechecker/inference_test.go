package typechecker

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TestInferLiteral tests literal type inference
func TestInferLiteral(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{name: "string", value: "hello", expected: "string!"},
		{name: "int", value: 42, expected: "int!"},
		{name: "float", value: 3.14, expected: "float!"},
		{name: "bool", value: true, expected: "bool!"},
		{name: "nil", value: nil, expected: "nil?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lit := &ast.LiteralExpr{Value: tt.value}
			typ, err := tc.inferLiteral(lit)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if typ.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, typ.String())
			}
		})
	}
}

// TestInferBinaryOperations tests binary expression type inference
func TestInferBinaryOperations(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		name       string
		leftValue  interface{}
		rightValue interface{}
		op         string
		expected   string
	}{
		{name: "int + int", leftValue: 1, rightValue: 2, op: "+", expected: "int!"},
		{name: "float + int", leftValue: 1.5, rightValue: 2, op: "+", expected: "float!"},
		{name: "int == int", leftValue: 1, rightValue: 2, op: "==", expected: "bool!"},
		{name: "int < int", leftValue: 1, rightValue: 2, op: "<", expected: "bool!"},
		{name: "int ** int", leftValue: 2, rightValue: 3, op: "**", expected: "float!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a binary expression with appropriate literal types
			bin := &ast.BinaryExpr{
				Left:     &ast.LiteralExpr{Value: tt.leftValue},
				Right:    &ast.LiteralExpr{Value: tt.rightValue},
				Operator: tt.op,
			}

			typ, err := tc.inferBinary(bin)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if typ.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, typ.String())
			}
		})
	}
}

// TestSafeNavigation tests safe navigation operator
func TestSafeNavigation(t *testing.T) {
	tc := NewTypeChecker()

	// Create a resource with nullable and required fields
	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{
				Name:     "email",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "email"},
				Nullable: true,
			},
		},
	}

	tc.resources["User"] = resource
	tc.currentResource = resource

	// Test safe navigation on nullable field
	sn := &ast.SafeNavigationExpr{
		Object: &ast.IdentifierExpr{Name: "User"},
		Field:  "email",
		Loc:    ast.SourceLocation{Line: 1, Column: 1},
	}

	typ, err := tc.inferSafeNavigation(sn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !typ.IsNullable() {
		t.Error("Safe navigation should always return nullable type")
	}
}

// TestArrayIndexing tests array and hash indexing
func TestArrayIndexing(t *testing.T) {
	tc := NewTypeChecker()

	// Test array indexing
	arr := &ast.IndexExpr{
		Object: &ast.ArrayLiteralExpr{
			Elements: []ast.ExprNode{
				&ast.LiteralExpr{Value: 1},
				&ast.LiteralExpr{Value: 2},
			},
		},
		Index: &ast.LiteralExpr{Value: 0},
		Loc:   ast.SourceLocation{Line: 1, Column: 1},
	}

	typ, err := tc.inferIndex(arr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !typ.IsNullable() {
		t.Error("Array indexing should return nullable type")
	}
}

// TestLetStatement tests let statement type checking
func TestLetStatement(t *testing.T) {
	tc := NewTypeChecker()

	// Test let with explicit type
	letStmt := &ast.LetStmt{
		Name:  "myVar",
		Type:  &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
		Value: &ast.LiteralExpr{Value: "hello"},
		Loc:   ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkLet(letStmt)

	// Check that variable was added to scope
	if typ, ok := tc.currentScope["myVar"]; !ok {
		t.Error("Variable not added to scope")
	} else if typ.String() != "string!" {
		t.Errorf("Expected string!, got %s", typ.String())
	}

	// Test type mismatch
	tc2 := NewTypeChecker()
	letStmt2 := &ast.LetStmt{
		Name:  "myVar",
		Type:  &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"},
		Value: &ast.LiteralExpr{Value: "hello"},
		Loc:   ast.SourceLocation{Line: 1, Column: 1},
	}

	tc2.checkLet(letStmt2)

	if len(tc2.errors) == 0 {
		t.Error("Expected type mismatch error")
	}
}

// TestIfStatement tests if statement type checking
func TestIfStatement(t *testing.T) {
	tc := NewTypeChecker()

	// Test valid if statement
	ifStmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Value: true},
		ThenBranch: []ast.StmtNode{
			&ast.ExprStmt{
				Expr: &ast.LiteralExpr{Value: 1},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkIf(ifStmt)

	if len(tc.errors) > 0 {
		t.Errorf("Unexpected errors: %v", tc.errors)
	}

	// Test non-boolean condition
	tc2 := NewTypeChecker()
	ifStmt2 := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Value: 42},
		ThenBranch: []ast.StmtNode{
			&ast.ExprStmt{
				Expr: &ast.LiteralExpr{Value: 1},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	tc2.checkIf(ifStmt2)

	if len(tc2.errors) == 0 {
		t.Error("Expected type error for non-boolean condition")
	}
}

// TestValidation tests validation block type checking
func TestValidation(t *testing.T) {
	tc := NewTypeChecker()

	resource := &ast.ResourceNode{
		Name: "Test",
		Fields: []*ast.FieldNode{
			{
				Name:     "active",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "bool"},
				Nullable: false,
			},
		},
	}

	tc.resources["Test"] = resource
	tc.currentResource = resource

	// Test valid validation with boolean condition
	validation := &ast.ValidationNode{
		Name:      "test_validation",
		Condition: &ast.LiteralExpr{Value: true},
		Error:     "Test error",
		Loc:       ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkValidation(validation)

	if len(tc.errors) > 0 {
		t.Errorf("Unexpected errors: %v", tc.errors)
	}

	// Test validation with non-boolean condition
	tc2 := NewTypeChecker()
	tc2.resources["Test"] = resource
	tc2.currentResource = resource

	validation2 := &ast.ValidationNode{
		Name:      "test_validation",
		Condition: &ast.LiteralExpr{Value: 42},
		Error:     "Test error",
		Loc:       ast.SourceLocation{Line: 1, Column: 1},
	}

	tc2.checkValidation(validation2)

	if len(tc2.errors) == 0 {
		t.Error("Expected type error for non-boolean validation condition")
	}
}

// TestConstraint tests constraint block type checking
func TestConstraint(t *testing.T) {
	tc := NewTypeChecker()

	resource := &ast.ResourceNode{
		Name: "Test",
		Fields: []*ast.FieldNode{
			{
				Name:     "price",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "float"},
				Nullable: false,
			},
		},
	}

	tc.resources["Test"] = resource
	tc.currentResource = resource

	// Test valid constraint
	constraint := &ast.ConstraintNode{
		Name:      "price_positive",
		Condition: &ast.LiteralExpr{Value: true},
		Error:     "Price must be positive",
		Loc:       ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkConstraint(constraint)

	if len(tc.errors) > 0 {
		t.Errorf("Unexpected errors: %v", tc.errors)
	}
}

// TestComputed tests computed field type checking
func TestComputed(t *testing.T) {
	tc := NewTypeChecker()

	resource := &ast.ResourceNode{
		Name: "Test",
		Fields: []*ast.FieldNode{
			{
				Name:     "first_name",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
			},
			{
				Name:     "last_name",
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
			},
		},
	}

	tc.resources["Test"] = resource
	tc.currentResource = resource

	// Test valid computed field
	computed := &ast.ComputedNode{
		Name: "full_name",
		Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
		Body: &ast.LiteralExpr{Value: "John Doe"},
		Loc:  ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkComputed(computed)

	if len(tc.errors) > 0 {
		t.Errorf("Unexpected errors: %v", tc.errors)
	}
}

// TestDefaultValue tests default value type checking
func TestDefaultValue(t *testing.T) {
	tc := NewTypeChecker()

	// Test valid default
	field := &ast.FieldNode{
		Name:     "count",
		Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"},
		Nullable: false,
		Default:  &ast.LiteralExpr{Value: 0},
		Loc:      ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkDefaultValue(field)

	if len(tc.errors) > 0 {
		t.Errorf("Unexpected errors: %v", tc.errors)
	}

	// Test invalid default (type mismatch)
	tc2 := NewTypeChecker()
	field2 := &ast.FieldNode{
		Name:     "count",
		Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"},
		Nullable: false,
		Default:  &ast.LiteralExpr{Value: "not a number"},
		Loc:      ast.SourceLocation{Line: 1, Column: 1},
	}

	tc2.checkDefaultValue(field2)

	if len(tc2.errors) == 0 {
		t.Error("Expected type mismatch error for invalid default")
	}
}

// TestErrorFormatting tests error message formatting
func TestErrorFormatting(t *testing.T) {
	err := NewNullabilityViolation(
		ast.SourceLocation{Line: 5, Column: 10},
		NewPrimitiveType("string", false),
		NewPrimitiveType("string", true),
	)

	formatted := err.Format()
	if formatted == "" {
		t.Error("Expected non-empty formatted error")
	}

	if err.Error() != formatted {
		t.Error("Error() should return formatted message")
	}

	// Test JSON serialization
	json, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Errorf("Unexpected JSON error: %v", jsonErr)
	}
	if json == "" {
		t.Error("Expected non-empty JSON")
	}
}

// TestErrorList tests error list functionality
func TestErrorList(t *testing.T) {
	errList := ErrorList{
		NewTypeMismatch(
			ast.SourceLocation{Line: 1, Column: 1},
			NewPrimitiveType("int", false),
			NewPrimitiveType("string", false),
			"test",
		),
	}

	if !errList.HasErrors() {
		t.Error("Expected HasErrors to return true")
	}

	formatted := errList.Error()
	if formatted == "" {
		t.Error("Expected non-empty error string")
	}

	json, err := errList.ToJSON()
	if err != nil {
		t.Errorf("Unexpected JSON error: %v", err)
	}
	if json == "" {
		t.Error("Expected non-empty JSON")
	}

	// Test empty list
	emptyList := ErrorList{}
	if emptyList.HasErrors() {
		t.Error("Expected HasErrors to return false for empty list")
	}
}

// TestMatchStatement tests match statement type checking
func TestMatchStatement(t *testing.T) {
	tc := NewTypeChecker()

	match := &ast.MatchStmt{
		Value: &ast.LiteralExpr{Value: "test"},
		Cases: []*ast.MatchCase{
			{
				Pattern: &ast.LiteralExpr{Value: "test"},
				Body: []ast.StmtNode{
					&ast.ExprStmt{Expr: &ast.LiteralExpr{Value: 1}},
				},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkMatch(match)

	if len(tc.errors) > 0 {
		t.Errorf("Unexpected errors: %v", tc.errors)
	}
}

// TestTypeFromASTNode tests AST type node conversion
func TestTypeFromASTNode(t *testing.T) {
	tests := []struct {
		name     string
		node     *ast.TypeNode
		nullable bool
		expected string
		wantErr  bool
	}{
		{
			name:     "primitive string!",
			node:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
			nullable: false,
			expected: "string!",
			wantErr:  false,
		},
		{
			name: "array<int!>!",
			node: &ast.TypeNode{
				Kind:        ast.TypeArray,
				ElementType: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"},
			},
			nullable: false,
			expected: "array<int!>!",
			wantErr:  false,
		},
		{
			name: "enum",
			node: &ast.TypeNode{
				Kind:       ast.TypeEnum,
				EnumValues: []string{"a", "b"},
			},
			nullable: false,
			expected: `enum ["a", "b"]!`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, err := TypeFromASTNode(tt.node, tt.nullable)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr && typ.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, typ.String())
			}
		})
	}
}

// TestInferUnaryOperations tests unary expression type inference
func TestInferUnaryOperations(t *testing.T) {
	tc := NewTypeChecker()

	// Test negation
	neg := &ast.UnaryExpr{
		Operator: "-",
		Operand:  &ast.LiteralExpr{Value: 42},
		Loc:      ast.SourceLocation{Line: 1, Column: 1},
	}

	typ, err := tc.inferUnary(neg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if typ.String() != "int!" {
		t.Errorf("Expected int!, got %s", typ.String())
	}

	// Test logical not
	not := &ast.UnaryExpr{
		Operator: "not",
		Operand:  &ast.LiteralExpr{Value: true},
		Loc:      ast.SourceLocation{Line: 1, Column: 1},
	}

	typ, err = tc.inferUnary(not)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if typ.String() != "bool!" {
		t.Errorf("Expected bool!, got %s", typ.String())
	}

	// Test invalid negation on non-numeric
	tc2 := NewTypeChecker()
	invalid := &ast.UnaryExpr{
		Operator: "-",
		Operand:  &ast.LiteralExpr{Value: "hello"},
		Loc:      ast.SourceLocation{Line: 1, Column: 1},
	}

	_, _ = tc2.inferUnary(invalid)
	if len(tc2.errors) == 0 {
		t.Error("Expected error for negating non-numeric type")
	}
}

// TestInferLogical tests logical expression type inference
func TestInferLogical(t *testing.T) {
	tc := NewTypeChecker()

	logical := &ast.LogicalExpr{
		Left:     &ast.LiteralExpr{Value: true},
		Operator: "&&",
		Right:    &ast.LiteralExpr{Value: false},
		Loc:      ast.SourceLocation{Line: 1, Column: 1},
	}

	typ, err := tc.inferLogical(logical)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if typ.String() != "bool!" {
		t.Errorf("Expected bool!, got %s", typ.String())
	}
}

// TestInferHashLiteral tests hash literal type inference
func TestInferHashLiteral(t *testing.T) {
	tc := NewTypeChecker()

	// Non-empty hash
	hash := &ast.HashLiteralExpr{
		Pairs: []ast.HashPair{
			{
				Key:   &ast.LiteralExpr{Value: "name"},
				Value: &ast.LiteralExpr{Value: "Alice"},
			},
			{
				Key:   &ast.LiteralExpr{Value: "email"},
				Value: &ast.LiteralExpr{Value: "alice@example.com"},
			},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	typ, err := tc.inferHashLiteral(hash)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if typ.String() != "hash<string!, string!>!" {
		t.Errorf("Expected hash<string!, string!>!, got %s", typ.String())
	}

	// Empty hash
	emptyHash := &ast.HashLiteralExpr{
		Pairs: []ast.HashPair{},
		Loc:   ast.SourceLocation{Line: 1, Column: 1},
	}

	typ, err = tc.inferHashLiteral(emptyHash)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Empty hash should infer to hash<string!, any!>!
	if typ.IsNullable() {
		t.Error("Expected hash to be required")
	}
}

// TestInferParenExpr tests parenthesized expression type inference
func TestInferParenExpr(t *testing.T) {
	tc := NewTypeChecker()

	paren := &ast.ParenExpr{
		Expr: &ast.LiteralExpr{Value: 42},
		Loc:  ast.SourceLocation{Line: 1, Column: 1},
	}

	typ, err := tc.inferExpr(paren)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if typ.String() != "int!" {
		t.Errorf("Expected int!, got %s", typ.String())
	}
}

// TestInferInterpolatedString tests interpolated string type inference
func TestInferInterpolatedString(t *testing.T) {
	tc := NewTypeChecker()

	interp := &ast.InterpolatedStringExpr{
		Parts: []ast.ExprNode{
			&ast.LiteralExpr{Value: "Hello, "},
			&ast.LiteralExpr{Value: "World"},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	typ, err := tc.inferExpr(interp)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if typ.String() != "string!" {
		t.Errorf("Expected string!, got %s", typ.String())
	}
}

// TestCheckRescueStmt tests rescue statement type checking
func TestCheckRescueStmt(t *testing.T) {
	tc := NewTypeChecker()

	rescue := &ast.RescueStmt{
		Try: []ast.StmtNode{
			&ast.ExprStmt{Expr: &ast.LiteralExpr{Value: 1}},
		},
		ErrorVar: "err",
		RescueBody: []ast.StmtNode{
			&ast.ExprStmt{Expr: &ast.LiteralExpr{Value: 2}},
		},
		Loc: ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkStmt(rescue)

	// Rescue statements should not generate errors on their own
	if len(tc.errors) > 0 {
		t.Errorf("Unexpected errors: %v", tc.errors)
	}
}

// TestCheckBlockStmt tests block statement type checking
func TestCheckBlockStmt(t *testing.T) {
	tc := NewTypeChecker()

	block := &ast.BlockStmt{
		Statements: []ast.StmtNode{
			&ast.ExprStmt{Expr: &ast.LiteralExpr{Value: 1}},
			&ast.ExprStmt{Expr: &ast.LiteralExpr{Value: 2}},
		},
		IsAsync: false,
		Loc:     ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkStmt(block)

	if len(tc.errors) > 0 {
		t.Errorf("Unexpected errors: %v", tc.errors)
	}
}

// TestCheckReturnStmt tests return statement type checking
func TestCheckReturnStmt(t *testing.T) {
	tc := NewTypeChecker()

	ret := &ast.ReturnStmt{
		Value: &ast.LiteralExpr{Value: 42},
		Loc:   ast.SourceLocation{Line: 1, Column: 1},
	}

	tc.checkStmt(ret)

	if len(tc.errors) > 0 {
		t.Errorf("Unexpected errors: %v", tc.errors)
	}
}

// TestInvalidBinaryOperation tests invalid binary operations
func TestInvalidBinaryOperation(t *testing.T) {
	tc := NewTypeChecker()

	// Try to add a string and an int
	bin := &ast.BinaryExpr{
		Left:     &ast.LiteralExpr{Value: "hello"},
		Right:    &ast.LiteralExpr{Value: 42},
		Operator: "+",
		Loc:      ast.SourceLocation{Line: 1, Column: 1},
	}

	_, _ = tc.inferBinary(bin)

	if len(tc.errors) == 0 {
		t.Error("Expected error for invalid binary operation")
	}
}

// TestInvalidIndexOperation tests invalid index operations
func TestInvalidIndexOperation(t *testing.T) {
	tc := NewTypeChecker()

	// Try to index a string (not indexable)
	idx := &ast.IndexExpr{
		Object: &ast.LiteralExpr{Value: "hello"},
		Index:  &ast.LiteralExpr{Value: 0},
		Loc:    ast.SourceLocation{Line: 1, Column: 1},
	}

	_, _ = tc.inferIndex(idx)

	if len(tc.errors) == 0 {
		t.Error("Expected error for indexing non-indexable type")
	}
}
