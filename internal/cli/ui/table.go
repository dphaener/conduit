package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// Table represents a simple table for displaying tabular data
type Table struct {
	writer  io.Writer
	headers []string
	rows    [][]string
	noColor bool
}

// TableOptions configures table behavior
type TableOptions struct {
	NoColor bool
}

// NewTable creates a new table with the given headers
func NewTable(w io.Writer, headers []string, opts *TableOptions) *Table {
	noColor := false
	if opts != nil {
		noColor = opts.NoColor
	}

	return &Table{
		writer:  w,
		headers: headers,
		rows:    make([][]string, 0),
		noColor: noColor,
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	t.rows = append(t.rows, cells)
}

// Render renders the table to the writer
func (t *Table) Render() {
	if len(t.headers) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(t.headers))
	for i, header := range t.headers {
		widths[i] = len(header)
	}

	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Render header
	bold := color.New(color.Bold, color.FgCyan)
	if t.noColor {
		bold.DisableColor()
	}
	for i, header := range t.headers {
		bold.Fprint(t.writer, padRight(header, widths[i]))
		if i < len(t.headers)-1 {
			fmt.Fprint(t.writer, "  ")
		}
	}
	fmt.Fprintln(t.writer)

	// Render separator
	gray := color.New(color.FgHiBlack)
	if t.noColor {
		gray.DisableColor()
	}
	for i, width := range widths {
		gray.Fprint(t.writer, strings.Repeat("─", width))
		if i < len(widths)-1 {
			gray.Fprint(t.writer, "  ")
		}
	}
	fmt.Fprintln(t.writer)

	// Render rows
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprint(t.writer, padRight(cell, widths[i]))
				if i < len(row)-1 {
					fmt.Fprint(t.writer, "  ")
				}
			}
		}
		fmt.Fprintln(t.writer)
	}
}

// padRight pads a string with spaces on the right to reach the target width
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// KeyValueTable renders a simple key-value table (2 columns)
type KeyValueTable struct {
	writer  io.Writer
	rows    []kvRow
	noColor bool
}

type kvRow struct {
	key   string
	value string
}

// NewKeyValueTable creates a new key-value table
func NewKeyValueTable(w io.Writer, noColor bool) *KeyValueTable {
	return &KeyValueTable{
		writer:  w,
		rows:    make([]kvRow, 0),
		noColor: noColor,
	}
}

// AddRow adds a key-value pair to the table
func (t *KeyValueTable) AddRow(key, value string) {
	t.rows = append(t.rows, kvRow{key: key, value: value})
}

// Render renders the key-value table
func (t *KeyValueTable) Render() {
	if len(t.rows) == 0 {
		return
	}

	// Calculate max key width
	maxKeyWidth := 0
	for _, row := range t.rows {
		if len(row.key) > maxKeyWidth {
			maxKeyWidth = len(row.key)
		}
	}

	// Render rows
	cyan := color.New(color.FgCyan)
	if t.noColor {
		cyan.DisableColor()
	}
	for _, row := range t.rows {
		cyan.Fprint(t.writer, padRight(row.key+":", maxKeyWidth+1))
		fmt.Fprintf(t.writer, " %s\n", row.value)
	}
}

// Section represents a titled section with content
type Section struct {
	writer  io.Writer
	title   string
	content []string
	noColor bool
}

// NewSection creates a new section
func NewSection(w io.Writer, title string, noColor bool) *Section {
	return &Section{
		writer:  w,
		title:   title,
		content: make([]string, 0),
		noColor: noColor,
	}
}

// AddLine adds a line to the section content
func (s *Section) AddLine(line string) {
	s.content = append(s.content, line)
}

// Render renders the section
func (s *Section) Render() {
	// Render title
	bold := color.New(color.Bold, color.FgCyan)
	if s.noColor {
		bold.DisableColor()
	}
	bold.Fprintln(s.writer, s.title)

	// Render content with indentation
	for _, line := range s.content {
		fmt.Fprintf(s.writer, "  %s\n", line)
	}

	// Add spacing after section
	fmt.Fprintln(s.writer)
}

// List represents a bulleted or numbered list
type List struct {
	writer   io.Writer
	items    []string
	numbered bool
	noColor  bool
}

// ListOptions configures list behavior
type ListOptions struct {
	Numbered bool
	NoColor  bool
}

// NewList creates a new list
func NewList(w io.Writer, opts ListOptions) *List {
	return &List{
		writer:   w,
		items:    make([]string, 0),
		numbered: opts.Numbered,
		noColor:  opts.NoColor,
	}
}

// AddItem adds an item to the list
func (l *List) AddItem(item string) {
	l.items = append(l.items, item)
}

// Render renders the list
func (l *List) Render() {
	if len(l.items) == 0 {
		return
	}

	cyan := color.New(color.FgCyan)
	if l.noColor {
		cyan.DisableColor()
	}

	for i, item := range l.items {
		if l.numbered {
			cyan.Fprintf(l.writer, "%d. ", i+1)
		} else {
			cyan.Fprint(l.writer, "• ")
		}
		fmt.Fprintln(l.writer, item)
	}
}

// Divider renders a horizontal divider line
func Divider(w io.Writer, width int, noColor bool) {
	if width == 0 {
		width = 80
	}

	gray := color.New(color.FgHiBlack)
	if noColor {
		gray.DisableColor()
	}
	gray.Fprintln(w, strings.Repeat("─", width))
}

// Header renders a styled header
func Header(w io.Writer, title string, noColor bool) {
	bold := color.New(color.Bold, color.FgCyan)
	if noColor {
		bold.DisableColor()
	}
	bold.Fprintln(w, title)
	Divider(w, len(title), noColor)
}
