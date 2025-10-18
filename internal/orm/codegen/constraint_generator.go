// Package codegen provides constraint generation for database schemas
package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// ConstraintGenerator generates constraint DDL statements
type ConstraintGenerator struct {
	typeMapper *TypeMapper
}

// NewConstraintGenerator creates a new constraint generator
func NewConstraintGenerator() *ConstraintGenerator {
	return &ConstraintGenerator{
		typeMapper: NewTypeMapper(),
	}
}

// GenerateCheckConstraints generates CHECK constraints for field validations
func (g *ConstraintGenerator) GenerateCheckConstraints(resource *schema.ResourceSchema) []string {
	var constraints []string
	tableName := resource.TableName
	if tableName == "" {
		tableName = toSnakeCase(resource.Name)
	}

	// Process each field for constraints
	for fieldName, field := range resource.Fields {
		columnName := toSnakeCase(fieldName)

		for _, constraint := range field.Constraints {
			switch constraint.Type {
			case schema.ConstraintMin:
				checkName := fmt.Sprintf("%s_%s_min", tableName, columnName)
				checkSQL := g.generateMinConstraint(tableName, columnName, field.Type, constraint)
				if checkSQL != "" {
					constraints = append(constraints,
						fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s;",
							QuoteIdentifier(tableName), QuoteIdentifier(checkName), checkSQL))
				}

			case schema.ConstraintMax:
				checkName := fmt.Sprintf("%s_%s_max", tableName, columnName)
				checkSQL := g.generateMaxConstraint(tableName, columnName, field.Type, constraint)
				if checkSQL != "" {
					constraints = append(constraints,
						fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s;",
							QuoteIdentifier(tableName), QuoteIdentifier(checkName), checkSQL))
				}

			case schema.ConstraintPattern:
				checkName := fmt.Sprintf("%s_%s_pattern", tableName, columnName)
				checkSQL := g.generatePatternConstraint(columnName, constraint)
				if checkSQL != "" {
					constraints = append(constraints,
						fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s;",
							QuoteIdentifier(tableName), QuoteIdentifier(checkName), checkSQL))
				}
			}
		}

		// Generate built-in validation constraints for validated types
		if field.Type.IsValidated() {
			checkSQL := g.generateValidatedTypeConstraint(columnName, field.Type)
			if checkSQL != "" {
				checkName := fmt.Sprintf("%s_%s_valid", tableName, columnName)
				constraints = append(constraints,
					fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s;",
						QuoteIdentifier(tableName), QuoteIdentifier(checkName), checkSQL))
			}
		}
	}

	// Sort for deterministic output
	sort.Strings(constraints)

	return constraints
}

// generateMinConstraint generates a CHECK constraint for @min
func (g *ConstraintGenerator) generateMinConstraint(tableName, columnName string, typeSpec *schema.TypeSpec, constraint schema.Constraint) string {
	if typeSpec.IsNumeric() {
		// Numeric minimum
		return fmt.Sprintf("CHECK (%s >= %v)", QuoteIdentifier(columnName), constraint.Value)
	} else if typeSpec.IsText() {
		// Text length minimum
		return fmt.Sprintf("CHECK (LENGTH(%s) >= %v)", QuoteIdentifier(columnName), constraint.Value)
	}
	return ""
}

// generateMaxConstraint generates a CHECK constraint for @max
func (g *ConstraintGenerator) generateMaxConstraint(tableName, columnName string, typeSpec *schema.TypeSpec, constraint schema.Constraint) string {
	if typeSpec.IsNumeric() {
		// Numeric maximum
		return fmt.Sprintf("CHECK (%s <= %v)", QuoteIdentifier(columnName), constraint.Value)
	} else if typeSpec.IsText() {
		// Text length maximum
		return fmt.Sprintf("CHECK (LENGTH(%s) <= %v)", QuoteIdentifier(columnName), constraint.Value)
	}
	return ""
}

// generatePatternConstraint generates a CHECK constraint for @pattern
func (g *ConstraintGenerator) generatePatternConstraint(columnName string, constraint schema.Constraint) string {
	if pattern, ok := constraint.Value.(string); ok {
		// Escape single quotes in the pattern
		escapedPattern := strings.ReplaceAll(pattern, "'", "''")
		return fmt.Sprintf("CHECK (%s ~ '%s')", QuoteIdentifier(columnName), escapedPattern)
	}
	return ""
}

// generateValidatedTypeConstraint generates CHECK constraints for validated types
func (g *ConstraintGenerator) generateValidatedTypeConstraint(columnName string, typeSpec *schema.TypeSpec) string {
	switch typeSpec.BaseType {
	case schema.TypeEmail:
		// Simple email validation regex
		return fmt.Sprintf("CHECK (%s ~ '^[A-Za-z0-9._%%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$')", QuoteIdentifier(columnName))

	case schema.TypeURL:
		// Simple URL validation regex
		return fmt.Sprintf("CHECK (%s ~ '^https?://.+')", QuoteIdentifier(columnName))

	case schema.TypePhone:
		// Simple phone validation (digits, spaces, dashes, parens, plus)
		return fmt.Sprintf("CHECK (%s ~ '^[+0-9\\\\s\\\\-\\\\(\\\\)]+$')", QuoteIdentifier(columnName))
	}

	return ""
}

// GenerateForeignKeyConstraints generates FOREIGN KEY constraints for relationships
func (g *ConstraintGenerator) GenerateForeignKeyConstraints(resource *schema.ResourceSchema, registry map[string]*schema.ResourceSchema) ([]string, error) {
	var constraints []string
	tableName := resource.TableName
	if tableName == "" {
		tableName = toSnakeCase(resource.Name)
	}

	// Process belongs_to relationships
	for relName, rel := range resource.Relationships {
		if rel.Type != schema.RelationshipBelongsTo {
			continue
		}

		// Get target resource
		targetResource, exists := registry[rel.TargetResource]
		if !exists {
			return nil, fmt.Errorf("relationship %s references unknown resource %s", relName, rel.TargetResource)
		}

		// Get target primary key
		targetPK, err := targetResource.GetPrimaryKey()
		if err != nil {
			return nil, fmt.Errorf("relationship %s: %w", relName, err)
		}

		// Determine foreign key column name
		foreignKeyColumn := rel.ForeignKey
		if foreignKeyColumn == "" {
			foreignKeyColumn = toSnakeCase(rel.TargetResource) + "_id"
		} else {
			foreignKeyColumn = toSnakeCase(foreignKeyColumn)
		}

		// Determine target table and column
		targetTable := targetResource.TableName
		if targetTable == "" {
			targetTable = toSnakeCase(targetResource.Name)
		}
		targetColumn := toSnakeCase(targetPK.Name)

		// Build foreign key constraint
		constraintName := fmt.Sprintf("%s_%s_fkey", tableName, foreignKeyColumn)

		fkSQL := fmt.Sprintf(
			"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
			QuoteIdentifier(tableName),
			QuoteIdentifier(constraintName),
			QuoteIdentifier(foreignKeyColumn),
			QuoteIdentifier(targetTable),
			QuoteIdentifier(targetColumn),
		)

		// Add ON DELETE action
		fkSQL += g.formatCascadeAction("ON DELETE", rel.OnDelete)

		// Add ON UPDATE action
		fkSQL += g.formatCascadeAction("ON UPDATE", rel.OnUpdate)

		fkSQL += ";"

		constraints = append(constraints, fkSQL)
	}

	// Sort for deterministic output
	sort.Strings(constraints)

	return constraints, nil
}

// formatCascadeAction formats a cascade action for SQL
func (g *ConstraintGenerator) formatCascadeAction(prefix string, action schema.CascadeAction) string {
	var actionStr string
	switch action {
	case schema.CascadeRestrict:
		actionStr = "RESTRICT"
	case schema.CascadeCascade:
		actionStr = "CASCADE"
	case schema.CascadeSetNull:
		actionStr = "SET NULL"
	case schema.CascadeNoAction:
		actionStr = "NO ACTION"
	default:
		// Default to RESTRICT for safety
		actionStr = "RESTRICT"
	}

	return fmt.Sprintf(" %s %s", prefix, actionStr)
}

// GenerateUniqueConstraints generates UNIQUE constraints for fields
func (g *ConstraintGenerator) GenerateUniqueConstraints(resource *schema.ResourceSchema) []string {
	var constraints []string
	tableName := resource.TableName
	if tableName == "" {
		tableName = toSnakeCase(resource.Name)
	}

	for fieldName, field := range resource.Fields {
		hasUnique := false
		for _, annotation := range field.Annotations {
			if annotation.Name == "unique" {
				hasUnique = true
				break
			}
		}

		if hasUnique {
			columnName := toSnakeCase(fieldName)
			constraintName := fmt.Sprintf("%s_%s_unique", tableName, columnName)
			constraints = append(constraints,
				fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);",
					QuoteIdentifier(tableName), QuoteIdentifier(constraintName), QuoteIdentifier(columnName)))
		}
	}

	// Sort for deterministic output
	sort.Strings(constraints)

	return constraints
}

// GenerateAllConstraints generates all constraints for a resource
func (g *ConstraintGenerator) GenerateAllConstraints(resource *schema.ResourceSchema, registry map[string]*schema.ResourceSchema) ([]string, error) {
	var allConstraints []string

	// CHECK constraints
	checkConstraints := g.GenerateCheckConstraints(resource)
	allConstraints = append(allConstraints, checkConstraints...)

	// UNIQUE constraints
	uniqueConstraints := g.GenerateUniqueConstraints(resource)
	allConstraints = append(allConstraints, uniqueConstraints...)

	// FOREIGN KEY constraints
	fkConstraints, err := g.GenerateForeignKeyConstraints(resource, registry)
	if err != nil {
		return nil, err
	}
	allConstraints = append(allConstraints, fkConstraints...)

	return allConstraints, nil
}

// GenerateDropConstraints generates DROP CONSTRAINT statements
func (g *ConstraintGenerator) GenerateDropConstraints(resource *schema.ResourceSchema, registry map[string]*schema.ResourceSchema) []string {
	var dropStatements []string
	tableName := resource.TableName
	if tableName == "" {
		tableName = toSnakeCase(resource.Name)
	}

	// Drop CHECK constraints
	for fieldName, field := range resource.Fields {
		columnName := toSnakeCase(fieldName)

		for _, constraint := range field.Constraints {
			var constraintName string
			switch constraint.Type {
			case schema.ConstraintMin:
				constraintName = fmt.Sprintf("%s_%s_min", tableName, columnName)
			case schema.ConstraintMax:
				constraintName = fmt.Sprintf("%s_%s_max", tableName, columnName)
			case schema.ConstraintPattern:
				constraintName = fmt.Sprintf("%s_%s_pattern", tableName, columnName)
			}
			if constraintName != "" {
				dropStatements = append(dropStatements,
					fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;", QuoteIdentifier(tableName), QuoteIdentifier(constraintName)))
			}
		}

		// Drop validated type constraints
		if field.Type.IsValidated() {
			constraintName := fmt.Sprintf("%s_%s_valid", tableName, columnName)
			dropStatements = append(dropStatements,
				fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;", QuoteIdentifier(tableName), QuoteIdentifier(constraintName)))
		}
	}

	// Drop UNIQUE constraints
	for fieldName, field := range resource.Fields {
		hasUnique := false
		for _, annotation := range field.Annotations {
			if annotation.Name == "unique" {
				hasUnique = true
				break
			}
		}
		if hasUnique {
			columnName := toSnakeCase(fieldName)
			constraintName := fmt.Sprintf("%s_%s_unique", tableName, columnName)
			dropStatements = append(dropStatements,
				fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;", QuoteIdentifier(tableName), QuoteIdentifier(constraintName)))
		}
	}

	// Drop FOREIGN KEY constraints
	for _, rel := range resource.Relationships {
		if rel.Type != schema.RelationshipBelongsTo {
			continue
		}

		foreignKeyColumn := rel.ForeignKey
		if foreignKeyColumn == "" {
			foreignKeyColumn = toSnakeCase(rel.TargetResource) + "_id"
		} else {
			foreignKeyColumn = toSnakeCase(foreignKeyColumn)
		}

		constraintName := fmt.Sprintf("%s_%s_fkey", tableName, foreignKeyColumn)
		dropStatements = append(dropStatements,
			fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;", QuoteIdentifier(tableName), QuoteIdentifier(constraintName)))
	}

	// Sort for deterministic output
	sort.Strings(dropStatements)

	return dropStatements
}
