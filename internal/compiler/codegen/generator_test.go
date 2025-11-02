package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestGenerateResource_SimpleStruct(t *testing.T) {
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

	// Verify package declaration
	if !strings.Contains(code, "package models") {
		t.Error("Generated code should contain package declaration")
	}

	// Verify struct definition
	if !strings.Contains(code, "type User struct {") {
		t.Error("Generated code should contain struct definition")
	}

	// Verify fields (with potential alignment padding)
	if !strings.Contains(code, "Username") {
		t.Error("Generated code should contain Username field")
	}

	if !strings.Contains(code, "Email") {
		t.Error("Generated code should contain Email field")
	}

	// Verify field types
	if !strings.Contains(code, "string") {
		t.Error("Generated code should contain string type")
	}

	// Verify struct tags
	if !strings.Contains(code, "`db:\"username\" json:\"username\"`") {
		t.Error("Generated code should contain struct tags for username")
	}

	// Verify methods
	if !strings.Contains(code, "func (u *User) TableName() string") {
		t.Error("Generated code should contain TableName method")
	}

	if !strings.Contains(code, "func (u *User) Validate() error") {
		t.Error("Generated code should contain Validate method")
	}

	if !strings.Contains(code, "func (u *User) Create(ctx context.Context, db *sql.DB) error") {
		t.Error("Generated code should contain Create method")
	}
}

func TestGenerateResource_NullableFields(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Post",
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

	// Verify nullable field uses pointer type (with potential alignment padding)
	if !strings.Contains(code, "Bio") || !strings.Contains(code, "*string") {
		t.Error("Generated code should use *string for nullable text field")
	}

	// Verify JSON tag includes omitempty for nullable
	if !strings.Contains(code, "json:\"bio,omitempty\"") {
		t.Error("Generated code should include omitempty for nullable field")
	}
}

func TestGenerateResource_WithConstraints(t *testing.T) {
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
				},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Verify validation includes min/max checks
	if !strings.Contains(code, "if len(u.Username) < 3") {
		t.Error("Generated code should include min length validation")
	}

	if !strings.Contains(code, "if len(u.Username) > 50") {
		t.Error("Generated code should include max length validation")
	}
}

func TestGenerateMigrations_SimpleTable(t *testing.T) {
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
				},
				{
					Name: "email",
					Type: &ast.TypeNode{
						Kind:     ast.TypePrimitive,
						Name:     "string",
						Nullable: false,
					},
					Nullable: false,
					Constraints: []*ast.ConstraintNode{
						{Name: "unique"},
					},
				},
			},
		},
	}

	gen := NewGenerator()
	sql, err := gen.GenerateMigrations(resources)
	if err != nil {
		t.Fatalf("GenerateMigrations failed: %v", err)
	}

	// Verify CREATE TABLE statement
	if !strings.Contains(sql, "CREATE TABLE users") {
		t.Error("Generated SQL should contain CREATE TABLE users")
	}

	// Verify columns
	if !strings.Contains(sql, "username VARCHAR") {
		t.Error("Generated SQL should contain username column")
	}

	if !strings.Contains(sql, "email VARCHAR") {
		t.Error("Generated SQL should contain email column")
	}

	// Verify NOT NULL constraints
	if !strings.Contains(sql, "NOT NULL") {
		t.Error("Generated SQL should contain NOT NULL constraints")
	}

	// Verify UNIQUE index for email
	if !strings.Contains(sql, "CREATE UNIQUE INDEX") {
		t.Error("Generated SQL should create unique index for email")
	}
}

func TestToGoType(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		field    *ast.FieldNode
		expected string
	}{
		{
			name: "required string",
			field: &ast.FieldNode{
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: false,
			},
			expected: "string",
		},
		{
			name: "nullable string",
			field: &ast.FieldNode{
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"},
				Nullable: true,
			},
			expected: "*string",
		},
		{
			name: "required int",
			field: &ast.FieldNode{
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"},
				Nullable: false,
			},
			expected: "int64",
		},
		{
			name: "nullable int",
			field: &ast.FieldNode{
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"},
				Nullable: true,
			},
			expected: "*int64",
		},
		{
			name: "required uuid",
			field: &ast.FieldNode{
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid"},
				Nullable: false,
			},
			expected: "uuid.UUID",
		},
		{
			name: "nullable uuid",
			field: &ast.FieldNode{
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid"},
				Nullable: true,
			},
			expected: "*uuid.UUID",
		},
		{
			name: "required timestamp",
			field: &ast.FieldNode{
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "timestamp"},
				Nullable: false,
			},
			expected: "time.Time",
		},
		{
			name: "nullable timestamp",
			field: &ast.FieldNode{
				Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "timestamp"},
				Nullable: true,
			},
			expected: "*time.Time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.toGoType(tt.field)
			if result != tt.expected {
				t.Errorf("toGoType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToGoFieldName(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		input    string
		expected string
	}{
		{"username", "Username"},
		{"email_address", "EmailAddress"},
		{"created_at", "CreatedAt"},
		{"is_admin", "IsAdmin"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := gen.toGoFieldName(tt.input)
			if result != tt.expected {
				t.Errorf("toGoFieldName(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToTableName(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		input    string
		expected string
	}{
		{"User", "users"},
		{"Post", "posts"},
		{"Comment", "comments"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := gen.toTableName(tt.input)
			if result != tt.expected {
				t.Errorf("toTableName(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
