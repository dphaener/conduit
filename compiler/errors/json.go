package errors

import (
	"encoding/json"
)

// JSONOutput represents the JSON structure for error output
type JSONOutput struct {
	Status   string          `json:"status"`
	Errors   []CompilerError `json:"errors"`
	Warnings []CompilerError `json:"warnings"`
	Summary  Summary         `json:"summary"`
}

// Summary contains error and warning counts
type Summary struct {
	ErrorCount   int `json:"error_count"`
	WarningCount int `json:"warning_count"`
	TotalCount   int `json:"total_count"`
}

// FormatAsJSON formats a CompilerError as JSON
func (e CompilerError) FormatAsJSON() (string, error) {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatErrorsAsJSON formats multiple errors as JSON
func FormatErrorsAsJSON(errors []CompilerError) (string, error) {
	// Separate errors and warnings
	var errorList []CompilerError
	var warningList []CompilerError

	for _, err := range errors {
		if err.IsError() {
			errorList = append(errorList, err)
		} else if err.IsWarning() {
			warningList = append(warningList, err)
		}
	}

	// Determine overall status
	status := "success"
	if len(errorList) > 0 {
		status = "error"
	} else if len(warningList) > 0 {
		status = "warning"
	}

	// Build output structure
	output := JSONOutput{
		Status:   status,
		Errors:   errorList,
		Warnings: warningList,
		Summary: Summary{
			ErrorCount:   len(errorList),
			WarningCount: len(warningList),
			TotalCount:   len(errors),
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatAsJSONCompact formats a CompilerError as compact JSON (no indentation)
func (e CompilerError) FormatAsJSONCompact() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatErrorsAsJSONCompact formats multiple errors as compact JSON
func FormatErrorsAsJSONCompact(errors []CompilerError) (string, error) {
	// Separate errors and warnings
	var errorList []CompilerError
	var warningList []CompilerError

	for _, err := range errors {
		if err.IsError() {
			errorList = append(errorList, err)
		} else if err.IsWarning() {
			warningList = append(warningList, err)
		}
	}

	// Determine overall status
	status := "success"
	if len(errorList) > 0 {
		status = "error"
	} else if len(warningList) > 0 {
		status = "warning"
	}

	// Build output structure
	output := JSONOutput{
		Status:   status,
		Errors:   errorList,
		Warnings: warningList,
		Summary: Summary{
			ErrorCount:   len(errorList),
			WarningCount: len(warningList),
			TotalCount:   len(errors),
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(output)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
