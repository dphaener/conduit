package migrate

import (
	"fmt"
	"strings"
	"time"

	"github.com/conduit-lang/conduit/internal/orm/codegen"
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Generator generates migration SQL from schema changes
type Generator struct {
	ddlGen     *codegen.DDLGenerator
	typeMapper *codegen.TypeMapper
}

// NewGenerator creates a new migration generator
func NewGenerator() *Generator {
	return &Generator{
		ddlGen:     codegen.NewDDLGenerator(),
		typeMapper: codegen.NewTypeMapper(),
	}
}

// GenerateMigration creates a migration from schema changes
func (g *Generator) GenerateMigration(oldSchemas, newSchemas map[string]*schema.ResourceSchema) (*Migration, error) {
	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	if len(changes) == 0 {
		return nil, nil // No changes
	}

	migration := &Migration{
		Version: time.Now().UnixMilli(), // Use millisecond precision to avoid collisions
		Name:    GenerateMigrationName(changes),
	}

	// Check for breaking changes and data loss
	for _, change := range changes {
		if change.Breaking {
			migration.Breaking = true
		}
		if change.DataLoss {
			migration.DataLoss = true
		}
	}

	// Generate forward migration SQL
	upSQL, err := g.generateUpSQL(changes, newSchemas)
	if err != nil {
		return nil, fmt.Errorf("generating up SQL: %w", err)
	}
	migration.Up = upSQL

	// Generate reverse migration SQL
	downSQL, err := g.generateDownSQL(changes, oldSchemas)
	if err != nil {
		return nil, fmt.Errorf("generating down SQL: %w", err)
	}
	migration.Down = downSQL

	return migration, nil
}

// generateUpSQL generates forward migration SQL
func (g *Generator) generateUpSQL(changes []SchemaChange, newSchemas map[string]*schema.ResourceSchema) (string, error) {
	var sql strings.Builder

	sql.WriteString("-- Auto-generated migration\n")
	sql.WriteString(fmt.Sprintf("-- Generated at: %s\n\n", time.Now().Format(time.RFC3339)))

	// Process changes in safe order
	for _, change := range changes {
		switch change.Type {
		case ChangeAddResource:
			resourceSQL, err := g.generateAddResource(change, newSchemas)
			if err != nil {
				return "", err
			}
			sql.WriteString(resourceSQL)
			sql.WriteString("\n")

		case ChangeDropResource:
			sql.WriteString(g.generateDropResource(change))
			sql.WriteString("\n")

		case ChangeAddField:
			sql.WriteString(g.generateAddField(change))
			sql.WriteString("\n")

		case ChangeDropField:
			sql.WriteString(g.generateDropField(change))
			sql.WriteString("\n")

		case ChangeModifyField:
			modSQL, err := g.generateModifyField(change)
			if err != nil {
				return "", err
			}
			sql.WriteString(modSQL)
			sql.WriteString("\n")

		case ChangeAddRelationship:
			relSQL, err := g.generateAddRelationship(change, newSchemas)
			if err != nil {
				return "", err
			}
			sql.WriteString(relSQL)
			sql.WriteString("\n")

		case ChangeDropRelationship:
			sql.WriteString(g.generateDropRelationship(change))
			sql.WriteString("\n")
		}
	}

	return sql.String(), nil
}

// generateDownSQL generates reverse migration SQL
func (g *Generator) generateDownSQL(changes []SchemaChange, oldSchemas map[string]*schema.ResourceSchema) (string, error) {
	var sql strings.Builder

	sql.WriteString("-- Rollback migration\n")
	sql.WriteString(fmt.Sprintf("-- Generated at: %s\n\n", time.Now().Format(time.RFC3339)))

	// Process changes in reverse order
	for i := len(changes) - 1; i >= 0; i-- {
		change := changes[i]

		switch change.Type {
		case ChangeAddResource:
			// Reverse: drop the table
			sql.WriteString(g.generateDropResource(change))
			sql.WriteString("\n")

		case ChangeDropResource:
			// Reverse: recreate the table
			resourceSQL, err := g.generateAddResource(change, oldSchemas)
			if err != nil {
				return "", err
			}
			sql.WriteString(resourceSQL)
			sql.WriteString("\n")

		case ChangeAddField:
			// Reverse: drop the field
			sql.WriteString(g.generateDropField(change))
			sql.WriteString("\n")

		case ChangeDropField:
			// Reverse: add the field back
			sql.WriteString(g.generateAddField(change))
			sql.WriteString("\n")

		case ChangeModifyField:
			// Reverse: restore old field definition
			reverseChange := change
			reverseChange.OldValue, reverseChange.NewValue = change.NewValue, change.OldValue
			modSQL, err := g.generateModifyField(reverseChange)
			if err != nil {
				return "", err
			}
			sql.WriteString(modSQL)
			sql.WriteString("\n")

		case ChangeAddRelationship:
			// Reverse: drop the foreign key
			sql.WriteString(g.generateDropRelationship(change))
			sql.WriteString("\n")

		case ChangeDropRelationship:
			// Reverse: add the foreign key back
			relSQL, err := g.generateAddRelationship(change, oldSchemas)
			if err != nil {
				return "", err
			}
			sql.WriteString(relSQL)
			sql.WriteString("\n")
		}
	}

	return sql.String(), nil
}

// generateAddResource generates SQL to add a new resource table
func (g *Generator) generateAddResource(change SchemaChange, schemas map[string]*schema.ResourceSchema) (string, error) {
	resourceSchema := schemas[change.Resource]
	if resourceSchema == nil {
		if change.NewValue != nil {
			resourceSchema = change.NewValue.(*schema.ResourceSchema)
		} else if change.OldValue != nil {
			resourceSchema = change.OldValue.(*schema.ResourceSchema)
		} else {
			return "", fmt.Errorf("no schema found for resource: %s", change.Resource)
		}
	}

	createSQL, err := g.ddlGen.GenerateCreateTable(resourceSchema)
	if err != nil {
		return "", fmt.Errorf("generating CREATE TABLE: %w", err)
	}

	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("-- Add resource: %s\n", change.Resource))
	sql.WriteString(createSQL)
	sql.WriteString("\n")

	// Add indexes
	indexGen := codegen.NewIndexGenerator()
	indexes := indexGen.GenerateIndexes(resourceSchema)

	if len(indexes) > 0 {
		sql.WriteString("\n")
		sql.WriteString(strings.Join(indexes, "\n"))
		sql.WriteString("\n")
	}

	return sql.String(), nil
}

// generateDropResource generates SQL to drop a resource table
func (g *Generator) generateDropResource(change SchemaChange) string {
	tableName := toSnakeCase(change.Resource)
	return fmt.Sprintf("-- Drop resource: %s\nDROP TABLE IF EXISTS %s CASCADE;\n",
		change.Resource, codegen.QuoteIdentifier(tableName))
}

// generateAddField generates SQL to add a field
func (g *Generator) generateAddField(change SchemaChange) string {
	var field *schema.Field
	if change.NewValue != nil {
		field = change.NewValue.(*schema.Field)
	} else if change.OldValue != nil {
		field = change.OldValue.(*schema.Field)
	} else {
		// Should not happen, but handle gracefully
		return fmt.Sprintf("-- Unable to generate ADD COLUMN for %s.%s (missing field data)\n",
			change.Resource, change.Field)
	}
	tableName := toSnakeCase(change.Resource)
	columnName := toSnakeCase(field.Name)

	// Map type
	mappedType, _ := g.typeMapper.MapType(field.Type)
	nullability := g.typeMapper.MapNullability(field.Type)

	var parts []string
	parts = append(parts, mappedType)
	parts = append(parts, nullability)

	// Default value
	if defaultVal, _ := g.typeMapper.MapDefault(field.Type); defaultVal != "" {
		parts = append(parts, "DEFAULT "+defaultVal)
	}

	return fmt.Sprintf("-- Add field: %s.%s\nALTER TABLE %s ADD COLUMN %s %s;\n",
		change.Resource, field.Name,
		codegen.QuoteIdentifier(tableName),
		codegen.QuoteIdentifier(columnName),
		strings.Join(parts, " "))
}

// generateDropField generates SQL to drop a field
func (g *Generator) generateDropField(change SchemaChange) string {
	tableName := toSnakeCase(change.Resource)
	columnName := toSnakeCase(change.Field)

	return fmt.Sprintf("-- Drop field: %s.%s\nALTER TABLE %s DROP COLUMN IF EXISTS %s CASCADE;\n",
		change.Resource, change.Field,
		codegen.QuoteIdentifier(tableName),
		codegen.QuoteIdentifier(columnName))
}

// generateModifyField generates SQL to modify a field
func (g *Generator) generateModifyField(change SchemaChange) (string, error) {
	oldField := change.OldValue.(*schema.Field)
	newField := change.NewValue.(*schema.Field)
	tableName := toSnakeCase(change.Resource)
	columnName := toSnakeCase(change.Field)

	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("-- Modify field: %s.%s\n", change.Resource, change.Field))

	// Change type if needed
	if oldField.Type.BaseType != newField.Type.BaseType {
		mappedType, err := g.typeMapper.MapType(newField.Type)
		if err != nil {
			return "", err
		}

		sql.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;\n",
			codegen.QuoteIdentifier(tableName),
			codegen.QuoteIdentifier(columnName),
			mappedType))
	}

	// Change nullability if needed
	if oldField.Type.Nullable != newField.Type.Nullable {
		if newField.Type.Nullable {
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;\n",
				codegen.QuoteIdentifier(tableName),
				codegen.QuoteIdentifier(columnName)))
		} else {
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;\n",
				codegen.QuoteIdentifier(tableName),
				codegen.QuoteIdentifier(columnName)))
		}
	}

	// Handle DEFAULT value changes
	oldDefault := g.getConstraintValue(oldField, schema.ConstraintDefault)
	newDefault := g.getConstraintValue(newField, schema.ConstraintDefault)
	if oldDefault != newDefault {
		if newDefault != nil {
			defaultVal, _ := g.typeMapper.MapDefault(newField.Type)
			if defaultVal != "" {
				sql.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;\n",
					codegen.QuoteIdentifier(tableName),
					codegen.QuoteIdentifier(columnName),
					defaultVal))
			}
		} else if oldDefault != nil {
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;\n",
				codegen.QuoteIdentifier(tableName),
				codegen.QuoteIdentifier(columnName)))
		}
	}

	// Handle UNIQUE constraint changes
	oldUnique := g.hasConstraint(oldField, schema.ConstraintUnique)
	newUnique := g.hasConstraint(newField, schema.ConstraintUnique)
	if oldUnique != newUnique {
		constraintName := fmt.Sprintf("uq_%s_%s", tableName, columnName)
		if newUnique {
			// Add unique constraint
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);\n",
				codegen.QuoteIdentifier(tableName),
				codegen.QuoteIdentifier(constraintName),
				codegen.QuoteIdentifier(columnName)))
		} else {
			// Drop unique constraint
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n",
				codegen.QuoteIdentifier(tableName),
				codegen.QuoteIdentifier(constraintName)))
		}
	}

	// Handle CHECK constraint changes (for min/max/pattern)
	oldCheckConstraints := g.getCheckConstraints(oldField)
	newCheckConstraints := g.getCheckConstraints(newField)
	if oldCheckConstraints != newCheckConstraints {
		constraintName := fmt.Sprintf("chk_%s_%s", tableName, columnName)

		// Drop old check constraint if exists
		if oldCheckConstraints != "" {
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n",
				codegen.QuoteIdentifier(tableName),
				codegen.QuoteIdentifier(constraintName)))
		}

		// Add new check constraint if needed
		if newCheckConstraints != "" {
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s);\n",
				codegen.QuoteIdentifier(tableName),
				codegen.QuoteIdentifier(constraintName),
				newCheckConstraints))
		}
	}

	return sql.String(), nil
}

// hasConstraint checks if a field has a specific constraint type
func (g *Generator) hasConstraint(field *schema.Field, cType schema.ConstraintType) bool {
	for i := range field.Constraints {
		if field.Constraints[i].Type == cType {
			return true
		}
	}
	return false
}

// getConstraintValue gets the value of a specific constraint type, or nil if not found
func (g *Generator) getConstraintValue(field *schema.Field, cType schema.ConstraintType) interface{} {
	for i := range field.Constraints {
		if field.Constraints[i].Type == cType {
			return field.Constraints[i].Value
		}
	}
	return nil
}

// getCheckConstraints builds CHECK constraint SQL for min/max/pattern constraints
func (g *Generator) getCheckConstraints(field *schema.Field) string {
	var conditions []string
	columnName := toSnakeCase(field.Name)

	for i := range field.Constraints {
		c := &field.Constraints[i]
		switch c.Type {
		case schema.ConstraintMin:
			if val, ok := c.Value.(int); ok {
				conditions = append(conditions, fmt.Sprintf("%s >= %d", codegen.QuoteIdentifier(columnName), val))
			}
		case schema.ConstraintMax:
			if val, ok := c.Value.(int); ok {
				conditions = append(conditions, fmt.Sprintf("%s <= %d", codegen.QuoteIdentifier(columnName), val))
			}
		case schema.ConstraintPattern:
			if val, ok := c.Value.(string); ok {
				// Escape single quotes in pattern
				escapedVal := strings.ReplaceAll(val, "'", "''")
				conditions = append(conditions, fmt.Sprintf("%s ~ '%s'", codegen.QuoteIdentifier(columnName), escapedVal))
			}
		}
	}

	if len(conditions) == 0 {
		return ""
	}

	return strings.Join(conditions, " AND ")
}

// generateAddRelationship generates SQL to add a foreign key
func (g *Generator) generateAddRelationship(change SchemaChange, schemas map[string]*schema.ResourceSchema) (string, error) {
	rel := change.NewValue.(*schema.Relationship)
	if rel == nil {
		rel = change.OldValue.(*schema.Relationship)
	}

	// Only generate FK for belongs_to relationships
	if rel.Type != schema.RelationshipBelongsTo {
		return "", nil
	}

	tableName := toSnakeCase(change.Resource)
	foreignKey := rel.ForeignKey
	if foreignKey == "" {
		foreignKey = toSnakeCase(rel.TargetResource) + "_id"
	}

	targetTable := toSnakeCase(rel.TargetResource)
	constraintName := fmt.Sprintf("fk_%s_%s", tableName, foreignKey)

	onDelete := mapCascadeAction(rel.OnDelete)
	onUpdate := mapCascadeAction(rel.OnUpdate)

	return fmt.Sprintf("-- Add relationship: %s.%s -> %s\nALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(id) ON DELETE %s ON UPDATE %s;\n",
		change.Resource, change.Relation, rel.TargetResource,
		codegen.QuoteIdentifier(tableName),
		codegen.QuoteIdentifier(constraintName),
		codegen.QuoteIdentifier(foreignKey),
		codegen.QuoteIdentifier(targetTable),
		onDelete,
		onUpdate), nil
}

// generateDropRelationship generates SQL to drop a foreign key
func (g *Generator) generateDropRelationship(change SchemaChange) string {
	var rel *schema.Relationship
	if change.OldValue != nil {
		rel = change.OldValue.(*schema.Relationship)
	} else if change.NewValue != nil {
		rel = change.NewValue.(*schema.Relationship)
	} else {
		return fmt.Sprintf("-- Unable to generate DROP CONSTRAINT for %s.%s (missing relationship data)\n",
			change.Resource, change.Relation)
	}

	tableName := toSnakeCase(change.Resource)
	foreignKey := rel.ForeignKey
	if foreignKey == "" {
		foreignKey = toSnakeCase(rel.TargetResource) + "_id"
	}

	constraintName := fmt.Sprintf("fk_%s_%s", tableName, foreignKey)

	return fmt.Sprintf("-- Drop relationship: %s.%s\nALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n",
		change.Resource, change.Relation,
		codegen.QuoteIdentifier(tableName),
		codegen.QuoteIdentifier(constraintName))
}

// Helper functions

func mapCascadeAction(action schema.CascadeAction) string {
	switch action {
	case schema.CascadeRestrict:
		return "RESTRICT"
	case schema.CascadeCascade:
		return "CASCADE"
	case schema.CascadeSetNull:
		return "SET NULL"
	case schema.CascadeNoAction:
		return "NO ACTION"
	default:
		return "RESTRICT"
	}
}

func toSnakeCase(s string) string {
	var result []rune
	runes := []rune(s)

	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			if prev >= 'a' && prev <= 'z' {
				result = append(result, '_')
			} else if i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
				result = append(result, '_')
			}
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+('a'-'A'))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
