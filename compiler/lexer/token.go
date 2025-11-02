package lexer

import "fmt"

// TokenType represents the type of token in the Conduit language
type TokenType int

const (
	// Special tokens
	TOKEN_EOF TokenType = iota
	TOKEN_ERROR
	TOKEN_COMMENT
	TOKEN_NEWLINE

	// Keywords - Resource definition
	TOKEN_RESOURCE

	// Keywords - Lifecycle hooks
	TOKEN_BEFORE
	TOKEN_AFTER
	TOKEN_ON

	// Keywords - Transactions and async
	TOKEN_TRANSACTION
	TOKEN_ASYNC
	TOKEN_RESCUE

	// Keywords - Relationships
	TOKEN_HAS
	TOKEN_THROUGH
	TOKEN_AS
	TOKEN_UNDER
	TOKEN_NESTED

	// Keywords - Annotations
	TOKEN_FUNCTION
	TOKEN_VALIDATE
	TOKEN_CONSTRAINT
	TOKEN_INVARIANT
	TOKEN_COMPUTED
	TOKEN_SCOPE
	TOKEN_MIDDLEWARE
	TOKEN_OPERATIONS
	TOKEN_STRICT

	// Keywords - Control flow
	TOKEN_IF
	TOKEN_ELSIF
	TOKEN_ELSE
	TOKEN_UNLESS
	TOKEN_MATCH
	TOKEN_WHEN
	TOKEN_RETURN
	TOKEN_LET
	TOKEN_ERROR_KW // error() function keyword

	// Keywords - Operations
	TOKEN_CREATE
	TOKEN_UPDATE
	TOKEN_DELETE
	TOKEN_SAVE
	TOKEN_LIST
	TOKEN_GET

	// Type keywords - Primitives
	TOKEN_STRING
	TOKEN_TEXT
	TOKEN_INT
	TOKEN_FLOAT
	TOKEN_DECIMAL
	TOKEN_BOOL
	TOKEN_TIMESTAMP
	TOKEN_DATE
	TOKEN_TIME
	TOKEN_UUID
	TOKEN_ULID
	TOKEN_EMAIL
	TOKEN_URL
	TOKEN_PHONE
	TOKEN_JSON
	TOKEN_MARKDOWN

	// Type keywords - Structural
	TOKEN_ENUM
	TOKEN_ARRAY
	TOKEN_HASH

	// Literals
	TOKEN_IDENTIFIER
	TOKEN_INT_LITERAL
	TOKEN_FLOAT_LITERAL
	TOKEN_STRING_LITERAL
	TOKEN_TRUE
	TOKEN_FALSE
	TOKEN_NIL
	TOKEN_SELF

	// Operators - Single character
	TOKEN_BANG     // !
	TOKEN_QUESTION // ?
	TOKEN_AT       // @
	TOKEN_PIPE     // |
	TOKEN_COLON    // :
	TOKEN_DOT      // .
	TOKEN_COMMA    // ,
	TOKEN_PLUS     // +
	TOKEN_MINUS    // -
	TOKEN_STAR     // *
	TOKEN_SLASH    // /
	TOKEN_PERCENT  // %
	TOKEN_LESS     // <
	TOKEN_GREATER  // >
	TOKEN_EQUAL    // =
	TOKEN_AMPERSAND // &

	// Operators - Multi-character
	TOKEN_ARROW          // ->
	TOKEN_EQUAL_EQUAL    // ==
	TOKEN_BANG_EQUAL     // !=
	TOKEN_LESS_EQUAL     // <=
	TOKEN_GREATER_EQUAL  // >=
	TOKEN_STAR_STAR      // **
	TOKEN_QUESTION_DOT   // ?.
	TOKEN_QUESTION_QUESTION // ??
	TOKEN_PIPE_PIPE      // ||
	TOKEN_AMPERSAND_AMPERSAND // &&
	TOKEN_FAT_ARROW      // =>
	TOKEN_STRING_INTERP_START // #{

	// Delimiters
	TOKEN_LBRACE   // {
	TOKEN_RBRACE   // }
	TOKEN_LPAREN   // (
	TOKEN_RPAREN   // )
	TOKEN_LBRACKET // [
	TOKEN_RBRACKET // ]

	// Keywords - Query operations
	TOKEN_WHERE
	TOKEN_IN
	TOKEN_NOT_IN
	TOKEN_ORDER_BY
	TOKEN_LIMIT
	TOKEN_OFFSET
	TOKEN_JOINS
	TOKEN_INCLUDES

	// Keywords - Relationship actions
	TOKEN_RESTRICT
	TOKEN_CASCADE
	TOKEN_SET_NULL
	TOKEN_NO_ACTION

	// Keywords - Field modifiers
	TOKEN_PRIMARY
	TOKEN_AUTO
	TOKEN_AUTO_UPDATE
	TOKEN_UNIQUE
	TOKEN_DEFAULT
	TOKEN_MIN
	TOKEN_MAX
	TOKEN_PATTERN
	TOKEN_REQUIRED

	// Keywords - Other
	TOKEN_CONDITION
	TOKEN_FOREIGN_KEY
	TOKEN_ON_DELETE
	TOKEN_ON_UPDATE
	TOKEN_NULLABILITY
)

// Token represents a single lexical token
type Token struct {
	Type    TokenType
	Lexeme  string
	Literal interface{} // For literals (numbers, strings, etc.)
	Line    int
	Column  int
	File    string // Source file path
	Start   int    // Byte offset in source where token starts
	End     int    // Byte offset in source where token ends (exclusive)
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	switch t {
	case TOKEN_EOF:
		return "EOF"
	case TOKEN_ERROR:
		return "ERROR"
	case TOKEN_COMMENT:
		return "COMMENT"
	case TOKEN_NEWLINE:
		return "NEWLINE"
	case TOKEN_RESOURCE:
		return "RESOURCE"
	case TOKEN_BEFORE:
		return "BEFORE"
	case TOKEN_AFTER:
		return "AFTER"
	case TOKEN_ON:
		return "ON"
	case TOKEN_TRANSACTION:
		return "TRANSACTION"
	case TOKEN_ASYNC:
		return "ASYNC"
	case TOKEN_RESCUE:
		return "RESCUE"
	case TOKEN_HAS:
		return "HAS"
	case TOKEN_THROUGH:
		return "THROUGH"
	case TOKEN_AS:
		return "AS"
	case TOKEN_UNDER:
		return "UNDER"
	case TOKEN_NESTED:
		return "NESTED"
	case TOKEN_FUNCTION:
		return "FUNCTION"
	case TOKEN_VALIDATE:
		return "VALIDATE"
	case TOKEN_CONSTRAINT:
		return "CONSTRAINT"
	case TOKEN_INVARIANT:
		return "INVARIANT"
	case TOKEN_COMPUTED:
		return "COMPUTED"
	case TOKEN_SCOPE:
		return "SCOPE"
	case TOKEN_MIDDLEWARE:
		return "MIDDLEWARE"
	case TOKEN_OPERATIONS:
		return "OPERATIONS"
	case TOKEN_STRICT:
		return "STRICT"
	case TOKEN_IF:
		return "IF"
	case TOKEN_ELSIF:
		return "ELSIF"
	case TOKEN_ELSE:
		return "ELSE"
	case TOKEN_UNLESS:
		return "UNLESS"
	case TOKEN_MATCH:
		return "MATCH"
	case TOKEN_WHEN:
		return "WHEN"
	case TOKEN_RETURN:
		return "RETURN"
	case TOKEN_LET:
		return "LET"
	case TOKEN_ERROR_KW:
		return "ERROR_KW"
	case TOKEN_CREATE:
		return "CREATE"
	case TOKEN_UPDATE:
		return "UPDATE"
	case TOKEN_DELETE:
		return "DELETE"
	case TOKEN_SAVE:
		return "SAVE"
	case TOKEN_LIST:
		return "LIST"
	case TOKEN_GET:
		return "GET"
	case TOKEN_STRING:
		return "STRING"
	case TOKEN_TEXT:
		return "TEXT"
	case TOKEN_INT:
		return "INT"
	case TOKEN_FLOAT:
		return "FLOAT"
	case TOKEN_DECIMAL:
		return "DECIMAL"
	case TOKEN_BOOL:
		return "BOOL"
	case TOKEN_TIMESTAMP:
		return "TIMESTAMP"
	case TOKEN_DATE:
		return "DATE"
	case TOKEN_TIME:
		return "TIME"
	case TOKEN_UUID:
		return "UUID"
	case TOKEN_ULID:
		return "ULID"
	case TOKEN_EMAIL:
		return "EMAIL"
	case TOKEN_URL:
		return "URL"
	case TOKEN_PHONE:
		return "PHONE"
	case TOKEN_JSON:
		return "JSON"
	case TOKEN_MARKDOWN:
		return "MARKDOWN"
	case TOKEN_ENUM:
		return "ENUM"
	case TOKEN_ARRAY:
		return "ARRAY"
	case TOKEN_HASH:
		return "HASH"
	case TOKEN_IDENTIFIER:
		return "IDENTIFIER"
	case TOKEN_INT_LITERAL:
		return "INT_LITERAL"
	case TOKEN_FLOAT_LITERAL:
		return "FLOAT_LITERAL"
	case TOKEN_STRING_LITERAL:
		return "STRING_LITERAL"
	case TOKEN_TRUE:
		return "TRUE"
	case TOKEN_FALSE:
		return "FALSE"
	case TOKEN_NIL:
		return "NIL"
	case TOKEN_SELF:
		return "SELF"
	case TOKEN_BANG:
		return "BANG"
	case TOKEN_QUESTION:
		return "QUESTION"
	case TOKEN_AT:
		return "AT"
	case TOKEN_PIPE:
		return "PIPE"
	case TOKEN_COLON:
		return "COLON"
	case TOKEN_DOT:
		return "DOT"
	case TOKEN_COMMA:
		return "COMMA"
	case TOKEN_PLUS:
		return "PLUS"
	case TOKEN_MINUS:
		return "MINUS"
	case TOKEN_STAR:
		return "STAR"
	case TOKEN_SLASH:
		return "SLASH"
	case TOKEN_PERCENT:
		return "PERCENT"
	case TOKEN_LESS:
		return "LESS"
	case TOKEN_GREATER:
		return "GREATER"
	case TOKEN_EQUAL:
		return "EQUAL"
	case TOKEN_AMPERSAND:
		return "AMPERSAND"
	case TOKEN_ARROW:
		return "ARROW"
	case TOKEN_EQUAL_EQUAL:
		return "EQUAL_EQUAL"
	case TOKEN_BANG_EQUAL:
		return "BANG_EQUAL"
	case TOKEN_LESS_EQUAL:
		return "LESS_EQUAL"
	case TOKEN_GREATER_EQUAL:
		return "GREATER_EQUAL"
	case TOKEN_STAR_STAR:
		return "STAR_STAR"
	case TOKEN_QUESTION_DOT:
		return "QUESTION_DOT"
	case TOKEN_QUESTION_QUESTION:
		return "QUESTION_QUESTION"
	case TOKEN_PIPE_PIPE:
		return "PIPE_PIPE"
	case TOKEN_AMPERSAND_AMPERSAND:
		return "AMPERSAND_AMPERSAND"
	case TOKEN_FAT_ARROW:
		return "FAT_ARROW"
	case TOKEN_STRING_INTERP_START:
		return "STRING_INTERP_START"
	case TOKEN_LBRACE:
		return "LBRACE"
	case TOKEN_RBRACE:
		return "RBRACE"
	case TOKEN_LPAREN:
		return "LPAREN"
	case TOKEN_RPAREN:
		return "RPAREN"
	case TOKEN_LBRACKET:
		return "LBRACKET"
	case TOKEN_RBRACKET:
		return "RBRACKET"
	case TOKEN_WHERE:
		return "WHERE"
	case TOKEN_IN:
		return "IN"
	case TOKEN_NOT_IN:
		return "NOT_IN"
	case TOKEN_ORDER_BY:
		return "ORDER_BY"
	case TOKEN_LIMIT:
		return "LIMIT"
	case TOKEN_OFFSET:
		return "OFFSET"
	case TOKEN_JOINS:
		return "JOINS"
	case TOKEN_INCLUDES:
		return "INCLUDES"
	case TOKEN_RESTRICT:
		return "RESTRICT"
	case TOKEN_CASCADE:
		return "CASCADE"
	case TOKEN_SET_NULL:
		return "SET_NULL"
	case TOKEN_NO_ACTION:
		return "NO_ACTION"
	case TOKEN_PRIMARY:
		return "PRIMARY"
	case TOKEN_AUTO:
		return "AUTO"
	case TOKEN_AUTO_UPDATE:
		return "AUTO_UPDATE"
	case TOKEN_UNIQUE:
		return "UNIQUE"
	case TOKEN_DEFAULT:
		return "DEFAULT"
	case TOKEN_MIN:
		return "MIN"
	case TOKEN_MAX:
		return "MAX"
	case TOKEN_PATTERN:
		return "PATTERN"
	case TOKEN_REQUIRED:
		return "REQUIRED"
	case TOKEN_CONDITION:
		return "CONDITION"
	case TOKEN_FOREIGN_KEY:
		return "FOREIGN_KEY"
	case TOKEN_ON_DELETE:
		return "ON_DELETE"
	case TOKEN_ON_UPDATE:
		return "ON_UPDATE"
	case TOKEN_NULLABILITY:
		return "NULLABILITY"
	default:
		return "UNKNOWN"
	}
}

// String returns a string representation of the token
func (t Token) String() string {
	if t.Literal != nil {
		return fmt.Sprintf("%s(%v) [%d:%d]", t.Type, t.Literal, t.Line, t.Column)
	}
	return fmt.Sprintf("%s(%s) [%d:%d]", t.Type, t.Lexeme, t.Line, t.Column)
}

// LexError represents a lexical analysis error
type LexError struct {
	Message string
	Line    int
	Column  int
	File    string
}

// Error implements the error interface
func (e LexError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
}
