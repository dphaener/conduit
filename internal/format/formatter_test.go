package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatterBasicResource(t *testing.T) {
	input := `resource User {
id: uuid!  @primary @auto
name:string!
email: string! @unique
}`

	expected := `resource User {
  id   : uuid! @primary @auto
  name : string!
  email: string! @unique
}
`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if result != expected {
		t.Errorf("Format mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatterResourceWithDocumentation(t *testing.T) {
	// TODO: Documentation comments are not yet supported by the parser
	// Skip this test for now
	t.Skip("Documentation comments not yet supported by parser")
}

func TestFormatterRelationships(t *testing.T) {
	input := `resource Post {
id: uuid! @primary @auto
title: string!
author: User! {
foreign_key: "author_id"
on_delete: restrict
}
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Check that relationship metadata is formatted correctly
	if !strings.Contains(result, "author: User!") {
		t.Errorf("Relationship not formatted correctly")
	}
	if !strings.Contains(result, "foreign_key:") {
		t.Errorf("Foreign key metadata not preserved")
	}
	if !strings.Contains(result, "on_delete:") {
		t.Errorf("On delete metadata not preserved")
	}
}

func TestFormatterComplexTypes(t *testing.T) {
	input := `resource Config {
id: uuid! @primary @auto
tags: array<string>!
metadata: hash<string,string>!
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "array<string>!") {
		t.Errorf("Array type not formatted correctly")
	}
	if !strings.Contains(result, "hash<string, string>!") {
		t.Errorf("Hash type not formatted correctly")
	}
}

func TestFormatterMultipleResources(t *testing.T) {
	input := `resource User {
id: uuid! @primary @auto
}
resource Post {
id: uuid! @primary @auto
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Check that resources are separated by blank lines
	lines := strings.Split(result, "\n")
	blankLineCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankLineCount++
		}
	}

	if blankLineCount == 0 {
		t.Errorf("Resources should be separated by blank lines")
	}
}

func TestFormatterIndentation(t *testing.T) {
	tests := []struct {
		name       string
		indentSize int
		input      string
		wantIndent string
	}{
		{
			name:       "2 spaces",
			indentSize: 2,
			input:      "resource User {\nid: uuid!\n}",
			wantIndent: "  ",
		},
		{
			name:       "4 spaces",
			indentSize: 4,
			input:      "resource User {\nid: uuid!\n}",
			wantIndent: "    ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				IndentSize:  tt.indentSize,
				AlignFields: false,
			}
			formatter := New(config)
			result, err := formatter.Format(tt.input)

			if err != nil {
				t.Fatalf("Formatting failed: %v", err)
			}

			if !strings.Contains(result, tt.wantIndent+"id:") {
				t.Errorf("Expected indent '%s' not found in result:\n%s", tt.wantIndent, result)
			}
		})
	}
}

func TestFormatterAlignFields(t *testing.T) {
	input := `resource User {
id: uuid!
name: string!
email: string!
}`

	config := &Config{
		IndentSize:  2,
		AlignFields: true,
	}
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// With alignment, field names should be padded to align types
	lines := strings.Split(result, "\n")
	colonPositions := []int{}
	for _, line := range lines {
		if idx := strings.Index(line, ":"); idx > 0 {
			colonPositions = append(colonPositions, idx)
		}
	}

	// All colons should be at the same position
	if len(colonPositions) > 1 {
		firstPos := colonPositions[0]
		for _, pos := range colonPositions[1:] {
			if pos != firstPos {
				t.Errorf("Fields not aligned: colon positions %v", colonPositions)
				break
			}
		}
	}
}

func TestFormatterNoAlignFields(t *testing.T) {
	input := `resource User {
id: uuid!
name: string!
}`

	config := &Config{
		IndentSize:  2,
		AlignFields: false,
	}
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Without alignment, fields should not have extra padding
	if strings.Contains(result, "id:    uuid") {
		t.Errorf("Fields should not be aligned when AlignFields is false")
	}
}

func TestFormatterDeterministic(t *testing.T) {
	input := `resource User {
id: uuid! @primary @auto
name: string!
email: string! @unique
}`

	config := DefaultConfig()
	formatter := New(config)

	// Format multiple times
	result1, err1 := formatter.Format(input)
	if err1 != nil {
		t.Fatalf("First formatting failed: %v", err1)
	}

	result2, err2 := formatter.Format(result1)
	if err2 != nil {
		t.Fatalf("Second formatting failed: %v", err2)
	}

	result3, err3 := formatter.Format(result2)
	if err3 != nil {
		t.Fatalf("Third formatting failed: %v", err3)
	}

	// All results should be identical
	if result1 != result2 {
		t.Errorf("First and second format results differ")
	}
	if result2 != result3 {
		t.Errorf("Second and third format results differ")
	}
}

func TestFormatterInvalidSyntax(t *testing.T) {
	input := `resource User {
id: uuid! @primary
name: invalid_syntax here
}`

	config := DefaultConfig()
	formatter := New(config)
	_, err := formatter.Format(input)

	if err == nil {
		t.Errorf("Expected error for invalid syntax, got nil")
	}
}

func TestFormatterEmptyResource(t *testing.T) {
	input := `resource User {
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "resource User") {
		t.Errorf("Empty resource not formatted correctly")
	}
}

func TestFormatterConstraintArguments(t *testing.T) {
	input := `resource User {
age: int! @min(18) @max(120)
email: string! @pattern("^[a-z]+@[a-z]+\\.[a-z]+$")
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "@min(18)") {
		t.Errorf("Min constraint not preserved")
	}
	if !strings.Contains(result, "@max(120)") {
		t.Errorf("Max constraint not preserved")
	}
	if !strings.Contains(result, "@pattern(") {
		t.Errorf("Pattern constraint not preserved")
	}
}

func TestFormatterNullable(t *testing.T) {
	input := `resource User {
name: string!
bio: text?
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "string!") {
		t.Errorf("Required field marker (!) not preserved")
	}
	if !strings.Contains(result, "text?") {
		t.Errorf("Optional field marker (?) not preserved")
	}
}

func TestDiff(t *testing.T) {
	original := "line1\nline2\nline3"
	formatted := "line1\nmodified\nline3"

	diff := Diff(original, formatted)

	if !diff.Changed {
		t.Errorf("Expected diff to detect changes")
	}

	diffStr := diff.String()
	if !strings.Contains(diffStr, "line2") {
		t.Errorf("Diff should show removed line")
	}
	if !strings.Contains(diffStr, "modified") {
		t.Errorf("Diff should show added line")
	}
}

func TestDiffNoChanges(t *testing.T) {
	original := "line1\nline2\nline3"
	formatted := "line1\nline2\nline3"

	diff := Diff(original, formatted)

	if diff.Changed {
		t.Errorf("Expected no changes")
	}

	diffStr := diff.String()
	if !strings.Contains(diffStr, "No changes") {
		t.Errorf("Diff should indicate no changes")
	}
}

func TestConfigLoadDefault(t *testing.T) {
	config, err := LoadConfig("nonexistent.yml")
	if err != nil {
		t.Fatalf("Loading nonexistent config should return default, got error: %v", err)
	}

	if config.IndentSize != 2 {
		t.Errorf("Expected default indent size 2, got %d", config.IndentSize)
	}
	if !config.AlignFields {
		t.Errorf("Expected default align fields true")
	}
}

func TestFormatterEnumTypes(t *testing.T) {
	// TODO: Enum syntax not yet fully supported by parser
	t.Skip("Enum syntax not yet fully supported")
}

func TestFormatterNewCreatesFormatter(t *testing.T) {
	formatter := New(nil)
	if formatter == nil {
		t.Errorf("New() should create a formatter even with nil config")
	}
	if formatter.config == nil {
		t.Errorf("Formatter should have default config when created with nil")
	}
}

func TestFormatterFormatFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.cdt")

	input := `resource User {
id: uuid!
name: string!
}`

	err := os.WriteFile(filePath, []byte(input), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Format file
	result, err := FormatFile(filePath, DefaultConfig())
	if err != nil {
		t.Fatalf("FormatFile failed: %v", err)
	}

	if !strings.Contains(result, "resource User") {
		t.Errorf("FormatFile should preserve resource content")
	}
}

func TestFormatterFormatFileNotFound(t *testing.T) {
	_, err := FormatFile("/nonexistent/file.cdt", DefaultConfig())
	if err == nil {
		t.Errorf("FormatFile should return error for nonexistent file")
	}
}

func TestDiffStats(t *testing.T) {
	original := "line1\nline2\nline3"
	formatted := "line1\nmodified\nline3\nline4"

	diff := Diff(original, formatted)
	stats := diff.Stats()

	if !strings.Contains(stats, "changed") {
		t.Errorf("Stats should mention changed lines")
	}
	if !strings.Contains(stats, "added") {
		t.Errorf("Stats should mention added lines")
	}
}

func TestDiffUnifiedDiff(t *testing.T) {
	original := "line1\nline2"
	formatted := "line1\nmodified"

	diff := Diff(original, formatted)
	unified := diff.UnifiedDiff("test.cdt")

	if !strings.Contains(unified, "---") {
		t.Errorf("Unified diff should have header")
	}
	if !strings.Contains(unified, "+++") {
		t.Errorf("Unified diff should have header")
	}
}

func TestDiffUnifiedDiffNoChanges(t *testing.T) {
	original := "line1\nline2"
	formatted := "line1\nline2"

	diff := Diff(original, formatted)
	unified := diff.UnifiedDiff("test.cdt")

	if unified != "" {
		t.Errorf("Unified diff should be empty when no changes")
	}
}

func TestFormatterResourceType(t *testing.T) {
	input := `resource Comment {
id: uuid! @primary @auto
author: User!
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "author: User!") {
		t.Errorf("Resource type not formatted correctly")
	}
}

func TestFormatterWithOnUpdate(t *testing.T) {
	input := `resource Post {
id: uuid! @primary @auto
author: User! {
foreign_key: "author_id"
on_delete: restrict
on_update: cascade
}
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "on_update:") {
		t.Errorf("on_update not preserved in relationship metadata")
	}
}

func TestFormatterConstraintBooleanArg(t *testing.T) {
	// Test would require a constraint that accepts boolean
	// Skip for now as not clear which constraints use booleans
	t.Skip("Boolean constraint arguments not tested")
}

func TestFormatterConstraintFloatArg(t *testing.T) {
	// Test would require a constraint that accepts floats
	// Skip for now as not clear which constraints use floats
	t.Skip("Float constraint arguments not tested")
}

func TestFormatterWriteLine(t *testing.T) {
	config := DefaultConfig()
	formatter := New(config)

	// Test writeLine with empty string
	formatter.writeLine("")
	if formatter.buf.Len() != 1 { // Just newline
		t.Errorf("writeLine with empty string should add just newline")
	}

	// Test writeLine with content
	formatter.buf.Reset()
	formatter.writeLine("test content")
	if !strings.Contains(formatter.buf.String(), "test content") {
		t.Errorf("writeLine should add content")
	}
}

func TestFormatterFieldWithoutAlignment(t *testing.T) {
	input := `resource User {
id: uuid!
verylongfieldname: string!
x: int!
}`

	config := &Config{
		IndentSize:  2,
		AlignFields: false,
	}

	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Without alignment, each field should have minimal spacing
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") && strings.Contains(line, "  : ") {
			t.Errorf("Fields should not have extra padding when alignment is off: %s", line)
		}
	}
}

func TestFormatterRelationshipWithoutMetadata(t *testing.T) {
	input := `resource Post {
id: uuid! @primary @auto
author: User!
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Relationship without metadata shouldn't have braces
	if strings.Contains(result, "author: User! {") {
		t.Errorf("Relationship without metadata should not have braces")
	}
}

func TestFormatterRelationshipOnlyForeignKey(t *testing.T) {
	input := `resource Post {
id: uuid! @primary @auto
author: User! {
foreign_key: "author_id"
}
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "foreign_key:") {
		t.Errorf("Relationship metadata should preserve foreign_key")
	}
}

func TestFormatterRelationshipOnlyOnDelete(t *testing.T) {
	input := `resource Post {
id: uuid! @primary @auto
author: User! {
on_delete: cascade
}
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "on_delete:") {
		t.Errorf("Relationship metadata should preserve on_delete")
	}
}

func TestDiffStatsNoChanges(t *testing.T) {
	original := "line1\nline2"
	formatted := "line1\nline2"

	diff := Diff(original, formatted)
	stats := diff.Stats()

	if stats != "No changes" {
		t.Errorf("Stats should say 'No changes' when there are none, got: %s", stats)
	}
}

func TestFormatterArrayTypes(t *testing.T) {
	input := `resource User {
id: uuid! @primary @auto
tags: array<string>!
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "array<string>!") {
		t.Errorf("Array type not formatted correctly")
	}
}

func TestFormatterHashTypes(t *testing.T) {
	input := `resource Config {
id: uuid! @primary @auto
metadata: hash<string, string>!
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "hash<string, string>!") {
		t.Errorf("Hash type not formatted correctly")
	}
}

func TestUnifiedDiffCorrectness(t *testing.T) {
	original := "line1\nline2\nline3\nline4"
	formatted := "line1\nMODIFIED\nline3\nline4"

	diff := Diff(original, formatted)
	unified := diff.UnifiedDiff("test.cdt")

	// Should show line 2 as changed
	// Note: Simple line-by-line diff creates one hunk per changed line
	if !strings.Contains(unified, "@@") {
		t.Errorf("Should contain diff hunks")
	}
	if !strings.Contains(unified, "-line2") {
		t.Errorf("Should show original line2 removed")
	}
	if !strings.Contains(unified, "+MODIFIED") {
		t.Errorf("Should show MODIFIED added")
	}
	if !strings.Contains(unified, "--- a/test.cdt") {
		t.Errorf("Should have proper unified diff header")
	}
	if !strings.Contains(unified, "+++ b/test.cdt") {
		t.Errorf("Should have proper unified diff header")
	}

	// Verify that unchanged lines are not shown
	unchangedLineCount := strings.Count(unified, "line1") + strings.Count(unified, "line3") + strings.Count(unified, "line4")
	if unchangedLineCount > 0 {
		t.Errorf("Unchanged lines should not appear in diff output")
	}
}
