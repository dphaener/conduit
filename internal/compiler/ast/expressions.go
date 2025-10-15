package ast

// LiteralExpr represents a literal value (string, int, float, bool, null)
type LiteralExpr struct {
	Value interface{} // The actual value
	Loc   SourceLocation
}

func (l *LiteralExpr) node()     {}
func (l *LiteralExpr) exprNode() {}

// Location returns the source location of the literal expression in the AST.
func (l *LiteralExpr) Location() SourceLocation {
	return l.Loc
}

// IdentifierExpr represents a variable or field reference
type IdentifierExpr struct {
	Name string
	Loc  SourceLocation
}

func (i *IdentifierExpr) node()     {}
func (i *IdentifierExpr) exprNode() {}

// Location returns the source location of the identifier expression in the AST.
func (i *IdentifierExpr) Location() SourceLocation {
	return i.Loc
}

// BinaryExpr represents a binary operation (a + b, a == b, etc.)
type BinaryExpr struct {
	Left     ExprNode
	Operator string // "+", "-", "*", "/", "==", "!=", "<", ">", etc.
	Right    ExprNode
	Loc      SourceLocation
}

func (b *BinaryExpr) node()     {}
func (b *BinaryExpr) exprNode() {}

// Location returns the source location of the binary expression in the AST.
func (b *BinaryExpr) Location() SourceLocation {
	return b.Loc
}

// UnaryExpr represents a unary operation (!x, -x)
type UnaryExpr struct {
	Operator string // "!", "-", "not"
	Operand  ExprNode
	Loc      SourceLocation
}

func (u *UnaryExpr) node()     {}
func (u *UnaryExpr) exprNode() {}

// Location returns the source location of the unary expression in the AST.
func (u *UnaryExpr) Location() SourceLocation {
	return u.Loc
}

// LogicalExpr represents logical operations (and, or)
type LogicalExpr struct {
	Left     ExprNode
	Operator string // "and", "or", "&&", "||"
	Right    ExprNode
	Loc      SourceLocation
}

func (l *LogicalExpr) node()     {}
func (l *LogicalExpr) exprNode() {}

// Location returns the source location of the logical expression in the AST.
func (l *LogicalExpr) Location() SourceLocation {
	return l.Loc
}

// CallExpr represents a function call
type CallExpr struct {
	Namespace string     // Optional namespace (e.g., "String" in "String.slugify()")
	Function  string     // Function name
	Arguments []ExprNode // Function arguments
	Loc       SourceLocation
}

func (c *CallExpr) node()     {}
func (c *CallExpr) exprNode() {}

// Location returns the source location of the call expression in the AST.
func (c *CallExpr) Location() SourceLocation {
	return c.Loc
}

// FieldAccessExpr represents field access (self.title, user.email)
type FieldAccessExpr struct {
	Object ExprNode // The object being accessed
	Field  string   // Field name
	Loc    SourceLocation
}

func (f *FieldAccessExpr) node()     {}
func (f *FieldAccessExpr) exprNode() {}

// Location returns the source location of the field access expression in the AST.
func (f *FieldAccessExpr) Location() SourceLocation {
	return f.Loc
}

// SafeNavigationExpr represents safe navigation (?.)
type SafeNavigationExpr struct {
	Object ExprNode
	Field  string
	Loc    SourceLocation
}

func (s *SafeNavigationExpr) node()     {}
func (s *SafeNavigationExpr) exprNode() {}

// Location returns the source location of the safe navigation expression in the AST.
func (s *SafeNavigationExpr) Location() SourceLocation {
	return s.Loc
}

// ArrayLiteralExpr represents an array literal [1, 2, 3]
type ArrayLiteralExpr struct {
	Elements []ExprNode
	Loc      SourceLocation
}

func (a *ArrayLiteralExpr) node()     {}
func (a *ArrayLiteralExpr) exprNode() {}

// Location returns the source location of the array literal expression in the AST.
func (a *ArrayLiteralExpr) Location() SourceLocation {
	return a.Loc
}

// HashLiteralExpr represents a hash literal {key: value}
type HashLiteralExpr struct {
	Pairs []HashPair
	Loc   SourceLocation
}

func (h *HashLiteralExpr) node()     {}
func (h *HashLiteralExpr) exprNode() {}

// Location returns the source location of the hash literal expression in the AST.
func (h *HashLiteralExpr) Location() SourceLocation {
	return h.Loc
}

// HashPair represents a key-value pair in a hash literal
type HashPair struct {
	Key   ExprNode
	Value ExprNode
	Loc   SourceLocation
}

// IndexExpr represents array/hash indexing (arr[0], hash["key"])
type IndexExpr struct {
	Object ExprNode
	Index  ExprNode
	Loc    SourceLocation
}

func (i *IndexExpr) node()     {}
func (i *IndexExpr) exprNode() {}

// Location returns the source location of the index expression in the AST.
func (i *IndexExpr) Location() SourceLocation {
	return i.Loc
}

// NullCoalesceExpr represents null coalescing operator (??)
type NullCoalesceExpr struct {
	Left  ExprNode
	Right ExprNode
	Loc   SourceLocation
}

func (n *NullCoalesceExpr) node()     {}
func (n *NullCoalesceExpr) exprNode() {}

// Location returns the source location of the null coalesce expression in the AST.
func (n *NullCoalesceExpr) Location() SourceLocation {
	return n.Loc
}

// ParenExpr represents a parenthesized expression
type ParenExpr struct {
	Expr ExprNode
	Loc  SourceLocation
}

func (p *ParenExpr) node()     {}
func (p *ParenExpr) exprNode() {}

// Location returns the source location of the parenthesized expression in the AST.
func (p *ParenExpr) Location() SourceLocation {
	return p.Loc
}

// SelfExpr represents the 'self' keyword
type SelfExpr struct {
	Loc SourceLocation
}

func (s *SelfExpr) node()     {}
func (s *SelfExpr) exprNode() {}

// Location returns the source location of the self expression in the AST.
func (s *SelfExpr) Location() SourceLocation {
	return s.Loc
}

// InterpolatedStringExpr represents a string with interpolation
type InterpolatedStringExpr struct {
	Parts []ExprNode // Alternates between string literals and expressions
	Loc   SourceLocation
}

func (i *InterpolatedStringExpr) node()     {}
func (i *InterpolatedStringExpr) exprNode() {}

// Location returns the source location of the interpolated string expression in the AST.
func (i *InterpolatedStringExpr) Location() SourceLocation {
	return i.Loc
}

// RangeExpr represents a range (1..10, 1...10)
type RangeExpr struct {
	Start     ExprNode
	End       ExprNode
	Exclusive bool // true for ..., false for ..
	Loc       SourceLocation
}

func (r *RangeExpr) node()     {}
func (r *RangeExpr) exprNode() {}

// Location returns the source location of the range expression in the AST.
func (r *RangeExpr) Location() SourceLocation {
	return r.Loc
}

// LambdaExpr represents a lambda/closure expression
type LambdaExpr struct {
	Parameters []*ArgumentNode
	Body       []StmtNode
	Loc        SourceLocation
}

func (l *LambdaExpr) node()     {}
func (l *LambdaExpr) exprNode() {}

// Location returns the source location of the lambda expression in the AST.
func (l *LambdaExpr) Location() SourceLocation {
	return l.Loc
}
