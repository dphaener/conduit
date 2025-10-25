package format

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// DiffResult represents the difference between original and formatted code
type DiffResult struct {
	Original  string
	Formatted string
	Changed   bool
}

// Diff compares original and formatted code and returns the difference
func Diff(original, formatted string) *DiffResult {
	return &DiffResult{
		Original:  original,
		Formatted: formatted,
		Changed:   original != formatted,
	}
}

// String returns a human-readable diff with color highlighting
func (d *DiffResult) String() string {
	if !d.Changed {
		return color.GreenString("No changes needed")
	}

	var buf bytes.Buffer

	// Split into lines
	originalLines := strings.Split(d.Original, "\n")
	formattedLines := strings.Split(d.Formatted, "\n")

	// Simple line-by-line diff
	maxLines := len(originalLines)
	if len(formattedLines) > maxLines {
		maxLines = len(formattedLines)
	}

	red := color.New(color.FgRed)
	green := color.New(color.FgGreen)
	cyan := color.New(color.FgCyan)

	for i := 0; i < maxLines; i++ {
		origLine := ""
		if i < len(originalLines) {
			origLine = originalLines[i]
		}

		formLine := ""
		if i < len(formattedLines) {
			formLine = formattedLines[i]
		}

		if origLine != formLine {
			// Show line number
			cyan.Fprintf(&buf, "@@ Line %d @@\n", i+1)

			// Show original line if it exists
			if origLine != "" {
				red.Fprintf(&buf, "- %s\n", origLine)
			}

			// Show formatted line if it exists
			if formLine != "" {
				green.Fprintf(&buf, "+ %s\n", formLine)
			}
		}
	}

	return buf.String()
}

// UnifiedDiff returns a unified diff format string
func (d *DiffResult) UnifiedDiff(filename string) string {
	if !d.Changed {
		return ""
	}

	var buf bytes.Buffer

	// Write header
	fmt.Fprintf(&buf, "--- a/%s\n", filename)
	fmt.Fprintf(&buf, "+++ b/%s\n", filename)

	// Split into lines
	originalLines := strings.Split(d.Original, "\n")
	formattedLines := strings.Split(d.Formatted, "\n")

	maxLines := len(originalLines)
	if len(formattedLines) > maxLines {
		maxLines = len(formattedLines)
	}

	for i := 0; i < maxLines; i++ {
		origLine := ""
		if i < len(originalLines) {
			origLine = originalLines[i]
		}

		formLine := ""
		if i < len(formattedLines) {
			formLine = formattedLines[i]
		}

		if origLine != formLine {
			fmt.Fprintf(&buf, "@@ -%d +%d @@\n", i+1, i+1)
			if origLine != "" {
				fmt.Fprintf(&buf, "-%s\n", origLine)
			}
			if formLine != "" {
				fmt.Fprintf(&buf, "+%s\n", formLine)
			}
		}
	}

	return buf.String()
}

// Stats returns statistics about the changes
func (d *DiffResult) Stats() string {
	if !d.Changed {
		return "No changes"
	}

	originalLines := strings.Split(d.Original, "\n")
	formattedLines := strings.Split(d.Formatted, "\n")

	added := 0
	removed := 0
	changed := 0

	maxLines := len(originalLines)
	if len(formattedLines) > maxLines {
		maxLines = len(formattedLines)
	}

	for i := 0; i < maxLines; i++ {
		origLine := ""
		if i < len(originalLines) {
			origLine = originalLines[i]
		}

		formLine := ""
		if i < len(formattedLines) {
			formLine = formattedLines[i]
		}

		if origLine == "" && formLine != "" {
			added++
		} else if origLine != "" && formLine == "" {
			removed++
		} else if origLine != formLine {
			changed++
		}
	}

	return fmt.Sprintf("%d lines changed, %d added, %d removed", changed, added, removed)
}
