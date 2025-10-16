package errors

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Syntax error codes (SYN001-099)
const (
	// ErrUnexpectedToken indicates an unexpected token was encountered
	ErrUnexpectedToken ErrorCode = "SYN001"
	// ErrExpectedToken indicates a specific token was expected but not found
	ErrExpectedToken ErrorCode = "SYN002"
	// ErrInvalidResourceName indicates an invalid resource name
	ErrInvalidResourceName ErrorCode = "SYN003"
	// ErrInvalidFieldName indicates an invalid field name
	ErrInvalidFieldName ErrorCode = "SYN004"
	// ErrMissingNullability indicates a type is missing nullability annotation
	ErrMissingNullability ErrorCode = "SYN005"
	// ErrInvalidTypeSpec indicates an invalid type specification
	ErrInvalidTypeSpec ErrorCode = "SYN006"
	// ErrUnterminatedString indicates a string literal was not terminated
	ErrUnterminatedString ErrorCode = "SYN007"
	// ErrInvalidNumber indicates an invalid number literal
	ErrInvalidNumber ErrorCode = "SYN008"
	// ErrInvalidEscape indicates an invalid escape sequence in string
	ErrInvalidEscape ErrorCode = "SYN009"
	// ErrUnexpectedEOF indicates unexpected end of file
	ErrUnexpectedEOF ErrorCode = "SYN010"
	// ErrMismatchedBrace indicates mismatched braces
	ErrMismatchedBrace ErrorCode = "SYN011"
	// ErrInvalidAnnotation indicates an invalid annotation
	ErrInvalidAnnotation ErrorCode = "SYN012"
	// ErrDuplicateField indicates a duplicate field name in resource
	ErrDuplicateField ErrorCode = "SYN013"
	// ErrInvalidHookTiming indicates an invalid hook timing (must be before/after)
	ErrInvalidHookTiming ErrorCode = "SYN014"
	// ErrInvalidHookEvent indicates an invalid hook event
	ErrInvalidHookEvent ErrorCode = "SYN015"
	// ErrInvalidConstraintName indicates an invalid constraint name
	ErrInvalidConstraintName ErrorCode = "SYN016"
	// ErrMissingBlockBody indicates a block is missing its body
	ErrMissingBlockBody ErrorCode = "SYN017"
)

// NewUnexpectedToken creates a SYN001 error
func NewUnexpectedToken(loc ast.SourceLocation, found, context string) *CompilerError {
	message := fmt.Sprintf("Unexpected token '%s'", found)
	if context != "" {
		message = fmt.Sprintf("Unexpected token '%s' in %s", found, context)
	}

	return newError(
		ErrUnexpectedToken,
		"unexpected_token",
		CategorySyntax,
		SeverityError,
		message,
		loc,
	)
}

// NewExpectedToken creates a SYN002 error
func NewExpectedToken(loc ast.SourceLocation, expected, found string) *CompilerError {
	return newError(
		ErrExpectedToken,
		"expected_token",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Expected '%s' but found '%s'", expected, found),
		loc,
	).WithExpected(expected).WithActual(found)
}

// NewInvalidResourceName creates a SYN003 error
func NewInvalidResourceName(loc ast.SourceLocation, name string) *CompilerError {
	return newError(
		ErrInvalidResourceName,
		"invalid_resource_name",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Invalid resource name '%s'", name),
		loc,
	).WithSuggestion("Resource names must start with an uppercase letter and use PascalCase").
		WithExamples("User", "BlogPost", "OrderItem")
}

// NewInvalidFieldName creates a SYN004 error
func NewInvalidFieldName(loc ast.SourceLocation, name string) *CompilerError {
	return newError(
		ErrInvalidFieldName,
		"invalid_field_name",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Invalid field name '%s'", name),
		loc,
	).WithSuggestion("Field names must start with a lowercase letter and use snake_case").
		WithExamples("title", "author_id", "created_at")
}

// NewMissingNullability creates a SYN005 error
func NewMissingNullability(loc ast.SourceLocation, typeName string) *CompilerError {
	return newError(
		ErrMissingNullability,
		"missing_nullability",
		CategorySyntax,
		SeverityError,
		"Type must specify nullability with ! (required) or ? (optional)",
		loc,
	).WithSuggestion("Add ! for required fields or ? for optional fields").
		WithExamples(
			fmt.Sprintf("%s! // Required", typeName),
			fmt.Sprintf("%s? // Optional", typeName),
		)
}

// NewInvalidTypeSpec creates a SYN006 error
func NewInvalidTypeSpec(loc ast.SourceLocation, spec string) *CompilerError {
	return newError(
		ErrInvalidTypeSpec,
		"invalid_type_spec",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Invalid type specification '%s'", spec),
		loc,
	).WithSuggestion("Check type syntax and ensure proper generic parameters").
		WithExamples(
			"string!",
			"array<User>!",
			"hash<string, int>?",
		)
}

// NewUnterminatedString creates a SYN007 error
func NewUnterminatedString(loc ast.SourceLocation) *CompilerError {
	return newError(
		ErrUnterminatedString,
		"unterminated_string",
		CategorySyntax,
		SeverityError,
		"Unterminated string literal",
		loc,
	).WithSuggestion("Add closing quote to string literal")
}

// NewInvalidNumber creates a SYN008 error
func NewInvalidNumber(loc ast.SourceLocation, literal string) *CompilerError {
	return newError(
		ErrInvalidNumber,
		"invalid_number",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Invalid number literal '%s'", literal),
		loc,
	).WithSuggestion("Check number format - use integers (42), decimals (3.14), or scientific notation (1e10)")
}

// NewInvalidEscape creates a SYN009 error
func NewInvalidEscape(loc ast.SourceLocation, escape string) *CompilerError {
	return newError(
		ErrInvalidEscape,
		"invalid_escape",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Invalid escape sequence '\\%s'", escape),
		loc,
	).WithSuggestion("Use valid escape sequences: \\n, \\t, \\r, \\\\, \\\"")
}

// NewUnexpectedEOF creates a SYN010 error
func NewUnexpectedEOF(loc ast.SourceLocation, context string) *CompilerError {
	message := "Unexpected end of file"
	if context != "" {
		message = fmt.Sprintf("Unexpected end of file while parsing %s", context)
	}

	return newError(
		ErrUnexpectedEOF,
		"unexpected_eof",
		CategorySyntax,
		SeverityError,
		message,
		loc,
	).WithSuggestion("Check for missing closing braces or incomplete statements")
}

// NewMismatchedBrace creates a SYN011 error
func NewMismatchedBrace(loc ast.SourceLocation, expected, found string) *CompilerError {
	return newError(
		ErrMismatchedBrace,
		"mismatched_brace",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Mismatched brace: expected '%s' but found '%s'", expected, found),
		loc,
	).WithExpected(expected).WithActual(found)
}

// NewInvalidAnnotation creates a SYN012 error
func NewInvalidAnnotation(loc ast.SourceLocation, annotation string) *CompilerError {
	return newError(
		ErrInvalidAnnotation,
		"invalid_annotation",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Unknown annotation '@%s'", annotation),
		loc,
	).WithSuggestion("Check annotation name - common annotations: @primary, @unique, @min, @max, @before, @after")
}

// NewDuplicateField creates a SYN013 error
func NewDuplicateField(loc ast.SourceLocation, fieldName, resourceName string) *CompilerError {
	return newError(
		ErrDuplicateField,
		"duplicate_field",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Duplicate field '%s' in resource %s", fieldName, resourceName),
		loc,
	).WithSuggestion("Remove the duplicate field or rename it to a unique name")
}

// NewInvalidHookTiming creates a SYN014 error
func NewInvalidHookTiming(loc ast.SourceLocation, timing string) *CompilerError {
	return newError(
		ErrInvalidHookTiming,
		"invalid_hook_timing",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Invalid hook timing '%s'", timing),
		loc,
	).WithSuggestion("Hook timing must be 'before' or 'after'").
		WithExamples("@before create", "@after update")
}

// NewInvalidHookEvent creates a SYN015 error
func NewInvalidHookEvent(loc ast.SourceLocation, event string) *CompilerError {
	return newError(
		ErrInvalidHookEvent,
		"invalid_hook_event",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Invalid hook event '%s'", event),
		loc,
	).WithSuggestion("Valid hook events: create, update, delete, save").
		WithExamples("@before create", "@after update", "@before save")
}

// NewInvalidConstraintName creates a SYN016 error
func NewInvalidConstraintName(loc ast.SourceLocation, name string) *CompilerError {
	return newError(
		ErrInvalidConstraintName,
		"invalid_constraint_name",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("Unknown constraint '@%s'", name),
		loc,
	).WithSuggestion("Valid constraints: @min, @max, @unique, @primary, @auto, @pattern, @email")
}

// NewMissingBlockBody creates a SYN017 error
func NewMissingBlockBody(loc ast.SourceLocation, blockType string) *CompilerError {
	return newError(
		ErrMissingBlockBody,
		"missing_block_body",
		CategorySyntax,
		SeverityError,
		fmt.Sprintf("%s block is missing body", blockType),
		loc,
	).WithSuggestion("Add { } with statements inside the block")
}
