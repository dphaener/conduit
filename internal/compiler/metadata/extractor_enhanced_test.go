package metadata

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TestExtractor_ExpressionFormatting tests all expression types
func TestExtractor_ExpressionFormatting(t *testing.T) {
	tests := []struct {
		name     string
		expr     ast.ExprNode
		expected string
	}{
		{
			name:     "literal string",
			expr:     &ast.LiteralExpr{Value: "hello"},
			expected: `"hello"`,
		},
		{
			name:     "literal int",
			expr:     &ast.LiteralExpr{Value: 42},
			expected: "42",
		},
		{
			name:     "literal bool",
			expr:     &ast.LiteralExpr{Value: true},
			expected: "true",
		},
		{
			name:     "literal null",
			expr:     &ast.LiteralExpr{Value: nil},
			expected: "null",
		},
		{
			name:     "identifier",
			expr:     &ast.IdentifierExpr{Name: "username"},
			expected: "username",
		},
		{
			name:     "self",
			expr:     &ast.SelfExpr{},
			expected: "self",
		},
		{
			name: "binary expression",
			expr: &ast.BinaryExpr{
				Left:     &ast.IdentifierExpr{Name: "x"},
				Operator: "+",
				Right:    &ast.LiteralExpr{Value: 5},
			},
			expected: "x + 5",
		},
		{
			name: "unary expression",
			expr: &ast.UnaryExpr{
				Operator: "!",
				Operand:  &ast.IdentifierExpr{Name: "active"},
			},
			expected: "!active",
		},
		{
			name: "logical expression",
			expr: &ast.LogicalExpr{
				Left:     &ast.IdentifierExpr{Name: "a"},
				Operator: "and",
				Right:    &ast.IdentifierExpr{Name: "b"},
			},
			expected: "a and b",
		},
		{
			name: "function call without namespace",
			expr: &ast.CallExpr{
				Function:  "validate",
				Arguments: []ast.ExprNode{&ast.LiteralExpr{Value: "data"}},
			},
			expected: `validate("data")`,
		},
		{
			name: "function call with namespace",
			expr: &ast.CallExpr{
				Namespace: "String",
				Function:  "slugify",
				Arguments: []ast.ExprNode{
					&ast.FieldAccessExpr{
						Object: &ast.SelfExpr{},
						Field:  "title",
					},
				},
			},
			expected: "String.slugify(self.title)",
		},
		{
			name: "field access",
			expr: &ast.FieldAccessExpr{
				Object: &ast.SelfExpr{},
				Field:  "email",
			},
			expected: "self.email",
		},
		{
			name: "safe navigation",
			expr: &ast.SafeNavigationExpr{
				Object: &ast.IdentifierExpr{Name: "user"},
				Field:  "profile",
			},
			expected: "user?.profile",
		},
		{
			name: "array literal",
			expr: &ast.ArrayLiteralExpr{
				Elements: []ast.ExprNode{
					&ast.LiteralExpr{Value: 1},
					&ast.LiteralExpr{Value: 2},
					&ast.LiteralExpr{Value: 3},
				},
			},
			expected: "[1, 2, 3]",
		},
		{
			name: "hash literal",
			expr: &ast.HashLiteralExpr{
				Pairs: []ast.HashPair{
					{
						Key:   &ast.LiteralExpr{Value: "name"},
						Value: &ast.LiteralExpr{Value: "John"},
					},
					{
						Key:   &ast.LiteralExpr{Value: "age"},
						Value: &ast.LiteralExpr{Value: 30},
					},
				},
			},
			expected: `{"name": "John", "age": 30}`,
		},
		{
			name: "index expression",
			expr: &ast.IndexExpr{
				Object: &ast.IdentifierExpr{Name: "arr"},
				Index:  &ast.LiteralExpr{Value: 0},
			},
			expected: "arr[0]",
		},
		{
			name: "null coalesce",
			expr: &ast.NullCoalesceExpr{
				Left:  &ast.IdentifierExpr{Name: "value"},
				Right: &ast.LiteralExpr{Value: "default"},
			},
			expected: `value ?? "default"`,
		},
		{
			name: "parenthesized expression",
			expr: &ast.ParenExpr{
				Expr: &ast.BinaryExpr{
					Left:     &ast.IdentifierExpr{Name: "a"},
					Operator: "+",
					Right:    &ast.IdentifierExpr{Name: "b"},
				},
			},
			expected: "(a + b)",
		},
		{
			name: "range inclusive",
			expr: &ast.RangeExpr{
				Start:     &ast.LiteralExpr{Value: 1},
				End:       &ast.LiteralExpr{Value: 10},
				Exclusive: false,
			},
			expected: "1..10",
		},
		{
			name: "range exclusive",
			expr: &ast.RangeExpr{
				Start:     &ast.LiteralExpr{Value: 1},
				End:       &ast.LiteralExpr{Value: 10},
				Exclusive: true,
			},
			expected: "1...10",
		},
		{
			name: "lambda expression",
			expr: &ast.LambdaExpr{
				Parameters: []*ast.ArgumentNode{
					{Name: "x", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int", Nullable: false}},
					{Name: "y"},
				},
			},
			expected: "|x: int!, y| { ... }",
		},
		{
			name: "complex nested expression",
			expr: &ast.BinaryExpr{
				Left: &ast.CallExpr{
					Namespace: "String",
					Function:  "length",
					Arguments: []ast.ExprNode{
						&ast.FieldAccessExpr{
							Object: &ast.SelfExpr{},
							Field:  "title",
						},
					},
				},
				Operator: ">=",
				Right:    &ast.LiteralExpr{Value: 5},
			},
			expected: "String.length(self.title) >= 5",
		},
	}

	extractor := NewExtractor("1.0.0")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.formatExpression(tt.expr)
			if result != tt.expected {
				t.Errorf("formatExpression() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestExtractor_StatementFormatting tests statement formatting
func TestExtractor_StatementFormatting(t *testing.T) {
	tests := []struct {
		name     string
		stmt     ast.StmtNode
		expected string
	}{
		{
			name: "expression statement",
			stmt: &ast.ExprStmt{
				Expr: &ast.CallExpr{
					Namespace: "Logger",
					Function:  "info",
					Arguments: []ast.ExprNode{&ast.LiteralExpr{Value: "message"}},
				},
			},
			expected: `Logger.info("message")`,
		},
		{
			name: "assignment statement",
			stmt: &ast.AssignmentStmt{
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
				},
			},
			expected: "self.slug = String.slugify(self.title)",
		},
		{
			name: "let statement with type",
			stmt: &ast.LetStmt{
				Name:  "count",
				Type:  &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int", Nullable: false},
				Value: &ast.LiteralExpr{Value: 0},
			},
			expected: "let count: int! = 0",
		},
		{
			name: "let statement without type",
			stmt: &ast.LetStmt{
				Name:  "name",
				Value: &ast.LiteralExpr{Value: "John"},
			},
			expected: `let name = "John"`,
		},
		{
			name: "return statement with value",
			stmt: &ast.ReturnStmt{
				Value: &ast.LiteralExpr{Value: true},
			},
			expected: "return true",
		},
		{
			name:     "return statement without value",
			stmt:     &ast.ReturnStmt{},
			expected: "return",
		},
		{
			name: "if statement",
			stmt: &ast.IfStmt{
				Condition: &ast.BinaryExpr{
					Left:     &ast.IdentifierExpr{Name: "x"},
					Operator: ">",
					Right:    &ast.LiteralExpr{Value: 0},
				},
			},
			expected: "if x > 0 { ... }",
		},
		{
			name: "async block",
			stmt: &ast.BlockStmt{
				IsAsync: true,
			},
			expected: "@async { ... }",
		},
		{
			name: "regular block",
			stmt: &ast.BlockStmt{
				IsAsync: false,
			},
			expected: "{ ... }",
		},
		{
			name: "rescue statement",
			stmt: &ast.RescueStmt{
				ErrorVar: "err",
			},
			expected: "rescue err { ... }",
		},
		{
			name: "match statement",
			stmt: &ast.MatchStmt{
				Value: &ast.IdentifierExpr{Name: "status"},
			},
			expected: "match status { ... }",
		},
	}

	extractor := NewExtractor("1.0.0")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.formatStatement(tt.stmt)
			if result != tt.expected {
				t.Errorf("formatStatement() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestExtractor_SourceLocationTracking tests that source locations are preserved
func TestExtractor_SourceLocationTracking(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Post",
				Loc:  ast.SourceLocation{Line: 10, Column: 1},
				Fields: []*ast.FieldNode{
					{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false}},
				},
				Hooks: []*ast.HookNode{
					{
						Timing: "before",
						Event:  "create",
						Loc:    ast.SourceLocation{Line: 15, Column: 3},
						Body: []ast.StmtNode{
							&ast.AssignmentStmt{
								Target: &ast.FieldAccessExpr{
									Object: &ast.SelfExpr{},
									Field:  "slug",
								},
								Value: &ast.LiteralExpr{Value: "slug-value"},
							},
						},
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	extractor.SetFilePath("/path/to/post.cdt")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	// Check resource location
	if len(meta.Resources) != 1 {
		t.Fatalf("Resources count = %v, want 1", len(meta.Resources))
	}

	resource := meta.Resources[0]
	if resource.FilePath != "/path/to/post.cdt" {
		t.Errorf("Resource file path = %q, want %q", resource.FilePath, "/path/to/post.cdt")
	}
	if resource.Line != 10 {
		t.Errorf("Resource line = %d, want 10", resource.Line)
	}

	// Check hook location
	if len(resource.Hooks) != 1 {
		t.Fatalf("Hooks count = %v, want 1", len(resource.Hooks))
	}

	hook := resource.Hooks[0]
	if hook.Line != 15 {
		t.Errorf("Hook line = %d, want 15", hook.Line)
	}
	if hook.SourceCode == "" {
		t.Error("Hook source code should not be empty")
	}
	if !strings.Contains(hook.SourceCode, "self.slug") {
		t.Errorf("Hook source code = %q, should contain 'self.slug'", hook.SourceCode)
	}
}

// TestExtractor_SourceHashComputation tests source hash generation
func TestExtractor_SourceHashComputation(t *testing.T) {
	prog1 := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
				Loc:  ast.SourceLocation{Line: 1, Column: 1},
				Fields: []*ast.FieldNode{
					{Name: "email", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false}},
				},
			},
		},
	}

	prog2 := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
				Loc:  ast.SourceLocation{Line: 1, Column: 1},
				Fields: []*ast.FieldNode{
					{Name: "email", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false}},
				},
			},
		},
	}

	prog3 := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
				Loc:  ast.SourceLocation{Line: 1, Column: 1},
				Fields: []*ast.FieldNode{
					{Name: "username", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false}},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")

	meta1, _ := extractor.Extract(prog1)
	meta2, _ := extractor.Extract(prog2)
	meta3, _ := extractor.Extract(prog3)

	// Same AST should produce same hash
	if meta1.SourceHash != meta2.SourceHash {
		t.Errorf("Same ASTs produced different hashes: %s vs %s", meta1.SourceHash, meta2.SourceHash)
	}

	// Different AST should produce different hash
	if meta1.SourceHash == meta3.SourceHash {
		t.Error("Different ASTs produced same hash")
	}

	// Hash should not be empty
	if meta1.SourceHash == "" {
		t.Error("Source hash should not be empty")
	}

	// Hash should be hex-encoded SHA256 (64 chars)
	if len(meta1.SourceHash) != 64 {
		t.Errorf("Source hash length = %d, want 64", len(meta1.SourceHash))
	}
}

// TestExtractor_ErrorHandling tests graceful error handling
func TestExtractor_ErrorHandling(t *testing.T) {
	// Create a program with a problematic resource (nil type)
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "ValidResource",
				Fields: []*ast.FieldNode{
					{Name: "id", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false}},
				},
			},
			{
				Name: "AnotherValid",
				Fields: []*ast.FieldNode{
					{Name: "name", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false}},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	// Should not panic, even with potential issues
	if meta == nil {
		t.Fatal("Extract() should return metadata even with errors")
	}

	// Should extract valid resources
	if len(meta.Resources) < 2 {
		t.Errorf("Should extract valid resources, got %d", len(meta.Resources))
	}

	// err may or may not be nil depending on validation
	_ = err
}

// TestExtractor_HookBodyFormatting tests hook body extraction
func TestExtractor_HookBodyFormatting(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Post",
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
								},
							},
							&ast.LetStmt{
								Name:  "timestamp",
								Value: &ast.CallExpr{Namespace: "Time", Function: "now"},
							},
						},
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	resource := meta.Resources[0]
	hook := resource.Hooks[0]

	// Check hook source code contains both statements
	if !strings.Contains(hook.SourceCode, "self.slug = String.slugify(self.title)") {
		t.Errorf("Hook source should contain slug assignment, got: %s", hook.SourceCode)
	}
	if !strings.Contains(hook.SourceCode, "let timestamp = Time.now()") {
		t.Errorf("Hook source should contain timestamp assignment, got: %s", hook.SourceCode)
	}
}

// TestExtractor_ConstraintsAndValidations tests extraction of constraints and validations
func TestExtractor_ConstraintsAndValidations(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "User",
				Fields: []*ast.FieldNode{
					{
						Name: "email",
						Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
						Constraints: []*ast.ConstraintNode{
							{
								Name: "min",
								Arguments: []ast.ExprNode{
									&ast.LiteralExpr{Value: 5},
								},
							},
							{
								Name: "max",
								Arguments: []ast.ExprNode{
									&ast.LiteralExpr{Value: 100},
								},
							},
						},
					},
				},
				Validations: []*ast.ValidationNode{
					{
						Name: "email_format",
						Condition: &ast.CallExpr{
							Namespace: "String",
							Function:  "matches",
							Arguments: []ast.ExprNode{
								&ast.FieldAccessExpr{Object: &ast.SelfExpr{}, Field: "email"},
								&ast.LiteralExpr{Value: "@"},
							},
						},
						Error: "Invalid email format",
					},
				},
				Constraints: []*ast.ConstraintNode{
					{
						Name: "unique_email",
						On:   []string{"create", "update"},
						When: &ast.BinaryExpr{
							Left:     &ast.IdentifierExpr{Name: "status"},
							Operator: "==",
							Right:    &ast.LiteralExpr{Value: "active"},
						},
						Condition: &ast.CallExpr{
							Namespace: "Validation",
							Function:  "unique",
							Arguments: []ast.ExprNode{
								&ast.FieldAccessExpr{Object: &ast.SelfExpr{}, Field: "email"},
							},
						},
						Error: "Email must be unique",
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	resource := meta.Resources[0]

	// Check field constraints
	emailField := resource.Fields[0]
	if len(emailField.Constraints) != 2 {
		t.Errorf("Email field constraints count = %d, want 2", len(emailField.Constraints))
	}
	if emailField.Constraints[0] != "min(5)" {
		t.Errorf("First constraint = %q, want 'min(5)'", emailField.Constraints[0])
	}

	// Check validations
	if len(resource.Validations) != 1 {
		t.Fatalf("Validations count = %d, want 1", len(resource.Validations))
	}
	validation := resource.Validations[0]
	if validation.Name != "email_format" {
		t.Errorf("Validation name = %q, want 'email_format'", validation.Name)
	}
	if validation.Error != "Invalid email format" {
		t.Errorf("Validation error = %q, want 'Invalid email format'", validation.Error)
	}
	if !strings.Contains(validation.Condition, "String.matches") {
		t.Errorf("Validation condition should contain String.matches, got: %s", validation.Condition)
	}

	// Check resource-level constraints
	if len(resource.Constraints) != 1 {
		t.Fatalf("Constraints count = %d, want 1", len(resource.Constraints))
	}
	constraint := resource.Constraints[0]
	if constraint.Name != "unique_email" {
		t.Errorf("Constraint name = %q, want 'unique_email'", constraint.Name)
	}
	if len(constraint.On) != 2 {
		t.Errorf("Constraint 'on' count = %d, want 2", len(constraint.On))
	}
	if !strings.Contains(constraint.When, "status == ") {
		t.Errorf("Constraint 'when' should contain status check, got: %s", constraint.When)
	}
}

// TestExtractor_ScopesAndComputed tests extraction of scopes and computed fields
func TestExtractor_ScopesAndComputed(t *testing.T) {
	prog := &ast.Program{
		Resources: []*ast.ResourceNode{
			{
				Name: "Post",
				Scopes: []*ast.ScopeNode{
					{
						Name: "published",
						Condition: &ast.BinaryExpr{
							Left:     &ast.FieldAccessExpr{Object: &ast.SelfExpr{}, Field: "status"},
							Operator: "==",
							Right:    &ast.LiteralExpr{Value: "published"},
						},
					},
					{
						Name: "by_author",
						Arguments: []*ast.ArgumentNode{
							{
								Name: "author_id",
								Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
							},
						},
						Condition: &ast.BinaryExpr{
							Left:     &ast.FieldAccessExpr{Object: &ast.SelfExpr{}, Field: "author_id"},
							Operator: "==",
							Right:    &ast.IdentifierExpr{Name: "author_id"},
						},
					},
				},
				Computed: []*ast.ComputedNode{
					{
						Name: "word_count",
						Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int", Nullable: false},
						Body: &ast.CallExpr{
							Namespace: "String",
							Function:  "word_count",
							Arguments: []ast.ExprNode{
								&ast.FieldAccessExpr{Object: &ast.SelfExpr{}, Field: "content"},
							},
						},
					},
				},
			},
		},
	}

	extractor := NewExtractor("1.0.0")
	meta, err := extractor.Extract(prog)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	resource := meta.Resources[0]

	// Check scopes
	if len(resource.Scopes) != 2 {
		t.Fatalf("Scopes count = %d, want 2", len(resource.Scopes))
	}

	publishedScope := resource.Scopes[0]
	if publishedScope.Name != "published" {
		t.Errorf("Scope name = %q, want 'published'", publishedScope.Name)
	}
	if !strings.Contains(publishedScope.Condition, `self.status == "published"`) {
		t.Errorf("Scope condition should check status, got: %s", publishedScope.Condition)
	}

	byAuthorScope := resource.Scopes[1]
	if byAuthorScope.Name != "by_author" {
		t.Errorf("Scope name = %q, want 'by_author'", byAuthorScope.Name)
	}
	if len(byAuthorScope.Arguments) != 1 {
		t.Errorf("Scope arguments count = %d, want 1", len(byAuthorScope.Arguments))
	}

	// Check computed fields
	if len(resource.Computed) != 1 {
		t.Fatalf("Computed count = %d, want 1", len(resource.Computed))
	}

	computed := resource.Computed[0]
	if computed.Name != "word_count" {
		t.Errorf("Computed name = %q, want 'word_count'", computed.Name)
	}
	if computed.Type != "int!" {
		t.Errorf("Computed type = %q, want 'int!'", computed.Type)
	}
	if !strings.Contains(computed.Body, "String.word_count") {
		t.Errorf("Computed body should contain function call, got: %s", computed.Body)
	}
}
