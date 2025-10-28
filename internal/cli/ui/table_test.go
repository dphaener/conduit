package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestTable(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	table := NewTable(&buf, []string{"Name", "Type", "Required"}, &TableOptions{NoColor: true})

	table.AddRow("id", "uuid", "yes")
	table.AddRow("title", "string", "yes")
	table.AddRow("content", "text", "no")

	table.Render()

	output := buf.String()

	// Check headers
	if !strings.Contains(output, "Name") {
		t.Errorf("Table output missing header 'Name'")
	}
	if !strings.Contains(output, "Type") {
		t.Errorf("Table output missing header 'Type'")
	}
	if !strings.Contains(output, "Required") {
		t.Errorf("Table output missing header 'Required'")
	}

	// Check rows
	if !strings.Contains(output, "id") {
		t.Errorf("Table output missing row data 'id'")
	}
	if !strings.Contains(output, "uuid") {
		t.Errorf("Table output missing row data 'uuid'")
	}
	if !strings.Contains(output, "title") {
		t.Errorf("Table output missing row data 'title'")
	}

	// Check separator
	if !strings.Contains(output, "─") {
		t.Errorf("Table output missing separator")
	}
}

func TestTableEmpty(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	table := NewTable(&buf, []string{}, &TableOptions{NoColor: true})

	table.Render()

	output := buf.String()
	if output != "" {
		t.Errorf("Expected empty output for table with no headers, got: %q", output)
	}
}

func TestKeyValueTable(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	kvTable := NewKeyValueTable(&buf, true)

	kvTable.AddRow("Name", "Post")
	kvTable.AddRow("Type", "Resource")
	kvTable.AddRow("Fields", "5")

	kvTable.Render()

	output := buf.String()

	expected := []string{
		"Name:",
		"Post",
		"Type:",
		"Resource",
		"Fields:",
		"5",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("KeyValueTable output missing: %q", exp)
		}
	}
}

func TestKeyValueTableEmpty(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	kvTable := NewKeyValueTable(&buf, true)

	kvTable.Render()

	output := buf.String()
	if output != "" {
		t.Errorf("Expected empty output for empty KeyValueTable, got: %q", output)
	}
}

func TestSection(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	section := NewSection(&buf, "Fields", true)

	section.AddLine("id: uuid!")
	section.AddLine("title: string!")
	section.AddLine("content: text?")

	section.Render()

	output := buf.String()

	if !strings.Contains(output, "Fields") {
		t.Errorf("Section output missing title 'Fields'")
	}

	expected := []string{
		"id: uuid!",
		"title: string!",
		"content: text?",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Section output missing line: %q", exp)
		}
	}
}

func TestSectionEmpty(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	section := NewSection(&buf, "Empty Section", true)

	section.Render()

	output := buf.String()
	if !strings.Contains(output, "Empty Section") {
		t.Errorf("Expected title even for empty section")
	}
}

func TestList(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	list := NewList(&buf, ListOptions{Numbered: false, NoColor: true})

	list.AddItem("First item")
	list.AddItem("Second item")
	list.AddItem("Third item")

	list.Render()

	output := buf.String()

	if !strings.Contains(output, "•") {
		t.Errorf("List output missing bullet points")
	}

	expected := []string{
		"First item",
		"Second item",
		"Third item",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("List output missing item: %q", exp)
		}
	}
}

func TestListNumbered(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	list := NewList(&buf, ListOptions{Numbered: true, NoColor: true})

	list.AddItem("First item")
	list.AddItem("Second item")
	list.AddItem("Third item")

	list.Render()

	output := buf.String()

	expected := []string{
		"1.",
		"2.",
		"3.",
		"First item",
		"Second item",
		"Third item",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Numbered list output missing: %q", exp)
		}
	}
}

func TestListEmpty(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	list := NewList(&buf, ListOptions{NoColor: true})

	list.Render()

	output := buf.String()
	if output != "" {
		t.Errorf("Expected empty output for empty list, got: %q", output)
	}
}

func TestDivider(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	Divider(&buf, 40, true)

	output := buf.String()

	if !strings.Contains(output, "─") {
		t.Errorf("Divider output missing line character")
	}

	// Should have 40 characters plus newline
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 && len(lines[0]) < 30 {
		t.Errorf("Divider seems too short")
	}
}

func TestDividerDefaultWidth(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	Divider(&buf, 0, true) // 0 should use default width of 80

	output := buf.String()

	if !strings.Contains(output, "─") {
		t.Errorf("Divider output missing line character")
	}
}

func TestHeader(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	Header(&buf, "Test Header", true)

	output := buf.String()

	if !strings.Contains(output, "Test Header") {
		t.Errorf("Header output missing title")
	}

	if !strings.Contains(output, "─") {
		t.Errorf("Header output missing divider")
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"test", 10, "test      "},
		{"test", 4, "test"},
		{"test", 2, "test"},
		{"", 5, "     "},
	}

	for _, tt := range tests {
		result := padRight(tt.input, tt.width)
		if result != tt.expected {
			t.Errorf("padRight(%q, %d) = %q; want %q", tt.input, tt.width, result, tt.expected)
		}
	}
}

func TestTableAlignment(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	table := NewTable(&buf, []string{"Short", "VeryLongHeader"}, &TableOptions{NoColor: true})

	table.AddRow("a", "b")
	table.AddRow("longer", "c")

	table.Render()

	output := buf.String()

	// The columns should be aligned based on the longest content
	lines := strings.Split(output, "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines (header, separator, row)")
	}

	// Check that each row has consistent column positions
	// This is a basic check - more sophisticated alignment testing could be added
	for i, line := range lines {
		if line == "" {
			continue
		}
		if i > 0 && len(line) < 10 {
			t.Errorf("Line %d seems too short for proper alignment: %q", i, line)
		}
	}
}
