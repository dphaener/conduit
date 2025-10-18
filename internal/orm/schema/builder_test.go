package schema

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TestBuildField tests field building from AST nodes
func TestBuildField(t *testing.T) {
	tests := []struct {
		name      string
		fieldNode *ast.FieldNode
		wantErr   bool
		validate  func(*testing.T, *Field)
	}{
		{
			name: "simple string field",
			fieldNode: &ast.FieldNode{
				Name: "title",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Constraints: []*ast.ConstraintNode{},
				Loc:         ast.SourceLocation{Line: 1, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, f *Field) {
				if f.Name != "title" {
					t.Errorf("expected name 'title', got %s", f.Name)
				}
				if f.Type.BaseType != TypeString {
					t.Errorf("expected TypeString, got %v", f.Type.BaseType)
				}
				if f.Type.Nullable {
					t.Error("expected non-nullable")
				}
				if !f.Type.NullabilitySet {
					t.Error("expected NullabilitySet to be true")
				}
			},
		},
		{
			name: "nullable text field",
			fieldNode: &ast.FieldNode{
				Name: "bio",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "text",
					Nullable: true,
				},
				Constraints: []*ast.ConstraintNode{},
				Loc:         ast.SourceLocation{Line: 2, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, f *Field) {
				if f.Name != "bio" {
					t.Errorf("expected name 'bio', got %s", f.Name)
				}
				if f.Type.BaseType != TypeText {
					t.Errorf("expected TypeText, got %v", f.Type.BaseType)
				}
				if !f.Type.Nullable {
					t.Error("expected nullable")
				}
				if !f.Type.NullabilitySet {
					t.Error("expected NullabilitySet to be true")
				}
			},
		},
		{
			name: "field with constraints",
			fieldNode: &ast.FieldNode{
				Name: "age",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "int",
					Nullable: false,
				},
				Constraints: []*ast.ConstraintNode{
					{
						Name: "min",
						Arguments: []ast.ExprNode{
							&ast.LiteralExpr{Value: 0},
						},
						Loc: ast.SourceLocation{Line: 3, Column: 10},
					},
					{
						Name: "max",
						Arguments: []ast.ExprNode{
							&ast.LiteralExpr{Value: 150},
						},
						Loc: ast.SourceLocation{Line: 3, Column: 20},
					},
				},
				Loc: ast.SourceLocation{Line: 3, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, f *Field) {
				if len(f.Constraints) != 2 {
					t.Errorf("expected 2 constraints, got %d", len(f.Constraints))
				}
				if f.Constraints[0].Type != ConstraintMin {
					t.Errorf("expected ConstraintMin, got %v", f.Constraints[0].Type)
				}
				if f.Constraints[1].Type != ConstraintMax {
					t.Errorf("expected ConstraintMax, got %v", f.Constraints[1].Type)
				}
			},
		},
		{
			name: "invalid primitive type",
			fieldNode: &ast.FieldNode{
				Name: "invalid",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "unknown_type",
					Nullable: false,
				},
				Constraints: []*ast.ConstraintNode{},
				Loc:         ast.SourceLocation{Line: 4, Column: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			field, err := b.buildField(tt.fieldNode)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, field)
			}
		})
	}
}

// TestBuildTypeSpec tests type spec building
func TestBuildTypeSpec(t *testing.T) {
	tests := []struct {
		name     string
		typeNode *ast.TypeNode
		wantErr  bool
		validate func(*testing.T, *TypeSpec)
	}{
		{
			name: "primitive string type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypePrimitive,
				Name:     "string",
				Nullable: false,
			},
			wantErr: false,
			validate: func(t *testing.T, ts *TypeSpec) {
				if ts.BaseType != TypeString {
					t.Errorf("expected TypeString, got %v", ts.BaseType)
				}
				if !ts.NullabilitySet {
					t.Error("expected NullabilitySet to be true")
				}
			},
		},
		{
			name: "array type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeArray,
				Nullable: false,
				ElementType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "int",
					Nullable: false,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, ts *TypeSpec) {
				if ts.ArrayElement == nil {
					t.Error("expected ArrayElement to be set")
					return
				}
				if ts.ArrayElement.BaseType != TypeInt {
					t.Errorf("expected element type TypeInt, got %v", ts.ArrayElement.BaseType)
				}
				if !ts.ArrayElement.NullabilitySet {
					t.Error("expected element NullabilitySet to be true")
				}
			},
		},
		{
			name: "hash type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeHash,
				Nullable: false,
				KeyType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				ValueType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "int",
					Nullable: false,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, ts *TypeSpec) {
				if ts.HashKey == nil || ts.HashValue == nil {
					t.Error("expected HashKey and HashValue to be set")
					return
				}
				if ts.HashKey.BaseType != TypeString {
					t.Errorf("expected key type TypeString, got %v", ts.HashKey.BaseType)
				}
				if ts.HashValue.BaseType != TypeInt {
					t.Errorf("expected value type TypeInt, got %v", ts.HashValue.BaseType)
				}
			},
		},
		{
			name: "enum type",
			typeNode: &ast.TypeNode{
				Kind:       ast.TypeEnum,
				Nullable:   false,
				EnumValues: []string{"active", "inactive", "pending"},
			},
			wantErr: false,
			validate: func(t *testing.T, ts *TypeSpec) {
				if ts.BaseType != TypeEnum {
					t.Errorf("expected TypeEnum, got %v", ts.BaseType)
				}
				if len(ts.EnumValues) != 3 {
					t.Errorf("expected 3 enum values, got %d", len(ts.EnumValues))
				}
			},
		},
		{
			name: "struct type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeStruct,
				Nullable: false,
				StructFields: []*ast.FieldNode{
					{
						Name: "lat",
						Type: &ast.TypeNode{
							Kind:     ast.TypePrimitive,
							Name:     "float",
							Nullable: false,
						},
						Constraints: []*ast.ConstraintNode{},
						Loc:         ast.SourceLocation{Line: 1, Column: 1},
					},
					{
						Name: "lng",
						Type: &ast.TypeNode{
							Kind:     ast.TypePrimitive,
							Name:     "float",
							Nullable: false,
						},
						Constraints: []*ast.ConstraintNode{},
						Loc:         ast.SourceLocation{Line: 1, Column: 2},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, ts *TypeSpec) {
				if len(ts.StructFields) != 2 {
					t.Errorf("expected 2 struct fields, got %d", len(ts.StructFields))
				}
				if ts.StructFields["lat"] == nil {
					t.Error("expected 'lat' field in struct")
				}
				if ts.StructFields["lng"] == nil {
					t.Error("expected 'lng' field in struct")
				}
			},
		},
		{
			name: "array without element type",
			typeNode: &ast.TypeNode{
				Kind:        ast.TypeArray,
				Nullable:    false,
				ElementType: nil,
			},
			wantErr: true,
		},
		{
			name: "hash without key type",
			typeNode: &ast.TypeNode{
				Kind:     ast.TypeHash,
				Nullable: false,
				KeyType:  nil,
				ValueType: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "int",
					Nullable: false,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			typeSpec, err := b.buildTypeSpec(tt.typeNode)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, typeSpec)
			}
		})
	}
}

// TestBuildConstraint tests constraint building
func TestBuildConstraint(t *testing.T) {
	tests := []struct {
		name           string
		constraintNode *ast.ConstraintNode
		wantErr        bool
		validate       func(*testing.T, Constraint)
	}{
		{
			name: "min constraint",
			constraintNode: &ast.ConstraintNode{
				Name: "min",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: 5},
				},
				Loc: ast.SourceLocation{Line: 1, Column: 10},
			},
			wantErr: false,
			validate: func(t *testing.T, c Constraint) {
				if c.Type != ConstraintMin {
					t.Errorf("expected ConstraintMin, got %v", c.Type)
				}
				if c.Value != 5 {
					t.Errorf("expected value 5, got %v", c.Value)
				}
			},
		},
		{
			name: "max constraint",
			constraintNode: &ast.ConstraintNode{
				Name: "max",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: 100},
				},
				Loc: ast.SourceLocation{Line: 1, Column: 20},
			},
			wantErr: false,
			validate: func(t *testing.T, c Constraint) {
				if c.Type != ConstraintMax {
					t.Errorf("expected ConstraintMax, got %v", c.Type)
				}
				if c.Value != 100 {
					t.Errorf("expected value 100, got %v", c.Value)
				}
			},
		},
		{
			name: "pattern constraint",
			constraintNode: &ast.ConstraintNode{
				Name: "pattern",
				Arguments: []ast.ExprNode{
					&ast.LiteralExpr{Value: "^[a-z]+$"},
				},
				Loc: ast.SourceLocation{Line: 1, Column: 30},
			},
			wantErr: false,
			validate: func(t *testing.T, c Constraint) {
				if c.Type != ConstraintPattern {
					t.Errorf("expected ConstraintPattern, got %v", c.Type)
				}
				if c.Value != "^[a-z]+$" {
					t.Errorf("expected regex pattern, got %v", c.Value)
				}
			},
		},
		{
			name: "unique constraint",
			constraintNode: &ast.ConstraintNode{
				Name:      "unique",
				Arguments: []ast.ExprNode{},
				Loc:       ast.SourceLocation{Line: 1, Column: 40},
			},
			wantErr: false,
			validate: func(t *testing.T, c Constraint) {
				if c.Type != ConstraintUnique {
					t.Errorf("expected ConstraintUnique, got %v", c.Type)
				}
			},
		},
		{
			name: "primary constraint",
			constraintNode: &ast.ConstraintNode{
				Name:      "primary",
				Arguments: []ast.ExprNode{},
				Loc:       ast.SourceLocation{Line: 1, Column: 50},
			},
			wantErr: false,
			validate: func(t *testing.T, c Constraint) {
				if c.Type != ConstraintPrimary {
					t.Errorf("expected ConstraintPrimary, got %v", c.Type)
				}
			},
		},
		{
			name: "auto constraint",
			constraintNode: &ast.ConstraintNode{
				Name:      "auto",
				Arguments: []ast.ExprNode{},
				Loc:       ast.SourceLocation{Line: 1, Column: 60},
			},
			wantErr: false,
			validate: func(t *testing.T, c Constraint) {
				if c.Type != ConstraintAuto {
					t.Errorf("expected ConstraintAuto, got %v", c.Type)
				}
			},
		},
		{
			name: "unknown constraint",
			constraintNode: &ast.ConstraintNode{
				Name:      "invalid_constraint",
				Arguments: []ast.ExprNode{},
				Loc:       ast.SourceLocation{Line: 1, Column: 70},
			},
			wantErr: true,
		},
		{
			name: "constraint with identifier expression (should error)",
			constraintNode: &ast.ConstraintNode{
				Name: "min",
				Arguments: []ast.ExprNode{
					&ast.IdentifierExpr{Name: "some_var"},
				},
				Loc: ast.SourceLocation{Line: 1, Column: 80},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			constraint, err := b.buildConstraint(tt.constraintNode)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, constraint)
			}
		})
	}
}

// TestBuildRelationship tests relationship building
func TestBuildRelationship(t *testing.T) {
	tests := []struct {
		name    string
		relNode *ast.RelationshipNode
		wantErr bool
		validate func(*testing.T, *Relationship)
	}{
		{
			name: "belongs_to relationship",
			relNode: &ast.RelationshipNode{
				Kind:       ast.RelationshipBelongsTo,
				Name:       "author",
				Type:       "User",
				Nullable:   false,
				ForeignKey: "author_id",
				OnDelete:   "restrict",
				Loc:        ast.SourceLocation{Line: 5, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, r *Relationship) {
				if r.Type != RelationshipBelongsTo {
					t.Errorf("expected RelationshipBelongsTo, got %v", r.Type)
				}
				if r.FieldName != "author" {
					t.Errorf("expected field name 'author', got %s", r.FieldName)
				}
				if r.TargetResource != "User" {
					t.Errorf("expected target 'User', got %s", r.TargetResource)
				}
				if r.OnDelete != CascadeRestrict {
					t.Errorf("expected CascadeRestrict, got %v", r.OnDelete)
				}
			},
		},
		{
			name: "has_many relationship",
			relNode: &ast.RelationshipNode{
				Kind:     ast.RelationshipHasMany,
				Name:     "posts",
				Type:     "Post",
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 6, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, r *Relationship) {
				if r.Type != RelationshipHasMany {
					t.Errorf("expected RelationshipHasMany, got %v", r.Type)
				}
				if r.FieldName != "posts" {
					t.Errorf("expected field name 'posts', got %s", r.FieldName)
				}
			},
		},
		{
			name: "has_one relationship",
			relNode: &ast.RelationshipNode{
				Kind:     ast.RelationshipHasOne,
				Name:     "profile",
				Type:     "Profile",
				Nullable: true,
				Loc:      ast.SourceLocation{Line: 7, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, r *Relationship) {
				if r.Type != RelationshipHasOne {
					t.Errorf("expected RelationshipHasOne, got %v", r.Type)
				}
				if !r.Nullable {
					t.Error("expected nullable relationship")
				}
			},
		},
		{
			name: "has_many_through relationship",
			relNode: &ast.RelationshipNode{
				Kind:     ast.RelationshipHasManyThrough,
				Name:     "tags",
				Type:     "Tag",
				Through:  "PostTag",
				Nullable: false,
				Loc:      ast.SourceLocation{Line: 8, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, r *Relationship) {
				if r.Type != RelationshipHasManyThrough {
					t.Errorf("expected RelationshipHasManyThrough, got %v", r.Type)
				}
				if r.ThroughResource != "PostTag" {
					t.Errorf("expected through resource 'PostTag', got %s", r.ThroughResource)
				}
			},
		},
		{
			name: "invalid cascade action",
			relNode: &ast.RelationshipNode{
				Kind:       ast.RelationshipBelongsTo,
				Name:       "author",
				Type:       "User",
				Nullable:   false,
				ForeignKey: "author_id",
				OnDelete:   "invalid_action",
				Loc:        ast.SourceLocation{Line: 9, Column: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			rel, err := b.buildRelationship(tt.relNode)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, rel)
			}
		})
	}
}

// TestBuildHook tests hook building
func TestBuildHook(t *testing.T) {
	tests := []struct {
		name     string
		hookNode *ast.HookNode
		wantErr  bool
		validate func(*testing.T, *Hook)
	}{
		{
			name: "before_create hook",
			hookNode: &ast.HookNode{
				Timing:        "before",
				Event:         "create",
				IsTransaction: false,
				IsAsync:       false,
				Body:          []ast.StmtNode{},
				Loc:           ast.SourceLocation{Line: 10, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, h *Hook) {
				if h.Type != BeforeCreate {
					t.Errorf("expected BeforeCreate, got %v", h.Type)
				}
				if h.Transaction {
					t.Error("expected Transaction to be false")
				}
			},
		},
		{
			name: "after_create with transaction",
			hookNode: &ast.HookNode{
				Timing:        "after",
				Event:         "create",
				IsTransaction: true,
				IsAsync:       false,
				Body:          []ast.StmtNode{},
				Loc:           ast.SourceLocation{Line: 11, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, h *Hook) {
				if h.Type != AfterCreate {
					t.Errorf("expected AfterCreate, got %v", h.Type)
				}
				if !h.Transaction {
					t.Error("expected Transaction to be true")
				}
			},
		},
		{
			name: "after_save with async",
			hookNode: &ast.HookNode{
				Timing:        "after",
				Event:         "save",
				IsTransaction: false,
				IsAsync:       true,
				Body:          []ast.StmtNode{},
				Loc:           ast.SourceLocation{Line: 12, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, h *Hook) {
				if h.Type != AfterSave {
					t.Errorf("expected AfterSave, got %v", h.Type)
				}
				if !h.Async {
					t.Error("expected Async to be true")
				}
			},
		},
		{
			name: "invalid hook type",
			hookNode: &ast.HookNode{
				Timing:        "during",
				Event:         "create",
				IsTransaction: false,
				IsAsync:       false,
				Body:          []ast.StmtNode{},
				Loc:           ast.SourceLocation{Line: 13, Column: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			hook, err := b.buildHook(tt.hookNode)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, hook)
			}
		})
	}
}

// TestBuildScope tests scope building
func TestBuildScope(t *testing.T) {
	tests := []struct {
		name      string
		scopeNode *ast.ScopeNode
		wantErr   bool
		validate  func(*testing.T, *Scope)
	}{
		{
			name: "simple scope",
			scopeNode: &ast.ScopeNode{
				Name:      "active",
				Arguments: []*ast.ArgumentNode{},
				Loc:       ast.SourceLocation{Line: 14, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, s *Scope) {
				if s.Name != "active" {
					t.Errorf("expected name 'active', got %s", s.Name)
				}
				if len(s.Arguments) != 0 {
					t.Errorf("expected 0 arguments, got %d", len(s.Arguments))
				}
			},
		},
		{
			name: "scope with arguments",
			scopeNode: &ast.ScopeNode{
				Name: "by_status",
				Arguments: []*ast.ArgumentNode{
					{
						Name: "status",
						Type: &ast.TypeNode{
							Kind:     ast.TypePrimitive,
							Name:     "string",
							Nullable: false,
						},
						Loc: ast.SourceLocation{Line: 15, Column: 10},
					},
				},
				Loc: ast.SourceLocation{Line: 15, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, s *Scope) {
				if len(s.Arguments) != 1 {
					t.Errorf("expected 1 argument, got %d", len(s.Arguments))
				}
				if s.Arguments[0].Name != "status" {
					t.Errorf("expected argument name 'status', got %s", s.Arguments[0].Name)
				}
			},
		},
		{
			name: "scope with default argument",
			scopeNode: &ast.ScopeNode{
				Name: "recent",
				Arguments: []*ast.ArgumentNode{
					{
						Name: "days",
						Type: &ast.TypeNode{
							Kind:     ast.TypePrimitive,
							Name:     "int",
							Nullable: false,
						},
						Default: &ast.LiteralExpr{Value: 7},
						Loc:     ast.SourceLocation{Line: 16, Column: 10},
					},
				},
				Loc: ast.SourceLocation{Line: 16, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, s *Scope) {
				if s.Arguments[0].Default != 7 {
					t.Errorf("expected default value 7, got %v", s.Arguments[0].Default)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			scope, err := b.buildScope(tt.scopeNode)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, scope)
			}
		})
	}
}

// TestBuildComputedField tests computed field building
func TestBuildComputedField(t *testing.T) {
	tests := []struct {
		name         string
		computedNode *ast.ComputedNode
		wantErr      bool
		validate     func(*testing.T, *ComputedField)
	}{
		{
			name: "simple computed field",
			computedNode: &ast.ComputedNode{
				Name: "full_name",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Body: &ast.LiteralExpr{Value: "computed"},
				Loc:  ast.SourceLocation{Line: 17, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, cf *ComputedField) {
				if cf.Name != "full_name" {
					t.Errorf("expected name 'full_name', got %s", cf.Name)
				}
				if cf.Type.BaseType != TypeString {
					t.Errorf("expected TypeString, got %v", cf.Type.BaseType)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			computed, err := b.buildComputedField(tt.computedNode)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, computed)
			}
		})
	}
}

// TestBuild tests the full Build method
func TestBuild(t *testing.T) {
	tests := []struct {
		name         string
		resourceNode *ast.ResourceNode
		wantErr      bool
		validate     func(*testing.T, *ResourceSchema)
	}{
		{
			name: "complete resource",
			resourceNode: &ast.ResourceNode{
				Name:          "Post",
				Documentation: "A blog post",
				Fields: []*ast.FieldNode{
					{
						Name: "id",
						Type: &ast.TypeNode{
							Kind:     ast.TypePrimitive,
							Name:     "uuid",
							Nullable: false,
						},
						Constraints: []*ast.ConstraintNode{},
						Loc:         ast.SourceLocation{Line: 1, Column: 1},
					},
					{
						Name: "title",
						Type: &ast.TypeNode{
							Kind:     ast.TypePrimitive,
							Name:     "string",
							Nullable: false,
						},
						Constraints: []*ast.ConstraintNode{},
						Loc:         ast.SourceLocation{Line: 2, Column: 1},
					},
				},
				Relationships: []*ast.RelationshipNode{},
				Hooks:         []*ast.HookNode{},
				Constraints:   []*ast.ConstraintNode{},
				Scopes:        []*ast.ScopeNode{},
				Computed:      []*ast.ComputedNode{},
				Loc:           ast.SourceLocation{Line: 1, Column: 1},
			},
			wantErr: false,
			validate: func(t *testing.T, rs *ResourceSchema) {
				if rs.Name != "Post" {
					t.Errorf("expected name 'Post', got %s", rs.Name)
				}
				if rs.Documentation != "A blog post" {
					t.Errorf("expected documentation 'A blog post', got %s", rs.Documentation)
				}
				if len(rs.Fields) != 2 {
					t.Errorf("expected 2 fields, got %d", len(rs.Fields))
				}
				if rs.TableName != "post" {
					t.Errorf("expected table name 'post', got %s", rs.TableName)
				}
			},
		},
		{
			name: "resource with invalid field",
			resourceNode: &ast.ResourceNode{
				Name: "Invalid",
				Fields: []*ast.FieldNode{
					{
						Name: "bad_field",
						Type: &ast.TypeNode{
							Kind:     ast.TypePrimitive,
							Name:     "unknown_type",
							Nullable: false,
						},
						Constraints: []*ast.ConstraintNode{},
						Loc:         ast.SourceLocation{Line: 1, Column: 1},
					},
				},
				Relationships: []*ast.RelationshipNode{},
				Hooks:         []*ast.HookNode{},
				Constraints:   []*ast.ConstraintNode{},
				Scopes:        []*ast.ScopeNode{},
				Computed:      []*ast.ComputedNode{},
				Loc:           ast.SourceLocation{Line: 1, Column: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			schema, err := b.Build(tt.resourceNode)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, schema)
			}
		})
	}
}

// TestExtractValue tests extractValue error handling
func TestExtractValue(t *testing.T) {
	tests := []struct {
		name    string
		expr    ast.ExprNode
		wantErr bool
		want    interface{}
	}{
		{
			name:    "literal expression",
			expr:    &ast.LiteralExpr{Value: 42},
			wantErr: false,
			want:    42,
		},
		{
			name:    "string literal",
			expr:    &ast.LiteralExpr{Value: "hello"},
			wantErr: false,
			want:    "hello",
		},
		{
			name:    "identifier expression should error",
			expr:    &ast.IdentifierExpr{Name: "var_name"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			value, err := b.extractValue(tt.expr)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if value != tt.want {
				t.Errorf("expected value %v, got %v", tt.want, value)
			}
		})
	}
}

// TestBuilderErrorAccumulation tests that builder accumulates all errors
func TestBuilderErrorAccumulation(t *testing.T) {
	resourceNode := &ast.ResourceNode{
		Name: "ErrorTest",
		Fields: []*ast.FieldNode{
			{
				Name: "field1",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "unknown1",
					Nullable: false,
				},
				Constraints: []*ast.ConstraintNode{},
				Loc:         ast.SourceLocation{Line: 1, Column: 1},
			},
			{
				Name: "field2",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "unknown2",
					Nullable: false,
				},
				Constraints: []*ast.ConstraintNode{},
				Loc:         ast.SourceLocation{Line: 2, Column: 1},
			},
		},
		Relationships: []*ast.RelationshipNode{},
		Hooks:         []*ast.HookNode{},
		Constraints:   []*ast.ConstraintNode{},
		Scopes:        []*ast.ScopeNode{},
		Computed:      []*ast.ComputedNode{},
		Loc:           ast.SourceLocation{Line: 1, Column: 1},
	}

	b := NewBuilder()
	_, err := b.Build(resourceNode)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check that error message contains information about multiple errors
	errMsg := err.Error()
	if !contains(errMsg, "2 errors") {
		t.Errorf("expected error message to mention '2 errors', got: %s", errMsg)
	}

	// Check that both field names appear in the error message
	if !contains(errMsg, "field1") || !contains(errMsg, "field2") {
		t.Errorf("expected error message to mention both fields, got: %s", errMsg)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
