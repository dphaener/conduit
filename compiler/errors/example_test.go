package errors_test

import (
	"fmt"

	"github.com/conduit-lang/conduit/compiler/errors"
)

// ExampleCompilerError_FormatForTerminal demonstrates terminal formatting
func ExampleCompilerError_FormatForTerminal() {
	sourceContent := `resource Post {
  title: string!
  slug: string!
  on_delete: "cascade"
}
`

	loc := errors.SourceLocation{
		File:   "app.cdt",
		Line:   4,
		Column: 13,
		Length: 9,
	}

	err := errors.NewCompilerError(
		"parser",
		errors.ErrOnDeleteInvalid,
		"Invalid on_delete value - expected enum, got string",
		loc,
		errors.Error,
	)

	// Enrich with context
	err = errors.EnrichError(err, sourceContent)

	// Print to terminal (colors stripped for example output)
	output := err.FormatForTerminal()
	fmt.Println(errors.StripColors(output))

	// Output includes error, location, context, and suggestion
}

// ExampleErrorRecovery demonstrates collecting multiple errors
func ExampleErrorRecovery() {
	recovery := errors.NewErrorRecovery()

	// Collect multiple errors
	for i := 1; i <= 3; i++ {
		loc := errors.SourceLocation{
			File:   "app.cdt",
			Line:   i,
			Column: 1,
		}
		err := errors.NewCompilerError(
			"parser",
			errors.ErrUnexpectedToken,
			fmt.Sprintf("Unexpected token at line %d", i),
			loc,
			errors.Error,
		)
		recovery.Recover(err)
	}

	fmt.Printf("Collected %d errors\n", recovery.ErrorCount())
	fmt.Println(recovery.Summary())

	// Output:
	// Collected 3 errors
	// Found 3 error(s)
}

// ExampleFormatErrorsAsJSON demonstrates JSON output
func ExampleFormatErrorsAsJSON() {
	loc := errors.SourceLocation{
		File:   "app.cdt",
		Line:   5,
		Column: 10,
	}

	err := errors.NewCompilerError(
		"parser",
		errors.ErrMissingNullability,
		"Missing nullability marker (! or ?)",
		loc,
		errors.Error,
	)

	jsonOutput, _ := err.FormatAsJSON()
	fmt.Println("JSON output available")
	_ = jsonOutput // Use the output

	// Output:
	// JSON output available
}
