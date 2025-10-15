package typechecker

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TestAllErrorConstructors tests all error constructor functions
func TestAllErrorConstructors(t *testing.T) {
	loc := ast.SourceLocation{Line: 1, Column: 1}

	tests := []struct {
		name string
		err  *TypeError
	}{
		{
			name: "NewNullabilityViolation",
			err: NewNullabilityViolation(
				loc,
				NewPrimitiveType("string", false),
				NewPrimitiveType("string", true),
			),
		},
		{
			name: "NewTypeMismatch",
			err: NewTypeMismatch(
				loc,
				NewPrimitiveType("int", false),
				NewPrimitiveType("string", false),
				"test context",
			),
		},
		{
			name: "NewUnnecessaryUnwrap",
			err:  NewUnnecessaryUnwrap(loc, NewPrimitiveType("string", false)),
		},
		{
			name: "NewUndefinedType",
			err:  NewUndefinedType(loc, "UnknownType"),
		},
		{
			name: "NewUndefinedField",
			err:  NewUndefinedField(loc, "field", "TypeName"),
		},
		{
			name: "NewUndefinedResource",
			err:  NewUndefinedResource(loc, "UnknownResource"),
		},
		{
			name: "NewUndefinedFunction",
			err:  NewUndefinedFunction(loc, "String", "unknown"),
		},
		{
			name: "NewInvalidArgumentCount",
			err:  NewInvalidArgumentCount(loc, "test", 2, 1),
		},
		{
			name: "NewInvalidArgumentType",
			err: NewInvalidArgumentType(
				loc,
				"test",
				0,
				NewPrimitiveType("int", false),
				NewPrimitiveType("string", false),
			),
		},
		{
			name: "NewInvalidConstraintType",
			err: NewInvalidConstraintType(
				loc,
				"min",
				NewPrimitiveType("bool", false),
				"only valid for numeric types",
			),
		},
		{
			name: "NewConstraintTypeMismatch",
			err: NewConstraintTypeMismatch(
				loc,
				"min",
				NewPrimitiveType("int", false),
				NewPrimitiveType("string", false),
			),
		},
		{
			name: "NewInvalidBinaryOp",
			err: NewInvalidBinaryOp(
				loc,
				"+",
				NewPrimitiveType("string", false),
				NewPrimitiveType("int", false),
			),
		},
		{
			name: "NewInvalidUnaryOp",
			err:  NewInvalidUnaryOp(loc, "-", NewPrimitiveType("string", false)),
		},
		{
			name: "NewInvalidIndexOp",
			err:  NewInvalidIndexOp(loc, NewPrimitiveType("string", false)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Error method
			errMsg := tt.err.Error()
			if errMsg == "" {
				t.Error("Error() should return non-empty string")
			}

			// Test Format method
			formatted := tt.err.Format()
			if formatted == "" {
				t.Error("Format() should return non-empty string")
			}

			// Test ToJSON method
			json, err := tt.err.ToJSON()
			if err != nil {
				t.Errorf("ToJSON() returned error: %v", err)
			}
			if json == "" {
				t.Error("ToJSON() should return non-empty JSON")
			}

			// Ensure Error and Format return the same thing
			if errMsg != formatted {
				t.Error("Error() and Format() should return the same string")
			}
		})
	}
}
