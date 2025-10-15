// Package lexer provides lexical analysis for Conduit source code.
// It tokenizes .cdt files into a stream of tokens for the parser.
package lexer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Lexer tokenizes Conduit source code.
//
// Thread Safety: Lexer instances are NOT thread-safe. Each goroutine must
// create its own Lexer instance via New(). This is the recommended approach
// for parallel lexing in scenarios like LSP diagnostics.
type Lexer struct {
	source  string     // Source code to tokenize
	start   int        // Start position of current token
	current int        // Current position in source
	line    int        // Current line number (1-indexed)
	column  int        // Current column number (1-indexed)
	tokens  []Token    // Collected tokens
	errors  []LexError // Collected errors

	// State for string interpolation tracking
	interpolationDepth int // Tracks nesting level of interpolation
}

// New creates a new Lexer for the given source code
func New(source string) *Lexer {
	return &Lexer{
		source:             source,
		start:              0,
		current:            0,
		line:               1,
		column:             1,
		tokens:             make([]Token, 0),
		errors:             make([]LexError, 0),
		interpolationDepth: 0,
	}
}

// ScanTokens tokenizes the entire source and returns tokens and errors
func (l *Lexer) ScanTokens() ([]Token, []LexError) {
	for !l.isAtEnd() {
		l.start = l.current
		l.scanToken()
	}

	// Add EOF token
	l.tokens = append(l.tokens, Token{
		Type:   TOKEN_EOF,
		Lexeme: "",
		Line:   l.line,
		Column: l.column,
	})

	return l.tokens, l.errors
}

// scanToken processes the next token.
// This function has inherently high cyclomatic complexity as it dispatches
// to handlers for 40+ different token types. This is a standard pattern in
// lexer design and the complexity is managed by delegating actual logic to
// focused helper methods.
//
//nolint:gocyclo,cyclop // Lexer dispatch function - complexity is inherent to the pattern
func (l *Lexer) scanToken() {
	c := l.advance()

	// Categorize and delegate based on character type
	switch {
	case c == '(' || c == ')' || c == '{' || c == '}' || c == '[' || c == ']':
		l.scanDelimiter(c)
	case c == ',' || c == '+' || c == '%':
		l.scanSimpleOperator(c)
	case c == '!' || c == '=' || c == '<' || c == '>' || c == '|' || c == '&' ||
		c == '*' || c == '-' || c == '?' || c == '.' || c == ':' || c == '/':
		l.scanCompoundOperator(c)
	case c == '#':
		l.scanHashToken()
	case c == '@':
		l.annotation()
	case c == '"':
		l.string()
	case c == ' ' || c == '\r' || c == '\t':
		// Ignore whitespace
	case c == '\n':
		l.line++
		l.column = 0
	default:
		l.scanDefault(c)
	}
}

// scanDelimiter handles delimiter tokens: ( ) { } [ ]
func (l *Lexer) scanDelimiter(c byte) {
	switch c {
	case '(':
		l.addToken(TOKEN_LPAREN)
	case ')':
		l.addToken(TOKEN_RPAREN)
	case '{':
		l.addToken(TOKEN_LBRACE)
	case '}':
		l.addToken(TOKEN_RBRACE)
	case '[':
		l.addToken(TOKEN_LBRACKET)
	case ']':
		l.addToken(TOKEN_RBRACKET)
	}
}

// scanSimpleOperator handles single-character operators: , + %
func (l *Lexer) scanSimpleOperator(c byte) {
	switch c {
	case ',':
		l.addToken(TOKEN_COMMA)
	case '+':
		l.addToken(TOKEN_PLUS)
	case '%':
		l.addToken(TOKEN_PERCENT)
	}
}

// scanCompoundOperator dispatches to specific multi-character operator handlers
func (l *Lexer) scanCompoundOperator(c byte) {
	switch c {
	case '!':
		l.scanBangToken()
	case '=':
		l.scanEqualsToken()
	case '<':
		l.scanLessThanToken()
	case '>':
		l.scanGreaterThanToken()
	case '|':
		l.scanPipeToken()
	case '&':
		l.scanAmpersandToken()
	case '*':
		l.scanStarToken()
	case '-':
		l.scanMinusToken()
	case '?':
		l.scanQuestionToken()
	case '.':
		l.scanDotToken()
	case ':':
		l.scanColonToken()
	case '/':
		l.scanSlashToken()
	}
}

// scanBangToken handles ! and !=
func (l *Lexer) scanBangToken() {
	if l.match('=') {
		l.addToken(TOKEN_NEQ)
	} else {
		l.addToken(TOKEN_BANG)
	}
}

// scanEqualsToken handles =, ==, and =>
func (l *Lexer) scanEqualsToken() {
	if l.match('=') {
		l.addToken(TOKEN_EQ)
	} else if l.match('>') {
		l.addToken(TOKEN_ARROW)
	} else {
		l.addToken(TOKEN_EQUALS)
	}
}

// scanLessThanToken handles < and <=
func (l *Lexer) scanLessThanToken() {
	if l.match('=') {
		l.addToken(TOKEN_LTE)
	} else {
		l.addToken(TOKEN_LT)
	}
}

// scanGreaterThanToken handles > and >=
func (l *Lexer) scanGreaterThanToken() {
	if l.match('=') {
		l.addToken(TOKEN_GTE)
	} else {
		l.addToken(TOKEN_GT)
	}
}

// scanPipeToken handles | and ||
func (l *Lexer) scanPipeToken() {
	if l.match('|') {
		l.addToken(TOKEN_DOUBLE_PIPE)
	} else {
		l.addToken(TOKEN_PIPE)
	}
}

// scanAmpersandToken handles & and && (single & is an error)
func (l *Lexer) scanAmpersandToken() {
	if l.match('&') {
		l.addToken(TOKEN_DOUBLE_AMP)
	} else {
		l.addError("Unexpected character '&' (did you mean '&&'?)")
	}
}

// scanStarToken handles * and **
func (l *Lexer) scanStarToken() {
	if l.match('*') {
		l.addToken(TOKEN_DOUBLE_STAR)
	} else {
		l.addToken(TOKEN_STAR)
	}
}

// scanMinusToken handles - and ->
func (l *Lexer) scanMinusToken() {
	if l.match('>') {
		l.addToken(TOKEN_ARROW)
	} else {
		l.addToken(TOKEN_MINUS)
	}
}

// scanQuestionToken handles ?, ?., and ??
func (l *Lexer) scanQuestionToken() {
	if l.match('.') {
		l.addToken(TOKEN_SAFE_NAV)
	} else if l.match('?') {
		l.addToken(TOKEN_DOUBLE_QUESTION)
	} else {
		l.addToken(TOKEN_QUESTION)
	}
}

// scanDotToken handles . and numbers starting with .
func (l *Lexer) scanDotToken() {
	if l.isDigit(l.peek()) {
		l.number()
	} else {
		l.addToken(TOKEN_DOT)
	}
}

// scanColonToken handles : and ::
func (l *Lexer) scanColonToken() {
	if l.match(':') {
		l.addToken(TOKEN_DOUBLE_COLON)
	} else {
		l.addToken(TOKEN_COLON)
	}
}

// scanSlashToken handles / and // comments
func (l *Lexer) scanSlashToken() {
	if l.match('/') {
		l.comment()
	} else {
		l.addToken(TOKEN_SLASH)
	}
}

// scanHashToken handles # comments and ### multiline comments
func (l *Lexer) scanHashToken() {
	if l.peek() == '#' && l.peekNext() == '#' {
		l.multilineComment()
	} else {
		l.comment()
	}
}

// scanDefault handles the default case: numbers, identifiers, or errors
func (l *Lexer) scanDefault(c byte) {
	if l.isDigit(c) {
		l.number()
	} else if l.isAlpha(c) {
		l.identifier()
	} else {
		l.addError(fmt.Sprintf("Unexpected character: '%c'", c))
	}
}

// annotation handles @ symbols and following identifiers
func (l *Lexer) annotation() {
	// We've already consumed the @
	if !l.isAlpha(l.peek()) && l.peek() != '_' {
		// Just an @ symbol by itself
		l.addToken(TOKEN_AT)
		return
	}

	// Read the annotation name
	startPos := l.current
	for l.isAlphaNumeric(l.peek()) {
		l.advance()
	}

	annotationName := l.source[startPos:l.current]

	// Check if it's a known annotation keyword
	if tokenType, ok := AnnotationKeywords[annotationName]; ok {
		// Create token with full lexeme including @
		token := Token{
			Type:   tokenType,
			Lexeme: "@" + annotationName,
			Line:   l.line,
			Column: l.column - (l.current - l.start),
		}
		l.tokens = append(l.tokens, token)
	} else {
		// Unknown annotation - emit @ and identifier separately
		// @ position is at the start of the entire token
		atColumn := l.column - (l.current - l.start)
		l.tokens = append(l.tokens, Token{
			Type:   TOKEN_AT,
			Lexeme: "@",
			Line:   l.line,
			Column: atColumn,
		})
		l.tokens = append(l.tokens, Token{
			Type:   TOKEN_IDENTIFIER,
			Lexeme: annotationName,
			Line:   l.line,
			Column: atColumn + 1, // identifier starts right after @
		})
	}
}

// comment handles single-line comments starting with #
func (l *Lexer) comment() {
	// Consume until end of line
	for l.peek() != '\n' && !l.isAtEnd() {
		l.advance()
	}
	// Optionally add comment token
	// lexeme := l.source[l.start:l.current]
	// l.addTokenWithLiteral(TOKEN_COMMENT, lexeme)
}

// multilineComment handles multi-line comments ###...###
func (l *Lexer) multilineComment() {
	// Consume opening ###
	l.advance() // second #
	l.advance() // third #

	// Look for closing ###
	for !l.isAtEnd() {
		if l.peek() == '#' && l.peekNext() == '#' && l.peekNextNext() == '#' {
			// Found closing ###
			l.advance() // first #
			l.advance() // second #
			l.advance() // third #
			return
		}
		if l.peek() == '\n' {
			l.line++
			l.column = 0
		}
		l.advance()
	}

	l.addError("Unterminated multi-line comment")
}

// string handles string literals with support for escapes and interpolation
func (l *Lexer) string() {
	startLine := l.line
	startColumn := l.column - 1
	value := strings.Builder{}

	for !l.isAtEnd() && l.peek() != '"' {
		// Handle escape sequences
		if l.peek() == '\\' {
			l.advance() // consume backslash
			if l.isAtEnd() {
				break
			}

			escaped := l.advance()
			switch escaped {
			case 'n':
				value.WriteByte('\n')
			case 't':
				value.WriteByte('\t')
			case 'r':
				value.WriteByte('\r')
			case '\\':
				value.WriteByte('\\')
			case '"':
				value.WriteByte('"')
			case '#':
				value.WriteByte('#')
			default:
				// Unknown escape sequence - keep as-is
				value.WriteByte('\\')
				value.WriteByte(escaped)
			}
		} else if l.peek() == '\n' {
			// Multi-line strings are allowed
			value.WriteByte('\n')
			l.line++
			l.column = 0
			l.advance()
		} else {
			value.WriteByte(l.advance())
		}
	}

	if l.isAtEnd() {
		l.addError(fmt.Sprintf("Unterminated string starting at %d:%d", startLine, startColumn))
		return
	}

	// Consume closing "
	l.advance()

	// Create token with parsed string value
	token := Token{
		Type:    TOKEN_STRING_LITERAL,
		Lexeme:  l.source[l.start:l.current],
		Literal: value.String(),
		Line:    startLine,
		Column:  startColumn,
	}
	l.tokens = append(l.tokens, token)
}

// number handles integer and float literals
func (l *Lexer) number() {
	// Consume digits before decimal point
	for l.isDigit(l.peek()) || l.peek() == '_' {
		l.advance()
	}

	// Check for decimal point
	isFloat := false
	if l.peek() == '.' && l.isDigit(l.peekNext()) {
		isFloat = true
		l.advance() // consume .

		// Consume fractional digits
		for l.isDigit(l.peek()) || l.peek() == '_' {
			l.advance()
		}
	}

	// Check for scientific notation
	if l.peek() == 'e' || l.peek() == 'E' {
		isFloat = true
		l.advance() // consume e/E

		// Optional +/- sign
		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}

		// Consume exponent digits
		if !l.isDigit(l.peek()) {
			l.addError("Invalid number: expected digits after exponent")
			return
		}

		for l.isDigit(l.peek()) {
			l.advance()
		}
	}

	lexeme := l.source[l.start:l.current]
	// Remove underscores for parsing
	cleanLexeme := strings.ReplaceAll(lexeme, "_", "")

	if isFloat {
		value, err := strconv.ParseFloat(cleanLexeme, 64)
		if err != nil {
			l.addError(fmt.Sprintf("Invalid float literal: %s", lexeme))
			return
		}
		l.addTokenWithLiteral(TOKEN_FLOAT_LITERAL, value)
	} else {
		value, err := strconv.ParseInt(cleanLexeme, 10, 64)
		if err != nil {
			l.addError(fmt.Sprintf("Invalid integer literal: %s", lexeme))
			return
		}
		l.addTokenWithLiteral(TOKEN_INT_LITERAL, value)
	}
}

// identifier handles identifiers and keywords
func (l *Lexer) identifier() {
	for l.isAlphaNumeric(l.peek()) {
		l.advance()
	}

	text := l.source[l.start:l.current]

	// Check if it's a keyword
	tokenType, isKeyword := Keywords[text]
	if !isKeyword {
		tokenType = TOKEN_IDENTIFIER
	}

	// For boolean literals, set the literal value
	var literal interface{}
	if tokenType == TOKEN_TRUE {
		literal = true
	} else if tokenType == TOKEN_FALSE {
		literal = false
	} else if tokenType == TOKEN_NULL {
		literal = nil
	}

	if literal != nil {
		l.addTokenWithLiteral(tokenType, literal)
	} else {
		l.addToken(tokenType)
	}
}

// Helper methods

// isAtEnd checks if we've reached the end of the source
func (l *Lexer) isAtEnd() bool {
	return l.current >= len(l.source)
}

// advance consumes and returns the current character
func (l *Lexer) advance() byte {
	if l.isAtEnd() {
		return 0
	}
	c := l.source[l.current]
	l.current++
	l.column++
	return c
}

// match checks if the current character matches expected and consumes it
func (l *Lexer) match(expected byte) bool {
	if l.isAtEnd() {
		return false
	}
	if l.source[l.current] != expected {
		return false
	}
	l.current++
	l.column++
	return true
}

// peek returns the current character without consuming it
func (l *Lexer) peek() byte {
	if l.isAtEnd() {
		return 0
	}
	return l.source[l.current]
}

// peekNext returns the next character without consuming
func (l *Lexer) peekNext() byte {
	if l.current+1 >= len(l.source) {
		return 0
	}
	return l.source[l.current+1]
}

// peekNextNext returns the character two positions ahead
func (l *Lexer) peekNextNext() byte {
	if l.current+2 >= len(l.source) {
		return 0
	}
	return l.source[l.current+2]
}

// isDigit checks if a character is a digit
func (l *Lexer) isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isAlpha checks if a character is alphabetic or underscore
func (l *Lexer) isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_'
}

// isAlphaNumeric checks if a character is alphanumeric or underscore
func (l *Lexer) isAlphaNumeric(c byte) bool {
	return l.isAlpha(c) || l.isDigit(c)
}

// addToken adds a token with the current lexeme
func (l *Lexer) addToken(tokenType TokenType) {
	l.addTokenWithLiteral(tokenType, nil)
}

// addTokenWithLiteral adds a token with a literal value
func (l *Lexer) addTokenWithLiteral(tokenType TokenType, literal interface{}) {
	lexeme := l.source[l.start:l.current]
	token := Token{
		Type:    tokenType,
		Lexeme:  lexeme,
		Literal: literal,
		Line:    l.line,
		Column:  l.column - (l.current - l.start),
	}
	l.tokens = append(l.tokens, token)
}

// addError records a lexical error
func (l *Lexer) addError(message string) {
	lexeme := ""
	if l.start < len(l.source) {
		end := l.current
		if end > l.start+20 {
			end = l.start + 20
		}
		lexeme = l.source[l.start:end]
	}

	err := LexError{
		Message: message,
		Line:    l.line,
		Column:  l.column - (l.current - l.start),
		Lexeme:  lexeme,
	}
	l.errors = append(l.errors, err)
}

// IsKeyword checks if a string is a Conduit keyword
func IsKeyword(s string) bool {
	_, ok := Keywords[s]
	return ok
}

// IsType checks if a token type represents a primitive type
func IsType(t TokenType) bool {
	return t >= TOKEN_STRING && t <= TOKEN_JSON
}

// IsPrimitiveType checks if a string is a primitive type name
func IsPrimitiveType(s string) bool {
	primitiveTypes := map[string]bool{
		"string": true, "text": true, "markdown": true,
		"int": true, "float": true, "decimal": true, "bool": true,
		"timestamp": true, "date": true, "time": true,
		"uuid": true, "ulid": true,
		"email": true, "url": true, "phone": true,
		"json":  true,
		"array": true, "hash": true, "enum": true,
	}
	return primitiveTypes[s]
}

// IsValidIdentifier checks if a string is a valid identifier
func IsValidIdentifier(s string) bool {
	if s == "" {
		return false
	}

	// First character must be letter or underscore
	first := rune(s[0])
	if !unicode.IsLetter(first) && first != '_' {
		return false
	}

	// Remaining characters must be letters, digits, or underscores
	for _, r := range s[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}

	// Must not be a keyword
	return !IsKeyword(s)
}
