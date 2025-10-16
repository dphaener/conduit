package errors

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Validation error codes (VAL500-599)
const (
	// ErrInvalidConstraintValue indicates an invalid constraint argument value
	ErrInvalidConstraintValue ErrorCode = "VAL500"
	// ErrConflictingConstraints indicates conflicting constraint definitions
	ErrConflictingConstraints ErrorCode = "VAL501"
	// ErrInvalidPatternRegex indicates an invalid regex pattern in @pattern constraint
	ErrInvalidPatternRegex ErrorCode = "VAL502"
	// ErrInvalidMinMaxRange indicates min value greater than max value
	ErrInvalidMinMaxRange ErrorCode = "VAL503"
	// ErrInvalidEnumValue indicates an invalid enum value
	ErrInvalidEnumValue ErrorCode = "VAL504"
	// ErrEmptyEnumDefinition indicates an enum with no values
	ErrEmptyEnumDefinition ErrorCode = "VAL505"
	// ErrDuplicateEnumValue indicates duplicate values in enum
	ErrDuplicateEnumValue ErrorCode = "VAL506"
	// ErrInvalidDefaultValue indicates a default value that doesn't match field type
	ErrInvalidDefaultValue ErrorCode = "VAL507"
	// ErrRequiredFieldWithDefault indicates a required field with a default value (redundant)
	ErrRequiredFieldWithDefault ErrorCode = "VAL508"
	// ErrInvalidConstraintCombination indicates constraints that cannot be combined
	ErrInvalidConstraintCombination ErrorCode = "VAL509"
)

// NewInvalidConstraintValue creates a VAL500 error
func NewInvalidConstraintValue(loc ast.SourceLocation, constraint, value, reason string) *CompilerError {
	return newError(
		ErrInvalidConstraintValue,
		"invalid_constraint_value",
		CategoryValidation,
		SeverityError,
		fmt.Sprintf("Invalid value '%s' for constraint @%s: %s", value, constraint, reason),
		loc,
	).WithSuggestion("Check the constraint documentation for valid values")
}

// NewConflictingConstraints creates a VAL501 error
func NewConflictingConstraints(loc ast.SourceLocation, constraint1, constraint2 string) *CompilerError {
	return newError(
		ErrConflictingConstraints,
		"conflicting_constraints",
		CategoryValidation,
		SeverityError,
		fmt.Sprintf("Conflicting constraints: @%s and @%s cannot be used together", constraint1, constraint2),
		loc,
	).WithSuggestion("Remove one of the conflicting constraints")
}

// NewInvalidPatternRegex creates a VAL502 error
func NewInvalidPatternRegex(loc ast.SourceLocation, pattern, reason string) *CompilerError {
	return newError(
		ErrInvalidPatternRegex,
		"invalid_pattern_regex",
		CategoryValidation,
		SeverityError,
		fmt.Sprintf("Invalid regex pattern '%s': %s", pattern, reason),
		loc,
	).WithSuggestion("Ensure the regex pattern is valid Go regex syntax")
}

// NewInvalidMinMaxRange creates a VAL503 error
func NewInvalidMinMaxRange(loc ast.SourceLocation, minVal, maxVal string) *CompilerError {
	return newError(
		ErrInvalidMinMaxRange,
		"invalid_min_max_range",
		CategoryValidation,
		SeverityError,
		fmt.Sprintf("Min value (%s) is greater than max value (%s)", minVal, maxVal),
		loc,
	).WithSuggestion("Ensure @min value is less than or equal to @max value")
}

// NewInvalidEnumValue creates a VAL504 error
func NewInvalidEnumValue(loc ast.SourceLocation, value string, validValues []string) *CompilerError {
	suggestion := "Valid values: " + fmt.Sprint(validValues)
	return newError(
		ErrInvalidEnumValue,
		"invalid_enum_value",
		CategoryValidation,
		SeverityError,
		fmt.Sprintf("'%s' is not a valid enum value", value),
		loc,
	).WithSuggestion(suggestion)
}

// NewEmptyEnumDefinition creates a VAL505 error
func NewEmptyEnumDefinition(loc ast.SourceLocation, fieldName string) *CompilerError {
	return newError(
		ErrEmptyEnumDefinition,
		"empty_enum_definition",
		CategoryValidation,
		SeverityError,
		fmt.Sprintf("Enum field '%s' has no values defined", fieldName),
		loc,
	).WithSuggestion("Define at least one enum value").
		WithExamples(
			"status: enum! { draft, published, archived }",
		)
}

// NewDuplicateEnumValue creates a VAL506 error
func NewDuplicateEnumValue(loc ast.SourceLocation, value string) *CompilerError {
	return newError(
		ErrDuplicateEnumValue,
		"duplicate_enum_value",
		CategoryValidation,
		SeverityError,
		fmt.Sprintf("Duplicate enum value '%s'", value),
		loc,
	).WithSuggestion("Remove the duplicate enum value")
}

// NewInvalidDefaultValue creates a VAL507 error
func NewInvalidDefaultValue(loc ast.SourceLocation, fieldType, defaultValue string) *CompilerError {
	return newError(
		ErrInvalidDefaultValue,
		"invalid_default_value",
		CategoryValidation,
		SeverityError,
		fmt.Sprintf("Default value '%s' does not match field type %s", defaultValue, fieldType),
		loc,
	).WithSuggestion("Ensure the default value matches the field type")
}

// NewRequiredFieldWithDefault creates a VAL508 warning
func NewRequiredFieldWithDefault(loc ast.SourceLocation, fieldName string) *CompilerError {
	return newError(
		ErrRequiredFieldWithDefault,
		"required_field_with_default",
		CategoryValidation,
		SeverityWarning,
		fmt.Sprintf("Required field '%s' has a default value (default is never used)", fieldName),
		loc,
	).WithSuggestion("Remove the default value or make the field optional with ?")
}

// NewInvalidConstraintCombination creates a VAL509 error
func NewInvalidConstraintCombination(loc ast.SourceLocation, constraints []string, reason string) *CompilerError {
	constraintList := ""
	for i, c := range constraints {
		if i > 0 {
			constraintList += ", "
		}
		constraintList += "@" + c
	}

	return newError(
		ErrInvalidConstraintCombination,
		"invalid_constraint_combination",
		CategoryValidation,
		SeverityError,
		fmt.Sprintf("Invalid constraint combination [%s]: %s", constraintList, reason),
		loc,
	).WithSuggestion("Remove incompatible constraints or restructure the field")
}
