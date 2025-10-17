package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// GenerateMigrations generates SQL migration file for all resources
func (g *Generator) GenerateMigrations(resources []*ast.ResourceNode) (string, error) {
	var sql strings.Builder

	sql.WriteString("-- Initial migration for Conduit resources\n")
	sql.WriteString("-- Generated automatically - do not edit\n\n")

	for _, resource := range resources {
		tableDDL, err := g.generateCreateTable(resource)
		if err != nil {
			return "", fmt.Errorf("failed to generate table for %s: %w", resource.Name, err)
		}
		sql.WriteString(tableDDL)
		sql.WriteString("\n")

		// Generate indexes
		indexDDL := g.generateIndexes(resource)
		if indexDDL != "" {
			sql.WriteString(indexDDL)
			sql.WriteString("\n")
		}
	}

	return sql.String(), nil
}

// generateCreateTable generates a CREATE TABLE statement for a resource
func (g *Generator) generateCreateTable(resource *ast.ResourceNode) (string, error) {
	var sql strings.Builder

	tableName := g.toTableName(resource.Name)
	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", tableName))

	// Check if resource has an ID field
	hasID := false
	for _, field := range resource.Fields {
		if field.Name == "id" {
			hasID = true
			break
		}
	}

	// Add default ID only if not explicitly defined
	if !hasID {
		sql.WriteString("  id BIGSERIAL PRIMARY KEY")
	}

	// Generate all explicitly defined fields
	for i, field := range resource.Fields {
		// Add comma before this field if we've already written any columns
		if i > 0 || !hasID {
			sql.WriteString(",\n")
		}

		columnDef, err := g.generateColumn(field)
		if err != nil {
			return "", err
		}
		sql.WriteString("  " + columnDef)
	}

	sql.WriteString("\n);\n")

	return sql.String(), nil
}

// generateColumn generates a column definition for a field
func (g *Generator) generateColumn(field *ast.FieldNode) (string, error) {
	columnName := g.toDBColumnName(field.Name)
	sqlType, err := g.toSQLType(field)
	if err != nil {
		return "", err
	}

	var parts []string
	parts = append(parts, columnName, sqlType)

	// Add constraints
	constraints := g.generateSQLConstraints(field)
	if constraints != "" {
		parts = append(parts, constraints)
	}

	return strings.Join(parts, " "), nil
}

// toSQLType converts a Conduit type to a PostgreSQL type
func (g *Generator) toSQLType(field *ast.FieldNode) (string, error) {
	var sqlType string

	switch field.Type.Name {
	case "string":
		// Check for max constraint to determine VARCHAR size
		maxLen := 255 // default
		for _, constraint := range field.Constraints {
			if constraint.Name == "max" && len(constraint.Arguments) > 0 {
				if max, ok := extractLiteralValue(constraint.Arguments[0]).(int); ok {
					maxLen = max
				} else if max64, ok := extractLiteralValue(constraint.Arguments[0]).(int64); ok {
					maxLen = int(max64)
				}
			}
		}
		sqlType = fmt.Sprintf("VARCHAR(%d)", maxLen)

	case "text", "markdown":
		sqlType = "TEXT"

	case "int":
		sqlType = "BIGINT"

	case "float":
		sqlType = "DOUBLE PRECISION"

	case "bool":
		sqlType = "BOOLEAN"

	case "uuid":
		sqlType = "UUID"

	case "timestamp":
		sqlType = "TIMESTAMP WITH TIME ZONE"

	case "json":
		sqlType = "JSONB"

	default:
		// For resource types (foreign keys)
		if field.Type.Kind == ast.TypeResource {
			sqlType = "BIGINT" // Assuming integer foreign keys
		} else {
			return "", fmt.Errorf("unsupported type: %s", field.Type.Name)
		}
	}

	return sqlType, nil
}

// generateSQLConstraints generates SQL constraints for a field
func (g *Generator) generateSQLConstraints(field *ast.FieldNode) string {
	var constraints []string

	// Automatically make 'id' fields PRIMARY KEY
	isPrimaryKey := field.Name == "id"

	// NOT NULL for required fields
	if !field.Nullable {
		constraints = append(constraints, "NOT NULL")
	}

	// Process field constraints
	for _, constraint := range field.Constraints {
		switch constraint.Name {
		case "primary":
			isPrimaryKey = true

		case "unique":
			constraints = append(constraints, "UNIQUE")

		case "auto":
			// For UUID auto generation
			if field.Type.Name == "uuid" {
				constraints = append(constraints, "DEFAULT gen_random_uuid()")
			}
			// For timestamp auto generation
			if field.Type.Name == "timestamp" {
				constraints = append(constraints, "DEFAULT CURRENT_TIMESTAMP")
			}

		case "min":
			// For string types, add CHECK constraint
			if field.Type.Name == "string" || field.Type.Name == "text" {
				if len(constraint.Arguments) > 0 {
					minVal := extractLiteralValue(constraint.Arguments[0])
					constraints = append(constraints,
						fmt.Sprintf("CHECK (length(%s) >= %v)", g.toDBColumnName(field.Name), minVal))
				}
			}

		case "max":
			// For numeric types, add CHECK constraint
			if field.Type.Name == "int" || field.Type.Name == "float" {
				if len(constraint.Arguments) > 0 {
					maxVal := extractLiteralValue(constraint.Arguments[0])
					constraints = append(constraints,
						fmt.Sprintf("CHECK (%s <= %v)", g.toDBColumnName(field.Name), maxVal))
				}
			}
		}
	}

	// Add PRIMARY KEY constraint if this field is a primary key
	if isPrimaryKey {
		constraints = append(constraints, "PRIMARY KEY")
	}

	// Add DEFAULT constraint if specified
	if field.Default != nil {
		defaultValue := g.formatDefaultValue(field.Default)
		if defaultValue != "" {
			constraints = append(constraints, fmt.Sprintf("DEFAULT %s", defaultValue))
		}
	}

	return strings.Join(constraints, " ")
}

// formatDefaultValue formats a default value for SQL
func (g *Generator) formatDefaultValue(expr ast.ExprNode) string {
	if lit, ok := expr.(*ast.LiteralExpr); ok {
		switch v := lit.Value.(type) {
		case string:
			return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
		case int, int64, float64:
			return fmt.Sprintf("%v", v)
		case bool:
			if v {
				return "TRUE"
			}
			return "FALSE"
		}
	}
	return ""
}

// generateIndexes generates index statements for a resource
func (g *Generator) generateIndexes(resource *ast.ResourceNode) string {
	var sql strings.Builder
	tableName := g.toTableName(resource.Name)

	for _, field := range resource.Fields {
		// Create index for unique constraints
		if hasConstraint(field, "unique") {
			columnName := g.toDBColumnName(field.Name)
			indexName := fmt.Sprintf("idx_%s_%s", tableName, columnName)
			sql.WriteString(fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s(%s);\n",
				indexName, tableName, columnName))
		}

		// Create index for foreign keys
		if field.Type.Kind == ast.TypeResource {
			columnName := g.toDBColumnName(field.Name)
			indexName := fmt.Sprintf("idx_%s_%s", tableName, columnName)
			sql.WriteString(fmt.Sprintf("CREATE INDEX %s ON %s(%s);\n",
				indexName, tableName, columnName))
		}
	}

	return sql.String()
}
