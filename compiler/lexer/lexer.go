package lexer

import (
	"strconv"
	"strings"
	"unicode"
)

// Lexer tokenizes Conduit source code
type Lexer struct {
	source          []rune   // Source code as runes for Unicode support
	start           int      // Start position of current token
	current         int      // Current position in source
	line            int      // Current line number
	column          int      // Current column number
	startColumn     int      // Column where current token started
	file            string   // Source file path
	tokens          []Token  // Collected tokens
	errors          []LexError // Collected errors
	preserveComments bool     // Flag to preserve comments for LSP
}

// New creates a new Lexer for the given source code
func New(source, file string) *Lexer {
	return &Lexer{
		source:          []rune(source),
		start:           0,
		current:         0,
		line:            1,
		column:          1,
		startColumn:     1,
		file:            file,
		tokens:          make([]Token, 0, len(source)/10), // Pre-allocate based on estimate
		errors:          make([]LexError, 0),
		preserveComments: false,
	}
}

// SetPreserveComments sets whether to preserve comments in the token stream
func (l *Lexer) SetPreserveComments(preserve bool) {
	l.preserveComments = preserve
}

// ScanTokens scans all tokens from the source and returns them with any errors
func (l *Lexer) ScanTokens() ([]Token, []LexError) {
	for !l.isAtEnd() {
		l.start = l.current
		l.startColumn = l.column
		l.scanToken()
	}

	// Add EOF token
	l.tokens = append(l.tokens, Token{
		Type:   TOKEN_EOF,
		Lexeme: "",
		Line:   l.line,
		Column: l.column,
		File:   l.file,
		Start:  l.current,
		End:    l.current,
	})

	return l.tokens, l.errors
}

// scanToken scans a single token
func (l *Lexer) scanToken() {
	r := l.advance()

	switch r {
	// Single-character tokens
	case '(':
		l.addToken(TOKEN_LPAREN, nil)
	case ')':
		l.addToken(TOKEN_RPAREN, nil)
	case '{':
		l.addToken(TOKEN_LBRACE, nil)
	case '}':
		l.addToken(TOKEN_RBRACE, nil)
	case '[':
		l.addToken(TOKEN_LBRACKET, nil)
	case ']':
		l.addToken(TOKEN_RBRACKET, nil)
	case ',':
		l.addToken(TOKEN_COMMA, nil)
	case ':':
		l.addToken(TOKEN_COLON, nil)
	case '%':
		l.addToken(TOKEN_PERCENT, nil)
	case '+':
		l.addToken(TOKEN_PLUS, nil)

	// Potentially multi-character tokens
	case '@':
		l.addToken(TOKEN_AT, nil)
	case '!':
		if l.match('=') {
			l.addToken(TOKEN_BANG_EQUAL, nil)
		} else {
			l.addToken(TOKEN_BANG, nil)
		}
	case '?':
		if l.match('.') {
			l.addToken(TOKEN_QUESTION_DOT, nil)
		} else if l.match('?') {
			l.addToken(TOKEN_QUESTION_QUESTION, nil)
		} else {
			l.addToken(TOKEN_QUESTION, nil)
		}
	case '=':
		if l.match('=') {
			l.addToken(TOKEN_EQUAL_EQUAL, nil)
		} else if l.match('>') {
			l.addToken(TOKEN_FAT_ARROW, nil)
		} else {
			l.addToken(TOKEN_EQUAL, nil)
		}
	case '<':
		if l.match('=') {
			l.addToken(TOKEN_LESS_EQUAL, nil)
		} else {
			l.addToken(TOKEN_LESS, nil)
		}
	case '>':
		if l.match('=') {
			l.addToken(TOKEN_GREATER_EQUAL, nil)
		} else {
			l.addToken(TOKEN_GREATER, nil)
		}
	case '*':
		if l.match('*') {
			l.addToken(TOKEN_STAR_STAR, nil)
		} else {
			l.addToken(TOKEN_STAR, nil)
		}
	case '|':
		if l.match('|') {
			l.addToken(TOKEN_PIPE_PIPE, nil)
		} else {
			l.addToken(TOKEN_PIPE, nil)
		}
	case '&':
		if l.match('&') {
			l.addToken(TOKEN_AMPERSAND_AMPERSAND, nil)
		} else {
			l.addToken(TOKEN_AMPERSAND, nil)
		}
	case '-':
		if l.match('>') {
			l.addToken(TOKEN_ARROW, nil)
		} else {
			l.addToken(TOKEN_MINUS, nil)
		}
	case '.':
		// Check if it's part of a float literal
		if l.isDigit(l.peek()) {
			// This is the decimal point of a float that started without a leading digit
			// Go back and scan as number
			l.current--
			l.column--
			l.scanNumber()
		} else {
			l.addToken(TOKEN_DOT, nil)
		}

	// Comments
	case '#':
		if l.match('{') {
			// String interpolation start
			l.addToken(TOKEN_STRING_INTERP_START, nil)
		} else {
			// Single-line comment
			l.scanComment()
		}

	// Division or regex (context-free, just tokenize as division)
	case '/':
		// Check for regex patterns /pattern/
		// For now, treat as division. Parser will handle regex context.
		l.addToken(TOKEN_SLASH, nil)

	// String literals
	case '"':
		l.scanString()

	// Whitespace
	case ' ', '\r', '\t':
		// Ignore whitespace
		break

	case '\n':
		l.line++
		l.column = 0 // Will be incremented to 1 on next advance
		// Optionally add newline tokens for LSP
		if l.preserveComments {
			l.addToken(TOKEN_NEWLINE, nil)
		}

	default:
		if l.isDigit(r) {
			l.scanNumber()
		} else if l.isAlpha(r) {
			l.scanIdentifier()
		} else {
			l.addError("Unexpected character: " + string(r))
		}
	}
}

// scanComment scans a single-line comment starting with #
func (l *Lexer) scanComment() {
	// Consume until end of line
	for !l.isAtEnd() && l.peek() != '\n' {
		l.advance()
	}

	if l.preserveComments {
		lexeme := string(l.source[l.start:l.current])
		l.addToken(TOKEN_COMMENT, lexeme)
	}
}

// scanString scans a string literal, handling escape sequences and multi-line strings
func (l *Lexer) scanString() {
	startLine := l.line
	var builder strings.Builder

	for !l.isAtEnd() && l.peek() != '"' {
		if l.peek() == '\n' {
			l.line++
			l.column = 0
		}

		if l.peek() == '\\' {
			// Handle escape sequences
			l.advance() // consume backslash
			if l.isAtEnd() {
				l.addError("Unterminated string")
				return
			}

			escaped := l.advance()
			switch escaped {
			case 'n':
				builder.WriteRune('\n')
			case 't':
				builder.WriteRune('\t')
			case 'r':
				builder.WriteRune('\r')
			case '\\':
				builder.WriteRune('\\')
			case '"':
				builder.WriteRune('"')
			case '#':
				builder.WriteRune('#')
			default:
				// Invalid escape sequence, but include it
				builder.WriteRune('\\')
				builder.WriteRune(escaped)
			}
		} else {
			builder.WriteRune(l.advance())
		}
	}

	if l.isAtEnd() {
		l.addError("Unterminated string starting at line " + strconv.Itoa(startLine))
		return
	}

	// Consume closing quote
	l.advance()

	l.addToken(TOKEN_STRING_LITERAL, builder.String())
}

// scanNumber scans an integer or float literal
func (l *Lexer) scanNumber() {
	// Scan integer part
	for l.isDigit(l.peek()) || l.peek() == '_' {
		l.advance()
	}

	// Check for decimal point
	isFloat := false
	if l.peek() == '.' && l.isDigit(l.peekNext()) {
		isFloat = true
		l.advance() // consume '.'

		// Scan fractional part
		for l.isDigit(l.peek()) || l.peek() == '_' {
			l.advance()
		}
	}

	// Check for scientific notation
	if l.peek() == 'e' || l.peek() == 'E' {
		isFloat = true
		l.advance() // consume 'e' or 'E'

		if l.peek() == '+' || l.peek() == '-' {
			l.advance() // consume sign
		}

		if !l.isDigit(l.peek()) {
			l.addError("Invalid scientific notation")
			return
		}

		for l.isDigit(l.peek()) {
			l.advance()
		}
	}

	// Get the lexeme and remove underscores
	lexeme := string(l.source[l.start:l.current])
	cleanLexeme := strings.ReplaceAll(lexeme, "_", "")

	if isFloat {
		value, err := strconv.ParseFloat(cleanLexeme, 64)
		if err != nil {
			l.addError("Invalid float literal: " + err.Error())
			return
		}
		l.addToken(TOKEN_FLOAT_LITERAL, value)
	} else {
		value, err := strconv.ParseInt(cleanLexeme, 10, 64)
		if err != nil {
			l.addError("Invalid integer literal: " + err.Error())
			return
		}
		l.addToken(TOKEN_INT_LITERAL, value)
	}
}

// scanIdentifier scans an identifier or keyword
func (l *Lexer) scanIdentifier() {
	for l.isAlphaNumeric(l.peek()) {
		l.advance()
	}

	lexeme := string(l.source[l.start:l.current])

	// Check if it's a keyword first
	tokenType, isKeyword := lookupKeyword(lexeme)
	if isKeyword {
		// Keywords don't include the ? suffix
		l.addToken(tokenType, nil)
		return
	}

	// Not a keyword - check if identifier ends with ? (for predicates like empty?, exists?)
	// But NOT if it's followed by . (that would be safe navigation ?.)
	if l.peek() == '?' && l.peekNext() != '.' {
		l.advance()
		lexeme = string(l.source[l.start:l.current])
	}

	l.addToken(TOKEN_IDENTIFIER, lexeme)
}

// Helper methods

// isAtEnd checks if we've reached the end of the source
func (l *Lexer) isAtEnd() bool {
	return l.current >= len(l.source)
}

// advance consumes and returns the current character
func (l *Lexer) advance() rune {
	if l.isAtEnd() {
		return 0
	}
	r := l.source[l.current]
	l.current++
	l.column++
	return r
}

// match checks if the current character matches the expected character
// If it matches, consumes it and returns true
func (l *Lexer) match(expected rune) bool {
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
func (l *Lexer) peek() rune {
	if l.isAtEnd() {
		return 0
	}
	return l.source[l.current]
}

// peekNext returns the next character without consuming it
func (l *Lexer) peekNext() rune {
	if l.current+1 >= len(l.source) {
		return 0
	}
	return l.source[l.current+1]
}

// isDigit checks if a rune is a digit
func (l *Lexer) isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isAlpha checks if a rune is alphabetic or underscore
func (l *Lexer) isAlpha(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

// isAlphaNumeric checks if a rune is alphanumeric or underscore
func (l *Lexer) isAlphaNumeric(r rune) bool {
	return l.isAlpha(r) || l.isDigit(r)
}

// addToken adds a token to the token list
func (l *Lexer) addToken(tokenType TokenType, literal interface{}) {
	lexeme := string(l.source[l.start:l.current])
	l.tokens = append(l.tokens, Token{
		Type:    tokenType,
		Lexeme:  lexeme,
		Literal: literal,
		Line:    l.line,
		Column:  l.startColumn,
		File:    l.file,
		Start:   l.start,
		End:     l.current,
	})
}

// addError adds an error to the error list
func (l *Lexer) addError(message string) {
	l.errors = append(l.errors, LexError{
		Message: message,
		Line:    l.line,
		Column:  l.column,
		File:    l.file,
	})
}
