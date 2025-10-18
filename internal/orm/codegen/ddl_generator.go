// Package codegen provides DDL generation for database schemas
package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// DDLGenerator generates PostgreSQL DDL statements from resource schemas
type DDLGenerator struct {
	typeMapper *TypeMapper
}

// NewDDLGenerator creates a new DDL generator
func NewDDLGenerator() *DDLGenerator {
	return &DDLGenerator{
		typeMapper: NewTypeMapper(),
	}
}

// GenerateCreateTable generates a CREATE TABLE statement for a resource
func (g *DDLGenerator) GenerateCreateTable(resource *schema.ResourceSchema) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("resource cannot be nil")
	}

	var b strings.Builder

	// Start CREATE TABLE statement
	tableName := resource.TableName
	if tableName == "" {
		tableName = toSnakeCase(resource.Name)
	}

	b.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", QuoteIdentifier(tableName)))

	// Collect and sort fields for optimal column ordering
	// Fixed-length types first, then variable-length types
	fields := g.orderFields(resource)

	// Generate column definitions
	columnDefs := make([]string, 0, len(fields))
	for _, field := range fields {
		columnDef, err := g.generateColumnDefinition(resource.Name, field)
		if err != nil {
			return "", fmt.Errorf("field %s: %w", field.Name, err)
		}
		columnDefs = append(columnDefs, columnDef)
	}

	// Write column definitions
	for i, def := range columnDefs {
		b.WriteString("  ")
		b.WriteString(def)
		if i < len(columnDefs)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}

	b.WriteString(");")

	return b.String(), nil
}

// generateColumnDefinition generates a column definition for a field
func (g *DDLGenerator) generateColumnDefinition(resourceName string, field *schema.Field) (string, error) {
	var parts []string

	// Column name (quoted to prevent SQL injection)
	columnName := toSnakeCase(field.Name)
	parts = append(parts, QuoteIdentifier(columnName))

	// Column type
	var columnType string
	if len(field.Type.EnumValues) > 0 {
		// For enum types, use the generated enum type name
		columnType = g.typeMapper.GetEnumTypeName(resourceName, field.Name)
	} else {
		mappedType, err := g.typeMapper.MapType(field.Type)
		if err != nil {
			return "", fmt.Errorf("mapping type: %w", err)
		}
		columnType = mappedType
	}
	parts = append(parts, columnType)

	// Nullability
	nullability := g.typeMapper.MapNullability(field.Type)
	parts = append(parts, nullability)

	// Default value
	defaultValue, err := g.typeMapper.MapDefault(field.Type)
	if err != nil {
		return "", fmt.Errorf("mapping default value: %w", err)
	}

	// Handle @auto annotation for UUIDs and timestamps
	hasAuto := false
	for _, annotation := range field.Annotations {
		if annotation.Name == "auto" {
			hasAuto = true
			break
		}
	}

	if hasAuto {
		if field.Type.BaseType == schema.TypeUUID {
			parts = append(parts, "DEFAULT gen_random_uuid()")
		} else if field.Type.BaseType == schema.TypeTimestamp {
			parts = append(parts, "DEFAULT CURRENT_TIMESTAMP")
		}
	} else if defaultValue != "" {
		parts = append(parts, "DEFAULT "+defaultValue)
	}

	// Handle @auto_update annotation for timestamps
	for _, annotation := range field.Annotations {
		if annotation.Name == "auto_update" {
			if field.Type.BaseType == schema.TypeTimestamp {
				// This will require a trigger, noted for later implementation
				// For now, we just document it
			}
		}
	}

	// Primary key constraint (inline)
	for _, annotation := range field.Annotations {
		if annotation.Name == "primary" {
			parts = append(parts, "PRIMARY KEY")
			break
		}
	}

	return strings.Join(parts, " "), nil
}

// orderFields orders fields for optimal column layout
// Fixed-length types come first, then variable-length types
func (g *DDLGenerator) orderFields(resource *schema.ResourceSchema) []*schema.Field {
	fields := make([]*schema.Field, 0, len(resource.Fields))
	for _, field := range resource.Fields {
		fields = append(fields, field)
	}

	// Sort by column order priority
	sort.Slice(fields, func(i, j int) bool {
		return g.getColumnOrderPriority(fields[i]) < g.getColumnOrderPriority(fields[j])
	})

	return fields
}

// getColumnOrderPriority returns a priority for column ordering
// Lower numbers come first
func (g *DDLGenerator) getColumnOrderPriority(field *schema.Field) int {
	// Primary key always first
	for _, annotation := range field.Annotations {
		if annotation.Name == "primary" {
			return 0
		}
	}

	// Then fixed-length types
	switch field.Type.BaseType {
	case schema.TypeBool:
		return 1
	case schema.TypeInt:
		return 2
	case schema.TypeBigInt:
		return 3
	case schema.TypeFloat, schema.TypeDecimal:
		return 4
	case schema.TypeUUID:
		return 5
	case schema.TypeULID:
		return 6
	case schema.TypeDate:
		return 7
	case schema.TypeTime:
		return 8
	case schema.TypeTimestamp:
		return 9
	case schema.TypeString:
		// Short strings
		if field.Type.Length != nil && *field.Type.Length <= 50 {
			return 10
		}
		return 20
	case schema.TypeEmail, schema.TypeURL, schema.TypePhone:
		return 11
	case schema.TypeText, schema.TypeMarkdown:
		// Variable-length text
		return 30
	case schema.TypeJSON, schema.TypeJSONB:
		return 40
	}

	// Arrays and complex types last
	if field.Type.ArrayElement != nil {
		return 50
	}
	if field.Type.HashKey != nil {
		return 60
	}

	return 100
}

// GenerateEnumType generates a CREATE TYPE statement for an enum field
func (g *DDLGenerator) GenerateEnumType(resourceName, fieldName string, values []string) string {
	enumTypeName := g.typeMapper.GetEnumTypeName(resourceName, fieldName)

	// Quote each enum value and escape single quotes
	quotedValues := make([]string, len(values))
	for i, v := range values {
		// Escape single quotes by doubling them (PostgreSQL standard)
		escaped := strings.ReplaceAll(v, "'", "''")
		quotedValues[i] = fmt.Sprintf("'%s'", escaped)
	}

	return fmt.Sprintf(
		"CREATE TYPE %s AS ENUM (%s);",
		QuoteIdentifier(enumTypeName),
		strings.Join(quotedValues, ", "),
	)
}

// GenerateEnumTypes generates all enum type definitions for a resource
func (g *DDLGenerator) GenerateEnumTypes(resource *schema.ResourceSchema) []string {
	var enumTypes []string

	for fieldName, field := range resource.Fields {
		if len(field.Type.EnumValues) > 0 {
			enumType := g.GenerateEnumType(resource.Name, fieldName, field.Type.EnumValues)
			enumTypes = append(enumTypes, enumType)
		}
	}

	// Sort for deterministic output
	sort.Strings(enumTypes)

	return enumTypes
}

// GenerateSchema generates complete DDL for a resource (enums + table)
func (g *DDLGenerator) GenerateSchema(resource *schema.ResourceSchema) (string, error) {
	var b strings.Builder

	// Generate enum types first
	enumTypes := g.GenerateEnumTypes(resource)
	for _, enumType := range enumTypes {
		b.WriteString(enumType)
		b.WriteString("\n")
	}

	if len(enumTypes) > 0 {
		b.WriteString("\n")
	}

	// Generate CREATE TABLE
	createTable, err := g.GenerateCreateTable(resource)
	if err != nil {
		return "", err
	}

	b.WriteString(createTable)

	return b.String(), nil
}

// GenerateDropTable generates a DROP TABLE statement
func (g *DDLGenerator) GenerateDropTable(resource *schema.ResourceSchema) string {
	tableName := resource.TableName
	if tableName == "" {
		tableName = toSnakeCase(resource.Name)
	}

	return fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", QuoteIdentifier(tableName))
}

// GenerateDropEnumTypes generates DROP TYPE statements for all enum fields
func (g *DDLGenerator) GenerateDropEnumTypes(resource *schema.ResourceSchema) []string {
	var dropStatements []string

	for fieldName, field := range resource.Fields {
		if len(field.Type.EnumValues) > 0 {
			enumTypeName := g.typeMapper.GetEnumTypeName(resource.Name, fieldName)
			dropStatements = append(dropStatements, fmt.Sprintf("DROP TYPE IF EXISTS %s;", QuoteIdentifier(enumTypeName)))
		}
	}

	// Sort for deterministic output
	sort.Strings(dropStatements)

	return dropStatements
}
