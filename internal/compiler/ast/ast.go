// Package ast defines the Abstract Syntax Tree (AST) node types for the Conduit programming language.
// It provides structures for representing resources, fields, types, hooks, validations, and expressions.
package ast

import "github.com/conduit-lang/conduit/internal/compiler/lexer"

// SourceLocation tracks the position of an AST node in source code
type SourceLocation struct {
	Line   int // Line number (1-indexed)
	Column int // Column number (1-indexed)
}

// Node is the base interface for all AST nodes
type Node interface {
	Location() SourceLocation
	node()
}

// Program is the root node of the AST
type Program struct {
	Resources []*ResourceNode
}

func (p *Program) node() {}

// Location returns the source location of the program node in the AST.
func (p *Program) Location() SourceLocation {
	if len(p.Resources) > 0 {
		return p.Resources[0].Loc
	}
	return SourceLocation{Line: 1, Column: 1}
}

// ResourceNode represents a resource definition
type ResourceNode struct {
	Name          string
	Documentation string
	Fields        []*FieldNode
	Hooks         []*HookNode
	Validations   []*ValidationNode
	Constraints   []*ConstraintNode
	Relationships []*RelationshipNode
	Scopes        []*ScopeNode
	Computed      []*ComputedNode
	Operations    []string // List of allowed operations (create, update, delete, etc.)
	Middleware    []string // Middleware stack for this resource
	Loc           SourceLocation
}

func (r *ResourceNode) node() {}

// Location returns the source location of the resource node in the AST.
func (r *ResourceNode) Location() SourceLocation {
	return r.Loc
}

// FieldNode represents a field declaration in a resource
type FieldNode struct {
	Name        string
	Type        *TypeNode
	Nullable    bool              // true for ?, false for !
	Default     ExprNode          // Default value expression
	Constraints []*ConstraintNode // Field-level constraints (@min, @max, etc.)
	Loc         SourceLocation
}

func (f *FieldNode) node() {}

// Location returns the source location of the field node in the AST.
func (f *FieldNode) Location() SourceLocation {
	return f.Loc
}

// TypeKind represents the kind of type
type TypeKind int

const (
	// TypePrimitive represents primitive types (string, int, bool, etc.)
	TypePrimitive TypeKind = iota
	// TypeArray represents array types (array<T>)
	TypeArray
	// TypeHash represents hash/map types (hash<K,V>)
	TypeHash
	// TypeEnum represents inline enum types
	TypeEnum
	// TypeResource represents resource types (relationships)
	TypeResource
	// TypeStruct represents inline struct types
	TypeStruct
)

// TypeNode represents a type specification
type TypeNode struct {
	Kind         TypeKind
	Name         string       // Name of the type (e.g., "string", "User")
	Nullable     bool         // true for ?, false for !
	ElementType  *TypeNode    // For array<T>
	KeyType      *TypeNode    // For hash<K,V>
	ValueType    *TypeNode    // For hash<K,V>
	EnumValues   []string     // For inline enums
	StructFields []*FieldNode // For inline struct types
	Loc          SourceLocation
}

func (t *TypeNode) node() {}

// Location returns the source location of the type node in the AST.
func (t *TypeNode) Location() SourceLocation {
	return t.Loc
}

// HookNode represents a lifecycle hook (@before/@after create/update/delete/save)
type HookNode struct {
	Timing        string   // "before" or "after"
	Event         string   // "create", "update", "delete", "save"
	Middleware    []string // Middleware stack for this hook
	IsAsync       bool     // @async annotation
	IsTransaction bool     // @transaction annotation
	Body          []StmtNode
	Loc           SourceLocation
}

func (h *HookNode) node() {}

// Location returns the source location of the hook node in the AST.
func (h *HookNode) Location() SourceLocation {
	return h.Loc
}

// ValidationNode represents a validation block (@validate)
type ValidationNode struct {
	Name      string
	Condition ExprNode
	Error     string // Error message
	Loc       SourceLocation
}

func (v *ValidationNode) node() {}

// Location returns the source location of the validation node in the AST.
func (v *ValidationNode) Location() SourceLocation {
	return v.Loc
}

// ConstraintNode represents a constraint annotation or block
type ConstraintNode struct {
	Name      string     // Constraint name (e.g., "min", "max", "unique")
	Arguments []ExprNode // Arguments to the constraint
	On        []string   // Events this constraint applies to (create, update)
	When      ExprNode   // Condition for constraint
	Condition ExprNode   // Constraint condition
	Error     string     // Custom error message
	Loc       SourceLocation
}

func (c *ConstraintNode) node() {}

// Location returns the source location of the constraint node in the AST.
func (c *ConstraintNode) Location() SourceLocation {
	return c.Loc
}

// RelationshipNode represents a relationship between resources
type RelationshipNode struct {
	Name       string // Field name for the relationship
	Type       string // Target resource type
	Kind       RelationshipKind
	ForeignKey string // Foreign key column name
	Through    string // Join table name (for has-many-through)
	OnDelete   string // cascade, restrict, nullify
	Nullable   bool   // Whether the relationship is nullable
	Loc        SourceLocation
}

func (r *RelationshipNode) node() {}

// Location returns the source location of the relationship node in the AST.
func (r *RelationshipNode) Location() SourceLocation {
	return r.Loc
}

// RelationshipKind represents the type of relationship
type RelationshipKind int

const (
	// RelationshipBelongsTo represents a belongs-to relationship
	RelationshipBelongsTo RelationshipKind = iota
	// RelationshipHasMany represents a has-many relationship
	RelationshipHasMany
	// RelationshipHasManyThrough represents a has-many-through relationship
	RelationshipHasManyThrough
	// RelationshipHasOne represents a has-one relationship
	RelationshipHasOne
)

// ScopeNode represents a named scope definition (@scope)
type ScopeNode struct {
	Name      string
	Arguments []*ArgumentNode
	Condition ExprNode
	Loc       SourceLocation
}

func (s *ScopeNode) node() {}

// Location returns the source location of the scope node in the AST.
func (s *ScopeNode) Location() SourceLocation {
	return s.Loc
}

// ComputedNode represents a computed field (@computed)
type ComputedNode struct {
	Name string
	Type *TypeNode
	Body ExprNode
	Loc  SourceLocation
}

func (c *ComputedNode) node() {}

// Location returns the source location of the computed node in the AST.
func (c *ComputedNode) Location() SourceLocation {
	return c.Loc
}

// ArgumentNode represents a function argument
type ArgumentNode struct {
	Name    string
	Type    *TypeNode
	Default ExprNode
	Loc     SourceLocation
}

func (a *ArgumentNode) node() {}

// Location returns the source location of the argument node in the AST.
func (a *ArgumentNode) Location() SourceLocation {
	return a.Loc
}

// StmtNode is the interface for all statement nodes
type StmtNode interface {
	Node
	stmtNode()
}

// ExprStmt represents an expression used as a statement
type ExprStmt struct {
	Expr ExprNode
	Loc  SourceLocation
}

func (e *ExprStmt) node()     {}
func (e *ExprStmt) stmtNode() {}

// Location returns the source location of the expression statement in the AST.
func (e *ExprStmt) Location() SourceLocation {
	return e.Loc
}

// AssignmentStmt represents an assignment statement
type AssignmentStmt struct {
	Target ExprNode // Field access or identifier
	Value  ExprNode
	Loc    SourceLocation
}

func (a *AssignmentStmt) node()     {}
func (a *AssignmentStmt) stmtNode() {}

// Location returns the source location of the assignment statement in the AST.
func (a *AssignmentStmt) Location() SourceLocation {
	return a.Loc
}

// LetStmt represents a local variable declaration
type LetStmt struct {
	Name  string
	Type  *TypeNode // Optional type annotation
	Value ExprNode
	Loc   SourceLocation
}

func (l *LetStmt) node()     {}
func (l *LetStmt) stmtNode() {}

// Location returns the source location of the let statement in the AST.
func (l *LetStmt) Location() SourceLocation {
	return l.Loc
}

// ReturnStmt represents a return statement
type ReturnStmt struct {
	Value ExprNode
	Loc   SourceLocation
}

func (r *ReturnStmt) node()     {}
func (r *ReturnStmt) stmtNode() {}

// Location returns the source location of the return statement in the AST.
func (r *ReturnStmt) Location() SourceLocation {
	return r.Loc
}

// IfStmt represents an if/elsif/else statement
type IfStmt struct {
	Condition     ExprNode
	ThenBranch    []StmtNode
	ElsIfBranches []*ElsIfBranch
	ElseBranch    []StmtNode
	Loc           SourceLocation
}

func (i *IfStmt) node()     {}
func (i *IfStmt) stmtNode() {}

// Location returns the source location of the if statement in the AST.
func (i *IfStmt) Location() SourceLocation {
	return i.Loc
}

// ElsIfBranch represents an elsif clause
type ElsIfBranch struct {
	Condition ExprNode
	Body      []StmtNode
	Loc       SourceLocation
}

// BlockStmt represents a block of statements
type BlockStmt struct {
	Statements []StmtNode
	IsAsync    bool // @async block
	Loc        SourceLocation
}

func (b *BlockStmt) node()     {}
func (b *BlockStmt) stmtNode() {}

// Location returns the source location of the block statement in the AST.
func (b *BlockStmt) Location() SourceLocation {
	return b.Loc
}

// RescueStmt represents error handling (rescue block)
type RescueStmt struct {
	Try        []StmtNode
	ErrorVar   string // Variable name for caught error
	RescueBody []StmtNode
	Loc        SourceLocation
}

func (r *RescueStmt) node()     {}
func (r *RescueStmt) stmtNode() {}

// Location returns the source location of the rescue statement in the AST.
func (r *RescueStmt) Location() SourceLocation {
	return r.Loc
}

// MatchStmt represents a match/when statement
type MatchStmt struct {
	Value ExprNode
	Cases []*MatchCase
	Loc   SourceLocation
}

func (m *MatchStmt) node()     {}
func (m *MatchStmt) stmtNode() {}

// Location returns the source location of the match statement in the AST.
func (m *MatchStmt) Location() SourceLocation {
	return m.Loc
}

// MatchCase represents a single case in a match statement
type MatchCase struct {
	Pattern ExprNode // Pattern to match against
	Body    []StmtNode
	Loc     SourceLocation
}

// ExprNode is the interface for all expression nodes
type ExprNode interface {
	Node
	exprNode()
}

// TokenLocation creates a SourceLocation from a lexer token
func TokenLocation(token lexer.Token) SourceLocation {
	return SourceLocation{
		Line:   token.Line,
		Column: token.Column,
	}
}
