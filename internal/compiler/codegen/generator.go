// Package codegen generates idiomatic Go code from Conduit AST.
// It transforms resources into structs, CRUD operations, and database schema.
package codegen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Generator transforms AST nodes into Go code
type Generator struct {
	buf     *bytes.Buffer
	indent  int
	imports map[string]bool
}

// NewGenerator creates a new code generator
func NewGenerator() *Generator {
	return &Generator{
		buf:     &bytes.Buffer{},
		indent:  0,
		imports: make(map[string]bool),
	}
}

// GenerateProgram generates Go code for an entire program
func (g *Generator) GenerateProgram(prog *ast.Program) (map[string]string, error) {
	files := make(map[string]string)

	// Generate models for each resource
	for _, resource := range prog.Resources {
		code, err := g.GenerateResource(resource)
		if err != nil {
			return nil, fmt.Errorf("failed to generate resource %s: %w", resource.Name, err)
		}
		filename := fmt.Sprintf("models/%s.go", strings.ToLower(resource.Name))
		files[filename] = code
	}

	// Generate migrations
	migrations, err := g.GenerateMigrations(prog.Resources)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migrations: %w", err)
	}
	files["migrations/001_init.sql"] = migrations

	// Generate introspection metadata
	metaJSON, err := g.GenerateMetadata(prog)
	if err != nil {
		return nil, fmt.Errorf("failed to generate metadata: %w", err)
	}
	files["introspection/metadata.json"] = metaJSON

	// Generate metadata accessor Go file
	metaCode, err := g.GenerateMetadataAccessor(metaJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to generate metadata accessor: %w", err)
	}
	files["introspection/introspection.go"] = metaCode

	return files, nil
}

// GenerateResource generates Go code for a single resource
func (g *Generator) GenerateResource(resource *ast.ResourceNode) (string, error) {
	g.reset()

	// Validate resource name (should be caught by type checker, but defensive)
	if len(resource.Name) == 0 {
		return "", fmt.Errorf("codegen: resource name cannot be empty (should be caught by type checker)")
	}

	// Add package declaration
	g.writeLine("package models")
	g.writeLine("")

	// Collect imports needed for this resource
	g.collectImports(resource)

	// Write imports
	if len(g.imports) > 0 {
		g.writeImports()
		g.writeLine("")
	}

	// Generate struct
	if err := g.generateStruct(resource); err != nil {
		return "", err
	}
	g.writeLine("")

	// Generate TableName method
	g.generateTableName(resource)
	g.writeLine("")

	// Generate Validate method
	g.generateValidate(resource)
	g.writeLine("")

	// Generate CRUD methods
	g.generateCreate(resource)
	g.writeLine("")

	g.generateFindByID(resource)
	g.writeLine("")

	g.generateUpdate(resource)
	g.writeLine("")

	g.generateDelete(resource)
	g.writeLine("")

	g.generateFindAll(resource)

	return g.buf.String(), nil
}

// reset clears the generator state
func (g *Generator) reset() {
	g.buf.Reset()
	g.indent = 0
	g.imports = make(map[string]bool)
}

// writeLine writes a formatted line with proper indentation
func (g *Generator) writeLine(format string, args ...interface{}) {
	if format == "" {
		g.buf.WriteString("\n")
		return
	}

	// Add indentation
	for i := 0; i < g.indent; i++ {
		g.buf.WriteString("\t")
	}

	// Write the formatted string
	if len(args) > 0 {
		g.buf.WriteString(fmt.Sprintf(format, args...))
	} else {
		g.buf.WriteString(format)
	}
	g.buf.WriteString("\n")
}

// collectImports scans the resource and determines which imports are needed
func (g *Generator) collectImports(resource *ast.ResourceNode) {
	needsSQL := false
	needsTime := false
	needsUUID := false
	needsContext := false

	for _, field := range resource.Fields {
		switch field.Type.Name {
		case "timestamp":
			needsTime = true
		case "uuid":
			needsUUID = true
		}

		if field.Nullable && field.Type.Name != "bool" {
			needsSQL = true
		}
	}

	// CRUD operations always need context and database/sql
	needsContext = true
	needsSQL = true

	if needsContext {
		g.imports["context"] = true
	}
	if needsSQL {
		g.imports["database/sql"] = true
	}
	if needsTime {
		g.imports["time"] = true
	}
	if needsUUID {
		g.imports["github.com/google/uuid"] = true
	}

	// Always need fmt for error handling
	g.imports["fmt"] = true
}

// writeImports writes the import block
func (g *Generator) writeImports() {
	g.writeLine("import (")
	g.indent++

	// Sort imports: stdlib first, then external
	var stdlibImports []string
	var externalImports []string

	for imp := range g.imports {
		if strings.Contains(imp, ".") {
			externalImports = append(externalImports, imp)
		} else {
			stdlibImports = append(stdlibImports, imp)
		}
	}

	// Write stdlib imports
	for _, imp := range sortStrings(stdlibImports) {
		g.writeLine("%q", imp)
	}

	// Add blank line if we have both types
	if len(stdlibImports) > 0 && len(externalImports) > 0 {
		g.writeLine("")
	}

	// Write external imports
	for _, imp := range sortStrings(externalImports) {
		g.writeLine("%q", imp)
	}

	g.indent--
	g.writeLine(")")
}

// toGoType converts a Conduit type to a Go type string
func (g *Generator) toGoType(field *ast.FieldNode) string {
	typeName := field.Type.Name

	var goType string
	switch typeName {
	case "string", "text", "markdown":
		goType = "string"
	case "int":
		goType = "int64"
	case "float":
		goType = "float64"
	case "bool":
		goType = "bool"
	case "uuid":
		goType = "uuid.UUID"
	case "timestamp":
		goType = "time.Time"
	case "json":
		goType = "[]byte"
	default:
		// For resource types (relationships)
		goType = typeName
	}

	// Handle nullable fields
	if field.Nullable {
		switch typeName {
		case "string", "text", "markdown":
			return "sql.NullString"
		case "int":
			return "sql.NullInt64"
		case "float":
			return "sql.NullFloat64"
		case "bool":
			return "sql.NullBool"
		case "timestamp":
			return "sql.NullTime"
		default:
			// For complex types, use pointers
			return "*" + goType
		}
	}

	return goType
}

// toGoFieldName converts a snake_case field name to PascalCase
func (g *Generator) toGoFieldName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[0:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// toDBColumnName converts a field name to snake_case for database columns
func (g *Generator) toDBColumnName(name string) string {
	// For now, assume field names are already in snake_case
	return strings.ToLower(name)
}

// toTableName converts a resource name to a database table name (pluralized, snake_case)
func (g *Generator) toTableName(name string) string {
	// Simple pluralization: just add 's'
	// TODO: Handle irregular plurals
	return strings.ToLower(name) + "s"
}

// generateStructTags generates struct tags for a field
func (g *Generator) generateStructTags(field *ast.FieldNode) string {
	dbTag := g.toDBColumnName(field.Name)
	jsonTag := field.Name

	// For nullable fields, add omitempty to JSON
	if field.Nullable {
		jsonTag += ",omitempty"
	}

	return fmt.Sprintf("`db:%q json:%q`", dbTag, jsonTag)
}

// sortStrings is a simple bubble sort for string slices
func sortStrings(strs []string) []string {
	result := make([]string, len(strs))
	copy(result, strs)

	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}
