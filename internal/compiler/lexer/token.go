package lexer

import "fmt"

// TokenType represents the type of a token in the Conduit language
type TokenType int

const (
	// TOKEN_EOF marks the end of the token stream.
	TOKEN_EOF TokenType = iota
	// TOKEN_ERROR represents a lexical error encountered during scanning.
	TOKEN_ERROR
	// TOKEN_COMMENT contains comment text (currently not emitted).
	TOKEN_COMMENT
	// TOKEN_NEWLINE represents a line break (currently not emitted).
	TOKEN_NEWLINE

	// TOKEN_RESOURCE marks the 'resource' keyword for defining resources.
	TOKEN_RESOURCE
	// TOKEN_ENUM marks the 'enum' keyword for inline enum types.
	TOKEN_ENUM
	// TOKEN_ARRAY marks the 'array' keyword for array types.
	TOKEN_ARRAY
	// TOKEN_HASH marks the 'hash' keyword for hash/map types.
	TOKEN_HASH

	// TOKEN_ON marks the 'on' keyword for lifecycle hooks.
	TOKEN_ON
	// TOKEN_AFTER marks the 'after' keyword for post-operation hooks.
	TOKEN_AFTER
	// TOKEN_BEFORE marks the 'before' keyword for pre-operation hooks.
	TOKEN_BEFORE
	// TOKEN_TRANSACTION marks the '@transaction' annotation.
	TOKEN_TRANSACTION
	// TOKEN_ASYNC marks the '@async' annotation.
	TOKEN_ASYNC
	// TOKEN_RESCUE marks the 'rescue' keyword for error handling.
	TOKEN_RESCUE

	// Keywords - Query and scope
	TOKEN_WHERE // where
	TOKEN_HAS   // has

	// Keywords - Annotations
	TOKEN_HAS_MANY    // @has_many
	TOKEN_NESTED      // @nested
	TOKEN_MIDDLEWARE  // @middleware
	TOKEN_FUNCTION    // @function
	TOKEN_VALIDATE    // @validate
	TOKEN_CONSTRAINT  // @constraint
	TOKEN_INVARIANT   // @invariant
	TOKEN_COMPUTED    // @computed
	TOKEN_SCOPE       // @scope
	TOKEN_OPERATIONS  // @operations
	TOKEN_PRIMARY     // @primary
	TOKEN_AUTO        // @auto
	TOKEN_AUTO_UPDATE // @auto_update
	TOKEN_UNIQUE      // @unique
	TOKEN_REQUIRED    // @required (deprecated but recognized)
	TOKEN_DEFAULT     // @default
	TOKEN_MIN         // @min
	TOKEN_MAX         // @max
	TOKEN_PATTERN     // @pattern
	TOKEN_STRICT      // @strict

	// Keywords - Control flow
	TOKEN_IF        // if
	TOKEN_ELSIF     // elsif
	TOKEN_ELSE      // else
	TOKEN_UNLESS    // unless
	TOKEN_RETURN    // return
	TOKEN_LET       // let
	TOKEN_MATCH     // match
	TOKEN_WHEN      // when
	TOKEN_CONDITION // condition
	TOKEN_ERROR_KW  // error (keyword for error messages in constraints)

	// Keywords - Special identifiers
	TOKEN_SELF    // self
	TOKEN_THROUGH // through
	TOKEN_AS      // as
	TOKEN_UNDER   // under
	TOKEN_IN      // in
	TOKEN_NOT_IN  // not_in
	TOKEN_OR      // or
	TOKEN_AND     // and
	TOKEN_NOT     // not
	TOKEN_BY      // by

	// Primitive types
	TOKEN_STRING    // string
	TOKEN_TEXT      // text
	TOKEN_MARKDOWN  // markdown
	TOKEN_INT       // int
	TOKEN_FLOAT     // float
	TOKEN_DECIMAL   // decimal
	TOKEN_BOOL      // bool
	TOKEN_TIMESTAMP // timestamp
	TOKEN_DATE      // date
	TOKEN_TIME      // time
	TOKEN_UUID      // uuid
	TOKEN_ULID      // ulid
	TOKEN_EMAIL     // email
	TOKEN_URL       // url
	TOKEN_PHONE     // phone
	TOKEN_JSON      // json

	// Literals
	TOKEN_IDENTIFIER     // user_name, slugify, etc.
	TOKEN_INT_LITERAL    // 42, 1000, etc.
	TOKEN_FLOAT_LITERAL  // 3.14, 2.5e10, etc.
	TOKEN_STRING_LITERAL // "hello", "multi\nline", etc.
	TOKEN_TRUE           // true
	TOKEN_FALSE          // false
	TOKEN_NULL           // null, nil

	// Operators - Single character
	TOKEN_BANG     // !
	TOKEN_QUESTION // ?
	TOKEN_AT       // @
	TOKEN_PIPE     // |
	TOKEN_COLON    // :
	TOKEN_DOT      // .
	TOKEN_COMMA    // ,
	TOKEN_EQUALS   // =
	TOKEN_PLUS     // +
	TOKEN_MINUS    // -
	TOKEN_STAR     // *
	TOKEN_SLASH    // /
	TOKEN_PERCENT  // %
	TOKEN_LT       // <
	TOKEN_GT       // >

	// Operators - Two character
	TOKEN_ARROW           // ->
	TOKEN_EQ              // ==
	TOKEN_NEQ             // !=
	TOKEN_LTE             // <=
	TOKEN_GTE             // >=
	TOKEN_DOUBLE_PIPE     // ||
	TOKEN_DOUBLE_AMP      // &&
	TOKEN_DOUBLE_STAR     // **
	TOKEN_DOUBLE_COLON    // ::
	TOKEN_DOUBLE_QUESTION // ??
	TOKEN_SAFE_NAV        // ?.

	// Delimiters
	TOKEN_LBRACE    // {
	TOKEN_RBRACE    // }
	TOKEN_LPAREN    // (
	TOKEN_RPAREN    // )
	TOKEN_LBRACKET  // [
	TOKEN_RBRACKET  // ]
	TOKEN_HASH_MARK // # (for comments)

	// String interpolation
	TOKEN_INTERPOLATION_START // #{
	TOKEN_INTERPOLATION_END   // } (when inside interpolation)
)

// TokenTypeNames maps token types to their string representations
var TokenTypeNames = map[TokenType]string{
	TOKEN_EOF:                 "EOF",
	TOKEN_ERROR:               "ERROR",
	TOKEN_COMMENT:             "COMMENT",
	TOKEN_NEWLINE:             "NEWLINE",
	TOKEN_RESOURCE:            "RESOURCE",
	TOKEN_ENUM:                "ENUM",
	TOKEN_ARRAY:               "ARRAY",
	TOKEN_HASH:                "HASH",
	TOKEN_ON:                  "ON",
	TOKEN_AFTER:               "AFTER",
	TOKEN_BEFORE:              "BEFORE",
	TOKEN_TRANSACTION:         "TRANSACTION",
	TOKEN_ASYNC:               "ASYNC",
	TOKEN_RESCUE:              "RESCUE",
	TOKEN_WHERE:               "WHERE",
	TOKEN_HAS:                 "HAS",
	TOKEN_HAS_MANY:            "HAS_MANY",
	TOKEN_NESTED:              "NESTED",
	TOKEN_MIDDLEWARE:          "MIDDLEWARE",
	TOKEN_FUNCTION:            "FUNCTION",
	TOKEN_VALIDATE:            "VALIDATE",
	TOKEN_CONSTRAINT:          "CONSTRAINT",
	TOKEN_INVARIANT:           "INVARIANT",
	TOKEN_COMPUTED:            "COMPUTED",
	TOKEN_SCOPE:               "SCOPE",
	TOKEN_OPERATIONS:          "OPERATIONS",
	TOKEN_PRIMARY:             "PRIMARY",
	TOKEN_AUTO:                "AUTO",
	TOKEN_AUTO_UPDATE:         "AUTO_UPDATE",
	TOKEN_UNIQUE:              "UNIQUE",
	TOKEN_REQUIRED:            "REQUIRED",
	TOKEN_DEFAULT:             "DEFAULT",
	TOKEN_MIN:                 "MIN",
	TOKEN_MAX:                 "MAX",
	TOKEN_PATTERN:             "PATTERN",
	TOKEN_STRICT:              "STRICT",
	TOKEN_IF:                  "IF",
	TOKEN_ELSIF:               "ELSIF",
	TOKEN_ELSE:                "ELSE",
	TOKEN_UNLESS:              "UNLESS",
	TOKEN_RETURN:              "RETURN",
	TOKEN_LET:                 "LET",
	TOKEN_MATCH:               "MATCH",
	TOKEN_WHEN:                "WHEN",
	TOKEN_CONDITION:           "CONDITION",
	TOKEN_ERROR_KW:            "ERROR_KW",
	TOKEN_SELF:                "SELF",
	TOKEN_THROUGH:             "THROUGH",
	TOKEN_AS:                  "AS",
	TOKEN_UNDER:               "UNDER",
	TOKEN_IN:                  "IN",
	TOKEN_NOT_IN:              "NOT_IN",
	TOKEN_OR:                  "OR",
	TOKEN_AND:                 "AND",
	TOKEN_NOT:                 "NOT",
	TOKEN_BY:                  "BY",
	TOKEN_STRING:              "STRING",
	TOKEN_TEXT:                "TEXT",
	TOKEN_MARKDOWN:            "MARKDOWN",
	TOKEN_INT:                 "INT",
	TOKEN_FLOAT:               "FLOAT",
	TOKEN_DECIMAL:             "DECIMAL",
	TOKEN_BOOL:                "BOOL",
	TOKEN_TIMESTAMP:           "TIMESTAMP",
	TOKEN_DATE:                "DATE",
	TOKEN_TIME:                "TIME",
	TOKEN_UUID:                "UUID",
	TOKEN_ULID:                "ULID",
	TOKEN_EMAIL:               "EMAIL",
	TOKEN_URL:                 "URL",
	TOKEN_PHONE:               "PHONE",
	TOKEN_JSON:                "JSON",
	TOKEN_IDENTIFIER:          "IDENTIFIER",
	TOKEN_INT_LITERAL:         "INT_LITERAL",
	TOKEN_FLOAT_LITERAL:       "FLOAT_LITERAL",
	TOKEN_STRING_LITERAL:      "STRING_LITERAL",
	TOKEN_TRUE:                "TRUE",
	TOKEN_FALSE:               "FALSE",
	TOKEN_NULL:                "NULL",
	TOKEN_BANG:                "BANG",
	TOKEN_QUESTION:            "QUESTION",
	TOKEN_AT:                  "AT",
	TOKEN_PIPE:                "PIPE",
	TOKEN_COLON:               "COLON",
	TOKEN_DOT:                 "DOT",
	TOKEN_COMMA:               "COMMA",
	TOKEN_EQUALS:              "EQUALS",
	TOKEN_PLUS:                "PLUS",
	TOKEN_MINUS:               "MINUS",
	TOKEN_STAR:                "STAR",
	TOKEN_SLASH:               "SLASH",
	TOKEN_PERCENT:             "PERCENT",
	TOKEN_LT:                  "LT",
	TOKEN_GT:                  "GT",
	TOKEN_ARROW:               "ARROW",
	TOKEN_EQ:                  "EQ",
	TOKEN_NEQ:                 "NEQ",
	TOKEN_LTE:                 "LTE",
	TOKEN_GTE:                 "GTE",
	TOKEN_DOUBLE_PIPE:         "DOUBLE_PIPE",
	TOKEN_DOUBLE_AMP:          "DOUBLE_AMP",
	TOKEN_DOUBLE_STAR:         "DOUBLE_STAR",
	TOKEN_DOUBLE_COLON:        "DOUBLE_COLON",
	TOKEN_DOUBLE_QUESTION:     "DOUBLE_QUESTION",
	TOKEN_SAFE_NAV:            "SAFE_NAV",
	TOKEN_LBRACE:              "LBRACE",
	TOKEN_RBRACE:              "RBRACE",
	TOKEN_LPAREN:              "LPAREN",
	TOKEN_RPAREN:              "RPAREN",
	TOKEN_LBRACKET:            "LBRACKET",
	TOKEN_RBRACKET:            "RBRACKET",
	TOKEN_HASH_MARK:           "HASH_MARK",
	TOKEN_INTERPOLATION_START: "INTERPOLATION_START",
	TOKEN_INTERPOLATION_END:   "INTERPOLATION_END",
}

// String returns the string representation of a TokenType
func (t TokenType) String() string {
	if name, ok := TokenTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", t)
}

// Token represents a single lexical token in Conduit source code
type Token struct {
	Type    TokenType   // The type of the token
	Lexeme  string      // The raw text of the token
	Literal interface{} // The parsed value (for literals)
	Line    int         // Line number (1-indexed)
	Column  int         // Column number (1-indexed)
}

// String returns a string representation of the token
func (t Token) String() string {
	if t.Literal != nil {
		return fmt.Sprintf("%s '%s' (%v) at %d:%d",
			t.Type.String(), t.Lexeme, t.Literal, t.Line, t.Column)
	}
	return fmt.Sprintf("%s '%s' at %d:%d",
		t.Type.String(), t.Lexeme, t.Line, t.Column)
}

// Keywords maps reserved words to their token types
var Keywords = map[string]TokenType{
	// Resource definition
	"resource": TOKEN_RESOURCE,
	"enum":     TOKEN_ENUM,
	"array":    TOKEN_ARRAY,
	"hash":     TOKEN_HASH,

	// Lifecycle hooks
	"on":          TOKEN_ON,
	"after":       TOKEN_AFTER,
	"before":      TOKEN_BEFORE,
	"transaction": TOKEN_TRANSACTION,
	"async":       TOKEN_ASYNC,
	"rescue":      TOKEN_RESCUE,

	// Query
	"where": TOKEN_WHERE,
	"has":   TOKEN_HAS,

	// Control flow
	"if":        TOKEN_IF,
	"elsif":     TOKEN_ELSIF,
	"else":      TOKEN_ELSE,
	"unless":    TOKEN_UNLESS,
	"return":    TOKEN_RETURN,
	"let":       TOKEN_LET,
	"match":     TOKEN_MATCH,
	"when":      TOKEN_WHEN,
	"condition": TOKEN_CONDITION,
	"error":     TOKEN_ERROR_KW,

	// Special identifiers
	"self":    TOKEN_SELF,
	"through": TOKEN_THROUGH,
	"as":      TOKEN_AS,
	"under":   TOKEN_UNDER,
	"in":      TOKEN_IN,
	"not_in":  TOKEN_NOT_IN,
	"or":      TOKEN_OR,
	"and":     TOKEN_AND,
	"not":     TOKEN_NOT,
	"by":      TOKEN_BY,

	// Primitive types
	"string":    TOKEN_STRING,
	"text":      TOKEN_TEXT,
	"markdown":  TOKEN_MARKDOWN,
	"int":       TOKEN_INT,
	"float":     TOKEN_FLOAT,
	"decimal":   TOKEN_DECIMAL,
	"bool":      TOKEN_BOOL,
	"timestamp": TOKEN_TIMESTAMP,
	"date":      TOKEN_DATE,
	"time":      TOKEN_TIME,
	"uuid":      TOKEN_UUID,
	"ulid":      TOKEN_ULID,
	"email":     TOKEN_EMAIL,
	"url":       TOKEN_URL,
	"phone":     TOKEN_PHONE,
	"json":      TOKEN_JSON,

	// Boolean literals
	"true":  TOKEN_TRUE,
	"false": TOKEN_FALSE,
	"null":  TOKEN_NULL,
	"nil":   TOKEN_NULL,
}

// AnnotationKeywords maps annotation names (without @) to their token types
var AnnotationKeywords = map[string]TokenType{
	// Lifecycle hooks (when used as annotations)
	"on":          TOKEN_ON,
	"before":      TOKEN_BEFORE,
	"after":       TOKEN_AFTER,
	"transaction": TOKEN_TRANSACTION,
	"async":       TOKEN_ASYNC,

	// Resource annotations
	"has_many":   TOKEN_HAS_MANY,
	"nested":     TOKEN_NESTED,
	"middleware": TOKEN_MIDDLEWARE,
	"function":   TOKEN_FUNCTION,
	"validate":   TOKEN_VALIDATE,
	"constraint": TOKEN_CONSTRAINT,
	"invariant":  TOKEN_INVARIANT,
	"computed":   TOKEN_COMPUTED,
	"scope":      TOKEN_SCOPE,
	"operations": TOKEN_OPERATIONS,

	// Field annotations
	"primary":     TOKEN_PRIMARY,
	"auto":        TOKEN_AUTO,
	"auto_update": TOKEN_AUTO_UPDATE,
	"unique":      TOKEN_UNIQUE,
	"required":    TOKEN_REQUIRED,
	"default":     TOKEN_DEFAULT,
	"min":         TOKEN_MIN,
	"max":         TOKEN_MAX,
	"pattern":     TOKEN_PATTERN,
	"strict":      TOKEN_STRICT,
}

// LexError represents an error encountered during lexical analysis
type LexError struct {
	Message string // Error message
	Line    int    // Line number where error occurred
	Column  int    // Column number where error occurred
	Lexeme  string // The problematic text
}

// Error implements the error interface
func (e LexError) Error() string {
	return fmt.Sprintf("Lexical error at %d:%d: %s (near '%s')",
		e.Line, e.Column, e.Message, e.Lexeme)
}
