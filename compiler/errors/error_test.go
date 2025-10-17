package errors

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestError_Creation tests basic error creation
func TestError_Creation(t *testing.T) {
	loc := SourceLocation{
		File:   "app.cdt",
		Line:   15,
		Column: 7,
		Length: 9,
	}

	err := NewCompilerError("parser", ErrTypeMismatch, "Type mismatch in assignment", loc, Error)

	if err.Phase != "parser" {
		t.Errorf("Expected phase 'parser', got '%s'", err.Phase)
	}
	if err.Code != ErrTypeMismatch {
		t.Errorf("Expected code '%s', got '%s'", ErrTypeMismatch, err.Code)
	}
	if err.Severity != Error {
		t.Errorf("Expected severity Error, got %v", err.Severity)
	}
	if err.Location.Line != 15 {
		t.Errorf("Expected line 15, got %d", err.Location.Line)
	}
}

// TestError_TerminalFormat tests terminal formatting
func TestError_TerminalFormat(t *testing.T) {
	loc := SourceLocation{
		File:   "app.cdt",
		Line:   15,
		Column: 7,
		Length: 9,
	}

	ctx := ErrorContext{
		SourceLines: []string{
			"  author: User! {",
			"    foreign_key: \"author_id\"",
			"    on_delete: \"cascade\"",
			"  }",
		},
		Highlight: Highlight{
			Line:  2,
			Start: 15,
			End:   24,
		},
	}

	suggestion := FixSuggestion{
		Description: "Remove quotes to use the enum value",
		OldCode:     `on_delete: "cascade"`,
		NewCode:     `on_delete: cascade`,
		Confidence:  0.92,
	}

	err := NewCompilerError("parser", ErrOnDeleteInvalid, "Invalid on_delete value", loc, Error)
	err = err.WithContext(ctx).WithSuggestion(suggestion)

	output := err.FormatForTerminal()

	// Check that output contains key elements
	if !strings.Contains(output, "Error") {
		t.Error("Output should contain 'Error'")
	}
	if !strings.Contains(output, "Invalid on_delete value") {
		t.Error("Output should contain error message")
	}
	if !strings.Contains(output, "app.cdt:15:7") {
		t.Error("Output should contain location")
	}
	if !strings.Contains(output, "on_delete") {
		t.Error("Output should contain source context")
	}
	if !strings.Contains(output, "Help") {
		t.Error("Output should contain suggestion")
	}

	// Verify colors are present (before stripping)
	if !strings.Contains(output, "\033[") {
		t.Error("Output should contain ANSI color codes")
	}

	// Strip colors and verify structure
	stripped := StripColors(output)
	if !strings.Contains(stripped, "Error") {
		t.Error("Stripped output should still contain 'Error'")
	}
}

// TestError_JSONFormat tests JSON formatting
func TestError_JSONFormat(t *testing.T) {
	loc := SourceLocation{
		File:   "app.cdt",
		Line:   15,
		Column: 7,
		Length: 9,
	}

	err := NewCompilerError("parser", ErrTypeMismatch, "Type mismatch in assignment", loc, Error)

	jsonStr, jsonErr := err.FormatAsJSON()
	if jsonErr != nil {
		t.Fatalf("Failed to format as JSON: %v", jsonErr)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Debug: Print the JSON structure
	t.Logf("JSON: %s", jsonStr)
	t.Logf("Result keys: %v", result)

	// Verify fields
	if result["phase"] != "parser" {
		t.Errorf("Expected phase 'parser', got '%v'", result["phase"])
	}
	if result["code"] != ErrTypeMismatch {
		t.Errorf("Expected code '%s', got '%v'", ErrTypeMismatch, result["code"])
	}
	if result["severity"] != "error" {
		t.Errorf("Expected severity 'error', got '%v'", result["severity"])
	}

	// Verify location
	location, ok := result["location"].(map[string]interface{})
	if !ok {
		t.Fatalf("location is not a map: %T %v", result["location"], result["location"])
	}
	if location["file"] != "app.cdt" {
		t.Errorf("Expected file 'app.cdt', got '%v'", location["file"])
	}
	if location["line"] != float64(15) {
		t.Errorf("Expected line 15, got %v", location["line"])
	}
}

// TestError_ContextExtraction tests context extraction
func TestError_ContextExtraction(t *testing.T) {
	sourceContent := `resource Post {
  id: uuid! @primary @auto
  title: string! @min(5)
  content: text!
  author: User! {
    foreign_key: "author_id"
    on_delete: cascade
  }
}
`

	loc := SourceLocation{
		File:   "app.cdt",
		Line:   5,
		Column: 10,
		Length: 4,
	}

	ctx := extractSourceContext(loc, sourceContent)

	if len(ctx.SourceLines) == 0 {
		t.Fatal("Expected source lines, got none")
	}

	// Should have up to 7 lines (3 before + error line + 3 after)
	if len(ctx.SourceLines) > 7 {
		t.Errorf("Expected at most 7 lines, got %d", len(ctx.SourceLines))
	}

	// Error line should be in the context
	if ctx.Highlight.Line < 0 || ctx.Highlight.Line >= len(ctx.SourceLines) {
		t.Errorf("Highlight line %d is out of range", ctx.Highlight.Line)
	}

	// Check that the error line contains "author"
	errorLine := ctx.SourceLines[ctx.Highlight.Line]
	if !strings.Contains(errorLine, "author") {
		t.Errorf("Expected error line to contain 'author', got '%s'", errorLine)
	}
}

// TestError_AutoFixSuggestions tests auto-fix suggestions
func TestError_AutoFixSuggestions(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool // whether a suggestion should be generated
	}{
		{"Missing nullability", ErrMissingNullability, true},
		{"Invalid on_delete", ErrOnDeleteInvalid, true},
		{"Undefined function", ErrUndefinedFunction, true},
		{"Expected colon", ErrExpectedColon, true},
		{"Unterminated string", ErrUnterminatedString, true},
		{"Unknown error", "E999", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := SourceLocation{File: "test.cdt", Line: 1, Column: 1}
			err := NewCompilerError("parser", tt.code, "Test error", loc, Error)
			err = err.WithContext(ErrorContext{
				SourceLines: []string{"title: string"},
				Highlight:   Highlight{Line: 0, Start: 0, End: 5},
			})

			suggestion := suggestFix(err)

			if tt.expected && suggestion == nil {
				t.Error("Expected a suggestion but got none")
			}
			if !tt.expected && suggestion != nil {
				t.Error("Expected no suggestion but got one")
			}

			if suggestion != nil {
				if suggestion.Description == "" {
					t.Error("Suggestion should have a description")
				}
				if suggestion.Confidence < 0 || suggestion.Confidence > 1 {
					t.Errorf("Confidence should be 0-1, got %f", suggestion.Confidence)
				}
			}
		})
	}
}

// TestRecovery_CollectsAllErrors tests error recovery
func TestRecovery_CollectsAllErrors(t *testing.T) {
	recovery := NewErrorRecovery()

	// Add multiple errors
	for i := 1; i <= 5; i++ {
		loc := SourceLocation{File: "test.cdt", Line: i, Column: 1}
		err := NewCompilerError("parser", ErrUnexpectedToken, "Unexpected token", loc, Error)
		recovery.Recover(err)
	}

	if recovery.ErrorCount() != 5 {
		t.Errorf("Expected 5 errors, got %d", recovery.ErrorCount())
	}

	if !recovery.HasErrors() {
		t.Error("Expected HasErrors() to be true")
	}
}

// TestRecovery_SummaryCount tests error and warning counts
func TestRecovery_SummaryCount(t *testing.T) {
	recovery := NewErrorRecovery()

	// Add errors
	for i := 1; i <= 3; i++ {
		loc := SourceLocation{File: "test.cdt", Line: i, Column: 1}
		err := NewCompilerError("parser", ErrUnexpectedToken, "Error", loc, Error)
		recovery.Recover(err)
	}

	// Add warnings
	for i := 4; i <= 6; i++ {
		loc := SourceLocation{File: "test.cdt", Line: i, Column: 1}
		warn := NewCompilerError("parser", ErrUnexpectedToken, "Warning", loc, Warning)
		recovery.Recover(warn)
	}

	if recovery.ErrorCount() != 3 {
		t.Errorf("Expected 3 errors, got %d", recovery.ErrorCount())
	}

	if recovery.WarningCount() != 3 {
		t.Errorf("Expected 3 warnings, got %d", recovery.WarningCount())
	}

	if recovery.TotalCount() != 6 {
		t.Errorf("Expected 6 total, got %d", recovery.TotalCount())
	}

	summary := recovery.Summary()
	if !strings.Contains(summary, "3 error(s)") {
		t.Errorf("Summary should mention 3 errors: %s", summary)
	}
	if !strings.Contains(summary, "3 warning(s)") {
		t.Errorf("Summary should mention 3 warnings: %s", summary)
	}
}

// TestRecovery_MaxErrors tests error limit
func TestRecovery_MaxErrors(t *testing.T) {
	recovery := NewErrorRecoveryWithMax(10)

	// Try to add 15 errors
	for i := 1; i <= 15; i++ {
		loc := SourceLocation{File: "test.cdt", Line: i, Column: 1}
		err := NewCompilerError("parser", ErrUnexpectedToken, "Error", loc, Error)
		recovery.Recover(err)
	}

	// Should only have 10
	if recovery.ErrorCount() != 10 {
		t.Errorf("Expected 10 errors (max), got %d", recovery.ErrorCount())
	}
}

// TestRecovery_TerminalFormat tests terminal formatting of multiple errors
func TestRecovery_TerminalFormat(t *testing.T) {
	recovery := NewErrorRecovery()

	// Add a couple of errors
	for i := 1; i <= 2; i++ {
		loc := SourceLocation{File: "test.cdt", Line: i, Column: 1}
		err := NewCompilerError("parser", ErrUnexpectedToken, "Unexpected token", loc, Error)
		recovery.Recover(err)
	}

	output := recovery.FormatForTerminal()

	if !strings.Contains(output, "Error") {
		t.Error("Output should contain 'Error'")
	}
	if !strings.Contains(output, "2 error(s)") {
		t.Error("Output should contain error count")
	}
}

// TestRecovery_JSONFormat tests JSON formatting of multiple errors
func TestRecovery_JSONFormat(t *testing.T) {
	recovery := NewErrorRecovery()

	// Add errors and warnings
	loc1 := SourceLocation{File: "test.cdt", Line: 1, Column: 1}
	err1 := NewCompilerError("parser", ErrUnexpectedToken, "Error 1", loc1, Error)
	recovery.Recover(err1)

	loc2 := SourceLocation{File: "test.cdt", Line: 2, Column: 1}
	warn1 := NewCompilerError("parser", ErrUnexpectedToken, "Warning 1", loc2, Warning)
	recovery.Recover(warn1)

	jsonStr, jsonErr := recovery.FormatAsJSON()
	if jsonErr != nil {
		t.Fatalf("Failed to format as JSON: %v", jsonErr)
	}

	// Verify it's valid JSON
	var result JSONOutput
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if result.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", result.Status)
	}

	if result.Summary.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", result.Summary.ErrorCount)
	}

	if result.Summary.WarningCount != 1 {
		t.Errorf("Expected 1 warning, got %d", result.Summary.WarningCount)
	}
}

// TestErrorHandling_EndToEnd is an integration test with 5 errors
func TestErrorHandling_EndToEnd(t *testing.T) {
	sourceContent := `resource Post {
  id: uuid @primary
  title = "default"
  content: text!
  status enum ["draft", "published"]
  author: User! {
    on_delete: "cascade"
  }
}
`

	recovery := NewErrorRecovery()

	// Error 1: Missing nullability on id
	loc1 := SourceLocation{File: "app.cdt", Line: 2, Column: 6, Length: 4}
	err1 := NewCompilerError("parser", ErrMissingNullability, "Missing nullability marker", loc1, Error)
	err1 = EnrichError(err1, sourceContent)
	recovery.Recover(err1)

	// Error 2: Expected colon instead of equals
	loc2 := SourceLocation{File: "app.cdt", Line: 3, Column: 8, Length: 1}
	err2 := NewCompilerError("parser", ErrExpectedColon, "Expected ':'", loc2, Error)
	err2 = EnrichError(err2, sourceContent)
	recovery.Recover(err2)

	// Error 3: Invalid enum syntax
	loc3 := SourceLocation{File: "app.cdt", Line: 5, Column: 10, Length: 4}
	err3 := NewCompilerError("parser", ErrInvalidEnumValue, "Invalid enum definition", loc3, Error)
	err3 = EnrichError(err3, sourceContent)
	recovery.Recover(err3)

	// Error 4: Invalid on_delete (quoted instead of enum)
	loc4 := SourceLocation{File: "app.cdt", Line: 7, Column: 16, Length: 9}
	err4 := NewCompilerError("parser", ErrOnDeleteInvalid, "Invalid on_delete value", loc4, Error)
	err4 = EnrichError(err4, sourceContent)
	recovery.Recover(err4)

	// Error 5: Undefined resource type (warning)
	loc5 := SourceLocation{File: "app.cdt", Line: 6, Column: 10, Length: 4}
	err5 := NewCompilerError("type_checker", ErrUndefinedResource, "Undefined resource 'User'", loc5, Warning)
	err5 = EnrichError(err5, sourceContent)
	recovery.Recover(err5)

	// Verify counts
	if recovery.ErrorCount() != 4 {
		t.Errorf("Expected 4 errors, got %d", recovery.ErrorCount())
	}

	if recovery.WarningCount() != 1 {
		t.Errorf("Expected 1 warning, got %d", recovery.WarningCount())
	}

	// Verify terminal output
	terminalOutput := recovery.FormatForTerminal()
	if !strings.Contains(terminalOutput, "4 error(s)") {
		t.Error("Terminal output should show 4 errors")
	}
	if !strings.Contains(terminalOutput, "1 warning(s)") {
		t.Error("Terminal output should show 1 warning")
	}

	// Verify JSON output
	jsonOutput, err := recovery.FormatAsJSON()
	if err != nil {
		t.Fatalf("Failed to format as JSON: %v", err)
	}

	var result JSONOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if result.Summary.ErrorCount != 4 {
		t.Errorf("Expected 4 errors in JSON, got %d", result.Summary.ErrorCount)
	}

	if result.Summary.WarningCount != 1 {
		t.Errorf("Expected 1 warning in JSON, got %d", result.Summary.WarningCount)
	}

	// Verify each error has a suggestion
	suggestionsCount := 0
	for _, e := range recovery.GetErrors() {
		if e.Suggestion != nil {
			suggestionsCount++
		}
	}

	if suggestionsCount < 2 {
		t.Errorf("Expected at least 2 errors with suggestions, got %d", suggestionsCount)
	}
}

// TestSeverity tests severity levels
func TestSeverity(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{Info, "info"},
		{Warning, "warning"},
		{Error, "error"},
		{Fatal, "fatal"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.severity.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.severity.String())
			}
		})
	}
}

// TestErrorCodes tests error code constants
func TestErrorCodes(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{ErrUnterminatedString, "E001"},
		{ErrUnexpectedToken, "E100"},
		{ErrTypeMismatch, "E200"},
		{ErrConstraintViolation, "E300"},
		{ErrCodegenFailed, "E400"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.code)
			}

			// Verify message exists
			msg := GetErrorMessage(tt.code)
			if msg == "Unknown error" {
				t.Errorf("No message defined for %s", tt.code)
			}

			// Verify phase
			phase := GetPhaseForCode(tt.code)
			if phase == "unknown" {
				t.Errorf("Could not determine phase for %s", tt.code)
			}
		})
	}
}

// TestGetPhaseForCode tests phase detection from error codes
func TestGetPhaseForCode(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"E001", "lexer"},
		{"E050", "lexer"},
		{"E100", "parser"},
		{"E150", "parser"},
		{"E200", "type_checker"},
		{"E250", "type_checker"},
		{"E300", "constraint"},
		{"E350", "constraint"},
		{"E400", "codegen"},
		{"E450", "codegen"},
		{"E999", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			phase := GetPhaseForCode(tt.code)
			if phase != tt.expected {
				t.Errorf("Expected phase '%s' for code %s, got '%s'", tt.expected, tt.code, phase)
			}
		})
	}
}

// TestStripColors tests ANSI color stripping
func TestStripColors(t *testing.T) {
	input := "\033[31mError\033[0m: \033[1mBold text\033[0m"
	expected := "Error: Bold text"

	result := StripColors(input)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestRelatedErrors tests related error tracking
func TestRelatedErrors(t *testing.T) {
	loc1 := SourceLocation{File: "app.cdt", Line: 1, Column: 1}
	err1 := NewCompilerError("parser", ErrTypeMismatch, "Main error", loc1, Error)

	loc2 := SourceLocation{File: "app.cdt", Line: 2, Column: 1}
	err2 := NewCompilerError("parser", ErrTypeMismatch, "Related error", loc2, Error)

	err1 = err1.WithRelatedError(err2)

	if len(err1.RelatedErrors) != 1 {
		t.Errorf("Expected 1 related error, got %d", len(err1.RelatedErrors))
	}

	if err1.RelatedErrors[0].Message != "Related error" {
		t.Errorf("Related error message mismatch")
	}
}
