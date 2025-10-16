package errors

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestErrorCodeUniqueness(t *testing.T) {
	// Collect all error codes
	codes := make(map[ErrorCode]string)

	// Syntax errors (SYN001-099)
	syntaxCodes := []ErrorCode{
		ErrUnexpectedToken, ErrExpectedToken, ErrInvalidResourceName,
		ErrInvalidFieldName, ErrMissingNullability, ErrInvalidTypeSpec,
		ErrUnterminatedString, ErrInvalidNumber, ErrInvalidEscape,
		ErrUnexpectedEOF, ErrMismatchedBrace, ErrInvalidAnnotation,
		ErrDuplicateField, ErrInvalidHookTiming, ErrInvalidHookEvent,
		ErrInvalidConstraintName, ErrMissingBlockBody,
	}

	for _, code := range syntaxCodes {
		if prev, exists := codes[code]; exists {
			t.Errorf("Duplicate error code %s (previously used for %s)", code, prev)
		}
		codes[code] = "syntax"
	}

	// Semantic errors (SEM200-299)
	semanticCodes := []ErrorCode{
		ErrUndefinedVariable, ErrUndefinedFunction, ErrUndefinedType,
		ErrUndefinedField, ErrUndefinedResource, ErrCircularDependency,
		ErrRedeclaredVariable, ErrRedeclaredResource, ErrInvalidSelfReference,
		ErrInvalidReturnContext, ErrMissingReturn, ErrInvalidBreakContext,
		ErrInvalidContinueContext, ErrUnreachableCode, ErrInvalidAssignmentTarget,
		ErrConstantReassignment, ErrInvalidHookContext, ErrInvalidAsyncContext,
		ErrInvalidTransactionContext,
	}

	for _, code := range semanticCodes {
		if prev, exists := codes[code]; exists {
			t.Errorf("Duplicate error code %s (previously used for %s)", code, prev)
		}
		codes[code] = "semantic"
	}

	// Type errors (TYP100-199)
	typeCodes := []ErrorCode{
		ErrNullabilityViolation, ErrTypeMismatch, ErrUnnecessaryUnwrap,
		ErrInvalidBinaryOp, ErrInvalidUnaryOp, ErrInvalidIndexOp,
		ErrInvalidArgumentCount, ErrInvalidArgumentType,
		ErrInvalidConstraintType, ErrConstraintTypeMismatch,
	}

	for _, code := range typeCodes {
		if prev, exists := codes[code]; exists {
			t.Errorf("Duplicate error code %s (previously used for %s)", code, prev)
		}
		codes[code] = "type"
	}

	// Relationship errors (REL300-399)
	relationshipCodes := []ErrorCode{
		ErrInvalidRelationshipType, ErrMissingForeignKey, ErrInvalidForeignKey,
		ErrInvalidOnDelete, ErrInvalidThroughTable, ErrSelfReferentialRelationship,
		ErrConflictingRelationships, ErrMissingInverseRelationship,
		ErrInvalidRelationshipNullability, ErrPolymorphicNotSupported,
	}

	for _, code := range relationshipCodes {
		if prev, exists := codes[code]; exists {
			t.Errorf("Duplicate error code %s (previously used for %s)", code, prev)
		}
		codes[code] = "relationship"
	}

	// Pattern warnings (PAT400-499)
	patternCodes := []ErrorCode{
		ErrUnconventionalNaming, ErrMissingDocumentation, ErrUnusedField,
		ErrUnusedVariable, ErrMissingPrimaryKey, ErrMissingTimestamps,
		ErrComplexHook, ErrMagicNumber, ErrDeepNesting, ErrLongFunction,
		ErrInconsistentNullability,
	}

	for _, code := range patternCodes {
		if prev, exists := codes[code]; exists {
			t.Errorf("Duplicate error code %s (previously used for %s)", code, prev)
		}
		codes[code] = "pattern"
	}

	// Validation errors (VAL500-599)
	validationCodes := []ErrorCode{
		ErrInvalidConstraintValue, ErrConflictingConstraints, ErrInvalidPatternRegex,
		ErrInvalidMinMaxRange, ErrInvalidEnumValue, ErrEmptyEnumDefinition,
		ErrDuplicateEnumValue, ErrInvalidDefaultValue, ErrRequiredFieldWithDefault,
		ErrInvalidConstraintCombination,
	}

	for _, code := range validationCodes {
		if prev, exists := codes[code]; exists {
			t.Errorf("Duplicate error code %s (previously used for %s)", code, prev)
		}
		codes[code] = "validation"
	}

	// Code generation errors (GEN600-699)
	codegenCodes := []ErrorCode{
		ErrCodeGenFailed, ErrInvalidGoIdentifier, ErrUnsupportedFeature,
		ErrMigrationConflict, ErrUnsafeMigration, ErrInvalidSQLGeneration,
		ErrTypeConversionFailed, ErrInvalidExpressionContext, ErrGoReservedWord,
	}

	for _, code := range codegenCodes {
		if prev, exists := codes[code]; exists {
			t.Errorf("Duplicate error code %s (previously used for %s)", code, prev)
		}
		codes[code] = "codegen"
	}

	// Optimization hints (OPT700-799)
	optimizationCodes := []ErrorCode{
		ErrMissingIndex, ErrIneffectiveQuery, ErrNPlusOneQuery,
		ErrLargePayload, ErrUnusedEagerLoading, ErrMissingCaching,
		ErrIneffectiveIndex, ErrSlowFunction, ErrMemoryIntensive,
	}

	for _, code := range optimizationCodes {
		if prev, exists := codes[code]; exists {
			t.Errorf("Duplicate error code %s (previously used for %s)", code, prev)
		}
		codes[code] = "optimization"
	}
}

func TestErrorJSONSerialization(t *testing.T) {
	loc := ast.SourceLocation{Line: 10, Column: 5}
	err := NewTypeMismatch(loc, "string!", "int!", "assignment")

	jsonStr, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf("Failed to serialize error to JSON: %v", jsonErr)
	}

	// Parse JSON back
	var parsed CompilerError
	if unmarshalErr := json.Unmarshal([]byte(jsonStr), &parsed); unmarshalErr != nil {
		t.Fatalf("Failed to parse error JSON: %v", unmarshalErr)
	}

	// Verify fields
	if parsed.Code != ErrTypeMismatch {
		t.Errorf("Expected code %s, got %s", ErrTypeMismatch, parsed.Code)
	}
	if parsed.Type != "type_mismatch" {
		t.Errorf("Expected type 'type_mismatch', got '%s'", parsed.Type)
	}
	if parsed.Category != CategoryType {
		t.Errorf("Expected category %s, got %s", CategoryType, parsed.Category)
	}
	if parsed.Severity != SeverityError {
		t.Errorf("Expected severity %s, got %s", SeverityError, parsed.Severity)
	}
	if parsed.Location.Line != 10 {
		t.Errorf("Expected line 10, got %d", parsed.Location.Line)
	}
	if parsed.Location.Column != 5 {
		t.Errorf("Expected column 5, got %d", parsed.Location.Column)
	}
	if parsed.Expected != "string!" {
		t.Errorf("Expected 'string!', got '%s'", parsed.Expected)
	}
	if parsed.Actual != "int!" {
		t.Errorf("Expected 'int!', got '%s'", parsed.Actual)
	}
}

func TestErrorListJSONSerialization(t *testing.T) {
	loc1 := ast.SourceLocation{Line: 5, Column: 10}
	loc2 := ast.SourceLocation{Line: 12, Column: 3}

	errors := ErrorList{
		NewTypeMismatch(loc1, "string!", "int!", "assignment"),
		NewUndefinedVariable(loc2, "foo"),
	}

	jsonStr, jsonErr := errors.ToJSON()
	if jsonErr != nil {
		t.Fatalf("Failed to serialize error list to JSON: %v", jsonErr)
	}

	// Parse JSON back
	var parsed ErrorList
	if unmarshalErr := json.Unmarshal([]byte(jsonStr), &parsed); unmarshalErr != nil {
		t.Fatalf("Failed to parse error list JSON: %v", unmarshalErr)
	}

	if len(parsed) != 2 {
		t.Fatalf("Expected 2 errors, got %d", len(parsed))
	}
}

func TestErrorFormatting(t *testing.T) {
	loc := ast.SourceLocation{Line: 10, Column: 5}
	err := NewNullabilityViolation(loc, "string!", "string?").
		WithFile("post.cdt").
		WithContext("self.title = self.bio", []string{
			"  @before create {",
			"    self.title = self.bio",
			"  }",
		})

	formatted := err.Format()

	// Check for key components
	if !strings.Contains(formatted, "Type Error") {
		t.Error("Formatted error should contain 'Type Error'")
	}
	if !strings.Contains(formatted, "post.cdt") {
		t.Error("Formatted error should contain filename")
	}
	if !strings.Contains(formatted, "Line 10") {
		t.Error("Formatted error should contain line number")
	}
	if !strings.Contains(formatted, "Expected: string!") {
		t.Error("Formatted error should contain expected type")
	}
	if !strings.Contains(formatted, "Actual:   string?") {
		t.Error("Formatted error should contain actual type")
	}
	if !strings.Contains(formatted, "Quick Fixes:") {
		t.Error("Formatted error should contain quick fixes section")
	}
	if !strings.Contains(formatted, "https://docs.conduit-lang.org/errors/TYP101") {
		t.Error("Formatted error should contain documentation URL")
	}
}

func TestErrorListFormatting(t *testing.T) {
	loc1 := ast.SourceLocation{Line: 5, Column: 10}
	loc2 := ast.SourceLocation{Line: 12, Column: 3}

	errors := ErrorList{
		NewTypeMismatch(loc1, "string!", "int!", "assignment"),
		NewUndefinedVariable(loc2, "foo"),
	}

	formatted := errors.Error()

	if !strings.Contains(formatted, "2 error(s)") {
		t.Error("Formatted error list should contain error count")
	}
	if !strings.Contains(formatted, "Type Error") {
		t.Error("Formatted error list should contain first error")
	}
	if !strings.Contains(formatted, "Semantic Error") {
		t.Error("Formatted error list should contain second error")
	}
}

func TestErrorListErrorCount(t *testing.T) {
	errors := ErrorList{
		NewTypeMismatch(ast.SourceLocation{Line: 1, Column: 1}, "string!", "int!", ""),
		NewUnconventionalNaming(ast.SourceLocation{Line: 2, Column: 1}, "Resource", "post", "Use PascalCase"),
		NewMissingIndex(ast.SourceLocation{Line: 3, Column: 1}, "email", "User"),
	}

	errCount, warnCount, infoCount := errors.ErrorCount()

	if errCount != 1 {
		t.Errorf("Expected 1 error, got %d", errCount)
	}
	if warnCount != 1 {
		t.Errorf("Expected 1 warning, got %d", warnCount)
	}
	if infoCount != 1 {
		t.Errorf("Expected 1 info, got %d", infoCount)
	}
}

func TestErrorListHasErrors(t *testing.T) {
	errorsWithError := ErrorList{
		NewTypeMismatch(ast.SourceLocation{Line: 1, Column: 1}, "string!", "int!", ""),
	}

	warningsOnly := ErrorList{
		NewUnconventionalNaming(ast.SourceLocation{Line: 2, Column: 1}, "Resource", "post", "Use PascalCase"),
	}

	if !errorsWithError.HasErrors() {
		t.Error("Expected HasErrors() to return true when list contains errors")
	}

	if warningsOnly.HasErrors() {
		t.Error("Expected HasErrors() to return false when list contains only warnings")
	}
}

func TestErrorCategories(t *testing.T) {
	tests := []struct {
		name     string
		err      *CompilerError
		category ErrorCategory
	}{
		{"Syntax error", NewUnexpectedToken(ast.SourceLocation{Line: 1, Column: 1}, "}", ""), CategorySyntax},
		{"Type error", NewTypeMismatch(ast.SourceLocation{Line: 1, Column: 1}, "string!", "int!", ""), CategoryType},
		{"Semantic error", NewUndefinedVariable(ast.SourceLocation{Line: 1, Column: 1}, "foo"), CategorySemantic},
		{"Relationship error", NewMissingForeignKey(ast.SourceLocation{Line: 1, Column: 1}, "author"), CategoryRelationship},
		{"Pattern warning", NewMissingPrimaryKey(ast.SourceLocation{Line: 1, Column: 1}, "Post"), CategoryPattern},
		{"Validation error", NewInvalidMinMaxRange(ast.SourceLocation{Line: 1, Column: 1}, "10", "5"), CategoryValidation},
		{"Codegen error", NewUnsupportedFeature(ast.SourceLocation{Line: 1, Column: 1}, "polymorphic"), CategoryCodeGen},
		{"Optimization hint", NewMissingIndex(ast.SourceLocation{Line: 1, Column: 1}, "email", "User"), CategoryOptimization},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Category != tt.category {
				t.Errorf("Expected category %s, got %s", tt.category, tt.err.Category)
			}
		})
	}
}

func TestErrorSeverities(t *testing.T) {
	tests := []struct {
		name     string
		err      *CompilerError
		severity ErrorSeverity
	}{
		{"Error severity", NewTypeMismatch(ast.SourceLocation{Line: 1, Column: 1}, "string!", "int!", ""), SeverityError},
		{"Warning severity", NewUnnecessaryUnwrap(ast.SourceLocation{Line: 1, Column: 1}, "string!"), SeverityWarning},
		{"Info severity", NewMissingIndex(ast.SourceLocation{Line: 1, Column: 1}, "email", "User"), SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Severity != tt.severity {
				t.Errorf("Expected severity %s, got %s", tt.severity, tt.err.Severity)
			}
		})
	}
}

func TestWithMethods(t *testing.T) {
	loc := ast.SourceLocation{Line: 5, Column: 10}
	err := NewTypeMismatch(loc, "string!", "int!", "assignment").
		WithFile("test.cdt").
		WithContext("self.name = age", []string{"  self.name = age"}).
		WithSuggestion("Convert int to string").
		WithExamples("String.from(age)", "age.to_string()")

	if err.File != "test.cdt" {
		t.Errorf("Expected file 'test.cdt', got '%s'", err.File)
	}
	if err.Context == nil {
		t.Fatal("Expected context to be set")
	}
	if err.Context.Current != "self.name = age" {
		t.Errorf("Expected current context 'self.name = age', got '%s'", err.Context.Current)
	}
	if err.Suggestion != "Convert int to string" {
		t.Errorf("Expected suggestion 'Convert int to string', got '%s'", err.Suggestion)
	}
	if len(err.Examples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(err.Examples))
	}
}
