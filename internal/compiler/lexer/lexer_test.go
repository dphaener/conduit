package lexer

import (
	"strings"
	"testing"
)

// Helper function to create a lexer and scan tokens
func scanSource(source string) ([]Token, []LexError) {
	lexer := New(source)
	return lexer.ScanTokens()
}

// Helper to check if tokens match expected types
func checkTokenTypes(t *testing.T, tokens []Token, expected []TokenType) {
	t.Helper()

	// Remove EOF token for comparison
	actual := tokens
	if len(actual) > 0 && actual[len(actual)-1].Type == TOKEN_EOF {
		actual = actual[:len(actual)-1]
	}

	if len(actual) != len(expected) {
		t.Errorf("Expected %d tokens, got %d", len(expected), len(actual))
		t.Logf("Expected: %v", expected)
		t.Logf("Got: %v", tokensToTypes(actual))
		return
	}

	for i, token := range actual {
		if token.Type != expected[i] {
			t.Errorf("Token %d: expected %s, got %s", i, expected[i], token.Type)
		}
	}
}

func tokensToTypes(tokens []Token) []TokenType {
	types := make([]TokenType, len(tokens))
	for i, t := range tokens {
		types[i] = t.Type
	}
	return types
}

// Test basic single-character tokens
func TestLexer_SingleCharTokens(t *testing.T) {
	source := "(){}[],:@!?.|+"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_LPAREN, TOKEN_RPAREN,
		TOKEN_LBRACE, TOKEN_RBRACE,
		TOKEN_LBRACKET, TOKEN_RBRACKET,
		TOKEN_COMMA, TOKEN_COLON,
		TOKEN_AT, TOKEN_BANG, TOKEN_SAFE_NAV, // ?. is tokenized as SAFE_NAV
		TOKEN_PIPE, TOKEN_PLUS,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test two-character operators
func TestLexer_TwoCharOperators(t *testing.T) {
	source := "== != <= >= || && ** -> ?? ?."
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_EQ, TOKEN_NEQ,
		TOKEN_LTE, TOKEN_GTE,
		TOKEN_DOUBLE_PIPE, TOKEN_DOUBLE_AMP,
		TOKEN_DOUBLE_STAR, TOKEN_ARROW,
		TOKEN_DOUBLE_QUESTION, TOKEN_SAFE_NAV,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test keywords
func TestLexer_Keywords(t *testing.T) {
	source := "resource on after before transaction async rescue"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_RESOURCE, TOKEN_ON, TOKEN_AFTER,
		TOKEN_BEFORE, TOKEN_TRANSACTION, TOKEN_ASYNC, TOKEN_RESCUE,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test primitive types
func TestLexer_PrimitiveTypes(t *testing.T) {
	source := "string text int float bool timestamp uuid email url"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_STRING, TOKEN_TEXT, TOKEN_INT, TOKEN_FLOAT,
		TOKEN_BOOL, TOKEN_TIMESTAMP, TOKEN_UUID, TOKEN_EMAIL, TOKEN_URL,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test nullability markers
func TestLexer_NullabilityMarkers(t *testing.T) {
	source := "string! string? int! uuid?"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_STRING, TOKEN_BANG,
		TOKEN_STRING, TOKEN_QUESTION,
		TOKEN_INT, TOKEN_BANG,
		TOKEN_UUID, TOKEN_QUESTION,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test identifiers
func TestLexer_Identifiers(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"user_name", "user_name"},
		{"PostTitle", "PostTitle"},
		{"_private", "_private"},
		{"value123", "value123"},
		{"camelCase", "camelCase"},
	}

	for _, tt := range tests {
		tokens, errors := scanSource(tt.source)

		if len(errors) > 0 {
			t.Errorf("Unexpected errors for %s: %v", tt.source, errors)
		}

		if len(tokens) < 2 { // Need at least identifier + EOF
			t.Errorf("Expected tokens for %s", tt.source)
			continue
		}

		if tokens[0].Type != TOKEN_IDENTIFIER {
			t.Errorf("Expected identifier token, got %s", tokens[0].Type)
		}

		if tokens[0].Lexeme != tt.want {
			t.Errorf("Expected lexeme %s, got %s", tt.want, tokens[0].Lexeme)
		}
	}
}

// Test annotations
func TestLexer_Annotations(t *testing.T) {
	source := "@primary @auto @unique @min @max @default"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_PRIMARY, TOKEN_AUTO, TOKEN_UNIQUE,
		TOKEN_MIN, TOKEN_MAX, TOKEN_DEFAULT,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test complex annotations
func TestLexer_ComplexAnnotations(t *testing.T) {
	source := "@has_many @computed @constraint @invariant"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_HAS_MANY, TOKEN_COMPUTED, TOKEN_CONSTRAINT, TOKEN_INVARIANT,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test unknown annotation
func TestLexer_UnknownAnnotation(t *testing.T) {
	source := "@custom_annotation"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	// Should be tokenized as @ followed by identifier
	expected := []TokenType{TOKEN_AT, TOKEN_IDENTIFIER}
	checkTokenTypes(t, tokens, expected)
}

// Test annotation column positions
func TestLexer_AnnotationColumnPositions(t *testing.T) {
	source := "@primary @custom_annotation"
	tokens, _ := scanSource(source)

	// @primary should be at column 1
	if tokens[0].Column != 1 {
		t.Errorf("@primary: expected column 1, got %d", tokens[0].Column)
	}

	// @ in @custom should be at column 10 (after space)
	if tokens[1].Column != 10 {
		t.Errorf("@ in @custom: expected column 10, got %d", tokens[1].Column)
	}

	// "custom_annotation" identifier should be at column 11
	if tokens[2].Column != 11 {
		t.Errorf("custom_annotation: expected column 11, got %d", tokens[2].Column)
	}
}

// Test integer literals
func TestLexer_IntegerLiterals(t *testing.T) {
	tests := []struct {
		source   string
		expected int64
	}{
		{"0", 0},
		{"42", 42},
		{"1000", 1000},
		{"1_000_000", 1000000},
	}

	for _, tt := range tests {
		tokens, errors := scanSource(tt.source)

		if len(errors) > 0 {
			t.Errorf("Unexpected errors for %s: %v", tt.source, errors)
		}

		if tokens[0].Type != TOKEN_INT_LITERAL {
			t.Errorf("Expected int literal, got %s", tokens[0].Type)
		}

		if tokens[0].Literal != tt.expected {
			t.Errorf("Expected value %d, got %v", tt.expected, tokens[0].Literal)
		}
	}
}

// Test float literals
func TestLexer_FloatLiterals(t *testing.T) {
	tests := []struct {
		source   string
		expected float64
	}{
		{"3.14", 3.14},
		{"0.5", 0.5},
		{"2.5e10", 2.5e10},
		{"1.0e-5", 1.0e-5},
	}

	for _, tt := range tests {
		tokens, errors := scanSource(tt.source)

		if len(errors) > 0 {
			t.Errorf("Unexpected errors for %s: %v", tt.source, errors)
		}

		if tokens[0].Type != TOKEN_FLOAT_LITERAL {
			t.Errorf("Expected float literal for %s, got %s", tt.source, tokens[0].Type)
		}

		if tokens[0].Literal != tt.expected {
			t.Errorf("Expected value %f, got %v", tt.expected, tokens[0].Literal)
		}
	}
}

// Test string literals
func TestLexer_StringLiterals(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{`"hello"`, "hello"},
		{`"hello world"`, "hello world"},
		{`""`, ""},
		{`"with \"quotes\""`, `with "quotes"`},
		{`"with\nnewline"`, "with\nnewline"},
		{`"with\ttab"`, "with\ttab"},
	}

	for _, tt := range tests {
		tokens, errors := scanSource(tt.source)

		if len(errors) > 0 {
			t.Errorf("Unexpected errors for %s: %v", tt.source, errors)
		}

		if tokens[0].Type != TOKEN_STRING_LITERAL {
			t.Errorf("Expected string literal, got %s", tokens[0].Type)
		}

		if tokens[0].Literal != tt.expected {
			t.Errorf("Expected value %q, got %q", tt.expected, tokens[0].Literal)
		}
	}
}

// Test multi-line strings
func TestLexer_MultilineStrings(t *testing.T) {
	source := `"line one
line two
line three"`

	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	if tokens[0].Type != TOKEN_STRING_LITERAL {
		t.Errorf("Expected string literal, got %s", tokens[0].Type)
	}

	expected := "line one\nline two\nline three"
	if tokens[0].Literal != expected {
		t.Errorf("Expected %q, got %q", expected, tokens[0].Literal)
	}
}

// Test boolean literals
func TestLexer_BooleanLiterals(t *testing.T) {
	source := "true false"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	if tokens[0].Type != TOKEN_TRUE || tokens[0].Literal != true {
		t.Errorf("Expected true literal")
	}

	if tokens[1].Type != TOKEN_FALSE || tokens[1].Literal != false {
		t.Errorf("Expected false literal")
	}
}

// Test null literals
func TestLexer_NullLiterals(t *testing.T) {
	source := "null nil"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{TOKEN_NULL, TOKEN_NULL}
	checkTokenTypes(t, tokens, expected)
}

// Test comments
func TestLexer_SingleLineComments(t *testing.T) {
	source := `# This is a comment
resource User {
  # Another comment
  id: uuid!
}`

	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	// Comments should be ignored
	expected := []TokenType{
		TOKEN_RESOURCE, TOKEN_IDENTIFIER, TOKEN_LBRACE,
		TOKEN_IDENTIFIER, TOKEN_COLON, TOKEN_UUID, TOKEN_BANG,
		TOKEN_RBRACE,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test simple resource definition
func TestLexer_SimpleResource(t *testing.T) {
	source := `resource User {
  username: string!
  email_address: email!
}`

	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_RESOURCE, TOKEN_IDENTIFIER, TOKEN_LBRACE,
		TOKEN_IDENTIFIER, TOKEN_COLON, TOKEN_STRING, TOKEN_BANG,
		TOKEN_IDENTIFIER, TOKEN_COLON, TOKEN_EMAIL, TOKEN_BANG,
		TOKEN_RBRACE,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test resource with annotations
func TestLexer_ResourceWithAnnotations(t *testing.T) {
	source := `resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
}`

	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	// Check key tokens are present
	hasResource := false
	hasPrimary := false
	hasMin := false
	hasMax := false

	for _, token := range tokens {
		switch token.Type {
		case TOKEN_RESOURCE:
			hasResource = true
		case TOKEN_PRIMARY:
			hasPrimary = true
		case TOKEN_MIN:
			hasMin = true
		case TOKEN_MAX:
			hasMax = true
		}
	}

	if !hasResource {
		t.Error("Missing resource keyword")
	}
	if !hasPrimary {
		t.Error("Missing @primary annotation")
	}
	if !hasMin {
		t.Error("Missing @min annotation")
	}
	if !hasMax {
		t.Error("Missing @max annotation")
	}
}

// Test namespace separator
func TestLexer_NamespacedCalls(t *testing.T) {
	source := "String.slugify(self.title)"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_IDENTIFIER, TOKEN_DOT, TOKEN_IDENTIFIER,
		TOKEN_LPAREN, TOKEN_SELF, TOKEN_DOT, TOKEN_IDENTIFIER, TOKEN_RPAREN,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test position tracking
func TestLexer_PositionTracking(t *testing.T) {
	source := `resource User {
  id: uuid!
}`

	tokens, _ := scanSource(source)

	// Check first token position
	if tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Errorf("Expected resource at 1:1, got %d:%d", tokens[0].Line, tokens[0].Column)
	}

	// Find id token (should be on line 2)
	for _, token := range tokens {
		if token.Lexeme == "id" {
			if token.Line != 2 {
				t.Errorf("Expected 'id' on line 2, got line %d", token.Line)
			}
			break
		}
	}
}

// Test error cases
func TestLexer_UnterminatedString(t *testing.T) {
	source := `"unterminated string`
	_, errors := scanSource(source)

	if len(errors) == 0 {
		t.Error("Expected error for unterminated string")
	}

	if !strings.Contains(errors[0].Message, "Unterminated string") {
		t.Errorf("Wrong error message: %s", errors[0].Message)
	}
}

func TestLexer_InvalidCharacter(t *testing.T) {
	source := `resource User { $ }`
	_, errors := scanSource(source)

	if len(errors) == 0 {
		t.Error("Expected error for invalid character")
	}
}

func TestLexer_InvalidNumber(t *testing.T) {
	source := "3.14.15"
	tokens, _ := scanSource(source)

	// Should tokenize as 3.14 then .15
	if tokens[0].Type != TOKEN_FLOAT_LITERAL {
		t.Error("Expected float for first token")
	}
}

// Test lifecycle hook syntax
func TestLexer_LifecycleHook(t *testing.T) {
	source := `@before create @transaction {
  self.slug = String.slugify(self.title)
}`

	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	// Verify key tokens
	hasBefore := false
	hasCreate := false
	hasTransaction := false

	for _, token := range tokens {
		if token.Lexeme == "@before" {
			hasBefore = true
		}
		if token.Lexeme == "create" {
			hasCreate = true
		}
		if token.Lexeme == "@transaction" {
			hasTransaction = true
		}
	}

	if !hasBefore {
		t.Error("Missing @before annotation")
	}
	if !hasCreate {
		t.Error("Missing create identifier")
	}
	if !hasTransaction {
		t.Error("Missing @transaction annotation")
	}
}

// Test complex expression
func TestLexer_ComplexExpression(t *testing.T) {
	source := `self.price * (1.0 - discount / 100.0)`
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_SELF, TOKEN_DOT, TOKEN_IDENTIFIER,
		TOKEN_STAR, TOKEN_LPAREN, TOKEN_FLOAT_LITERAL,
		TOKEN_MINUS, TOKEN_IDENTIFIER, TOKEN_SLASH,
		TOKEN_FLOAT_LITERAL, TOKEN_RPAREN,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test safe navigation operator
func TestLexer_SafeNavigation(t *testing.T) {
	source := "self.author?.name"
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_SELF, TOKEN_DOT, TOKEN_IDENTIFIER,
		TOKEN_SAFE_NAV, TOKEN_IDENTIFIER,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test null coalescing
func TestLexer_NullCoalescing(t *testing.T) {
	source := `self.excerpt ?? "No excerpt"`
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_SELF, TOKEN_DOT, TOKEN_IDENTIFIER,
		TOKEN_DOUBLE_QUESTION, TOKEN_STRING_LITERAL,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test array syntax
func TestLexer_ArraySyntax(t *testing.T) {
	source := `array<uuid>!`
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_ARRAY, TOKEN_LT, TOKEN_UUID, TOKEN_GT, TOKEN_BANG,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test hash syntax
func TestLexer_HashSyntax(t *testing.T) {
	source := `hash<string, int>!`
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_HASH, TOKEN_LT, TOKEN_STRING, TOKEN_COMMA,
		TOKEN_INT, TOKEN_GT, TOKEN_BANG,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test enum syntax
func TestLexer_EnumSyntax(t *testing.T) {
	source := `status: enum ["draft", "published"]!`
	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	expected := []TokenType{
		TOKEN_IDENTIFIER, TOKEN_COLON, TOKEN_ENUM,
		TOKEN_LBRACKET, TOKEN_STRING_LITERAL, TOKEN_COMMA,
		TOKEN_STRING_LITERAL, TOKEN_RBRACKET, TOKEN_BANG,
	}

	checkTokenTypes(t, tokens, expected)
}

// Test match expression
func TestLexer_MatchExpression(t *testing.T) {
	source := `match self.status {
  "draft" => 0,
  "published" => 1
}`

	tokens, errors := scanSource(source)

	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}

	hasMatch := false
	hasArrow := false

	for _, token := range tokens {
		if token.Type == TOKEN_MATCH {
			hasMatch = true
		}
		if token.Type == TOKEN_ARROW {
			hasArrow = true
		}
	}

	if !hasMatch {
		t.Error("Missing match keyword")
	}
	if !hasArrow {
		t.Error("Missing arrow operator")
	}
}

// Test helper functions
func TestIsKeyword(t *testing.T) {
	tests := []struct {
		word     string
		expected bool
	}{
		{"resource", true},
		{"string", true},
		{"user_name", false},
		{"create", false},
	}

	for _, tt := range tests {
		result := IsKeyword(tt.word)
		if result != tt.expected {
			t.Errorf("IsKeyword(%s): expected %v, got %v", tt.word, tt.expected, result)
		}
	}
}

func TestIsPrimitiveType(t *testing.T) {
	tests := []struct {
		word     string
		expected bool
	}{
		{"string", true},
		{"int", true},
		{"uuid", true},
		{"User", false},
		{"custom", false},
	}

	for _, tt := range tests {
		result := IsPrimitiveType(tt.word)
		if result != tt.expected {
			t.Errorf("IsPrimitiveType(%s): expected %v, got %v", tt.word, tt.expected, result)
		}
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		word     string
		expected bool
	}{
		{"user_name", true},
		{"PostTitle", true},
		{"_private", true},
		{"resource", false}, // keyword
		{"123invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsValidIdentifier(tt.word)
		if result != tt.expected {
			t.Errorf("IsValidIdentifier(%s): expected %v, got %v", tt.word, tt.expected, result)
		}
	}
}
