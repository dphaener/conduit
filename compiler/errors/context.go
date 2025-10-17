package errors

import (
	"bufio"
	"os"
	"strings"
)

// EnrichError adds source context and suggestions to an error
func EnrichError(err CompilerError, sourceContent string) CompilerError {
	// Add source context
	err = err.WithContext(extractSourceContext(err.Location, sourceContent))

	// Try to add auto-fix suggestion
	if suggestion := suggestFix(err); suggestion != nil {
		err = err.WithSuggestion(*suggestion)
	}

	return err
}

// extractSourceContext extracts 3 lines before, the error line, and 3 lines after
func extractSourceContext(location SourceLocation, sourceContent string) ErrorContext {
	lines := strings.Split(sourceContent, "\n")

	// Validate line number
	if location.Line < 1 || location.Line > len(lines) {
		return ErrorContext{}
	}

	// Calculate range (3 lines before, error line, 3 lines after)
	errorLineIndex := location.Line - 1 // Convert to 0-based
	startLine := max(0, errorLineIndex-3)
	endLine := min(len(lines), errorLineIndex+4)

	// Extract context lines
	contextLines := make([]string, 0, endLine-startLine)
	for i := startLine; i < endLine; i++ {
		contextLines = append(contextLines, lines[i])
	}

	// Calculate the error line index within the context
	errorLineInContext := errorLineIndex - startLine

	// Calculate highlight position
	start := location.Column - 1 // Convert to 0-based
	end := start + location.Length
	if location.Length == 0 {
		end = start + 1
	}

	return ErrorContext{
		SourceLines: contextLines,
		Highlight: Highlight{
			Line:  errorLineInContext,
			Start: start,
			End:   end,
		},
	}
}

// ReadSourceFile reads a source file and returns its contents
func ReadSourceFile(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// EnrichErrorFromFile reads the source file and enriches the error
func EnrichErrorFromFile(err CompilerError) CompilerError {
	content, readErr := ReadSourceFile(err.Location.File)
	if readErr != nil {
		// If we can't read the file, return the error as-is
		return err
	}

	return EnrichError(err, content)
}

// extractLineFromFile extracts a specific line from a file
func extractLineFromFile(filepath string, lineNum int) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentLine := 1

	for scanner.Scan() {
		if currentLine == lineNum {
			return scanner.Text(), nil
		}
		currentLine++
	}

	return "", scanner.Err()
}

// extractLinesFromFile extracts a range of lines from a file
func extractLinesFromFile(filepath string, startLine, endLine int) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	currentLine := 1

	for scanner.Scan() {
		if currentLine >= startLine && currentLine <= endLine {
			lines = append(lines, scanner.Text())
		}
		if currentLine > endLine {
			break
		}
		currentLine++
	}

	return lines, scanner.Err()
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
