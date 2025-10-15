// Package parser implements the Conduit language parser, transforming token streams into Abstract Syntax Trees (ASTs).
// It uses recursive descent parsing with panic mode error recovery to handle syntax errors gracefully.
package parser

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
)

// ParseError represents an error encountered during parsing
type ParseError struct {
	Message  string
	Location ast.SourceLocation
	Token    lexer.Token
}

// Error implements the error interface
func (e *ParseError) Error() string {
	return fmt.Sprintf("Parse error at %d:%d: %s (near '%s')",
		e.Location.Line, e.Location.Column, e.Message, e.Token.Lexeme)
}

// NewParseError creates a new parse error
func NewParseError(message string, token lexer.Token) ParseError {
	return ParseError{
		Message: message,
		Location: ast.SourceLocation{
			Line:   token.Line,
			Column: token.Column,
		},
		Token: token,
	}
}

// ErrorType represents different categories of parse errors
type ErrorType int

const (
	// ErrorSyntax represents a general syntax error
	ErrorSyntax ErrorType = iota
	// ErrorUnexpectedToken represents an unexpected token error
	ErrorUnexpectedToken
	// ErrorMissingToken represents a missing expected token error
	ErrorMissingToken
	// ErrorInvalidType represents an invalid type specification error
	ErrorInvalidType
	// ErrorInvalidExpression represents an invalid expression error
	ErrorInvalidExpression
)

// ErrorRecoveryStrategy defines how the parser should recover from errors
type ErrorRecoveryStrategy int

const (
	// PanicMode skips tokens until a synchronization point
	PanicMode ErrorRecoveryStrategy = iota
	// PhraseLevel skips to the next valid phrase
	PhraseLevel
	// ErrorProduction inserts an error node in the AST
	ErrorProduction
)
