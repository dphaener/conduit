package parser

import "github.com/conduit-lang/conduit/compiler/lexer"

// ExprNode is the interface for all expression AST nodes
type ExprNode interface {
	exprNode()
	GetLocation() SourceLocation
}

// StmtNode is the interface for all statement AST nodes
type StmtNode interface {
	stmtNode()
	GetLocation() SourceLocation
}

// LiteralExpr represents a literal value (string, number, boolean, nil, array, hash)
type LiteralExpr struct {
	Value    interface{} // string, int64, float64, bool, nil, []ExprNode, map[string]ExprNode
	Location SourceLocation
}

func (e *LiteralExpr) exprNode()                    {}
func (e *LiteralExpr) GetLocation() SourceLocation  { return e.Location }

// IdentifierExpr represents an identifier reference
type IdentifierExpr struct {
	Name     string
	Location SourceLocation
}

func (e *IdentifierExpr) exprNode()                   {}
func (e *IdentifierExpr) GetLocation() SourceLocation { return e.Location }

// SelfExpr represents the `self` keyword
type SelfExpr struct {
	Location SourceLocation
}

func (e *SelfExpr) exprNode()                   {}
func (e *SelfExpr) GetLocation() SourceLocation { return e.Location }

// BinaryExpr represents a binary operation (e.g., a + b, a == b)
type BinaryExpr struct {
	Left     ExprNode
	Operator lexer.TokenType
	Right    ExprNode
	Location SourceLocation
}

func (e *BinaryExpr) exprNode()                   {}
func (e *BinaryExpr) GetLocation() SourceLocation { return e.Location }

// UnaryExpr represents a unary operation (e.g., !x, -x)
type UnaryExpr struct {
	Operator lexer.TokenType
	Operand  ExprNode
	Location SourceLocation
}

func (e *UnaryExpr) exprNode()                   {}
func (e *UnaryExpr) GetLocation() SourceLocation { return e.Location }

// CallExpr represents a function call (namespaced or method call)
type CallExpr struct {
	Namespace string      // For String.slugify(), namespace is "String"
	Function  string      // Function or method name
	Arguments []ExprNode
	Location  SourceLocation
}

func (e *CallExpr) exprNode()                   {}
func (e *CallExpr) GetLocation() SourceLocation { return e.Location }

// FieldAccessExpr represents field access (e.g., self.name, obj.field)
type FieldAccessExpr struct {
	Object   ExprNode // Can be SelfExpr, IdentifierExpr, or another FieldAccessExpr for chaining
	Field    string
	Location SourceLocation
}

func (e *FieldAccessExpr) exprNode()                   {}
func (e *FieldAccessExpr) GetLocation() SourceLocation { return e.Location }

// SafeNavigationExpr represents safe navigation (e.g., self.parent?.name)
type SafeNavigationExpr struct {
	Object   ExprNode
	Field    string
	Location SourceLocation
}

func (e *SafeNavigationExpr) exprNode()                   {}
func (e *SafeNavigationExpr) GetLocation() SourceLocation { return e.Location }

// IndexExpr represents array/hash indexing (e.g., arr[0], hash["key"])
type IndexExpr struct {
	Object   ExprNode
	Index    ExprNode
	Location SourceLocation
}

func (e *IndexExpr) exprNode()                   {}
func (e *IndexExpr) GetLocation() SourceLocation { return e.Location }

// TernaryExpr represents ternary conditional (condition ? true_val : false_val)
type TernaryExpr struct {
	Condition ExprNode
	TrueExpr  ExprNode
	FalseExpr ExprNode
	Location  SourceLocation
}

func (e *TernaryExpr) exprNode()                   {}
func (e *TernaryExpr) GetLocation() SourceLocation { return e.Location }

// CoalesceExpr represents null coalescing (left ?? right)
type CoalesceExpr struct {
	Left     ExprNode
	Right    ExprNode
	Location SourceLocation
}

func (e *CoalesceExpr) exprNode()                   {}
func (e *CoalesceExpr) GetLocation() SourceLocation { return e.Location }

// AssignmentExpr represents assignment (self.field = value)
type AssignmentExpr struct {
	Target   ExprNode // FieldAccessExpr or IdentifierExpr
	Value    ExprNode
	Location SourceLocation
}

func (e *AssignmentExpr) exprNode()                   {}
func (e *AssignmentExpr) GetLocation() SourceLocation { return e.Location }

// GroupExpr represents a parenthesized expression
type GroupExpr struct {
	Expr     ExprNode
	Location SourceLocation
}

func (e *GroupExpr) exprNode()                   {}
func (e *GroupExpr) GetLocation() SourceLocation { return e.Location }

// ArrayLiteralExpr represents an array literal [1, 2, 3]
type ArrayLiteralExpr struct {
	Elements []ExprNode
	Location SourceLocation
}

func (e *ArrayLiteralExpr) exprNode()                   {}
func (e *ArrayLiteralExpr) GetLocation() SourceLocation { return e.Location }

// HashLiteralExpr represents a hash literal {name: "Alice", age: 30}
type HashLiteralExpr struct {
	Pairs    []HashPair
	Location SourceLocation
}

type HashPair struct {
	Key   string   // Always a string (identifier or string literal)
	Value ExprNode
}

func (e *HashLiteralExpr) exprNode()                   {}
func (e *HashLiteralExpr) GetLocation() SourceLocation { return e.Location }

// StringInterpolationExpr represents string interpolation "Hello #{name}!"
type StringInterpolationExpr struct {
	Parts    []ExprNode // Alternating LiteralExpr (strings) and interpolated expressions
	Location SourceLocation
}

func (e *StringInterpolationExpr) exprNode()                   {}
func (e *StringInterpolationExpr) GetLocation() SourceLocation { return e.Location }

// MatchExpr represents a match expression
type MatchExpr struct {
	Value    ExprNode
	Cases    []MatchCase
	Location SourceLocation
}

type MatchCase struct {
	Pattern string   // The pattern to match (string literal)
	Expr    ExprNode
}

func (e *MatchExpr) exprNode()                   {}
func (e *MatchExpr) GetLocation() SourceLocation { return e.Location }

// IfExpr represents an if expression (can return a value)
type IfExpr struct {
	Condition ExprNode
	ThenBody  []StmtNode
	ElsifBranches []ElsifBranch
	ElseBody  []StmtNode
	Location  SourceLocation
}

type ElsifBranch struct {
	Condition ExprNode
	Body      []StmtNode
}

func (e *IfExpr) exprNode()                   {}
func (e *IfExpr) GetLocation() SourceLocation { return e.Location }

// UnlessExpr represents an unless expression
type UnlessExpr struct {
	Condition ExprNode
	Body      []StmtNode
	Location  SourceLocation
}

func (e *UnlessExpr) exprNode()                   {}
func (e *UnlessExpr) GetLocation() SourceLocation { return e.Location }

// MethodCallExpr represents a method call on an object (e.g., str.downcase())
type MethodCallExpr struct {
	Object    ExprNode
	Method    string
	Arguments []ExprNode
	Location  SourceLocation
}

func (e *MethodCallExpr) exprNode()                   {}
func (e *MethodCallExpr) GetLocation() SourceLocation { return e.Location }

// --- Statement Nodes ---

// ExprStmt represents an expression used as a statement
type ExprStmt struct {
	Expr     ExprNode
	Location SourceLocation
}

func (s *ExprStmt) stmtNode()                   {}
func (s *ExprStmt) GetLocation() SourceLocation { return s.Location }

// LetStmt represents a variable declaration (let x = expr)
type LetStmt struct {
	Name     string
	Value    ExprNode
	Location SourceLocation
}

func (s *LetStmt) stmtNode()                   {}
func (s *LetStmt) GetLocation() SourceLocation { return s.Location }

// ReturnStmt represents a return statement
type ReturnStmt struct {
	Value    ExprNode // Can be nil for empty return
	Location SourceLocation
}

func (s *ReturnStmt) stmtNode()                   {}
func (s *ReturnStmt) GetLocation() SourceLocation { return s.Location }

// IfStmt represents an if statement (no return value)
type IfStmt struct {
	Condition ExprNode
	ThenBody  []StmtNode
	ElsifBranches []ElsifBranch
	ElseBody  []StmtNode
	Location  SourceLocation
}

func (s *IfStmt) stmtNode()                   {}
func (s *IfStmt) GetLocation() SourceLocation { return s.Location }

// UnlessStmt represents an unless statement
type UnlessStmt struct {
	Condition ExprNode
	Body      []StmtNode
	Location  SourceLocation
}

func (s *UnlessStmt) stmtNode()                   {}
func (s *UnlessStmt) GetLocation() SourceLocation { return s.Location }

// --- Constructor functions ---

// NewLiteralExpr creates a new literal expression
func NewLiteralExpr(value interface{}, loc SourceLocation) *LiteralExpr {
	return &LiteralExpr{Value: value, Location: loc}
}

// NewIdentifierExpr creates a new identifier expression
func NewIdentifierExpr(name string, loc SourceLocation) *IdentifierExpr {
	return &IdentifierExpr{Name: name, Location: loc}
}

// NewSelfExpr creates a new self expression
func NewSelfExpr(loc SourceLocation) *SelfExpr {
	return &SelfExpr{Location: loc}
}

// NewBinaryExpr creates a new binary expression
func NewBinaryExpr(left ExprNode, operator lexer.TokenType, right ExprNode, loc SourceLocation) *BinaryExpr {
	return &BinaryExpr{Left: left, Operator: operator, Right: right, Location: loc}
}

// NewUnaryExpr creates a new unary expression
func NewUnaryExpr(operator lexer.TokenType, operand ExprNode, loc SourceLocation) *UnaryExpr {
	return &UnaryExpr{Operator: operator, Operand: operand, Location: loc}
}

// NewCallExpr creates a new function call expression
func NewCallExpr(namespace, function string, args []ExprNode, loc SourceLocation) *CallExpr {
	return &CallExpr{Namespace: namespace, Function: function, Arguments: args, Location: loc}
}

// NewFieldAccessExpr creates a new field access expression
func NewFieldAccessExpr(object ExprNode, field string, loc SourceLocation) *FieldAccessExpr {
	return &FieldAccessExpr{Object: object, Field: field, Location: loc}
}

// NewSafeNavigationExpr creates a new safe navigation expression
func NewSafeNavigationExpr(object ExprNode, field string, loc SourceLocation) *SafeNavigationExpr {
	return &SafeNavigationExpr{Object: object, Field: field, Location: loc}
}

// NewIndexExpr creates a new index expression
func NewIndexExpr(object ExprNode, index ExprNode, loc SourceLocation) *IndexExpr {
	return &IndexExpr{Object: object, Index: index, Location: loc}
}

// NewTernaryExpr creates a new ternary expression
func NewTernaryExpr(condition, trueExpr, falseExpr ExprNode, loc SourceLocation) *TernaryExpr {
	return &TernaryExpr{Condition: condition, TrueExpr: trueExpr, FalseExpr: falseExpr, Location: loc}
}

// NewCoalesceExpr creates a new null coalescing expression
func NewCoalesceExpr(left, right ExprNode, loc SourceLocation) *CoalesceExpr {
	return &CoalesceExpr{Left: left, Right: right, Location: loc}
}

// NewAssignmentExpr creates a new assignment expression
func NewAssignmentExpr(target, value ExprNode, loc SourceLocation) *AssignmentExpr {
	return &AssignmentExpr{Target: target, Value: value, Location: loc}
}

// NewGroupExpr creates a new grouped expression
func NewGroupExpr(expr ExprNode, loc SourceLocation) *GroupExpr {
	return &GroupExpr{Expr: expr, Location: loc}
}

// NewArrayLiteralExpr creates a new array literal expression
func NewArrayLiteralExpr(elements []ExprNode, loc SourceLocation) *ArrayLiteralExpr {
	return &ArrayLiteralExpr{Elements: elements, Location: loc}
}

// NewHashLiteralExpr creates a new hash literal expression
func NewHashLiteralExpr(pairs []HashPair, loc SourceLocation) *HashLiteralExpr {
	return &HashLiteralExpr{Pairs: pairs, Location: loc}
}

// NewStringInterpolationExpr creates a new string interpolation expression
func NewStringInterpolationExpr(parts []ExprNode, loc SourceLocation) *StringInterpolationExpr {
	return &StringInterpolationExpr{Parts: parts, Location: loc}
}

// NewMatchExpr creates a new match expression
func NewMatchExpr(value ExprNode, cases []MatchCase, loc SourceLocation) *MatchExpr {
	return &MatchExpr{Value: value, Cases: cases, Location: loc}
}

// NewIfExpr creates a new if expression
func NewIfExpr(condition ExprNode, thenBody []StmtNode, elsifBranches []ElsifBranch, elseBody []StmtNode, loc SourceLocation) *IfExpr {
	return &IfExpr{Condition: condition, ThenBody: thenBody, ElsifBranches: elsifBranches, ElseBody: elseBody, Location: loc}
}

// NewUnlessExpr creates a new unless expression
func NewUnlessExpr(condition ExprNode, body []StmtNode, loc SourceLocation) *UnlessExpr {
	return &UnlessExpr{Condition: condition, Body: body, Location: loc}
}

// NewMethodCallExpr creates a new method call expression
func NewMethodCallExpr(object ExprNode, method string, args []ExprNode, loc SourceLocation) *MethodCallExpr {
	return &MethodCallExpr{Object: object, Method: method, Arguments: args, Location: loc}
}

// NewExprStmt creates a new expression statement
func NewExprStmt(expr ExprNode, loc SourceLocation) *ExprStmt {
	return &ExprStmt{Expr: expr, Location: loc}
}

// NewLetStmt creates a new let statement
func NewLetStmt(name string, value ExprNode, loc SourceLocation) *LetStmt {
	return &LetStmt{Name: name, Value: value, Location: loc}
}

// NewReturnStmt creates a new return statement
func NewReturnStmt(value ExprNode, loc SourceLocation) *ReturnStmt {
	return &ReturnStmt{Value: value, Location: loc}
}

// NewIfStmt creates a new if statement
func NewIfStmt(condition ExprNode, thenBody []StmtNode, elsifBranches []ElsifBranch, elseBody []StmtNode, loc SourceLocation) *IfStmt {
	return &IfStmt{Condition: condition, ThenBody: thenBody, ElsifBranches: elsifBranches, ElseBody: elseBody, Location: loc}
}

// NewUnlessStmt creates a new unless statement
func NewUnlessStmt(condition ExprNode, body []StmtNode, loc SourceLocation) *UnlessStmt {
	return &UnlessStmt{Condition: condition, Body: body, Location: loc}
}
