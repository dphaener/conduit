package errors

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Type error codes (TYP100-199) - integrates with existing typechecker errors
const (
	// ErrNullabilityViolation indicates a nullable value was assigned to a required field.
	ErrNullabilityViolation ErrorCode = "TYP101"
	// ErrTypeMismatch indicates a type mismatch between expected and actual types.
	ErrTypeMismatch ErrorCode = "TYP102"
	// ErrUnnecessaryUnwrap indicates an unnecessary unwrap operator on a required type.
	ErrUnnecessaryUnwrap ErrorCode = "TYP103"

	// ErrInvalidBinaryOp indicates an invalid binary operation between types.
	ErrInvalidBinaryOp ErrorCode = "TYP120"
	// ErrInvalidUnaryOp indicates an invalid unary operation on a type.
	ErrInvalidUnaryOp ErrorCode = "TYP121"
	// ErrInvalidIndexOp indicates an invalid index operation on a non-indexable type.
	ErrInvalidIndexOp ErrorCode = "TYP122"

	// ErrInvalidArgumentCount indicates wrong number of arguments in a function call.
	ErrInvalidArgumentCount ErrorCode = "TYP130"
	// ErrInvalidArgumentType indicates wrong type of argument in a function call.
	ErrInvalidArgumentType ErrorCode = "TYP131"

	// ErrInvalidConstraintType indicates a constraint was applied to an incompatible type.
	ErrInvalidConstraintType ErrorCode = "TYP140"
	// ErrConstraintTypeMismatch indicates a constraint argument has the wrong type.
	ErrConstraintTypeMismatch ErrorCode = "TYP141"
)

// NewNullabilityViolation creates a TYP101 error
func NewNullabilityViolation(loc ast.SourceLocation, targetType, sourceType string) *CompilerError {
	return newError(
		ErrNullabilityViolation,
		"nullability_violation",
		CategoryType,
		SeverityError,
		"Cannot assign nullable type to required type without unwrap or coalescing",
		loc,
	).WithExpected(targetType).
		WithActual(sourceType).
		WithSuggestion("Use the unwrap operator (!) or nil coalescing (??)").
		WithExamples(
			"self.field = value!  // Unwrap (panics if nil)",
			"self.field = value ?? default  // Nil coalescing",
		)
}

// NewTypeMismatch creates a TYP102 error
func NewTypeMismatch(loc ast.SourceLocation, expected, actual, context string) *CompilerError {
	message := "Type mismatch"
	if context != "" {
		message = fmt.Sprintf("Type mismatch in %s", context)
	}

	return newError(
		ErrTypeMismatch,
		"type_mismatch",
		CategoryType,
		SeverityError,
		message,
		loc,
	).WithExpected(expected).WithActual(actual)
}

// NewUnnecessaryUnwrap creates a TYP103 warning
func NewUnnecessaryUnwrap(loc ast.SourceLocation, typ string) *CompilerError {
	return newError(
		ErrUnnecessaryUnwrap,
		"unnecessary_unwrap",
		CategoryType,
		SeverityWarning,
		"Unnecessary unwrap operator on required type",
		loc,
	).WithActual(typ).
		WithSuggestion("Remove the ! operator as this type is already required")
}

// NewInvalidBinaryOp creates a TYP120 error
func NewInvalidBinaryOp(loc ast.SourceLocation, op, left, right string) *CompilerError {
	return newError(
		ErrInvalidBinaryOp,
		"invalid_binary_operation",
		CategoryType,
		SeverityError,
		fmt.Sprintf("Binary operator '%s' cannot be applied to types %s and %s", op, left, right),
		loc,
	).WithSuggestion("Ensure both operands are compatible for this operation")
}

// NewInvalidUnaryOp creates a TYP121 error
func NewInvalidUnaryOp(loc ast.SourceLocation, op, operand string) *CompilerError {
	return newError(
		ErrInvalidUnaryOp,
		"invalid_unary_operation",
		CategoryType,
		SeverityError,
		fmt.Sprintf("Unary operator '%s' cannot be applied to type %s", op, operand),
		loc,
	).WithSuggestion("Check operator compatibility with the operand type")
}

// NewInvalidIndexOp creates a TYP122 error
func NewInvalidIndexOp(loc ast.SourceLocation, typ string) *CompilerError {
	return newError(
		ErrInvalidIndexOp,
		"invalid_index_operation",
		CategoryType,
		SeverityError,
		fmt.Sprintf("Type %s is not indexable", typ),
		loc,
	).WithSuggestion("Only arrays and hashes can be indexed")
}

// NewInvalidArgumentCount creates a TYP130 error
func NewInvalidArgumentCount(loc ast.SourceLocation, funcName string, expected, actual int) *CompilerError {
	return newError(
		ErrInvalidArgumentCount,
		"invalid_argument_count",
		CategoryType,
		SeverityError,
		fmt.Sprintf("Function %s expects %d arguments, got %d", funcName, expected, actual),
		loc,
	).WithExpected(fmt.Sprintf("%d arguments", expected)).
		WithActual(fmt.Sprintf("%d arguments", actual))
}

// NewInvalidArgumentType creates a TYP131 error
func NewInvalidArgumentType(loc ast.SourceLocation, funcName string, argIndex int, expected, actual string) *CompilerError {
	return newError(
		ErrInvalidArgumentType,
		"invalid_argument_type",
		CategoryType,
		SeverityError,
		fmt.Sprintf("Function %s: argument %d has wrong type", funcName, argIndex+1),
		loc,
	).WithExpected(expected).WithActual(actual)
}

// NewInvalidConstraintType creates a TYP140 error
func NewInvalidConstraintType(loc ast.SourceLocation, constraint, fieldType, reason string) *CompilerError {
	return newError(
		ErrInvalidConstraintType,
		"invalid_constraint_type",
		CategoryType,
		SeverityError,
		fmt.Sprintf("Constraint @%s cannot be applied to type %s: %s", constraint, fieldType, reason),
		loc,
	).WithSuggestion("Check constraint compatibility with the field type")
}

// NewConstraintTypeMismatch creates a TYP141 error
func NewConstraintTypeMismatch(loc ast.SourceLocation, constraint, expected, actual string) *CompilerError {
	return newError(
		ErrConstraintTypeMismatch,
		"constraint_type_mismatch",
		CategoryType,
		SeverityError,
		fmt.Sprintf("Constraint @%s argument has wrong type", constraint),
		loc,
	).WithExpected(expected).WithActual(actual)
}
