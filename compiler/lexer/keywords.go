package lexer

// keywords maps keyword strings to their token types for O(1) lookup
var keywords = map[string]TokenType{
	// Resource definition
	"resource": TOKEN_RESOURCE,

	// Lifecycle hooks
	"before": TOKEN_BEFORE,
	"after":  TOKEN_AFTER,
	"on":     TOKEN_ON,

	// Transactions and async
	"transaction": TOKEN_TRANSACTION,
	"async":       TOKEN_ASYNC,
	"rescue":      TOKEN_RESCUE,

	// Relationships
	"has":     TOKEN_HAS,
	"through": TOKEN_THROUGH,
	"as":      TOKEN_AS,
	"under":   TOKEN_UNDER,
	"nested":  TOKEN_NESTED,

	// Annotations
	"function":   TOKEN_FUNCTION,
	"validate":   TOKEN_VALIDATE,
	"constraint": TOKEN_CONSTRAINT,
	"invariant":  TOKEN_INVARIANT,
	"computed":   TOKEN_COMPUTED,
	"scope":      TOKEN_SCOPE,
	"middleware": TOKEN_MIDDLEWARE,
	"operations": TOKEN_OPERATIONS,
	"strict":     TOKEN_STRICT,

	// Control flow
	"if":     TOKEN_IF,
	"elsif":  TOKEN_ELSIF,
	"else":   TOKEN_ELSE,
	"unless": TOKEN_UNLESS,
	"match":  TOKEN_MATCH,
	"when":   TOKEN_WHEN,
	"return": TOKEN_RETURN,
	"let":    TOKEN_LET,
	"error":  TOKEN_ERROR_KW,

	// Operations
	"create": TOKEN_CREATE,
	"update": TOKEN_UPDATE,
	"delete": TOKEN_DELETE,
	"save":   TOKEN_SAVE,
	"list":   TOKEN_LIST,
	"get":    TOKEN_GET,

	// Type keywords - Primitives
	"string":    TOKEN_STRING,
	"text":      TOKEN_TEXT,
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
	"markdown":  TOKEN_MARKDOWN,

	// Type keywords - Structural
	"enum":  TOKEN_ENUM,
	"array": TOKEN_ARRAY,
	"hash":  TOKEN_HASH,

	// Literals
	"true":  TOKEN_TRUE,
	"false": TOKEN_FALSE,
	"nil":   TOKEN_NIL,
	"self":  TOKEN_SELF,

	// Query operations
	"where":    TOKEN_WHERE,
	"in":       TOKEN_IN,
	"not_in":   TOKEN_NOT_IN,
	"order_by": TOKEN_ORDER_BY,
	"limit":    TOKEN_LIMIT,
	"offset":   TOKEN_OFFSET,
	"joins":    TOKEN_JOINS,
	"includes": TOKEN_INCLUDES,

	// Relationship actions
	"restrict":  TOKEN_RESTRICT,
	"cascade":   TOKEN_CASCADE,
	"set_null":  TOKEN_SET_NULL,
	"no_action": TOKEN_NO_ACTION,

	// Field modifiers
	"primary":     TOKEN_PRIMARY,
	"auto":        TOKEN_AUTO,
	"auto_update": TOKEN_AUTO_UPDATE,
	"unique":      TOKEN_UNIQUE,
	"default":     TOKEN_DEFAULT,
	"min":         TOKEN_MIN,
	"max":         TOKEN_MAX,
	"pattern":     TOKEN_PATTERN,
	"required":    TOKEN_REQUIRED,

	// Other keywords
	"condition":   TOKEN_CONDITION,
	"foreign_key": TOKEN_FOREIGN_KEY,
	"on_delete":   TOKEN_ON_DELETE,
	"on_update":   TOKEN_ON_UPDATE,
	"nullability": TOKEN_NULLABILITY,
}

// lookupKeyword checks if an identifier is a keyword
// Returns the token type and true if it's a keyword, TOKEN_IDENTIFIER and false otherwise
func lookupKeyword(identifier string) (TokenType, bool) {
	if tokenType, ok := keywords[identifier]; ok {
		return tokenType, true
	}
	return TOKEN_IDENTIFIER, false
}
