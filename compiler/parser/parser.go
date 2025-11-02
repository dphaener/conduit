package parser

import (
	"fmt"
	"github.com/conduit-lang/conduit/compiler/lexer"
)

// Parser transforms token streams into an Abstract Syntax Tree
type Parser struct {
	tokens    []lexer.Token
	current   int
	errors    []ParseError
	panicMode bool
	source    string // Original source text for preserving formatting
}

// New creates a new Parser from a token stream
func New(tokens []lexer.Token) *Parser {
	return &Parser{
		tokens:    tokens,
		current:   0,
		errors:    []ParseError{},
		panicMode: false,
		source:    "",
	}
}

// NewWithSource creates a new Parser with access to the original source text
// This enables better formatting preservation for block bodies
func NewWithSource(tokens []lexer.Token, source string) *Parser {
	return &Parser{
		tokens:    tokens,
		current:   0,
		errors:    []ParseError{},
		panicMode: false,
		source:    source,
	}
}

// Parse parses the token stream and returns the AST and any errors
func (p *Parser) Parse() (*Program, []ParseError) {
	program := p.parseProgram()
	return program, p.errors
}

// parseProgram parses the top-level program
func (p *Parser) parseProgram() *Program {
	resources := []*ResourceNode{}
	startToken := p.peek()

	for !p.isAtEnd() {
		// Skip newlines and comments at the top level
		if p.match(lexer.TOKEN_NEWLINE, lexer.TOKEN_COMMENT) {
			continue
		}

		if p.check(lexer.TOKEN_RESOURCE) {
			if res := p.parseResource(); res != nil {
				resources = append(resources, res)
			}
		} else {
			p.addError(ParseError{
				Message:  fmt.Sprintf("Unexpected token: %s. Expected 'resource' keyword.", p.peek().Lexeme),
				Location: TokenToLocation(p.peek()),
			})
			p.synchronize()
		}
	}

	return NewProgram(resources, TokenToLocation(startToken))
}

// Helper methods for token manipulation

// isAtEnd checks if we're at the end of the token stream
func (p *Parser) isAtEnd() bool {
	if p.current >= len(p.tokens) {
		return true
	}
	return p.tokens[p.current].Type == lexer.TOKEN_EOF
}

// peek returns the current token without consuming it
func (p *Parser) peek() lexer.Token {
	if p.current >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1] // Return EOF
	}
	return p.tokens[p.current]
}

// previous returns the previous token
func (p *Parser) previous() lexer.Token {
	if p.current > 0 {
		return p.tokens[p.current-1]
	}
	return p.tokens[0]
}

// advance consumes and returns the current token
func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

// check checks if the current token is of the given type
func (p *Parser) check(tokenType lexer.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == tokenType
}

// match checks if the current token matches any of the given types
// If it matches, consumes the token and returns true
func (p *Parser) match(types ...lexer.TokenType) bool {
	for _, tokenType := range types {
		if p.check(tokenType) {
			p.advance()
			return true
		}
	}
	return false
}

// consume consumes a token of the given type or adds an error
func (p *Parser) consume(tokenType lexer.TokenType, message string) (lexer.Token, bool) {
	if p.check(tokenType) {
		return p.advance(), true
	}

	p.addError(ParseError{
		Message:  message,
		Location: TokenToLocation(p.peek()),
	})
	return lexer.Token{}, false
}

// skipNewlines skips any newline tokens
func (p *Parser) skipNewlines() {
	for p.match(lexer.TOKEN_NEWLINE) {
		// Keep skipping
	}
}

// skipNewlinesAndComments skips newlines and comments
// Used when we want to skip over all whitespace and comments
func (p *Parser) skipNewlinesAndComments() {
	for p.match(lexer.TOKEN_NEWLINE, lexer.TOKEN_COMMENT) {
		// Keep skipping
	}
}

// collectLeadingComments collects comments that appear before the current token
// This captures all comment tokens that appear on their own lines before the target element
// It only looks back until it hits a non-comment, non-newline token or reaches a token
// that's on a different line than the comments
// Comments separated by more than 1 blank line from the current token are not collected
func (p *Parser) collectLeadingComments() string {
	var comments []string
	startIdx := p.current

	// First, calculate the distance from the current token to any preceding comments
	// Find the position of the first non-newline token before current
	firstNonNewlineIdx := startIdx - 1
	for firstNonNewlineIdx >= 0 && p.tokens[firstNonNewlineIdx].Type == lexer.TOKEN_NEWLINE {
		firstNonNewlineIdx--
	}

	// If we have a comment, check the line distance from current token to that comment
	if firstNonNewlineIdx >= 0 && p.tokens[firstNonNewlineIdx].Type == lexer.TOKEN_COMMENT {
		currentLine := p.peek().Line
		commentLine := p.tokens[firstNonNewlineIdx].Line
		lineDistance := currentLine - commentLine

		// If the comment is more than 2 lines away (allowing max 1 blank line), don't collect it
		if lineDistance > 2 {
			return ""
		}
	}

	// Skip back over newlines and comments to find comments on previous lines
	lastNonCommentLine := -1
	for i := startIdx - 1; i >= 0; i-- {
		tok := p.tokens[i]

		if tok.Type == lexer.TOKEN_COMMENT {
			// Only collect comments that are on their own line (not trailing comments)
			// Check if there's a newline before this comment or if it's at the start
			isLeadingComment := false
			if i == 0 {
				isLeadingComment = true
			} else {
				prevTok := p.tokens[i-1]
				// It's a leading comment if preceded by newline or another comment
				isLeadingComment = prevTok.Type == lexer.TOKEN_NEWLINE || prevTok.Type == lexer.TOKEN_COMMENT
			}

			if isLeadingComment {
				// Check if there's a gap of more than 1 blank line between consecutive comment blocks
				if len(comments) > 0 && lastNonCommentLine != -1 {
					lineGap := lastNonCommentLine - tok.Line
					// If gap is > 2 lines (more than 1 blank line), stop collecting
					if lineGap > 2 {
						break
					}
				}

				// Prepend comment (since we're going backward)
				comments = append([]string{tok.Lexeme}, comments...)
				lastNonCommentLine = tok.Line
			} else {
				// This is a trailing comment from the previous element, stop
				break
			}
		} else if tok.Type == lexer.TOKEN_NEWLINE {
			// Keep looking for more comments
			continue
		} else {
			// Hit a non-comment, non-newline token
			// If we've already collected comments, check the distance from this token to the first comment
			if len(comments) > 0 {
				firstCommentLine := lastNonCommentLine // Since we're going backward, last collected is the first
				tokenLine := tok.Line
				gap := firstCommentLine - tokenLine
				// If gap > 2 (more than 1 blank line between token and first comment), discard all comments
				if gap > 2 {
					return ""
				}
			}
			break
		}
	}

	if len(comments) == 0 {
		return ""
	}

	// Join comments with newlines
	result := ""
	for _, comment := range comments {
		result += comment + "\n"
	}
	return result
}

// peekForTrailingComment checks if there's a comment on the same line after the current position
// Returns the comment text if found, empty string otherwise
func (p *Parser) peekForTrailingComment() string {
	// Look ahead from current position
	for i := p.current; i < len(p.tokens); i++ {
		tok := p.tokens[i]
		if tok.Type == lexer.TOKEN_COMMENT {
			return tok.Lexeme
		} else if tok.Type == lexer.TOKEN_NEWLINE {
			// Hit newline before finding comment
			return ""
		} else if tok.Type == lexer.TOKEN_EOF || tok.Type == lexer.TOKEN_RBRACE {
			// Hit end of block
			return ""
		}
	}
	return ""
}

// consumeTrailingComment consumes a trailing comment if present on the current line
func (p *Parser) consumeTrailingComment() string {
	// Look for a comment on the SAME LINE as the previous token
	// We need to check if the next token is on the same line
	if p.isAtEnd() {
		return ""
	}

	previousLine := p.previous().Line

	// Check if the next non-newline token is a comment on the same line
	for !p.isAtEnd() {
		currentTok := p.peek()

		// If we hit a newline token, we're done (no trailing comment)
		if currentTok.Type == lexer.TOKEN_NEWLINE {
			return ""
		}

		// If we hit a closing brace, we're done
		if currentTok.Type == lexer.TOKEN_RBRACE {
			return ""
		}

		// If the current token is on a different line, we're done
		if currentTok.Line != previousLine {
			return ""
		}

		// If we found a comment on the same line, consume and return it
		if currentTok.Type == lexer.TOKEN_COMMENT {
			return p.advance().Lexeme
		}

		// Otherwise, this shouldn't happen in well-formed code
		// But we should NOT advance here - just return empty
		// The token will be processed by the next parser call
		return ""
	}

	return ""
}

// expectNewlineOrEOF expects a newline or EOF
func (p *Parser) expectNewlineOrEOF() {
	// If we're at EOF, closing brace, or already at a newline, we're good
	if p.isAtEnd() || p.check(lexer.TOKEN_RBRACE) {
		return
	}

	// Skip any newlines
	p.skipNewlines()
}

// Helper methods for parsing primitives

// parseIdentifier parses an identifier token
// Also accepts type keywords as identifiers (for field names)
func (p *Parser) parseIdentifier() (string, bool) {
	// Check if current token can be used as an identifier
	if p.check(lexer.TOKEN_IDENTIFIER) {
		token := p.advance()
		return token.Lexeme, true
	}

	// Allow type keywords to be used as field names
	if p.canBeFieldName() {
		token := p.advance()
		return token.Lexeme, true
	}

	p.addError(ParseError{
		Message:  "Expected identifier",
		Location: TokenToLocation(p.peek()),
	})
	return "", false
}

// canBeFieldName checks if the current token can be used as a field name
// Type keywords can be used as field names
func (p *Parser) canBeFieldName() bool {
	tokenType := p.peek().Type
	return p.isPrimitiveType(tokenType) || p.isStructuralType(tokenType)
}

// parseStringLiteral parses a string literal
func (p *Parser) parseStringLiteral() (string, bool) {
	token, ok := p.consume(lexer.TOKEN_STRING_LITERAL, "Expected string literal")
	if !ok {
		return "", false
	}
	if token.Literal != nil {
		return token.Literal.(string), true
	}
	return token.Lexeme, true
}

// parseIntLiteral parses an integer literal
func (p *Parser) parseIntLiteral() (int64, bool) {
	token, ok := p.consume(lexer.TOKEN_INT_LITERAL, "Expected integer literal")
	if !ok {
		return 0, false
	}
	if token.Literal != nil {
		return token.Literal.(int64), true
	}
	return 0, false
}

// parseFloatLiteral parses a float literal
func (p *Parser) parseFloatLiteral() (float64, bool) {
	token, ok := p.consume(lexer.TOKEN_FLOAT_LITERAL, "Expected float literal")
	if !ok {
		return 0.0, false
	}
	if token.Literal != nil {
		return token.Literal.(float64), true
	}
	return 0.0, false
}

// isPrimitiveType checks if the current token is a primitive type
func (p *Parser) isPrimitiveType(tokenType lexer.TokenType) bool {
	primitiveTypes := []lexer.TokenType{
		lexer.TOKEN_STRING,
		lexer.TOKEN_TEXT,
		lexer.TOKEN_INT,
		lexer.TOKEN_FLOAT,
		lexer.TOKEN_DECIMAL,
		lexer.TOKEN_BOOL,
		lexer.TOKEN_TIMESTAMP,
		lexer.TOKEN_DATE,
		lexer.TOKEN_TIME,
		lexer.TOKEN_UUID,
		lexer.TOKEN_ULID,
		lexer.TOKEN_EMAIL,
		lexer.TOKEN_URL,
		lexer.TOKEN_PHONE,
		lexer.TOKEN_JSON,
		lexer.TOKEN_MARKDOWN,
	}

	for _, pt := range primitiveTypes {
		if tokenType == pt {
			return true
		}
	}
	return false
}

// isStructuralType checks if the current token is a structural type
func (p *Parser) isStructuralType(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TOKEN_ARRAY ||
		tokenType == lexer.TOKEN_HASH ||
		tokenType == lexer.TOKEN_ENUM
}

// parseDocumentation parses documentation comments (///)
// This is called before parsing a resource
func (p *Parser) parseDocumentation() string {
	doc := ""

	// Look backward in tokens to find comment before current position
	for i := p.current - 1; i >= 0; i-- {
		token := p.tokens[i]
		if token.Type == lexer.TOKEN_COMMENT {
			// Extract documentation from comment
			if len(token.Lexeme) > 0 && token.Lexeme[0] == '#' {
				// Strip leading # and whitespace
				comment := token.Lexeme[1:]
				if len(comment) > 0 && comment[0] == ' ' {
					comment = comment[1:]
				}
				if doc == "" {
					doc = comment
				} else {
					doc = comment + "\n" + doc
				}
			}
		} else if token.Type != lexer.TOKEN_NEWLINE {
			// Stop at non-comment, non-newline
			break
		}
	}

	return doc
}

// addError adds a parse error to the error list
func (p *Parser) addError(err ParseError) {
	p.errors = append(p.errors, err)
	p.panicMode = true
}

// synchronize implements panic mode error recovery
// Skips tokens until we reach a synchronization point
func (p *Parser) synchronize() {
	p.panicMode = false
	p.advance()

	for !p.isAtEnd() {
		// Newlines are natural synchronization points
		if p.previous().Type == lexer.TOKEN_NEWLINE {
			return
		}

		// Start of new constructs are synchronization points
		switch p.peek().Type {
		case lexer.TOKEN_RESOURCE:
			return
		case lexer.TOKEN_AT:
			return
		}

		p.advance()
	}
}
