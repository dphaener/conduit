package errors

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Code generation error codes (GEN600-699)
const (
	// ErrCodeGenFailed indicates a general code generation failure
	ErrCodeGenFailed ErrorCode = "GEN600"
	// ErrInvalidGoIdentifier indicates a name that can't be converted to valid Go
	ErrInvalidGoIdentifier ErrorCode = "GEN601"
	// ErrUnsupportedFeature indicates a feature not yet implemented in codegen
	ErrUnsupportedFeature ErrorCode = "GEN602"
	// ErrMigrationConflict indicates a migration conflict
	ErrMigrationConflict ErrorCode = "GEN603"
	// ErrUnsafeMigration indicates a potentially destructive migration
	ErrUnsafeMigration ErrorCode = "GEN604"
	// ErrInvalidSQLGeneration indicates invalid SQL generation
	ErrInvalidSQLGeneration ErrorCode = "GEN605"
	// ErrTypeConversionFailed indicates a type conversion failure
	ErrTypeConversionFailed ErrorCode = "GEN606"
	// ErrInvalidExpressionContext indicates an expression in invalid context
	ErrInvalidExpressionContext ErrorCode = "GEN607"
	// ErrGoReservedWord indicates use of Go reserved word
	ErrGoReservedWord ErrorCode = "GEN608"
)

// NewCodeGenFailed creates a GEN600 error
func NewCodeGenFailed(loc ast.SourceLocation, reason string) *CompilerError {
	return newError(
		ErrCodeGenFailed,
		"codegen_failed",
		CategoryCodeGen,
		SeverityError,
		fmt.Sprintf("Code generation failed: %s", reason),
		loc,
	).WithSuggestion("This is likely a compiler bug - please report it")
}

// NewInvalidGoIdentifier creates a GEN601 error
func NewInvalidGoIdentifier(loc ast.SourceLocation, name, reason string) *CompilerError {
	return newError(
		ErrInvalidGoIdentifier,
		"invalid_go_identifier",
		CategoryCodeGen,
		SeverityError,
		fmt.Sprintf("Name '%s' cannot be converted to valid Go identifier: %s", name, reason),
		loc,
	).WithSuggestion("Use alphanumeric characters and underscores only")
}

// NewUnsupportedFeature creates a GEN602 error
func NewUnsupportedFeature(loc ast.SourceLocation, feature string) *CompilerError {
	return newError(
		ErrUnsupportedFeature,
		"unsupported_feature",
		CategoryCodeGen,
		SeverityError,
		fmt.Sprintf("Feature '%s' is not yet supported in code generation", feature),
		loc,
	).WithSuggestion("Check the documentation for supported features or use an alternative approach")
}

// NewMigrationConflict creates a GEN603 error
func NewMigrationConflict(loc ast.SourceLocation, table, conflictType string) *CompilerError {
	return newError(
		ErrMigrationConflict,
		"migration_conflict",
		CategoryCodeGen,
		SeverityError,
		fmt.Sprintf("Migration conflict on table '%s': %s", table, conflictType),
		loc,
	).WithSuggestion("Resolve the conflict manually or create a new migration")
}

// NewUnsafeMigration creates a GEN604 warning
func NewUnsafeMigration(loc ast.SourceLocation, operation, reason string) *CompilerError {
	return newError(
		ErrUnsafeMigration,
		"unsafe_migration",
		CategoryCodeGen,
		SeverityWarning,
		fmt.Sprintf("Potentially destructive migration: %s (%s)", operation, reason),
		loc,
	).WithSuggestion("Review the migration carefully and add data backup before applying")
}

// NewInvalidSQLGeneration creates a GEN605 error
func NewInvalidSQLGeneration(loc ast.SourceLocation, reason string) *CompilerError {
	return newError(
		ErrInvalidSQLGeneration,
		"invalid_sql_generation",
		CategoryCodeGen,
		SeverityError,
		fmt.Sprintf("Invalid SQL generation: %s", reason),
		loc,
	).WithSuggestion("This may indicate an unsupported database feature or schema design")
}

// NewTypeConversionFailed creates a GEN606 error
func NewTypeConversionFailed(loc ast.SourceLocation, fromType, toType, reason string) *CompilerError {
	return newError(
		ErrTypeConversionFailed,
		"type_conversion_failed",
		CategoryCodeGen,
		SeverityError,
		fmt.Sprintf("Cannot convert %s to %s: %s", fromType, toType, reason),
		loc,
	).WithSuggestion("Check type compatibility in the generated code")
}

// NewInvalidExpressionContext creates a GEN607 error
func NewInvalidExpressionContext(loc ast.SourceLocation, expr, context string) *CompilerError {
	return newError(
		ErrInvalidExpressionContext,
		"invalid_expression_context",
		CategoryCodeGen,
		SeverityError,
		fmt.Sprintf("Expression '%s' cannot be used in %s context", expr, context),
		loc,
	).WithSuggestion("Ensure the expression is valid for the current context")
}

// NewGoReservedWord creates a GEN608 error
func NewGoReservedWord(loc ast.SourceLocation, word string) *CompilerError {
	return newError(
		ErrGoReservedWord,
		"go_reserved_word",
		CategoryCodeGen,
		SeverityError,
		fmt.Sprintf("'%s' is a reserved word in Go", word),
		loc,
	).WithSuggestion("Use a different name that doesn't conflict with Go keywords").
		WithExamples(
			"Common Go keywords: type, func, interface, struct, import, package, return, if, else, for, range",
		)
}
