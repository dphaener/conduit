package errors

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Pattern warning codes (PAT400-499) - conventions and best practices
const (
	// ErrUnconventionalNaming indicates naming doesn't follow conventions
	ErrUnconventionalNaming ErrorCode = "PAT400"
	// ErrMissingDocumentation indicates missing documentation comment
	ErrMissingDocumentation ErrorCode = "PAT401"
	// ErrUnusedField indicates a field that is never used
	ErrUnusedField ErrorCode = "PAT402"
	// ErrUnusedVariable indicates a variable that is never used
	ErrUnusedVariable ErrorCode = "PAT403"
	// ErrMissingPrimaryKey indicates a resource without a primary key
	ErrMissingPrimaryKey ErrorCode = "PAT404"
	// ErrMissingTimestamps indicates a resource without created_at/updated_at
	ErrMissingTimestamps ErrorCode = "PAT405"
	// ErrComplexHook indicates a hook with high complexity
	ErrComplexHook ErrorCode = "PAT406"
	// ErrMagicNumber indicates use of magic numbers instead of constants
	ErrMagicNumber ErrorCode = "PAT407"
	// ErrDeepNesting indicates deeply nested code
	ErrDeepNesting ErrorCode = "PAT408"
	// ErrLongFunction indicates a function that is too long
	ErrLongFunction ErrorCode = "PAT409"
	// ErrInconsistentNullability indicates inconsistent nullability patterns
	ErrInconsistentNullability ErrorCode = "PAT410"
)

// NewUnconventionalNaming creates a PAT400 warning
func NewUnconventionalNaming(loc ast.SourceLocation, kind, name, suggestion string) *CompilerError {
	return newError(
		ErrUnconventionalNaming,
		"unconventional_naming",
		CategoryPattern,
		SeverityWarning,
		fmt.Sprintf("%s name '%s' doesn't follow Conduit conventions", kind, name),
		loc,
	).WithSuggestion(suggestion)
}

// NewMissingDocumentation creates a PAT401 info message
func NewMissingDocumentation(loc ast.SourceLocation, kind, name string) *CompilerError {
	return newError(
		ErrMissingDocumentation,
		"missing_documentation",
		CategoryPattern,
		SeverityInfo,
		fmt.Sprintf("%s '%s' is missing documentation", kind, name),
		loc,
	).WithSuggestion("Add a documentation comment starting with ///").
		WithExamples(
			"/// User account with authentication",
			"/// Blog post with title and content",
		)
}

// NewUnusedField creates a PAT402 warning
func NewUnusedField(loc ast.SourceLocation, fieldName, resourceName string) *CompilerError {
	return newError(
		ErrUnusedField,
		"unused_field",
		CategoryPattern,
		SeverityWarning,
		fmt.Sprintf("Field '%s' in resource %s is never used", fieldName, resourceName),
		loc,
	).WithSuggestion("Remove the field or use it in hooks, validations, or computed fields")
}

// NewUnusedVariable creates a PAT403 warning
func NewUnusedVariable(loc ast.SourceLocation, varName string) *CompilerError {
	return newError(
		ErrUnusedVariable,
		"unused_variable",
		CategoryPattern,
		SeverityWarning,
		fmt.Sprintf("Variable '%s' is declared but never used", varName),
		loc,
	).WithSuggestion("Remove the variable or use it in your code")
}

// NewMissingPrimaryKey creates a PAT404 warning
func NewMissingPrimaryKey(loc ast.SourceLocation, resourceName string) *CompilerError {
	return newError(
		ErrMissingPrimaryKey,
		"missing_primary_key",
		CategoryPattern,
		SeverityWarning,
		fmt.Sprintf("Resource '%s' has no primary key", resourceName),
		loc,
	).WithSuggestion("Add a primary key field with @primary annotation").
		WithExamples(
			"id: uuid! @primary @auto",
			"id: int! @primary @auto",
		)
}

// NewMissingTimestamps creates a PAT405 info message
func NewMissingTimestamps(loc ast.SourceLocation, resourceName string) *CompilerError {
	return newError(
		ErrMissingTimestamps,
		"missing_timestamps",
		CategoryPattern,
		SeverityInfo,
		fmt.Sprintf("Resource '%s' has no timestamp fields", resourceName),
		loc,
	).WithSuggestion("Consider adding created_at and updated_at fields").
		WithExamples(
			"created_at: timestamp! @auto",
			"updated_at: timestamp! @auto",
		)
}

// NewComplexHook creates a PAT406 warning
func NewComplexHook(loc ast.SourceLocation, hookName string, complexity int) *CompilerError {
	return newError(
		ErrComplexHook,
		"complex_hook",
		CategoryPattern,
		SeverityWarning,
		fmt.Sprintf("Hook '%s' has high complexity (%d)", hookName, complexity),
		loc,
	).WithSuggestion("Consider extracting logic into separate functions or simplifying the hook")
}

// NewMagicNumber creates a PAT407 warning
func NewMagicNumber(loc ast.SourceLocation, value string) *CompilerError {
	return newError(
		ErrMagicNumber,
		"magic_number",
		CategoryPattern,
		SeverityWarning,
		fmt.Sprintf("Magic number '%s' used without explanation", value),
		loc,
	).WithSuggestion("Define a constant with a descriptive name instead")
}

// NewDeepNesting creates a PAT408 warning
func NewDeepNesting(loc ast.SourceLocation, depth int) *CompilerError {
	return newError(
		ErrDeepNesting,
		"deep_nesting",
		CategoryPattern,
		SeverityWarning,
		fmt.Sprintf("Code nesting depth is %d levels", depth),
		loc,
	).WithSuggestion("Consider refactoring to reduce nesting (max recommended: 3 levels)")
}

// NewLongFunction creates a PAT409 warning
func NewLongFunction(loc ast.SourceLocation, functionName string, lines int) *CompilerError {
	return newError(
		ErrLongFunction,
		"long_function",
		CategoryPattern,
		SeverityWarning,
		fmt.Sprintf("Function '%s' is %d lines long", functionName, lines),
		loc,
	).WithSuggestion("Consider breaking into smaller functions (max recommended: 50 lines)")
}

// NewInconsistentNullability creates a PAT410 warning
func NewInconsistentNullability(loc ast.SourceLocation, field1, field2 string) *CompilerError {
	return newError(
		ErrInconsistentNullability,
		"inconsistent_nullability",
		CategoryPattern,
		SeverityWarning,
		fmt.Sprintf("Fields '%s' and '%s' have inconsistent nullability", field1, field2),
		loc,
	).WithSuggestion("Related fields should have consistent nullability (both required or both optional)")
}
