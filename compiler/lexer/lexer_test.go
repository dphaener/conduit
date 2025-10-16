package lexer

import (
	"testing"
)

// TestKeywords tests tokenization of all keywords
func TestKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"resource", TOKEN_RESOURCE},
		{"before", TOKEN_BEFORE},
		{"after", TOKEN_AFTER},
		{"on", TOKEN_ON},
		{"transaction", TOKEN_TRANSACTION},
		{"async", TOKEN_ASYNC},
		{"rescue", TOKEN_RESCUE},
		{"has", TOKEN_HAS},
		{"through", TOKEN_THROUGH},
		{"as", TOKEN_AS},
		{"under", TOKEN_UNDER},
		{"nested", TOKEN_NESTED},
		{"function", TOKEN_FUNCTION},
		{"validate", TOKEN_VALIDATE},
		{"constraint", TOKEN_CONSTRAINT},
		{"invariant", TOKEN_INVARIANT},
		{"computed", TOKEN_COMPUTED},
		{"scope", TOKEN_SCOPE},
		{"if", TOKEN_IF},
		{"elsif", TOKEN_ELSIF},
		{"else", TOKEN_ELSE},
		{"unless", TOKEN_UNLESS},
		{"match", TOKEN_MATCH},
		{"when", TOKEN_WHEN},
		{"return", TOKEN_RETURN},
		{"let", TOKEN_LET},
		{"true", TOKEN_TRUE},
		{"false", TOKEN_FALSE},
		{"nil", TOKEN_NIL},
		{"self", TOKEN_SELF},
		{"string", TOKEN_STRING},
		{"int", TOKEN_INT},
		{"float", TOKEN_FLOAT},
		{"bool", TOKEN_BOOL},
		{"uuid", TOKEN_UUID},
		{"timestamp", TOKEN_TIMESTAMP},
		{"array", TOKEN_ARRAY},
		{"enum", TOKEN_ENUM},
		{"hash", TOKEN_HASH},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			tokens, errors := lexer.ScanTokens()

			if len(errors) > 0 {
				t.Fatalf("Unexpected errors: %v", errors)
			}

			if len(tokens) != 2 { // keyword + EOF
				t.Fatalf("Expected 2 tokens, got %d", len(tokens))
			}

			if tokens[0].Type != tt.expected {
				t.Errorf("Expected token type %v, got %v", tt.expected, tokens[0].Type)
			}
		})
	}
}

// TestIdentifiers tests identifier tokenization including Unicode support
func TestIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "username", "username"},
		{"underscore", "user_name", "user_name"},
		{"numbers", "user123", "user123"},
		{"camelCase", "userName", "userName"},
		{"predicate", "empty?", "empty?"},
		{"unicode", "用户名", "用户名"},
		{"mixed_unicode", "user_名前", "user_名前"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			tokens, errors := lexer.ScanTokens()

			if len(errors) > 0 {
				t.Fatalf("Unexpected errors: %v", errors)
			}

			if len(tokens) != 2 {
				t.Fatalf("Expected 2 tokens, got %d", len(tokens))
			}

			if tokens[0].Type != TOKEN_IDENTIFIER {
				t.Errorf("Expected IDENTIFIER, got %v", tokens[0].Type)
			}

			if tokens[0].Literal != tt.expected {
				t.Errorf("Expected identifier %q, got %q", tt.expected, tokens[0].Literal)
			}
		})
	}
}

// TestNullabilityMarkers tests ! and ? tokenization
func TestNullabilityMarkers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{"required", "string!", []TokenType{TOKEN_STRING, TOKEN_BANG}},
		{"optional", "string?", []TokenType{TOKEN_STRING, TOKEN_QUESTION}},
		{"not_equal", "!=", []TokenType{TOKEN_BANG_EQUAL}},
		{"safe_nav", "?.", []TokenType{TOKEN_QUESTION_DOT}},
		{"null_coalesce", "??", []TokenType{TOKEN_QUESTION_QUESTION}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			tokens, errors := lexer.ScanTokens()

			if len(errors) > 0 {
				t.Fatalf("Unexpected errors: %v", errors)
			}

			// Check that we have expected tokens plus EOF
			if len(tokens) != len(tt.expected)+1 {
				t.Fatalf("Expected %d tokens (plus EOF), got %d", len(tt.expected), len(tokens))
			}

			for i, expectedType := range tt.expected {
				if tokens[i].Type != expectedType {
					t.Errorf("Token %d: expected %v, got %v", i, expectedType, tokens[i].Type)
				}
			}

			// Check last token is EOF
			if tokens[len(tokens)-1].Type != TOKEN_EOF {
				t.Errorf("Last token should be EOF, got %v", tokens[len(tokens)-1].Type)
			}
		})
	}
}

// TestOperators tests all operators
func TestOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"@", TOKEN_AT},
		{":", TOKEN_COLON},
		{".", TOKEN_DOT},
		{",", TOKEN_COMMA},
		{"+", TOKEN_PLUS},
		{"-", TOKEN_MINUS},
		{"*", TOKEN_STAR},
		{"/", TOKEN_SLASH},
		{"%", TOKEN_PERCENT},
		{"<", TOKEN_LESS},
		{">", TOKEN_GREATER},
		{"=", TOKEN_EQUAL},
		{"|", TOKEN_PIPE},
		{"&", TOKEN_AMPERSAND},
		{"->", TOKEN_ARROW},
		{"==", TOKEN_EQUAL_EQUAL},
		{"!=", TOKEN_BANG_EQUAL},
		{"<=", TOKEN_LESS_EQUAL},
		{">=", TOKEN_GREATER_EQUAL},
		{"**", TOKEN_STAR_STAR},
		{"?.", TOKEN_QUESTION_DOT},
		{"??", TOKEN_QUESTION_QUESTION},
		{"||", TOKEN_PIPE_PIPE},
		{"&&", TOKEN_AMPERSAND_AMPERSAND},
		{"=>", TOKEN_FAT_ARROW},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			tokens, errors := lexer.ScanTokens()

			if len(errors) > 0 {
				t.Fatalf("Unexpected errors: %v", errors)
			}

			if len(tokens) != 2 {
				t.Fatalf("Expected 2 tokens, got %d", len(tokens))
			}

			if tokens[0].Type != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tokens[0].Type)
			}
		})
	}
}

// TestDelimiters tests all delimiters
func TestDelimiters(t *testing.T) {
	input := "()[]{}"
	expected := []TokenType{
		TOKEN_LPAREN, TOKEN_RPAREN,
		TOKEN_LBRACKET, TOKEN_RBRACKET,
		TOKEN_LBRACE, TOKEN_RBRACE,
		TOKEN_EOF,
	}

	lexer := New(input, "test.cdt")
	tokens, errors := lexer.ScanTokens()

	if len(errors) > 0 {
		t.Fatalf("Unexpected errors: %v", errors)
	}

	if len(tokens) != len(expected) {
		t.Fatalf("Expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, expectedType := range expected {
		if tokens[i].Type != expectedType {
			t.Errorf("Token %d: expected %v, got %v", i, expectedType, tokens[i].Type)
		}
	}
}

// TestNumbers tests integer and float literal tokenization
func TestNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
		tokenType TokenType
	}{
		{"integer", "42", int64(42), TOKEN_INT_LITERAL},
		{"negative", "-17", int64(-17), TOKEN_MINUS}, // Note: - is separate token
		{"zero", "0", int64(0), TOKEN_INT_LITERAL},
		{"underscore", "1_000_000", int64(1000000), TOKEN_INT_LITERAL},
		{"float", "3.14", float64(3.14), TOKEN_FLOAT_LITERAL},
		{"float_underscore", "1_000.50", float64(1000.50), TOKEN_FLOAT_LITERAL},
		{"scientific", "2.5e10", float64(2.5e10), TOKEN_FLOAT_LITERAL},
		{"scientific_neg", "1.5e-3", float64(1.5e-3), TOKEN_FLOAT_LITERAL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			tokens, errors := lexer.ScanTokens()

			if len(errors) > 0 {
				t.Fatalf("Unexpected errors: %v", errors)
			}

			if tt.name == "negative" {
				// Special case: negative numbers are two tokens
				if len(tokens) < 2 {
					t.Fatalf("Expected at least 2 tokens, got %d", len(tokens))
				}
				if tokens[0].Type != TOKEN_MINUS {
					t.Errorf("Expected MINUS, got %v", tokens[0].Type)
				}
				return
			}

			if len(tokens) != 2 {
				t.Fatalf("Expected 2 tokens, got %d", len(tokens))
			}

			if tokens[0].Type != tt.tokenType {
				t.Errorf("Expected %v, got %v", tt.tokenType, tokens[0].Type)
			}

			if tokens[0].Literal != tt.expected {
				t.Errorf("Expected literal %v, got %v", tt.expected, tokens[0].Literal)
			}
		})
	}
}

// TestStrings tests string literal tokenization
func TestStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", `"hello"`, "hello"},
		{"empty", `""`, ""},
		{"with_spaces", `"hello world"`, "hello world"},
		{"escape_newline", `"line1\nline2"`, "line1\nline2"},
		{"escape_tab", `"hello\tworld"`, "hello\tworld"},
		{"escape_quote", `"say \"hello\""`, `say "hello"`},
		{"escape_backslash", `"path\\to\\file"`, `path\to\file`},
		{"multiline", "\"line1\nline2\"", "line1\nline2"},
		{"unicode", `"Hello 世界"`, "Hello 世界"},
		{"escape_hash", `"not \#interpolated"`, "not #interpolated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			tokens, errors := lexer.ScanTokens()

			if len(errors) > 0 {
				t.Fatalf("Unexpected errors: %v", errors)
			}

			if len(tokens) != 2 {
				t.Fatalf("Expected 2 tokens, got %d", len(tokens))
			}

			if tokens[0].Type != TOKEN_STRING_LITERAL {
				t.Errorf("Expected STRING_LITERAL, got %v", tokens[0].Type)
			}

			if tokens[0].Literal != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, tokens[0].Literal)
			}
		})
	}
}

// TestStringInterpolation tests string interpolation markers
func TestStringInterpolation(t *testing.T) {
	input := `"Hello #{name}!"`
	lexer := New(input, "test.cdt")
	tokens, errors := lexer.ScanTokens()

	if len(errors) > 0 {
		t.Fatalf("Unexpected errors: %v", errors)
	}

	// Should have: STRING_LITERAL, EOF
	// Note: Full interpolation parsing is parser's job, lexer just handles the string
	if len(tokens) != 2 {
		t.Fatalf("Expected 2 tokens, got %d", len(tokens))
	}

	if tokens[0].Type != TOKEN_STRING_LITERAL {
		t.Errorf("Expected STRING_LITERAL, got %v", tokens[0].Type)
	}
}

// TestComments tests comment tokenization
func TestComments(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		preserveComments  bool
		expectedTokens    int
		hasCommentToken   bool
	}{
		{"single_line", "# This is a comment", false, 1, false}, // Just EOF
		{"single_line_preserve", "# This is a comment", true, 2, true}, // COMMENT + EOF
		{"inline", "username # comment", false, 2, false}, // IDENTIFIER + EOF
		{"inline_preserve", "username # comment", true, 3, true}, // IDENTIFIER + COMMENT + EOF
		{"multiple", "# line1\n# line2", false, 1, false},
		{"multiple_preserve", "# line1\n# line2", true, 4, true}, // COMMENT + NEWLINE + COMMENT + EOF
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			lexer.SetPreserveComments(tt.preserveComments)
			tokens, errors := lexer.ScanTokens()

			if len(errors) > 0 {
				t.Fatalf("Unexpected errors: %v", errors)
			}

			if len(tokens) != tt.expectedTokens {
				t.Fatalf("Expected %d tokens, got %d", tt.expectedTokens, len(tokens))
			}

			hasComment := false
			for _, token := range tokens {
				if token.Type == TOKEN_COMMENT {
					hasComment = true
					break
				}
			}

			if hasComment != tt.hasCommentToken {
				t.Errorf("Expected hasComment=%v, got %v", tt.hasCommentToken, hasComment)
			}
		})
	}
}

// TestPositionTracking tests accurate line and column tracking
func TestPositionTracking(t *testing.T) {
	input := "resource User {\n  username: string!\n}"
	lexer := New(input, "test.cdt")
	tokens, errors := lexer.ScanTokens()

	if len(errors) > 0 {
		t.Fatalf("Unexpected errors: %v", errors)
	}

	// Check positions of key tokens
	// Note: column tracking starts from column when token starts being scanned
	expectedPositions := []struct {
		tokenType TokenType
		line      int
		column    int
	}{
		{TOKEN_RESOURCE, 1, 1},
		{TOKEN_IDENTIFIER, 1, 10}, // "User"
		{TOKEN_LBRACE, 1, 15},
		{TOKEN_IDENTIFIER, 2, 2}, // "username" (line 2, column after 2 spaces, 0-indexed becomes 2)
		{TOKEN_COLON, 2, 10},
		{TOKEN_STRING, 2, 12},
		{TOKEN_BANG, 2, 18},
		{TOKEN_RBRACE, 3, 0},
	}

	tokenIndex := 0
	for _, expected := range expectedPositions {
		if tokenIndex >= len(tokens) {
			t.Fatalf("Not enough tokens")
		}

		token := tokens[tokenIndex]
		if token.Type != expected.tokenType {
			t.Errorf("Token %d: expected type %v, got %v", tokenIndex, expected.tokenType, token.Type)
		}

		if token.Line != expected.line {
			t.Errorf("Token %d (%v): expected line %d, got %d", tokenIndex, token.Type, expected.line, token.Line)
		}

		if token.Column != expected.column {
			t.Errorf("Token %d (%v): expected column %d, got %d", tokenIndex, token.Type, expected.column, token.Column)
		}

		tokenIndex++
	}
}

// TestErrorRecovery tests that lexer continues after errors
func TestErrorRecovery(t *testing.T) {
	input := "username @ 123 ^ email"
	lexer := New(input, "test.cdt")
	tokens, errors := lexer.ScanTokens()

	// Should have errors for '^' but continue tokenizing
	if len(errors) == 0 {
		t.Error("Expected errors for invalid character '^'")
	}

	// Should still tokenize valid tokens
	if len(tokens) < 4 { // username, @, 123, email, EOF
		t.Errorf("Expected at least 4 tokens despite errors, got %d", len(tokens))
	}

	// Check that valid tokens were recognized
	expectedTypes := []TokenType{TOKEN_IDENTIFIER, TOKEN_AT, TOKEN_INT_LITERAL}
	for i, expected := range expectedTypes {
		if tokens[i].Type != expected {
			t.Errorf("Token %d: expected %v, got %v", i, expected, tokens[i].Type)
		}
	}
}

// TestUnterminatedString tests error handling for unterminated strings
func TestUnterminatedString(t *testing.T) {
	input := `"unterminated string`
	lexer := New(input, "test.cdt")
	_, errors := lexer.ScanTokens()

	if len(errors) == 0 {
		t.Error("Expected error for unterminated string")
	}

	if len(errors) > 0 {
		if errors[0].Message != "Unterminated string starting at line 1" {
			t.Errorf("Unexpected error message: %s", errors[0].Message)
		}
	}
}

// TestComplexResource tests tokenization of a complete resource definition
func TestComplexResource(t *testing.T) {
	input := `
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  content: text!
  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  @before create {
    self.slug = String.slugify(self.title)
  }

  @after update @transaction {
    if self.status == "published" {
      self.published_at = Time.now()
    }
  }
}
`

	lexer := New(input, "test.cdt")
	tokens, errors := lexer.ScanTokens()

	if len(errors) > 0 {
		t.Fatalf("Unexpected errors: %v", errors)
	}

	// Should have many tokens
	if len(tokens) < 50 {
		t.Errorf("Expected at least 50 tokens for complex resource, got %d", len(tokens))
	}

	// Check for presence of key token types
	expectedTypes := map[TokenType]bool{
		TOKEN_RESOURCE:     false,
		TOKEN_IDENTIFIER:   false,
		TOKEN_UUID:         false,
		TOKEN_BANG:         false,
		TOKEN_AT:           false,
		TOKEN_BEFORE:       false,
		TOKEN_AFTER:        false,
		TOKEN_TRANSACTION:  false,
		TOKEN_IF:           false,
		TOKEN_EQUAL_EQUAL:  false,
		TOKEN_STRING_LITERAL: false,
		TOKEN_DOT:          false,
	}

	for _, token := range tokens {
		if _, exists := expectedTypes[token.Type]; exists {
			expectedTypes[token.Type] = true
		}
	}

	for tokenType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected to find token type %v", tokenType)
		}
	}
}

// TestUnicodeSupport tests full Unicode support
func TestUnicodeSupport(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"japanese", "変数名: string!"},
		{"chinese", "用户: User!"},
		{"arabic", "الاسم: string!"},
		{"emoji", "name: string! # 👍"},
		{"mixed", "user_名前: string!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			_, errors := lexer.ScanTokens()

			if len(errors) > 0 {
				t.Fatalf("Unexpected errors for Unicode input: %v", errors)
			}
		})
	}
}

// TestNamespacedFunctionCalls tests tokenization of namespaced function calls
func TestNamespacedFunctionCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			"string_slugify",
			"String.slugify(title)",
			[]TokenType{TOKEN_IDENTIFIER, TOKEN_DOT, TOKEN_IDENTIFIER, TOKEN_LPAREN, TOKEN_IDENTIFIER, TOKEN_RPAREN, TOKEN_EOF},
		},
		{
			"time_now",
			"Time.now()",
			[]TokenType{TOKEN_IDENTIFIER, TOKEN_DOT, TOKEN_IDENTIFIER, TOKEN_LPAREN, TOKEN_RPAREN, TOKEN_EOF},
		},
		{
			"chained",
			"self.author.name",
			[]TokenType{TOKEN_SELF, TOKEN_DOT, TOKEN_IDENTIFIER, TOKEN_DOT, TOKEN_IDENTIFIER, TOKEN_EOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			tokens, errors := lexer.ScanTokens()

			if len(errors) > 0 {
				t.Fatalf("Unexpected errors: %v", errors)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expected := range tt.expected {
				if tokens[i].Type != expected {
					t.Errorf("Token %d: expected %v, got %v", i, expected, tokens[i].Type)
				}
			}
		})
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		expectError bool
	}{
		{"empty", "", false},
		{"whitespace_only", "   \n\t\r\n   ", false},
		{"single_char", "a", false},
		{"just_operator", "+", false},
		{"unclosed_brace", "{", false}, // Not lexer's job to balance
		{"float_no_leading", ".5", false}, // Should tokenize as DOT and INT
		{"multiple_dots", "...", false},
		{"mixed_operators", "+-*/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.input, "test.cdt")
			tokens, errors := lexer.ScanTokens()

			if tt.expectError && len(errors) == 0 {
				t.Error("Expected errors but got none")
			}

			if !tt.expectError && len(errors) > 0 {
				t.Errorf("Unexpected errors: %v", errors)
			}

			// Should always have at least EOF
			if len(tokens) < 1 {
				t.Error("Expected at least EOF token")
			}

			// Last token should always be EOF
			if tokens[len(tokens)-1].Type != TOKEN_EOF {
				t.Error("Last token should be EOF")
			}
		})
	}
}
