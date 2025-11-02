package format

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/conduit-lang/conduit/compiler/lexer"
	"github.com/conduit-lang/conduit/compiler/parser"
)

// Formatter formats Conduit source code
type Formatter struct {
	config *Config
	buf    *bytes.Buffer
	indent int
}

// New creates a new Formatter with the given configuration
func New(config *Config) *Formatter {
	if config == nil {
		config = DefaultConfig()
	}
	return &Formatter{
		config: config,
		buf:    new(bytes.Buffer),
		indent: 0,
	}
}

// Format formats Conduit source code and returns the formatted result
func (f *Formatter) Format(source string) (string, error) {
	// Tokenize
	l := lexer.New(source, "")
	l.SetPreserveComments(true) // Enable comment preservation
	tokens, lexErrors := l.ScanTokens()

	// Check for lexer errors
	if len(lexErrors) > 0 {
		return "", fmt.Errorf("lexer errors: %v", lexErrors)
	}

	// Parse
	p := parser.NewWithSource(tokens, source)
	program, parseErrors := p.Parse()

	if len(parseErrors) > 0 {
		return "", fmt.Errorf("parse errors: %v", parseErrors)
	}

	// Reset buffer
	f.buf.Reset()
	f.indent = 0

	// Format the AST
	f.formatProgram(program)

	return f.buf.String(), nil
}

// FormatFile formats a Conduit source file
func FormatFile(path string, config *Config) (string, error) {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Format
	formatter := New(config)
	return formatter.Format(string(content))
}

// formatProgram formats a Program node
func (f *Formatter) formatProgram(program *parser.Program) {
	for i, resource := range program.Resources {
		f.formatResource(resource)

		// Add blank line between resources (except after last one)
		if i < len(program.Resources)-1 {
			f.writeLine("")
		}
	}
}

// formatResource formats a ResourceNode
func (f *Formatter) formatResource(resource *parser.ResourceNode) {
	// Write leading comments if present
	if resource.LeadingComment != "" {
		f.writeComments(resource.LeadingComment)
	}

	// Write documentation if present
	if resource.Documentation != "" {
		lines := strings.Split(resource.Documentation, "\n")
		for _, line := range lines {
			f.writeIndent()
			f.buf.WriteString("/// ")
			f.buf.WriteString(strings.TrimSpace(line))
			f.buf.WriteString("\n")
		}
	}

	// Write resource declaration
	f.writeIndent()
	f.buf.WriteString("resource ")
	f.buf.WriteString(resource.Name)
	f.buf.WriteString(" {\n")
	f.indent++

	// Calculate max field name length for alignment if enabled
	maxFieldLen := 0
	if f.config.AlignFields {
		for _, field := range resource.Fields {
			if len(field.Name) > maxFieldLen {
				maxFieldLen = len(field.Name)
			}
		}
		for _, rel := range resource.Relationships {
			if len(rel.Name) > maxFieldLen {
				maxFieldLen = len(rel.Name)
			}
		}
	}

	// Write fields
	for _, field := range resource.Fields {
		f.formatField(field, maxFieldLen)
	}

	// Write relationships
	if len(resource.Relationships) > 0 && len(resource.Fields) > 0 {
		f.writeLine("")
	}
	for _, rel := range resource.Relationships {
		f.formatRelationship(rel, maxFieldLen)
	}

	// Write hooks
	if len(resource.Hooks) > 0 && (len(resource.Fields) > 0 || len(resource.Relationships) > 0) {
		f.writeLine("")
	}
	for _, hook := range resource.Hooks {
		f.formatHook(hook)
	}

	// Write custom constraints
	if len(resource.Constraints) > 0 && (len(resource.Fields) > 0 || len(resource.Relationships) > 0 || len(resource.Hooks) > 0) {
		f.writeLine("")
	}
	for _, constraint := range resource.Constraints {
		f.formatCustomConstraint(constraint)
	}

	f.indent--
	f.writeIndent()
	f.buf.WriteString("}\n")
}

// formatField formats a FieldNode
func (f *Formatter) formatField(field *parser.FieldNode, maxLen int) {
	// Write leading comments if present
	if field.LeadingComment != "" {
		f.writeComments(field.LeadingComment)
	}

	f.writeIndent()
	f.buf.WriteString(field.Name)

	// Add padding for alignment if enabled
	if f.config.AlignFields && maxLen > 0 {
		padding := maxLen - len(field.Name)
		f.buf.WriteString(strings.Repeat(" ", padding))
	}

	f.buf.WriteString(": ")
	f.formatType(field.Type)

	// Write nullability
	if field.Nullable {
		f.buf.WriteString("?")
	} else {
		f.buf.WriteString("!")
	}

	// Write constraints
	for _, constraint := range field.Constraints {
		f.buf.WriteString(" ")
		f.formatConstraint(constraint)
	}

	// Write trailing comment if present
	if field.TrailingComment != "" {
		f.buf.WriteString(" ")
		f.buf.WriteString(field.TrailingComment)
	}

	f.buf.WriteString("\n")
}

// formatRelationship formats a RelationshipNode
func (f *Formatter) formatRelationship(rel *parser.RelationshipNode, maxLen int) {
	// Write leading comments if present
	if rel.LeadingComment != "" {
		f.writeComments(rel.LeadingComment)
	}

	f.writeIndent()
	f.buf.WriteString(rel.Name)

	// Add padding for alignment if enabled
	if f.config.AlignFields && maxLen > 0 {
		padding := maxLen - len(rel.Name)
		f.buf.WriteString(strings.Repeat(" ", padding))
	}

	f.buf.WriteString(": ")
	f.buf.WriteString(rel.TargetType)

	// Write nullability
	if rel.Nullable {
		f.buf.WriteString("?")
	} else {
		f.buf.WriteString("!")
	}

	// Write relationship metadata if present
	if rel.ForeignKey != "" || rel.OnDelete != "" || rel.OnUpdate != "" {
		f.buf.WriteString(" {\n")
		f.indent++

		if rel.ForeignKey != "" {
			f.writeIndent()
			f.buf.WriteString("foreign_key: \"")
			f.buf.WriteString(rel.ForeignKey)
			f.buf.WriteString("\"\n")
		}

		if rel.OnDelete != "" {
			f.writeIndent()
			f.buf.WriteString("on_delete: ")
			f.buf.WriteString(rel.OnDelete)
			f.buf.WriteString("\n")
		}

		if rel.OnUpdate != "" {
			f.writeIndent()
			f.buf.WriteString("on_update: ")
			f.buf.WriteString(rel.OnUpdate)
			f.buf.WriteString("\n")
		}

		f.indent--
		f.writeIndent()
		f.buf.WriteString("}")
	}

	// Write trailing comment if present
	if rel.TrailingComment != "" {
		f.buf.WriteString(" ")
		f.buf.WriteString(rel.TrailingComment)
	}

	f.buf.WriteString("\n")
}

// formatType formats a TypeNode
func (f *Formatter) formatType(typ parser.TypeNode) {
	switch typ.Kind {
	case parser.TypeKindPrimitive:
		f.buf.WriteString(typ.Name)
	case parser.TypeKindArray:
		f.buf.WriteString("array<")
		if typ.ElementType != nil {
			f.formatType(*typ.ElementType)
		}
		f.buf.WriteString(">")
	case parser.TypeKindHash:
		f.buf.WriteString("hash<")
		if typ.KeyType != nil {
			f.formatType(*typ.KeyType)
		}
		f.buf.WriteString(", ")
		if typ.ValueType != nil {
			f.formatType(*typ.ValueType)
		}
		f.buf.WriteString(">")
	case parser.TypeKindEnum:
		f.buf.WriteString("enum(")
		for i, val := range typ.EnumValues {
			f.buf.WriteString("\"")
			f.buf.WriteString(val)
			f.buf.WriteString("\"")
			if i < len(typ.EnumValues)-1 {
				f.buf.WriteString(", ")
			}
		}
		f.buf.WriteString(")")
	case parser.TypeKindStruct:
		f.buf.WriteString("struct")
	case parser.TypeKindResource:
		f.buf.WriteString(typ.Name)
	}
}

// formatConstraint formats a ConstraintNode
func (f *Formatter) formatConstraint(constraint *parser.ConstraintNode) {
	f.buf.WriteString("@")
	f.buf.WriteString(constraint.Name)

	if len(constraint.Arguments) > 0 {
		f.buf.WriteString("(")
		for i, arg := range constraint.Arguments {
			f.formatConstraintArg(arg)
			if i < len(constraint.Arguments)-1 {
				f.buf.WriteString(", ")
			}
		}
		f.buf.WriteString(")")
	}
}

// formatConstraintArg formats a constraint argument
func (f *Formatter) formatConstraintArg(arg interface{}) {
	switch v := arg.(type) {
	case string:
		f.buf.WriteString("\"")
		f.buf.WriteString(v)
		f.buf.WriteString("\"")
	case int, int64, float64:
		f.buf.WriteString(fmt.Sprintf("%v", v))
	case bool:
		f.buf.WriteString(fmt.Sprintf("%v", v))
	default:
		f.buf.WriteString(fmt.Sprintf("%v", v))
	}
}

// writeIndent writes the current indentation level
func (f *Formatter) writeIndent() {
	spaces := f.indent * f.config.IndentSize
	f.buf.WriteString(strings.Repeat(" ", spaces))
}

// formatHook formats a HookNode
func (f *Formatter) formatHook(hook *parser.HookNode) {
	f.writeIndent()
	f.buf.WriteString("@")
	f.buf.WriteString(hook.Type)
	f.buf.WriteString(" ")
	f.buf.WriteString(hook.Trigger)
	f.buf.WriteString(" {\n")
	f.indent++

	// Write body with normalized indentation
	if hook.Body != "" {
		f.writeBodyWithNormalizedIndentation(hook.Body)
	}

	f.indent--
	f.writeIndent()
	f.buf.WriteString("}\n")
}

// formatCustomConstraint formats a CustomConstraintNode
func (f *Formatter) formatCustomConstraint(constraint *parser.CustomConstraintNode) {
	f.writeIndent()
	f.buf.WriteString("@constraint ")
	f.buf.WriteString(constraint.Name)
	f.buf.WriteString(" {\n")
	f.indent++

	// Write body with normalized indentation
	if constraint.Body != "" {
		f.writeBodyWithNormalizedIndentation(constraint.Body)
	}

	f.indent--
	f.writeIndent()
	f.buf.WriteString("}\n")
}

// writeLine writes a line with indentation
func (f *Formatter) writeLine(text string) {
	if text != "" {
		f.writeIndent()
		f.buf.WriteString(text)
	}
	f.buf.WriteString("\n")
}

// writeComments writes comments with proper indentation
// Comments are expected to include the # prefix
func (f *Formatter) writeComments(comments string) {
	if comments == "" {
		return
	}

	lines := strings.Split(strings.TrimRight(comments, "\n"), "\n")
	for _, line := range lines {
		f.writeIndent()
		f.buf.WriteString(strings.TrimSpace(line))
		f.buf.WriteString("\n")
	}
}

// writeBodyWithNormalizedIndentation writes a body string with normalized indentation
// It strips the common leading indentation from all lines and re-applies the formatter's base indentation
func (f *Formatter) writeBodyWithNormalizedIndentation(body string) {
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return
	}

	// Find minimum indentation (ignore empty lines)
	minIndent := -1
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		// Count leading spaces
		indent := 0
		for _, ch := range line {
			if ch == ' ' {
				indent++
			} else if ch == '\t' {
				indent += 4 // Treat tab as 4 spaces
			} else {
				break
			}
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	// If no non-empty lines, nothing to do
	if minIndent == -1 {
		return
	}

	// Write lines with normalized indentation
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			f.buf.WriteString("\n") // Preserve empty line
			continue
		}

		// Strip the common prefix indentation
		dedented := line
		if minIndent > 0 && len(line) >= minIndent {
			// Remove minIndent spaces/tabs from the beginning
			stripped := 0
			for i, ch := range line {
				if stripped >= minIndent {
					dedented = line[i:]
					break
				}
				if ch == ' ' {
					stripped++
				} else if ch == '\t' {
					stripped += 4
				} else {
					dedented = line[i:]
					break
				}
			}
		}

		// Write with formatter's base indentation
		f.writeIndent()
		f.buf.WriteString(dedented)
		f.buf.WriteString("\n")
	}
}
