package parser

import (
	"testing"

	"github.com/conduit-lang/conduit/compiler/lexer"
)

// Helper function to create a parser from source
func parseExpr(source string) (ExprNode, []ParseError) {
	l := lexer.New(source, "test.cdt")
	tokens, _ := l.ScanTokens()
	p := New(tokens)
	expr := p.parseExpression()
	return expr, p.errors
}

// Helper function to parse a statement
func parseStmt(source string) (StmtNode, []ParseError) {
	l := lexer.New(source, "test.cdt")
	tokens, _ := l.ScanTokens()
	p := New(tokens)
	stmt := p.parseStatement()
	return stmt, p.errors
}

// --- Literal Tests ---

func TestParseLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
		isUnary  bool
	}{
		{"integer", "42", int64(42), false},
		{"negative integer", "-17", nil, true}, // Unary minus
		{"float", "3.14", float64(3.14), false},
		{"string", `"hello"`, "hello", false},
		{"true", "true", true, false},
		{"false", "false", false, false},
		{"nil", "nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			if tt.isUnary {
				// Special case for unary expressions
				if _, ok := expr.(*UnaryExpr); !ok {
					t.Errorf("Expected UnaryExpr, got %T", expr)
				}
				return
			}

			lit, ok := expr.(*LiteralExpr)
			if !ok {
				t.Fatalf("Expected LiteralExpr, got %T", expr)
			}

			if lit.Value != tt.expected {
				t.Errorf("Expected value %v, got %v", tt.expected, lit.Value)
			}
		})
	}
}

func TestParseArrayLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // number of elements
	}{
		{"empty array", "[]", 0},
		{"single element", "[1]", 1},
		{"multiple elements", "[1, 2, 3]", 3},
		{"mixed types", `[1, "hello", true]`, 3},
		{"nested arrays", "[[1, 2], [3, 4]]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			arr, ok := expr.(*ArrayLiteralExpr)
			if !ok {
				t.Fatalf("Expected ArrayLiteralExpr, got %T", expr)
			}

			if len(arr.Elements) != tt.expected {
				t.Errorf("Expected %d elements, got %d", tt.expected, len(arr.Elements))
			}
		})
	}
}

func TestParseHashLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // number of pairs
	}{
		{"empty hash", "{}", 0},
		{"single pair", `{name: "Alice"}`, 1},
		{"multiple pairs", `{name: "Alice", age: 30}`, 2},
		{"nested hash", `{user: {name: "Alice", age: 30}}`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			hash, ok := expr.(*HashLiteralExpr)
			if !ok {
				t.Fatalf("Expected HashLiteralExpr, got %T", expr)
			}

			if len(hash.Pairs) != tt.expected {
				t.Errorf("Expected %d pairs, got %d", tt.expected, len(hash.Pairs))
			}
		})
	}
}

// --- Operator Precedence Tests ---

func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // description of expected structure
	}{
		{
			name:     "addition and multiplication",
			input:    "1 + 2 * 3",
			expected: "1 + (2 * 3)",
		},
		{
			name:     "multiplication and addition",
			input:    "2 * 3 + 4",
			expected: "(2 * 3) + 4",
		},
		{
			name:     "exponentiation right associative",
			input:    "2 ** 3 ** 4",
			expected: "2 ** (3 ** 4)",
		},
		{
			name:     "comparison and equality",
			input:    "a < b == c > d",
			expected: "(a < b) == (c > d)",
		},
		{
			name:     "logical and/or",
			input:    "a || b && c",
			expected: "a || (b && c)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			// Verify it's a binary expression
			if _, ok := expr.(*BinaryExpr); !ok {
				t.Errorf("Expected BinaryExpr at root, got %T", expr)
			}

			// Note: Full structure verification would require a tree walker
			// For now, we verify parsing succeeded and type is correct
		})
	}
}

func TestExponentiationRightAssociative(t *testing.T) {
	input := "2 ** 3 ** 4"
	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	// Should be: BinaryExpr(2, **, BinaryExpr(3, **, 4))
	binary, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected BinaryExpr, got %T", expr)
	}

	if binary.Operator != lexer.TOKEN_STAR_STAR {
		t.Errorf("Expected ** operator")
	}

	// Left should be literal 2
	if _, ok := binary.Left.(*LiteralExpr); !ok {
		t.Errorf("Expected left to be LiteralExpr, got %T", binary.Left)
	}

	// Right should be another binary expression (3 ** 4)
	rightBinary, ok := binary.Right.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected right to be BinaryExpr, got %T", binary.Right)
	}

	if rightBinary.Operator != lexer.TOKEN_STAR_STAR {
		t.Errorf("Expected ** operator in right expression")
	}
}

// --- Unary Operators ---

func TestUnaryOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		operator lexer.TokenType
	}{
		{"logical not", "!true", lexer.TOKEN_BANG},
		{"negation", "-42", lexer.TOKEN_MINUS},
		{"double negation", "!!true", lexer.TOKEN_BANG},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			unary, ok := expr.(*UnaryExpr)
			if !ok {
				t.Fatalf("Expected UnaryExpr, got %T", expr)
			}

			if unary.Operator != tt.operator {
				t.Errorf("Expected operator %v, got %v", tt.operator, unary.Operator)
			}
		})
	}
}

// --- Field Access and Method Calls ---

func TestFieldAccess(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple field", "self.name"},
		{"chained field", "self.author.name"},
		{"deep chain", "self.author.profile.name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			if _, ok := expr.(*FieldAccessExpr); !ok {
				t.Fatalf("Expected FieldAccessExpr, got %T", expr)
			}
		})
	}
}

func TestSafeNavigation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		safeField string
	}{
		{"simple safe nav", "parent?.name", "name"},
		{"self safe nav", "self.parent?.name", "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			// Find the SafeNavigationExpr in the AST
			var safeNav *SafeNavigationExpr

			if sn, ok := expr.(*SafeNavigationExpr); ok {
				safeNav = sn
			} else if fa, ok := expr.(*FieldAccessExpr); ok {
				// Safe nav might be in the chain
				if sn, ok := fa.Object.(*SafeNavigationExpr); ok {
					safeNav = sn
				}
			}

			if safeNav == nil {
				t.Fatalf("Expected SafeNavigationExpr in AST, got %T", expr)
			}

			if safeNav.Field != tt.safeField {
				t.Errorf("Expected safe field '%s', got '%s'", tt.safeField, safeNav.Field)
			}
		})
	}
}

func TestMethodCall(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"with arguments", "self.title.truncate(50)"},
		{"chained method", "self.name.upcase().truncate(10)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			if _, ok := expr.(*MethodCallExpr); !ok {
				t.Fatalf("Expected MethodCallExpr, got %T", expr)
			}
		})
	}
}

func TestMethodCallDirectly(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		method string
	}{
		{"simple method", "self.name.upcase()", "upcase"},
		{"method with args", "self.title.truncate(50)", "truncate"},
		{"chained method", "self.author.name.capitalize()", "capitalize"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			// For method calls, we need to traverse to find the MethodCallExpr
			// It might be wrapped in field access if chained
			var methodCall *MethodCallExpr

			if mc, ok := expr.(*MethodCallExpr); ok {
				methodCall = mc
			} else if fa, ok := expr.(*FieldAccessExpr); ok {
				// Check if the object is a method call
				if mc, ok := fa.Object.(*MethodCallExpr); ok {
					methodCall = mc
				}
			}

			if methodCall == nil {
				t.Fatalf("Expected MethodCallExpr somewhere in AST, got %T", expr)
			}

			if methodCall.Method != tt.method {
				t.Errorf("Expected method '%s', got '%s'", tt.method, methodCall.Method)
			}
		})
	}
}

// --- Namespaced Function Calls ---

func TestNamespacedFunctionCalls(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		namespace string
		function  string
	}{
		{"String.slugify", "String.slugify(self.title)", "String", "slugify"},
		{"Time.now", "Time.now()", "Time", "now"},
		{"Array.first", "Array.first(self.items)", "Array", "first"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			call, ok := expr.(*CallExpr)
			if !ok {
				t.Fatalf("Expected CallExpr, got %T", expr)
			}

			if call.Namespace != tt.namespace {
				t.Errorf("Expected namespace %s, got %s", tt.namespace, call.Namespace)
			}

			if call.Function != tt.function {
				t.Errorf("Expected function %s, got %s", tt.function, call.Function)
			}
		})
	}
}

// --- Ternary and Coalescing ---

func TestTernaryOperator(t *testing.T) {
	input := `true ? "yes" : "no"`
	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	ternary, ok := expr.(*TernaryExpr)
	if !ok {
		t.Fatalf("Expected TernaryExpr, got %T", expr)
	}

	if _, ok := ternary.Condition.(*LiteralExpr); !ok {
		t.Errorf("Expected condition to be LiteralExpr")
	}

	if _, ok := ternary.TrueExpr.(*LiteralExpr); !ok {
		t.Errorf("Expected true expression to be LiteralExpr")
	}

	if _, ok := ternary.FalseExpr.(*LiteralExpr); !ok {
		t.Errorf("Expected false expression to be LiteralExpr")
	}
}

func TestNullCoalescing(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple coalesce", `self.name ?? "Anonymous"`},
		{"chained coalesce", `self.a ?? self.b ?? "default"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			if _, ok := expr.(*CoalesceExpr); !ok {
				t.Fatalf("Expected CoalesceExpr, got %T", expr)
			}
		})
	}
}

// --- Assignment ---

func TestAssignment(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple assignment", "self.slug = value"},
		{"assignment with expression", `self.slug = String.slugify(self.title)`},
		{"assignment with coalesce", `self.slug = self.slug ?? "default"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			assign, ok := expr.(*AssignmentExpr)
			if !ok {
				t.Fatalf("Expected AssignmentExpr, got %T", expr)
			}

			if _, ok := assign.Target.(*FieldAccessExpr); !ok {
				t.Errorf("Expected target to be FieldAccessExpr, got %T", assign.Target)
			}
		})
	}
}

// --- Control Flow ---

func TestIfExpression(t *testing.T) {
	input := `if true { return true } else { return false }`
	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	ifExpr, ok := expr.(*IfExpr)
	if !ok {
		t.Fatalf("Expected IfExpr, got %T", expr)
	}

	if len(ifExpr.ThenBody) == 0 {
		t.Errorf("Expected non-empty then body")
	}

	if len(ifExpr.ElseBody) == 0 {
		t.Errorf("Expected non-empty else body")
	}
}

func TestUnlessExpression(t *testing.T) {
	input := `unless false { return false }`
	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	unlessExpr, ok := expr.(*UnlessExpr)
	if !ok {
		t.Fatalf("Expected UnlessExpr, got %T", expr)
	}

	if len(unlessExpr.Body) == 0 {
		t.Errorf("Expected non-empty body")
	}
}

func TestMatchExpression(t *testing.T) {
	input := `match currency { "USD" => "$", "EUR" => "€" }`
	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	matchExpr, ok := expr.(*MatchExpr)
	if !ok {
		t.Fatalf("Expected MatchExpr, got %T", expr)
	}

	if len(matchExpr.Cases) != 2 {
		t.Errorf("Expected 2 cases, got %d", len(matchExpr.Cases))
	}

	if matchExpr.Cases[0].Pattern != "USD" {
		t.Errorf("Expected first pattern to be 'USD', got '%s'", matchExpr.Cases[0].Pattern)
	}
}

// --- Statements ---

func TestLetStatement(t *testing.T) {
	input := `let slug = String.slugify(self.title)`
	stmt, errs := parseStmt(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	letStmt, ok := stmt.(*LetStmt)
	if !ok {
		t.Fatalf("Expected LetStmt, got %T", stmt)
	}

	if letStmt.Name != "slug" {
		t.Errorf("Expected variable name 'slug', got '%s'", letStmt.Name)
	}

	if _, ok := letStmt.Value.(*CallExpr); !ok {
		t.Errorf("Expected value to be CallExpr, got %T", letStmt.Value)
	}
}

func TestReturnStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasValue bool
	}{
		{"return with value", "return true", true},
		{"return without value", "return", false},
		{"return expression", `return self.slug ?? "default"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, errs := parseStmt(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			retStmt, ok := stmt.(*ReturnStmt)
			if !ok {
				t.Fatalf("Expected ReturnStmt, got %T", stmt)
			}

			hasValue := retStmt.Value != nil
			if hasValue != tt.hasValue {
				t.Errorf("Expected hasValue=%v, got %v", tt.hasValue, hasValue)
			}
		})
	}
}

func TestIfStatement(t *testing.T) {
	input := `if a > b { let result = a } elsif b > a { let result = b } else { let result = 0 }`
	stmt, errs := parseStmt(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	ifStmt, ok := stmt.(*IfStmt)
	if !ok {
		t.Fatalf("Expected IfStmt, got %T", stmt)
	}

	if len(ifStmt.ThenBody) == 0 {
		t.Errorf("Expected non-empty then body")
	}

	if len(ifStmt.ElsifBranches) != 1 {
		t.Errorf("Expected 1 elsif branch, got %d", len(ifStmt.ElsifBranches))
	}

	if len(ifStmt.ElseBody) == 0 {
		t.Errorf("Expected non-empty else body")
	}
}

// --- Complex Expressions ---

func TestComplexExpression(t *testing.T) {
	// Test the example from the spec
	input := `self.slug = String.slugify(self.title) ?? "untitled"`

	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	// Should be: AssignmentExpr(FieldAccess(self, slug), CoalesceExpr(...))
	assign, ok := expr.(*AssignmentExpr)
	if !ok {
		t.Fatalf("Expected AssignmentExpr, got %T", expr)
	}

	// Target should be self.slug
	fieldAccess, ok := assign.Target.(*FieldAccessExpr)
	if !ok {
		t.Fatalf("Expected target to be FieldAccessExpr, got %T", assign.Target)
	}

	if fieldAccess.Field != "slug" {
		t.Errorf("Expected field 'slug', got '%s'", fieldAccess.Field)
	}

	// Value should be coalesce expression
	coalesce, ok := assign.Value.(*CoalesceExpr)
	if !ok {
		t.Fatalf("Expected value to be CoalesceExpr, got %T", assign.Value)
	}

	// Left of coalesce should be String.slugify(...)
	call, ok := coalesce.Left.(*CallExpr)
	if !ok {
		t.Fatalf("Expected left of coalesce to be CallExpr, got %T", coalesce.Left)
	}

	if call.Namespace != "String" || call.Function != "slugify" {
		t.Errorf("Expected String.slugify, got %s.%s", call.Namespace, call.Function)
	}

	// Right of coalesce should be "untitled"
	literal, ok := coalesce.Right.(*LiteralExpr)
	if !ok {
		t.Fatalf("Expected right of coalesce to be LiteralExpr, got %T", coalesce.Right)
	}

	if literal.Value != "untitled" {
		t.Errorf("Expected literal 'untitled', got '%v'", literal.Value)
	}
}

func TestStringInterpolation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minParts int
	}{
		{"field access", `"Hello #{user.name}!"`, 2},
		{"simple identifier", `"Hello #{name}!"`, 2},
		{"method call", `"Total: #{order.total}!"`, 2},
		{"multiple interpolations", `"#{first} #{last}"`, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, errs := parseExpr(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}

			interp, ok := expr.(*StringInterpolationExpr)
			if !ok {
				t.Fatalf("Expected StringInterpolationExpr, got %T", expr)
			}

			if len(interp.Parts) < tt.minParts {
				t.Errorf("Expected at least %d parts for interpolation, got %d", tt.minParts, len(interp.Parts))
			}
		})
	}
}

func TestArrayIndexing(t *testing.T) {
	input := `self.items[0]`
	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	indexExpr, ok := expr.(*IndexExpr)
	if !ok {
		t.Fatalf("Expected IndexExpr, got %T", expr)
	}

	// Object should be field access
	if _, ok := indexExpr.Object.(*FieldAccessExpr); !ok {
		t.Errorf("Expected object to be FieldAccessExpr, got %T", indexExpr.Object)
	}

	// Index should be literal 0
	indexLit, ok := indexExpr.Index.(*LiteralExpr)
	if !ok {
		t.Fatalf("Expected index to be LiteralExpr, got %T", indexExpr.Index)
	}

	if indexLit.Value != int64(0) {
		t.Errorf("Expected index 0, got %v", indexLit.Value)
	}
}

// --- Edge Cases ---

func TestGroupedExpression(t *testing.T) {
	input := `(1 + 2) * 3`
	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	// Root should be binary expr (multiplication)
	binary, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected BinaryExpr, got %T", expr)
	}

	if binary.Operator != lexer.TOKEN_STAR {
		t.Errorf("Expected * operator")
	}

	// Left should be grouped expression
	group, ok := binary.Left.(*GroupExpr)
	if !ok {
		t.Fatalf("Expected left to be GroupExpr, got %T", binary.Left)
	}

	// Inside group should be addition
	innerBinary, ok := group.Expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected inner expr to be BinaryExpr, got %T", group.Expr)
	}

	if innerBinary.Operator != lexer.TOKEN_PLUS {
		t.Errorf("Expected + operator inside group")
	}
}

func TestSelfKeyword(t *testing.T) {
	input := `self`
	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	if _, ok := expr.(*SelfExpr); !ok {
		t.Fatalf("Expected SelfExpr, got %T", expr)
	}
}

func TestMembershipOperator(t *testing.T) {
	input := `self.status in ["published", "archived"]`
	expr, errs := parseExpr(input)
	if len(errs) > 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	binary, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected BinaryExpr, got %T", expr)
	}

	if binary.Operator != lexer.TOKEN_IN {
		t.Errorf("Expected IN operator, got %v", binary.Operator)
	}

	// Right side should be array literal
	if _, ok := binary.Right.(*ArrayLiteralExpr); !ok {
		t.Errorf("Expected right side to be ArrayLiteralExpr, got %T", binary.Right)
	}
}

// --- Benchmarks ---

func BenchmarkParseSimpleExpression(b *testing.B) {
	source := "1 + 2 * 3"
	for i := 0; i < b.N; i++ {
		parseExpr(source)
	}
}

func BenchmarkParseComplexExpression(b *testing.B) {
	source := `self.slug = String.slugify(self.title) ?? "untitled"`
	for i := 0; i < b.N; i++ {
		parseExpr(source)
	}
}

func BenchmarkParseMethodChain(b *testing.B) {
	source := `self.email.downcase().trim().truncate(50)`
	for i := 0; i < b.N; i++ {
		parseExpr(source)
	}
}

func BenchmarkParseArrayLiteral(b *testing.B) {
	source := `[1, 2, 3, 4, 5, 6, 7, 8, 9, 10]`
	for i := 0; i < b.N; i++ {
		parseExpr(source)
	}
}

func BenchmarkParseHashLiteral(b *testing.B) {
	source := `{name: "Alice", age: 30, email: "alice@example.com", active: true}`
	for i := 0; i < b.N; i++ {
		parseExpr(source)
	}
}
