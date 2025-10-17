package errors

import (
	"fmt"
	"strings"
)

// suggestFix generates auto-fix suggestions based on error code
func suggestFix(err CompilerError) *FixSuggestion {
	switch err.Code {
	case ErrMissingNullability:
		return suggestNullability(err)
	case ErrExpectedColon:
		return suggestColonInsteadOfEquals(err)
	case ErrUndefinedFunction:
		return suggestNamespacedFunction(err)
	case ErrOnDeleteInvalid:
		return suggestValidOnDelete(err)
	case ErrOnUpdateInvalid:
		return suggestValidOnUpdate(err)
	case ErrTypeMismatch:
		return suggestTypeFix(err)
	case ErrUnexpectedToken:
		return suggestTokenFix(err)
	case ErrExpectedBrace:
		return suggestBrace(err)
	case ErrExpectedParen:
		return suggestParen(err)
	case ErrExpectedBracket:
		return suggestBracket(err)
	case ErrUnterminatedString:
		return suggestCloseString(err)
	case ErrInvalidEscape:
		return suggestValidEscape(err)
	case ErrDuplicateField:
		return suggestRenameDuplicate(err)
	case ErrInvalidEnumValue:
		return suggestValidEnumValue(err)
	case ErrEmptyEnum:
		return suggestEnumValues(err)
	case ErrUniqueViolation:
		return suggestUniqueConstraint(err)
	case ErrMinValueViolation:
		return suggestMinValue(err)
	case ErrMaxValueViolation:
		return suggestMaxValue(err)
	case ErrPatternViolation:
		return suggestPattern(err)
	case ErrInvalidNamespace:
		return suggestValidNamespace(err)
	case ErrNullabilityViolation:
		return suggestNullabilityFix(err)
	default:
		return nil
	}
}

// suggestNullability suggests adding ! or ?
func suggestNullability(err CompilerError) *FixSuggestion {
	if len(err.Context.SourceLines) == 0 {
		return nil
	}

	errorLine := err.Context.SourceLines[err.Context.Highlight.Line]

	// Try to extract the type from the line
	parts := strings.Fields(errorLine)
	if len(parts) < 2 {
		return nil
	}

	// Suggest both options
	return &FixSuggestion{
		Description: "Add nullability marker: '!' for required or '?' for optional",
		OldCode:     strings.TrimSpace(errorLine),
		NewCode:     strings.Replace(errorLine, parts[1], parts[1]+"!", 1) + "\n  OR\n" + strings.Replace(errorLine, parts[1], parts[1]+"?", 1),
		Confidence:  0.85,
	}
}

// suggestColonInsteadOfEquals suggests using : instead of =
func suggestColonInsteadOfEquals(err CompilerError) *FixSuggestion {
	if len(err.Context.SourceLines) == 0 {
		// Provide generic suggestion if no context
		return &FixSuggestion{
			Description: "Use ':' for field definitions, not '='",
			OldCode:     "field = value",
			NewCode:     "field: value",
			Confidence:  0.85,
		}
	}

	errorLine := err.Context.SourceLines[err.Context.Highlight.Line]
	if !strings.Contains(errorLine, "=") {
		return &FixSuggestion{
			Description: "Use ':' for field definitions, not '='",
			OldCode:     "field = value",
			NewCode:     "field: value",
			Confidence:  0.85,
		}
	}

	newLine := strings.Replace(errorLine, "=", ":", 1)

	return &FixSuggestion{
		Description: "Use ':' for field definitions, not '='",
		OldCode:     strings.TrimSpace(errorLine),
		NewCode:     strings.TrimSpace(newLine),
		Confidence:  0.95,
	}
}

// suggestNamespacedFunction suggests using namespaced function
func suggestNamespacedFunction(err CompilerError) *FixSuggestion {
	// Extract function name from error message or context
	msg := strings.ToLower(err.Message)

	// Common namespace mappings
	namespaces := map[string]string{
		"slugify":   "String.slugify",
		"length":    "String.length",
		"uppercase": "String.uppercase",
		"lowercase": "String.lowercase",
		"now":       "Time.now",
		"format":    "Time.format",
		"parse":     "Time.parse",
		"abs":       "Math.abs",
		"round":     "Math.round",
		"floor":     "Math.floor",
		"ceil":      "Math.ceil",
	}

	// Check message for function names
	for unnamespaced, namespaced := range namespaces {
		if strings.Contains(msg, unnamespaced) {
			return &FixSuggestion{
				Description: fmt.Sprintf("Use namespaced function '%s' instead of '%s'", namespaced, unnamespaced),
				OldCode:     unnamespaced + "(...)",
				NewCode:     namespaced + "(...)",
				Confidence:  0.90,
			}
		}
	}

	// Check context lines for function names
	if len(err.Context.SourceLines) > 0 {
		errorLine := strings.ToLower(err.Context.SourceLines[err.Context.Highlight.Line])
		for unnamespaced, namespaced := range namespaces {
			if strings.Contains(errorLine, unnamespaced) {
				return &FixSuggestion{
					Description: fmt.Sprintf("Use namespaced function '%s' instead of '%s'", namespaced, unnamespaced),
					OldCode:     unnamespaced + "(...)",
					NewCode:     namespaced + "(...)",
					Confidence:  0.90,
				}
			}
		}
	}

	// Generic suggestion if no specific function found
	return &FixSuggestion{
		Description: "Use namespaced functions (e.g., String.slugify, Time.now, Math.abs)",
		OldCode:     "function(...)",
		NewCode:     "Namespace.function(...)",
		Confidence:  0.70,
	}
}

// suggestValidOnDelete suggests valid on_delete values
func suggestValidOnDelete(err CompilerError) *FixSuggestion {
	if len(err.Context.SourceLines) == 0 {
		return nil
	}

	errorLine := err.Context.SourceLines[err.Context.Highlight.Line]

	return &FixSuggestion{
		Description: "Remove quotes - on_delete expects an enum value",
		OldCode:     strings.TrimSpace(errorLine),
		NewCode:     strings.Replace(errorLine, `"cascade"`, `cascade`, 1),
		Confidence:  0.92,
	}
}

// suggestValidOnUpdate suggests valid on_update values
func suggestValidOnUpdate(err CompilerError) *FixSuggestion {
	if len(err.Context.SourceLines) == 0 {
		return nil
	}

	errorLine := err.Context.SourceLines[err.Context.Highlight.Line]

	return &FixSuggestion{
		Description: "Remove quotes - on_update expects an enum value",
		OldCode:     strings.TrimSpace(errorLine),
		NewCode:     strings.Replace(errorLine, `"cascade"`, `cascade`, 1),
		Confidence:  0.92,
	}
}

// suggestTypeFix suggests type corrections
func suggestTypeFix(err CompilerError) *FixSuggestion {
	msg := strings.ToLower(err.Message)

	// Common type mismatches
	if strings.Contains(msg, "string") && strings.Contains(msg, "int") {
		return &FixSuggestion{
			Description: "Convert string to integer or change field type",
			OldCode:     "Current type mismatch",
			NewCode:     "Use int! or string! consistently",
			Confidence:  0.70,
		}
	}

	if strings.Contains(msg, "expected") && strings.Contains(msg, "found") {
		return &FixSuggestion{
			Description: "Type mismatch detected - check the expected type",
			OldCode:     "Incorrect type",
			NewCode:     "Match the expected type from the error message",
			Confidence:  0.65,
		}
	}

	return nil
}

// suggestTokenFix suggests fixing unexpected tokens
func suggestTokenFix(err CompilerError) *FixSuggestion {
	if len(err.Context.SourceLines) == 0 {
		return nil
	}

	return &FixSuggestion{
		Description: "Check for missing or extra tokens",
		OldCode:     "",
		NewCode:     "Verify syntax matches the language specification",
		Confidence:  0.50,
	}
}

// suggestBrace suggests missing or extra braces
func suggestBrace(err CompilerError) *FixSuggestion {
	msg := strings.ToLower(err.Message)

	if strings.Contains(msg, "missing") || strings.Contains(msg, "expected") {
		return &FixSuggestion{
			Description: "Add the missing brace",
			OldCode:     "",
			NewCode:     "Add '{' or '}'",
			Confidence:  0.80,
		}
	}

	return nil
}

// suggestParen suggests missing or extra parentheses
func suggestParen(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Check parentheses balance",
		OldCode:     "",
		NewCode:     "Ensure all '(' have matching ')'",
		Confidence:  0.75,
	}
}

// suggestBracket suggests missing or extra brackets
func suggestBracket(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Check brackets balance",
		OldCode:     "",
		NewCode:     "Ensure all '[' have matching ']'",
		Confidence:  0.75,
	}
}

// suggestCloseString suggests closing unterminated string
func suggestCloseString(err CompilerError) *FixSuggestion {
	if len(err.Context.SourceLines) == 0 {
		return nil
	}

	errorLine := err.Context.SourceLines[err.Context.Highlight.Line]

	return &FixSuggestion{
		Description: "Add closing quote",
		OldCode:     strings.TrimSpace(errorLine),
		NewCode:     strings.TrimSpace(errorLine) + `"`,
		Confidence:  0.90,
	}
}

// suggestValidEscape suggests valid escape sequences
func suggestValidEscape(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Use valid escape sequences: \\n, \\t, \\r, \\\\, \\\", \\'",
		OldCode:     "Invalid escape",
		NewCode:     "Use standard escape sequences",
		Confidence:  0.85,
	}
}

// suggestRenameDuplicate suggests renaming duplicate fields
func suggestRenameDuplicate(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Rename the duplicate field to a unique name",
		OldCode:     "Duplicate field name",
		NewCode:     "Use a different field name",
		Confidence:  0.70,
	}
}

// suggestValidEnumValue suggests valid enum values
func suggestValidEnumValue(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Enum values must be valid identifiers (alphanumeric + underscore)",
		OldCode:     "Invalid enum value",
		NewCode:     "Use valid_enum_value format",
		Confidence:  0.80,
	}
}

// suggestEnumValues suggests adding values to empty enum
func suggestEnumValues(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Add at least one value to the enum",
		OldCode:     "enum []",
		NewCode:     "enum [value1, value2, value3]",
		Confidence:  0.85,
	}
}

// suggestUniqueConstraint suggests adding @unique
func suggestUniqueConstraint(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Add @unique constraint to the field",
		OldCode:     "fieldname: type!",
		NewCode:     "fieldname: type! @unique",
		Confidence:  0.80,
	}
}

// suggestMinValue suggests adjusting @min constraint
func suggestMinValue(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Adjust @min constraint or provide larger value",
		OldCode:     "@min(value)",
		NewCode:     "Increase the minimum value or change input",
		Confidence:  0.70,
	}
}

// suggestMaxValue suggests adjusting @max constraint
func suggestMaxValue(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Adjust @max constraint or provide smaller value",
		OldCode:     "@max(value)",
		NewCode:     "Decrease the maximum value or change input",
		Confidence:  0.70,
	}
}

// suggestPattern suggests fixing pattern constraint
func suggestPattern(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Ensure value matches the @pattern constraint",
		OldCode:     "Invalid pattern match",
		NewCode:     "Adjust value to match the regex pattern",
		Confidence:  0.65,
	}
}

// suggestValidNamespace suggests using valid namespace
func suggestValidNamespace(err CompilerError) *FixSuggestion {
	return &FixSuggestion{
		Description: "Use valid namespaces: String, Time, Math, Array, Hash",
		OldCode:     "InvalidNamespace.function()",
		NewCode:     "ValidNamespace.function()",
		Confidence:  0.75,
	}
}

// suggestNullabilityFix suggests fixing nullability issues
func suggestNullabilityFix(err CompilerError) *FixSuggestion {
	msg := strings.ToLower(err.Message)

	if strings.Contains(msg, "required") {
		return &FixSuggestion{
			Description: "Field is required (!) but received null - provide a value or change to optional (?)",
			OldCode:     "fieldname: type!",
			NewCode:     "fieldname: type?  // or provide a non-null value",
			Confidence:  0.80,
		}
	}

	return &FixSuggestion{
		Description: "Check nullability markers (! vs ?) match your intent",
		OldCode:     "",
		NewCode:     "Ensure ! (required) and ? (optional) are correct",
		Confidence:  0.65,
	}
}
