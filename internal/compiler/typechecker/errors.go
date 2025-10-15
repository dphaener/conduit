package typechecker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// ErrorCode represents a specific type error code
type ErrorCode string

const (
	// ErrNullabilityViolation indicates a nullable value was assigned to a required field.
	ErrNullabilityViolation ErrorCode = "TYP101"
	// ErrTypeMismatch indicates a type mismatch between expected and actual types.
	ErrTypeMismatch ErrorCode = "TYP102"
	// ErrUnnecessaryUnwrap indicates an unnecessary unwrap operator on a required type.
	ErrUnnecessaryUnwrap ErrorCode = "TYP103"

	// ErrUndefinedType indicates an undefined or unknown type was referenced.
	ErrUndefinedType ErrorCode = "TYP200"
	// ErrUndefinedField indicates an undefined field was accessed.
	ErrUndefinedField ErrorCode = "TYP201"
	// ErrUndefinedResource indicates an undefined resource was referenced.
	ErrUndefinedResource ErrorCode = "TYP202"

	// ErrUndefinedFunction indicates an undefined function was called.
	ErrUndefinedFunction ErrorCode = "TYP300"
	// ErrInvalidArgumentCount indicates wrong number of arguments in a function call.
	ErrInvalidArgumentCount ErrorCode = "TYP301"
	// ErrInvalidArgumentType indicates wrong type of argument in a function call.
	ErrInvalidArgumentType ErrorCode = "TYP302"

	// ErrInvalidConstraintType indicates a constraint was applied to an incompatible type.
	ErrInvalidConstraintType ErrorCode = "TYP400"
	// ErrConstraintTypeMismatch indicates a constraint argument has the wrong type.
	ErrConstraintTypeMismatch ErrorCode = "TYP401"

	// ErrInvalidBinaryOp indicates an invalid binary operation between types.
	ErrInvalidBinaryOp ErrorCode = "TYP500"
	// ErrInvalidUnaryOp indicates an invalid unary operation on a type.
	ErrInvalidUnaryOp ErrorCode = "TYP501"
	// ErrInvalidIndexOp indicates an invalid index operation on a non-indexable type.
	ErrInvalidIndexOp ErrorCode = "TYP502"
)

// ErrorSeverity indicates the severity level of a type error
type ErrorSeverity string

const (
	// SeverityError indicates a type error that prevents compilation.
	SeverityError ErrorSeverity = "error"
	// SeverityWarning indicates a type warning that suggests potential issues.
	SeverityWarning ErrorSeverity = "warning"
)

// TypeError represents a type checking error with comprehensive information
// for both human-readable output and LLM consumption
type TypeError struct {
	Code       ErrorCode          `json:"code"`
	Type       string             `json:"type"`
	Severity   ErrorSeverity      `json:"severity"`
	Message    string             `json:"message"`
	Location   ast.SourceLocation `json:"location"`
	Expected   string             `json:"expected,omitempty"`
	Actual     string             `json:"actual,omitempty"`
	Suggestion string             `json:"suggestion,omitempty"`
	Examples   []string           `json:"examples,omitempty"`
}

// Error implements the error interface
func (e *TypeError) Error() string {
	return e.Format()
}

// Format returns a human-readable error message for terminal output
func (e *TypeError) Format() string {
	var b strings.Builder

	// Error header with location
	fmt.Fprintf(&b, "%s:%d:%d: %s [%s]\n",
		"<source>", e.Location.Line, e.Location.Column,
		strings.ToUpper(string(e.Severity)), e.Code)

	// Main message
	fmt.Fprintf(&b, "  %s\n", e.Message)

	// Expected vs Actual (if provided)
	if e.Expected != "" || e.Actual != "" {
		fmt.Fprintf(&b, "\n")
		if e.Expected != "" {
			fmt.Fprintf(&b, "  Expected: %s\n", e.Expected)
		}
		if e.Actual != "" {
			fmt.Fprintf(&b, "  Actual:   %s\n", e.Actual)
		}
	}

	// Suggestion (if provided)
	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n  Suggestion: %s\n", e.Suggestion)
	}

	// Examples (if provided)
	if len(e.Examples) > 0 {
		fmt.Fprintf(&b, "\n  Examples:\n")
		for _, example := range e.Examples {
			fmt.Fprintf(&b, "    %s\n", example)
		}
	}

	return b.String()
}

// ToJSON returns the error as a JSON string for LLM consumption
func (e *TypeError) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ErrorList is a collection of type errors
type ErrorList []*TypeError

// Error implements the error interface
func (el ErrorList) Error() string {
	if len(el) == 0 {
		return "no errors"
	}
	var b strings.Builder
	for i, err := range el {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(err.Format())
	}
	return b.String()
}

// HasErrors returns true if the list contains any errors
func (el ErrorList) HasErrors() bool {
	return len(el) > 0
}

// ToJSON returns all errors as a JSON array
func (el ErrorList) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(el, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// NewNullabilityViolation creates a TYP101 error
func NewNullabilityViolation(loc ast.SourceLocation, targetType, sourceType Type) *TypeError {
	return &TypeError{
		Code:       ErrNullabilityViolation,
		Type:       "nullability_violation",
		Severity:   SeverityError,
		Message:    "Cannot assign nullable type to required type without unwrap or coalescing",
		Location:   loc,
		Expected:   targetType.String(),
		Actual:     sourceType.String(),
		Suggestion: "Use the unwrap operator (!) or nil coalescing (??)",
		Examples: []string{
			"self.field = value!  // Unwrap (panics if nil)",
			"self.field = value ?? default  // Nil coalescing",
		},
	}
}

// NewTypeMismatch creates a TYP102 error
func NewTypeMismatch(loc ast.SourceLocation, expected, actual Type, context string) *TypeError {
	message := "Type mismatch"
	if context != "" {
		message = fmt.Sprintf("Type mismatch in %s", context)
	}

	return &TypeError{
		Code:     ErrTypeMismatch,
		Type:     "type_mismatch",
		Severity: SeverityError,
		Message:  message,
		Location: loc,
		Expected: expected.String(),
		Actual:   actual.String(),
	}
}

// NewUnnecessaryUnwrap creates a TYP103 warning
func NewUnnecessaryUnwrap(loc ast.SourceLocation, typ Type) *TypeError {
	return &TypeError{
		Code:       ErrUnnecessaryUnwrap,
		Type:       "unnecessary_unwrap",
		Severity:   SeverityWarning,
		Message:    "Unnecessary unwrap operator on required type",
		Location:   loc,
		Actual:     typ.String(),
		Suggestion: "Remove the ! operator as this type is already required",
	}
}

// NewUndefinedType creates a TYP200 error
func NewUndefinedType(loc ast.SourceLocation, typeName string) *TypeError {
	return &TypeError{
		Code:       ErrUndefinedType,
		Type:       "undefined_type",
		Severity:   SeverityError,
		Message:    fmt.Sprintf("Undefined type: %s", typeName),
		Location:   loc,
		Suggestion: "Check for typos or ensure the resource/type is defined",
	}
}

// NewUndefinedField creates a TYP201 error
func NewUndefinedField(loc ast.SourceLocation, fieldName, typeName string) *TypeError {
	return &TypeError{
		Code:     ErrUndefinedField,
		Type:     "undefined_field",
		Severity: SeverityError,
		Message:  fmt.Sprintf("Type %s has no field named '%s'", typeName, fieldName),
		Location: loc,
	}
}

// NewUndefinedResource creates a TYP202 error
func NewUndefinedResource(loc ast.SourceLocation, resourceName string) *TypeError {
	return &TypeError{
		Code:       ErrUndefinedResource,
		Type:       "undefined_resource",
		Severity:   SeverityError,
		Message:    fmt.Sprintf("Undefined resource: %s", resourceName),
		Location:   loc,
		Suggestion: "Ensure the resource is defined in a .cdt file",
	}
}

// NewUndefinedFunction creates a TYP300 error
func NewUndefinedFunction(loc ast.SourceLocation, namespace, function string) *TypeError {
	var funcName string
	if namespace != "" {
		funcName = namespace + "." + function
	} else {
		funcName = function
	}

	suggestion := "Check the function name and namespace"
	if namespace == "" {
		suggestion = "Custom functions must be defined with @function. Use namespaced stdlib functions (e.g., String.slugify())"
	}

	return &TypeError{
		Code:       ErrUndefinedFunction,
		Type:       "undefined_function",
		Severity:   SeverityError,
		Message:    fmt.Sprintf("Undefined function: %s", funcName),
		Location:   loc,
		Suggestion: suggestion,
	}
}

// NewInvalidArgumentCount creates a TYP301 error
func NewInvalidArgumentCount(loc ast.SourceLocation, funcName string, expected, actual int) *TypeError {
	return &TypeError{
		Code:     ErrInvalidArgumentCount,
		Type:     "invalid_argument_count",
		Severity: SeverityError,
		Message:  fmt.Sprintf("Function %s expects %d arguments, got %d", funcName, expected, actual),
		Location: loc,
		Expected: fmt.Sprintf("%d arguments", expected),
		Actual:   fmt.Sprintf("%d arguments", actual),
	}
}

// NewInvalidArgumentType creates a TYP302 error
func NewInvalidArgumentType(loc ast.SourceLocation, funcName string, argIndex int, expected, actual Type) *TypeError {
	return &TypeError{
		Code:     ErrInvalidArgumentType,
		Type:     "invalid_argument_type",
		Severity: SeverityError,
		Message:  fmt.Sprintf("Function %s: argument %d has wrong type", funcName, argIndex+1),
		Location: loc,
		Expected: expected.String(),
		Actual:   actual.String(),
	}
}

// NewInvalidConstraintType creates a TYP400 error
func NewInvalidConstraintType(loc ast.SourceLocation, constraint string, fieldType Type, reason string) *TypeError {
	return &TypeError{
		Code:     ErrInvalidConstraintType,
		Type:     "invalid_constraint_type",
		Severity: SeverityError,
		Message:  fmt.Sprintf("Constraint @%s cannot be applied to type %s: %s", constraint, fieldType.String(), reason),
		Location: loc,
	}
}

// NewConstraintTypeMismatch creates a TYP401 error
func NewConstraintTypeMismatch(loc ast.SourceLocation, constraint string, expected, actual Type) *TypeError {
	return &TypeError{
		Code:     ErrConstraintTypeMismatch,
		Type:     "constraint_type_mismatch",
		Severity: SeverityError,
		Message:  fmt.Sprintf("Constraint @%s argument has wrong type", constraint),
		Location: loc,
		Expected: expected.String(),
		Actual:   actual.String(),
	}
}

// NewInvalidBinaryOp creates a TYP500 error
func NewInvalidBinaryOp(loc ast.SourceLocation, op string, left, right Type) *TypeError {
	return &TypeError{
		Code:     ErrInvalidBinaryOp,
		Type:     "invalid_binary_operation",
		Severity: SeverityError,
		Message:  fmt.Sprintf("Binary operator '%s' cannot be applied to types %s and %s", op, left.String(), right.String()),
		Location: loc,
	}
}

// NewInvalidUnaryOp creates a TYP501 error
func NewInvalidUnaryOp(loc ast.SourceLocation, op string, operand Type) *TypeError {
	return &TypeError{
		Code:     ErrInvalidUnaryOp,
		Type:     "invalid_unary_operation",
		Severity: SeverityError,
		Message:  fmt.Sprintf("Unary operator '%s' cannot be applied to type %s", op, operand.String()),
		Location: loc,
	}
}

// NewInvalidIndexOp creates a TYP502 error
func NewInvalidIndexOp(loc ast.SourceLocation, typ Type) *TypeError {
	return &TypeError{
		Code:       ErrInvalidIndexOp,
		Type:       "invalid_index_operation",
		Severity:   SeverityError,
		Message:    fmt.Sprintf("Type %s is not indexable", typ.String()),
		Location:   loc,
		Suggestion: "Only arrays and hashes can be indexed",
	}
}
