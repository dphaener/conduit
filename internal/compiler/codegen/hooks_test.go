package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestGenerateHooks_BeforeCreate(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			{Name: "slug", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
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
						},
					},
				},
			},
		},
	}

	gen := NewGenerator()
	hooksCode := gen.generateHooks(resource)

	// Verify method name
	if !strings.Contains(hooksCode, "func (p *Post) BeforeCreate(ctx context.Context, db *sql.DB) error") {
		t.Error("Generated code should contain BeforeCreate method")
	}

	// Verify slug assignment
	if !strings.Contains(hooksCode, "p.Slug = stdlib.StringSlugify(p.Title)") {
		t.Error("Generated code should contain slug assignment")
	}
}

func TestGenerateHooks_AfterCreateWithTransaction(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{Name: "email", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
		},
		Hooks: []*ast.HookNode{
			{
				Timing:        "after",
				Event:         "create",
				IsTransaction: true,
				Body: []ast.StmtNode{
					&ast.ExprStmt{
						Expr: &ast.CallExpr{
							Namespace: "Logger",
							Function:  "info",
							Arguments: []ast.ExprNode{
								&ast.LiteralExpr{Value: "User created"},
							},
						},
					},
				},
			},
		},
	}

	gen := NewGenerator()
	hooksCode := gen.generateHooks(resource)

	// Verify method name
	if !strings.Contains(hooksCode, "func (u *User) AfterCreate(ctx context.Context, db *sql.DB) error") {
		t.Error("Generated code should contain AfterCreate method")
	}

	// Verify transaction wrapper
	if !strings.Contains(hooksCode, "tx, err := db.Begin()") {
		t.Error("Generated code should begin transaction")
	}

	if !strings.Contains(hooksCode, "defer tx.Rollback()") {
		t.Error("Generated code should defer rollback")
	}
}

func TestGenerateHooks_WithAsyncBlock(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Order",
		Fields: []*ast.FieldNode{
			{Name: "total", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "float"}, Nullable: false},
		},
		Hooks: []*ast.HookNode{
			{
				Timing:        "after",
				Event:         "create",
				IsTransaction: true,
				Body: []ast.StmtNode{
					&ast.BlockStmt{
						IsAsync: true,
						Statements: []ast.StmtNode{
							&ast.ExprStmt{
								Expr: &ast.CallExpr{
									Namespace: "Logger",
									Function:  "info",
									Arguments: []ast.ExprNode{
										&ast.LiteralExpr{Value: "Async task"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	gen := NewGenerator()
	hooksCode := gen.generateHooks(resource)

	// Verify async block with goroutine
	if !strings.Contains(hooksCode, "go func()") {
		t.Error("Generated code should contain goroutine for async block")
	}

	// Verify async resource copy
	if !strings.Contains(hooksCode, "asyncResource := *o") {
		t.Error("Generated code should copy resource for async access")
	}
}

func TestGenerateStatement_IfStatement(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Product",
		Fields: []*ast.FieldNode{
			{Name: "price", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "float"}, Nullable: false},
			{Name: "discount", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "float"}, Nullable: true},
		},
	}

	gen := NewGenerator()
	gen.reset()

	stmt := &ast.IfStmt{
		Condition: &ast.BinaryExpr{
			Left: &ast.FieldAccessExpr{
				Object: &ast.SelfExpr{},
				Field:  "price",
			},
			Operator: ">",
			Right:    &ast.LiteralExpr{Value: float64(100)},
		},
		ThenBranch: []ast.StmtNode{
			&ast.AssignmentStmt{
				Target: &ast.FieldAccessExpr{
					Object: &ast.SelfExpr{},
					Field:  "discount",
				},
				Value: &ast.LiteralExpr{Value: float64(10)},
			},
		},
	}

	gen.generateStatement(resource, stmt)
	code := gen.buf.String()

	// Verify if statement structure
	if !strings.Contains(code, "if p.Price > 100") {
		t.Errorf("Generated code should contain if condition, got: %s", code)
	}

	if !strings.Contains(code, "p.Discount = 10") {
		t.Errorf("Generated code should contain assignment in then branch, got: %s", code)
	}
}

func TestGenerateStatement_LetStatement(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Article",
		Fields: []*ast.FieldNode{
			{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
		},
	}

	gen := NewGenerator()
	gen.reset()

	stmt := &ast.LetStmt{
		Name: "slug",
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
	}

	gen.generateStatement(resource, stmt)
	code := gen.buf.String()

	// Verify let statement
	if !strings.Contains(code, "slug := stdlib.StringSlugify(a.Title)") {
		t.Errorf("Generated code should contain let statement, got: %s", code)
	}
}
